// Reference side of flutter-go's text boundary and characters differential
// tests: services' CharacterBoundary, ParagraphBoundary, DocumentBoundary
// and LineBoundary (over a fixed TextLayoutMetrics) at every offset of many
// strings, and package:characters' grapheme clusters and CharacterRange
// moves on the same strings. Output: build/text_boundary.json, replayed by
// difftest/text_boundary_test.go. Strings are written as UTF-16 code units.
import 'dart:convert';
import 'dart:io';

import 'package:characters/characters.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

const List<String> samples = <String>[
  '',
  'a',
  'abc',
  'a❄︎c',
  'é̂x',
  '\u{1F468}‍\u{1F469}‍\u{1F467}‍\u{1F466}!',
  '\u{1F3F3}️‍\u{1F308}',
  '\u{1F1E7}\u{1F1F7}\u{1F1FA}\u{1F1F8}\u{1F1E9}',
  'x\u{1F1E9}\u{1F1F0}\u{1F1FE}',
  'a\r\nb\n\rc\r\r\n',
  'abcd efg hi\r\n\r\n\n\njklmno\npqrstuv',
  '각가각ᄀᄀ',
  '한국어 읍쌍',
  'क्षि न्‍द',
  '؀١ ःक',
  '\u{1F44D}\u{1F3FD}\u{1F469}\u{1F3FF}‍\u{1F4BB}',
  'x\uD83Dy',
  '\uDC00abc\uD800',
  '\uDE00\uD83D',
  'line one line two three\u0085four\u000Bfive\u000Cend',
  'Z͑ͫ̓ͪ A̴̵',
  '\t\u0000​‌‍­',
  '12⃣#️⃣',
  '\u{1F469}‍❤️‍\u{1F48B}‍\u{1F468}x',
];

// A fixed layout: lines end after each '\n' (the terminator belongs to
// its line).
class FakeLayout extends TextLayoutMetrics {
  FakeLayout(this.text);
  final String text;

  @override
  TextSelection getLineAtOffset(TextPosition position) {
    int start = 0;
    for (int i = 0; i < text.length; i++) {
      if (text.codeUnitAt(i) == 0x0A) {
        if (position.offset <= i) {
          return TextSelection(baseOffset: start, extentOffset: i + 1);
        }
        start = i + 1;
      }
    }
    return TextSelection(baseOffset: start, extentOffset: text.length);
  }

  @override
  TextRange getWordBoundary(TextPosition position) => throw UnimplementedError();
  @override
  TextPosition getTextPositionAbove(TextPosition position) => throw UnimplementedError();
  @override
  TextPosition getTextPositionBelow(TextPosition position) => throw UnimplementedError();
}

Object? guard(Object? Function() f) {
  try {
    return f();
  } catch (e) {
    return 'error';
  }
}

List<Object?> boundaryAt(TextBoundary b, int i) => <Object?>[
  guard(() => b.getLeadingTextBoundaryAt(i)),
  guard(() => b.getTrailingTextBoundaryAt(i)),
  guard(() {
    final TextRange r = b.getTextBoundaryAt(i);
    return <int>[r.start, r.end];
  }),
];

List<int> u(String s) => s.codeUnits;

