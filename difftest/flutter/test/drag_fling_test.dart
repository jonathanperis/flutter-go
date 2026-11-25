// Reference side of flutter-go's drag-fling tests: tester.fling on a
// ListView (real drag gesture, VelocityTracker, ClampingScrollPhysics),
// then the offset after each 16ms frame. Output: build/drag_fling.json.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

// name: [start offset, drag dy, speed px/s]
const Map<String, List<double>> scenes = <String, List<double>>{
  'drag_fling_up_1000': <double>[0, -300, 1000],
  'drag_fling_up_3000': <double>[0, -200, 3000],
  'drag_fling_down_2000': <double>[2000, 150, 2000],
  'drag_fling_slow_100': <double>[0, -100, 100],
  'drag_too_slow_40': <double>[0, -60, 40],
  'drag_fling_max_velocity': <double>[0, -400, 20000],
  // Items with tap handlers: the drag competes with a tap in the arena.
  'tappable_drag_fling_up_1000': <double>[0, -300, 1000],
};

void main() {
  final Map<String, List<double>> out = <String, List<double>>{};
  for (final MapEntry<String, List<double>> e in scenes.entries) {
    testWidgets(e.key, (WidgetTester tester) async {
      final ScrollController c = ScrollController(initialScrollOffset: e.value[0]);
      await tester.pumpWidget(Directionality(
        textDirection: TextDirection.ltr,
        child: ScrollConfiguration(
          behavior: const ScrollBehavior().copyWith(overscroll: false, scrollbars: false),
          child: ListView.builder(
            controller: c,
            itemCount: 1000,
            itemExtent: 50,
            itemBuilder: (BuildContext context, int i) => e.key.startsWith('tappable')
                ? GestureDetector(onTap: () {}, child: const ColoredBox(color: Color(0xffeeeeee)))
                : const SizedBox(),
          ),
        ),
      ));
      final List<double> s = <double>[c.offset];
      await tester.fling(find.byType(ListView), Offset(0, e.value[1]), e.value[2]);
      s.add(c.offset);
      for (int i = 0; i < 60; i++) {
        await tester.pump(const Duration(milliseconds: 16));
        s.add(c.offset);
      }
      out[e.key] = s;
    });
  }
  // An accelerating drag with explicit timestamps (y = -1.5·i² every 8 ms):
  // a quadratic fit and a straight line give different release velocities.
  testWidgets('accelerating_drag', (WidgetTester tester) async {
    final ScrollController c = ScrollController();
    await tester.pumpWidget(Directionality(
      textDirection: TextDirection.ltr,
      child: ScrollConfiguration(
        behavior: const ScrollBehavior().copyWith(overscroll: false, scrollbars: false),
        child: ListView.builder(
          controller: c,
          itemCount: 1000,
          itemExtent: 50,
          itemBuilder: (BuildContext context, int i) => const SizedBox(),
        ),
      ),
    ));
    final Offset center = tester.getCenter(find.byType(ListView));
    final TestGesture g = await tester.startGesture(center);
    for (int i = 1; i <= 15; i++) {
      await g.moveTo(center + Offset(0, -1.5 * i * i), timeStamp: Duration(milliseconds: 8 * i));
    }
    await g.up(timeStamp: const Duration(milliseconds: 120));
    final List<double> s = <double>[c.offset];
    for (int i = 0; i < 60; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      s.add(c.offset);
    }
    out['accelerating_drag'] = s;
  });

  tearDownAll(() {
    File('build/drag_fling.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
