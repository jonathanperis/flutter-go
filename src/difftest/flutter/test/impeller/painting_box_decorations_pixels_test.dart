// Reference side of flutter-go's BoxDecoration/ShapeDecoration pixel
// differential test: each decoration painted with createBoxPainter().paint
// into a 64×64 picture, rasterized with Picture.toImage and read back as
// rawRgba. Run with --enable-impeller (Impeller on Vulkan/SwiftShader);
// without it the test writes nothing (see painting_pixels_test.dart).
// Output: build/painting_box_decorations_pixels.json as {scene: base64}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

const int kSize = 64;

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
  final c = Completer<ui.Image>();
  ui.decodeImageFromPixels(px, 16, 16, ui.PixelFormat.rgba8888, c.complete);
  return c.future;
}

class Img extends ImageProvider<Img> {
  Img(this.image);
  final ui.Image image;

  @override
  Future<Img> obtainKey(ImageConfiguration configuration) => SynchronousFuture<Img>(this);

  @override
  ImageStreamCompleter loadImage(Img key, ImageDecoderCallback decode) =>
      OneFrameImageStreamCompleter(SynchronousFuture<ImageInfo>(ImageInfo(image: image.clone())));
}

BorderSide bs(int color, double width, [double strokeAlign = BorderSide.strokeAlignInside]) =>
    BorderSide(color: Color(color), width: width, strokeAlign: strokeAlign);

