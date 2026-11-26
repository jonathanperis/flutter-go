// Reference side of flutter-go's scheduler differential test: a
// BindingBase with SchedulerBinding, driven in fake_async by a pump that
// copies flutter_test's AutomatedTestWidgetsFlutterBinding.pump (elapse;
// if a frame is scheduled: flush microtasks, handleBeginFrame(clock),
// flush microtasks, handleDrawFrame; flush microtasks). Every scene logs
// the order of transient/persistent/post-frame callbacks, microtasks,
// timers and futures, the phase each one sees, frame time stamps, ticker
// ticks (elapsed), TickerFuture completion/orCancel/whenCompleteOrCancel,
// time dilation, task order by priority, lifecycle, warm-up frames,
// errors, timings and the frame banners. Output: build/scheduler.json as
// {"log": [...]}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:fake_async/fake_async.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> entries = <String>[];
void log(String s) { entries.add(s); if (Platform.environment["TRACE"] != null) stderr.writeln(s); }

class TestBinding extends BindingBase with SchedulerBinding {
  @override
  void handleBeginFrame(Duration? rawTimeStamp) {
    log('handleBeginFrame ${rawTimeStamp == null ? 'null' : (rawTimeStamp.inMicroseconds - epoch)} phase=${schedulerPhase.name}');
    super.handleBeginFrame(rawTimeStamp);
    log('handleBeginFrame done phase=${schedulerPhase.name}');
  }

  @override
  void handleDrawFrame() {
    log('handleDrawFrame phase=${schedulerPhase.name}');
    super.handleDrawFrame();
    log('handleDrawFrame done phase=${schedulerPhase.name}');
  }

  @override
  void postEvent(String eventKind, Map<String, dynamic> eventData) {
    log('postEvent $eventKind ${json.encode(eventData)}');
  }

  @override
  void unlocked() {
    log('unlocked');
    super.unlocked();
  }
}

final int epoch = DateTime.utc(2015).microsecondsSinceEpoch;
late TestBinding binding;
late FakeAsync fa;

String phase() => binding.schedulerPhase.name;
String us(Duration d) => '${d.inMicroseconds}';

/// flutter_test's pump.
void pump([Duration? duration]) {
  log('pump ${duration?.inMicroseconds}');
  if (duration != null) {
    fa.elapse(duration);
  }
  if (binding.hasScheduledFrame) {
    fa.flushMicrotasks();
    binding.handleBeginFrame(Duration(microseconds: epoch + fa.elapsed.inMicroseconds));
    fa.flushMicrotasks();
    binding.handleDrawFrame();
  }
  fa.flushMicrotasks();
  log('pumped hasScheduledFrame=${binding.hasScheduledFrame} transient=${binding.transientCallbackCount}');
}

String futureState(TickerFuture f) => f.toString().replaceFirst(RegExp(r'^TickerFuture#[0-9a-f]{5}'), 'TickerFuture');

void listen(String name, Future<void> f) {
  f.then<void>((_) => log('$name complete phase=${phase()}'), onError: (Object e) => log('$name error ${e.runtimeType} $e phase=${phase()}'));
}

