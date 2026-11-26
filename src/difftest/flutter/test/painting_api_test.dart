// Reference side of flutter-go's dart:ui drawing-API differential test:
// Paint (stored values, toString), MaskFilter, ColorFilter, ImageFilter,
// Gradient/Vertices argument checks, Shadow, enums, and Canvas state
// (transforms, save counts, clip bounds) — everything that is computed
// without rasterizing. Output: build/painting_api.json as {case: value}.
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

List<Object> c(Color x) => <Object>[x.a, x.r, x.g, x.b, x.colorSpace.index];
List<double> r(Rect x) => <double>[x.left, x.top, x.right, x.bottom];

String err(void Function() f) {
  try {
    f();
  } catch (e) {
    return '$e';
  }
  return 'no error';
}

List<Object> paintState(Paint p) => <Object>[
  p.isAntiAlias,
  c(p.color),
  p.blendMode.index,
  p.style.index,
  p.strokeWidth,
  p.strokeCap.index,
  p.strokeJoin.index,
  p.strokeMiterLimit,
  p.filterQuality.index,
  p.invertColors,
  '${p.maskFilter}',
  '${p.colorFilter}',
  '${p.imageFilter}',
];

// Canvas state after a script of operations.
Map<String, Object> canvasState(void Function(Canvas c) script, [Rect? cull]) {
  final PictureRecorder rec = PictureRecorder();
  final Canvas canvas = cull == null ? Canvas(rec) : Canvas(rec, cull);
  script(canvas);
  final Map<String, Object> out = <String, Object>{
    'transform': canvas.getTransform().toList(),
    'saveCount': canvas.getSaveCount(),
    'local': r(canvas.getLocalClipBounds()),
    'device': r(canvas.getDestinationClipBounds()),
  };
  rec.endRecording();
  return out;
}

