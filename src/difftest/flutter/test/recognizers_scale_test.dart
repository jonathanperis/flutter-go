// Reference side of flutter-go's scale recognizer differential test.
// Scenes are scripts of operations (create recognizers and test pointers,
// make events, add pointers to recognizers, close arenas, route events in
// gesture_tester's way); this file runs them and records both the script
// and the log of every callback (details' fields at full precision and
// their toString), arena accept/reject, and pointerCount after each route.
// The Go twin (../../recognizers_scale_test.go) replays the recorded script
// and compares the log line by line.
// Output: build/recognizers_scale.json as {"script": [...], "log": [...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;

import 'package:fake_async/fake_async.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];
final List<Map<String, Object?>> script = <Map<String, Object?>>[];

String mask(String s) => s.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#');
String d(double v) => '$v';
String o(Offset v) => '(${d(v.dx)}, ${d(v.dy)})';
String ts(Duration? t) => t == null ? 'null' : '${t.inMicroseconds}';

final List<Matrix4> transforms = <Matrix4>[
  Matrix4.translationValues(-350, -250, 0),
  Matrix4.diagonal3Values(2, 0.5, 1),
  Matrix4.fromList(<double>[0.8, 0.6, 0, 0, -0.6, 0.8, 0, 0, 0, 0, 1, 0, 10, 20, 0, 1]),
];

class LogScale extends ScaleGestureRecognizer {
  LogScale(this.id, {super.dragStartBehavior, super.trackpadScrollCausesScale, super.trackpadScrollToScaleFactor, super.supportedDevices});
  final int id;
  @override
  void acceptGesture(int pointer) {
    log.add('r$id accept $pointer');
    super.acceptGesture(pointer);
  }

  @override
  void rejectGesture(int pointer) {
    log.add('r$id reject $pointer');
    super.rejectGesture(pointer);
  }
}

class LogEager extends EagerGestureRecognizer {
  LogEager(this.id);
  final int id;
  @override
  void acceptGesture(int pointer) {
    log.add('r$id accept $pointer');
    super.acceptGesture(pointer);
  }

  @override
  void rejectGesture(int pointer) {
    log.add('r$id reject $pointer');
    super.rejectGesture(pointer);
  }
}

final Map<int, GestureRecognizer> recognizers = <int, GestureRecognizer>{};
final Map<int, TestPointer> pointers = <int, TestPointer>{};
final Map<int, PointerEvent> lastEvent = <int, PointerEvent>{};
late FakeAsync fake;

PointerDeviceKind kindNamed(String name) =>
    PointerDeviceKind.values.firstWhere((PointerDeviceKind k) => k.name == name);

