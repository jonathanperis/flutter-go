// Reference side of flutter-go's painting decorations differential test:
// box_decoration.dart and shape_decoration.dart. Every value is printed as
// a string with Dart's toString (doubles round-trip exactly), so the Go
// side (src/difftest/painting_decorations_test.go) must match bit for bit:
// lerps (Box, Shape and across via Decoration.lerp), scale, padding,
// isComplex, toString and toStringDeep, fromBoxDecoration, getClipPath
// (bounds, metrics, contains grid), hitTest grids, and the canvas calls of
// createBoxPainter().paint (printed when made). Output:
// build/painting_decorations.json as {case: value}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

String d(double v) => '$v';

String col(Color c) => '${d(c.a)},${d(c.r)},${d(c.g)},${d(c.b)}';

String rect(Rect r) => '${d(r.left)},${d(r.top)},${d(r.right)},${d(r.bottom)}';

String rrect(RRect r) =>
    '${d(r.left)},${d(r.top)},${d(r.right)},${d(r.bottom)},'
    '${d(r.tlRadiusX)},${d(r.tlRadiusY)},${d(r.trRadiusX)},${d(r.trRadiusY)},'
    '${d(r.brRadiusX)},${d(r.brRadiusY)},${d(r.blRadiusX)},${d(r.blRadiusY)}';

String rse(RSuperellipse r) =>
    '${d(r.left)},${d(r.top)},${d(r.right)},${d(r.bottom)},'
    '${d(r.tlRadiusX)},${d(r.tlRadiusY)},${d(r.trRadiusX)},${d(r.trRadiusY)},'
    '${d(r.brRadiusX)},${d(r.brRadiusY)},${d(r.blRadiusX)},${d(r.blRadiusY)}';

String off(Offset o) => '${d(o.dx)},${d(o.dy)}';

String insets(EdgeInsetsGeometry g, TextDirection td) {
  final EdgeInsets e = g.resolve(td);
  return '${d(e.left)},${d(e.top)},${d(e.right)},${d(e.bottom)}';
}

String pathSig(Path p, Rect area, {int n = 9}) {
  final b = StringBuffer('b=${rect(p.getBounds())};m=');
  for (final ui.PathMetric m in p.computeMetrics()) {
    b.write('[${d(m.length)},${m.isClosed}');
    for (final double f in <double>[0, 0.25, 0.5, 0.75, 1]) {
      final ui.Tangent? t = m.getTangentForOffset(m.length * f);
      b.write(t == null ? ',null' : ',${off(t.position)},${off(t.vector)}');
    }
    b.write(']');
  }
  b.write(';c=');
  final Rect a = area.inflate(3);
  for (int i = 0; i < n; i++) {
    for (int j = 0; j < n; j++) {
      final x = a.left + a.width * i / (n - 1);
      final y = a.top + a.height * j / (n - 1);
      b.write(p.contains(Offset(x, y)) ? '1' : '0');
    }
  }
  return b.toString();
}

// shaderId numbers shaders in the order a painter first uses them, so a
// shader re-created for a new rect (or a cached one) shows.
String shaderId(Shader? s, List<Shader> seen) {
  if (s == null) {
    return 'noshader';
  }
  int i = seen.indexWhere((Shader x) => identical(x, s));
  if (i < 0) {
    seen.add(s);
    i = seen.length - 1;
  }
  return 'shader#$i';
}

String paintF(Paint p, List<Shader> seen) =>
    '${col(p.color)}/${d(p.strokeWidth)}/${p.style}/${p.blendMode}/${p.maskFilter}/'
    '${shaderId(p.shader, seen)}/${p.filterQuality}/${p.invertColors}/${p.isAntiAlias}';

class Rec implements Canvas {
  Rec(this.area, this.shaders);
  final Rect area;
  final List<Shader> shaders;
  final List<String> log = <String>[];

  String arg(Object? a) {
    if (a is Rect) {
      return rect(a);
    }
    if (a is RRect) {
      return rrect(a);
    }
    if (a is RSuperellipse) {
      return rse(a);
    }
    if (a is Path) {
      return pathSig(a, area, n: 7);
    }
    if (a is Paint) {
      return paintF(a, shaders);
    }
    if (a is Offset) {
      return off(a);
    }
    if (a is double) {
      return d(a);
    }
    if (a is ui.Image) {
      return '${a.width}x${a.height}';
    }
    return '$a';
  }

  @override
  void noSuchMethod(Invocation invocation) {
    final String name = invocation.memberName.toString().replaceAll('Symbol("', '').replaceAll('")', '');
    log.add('$name(${invocation.positionalArguments.map(arg).join(' ; ')})');
  }
}