void main() {
  test('painting_api', () {
    final Map<String, Object?> out = <String, Object?>{};

    // ---- Paint ----
    out['paint.default'] = paintState(Paint());
    out['paint.default.toString'] = '${Paint()}';
    final Paint p1 = Paint()
      ..color = const Color.from(alpha: 0.3, red: 0.1, green: 0.7, blue: 0.123456789)
      ..strokeWidth = 0.1
      ..strokeMiterLimit = 5.3
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.bevel
      ..blendMode = BlendMode.multiply
      ..isAntiAlias = false
      ..filterQuality = FilterQuality.high
      ..invertColors = true
      ..maskFilter = const MaskFilter.blur(BlurStyle.outer, 2.25)
      ..colorFilter = const ColorFilter.mode(Color(0x80123456), BlendMode.plus)
      ..imageFilter = ImageFilter.blur(sigmaX: 1.5, sigmaY: 2);
    out['paint.set'] = paintState(p1);
    out['paint.set.toString'] = '$p1';
    out['paint.from'] = paintState(Paint.from(p1));
    out['paint.p3'] = c((Paint()..color = const Color.from(alpha: 1, red: 1.2, green: -0.1, blue: 0.3, colorSpace: ColorSpace.displayP3)).color);
    out['paint.ext'] = c((Paint()..color = const Color.from(alpha: 0.5, red: 1.5, green: -0.5, blue: 0.3, colorSpace: ColorSpace.extendedSRGB)).color);
    out['paint.alphaRounding'] = c((Paint()..color = const Color.from(alpha: 0.07823470411233702, red: 0, green: 0, blue: 0)).color);
    out['paint.stroke.hairline'] = '${Paint()..style = PaintingStyle.stroke}';
    out['paint.stroke.miter4'] = '${Paint()..style = PaintingStyle.stroke..strokeMiterLimit = 4.0..strokeWidth = 3.25}';
    out['paint.stroke.miter10'] = '${Paint()..style = PaintingStyle.stroke..strokeMiterLimit = 10.0..strokeWidth = 1.0}';
    out['paint.stroke.square.round'] = '${Paint()..style = PaintingStyle.stroke..strokeCap = StrokeCap.square..strokeJoin = StrokeJoin.round}';
    out['paint.color.toString'] = '${Paint()..color = const Color(0x12345678)}';
    out['paint.shader.toString'] = '${Paint()..shader = Gradient.linear(Offset.zero, const Offset(1, 1), const <Color>[Color(0xFF000000), Color(0xFFFFFFFF)])}';
    out['paint.filters.toString'] = '${Paint()..colorFilter = const ColorFilter.linearToSrgbGamma()..imageFilter = ImageFilter.dilate(radiusX: 1, radiusY: 2)..filterQuality = FilterQuality.low}';
    out['paint.blend.toString'] = '${Paint()..blendMode = BlendMode.clear}';
    out['paint.invert.toString'] = '${Paint()..invertColors = true}';
    final ImageFilter blurA = ImageFilter.blur(sigmaX: 3, sigmaY: 3);
    final ImageFilter blurB = ImageFilter.blur(sigmaX: 3, sigmaY: 3);
    final Paint keep = Paint()..imageFilter = blurA;
    keep.imageFilter = blurB;
    out['paint.imageFilter.keepsEqual'] = identical(keep.imageFilter, blurA);
    keep.imageFilter = ImageFilter.blur(sigmaX: 4, sigmaY: 3);
    out['paint.imageFilter.replacesUnequal'] = identical(keep.imageFilter, blurA);
    out['paint.widths'] = <double>[
      for (final double w in <double>[0.1, 1 / 3, 1e-50, 1e50, -2.5]) (Paint()..strokeWidth = w).strokeWidth,
    ];
    out['paint.miters'] = <double>[
      for (final double w in <double>[0.1, 1 / 3, 4.0, 100, -2.5]) (Paint()..strokeMiterLimit = w).strokeMiterLimit,
    ];
    out['paint.maskSigma'] = '${(Paint()..maskFilter = const MaskFilter.blur(BlurStyle.normal, 1 / 3)).maskFilter}';
    out['paint.maskNull'] = '${(Paint()..maskFilter = const MaskFilter.blur(BlurStyle.normal, 1 / 3)..maskFilter = null).maskFilter}';

    // ---- MaskFilter ----
    out['mask.toString'] = <String>['${const MaskFilter.blur(BlurStyle.normal, 0.05)}', '${const MaskFilter.blur(BlurStyle.inner, 12.345)}'];
    out['mask.equal'] = <bool>[
      const MaskFilter.blur(BlurStyle.normal, 1) == const MaskFilter.blur(BlurStyle.normal, 1),
      const MaskFilter.blur(BlurStyle.normal, 1) == const MaskFilter.blur(BlurStyle.solid, 1),
      const MaskFilter.blur(BlurStyle.normal, 1) == const MaskFilter.blur(BlurStyle.normal, 2),
    ];

    // ---- ColorFilter ----
    final List<ColorFilter> cfs = <ColorFilter>[
      const ColorFilter.mode(Color(0xFF00FF00), BlendMode.srcIn),
      const ColorFilter.mode(Color(0xFF00FF00), BlendMode.srcOut),
      const ColorFilter.mode(Color(0x8000FF00), BlendMode.srcIn),
      const ColorFilter.matrix(<double>[1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0]),
      const ColorFilter.matrix(<double>[1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0.5]),
      const ColorFilter.linearToSrgbGamma(),
      const ColorFilter.srgbToLinearGamma(),
      ColorFilter.saturation(0.3),
    ];
    out['colorFilter.toString'] = <String>[for (final ColorFilter f in cfs) '$f'];
    out['colorFilter.short'] = <String>[for (final ColorFilter f in cfs) f.debugShortDescription];
    out['colorFilter.equal'] = <bool>[for (final ColorFilter a in cfs) for (final ColorFilter b in cfs) a == b];
    out['colorFilter.equal.copies'] = <bool>[
      const ColorFilter.matrix(<double>[1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0]) ==
          ColorFilter.matrix(List<double>.of(<double>[1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0])),
      ColorFilter.saturation(0.3) == ColorFilter.saturation(0.3),
      ColorFilter.saturation(0.3) == ColorFilter.saturation(0.31),
    ];
    out['colorFilter.saturation'] = <String>['${ColorFilter.saturation(0)}', '${ColorFilter.saturation(1.7)}', '${ColorFilter.saturation(-0.25)}'];

    // ---- ImageFilter ----
    final List<ImageFilter> ifs = <ImageFilter>[
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1),
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1, tileMode: TileMode.decal),
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1, tileMode: TileMode.clamp),
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1, tileMode: TileMode.mirror),
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1, tileMode: TileMode.repeated),
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1, bounds: const Rect.fromLTRB(1, 2, 3.5, 4)),
      ImageFilter.blur(),
      ImageFilter.dilate(radiusX: 1, radiusY: 2.5),
      ImageFilter.erode(radiusX: 0.5),
      ImageFilter.matrix(Float64List.fromList(<double>[2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1, 0, -1.5, 0.25, 0, 1])),
      ImageFilter.matrix(Float64List.fromList(<double>[2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1, 0, -1.5, 0.25, 0, 1]), filterQuality: FilterQuality.none),
      ImageFilter.compose(outer: ImageFilter.blur(sigmaX: 20, sigmaY: 20, tileMode: TileMode.decal), inner: ImageFilter.blur(sigmaX: 10, sigmaY: 10)),
      ImageFilter.compose(
        outer: ImageFilter.compose(
          outer: ImageFilter.blur(sigmaX: 30, sigmaY: 30, tileMode: TileMode.mirror),
          inner: ImageFilter.blur(sigmaX: 20, sigmaY: 20, tileMode: TileMode.repeated),
        ),
        inner: ImageFilter.compose(
          outer: const ColorFilter.mode(Color(0xFFABCDEF), BlendMode.color),
          inner: ImageFilter.matrix(Float64List.fromList(<double>[10, 0, 0, 0, 0, 10, 0, 0, 0, 0, 1, 0, -0.0, -0.0, 0, 1]), filterQuality: FilterQuality.low),
        ),
      ),
      const ColorFilter.mode(Color(0xFF00FF00), BlendMode.srcIn),
    ];
    out['imageFilter.toString'] = <String>[for (final ImageFilter f in ifs) '$f'];
    out['imageFilter.short'] = <String>[for (final ImageFilter f in ifs) f.debugShortDescription];
    final List<ImageFilter> ifs2 = <ImageFilter>[
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1),
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1, tileMode: TileMode.decal),
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1, tileMode: TileMode.clamp),
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1, tileMode: TileMode.mirror),
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1, tileMode: TileMode.repeated),
      ImageFilter.blur(sigmaX: 1.9, sigmaY: 2.1, bounds: const Rect.fromLTRB(1, 2, 3.5, 4)),
      ImageFilter.blur(),
      ImageFilter.dilate(radiusX: 1, radiusY: 2.5),
      ImageFilter.erode(radiusX: 0.5),
      ImageFilter.matrix(Float64List.fromList(<double>[2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1, 0, -1.5, 0.25, 0, 1])),
      ImageFilter.matrix(Float64List.fromList(<double>[2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1, 0, -1.5, 0.25, 0, 1]), filterQuality: FilterQuality.none),
      ImageFilter.compose(outer: ImageFilter.blur(sigmaX: 20, sigmaY: 20, tileMode: TileMode.decal), inner: ImageFilter.blur(sigmaX: 10, sigmaY: 10)),
      ImageFilter.compose(outer: ImageFilter.blur(sigmaX: 10, sigmaY: 10), inner: ImageFilter.blur(sigmaX: 20, sigmaY: 20, tileMode: TileMode.decal)),
      const ColorFilter.mode(Color(0xFF00FF00), BlendMode.srcIn),
    ];
    out['imageFilter.equal'] = <bool>[for (final ImageFilter a in ifs2) for (final ImageFilter b in ifs2) a == b];
    out['imageFilter.matrix.badLength'] = err(() => ImageFilter.matrix(Float64List(15)));

    // ---- Gradient / Vertices argument checks ----
    const List<Color> two = <Color>[Color(0xFF000000), Color(0xFFFFFFFF)];
    const List<Color> three = <Color>[Color(0xFF000000), Color(0xFF888888), Color(0xFFFFFFFF)];
    out['gradient.errors'] = <String>[
      err(() => Gradient.linear(Offset.zero, const Offset(1, 0), three)),
      err(() => Gradient.linear(Offset.zero, const Offset(1, 0), three, <double>[0, 1])),
      err(() => Gradient.radial(Offset.zero, 1, three)),
      err(() => Gradient.sweep(Offset.zero, three, <double>[0, 0.5])),
      err(() => Gradient.linear(Offset.zero, const Offset(1, 0), two)),
      err(() => Gradient.linear(Offset.zero, const Offset(1, 0), three, <double>[0, 0.5, 1])),
    ];
    const List<Offset> tri = <Offset>[Offset.zero, Offset(10, 0), Offset(0, 10)];
    out['vertices.errors'] = <String>[
      err(() => Vertices(VertexMode.triangles, tri, colors: const <Color>[Color(0xFF000000)])),
      err(() => Vertices(VertexMode.triangles, tri, textureCoordinates: const <Offset>[Offset.zero])),
      err(() => Vertices(VertexMode.triangles, tri, indices: <int>[0, 1, 3])),
      err(() => Vertices(VertexMode.triangles, tri, indices: <int>[0, 1, 2])),
      err(() => Vertices.raw(VertexMode.triangles, Float32List(5))),
      err(() => Vertices.raw(VertexMode.triangles, Float32List(6), colors: Int32List(2))),
      err(() => Vertices.raw(VertexMode.triangles, Float32List(6), textureCoordinates: Float32List(4))),
      err(() => Vertices.raw(VertexMode.triangles, Float32List(6), indices: Uint16List.fromList(<int>[0, 3]))),
    ];

    // ---- Shadow ----
    const Shadow s1 = Shadow(color: Color(0x80112233), offset: Offset(1.5, -2), blurRadius: 3.3);
    const Shadow s2 = Shadow(offset: Offset(10, 20), blurRadius: 0.7);
    out['shadow.toString'] = <String>['$s1', '${const Shadow()}', '${s1.scale(2.5)}'];
    out['shadow.sigma'] = <double>[Shadow.convertRadiusToSigma(0), Shadow.convertRadiusToSigma(3.3), Shadow.convertRadiusToSigma(-1), s2.blurSigma];
    out['shadow.lerp'] = <String>['${Shadow.lerp(s1, s2, 0.3)}', '${Shadow.lerp(null, s2, 0.3)}', '${Shadow.lerp(s1, null, 0.3)}', '${Shadow.lerp(null, null, 0.3)}'];
    out['shadow.lerpList'] = <String>['${Shadow.lerpList(<Shadow>[s1, s2], <Shadow>[s2], 0.25)}', '${Shadow.lerpList(null, <Shadow>[s1], 0.5)}', '${Shadow.lerpList(null, null, 0.5)}'];
    out['shadow.equal'] = <bool>[s1 == const Shadow(color: Color(0x80112233), offset: Offset(1.5, -2), blurRadius: 3.3), s1 == s2];
    out['shadow.toPaint'] = '${s1.toPaint()}';

    // ---- enums ----
    out['enums'] = <String>[
      ...BlendMode.values.map((BlendMode m) => '$m'),
      ...FilterQuality.values.map((FilterQuality m) => '$m'),
      ...StrokeCap.values.map((StrokeCap m) => '$m'),
      ...StrokeJoin.values.map((StrokeJoin m) => '$m'),
      ...PaintingStyle.values.map((PaintingStyle m) => '$m'),
      ...Clip.values.map((Clip m) => '$m'),
      ...BlurStyle.values.map((BlurStyle m) => '$m'),
      ...TileMode.values.map((TileMode m) => '$m'),
      ...VertexMode.values.map((VertexMode m) => '$m'),
      ...PointMode.values.map((PointMode m) => '$m'),
      ...ClipOp.values.map((ClipOp m) => '$m'),
    ];

    // ---- Canvas state ----
    out['canvas.fresh'] = canvasState((Canvas c) {});
    out['canvas.fresh.cull'] = canvasState((Canvas c) {}, const Rect.fromLTRB(10.5, 20.25, 300.75, 400.125));
    out['canvas.translate.scale'] = canvasState((Canvas c) {
      c.translate(10.3, -4.7);
      c.scale(2.5, 0.3);
      c.clipRect(const Rect.fromLTRB(1.1, 2.2, 50.7, 60.3));
    });
    out['canvas.scale.one'] = canvasState((Canvas c) {
      c.scale(3);
      c.clipRect(const Rect.fromLTRB(1.1, 2.2, 50.7, 60.3), doAntiAlias: false);
    });
    for (final double angle in <double>[0.3, math.pi / 2, math.pi, 1.0e-3, -2.2, 0.5, 1.3]) {
      out['canvas.rotate.$angle'] = canvasState((Canvas c) {
        c.translate(100, 50);
        c.rotate(angle);
        c.clipRect(const Rect.fromLTRB(-20.5, -10.25, 30, 40));
      });
    }
    out['canvas.skew'] = canvasState((Canvas c) {
      c.skew(0.25, -0.5);
      c.clipRect(const Rect.fromLTRB(0, 0, 100, 50));
    });
    out['canvas.transform'] = canvasState((Canvas c) {
      c.transform(Float64List.fromList(<double>[1.5, 0.2, 0, 0, -0.3, 0.8, 0, 0, 0, 0, 1, 0, 12.5, -7.25, 0, 1]));
      c.clipRect(const Rect.fromLTRB(3, 4, 50, 60));
    });
    out['canvas.perspective'] = canvasState((Canvas c) {
      c.transform(Float64List.fromList(<double>[1, 0, 0, 0.001, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]));
      c.clipRect(const Rect.fromLTRB(3, 4, 50, 60));
    });
    out['canvas.transform.badLength'] = err(() {
      final PictureRecorder rec = PictureRecorder();
      Canvas(rec).transform(Float64List(4));
    });
    out['canvas.clip.difference'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(0, 0, 100, 100));
      c.clipRect(const Rect.fromLTRB(0, 0, 100, 40.4), clipOp: ClipOp.difference);
    });
    out['canvas.clip.difference.noaa'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(0, 0, 100, 100));
      c.clipRect(const Rect.fromLTRB(-5, 60.6, 105, 120), clipOp: ClipOp.difference, doAntiAlias: false);
    });
    out['canvas.clip.difference.middle'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(0, 0, 100, 100));
      c.clipRect(const Rect.fromLTRB(20, 20, 80, 80), clipOp: ClipOp.difference);
    });
    out['canvas.clip.unsorted'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(80, 90, 10, 20));
    });
    out['canvas.clip.empty'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(0, 0, 10, 10));
      c.clipRect(const Rect.fromLTRB(20, 20, 30, 30));
    });
    out['canvas.clip.rrect'] = canvasState((Canvas c) {
      c.scale(1.5);
      c.clipRRect(RRect.fromLTRBR(10.2, 20.7, 90.1, 80.3, const Radius.circular(5)));
    });
    out['canvas.clip.rrect.noaa'] = canvasState((Canvas c) {
      c.rotate(0.5);
      c.clipRRect(RRect.fromLTRBR(10.2, 20.7, 90.1, 80.3, const Radius.circular(5)), doAntiAlias: false);
    });
    out['canvas.clip.rse'] = canvasState((Canvas c) {
      c.translate(0.5, 0.25);
      c.clipRSuperellipse(RSuperellipse.fromLTRBR(10.2, 20.7, 90.1, 80.3, const Radius.circular(5)));
    });
    out['canvas.clip.path'] = canvasState((Canvas c) {
      c.scale(2);
      c.clipPath(Path()..moveTo(10.3, 5)..lineTo(50.5, 70.2)..quadraticBezierTo(0, 100, -5.5, 30)..close());
    });
    out['canvas.save.restore'] = canvasState((Canvas c) {
      c.save();
      c.translate(10, 10);
      c.clipRect(const Rect.fromLTRB(0, 0, 50, 50));
      c.save();
      c.scale(2);
      c.restore();
    });
    out['canvas.restoreToCount'] = canvasState((Canvas c) {
      c.save();
      c.clipRect(const Rect.fromLTRB(0, 0, 50, 50));
      c.save();
      c.translate(5, 5);
      c.save();
      c.restoreToCount(2);
    });
    out['canvas.restoreToCount.zero'] = canvasState((Canvas c) {
      c.save();
      c.save();
      c.restoreToCount(0);
      c.restore();
    });
    out['canvas.saveLayer.bounds'] = canvasState((Canvas c) {
      c.translate(3.3, 4.4);
      c.saveLayer(const Rect.fromLTRB(10.5, 10.5, 60.25, 70.75), Paint());
    });
    out['canvas.saveLayer.null'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(0, 0, 100, 100));
      c.saveLayer(null, Paint()..color = const Color(0x80000000));
    });
    out['canvas.saveLayer.blur'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(10, 10, 100, 100));
      c.scale(2);
      c.saveLayer(null, Paint()..imageFilter = ImageFilter.blur(sigmaX: 2.5, sigmaY: 1));
    });
    out['canvas.saveLayer.blur.bounds'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(10, 10, 100, 100));
      c.saveLayer(const Rect.fromLTRB(20, 20, 40, 40), Paint()..imageFilter = ImageFilter.blur(sigmaX: 2.5, sigmaY: 1));
    });
    out['canvas.saveLayer.dilate'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(10, 10, 100, 100));
      c.saveLayer(null, Paint()..imageFilter = ImageFilter.dilate(radiusX: 3.5, radiusY: 1));
    });
    out['canvas.saveLayer.erode'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(10, 10, 100, 100));
      c.rotate(0.2);
      c.saveLayer(null, Paint()..imageFilter = ImageFilter.erode(radiusX: 3.5, radiusY: 1));
    });
    out['canvas.saveLayer.restore'] = canvasState((Canvas c) {
      c.clipRect(const Rect.fromLTRB(10, 10, 100, 100));
      c.saveLayer(const Rect.fromLTRB(20, 20, 40, 40), Paint());
      c.restore();
    });
    out['canvas.many'] = canvasState((Canvas c) {
      c.translate(17.25, -3.5);
      c.rotate(0.7);
      c.scale(1.3, 0.9);
      c.skew(0.1, 0.05);
      c.clipRect(const Rect.fromLTRB(-30, -40, 120, 80));
      c.rotate(-0.3);
      c.clipRect(const Rect.fromLTRB(-10, -20, 70, 60), doAntiAlias: false);
    });
    out['canvas.nonfinite'] = canvasState((Canvas c) {
      c.translate(double.infinity, 2);
      c.scale(double.nan);
      c.clipRect(const Rect.fromLTRB(0, 0, double.infinity, 10));
    });

    // ---- Canvas/recorder plumbing ----
    final PictureRecorder rec = PictureRecorder();
    out['recorder.before'] = '$rec ${rec.isRecording}';
    final Canvas canvas = Canvas(rec);
    out['recorder.during'] = '$rec ${rec.isRecording} $canvas';
    out['recorder.twice'] = err(() => Canvas(rec));
    final Picture pic = rec.endRecording();
    out['recorder.after'] = '$rec ${rec.isRecording} $canvas $pic';
    out['recorder.endTwice'] = err(() => rec.endRecording());
    out['canvas.afterEnd'] = <String>[err(() => canvas.getSaveCount()), err(() => canvas.getTransform()), err(() => canvas.save()), err(() => canvas.drawRect(Rect.zero, Paint())), err(() => canvas.clipRect(Rect.zero))];
    out['picture.toImage.bad'] = err(() => pic.toImageSync(0, 10));
    out['canvas.drawRawPoints.odd'] = err(() {
      final PictureRecorder r2 = PictureRecorder();
      Canvas(r2).drawRawPoints(PointMode.points, Float32List(3), Paint());
    });

    File('build/painting_api.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(JsonEncoder.withIndent(' ', (Object? o) => '$o').convert(out));
  });
}
