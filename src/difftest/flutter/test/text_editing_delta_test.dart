// Reference side of flutter-go's text editing delta differential test
// (services/text_editing_delta.dart): TextEditingDelta.fromJSON on every
// replacement range of several texts (including non-BMP and grapheme
// clusters) with several replacement strings, selections and composing
// regions; each delta's kind, fields, debugFillProperties and the value it
// applies to, or the assertion it fails.
// The Go twin is ../../text_editing_delta_test.go.
// Output: build/text_editing_delta.json as {"cases": [{"json": ..., "out": [...]}]}.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

List<String> describe(TextEditingDelta d) {
  final builder = DiagnosticPropertiesBuilder();
  d.debugFillProperties(builder);
  return builder.properties
      .where((DiagnosticsNode node) => !node.isFiltered(DiagnosticLevel.info))
      .map((DiagnosticsNode node) => node.toString())
      .toList();
}

// Each output line is JSON-encoded, so lone surrogates survive the file
// (Go's JSON decoders replace them).
List<String> run(Map<String, Object?> encoded) {
  final raw = <String>[];
  final out = _Lines(raw);
  try {
    final TextEditingDelta d = TextEditingDelta.fromJSON(encoded);
    out.add('${d.runtimeType} ${jsonEncode(d.oldText)} ${d.selection} ${d.composing}');
    switch (d) {
      case final TextEditingDeltaInsertion i:
        out.add('inserted ${jsonEncode(i.textInserted)} at ${i.insertionOffset}');
      case final TextEditingDeltaDeletion del:
        out.add('deleted ${del.deletedRange} ${jsonEncode(del.textDeleted)}');
      case final TextEditingDeltaReplacement r:
        out.add('replaced ${r.replacedRange} ${jsonEncode(r.textReplaced)} with ${jsonEncode(r.replacementText)}');
      case TextEditingDeltaNonTextUpdate():
        out.add('nonText');
    }
    out.addAll(describe(d));
    final TextEditingValue applied = d.apply(TextEditingValue(text: 'ignored ${d.oldText}'));
    out.add('apply ${jsonEncode(applied.text)} ${applied.selection} ${applied.composing}');
  } on AssertionError catch (e) {
    out.add('assert ${e.message}');
  } on RangeError catch (e) {
    out.add('range $e');
  }
  return raw.map(jsonEncode).toList();
}

class _Lines {
  _Lines(this.lines);
  final List<String> lines;
  void add(String s) => lines.add(s);
  void addAll(Iterable<String> s) => lines.addAll(s);
}

void main() {
  test('text editing deltas', () {
    final cases = <Map<String, Object?>>[];
    const texts = <String>['', 'hello', 'hello world', 'a😀b', '👍🏽x', 'é日本'];
    for (final String text in texts) {
      final int n = text.length;
      final ranges = <List<int>>[<int>[-1, -1]];
      for (var start = 0; start <= n; start++) {
        for (var end = start; end <= n; end++) {
          ranges.add(<int>[start, end]);
        }
      }
      ranges.add(<int>[0, n + 1]);
      for (final List<int> r in ranges) {
        final String inside = r[0] >= 0 && r[1] <= n ? text.substring(r[0], r[1]) : '';
        final deltaTexts = <String>{'', 'x', 'worl', '😀', inside, '${inside}d', if (inside.isNotEmpty) inside.substring(1)};
        for (final String deltaText in deltaTexts) {
          final int newLength = r[0] < 0 ? n : n - (r[1] - r[0]) + deltaText.length;
          final int caret = r[0] < 0 ? n : r[0] + deltaText.length;
          for (final List<int> composing in <List<int>>[
            <int>[-1, -1],
            <int>[r[0] < 0 ? 0 : r[0], caret],
            <int>[0, newLength + 1],
          ]) {
            final encoded = <String, Object?>{
              'oldText': text,
              'deltaText': deltaText,
              'deltaStart': r[0],
              'deltaEnd': r[1],
              'selectionBase': caret,
              'selectionExtent': caret,
              'selectionAffinity': caret.isEven ? 'TextAffinity.downstream' : 'TextAffinity.upstream',
              'selectionIsDirectional': caret % 3 == 0,
              'composingBase': composing[0],
              'composingExtent': composing[1],
            };
            cases.add(<String, Object?>{
              'json': jsonEncode(encoded),
              'oldUnits': text.codeUnits,
              'deltaUnits': deltaText.codeUnits,
              'out': run(encoded),
            });
          }
        }
      }
    }
    // Missing optional fields take their defaults.
    for (final String json in <String>[
      '{"oldText": "abc", "deltaText": "d", "deltaStart": 3, "deltaEnd": 3}',
      '{"oldText": "abc", "deltaText": "", "deltaStart": -1, "deltaEnd": -1, "selectionBase": 1}',
      '{"oldText": "abc", "deltaText": "", "deltaStart": 0, "deltaEnd": 2, "selectionBase": 0, "selectionExtent": 1}',
      '{"oldText": "abc", "deltaText": "x", "deltaStart": -1, "deltaEnd": 2, "selectionBase": 1, "selectionExtent": 1}',
    ]) {
      cases.add(<String, Object?>{'json': json, 'out': run(jsonDecode(json) as Map<String, Object?>)});
    }
    File('build/text_editing_delta.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'cases': cases}));
  });
}