void sceneFrameOrder() {
  log('== frame order');
  binding.addPersistentFrameCallback((Duration ts) {
    log('persistent A ts=${us(ts)} phase=${phase()} current=${us(binding.currentFrameTimeStamp)} system=${binding.currentSystemFrameTimeStamp.inMicroseconds - epoch}');
  });
  var addedNested = false;
  binding.addPersistentFrameCallback((Duration ts) {
    log('persistent B phase=${phase()}');
    if (!addedNested) {
      addedNested = true;
      binding.addPersistentFrameCallback((Duration ts) => log('persistent C (added during B)'));
      binding.addPostFrameCallback((Duration ts) {
        log('post 1 (added in persistent) phase=${phase()}');
        binding.addPostFrameCallback((Duration ts) => log('post 3 (added in post 1) phase=${phase()}'));
        scheduleMicrotask(() => log('microtask from post 1 phase=${phase()}'));
      });
    }
  });
  log('hasScheduledFrame=${binding.hasScheduledFrame}');
  pump(); // nothing scheduled: no frame
  binding.addPostFrameCallback((Duration ts) => log('post 2 (added before frame) ts=${us(ts)}'), debugLabel: 'two');
  final int id1 = binding.scheduleFrameCallback((Duration ts) {
    log('transient 1 ts=${us(ts)} phase=${phase()}');
    scheduleMicrotask(() => log('microtask from transient 1 phase=${phase()}'));
    final c = Completer<void>();
    c.future.then((_) => log('future from transient 1 phase=${phase()}'));
    c.complete();
    binding.scheduleFrameCallback((Duration ts) => log('transient 3 (added in 1) ts=${us(ts)} phase=${phase()}'));
    log('in transient: hasScheduledFrame=${binding.hasScheduledFrame} count=${binding.transientCallbackCount}');
  });
  final int id2 = binding.scheduleFrameCallback((Duration ts) => log('transient 2 ts=${us(ts)}'));
  log('ids $id1 $id2 count=${binding.transientCallbackCount} hasScheduledFrame=${binding.hasScheduledFrame}');
  scheduleMicrotask(() => log('microtask before frame phase=${phase()}'));
  Timer(const Duration(milliseconds: 5), () => log('timer 5ms phase=${phase()}'));
  Timer(const Duration(milliseconds: 16), () => log('timer 16ms phase=${phase()}'));
  pump(const Duration(milliseconds: 16));
  pump(const Duration(microseconds: 16667));
  pump(const Duration(milliseconds: 1)); // post 3 ran? no frame scheduled after post-frame
  binding.scheduleFrame();
  pump(const Duration(milliseconds: 3));
  // Cancel: a callback canceled before its frame, one canceled by an earlier
  // callback in the same frame, and canceling the current one.
  late int idB;
  late int idSelf;
  binding.scheduleFrameCallback((Duration ts) {
    log('transient cancel A');
    binding.cancelFrameCallbackWithId(idB);
  });
  idB = binding.scheduleFrameCallback((Duration ts) => log('transient cancel B (must not run)'));
  final int idC = binding.scheduleFrameCallback((Duration ts) => log('transient cancel C (must not run)'));
  idSelf = binding.scheduleFrameCallback((Duration ts) {
    log('transient self');
    binding.cancelFrameCallbackWithId(idSelf);
  });
  binding.cancelFrameCallbackWithId(idC);
  log('count after cancel=${binding.transientCallbackCount}');
  pump(const Duration(milliseconds: 10));
  // scheduleNewFrame: false
  binding.scheduleFrameCallback((Duration ts) => log('transient no new frame ts=${us(ts)}'), scheduleNewFrame: false);
  log('hasScheduledFrame=${binding.hasScheduledFrame}');
  pump(const Duration(milliseconds: 10));
  binding.ensureVisualUpdate();
  pump(const Duration(milliseconds: 10));
  // ensureVisualUpdate in each phase.
  binding.scheduleFrameCallback((Duration ts) {
    binding.ensureVisualUpdate();
    log('ensureVisualUpdate in transient: ${binding.hasScheduledFrame}');
    scheduleMicrotask(() {
      binding.ensureVisualUpdate();
      log('ensureVisualUpdate in midFrameMicrotasks: ${binding.hasScheduledFrame}');
    });
  });
  var once = false;
  binding.addPersistentFrameCallback((Duration ts) {
    if (!once) {
      once = true;
      binding.ensureVisualUpdate();
      log('ensureVisualUpdate in persistent: ${binding.hasScheduledFrame}');
      binding.addPostFrameCallback((Duration ts) {
        binding.ensureVisualUpdate();
        log('ensureVisualUpdate in post: ${binding.hasScheduledFrame}');
      });
    }
  });
  pump(const Duration(milliseconds: 10));
  pump(const Duration(milliseconds: 10));
  // Null time stamp: reuses the last one.
  binding.scheduleFrameCallback((Duration ts) => log('transient warm ts=${us(ts)}'));
  binding.handleBeginFrame(null);
  fa.flushMicrotasks();
  binding.handleDrawFrame();
  pump(const Duration(milliseconds: 7));
}