String err(Object e) => e is AssertionError || e is FlutterError ? 'assert' : 'throws ${e.runtimeType}';

String tryS(String Function() f) {
  try {
    return f();
  } catch (e) {
    return err(e);
  }
}

class Img extends ImageProvider<Img> {
  Img(this.image, this.name);
  final ui.Image image;
  final String name;

  @override
  Future<Img> obtainKey(ImageConfiguration configuration) => SynchronousFuture<Img>(this);

  @override
  ImageStreamCompleter loadImage(Img key, ImageDecoderCallback decode) =>
      OneFrameImageStreamCompleter(SynchronousFuture<ImageInfo>(ImageInfo(image: image.clone())));

  @override
  String toString() => 'Img($name)';
}

Future<ui.Image> makeImage(int w, int h) {
  final c = Completer<ui.Image>();
  ui.decodeImageFromPixels(Uint8List(w * h * 4), w, h, ui.PixelFormat.rgba8888, c.complete);
  return c.future;
}

BorderSide bs(int color, double width, {double strokeAlign = BorderSide.strokeAlignInside,
        BorderStyle style = BorderStyle.solid}) =>
    BorderSide(color: Color(color), width: width, strokeAlign: strokeAlign, style: style);

const LinearGradient lin = LinearGradient(
  begin: AlignmentDirectional.topStart,
  end: Alignment.bottomRight,
  colors: <Color>[Color(0xFFFF0000), Color(0xFF00FF00), Color(0x800000FF)],
  stops: <double>[0, 0.4, 1],
);
const RadialGradient rad = RadialGradient(
  center: Alignment(0.2, -0.3),
  radius: 0.7,
  colors: <Color>[Color(0xFFFFFF00), Color(0xFF8000FF)],
);
const SweepGradient swp = SweepGradient(
  colors: <Color>[Color(0xFF00FFFF), Color(0xFFFF00FF), Color(0xFF00FFFF)],
  transform: GradientRotation(0.5),
);

const List<BoxShadow> shadows1 = <BoxShadow>[
  BoxShadow(color: Color(0x80000000), offset: Offset(2, 3), blurRadius: 4, spreadRadius: 1),
  BoxShadow(color: Color(0x40FF0000), offset: Offset(-1, 5), blurRadius: 10, spreadRadius: -2, blurStyle: BlurStyle.outer),
];
const List<BoxShadow> shadows2 = <BoxShadow>[
  BoxShadow(color: Color(0xFF123456), offset: Offset(0, 1), blurRadius: 2.5, blurStyle: BlurStyle.solid),
  BoxShadow(color: Color(0x33000000), blurRadius: 0, spreadRadius: 3, blurStyle: BlurStyle.inner),
  BoxShadow(color: Color(0x99009900), offset: Offset(4, -4), blurRadius: 6, blurStyle: BlurStyle.outer),
];

