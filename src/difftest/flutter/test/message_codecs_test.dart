// Reference side of flutter-go's message codec differential test: the exact
// bytes Flutter's codecs (StandardMessageCodec, StandardMethodCodec,
// JSONMessageCodec, JSONMethodCodec, StringCodec) write for a wide set of
// values, what they decode those bytes (and hand-made ones) to, and their
// errors. The Go twin is ../../message_codecs_test.go; cases are matched by
// name.
// Output: build/message_codecs.json as {"cases": [{"name", "out"}...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

String hex(ByteData? d) {
  if (d == null) {
    return 'null';
  }
  final Uint8List b = d.buffer.asUint8List(d.offsetInBytes, d.lengthInBytes);
  return b.map((int x) => x.toRadixString(16).padLeft(2, '0')).join();
}

String bits(double v) {
  final ByteData b = ByteData(8)..setFloat64(0, v);
  return hex(b);
}

// desc is a canonical, type-tagged description of a decoded value (maps in
// their iteration order, which flutter-go's decoded maps keep).
String desc(Object? v) {
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
    return 'double:${bits(v)}';
  }
  if (v is String) {
    return 'String:${jsonEncode(v)}';
  }
  if (v is Uint8List) {
    return 'Uint8List:${v.join(',')}';
  }
  if (v is Int32List) {
    return 'Int32List:${v.join(',')}';
  }
  if (v is Int64List) {
    return 'Int64List:${v.join(',')}';
  }
  if (v is Float32List) {
    return 'Float32List:${v.map(bits).join(',')}';
  }
  if (v is Float64List) {
    return 'Float64List:${v.map(bits).join(',')}';
  }
  if (v is List) {
    return 'List:[${v.map(desc).join(',')}]';
  }
  if (v is Map) {
    final List<String> entries =
        v.entries.map((MapEntry<Object?, Object?> e) => '${desc(e.key)}=${desc(e.value)}').toList();
    return 'Map:{${entries.join(',')}}';
  }
  return 'other:$v';
}

String errorText(Object e) {
  if (e is FormatException) {
    return 'FormatException:$e';
  }
  if (e is PlatformException) {
    return 'PlatformException:${e.code}|${e.message}|${desc(e.details)}|${e.stacktrace}|$e';
  }
  if (e is MissingPluginException) {
    return 'MissingPluginException:$e';
  }
  if (e is RangeError) {
    return 'RangeError';
  }
  if (e is ArgumentError) {
    return 'ArgumentError:$e';
  }
  if (e is JsonUnsupportedObjectError) {
    return 'JsonUnsupportedObjectError:$e';
  }
  if (e is RangeError) {
    return 'RangeError';
  }
  if (e is TypeError) {
    return 'TypeError:$e';
  }
  return 'other:${e.runtimeType}';
}

final List<Map<String, Object?>> cases = <Map<String, Object?>>[];

void record(String name, String Function() body) {
  String out;
  try {
    out = body();
  } catch (e) {
    out = 'throws ${errorText(e)}';
  }
  cases.add(<String, Object?>{'name': name, 'out': out});
}

ByteData bytesOf(List<int> b) => ByteData.sublistView(Uint8List.fromList(b));

const StandardMessageCodec standard = StandardMessageCodec();
const StandardMethodCodec standardMethod = StandardMethodCodec();
const JSONMessageCodec json = JSONMessageCodec();
const JSONMethodCodec jsonMethod = JSONMethodCodec();
const StringCodec string = StringCodec();

// roundTrip encodes value with codec and decodes the bytes back.
void roundTrip(String name, MessageCodec<Object?> codec, Object? value) {
  record(name, () {
    final ByteData? encoded = codec.encodeMessage(value);
    String decoded;
    try {
      decoded = desc(codec.decodeMessage(encoded));
    } catch (e) {
      decoded = 'throws ${errorText(e)}';
    }
    return '${hex(encoded)} -> $decoded';
  });
}

void decode(String name, MessageCodec<Object?> codec, List<int>? bytes) {
  record(name, () => desc(codec.decodeMessage(bytes == null ? null : bytesOf(bytes))));
}