void sceneEndOfFrame() {
  log('== endOfFrame');
  listen('endOfFrame idle', binding.endOfFrame);
  log('identical=${identical(binding.endOfFrame, binding.endOfFrame)} hasScheduledFrame=${binding.hasScheduledFrame}');
  pump(const Duration(milliseconds: 4));
  binding.scheduleFrameCallback((Duration ts) {
    listen('endOfFrame in transient', binding.endOfFrame);
  });
  var done = false;
  binding.addPersistentFrameCallback((Duration ts) {
    if (!done) {
      done = true;
      binding.addPostFrameCallback((Duration ts) {
        listen('endOfFrame in post', binding.endOfFrame);
        log('in post hasScheduledFrame=${binding.hasScheduledFrame}');
      });
    }
  });
  pump(const Duration(milliseconds: 4));
  pump(const Duration(milliseconds: 4));
  pump(const Duration(milliseconds: 4));
}

void sceneTickers() {
  log('== tickers');
  Ticker make(String name) => Ticker((Duration elapsed) => log('$name tick ${us(elapsed)} phase=${phase()}'), debugLabel: name);
  final Ticker outside = make('outside');
  log('${outside} isActive=${outside.isActive} isTicking=${outside.isTicking}');
  final TickerFuture f1 = outside.start();
  listen('outside future', f1);
  f1.whenCompleteOrCancel(() => log('outside whenCompleteOrCancel'));
  log('${futureState(f1)} isActive=${outside.isActive} isTicking=${outside.isTicking}');
  pump(const Duration(milliseconds: 10));
  late Ticker inTransient;
  late Ticker inMicrotask;
  late Ticker inPersistent;
  late Ticker inPost;
  binding.scheduleFrameCallback((Duration ts) {
    inTransient = make('inTransient')..start();
    scheduleMicrotask(() => inMicrotask = make('inMicrotask')..start());
  });
  var started = false;
  binding.addPersistentFrameCallback((Duration ts) {
    if (!started) {
      started = true;
      inPersistent = make('inPersistent')..start();
      binding.addPostFrameCallback((Duration ts) => inPost = make('inPost')..start());
    }
  });
  pump(const Duration(milliseconds: 7));
  pump(const Duration(milliseconds: 16));
  pump(const Duration(milliseconds: 16));
  // Muting.
  inTransient.muted = true;
  log('muted: isActive=${inTransient.isActive} isTicking=${inTransient.isTicking}');
  pump(const Duration(milliseconds: 16));
  inTransient.muted = false;
  late Ticker selfMuter;
  var muterTicks = 0;
  selfMuter = Ticker((Duration elapsed) {
    log('selfMuter tick ${us(elapsed)}');
    if (++muterTicks == 2) {
      selfMuter.muted = true;
    }
  })..start();
  pump(const Duration(milliseconds: 16));
  // Stop and restart inside onTick.
  late Ticker restarter;
  var restarts = 0;
  restarter = Ticker((Duration elapsed) {
    log('restarter tick ${us(elapsed)}');
    if (restarts++ < 2) {
      restarter.stop();
      listen('restarter future $restarts', restarter.start());
    }
  });
  restarter.start();
  pump(const Duration(milliseconds: 5));
  pump(const Duration(milliseconds: 5));
  pump(const Duration(milliseconds: 5));
  pump(const Duration(milliseconds: 5));
  // stop, stop canceled, dispose, orCancel before/after.
  final TickerFuture f2 = TickerFuture.complete();
  log('complete(): ${futureState(f2)}');
  listen('complete() orCancel', f2.orCancel);
  outside.stop();
  log('after stop ${futureState(f1)}');
  listen('outside orCancel after stop', f1.orCancel);
  inPersistent.stop(canceled: true);
  pump(const Duration(milliseconds: 1));
  final Ticker c = make('canceled');
  final TickerFuture fc = c.start();
  listen('canceled future', fc);
  listen('canceled orCancel', fc.orCancel);
  fc.whenCompleteOrCancel(() => log('canceled whenCompleteOrCancel'));
  c.stop(canceled: true);
  log('${futureState(fc)}');
  listen('canceled orCancel late', fc.orCancel);
  final Ticker c2 = make('canceled2');
  final TickerFuture fc2 = c2.start();
  c2.stop(canceled: true);
  listen('orCancel first read after cancel', fc2.orCancel);
  final Ticker d = make('disposed');
  final TickerFuture fd = d.start();
  listen('disposed orCancel', fd.orCancel);
  pump(const Duration(milliseconds: 2));
  d.dispose();
  log('disposed ${futureState(fd)} transient=${binding.transientCallbackCount}');
  final Ticker t2 = Ticker((_) {});
  final TickerFuture fd2 = t2.start()..ignore();
  log('fd2 ${futureState(fd2)}');
  t2.dispose();
  log('fd2 after dispose ${futureState(fd2)}');
  // await a TickerFuture, and timeout/whenComplete through it.
  final Ticker w = make('awaited');
  final TickerFuture fw = w.start();
  () async {
    log('awaiting');
    await fw;
    log('await returned phase=${phase()}');
  }();
  fw.whenComplete(() => log('whenComplete phase=${phase()}'));
  fw.timeout(const Duration(milliseconds: 100)).then((_) => log('timeout result complete'), onError: (Object e) => log('timeout error ${e.runtimeType}'));
  pump(const Duration(milliseconds: 3));
  w.stop();
  log('stopped awaited');
  pump(const Duration(milliseconds: 3));
  // absorbTicker.
  final Ticker original = make('original');
  final TickerFuture fo = original.start();
  listen('original future', fo);
  pump(const Duration(milliseconds: 4));
  pump(const Duration(milliseconds: 4));
  final Ticker successor = make('successor');
  successor.absorbTicker(original);
  log('absorbed: successor active=${successor.isActive} original active=${original.isActive} transient=${binding.transientCallbackCount}');
  pump(const Duration(milliseconds: 4));
  successor.stop();
  pump(const Duration(milliseconds: 4));
  final Ticker idleOriginal = make('idleOriginal');
  make('idleSuccessor').absorbTicker(idleOriginal);
  // Ticker started twice.
  final Ticker twice = make('twice')..start();
  try {
    twice.start();
  } on FlutterError catch (e) {
    log('started twice: ${e.diagnostics.length} ${e.diagnostics.first} | ${e.diagnostics[1]} | ${e.diagnostics.last.name}');
  }
  twice.dispose();
  log('selfMuter active=${selfMuter.isActive} ticking=${selfMuter.isTicking}');
  for (final Ticker t in <Ticker>[inTransient, inMicrotask, inPost, restarter, selfMuter]) {
    t.dispose();
  }
  pump(const Duration(milliseconds: 1));
  log('${TickerCanceled()} | ${TickerCanceled(make('x'))}');
}

