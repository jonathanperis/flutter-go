// Reference side of flutter-go's dart:ui pixel differential test: scenes
// drawn with Canvas, rasterized with Picture.toImage and read back with
// Image.toByteData (rawRgba, premultiplied). Run with --enable-impeller
// (Impeller on Vulkan/SwiftShader in flutter_tester); without it the test
// writes nothing, so the Skia pass of `flutter test` keeps the Impeller
// output. Output: build/painting_pixels.json as {scene: base64 bytes}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

const int kSize = 64;

Future<Image> checkerImage() {
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
  final Completer<Image> c = Completer<Image>();
  decodeImageFromPixels(px, 16, 16, PixelFormat.rgba8888, c.complete);
  return c.future;
}

Map<String, void Function(Canvas c)> scenes(Image img) {
  Paint p(int color) => Paint()..color = Color(color);
  return <String, void Function(Canvas c)>{
    'rect': (Canvas c) => c.drawRect(const Rect.fromLTRB(8, 8, 40, 30), p(0xFF3366CC)),
    'rect.fractional': (Canvas c) => c.drawRect(const Rect.fromLTRB(8.3, 8.6, 40.2, 30.7), p(0xFFCC6633)),
    'rrect': (Canvas c) => c.drawRRect(RRect.fromLTRBR(6, 10, 58, 50, const Radius.circular(12)), p(0xFF22AA44)),
    'oval': (Canvas c) => c.drawOval(const Rect.fromLTRB(4, 12, 60, 52), p(0xFFAA2244)),
    'circle': (Canvas c) => c.drawCircle(const Offset(32, 32), 20, p(0xFF4422AA)),
    'drrect': (Canvas c) => c.drawDRRect(RRect.fromLTRBR(4, 4, 60, 60, const Radius.circular(10)),
        RRect.fromLTRBR(16, 16, 48, 48, const Radius.circular(6)), p(0xFF119988)),
    'stroke.rect': (Canvas c) => c.drawRect(const Rect.fromLTRB(10, 10, 54, 54), p(0xFF000000)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 4),
    'stroke.line.round': (Canvas c) => c.drawLine(const Offset(8, 50), const Offset(56, 12), p(0xFFEE5500)
      ..strokeWidth = 6
      ..strokeCap = StrokeCap.round),
    'paint': (Canvas c) => c.drawPaint(p(0x80FF0000)),
    'color.blend': (Canvas c) {
      c.drawColor(const Color(0xFF2040A0), BlendMode.src);
      c.drawRect(const Rect.fromLTRB(10, 10, 54, 54), p(0xFFFFCC00)..blendMode = BlendMode.multiply);
    },
    'alpha.srcOver': (Canvas c) {
      c.drawRect(const Rect.fromLTRB(4, 4, 40, 40), p(0xFF0000FF));
      c.drawRect(const Rect.fromLTRB(20, 20, 60, 60), p(0x8000FF00));
    },
    'gradient.linear': (Canvas c) => c.drawRect(const Rect.fromLTWH(0, 0, 64, 64), Paint()
      ..shader = Gradient.linear(const Offset(4, 4), const Offset(60, 40),
          const <Color>[Color(0xFFFF0000), Color(0xFF0000FF)])),
    'gradient.linear.stops.repeat': (Canvas c) => c.drawRect(const Rect.fromLTWH(0, 0, 64, 64), Paint()
      ..shader = Gradient.linear(const Offset(20, 0), const Offset(40, 0),
          const <Color>[Color(0xFF00FF00), Color(0xFFFFFFFF), Color(0xFF000000)], <double>[0, 0.3, 1], TileMode.repeated)),
    'gradient.radial': (Canvas c) => c.drawRect(const Rect.fromLTWH(0, 0, 64, 64), Paint()
      ..shader = Gradient.radial(const Offset(32, 32), 28, const <Color>[Color(0xFFFFFF00), Color(0xFF8000FF)])),
    'gradient.sweep': (Canvas c) => c.drawRect(const Rect.fromLTWH(0, 0, 64, 64), Paint()
      ..shader = Gradient.sweep(const Offset(32, 32), const <Color>[Color(0xFFFF0000), Color(0xFF00FF00), Color(0xFFFF0000)],
          <double>[0, 0.5, 1])),
    'gradient.conical': (Canvas c) => c.drawRect(const Rect.fromLTWH(0, 0, 64, 64), Paint()
      ..shader = Gradient.radial(const Offset(32, 32), 30, const <Color>[Color(0xFFFFFFFF), Color(0xFF004080)], null,
          TileMode.clamp, null, const Offset(24, 24), 4)),
    'clip.rect': (Canvas c) {
      c.clipRect(const Rect.fromLTRB(10.5, 10.5, 50.5, 40.5));
      c.drawPaint(p(0xFF336699));
    },
    'clip.rrect': (Canvas c) {
      c.clipRRect(RRect.fromLTRBR(8, 8, 56, 56, const Radius.circular(16)));
      c.drawPaint(p(0xFF993366));
    },
    'clip.difference': (Canvas c) {
      c.clipRect(const Rect.fromLTRB(16, 16, 48, 48), clipOp: ClipOp.difference);
      c.drawPaint(p(0xFF666666));
    },
    'transform.rotate': (Canvas c) {
      c.translate(32, 32);
      c.rotate(math.pi / 6);
      c.drawRect(const Rect.fromLTRB(-16, -10, 16, 10), p(0xFF0088FF));
    },
    'transform.skew.scale': (Canvas c) {
      c.scale(1.5, 0.75);
      c.skew(0.3, 0.1);
      c.drawRect(const Rect.fromLTRB(4, 8, 30, 40), p(0xFF88FF00));
    },
    'saveLayer.alpha': (Canvas c) {
      c.saveLayer(null, Paint()..color = const Color(0x80000000));
      c.drawRect(const Rect.fromLTRB(4, 4, 40, 40), p(0xFFFF0000));
      c.drawRect(const Rect.fromLTRB(24, 24, 60, 60), p(0xFF0000FF));
      c.restore();
    },
    'colorFilter.matrix': (Canvas c) => c.drawRect(const Rect.fromLTRB(8, 8, 56, 56), p(0xFF00AA00)
      ..colorFilter = const ColorFilter.matrix(<double>[0.2126, 0.7152, 0.0722, 0, 0, 0.2126, 0.7152, 0.0722, 0, 0,
          0.2126, 0.7152, 0.0722, 0, 0, 0, 0, 0, 1, 0])),
    'colorFilter.matrix.translate': (Canvas c) => c.drawRect(const Rect.fromLTRB(8, 8, 56, 56), p(0xFF204060)
      ..colorFilter = const ColorFilter.matrix(<double>[1, 0, 0, 0, 51, 0, 1, 0, 0, -20, 0, 0, 1, 0, 100, 0, 0, 0, 1, 0])),
    'colorFilter.mode': (Canvas c) => c.drawRect(const Rect.fromLTRB(8, 8, 56, 56), p(0xFF00AA00)
      ..colorFilter = const ColorFilter.mode(Color(0xFFAA0000), BlendMode.color)),
    'invertColors': (Canvas c) => c.drawRect(const Rect.fromLTRB(8, 8, 56, 56), p(0xFF3366CC)..invertColors = true),
    'maskFilter.blur': (Canvas c) => c.drawRect(const Rect.fromLTRB(16, 16, 48, 48), p(0xFF000000)
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 4)),
    'imageFilter.blur': (Canvas c) {
      c.saveLayer(null, Paint()..imageFilter = ImageFilter.blur(sigmaX: 3, sigmaY: 3));
      c.drawRect(const Rect.fromLTRB(16, 16, 48, 48), p(0xFFFF8800));
      c.restore();
    },
    'imageFilter.dilate': (Canvas c) {
      c.saveLayer(null, Paint()..imageFilter = ImageFilter.dilate(radiusX: 4, radiusY: 2));
      c.drawRect(const Rect.fromLTRB(20, 20, 44, 44), p(0xFF8800FF));
      c.restore();
    },
    'image.draw': (Canvas c) => c.drawImage(img, const Offset(8, 8), Paint()),
    'image.rect.nearest': (Canvas c) => c.drawImageRect(img, const Rect.fromLTRB(0, 0, 16, 16),
        const Rect.fromLTRB(0, 0, 64, 64), Paint()..filterQuality = FilterQuality.none),
    'image.shader': (Canvas c) => c.drawRect(const Rect.fromLTWH(0, 0, 64, 64), Paint()
      ..shader = ImageShader(img, TileMode.repeated, TileMode.mirror, Float64List.fromList(<double>[
        1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 3, 5, 0, 1]), filterQuality: FilterQuality.none)),
    'path.cubic': (Canvas c) => c.drawPath(Path()
      ..moveTo(8, 56)
      ..cubicTo(8, 0, 56, 64, 56, 8)
      ..lineTo(32, 32)
      ..close(), p(0xFF884422)),
    'path.evenOdd': (Canvas c) => c.drawPath(Path()
      ..fillType = PathFillType.evenOdd
      ..addOval(const Rect.fromLTRB(4, 4, 60, 60))
      ..addRect(const Rect.fromLTRB(20, 20, 44, 44)), p(0xFF2288AA)),
    'arc': (Canvas c) => c.drawArc(const Rect.fromLTRB(8, 8, 56, 56), 0.3, 2.5, true, p(0xFFCC0044)),
    'points': (Canvas c) => c.drawPoints(PointMode.polygon,
        const <Offset>[Offset(8, 8), Offset(56, 20), Offset(12, 40), Offset(50, 56)], p(0xFF000000)..strokeWidth = 3),
    'shadow': (Canvas c) => c.drawShadow(Path()..addRect(const Rect.fromLTRB(16, 16, 48, 48)), const Color(0xFF000000), 6, false),
  };
}

void main() {
  test('painting_pixels', () async {
    if (Platform.environment['FLUTTER_TEST_IMPELLER'] != 'true') {
      return; // the reference is Impeller's; see the header
    }
    final Image img = await checkerImage();
    final Map<String, Object> out = <String, Object>{};
    for (final MapEntry<String, void Function(Canvas)> e in scenes(img).entries) {
      final PictureRecorder rec = PictureRecorder();
      e.value(Canvas(rec));
      final Image image = await rec.endRecording().toImage(kSize, kSize);
      final ByteData bytes = (await image.toByteData())!;
      out[e.key] = base64Encode(bytes.buffer.asUint8List());
    }
    File('build/painting_pixels.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
