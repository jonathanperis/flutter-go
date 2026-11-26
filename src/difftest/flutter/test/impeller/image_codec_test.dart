// Reference side of flutter-go's image codec differential test (dart:ui's
// painting.dart images and codecs): decodes the engine's image fixtures
// (and generated images) through instantiateImageCodec, ImageDescriptor and
// decodeImageFromPixels, with and without target sizes, and records frame
// counts, repetition counts, durations, sizes, color spaces, errors and the
// pixels toByteData returns in every format. Output: build/image_codec.json
// (only when run on Impeller: flutter test --enable-impeller; the Skia
// backend decodes differently and fails some cases) as
// {case: {field: value}}. Pixel buffers up to 1.1 MB are stored in base64,
// larger ones as an FNV-1a hash.
//
// The fixtures are read from the Flutter clone at
// ~/Projetos/jonathanperis/flutter (engine/src/flutter/lib/ui/fixtures and
// engine/src/flutter/testing/resources).
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter_test/flutter_test.dart';

final String flutterRoot = '${Platform.environment['HOME']}/Projetos/jonathanperis/flutter';
final String fixtures = '$flutterRoot/engine/src/flutter/lib/ui/fixtures';
final String resources = '$flutterRoot/engine/src/flutter/testing/resources';

String fnv(Uint8List b) {
  // FNV-1a, 64-bit, wrapping as Dart ints do.
  var h = 0xcbf29ce484222325;
  for (final int x in b) {
    h ^= x;
    h *= 0x100000001b3;
  }
  return h.toUnsigned(64).toRadixString(16);
}

Object bytes(Uint8List b) => b.length <= 1100000 ? base64.encode(b) : 'fnv:${fnv(b)}';

String err(Object e) => e.toString();

Future<Map<String, Object?>> imageInfo(ui.Image image) async {
  final out = <String, Object?>{
    'w': image.width,
    'h': image.height,
    'str': image.toString(),
  };
  try {
    out['colorSpace'] = image.colorSpace.name;
  } catch (e) {
    out['colorSpace'] = err(e);
  }
  for (final ui.ImageByteFormat f in ui.ImageByteFormat.values) {
    try {
      final ByteData? d = await image.toByteData(format: f);
      if (d == null) {
        out[f.name] = null;
      } else if (f == ui.ImageByteFormat.png) {
        final Uint8List png = d.buffer.asUint8List(d.offsetInBytes, d.lengthInBytes);
        out['pngLen'] = png.length;
        out['pngHash'] = fnv(png);
        // Decode the PNG back: its pixels are what a lossless encoder must keep.
        final ui.Codec c = await ui.instantiateImageCodec(png);
        final ui.FrameInfo fi = await c.getNextFrame();
        final ByteData? back = await fi.image.toByteData(
          format: ui.ImageByteFormat.rawStraightRgba,
        );
        out['pngBack'] = bytes(back!.buffer.asUint8List(back.offsetInBytes, back.lengthInBytes));
        out['pngBackSize'] = '${fi.image.width}x${fi.image.height}';
        fi.image.dispose();
        c.dispose();
      } else {
        out[f.name] = bytes(d.buffer.asUint8List(d.offsetInBytes, d.lengthInBytes));
      }
    } catch (e) {
      out[f.name] = 'error: ${err(e)}';
    }
  }
  return out;
}

