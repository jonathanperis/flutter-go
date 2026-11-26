// Reference side of flutter-go's pointer events differential test (the
// gestures library's base layer): every PointerEvent class (defaults and
// full fields) with its getters, toString, toStringShort, toStringFull,
// toStringDeep and properties; transformed variants (translation, scale,
// rotation, perspective; transformed twice, back to null, identity),
// copyWith on originals and transformed events, fromMouseEvent; the static
// transform helpers and slops; PointerEventConverter.expand over packets
// of every change and signal kind at device pixel ratios 1, 2 and 3.5 (and
// a missing view); PointerEventResampler scripts; GestureArenaManager
// scripts with the arena diagnostics printed; PointerRouter with
// transforms, re-entrancy and errors; PointerSignalResolver; HitTestResult
// transforms; drag details and PointerData/KeyData toString. Identity
// hashes are masked. The Go twin (../pointer_events_test.go) is compared
// exactly. Output: build/pointer_events.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:fake_async/fake_async.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> lines = <String>[];
void log(String s) => lines.add(s);

String mask(Object? o) => '$o'.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH');

String off(Offset o) => '${o.dx} ${o.dy}';
String mat(Matrix4? m) => m == null ? 'null' : m.storage.join(',');
String typeOf(Object e) => e is Diagnosticable ? e.toStringShort().split('#').first : '${e.runtimeType}';

String fields(PointerEvent e) {
  final parts = <String>[
    typeOf(e), '${e.viewId}', '${e.embedderId}', '${e.timeStamp.inMicroseconds}', '${e.pointer}', e.kind.name,
    '${e.device}', off(e.position), off(e.delta), off(e.localPosition), off(e.localDelta), '${e.buttons}',
    '${e.down}', '${e.obscured}', '${e.pressure}', '${e.pressureMin}', '${e.pressureMax}', '${e.distance}',
    '${e.distanceMin}', '${e.distanceMax}', '${e.size}', '${e.radiusMajor}', '${e.radiusMinor}', '${e.radiusMin}',
    '${e.radiusMax}', '${e.orientation}', '${e.tilt}', '${e.platformData}', '${e.synthesized}', mat(e.transform),
    e.original == null ? 'orig:null' : 'orig:${typeOf(e.original!)}',
  ];
  if (e is PointerScrollEvent) {
    parts.add('scrollDelta ${off(e.scrollDelta)}');
  }
  if (e is PointerScaleEvent) {
    parts.add('scale ${e.scale}');
  }
  if (e is PointerPanZoomUpdateEvent) {
    parts.add('pan ${off(e.pan)} ${off(e.localPan)} ${off(e.panDelta)} ${off(e.localPanDelta)} ${e.scale} ${e.rotation}');
  }
  if (e is PointerSignalEvent) {
    parts.add('signal');
  }
  return parts.join(' ');
}

void describe(String label, PointerEvent e) {
  log('$label ${fields(e)}');
  log('  toString ${mask(e)}');
  log('  short ${mask(e.toStringShort())}');
  log('  full ${mask((e as dynamic).toStringFull())}');
  log('  deep ${mask(e.toDiagnosticsNode().toStringDeep())}');
  final builder = DiagnosticPropertiesBuilder();
  e.debugFillProperties(builder);
  log('  props ${builder.properties.map((DiagnosticsNode p) => '${mask(p)}/${p.level.name}/${p.isFiltered(DiagnosticLevel.info)}').join(' | ')}');
}

