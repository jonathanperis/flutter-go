// Reference side of flutter-go's painting decorations pixel differential
// test: the painting library's gradients (createShader), box shadows
// (BoxShadow.toPaint, drawn as BoxDecoration draws them), paintImage and
// the Flutter logo, drawn into 64x64 pictures, rasterized with
// Picture.toImage and read back as rawRgba. Run with --enable-impeller
// (Impeller on Vulkan/SwiftShader in flutter_tester); without it the test
// writes nothing. Output: build/painting_decorations_pixels.json as
// {scene: base64 bytes}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

const int kSize = 64;
const Rect full = Rect.fromLTWH(0, 0, 64, 64);

Future<ui.Image> checkerImage() {
  final Uint8List px = Uint8List(16 * 16 * 4);
  for (int y = 0; y < 16; y++) {
    for (int x = 0; x < 16; x++) {
      final int i = (y * 16 + x) * 4;
      final bool on = ((x ~/ 4) + (y ~/ 4)).isEven;
      px[i] = on ? 255 : 30;
      px[i + 1] = on ? 120 : 200;
      px[i + 2] = (x * 16) & 255;
      px[i + 3] = 255;
    }
  }
  final Completer<ui.Image> c = Completer<ui.Image>();
  ui.decodeImageFromPixels(px, 16, 16, ui.PixelFormat.rgba8888, c.complete);
  return c.future;
}

void shaderRect(Canvas c, Gradient g, {Rect rect = full, TextDirection? textDirection}) =>
    c.drawRect(full, Paint()..shader = g.createShader(rect, textDirection: textDirection));

// BoxDecoration's shadow drawing (_paintShadows for a rectangle).
void shadow(Canvas c, BoxShadow s, Rect rect) {
  c.drawRect(const Rect.fromLTWH(0, 0, 64, 64), Paint()..color = const Color(0xFFFFFFFF));
  c.drawRect(rect.shift(s.offset).inflate(s.spreadRadius), s.toPaint());
}