void run(Map<String, Object?> op) {
  script.add(op);
  final String name = op['op']! as String;
  int i(String k) => op[k]! as int;
  double n(String k) => (op[k]! as num).toDouble();
  Duration t() => Duration(microseconds: (op['ts'] as int?) ?? 0);
  switch (name) {
    case 'scene':
      log.add('== ${op['name']}');
    case 'scale':
      final int id = i('id');
      final List<Object?>? factor = op['factor'] as List<Object?>?;
      final List<Object?>? devices = op['devices'] as List<Object?>?;
      final s = LogScale(
        id,
        dragStartBehavior: op['start'] == true ? DragStartBehavior.start : DragStartBehavior.down,
        trackpadScrollCausesScale: op['tscs'] == true,
        trackpadScrollToScaleFactor: factor == null
            ? kDefaultTrackpadScrollToScaleFactor
            : Offset((factor[0]! as num).toDouble(), (factor[1]! as num).toDouble()),
        supportedDevices: devices?.map((Object? k) => kindNamed(k! as String)).toSet(),
      );
      s.onStart = (ScaleStartDetails x) => log.add(
        'r$id start focal=${o(x.focalPoint)} local=${o(x.localFocalPoint)} count=${x.pointerCount} '
        'ts=${ts(x.sourceTimeStamp)} kind=${x.kind?.name} | ${mask(x.toString())}',
      );
      s.onUpdate = (ScaleUpdateDetails x) => log.add(
        'r$id update focal=${o(x.focalPoint)} local=${o(x.localFocalPoint)} delta=${o(x.focalPointDelta)} '
        'scale=${d(x.scale)} h=${d(x.horizontalScale)} v=${d(x.verticalScale)} rot=${d(x.rotation)} '
        'count=${x.pointerCount} ts=${ts(x.sourceTimeStamp)} | ${mask(x.toString())}',
      );
      s.onEnd = (ScaleEndDetails x) => log.add(
        'r$id end v=${o(x.velocity.pixelsPerSecond)} sv=${d(x.scaleVelocity)} count=${x.pointerCount} | ${mask(x.toString())}',
      );
      recognizers[id] = s;
    case 'eager':
      recognizers[i('id')] = LogEager(i('id'));
    case 'factor':
      (recognizers[i('id')]! as ScaleGestureRecognizer).trackpadScrollToScaleFactor = Offset(n('x'), n('y'));
    case 'pointer':
      pointers[i('p')] = TestPointer(i('p'), kindNamed(op['kind']! as String));
    case 'down':
      lastEvent[i('p')] = pointers[i('p')]!.down(Offset(n('x'), n('y')), timeStamp: t());
    case 'move':
      lastEvent[i('p')] = pointers[i('p')]!.move(Offset(n('x'), n('y')), timeStamp: t());
    case 'synthMove':
      final TestPointer p = pointers[i('p')]!;
      final Offset to = Offset(n('x'), n('y'));
      lastEvent[i('p')] = PointerMoveEvent(
        timeStamp: t(),
        kind: p.kind,
        device: p.device,
        pointer: p.pointer,
        position: to,
        delta: to - p.location!,
        synthesized: true,
      );
      p.setDownInfo(lastEvent[i('p')]!, to);
    case 'up':
      lastEvent[i('p')] = pointers[i('p')]!.up(timeStamp: t());
    case 'cancel':
      lastEvent[i('p')] = pointers[i('p')]!.cancel(timeStamp: t());
    case 'pzStart':
      lastEvent[i('p')] = pointers[i('p')]!.panZoomStart(Offset(n('x'), n('y')), timeStamp: t());
    case 'pzUpdate':
      lastEvent[i('p')] = pointers[i('p')]!.panZoomUpdate(
        Offset(n('x'), n('y')),
        pan: Offset(n('px'), n('py')),
        scale: n('scale'),
        rotation: n('rot'),
        timeStamp: t(),
      );
    case 'pzEnd':
      lastEvent[i('p')] = pointers[i('p')]!.panZoomEnd(timeStamp: t());
    case 'add':
      PointerEvent e = lastEvent[i('p')]!;
      if (op['t'] != null) {
        e = e.transformed(transforms[i('t')]);
      }
      final GestureRecognizer r = recognizers[i('id')]!;
      if (e is PointerPanZoomStartEvent) {
        r.addPointerPanZoom(e);
      } else {
        r.addPointer(e as PointerDownEvent);
      }
    case 'close':
      GestureBinding.instance.gestureArena.close(i('p'));
    case 'sweep':
      GestureBinding.instance.gestureArena.sweep(i('p'));
      fake.flushMicrotasks();
    case 'route':
      GestureBinding.instance.pointerRouter.route(lastEvent[i('p')]!);
      fake.flushMicrotasks();
      final List<String> counts = <String>[];
      for (final MapEntry<int, GestureRecognizer> e in recognizers.entries) {
        if (e.value is ScaleGestureRecognizer) {
          counts.add('r${e.key}=${(e.value as ScaleGestureRecognizer).pointerCount}');
        }
      }
      log.add('routed ${lastEvent[i('p')]!.runtimeType} p${i('p')} ${counts.join(' ')}');
    case 'dispose':
      recognizers.remove(i('id'))!.dispose();
    case 'disposeAll':
      for (final GestureRecognizer r in recognizers.values) {
        r.dispose();
      }
      recognizers.clear();
      pointers.clear();
      lastEvent.clear();
  }
}

// Script helpers.
void scene(String name) => run(<String, Object?>{'op': 'scene', 'name': name});
void scale(int id, {bool start = false, bool tscs = false, List<double>? factor, List<String>? devices}) => run(
  <String, Object?>{'op': 'scale', 'id': id, 'start': start, 'tscs': tscs, 'factor': factor, 'devices': devices},
);
void eager(int id) => run(<String, Object?>{'op': 'eager', 'id': id});
void pointer(int p, [String kind = 'touch']) => run(<String, Object?>{'op': 'pointer', 'p': p, 'kind': kind});
void down(int p, double x, double y, [int ts = 0]) =>
    run(<String, Object?>{'op': 'down', 'p': p, 'x': x, 'y': y, 'ts': ts});
void move(int p, double x, double y, [int ts = 0]) =>
    run(<String, Object?>{'op': 'move', 'p': p, 'x': x, 'y': y, 'ts': ts});