String decodeEnvelope(MethodCodec codec, ByteData envelope) => desc(codec.decodeEnvelope(envelope));

String decodeCall(MethodCodec codec, ByteData? bytes) {
  final MethodCall call = codec.decodeMethodCall(bytes);
  return '${call.method} ${desc(call.arguments)} | $call';
}

List<Object?> standardValues() {
  return <Object?>[
    null,
    true,
    false,
    0,
    1,
    -1,
    127,
    255,
    -0x7fffffff - 1,
    0x7fffffff,
    -0x7fffffff - 2,
    0x7fffffff + 1,
    -0x7fffffffffffffff - 1,
    0x7fffffffffffffff,
    9007199254740993,
    0.0,
    -0.0,
    1.0,
    0.1,
    -3.14,
    double.nan,
    double.infinity,
    double.negativeInfinity,
    double.maxFinite,
    double.minPositive,
    2.2250738585072014e-308,
    1e300,
    '',
    'a',
    'hello world',
    'a' * 253,
    'b' * 254,
    'c' * 300,
    'd' * 65535,
    'e' * 65536,
    'é',
    '日本語',
    '\u{1F602}',
    'a\u{1F602}b\u0000c\n\t"\\',
    '﻿bom',
    'x﻿',
    'mixed ascii then é and 😂 end',
    '\u0080߿ࠀ￿\u{10000}\u{10FFFF}',
    Uint8List(0),
    Uint8List.fromList(<int>[1, 2, 3, 255]),
    Uint8List(254),
    Int32List.fromList(<int>[-0x7fffffff - 1, 0, 0x7fffffff]),
    Int64List.fromList(<int>[-0x7fffffffffffffff - 1, 0, 0x7fffffffffffffff]),
    Float32List.fromList(<double>[0.1, -0.0, double.infinity, double.nan, 3.4028234663852886e38, 1e-45]),
    Float64List.fromList(<double>[0.1, -0.0, double.infinity, double.nan, double.maxFinite, double.minPositive]),
    Int32List(0),
    Float64List(0),
    <Object?>[],
    <Object?>[null, 1, 'two', 3.0, <Object?>[]],
    <Object?>[<Object?>[<Object?>[1]]],
    for (int pad = 0; pad < 8; pad++)
      <Object?>[
        ...List<Object?>.filled(pad, null),
        Int32List.fromList(<int>[7]),
        null,
        Int64List.fromList(<int>[8]),
        true,
        Float32List.fromList(<double>[1.5]),
        false,
        Float64List.fromList(<double>[2.5]),
        3.5,
      ],
    <Object?, Object?>{},
    <Object?, Object?>{'b': 1, 'a': 2},
    <Object?, Object?>{1: 'one', 2: 'two', null: 'null', true: 'true', 2.5: 'double'},
    <Object?, Object?>{'nested': <Object?, Object?>{'list': <Object?>[1, 2], 'map': <Object?, Object?>{}}},
    <Object?, Object?>{'z': Uint8List.fromList(<int>[1]), 'y': Float64List.fromList(<double>[1.0])},
    List<Object?>.generate(300, (int i) => i),
    <Object?, Object?>{for (int i = 0; i < 260; i++) 'k$i': i},
  ];
}

