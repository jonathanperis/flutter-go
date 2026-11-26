// Reference side of flutter-go's tap recognizers differential test: tap,
// double tap, multi tap, serial tap, long press and force press
// recognizers (with eager recognizers and teams competing) driven by
// scripted pointer sequences, as gesture_tester.dart drives them (events
// routed through GestureBinding.instance.pointerRouter, arenas closed and
// swept by hand, fake time), in the test binding. Every callback is logged
// with its details, plus the recognizer callback trace and the arena
// diagnostics (debugPrintRecognizerCallbacksTrace and
// debugPrintGestureArenaDiagnostics), recognizer toString and fake time.
// Identity hashes are masked. The Go twin (../recognizers_tap_test.go) is
// compared exactly. Output: build/recognizers_tap.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';

import 'package:fake_async/fake_async.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> lines = <String>[];
void log(String s) => lines.add(s);

String mask(Object? o) => '$o'.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH');

late FakeAsync fa;

String now() => '${fa.elapsed.inMicroseconds}';

void route(PointerEvent e) {
  GestureBinding.instance.pointerRouter.route(e);
  fa.flushMicrotasks();
}

// down is GestureBinding's dispatch of a down event: the hit test's
// recognizers add the pointer, then the router routes it and the arena
// closes.
void down(List<GestureRecognizer> rs, PointerDownEvent e) {
  log('[${now()}] down ${e.pointer} ${e.position} buttons=${e.buttons} kind=${e.kind.name}');
  for (final r in rs) {
    r.addPointer(e);
  }
  route(e);
  GestureBinding.instance.gestureArena.close(e.pointer);
  fa.flushMicrotasks();
}

void move(PointerMoveEvent e) {
  log('[${now()}] move ${e.pointer} ${e.position} buttons=${e.buttons}');
  route(e);
}

// up routes the up event and sweeps the arena, as GestureBinding does.
void up(PointerUpEvent e) {
  log('[${now()}] up ${e.pointer} ${e.position}');
  route(e);
  GestureBinding.instance.gestureArena.sweep(e.pointer);
  fa.flushMicrotasks();
}

void cancel(PointerCancelEvent e) {
  log('[${now()}] cancel ${e.pointer}');
  route(e);
}

void elapse(int ms) {
  fa.elapse(Duration(milliseconds: ms));
  log('[${now()}] elapsed $ms');
}

void show(String label, Object o) => log('$label: ${mask(o)}');

TapGestureRecognizer tapRec(String name, {int buttons = 7, Object? owner}) {
  final r = TapGestureRecognizer(debugOwner: owner);
  if (buttons & 1 != 0) {
    r
      ..onTapDown = (TapDownDetails d) { log('$name onTapDown ${mask(d)}'); }
      ..onTapUp = (TapUpDetails d) { log('$name onTapUp ${mask(d)}'); }
      ..onTap = () { log('$name onTap'); }
      ..onTapMove = (TapMoveDetails d) {
        log('$name onTapMove ${d.globalPosition} ${d.localPosition} ${d.delta} ${d.kind.name}');
      }
      ..onTapCancel = () { log('$name onTapCancel'); };
  }
  if (buttons & 2 != 0) {
    r
      ..onSecondaryTapDown = (TapDownDetails d) { log('$name onSecondaryTapDown ${mask(d)}'); }
      ..onSecondaryTapUp = (TapUpDetails d) { log('$name onSecondaryTapUp ${mask(d)}'); }
      ..onSecondaryTap = () { log('$name onSecondaryTap'); }
      ..onSecondaryTapCancel = () { log('$name onSecondaryTapCancel'); };
  }
  if (buttons & 4 != 0) {
    r
      ..onTertiaryTapDown = (TapDownDetails d) { log('$name onTertiaryTapDown ${mask(d)}'); }
      ..onTertiaryTapUp = (TapUpDetails d) { log('$name onTertiaryTapUp ${mask(d)}'); }
      ..onTertiaryTapCancel = () { log('$name onTertiaryTapCancel'); };
  }
  return r;
}