List<(String, PointerEvent)> events() => <(String, PointerEvent)>[
  ('added', const PointerAddedEvent()),
  ('added full', const PointerAddedEvent(viewId: 3, timeStamp: Duration(microseconds: 1234567), pointer: 4, kind: PointerDeviceKind.stylus,
      device: 5, position: Offset(10.5, -20.25), obscured: true, pressureMin: 0.1, pressureMax: 2.5, distance: 1.1,
      distanceMax: 7.7, radiusMin: 0.3, radiusMax: 9.9, orientation: 0.7, tilt: 0.2, embedderId: 12)),
  ('removed', const PointerRemovedEvent()),
  ('removed full', const PointerRemovedEvent(viewId: 3, timeStamp: Duration(milliseconds: 77), pointer: 6, kind: PointerDeviceKind.mouse,
      device: 2, position: Offset(1.25, 2.5), obscured: true, pressureMin: 0.5, pressureMax: 1.5, distanceMax: 3.3,
      radiusMin: 0.25, radiusMax: 4.75, embedderId: 8)),
  ('hover', const PointerHoverEvent()),
  ('hover full', const PointerHoverEvent(viewId: 1, timeStamp: Duration(seconds: 3), kind: PointerDeviceKind.mouse, pointer: 2,
      device: 3, position: Offset(100.125, 200.5), delta: Offset(-1.5, 2.25), buttons: 6, obscured: true, pressureMin: 0.2,
      pressureMax: 3.1, distance: 0.4, distanceMax: 5.5, size: 0.33, radiusMajor: 6.6, radiusMinor: 2.2, radiusMin: 1.1,
      radiusMax: 8.8, orientation: -0.5, tilt: 0.9, synthesized: true, embedderId: 99)),
  ('enter', const PointerEnterEvent()),
  ('enter full', const PointerEnterEvent(timeStamp: Duration(days: 1), pointer: 7, kind: PointerDeviceKind.invertedStylus, device: 1,
      position: Offset(3, 4), delta: Offset(0.1, 0.2), buttons: 1, obscured: true, pressureMin: 0.0, pressureMax: 0.0,
      distance: 2, distanceMax: 4, size: 1, radiusMajor: 2, radiusMinor: 1, radiusMin: 0.5, radiusMax: 3, orientation: 1,
      tilt: 1, down: true, synthesized: true, embedderId: 4)),
  ('exit', const PointerExitEvent()),
  ('exit full', const PointerExitEvent(timeStamp: Duration(hours: 2), pointer: 8, kind: PointerDeviceKind.unknown, device: 9,
      position: Offset(-3, -4), delta: Offset(-0.1, -0.2), buttons: 2, down: true, synthesized: true)),
  ('down', const PointerDownEvent()),
  ('down full', const PointerDownEvent(viewId: 2, timeStamp: Duration(microseconds: 999), pointer: 11, kind: PointerDeviceKind.stylus,
      device: 4, position: Offset(55.5, 66.25), buttons: 3, obscured: true, pressure: 0.75, pressureMin: 0.25, pressureMax: 1.75,
      distanceMax: 2.5, size: 0.125, radiusMajor: 3.5, radiusMinor: 1.5, radiusMin: 0.5, radiusMax: 4.5, orientation: 0.3,
      tilt: 0.6, embedderId: 21)),
  ('move', const PointerMoveEvent()),
  ('move full', const PointerMoveEvent(viewId: 2, timeStamp: Duration(microseconds: 16999), pointer: 11, kind: PointerDeviceKind.touch,
      device: 4, position: Offset(57.5, 60.25), delta: Offset(2.0, -6.0), buttons: 1, obscured: true, pressure: 0.85,
      pressureMin: 0.25, pressureMax: 1.75, distanceMax: 2.5, size: 0.125, radiusMajor: 3.5, radiusMinor: 1.5, radiusMin: 0.5,
      radiusMax: 4.5, orientation: 0.3, tilt: 0.6, platformData: 42, synthesized: true, embedderId: 22)),
  ('up', const PointerUpEvent()),
  ('up full', const PointerUpEvent(viewId: 2, timeStamp: Duration(microseconds: 33999), pointer: 11, kind: PointerDeviceKind.touch,
      device: 4, position: Offset(57.5, 60.25), buttons: 0, obscured: true, pressure: 0.05, pressureMin: 0.25, pressureMax: 1.75,
      distance: 0.5, distanceMax: 2.5, size: 0.125, radiusMajor: 3.5, radiusMinor: 1.5, radiusMin: 0.5, radiusMax: 4.5,
      orientation: 0.3, tilt: 0.6, embedderId: 23)),
  ('cancel', const PointerCancelEvent()),
  ('cancel full', const PointerCancelEvent(viewId: 2, timeStamp: Duration(microseconds: 1), pointer: 12, kind: PointerDeviceKind.mouse,
      device: 1, position: Offset(1, 1), buttons: 4, obscured: true, pressureMin: 0.5, pressureMax: 0.5, distance: 1, distanceMax: 2,
      size: 3, radiusMajor: 4, radiusMinor: 5, radiusMin: 6, radiusMax: 7, orientation: 8, tilt: 9, embedderId: 10)),
  ('scroll', const PointerScrollEvent()),
  ('scroll full', const PointerScrollEvent(viewId: 1, timeStamp: Duration(milliseconds: 5), kind: PointerDeviceKind.trackpad,
      device: 2, position: Offset(30, 40), scrollDelta: Offset(-3.5, 120.25), embedderId: 6)),
  ('inertia', const PointerScrollInertiaCancelEvent()),
  ('inertia full', const PointerScrollInertiaCancelEvent(viewId: 1, timeStamp: Duration(milliseconds: 6),
      kind: PointerDeviceKind.trackpad, device: 2, position: Offset(31, 41), embedderId: 7)),
  ('scale', const PointerScaleEvent()),
  ('scale full', const PointerScaleEvent(viewId: 1, timeStamp: Duration(milliseconds: 7), kind: PointerDeviceKind.touch, device: 2,
      position: Offset(32, 42), embedderId: 8, scale: 1.75)),
  ('pzstart', const PointerPanZoomStartEvent()),
  ('pzstart full', const PointerPanZoomStartEvent(viewId: 1, timeStamp: Duration(milliseconds: 8), device: 3, pointer: 4,
      position: Offset(33, 43), embedderId: 9, synthesized: true)),
  ('pzupdate', const PointerPanZoomUpdateEvent()),
  ('pzupdate full', const PointerPanZoomUpdateEvent(viewId: 1, timeStamp: Duration(milliseconds: 9), device: 3, pointer: 4,
      position: Offset(33, 43), embedderId: 10, pan: Offset(12.5, -7.25), panDelta: Offset(1.5, -0.75), scale: 1.2,
      rotation: 0.35, synthesized: true)),
  ('pzend', const PointerPanZoomEndEvent()),
  ('pzend full', const PointerPanZoomEndEvent(viewId: 1, timeStamp: Duration(milliseconds: 10), device: 3, pointer: 4,
      position: Offset(34, 44), embedderId: 11, synthesized: true)),
];

