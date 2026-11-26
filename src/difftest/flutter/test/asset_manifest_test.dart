// Reference side of flutter-go's asset manifest differential test: binary
// asset manifests (StandardMessageCodec maps of keys to variant lists)
// parsed by AssetManifest.loadFromAssetBundle, read in various orders
// (listAssets, getAssetVariants: key, dpr, main with content-hash
// stripping), malformed manifests, and the keys PlatformAssetBundle sends
// on flutter/assets (Uri.encodeFull + Uri(path:)). The Go twin is
// ../../asset_manifest_test.go.
// Output: build/asset_manifest.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];

class ManifestBundle extends AssetBundle {
  ManifestBundle(this.manifest);

  final Object? manifest;

  @override
  Future<ByteData> load(String key) async {
    if (key == 'AssetManifest.bin') {
      return const StandardMessageCodec().encodeMessage(manifest)!;
    }
    throw ArgumentError('Unexpected key: $key');
  }
}

String describe(List<AssetMetadata>? variants) {
  if (variants == null) {
    return 'null';
  }
  return variants
      .map((AssetMetadata v) => '(${v.key}, ${v.targetDevicePixelRatio}, ${v.main})')
      .join(' ');
}

String errorName(Object e) => e is TypeError ? 'TypeError' : '${e.runtimeType}';

Future<void> scene(String name, Object? manifest, List<String> reads) async {
  log.add('scene $name');
  final AssetManifest m;
  try {
    m = await AssetManifest.loadFromAssetBundle(ManifestBundle(manifest));
  } catch (e) {
    log.add('load !! ${errorName(e)}');
    return;
  }
  log.add('list ${m.listAssets()}');
  for (final key in reads) {
    try {
      log.add('variants $key ${describe(m.getAssetVariants(key))}');
    } catch (e) {
      log.add('variants $key !! ${errorName(e)}');
    }
    log.add('list ${m.listAssets()}');
  }
}

Map<String, Object?> v(String asset, [Object? dpr]) =>
    <String, Object?>{'asset': asset, if (dpr != null) 'dpr': dpr};

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('asset manifest', () async {
    await scene('flutter test', <String, Object?>{
      'assets/foo.png': <Object?>[v('assets/foo.png'), v('assets/2x/foo.png', 2.0)],
      'assets/bar.png': <Object?>[v('assets/bar.png')],
    }, <String>['assets/bar.png', 'assets/foo.png', 'assets/bar.png', 'nope']);

    await scene('resolutions', <String, Object?>{
      'icons/a.png': <Object?>[
        v('icons/a.png'),
        v('icons/1.5x/a.png', 1.5),
        v('icons/2.0x/a.png', 2.0),
        v('icons/3.0x/a.png', 3.0),
        v('icons/4.0x/a.png', 4.0),
      ],
      'icons/b.png': <Object?>[v('icons/2.0x/b.png', 2.0)],
      'empty': <Object?>[],
      'z': <Object?>[v('z')],
      'a': <Object?>[v('a')],
      'm': <Object?>[v('m')],
    }, <String>['m', 'icons/b.png', 'empty', 'a', 'icons/a.png', 'z', 'm']);

    await scene('content hashes', <String, Object?>{
      'assets/foo.png': <Object?>[v('assets/foo.9f64a747.png'), v('assets/2.0x/foo.74f81fe1.png', 2.0)],
      'assets/sub dir/space image.png': <Object?>[v('assets/sub dir/space image.1a2b3c4d.png')],
      'a/b.js.map': <Object?>[v('a/b.abcdef01.js.map')],
      'a/c.wasm.map': <Object?>[v('a/c.00000000.wasm.map')],
      'a/d.mjs.map': <Object?>[v('a/d.12345678.mjs.map')],
      'a/e.map': <Object?>[v('a/e.12345678.map')],
      'x': <Object?>[v('x.abcdef01')],
      'upper.png': <Object?>[v('upper.ABCDEF01.png')],
      'short.png': <Object?>[v('short.abcdef0.png')],
      'long.png': <Object?>[v('long.abcdef012.png')],
      'nothex.png': <Object?>[v('nothex.abcdefgh.png')],
      'two.11111111.png': <Object?>[v('two.11111111.22222222.png')],
      'two.png': <Object?>[v('two.11111111.22222222.png')],
      'dir.12345678/file.png': <Object?>[v('dir.12345678/file.png')],
      'dir/file.png': <Object?>[v('dir.12345678/file.png')],
      'multi.tar.gz': <Object?>[v('multi.12345678.tar.gz')],
      'multi.gz': <Object?>[v('multi.12345678.gz')],
    }, <String>[
      'assets/foo.png',
      'assets/sub dir/space image.png',
      'a/b.js.map',
      'a/c.wasm.map',
      'a/d.mjs.map',
      'a/e.map',
      'x',
      'upper.png',
      'short.png',
      'long.png',
      'nothex.png',
      'two.11111111.png',
      'two.png',
      'dir.12345678/file.png',
      'dir/file.png',
      'multi.tar.gz',
      'multi.gz',
    ]);

    await scene('malformed', <String, Object?>{
      'not a list': 'x',
      'null variant': <Object?>[null],
      'no asset': <Object?>[<String, Object?>{'dpr': 2.0}],
      'int dpr': <Object?>[v('a', 2)],
      'string dpr': <Object?>[v('a', '2.0')],
      'asset not a string': <Object?>[<String, Object?>{'asset': 1}],
      'null value': null,
      'ok': <Object?>[v('ok')],
    }, <String>['not a list', 'null variant', 'no asset', 'int dpr', 'string dpr', 'asset not a string', 'null value', 'ok']);

    await scene('not a map', <Object?>['a'], <String>[]);
    await scene('empty', <String, Object?>{}, <String>['a']);

    log.add('scene asset keys');
    final keys = <String>[];
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMessageHandler(
      'flutter/assets',
      (ByteData? message) async {
        keys.add(utf8.decode(message!.buffer.asUint8List(message.offsetInBytes, message.lengthInBytes)));
        return ByteData(1);
      },
    );
    for (final key in <String>[
      'assets/a.png',
      'assets/sub dir/space image.png',
      'ünïcødé/✓.png',
      'q?x=1#frag',
      'percent %41 %zz %',
      'brackets[]{}|^`"<>',
      r'back\slash',
      './dot/../a/./b',
      '../up',
      'a:b/c',
      'ab:c',
      '1a:b',
      '/abs/../x',
      'emoji 😀.png',
      "sub-delims !\$&'()*+,;=@~_.-",
      '',
    ]) {
      await rootBundle.load(key);
      log.add('key ${jsonEncode(key)} -> ${jsonEncode(keys.last)}');
    }

    File('build/asset_manifest.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'log': log}));
  });
}
