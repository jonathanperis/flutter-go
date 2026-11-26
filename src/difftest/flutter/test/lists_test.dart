// Reference side of flutter-go's list (lazy scrolling) differential tests.
//
// Each scene has a twin in ../lists.go. For every built item (visible or in
// the cache extent) it records: index, x, y, width, height. It also records
// the final scroll offset. Output: build/lists.json.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

class ListScene {
  const ListScene(this.count, this.extent, this.jumpTo, {this.inColumn = false});
  final int count;
  final double extent;
  final double jumpTo;
  final bool inColumn;
}

const Map<String, ListScene> scenes = <String, ListScene>{
  'list_top': ListScene(100, 50, 0),
  'list_scrolled': ListScene(100, 50, 1234),
  'list_end': ListScene(100, 50, 1e9),
  'list_in_column': ListScene(1000, 37, 777, inColumn: true),
  'list_short': ListScene(5, 50, 100),
  'list_fractional': ListScene(10000, 33.3, 1000.5),
};

void main() {
  final Map<String, Object> out = <String, Object>{};

  for (final MapEntry<String, ListScene> e in scenes.entries) {
    testWidgets(e.key, (WidgetTester tester) async {
      final ListScene s = e.value;
      final ScrollController controller = ScrollController();
      Widget list = ListView.builder(
        controller: controller,
        itemCount: s.count,
        itemExtent: s.extent,
        itemBuilder: (BuildContext context, int i) =>
            ColoredBox(key: ValueKey<int>(i), color: const Color(0xffeeeeee)),
      );
      if (s.inColumn) {
        list = Column(children: <Widget>[const SizedBox(height: 100), Expanded(child: list)]);
      }
      await tester.pumpWidget(Directionality(textDirection: TextDirection.ltr, child: list));
      controller.jumpTo(s.jumpTo);
      // Flutter accepts an out-of-range jump and lets the scroll physics
      // animate back into range; compare the settled state. (flutter-go
      // clamps immediately: it has no ballistic animation yet.)
      await tester.pumpAndSettle();

      final List<List<double>> items = <List<double>>[];
      // skipOffstage: false includes items in the cache extent.
      for (final Element el in find.byType(ColoredBox, skipOffstage: false).evaluate()) {
        final RenderBox box = el.renderObject! as RenderBox;
        final Offset o = box.localToGlobal(Offset.zero);
        final int index = (el.widget.key! as ValueKey<int>).value;
        items.add(<double>[index.toDouble(), o.dx, o.dy, box.size.width, box.size.height]);
      }
      items.sort((List<double> a, List<double> b) => a[0].compareTo(b[0]));
      out[e.key] = <String, Object>{'offset': controller.offset, 'items': items};
    });
  }

  tearDownAll(() {
    File('build/lists.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
