// Reference side of flutter-go's licenses differential test. Output:
// build/licenses.json:
//   inputs:     license texts (Flutter's test text, real LICENSE files of
//               the repository, and 400 random texts);
//   paragraphs: for each, [[text, indent]...], or "ERROR" when Flutter's
//               parser throws (an assertion);
//   registry:   the order of LicenseRegistry.licenses events against
//               microtasks and timers, in fake_async.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:fake_async/fake_async.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

const String flutterTestText = '''
A
A
A
  B
B
B
   C
C
C
                  D
D
D

E
E
 F
  G
 G
G

[H
 H
 H]

I\u000cJ
\u000cK
K
\u000c
L
L L
L  L
L   L
L    L
L     L

   M
M
M

N

O
O


P



QQQ

RR RRR RRRR RRRRR
R

S

   T

      U
         V

        W

       X

      Y''';

Object paragraphsOf(String text) {
  try {
    return <List<Object>>[
      for (final LicenseParagraph p in LicenseEntryWithLineBreaks(const <String>[], text).paragraphs)
        <Object>[p.text, p.indent],
    ];
  } catch (e) {
    return 'ERROR';
  }
}

void main() {
  test('licenses', () {
    final List<String> inputs = <String>[
      flutterTestText,
      '    \n\n    ',
      '    \r\n\r\n    ',
      '    \nA\n    ',
      '\n\n\nA\n\n\n',
      'A\r\nB\r\nC\r\n\r\nD\r\nE',
      '',
      'x',
      '\r',
      'a\rb',
      '\t\tindented by tabs\n\t\tmore',
      '  nbsp',
      '\u000b vertical tab',
      'é\n  ü [x\n\u000c\u000c',
      for (final String path in <String>[
        '../../../LICENSE',
        '../../../packages/flutter/LICENSE',
        '../../../engine/src/flutter/LICENSE',
        '../../../engine/src/flutter/sky/packages/sky_engine/LICENSE',
      ])
        File(path).readAsStringSync(),
    ];
    const String alphabet = '    \t\n\n\r\u000c[abé\u000b ';
    int x = 42;
    for (int i = 0; i < 400; i++) {
      final StringBuffer b = StringBuffer();
      x = (x * 1103515245 + 12345) & 0x7fffffff;
      final int length = x % 60;
      for (int j = 0; j < length; j++) {
        x = (x * 1103515245 + 12345) & 0x7fffffff;
        final String letters = i < 300 ? alphabet.substring(0, alphabet.length - 2) : alphabet;
        b.write(letters[(x >> 8) % letters.length]);
      }
      inputs.add(b.toString());
    }

    final List<String> registry = <String>[];
    FakeAsync().run((FakeAsync fa) {
      void log(String s) => registry.add(s);
      LicenseRegistry.licenses.listen(
        (LicenseEntry e) => log('never'),
        onDone: () => log('empty registry done'),
      );
      LicenseRegistry.addLicense(
        () => Stream<LicenseEntry>.fromIterable(<LicenseEntry>[
          const LicenseEntryWithLineBreaks(<String>['a'], 'A'),
          const LicenseEntryWithLineBreaks(<String>['b'], 'B\n\nB2'),
        ]),
      );
      LicenseRegistry.addLicense(() {
        final StreamController<LicenseEntry> c = StreamController<LicenseEntry>();
        Timer(const Duration(milliseconds: 2), () {
          log('timer adds C');
          c.add(const LicenseEntryWithLineBreaks(<String>['c', 'c2'], 'C'));
          c.close();
        });
        return c.stream;
      });
      LicenseRegistry.addLicense(() => const Stream<LicenseEntry>.empty());
      LicenseRegistry.addLicense(
        () => Stream<LicenseEntry>.fromIterable(<LicenseEntry>[
          const LicenseEntryWithLineBreaks(<String>[], 'D'),
        ]),
      );
      for (final String name in <String>['first', 'second']) {
        LicenseRegistry.licenses.listen(
          (LicenseEntry e) => log(
            '$name: ${e.packages.join(',')} ${e.paragraphs.map((LicenseParagraph p) => p.text).join('|')}',
          ),
          onDone: () => log('$name done'),
        );
        scheduleMicrotask(() => log('micro after $name'));
      }
      log('sync end');
      fa.elapse(const Duration(milliseconds: 5));
      LicenseRegistry.reset();
    });

    File('build/licenses.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(
        const JsonEncoder.withIndent(' ').convert(<String, Object>{
          'inputs': inputs,
          'paragraphs': inputs.map(paragraphsOf).toList(),
          'registry': registry,
        }),
      );
  });
}