void synthMove(int p, double x, double y, [int ts = 0]) =>
    run(<String, Object?>{'op': 'synthMove', 'p': p, 'x': x, 'y': y, 'ts': ts});
void up(int p, [int ts = 0]) => run(<String, Object?>{'op': 'up', 'p': p, 'ts': ts});
void cancel(int p, [int ts = 0]) => run(<String, Object?>{'op': 'cancel', 'p': p, 'ts': ts});
void pzStart(int p, double x, double y, [int ts = 0]) =>
    run(<String, Object?>{'op': 'pzStart', 'p': p, 'x': x, 'y': y, 'ts': ts});
void pzUpdate(int p, double x, double y, {double px = 0, double py = 0, double scale = 1, double rot = 0, int ts = 0}) =>
    run(<String, Object?>{'op': 'pzUpdate', 'p': p, 'x': x, 'y': y, 'px': px, 'py': py, 'scale': scale, 'rot': rot, 'ts': ts});
void pzEnd(int p, [int ts = 0]) => run(<String, Object?>{'op': 'pzEnd', 'p': p, 'ts': ts});
void add(int id, int p, [int? t]) => run(<String, Object?>{'op': 'add', 'id': id, 'p': p, 't': t});
void close(int p) => run(<String, Object?>{'op': 'close', 'p': p});
void sweep(int p) => run(<String, Object?>{'op': 'sweep', 'p': p});
void route(int p) => run(<String, Object?>{'op': 'route', 'p': p});
void factor(int id, double x, double y) => run(<String, Object?>{'op': 'factor', 'id': id, 'x': x, 'y': y});
void disposeAll() => run(<String, Object?>{'op': 'disposeAll'});

// down + add to recognizers + close + route.
void press(int p, double x, double y, List<int> ids, {int ts = 0, int? t}) {
  down(p, x, y, ts);
  for (final int id in ids) {
    add(id, p, t);
  }
  close(p);
  route(p);
}