Future<Map<String, Object?>> codecCase(
  Future<ui.Codec> Function() make, {
  int maxFrames = 8,
  bool pixelsEveryFrame = true,
}) async {
  final out = <String, Object?>{};
  final ui.Codec codec;
  try {
    codec = await make();
  } catch (e) {
    out['error'] = err(e);
    return out;
  }
  out['strBefore'] = codec.toString();
  out['frameCount'] = codec.frameCount;
  out['repetitionCount'] = codec.repetitionCount;
  out['strAfter'] = codec.toString();
  final frames = <Object?>[];
  final int n = codec.frameCount + 1 < maxFrames ? codec.frameCount + 1 : maxFrames;
  for (var i = 0; i < n; i++) {
    try {
      final ui.FrameInfo fi = await codec.getNextFrame();
      final Map<String, Object?> f;
      if (pixelsEveryFrame || i == n - 1) {
        f = await imageInfo(fi.image);
      } else {
        final ByteData? d = await fi.image.toByteData();
        f = <String, Object?>{
          'w': fi.image.width,
          'h': fi.image.height,
          'rawRgba': 'fnv:${fnv(d!.buffer.asUint8List(d.offsetInBytes, d.lengthInBytes))}',
        };
      }
      f['duration'] = fi.duration.inMilliseconds;
      frames.add(f);
      fi.image.dispose();
    } catch (e) {
      frames.add(<String, Object?>{'error': err(e)});
    }
  }
  out['frames'] = frames;
  codec.dispose();
  return out;
}

Uint8List fixture(String name) => File('$fixtures/$name').readAsBytesSync();
Uint8List resource(String name) => File('$resources/$name').readAsBytesSync();

Future<ui.Image> fromPixels(
  Uint8List pixels,
  int w,
  int h,
  ui.PixelFormat format, {
  int? rowBytes,
  int? targetWidth,
  int? targetHeight,
  bool allowUpscaling = true,
  ui.TargetPixelFormat targetFormat = ui.TargetPixelFormat.dontCare,
}) {
  final completer = Completer<ui.Image>();
  ui.decodeImageFromPixels(
    pixels,
    w,
    h,
    format,
    completer.complete,
    rowBytes: rowBytes,
    targetWidth: targetWidth,
    targetHeight: targetHeight,
    allowUpscaling: allowUpscaling,
    targetFormat: targetFormat,
  );
  return completer.future;
}

// A 5x3 premultiplied pattern with every kind of alpha.
Uint8List pattern(int w, int h) {
  final b = Uint8List(w * h * 4);
  for (var y = 0; y < h; y++) {
    for (var x = 0; x < w; x++) {
      final int i = (y * w + x) * 4;
      final int a = (x * 61 + y * 97) % 256;
      b[i + 0] = (x * 37 + y * 11) % (a + 1);
      b[i + 1] = (x * 13 + y * 71) % (a + 1);
      b[i + 2] = (x * 7 + y * 29) % (a + 1);
      b[i + 3] = a;
    }
  }
  return b;
}