List<BoxDecoration> boxes(ui.Image img) => <BoxDecoration>[
  const BoxDecoration(),
  const BoxDecoration(color: Color(0xFF3366CC)),
  BoxDecoration(
    color: const Color(0xFF22AA44),
    border: Border.all(width: 3, color: const Color(0xFF2196F3)),
    borderRadius: const BorderRadius.all(Radius.circular(12)),
  ),
  BoxDecoration(color: const Color(0xFFAA2244), shape: BoxShape.circle, border: Border.all(width: 4, color: const Color(0x802196F3))),
  const BoxDecoration(
    gradient: lin,
    borderRadius: BorderRadiusDirectional.only(topStart: Radius.circular(20), bottomEnd: Radius.elliptical(8, 30)),
  ),
  const BoxDecoration(color: Color(0xFF444444), boxShadow: shadows1),
  BoxDecoration(
    color: const Color(0xFF119988),
    border: BorderDirectional(start: bs(0xFF000000, 4), end: bs(0x80000000, 2), top: bs(0xFFFF0000, 6, strokeAlign: BorderSide.strokeAlignCenter)),
  ),
  BoxDecoration(
    color: const Color(0xFFFFCC00),
    backgroundBlendMode: BlendMode.multiply,
    border: Border(top: bs(0xFF0000FF, 5), left: bs(0xFF00FF00, 3)),
  ),
  const BoxDecoration(gradient: rad, shape: BoxShape.circle, boxShadow: shadows2),
  BoxDecoration(
    color: const Color(0xFF8800FF),
    border: Border.all(width: 6, color: const Color(0x80FFFFFF)),
    borderRadius: const BorderRadius.all(Radius.circular(9)),
  ),
  BoxDecoration(color: const Color(0xFF00AAFF), border: Border.all(width: 5, strokeAlign: BorderSide.strokeAlignOutside)),
  const BoxDecoration(color: Color(0xFF654321), borderRadius: BorderRadius.zero),
  const BoxDecoration(color: Color(0xFF123456), borderRadius: BorderRadiusDirectional.zero),
  const BoxDecoration(
    gradient: swp,
    borderRadius: BorderRadius.only(topLeft: Radius.elliptical(30, 10), bottomRight: Radius.circular(15)),
    boxShadow: shadows1,
  ),
  BoxDecoration(color: const Color(0xFF00FF00), image: DecorationImage(image: Img(img, 'a'), fit: BoxFit.cover), shape: BoxShape.circle),
  BoxDecoration(
    image: DecorationImage(image: Img(img, 'b'), alignment: AlignmentDirectional.bottomEnd),
    borderRadius: const BorderRadius.all(Radius.circular(10)),
    border: Border.all(width: 2),
  ),
  BoxDecoration(image: DecorationImage(image: Img(img, 'c'), repeat: ImageRepeat.repeatX, scale: 2)),
  BoxDecoration(color: const Color(0xFF444444), boxShadow: <BoxShadow>[shadows2[0], shadows2[1]]),
  BoxDecoration(
    color: const Color(0xFF00FF88),
    border: BorderDirectional(top: bs(0xFF000000, 4), start: bs(0xFF000000, 4), end: bs(0xFF000000, 4), bottom: bs(0xFF000000, 4)),
  ),
  BoxDecoration(color: const Color(0xFF884488), shape: BoxShape.circle, border: Border(top: bs(0xFF000000, 2))),
];

List<ShapeDecoration> shapeDecos(ui.Image img) => <ShapeDecoration>[
  const ShapeDecoration(shape: Border()),
  ShapeDecoration(color: const Color(0xFFAA2244), shape: CircleBorder(side: bs(0xFF2196F3, 3))),
  ShapeDecoration(
    gradient: lin,
    shape: RoundedRectangleBorder(side: bs(0xFF000000, 2), borderRadius: const BorderRadius.all(Radius.circular(12))),
  ),
  ShapeDecoration(color: const Color(0xFF22AA44), shadows: shadows1, shape: StadiumBorder(side: bs(0x80FF0000, 4))),
  ShapeDecoration(
    color: const Color(0xFF4422AA),
    shadows: shadows2,
    shape: StarBorder(side: bs(0xFF000000, 1.5), points: 6, innerRadiusRatio: 0.5, pointRounding: 0.2, rotation: 15),
  ),
  ShapeDecoration(
    color: const Color(0xFF884400),
    shadows: shadows1,
    shape: RoundedSuperellipseBorder(side: bs(0xFFFFFFFF, 3), borderRadius: const BorderRadius.all(Radius.circular(18))),
  ),
  ShapeDecoration(
    gradient: rad,
    shadows: shadows2,
    shape: const BeveledRectangleBorder(borderRadius: BorderRadiusDirectional.only(topStart: Radius.circular(14))),
  ),
  ShapeDecoration(color: const Color(0xFF00FFAA), shadows: shadows1, shape: Border.all(width: 4, color: const Color(0xFF333333))),
  ShapeDecoration(color: const Color(0xFF0088FF), shape: OvalBorder(side: bs(0x802196F3, 5, strokeAlign: BorderSide.strokeAlignCenter))),
  ShapeDecoration(
    color: const Color(0xFFFF8800),
    shape: CircleBorder(side: bs(0xFF000000, 2)) + RoundedRectangleBorder(side: bs(0xFF0000FF, 3)),
  ),
  ShapeDecoration(color: const Color(0xFF777777), shape: LinearBorder.bottom(side: bs(0xFF000000, 3))),
  ShapeDecoration(gradient: swp, shape: ContinuousRectangleBorder(side: bs(0xFF00FF00, 2), borderRadius: const BorderRadius.all(Radius.circular(30)))),
  ShapeDecoration(image: DecorationImage(image: Img(img, 's'), fit: BoxFit.fill), shape: CircleBorder(side: bs(0xFF000000, 2))),
  ShapeDecoration(
    color: const Color(0xFFCCCCCC),
    image: DecorationImage(image: Img(img, 't'), alignment: AlignmentDirectional.centerStart),
    shape: const RoundedRectangleBorder(borderRadius: BorderRadiusDirectional.only(topEnd: Radius.circular(25))),
  ),
  ShapeDecoration(image: DecorationImage(image: Img(img, 's2'), fit: BoxFit.fill), shape: CircleBorder(side: bs(0xFF000000, 2))),
];

