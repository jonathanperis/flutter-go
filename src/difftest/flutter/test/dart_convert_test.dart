// Reference side of flutter-go's dart:convert differential test (src/dart's
// UTF8Encode/UTF8Decode, JSONUTF8Encode, JSONUTF8Decode, JSONDecode): the
// exact bytes JsonUtf8Encoder writes for a table of values, and what
// utf8.decode, json.decode and utf8.decoder.fuse(json.decoder) (the fused
// decoder Flutter's JSONMessageCodec uses) return or throw for a table of
// inputs. Output: build/dart_convert.json. The Go side
// (difftest/dart_convert_test.go) builds the same encode values (by label)
// and replays the decode inputs read from this file.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';

class NoToJson {}

class WithToJson {
  WithToJson(this.value);
  final Object? value;
  Object? toJson() => value;
}

String hex(int b) => b.toRadixString(16).padLeft(2, '0');

// A canonical text of a decoded value: maps in their iteration order,
// strings as UTF-16 code units, doubles by toString.
String describe(Object? v) {
  if (v == null) {
    return 'null';
  }
  if (v is bool) {
    return 'bool:$v';
  }
  if (v is int) {
    return 'int:$v';
  }
  if (v is double) {
    return 'double:$v';
  }
  if (v is String) {
    return 'str:${v.codeUnits.map((int c) => c.toRadixString(16)).join(',')}';
  }
  if (v is List) {
    return '[${v.map(describe).join(',')}]';
  }
  if (v is Map) {
    final List<String> entries = <String>[
      for (final MapEntry<Object?, Object?> e in v.entries) '${describe(e.key)}=${describe(e.value)}',
    ];
    return '{${entries.join(',')}}';
  }
  return 'other:${v.runtimeType}';
}

Map<String, Object?> run(String kind, List<int> input, Object? Function() f) {
  try {
    return <String, Object?>{'kind': kind, 'in': input, 'ok': describe(f())};
  } catch (e) {
    return <String, Object?>{'kind': kind, 'in': input, 'err': '${e.runtimeType}|$e'};
  }
}