void main() {
  final Map<String, Object?> out = <String, Object?>{};

  test('boundaries', () {
    final List<Object?> cases = <Object?>[];
    for (final String s in samples) {
      final Map<String, TextBoundary> boundaries = <String, TextBoundary>{
        'character': CharacterBoundary(s),
        'paragraph': ParagraphBoundary(s),
        'document': DocumentBoundary(s),
        'line': LineBoundary(FakeLayout(s)),
      };
      for (final MapEntry<String, TextBoundary> b in boundaries.entries) {
        cases.add(<String, Object?>{
          's': u(s),
          'kind': b.key,
          'at': <Object?>[for (int i = -2; i <= s.length + 2; i++) boundaryAt(b.value, i)],
        });
      }
    }
    out['boundaries'] = cases;
  });

  test('characters', () {
    final List<Object?> cases = <Object?>[];
    for (final String s in samples) {
      final Characters cs = s.characters;
      final List<String> graphemes = cs.toList();
      final Map<String, Object?> c = <String, Object?>{
        's': u(s),
        'graphemes': <Object?>[for (final String g in graphemes) u(g)],
        'length': cs.length,
        'first': guard(() => u(cs.first)),
        'last': guard(() => u(cs.last)),
        'single': guard(() => u(cs.single)),
        'at': <Object?>[
          for (int i = 0; i <= s.length; i++)
            guard(() {
              final CharacterRange r = CharacterRange.at(s, i);
              return <Object?>[r.stringBeforeLength, u(r.current)];
            }),
        ],
        'atRange': <Object?>[
          for (int i = 0; i <= s.length; i++)
            for (int j = i; j <= s.length; j++)
              guard(() {
                final CharacterRange r = CharacterRange.at(s, i, j);
                return <Object?>[r.stringBeforeLength, r.stringAfterLength];
              }),
        ],
        'skipTake': <Object?>[
          for (int n = 0; n <= 3; n++)
            <Object?>[
              guard(() => u(cs.skip(n).string)),
              guard(() => u(cs.take(n).string)),
              guard(() => u(cs.skipLast(n).string)),
              guard(() => u(cs.takeLast(n).string)),
              guard(() => u(cs.getRange(n, n + 2).string)),
              guard(() => u(cs.characterAt(n).string)),
              guard(() => u(cs.elementAt(n))),
            ],
        ],
        'contains': guard(() => <Object?>[
          for (final String g in graphemes) cs.contains(g),
          for (int i = 1; i < graphemes.length; i++) cs.contains(graphemes[i - 1] + graphemes[i]),
          for (final String g in graphemes) cs.containsAll(g.characters),
          for (final String g in graphemes) cs.startsWith(g.characters),
          for (final String g in graphemes) cs.endsWith(g.characters),
          for (int i = 0; i < s.length; i++) cs.containsAll(s.substring(i, i + 1).characters),
        ]),
        'find': <Object?>[
          for (final String g in graphemes)
            <Object?>[
              guard(() => cs.findFirst(g.characters)?.stringBeforeLength),
              guard(() => cs.findLast(g.characters)?.stringBeforeLength),
            ],
        ],
        'moves': () {
          final List<Object?> log = <Object?>[];
          final CharacterRange it = cs.iterator;
          final List<Object? Function()> steps = <Object? Function()>[
            for (int k = 1; k <= 3; k++) () => <Object?>[it.moveNext(k), it.stringBeforeLength, u(it.current)],
            () => <Object?>[it.moveBack(2), it.stringBeforeLength, u(it.current)],
            () => <Object?>[it.expandNext(2), it.stringBeforeLength, u(it.current)],
            () => <Object?>[it.dropFirst(), it.stringBeforeLength, u(it.current)],
            () => <Object?>[it.expandBack(3), it.stringBeforeLength, u(it.current)],
            () => <Object?>[it.dropLast(), it.stringBeforeLength, u(it.current)],
            () {
              it.expandAll();
              return <Object?>[it.stringBeforeLength, u(it.current)];
            },
            () {
              it.collapseToEnd();
              return <Object?>[it.moveBack(), it.stringBeforeLength, u(it.current)];
            },
            () {
              it.moveBackAll();
              return <Object?>[it.stringBeforeLength, u(it.current)];
            },
          ];
          for (final Object? Function() step in steps) {
            final Object? r = guard(step);
            log.add(r);
            if (r == 'error') {
              break;
            }
          }
          return log;
        }(),
        'replace': graphemes.isEmpty
            ? null
            : guard(() => <Object?>[
                u(cs.replaceAll(graphemes.first.characters, 'X'.characters).string),
                u(cs.replaceFirst(graphemes.last.characters, '́'.characters).string),
                u(cs.replaceAll(''.characters, '|'.characters).string),
                <Object?>[for (final Characters p in cs.split(graphemes.first.characters)) u(p.string)],
                <Object?>[for (final Characters p in cs.split(''.characters, 3)) u(p.string)],
                u(cs.where((String g) => g.length == 1).string),
                u(cs.join('+')),
              ]),
      };
      cases.add(c);
    }
    out['characters'] = cases;
  });

  tearDownAll(() {
    File('build/text_boundary.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(jsonEncode(out));
  });
}
