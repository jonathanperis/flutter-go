// Reference side of flutter-go's GestureBinding differential test: a
// BindingBase with GestureBinding and SchedulerBinding in fake_async.
// Pointer data packets go through the binding: hit testing (targets added
// with pushed transforms, logging the transformed events they receive),
// the hit test kept per pointer, dispatch order, routing and the arena
// (close on down, sweep on up), cancelPointer, events queued while locked,
// errors from targets and from the router (library, context, event),
// pointer signal resolution, onHitTest, and resampling on the fake clock.
// Also the base recognizers: OneSequence/PrimaryPointer (deadline timer,
// slop rejection, buttons/devices filters), EagerGestureRecognizer,
// GestureArenaTeam (with and without a captain), invokeCallback's error
// report and its debugPrintRecognizerCallbacksTrace line. Identity hashes
// are masked. The Go twin is ../../gesture_binding_test.go.
// Output: build/gesture_binding.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:fake_async/fake_async.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> lines = <String>[];
void log(String s) => lines.add(s);

String mask(Object? o) => '$o'.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH');

String o(Offset v) => '${v.dx},${v.dy}';

String ev(PointerEvent e) =>
    '${e.toStringShort().split('#').first} p${e.pointer} pos=${o(e.position)} local=${o(e.localPosition)} '
    'delta=${o(e.delta)} localDelta=${o(e.localDelta)} t=${e.timeStamp.inMicroseconds} buttons=${e.buttons} '
    'kind=${e.kind.name} down=${e.down} device=${e.device}';

class Target implements HitTestTarget {
  Target(this.name);
  final String name;
  bool throwOnDown = false;

  @override
  void handleEvent(PointerEvent event, HitTestEntry entry) {
    log('$name.handleEvent ${ev(event)}');
    if (throwOnDown && event is PointerDownEvent) {
      throw StateError('$name failed');
    }
    if (event is PointerSignalEvent) {
      GestureBinding.instance.pointerSignalResolver.register(event, (PointerSignalEvent e) {
        log('$name resolved signal ${ev(e)}');
      });
    }
  }

  @override
  String toString() => 'Target($name)';
}

typedef HitTester = void Function(HitTestResult result, Offset position, int viewId);

class DiffGestureBinding extends BindingBase with GestureBinding, SchedulerBinding {
  HitTester? hitTester;
  FakeAsync? fake;

  @override
  void hitTestInView(HitTestResult result, Offset position, int viewId) {
    log('hitTestInView ${o(position)} view=$viewId dpr=${platformDispatcher.implicitView!.devicePixelRatio}');
    hitTester?.call(result, position, viewId);
    super.hitTestInView(result, position, viewId);
  }

  @override
  void handleEvent(PointerEvent event, HitTestEntry entry) {
    log('binding.handleEvent ${ev(event)}');
    super.handleEvent(event, entry);
  }

  @override
  SamplingClock? get debugSamplingClock => _FakeClock(fake!);
}

class _FakeClock implements SamplingClock {
  _FakeClock(this.fake);
  final FakeAsync fake;
  @override
  DateTime now() => fake.getClock(DateTime.utc(2015)).now();
  @override
  Stopwatch stopwatch() => fake.getClock(DateTime.utc(2015)).stopwatch();
}

ui.PointerData pd(
  ui.PointerChange change, {
  double x = 0,
  double y = 0,
  int pointer = 0,
  int device = 0,
  int buttons = 0,
  int ms = 0,
  ui.PointerDeviceKind kind = ui.PointerDeviceKind.touch,
  ui.PointerSignalKind? signal,
  double scrollX = 0,
  double scrollY = 0,
}) => ui.PointerData(
  change: change,
  physicalX: x,
  physicalY: y,
  pointerIdentifier: pointer,
  device: device,
  buttons: buttons,
  timeStamp: Duration(milliseconds: ms),
  kind: kind,
  signalKind: signal,
  scrollDeltaX: scrollX,
  scrollDeltaY: scrollY,
);

void send(List<ui.PointerData> data) {
  log('packet ${data.map((ui.PointerData d) => d.change.name).join(',')}');
  GestureBinding.instance.platformDispatcher.onPointerDataPacket!(ui.PointerDataPacket(data: data));
}

class Passive extends OneSequenceGestureRecognizer {
  Passive(this.name);
  final String name;
  @override
  void addPointer(PointerDownEvent event) {
    startTrackingPointer(event.pointer);
  }