Future<void> main() async {
  test('image_codec', () async {
    if (Platform.environment['FLUTTER_TEST_IMPELLER'] != 'true' &&
        !Platform.executableArguments.contains('--enable-impeller')) {
      return; // The reference is Impeller's (flutter test --enable-impeller).
    }
    final out = <String, Object?>{};

    // Encoded fixtures, intrinsic size.
    final encoded = <String, Uint8List Function()>{
      '2x2.png': () => resource('2x2.png'),
      'square.png': () => resource('square.png'),
      'transparent_image.png': () => resource('transparent_image.png'),
      'unmultiplied_alpha.png': () => fixture('unmultiplied_alpha.png'),
      'DisplayP3Logo.png': () => fixture('DisplayP3Logo.png'),
      'WideGamutIndexed.png': () => fixture('WideGamutIndexed.png'),
      'Horizontal.png': () => fixture('Horizontal.png'),
      'four_frame_with_reuse_end.png': () => fixture('four_frame_with_reuse_end.png'),
      'DisplayP3Logo.jpg': () => fixture('DisplayP3Logo.jpg'),
      'Horizontal.jpg': () => fixture('Horizontal.jpg'),
      'four_frame_with_reuse.gif': () => fixture('four_frame_with_reuse.gif'),
      'hello_loop_2.gif': () => fixture('hello_loop_2.gif'),
      'opaque_first_frame_then_alpha.gif': () => fixture('opaque_first_frame_then_alpha.gif'),
      'opaque_first_frame_then_alpha.webp': () => fixture('opaque_first_frame_then_alpha.webp'),
      'hello_loop_2.webp': () => fixture('hello_loop_2.webp'),
      'alpha_animated.apng': () => fixture('alpha_animated.apng'),
      'dispose_op_background.apng': () => fixture('dispose_op_background.apng'),
      'out_of_bounds.apng': () => fixture('out_of_bounds.apng'),
      'out_of_bounds_wrapping.apng': () => fixture('out_of_bounds_wrapping.apng'),
    };
    for (final MapEntry<String, Uint8List Function()> e in encoded.entries) {
      out['encoded ${e.key}'] = await codecCase(() => ui.instantiateImageCodec(e.value()));
    }
    out['encoded heart.webp'] = await codecCase(
      () => ui.instantiateImageCodec(fixture('heart.webp')),
      maxFrames: 70,
      pixelsEveryFrame: false,
    );
    out['encoded 2_dispose_op_restore_previous.apng'] = await codecCase(
      () => ui.instantiateImageCodec(fixture('2_dispose_op_restore_previous.apng')),
      maxFrames: 71,
      pixelsEveryFrame: false,
    );
    out['encoded DashInNooglerHat.jpg 150'] = await codecCase(
      () => ui.instantiateImageCodec(fixture('DashInNooglerHat.jpg'), targetWidth: 150),
    );

    // Truncated and invalid data.
    out['invalid 1 2 3'] = await codecCase(
      () => ui.instantiateImageCodec(Uint8List.fromList(<int>[1, 2, 3])),
    );
    out['invalid empty'] = await codecCase(() => ui.instantiateImageCodec(Uint8List(0)));
    for (final int n in <int>[20, 40, 100, 300, 700, 1200]) {
      out['truncated four_frame_with_reuse.gif $n'] = await codecCase(
        () => ui.instantiateImageCodec(
          Uint8List.sublistView(fixture('four_frame_with_reuse.gif'), 0, n),
        ),
      );
    }
    for (final int n in <int>[30, 60, 200, 1000]) {
      out['truncated Horizontal.png $n'] = await codecCase(
        () => ui.instantiateImageCodec(Uint8List.sublistView(fixture('Horizontal.png'), 0, n)),
      );
    }
    for (final int n in <int>[300, 3000]) {
      out['truncated DisplayP3Logo.jpg $n'] = await codecCase(
        () => ui.instantiateImageCodec(Uint8List.sublistView(fixture('DisplayP3Logo.jpg'), 0, n)),
      );
    }

    // Target sizes.
    final sizes = <String, List<Object?>>{
      '2x2.png h1': <Object?>['2x2.png', null, 1, true],
      '2x2.png w1': <Object?>['2x2.png', 1, null, true],
      '2x2.png w10': <Object?>['2x2.png', 10, null, true],
      '2x2.png w10 noup': <Object?>['2x2.png', 10, null, false],
      '2x2.png w10 h1': <Object?>['2x2.png', 10, 1, true],
      '2x2.png w10 h1 noup': <Object?>['2x2.png', 10, 1, false],
      '2x2.png w0 h-3': <Object?>['2x2.png', 0, -3, true],
      'transparent_image.png w8 h8': <Object?>['transparent_image.png', 8, 8, true],
      'transparent_image.png w3 h2': <Object?>['transparent_image.png', 3, 2, true],
      'transparent_image.png w7': <Object?>['transparent_image.png', 7, null, true],
      'Horizontal.png w150': <Object?>['Horizontal.png', 150, null, true],
      'Horizontal.png h37': <Object?>['Horizontal.png', null, 37, true],
      'Horizontal.png w97 h41': <Object?>['Horizontal.png', 97, 41, true],
      'Horizontal.png w450': <Object?>['Horizontal.png', 450, null, true],
      'DisplayP3Logo.png w33': <Object?>['DisplayP3Logo.png', 33, null, true],
      'Horizontal.jpg w50': <Object?>['Horizontal.jpg', 50, null, true],
      'Horizontal.jpg h300': <Object?>['Horizontal.jpg', null, 300, true],
      'Horizontal.jpg w130': <Object?>['Horizontal.jpg', 130, null, true],
      'Horizontal.jpg w300 h10': <Object?>['Horizontal.jpg', 300, 10, true],
      'four_frame_with_reuse.gif w10': <Object?>['four_frame_with_reuse.gif', 10, null, true],
      'alpha_animated.apng w10': <Object?>['alpha_animated.apng', 10, null, true],
    };
    for (final MapEntry<String, List<Object?>> e in sizes.entries) {
      final String file = e.value[0]! as String;
      final Uint8List data = file == '2x2.png' || file == 'transparent_image.png'
          ? resource(file)
          : fixture(file);
      out['size ${e.key}'] = await codecCase(
        () => ui.instantiateImageCodec(
          data,
          targetWidth: e.value[1] as int?,
          targetHeight: e.value[2] as int?,
          allowUpscaling: e.value[3]! as bool,
        ),
        maxFrames: 2,
      );
    }

    // instantiateImageCodecWithSize and ImageDescriptor directly.
    out['withSize Horizontal.png half'] = await codecCase(() async {
      final ui.ImmutableBuffer buffer = await ui.ImmutableBuffer.fromUint8List(
        fixture('Horizontal.png'),
      );
      return ui.instantiateImageCodecWithSize(
        buffer,
        getTargetSize: (int w, int h) => ui.TargetImageSize(width: w ~/ 2, height: h ~/ 2),
      );
    }, maxFrames: 1);
    out['withSize Horizontal.png default'] = await codecCase(() async {
      final ui.ImmutableBuffer buffer = await ui.ImmutableBuffer.fromUint8List(
        fixture('Horizontal.png'),
      );
      return ui.instantiateImageCodecWithSize(buffer);
    }, maxFrames: 1);
    for (final List<int?> wh in <List<int?>>[
      <int?>[7, null],
      <int?>[null, 7],
      <int?>[301, null],
      <int?>[null, 101],
      <int?>[-1, 5],
    ]) {
      final log = <String, Object?>{};
      final ui.ImmutableBuffer buffer = await ui.ImmutableBuffer.fromUint8List(
        fixture('Horizontal.png'),
      );
      log['bufferLength'] = buffer.length;
      final ui.ImageDescriptor d = await ui.ImageDescriptor.encoded(buffer);
      log['strBefore'] = d.toString();
      log['size'] = '${d.width}x${d.height}x${d.bytesPerPixel}';
      log['strAfter'] = d.toString();
      log['codec'] = await codecCase(
        () => d.instantiateCodec(targetWidth: wh[0], targetHeight: wh[1]),
        maxFrames: 1,
      );
      d.dispose();
      buffer.dispose();
      out['descriptor Horizontal.png ${wh[0]} ${wh[1]}'] = log;
    }
    for (final String f in <String>['2x2.png', 'square.png', 'transparent_image.png']) {
      final ui.ImmutableBuffer buffer = await ui.ImmutableBuffer.fromUint8List(resource(f));
      final ui.ImageDescriptor d = await ui.ImageDescriptor.encoded(buffer);
      out['descriptor $f'] = '${d.width}x${d.height}x${d.bytesPerPixel} ${d}';
    }
    for (final String f in <String>[
      'WideGamutIndexed.png',
      'DisplayP3Logo.png',
      'DisplayP3Logo.jpg',
      'Horizontal.jpg',
      'four_frame_with_reuse.gif',
      'hello_loop_2.webp',
      'alpha_animated.apng',
      'Horizontal.png',
    ]) {
      final ui.ImmutableBuffer buffer = await ui.ImmutableBuffer.fromUint8List(fixture(f));
      final ui.ImageDescriptor d = await ui.ImageDescriptor.encoded(buffer);
      out['descriptor $f'] = '${d.width}x${d.height}x${d.bytesPerPixel} ${d}';
    }
    {
      final ui.ImmutableBuffer buffer = await ui.ImmutableBuffer.fromUint8List(Uint8List(64));
      final d = ui.ImageDescriptor.raw(
        buffer,
        width: 4,
        height: 4,
        pixelFormat: ui.PixelFormat.rgbaFloat32,
      );
      out['descriptor raw float'] = '${d} ${d.width}x${d.height}x${d.bytesPerPixel}';
    }
    try {
      final ui.ImmutableBuffer buffer = await ui.ImmutableBuffer.fromUint8List(
        Uint8List.fromList(<int>[1, 2, 3]),
      );
      await ui.ImageDescriptor.encoded(buffer);
      out['descriptor invalid'] = 'ok';
    } catch (e) {
      out['descriptor invalid'] = err(e);
    }
    try {
      await ui.ImmutableBuffer.fromFilePath('/nonexistent/file.png');
      out['buffer missing file'] = 'ok';
    } catch (e) {
      out['buffer missing file'] = err(e);
    }
    {
      final ui.ImmutableBuffer b = await ui.ImmutableBuffer.fromFilePath(
        '$fixtures/four_frame_with_reuse.gif',
      );
      out['buffer file'] = '${b.length} ${b.debugDisposed}';
      b.dispose();
      out['buffer file disposed'] = '${b.debugDisposed}';
    }
    out['TargetImageSize'] = <String>[
      const ui.TargetImageSize().toString(),
      const ui.TargetImageSize(width: 3).toString(),
      const ui.TargetImageSize(width: 3, height: 4).toString(),
    ];

    // decodeImageFromPixels.
    final Uint8List pat = pattern(5, 3);
    out['pixels rgba8888'] = await imageInfo(await fromPixels(pat, 5, 3, ui.PixelFormat.rgba8888));
    out['pixels bgra8888'] = await imageInfo(await fromPixels(pat, 5, 3, ui.PixelFormat.bgra8888));
    {
      final padded = Uint8List(3 * 24);
      for (var y = 0; y < 3; y++) {
        padded.setRange(y * 24, y * 24 + 20, pat, y * 20);
        padded[y * 24 + 20] = 0xAB;
      }
      out['pixels rowBytes'] = await imageInfo(
        await fromPixels(padded, 5, 3, ui.PixelFormat.rgba8888, rowBytes: 24),
      );
    }
    final Uint8List big = pattern(16, 12);
    for (final List<Object?> s in <List<Object?>>[
      <Object?>[8, 6, true],
      <Object?>[5, null, true],
      <Object?>[null, 5, true],
      <Object?>[40, 30, true],
      <Object?>[23, 7, true],
      <Object?>[40, null, false],
      <Object?>[-100, -99999, false],
    ]) {
      try {
        out['pixels resize ${s[0]} ${s[1]} ${s[2]}'] = await imageInfo(
          await fromPixels(
            big,
            16,
            12,
            ui.PixelFormat.rgba8888,
            targetWidth: s[0] as int?,
            targetHeight: s[1] as int?,
            allowUpscaling: s[2]! as bool,
          ),
        );
      } catch (e) {
        out['pixels resize ${s[0]} ${s[1]} ${s[2]}'] = err(e);
      }
    }
    final floats = Float32List.fromList(<double>[
      1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 1, 1, 1, 1, 0, //
      0.5, 0.25, 0.125, 0.5, 1.5, -0.25, 0.3, 0.7, 0.1, 0.2, 0.3, 0.4, 0.999, 0.501, 0.0039, 1,
    ]);
    final Uint8List floatBytes = floats.buffer.asUint8List();
    out['pixels rgbaFloat32'] = await imageInfo(
      await fromPixels(floatBytes, 4, 2, ui.PixelFormat.rgbaFloat32),
    );
    out['pixels rgbaFloat32 to rgbaFloat32'] = await imageInfo(
      await fromPixels(
        floatBytes,
        4,
        2,
        ui.PixelFormat.rgbaFloat32,
        targetFormat: ui.TargetPixelFormat.rgbaFloat32,
      ),
    );
    out['pixels rgba8888 to rgbaFloat32'] = await imageInfo(
      await fromPixels(pat, 5, 3, ui.PixelFormat.rgba8888, targetFormat: ui.TargetPixelFormat.rgbaFloat32),
    );
    out['pixels rFloat32 to rFloat32'] = await imageInfo(
      await fromPixels(
        Float32List.fromList(<double>[0, 0.25, 0.5, 1, 2, -1]).buffer.asUint8List(),
        3,
        2,
        ui.PixelFormat.rFloat32,
        targetFormat: ui.TargetPixelFormat.rFloat32,
      ),
    );

    // decodeImageFromPixelsSync.
    for (final ui.PixelFormat f in <ui.PixelFormat>[
      ui.PixelFormat.rgba8888,
      ui.PixelFormat.bgra8888,
      ui.PixelFormat.rgbaFloat32,
      ui.PixelFormat.rFloat32,
    ]) {
      try {
        final ui.Image image = ui.decodeImageFromPixelsSync(
          f == ui.PixelFormat.rgbaFloat32 || f == ui.PixelFormat.rFloat32 ? floatBytes : pat,
          f == ui.PixelFormat.rgbaFloat32 ? 4 : 5,
          f == ui.PixelFormat.rgbaFloat32 ? 2 : 3,
          f,
        );
        out['sync ${f.name}'] = await imageInfo(image);
      } catch (e) {
        out['sync ${f.name}'] = 'threw ${e.runtimeType}: $e';
      }
    }
    try {
      ui.decodeImageFromPixelsSync(Uint8List(4), 0, 1, ui.PixelFormat.rgba8888);
      out['sync zero'] = 'ok';
    } catch (e) {
      out['sync zero'] = 'threw ${e.runtimeType}: $e';
    }

    // Image handles.
    {
      final log = <String>[];
      final ui.Image a = await fromPixels(pat, 5, 3, ui.PixelFormat.rgba8888);
      final ui.Image b = a.clone();
      final ui.Image c = b.clone();
      log.add('${a.isCloneOf(b)} ${b.isCloneOf(c)} ${a == b} ${a.debugDisposed}');
      log.add('stacks ${a.debugGetOpenHandleStackTraces()!.length}');
      a.dispose();
      log.add('${a.debugDisposed} ${b.debugDisposed} ${a.isCloneOf(c)}');
      log.add('stacks ${b.debugGetOpenHandleStackTraces()!.length}');
      try {
        a.clone();
      } catch (e) {
        log.add('clone disposed: ${e.runtimeType} ${(e as StateError).message}');
      }
      final ByteData? d = await c.toByteData();
      log.add('after dispose of a: ${d!.lengthInBytes}');
      b.dispose();
      c.dispose();
      log.add('stacks ${a.debugGetOpenHandleStackTraces()!.length}');
      out['handles'] = log;
    }
    {
      final log = <String>[];
      final ui.Codec codec = await ui.instantiateImageCodec(resource('2x2.png'));
      final ui.FrameInfo f1 = await codec.getNextFrame();
      final ui.FrameInfo f2 = await codec.getNextFrame();
      log.add('same ${f1.image.isCloneOf(f2.image)}');
      f1.image.dispose();
      final ui.FrameInfo f3 = await codec.getNextFrame();
      log.add('after one dispose ${f3.image.isCloneOf(f2.image)}');
      f2.image.dispose();
      f3.image.dispose();
      try {
        await codec.getNextFrame();
        log.add('ok');
      } catch (e) {
        log.add(err(e));
      }
      out['single frame cache'] = log;
    }
    File('build/image_codec.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  }, timeout: const Timeout(Duration(minutes: 5)));
}
