// Reference side of flutter-go's compositing pixel differential test:
// scenes built with SceneBuilder (nested transforms, clips, opacity, color
// and image filters, backdrop filters, shader masks, retained layers),
// rasterized with Scene.toImage and read back as rawRgba (premultiplied).
// Run with --enable-impeller; without it the test writes nothing.
// Output: build/compositing_pixels.json as {scene: base64 bytes}.
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

const int kSize = 64;

Picture pic(void Function(Canvas c) draw) {
  final PictureRecorder rec = PictureRecorder();
  draw(Canvas(rec));
  return rec.endRecording();
}

Paint p(int color) => Paint()..color = Color(color);

// Two overlapping rects: red over blue.
Picture overlap() => pic((Canvas c) {
  c.drawRect(const Rect.fromLTRB(8, 8, 40, 40), p(0xFF0000FF));
  c.drawRect(const Rect.fromLTRB(24, 24, 56, 56), p(0xFFFF0000));
});

// A background of four colored quadrants.
Picture quadrants() => pic((Canvas c) {
  c.drawRect(const Rect.fromLTRB(0, 0, 32, 32), p(0xFFFF0000));
  c.drawRect(const Rect.fromLTRB(32, 0, 64, 32), p(0xFF00FF00));
  c.drawRect(const Rect.fromLTRB(0, 32, 32, 64), p(0xFF0000FF));
  c.drawRect(const Rect.fromLTRB(32, 32, 64, 64), p(0xFFFFFF00));
});

Picture square(int color) => pic((Canvas c) => c.drawRect(const Rect.fromLTRB(16, 16, 48, 48), p(color)));

Float64List m4(List<double> v) => Float64List.fromList(v);

Float64List rotation(double r) {
  final double c = math.cos(r), s = math.sin(r);
  return m4(<double>[c, s, 0, 0, -s, c, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]);
}