  @override
  String get debugDescription => 'passive';
  @override
  void didStopTrackingLastPointer(int pointer) {
    log('$name.didStopTrackingLastPointer $pointer');
    resolve(GestureDisposition.rejected);
  }

  @override
  void handleEvent(PointerEvent event) {
    log('$name.handleEvent ${ev(event)}');
    if (event is PointerUpEvent || event is PointerCancelEvent) {
      stopTrackingPointer(event.pointer);
    }
  }

  @override
  void acceptGesture(int pointer) => log('$name.accept $pointer');
  @override
  void rejectGesture(int pointer) => log('$name.reject $pointer');
}

class Primary extends PrimaryPointerGestureRecognizer {
  Primary(
    this.name, {
    super.deadline,
    super.preAcceptSlopTolerance,
    super.postAcceptSlopTolerance,
    super.supportedDevices,
    super.allowedButtonsFilter,
  });
  final String name;
  GestureDisposition? resolveOnUp;

  @override
  void handlePrimaryPointer(PointerEvent event) {
    log('$name.handlePrimaryPointer ${ev(event)} state=${state.name}');
    if (event is PointerUpEvent && resolveOnUp != null) {
      resolve(resolveOnUp!);
    }
  }

  @override
  void didExceedDeadline() {
    log('$name.didExceedDeadline state=${state.name}');
    resolve(GestureDisposition.accepted);
  }

  @override
  void handleNonAllowedPointer(PointerDownEvent event) {
    log('$name.handleNonAllowedPointer ${ev(event)}');
    super.handleNonAllowedPointer(event);
  }

  @override
  void acceptGesture(int pointer) {
    super.acceptGesture(pointer);
    log('$name.accept $pointer state=${state.name}');
  }

  @override
  void rejectGesture(int pointer) {
    super.rejectGesture(pointer);
    log('$name.reject $pointer state=${state.name}');
  }

  @override
  void didStopTrackingLastPointer(int pointer) {
    log('$name.didStopTrackingLastPointer $pointer state=${state.name}');
    super.didStopTrackingLastPointer(pointer);
  }

  @override
  String get debugDescription => 'primary';
}

class _Adder implements HitTestTarget {
  _Adder(this.recognizers);
  final List<GestureRecognizer> recognizers;
  @override
  void handleEvent(PointerEvent event, HitTestEntry entry) {
    if (event is PointerDownEvent) {
      for (final r in recognizers) {
        r.addPointer(event);
      }
    }
  }
}

