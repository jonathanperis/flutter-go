// Reference side of flutter-go's text formatter differential test
// (services' FilteringTextInputFormatter, LengthLimitingTextInputFormatter,
// TextEditingValue.replaced, and dart:core's RegExp/String allMatches,
// which the filtering formatter is built on). Every case records its
// inputs, so the Go side (difftest/text_formatter_test.go) replays them
// from build/text_formatter.json and compares the results exactly.
// Strings are written as UTF-16 code units (lone surrogates included).
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

const List<String> texts = <String>[
  '',
  'abc',
  'Into the Woods',
  'a1b2c3\nd4e5f6',
  'AAABBBBBCCC',
  'ab\u{1F600}cd',
  '\u{1F984}\u{1F984}\u{1F984}\u{1F984}',
  'ééx',
  '\u{1F468}‍\u{1F469}‍\u{1F466}01234',
  '\u{1F1E7}\u{1F1F7}\u{1F1FA}\u{1F1F8}!',
  'x\uD83Dy',
  '\uDE00z\uD83D',
  'aaa\n\nbbb\r\nc',
  '12345@6',
  'हिन्दी ok',
  'ab&&ca@bcabc',
  'Hello World_42    end',
  'BBB',
];

// Formatter specs: [kind, pattern, isRegExp, replacement, regexp flags].
const List<List<Object?>> filterSpecs = <List<Object?>>[
  <Object?>['deny', '\n', false, '', ''],
  <Object?>['allow', '[0-9]', true, '', ''],
  <Object?>['deny', '[a-z]', true, '*', ''],
  <Object?>['allow', 'o', false, '*', ''],
  <Object?>['deny', 'o', false, '*', ''],
  <Object?>['allow', 'o+', true, '*', ''],
  <Object?>['deny', 'o+', true, '#', ''],
  <Object?>['allow', '[A-Za-z0-9.@-]*', true, '', ''],
  <Object?>['deny', '', true, '-', ''],
  <Object?>['deny', '', false, '-', ''],
  <Object?>['allow', '.', true, '', ''],
  <Object?>['deny', '.', true, '?', ''],
  <Object?>['allow', '.', true, '', 'u'],
  <Object?>['deny', r'\s', true, '_', ''],
  <Object?>['deny', r'\S+', true, '', ''],
  <Object?>['allow', r'\w+', true, '|', ''],
  <Object?>['deny', '[\uD800-\uDFFF]', true, '~', ''],
  <Object?>['deny', 'b', true, 'X', 'i'],
  <Object?>['deny', r'^a', true, '^', 'm'],
  <Object?>['deny', r'a$', true, r'$', 'm'],
  <Object?>['deny', r'^.', true, '', ''],
  <Object?>['deny', r'\bB', true, 'b', ''],
  <Object?>['allow', 'a*?', true, '.', ''],
  <Object?>['allow', '(ab|a)c?', true, '', ''],
  <Object?>['deny', 'x?', true, 'Y', ''],
  <Object?>['deny', 'BBB', false, '****', ''],
  <Object?>['allow', 'BBB', false, '****', ''],
  <Object?>['deny', 'BBB', false, '***', ''],
  <Object?>['deny', r'\p{L}', true, 'L', 'u'],
  <Object?>['allow', '[^a-z]', true, '', ''],
  <Object?>['deny', '[^]', true, '', ''],
  <Object?>['deny', '.', true, '', 's'],
  <Object?>['deny', r'́', true, '', ''],
];

List<int> units(String s) => s.codeUnits;

RegExp regExp(String pattern, String flags) => RegExp(
  pattern,
  multiLine: flags.contains('m'),
  caseSensitive: !flags.contains('i'),
  unicode: flags.contains('u'),
  dotAll: flags.contains('s'),
);

Map<String, Object?> encodeValue(TextEditingValue v) => <String, Object?>{
  't': units(v.text),
  'b': v.selection.baseOffset,
  'e': v.selection.extentOffset,
  'a': v.selection.affinity.name,
  'd': v.selection.isDirectional,
  'cs': v.composing.start,
  'ce': v.composing.end,
};

// The edited values each formatter sees for a text.
List<TextEditingValue> valuesFor(String text) {
  final int n = text.length;
  final List<TextEditingValue> values = <TextEditingValue>[
    TextEditingValue(text: text),
    TextEditingValue(text: text, selection: TextSelection.collapsed(offset: n)),
    TextEditingValue(text: text, selection: const TextSelection.collapsed(offset: 0)),
  ];
  if (n >= 2) {
    values.add(TextEditingValue(text: text, selection: TextSelection(baseOffset: n - 1, extentOffset: 1)));
    values.add(TextEditingValue(
      text: text,
      selection: TextSelection.collapsed(offset: n ~/ 2, affinity: TextAffinity.upstream),
      composing: TextRange(start: 1, end: n),
    ));
    values.add(TextEditingValue(
      text: text,
      selection: TextSelection(baseOffset: 1, extentOffset: n, isDirectional: true),
      composing: TextRange(start: 0, end: n ~/ 2),
    ));
    values.add(TextEditingValue(
      text: text,
      selection: TextSelection.collapsed(offset: n ~/ 2),
      composing: TextRange.collapsed(n ~/ 2),
    ));
  }
  return values;
}