DoubleTapGestureRecognizer doubleTapRec(String name) {
  return DoubleTapGestureRecognizer()
    ..onDoubleTapDown = (TapDownDetails d) { log('$name onDoubleTapDown ${mask(d)}'); }
    ..onDoubleTap = () { log('$name onDoubleTap'); }
    ..onDoubleTapCancel = () { log('$name onDoubleTapCancel'); };
}

LongPressGestureRecognizer longPressRec(String name, {Duration? duration, double? postAcceptSlopTolerance}) {
  final r = LongPressGestureRecognizer(duration: duration, postAcceptSlopTolerance: postAcceptSlopTolerance);
  for (final String b in <String>['', 'Secondary', 'Tertiary']) {
    void l(String cb, [Object? d]) => log('$name on${b}LongPress$cb${d == null ? '' : ' ${mask(d)}'}');
    switch (b) {
      case '':
        r
          ..onLongPressDown = (LongPressDownDetails d) { l('Down', d); }
          ..onLongPressCancel = () { l('Cancel'); }
          ..onLongPress = () { l(''); }
          ..onLongPressStart = (LongPressStartDetails d) { l('Start', d); }
          ..onLongPressMoveUpdate = (LongPressMoveUpdateDetails d) { l('MoveUpdate', d); }
          ..onLongPressUp = () { l('Up'); }
          ..onLongPressEnd = (LongPressEndDetails d) { l('End', d); };
      case 'Secondary':
        r
          ..onSecondaryLongPressDown = (LongPressDownDetails d) { l('Down', d); }
          ..onSecondaryLongPressCancel = () { l('Cancel'); }
          ..onSecondaryLongPress = () { l(''); }
          ..onSecondaryLongPressStart = (LongPressStartDetails d) { l('Start', d); }
          ..onSecondaryLongPressMoveUpdate = (LongPressMoveUpdateDetails d) { l('MoveUpdate', d); }
          ..onSecondaryLongPressUp = () { l('Up'); }
          ..onSecondaryLongPressEnd = (LongPressEndDetails d) { l('End', d); };
      case 'Tertiary':
        r
          ..onTertiaryLongPressDown = (LongPressDownDetails d) { l('Down', d); }
          ..onTertiaryLongPressCancel = () { l('Cancel'); }
          ..onTertiaryLongPress = () { l(''); }
          ..onTertiaryLongPressStart = (LongPressStartDetails d) { l('Start', d); }
          ..onTertiaryLongPressMoveUpdate = (LongPressMoveUpdateDetails d) { l('MoveUpdate', d); }
          ..onTertiaryLongPressUp = () { l('Up'); }
          ..onTertiaryLongPressEnd = (LongPressEndDetails d) { l('End', d); };
    }
  }
  return r;
}

MultiTapGestureRecognizer multiTapRec(String name, {Duration longTapDelay = Duration.zero}) {
  return MultiTapGestureRecognizer(longTapDelay: longTapDelay)
    ..onTapDown = (int p, TapDownDetails d) { log('$name onTapDown $p ${mask(d)}'); }
    ..onTapUp = (int p, TapUpDetails d) { log('$name onTapUp $p ${mask(d)}'); }
    ..onTap = (int p) { log('$name onTap $p'); }
    ..onTapCancel = (int p) { log('$name onTapCancel $p'); }
    ..onLongTapDown = (int p, TapDownDetails d) { log('$name onLongTapDown $p ${mask(d)}'); };
}

SerialTapGestureRecognizer serialTapRec(String name) {
  return SerialTapGestureRecognizer()
    ..onSerialTapDown = (SerialTapDownDetails d) { log('$name onSerialTapDown ${mask(d)}'); }
    ..onSerialTapUp = (SerialTapUpDetails d) { log('$name onSerialTapUp ${mask(d)}'); }
    ..onSerialTapCancel = (SerialTapCancelDetails d) { log('$name onSerialTapCancel ${mask(d)}'); };
}