const List<double> ts = <double>[-0.5, 0, 0.25, 0.5, 0.75, 1, 1.5];
const List<TextDirection?> tds = <TextDirection?>[TextDirection.ltr, TextDirection.rtl, null];

String paintLog(Decoration dec, List<(Offset, Size)> calls, TextDirection? td, [List<TextDirection?>? tdPerCall]) {
  final BoxPainter painter = dec.createBoxPainter(() {});
  final out = <String>[];
  final shaders = <Shader>[];
  for (int i = 0; i < calls.length; i++) {
    final (Offset o, Size s) = calls[i];
    final rec = Rec(o & s, shaders);
    final TextDirection? callTd = tdPerCall == null ? td : tdPerCall[i];
    out.add(tryS(() {
      painter.paint(rec, o, ImageConfiguration(size: s, textDirection: callTd));
      return rec.log.join('\n');
    }));
  }
  painter.dispose();
  return out.join('\n--\n');
}

String hitGrid(Decoration dec, Size size, TextDirection? td) {
  return tryS(() {
    final b = StringBuffer();
    const n = 11;
    for (int i = 0; i < n; i++) {
      for (int j = 0; j < n; j++) {
        b.write(dec.hitTest(size, Offset(size.width * i / (n - 1), size.height * j / (n - 1)), textDirection: td) ? '1' : '0');
      }
    }
    return b.toString();
  });
}

