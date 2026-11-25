// Reference side of flutter-go's repaint-boundary tests: counts how many
// times each painter paints per frame. Output: build/repaint.json.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

final Map<String, int> paints = <String, int>{};

class Counter extends CustomPainter {
  Counter(this.name, {super.repaint});
  final String name;
  @override
  void paint(Canvas canvas, Size size) => paints[name] = (paints[name] ?? 0) + 1;
  @override
  bool shouldRepaint(Counter old) => false;
}

Widget counted(String name, [Listenable? repaint]) =>
    CustomPaint(size: const Size(10, 10), painter: Counter(name, repaint: repaint));

void main() {
  final Map<String, List<Map<String, int>>> out = <String, List<Map<String, int>>>{};

  Future<void> run(WidgetTester tester, String scene, Widget w, List<VoidCallback> steps) async {
    final List<Map<String, int>> frames = <Map<String, int>>[];
    paints.clear();
    await tester.pumpWidget(Directionality(textDirection: TextDirection.ltr, child: w));
    frames.add(Map<String, int>.of(paints));
    for (final VoidCallback s in steps) {
      paints.clear();
      s();
      await tester.pump();
      frames.add(Map<String, int>.of(paints));
    }
    out[scene] = frames;
  }

  testWidgets('sibling_in_boundary', (WidgetTester tester) async {
    final ValueNotifier<int> a = ValueNotifier<int>(0), b = ValueNotifier<int>(0);
    await run(tester, 'sibling_in_boundary',
        Column(children: <Widget>[counted('A', a), RepaintBoundary(child: counted('B', b))]),
        <VoidCallback>[() => b.value++, () => a.value++, () {}]);
  });

  testWidgets('no_boundary', (WidgetTester tester) async {
    final ValueNotifier<int> a = ValueNotifier<int>(0), b = ValueNotifier<int>(0);
    await run(tester, 'no_boundary',
        Column(children: <Widget>[counted('A', a), counted('B', b)]),
        <VoidCallback>[() => b.value++, () => a.value++]);
  });

  testWidgets('nested_boundaries', (WidgetTester tester) async {
    final ValueNotifier<int> a = ValueNotifier<int>(0), b = ValueNotifier<int>(0), c = ValueNotifier<int>(0);
    await run(tester, 'nested_boundaries',
        Column(children: <Widget>[
          counted('A', a),
          RepaintBoundary(child: Column(children: <Widget>[counted('B', b), RepaintBoundary(child: counted('C', c))])),
        ]),
        <VoidCallback>[() => c.value++, () => b.value++, () => a.value++]);
  });

  testWidgets('list_scroll', (WidgetTester tester) async {
    final ScrollController controller = ScrollController();
    await run(tester, 'list_scroll',
        ListView.builder(
          controller: controller,
          itemCount: 100,
          itemExtent: 50,
          itemBuilder: (BuildContext context, int i) => CustomPaint(painter: Counter('item$i')),
        ),
        <VoidCallback>[() => controller.jumpTo(20), () => controller.jumpTo(40), () => controller.jumpTo(1000)]);
  });

  tearDownAll(() {
    File('build/repaint.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