ForcePressGestureRecognizer forcePressRec(String name) {
  return ForcePressGestureRecognizer()
    ..onStart = (ForcePressDetails d) { log('$name onStart ${mask(d)}'); }
    ..onPeak = (ForcePressDetails d) { log('$name onPeak ${mask(d)}'); }
    ..onUpdate = (ForcePressDetails d) { log('$name onUpdate ${mask(d)}'); }
    ..onEnd = (ForcePressDetails d) { log('$name onEnd ${mask(d)}'); };
}

PointerDownEvent pdown(int pointer, double x, double y,
    {int buttons = kPrimaryButton, PointerDeviceKind kind = PointerDeviceKind.touch, int ms = 0,
    double pressure = 1, double pressureMax = 1}) {
  return PointerDownEvent(
    pointer: pointer, position: Offset(x, y), buttons: buttons, kind: kind,
    timeStamp: Duration(milliseconds: ms), pressure: pressure, pressureMax: pressureMax,
  );
}

PointerMoveEvent pmove(int pointer, double x, double y,
    {int buttons = kPrimaryButton, PointerDeviceKind kind = PointerDeviceKind.touch, int ms = 0,
    double pressure = 1, double pressureMax = 1, double dx = 0, double dy = 0}) {
  return PointerMoveEvent(
    pointer: pointer, position: Offset(x, y), buttons: buttons, kind: kind,
    timeStamp: Duration(milliseconds: ms), pressure: pressure, pressureMax: pressureMax,
    delta: Offset(dx, dy),
  );
}

PointerUpEvent pup(int pointer, double x, double y,
    {PointerDeviceKind kind = PointerDeviceKind.touch, int ms = 0}) {
  return PointerUpEvent(pointer: pointer, position: Offset(x, y), kind: kind, timeStamp: Duration(milliseconds: ms));
}

PointerCancelEvent pcancel(int pointer) => PointerCancelEvent(pointer: pointer);

void scene(String name, void Function() body) {
  log('== $name');
  body();
  fa.flushMicrotasks();
}

