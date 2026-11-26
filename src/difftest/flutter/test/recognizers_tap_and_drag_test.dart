// Reference side of flutter-go's tap-and-drag recognizer differential
// test: the scenes of ../../recognizers_tap_and_drag_scenes.json (scripted
// pointer sequences routed through GestureBinding's pointer router and
// gesture arena, as gesture_tester.dart does) on
// TapAndHorizontalDragGestureRecognizer, TapAndPanGestureRecognizer and
// the deprecated TapAndDragGestureRecognizer, alone and competing with the
// other recognizers. Every callback is logged with all its details
// (doubles in full) and the details' toString, plus the
// debugPrintRecognizerCallbacksTrace and debugPrintGestureArenaDiagnostics
// output. Identity hashes are masked.
// The Go twin is ../../recognizers_tap_and_drag_test.go.
// Output: build/recognizers_tap_and_drag.json as {"log": [...]}.
// ignore_for_file: deprecated_member_use
import 'dart:convert';
import 'dart:io';

import 'package:fake_async/fake_async.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];

double d(Object? v) => (v! as num).toDouble();

String o(Offset v) => '(${v.dx},${v.dy})';

String ts(Duration? v) => v == null ? 'null' : '${v.inMicroseconds}';

String pd(double? v) => v == null ? 'null' : '$v';

String kd(PointerDeviceKind? k) => k == null ? 'null' : k.name;

String hashless(String s) => s.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH');

PointerDeviceKind kindOf(String name) =>
    PointerDeviceKind.values.firstWhere((PointerDeviceKind k) => k.name == name);

void logDetails(Object details) => log.add('  ${hashless('$details')}');

