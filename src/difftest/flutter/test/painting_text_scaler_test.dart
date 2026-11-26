// Reference side of flutter-go's TextScaler differential test
// (text_scaler.dart, plus text_painter.dart's enums and kDefaultFontSize):
// scale/textScaleFactor/toString/== of linear and clamped scalers built by
// chains of clamp calls, and which clamp arguments assert. Output:
// build/painting_text_scaler.json as a list of lines.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

const List<double> factors = <double>[0, 0.5, 1, 1.25, 3, 7.5];
const List<List<double>> ranges = <List<double>>[
  <double>[0, double.infinity], <double>[0, 1], <double>[0.5, 2], <double>[1, 1], <double>[2, 3],
  <double>[1.1, 1.3], <double>[4, 8], <double>[0.3, double.infinity], <double>[2.5, 2.5],
];
const List<double> sizes = <double>[0, 1, 9.5, 14, 33.3];

// A clamped scaler with maxScale infinity asserts on scale(0) (clampDouble
// gets 0 * infinity = NaN as its max).
String describe(TextScaler s) => '$s tsf=${s.textScaleFactor} '
    '${sizes.map((double f) => val(() => '${s.scale(f)}')).join(',')}';

String val(String Function() f) {
  try {
    return f();
  } on AssertionError {
    return 'AssertionError';
  }
}

String err(void Function() f) => val(() {
  f();
  return 'no error';
});

// A user-defined scaler: clamp is TextScaler's default, which builds the
// private _ClampedTextScaler.
class Custom extends TextScaler {
  const Custom(this.k);
  final double k;
  @override
  double scale(double fontSize) => fontSize * k + 1;
  @override
  double get textScaleFactor => k;
  @override
  String toString() => 'custom($k)';
}

void main() {
  test('painting_text_scaler', () {
    final List<String> out = <String>[];
    final List<TextScaler> built = <TextScaler>[];
    const Custom custom = Custom(1.5);
    for (final TextScaler base in <TextScaler>[for (final double f in factors) TextScaler.linear(f), custom]) {
      out.add('base: ${describe(base)}');
      for (final List<double> r in ranges) {
        final TextScaler c = base.clamp(minScaleFactor: r[0], maxScaleFactor: r[1]);
        built.add(c);
        out.add('$base clamp $r: ${describe(c)}${base is Custom ? " same=${identical(c, base)}" : ""}');
        for (final List<double> r2 in ranges) {
          final TextScaler c2 = c.clamp(minScaleFactor: r2[0], maxScaleFactor: r2[1]);
          out.add('$base clamp $r $r2: ${describe(c2)}');
        }
      }
    }
    for (int i = 0; i < built.length; i++) {
      out.add('eq $i: ${<String>[for (final TextScaler b in built) b == built[i] ? '1' : '0'].join()}');
    }
    for (final List<double> bad in <List<double>>[
      <double>[5, 4], <double>[-1, 2], <double>[double.infinity, double.infinity], <double>[1, double.nan],
      <double>[double.nan, 2], <double>[-0.0, 0], <double>[0, -0.0],
    ]) {
      out.add('assert $bad: ${err(() => TextScaler.noScaling.clamp(minScaleFactor: bad[0], maxScaleFactor: bad[1]))} '
          '${err(() => custom.clamp(minScaleFactor: bad[0], maxScaleFactor: bad[1]))} '
          '${err(() => custom.clamp(minScaleFactor: 1, maxScaleFactor: 5).clamp(minScaleFactor: bad[0], maxScaleFactor: bad[1]))}');
    }
    out.add('linear(-1): ${err(() => TextScaler.linear(-1))}');
    out.add('scale(-1): ${err(() => TextScaler.noScaling.scale(-1))}');
    out.add('scale(inf): ${err(() => TextScaler.noScaling.scale(double.infinity))}');
    out.add('scale(nan): ${err(() => TextScaler.noScaling.scale(double.nan))}');
    out.add('clamped scale(-1): ${err(() => custom.clamp(minScaleFactor: 1, maxScaleFactor: 2).scale(-1))}');
    out.add('clamped scale(nan): ${err(() => custom.clamp(minScaleFactor: 1, maxScaleFactor: 2).scale(double.nan))}');
    out.add('enums ${TextOverflow.values} ${TextWidthBasis.values} $kDefaultFontSize');
    File('build/painting_text_scaler.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