void scenes() {
  scene('three fingers, pointers added and removed');
  scale(1);
  pointer(1);
  pointer(2);
  pointer(3);
  press(1, 0, 0, <int>[1], ts: 1000);
  move(1, 20, 30, 9000);
  route(1);
  press(2, 10, 20, <int>[1], ts: 17000);
  for (var k = 1; k <= 6; k++) {
    move(2, 10.0 - 3.7 * k, 20 + 2.3 * k * k, 17000 + 8000 * k);
    route(2);
  }
  press(3, 25.5, 35.25, <int>[1], ts: 80000);
  move(3, 55.125, 65.75, 88000);
  route(3);
  move(1, 25, 35, 96000);
  route(1);
  move(2, 20, 30, 104000);
  route(2);
  up(2, 112000); // the middle pointer leaves: the line changes
  route(2);
  for (var k = 1; k <= 5; k++) {
    move(3, 55.125 - 11.1 * k, 65.75 + 7.3 * k, 112000 + 7000 * k);
    route(3);
    move(1, 25 + 0.7 * k, 35 - 13.0 * k, 113000 + 7000 * k);
    route(1);
  }
  up(3, 160000);
  route(3);
  move(1, 300, 400, 168000);
  route(1);
  up(1, 170000);
  route(1);
  disposeAll();

  scene('fast fling clamped, slow end');
  scale(1);
  pointer(1);
  pointer(2);
  press(1, 100, 100, <int>[1]);
  press(2, 200, 100, <int>[1]);
  for (var k = 1; k <= 8; k++) {
    move(2, 200.0 + 120 * k, 100.0 + 40 * k, 4000 * k);
    route(2);
  }
  up(2, 33000);
  route(2);
  for (var k = 1; k <= 6; k++) {
    move(1, 100.0 + 0.5 * k, 100.0 - 0.25 * k, 33000 + 16000 * k);
    route(1);
  }
  up(1, 140000);
  route(1);
  disposeAll();

  scene('rotation through pi and slop');
  scale(1);
  pointer(1);
  pointer(2);
  press(1, 0, 0, <int>[1]);
  press(2, 30, 40, <int>[1]);
  for (final List<double> m in <List<double>>[
    <double>[30.5, 40.2],
    <double>[31, 41],
    <double>[0, 10],
    <double>[-30, -40],
    <double>[-31, -39],
    <double>[10, -50],
  ]) {
    move(2, m[0], m[1]);
    route(2);
  }
  cancel(2);
  route(2);
  move(1, 3, 3);
  route(1);
  up(1);
  route(1);
  disposeAll();

  scene('eager competitor added first');
  eager(2);
  scale(1);
  pointer(1);
  down(1, 10, 10);
  add(2, 1);
  add(1, 1);
  close(1);
  route(1);
  move(1, 80, 80);
  route(1);
  up(1);
  route(1);
  disposeAll();

  scene('eager competitor added second');
  scale(1);
  eager(2);
  pointer(1);
  down(1, 10, 10);
  add(1, 1);
  add(2, 1);
  close(1);
  route(1);
  move(1, 80, 80);
  route(1);
  up(1);
  route(1);
  disposeAll();

  scene('two scale recognizers, start behavior');
  scale(1);
  scale(2, start: true);
  pointer(1);
  pointer(2);
  press(1, 50, 50, <int>[1, 2]);
  move(1, 60, 70, 5000);
  route(1);
  press(2, 150, 50, <int>[1, 2], ts: 6000);
  move(2, 170, 30, 9000);
  route(2);
  move(1, 20, 90, 12000);
  route(1);
  up(1, 13000);
  route(1);
  up(2, 14000);
  route(2);
  sweep(1);
  sweep(2);
  disposeAll();

  scene('start behavior, single recognizer wins by sweep');
  scale(1, start: true);
  pointer(1);
  press(1, 50, 50, <int>[1]);
  move(1, 52, 51, 1000);
  route(1);
  up(1, 2000);
  route(1);
  sweep(1);
  disposeAll();

  scene('scale slop by span');
  scale(1);
  scale(2);
  pointer(1);
  pointer(2);
  down(1, 0, 0);
  add(1, 1);
  add(2, 1);
  close(1);
  route(1);
  down(2, 100, 0);
  add(2, 2);
  add(1, 2);
  close(2);
  route(2);
  for (var k = 1; k <= 12; k++) {
    move(2, 100.0 + 4.5 * k, 0, 1000 * k);
    route(2);
  }
  up(1, 20000);
  route(1);
  up(2, 21000);
  route(2);
  disposeAll();

  scene('mouse and stylus pan slop');
  scale(1);
  scale(2);
  pointer(1, 'mouse');
  pointer(2, 'stylus');
  press(1, 0, 0, <int>[1, 2]);
  for (var k = 1; k <= 4; k++) {
    move(1, 0.9 * k, 0.4 * k, 1000 * k);
    route(1);
  }
  up(1, 6000);
  route(1);
  sweep(1);
  press(2, 0, 0, <int>[1, 2]);
  for (var k = 1; k <= 6; k++) {
    move(2, 3.1 * k, 4.2 * k, 1000 * k);
    route(2);
  }
  up(2, 8000);
  route(2);
  sweep(2);
  disposeAll();

  scene('supported devices');
  scale(1, devices: <String>['touch']);
  pointer(1, 'mouse');
  pointer(2);
  press(1, 0, 0, <int>[1]);
  move(1, 40, 40);
  route(1);
  press(2, 10, 10, <int>[1]);
  move(2, 60, 60);
  route(2);
  up(1);
  route(1);
  up(2);
  route(2);
  disposeAll();

  scene('synthesized moves are not tracked');
  scale(1);
  pointer(1);
  pointer(2);
  press(1, 0, 0, <int>[1]);
  press(2, 50, 0, <int>[1]);
  for (var k = 1; k <= 5; k++) {
    move(2, 50.0 + 30 * k, 10.0 * k, 10000 * k);
    route(2);
    synthMove(2, 52.0 + 30 * k, 10.0 * k + 1, 10000 * k + 5000);
    route(2);
  }
  up(2, 70000);
  route(2);
  up(1, 71000);
  route(1);
  disposeAll();

  scene('trackpad pan/zoom alone');
  scale(1);
  pointer(4, 'trackpad');
  pzStart(4, 5, 7, 1000);
  add(1, 4);
  close(4);
  route(4);
  pzUpdate(4, 5, 7, px: 20, py: 30, ts: 2000);
  route(4);
  pzUpdate(4, 5, 7, px: 20, py: 30, scale: 2, rot: 0.25, ts: 3000);
  route(4);
  pzUpdate(4, 6, 8, px: 25, py: 33, scale: 1.5, rot: -0.75, ts: 4000);
  route(4);
  pzUpdate(4, 6, 8, px: 25, py: 33, scale: 0.7, rot: math.pi / 3, ts: 5000);
  route(4);
  pzEnd(4, 6000);
  route(4);
  disposeAll();

  scene('trackpad zoom below and above 1.05');
  scale(1);
  scale(2);
  pointer(4, 'trackpad');
  pzStart(4, 0, 0);
  add(1, 4);
  add(2, 4);
  close(4);
  route(4);
  pzUpdate(4, 0, 0, scale: 1.04);
  route(4);
  pzUpdate(4, 0, 0, scale: 1.06);
  route(4);
  pzUpdate(4, 0, 0, scale: 1.2, rot: 1);
  route(4);
  pzEnd(4);
  route(4);
  disposeAll();

  scene('trackpad scroll causes scale');
  scale(1, start: true, tscs: true);
  pointer(4, 'trackpad');
  pzStart(4, 0, 0);
  add(1, 4);
  close(4);
  route(4);
  for (var k = 0; k < 30; k++) {
    pzUpdate(4, 0, 0, px: 3.0 * k, py: k * -10.0, ts: k * 25000);
    route(4);
  }
  pzEnd(4, 750000);
  route(4);
  factor(1, 1 / 125, 0);
  pzStart(4, 3, 4);
  add(1, 4);
  close(4);
  route(4);
  pzUpdate(4, 3, 4, px: 125, py: 0);
  route(4);
  pzUpdate(4, 3, 4, px: 125, py: 125, scale: 1.5);
  route(4);
  pzEnd(4);
  route(4);
  disposeAll();

  scene('pan/zoom alongside touches, start behavior');
  scale(1, start: true);
  pointer(2);
  pointer(3);
  pointer(4, 'trackpad');
  pzStart(4, 0, 0);
  add(1, 4);
  close(4);
  route(4);
  pzUpdate(4, 0, 0, px: 40, py: 40, ts: 1000);
  route(4);
  press(2, 40, 40, <int>[1], ts: 2000);
  move(2, 10, 10, 3000);
  route(2);
  press(3, 10, 40, <int>[1], ts: 4000);
  move(3, 40, 40, 5000);
  route(3);
  pzUpdate(4, 0, 0, px: 40, py: 40, scale: math.sqrt2, rot: math.pi / 3, ts: 6000);
  route(4);
  pzUpdate(4, 15, 15, px: 55, py: 55, scale: math.sqrt2, rot: math.pi / 3, ts: 7000);
  route(4);
  pzEnd(4, 8000);
  route(4);
  move(3, 45, 47, 9000);
  route(3);
  up(2, 10000);
  route(2);
  up(3, 11000);
  route(3);
  disposeAll();

  scene('two pan/zooms');
  scale(1);
  pointer(4, 'trackpad');
  pointer(5, 'trackpad');
  pzStart(4, 0, 0);
  add(1, 4);
  close(4);
  route(4);
  pzUpdate(4, 0, 0, px: 10, py: 40);
  route(4);
  pzStart(5, 10, 20);
  add(1, 5);
  close(5);
  route(5);
  pzUpdate(5, 10, 20, px: 20, scale: 1.3, rot: 0.1);
  route(5);
  pzUpdate(4, 0, 0, px: 10, py: 40, scale: 0.9, rot: 0.2);
  route(4);
  pzEnd(4);
  route(4);
  pzUpdate(5, 10, 20, px: 70, scale: 1.1, rot: 0.3);
  route(5);
  pzEnd(5);
  route(5);
  disposeAll();

  scene('configuration changes in a row');
  scale(1);
  pointer(1);
  pointer(2);
  pointer(3);
  pointer(4, 'trackpad');
  press(1, 10, 10, <int>[1], ts: 1000);
  move(1, 30, 50, 2000);
  route(1);
  press(2, 100, 20, <int>[1], ts: 3000);
  pzStart(4, 5, 5, 3500);
  add(1, 4);
  close(4);
  route(4);
  press(3, 60, 90, <int>[1], ts: 4000);
  pzUpdate(4, 5, 5, px: 7, py: -3, scale: 1.25, rot: 0.5, ts: 6000);
  route(4);
  move(3, 70, 95, 7000);
  route(3);
  up(1, 8000);
  route(1);
  press(1, 0, 0, <int>[1], ts: 9000);
  up(2, 10000);
  route(2);
  pzEnd(4, 11000);
  route(4);
  move(1, 5, 5, 12000);
  route(1);
  up(3, 13000);
  route(3);
  up(1, 14000);
  route(1);
  disposeAll();

  scene('pointer rejected mid-gesture changes the line');
  scale(2);
  scale(1);
  pointer(1);
  pointer(2);
  pointer(3);
  down(1, 0, 0);
  add(2, 1);
  add(1, 1);
  close(1);
  route(1);
  press(2, 40, 0, <int>[1], ts: 1000);
  press(3, 40, 40, <int>[1], ts: 2000);
  sweep(1);
  move(2, 45, 5, 3000);
  route(2);
  move(3, 30, 50, 4000);
  route(3);
  up(1, 5000);
  route(1);
  up(2, 6000);
  route(2);
  up(3, 7000);
  route(3);
  disposeAll();

  scene('middle pointer rejected');
  scale(1);
  scale(2);
  pointer(1);
  pointer(2);
  pointer(3);
  press(1, 0, 0, <int>[1], ts: 0);
  down(2, 40, 0, 1000);
  add(2, 2);
  add(1, 2);
  close(2);
  route(2);
  press(3, 40, 40, <int>[1], ts: 2000);
  sweep(2);
  move(1, 5, -5, 3000);
  route(1);
  move(3, 30, 50, 4000);
  route(3);
  up(1, 5000);
  route(1);
  up(2, 6000);
  route(2);
  up(3, 7000);
  route(3);
  disposeAll();

  scene('middle pointer lost before acceptance');
  scale(1);
  scale(2);
  scale(3);
  pointer(1);
  pointer(2);
  pointer(3);
  down(1, 0, 0);
  add(1, 1);
  add(3, 1);
  close(1);
  route(1);
  down(2, 40, 0, 1000);
  add(2, 2);
  add(1, 2);
  close(2);
  route(2);
  down(3, 40, 40, 2000);
  add(1, 3);
  add(3, 3);
  close(3);
  route(3);
  sweep(2);
  move(3, 10, 80, 3000);
  route(3);
  move(1, -5, 5, 4000);
  route(1);
  up(1, 5000);
  route(1);
  up(2, 6000);
  route(2);
  up(3, 7000);
  route(3);
  disposeAll();

  scene('start behavior wins the arena after moving');
  scale(2, start: true);
  scale(1);
  pointer(1);
  pointer(2);
  press(1, 50, 50, <int>[2, 1]);
  press(2, 150, 50, <int>[2, 1], ts: 1000);
  move(2, 160, 40, 2000);
  route(2);
  move(2, 200, 10, 3000);
  route(2);
  move(1, 40, 60, 4000);
  route(1);
  up(1, 5000);
  route(1);
  up(2, 6000);
  route(2);
  disposeAll();

  scene('trackpad zoom out below 1/1.05');
  scale(1);
  scale(2);
  pointer(4, 'trackpad');
  pzStart(4, 0, 0);
  add(1, 4);
  add(2, 4);
  close(4);
  route(4);
  pzUpdate(4, 0, 0, scale: 0.96);
  route(4);
  pzUpdate(4, 0, 0, scale: 0.94);
  route(4);
  pzEnd(4);
  route(4);
  disposeAll();

  scene('mixed transforms');
  scale(1);
  pointer(1);
  pointer(2);
  press(1, 380, 280, <int>[1], t: 0);
  press(2, 430, 330, <int>[1], t: 2);
  move(2, 420, 320, 1000);
  route(2);
  move(1, 300, 200, 2000);
  route(1);
  up(2, 3000);
  route(2);
  move(1, 310, 205, 4000);
  route(1);
  press(2, 400, 300, <int>[1], ts: 5000, t: 1);
  up(1, 6000);
  route(1);
  move(2, 410, 290, 7000);
  route(2);
  up(2, 8000);
  route(2);
  disposeAll();

  scene('transformed events');
  for (var t = 0; t < 3; t++) {
    scale(1);
    pointer(1);
    pointer(2);
    press(1, 380, 280, <int>[1], t: t);
    press(2, 430, 330, <int>[1], t: t);
    move(2, 420, 320, 1000);
    route(2);
    move(1, 300, 200, 2000);
    route(1);
    move(2, 500, 260, 3000);
    route(2);
    up(1, 4000);
    route(1);
    up(2, 5000);
    route(2);
    disposeAll();
  }
}

void main() {
  testWidgets('recognizers_scale', (WidgetTester tester) async {
    FakeAsync().run((FakeAsync async) {
      fake = async;
      scenes();
    });
    File('build/recognizers_scale.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'script': script, 'log': log}));
  });
}