List<(String, Matrix4)> matrices() {
  final perspective = Matrix4.identity()..setEntry(3, 0, 0.001)..setEntry(3, 1, -0.002);
  return <(String, Matrix4)>[
    ('translate', Matrix4.translationValues(10, -20, 0)),
    ('scale', Matrix4.diagonal3Values(2.5, 0.5, 1)),
    ('rotate', Matrix4.rotationZ(0.7)..setTranslationRaw(3, 4, 0)),
    ('perspective', perspective),
    ('composite', (Matrix4.identity()..scale(1 / 3.0, 1 / 7.0, 1.0)).multiplied(Matrix4.translationValues(-10.5, 30.25, 0))),
  ];
}

class Member extends GestureArenaMember {
  Member(this.name);
  final String name;
  GestureArenaEntry? entry;
  @override
  void acceptGesture(int pointer) => log('  $name accept $pointer');
  @override
  void rejectGesture(int pointer) => log('  $name reject $pointer');
  @override
  String toString() => 'Member($name)';
}

class HtTarget implements HitTestTarget {
  HtTarget(this.name);
  final String name;
  @override
  void handleEvent(PointerEvent event, HitTestEntry entry) {}
  @override
  String toString() => 'HtTarget($name)';
}

void main() {
  test('pointer events', () {
    debugPrint = (String? message, {int? wrapWidth}) => log('debugPrint $message');

    // ---- events, transformed, copyWith
    for (final (String label, PointerEvent e) in events()) {
      describe(label, e);
      log('  transformed(null) identical ${identical(e.transformed(null), e)}');
      for (final (String mname, Matrix4 m) in matrices()) {
        final PointerEvent t = e.transformed(m);
        describe('$label * $mname', t);
        log('  original identical ${identical(t.original, e)} again ${identical(t.transformed(m), t)} '
            'back ${identical(t.transformed(null), e)} twice-original ${identical(t.transformed(Matrix4.identity()).original, e)}');
        describe('$label * $mname copy', t.copyWith());
        describe('$label * $mname copy2', t.copyWith(
          viewId: 5, timeStamp: const Duration(microseconds: 4242), pointer: 77, device: 13, position: const Offset(1.5, 2.5),
          delta: const Offset(0.25, -0.5), buttons: 2, obscured: false, pressure: 0.6, pressureMin: 0.3, pressureMax: 0.9,
          distance: 4.4, distanceMax: 8.8, size: 0.7, radiusMajor: 2.2, radiusMinor: 1.1, radiusMin: 0.6, radiusMax: 3.3,
          orientation: 0.15, tilt: 0.25, synthesized: true, embedderId: 31));
      }
      describe('$label copy', e.copyWith());
    }
    const hover = PointerHoverEvent(position: Offset(5, 6), delta: Offset(1, 1), kind: PointerDeviceKind.mouse, device: 3,
        buttons: 1, embedderId: 4);
    final Matrix4 tm = Matrix4.translationValues(-1, -2, 0);
    describe('enter.fromMouseEvent', PointerEnterEvent.fromMouseEvent(hover));
    describe('exit.fromMouseEvent', PointerExitEvent.fromMouseEvent(hover));
    describe('enter.fromMouseEvent transformed', PointerEnterEvent.fromMouseEvent(hover.transformed(tm)));
    describe('exit.fromMouseEvent move', PointerExitEvent.fromMouseEvent(const PointerMoveEvent(position: Offset(9, 9), pointer: 3)));
    describe('removed with original', const PointerRemovedEvent(original: PointerRemovedEvent(position: Offset(1, 2))));
    describe('removed with original transformed',
        const PointerRemovedEvent(position: Offset(7, 8), original: PointerRemovedEvent(position: Offset(1, 2))).transformed(tm));

    // Scroll respond and the copy's respond.
    final scroll = PointerScrollEvent(onRespond: ({required bool allowPlatformDefault}) => log('respond $allowPlatformDefault'));
    scroll.respond(allowPlatformDefault: true);
    (scroll.transformed(tm) as PointerScrollEvent).respond(allowPlatformDefault: false);
    (scroll.copyWith() as PointerScrollEvent).respond(allowPlatformDefault: true);
    (scroll.transformed(tm).copyWith() as PointerScrollEvent).respond(allowPlatformDefault: false);
    (scroll.copyWith(onRespond: ({required bool allowPlatformDefault}) => log('other $allowPlatformDefault')) as PointerScrollEvent)
        .respond(allowPlatformDefault: true);

    // ---- statics
    for (final (String mname, Matrix4 m) in matrices()) {
      log('transformPosition $mname ${off(PointerEvent.transformPosition(m, const Offset(12.5, -7.25)))}');
      log('transformDelta $mname ${off(PointerEvent.transformDeltaViaPositions(untransformedEndPosition: const Offset(12.5, -7.25), untransformedDelta: const Offset(3.5, 1.25), transform: m))} '
          '${off(PointerEvent.transformDeltaViaPositions(untransformedEndPosition: const Offset(12.5, -7.25), transformedEndPosition: const Offset(100, 100), untransformedDelta: const Offset(3.5, 1.25), transform: m))}');
      log('removePerspective $mname ${mat(PointerEvent.removePerspectiveTransform(m))}');
    }
    log('transformPosition null ${off(PointerEvent.transformPosition(null, const Offset(1, 2)))}');
    for (final PointerDeviceKind kind in PointerDeviceKind.values) {
      log('slops ${kind.name} ${computeHitSlop(kind, null)} ${computePanSlop(kind, null)} ${computeScaleSlop(kind)} '
          '${computeHitSlop(kind, const DeviceGestureSettings(touchSlop: 7.5))} ${computePanSlop(kind, const DeviceGestureSettings(touchSlop: 7.5))} '
          '${computeHitSlop(kind, const DeviceGestureSettings())} ${computePanSlop(kind, const DeviceGestureSettings())}');
    }
    log('settings ${const DeviceGestureSettings(touchSlop: 7.5)} ${const DeviceGestureSettings()} '
        '${const DeviceGestureSettings(touchSlop: 7.5).panSlop} ${const DeviceGestureSettings().panSlop} '
        '${const DeviceGestureSettings(touchSlop: 7.5) == const DeviceGestureSettings(touchSlop: 7.5)} '
        '${const DeviceGestureSettings(touchSlop: 7.5) == const DeviceGestureSettings()}');
    log('buttons ${[for (int i = 1; i <= 8; i++) nthMouseButton(i)]} ${[for (int i = 1; i <= 8; i++) nthStylusButton(i)]} '
        '${[for (final int b in <int>[0, 1, 2, 3, 6, 0x220, 0x400, -4]) '${smallestButton(b)}/${isSingleButton(b)}']}');

    // ---- converter
    final data = <ui.PointerData>[
      for (final ui.PointerChange change in ui.PointerChange.values)
        for (final PointerDeviceKind kind in PointerDeviceKind.values)
          if (!(kind == PointerDeviceKind.trackpad &&
              <ui.PointerChange>[ui.PointerChange.down, ui.PointerChange.move, ui.PointerChange.up, ui.PointerChange.cancel].contains(change)))
            ui.PointerData(
              viewId: kind == PointerDeviceKind.unknown ? 99 : 0, embedderId: change.index * 10 + kind.index,
              timeStamp: Duration(microseconds: 1000 + change.index * 37 + kind.index), change: change, kind: kind,
              device: kind.index, pointerIdentifier: change.index + 1, physicalX: 301.5 + kind.index, physicalY: 207.25 - change.index,
              physicalDeltaX: 3.75, physicalDeltaY: -1.5, buttons: change.index % 3, obscured: kind.index.isOdd,
              synthesized: change.index.isEven, pressure: 0.55, pressureMin: 0.05, pressureMax: 1.95, distance: 0.7,
              distanceMax: 1.4, size: 0.21, radiusMajor: 7.5, radiusMinor: 3.25, radiusMin: 1.5, radiusMax: 10.5,
              orientation: 0.12, tilt: 0.34, platformData: 77, scrollDeltaX: 9.5, scrollDeltaY: -3.5, panX: 21.5, panY: -14.25,
              panDeltaX: 1.5, panDeltaY: -2.5, scale: 1.3, rotation: 0.4),
      for (final ui.PointerSignalKind signal in ui.PointerSignalKind.values)
        ui.PointerData(signalKind: signal, change: ui.PointerChange.hover, kind: PointerDeviceKind.mouse, physicalX: 50, physicalY: 70,
            scrollDeltaX: 30.5, scrollDeltaY: -60.25, scale: 0.8, embedderId: 500 + signal.index,
            onRespond: ({bool allowPlatformDefault = false}) => log('datum respond ${signal.name} $allowPlatformDefault')),
      const ui.PointerData(signalKind: ui.PointerSignalKind.scroll, scrollDeltaX: double.infinity, embedderId: 600),
      const ui.PointerData(signalKind: ui.PointerSignalKind.scroll, scrollDeltaY: double.nan, embedderId: 601),
      const ui.PointerData(change: ui.PointerChange.down, kind: PointerDeviceKind.mouse, buttons: 0, embedderId: 602),
      const ui.PointerData(change: ui.PointerChange.down, kind: PointerDeviceKind.stylus, buttons: 0, embedderId: 603),
      const ui.PointerData(change: ui.PointerChange.move, kind: PointerDeviceKind.invertedStylus, buttons: 4, embedderId: 604),
    ];
    for (final double dpr in <double>[1, 2, 3.5, 0]) {
      final List<PointerEvent> out = PointerEventConverter.expand(data, (int viewId) => viewId == 99 ? null : dpr).toList();
      log('expand dpr $dpr: ${out.length}');
      for (final e in out) {
        log('  ${fields(e)}');
        if (e is PointerScrollEvent) {
          e.respond(allowPlatformDefault: true);
        }
      }
    }
    log('datum ${data.first} ${data[3].toStringFull()}');

    // ---- resampler
    List<PointerEvent> script(int n) => switch (n) {
      0 => <PointerEvent>[
        const PointerAddedEvent(timeStamp: Duration(microseconds: 1000), position: Offset(0, 50)),
        const PointerHoverEvent(timeStamp: Duration(microseconds: 2000), position: Offset(10, 40), delta: Offset(10, -10)),
        const PointerDownEvent(timeStamp: Duration(microseconds: 2000), position: Offset(10, 40), pointer: 3),
        const PointerMoveEvent(timeStamp: Duration(microseconds: 3333), position: Offset(20.5, 30.25), delta: Offset(10.5, -9.75), pointer: 3),
        const PointerMoveEvent(timeStamp: Duration(microseconds: 4111), position: Offset(31, 19), delta: Offset(10.5, -11.25), pointer: 3, buttons: 3),
        const PointerUpEvent(timeStamp: Duration(microseconds: 4999), position: Offset(31, 19), pointer: 3),
        const PointerHoverEvent(timeStamp: Duration(microseconds: 6001), position: Offset(40, 10), delta: Offset(9, -9)),
        const PointerRemovedEvent(timeStamp: Duration(microseconds: 7003), position: Offset(40, 10)),
      ],
      2 => <PointerEvent>[
        const PointerUpEvent(timeStamp: Duration(microseconds: 1000), position: Offset(5, 7), pointer: 1),
        const PointerRemovedEvent(timeStamp: Duration(microseconds: 1200), position: Offset(5, 7)),
      ],
      _ => <PointerEvent>[
        const PointerDownEvent(timeStamp: Duration(microseconds: 10000), position: Offset(100, 100), pointer: 9),
        for (int i = 1; i <= 12; i++)
          PointerMoveEvent(timeStamp: Duration(microseconds: 10000 + i * 4100), position: Offset(100 + i * 3.7, 100 - i * i * 0.3),
              delta: Offset(3.7, -(2 * i - 1) * 0.3), pointer: 9, pressure: 0.5 + i / 100),
        const PointerUpEvent(timeStamp: Duration(microseconds: 60000), position: Offset(144.4, 56.8), pointer: 9),
      ],
    };
    for (final int n in <int>[0, 1, 2]) {
      for (final (int start, int step, int nextOffset) in <(int, int, int)>[(500, 1000, 0), (0, 16667, 16667), (3000, 7000, 3500), (9000, 2500, 2500), (500, 1500, 1500), (999, 1000, 1000), (14100, 4100, 4100)]) {
        final resampler = PointerEventResampler();
        for (final e in script(n)) {
          resampler.addEvent(e);
        }
        log('resample $n $start $step $nextOffset');
        var t = start;
        for (int k = 0; k < 40 && (resampler.hasPendingEvents || k < 3); k++) {
          resampler.sample(Duration(microseconds: t), Duration(microseconds: t + nextOffset), (PointerEvent e) => log('  $t ${fields(e)}'));
          log('  state $t ${resampler.hasPendingEvents} ${resampler.isTracked} ${resampler.isDown}');
          t += step;
        }
        resampler.stop((PointerEvent e) => log('  stop ${fields(e)}'));
      }
    }
    final stopper = PointerEventResampler();
    for (final e in script(1)) {
      stopper.addEvent(e);
    }
    stopper.sample(const Duration(microseconds: 20000), Duration.zero, (PointerEvent e) => log('  pre-stop ${fields(e)}'));
    stopper.stop((PointerEvent e) => log('  stop ${fields(e)}'));
    log('  after stop ${stopper.hasPendingEvents} ${stopper.isTracked} ${stopper.isDown}');

    // ---- arena
    debugPrintGestureArenaDiagnostics = true;
    void arenaScript(String name, void Function(GestureArenaManager arena, List<Member> m, void Function() flush) body) {
      fakeAsync((FakeAsync async) {
        log('arena $name');
        final arena = GestureArenaManager();
        final m = <Member>[Member('a'), Member('b'), Member('c')];
        body(arena, m, async.flushMicrotasks);
        async.flushMicrotasks();
        log('  end');
      });
    }
    arenaScript('sweep', (arena, m, flush) {
      for (final x in m) {
        x.entry = arena.add(1, x);
      }
      arena.close(1);
      arena.sweep(1);
      arena.sweep(1);
    });
    arenaScript('accept closed', (arena, m, flush) {
      for (final x in m) {
        x.entry = arena.add(2, x);
      }
      arena.close(2);
      m[1].entry!.resolve(GestureDisposition.accepted);
      m[0].entry!.resolve(GestureDisposition.accepted);
    });
    arenaScript('eager', (arena, m, flush) {
      for (final x in m) {
        x.entry = arena.add(3, x);
      }
      m[2].entry!.resolve(GestureDisposition.accepted);
      m[0].entry!.resolve(GestureDisposition.accepted);
      arena.close(3);
    });
    arenaScript('reject to one', (arena, m, flush) {
      for (final x in m) {
        x.entry = arena.add(4, x);
      }
      arena.close(4);
      m[0].entry!.resolve(GestureDisposition.rejected);
      m[2].entry!.resolve(GestureDisposition.rejected);
      log('  before flush');
      flush();
    });
    arenaScript('reject all', (arena, m, flush) {
      for (final x in m) {
        x.entry = arena.add(5, x);
      }
      arena.close(5);
      for (final x in m) {
        x.entry!.resolve(GestureDisposition.rejected);
      }
    });
    arenaScript('hold', (arena, m, flush) {
      for (final x in m.take(2)) {
        x.entry = arena.add(6, x);
      }
      arena.close(6);
      arena.hold(6);
      arena.sweep(6);
      log('  held');
      arena.release(6);
      arena.release(6);
    });
    arenaScript('hold accept', (arena, m, flush) {
      for (final x in m.take(2)) {
        x.entry = arena.add(7, x);
      }
      arena.hold(7);
      arena.close(7);
      m[1].entry!.resolve(GestureDisposition.accepted);
      arena.release(7);
    });
    arenaScript('single default', (arena, m, flush) {
      m[0].entry = arena.add(8, m[0]);
      arena.close(8);
      log('  closed');
      flush();
      m[0].entry!.resolve(GestureDisposition.accepted);
    });
    arenaScript('single then sweep', (arena, m, flush) {
      m[0].entry = arena.add(9, m[0]);
      arena.close(9);
      arena.sweep(9);
    });
    arenaScript('eager rejected open', (arena, m, flush) {
      for (final x in m) {
        x.entry = arena.add(10, x);
      }
      m[0].entry!.resolve(GestureDisposition.accepted);
      m[0].entry!.resolve(GestureDisposition.rejected);
      m[1].entry!.resolve(GestureDisposition.accepted);
      arena.close(10);
    });
    arenaScript('two pointers', (arena, m, flush) {
      m[0].entry = arena.add(11, m[0]);
      m[1].entry = arena.add(12, m[1]);
      m[2].entry = arena.add(11, m[2]);
      arena.close(12);
      arena.close(11);
      arena.close(13);
      arena.sweep(13);
      arena.hold(13);
      arena.release(13);
      flush();
      arena.sweep(11);
    });
    arenaScript('accept during reject', (arena, m, flush) {
      for (final x in m) {
        x.entry = arena.add(14, x);
      }
      arena.close(14);
      m[1].entry!.resolve(GestureDisposition.rejected);
      m[2].entry!.resolve(GestureDisposition.accepted);
      m[0].entry!.resolve(GestureDisposition.accepted);
    });
    debugPrintGestureArenaDiagnostics = false;

    // ---- router
    final router = PointerRouter();
    final Matrix4 rm = (Matrix4.identity()..scale(0.5, 0.25, 1.0)).multiplied(Matrix4.translationValues(-10, -30, 0));
    void r1(PointerEvent e) => log('  r1 ${fields(e)}');
    void r2(PointerEvent e) {
      log('  r2 ${fields(e)}');
      router.removeRoute(2, r1);
      router.addRoute(2, r1);
    }
    void g1(PointerEvent e) => log('  g1 ${fields(e)}');
    void bad(PointerEvent e) => throw StateError('route failed');
    router.addRoute(2, r1, rm);
    router.addRoute(2, r2);
    router.addGlobalRoute(g1, Matrix4.rotationZ(0.3));
    router.addRoute(3, bad);
    final saved = FlutterError.onError;
    FlutterError.onError = (FlutterErrorDetails d) => log('  error ${d.exception} | ${d.library} | ${d.context} | '
        '${d.informationCollector!().map((DiagnosticsNode n) => mask(n)).join(' ; ')}');
    log('route 2');
    router.route(const PointerDownEvent(pointer: 2, position: Offset(16, 36)));
    log('route 2 again');
    router.route(const PointerMoveEvent(pointer: 2, position: Offset(20, 40), delta: Offset(4, 4)));
    log('route 3');
    router.route(const PointerDownEvent(pointer: 3, position: Offset(1, 1)));
    router.removeRoute(2, r1);
    router.removeRoute(2, r2);
    router.removeGlobalRoute(g1);
    log('route 2 removed');
    router.route(const PointerUpEvent(pointer: 2));
    log('global count ${router.debugGlobalRouteCount}');
    FlutterError.onError = saved;

    // ---- signal resolver
    final resolver = PointerSignalResolver();
    final original = PointerScrollEvent(scrollDelta: const Offset(0, 10),
        onRespond: ({required bool allowPlatformDefault}) => log('  scroll respond $allowPlatformDefault'));
    resolver.resolve(original);
    final t1 = original.transformed(Matrix4.translationValues(5, 5, 0)) as PointerSignalEvent;
    final t2 = original.transformed(Matrix4.translationValues(7, 7, 0)) as PointerSignalEvent;
    resolver.register(t1, (PointerSignalEvent e) => log('  first ${fields(e)}'));
    resolver.register(t2, (PointerSignalEvent e) => log('  second ${fields(e)}'));
    resolver.resolve(original);
    resolver.register(original, (PointerSignalEvent e) => throw StateError('resolve failed'));
    FlutterError.onError = (FlutterErrorDetails d) => log('  error ${d.exception} | ${d.library} | ${d.context} | '
        '${d.informationCollector!().map((DiagnosticsNode n) => mask(n)).join(' ; ')}');
    resolver.resolve(original);
    FlutterError.onError = saved;
    resolver.resolve(original);

    // ---- hit test
    final result = HitTestResult();
    result.add(HitTestEntry(HtTarget('root')));
    result.pushOffset(const Offset(10, 20));
    result.add(HitTestEntry(HtTarget('offset')));
    result.pushTransform(Matrix4.rotationZ(0.25)..setTranslationRaw(1, 2, 0));
    final wrapped = HitTestResult.wrap(result);
    wrapped.add(HitTestEntry(HtTarget('rotated')));
    result.pushOffset(const Offset(-3.5, 4.25));
    result.popTransform();
    result.pushTransform(Matrix4.diagonal3Values(2, 3, 1));
    result.add(HitTestEntry(HtTarget('scaled')));
    result.popTransform();
    result.popTransform();
    result.add(HitTestEntry(HtTarget('back')));
    for (final HitTestEntry e in wrapped.path) {
      log('hit ${mask(e)} ${mat(e.transform)}');
    }
    log('hit result ${mask(result)}');
    log('hit empty ${HitTestResult()}');

    // ---- details, PointerData, KeyData
    for (final Diagnosticable d in <Diagnosticable>[
      DragDownDetails(globalPosition: const Offset(1, 2)),
      DragDownDetails(globalPosition: const Offset(1, 2), localPosition: const Offset(3, 4)),
      DragStartDetails(globalPosition: const Offset(5, 6), sourceTimeStamp: const Duration(milliseconds: 12), kind: PointerDeviceKind.stylus),
      DragUpdateDetails(globalPosition: const Offset(7, 8), delta: const Offset(0, 2.5), primaryDelta: 2.5,
          sourceTimeStamp: const Duration(microseconds: 3), kind: PointerDeviceKind.touch),
      DragEndDetails(velocity: const Velocity(pixelsPerSecond: Offset(100.25, 0)), primaryVelocity: 100.25),
      DragEndDetails(),
    ]) {
      log('details ${mask(d)} | ${mask(d.toDiagnosticsNode().toStringDeep())}');
    }
    log('pointerdata ${const ui.PointerData(viewId: 4, physicalX: 1.5, physicalY: 2.25)} '
        '${const ui.PointerData(signalKind: ui.PointerSignalKind.scale, timeStamp: Duration(seconds: 61)).toStringFull()}');
    for (final ui.KeyData k in <ui.KeyData>[
      const ui.KeyData(timeStamp: Duration(milliseconds: 1500), type: ui.KeyEventType.down, physical: 0x70004, logical: 0x61,
          character: 'a', synthesized: false),
      const ui.KeyData(timeStamp: Duration.zero, type: ui.KeyEventType.repeat, physical: 0x100070028, logical: 0x200000100,
          character: '\n', synthesized: true, deviceType: ui.KeyEventDeviceType.joystick),
      const ui.KeyData(timeStamp: Duration(microseconds: 7), type: ui.KeyEventType.up, physical: 0, logical: 0x1100000042,
          character: null, synthesized: false, deviceType: ui.KeyEventDeviceType.directionalPad),
      const ui.KeyData(timeStamp: Duration(microseconds: 8), type: ui.KeyEventType.down, physical: 1, logical: 0x1800000001,
          character: '\té', synthesized: false, deviceType: ui.KeyEventDeviceType.hdmi),
      const ui.KeyData(timeStamp: Duration(microseconds: 9), type: ui.KeyEventType.down, physical: 1, logical: 0x3300000001,
          character: '\u{1F600}', synthesized: false, deviceType: ui.KeyEventDeviceType.gamepad),
    ]) {
      log('key $k | ${k.toStringFull()}');
    }
    debugPrint = debugPrintThrottled;
  });

  tearDownAll(() {
    File('build/pointer_events.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': lines}));
  });
}
