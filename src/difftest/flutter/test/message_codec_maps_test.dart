// Reference side of flutter-go's map-order differential test for the
// message codecs: Dart's Maps keep their insertion order through
// StandardMessageCodec, StandardMethodCodec, JSONMessageCodec and
// JSONMethodCodec. Maps of every key type the standard codec carries
// (strings, ints, doubles, bools, null, lists, typed data, maps) are
// written in many insertion orders, nested, decoded, changed and written
// again; each case records the bytes, the decoded map's iteration order
// (desc) and its toString. The Go twin is ../../message_codec_maps_test.go;
// cases are matched by name.
// Output: build/message_codec_maps.json as {"cases": [{"name", "out"}...]}.
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

// desc is a type-tagged description of a decoded value, maps in their
// iteration order.
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
    return 'Map:{${v.entries.map((MapEntry<Object?, Object?> e) => '${desc(e.key)}=${desc(e.value)}').join(',')}}';
  }
  return 'other:$v';
}

final List<Map<String, Object?>> cases = <Map<String, Object?>>[];

void record(String name, String Function() body) {
  String out;
  try {
    out = body();
  } catch (e) {
    out = 'throws ${e.runtimeType}: $e';
  }
  cases.add(<String, Object?>{'name': name, 'out': out});
}

const StandardMessageCodec standard = StandardMessageCodec();
const StandardMethodCodec standardMethod = StandardMethodCodec();
const JSONMessageCodec json = JSONMessageCodec();
const JSONMethodCodec jsonMethod = JSONMethodCodec();

// A deterministic generator shared with the Go side (an LCG on 31 bits).
int _state = 0;
void seed(int s) => _state = s;
int next(int n) {
  _state = (_state * 1103515245 + 12345) & 0x7fffffff;
  return _state % n;
}

// shuffled is a Fisher-Yates shuffle of 0..n-1 driven by next.
List<int> shuffled(int n) {
  final List<int> order = List<int>.generate(n, (int i) => i);
  for (var i = n - 1; i > 0; i--) {
    final int j = next(i + 1);
    final int t = order[i];
    order[i] = order[j];
    order[j] = t;
  }
  return order;
}

// The keys of every type the standard codec carries; lists, typed data and
// maps are keys by identity (a new object per call).
List<Object?> allKeys() => <Object?>[
  'b',
  'a',
  '',
  3,
  -1,
  0x7fffffffffffffff,
  1.5,
  -0.25,
  true,
  false,
  null,
  <Object?>[1, 'x'],
  Uint8List.fromList(<int>[9, 8]),
  Int32List.fromList(<int>[7]),
  Float64List.fromList(<double>[2.5]),
  <Object?, Object?>{'k': 1, 'j': <Object?>[]},
];

// The value stored under the key at position i.
Object? valueFor(int i) => switch (i % 5) {
  0 => i,
  1 => 'v$i',
  2 => i + 0.5,
  3 => <Object?>[i, null],
  _ => <Object?, Object?>{'n$i': i, 'm': null},
};

String roundTrip(MessageCodec<Object?> codec, Object? message) {
  final ByteData? encoded = codec.encodeMessage(message);
  final Object? decoded = codec.decodeMessage(encoded);
  final ByteData? again = codec.encodeMessage(decoded);
  return '${hex(encoded)} -> ${desc(decoded)} | $decoded | again ${hex(again) == hex(encoded)}';
}