void main() {
  test('message codecs', () {
    // ---- StandardMessageCodec round trips.
    final List<Object?> values = standardValues();
    for (int i = 0; i < values.length; i++) {
      roundTrip('standard $i', standard, values[i]);
    }

    // ---- StandardMessageCodec decoding hand-made bytes.
    final Map<String, List<int>?> raw = <String, List<int>?>{
      'null message': null,
      'empty': <int>[],
      'trailing': <int>[0, 0],
      'unknown type': <int>[15],
      'type 127': <int>[127],
      'type 128': <int>[128],
      'truncated int32': <int>[3, 1],
      'truncated int64': <int>[4, 1, 2, 3, 4],
      'truncated float64': <int>[6, 0, 0, 0, 0, 0, 0, 0, 0],
      'truncated string': <int>[7, 5, 97],
      'missing size': <int>[7],
      'truncated size 254': <int>[7, 254, 1],
      'truncated size 255': <int>[7, 255, 1, 0],
      'missing list element': <int>[12, 2, 0],
      'missing map value': <int>[13, 1, 0],
      'bad utf8': <int>[7, 2, 0xc3, 0x28],
      'encoded surrogate': <int>[7, 3, 0xed, 0xa0, 0x80],
      'overlong': <int>[7, 2, 0xc0, 0x80],
      'unfinished utf8': <int>[7, 1, 0xe6],
      'bom string': <int>[7, 4, 0xef, 0xbb, 0xbf, 0x41],
      'double bom string': <int>[7, 6, 0xef, 0xbb, 0xbf, 0xef, 0xbb, 0xbf],
      'large int': <int>[5, 4, 0x31, 0x66, 0x66, 0x66],
      'size 254 form small': <int>[8, 254, 1, 0, 9],
      'size 255 form small': <int>[8, 255, 2, 0, 0, 0, 9, 8],
      'unaligned double': <int>[6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xf0, 0x3f],
      'int32 list no padding': <int>[9, 1, 0, 0, 1, 0, 0, 0],
    };
    raw.forEach((String name, List<int>? b) => decode('standard decode $name', standard, b));

    // ---- Maps with keys Dart's == merges (or doesn't): written entry by
    // entry, then decoded.
    final Map<String, List<Object?>> mapEntries = <String, List<Object?>>{
      'int then double': <Object?>[1, 'a', 1.0, 'b'],
      'double then int': <Object?>[1.0, 'a', 1, 'b'],
      'zeros': <Object?>[-0.0, 'a', 0.0, 'b', 0, 'c'],
      'int zero then neg zero': <Object?>[0, 'a', -0.0, 'b'],
      'pos zero then neg zero': <Object?>[0.0, 'a', -0.0, 'b'],
      'duplicate strings': <Object?>['a', 1, 'a', 2, 'b', 3],
      'nan keys': <Object?>[double.nan, 1, double.nan, 2],
      'int64 equal double': <Object?>[9007199254740992, 'a', 9007199254740992.0, 'b'],
      'int64 unequal double': <Object?>[9007199254740993, 'a', 9007199254740992.0, 'b'],
      'double then int64 unequal': <Object?>[9007199254740992.0, 'a', 9007199254740993, 'b'],
      'min int64 and double': <Object?>[-0x7fffffffffffffff - 1, 'a', -9223372036854775808.0, 'b'],
      'huge double': <Object?>[9223372036854775808.0, 'a', 0x7fffffffffffffff, 'b'],
      'int max then huge double': <Object?>[0x7fffffffffffffff, 'a', 9223372036854775808.0, 'b'],
      'bools and null': <Object?>[true, 1, null, 2, false, 3, true, 4, null, 5],
      'string and int': <Object?>['1', 1, 1, 2],
    };
    mapEntries.forEach((String name, List<Object?> kv) {
      record('standard map $name', () {
        final WriteBuffer buffer = WriteBuffer();
        buffer.putUint8(13);
        standard.writeSize(buffer, kv.length ~/ 2);
        for (final Object? v in kv) {
          standard.writeValue(buffer, v);
        }
        final ByteData bytes = buffer.done();
        return '${hex(bytes)} -> ${desc(standard.decodeMessage(bytes))}';
      });
    });

    // ---- Unencodable values.
    record('standard unencodable', () => hex(standard.encodeMessage(Object())));
    record('standard unencodable nested', () => hex(standard.encodeMessage(<Object?>[1, Object()])));

    // ---- StandardMethodCodec.
    final List<Object?> args = <Object?>[
      null,
      'hello',
      <Object?>[1, 2.5, 'x'],
      <Object?, Object?>{'a': 1, 'b': <Object?>[true, false]},
      Float64List.fromList(<double>[1.0]),
    ];
    for (int i = 0; i < args.length; i++) {
      record('method call $i', () {
        final ByteData encoded = standardMethod.encodeMethodCall(MethodCall('method$i', args[i]));
        return '${hex(encoded)} -> ${decodeCall(standardMethod, encoded)}';
      });
      record('success envelope $i', () {
        final ByteData encoded = standardMethod.encodeSuccessEnvelope(args[i]);
        return '${hex(encoded)} -> ${decodeEnvelope(standardMethod, encoded)}';
      });
      record('error envelope $i', () {
        final ByteData encoded = standardMethod.encodeErrorEnvelope(
          code: 'code$i',
          message: i.isEven ? 'message $i' : null,
          details: args[i],
        );
        return '${hex(encoded)} -> ${decodeEnvelope(standardMethod, encoded)}';
      });
    }
    final Map<String, List<int>?> rawMethod = <String, List<int>?>{
      'method name not string': <int>[3, 1, 0, 0, 0, 0],
      'method call trailing': <int>[7, 1, 109, 0, 0],
      'method call missing args': <int>[7, 1, 109],
      'method call empty': <int>[],
    };
    rawMethod.forEach((String name, List<int>? b) {
      record('standard $name', () => decodeCall(standardMethod, b == null ? null : bytesOf(b)));
    });
    record('standard method call null', () => decodeCall(standardMethod, null));
    final Map<String, List<int>> rawEnvelopes = <String, List<int>>{
      'empty': <int>[],
      'success': <int>[0, 7, 1, 120],
      'success trailing': <int>[0, 0, 0],
      'success missing value': <int>[0],
      'error with stacktrace': <int>[1, 7, 1, 99, 7, 1, 109, 0, 7, 2, 115, 116],
      'error null stacktrace': <int>[1, 7, 1, 99, 0, 0, 0],
      'error int stacktrace': <int>[1, 7, 1, 99, 0, 0, 3, 1, 0, 0, 0],
      'error code not string': <int>[1, 3, 1, 0, 0, 0, 0, 0],
      'error message not string': <int>[1, 7, 1, 99, 3, 1, 0, 0, 0, 0],
      'error trailing': <int>[1, 7, 1, 99, 0, 0, 0, 0],
      'error missing details': <int>[1, 7, 1, 99, 0],
      'error flag 2': <int>[2, 7, 1, 99, 0, 0],
      'error flag 255': <int>[255, 7, 1, 99, 0, 0],
    };
    rawEnvelopes.forEach((String name, List<int> b) {
      record('standard envelope $name', () => decodeEnvelope(standardMethod, bytesOf(b)));
    });

    // ---- JSONMessageCodec.
    final List<Object?> jsonValues = <Object?>[
      null,
      true,
      false,
      0,
      -7,
      98742923489,
      0x7fffffffffffffff,
      -0x7fffffffffffffff - 1,
      0.0,
      -0.0,
      3.14,
      1e21,
      1e-7,
      123456789.0,
      double.maxFinite,
      double.minPositive,
      '',
      'hello',
      'special chars >☺\u{1F602}<',
      'quote " backslash \\ slash / controls \u0000\u0001\u001f\b\f\n\r\t \u007f ',
      '${'a' * 254}\u{1F602}',
      <Object?>[],
      <Object?>[null, true, 1, 2.5, 'x', <Object?>[], <String, Object?>{}],
      <String, Object?>{'b': 1, 'a': <Object?>[1, 2], 'c': <String, Object?>{'d': null}},
      Uint8List.fromList(<int>[1, 2]),
      Int32List.fromList(<int>[-1]),
      Float64List.fromList(<double>[1.5, 2.0]),
      double.nan,
      double.infinity,
      <Object?>[1, double.negativeInfinity],
      <Object?, Object?>{1: 'x'},
    ];
    for (int i = 0; i < jsonValues.length; i++) {
      roundTrip('json $i', json, jsonValues[i]);
    }
    final Map<String, String?> rawJson = <String, String?>{
      'null message': null,
      'object': '{"a": 1, "b": [true, null, 2.5e3, -0, -0.0, 1E2]}',
      'whitespace': ' \t\r\n[ 1 , 2 ] \n',
      'big int': '9223372036854775808',
      'neg big int': '-9223372036854775809',
      'int max': '9223372036854775807',
      'exponent': '1e400',
      'escapes': r'"é😂\/\b\f\n\r\t\"\\"',
      'empty': '',
      'garbage': '[1,]',
      'trailing': '[1] x',
      'unterminated': '"abc',
      'control char': '"a\u0001"',
      'leading zero': '01',
      'duplicate keys': '{"a": 1, "a": 2}',
    };
    rawJson.forEach((String name, String? s) {
      decode('json decode $name', json, s == null ? null : utf8.encode(s));
    });
    decode('json decode bad utf8', json, <int>[0x22, 0xc3, 0x28, 0x22]);
    decode('json decode bom', json, <int>[0xef, 0xbb, 0xbf, 0x31]);

    // ---- JSONMethodCodec.
    for (int i = 0; i < args.length - 1; i++) {
      record('json method call $i', () {
        final ByteData encoded = jsonMethod.encodeMethodCall(MethodCall('method$i', args[i]));
        return '${hex(encoded)} -> ${decodeCall(jsonMethod, encoded)}';
      });
      record('json success envelope $i', () {
        final ByteData encoded = jsonMethod.encodeSuccessEnvelope(args[i]);
        return '${hex(encoded)} -> ${decodeEnvelope(jsonMethod, encoded)}';
      });
      record('json error envelope $i', () {
        final ByteData encoded = jsonMethod.encodeErrorEnvelope(
          code: 'code$i',
          message: i.isEven ? 'message $i' : null,
          details: args[i],
        );
        return '${hex(encoded)} -> ${decodeEnvelope(jsonMethod, encoded)}';
      });
    }
    final Map<String, String?> rawJsonCalls = <String, String?>{
      'null': null,
      'list': '[1, "x"]',
      'string': '"m"',
      'no method': '{"args": 1}',
      'method not string': '{"method": 1}',
      'method only': '{"method": "m"}',
      'method null args': '{"method": "m", "args": null}',
      'extra keys': '{"method": "m", "args": 1, "x": 2}',
    };
    rawJsonCalls.forEach((String name, String? s) {
      record('json method call decode $name', () => decodeCall(jsonMethod, s == null ? null : bytesOf(utf8.encode(s))));
    });
    final Map<String, String> rawJsonEnvelopes = <String, String>{
      'map': '{"a": 1}',
      'empty list': '[]',
      'one': '[{"x": [1]}]',
      'two': '[1, 2]',
      'three': '["c", "m", {"d": 1}]',
      'three null message': '["c", null, null]',
      'three bad code': '[1, "m", null]',
      'three bad message': '["c", 1, null]',
      'four': '["c", "m", 3, "trace"]',
      'four null': '["c", null, 3, null]',
      'four bad trace': '["c", "m", 3, 4]',
      'four bad message': '["c", 1, 3, "s"]',
      'five': '["c", "m", 3, "s", 5]',
      'null': 'null',
      'string': '"s"',
    };
    rawJsonEnvelopes.forEach((String name, String s) {
      record('json envelope $name', () => decodeEnvelope(jsonMethod, bytesOf(utf8.encode(s))));
    });

    // ---- StringCodec and BinaryCodec.
    for (final String? s in <String?>[null, '', 'hello', '日本😂', '﻿x']) {
      record('string ${jsonEncode(s)}', () {
        final ByteData? encoded = string.encodeMessage(s);
        return '${hex(encoded)} -> ${desc(string.decodeMessage(encoded))}';
      });
    }
    record('string decode bad utf8', () => desc(string.decodeMessage(bytesOf(<int>[0xff]))));
    record('string decode bom', () => desc(string.decodeMessage(bytesOf(<int>[0xef, 0xbb, 0xbf]))));
    record('binary', () => hex(const BinaryCodec().encodeMessage(bytesOf(<int>[1, 2]))));

    // ---- toString.
    record('toString', () => <String>[
          const MethodCall('m').toString(),
          const MethodCall('m', <Object?>[1, 'a', null, 2.0, <Object?>[]]).toString(),
          MethodCall('m', <String, Object?>{'k': 1.5}).toString(),
          PlatformException(code: 'c').toString(),
          PlatformException(code: 'c', message: 'm', details: <Object?>[1, 2], stacktrace: 's').toString(),
          MissingPluginException().toString(),
          MissingPluginException('x').toString(),
        ].join(' | '));

    File('build/message_codecs.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'cases': cases}));
  });
}
