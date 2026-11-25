// Reference side of flutter-go's bouncing-physics tests (iOS and macOS
// platforms). Records the scroll offset after each step. Output:
// build/bouncing.json.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

// name: [platform (0 iOS, 1 macOS), start offset, velocity]
const Map<String, List<double>> flings = <String, List<double>>{
  'ios_fling_within': <double>[0, 0, 1500],
  'ios_fling_overshoot_end': <double>[0, 48000, 5000],
  'ios_fling_overshoot_start': <double>[0, 300, -4000],
  'mac_fling_within': <double>[1, 0, 3000],
  'mac_fling_overshoot_end': <double>[1, 48000, 6000],
};

Widget list(ScrollController c) => Directionality(
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
);

void main() {
  final Map<String, List<double>> out = <String, List<double>>{};

  void platform(double p) {
    debugDefaultTargetPlatformOverride = p == 0 ? TargetPlatform.iOS : TargetPlatform.macOS;
  }

  Future<void> frames(WidgetTester tester, ScrollController c, List<double> s, int n) async {
    for (int i = 0; i < n; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      s.add(c.offset);
    }
  }

  for (final MapEntry<String, List<double>> e in flings.entries) {
    testWidgets(e.key, (WidgetTester tester) async {
      platform(e.value[0]);
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      final ScrollController c = ScrollController();
      await tester.pumpWidget(list(c));
      c.jumpTo(e.value[1]);
      await tester.pump();
      (c.position as ScrollPositionWithSingleContext).goBallistic(e.value[2]);
      await tester.pump();
      final List<double> s = <double>[c.offset];
      await frames(tester, c, s, 90);
      out[e.key] = s;
    });
  }

  testWidgets('ios_jump_out_of_range', (WidgetTester tester) async {
    platform(0);
    addTearDown(() => debugDefaultTargetPlatformOverride = null);
    final ScrollController c = ScrollController();
    await tester.pumpWidget(list(c));
    c.jumpTo(-150);
    final List<double> s = <double>[c.offset];
    await tester.pump();
    s.add(c.offset);
    await frames(tester, c, s, 40);
    out['ios_jump_out_of_range'] = s;
  });

  for (final double p in <double>[0, 1]) {
    final String name = p == 0 ? 'ios_drag_overscroll_top' : 'mac_drag_overscroll_top';
    testWidgets(name, (WidgetTester tester) async {
      platform(p);
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      final ScrollController c = ScrollController();
      await tester.pumpWidget(list(c));
      final Offset center = tester.getCenter(find.byType(ListView));
      final TestGesture g = await tester.startGesture(center);
      final List<double> s = <double>[];
      for (int i = 1; i <= 20; i++) {
        await g.moveTo(center + Offset(0, 10.0 * i), timeStamp: Duration(milliseconds: 8 * i));
        s.add(c.offset);
      }
      // Hold still, then release with no velocity: the list springs back.
      await tester.pump(const Duration(milliseconds: 100));
      await g.up(timeStamp: const Duration(milliseconds: 260));
      await tester.pump();
      s.add(c.offset);
      await frames(tester, c, s, 40);
      out[name] = s;
    });
  }

  tearDownAll(() {
    File('build/bouncing.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
