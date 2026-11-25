// Reference side of flutter-go's InheritedWidget tests: records which
// widgets build at each step. Output: build/inherited.json.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];

class Value extends InheritedWidget {
  const Value({super.key, required this.value, required super.child});
  final int value;
  @override
  bool updateShouldNotify(Value oldWidget) => oldWidget.value != value;
}

class Leaf extends StatelessWidget {
  // ignore: prefer_const_constructors_in_immutables
  Leaf(this.name, {super.key});
  final String name;
  @override
  Widget build(BuildContext context) {
    log.add(name);
    return const SizedBox();
  }
}

class Dependent extends StatelessWidget {
  // ignore: prefer_const_constructors_in_immutables
  Dependent(this.name, {super.key});
  final String name;
  @override
  Widget build(BuildContext context) {
    log.add('$name=${context.dependOnInheritedWidgetOfExactType<Value>()!.value}');
    return const SizedBox();
  }
}

class Holder extends StatelessWidget {
  // ignore: prefer_const_constructors_in_immutables
  Holder(this.name, this.child, {super.key});
  final String name;
  final Widget child;
  @override
  Widget build(BuildContext context) {
    log.add(name);
    return child;
  }
}

class Root extends StatefulWidget {
  // ignore: prefer_const_constructors_in_immutables
  Root({super.key, required this.child});
  final Widget child;
  @override
  State<Root> createState() => RootState();
}

class RootState extends State<Root> {
  int v = 0, c = 0;
  void set(int nv, int nc) => setState(() {
    v = nv;
    c = nc;
  });
  @override
  Widget build(BuildContext context) {
    log.add('Root v=$v c=$c');
    return Value(value: v, child: widget.child);
  }
}

Widget subtree() => Column(children: <Widget>[Leaf('A'), Dependent('B'), Holder('C', Dependent('D'))]);

void main() {
  testWidgets('inherited', (WidgetTester tester) async {
    final List<List<String>> steps = <List<String>>[];
    Future<void> step(Future<void> Function() f) async {
      log.clear();
      await f();
      steps.add(List<String>.of(log));
    }

    final Widget child = subtree();
    Widget app(Widget c) => Directionality(textDirection: TextDirection.ltr, child: Root(child: c));
    RootState state() => tester.state<RootState>(find.byType(Root));

    await step(() => tester.pumpWidget(app(child)));                // 0 initial
    await step(() async { state().set(1, 0); await tester.pump(); }); // 1 value changes
    await step(() async { state().set(1, 1); await tester.pump(); }); // 2 same value
    await step(() => tester.pumpWidget(app(child)));                // 3 same child instance
    await step(() async { state().set(2, 1); await tester.pump(); }); // 4 value changes again
    await step(() => tester.pumpWidget(app(subtree())));            // 5 new, equal subtree
    File('build/inherited.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(steps));
  });
}
