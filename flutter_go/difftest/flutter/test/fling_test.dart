// Reference side of flutter-go's fling differential tests: starts a ballistic
// scroll with a fixed velocity (ClampingScrollPhysics, the flutter_test
// default) and records [offset, frame still scheduled ? 1 : 0] every 16ms. Output: build/fling.json.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

const Map<String, List<double>> scenes = <String, List<double>>{
  // name: [start offset, velocity px/s]
  'fling_down': <double>[0, 1500],
  'fling_up': <double>[3000, -4000],
  'fling_hits_end': <double>[48000, 5000],
  'fling_below_tolerance': <double>[1000, 5],
};

void main() {
  final Map<String, List<List<double>>> out = <String, List<List<double>>>{};
  for (final MapEntry<String, List<double>> e in scenes.entries) {
    testWidgets(e.key, (WidgetTester tester) async {
      final ScrollController c = ScrollController();
      // No overscroll glow or scrollbars: flutter-go has neither, and their
      // animations keep frames scheduled after the fling itself is done.
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
      c.jumpTo(e.value[0]);
      await tester.pump();
      (c.position as ScrollPositionWithSingleContext).goBallistic(e.value[1]);
      final List<List<double>> s = <List<double>>[];
      List<double> sample() => <double>[c.offset, if (tester.binding.hasScheduledFrame) 1 else 0];
      await tester.pump();
      s.add(sample());
      for (int i = 0; i < 70; i++) {
        await tester.pump(const Duration(milliseconds: 16));
        s.add(sample());
      }
      out[e.key] = s;
    });
  }
  tearDownAll(() {
    File('build/fling.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