GestureRecognizer makeRecognizer(Map<String, Object?> spec) {
  final id = spec['id']! as String;
  final type = spec['type']! as String;
  void add(String s) => log.add('$id $s');
  switch (type) {
    case 'hd' || 'pan' || 'tad':
      final Set<PointerDeviceKind>? devices = spec['devices'] == null
          ? null
          : (spec['devices']! as List<Object?>).map((Object? k) => kindOf(k! as String)).toSet();
      final BaseTapAndDragGestureRecognizer r = switch (type) {
        'hd' => TapAndHorizontalDragGestureRecognizer(supportedDevices: devices),
        'pan' => TapAndPanGestureRecognizer(supportedDevices: devices),
        _ => TapAndDragGestureRecognizer(supportedDevices: devices),
      };
      if (spec['dsb'] != null) {
        r.dragStartBehavior = spec['dsb'] == 'down' ? DragStartBehavior.down : DragStartBehavior.start;
      }
      if (spec['eager'] != null) {
        r.eagerVictoryOnDrag = spec['eager']! as bool;
      }
      if (spec['max'] != null) {
        r.maxConsecutiveTap = spec['max']! as int;
      }
      if (spec['throttle'] != null) {
        r.dragUpdateThrottleFrequency = Duration(milliseconds: spec['throttle']! as int);
      }
      if (spec['slop'] != null) {
        r.gestureSettings = DeviceGestureSettings(touchSlop: d(spec['slop']));
      }
      r
        ..onTapTrackStart = () { add('onTapTrackStart'); }
        ..onTapTrackReset = () { add('onTapTrackReset'); }
        ..onTapDown = (TapDragDownDetails x) {
          add(
            'onTapDown g=${o(x.globalPosition)} l=${o(x.localPosition)} kind=${kd(x.kind)} '
            'count=${x.consecutiveTapCount}',
          );
          logDetails(x);
        }
        ..onTapUp = (TapDragUpDetails x) {
          add(
            'onTapUp g=${o(x.globalPosition)} l=${o(x.localPosition)} kind=${kd(x.kind)} '
            'count=${x.consecutiveTapCount}',
          );
          logDetails(x);
        }
        ..onDragStart = (TapDragStartDetails x) {
          add(
            'onDragStart g=${o(x.globalPosition)} l=${o(x.localPosition)} ts=${ts(x.sourceTimeStamp)} '
            'kind=${kd(x.kind)} count=${x.consecutiveTapCount}',
          );
          logDetails(x);
        }
        ..onDragUpdate = (TapDragUpdateDetails x) {
          add(
            'onDragUpdate g=${o(x.globalPosition)} l=${o(x.localPosition)} ts=${ts(x.sourceTimeStamp)} '
            'delta=${o(x.delta)} primary=${pd(x.primaryDelta)} kind=${kd(x.kind)} '
            'origin=${o(x.offsetFromOrigin)} localOrigin=${o(x.localOffsetFromOrigin)} '
            'count=${x.consecutiveTapCount}',
          );
          logDetails(x);
        }
        ..onDragEnd = (TapDragEndDetails x) {
          add(
            'onDragEnd g=${o(x.globalPosition)} l=${o(x.localPosition)} '
            'v=${o(x.velocity.pixelsPerSecond)} primary=${pd(x.primaryVelocity)} count=${x.consecutiveTapCount}',
          );
          logDetails(x);
        }
        ..onCancel = () { add('onCancel'); };
      return r;
    case 'tap':
      return TapGestureRecognizer()
        ..onTapDown = (TapDownDetails x) { add('onTapDown g=${o(x.globalPosition)}'); }
        ..onTapUp = (TapUpDetails x) { add('onTapUp g=${o(x.globalPosition)}'); }
        ..onTap = () { add('onTap'); }
        ..onTapCancel = () { add('onTapCancel'); };
    case 'hdrag' || 'vdrag' || 'panr':
      final DragGestureRecognizer r = switch (type) {
        'hdrag' => HorizontalDragGestureRecognizer(),
        'vdrag' => VerticalDragGestureRecognizer(),
        _ => PanGestureRecognizer(),
      };
      return r
        ..onDown = (DragDownDetails x) { add('onDown g=${o(x.globalPosition)}'); }
        ..onStart = (DragStartDetails x) { add('onStart g=${o(x.globalPosition)}'); }
        ..onUpdate = (DragUpdateDetails x) { add('onUpdate delta=${o(x.delta)}'); }
        ..onEnd = (DragEndDetails x) { add('onEnd v=${o(x.velocity.pixelsPerSecond)} primary=${pd(x.primaryVelocity)}'); }
        ..onCancel = () { add('onCancel'); };
    case 'scale':
      return ScaleGestureRecognizer()
        ..onStart = (ScaleStartDetails x) { add('onStart focal=${o(x.focalPoint)}'); }
        ..onUpdate = (ScaleUpdateDetails x) { add('onUpdate focal=${o(x.focalPoint)} scale=${x.scale}'); }
        ..onEnd = (ScaleEndDetails x) { add('onEnd'); };
    case 'lp':
      return LongPressGestureRecognizer()
        ..onLongPressStart = (LongPressStartDetails x) { add('onLongPressStart'); }
        ..onLongPressEnd = (LongPressEndDetails x) { add('onLongPressEnd'); }
        ..onLongPressCancel = () { add('onLongPressCancel'); };
    case 'dtap':
      return DoubleTapGestureRecognizer()
        ..onDoubleTapDown = (TapDownDetails x) { add('onDoubleTapDown'); }
        ..onDoubleTap = () { add('onDoubleTap'); }
        ..onDoubleTapCancel = () { add('onDoubleTapCancel'); };
    case 'eager':
      return EagerGestureRecognizer();
  }
  throw StateError(type);
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('recognizers tap and drag', (WidgetTester _) async {
    final scenes =
        json.decode(File('../recognizers_tap_and_drag_scenes.json').readAsStringSync()) as List<Object?>;
    final DebugPrintCallback oldPrint = debugPrint;
    debugPrint = (String? m, {int? wrapWidth}) => log.add('print ${hashless(m!)}');
    debugPrintRecognizerCallbacksTrace = true;
    debugPrintGestureArenaDiagnostics = true;
    FakeAsync().run((FakeAsync async) {
      for (final Object? sceneObj in scenes) {
        final scene = sceneObj! as Map<String, Object?>;
        log.add('scene ${scene['name']}');
        final recognizers = <String, GestureRecognizer>{};
        final pointers = <String, TestPointer>{};
        final downs = <String, PointerDownEvent>{};
        void route(PointerEvent e) {
          GestureBinding.instance.pointerRouter.route(e);
          async.flushMicrotasks();
        }

        for (final Object? opObj in scene['ops']! as List<Object?>) {
          final op = opObj! as Map<String, Object?>;
          final kind = op['op']! as String;
          log.add('> ${json.encode(op)}');
          switch (kind) {
            case 'new':
              recognizers[op['id']! as String] = makeRecognizer(op);
            case 'ptr':
              pointers[op['p']! as String] = TestPointer(
                op['pointer']! as int,
                kindOf(op['kind']! as String),
                null,
                op['buttons']! as int,
              );
            case 'down':
              final TestPointer p = pointers[op['p']! as String]!;
              final PointerDownEvent down = p.down(
                Offset(d(op['x']), d(op['y'])),
                timeStamp: Duration(milliseconds: op['ts']! as int),
                buttons: op['buttons'] as int?,
              );
              downs[op['p']! as String] = down;
              final PointerDownEvent added = op['m'] == null
                  ? down
                  : down.transformed(
                          Matrix4.fromList((op['m']! as List<Object?>).map(d).toList()),
                        )
                        as PointerDownEvent;
              for (final Object? id in op['add']! as List<Object?>) {
                recognizers[id! as String]!.addPointer(added);
              }
            case 'routedown':
              route(downs[op['p']! as String]!);
            case 'close':
              GestureBinding.instance.gestureArena.close(op['pointer']! as int);
            case 'sweep':
              GestureBinding.instance.gestureArena.sweep(op['pointer']! as int);
            case 'move':
              route(
                pointers[op['p']! as String]!.move(
                  Offset(d(op['x']), d(op['y'])),
                  timeStamp: Duration(milliseconds: op['ts']! as int),
                  buttons: op['buttons'] as int?,
                ),
              );
            case 'up':
              route(
                pointers[op['p']! as String]!.up(timeStamp: Duration(milliseconds: op['ts']! as int)),
              );
            case 'cancel':
              route(
                pointers[op['p']! as String]!.cancel(timeStamp: Duration(milliseconds: op['ts']! as int)),
              );
            case 'elapse':
              async.elapse(Duration(milliseconds: op['ms']! as int));
            case 'flush':
              async.flushMicrotasks();
            case 'state':
              final r = recognizers[op['id']! as String]! as BaseTapAndDragGestureRecognizer;
              log.add(
                'state count=${r.consecutiveTapCount} down=${r.currentDown?.pointer} '
                'up=${r.currentUp?.pointer} ${hashless('$r')} ${r.debugDescription}',
              );
            case 'dispose':
              recognizers[op['id']! as String]!.dispose();
          }
        }
        async.elapse(const Duration(seconds: 2));
      }
    });
    debugPrintRecognizerCallbacksTrace = false;
    debugPrintGestureArenaDiagnostics = false;
    debugPrint = oldPrint;
    File('build/recognizers_tap_and_drag.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'log': log}));
  });
}