Map<String, void Function(SceneBuilder b)> scenes() {
  final Float64List scale2 = m4(<double>[2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]);
  return <String, void Function(SceneBuilder b)>{
    'picture': (SceneBuilder b) => b.addPicture(const Offset(4, 6), overlap()),
    'offset.nested': (SceneBuilder b) {
      b.pushOffset(8, 4);
      b.pushOffset(-4, 8);
      b.addPicture(Offset.zero, square(0xFF336699));
      b.pop();
      b.pop();
    },
    'transform.scale': (SceneBuilder b) {
      b.pushTransform(scale2);
      b.addPicture(Offset.zero, pic((Canvas c) => c.drawRect(const Rect.fromLTRB(4, 4, 20, 12), p(0xFF884422))));
      b.pop();
    },
    'transform.rotate.offset': (SceneBuilder b) {
      b.pushOffset(32, 32);
      b.pushTransform(rotation(math.pi / 2));
      b.addPicture(Offset.zero, pic((Canvas c) => c.drawRect(const Rect.fromLTRB(0, 0, 24, 12), p(0xFF22AA44))));
      b.pop();
      b.pop();
    },
    'transform.zero': (SceneBuilder b) {
      b.pushTransform(Float64List(16));
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'clipRect.hardEdge': (SceneBuilder b) {
      b.pushClipRect(const Rect.fromLTRB(10, 12, 50, 40), clipBehavior: Clip.hardEdge);
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'clipRect.antiAlias': (SceneBuilder b) {
      b.pushClipRect(const Rect.fromLTRB(10, 12, 50, 40));
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'clipRect.saveLayer.opacity': (SceneBuilder b) {
      b.pushOpacity(128);
      b.pushClipRect(const Rect.fromLTRB(4, 4, 60, 60), clipBehavior: Clip.antiAliasWithSaveLayer);
      b.addPicture(Offset.zero, overlap());
      b.pop();
      b.pop();
    },
    'clipRRect': (SceneBuilder b) {
      b.pushClipRRect(RRect.fromLTRBR(8, 8, 56, 56, const Radius.circular(16)));
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'clipRRect.rect': (SceneBuilder b) {
      b.pushClipRRect(RRect.fromLTRBR(8, 16, 56, 48, Radius.zero), clipBehavior: Clip.hardEdge);
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'clipRSuperellipse': (SceneBuilder b) {
      b.pushClipRSuperellipse(RSuperellipse.fromLTRBR(6, 6, 58, 58, const Radius.circular(20)));
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'clipPath.oval': (SceneBuilder b) {
      b.pushClipPath(Path()..addOval(const Rect.fromLTRB(4, 12, 60, 52)));
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'clips.nested': (SceneBuilder b) {
      b.pushClipRect(const Rect.fromLTRB(0, 0, 40, 64), clipBehavior: Clip.hardEdge);
      b.pushOffset(8, 0);
      b.pushClipRect(const Rect.fromLTRB(0, 16, 64, 48), clipBehavior: Clip.hardEdge);
      b.addPicture(Offset.zero, quadrants());
      b.pop();
      b.pop();
      b.pop();
    },
    'opacity': (SceneBuilder b) {
      b.pushOpacity(128);
      b.addPicture(Offset.zero, overlap());
      b.pop();
    },
    'opacity.offset': (SceneBuilder b) {
      b.pushOpacity(200, offset: const Offset(4, -4));
      b.addPicture(Offset.zero, overlap());
      b.pop();
    },
    'opacity.nested': (SceneBuilder b) {
      b.pushOpacity(160);
      b.addPicture(Offset.zero, square(0xFF00FF00));
      b.pushOpacity(100);
      b.addPicture(const Offset(12, 12), square(0xFF0000FF));
      b.pop();
      b.pop();
    },
    'opacity.nested.direct': (SceneBuilder b) {
      b.pushOpacity(128);
      b.pushOpacity(128);
      b.addPicture(Offset.zero, overlap());
      b.pop();
      b.pop();
    },
    'transform.w2': (SceneBuilder b) {
      b.pushTransform(m4(<double>[1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 8, 4, 0, 2]));
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'backdrop.blur.full': (SceneBuilder b) {
      b.addPicture(Offset.zero, quadrants());
      b.pushBackdropFilter(ImageFilter.blur(sigmaX: 6, sigmaY: 6));
      b.pop();
    },
    'opacity.wrap300': (SceneBuilder b) {
      b.pushOpacity(300);
      b.addPicture(Offset.zero, overlap());
      b.pop();
    },
    'opacity.zero': (SceneBuilder b) {
      b.addPicture(Offset.zero, square(0xFF000000));
      b.pushOpacity(0);
      b.addPicture(Offset.zero, overlap());
      b.pop();
    },
    'colorFilter.mode': (SceneBuilder b) {
      b.pushColorFilter(const ColorFilter.mode(Color(0xFF00AA00), BlendMode.modulate));
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'colorFilter.matrix': (SceneBuilder b) {
      b.pushColorFilter(const ColorFilter.matrix(<double>[
        0.2126, 0.7152, 0.0722, 0, 0, 0.2126, 0.7152, 0.0722, 0, 0, 0.2126, 0.7152, 0.0722, 0, 0, 0, 0, 0, 1, 0,
      ]));
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'colorFilter.opacity.commute': (SceneBuilder b) {
      b.pushOpacity(128);
      b.pushColorFilter(const ColorFilter.mode(Color(0xFFFF8800), BlendMode.srcIn));
      b.addPicture(Offset.zero, overlap());
      b.pop();
      b.pop();
    },
    'colorFilter.opacity.noCommute': (SceneBuilder b) {
      b.pushOpacity(128);
      b.pushColorFilter(const ColorFilter.mode(Color(0x8000FFFF), BlendMode.srcOver));
      b.addPicture(Offset.zero, overlap());
      b.pop();
      b.pop();
    },
    'imageFilter.blur': (SceneBuilder b) {
      b.pushImageFilter(ImageFilter.blur(sigmaX: 3, sigmaY: 3));
      b.addPicture(Offset.zero, square(0xFFFF8800));
      b.pop();
    },
    'imageFilter.blur.offset': (SceneBuilder b) {
      b.pushImageFilter(ImageFilter.blur(sigmaX: 2, sigmaY: 4), offset: const Offset(6, -6));
      b.addPicture(Offset.zero, square(0xFF8800FF));
      b.pop();
    },
    'imageFilter.dilate': (SceneBuilder b) {
      b.pushImageFilter(ImageFilter.dilate(radiusX: 4, radiusY: 2));
      b.addPicture(Offset.zero, square(0xFF0088FF));
      b.pop();
    },
    'imageFilter.matrix': (SceneBuilder b) {
      b.pushImageFilter(ImageFilter.matrix(
          m4(<double>[1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 10, 5, 0, 1]),
          filterQuality: FilterQuality.none));
      b.addPicture(Offset.zero, square(0xFF00AA88));
      b.pop();
    },
    'imageFilter.opacity': (SceneBuilder b) {
      b.pushOpacity(128);
      b.pushImageFilter(ImageFilter.blur(sigmaX: 2, sigmaY: 2));
      b.addPicture(Offset.zero, overlap());
      b.pop();
      b.pop();
    },
    'backdrop.blur': (SceneBuilder b) {
      b.addPicture(Offset.zero, quadrants());
      b.pushClipRect(const Rect.fromLTRB(16, 16, 48, 48), clipBehavior: Clip.hardEdge);
      b.pushBackdropFilter(ImageFilter.blur(sigmaX: 4, sigmaY: 4));
      b.pop();
      b.pop();
    },
    'backdrop.blur.child': (SceneBuilder b) {
      b.addPicture(Offset.zero, quadrants());
      b.pushClipRect(const Rect.fromLTRB(8, 24, 56, 40), clipBehavior: Clip.hardEdge);
      b.pushBackdropFilter(ImageFilter.blur(sigmaX: 3, sigmaY: 3));
      b.addPicture(Offset.zero, pic((Canvas c) => c.drawRect(const Rect.fromLTRB(28, 0, 36, 64), p(0x80FFFFFF))));
      b.pop();
      b.pop();
    },
    'backdrop.matrix': (SceneBuilder b) {
      b.addPicture(Offset.zero, quadrants());
      b.pushClipRect(const Rect.fromLTRB(16, 16, 48, 48), clipBehavior: Clip.hardEdge);
      b.pushBackdropFilter(ImageFilter.matrix(
          m4(<double>[1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 8, 8, 0, 1]),
          filterQuality: FilterQuality.none));
      b.pop();
      b.pop();
    },
    'shaderMask.linear.dstIn': (SceneBuilder b) {
      b.pushShaderMask(
        Gradient.linear(Offset.zero, const Offset(48, 0), const <Color>[Color(0xFF000000), Color(0x00000000)]),
        const Rect.fromLTRB(8, 8, 56, 56),
        BlendMode.dstIn,
      );
      b.addPicture(Offset.zero, quadrants());
      b.pop();
    },
    'shaderMask.radial.srcATop': (SceneBuilder b) {
      b.pushShaderMask(
        Gradient.radial(const Offset(24, 24), 24, const <Color>[Color(0xFFFFFFFF), Color(0xFF000000)]),
        const Rect.fromLTRB(8, 8, 56, 56),
        BlendMode.srcATop,
      );
      b.addPicture(Offset.zero, overlap());
      b.pop();
    },
    'shaderMask.opacity': (SceneBuilder b) {
      b.pushOpacity(128);
      b.pushShaderMask(
        Gradient.linear(Offset.zero, const Offset(0, 64), const <Color>[Color(0xFF000000), Color(0x00000000)]),
        const Rect.fromLTRB(0, 0, 64, 64),
        BlendMode.dstIn,
      );
      b.addPicture(Offset.zero, quadrants());
      b.pop();
      b.pop();
    },
    'texture.platformView': (SceneBuilder b) {
      b.addPicture(Offset.zero, square(0xFF123456));
      b.addTexture(1, offset: const Offset(4, 4), width: 20, height: 20);
      b.addPlatformView(2, offset: const Offset(30, 30), width: 10, height: 10);
      b.addPerformanceOverlay(0, const Rect.fromLTRB(0, 0, 64, 64));
    },
    'retained': (SceneBuilder b) {
      final SceneBuilder first = SceneBuilder();
      final OpacityEngineLayer layer = first.pushOpacity(128);
      first.pushOffset(8, 8);
      first.addPicture(Offset.zero, overlap());
      first.pop();
      first.pop();
      first.build().dispose();
      b.addPicture(Offset.zero, square(0xFF00FF00));
      b.addRetained(layer);
    },
    'complex': (SceneBuilder b) {
      b.addPicture(Offset.zero, quadrants());
      b.pushTransform(m4(<double>[1, 0, 0, 0, 0.25, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]));
      b.pushClipRRect(RRect.fromLTRBR(4, 4, 52, 52, const Radius.circular(8)));
      b.pushOpacity(192);
      b.pushColorFilter(const ColorFilter.mode(Color(0xFF3366FF), BlendMode.modulate));
      b.addPicture(Offset.zero, overlap());
      b.pushOffset(16, 16);
      b.addPicture(Offset.zero, pic((Canvas c) => c.drawCircle(const Offset(8, 8), 8, p(0xFFFFFFFF))));
      b.pop();
      b.pop();
      b.pop();
      b.pop();
      b.pop();
    },
  };
}

void main() {
  test('compositing_pixels', () async {
    if (Platform.environment['FLUTTER_TEST_IMPELLER'] != 'true') {
      return; // the reference is Impeller's; see the header
    }
    final Map<String, Object> out = <String, Object>{};
    for (final MapEntry<String, void Function(SceneBuilder)> e in scenes().entries) {
      final SceneBuilder b = SceneBuilder();
      e.value(b);
      final Scene scene = b.build();
      final Image image = await scene.toImage(kSize, kSize);
      final ByteData bytes = (await image.toByteData())!;
      out[e.key] = base64Encode(bytes.buffer.asUint8List());
      scene.dispose();
    }
    File('build/compositing_pixels.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