void describe(Map<String, String> out, String key, Decoration dec, ui.Image img) {
  out['$key.str'] = tryS(() => '$dec');
  out['$key.prop'] = tryS(() => DiagnosticsProperty<Decoration>('decoration', dec).toStringDeep());
  out['$key.deep'] = tryS(() => dec.toDiagnosticsNode().toStringDeep());
  out['$key.complex'] = '${dec.isComplex}';
  for (final TextDirection td in <TextDirection>[TextDirection.ltr, TextDirection.rtl]) {
    out['$key.padding.$td'] = tryS(() => insets(dec.padding, td));
  }
  out['$key.valid'] = tryS(() => '${dec.debugAssertIsValid()}');
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test('painting_decorations', () async {
    // The test binding disables shadows (AutomatedTestWidgetsFlutterBinding's
    // disableShadows); paint them, as an app does, except where noted.
    debugDisableShadows = false;
    final ui.Image img = await makeImage(16, 8);
    final out = <String, String>{};
    final List<BoxDecoration> bx = boxes(img);
    final List<ShapeDecoration> sx = shapeDecos(img);
    const List<Rect> rects = <Rect>[Rect.fromLTRB(10, 20, 110, 80), Rect.fromLTWH(-5, 3.5, 40, 90)];
    const List<Size> sizes = <Size>[Size(100, 60), Size(40, 90)];
    final List<(Offset, Size)> calls = <(Offset, Size)>[
      (const Offset(5, 7), const Size(100, 60)),
      (const Offset(5, 7), const Size(100, 60)),
      (const Offset(-3, 2.5), const Size(40, 90)),
    ];

    final all = <String, Decoration>{
      for (int i = 0; i < bx.length; i++) 'box.$i': bx[i],
      for (int i = 0; i < sx.length; i++) 'shape.$i': sx[i],
    };
    for (final MapEntry<String, Decoration> e in all.entries) {
      final String k = e.key;
      final Decoration dec = e.value;
      describe(out, k, dec, img);
      for (int r = 0; r < rects.length; r++) {
        for (final TextDirection td in <TextDirection>[TextDirection.ltr, TextDirection.rtl]) {
          out['$k.clip.$r.$td'] = tryS(() => pathSig(dec.getClipPath(rects[r], td), rects[r]));
        }
      }
      for (int s = 0; s < sizes.length; s++) {
        for (final TextDirection? td in tds) {
          out['$k.hit.$s.$td'] = hitGrid(dec, sizes[s], td);
        }
      }
      for (final TextDirection? td in tds) {
        out['$k.paint.$td'] = paintLog(dec, calls, td);
      }
      // One painter, the same rect, the text direction switching.
      out['$k.paint.switch'] = paintLog(dec, <(Offset, Size)>[calls[0], calls[0], calls[0]], null,
          <TextDirection?>[TextDirection.ltr, TextDirection.rtl, TextDirection.rtl]);
      for (final double t in <double>[0, 0.3, 1]) {
        out['$k.lerpNull.$t'] = tryS(() => '${Decoration.lerp(dec, null, t)}');
        out['$k.nullLerp.$t'] = tryS(() => '${Decoration.lerp(null, dec, t)}');
      }
    }
    // debugDisableShadows: outer shadows are clipped.
    debugDisableShadows = true;
    for (final String k in <String>['box.5', 'box.8', 'box.13', 'shape.3', 'shape.4', 'shape.5', 'shape.6', 'shape.7']) {
      out['$k.paint.noShadows'] = paintLog(all[k]!, calls, TextDirection.ltr);
    }
    debugDisableShadows = false;
    // createBoxPainter without a callback.
    for (final MapEntry<String, Decoration> e in all.entries) {
      out['${e.key}.painterNoCallback'] = tryS(() {
        e.value.createBoxPainter();
        return 'ok';
      });
    }

    // BoxDecoration: scale, lerps, ==, copyWith, fromBoxDecoration.
    for (int i = 0; i < bx.length; i++) {
      for (final double f in <double>[-0.5, 0, 0.5, 2]) {
        out['box.$i.scale.$f'] = tryS(() => '${bx[i].scale(f)}');
      }
      out['box.$i.scale.paint'] = paintLog(bx[i].scale(0.5), calls, TextDirection.ltr);
      out['box.$i.fromBox'] = tryS(() => '${ShapeDecoration.fromBoxDecoration(bx[i])}');
      out['box.$i.copy'] = tryS(() => '${bx[i].copyWith(shape: BoxShape.circle, color: const Color(0xFF010203))}');
      for (final double t in ts) {
        out['box.$i.lerp.null.$t'] = tryS(() => '${BoxDecoration.lerp(null, bx[i], t)}');
        out['box.$i.lerp.$t.null'] = tryS(() => '${BoxDecoration.lerp(bx[i], null, t)}');
      }
      for (int j = 0; j < bx.length; j++) {
        out['box.$i.$j.=='] = '${bx[i] == bx[j]}';
        for (final double t in ts) {
          out['box.$i.$j.lerp.$t'] = tryS(() => '${BoxDecoration.lerp(bx[i], bx[j], t)}');
        }
      }
    }
    // ShapeDecoration lerps and ==.
    for (int i = 0; i < sx.length; i++) {
      for (final double t in ts) {
        out['shape.$i.lerp.null.$t'] = tryS(() => '${ShapeDecoration.lerp(null, sx[i], t)}');
        out['shape.$i.lerp.$t.null'] = tryS(() => '${ShapeDecoration.lerp(sx[i], null, t)}');
      }
      for (int j = 0; j < sx.length; j++) {
        out['shape.$i.$j.=='] = '${sx[i] == sx[j]}';
        for (final double t in ts) {
          out['shape.$i.$j.lerp.$t'] = tryS(() => '${ShapeDecoration.lerp(sx[i], sx[j], t)}');
        }
      }
    }
    // Across types through Decoration.lerp (lerpFrom/lerpTo).
    for (int i = 0; i < bx.length; i++) {
      for (int j = 0; j < sx.length; j++) {
        for (final double t in ts) {
          out['cross.$i.$j.$t'] = tryS(() => '${Decoration.lerp(bx[i], sx[j], t)}');
          out['cross.$j.$i.$t'] = tryS(() => '${Decoration.lerp(sx[j], bx[i], t)}');
        }
      }
    }
    // Lerps' results painted and hit-tested.
    for (final (int i, int j) in <(int, int)>[(2, 3), (4, 13), (5, 8), (6, 7), (9, 10)]) {
      for (final double t in <double>[0.3, 0.7]) {
        final k = 'boxlerp.$i.$j.$t';
        final Decoration dec = BoxDecoration.lerp(bx[i], bx[j], t)!;
        out['$k.paint'] = paintLog(dec, calls, TextDirection.rtl);
        out['$k.hit'] = hitGrid(dec, sizes[0], TextDirection.rtl);
      }
    }
    for (final (int i, int j) in <(int, int)>[(1, 2), (3, 4), (5, 6), (2, 9), (7, 11)]) {
      for (final double t in <double>[0.3, 0.7]) {
        final k = 'shapelerp.$i.$j.$t';
        final Decoration dec = ShapeDecoration.lerp(sx[i], sx[j], t)!;
        out['$k.paint'] = paintLog(dec, calls, TextDirection.ltr);
        out['$k.hit'] = hitGrid(dec, sizes[1], TextDirection.ltr);
        out['$k.clip'] = tryS(() => pathSig(dec.getClipPath(rects[0], TextDirection.ltr), rects[0]));
      }
    }

    File('build/painting_decorations.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
