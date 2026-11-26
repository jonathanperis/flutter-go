// Reference side of flutter-go's drag recognizer differential test: the
// scenes of ../../recognizers_drag_scenes.json (scripted pointer sequences
// routed through GestureBinding's pointer router and gesture arena, as
// gesture_tester.dart does) run on the vertical, horizontal and pan drag
// recognizers and the multi-drag recognizers; every callback is logged
// with all its details (doubles in full), plus the recognizers'
// debugFillProperties and the debugPrintRecognizerCallbacksTrace output.
// The Go twin is ../../recognizers_drag_test.go.
// Output: build/recognizers_drag.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';

import 'package:fake_async/fake_async.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];

double d(Object? v) => (v! as num).toDouble();

String o(Offset v) => '(${v.dx},${v.dy})';

String ts(Duration? v) => v == null ? 'null' : '${v.inMicroseconds}';

String pd(double? v) => v == null ? 'null' : '$v';

String hashless(String s) => s.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH');

PointerDeviceKind kindOf(String name) =>
    PointerDeviceKind.values.firstWhere((PointerDeviceKind k) => k.name == name);

class LogDrag extends Drag {
  LogDrag(this.name);
  final String name;
  @override
  void update(DragUpdateDetails details) {
    log.add(
      '$name drag.update g=${o(details.globalPosition)} l=${o(details.localPosition)} '
      'delta=${o(details.delta)} primary=${pd(details.primaryDelta)} ts=${ts(details.sourceTimeStamp)}',
    );
  }

  @override
  void end(DragEndDetails details) {
    log.add(
      '$name drag.end v=${o(details.velocity.pixelsPerSecond)} primary=${pd(details.primaryVelocity)} '
      'g=${o(details.globalPosition)} l=${o(details.localPosition)}',
    );
  }

  @override
  void cancel() {
    log.add('$name drag.cancel');
  }
}

GestureRecognizer makeRecognizer(Map<String, Object?> spec) {
  final id = spec['id']! as String;
  final type = spec['type']! as String;
  final Set<PointerDeviceKind>? devices = (spec['devices'] as List<Object?>?)
      ?.map((Object? k) => kindOf(k! as String))
      .toSet();
  switch (type) {
    case 'eager':
      return EagerGestureRecognizer(supportedDevices: devices);
    case 'immediate':
    case 'hmulti':
    case 'vmulti':
    case 'delayed':
      final MultiDragGestureRecognizer m = switch (type) {
        'immediate' => ImmediateMultiDragGestureRecognizer(supportedDevices: devices),
        'hmulti' => HorizontalMultiDragGestureRecognizer(supportedDevices: devices),
        'vmulti' => VerticalMultiDragGestureRecognizer(supportedDevices: devices),
        _ when spec['delay'] != null => DelayedMultiDragGestureRecognizer(
          supportedDevices: devices,
          delay: Duration(milliseconds: spec['delay']! as int),
        ),
        _ => DelayedMultiDragGestureRecognizer(supportedDevices: devices),
      };
      final nullDrag = spec['nullDrag'] == true;
      m.onStart = (Offset position) {
        log.add('$id onStart ${o(position)}');
        return nullDrag ? null : LogDrag(id);
      };
      return m;
  }
  final DragGestureRecognizer r = switch (type) {
    'vertical' => VerticalDragGestureRecognizer(supportedDevices: devices),
    'horizontal' => HorizontalDragGestureRecognizer(supportedDevices: devices),
    _ => PanGestureRecognizer(supportedDevices: devices),
  };
  if (spec['start'] == 'down') {
    r.dragStartBehavior = DragStartBehavior.down;
  }
  if (spec['strategy'] != null) {
    r.multitouchDragStrategy = MultitouchDragStrategy.values.firstWhere(
      (MultitouchDragStrategy s) => s.name == spec['strategy'],
    );
  }
  if (spec['minFlingDistance'] != null) {
    r.minFlingDistance = d(spec['minFlingDistance']);
  }
  if (spec['minFlingVelocity'] != null) {
    r.minFlingVelocity = d(spec['minFlingVelocity']);
  }
  if (spec['maxFlingVelocity'] != null) {
    r.maxFlingVelocity = d(spec['maxFlingVelocity']);
  }
  if (spec['onlyAcceptDragOnThreshold'] == true) {
    r.onlyAcceptDragOnThreshold = true;
  }
  if (spec['touchSlop'] != null) {
    r.gestureSettings = DeviceGestureSettings(touchSlop: d(spec['touchSlop']));
  }
  switch (spec['tracker']) {
    case 'ios':
      r.velocityTrackerBuilder = (PointerEvent e) => IOSScrollViewFlingVelocityTracker(e.kind);
    case 'macos':
      r.velocityTrackerBuilder = (PointerEvent e) => MacOSScrollViewFlingVelocityTracker(e.kind);
  }
  r
    ..onDown = (DragDownDetails x) {
      log.add('$id onDown g=${o(x.globalPosition)} l=${o(x.localPosition)}');
    }
    ..onStart = (DragStartDetails x) {
      log.add(
        '$id onStart g=${o(x.globalPosition)} l=${o(x.localPosition)} ts=${ts(x.sourceTimeStamp)} kind=${x.kind?.name}',
      );
    }
    ..onUpdate = (DragUpdateDetails x) {
      log.add(
        '$id onUpdate g=${o(x.globalPosition)} l=${o(x.localPosition)} delta=${o(x.delta)} '
        'primary=${pd(x.primaryDelta)} ts=${ts(x.sourceTimeStamp)} kind=${x.kind?.name}',
      );
    }
    ..onEnd = (DragEndDetails x) {
      log.add(
        '$id onEnd v=${o(x.velocity.pixelsPerSecond)} primary=${pd(x.primaryVelocity)} '
        'g=${o(x.globalPosition)} l=${o(x.localPosition)} global=${r.globalDistanceMoved} last=${o(r.lastPosition.global)}',
      );
    }
    ..onCancel = () {
      log.add('$id onCancel');
    };
  return r;
}

