// Reference side of flutter-go's dart:async differential test: each scene
// logs the order in which callbacks run, inside package:fake_async (the zone
// flutter_test runs widget tests in). Output: build/async.json as
// {scene: [log lines]}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:fake_async/fake_async.dart';
import 'package:flutter_test/flutter_test.dart';

typedef Scene = void Function(FakeAsync fa, void Function(String) log);

final Map<String, Scene> scenes = <String, Scene>{
  'queue_order': (FakeAsync fa, void Function(String) log) {
    Timer.run(() => log('timer.run'));
    scheduleMicrotask(() => log('micro1'));
    Future<int>.value(1).then((int v) => log('value $v'));
    Future<int>(() => 2).then((int v) => log('future() $v'));
    Future<int>.microtask(() => 3).then((int v) => log('microtask $v'));
    scheduleMicrotask(() {
      log('micro2');
      scheduleMicrotask(() => log('micro2.nested'));
    });
    log('sync end');
    fa.flushMicrotasks();
    log('after flushMicrotasks');
    fa.elapse(Duration.zero);
    log('after elapse 0');
  },
  'completers': (FakeAsync fa, void Function(String) log) {
    final Completer<String> a = Completer<String>();
    final Completer<String> s = Completer<String>.sync();
    a.future.then((String v) => log('async $v'));
    s.future.then((String v) => log('sync $v'));
    a.complete('a');
    log('after a.complete');
    s.complete('s');
    log('after s.complete');
    s.future.then((String v) => log('late sync $v'));
    log('end');
    fa.flushMicrotasks();
  },
  'listener_order_and_chains': (FakeAsync fa, void Function(String) log) {
    final Completer<int> c = Completer<int>.sync();
    c.future.then((int v) => v + 1).then((int v) => log('chain1 $v'));
    c.future.then((int v) => log('direct $v'));
    c.future.then((int v) => Future<int>.value(v * 10)).then((int v) => log('chainFuture $v'));
    c.future.whenComplete(() => log('whenComplete'));
    c.complete(1);
    log('completed');
    fa.flushMicrotasks();
  },
  'async_functions': (FakeAsync fa, void Function(String) log) {
    Future<int> f(String name, Future<int> dep) async {
      log('$name start');
      final int v = await dep;
      log('$name resumed $v');
      return v + 1;
    }
    final Completer<int> gate = Completer<int>();
    f('a', Future<int>.value(1)).then((int v) => log('a result $v'));
    f('b', gate.future).then((int v) => log('b result $v'));
    final Future<int> done = Future<int>.value(7);
    fa.flushMicrotasks();
    f('c', done).then((int v) => log('c result $v'));
    log('before gate');
    gate.complete(5);
    log('after gate');
    fa.flushMicrotasks();
  },
  'async_nested': (FakeAsync fa, void Function(String) log) {
    Future<int> inner() async {
      log('inner start');
      await null;
      log('inner after await null');
      return 3;
    }
    Future<void> outer() async {
      log('outer start');
      final int v = await inner();
      log('outer got $v');
      await Future<void>.delayed(const Duration(milliseconds: 10));
      log('outer after delay');
    }
    outer().then((_) => log('outer done'));
    log('sync end');
    fa.flushMicrotasks();
    log('flushed');
    fa.elapse(const Duration(milliseconds: 9));
    log('elapsed 9');
    fa.elapse(const Duration(milliseconds: 1));
    log('elapsed 10');
  },
  'errors': (FakeAsync fa, void Function(String) log) {
    Future<int> early() async {
      throw StateError('early');
    }
    Future<int> late() async {
      await Future<void>.value();
      throw StateError('late');
    }
    early().catchError((Object e) { log('caught ${(e as StateError).message}'); return 0; });
    late().catchError((Object e) { log('caught ${(e as StateError).message}'); return 0; });
    Future<int>.error(StateError('passes')).then((int v) => v + 1).then((int v) => v).catchError((Object e) {
      log('caught after chain ${(e as StateError).message}');
      return -1;
    }).then((int v) => log('recovered $v'));
    Future<int>.value(1).then<int>((int v) => throw StateError('in then')).catchError((Object e) {
      log('caught ${(e as StateError).message}');
      return 0;
    });
    Future<int>.error(StateError('filtered')).catchError((Object e) {
      log('wrong handler');
      return 0;
    }, test: (Object e) => false).catchError((Object e) {
      log('second handler ${(e as StateError).message}');
      return 0;
    });
    Future<int> awaitsError() async {
      try {
        await Future<int>.error(StateError('awaited'));
      } on StateError catch (e) {
        log('try caught ${e.message}');
      }
      return 1;
    }
    awaitsError().then((int v) => log('awaitsError $v'));
    log('sync end');
    fa.flushMicrotasks();
  },
  'timers': (FakeAsync fa, void Function(String) log) {
    Timer(const Duration(milliseconds: 20), () => log('t20 a ${fa.elapsed.inMilliseconds}'));
    Timer(const Duration(milliseconds: 10), () {
      log('t10 ${fa.elapsed.inMilliseconds}');
      scheduleMicrotask(() => log('micro in t10'));
      Timer(const Duration(milliseconds: 10), () => log('t10+10 ${fa.elapsed.inMilliseconds}'));
    });
    Timer(const Duration(milliseconds: 20), () => log('t20 b ${fa.elapsed.inMilliseconds}'));
    final Timer cancelled = Timer(const Duration(milliseconds: 15), () => log('never'));
    Timer.periodic(const Duration(milliseconds: 7), (Timer t) {
      log('periodic tick ${t.tick} at ${fa.elapsed.inMilliseconds}');
      if (t.tick >= 4) {
        t.cancel();
      }
    });
    cancelled.cancel();
    fa.elapse(const Duration(milliseconds: 5));
    log('at 5 active=${cancelled.isActive}');
    fa.elapse(const Duration(milliseconds: 25));
    log('at ${fa.elapsed.inMilliseconds} pending=${fa.pendingTimers.length}');
  },
  'periodic_catch_up': (FakeAsync fa, void Function(String) log) {
    late final Timer p;
    p = Timer.periodic(const Duration(milliseconds: 10), (Timer t) {
      log('tick ${t.tick} at ${fa.elapsed.inMilliseconds}');
      if (t.tick == 1) {
        fa.elapseBlocking(const Duration(milliseconds: 35));
      }
      if (t.tick > 6) {
        t.cancel();
      }
    });
    fa.elapse(const Duration(milliseconds: 100));
    log('active=${p.isActive}');
  },
  'flush_timers': (FakeAsync fa, void Function(String) log) {
    Timer(const Duration(seconds: 2), () {
      log('t2 at ${fa.elapsed.inMilliseconds}');
      Timer(const Duration(seconds: 1), () => log('t3 at ${fa.elapsed.inMilliseconds}'));
    });
    fa.flushTimers();
    log('done at ${fa.elapsed.inMilliseconds}');
  },
};

void main() {
  test('async', () {
    final Map<String, List<String>> out = <String, List<String>>{};
    for (final MapEntry<String, Scene> e in scenes.entries) {
      final List<String> log = <String>[];
      FakeAsync().run((FakeAsync fa) {
        e.value(fa, log.add);
        fa.flushMicrotasks();
      });
      out[e.key] = log;
    }
    File('build/async.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