Map<String, (Decoration, TextDirection)> scenes(ui.Image img) {
  const ltr = TextDirection.ltr;
  const lin = LinearGradient(
    begin: AlignmentDirectional.topStart,
    end: Alignment.bottomRight,
    colors: <Color>[Color(0xFFE53935), Color(0xFF43A047), Color(0xFF1E88E5)],
    stops: <double>[0, 0.4, 1],
  );
  const rad = RadialGradient(center: Alignment(0.2, -0.3), radius: 0.7, colors: <Color>[Color(0xFFFFEB3B), Color(0xFF8E24AA)]);
  const swp = SweepGradient(colors: <Color>[Color(0xFF00BCD4), Color(0xFFE91E63), Color(0xFF00BCD4)], transform: GradientRotation(0.5));
  const shadows = <BoxShadow>[
    BoxShadow(color: Color(0x80000000), offset: Offset(2, 3), blurRadius: 4, spreadRadius: 1),
    BoxShadow(color: Color(0x60E53935), offset: Offset(-2, -1), blurRadius: 3),
  ];
  return <String, (Decoration, TextDirection)>{
    'box.color.border.radius': (
      BoxDecoration(color: const Color(0xFF43A047), border: Border.all(color: const Color(0xFF1E88E5), width: 4),
          borderRadius: const BorderRadius.all(Radius.circular(10))),
      ltr
    ),
    'box.circle': (
      BoxDecoration(color: const Color(0xFFE53935), shape: BoxShape.circle, border: Border.all(width: 3)),
      ltr
    ),
    'box.gradient.linear': (const BoxDecoration(gradient: lin), ltr),
    'box.gradient.linear.rtl': (const BoxDecoration(gradient: lin, borderRadius: BorderRadiusDirectional.only(topStart: Radius.circular(16))), TextDirection.rtl),
    'box.gradient.radial.circle': (const BoxDecoration(gradient: rad, shape: BoxShape.circle), ltr),
    'box.gradient.sweep': (const BoxDecoration(gradient: swp, borderRadius: BorderRadius.all(Radius.circular(8))), ltr),
    'box.shadows': (const BoxDecoration(color: Color(0xFFFFFFFF), boxShadow: shadows), ltr),
    'box.shadow.spread': (
      const BoxDecoration(color: Color(0xFF1E88E5), borderRadius: BorderRadius.all(Radius.circular(6)),
          boxShadow: <BoxShadow>[BoxShadow(color: Color(0xFF000000), blurRadius: 5, spreadRadius: 4)]),
      ltr
    ),
    'box.shadow.solid': (
      const BoxDecoration(color: Color(0xFF8E24AA), boxShadow: <BoxShadow>[
        BoxShadow(color: Color(0xA0000000), offset: Offset(3, 3), blurRadius: 4, blurStyle: BlurStyle.solid),
      ]),
      ltr
    ),
    'box.shadow.outer.circle': (
      const BoxDecoration(color: Color(0x4000BCD4), shape: BoxShape.circle, boxShadow: <BoxShadow>[
        BoxShadow(color: Color(0xFF000000), blurRadius: 6, blurStyle: BlurStyle.outer),
      ]),
      ltr
    ),
    'box.shadow.inner': (
      const BoxDecoration(color: Color(0xFFFFEB3B), boxShadow: <BoxShadow>[
        BoxShadow(color: Color(0xFF000000), blurRadius: 4, spreadRadius: -3, blurStyle: BlurStyle.inner),
      ]),
      ltr
    ),
    'box.blendMode': (const BoxDecoration(color: Color(0xFFFFCC00), backgroundBlendMode: BlendMode.difference), ltr),
    'box.border.nonuniform': (
      BoxDecoration(color: const Color(0xFFFFEB3B), border: Border(top: bs(0xFFE53935, 6), left: bs(0xFF1E88E5, 3), bottom: bs(0x80000000, 2))),
      ltr
    ),
    'box.border.directional.rtl': (
      BoxDecoration(color: const Color(0xFF00BCD4), border: BorderDirectional(start: bs(0xFF000000, 5), end: bs(0xFFE53935, 2))),
      TextDirection.rtl
    ),
    'box.image': (BoxDecoration(image: DecorationImage(image: Img(img), fit: BoxFit.cover)), ltr),
    'box.image.circle': (
      BoxDecoration(color: const Color(0xFF000000), shape: BoxShape.circle, image: DecorationImage(image: Img(img), fit: BoxFit.fill)),
      ltr
    ),
    'box.image.radius.border': (
      BoxDecoration(
        image: DecorationImage(image: Img(img), repeat: ImageRepeat.repeat, alignment: Alignment.topLeft),
        borderRadius: const BorderRadius.all(Radius.circular(12)),
        border: Border.all(color: const Color(0xFF43A047), width: 3),
      ),
      ltr
    ),
    'shape.star.gradient.shadows': (
      ShapeDecoration(gradient: lin, shadows: shadows, shape: StarBorder(side: bs(0xFF000000, 2), innerRadiusRatio: 0.5)),
      ltr
    ),
    'shape.stadium.color.shadows': (
      ShapeDecoration(color: const Color(0xFF1E88E5), shadows: shadows, shape: StadiumBorder(side: bs(0xFFFFFFFF, 3))),
      ltr
    ),
    'shape.rrect.gradient': (
      ShapeDecoration(gradient: rad, shape: RoundedRectangleBorder(side: bs(0xFF8E24AA, 4), borderRadius: const BorderRadius.all(Radius.circular(14)))),
      ltr
    ),
    'shape.superellipse.shadows': (
      ShapeDecoration(color: const Color(0xFF43A047), shadows: shadows,
          shape: RoundedSuperellipseBorder(side: bs(0xFF000000, 2), borderRadius: const BorderRadius.all(Radius.circular(18)))),
      ltr
    ),
    'shape.circle.image': (
      ShapeDecoration(image: DecorationImage(image: Img(img), fit: BoxFit.cover), shape: CircleBorder(side: bs(0xFFE53935, 3))),
      ltr
    ),
    'shape.border.shadows': (
      ShapeDecoration(color: const Color(0xFFFFEB3B), shadows: shadows, shape: Border.all(color: const Color(0xFF1E88E5), width: 4)),
      ltr
    ),
    'shape.beveled.sweep': (
      const ShapeDecoration(gradient: swp, shape: BeveledRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(12)))),
      ltr
    ),
    'shape.oval.translucent': (
      ShapeDecoration(color: const Color(0x8000BCD4), shape: OvalBorder(side: bs(0xC0000000, 4, BorderSide.strokeAlignCenter))),
      ltr
    ),
    'lerp.box.shape': (
      Decoration.lerp(
        const BoxDecoration(color: Color(0xFFE53935), borderRadius: BorderRadius.all(Radius.circular(4))),
        ShapeDecoration(color: const Color(0xFF1E88E5), shape: CircleBorder(side: bs(0xFF000000, 2))),
        0.4,
      )!,
      ltr
    ),
  };
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test('painting_box_decorations_pixels', () async {
    if (Platform.environment['FLUTTER_TEST_IMPELLER'] != 'true') {
      return; // the reference is Impeller's; see the header
    }
    // The test binding disables shadows; paint them, as an app does.
    debugDisableShadows = false;
    final ui.Image img = await checkerImage();
    final out = <String, Object>{};
    for (final MapEntry<String, (Decoration, TextDirection)> e in scenes(img).entries) {
      final rec = ui.PictureRecorder();
      final BoxPainter painter = e.value.$1.createBoxPainter(() {});
      painter.paint(Canvas(rec), const Offset(10, 12),
          ImageConfiguration(size: const Size(44, 38), textDirection: e.value.$2));
      painter.dispose();
      final ui.Image image = await rec.endRecording().toImage(kSize, kSize);
      final ByteData bytes = (await image.toByteData())!;
      out[e.key] = base64Encode(bytes.buffer.asUint8List());
    }
    debugDisableShadows = true;
    File('build/painting_box_decorations_pixels.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
