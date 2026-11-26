// Reference side of flutter-go's toStringAsPrecision / debugDoublePrecision
// differential test. Output: build/double_precision.json:
//   values:  the doubles, as 16-digit hex bit patterns;
//   results: for each value, toStringAsPrecision(1..21);
//   debug:   debugFormatDouble of each value with debugDoublePrecision
//            unset and set to 1..21 (the first 40 values);
//   errors:  the RangeError messages for precisions 0 and 22.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

double _fromBits(int bits) {
  final ByteData b = ByteData(8)..setUint64(0, bits);
  return b.getFloat64(0);
}

String _hex(double v) {
  final ByteData b = ByteData(8)..setFloat64(0, v);
  return b.getUint32(0).toRadixString(16).padLeft(8, '0') +
      b.getUint32(4).toRadixString(16).padLeft(8, '0');
}

void main() {
  test('double_precision', () {
    final List<double> values = <double>[
      0.0, -0.0, 1.0, -1.0, 0.5, 1.5, 2.5, -2.5, 9.5, 99.5, 0.125, 0.375, 12.5,
      1e15, 1234567.0, 12345678901234567890.0, 0.00000012345, 0.0000012345,
      123.456, 0.1, 0.2, 0.3, 1 / 3, 2 / 3, 1e21, 1e22, 1e-7, 1e-6, 9.999999e-7,
      999999.5, 0.000001, 1.7976931348623157e308, 5e-324, 2.2250738585072014e-308,
      double.nan, double.infinity, double.negativeInfinity, 100.0, 10.0, 1e100,
    ];
    for (int i = -64; i <= 64; i++) {
      values.add(i / 8);
      values.add(i / 64 + 1000);
    }
    int x = 0x123456789;
    for (int i = 0; i < 400; i++) {
      x = x * 6364136223846793005 + 1442695040888963407;
      final double v = _fromBits(x);
      if (!v.isNaN) {
        values.add(v);
      }
      x = x * 6364136223846793005 + 1442695040888963407;
      // Short decimals, where exact ties are likely.
      values.add(((x >> 20) % 2000000) / ((x >> 8) % 3 == 0 ? 1000 : 16));
    }
    final List<List<String>> results = <List<String>>[
      for (final double v in values)
        <String>[for (int p = 1; p <= 21; p++) v.toStringAsPrecision(p)],
    ];
    final List<List<String>> debug = <List<String>>[];
    for (final double v in values.take(40)) {
      final List<String> row = <String>[debugFormatDouble(v)];
      for (int p = 1; p <= 21; p++) {
        debugDoublePrecision = p;
        row.add(debugFormatDouble(v));
      }
      debugDoublePrecision = null;
      debug.add(row);
    }
    final List<String> errors = <String>[];
    for (final int p in <int>[0, 22]) {
      try {
        1.0.toStringAsPrecision(p);
      } catch (e) {
        errors.add(e.toString());
      }
    }
    File('build/double_precision.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(
        const JsonEncoder.withIndent(' ').convert(<String, Object>{
          'values': values.map(_hex).toList(),
          'results': results,
          'debug': debug,
          'errors': errors,
        }),
      );
  });
}