void main() {
  final Map<String, Object?> out = <String, Object?>{};

  test('regexp and string allMatches', () {
    final List<Object?> cases = <Object?>[];
    for (final List<Object?> spec in filterSpecs) {
      final Pattern pattern = (spec[2]! as bool)
          ? regExp(spec[1]! as String, spec[4]! as String)
          : spec[1]! as String;
      for (final String text in texts) {
        for (final int start in <int>[0, 1, text.length]) {
          if (start > text.length) {
            continue;
          }
          cases.add(<String, Object?>{
            'p': units(spec[1]! as String),
            'r': spec[2],
            'f': spec[4],
            't': units(text),
            's': start,
            'm': <Object?>[
              for (final Match m in pattern.allMatches(text, start))
                <Object?>[m.start, m.end, for (int g = 1; g <= m.groupCount; g++) m.group(g)?.codeUnits],
            ],
          });
        }
      }
    }
    out['allMatches'] = cases;
  });

  test('filtering formatters', () {
    final List<Object?> cases = <Object?>[];
    for (int i = 0; i < filterSpecs.length; i++) {
      final List<Object?> spec = filterSpecs[i];
      final Pattern pattern = (spec[2]! as bool)
          ? regExp(spec[1]! as String, spec[4]! as String)
          : spec[1]! as String;
      final FilteringTextInputFormatter formatter = FilteringTextInputFormatter(
        pattern,
        allow: spec[0] == 'allow',
        replacementString: spec[3]! as String,
      );
      for (final String text in texts) {
        for (final TextEditingValue value in valuesFor(text)) {
          final TextEditingValue result = formatter.formatEditUpdate(TextEditingValue.empty, value);
          cases.add(<String, Object?>{
            'spec': i,
            'new': encodeValue(value),
            'out': encodeValue(result),
            'str': units(result.toString()),
          });
        }
      }
    }
    out['filtering'] = cases;
    out['filterSpecs'] = <Object?>[
      for (final List<Object?> spec in filterSpecs)
        <Object?>[spec[0], units(spec[1]! as String), spec[2], units(spec[3]! as String), spec[4]],
    ];
  });

  test('length limiting formatters', () {
    final List<Object?> cases = <Object?>[];
    for (final TargetPlatform platform in <TargetPlatform>[TargetPlatform.android, TargetPlatform.iOS]) {
      debugDefaultTargetPlatformOverride = platform;
      for (final int? maxLength in <int?>[null, -1, 1, 2, 3, 5]) {
        for (final MaxLengthEnforcement? enforcement in <MaxLengthEnforcement?>[null, ...MaxLengthEnforcement.values]) {
          final LengthLimitingTextInputFormatter formatter = LengthLimitingTextInputFormatter(
            maxLength,
            maxLengthEnforcement: enforcement,
          );
          for (final String text in texts) {
            final List<TextEditingValue> news = valuesFor(text);
            final List<TextEditingValue> olds = <TextEditingValue>[
              TextEditingValue.empty,
              if (maxLength != null && maxLength > 0)
                LengthLimitingTextInputFormatter.truncate(TextEditingValue(text: text), maxLength),
              if (maxLength != null && maxLength > 0)
                LengthLimitingTextInputFormatter.truncate(
                  TextEditingValue(text: text, selection: const TextSelection(baseOffset: 0, extentOffset: 1)),
                  maxLength,
                ),
              if (maxLength != null && maxLength > 0)
                LengthLimitingTextInputFormatter.truncate(
                  TextEditingValue(
                    text: text,
                    selection: const TextSelection.collapsed(offset: 0),
                    composing: const TextRange(start: 0, end: 1),
                  ),
                  maxLength,
                ),
            ];
            for (final TextEditingValue oldValue in olds) {
              for (final TextEditingValue newValue in news) {
                final TextEditingValue result = formatter.formatEditUpdate(oldValue, newValue);
                cases.add(<String, Object?>{
                  'platform': platform.name,
                  'max': maxLength,
                  'enf': enforcement?.name,
                  'old': encodeValue(oldValue),
                  'new': encodeValue(newValue),
                  'out': encodeValue(result),
                  'str': units(result.toString()),
                });
              }
            }
          }
        }
      }
      debugDefaultTargetPlatformOverride = null;
    }
    out['lengthLimiting'] = cases;
    out['defaults'] = <String, String>{
      for (final TargetPlatform p in TargetPlatform.values)
        p.name: LengthLimitingTextInputFormatter.getDefaultMaxLengthEnforcement(p).toString(),
    };
  });

  test('TextEditingValue.replaced and toJSON', () {
    final List<Object?> cases = <Object?>[];
    for (final String text in texts) {
      for (final TextEditingValue value in valuesFor(text)) {
        final int n = text.length;
        for (final TextRange range in <TextRange>[
          TextRange.empty,
          const TextRange.collapsed(0),
          TextRange(start: 0, end: n),
          if (n >= 2) TextRange(start: 1, end: n - 1),
          if (n >= 1) TextRange.collapsed(n),
        ]) {
          for (final String replacement in <String>['', 'X', 'YZW', '\u{1F600}']) {
            final TextEditingValue result = value.replaced(range, replacement);
            cases.add(<String, Object?>{
              'in': encodeValue(value),
              'rs': range.start,
              're': range.end,
              'rep': units(replacement),
              'out': encodeValue(result),
              'json': jsonEncode(result.toJSON()).codeUnits,
              'composingValid': result.isComposingRangeValid,
            });
          }
        }
      }
    }
    out['replaced'] = cases;
  });

  tearDownAll(() {
    File('build/text_formatter.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(jsonEncode(out));
  });
}