void sceneTimeDilation() {
  log('== timeDilation');
  final Ticker t = Ticker((Duration elapsed) => log('dilated tick ${us(elapsed)}'));
  t.start();
  binding.scheduleFrameCallback((Duration ts) => log('ts ${us(ts)}'));
  pump(const Duration(milliseconds: 10));
  pump(const Duration(milliseconds: 10));
  timeDilation = 2.0;
  binding.scheduleFrameCallback((Duration ts) => log('ts ${us(ts)}'));
  pump(const Duration(milliseconds: 10));
  pump(const Duration(microseconds: 16667));
  timeDilation = 3.0;
  pump(const Duration(microseconds: 16667));
  pump(const Duration(microseconds: 16667));
  pump(const Duration(microseconds: 1));
  pump(const Duration(microseconds: 2));
  timeDilation = 3.0; // same value: no epoch reset
  pump(const Duration(microseconds: 5));
  timeDilation = 0.3;
  pump(const Duration(microseconds: 7));
  pump(const Duration(microseconds: 16667));
  final Ticker tie = Ticker((Duration elapsed) => log('tie tick ${us(elapsed)}'))..start();
  timeDilation = 2.0;
  pump(const Duration(microseconds: 1));
  pump(const Duration(microseconds: 1)); // 1 / 2 = 0.5
  pump(const Duration(microseconds: 2)); // 3 / 2 = 1.5
  pump(const Duration(microseconds: 2)); // 5 / 2 = 2.5
  timeDilation = 1.0;
  tie.dispose();
  binding.scheduleFrameCallback((Duration ts) => log('ts ${us(ts)}'));
  pump(const Duration(milliseconds: 10));
  t.dispose();
  binding.resetEpoch();
  binding.scheduleFrameCallback((Duration ts) => log('after resetEpoch ts=${us(ts)} system=${binding.currentSystemFrameTimeStamp.inMicroseconds - epoch}'));
  pump(const Duration(milliseconds: 10));
}

