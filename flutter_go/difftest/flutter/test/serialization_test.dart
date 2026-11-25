// Reference side of flutter-go's WriteBuffer differential test: the exact
// bytes Flutter writes for a mixed sequence (alignment, endianness, growth).
// Output: build/serialization.json as {"bytes": [...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('serialization', () {
    final WriteBuffer w = WriteBuffer(startCapacity: 1);
    w.putUint8(7);
    w.putFloat64(1.5);
    w.putUint16(0xBEEF);
    w.putUint16(0xBEEF, endian: Endian.big);
    w.putInt32(-2);
    w.putUint32(0xDEADBEEF, endian: Endian.big);
    w.putUint8(1);
    w.putInt64List(Int64List.fromList(<int>[1, -1]));
    w.putUint8(2);
    w.putFloat32List(Float32List.fromList(<double>[0.1]));
    w.putUint8List(Uint8List.fromList(<int>[9, 8, 7]));
    w.putFloat64(-0.0, endian: Endian.big);
    w.putInt64(1 << 40);
    w.putInt32List(Int32List.fromList(<int>[5]));
    w.putFloat64List(Float64List.fromList(<double>[double.infinity]));
    final ByteData d = w.done();
    File('build/serialization.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(jsonEncode(<String, Object>{'bytes': d.buffer.asUint8List(d.offsetInBytes, d.lengthInBytes)}));
  });
}