void main() {
  final scenes =
      json.decode(File('../recognizers_drag_scenes.json').readAsStringSync()) as List<Object?>;

  testWidgets('recognizers_drag', (WidgetTester tester) async {
    final DebugPrintCallback previousPrint = debugPrint;
    debugPrint = (String? message, {int? wrapWidth}) {
      log.add('print ${hashless(message ?? 'null')}');
    };
    for (final Object? s in scenes) {
      final scene = s! as Map<String, Object?>;
      log.add('== ${scene['name']}');
      debugPrintRecognizerCallbacksTrace = scene['trace'] == true;
      FakeAsync().run((FakeAsync async) {
        final recognizers = <String, GestureRecognizer>{};
        for (final Object? spec in scene['recognizers']! as List<Object?>) {
          final m = spec! as Map<String, Object?>;
          recognizers[m['id']! as String] = makeRecognizer(m);
        }
        final pointers = <int, TestPointer>{};
        final events = <int, PointerEvent>{};
        Matrix4? transform;
        void route(PointerEvent e) {
          GestureBinding.instance.pointerRouter.route(e);
          async.flushMicrotasks();
        }

        for (final Object? opObject in scene['ops']! as List<Object?>) {
          final op = opObject! as List<Object?>;
          final p = op.length > 1 && op[1] is int ? op[1]! as int : 0;
          if (op[0] != 'transform') {
            log.add('> ${op[0]} $p');
          }
          switch (op[0]) {
            case 'transform':
              final List<double> m = (op[1]! as List<Object?>).map(d).toList();
              transform = Matrix4.fromList(m);
            case 'down':
              final pointer = TestPointer(p, kindOf(op[2]! as String), null, (op[6] as int?) ?? kPrimaryButton);
              pointers[p] = pointer;
              final PointerDownEvent down = pointer.down(
                Offset(d(op[3]), d(op[4])),
                timeStamp: Duration(microseconds: op[5]! as int),
              );
              events[p] = down;
              for (final Object? id in op[7]! as List<Object?>) {
                recognizers[id]!.addPointer(down.transformed(transform) as PointerDownEvent);
              }
            case 'pzstart':
              final pointer = TestPointer(p, PointerDeviceKind.trackpad);
              pointers[p] = pointer;
              final PointerPanZoomStartEvent start = pointer.panZoomStart(
                Offset(d(op[2]), d(op[3])),
                timeStamp: Duration(microseconds: op[4]! as int),
              );
              events[p] = start;
              for (final Object? id in op[5]! as List<Object?>) {
                recognizers[id]!.addPointerPanZoom(
                  start.transformed(transform) as PointerPanZoomStartEvent,
                );
              }
            case 'close':
              GestureBinding.instance.gestureArena.close(p);
            case 'sweep':
              GestureBinding.instance.gestureArena.sweep(p);
            case 'route_down':
            case 'route_pz':
              route(events[p]!);
            case 'move':
              final int? buttons = op[5] as int?;
              route(
                pointers[p]!.move(
                  Offset(d(op[2]), d(op[3])),
                  timeStamp: Duration(microseconds: op[4]! as int),
                  buttons: buttons,
                ),
              );
            case 'synth':
              route(
                PointerMoveEvent(
                  pointer: p,
                  position: Offset(d(op[2]), d(op[3])),
                  timeStamp: Duration(microseconds: op[4]! as int),
                  synthesized: true,
                ),
              );
            case 'up':
              route(pointers[p]!.up(timeStamp: Duration(microseconds: op[2]! as int)));
            case 'cancel':
              route(pointers[p]!.cancel(timeStamp: Duration(microseconds: op[2]! as int)));
            case 'pzupdate':
              route(
                pointers[p]!.panZoomUpdate(
                  pointers[p]!.location!,
                  pan: Offset(d(op[2]), d(op[3])),
                  timeStamp: Duration(microseconds: op[4]! as int),
                ),
              );
            case 'pzend':
              route(pointers[p]!.panZoomEnd(timeStamp: Duration(microseconds: op[2]! as int)));
            case 'frame':
              SchedulerBinding.instance.handleBeginFrame(Duration(microseconds: op[1]! as int));
              SchedulerBinding.instance.handleDrawFrame();
            case 'elapse':
              async.elapse(Duration(milliseconds: op[1]! as int));
          }
        }
        for (final MapEntry<String, GestureRecognizer> e in recognizers.entries) {
          final builder = DiagnosticPropertiesBuilder();
          e.value.debugFillProperties(builder);
          log.add(
            '${e.key} ${e.value.debugDescription} props=${builder.properties.map((DiagnosticsNode n) => n.toString()).join('; ')}',
          );
          e.value.dispose();
        }
        async.flushMicrotasks();
      });
    }
    debugPrintRecognizerCallbacksTrace = false;
    debugPrint = previousPrint;
    File('build/recognizers_drag.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'log': log}));
  });
}