void main() {
  test('dart:convert', () {
    final Map<String, Object?> encode = <String, Object?>{};
    void enc(String label, Object? value) {
      try {
        encode[label] = JsonUtf8Encoder().convert(value).map(hex).join();
      } catch (e) {
        encode[label] = 'ERR ${e.runtimeType}|$e';
      }
    }

    enc('null', null);
    enc('true', true);
    enc('false', false);
    enc('0', 0);
    enc('-1', -1);
    enc('maxInt', 9223372036854775807);
    enc('minInt', -9223372036854775807 - 1);
    enc('2^32', 4294967296);
    enc('0.0', 0.0);
    enc('-0.0', -0.0);
    enc('1.0', 1.0);
    enc('1.5', 1.5);
    enc('0.1+0.2', 0.1 + 0.2);
    enc('1e21', 1e21);
    enc('1e20', 1e20);
    enc('1e-7', 1e-7);
    enc('1e-6', 1e-6);
    enc('123456789.125', 123456789.125);
    enc('maxFinite', double.maxFinite);
    enc('-maxFinite', -double.maxFinite);
    enc('minPositive', double.minPositive);
    enc('1/3', 1 / 3);
    enc('empty string', '');
    enc('hello', 'hello');
    enc('controls', String.fromCharCodes(<int>[for (int i = 0; i < 32; i++) i, 0x7f]));
    enc('quotes', '"\\/\'');
    enc('latin1', 'é ÿ');
    enc('bmp', '€    ﻿ <>&');
    enc('astral', '😂𝄞');
    enc('254a+emoji', '${'a' * 254}😂');
    enc('255a+emoji', '${'a' * 255}😂');
    enc('lone lead', 'a\uD800b');
    enc('lone trail', '\uDC00');
    enc('lone lead at end', 'x\uDBFF');
    enc('reversed pair', '\uDE02\uD83D');
    enc('empty list', <Object?>[]);
    enc('nested lists', <Object?>[<Object?>[], <Object?>[1, <Object?>[2, <Object?>[3]]]]);
    enc('mixed list', <Object?>[null, true, 1, -2.5, 'x', <String, Object?>{}]);
    enc('empty map', <String, Object?>{});
    enc('empty int map', <int, Object?>{});
    enc('map', <String, Object?>{'a': 1, 'b': <Object?>[true, null]});
    enc('ordered map', <String, Object?>{'z': 1, 'a': 2, 'm': 3});
    enc('map key escapes', <String, Object?>{'a"b\n': 'é'});
    enc('nested map', <String, Object?>{
      'k': <String, Object?>{'x': <Object?>[1.5, <String, Object?>{'y': null}]},
    });
    enc('Uint8List', Uint8List.fromList(<int>[0, 1, 255]));
    enc('Int32List', Int32List.fromList(<int>[-2147483648, 2147483647]));
    enc('Int64List', Int64List.fromList(<int>[-9223372036854775807 - 1, 9223372036854775807]));
    enc('Float32List', Float32List.fromList(<double>[0.1, -0.0, 3.4028234663852886e38]));
    enc('Float64List', Float64List.fromList(<double>[1.5, -0.0, 5e-324]));
    enc('toJson', WithToJson(<String, Object?>{'x': 1}));
    enc('toJson in list', <Object?>[WithToJson('v'), 2]);
    enc('NaN', double.nan);
    enc('Infinity', double.infinity);
    enc('-Infinity', double.negativeInfinity);
    enc('NaN in list', <Object?>[1, double.nan]);
    enc('NaN in map', <String, Object?>{'a': double.infinity});
    enc('int key', <Object?, Object?>{1: 'a'});
    enc('null key', <Object?, Object?>{null: 'a'});
    enc('mixed keys', <Object?, Object?>{'a': 1, 2: 'b'});
    enc('no toJson', NoToJson());
    enc('toJson NaN', WithToJson(double.nan));
    enc('toJson list NaN', WithToJson(<Object?>[double.nan]));
    final List<Object?> cyclic = <Object?>[1];
    cyclic.add(cyclic);
    enc('cyclic list', cyclic);

    final List<Map<String, Object?>> decode = <Map<String, Object?>>[];
    final List<String> jsonTexts = <String>[
      'null', 'true', 'false', '0', '-0', '1', '-1', '01', '-01', '00', '1.', '.5', '1e', '1e+',
      '1e-5', '1E400', '-1E400', '1e-400', '0e0', '1.0', '-0.0', '-0e5', '0.5e1',
      '9223372036854775807', '9223372036854775808', '-9223372036854775808', '-9223372036854775809',
      '123456789012345678901234567890', '1.7976931348623157e308', '1.8e308',
      '2.2250738585072014e-308', '4.9e-324', '2e-324', '0.1', '3.141592653589793', '1.5e300',
      '12345678901234567.5', '0.000001', '-', '--1', '+1', 'NaN', 'Infinity', '-a', '1ea', '1e+a',
      '1.a', '"abc"', r'"é😂\/\b\f\n\r\t\"\\"', r'"\u12"', r'"\u12G4"', r'"ꯍx"',
      r'"\x"', r'"\', '"abc', '"a\nb"', '"a\u0000"', 'r"a\\', r'"a\nb' '\u0001"', r'"\' '\u0001"',
      'tru', 'trux', 'nul', 'nulx', 'fals', 'falsy', 'f', 'n', 't', 'truee', 'nullnull',
      '[', '[1', '[1,', '[1,]', '[,1]', '[1 2]', ']', '[]]', '[] x', '{', '{"a"', '{"a":', '{"a":1',
      '{"a":1,}', '{"a" 1}', '{1:2}', '{"a":1,"a":2}', '{"a":1,"b":2,"a":3}', '{,}', '{"a",1}',
      '{"a":{"b":[1,{"c":null}]}}', ' \t\r\n[ 1 , 2 ] \n', '', ' ', '\f1', '\u000b1', '"é"',
      '"😂"', '[1e5, 1E-5, -0.5e+3]', '1 2', '[0.1]', '[1e22]', '[1e23]', '[9007199254740993]',
      '[9007199254740993.0]', '[123.456e-20]', '[1.7976931348623157e308]', '[1e400]', '[-1e400]',
      '[0e999]', '[-0e999]', '[1e-999]', '[-0.0]', '[-0]', '[18446744073709551616]',
      '[9223372036854775807]', '[9223372036854775808]', '[-9223372036854775808]',
      '[-9223372036854775809]', '[922337203685477580]', '[-922337203685477580]',
      '[922337203685477589]', '[-922337203685477589]', '[1234567890123456789]',
      '[12345678901234567890]', '[0.30000000000000004]', '[1.00000000000000011102230246251565404236316680908203125]',
      '[5e-324, 2.4703282292062328e-324]', '[1e-22, 1e-23, 4.5e-22]', '[99999999999999999999e-30]',
      '[1.5e]', '[1.5e-]', '[1.]', '[-]', '[01]', '[00]', '[0.0e+0]',
      '[true,false,null]', '[true ,false, null ]', '[nul]', '[tru e]', '{"é":"😂","":""}',
      '${'x' * 100}', '[${'1,' * 50}x]', '[\n1,\n2,\nx]', '[\r\n1,\r\n x]', '[\r1,\n\rx]',
      '${'1' * 400}', '[${'2' * 400}]', '"${'a' * 300}', '[${'1, ' * 60}"\u0002"]',
    ];
    for (final String text in jsonTexts) {
      final List<int> bytes = utf8.encode(text);
      decode.add(run('json8', bytes, () => utf8.decoder.fuse(json.decoder).convert(bytes)));
      decode.add(run('json16', text.codeUnits, () => json.decode(text)));
    }
    // Non-ASCII outside strings (UTF-16 parser).
    for (final String text in <String>['é', 'Ā', '[1,é]', '"a"Ā', '[ 1]', ' []']) {
      decode.add(run('json16', text.codeUnits, () => json.decode(text)));
    }
    final List<List<int>> rawJson = <List<int>>[
      <int>[0xEF, 0xBB, 0xBF, 0x5B, 0x31, 0x5D],
      <int>[0xEF, 0xBB, 0xBF],
      <int>[0xEF, 0xBB],
      <int>[0xEF],
      <int>[0xEF, 0x31],
      <int>[0xEF, 0xBB, 0x31],
      <int>[0xEF, 0xBB, 0xBF, 0xEF, 0xBB, 0xBF, 0x31],
      <int>[0x20, 0xEF, 0xBB, 0xBF, 0x31],
      <int>[0x22, 0xEF, 0xBB, 0xBF, 0x61, 0x22],
      <int>[0x22, 0xC3, 0x22],
      <int>[0x22, 0x80, 0x22],
      <int>[0x22, 0xED, 0xA0, 0x80, 0x22],
      <int>[0x22, 0xC0, 0x80, 0x22],
      <int>[0x22, 0xC1, 0xBF, 0x22],
      <int>[0x22, 0xE0, 0x80, 0x80, 0x22],
      <int>[0x22, 0xF0, 0x80, 0x80, 0x80, 0x22],
      <int>[0x22, 0xF4, 0x90, 0x80, 0x80, 0x22],
      <int>[0x22, 0xF5, 0x80, 0x80, 0x80, 0x22],
      <int>[0x22, 0xFF, 0x22],
      <int>[0x22, 0xC3, 0x5C, 0x6E, 0x22],
      <int>[0x22, 0xE2, 0x82, 0x22],
      <int>[0x22, 0xC3, 0x41, 0x22],
      <int>[0x22, 0x61, 0x5C, 0x6E, 0x62, 0xC3, 0xA9, 0x80, 0x22],
      <int>[0x22, 0x61, 0x5C, 0x6E, 0xC3, 0xA9, 0x22],
      <int>[0x22, 0xC3, 0xA9, 0x5C, 0x75, 0x30, 0x30, 0x65, 0x39, 0xE2, 0x82, 0xAC, 0x22],
      <int>[0x22, 0xF0, 0x9F, 0x98, 0x22],
      <int>[0x22, 0xF0, 0x9F, 0x98, 0x82],
      <int>[0x22, 0xC3, 0xA9, 0x01, 0x22],
      <int>[0x22, 0xC3, 0x01, 0x22],
      <int>[0x5B, 0xC3, 0xA9, 0x5D],
      <int>[0x80],
      <int>[0xFF],
      <int>[0x5B, 0x31, 0x5D, 0xC3],
      <int>[0x7B, 0x22, 0xC3, 0xA9, 0x22, 0x3A, 0x22, 0x80, 0x22, 0x7D],
    ];
    for (final List<int> bytes in rawJson) {
      final Uint8List input = Uint8List.fromList(bytes);
      decode.add(run('json8', bytes, () => utf8.decoder.fuse(json.decoder).convert(input)));
    }
    final List<List<int>> rawUtf8 = <List<int>>[
      <int>[],
      utf8.encode('abc'),
      utf8.encode('é€😂'),
      <int>[0xEF, 0xBB, 0xBF, 0x61],
      <int>[0xEF, 0xBB, 0xBF],
      <int>[0xEF, 0xBB, 0xBF, 0xEF, 0xBB, 0xBF],
      <int>[0x61, 0xEF, 0xBB, 0xBF],
      <int>[0xEF, 0xBB],
      <int>[0xC3],
      <int>[0x80],
      <int>[0x61, 0x80, 0x62],
      <int>[0xC3, 0x41],
      <int>[0xC3, 0xC3, 0xA9],
      <int>[0xED, 0xA0, 0x80],
      <int>[0xED, 0x9F, 0xBF],
      <int>[0xC0, 0x80],
      <int>[0xC1, 0x80],
      <int>[0xE0, 0x80, 0x80],
      <int>[0xE0, 0xA0, 0x80],
      <int>[0xF0, 0x80, 0x80, 0x80],
      <int>[0xF0, 0x90, 0x80, 0x80],
      <int>[0xF4, 0x8F, 0xBF, 0xBF],
      <int>[0xF4, 0x90, 0x80, 0x80],
      <int>[0xF5, 0x80],
      <int>[0xFE],
      <int>[0xFF],
      <int>[0xE2, 0x82],
      <int>[0xF0, 0x9F, 0x98],
      <int>[0xE9, 0x61],
      <int>[0x61, 0x62, 0xC3, 0xA9, 0x80],
      <int>[0x61, 0x62, 0x63, 0xC3],
      <int>[0xC3, 0xA9, 0xC3],
      <int>[0xC2, 0x80, 0xDF, 0xBF],
      <int>[0xEF, 0xBF, 0xBF],
    ];
    for (final List<int> bytes in rawUtf8) {
      final Uint8List input = Uint8List.fromList(bytes);
      decode.add(run('utf8', bytes, () => utf8.decode(input)));
    }
    // utf8.encode of strings, given as code units (lone surrogates included).
    final List<Map<String, Object?>> utf8Encode = <Map<String, Object?>>[];
    for (final String s in <String>['', 'abc', 'é€😂', 'a\uD800b', '\uDC00', '\uD83D', '\uDE02\uD83D', '\u0000\u007f\u0080߿ࠀ￿']) {
      utf8Encode.add(<String, Object?>{'in': s.codeUnits, 'out': utf8.encode(s)});
    }
    File('build/dart_convert.json').writeAsStringSync(
      jsonEncode(<String, Object?>{'encode': encode, 'decode': decode, 'utf8Encode': utf8Encode}),
    );
  });
}