void sceneTasks() {
  log('== tasks');
  void schedule(String name, Priority p, {bool throws = false}) {
    binding.scheduleTask<String>(() {
      log('task $name phase=${phase()} transient=${binding.transientCallbackCount}');
      if (throws) {
        throw StateError('task $name failed');
      }
      return name;
    }, p, debugLabel: name).then((String v) => log('task result $v'));
  }
  schedule('idle', Priority.idle);
  schedule('touch', Priority.touch);
  schedule('animation', Priority.animation);
  schedule('animation+1', Priority.animation + 1);
  schedule('animation-1', Priority.animation - 1);
  schedule('idle+20000', Priority.idle + 20000);
  schedule('tie a', Priority.idle + 7);
  schedule('tie b', Priority.idle + 7);
  schedule('tie c', Priority.idle + 7);
  schedule('tie d', Priority.idle + 7);
  schedule('touch-99999', Priority.touch - 99999);
  schedule('thrower', Priority.animation + 5, throws: true);
  log('scheduled; timers=${fa.pendingTimers.length}');
  // A transient callback is registered: the default strategy runs only
  // animation priority and above.
  final int id = binding.scheduleFrameCallback((Duration ts) => log('transient during tasks'));
  // Run the loop by hand: while a task is blocked by the strategy,
  // handleEventLoopCallback returns true and _runTasks re-arms Timer.run, so
  // elapsing would spin forever.
  for (var i = 0; i < 7; i++) {
    log('handleEventLoopCallback=${binding.handleEventLoopCallback()}');
  }
  binding.cancelFrameCallbackWithId(id);
  log('transient=${binding.transientCallbackCount}');
  binding.scheduleTask<void>(() => log('task late'), Priority.idle + 3);
  fa.elapse(Duration.zero);
  pump(const Duration(milliseconds: 1));
  fa.elapse(Duration.zero);
  log('after drain');
  // A task returning a future.
  binding.scheduleTask<int>(() async {
    log('async task start');
    await null;
    log('async task after await');
    return 42;
  }, Priority.touch).then((int v) => log('async task result $v'));
  fa.elapse(Duration.zero);
  // Tasks wait while events are locked.
  final lock = Completer<void>();
  binding.lockEvents(() => lock.future);
  binding.scheduleTask<void>(() => log('task while locked'), Priority.touch);
  log('locked; handleEventLoopCallback=${binding.handleEventLoopCallback()}');
  fa.elapse(Duration.zero);
  log('still locked');
  lock.complete();
  fa.flushMicrotasks();
  log('lock released');
  fa.elapse(Duration.zero);
  // A custom strategy.
  for (final List<int> pattern in <List<int>>[
    <int>[5, 5, 3, 5, 3, 3, 5, 1, 5, 3, 1, 5, 5, 3],
    <int>[1, 2, 2, 1, 2, 1, 1, 2, 2, 2, 1],
    <int>[4, 4, 4, 4, 4, 4, 4, 4, 4, 4],
    <int>[9, 1, 9, 1, 9, 1, 9, 1],
  ]) {
    var n = 0;
    final List<String> order = <String>[];
    for (final int p in pattern) {
      final String name = '$p#${n++}';
      binding.scheduleTask<void>(() => order.add(name), Priority.idle + p);
    }
    while (binding.handleEventLoopCallback()) {}
    log('heap order ${order.join(' ')}');
    fa.elapse(Duration.zero);
  }
  // An unlock while a task timer is pending.
  binding.scheduleTask<void>(() => log('task before unlock'), Priority.touch);
  binding.lockEvents(() => Future<void>.value());
  fa.flushMicrotasks();
  log('timers after unlock=${fa.pendingTimers.length}');
  fa.elapse(Duration.zero);
  binding.schedulingStrategy = ({required int priority, required SchedulerBinding scheduler}) {
    log('strategy asked $priority');
    return priority > 5;
  };
  schedule('s3', Priority.idle + 3);
  schedule('s9', Priority.idle + 9);
  for (var i = 0; i < 3; i++) {
    log('handleEventLoopCallback=${binding.handleEventLoopCallback()}');
  }
  binding.schedulingStrategy = defaultSchedulingStrategy;
  fa.elapse(Duration.zero);
  log('handleEventLoopCallback empty=${binding.handleEventLoopCallback()}');
}

