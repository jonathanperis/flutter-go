// Reference side of flutter-go's animation differential tests. Each scene has
// a twin in ../animations_test.go and records a value sample after every
// pump. Output: build/animations.json.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/animation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

const Key boxKey = ValueKey<String>('box');

Widget box(double w, double h, Color c, Duration d, Curve curve) => Directionality(
  textDirection: TextDirection.ltr,
  child: Center(
    child: AnimatedContainer(key: boxKey, duration: d, curve: curve, width: w, height: h, color: c),
  ),
);

List<double> sampleBox(WidgetTester tester) {
  final Size size = tester.getSize(find.byKey(boxKey));
  Color? color;
  for (final Element e in find.descendant(of: find.byKey(boxKey), matching: find.byWidgetPredicate((Widget w) => w is ColoredBox || w is DecoratedBox)).evaluate()) {
    final Widget w = e.widget;
    color ??= w is ColoredBox ? w.color : ((w as DecoratedBox).decoration as BoxDecoration).color;
  }
  return <double>[size.width, size.height, color!.r, color.g, color.b, color.a];
}

void main() {
  final Map<String, List<List<double>>> out = <String, List<List<double>>>{};

  testWidgets('container_retarget', (WidgetTester tester) async {
    const Duration d = Duration(milliseconds: 300);
    final List<List<double>> s = <List<double>>[];
    await tester.pumpWidget(box(100, 50, const Color(0xffff0000), d, Curves.easeInOut));
    s.add(sampleBox(tester));
    await tester.pumpWidget(box(300, 150, const Color(0xff2196f3), d, Curves.easeInOut));
    s.add(sampleBox(tester));
    for (int i = 0; i < 3; i++) {
      await tester.pump(const Duration(milliseconds: 50));
      s.add(sampleBox(tester));
    }
    await tester.pumpWidget(box(50, 200, const Color(0xff4caf50), d, Curves.easeInOut));
    s.add(sampleBox(tester));
    for (int i = 0; i < 8; i++) {
      await tester.pump(const Duration(milliseconds: 50));
      s.add(sampleBox(tester));
    }
    out['container_retarget'] = s;
  });

  testWidgets('container_fast_out_slow_in', (WidgetTester tester) async {
    const Duration d = Duration(milliseconds: 250);
    final List<List<double>> s = <List<double>>[];
    await tester.pumpWidget(box(10, 10, const Color(0xff000000), d, Curves.fastOutSlowIn));
    await tester.pumpWidget(box(400, 20, const Color(0x80ffffff), d, Curves.fastOutSlowIn));
    for (int i = 0; i < 20; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      s.add(sampleBox(tester));
    }
    out['container_fast_out_slow_in'] = s;
  });

  testWidgets('scroll_animate_to', (WidgetTester tester) async {
    final ScrollController c = ScrollController();
    await tester.pumpWidget(Directionality(
      textDirection: TextDirection.ltr,
      child: ListView.builder(
        controller: c,
        itemCount: 1000,
        itemExtent: 50,
        itemBuilder: (BuildContext context, int i) => const SizedBox(),
      ),
    ));
    final List<List<double>> s = <List<double>>[];
    // Called outside a frame, like from a button handler.
    c.animateTo(2000, duration: const Duration(milliseconds: 400), curve: Curves.easeOut);
    await tester.pump();
    s.add(<double>[c.offset]);
    for (int i = 0; i < 30; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      s.add(<double>[c.offset]);
    }
    out['scroll_animate_to'] = s;
  });

  testWidgets('scroll_animate_to_after_idle', (WidgetTester tester) async {
    final ScrollController c = ScrollController();
    await tester.pumpWidget(Directionality(
      textDirection: TextDirection.ltr,
      child: ListView.builder(
        controller: c,
        itemCount: 1000,
        itemExtent: 50,
        itemBuilder: (BuildContext context, int i) => const SizedBox(),
      ),
    ));
    // Time passes with no frames, then a handler starts the animation.
    await tester.pump(const Duration(milliseconds: 100));
    final List<List<double>> s = <List<double>>[];
    c.animateTo(1000, duration: const Duration(milliseconds: 200), curve: Curves.linear);
    for (int i = 0; i < 16; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      s.add(<double>[c.offset]);
    }
    out['scroll_animate_to_after_idle'] = s;
  });

  tearDownAll(() {
    File('build/animations.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
