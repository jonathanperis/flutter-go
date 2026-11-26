// Reference side of flutter-go's scroll gesture differential test: real
// pointer gestures on a ListView through the gestures library on each
// platform's scroll configuration (Android: clamping physics and the
// default VelocityTracker; iOS: bouncing physics, minFlingVelocity 100 and
// IOSScrollViewFlingVelocityTracker; macOS: bouncing with fast
// deceleration and MacOSScrollViewFlingVelocityTracker): flings at several
// speeds, a timed drag with a pause (the bouncing physics' motion-start
// threshold uses the events' time stamps), a tap on an item (its global
// position), and a touch that holds a fling under a GestureDetector
// wrapping the list (the drag waits for the slop, the hold stops the fling,
// the tap wins on up). The Go twin is ../../scroll_gestures_test.go.
// Output: build/scroll_gestures.json.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

final Map<String, List<String>> out = <String, List<String>>{};

Widget list(ScrollController c, {bool tappable = false, List<String>? log}) => Directionality(
  textDirection: TextDirection.ltr,
  child: ScrollConfiguration(
    behavior: const ScrollBehavior().copyWith(overscroll: false, scrollbars: false),
    child: ListView.builder(
      controller: c,
      itemCount: 1000,
      itemExtent: 50,
      itemBuilder: (BuildContext context, int i) => tappable
          ? GestureDetector(
              onTapUp: (TapUpDetails d) => log!.add('tapUp $i ${d.globalPosition.dx} ${d.globalPosition.dy}'),
              onTap: () => log!.add('tap $i'),
              child: const ColoredBox(color: Color(0xffeeeeee)),
            )
          : const SizedBox(),
    ),
  ),
);

TargetPlatform platformOf(String name) => name.startsWith('ios')
    ? TargetPlatform.iOS
    : name.startsWith('mac')
    ? TargetPlatform.macOS
    : TargetPlatform.android;

void main() {
  // name: [drag dy, speed]
  const flings = <String, List<double>>{
    'android_fling_up_1500': <double>[-300, 1500],
    'ios_fling_up_1000': <double>[-300, 1000],
    'ios_fling_up_75': <double>[-100, 75],
    'ios_fling_down_2000': <double>[150, 2000],
    'mac_fling_up_1500': <double>[-300, 1500],
    'mac_fling_up_150': <double>[-100, 150],
  };
  for (final MapEntry<String, List<double>> e in flings.entries) {
    testWidgets(e.key, (WidgetTester tester) async {
      debugDefaultTargetPlatformOverride = platformOf(e.key);
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      final c = ScrollController(initialScrollOffset: e.key.contains('down') ? 2000 : 0);
      await tester.pumpWidget(list(c));
      final s = <String>['${c.offset}'];
      await tester.fling(find.byType(ListView), Offset(0, e.value[0]), e.value[1]);
      s.add('${c.offset}');
      for (var i = 0; i < 90; i++) {
        await tester.pump(const Duration(milliseconds: 16));
        s.add('${c.offset}');
      }
      out[e.key] = s;
    });
  }

  for (final name in <String>['android_drag_pause', 'ios_drag_pause', 'mac_drag_pause']) {
    testWidgets(name, (WidgetTester tester) async {
      debugDefaultTargetPlatformOverride = platformOf(name);
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      final c = ScrollController(initialScrollOffset: 500);
      await tester.pumpWidget(list(c));
      final s = <String>[];
      final Offset center = tester.getCenter(find.byType(ListView));
      final TestGesture g = await tester.startGesture(center);
      // Moves of 1, 2, 1, then a 80 ms pause, then small moves again.
      const moves = <List<num>>[
        <num>[8, -1], <num>[16, -3], <num>[24, -4], <num>[104, -5], <num>[112, -6],
        <num>[120, -8], <num>[128, -12], <num>[136, -20], <num>[144, -40],
      ];
      for (final m in moves) {
        await g.moveTo(center + Offset(0, m[1].toDouble()), timeStamp: Duration(milliseconds: m[0].toInt()));
        s.add('${c.offset}');
      }
      await g.up(timeStamp: const Duration(milliseconds: 152));
      s.add('${c.offset}');
      for (var i = 0; i < 60; i++) {
        await tester.pump(const Duration(milliseconds: 16));
        s.add('${c.offset}');
      }
      out[name] = s;
    });
  }

  testWidgets('android_tap_item', (WidgetTester tester) async {
    final log = <String>[];
    final c = ScrollController(initialScrollOffset: 30);
    await tester.pumpWidget(list(c, tappable: true, log: log));
    await tester.tapAt(const Offset(400, 125.5));
    await tester.pump();
    await tester.tapAt(const Offset(17.25, 590));
    await tester.pump();
    log.add('offset ${c.offset}');
    out['android_tap_item'] = log;
  });

  testWidgets('android_hold_under_outer_tap', (WidgetTester tester) async {
    final log = <String>[];
    final c = ScrollController();
    await tester.pumpWidget(
      GestureDetector(
        onTap: () => log.add('outer tap'),
        child: list(c),
      ),
    );
    (c.position as ScrollPositionWithSingleContext).goBallistic(3000);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    log.add('flinging ${c.offset}');
    final TestGesture g = await tester.startGesture(tester.getCenter(find.byType(ListView)));
    for (var i = 0; i < 5; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      log.add('held ${c.offset}');
    }
    await g.up();
    for (var i = 0; i < 5; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      log.add('after ${c.offset}');
    }
    out['android_hold_under_outer_tap'] = log;
  });

  testWidgets('ios_tappable_drag_stop', (WidgetTester tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
    addTearDown(() => debugDefaultTargetPlatformOverride = null);
    final log = <String>[];
    final c = ScrollController(initialScrollOffset: 500);
    await tester.pumpWidget(list(c, tappable: true, log: log));
    final Offset center = tester.getCenter(find.byType(ListView));
    final TestGesture g = await tester.startGesture(center);
    const moves = <List<num>>[
      <num>[50, -10], <num>[100, -25], <num>[120, -25], <num>[130, -27], <num>[140, -30],
      <num>[150, -34], <num>[260, -34], <num>[270, -35], <num>[280, -37], <num>[290, -45],
    ];
    for (final m in moves) {
      await g.moveTo(center + Offset(0, m[1].toDouble()), timeStamp: Duration(milliseconds: m[0].toInt()));
      log.add('${c.offset}');
    }
    await g.up(timeStamp: const Duration(milliseconds: 300));
    for (var i = 0; i < 30; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      log.add('${c.offset}');
    }
    out['ios_tappable_drag_stop'] = log;
  });

  testWidgets('ios_hold_overscroll_under_outer_tap', (WidgetTester tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
    addTearDown(() => debugDefaultTargetPlatformOverride = null);
    final log = <String>[];
    final c = ScrollController();
    await tester.pumpWidget(GestureDetector(onTap: () => log.add('outer tap'), child: list(c)));
    c.jumpTo(-100);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 48));
    log.add('springing ${c.offset}');
    final TestGesture g = await tester.startGesture(tester.getCenter(find.byType(ListView)));
    for (var i = 0; i < 3; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      log.add('held ${c.offset}');
    }
    await g.up();
    for (var i = 0; i < 30; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      log.add('after ${c.offset}');
    }
    out['ios_hold_overscroll_under_outer_tap'] = log;
  });

  tearDownAll(() {
    File('build/scroll_gestures.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