void sceneLifecycle() {
  log('== lifecycle');
  log('state=${binding.lifecycleState} framesEnabled=${binding.framesEnabled} hasScheduledFrame=${binding.hasScheduledFrame}');
  final Ticker t = Ticker((Duration e) => log('lifecycle tick ${us(e)}'));
  for (final ui.AppLifecycleState s in <ui.AppLifecycleState>[
    ui.AppLifecycleState.resumed,
    ui.AppLifecycleState.inactive,
    ui.AppLifecycleState.hidden,
  ]) {
    binding.handleAppLifecycleStateChanged(s);
    log('$s framesEnabled=${binding.framesEnabled} hasScheduledFrame=${binding.hasScheduledFrame}');
  }
  t.start();
  log('started while hidden: isTicking=${t.isTicking} hasScheduledFrame=${binding.hasScheduledFrame} transient=${binding.transientCallbackCount}');
  pump(const Duration(milliseconds: 10));
  binding.scheduleForcedFrame();
  log('forced: hasScheduledFrame=${binding.hasScheduledFrame}');
  pump(const Duration(milliseconds: 10));
  final Ticker forced = Ticker((Duration e) => log('forced tick ${us(e)}'))..forceFrames = true;
  forced.start();
  log('forceFrames ticker: hasScheduledFrame=${binding.hasScheduledFrame}');
  pump(const Duration(milliseconds: 10));
  pump(const Duration(milliseconds: 10));
  forced.dispose();
  binding.handleAppLifecycleStateChanged(ui.AppLifecycleState.paused);
  binding.handleAppLifecycleStateChanged(ui.AppLifecycleState.paused);
  log('paused framesEnabled=${binding.framesEnabled} hasScheduledFrame=${binding.hasScheduledFrame}');
  binding.handleAppLifecycleStateChanged(ui.AppLifecycleState.resumed);
  log('resumed framesEnabled=${binding.framesEnabled} hasScheduledFrame=${binding.hasScheduledFrame}');
  pump(const Duration(milliseconds: 10));
  pump(const Duration(milliseconds: 10));
  binding.handleAppLifecycleStateChanged(ui.AppLifecycleState.detached);
  log('detached framesEnabled=${binding.framesEnabled}');
  binding.handleAppLifecycleStateChanged(ui.AppLifecycleState.hidden);
  log('hidden framesEnabled=${binding.framesEnabled} hasScheduledFrame=${binding.hasScheduledFrame}');
  pump(const Duration(milliseconds: 10));
  binding.handleAppLifecycleStateChanged(ui.AppLifecycleState.inactive);
  log('inactive framesEnabled=${binding.framesEnabled} hasScheduledFrame=${binding.hasScheduledFrame}');
  pump(const Duration(milliseconds: 10));
  binding.resetInternalState();
  log('reset state=${binding.lifecycleState} framesEnabled=${binding.framesEnabled} hasScheduledFrame=${binding.hasScheduledFrame}');
  pump(const Duration(milliseconds: 10));
  t.dispose();
  pump(const Duration(milliseconds: 10));
}

