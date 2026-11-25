// Reference side of flutter-go's differential layout tests.
//
// Every scene here has a twin with the same name in ../scenes.go. This test
// pumps each scene in real Flutter and writes the geometry of every RenderBox
// (depth-first: depth, x, y, width, height) to build/layout.json.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

const red = ColoredBox(color: Color(0xffff0000));

Widget box(double w, double h) => SizedBox(width: w, height: h);

final Map<String, Widget> scenes = <String, Widget>{
  'center_padding': Center(
    child: Padding(padding: const EdgeInsets.all(10), child: box(100, 50)),
  ),
  'align_factors': Align(
    alignment: Alignment.bottomRight,
    widthFactor: 2,
    heightFactor: 1.5,
    child: box(100, 40),
  ),
  'root_colored_box': const ColoredBox(color: Color(0xff00ff00)),
  'row_expanded': Row(
    children: <Widget>[
      box(100, 20),
      const Expanded(child: red),
      const Expanded(flex: 3, child: red),
    ],
  ),
  'column_space_between_stretch': Column(
    mainAxisAlignment: MainAxisAlignment.spaceBetween,
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: <Widget>[box(50, 100), const SizedBox(height: 100)],
  ),
  'flexible_loose': Row(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: <Widget>[
      Flexible(child: box(50, 10)),
      const Expanded(flex: 2, child: red),
      Flexible(child: box(1000, 10)),
    ],
  ),
  'center_min_column': Center(
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        box(30, 10),
        Row(mainAxisSize: MainAxisSize.min, children: <Widget>[box(5, 5), box(7, 9)]),
      ],
    ),
  ),
  'nested_alignments': Column(
    children: <Widget>[
      Row(
        mainAxisAlignment: MainAxisAlignment.spaceEvenly,
        crossAxisAlignment: CrossAxisAlignment.end,
        children: <Widget>[box(10, 30), box(20, 10), box(30, 20)],
      ),
      Expanded(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 5),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceAround,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[box(40, 40), const Align(alignment: Alignment.topRight, child: SizedBox(width: 10, height: 10))],
          ),
        ),
      ),
      Align(alignment: Alignment.centerRight, child: box(25, 25)),
      Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: <Widget>[box(10, 10), const Expanded(child: SizedBox(height: 4)), box(10, 10)],
      ),
    ],
  ),
  'space_around_end': Column(
    mainAxisAlignment: MainAxisAlignment.spaceAround,
    crossAxisAlignment: CrossAxisAlignment.end,
    children: <Widget>[box(10, 10), box(20, 20), box(30, 30)],
  ),
  'text_center': const Center(child: Text('hello', style: TextStyle(fontSize: 20))),
  'default_text_style': const DefaultTextStyle(
    style: TextStyle(fontSize: 20),
    child: Center(child: Text('hello')),
  ),
  'text_wrap': const Align(
    alignment: Alignment.topLeft,
    child: SizedBox(width: 100, child: Text('hello wrapping world', style: TextStyle(fontSize: 10))),
  ),
};

void main() {
  final Map<String, List<List<double>>> out = <String, List<List<double>>>{};

  for (final MapEntry<String, Widget> scene in scenes.entries) {
    testWidgets(scene.key, (WidgetTester tester) async {
      const Key key = ValueKey<String>('root');
      await tester.pumpWidget(
        Directionality(
          textDirection: TextDirection.ltr,
          child: KeyedSubtree(key: key, child: scene.value),
        ),
      );
      final List<List<double>> boxes = <List<double>>[];
      void walk(RenderObject r, int depth) {
        if (r is RenderBox) {
          final Offset o = r.localToGlobal(Offset.zero);
          boxes.add(<double>[depth.toDouble(), o.dx, o.dy, r.size.width, r.size.height]);
        }
        r.visitChildren((RenderObject c) => walk(c, depth + 1));
      }

      walk(tester.renderObject(find.byKey(key)), 0);
      out[scene.key] = boxes;
    });
  }

  tearDownAll(() {
    File('build/layout.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