Map<String, void Function(Canvas c)> scenes(ui.Image img) {
  const List<Color> rgb = <Color>[Color(0xFFFF0000), Color(0xFF00FF00), Color(0xFF0000FF)];
  return <String, void Function(Canvas c)>{
    'linear': (Canvas c) => shaderRect(c, const LinearGradient(colors: <Color>[Color(0xFFFF0000), Color(0xFF0000FF)])),
    'linear.rotated.mirror': (Canvas c) => shaderRect(
        c,
        const LinearGradient(
          begin: Alignment(-0.2, -0.4),
          end: Alignment(0.1, 0.0),
          colors: rgb,
          stops: <double>[0.1, 0.4, 0.8],
          tileMode: TileMode.mirror,
          transform: GradientRotation(0.6),
        )),
    'linear.directional.rtl': (Canvas c) => shaderRect(
        c,
        const LinearGradient(
          begin: AlignmentDirectional.topStart,
          end: AlignmentDirectional(0.5, 1.0),
          colors: <Color>[Color(0xFF00FFFF), Color(0x80FF00FF)],
        ),
        rect: const Rect.fromLTWH(4, 8, 50, 40),
        textDirection: TextDirection.rtl),
    'radial': (Canvas c) => shaderRect(c, const RadialGradient(colors: <Color>[Color(0xFFFFFF00), Color(0xFF8000FF)])),
    'radial.focal.decal': (Canvas c) => shaderRect(
        c,
        const RadialGradient(
          center: Alignment(0.2, 0.1),
          radius: 0.6,
          colors: rgb,
          tileMode: TileMode.decal,
          focal: Alignment(-0.3, -0.2),
          focalRadius: 0.05,
          transform: GradientRotation(-0.9),
        )),
    'radial.repeated': (Canvas c) => shaderRect(
        c,
        const RadialGradient(
          radius: 0.2,
          colors: <Color>[Color(0xFFFFFFFF), Color(0xFF004080)],
          stops: <double>[0.2, 0.9],
          tileMode: TileMode.repeated,
        )),
    'sweep': (Canvas c) => shaderRect(c, const SweepGradient(colors: rgb)),
    'sweep.angles.rotated': (Canvas c) => shaderRect(
        c,
        SweepGradient(
          center: const Alignment(-0.25, 0.3),
          startAngle: 0.4,
          endAngle: 4.2,
          colors: rgb,
          stops: const <double>[0.0, 0.3, 1.0],
          tileMode: TileMode.repeated,
          transform: const GradientRotation(math.pi / 5),
        )),
    'gradient.lerp': (Canvas c) => shaderRect(
        c,
        Gradient.lerp(
          const LinearGradient(colors: <Color>[Color(0xFFFF0000), Color(0xFF0000FF)], stops: <double>[0.2, 0.7]),
          const LinearGradient(
              begin: Alignment.topCenter, end: Alignment.bottomCenter, colors: rgb, stops: <double>[0.0, 0.5, 1.0]),
          0.4,
        )!),
    'shadow.normal': (Canvas c) =>
        shadow(c, const BoxShadow(color: Color(0x99000000), offset: Offset(3, 4), blurRadius: 6), const Rect.fromLTWH(16, 16, 28, 24)),
    'shadow.outer.spread': (Canvas c) => shadow(
        c,
        const BoxShadow(color: Color(0xFF3366CC), blurRadius: 4, spreadRadius: 3, blurStyle: BlurStyle.outer),
        const Rect.fromLTWH(18, 18, 28, 28)),
    'shadow.inner': (Canvas c) => shadow(
        c,
        const BoxShadow(color: Color(0xFFCC3300), offset: Offset(-2, 1), blurRadius: 5, blurStyle: BlurStyle.inner),
        const Rect.fromLTWH(10, 12, 40, 36)),
    'shadow.solid': (Canvas c) => shadow(
        c,
        const BoxShadow(color: Color(0x80008000), blurRadius: 3, spreadRadius: -2, blurStyle: BlurStyle.solid),
        const Rect.fromLTWH(14, 14, 36, 36)),
    'image.cover': (Canvas c) =>
        paintImage(canvas: c, rect: const Rect.fromLTWH(4, 8, 56, 40), image: img, fit: BoxFit.cover, filterQuality: FilterQuality.none),
    'image.contain.bottomRight': (Canvas c) => paintImage(
        canvas: c,
        rect: const Rect.fromLTWH(2, 2, 60, 40),
        image: img,
        fit: BoxFit.contain,
        alignment: Alignment.bottomRight,
        filterQuality: FilterQuality.none),
    'image.none.flip': (Canvas c) => paintImage(
        canvas: c,
        rect: const Rect.fromLTWH(8, 8, 48, 48),
        image: img,
        fit: BoxFit.none,
        alignment: Alignment.topLeft,
        flipHorizontally: true),
    'image.repeat': (Canvas c) => paintImage(
        canvas: c,
        rect: const Rect.fromLTWH(3, 5, 58, 50),
        image: img,
        scale: 1.5,
        repeat: ImageRepeat.repeat,
        alignment: const Alignment(0.3, -0.6),
        filterQuality: FilterQuality.none),
    'image.repeatX.fitHeight': (Canvas c) => paintImage(
        canvas: c,
        rect: const Rect.fromLTWH(0, 16, 64, 24),
        image: img,
        fit: BoxFit.fitHeight,
        repeat: ImageRepeat.repeatX,
        filterQuality: FilterQuality.none),
    'image.centerSlice': (Canvas c) => paintImage(
        canvas: c,
        rect: const Rect.fromLTWH(4, 4, 56, 50),
        image: img,
        centerSlice: const Rect.fromLTRB(4, 4, 12, 12),
        filterQuality: FilterQuality.none),
    'image.invert': (Canvas c) => paintImage(
        canvas: c,
        rect: const Rect.fromLTWH(8, 8, 48, 48),
        image: img,
        fit: BoxFit.fill,
        invertColors: true,
        filterQuality: FilterQuality.none),
    'image.opacity.colorFilter': (Canvas c) => paintImage(
        canvas: c,
        rect: const Rect.fromLTWH(8, 8, 48, 48),
        image: img,
        fit: BoxFit.fill,
        opacity: 0.6,
        colorFilter: const ColorFilter.mode(Color(0xFF2040A0), BlendMode.modulate),
        filterQuality: FilterQuality.none),
    'image.colorFilter.matrix.invert': (Canvas c) => paintImage(
        canvas: c,
        rect: const Rect.fromLTWH(8, 8, 48, 48),
        image: img,
        fit: BoxFit.fill,
        invertColors: true,
        colorFilter: const ColorFilter.matrix(<double>[0.5, 0, 0, 0, 20, 0, 1, 0, 0, 0, 0, 0, 0.25, 0, 40, 0, 0, 0, 1, 0]),
        filterQuality: FilterQuality.none),
    'logo.mark': (Canvas c) => const FlutterLogoDecoration().createBoxPainter().paint(
        c, Offset.zero, const ImageConfiguration(size: Size(64, 64))),
    'logo.mark.margin': (Canvas c) => const FlutterLogoDecoration(margin: EdgeInsets.fromLTRB(4, 10, 2, 6))
        .createBoxPainter()
        .paint(c, const Offset(3, 1), const ImageConfiguration(size: Size(58, 40))),
    'logo.mark.fading': (Canvas c) => FlutterLogoDecoration.lerp(null, const FlutterLogoDecoration(), 0.5)!
        .createBoxPainter()
        .paint(c, Offset.zero, const ImageConfiguration(size: Size(64, 64))),
  };
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('painting_decorations_pixels', () async {
    if (Platform.environment['FLUTTER_TEST_IMPELLER'] != 'true') {
      return; // the reference is Impeller's; see the header
    }
    debugDisableShadows = false;
    final ui.Image img = await checkerImage();
    final Map<String, Object> out = <String, Object>{};
    for (final MapEntry<String, void Function(Canvas)> e in scenes(img).entries) {
      final ui.PictureRecorder rec = ui.PictureRecorder();
      e.value(Canvas(rec));
      final ui.Image image = await rec.endRecording().toImage(kSize, kSize);
      final ByteData bytes = (await image.toByteData())!;
      out[e.key] = base64Encode(bytes.buffer.asUint8List());
    }
    debugDisableShadows = true;
    File('build/painting_decorations_pixels.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