void sceneWarmUp() {
  log('== warm-up');
  final Ticker t = Ticker((Duration e) => log('warm tick ${us(e)} phase=${phase()}'));
  t.start();
  binding.scheduleFrameCallback((Duration ts) {
    log('warm transient ts=${us(ts)}');
    scheduleMicrotask(() => log('warm microtask phase=${phase()}'));
  });
  log('before warm-up hasScheduledFrame=${binding.hasScheduledFrame} locked=${binding.locked} timers=${fa.pendingTimers.length}');
  binding.scheduleWarmUpFrame();
  binding.scheduleWarmUpFrame(); // ignored
  log('warm-up scheduled locked=${binding.locked} timers=${fa.pendingTimers.length} hasScheduledFrame=${binding.hasScheduledFrame}');
  binding.scheduleTask<void>(() => log('task after warm-up'), Priority.touch);
  // The engine's frame arrives during the warm-up frame: postponed.
  scheduleMicrotask(() => log('microtask before warm-up timers'));
  void engineFrame(String when) {
    log('engine frame $when');
    binding.ensureFrameCallbacksRegistered();
    ui.PlatformDispatcher.instance.onBeginFrame!(Duration(microseconds: epoch + fa.elapsed.inMicroseconds));
    ui.PlatformDispatcher.instance.onDrawFrame!();
    log('engine frame returned hasScheduledFrame=${binding.hasScheduledFrame} phase=${phase()}');
  }
  // In a microtask of the warm-up begin frame, i.e. between its two timers.
  binding.scheduleFrameCallback((Duration ts) => scheduleMicrotask(() => engineFrame('during warm-up')));
  Timer(const Duration(milliseconds: 2), () => engineFrame('after warm-up'));
  fa.elapse(const Duration(milliseconds: 3));
  log('after warm-up locked=${binding.locked} hasScheduledFrame=${binding.hasScheduledFrame}');
  pump(const Duration(milliseconds: 16));
  pump(const Duration(milliseconds: 16));
  // The engine path: onBeginFrame/onDrawFrame outside a warm-up frame.
  binding.scheduleFrameCallback((Duration ts) => log('engine transient ts=${us(ts)}'));
  ui.PlatformDispatcher.instance.onBeginFrame!(Duration(microseconds: epoch + fa.elapsed.inMicroseconds + 5));
  fa.flushMicrotasks();
  ui.PlatformDispatcher.instance.onDrawFrame!();
  t.dispose();
  pump(const Duration(milliseconds: 1));
  // A warm-up frame while a frame is scheduled (and nothing reschedules):
  // the frame is requested again after it.
  binding.scheduleFrame();
  binding.scheduleWarmUpFrame();
  fa.elapse(Duration.zero);
  log('second warm-up: hasScheduledFrame=${binding.hasScheduledFrame} locked=${binding.locked}');
  pump(const Duration(milliseconds: 1));
  ui.PlatformDispatcher.instance
    ..onBeginFrame = null
    ..onDrawFrame = null;
}

void sceneErrors() {
  log('== errors');
  final FlutterExceptionHandler? previous = FlutterError.onError;
  FlutterError.onError = (FlutterErrorDetails d) {
    log('error: ${d.exceptionAsString()} | library=${d.library} | context=${d.context?.toDescription()} | collector=${d.informationCollector != null}');
  };
  binding.scheduleFrameCallback((Duration ts) => throw StateError('transient boom'));
  binding.scheduleFrameCallback((Duration ts) => log('transient after boom'));
  var once = false;
  binding.addPersistentFrameCallback((Duration ts) {
    if (!once) {
      once = true;
      binding.addPostFrameCallback((Duration ts) => throw ArgumentError('post boom'));
      binding.addPostFrameCallback((Duration ts) => log('post after boom'));
      throw 'persistent boom';
    }
  });
  pump(const Duration(milliseconds: 1));
  binding.scheduleTask<void>(() => throw 'task boom', Priority.touch);
  fa.elapse(Duration.zero);
  void cb(List<ui.FrameTiming> timings) => throw 'timings boom';
  binding.addTimingsCallback(cb);
  binding.platformDispatcher.onReportTimings!(<ui.FrameTiming>[]);
  binding.removeTimingsCallback(cb);
  final Ticker t = Ticker((_) {})..start();
  try {
    binding.scheduleFrameCallback((_) {}, rescheduling: true);
  } on FlutterError catch (e) {
    log('rescheduling outside a callback: ${e.diagnostics.first}');
  }
  t.dispose();
  binding.scheduleFrameCallback((Duration ts) {
    final int id = binding.scheduleFrameCallback((_) => log('rescheduled ran'), rescheduling: true);
    log('rescheduling inside a callback ok, id>0=${id > 0}');
  });
  pump(const Duration(milliseconds: 1));
  pump(const Duration(milliseconds: 1));
  binding.scheduleFrameCallback((_) {});
  binding.debugAssertNoTransientCallbacks('leftover');
  binding.scheduleFrameCallback((_) {});
  binding.debugAssertNoTransientCallbacks('leftovers');
  pump(const Duration(milliseconds: 1));
  binding.debugAssertNoTransientCallbacks('none');
  FlutterError.onError = previous;
}