void main() {
  test('message codec maps', () {
    // ---- Every key type, in many insertion orders.
    seed(20251126);
    for (var round = 0; round < 40; round++) {
      final List<Object?> keys = allKeys();
      final int count = 1 + next(keys.length);
      final List<int> order = shuffled(keys.length).sublist(0, count);
      final Map<Object?, Object?> m = <Object?, Object?>{};
      for (final int i in order) {
        m[keys[i]] = valueFor(i);
      }
      record('standard order $round ${order.join(',')}', () => roundTrip(standard, m));
    }

    // ---- All 24 orders of four keys of different types.
    final List<Object?> four = <Object?>['z', 2, null, 0.5];
    void permute(List<int> prefix, List<int> rest) {
      if (rest.isEmpty) {
        final Map<Object?, Object?> m = <Object?, Object?>{for (final int i in prefix) four[i]: i};
        record('standard permutation ${prefix.join(',')}', () => roundTrip(standard, m));
        record('json permutation ${prefix.join(',')}', () {
          final Map<String, Object?> j = <String, Object?>{for (final int i in prefix) '${four[i]}': i};
          final ByteData? encoded = json.encodeMessage(j);
          final Object? decoded = json.decodeMessage(encoded);
          return '${utf8.decode(encoded!.buffer.asUint8List())} -> ${desc(decoded)} | $decoded';
        });
        return;
      }
      for (var i = 0; i < rest.length; i++) {
        permute(<int>[...prefix, rest[i]], <int>[...rest.sublist(0, i), ...rest.sublist(i + 1)]);
      }
    }

    permute(<int>[], <int>[0, 1, 2, 3]);

    // ---- Nested maps, each level in its own order.
    seed(7);
    for (var round = 0; round < 12; round++) {
      Map<Object?, Object?> build(int depth) {
        final List<int> order = shuffled(5);
        final Map<Object?, Object?> m = <Object?, Object?>{};
        for (final int i in order.sublist(0, 2 + next(4))) {
          final Object? key = <Object?>['k$i', i, i * 1.25, i.isEven, null][next(5)];
          m[key] = depth == 0 || next(3) == 0 ? valueFor(i) : build(depth - 1);
        }
        if (next(2) == 0) {
          m[<Object?>[depth]] = <Object?>[build(0), build(0)];
        }
        return m;
      }

      final Map<Object?, Object?> m = build(2);
      record('standard nested $round', () => roundTrip(standard, m));
      record('standard nested $round method', () {
        final ByteData call = standardMethod.encodeMethodCall(MethodCall('m', m));
        final MethodCall decoded = standardMethod.decodeMethodCall(call);
        final Object? envelope = standardMethod.decodeEnvelope(standardMethod.encodeSuccessEnvelope(m));
        return '${hex(call)} -> ${desc(decoded.arguments)} | $decoded | ${desc(envelope)}';
      });
    }

    // ---- Keys the decoded map merges, in both orders, around other keys.
    final Map<String, List<Object?>> merging = <String, List<Object?>>{
      'int double': <Object?>['a', 0, 1, 1, 'b', 2, 1.0, 3, 'c', 4],
      'double int': <Object?>[1.0, 0, 'a', 1, 1, 2, 'b', 3],
      'zeros': <Object?>[-0.0, 0, 'x', 1, 0, 2, 0.0, 3],
      'strings': <Object?>['b', 0, 'a', 1, 'b', 2, 'c', 3, 'a', 4],
      'nulls': <Object?>[null, 0, 'n', 1, null, 2],
      'equal lists': <Object?>[<Object?>[1], 0, 'l', 1, <Object?>[1], 2],
      'empty lists': <Object?>[<Object?>[], 0, <Object?>[], 1, Uint8List(0), 2, Uint8List(0), 3, Int32List(0), 4, Int32List(0), 5],
      'equal maps': <Object?>[<Object?, Object?>{'a': 1}, 0, <Object?, Object?>{'a': 1}, 1],
      'nan': <Object?>[double.nan, 0, 'x', 1, double.nan, 2],
      'big ints': <Object?>[0x7fffffffffffffff, 0, 9223372036854775808.0, 1, 9007199254740993, 2, 9007199254740992.0, 3],
    };
    merging.forEach((String name, List<Object?> kv) {
      record('standard merging $name', () {
        final WriteBuffer buffer = WriteBuffer();
        buffer.putUint8(13);
        standard.writeSize(buffer, kv.length ~/ 2);
        for (final Object? v in kv) {
          standard.writeValue(buffer, v);
        }
        final ByteData bytes = buffer.done();
        final Object? decoded = standard.decodeMessage(bytes);
        return '${hex(bytes)} -> ${desc(decoded)} | $decoded | ${hex(standard.encodeMessage(decoded))}';
      });
    });

    // ---- Changing a decoded map: removed keys go last when set again,
    // replaced values keep their place.
    record('standard mutate', () {
      final Map<Object?, Object?> m =
          standard.decodeMessage(standard.encodeMessage(<Object?, Object?>{'a': 1, 'b': 2, 3: 'c', null: 'd', 1.5: 'e'}))!
              as Map<Object?, Object?>;
      final List<String> log = <String>[];
      log.add('remove b ${m.remove('b')}');
      log.add('remove x ${m.remove('x')}');
      m['b'] = 20;
      m['a'] = 10;
      m[3.0] = 'C';
      log.add('putIfAbsent ${m.putIfAbsent('e', () => 5)} ${m.putIfAbsent('a', () => 0)}');
      m.remove(null);
      m[null] = 'D';
      log.add('lookups ${m[3]} ${m[3.0]} ${m[1.5]} ${m['zz']} ${m.containsKey(3.0)} ${m.containsKey('b')}');
      log.add('keys ${m.keys.toList()} values ${m.values.toList()} length ${m.length}');
      return '${log.join('; ')} | $m | ${hex(standard.encodeMessage(m))}';
    });
    record('standard mutate all', () {
      final Map<Object?, Object?> m =
          standard.decodeMessage(standard.encodeMessage(<Object?, Object?>{'a': 1, 'b': 2}))! as Map<Object?, Object?>;
      m.remove('a');
      m.remove('b');
      m['b'] = 3;
      m.addAll(<Object?, Object?>{'c': 4, 'b': 5, 'a': 6});
      return '$m | ${hex(standard.encodeMessage(m))}';
    });

    // ---- JSON: objects keep the text's order (a duplicate key keeps its
    // first place and its last value), and are written in their order.
    final List<String> texts = <String>[
      '{"z":1,"a":2,"m":3}',
      '{"b":{"y":1,"x":[{"q":1,"p":2}]},"a":null}',
      '{"a":1,"b":2,"a":3}',
      '{"c":1,"b":2,"c":3,"a":4,"b":5}',
      '[{"2":0,"10":1,"1":2},{}]',
      '{"é":1,"e":2,"":3,"😂":4,"E":5}',
    ];
    for (final String text in texts) {
      record('json decode $text', () {
        final Object? decoded = json.decodeMessage(ByteData.sublistView(Uint8List.fromList(utf8.encode(text))));
        return '${desc(decoded)} | $decoded | ${utf8.decode(json.encodeMessage(decoded)!.buffer.asUint8List())}';
      });
    }
    record('json method call', () {
      final MethodCall call = jsonMethod.decodeMethodCall(
        ByteData.sublistView(Uint8List.fromList(utf8.encode('{"args":{"z":1,"a":[{"d":4,"c":3}]},"method":"m"}'))),
      );
      final ByteData encoded = jsonMethod.encodeMethodCall(call);
      return '$call | ${utf8.decode(encoded.buffer.asUint8List())}';
    });
    record('json envelope', () {
      final Object? result = jsonMethod.decodeEnvelope(
        ByteData.sublistView(Uint8List.fromList(utf8.encode('[{"b":1,"a":{"d":1,"c":2}}]'))),
      );
      return '${desc(result)} | ${utf8.decode(jsonMethod.encodeSuccessEnvelope(result).buffer.asUint8List())}';
    });
    record('json error envelope', () {
      try {
        jsonMethod.decodeEnvelope(
          ByteData.sublistView(Uint8List.fromList(utf8.encode('["code","message",{"y":1,"x":2}]'))),
        );
        return 'no error';
      } on PlatformException catch (e) {
        return '${desc(e.details)} | $e';
      }
    });
    record('standard error envelope', () {
      try {
        standardMethod.decodeEnvelope(
          standardMethod.encodeErrorEnvelope(code: 'c', details: <Object?, Object?>{'y': 1, 'x': <Object?, Object?>{2: 1, 1: 2}}),
        );
        return 'no error';
      } on PlatformException catch (e) {
        return '${desc(e.details)} | $e';
      }
    });

    File('build/message_codec_maps.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'cases': cases}));
  });
}