void main() {
  test('gesture binding', () {
    fakeAsync((FakeAsync async) {
      final binding = DiffGestureBinding()..fake = async;
      final a = Target('A');
      final b = Target('B');
      final c = Target('C');
      binding.hitTester = (HitTestResult result, Offset position, int viewId) {
        result.add(HitTestEntry(a));
        result.pushOffset(const Offset(-10, -20));
        result.add(HitTestEntry(b));
        result.pushTransform(Matrix4.diagonal3Values(2, 3, 1));
        result.add(HitTestEntry(c));
        result.popTransform();
        result.popTransform();
      };
      final errors = <FlutterErrorDetails>[];
      FlutterError.onError = errors.add;
      void flushErrors() {
        for (final d in errors) {
          log(
            'error ${d.exception} library=${d.library} context=${d.context} '
            'dispatcher=${d is FlutterErrorDetailsForPointerEventDispatcher} '
            '${d is FlutterErrorDetailsForPointerEventDispatcher ? 'event=${mask(d.event)} entry=${mask(d.hitTestEntry)}' : ''}',
          );
        }
        errors.clear();
      }

      final route = <PointerEvent>[];
      GestureBinding.instance.pointerRouter.addGlobalRoute(route.add);
      void flushRoute() {
        for (final e in route) {
          log('global route ${ev(e)}');
        }
        route.clear();
      }

      // 1. Down/move/up of one pointer, at a dpr of 1 (the implicit view).
      send(<ui.PointerData>[
        pd(ui.PointerChange.add, x: 5, y: 6, pointer: 1),
        pd(ui.PointerChange.down, x: 5, y: 6, pointer: 1, ms: 1),
        pd(ui.PointerChange.move, x: 7, y: 9, pointer: 1, ms: 2),
        pd(ui.PointerChange.up, x: 7, y: 9, pointer: 1, ms: 3),
      ]);
      flushRoute();
      flushErrors();

      // 2. Two pointers interleaved; hover and remove.
      send(<ui.PointerData>[
        pd(ui.PointerChange.down, x: 1, y: 1, pointer: 2, device: 1),
        pd(ui.PointerChange.down, x: 50, y: 60, pointer: 3, device: 2),
        pd(ui.PointerChange.move, x: 2, y: 3, pointer: 2, device: 1),
        pd(ui.PointerChange.up, x: 50, y: 60, pointer: 3, device: 2),
        pd(ui.PointerChange.cancel, x: 2, y: 3, pointer: 2, device: 1),
        pd(ui.PointerChange.hover, x: 8, y: 8, device: 5, kind: ui.PointerDeviceKind.mouse),
        pd(ui.PointerChange.remove, x: 8, y: 8, device: 5, kind: ui.PointerDeviceKind.mouse),
      ]);
      flushRoute();

      // 3. A target throws on down; the others still get the event.
      b.throwOnDown = true;
      send(<ui.PointerData>[
        pd(ui.PointerChange.down, x: 3, y: 4, pointer: 4),
        pd(ui.PointerChange.up, x: 3, y: 4, pointer: 4),
      ]);
      b.throwOnDown = false;
      flushErrors();
      flushRoute();

      // 4. A global route throws on an added event (no hit test).
      void bad(PointerEvent e) => throw StateError('route failed');
      GestureBinding.instance.pointerRouter.addGlobalRoute(bad);
      send(<ui.PointerData>[pd(ui.PointerChange.add, x: 1, y: 2, device: 9)]);
      GestureBinding.instance.pointerRouter.removeGlobalRoute(bad);
      flushErrors();
      flushRoute();

      // 5. Events while locked are queued until unlocked.
      binding.lockEvents(() async {
        send(<ui.PointerData>[
          pd(ui.PointerChange.down, x: 9, y: 9, pointer: 5),
          pd(ui.PointerChange.up, x: 9, y: 9, pointer: 5),
        ]);
        log('sent while locked');
        await Future<void>.delayed(const Duration(milliseconds: 5));
        log('unlocking');
      });
      async.elapse(const Duration(milliseconds: 10));
      flushRoute();

      // 6. cancelPointer from a route, with nothing queued.
      void canceler(PointerEvent e) {
        if (e is PointerDownEvent) {
          log('cancelPointer ${e.pointer}');
          binding.cancelPointer(e.pointer);
        }
      }

      GestureBinding.instance.pointerRouter.addGlobalRoute(canceler);
      send(<ui.PointerData>[pd(ui.PointerChange.down, x: 4, y: 4, pointer: 6)]);
      log('after packet');
      async.flushMicrotasks();
      GestureBinding.instance.pointerRouter.removeGlobalRoute(canceler);
      flushRoute();

      // 7. Pointer signals: resolved once, to the first registered target.
      send(<ui.PointerData>[
        pd(ui.PointerChange.add, x: 10, y: 10, device: 7, kind: ui.PointerDeviceKind.mouse),
        pd(
          ui.PointerChange.hover,
          x: 10,
          y: 10,
          device: 7,
          kind: ui.PointerDeviceKind.mouse,
          signal: ui.PointerSignalKind.scroll,
          scrollX: 3,
          scrollY: -4,
        ),
      ]);
      flushRoute();

      // 8. onHitTest.
      final ui.HitTestResponse response = binding.platformDispatcher.onHitTest!(
        ui.HitTestRequest(view: binding.platformDispatcher.implicitView!, offset: const Offset(3, 3)),
      );
      log('onHitTest hasPlatformView=${response.hasPlatformView}');
      flushRoute();

      // 9. Recognizers on the arena: a passive one in a team with an eager
      // one, a primary-pointer one with a deadline, competing.
      binding.hitTester = null;
      final team = GestureArenaTeam();
      final p1 = Passive('P1')..team = team;
      final eager = EagerGestureRecognizer();
      final prim = Primary('Prim', deadline: const Duration(milliseconds: 100));
      void downWith(List<GestureRecognizer> rs, int pointer, double x) {
        final down = PointerDownEvent(pointer: pointer, position: Offset(x, x));
        for (final r in rs) {
          r.addPointer(down);
        }
        binding.pointerRouter.route(down);
        binding.gestureArena.close(pointer);
        async.flushMicrotasks();
      }

      downWith(<GestureRecognizer>[p1, prim], 20, 5);
      log('elapse 50');
      async.elapse(const Duration(milliseconds: 50));
      log('elapse 50');
      async.elapse(const Duration(milliseconds: 50));
      log('elapse 10');
      async.elapse(const Duration(milliseconds: 10));
      binding.pointerRouter.route(const PointerUpEvent(pointer: 20, position: Offset(5, 5)));
      binding.gestureArena.sweep(20);
      log('prim state ${prim.state.name} ${mask(prim)}');

      // Captain: the team wins as its captain.
      final captainTeam = GestureArenaTeam();
      final p2 = Passive('P2')..team = captainTeam;
      final p3 = Passive('P3')..team = captainTeam;
      captainTeam.captain = p3;
      downWith(<GestureRecognizer>[p2, p3, eager], 21, 1);
      binding.pointerRouter.route(const PointerUpEvent(pointer: 21, position: Offset(1, 1)));
      binding.gestureArena.sweep(21);

      // Slop: a primary pointer recognizer rejected by a move beyond 18.
      final slop = Primary('Slop', preAcceptSlopTolerance: 18)..resolveOnUp = GestureDisposition.accepted;
      final other = Passive('P4');
      downWith(<GestureRecognizer>[slop, other], 22, 0);
      binding.pointerRouter.route(
        const PointerMoveEvent(pointer: 22, position: Offset(10, 10), delta: Offset(10, 10)),
      );
      binding.pointerRouter.route(
        const PointerMoveEvent(pointer: 22, position: Offset(13, 13), delta: Offset(3, 3)),
      );
      binding.pointerRouter.route(const PointerUpEvent(pointer: 22, position: Offset(13, 13)));
      binding.gestureArena.sweep(22);

      // Filters: devices and buttons.
      final filtered = Primary(
        'Filtered',
        supportedDevices: <PointerDeviceKind>{PointerDeviceKind.mouse},
        allowedButtonsFilter: (int buttons) => buttons == kSecondaryButton,
      );
      filtered.addPointer(const PointerDownEvent(pointer: 23));
      filtered.addPointer(
        const PointerDownEvent(pointer: 24, kind: PointerDeviceKind.mouse, buttons: kSecondaryButton),
      );
      log('filtered state ${filtered.state.name} primary=${filtered.primaryPointer}');
      log('filtered kind ${filtered.getKindForPointer(23).name} ${filtered.getButtonsForPointer(24)}');
      binding.gestureArena.close(23);
      binding.gestureArena.close(24);
      async.flushMicrotasks();

      // More filters: mouse with the primary button, touch with the
      // secondary one; a non-allowed pointer before and after acceptance.
      filtered.addPointer(const PointerDownEvent(pointer: 25, kind: PointerDeviceKind.mouse));
      filtered.addPointer(const PointerDownEvent(pointer: 26, buttons: kSecondaryButton));
      final f2 = Primary(
        'F2',
        supportedDevices: <PointerDeviceKind>{PointerDeviceKind.mouse},
        allowedButtonsFilter: (int buttons) => buttons == kSecondaryButton,
      );
      final f2rival = Passive('F2rival');
      final f2down = const PointerDownEvent(pointer: 27, kind: PointerDeviceKind.mouse, buttons: kSecondaryButton);
      f2.addPointer(f2down);
      f2rival.addPointer(f2down);
      f2.addPointer(const PointerDownEvent(pointer: 28));
      log('f2 state ${f2.state.name}');
      final f3 = Primary(
        'F3',
        supportedDevices: <PointerDeviceKind>{PointerDeviceKind.mouse},
      );
      f3.addPointer(const PointerDownEvent(pointer: 29, kind: PointerDeviceKind.mouse));
      binding.gestureArena.close(29);
      async.flushMicrotasks();
      f3.addPointer(const PointerDownEvent(pointer: 44, kind: PointerDeviceKind.mouse));
      final f3rival = Passive('F3rival')..addPointer(const PointerDownEvent(pointer: 44));
      f3.addPointer(const PointerDownEvent(pointer: 32));
      log('f3 state ${f3.state.name}');
      binding.gestureArena.close(44);
      binding.gestureArena.sweep(44);
      f3rival.dispose();

      // Cancel stops tracking a primary pointer.
      final canceled = Primary('Canceled');
      canceled.addPointer(const PointerDownEvent(pointer: 33));
      binding.pointerRouter.route(const PointerCancelEvent(pointer: 33));
      log('canceled state ${canceled.state.name}');
      binding.gestureArena.close(33);
      async.flushMicrotasks();

      // An up beyond the slop is not a move: handled as the primary pointer.
      final far = Primary('Far', preAcceptSlopTolerance: 18)..resolveOnUp = GestureDisposition.accepted;
      final farRival = Passive('FarRival');
      downWith(<GestureRecognizer>[far, farRival], 34, 0);
      binding.pointerRouter.route(const PointerUpEvent(pointer: 34, position: Offset(30, 30)));
      binding.gestureArena.sweep(34);

      // Accepted before its deadline: the timer is canceled.
      final early = Primary('Early', deadline: const Duration(milliseconds: 100));
      downWith(<GestureRecognizer>[early], 35, 0);
      async.elapse(const Duration(milliseconds: 150));
      binding.pointerRouter.route(const PointerUpEvent(pointer: 35));

      // A team member accepting on up: the captain wins.
      final cap = Passive('Cap');
      final tp = Primary('TP')..resolveOnUp = GestureDisposition.accepted;
      final team3 = GestureArenaTeam()..captain = cap;
      cap.team = team3;
      tp.team = team3;
      final out = Passive('Out');
      downWith(<GestureRecognizer>[out, tp, cap], 36, 0);
      binding.pointerRouter.route(const PointerUpEvent(pointer: 36));
      binding.gestureArena.sweep(36);
      // A team first in the arena wins the sweep: its captain gets it.
      final cap2 = Passive('Cap2');
      final p7 = Passive('P7');
      final team4 = GestureArenaTeam()..captain = cap2;
      cap2.team = team4;
      p7.team = team4;
      final out2 = Passive('Out2');
      final down37 = const PointerDownEvent(pointer: 37);
      p7.addPointer(down37);
      cap2.addPointer(down37);
      out2.addPointer(down37);
      binding.gestureArena.close(37);
      binding.gestureArena.sweep(37);
      binding.pointerRouter.route(const PointerUpEvent(pointer: 37));

      // resolvePointer, then resolve: the resolved pointer isn't resolved again.
      final multi = Passive('Multi');
      final multiRival = Passive('MultiRival');
      for (final int p in <int>[38, 39]) {
        final d = PointerDownEvent(pointer: p);
        multi.addPointer(d);
        multiRival.addPointer(d);
        binding.gestureArena.close(p);
      }
      multi.resolvePointer(38, GestureDisposition.rejected);
      multi.resolve(GestureDisposition.accepted);
      async.flushMicrotasks();
      for (final int p in <int>[38, 39]) {
        binding.pointerRouter.route(PointerUpEvent(pointer: p));
        binding.gestureArena.sweep(p);
      }

      // invokeCallback: the trace line and an error report.
      debugPrintRecognizerCallbacksTrace = true;
      final printed = <String>[];
      final DebugPrintCallback oldPrint = debugPrint;
      debugPrint = (String? m, {int? wrapWidth}) => printed.add(m!);
      final int? r1 = prim.invokeCallback<int>('onThing', () => 7, debugReport: () => 'report');
      final int? r2 = prim.invokeCallback<int>('onBad', () => throw StateError('bad callback'));
      prim.invokeCallback<void>('onEmpty', () {}, debugReport: () => '');
      debugPrintRecognizerCallbacksTrace = false;
      debugPrint = oldPrint;
      for (final p in printed) {
        log('print ${mask(p)}');
      }
      log('invokeCallback results $r1 $r2');
      for (final d in errors) {
        log('error ${d.exception} library=${d.library} context=${d.context}');
        log('  info ${d.informationCollector!().map(mask).join(' | ')}');
      }
      errors.clear();

      // 10. Resampling: touch events are resampled to the fake clock.
      binding.resamplingEnabled = true;
      binding.samplingOffset = const Duration(milliseconds: -5);
      binding.hitTester = (HitTestResult result, Offset position, int viewId) {
        result.add(HitTestEntry(a));
      };
      final int nowMs = DateTime.utc(2015).millisecondsSinceEpoch + async.elapsed.inMilliseconds;
      send(<ui.PointerData>[
        pd(ui.PointerChange.down, x: 0, pointer: 30, ms: nowMs + 1),
        pd(ui.PointerChange.move, x: 10, pointer: 30, ms: nowMs + 11),
        pd(ui.PointerChange.move, x: 20, pointer: 30, ms: nowMs + 21),
        pd(ui.PointerChange.up, x: 20, pointer: 30, ms: nowMs + 31),
      ]);
      for (var i = 0; i < 6; i++) {
        log('elapse 10 (frame ${binding.hasScheduledFrame})');
        async.elapse(const Duration(milliseconds: 10));
      }
      binding.resamplingEnabled = false;
      send(<ui.PointerData>[pd(ui.PointerChange.down, x: 1, pointer: 31, ms: 1)]);
      send(<ui.PointerData>[pd(ui.PointerChange.up, x: 1, pointer: 31, ms: 2)]);
      async.elapse(const Duration(milliseconds: 50));
      flushRoute();
      flushErrors();
      // 11. Recognizers added by a hit test target: the binding closes the
      // arena on down and sweeps it on up.
      final q1 = Passive('Q1');
      final q2 = Primary('Q2');
      binding.hitTester = (HitTestResult result, Offset position, int viewId) {
        result.add(HitTestEntry(_Adder(<GestureRecognizer>[q1, q2])));
      };
      send(<ui.PointerData>[pd(ui.PointerChange.down, x: 6, y: 6, pointer: 40)]);
      log('after down');
      async.flushMicrotasks();
      send(<ui.PointerData>[pd(ui.PointerChange.up, x: 6, y: 6, pointer: 40)]);
      log('after up');
      async.flushMicrotasks();
      final solo = Primary('Solo');
      binding.hitTester = (HitTestResult result, Offset position, int viewId) {
        result.add(HitTestEntry(_Adder(<GestureRecognizer>[solo])));
      };
      send(<ui.PointerData>[pd(ui.PointerChange.down, x: 6, y: 6, pointer: 41)]);
      log('solo before microtasks');
      async.flushMicrotasks();
      log('solo after microtasks');
      send(<ui.PointerData>[pd(ui.PointerChange.up, x: 6, y: 6, pointer: 41)]);

      // 12. cancelPointer outside a dispatch: flushed in a microtask.
      binding.hitTester = null;
      send(<ui.PointerData>[pd(ui.PointerChange.down, x: 2, y: 2, pointer: 42)]);
      binding.cancelPointer(42);
      log('cancel requested');
      async.flushMicrotasks();
      log('cancel flushed');

      // 13. Resampling a pointer that stays down between packets.
      binding.resamplingEnabled = true;
      final int now2 = DateTime.utc(2015).millisecondsSinceEpoch + async.elapsed.inMilliseconds;
      send(<ui.PointerData>[
        pd(ui.PointerChange.down, x: 0, pointer: 43, ms: now2 + 1),
        pd(ui.PointerChange.move, x: 6, pointer: 43, ms: now2 + 2),
      ]);
      for (var i = 0; i < 5; i++) {
        log('elapse 10');
        async.elapse(const Duration(milliseconds: 10));
      }
      send(<ui.PointerData>[pd(ui.PointerChange.move, x: 30, pointer: 43, ms: now2 + 55)]);
      for (var i = 0; i < 4; i++) {
        log('elapse 10');
        async.elapse(const Duration(milliseconds: 10));
      }
      send(<ui.PointerData>[pd(ui.PointerChange.up, x: 30, pointer: 43, ms: now2 + 95)]);
      for (var i = 0; i < 4; i++) {
        log('elapse 10');
        async.elapse(const Duration(milliseconds: 10));
      }
      binding.resamplingEnabled = false;
      flushRoute();

      // 14. Velocity trackers read the binding's sampling clock.
      final tracker = VelocityTracker.withKind(PointerDeviceKind.touch);
      tracker.addPosition(Duration.zero, Offset.zero);
      tracker.addPosition(const Duration(milliseconds: 10), const Offset(0, 10));
      tracker.addPosition(const Duration(milliseconds: 20), const Offset(0, 25));
      log('velocity now ${tracker.getVelocityEstimate()}');
      async.elapse(const Duration(milliseconds: 41));
      log('velocity after 41ms ${tracker.getVelocityEstimate()}');

      log('timers ${async.pendingTimers.length}');
    });
    File('build/gesture_binding.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'log': lines}));
  });
}