void sceneTimings() {
  log('== timings');
  void a(List<ui.FrameTiming> timings) {
    log('timings a ${timings.length} ${timings.map((ui.FrameTiming t) => t.toString()).join(' | ')}');
  }
  late void Function(List<ui.FrameTiming>) b;
  b = (List<ui.FrameTiming> timings) {
    log('timings b ${timings.length}');
    binding.removeTimingsCallback(a);
  };
  binding.addTimingsCallback(a);
  binding.addTimingsCallback(b);
  binding.addTimingsCallback(a);
  final List<ui.FrameTiming> timings = <ui.FrameTiming>[
    ui.FrameTiming(vsyncStart: 1000, buildStart: 1500, buildFinish: 9300, rasterStart: 9500, rasterFinish: 12345, rasterFinishWallTime: 99999, frameNumber: 7),
    ui.FrameTiming(vsyncStart: 0, buildStart: 1, buildFinish: 2, rasterStart: 3, rasterFinish: 4, rasterFinishWallTime: 5, layerCacheCount: 2, layerCacheBytes: 3145728, pictureCacheCount: 1, pictureCacheBytes: 1048577),
  ];
  log('${timings[1].layerCacheMegabytes} ${timings[1].pictureCacheMegabytes} ${timings[0].timestampInMicroseconds(ui.FramePhase.rasterFinishWallTime)}');
  binding.platformDispatcher.onReportTimings!(timings);
  binding.platformDispatcher.onReportTimings!(timings);
  binding.removeTimingsCallback(b);
  log('onReportTimings null=${binding.platformDispatcher.onReportTimings == null}');
}

void scenePerformanceMode() {
  log('== performance mode');
  final PerformanceModeRequestHandle? h1 = binding.requestPerformanceMode(ui.DartPerformanceMode.latency);
  final PerformanceModeRequestHandle? h2 = binding.requestPerformanceMode(ui.DartPerformanceMode.throughput);
  final PerformanceModeRequestHandle? h3 = binding.requestPerformanceMode(ui.DartPerformanceMode.latency);
  log('${h1 != null} ${h2 != null} ${h3 != null} ${binding.debugGetRequestedPerformanceMode()}');
  h1!.dispose();
  log('${binding.debugGetRequestedPerformanceMode()}');
  h3!.dispose();
  log('${binding.debugGetRequestedPerformanceMode()}');
  try {
    binding.debugAssertNoPendingPerformanceModeRequests('pending');
    log('no pending');
  } on FlutterError catch (e) {
    log('pending error $e');
  }
}

void sceneBanners() {
  log('== banners');
  final DebugPrintCallback previous = debugPrint;
  debugPrint = (String? message, {int? wrapWidth}) => log('debugPrint $message');
  debugPrintBeginFrameBanner = true;
  debugPrintEndFrameBanner = true;
  binding.scheduleFrame();
  pump(const Duration(milliseconds: 1));
  binding.scheduleFrame();
  pump(const Duration(days: 1, hours: 2, minutes: 3, seconds: 4, milliseconds: 5, microseconds: 6));
  binding.scheduleFrame();
  pump(const Duration(microseconds: 40));
  binding.scheduleFrame();
  pump(const Duration(microseconds: 843));
  binding.scheduleFrame();
  pump(const Duration(microseconds: 50));
  binding.handleBeginFrame(null);
  binding.handleDrawFrame();
  debugPrintBeginFrameBanner = false;
  binding.scheduleFrame();
  pump(const Duration(milliseconds: 2));
  debugPrintEndFrameBanner = false;
  SchedulerBinding.debugPrintTransientCallbackRegistrationStack();
  debugPrint = previous;
}

void main() {
  test('scheduler', () {
    FakeAsync().run((FakeAsync async) {
      fa = async;
      FlutterError.onError = (FlutterErrorDetails d) =>
          log('reported: ${d.exceptionAsString()} | ${d.library} | ${d.context?.toDescription()}');
      binding = TestBinding();
      log('constructed locked=${binding.locked} phase=${phase()} timeDilation=$timeDilation');
      sceneFrameOrder();
      sceneEndOfFrame();
      sceneTickers();
      sceneTimeDilation();
      sceneTasks();
      sceneLifecycle();
      sceneWarmUp();
      sceneErrors();
      sceneTimings();
      scenePerformanceMode();
      sceneBanners();
      fa.flushMicrotasks();
      log('end timers=${fa.pendingTimers.length} transient=${binding.transientCallbackCount}');
    });
    File('build/scheduler.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': entries}));
  });
}