void scenes() {
  scene('tap alone', () {
    final t = tapRec('tap');
    show('before', t);
    down(<GestureRecognizer>[t], pdown(1, 10, 10));
    show('after down', t);
    move(pmove(1, 12, 13, dx: 2, dy: 3));
    up(pup(1, 12, 13));
    show('after up', t);
    t.dispose();
  });
  scene('tap vs long press: deadline then up', () {
    final t = tapRec('tap');
    final lp = longPressRec('lp');
    down(<GestureRecognizer>[t, lp], pdown(1, 10, 10));
    elapse(50);
    elapse(60);
    show('tap after deadline', t);
    show('lp', lp);
    up(pup(1, 10, 10));
    t.dispose();
    lp.dispose();
  });
  scene('tap vs long press: long press wins', () {
    final t = tapRec('tap');
    final lp = longPressRec('lp');
    down(<GestureRecognizer>[t, lp], pdown(1, 10, 10, ms: 0));
    elapse(300);
    move(pmove(1, 11, 10, ms: 300, dx: 1));
    elapse(250);
    move(pmove(1, 20, 15, ms: 560, dx: 9, dy: 5));
    move(pmove(1, 40, 25, ms: 580, dx: 20, dy: 10));
    move(pmove(1, 60, 35, ms: 600, dx: 20, dy: 10));
    up(pup(1, 60, 35, ms: 610));
    t.dispose();
    lp.dispose();
  });
  scene('long press too short', () {
    final lp = longPressRec('lp');
    final t = tapRec('tap');
    down(<GestureRecognizer>[lp, t], pdown(1, 5, 5));
    elapse(300);
    up(pup(1, 5, 5));
    elapse(500);
    lp.dispose();
    t.dispose();
  });
  scene('tap slop rejection before deadline', () {
    final t = tapRec('tap');
    final lp = longPressRec('lp');
    down(<GestureRecognizer>[t, lp], pdown(1, 10, 10));
    move(pmove(1, 35, 10, dx: 25));
    elapse(600);
    up(pup(1, 35, 10));
    t.dispose();
    lp.dispose();
  });
  scene('tap won then slop: spontaneous cancel', () {
    final t = tapRec('tap');
    down(<GestureRecognizer>[t], pdown(1, 10, 10));
    move(pmove(1, 10, 40, dy: 30));
    up(pup(1, 10, 40));
    t.dispose();
  });
  scene('tap after deadline then slop: forced cancel', () {
    final t = tapRec('tap');
    final t2 = tapRec('tap2');
    down(<GestureRecognizer>[t, t2], pdown(1, 10, 10));
    elapse(150);
    move(pmove(1, 10, 40, dy: 30));
    up(pup(1, 10, 40));
    t.dispose();
    t2.dispose();
  });
  scene('two taps: first wins on sweep', () {
    final a = tapRec('a');
    final b = tapRec('b');
    down(<GestureRecognizer>[a, b], pdown(1, 10, 10));
    up(pup(1, 10, 10));
    a.dispose();
    b.dispose();
  });
  scene('secondary and tertiary', () {
    final t = tapRec('tap');
    final t2 = tapRec('tap2');
    down(<GestureRecognizer>[t, t2], pdown(1, 10, 10, buttons: kSecondaryButton));
    elapse(120);
    up(pup(1, 10, 10));
    down(<GestureRecognizer>[t, t2], pdown(2, 20, 20, buttons: kTertiaryButton));
    up(pup(2, 20, 20));
    down(<GestureRecognizer>[t], pdown(3, 20, 20, buttons: kSecondaryButton | kTertiaryButton));
    up(pup(3, 20, 20));
    final onlyPrimary = tapRec('primary', buttons: 1);
    down(<GestureRecognizer>[onlyPrimary, t], pdown(4, 30, 30, buttons: kSecondaryButton));
    up(pup(4, 30, 30));
    t.dispose();
    t2.dispose();
    onlyPrimary.dispose();
  });
  scene('buttons change during press', () {
    final t = tapRec('tap');
    final lp = longPressRec('lp');
    down(<GestureRecognizer>[t, lp], pdown(1, 10, 10));
    elapse(150);
    move(pmove(1, 10, 10, buttons: kPrimaryButton | kSecondaryButton));
    up(pup(1, 10, 10));
    t.dispose();
    lp.dispose();
  });
  scene('cancel event', () {
    final t = tapRec('tap');
    final lp = longPressRec('lp');
    down(<GestureRecognizer>[t, lp], pdown(1, 10, 10));
    elapse(150);
    cancel(pcancel(1));
    elapse(500);
    t.dispose();
    lp.dispose();
  });
  scene('dispose while won', () {
    final t = tapRec('tap');
    down(<GestureRecognizer>[t], pdown(1, 10, 10));
    t.dispose();
    up(pup(1, 10, 10));
  });
  scene('double tap', () {
    final dt = doubleTapRec('dt');
    final t = tapRec('tap');
    down(<GestureRecognizer>[dt, t], pdown(1, 10, 10));
    elapse(10);
    up(pup(1, 11, 10));
    elapse(100);
    show('dt between', dt);
    down(<GestureRecognizer>[dt, t], pdown(2, 12, 12));
    elapse(10);
    up(pup(2, 12, 12));
    elapse(400);
    dt.dispose();
    t.dispose();
  });
  scene('double tap timeout', () {
    final dt = doubleTapRec('dt');
    final t = tapRec('tap');
    down(<GestureRecognizer>[dt, t], pdown(1, 10, 10));
    up(pup(1, 10, 10));
    elapse(400);
    down(<GestureRecognizer>[dt, t], pdown(2, 10, 10));
    up(pup(2, 10, 10));
    elapse(400);
    dt.dispose();
    t.dispose();
  });
  scene('double tap: second tap too soon, too far, moved', () {
    final dt = doubleTapRec('dt');
    down(<GestureRecognizer>[dt], pdown(1, 10, 10));
    up(pup(1, 10, 10));
    elapse(20);
    down(<GestureRecognizer>[dt], pdown(2, 10, 10));
    up(pup(2, 10, 10));
    elapse(100);
    down(<GestureRecognizer>[dt], pdown(3, 200, 200));
    up(pup(3, 200, 200));
    elapse(50);
    down(<GestureRecognizer>[dt], pdown(4, 12, 10));
    move(pmove(4, 40, 10, dx: 28));
    up(pup(4, 40, 10));
    elapse(400);
    dt.dispose();
  });
  scene('double tap: cancel second and buttons mismatch', () {
    final dt = doubleTapRec('dt');
    down(<GestureRecognizer>[dt], pdown(1, 10, 10));
    up(pup(1, 10, 10));
    elapse(100);
    down(<GestureRecognizer>[dt], pdown(2, 10, 10));
    cancel(pcancel(2));
    elapse(100);
    down(<GestureRecognizer>[dt], pdown(3, 10, 10));
    up(pup(3, 10, 10));
    elapse(100);
    final DoubleTapGestureRecognizer dt2 = DoubleTapGestureRecognizer(allowedButtonsFilter: (int b) => true)
      ..onDoubleTap = () { log('dt2 onDoubleTap'); }
      ..onDoubleTapCancel = () { log('dt2 onDoubleTapCancel'); };
    down(<GestureRecognizer>[dt2], pdown(4, 10, 10));
    up(pup(4, 10, 10));
    elapse(100);
    down(<GestureRecognizer>[dt2], pdown(5, 10, 10, buttons: kSecondaryButton));
    up(pup(5, 10, 10));
    elapse(400);
    dt.dispose();
    dt2.dispose();
  });
  scene('double tap: two pointers in first tap', () {
    final dt = doubleTapRec('dt');
    down(<GestureRecognizer>[dt], pdown(1, 10, 10));
    down(<GestureRecognizer>[dt], pdown(2, 12, 10));
    up(pup(1, 10, 10));
    up(pup(2, 12, 10));
    elapse(400);
    dt.dispose();
  });
  scene('eager vs tap', () {
    final e = EagerGestureRecognizer();
    final t = tapRec('tap');
    down(<GestureRecognizer>[t, e], pdown(1, 10, 10));
    up(pup(1, 10, 10));
    e.dispose();
    t.dispose();
  });
  scene('team with captain', () {
    final team = GestureArenaTeam();
    final t = tapRec('tap');
    final lp = longPressRec('lp');
    final other = tapRec('other');
    t.team = team;
    lp.team = team;
    team.captain = lp;
    down(<GestureRecognizer>[t, lp, other], pdown(1, 10, 10));
    elapse(120);
    up(pup(1, 10, 10));
    t.dispose();
    lp.dispose();
    other.dispose();
  });
  scene('team without captain, long press accepts', () {
    final team = GestureArenaTeam();
    final t = tapRec('tap');
    final lp = longPressRec('lp');
    final other = tapRec('other');
    t.team = team;
    lp.team = team;
    down(<GestureRecognizer>[t, lp, other], pdown(1, 10, 10));
    elapse(600);
    up(pup(1, 10, 10));
    t.dispose();
    lp.dispose();
    other.dispose();
  });
  scene('long press secondary, tertiary, custom duration, post-accept slop', () {
    final lp = longPressRec('lp', duration: const Duration(milliseconds: 200));
    final t = tapRec('tap');
    down(<GestureRecognizer>[lp, t], pdown(1, 10, 10, buttons: kSecondaryButton, ms: 0));
    elapse(250);
    move(pmove(1, 15, 10, buttons: kSecondaryButton, ms: 260, dx: 5));
    up(pup(1, 15, 10, ms: 270));
    down(<GestureRecognizer>[lp], pdown(2, 10, 10, buttons: kTertiaryButton, ms: 300));
    elapse(250);
    up(pup(2, 10, 10, ms: 560));
    final lp2 = longPressRec('lp2', postAcceptSlopTolerance: 5);
    down(<GestureRecognizer>[lp2], pdown(3, 10, 10, ms: 600));
    elapse(600);
    move(pmove(3, 14, 10, ms: 1210, dx: 4));
    move(pmove(3, 30, 10, ms: 1220, dx: 16));
    up(pup(3, 30, 10, ms: 1230));
    show('lp2', lp2);
    lp.dispose();
    lp2.dispose();
    t.dispose();
  });
  scene('long press buttons change and slop', () {
    final lp = longPressRec('lp');
    down(<GestureRecognizer>[lp], pdown(1, 10, 10));
    move(pmove(1, 10, 10, buttons: 3));
    elapse(600);
    down(<GestureRecognizer>[lp], pdown(2, 10, 10));
    move(pmove(2, 40, 10, dx: 30));
    elapse(600);
    up(pup(2, 40, 10));
    down(<GestureRecognizer>[lp], pdown(3, 10, 10));
    elapse(600);
    move(pmove(3, 10, 10, buttons: 3));
    up(pup(3, 10, 10));
    lp.dispose();
  });
  scene('multitap', () {
    final m = multiTapRec('m', longTapDelay: const Duration(milliseconds: 300));
    down(<GestureRecognizer>[m], pdown(1, 10, 10));
    down(<GestureRecognizer>[m], pdown(2, 50, 50));
    move(pmove(1, 15, 12, dx: 5, dy: 2));
    elapse(350);
    up(pup(1, 15, 12));
    move(pmove(2, 90, 50, dx: 40));
    up(pup(2, 90, 50));
    down(<GestureRecognizer>[m], pdown(3, 10, 10, kind: PointerDeviceKind.mouse));
    move(pmove(3, 12, 10, kind: PointerDeviceKind.mouse, dx: 2));
    up(pup(3, 12, 10, kind: PointerDeviceKind.mouse));
    final t = tapRec('tap');
    down(<GestureRecognizer>[t, m], pdown(4, 10, 10));
    cancel(pcancel(4));
    down(<GestureRecognizer>[m], pdown(5, 10, 10));
    m.dispose();
    t.dispose();
  });
  scene('serial tap', () {
    final s = serialTapRec('s');
    final t = tapRec('tap');
    for (var i = 1; i <= 3; i++) {
      down(<GestureRecognizer>[s, t], pdown(i, 10, 10));
      up(pup(i, 10, 10));
      elapse(100);
    }
    elapse(300);
    down(<GestureRecognizer>[s], pdown(4, 10, 10));
    up(pup(4, 10, 10));
    elapse(50);
    down(<GestureRecognizer>[s], pdown(5, 10, 10));
    move(pmove(5, 40, 10, dx: 30));
    elapse(50);
    down(<GestureRecognizer>[s], pdown(6, 10, 10));
    up(pup(6, 10, 10));
    elapse(10);
    down(<GestureRecognizer>[s], pdown(7, 10, 10));
    up(pup(7, 10, 10));
    elapse(100);
    down(<GestureRecognizer>[s], pdown(8, 300, 10));
    cancel(pcancel(8));
    s.dispose();
    t.dispose();
  });
  scene('force press', () {
    final f = forcePressRec('f');
    final t = tapRec('tap');
    down(<GestureRecognizer>[f, t], pdown(1, 10, 10, pressure: 0, pressureMax: 6));
    move(pmove(1, 10, 10, pressure: 1, pressureMax: 6));
    move(pmove(1, 10, 10, pressure: 3, pressureMax: 6));
    move(pmove(1, 11, 10, pressure: 5.5, pressureMax: 6, dx: 1));
    move(pmove(1, 11, 10, pressure: 6, pressureMax: 6));
    up(pup(1, 11, 10));
    down(<GestureRecognizer>[f, t], pdown(2, 10, 10, pressure: 0, pressureMax: 1));
    up(pup(2, 10, 10));
    down(<GestureRecognizer>[f], pdown(3, 10, 10, pressure: 0, pressureMax: 6));
    move(pmove(3, 10, 10, pressure: 2, pressureMax: 6));
    move(pmove(3, 10, 10, pressure: 4, pressureMax: 6));
    up(pup(3, 10, 10));
    down(<GestureRecognizer>[f, t], pdown(4, 10, 10, pressure: 0, pressureMax: 6));
    move(pmove(4, 30, 10, pressure: 1, pressureMax: 6, dx: 5));
    up(pup(4, 30, 10));
    down(<GestureRecognizer>[f], pdown(5, 10, 10, pressure: 7, pressureMax: 6));
    up(pup(5, 10, 10));
    f.dispose();
    t.dispose();
  });
  scene('supported devices and slops', () {
    final t = TapGestureRecognizer(supportedDevices: <PointerDeviceKind>{PointerDeviceKind.mouse})
      ..onTap = () { log('mouseOnly onTap'); }
      ..onTapCancel = () { log('mouseOnly onTapCancel'); };
    final t2 = tapRec('any');
    down(<GestureRecognizer>[t, t2], pdown(1, 10, 10));
    up(pup(1, 10, 10));
    down(<GestureRecognizer>[t, t2], pdown(2, 10, 10, kind: PointerDeviceKind.mouse));
    up(pup(2, 10, 10, kind: PointerDeviceKind.mouse));
    final loose = TapGestureRecognizer(preAcceptSlopTolerance: null, postAcceptSlopTolerance: 50)
      ..onTap = () { log('loose onTap'); }
      ..onTapCancel = () { log('loose onTapCancel'); };
    final other = tapRec('other');
    down(<GestureRecognizer>[loose, other], pdown(3, 10, 10));
    move(pmove(3, 100, 10, dx: 90));
    up(pup(3, 100, 10));
    final tight = TapGestureRecognizer(preAcceptSlopTolerance: 2, postAcceptSlopTolerance: 3)
      ..onTap = () { log('tight onTap'); }
      ..onTapCancel = () { log('tight onTapCancel'); }
      ..onTapDown = (TapDownDetails d) { log('tight onTapDown'); };
    down(<GestureRecognizer>[tight], pdown(4, 10, 10));
    move(pmove(4, 12, 11, dx: 2, dy: 1));
    move(pmove(4, 14, 11, dx: 2));
    up(pup(4, 14, 11));
    show('tight', tight);
    t.dispose();
    t2.dispose();
    loose.dispose();
    other.dispose();
    tight.dispose();
  });
  scene('multiple pointers on tap', () {
    final t = tapRec('tap');
    final lp = longPressRec('lp');
    down(<GestureRecognizer>[t, lp], pdown(1, 10, 10));
    down(<GestureRecognizer>[t, lp], pdown(2, 20, 20));
    show('tap two pointers', t);
    up(pup(2, 20, 20));
    elapse(120);
    up(pup(1, 10, 10));
    t.dispose();
    lp.dispose();
  });
  scene('secondary-only callbacks, move with secondary button', () {
    final s = TapGestureRecognizer()..onSecondaryTap = () { log('s onSecondaryTap'); };
    down(<GestureRecognizer>[s], pdown(1, 10, 10, buttons: kSecondaryButton));
    up(pup(1, 10, 10));
    final t = tapRec('tap');
    down(<GestureRecognizer>[t], pdown(2, 10, 10, buttons: kSecondaryButton));
    move(pmove(2, 12, 10, buttons: kSecondaryButton, dx: 2));
    up(pup(2, 12, 10));
    s.dispose();
    t.dispose();
  });
  scene('transformed tap', () {
    final t = tapRec('tap');
    final m = Matrix4.translationValues(5, -3, 0);
    final PointerDownEvent d = pdown(1, 10, 10).transformed(m) as PointerDownEvent;
    down(<GestureRecognizer>[t], d);
    move(pmove(1, 12, 11, dx: 2, dy: 1));
    up(pup(1, 12, 11));
    t.dispose();
  });
  scene('double tap within double-tap slop, multitap without long tap', () {
    final dt = doubleTapRec('dt');
    down(<GestureRecognizer>[dt], pdown(1, 10, 10));
    up(pup(1, 10, 10));
    elapse(100);
    down(<GestureRecognizer>[dt], pdown(2, 70, 10));
    up(pup(2, 70, 10));
    elapse(400);
    dt.dispose();
    final m = multiTapRec('m0');
    down(<GestureRecognizer>[m], pdown(3, 10, 10));
    elapse(1000);
    up(pup(3, 10, 10));
    m.dispose();
  });
  scene('serial taps with different buttons', () {
    final SerialTapGestureRecognizer s = serialTapRec('s');
    down(<GestureRecognizer>[s], pdown(1, 10, 10));
    up(pup(1, 10, 10));
    elapse(100);
    down(<GestureRecognizer>[s], pdown(2, 10, 10, buttons: kSecondaryButton));
    up(pup(2, 10, 10));
    elapse(400);
    s.dispose();
  });
  scene('long press with synthesized moves', () {
    final lp = longPressRec('lp');
    down(<GestureRecognizer>[lp], pdown(1, 10, 10, ms: 0));
    elapse(600);
    move(PointerMoveEvent(pointer: 1, position: const Offset(30, 10), delta: const Offset(20, 0),
        timeStamp: const Duration(milliseconds: 610), synthesized: true));
    move(pmove(1, 31, 10, ms: 620, dx: 1));
    move(pmove(1, 33, 10, ms: 630, dx: 2));
    up(pup(1, 33, 10, ms: 640));
    lp.dispose();
  });
  scene('force press thresholds and rejection', () {
    final f = forcePressRec('f');
    final t = tapRec('tap');
    down(<GestureRecognizer>[f, t], pdown(1, 10, 10, pressure: 0, pressureMax: 6));
    move(pmove(1, 10, 10, pressure: 2.7, pressureMax: 6));
    move(pmove(1, 10, 10, pressure: 3.2, pressureMax: 6));
    up(pup(1, 10, 10));
    down(<GestureRecognizer>[f, t], pdown(2, 10, 10, pressure: 0, pressureMax: 6));
    up(pup(2, 10, 10));
    f.dispose();
    t.dispose();
  });
  scene('details defaults', () {
    show('TapDownDetails', TapDownDetails());
    show('TapUpDetails', TapUpDetails(kind: PointerDeviceKind.mouse, globalPosition: const Offset(1, 2)));
    show('LongPressDownDetails', const LongPressDownDetails());
    show('LongPressStartDetails', const LongPressStartDetails(globalPosition: Offset(3, 4)));
    show('LongPressMoveUpdateDetails', const LongPressMoveUpdateDetails(offsetFromOrigin: Offset(1, 1)));
    show('LongPressEndDetails', const LongPressEndDetails(velocity: Velocity(pixelsPerSecond: Offset(5, 6))));
    show('SerialTapDownDetails', SerialTapDownDetails(kind: PointerDeviceKind.stylus));
    show('SerialTapUpDetails', SerialTapUpDetails(count: 3));
    show('SerialTapCancelDetails', SerialTapCancelDetails(count: 2));
    show('ForcePressDetails', ForcePressDetails(globalPosition: const Offset(1, 1), pressure: 0.5));
    final t = TapGestureRecognizer(debugOwner: 'owner');
    show('tap with owner', t);
    log('descriptions ${t.debugDescription} ${doubleTapRec('x').debugDescription} '
        '${multiTapRec('x').debugDescription} ${serialTapRec('x').debugDescription} '
        '${longPressRec('x').debugDescription} ${forcePressRec('x').debugDescription}');
    t.dispose();
  });
}

void main() {
  testWidgets('recognizers tap', (WidgetTester tester) async {
    final DebugPrintCallback previous = debugPrint;
    debugPrint = (String? message, {int? wrapWidth}) => log('print: ${mask(message)}');
    debugPrintRecognizerCallbacksTrace = true;
    debugPrintGestureArenaDiagnostics = true;
    try {
      FakeAsync().run((FakeAsync async) {
        fa = async;
        scenes();
      });
    } finally {
      debugPrintRecognizerCallbacksTrace = false;
      debugPrintGestureArenaDiagnostics = false;
      debugPrint = previous;
    }
    File('build/recognizers_tap.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'log': lines}));
  });
}
