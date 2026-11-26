// Reference side of flutter-go's painting gradients/decorations
// differential test: Gradient (linear/radial/sweep) lerps between every
// pair, scale/withOpacity/fromColor, toString, ==, createShader's argument
// errors, GradientRotation's matrices; BoxShadow lerp/lerpList/scale/
// copyWith/toString/toPaint; DecorationImage toString/==/lerp; paintImage's
// recorded canvas calls over fits, alignments, repeats, flips, scales and
// center slices (and the oversized-image inversion); the DecorationImage
// painters (plain, matchTextDirection, blended); Decoration.lerp and
// FlutterLogoDecoration (lerps, toString, and the mark's painting).
// Output: build/painting_gradients.json as {case: value}; doubles are
// compared exactly (non-finite ones and -0.0 as strings).
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

Object? j(double? v) {
  if (v == null) {
    return null;
  }
  if (!v.isFinite) {
    return '$v';
  }
  if (v == 0 && v.isNegative) {
    return '-0.0';
  }
  return v;
}

List<Object?> js(Iterable<double> vs) => vs.map(j).toList();
List<Object?> rect(Rect r) => js(<double>[r.left, r.top, r.right, r.bottom]);
List<Object?> col(Color c) => <Object?>[j(c.a), j(c.r), j(c.g), j(c.b), c.colorSpace.index];
List<Object?> cols(List<Color> cs) => cs.map(col).toList();

String err(void Function() f) {
  try {
    f();
  } catch (e) {
    return e is AssertionError ? 'assertion' : '$e';
  }
  return 'no error';
}

Object? errOr(Object? Function() f) {
  try {
    return f();
  } catch (e) {
    return e is AssertionError ? 'assertion' : '$e';
  }
}

Object? align(AlignmentGeometry? a) {
  if (a == null) {
    return null;
  }
  return <Object?>[
    '$a',
    js(<double>[a.resolve(TextDirection.ltr).x, a.resolve(TextDirection.ltr).y]),
    js(<double>[a.resolve(TextDirection.rtl).x, a.resolve(TextDirection.rtl).y]),
  ];
}

Object? grad(Gradient? g) {
  if (g == null) {
    return null;
  }
  final List<Object?> common = <Object?>[
    g.runtimeType.toString(),
    cols(g.colors),
    g.stops == null ? null : js(g.stops!),
    g.transform == null ? null : '${g.transform}',
  ];
  switch (g) {
    case final LinearGradient l:
      return <Object?>[...common, align(l.begin), align(l.end), l.tileMode.index];
    case final RadialGradient r:
      return <Object?>[...common, align(r.center), j(r.radius), r.tileMode.index, align(r.focal), j(r.focalRadius)];
    case final SweepGradient s:
      return <Object?>[...common, align(s.center), j(s.startAngle), j(s.endAngle), s.tileMode.index];
  }
  return common;
}

Object? paintOf(Paint p) => <Object?>[
      col(p.color),
      p.blendMode.index,
      p.filterQuality.index,
      p.isAntiAlias,
      p.invertColors,
      p.colorFilter == null ? null : '${p.colorFilter}',
      p.shader != null,
      p.maskFilter == null ? null : '${p.maskFilter}',
    ];

// Records the canvas calls paintImage, the decoration image painters and
// the Flutter logo painter make.
class RecCanvas implements Canvas {
  final List<Object?> calls = <Object?>[];
  @override
  void save() => calls.add('save');
  @override
  void restore() => calls.add('restore');
  @override
  void saveLayer(Rect? bounds, Paint paint) =>
      calls.add(<Object?>['saveLayer', bounds == null ? null : rect(bounds), paintOf(paint)]);
  @override
  void translate(double dx, double dy) => calls.add(<Object?>['translate', j(dx), j(dy)]);
  @override
  void scale(double sx, [double? sy]) => calls.add(<Object?>['scale', j(sx), j(sy)]);
  @override
  void transform(Float64List matrix4) => calls.add(<Object?>['transform', js(matrix4)]);
  @override
  void clipRect(Rect r, {ui.ClipOp clipOp = ui.ClipOp.intersect, bool doAntiAlias = true}) =>
      calls.add(<Object?>['clipRect', rect(r), clipOp.index, doAntiAlias]);
  @override
  void clipPath(Path path, {bool doAntiAlias = true}) =>
      calls.add(<Object?>['clipPath', rect(path.getBounds()), doAntiAlias]);
  @override
  void drawImageRect(ui.Image image, Rect src, Rect dst, Paint paint) => calls.add(
      <Object?>['drawImageRect', <int>[image.width, image.height], rect(src), rect(dst), paintOf(paint)]);
  @override
  void drawImageNine(ui.Image image, Rect center, Rect dst, Paint paint) => calls.add(
      <Object?>['drawImageNine', <int>[image.width, image.height], rect(center), rect(dst), paintOf(paint)]);
  @override
  void drawPath(Path path, Paint paint) =>
      calls.add(<Object?>['drawPath', rect(path.getBounds()), paintOf(paint)]);
  @override
  void drawRect(Rect r, Paint paint) => calls.add(<Object?>['drawRect', rect(r), paintOf(paint)]);
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

// An image provider with a stable toString/== whose image is available
// synchronously.
// A decoration that lerps with nothing.
class PlainDecoration extends Decoration {
  const PlainDecoration();
  @override
  BoxPainter createBoxPainter([VoidCallback? onChanged]) => throw UnimplementedError();
}

class TestProvider extends ImageProvider<TestProvider> {
  const TestProvider(this.name, this.image, [this.scale = 1.0, this.label]);
  final String name;
  final ui.Image image;
  final double scale;
  final String? label;
  @override
  Future<TestProvider> obtainKey(ImageConfiguration configuration) => SynchronousFuture<TestProvider>(this);
  @override
  ImageStreamCompleter loadImage(TestProvider key, ImageDecoderCallback decode) =>
      OneFrameImageStreamCompleter(
          SynchronousFuture<ImageInfo>(ImageInfo(image: image.clone(), scale: scale, debugLabel: label)));
  @override
  bool operator ==(Object other) => other is TestProvider && other.name == name;
  @override
  int get hashCode => name.hashCode;
  @override
  String toString() => 'TestProvider($name)';
}

List<Gradient> gradients() => <Gradient>[
      const LinearGradient(colors: <Color>[Color(0xFF112233), Color(0x80FFEEDD)]),
      const LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment(0.7, 1.0),
        colors: <Color>[Color(0x00FFFFFF), Color(0x11777777), Color(0x44444444)],
        stops: <double>[0.1, 0.35, 0.9],
        tileMode: TileMode.mirror,
        transform: GradientRotation(0.7),
      ),
      const LinearGradient(
        begin: AlignmentDirectional.topStart,
        end: AlignmentDirectional(0.25, 0.5),
        colors: <Color>[Color(0xFF00FF00), Color(0xFF0000FF), Color(0xFFFF0000), Color(0x33333333)],
        tileMode: TileMode.repeated,
      ),
      const RadialGradient(colors: <Color>[Color(0xFFFFFF00), Color(0xFF8000FF)]),
      const RadialGradient(
        center: Alignment(0.3, -0.2),
        radius: 0.8,
        colors: <Color>[Color(0xFF123456), Color(0xFF654321), Color(0x80ABCDEF)],
        stops: <double>[0.0, 0.6, 1.0],
        tileMode: TileMode.decal,
        focal: Alignment(-0.1, 0.4),
        focalRadius: 0.15,
        transform: GradientRotation(-1.2),
      ),
      const RadialGradient(
        center: AlignmentDirectional.centerEnd,
        radius: 2.5,
        colors: <Color>[Color(0xFFFFFFFF), Color(0xFF000000)],
        stops: <double>[0.25, 0.75],
        focal: AlignmentDirectional(-0.5, 0.0),
      ),
      const SweepGradient(colors: <Color>[Color(0xFFFF0000), Color(0xFF00FF00), Color(0xFFFF0000)]),
      const SweepGradient(
        center: Alignment(-0.4, 0.6),
        startAngle: 0.5,
        endAngle: 4.0,
        colors: <Color>[Color(0xFF0F0F0F), Color(0xF0F0F0F0)],
        stops: <double>[0.3, 0.7],
        tileMode: TileMode.repeated,
        transform: GradientRotation(math.pi / 3),
      ),
    ];

List<BoxShadow> shadows() => const <BoxShadow>[
      BoxShadow(),
      BoxShadow(blurRadius: 4.0),
      BoxShadow(color: Color(0x80FF0000), offset: Offset(3, -2.5), blurRadius: 7.3, spreadRadius: 1.25),
      BoxShadow(color: Color(0xFF00FF00), offset: Offset(-1, 4), blurRadius: 0.5, blurStyle: BlurStyle.outer),
      BoxShadow(color: Color(0x33123456), spreadRadius: -2.0, blurStyle: BlurStyle.solid),
      BoxShadow(blurRadius: 12.0, blurStyle: BlurStyle.inner),
    ];

Object? shadow(BoxShadow? s) => s == null
    ? null
    : <Object?>[col(s.color), j(s.offset.dx), j(s.offset.dy), j(s.blurRadius), j(s.spreadRadius), s.blurStyle.index, '$s'];

const List<double> ts = <double>[-0.5, 0.0, 0.25, 0.5, 0.75, 1.0, 1.5];

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('painting_gradients', () async {
    final Map<String, Object?> out = <String, Object?>{};

    // ---- Gradients ----
    final List<Gradient> gs = gradients();
    final List<Gradient?> withNull = <Gradient?>[null, ...gs];
    out['gradient.describe'] = gs.map(grad).toList();
    out['gradient.toString'] = gs.map((Gradient g) => '$g').toList();
    out['gradient.equal'] = <Object?>[
      for (final Gradient a in gs)
        for (final Gradient b in gradients()) a == b,
    ];
    final List<Object?> lerps = <Object?>[];
    for (final Gradient? a in withNull) {
      for (final Gradient? b in withNull) {
        for (final double t in ts) {
          lerps.add(errOr(() => grad(Gradient.lerp(a, b, t))));
        }
      }
    }
    out['gradient.lerp'] = lerps;
    final List<LinearGradient?> lin = <LinearGradient?>[null, gs[0] as LinearGradient, gs[1] as LinearGradient, gs[2] as LinearGradient];
    final List<RadialGradient?> rad = <RadialGradient?>[null, gs[3] as RadialGradient, gs[4] as RadialGradient, gs[5] as RadialGradient];
    final List<SweepGradient?> swp = <SweepGradient?>[null, gs[6] as SweepGradient, gs[7] as SweepGradient];
    out['gradient.lerp.typed'] = <Object?>[
      for (final LinearGradient? a in lin)
        for (final LinearGradient? b in lin)
          for (final double t in ts) errOr(() => grad(LinearGradient.lerp(a, b, t))),
      for (final RadialGradient? a in rad)
        for (final RadialGradient? b in rad)
          for (final double t in ts) errOr(() => grad(RadialGradient.lerp(a, b, t))),
      for (final SweepGradient? a in swp)
        for (final SweepGradient? b in swp)
          for (final double t in ts) errOr(() => grad(SweepGradient.lerp(a, b, t))),
    ];
    out['gradient.scale'] = <Object?>[
      for (final Gradient g in gs)
        for (final double f in <double>[-0.5, 0.0, 0.3, 1.0, 2.0]) errOr(() => grad(g.scale(f))),
    ];
    out['gradient.withOpacity'] = <Object?>[
      for (final Gradient g in gs)
        for (final double o in <double>[0.0, 0.37, 1.0, 1.5]) errOr(() => grad(g.withOpacity(o))),
    ];
    // Duplicate stops (the sample's first/last-stop and lastIndexWhere rules)
    // and gradients that differ only in their stops.
    const List<Color> c4 = <Color>[Color(0xFFFF0000), Color(0xFF00FF00), Color(0xFF0000FF), Color(0x80FFFFFF)];
    final List<Gradient> dup = <Gradient>[
      const LinearGradient(colors: c4, stops: <double>[0.25, 0.25, 0.6, 1.0]),
      const LinearGradient(colors: c4, stops: <double>[0.0, 0.5, 0.5, 1.0]),
      const RadialGradient(colors: c4, stops: <double>[0.0, 0.4, 0.75, 0.75]),
      const SweepGradient(colors: c4, stops: <double>[0.3, 0.3, 0.3, 0.9]),
    ];
    out['gradient.duplicateStops'] = <Object?>[
      for (final Gradient a in dup)
        for (final Gradient b in <Gradient>[...dup, gs[1], gs[4], gs[7]])
          for (final double t in <double>[0.0, 0.3, 1.0]) errOr(() => grad(Gradient.lerp(a, b, t))),
    ];
    const List<Color> c2 = <Color>[Color(0xFF000000), Color(0xFFFFFFFF)];
    out['gradient.equalStops'] = <bool>[
      const LinearGradient(colors: c2) == const LinearGradient(colors: c2, stops: <double>[0, 1]),
      const LinearGradient(colors: c2, stops: <double>[0, 0.5]) == const LinearGradient(colors: c2, stops: <double>[0, 1]),
      const RadialGradient(colors: c2) == const RadialGradient(colors: c2, stops: <double>[0, 1]),
      const SweepGradient(colors: c2, stops: <double>[0, 0.5]) == const SweepGradient(colors: c2, stops: <double>[0, 1]),
      const LinearGradient(colors: c2, stops: <double>[0, 1]) == const LinearGradient(colors: c2, stops: <double>[0, 1]),
    ];
    out['gradient.fromColor'] = <Object?>[
      for (final Gradient g in gs) grad(g.fromColor(const Color(0xFF0080FF))),
    ];
    const Rect shaderRect = Rect.fromLTWH(1.0, 2.0, 30.0, 40.0);
    out['gradient.createShader'] = <Object?>[
      for (final Gradient g in gs)
        for (final TextDirection? d in <TextDirection?>[null, TextDirection.ltr, TextDirection.rtl])
          err(() => g.createShader(shaderRect, textDirection: d)),
      err(() => const LinearGradient(colors: <Color>[Color(0xFF000000), Color(0xFF000000), Color(0xFF000000)],
          stops: <double>[0, 1]).createShader(shaderRect)),
      err(() => const SweepGradient(colors: <Color>[Color(0xFF000000), Color(0xFF000000)], startAngle: 2, endAngle: 1)
          .createShader(shaderRect)),
    ];
    final List<Object?> rotations = <Object?>[];
    for (final double r in <double>[0.0, 0.5, math.pi / 4, math.pi / 2, -1.3, 3.0, 7.5]) {
      for (final Rect b in <Rect>[
        const Rect.fromLTWH(0, 0, 300, 400),
        const Rect.fromLTRB(-10.5, 3.25, 17.75, 99.0),
        Rect.zero,
      ]) {
        rotations.add(js(GradientRotation(r).transform(b, textDirection: TextDirection.rtl).storage));
      }
      rotations.add('${GradientRotation(r)}');
    }
    rotations.add(<bool>[
      const GradientRotation(1.0) == const GradientRotation(1.0),
      const GradientRotation(1.0) == const GradientRotation(1.5),
    ]);
    out['gradientRotation'] = rotations;

    // ---- BoxShadow ----
    final List<BoxShadow> ss = shadows();
    final List<BoxShadow?> ssNull = <BoxShadow?>[null, ...ss];
    out['boxShadow.lerp'] = <Object?>[
      for (final BoxShadow? a in ssNull)
        for (final BoxShadow? b in ssNull)
          for (final double t in ts) errOr(() => shadow(BoxShadow.lerp(a, b, t))),
    ];
    final List<List<BoxShadow>?> lists = <List<BoxShadow>?>[
      null,
      <BoxShadow>[],
      <BoxShadow>[ss[1]],
      <BoxShadow>[ss[2], ss[3]],
      <BoxShadow>[ss[5], ss[4], ss[0]],
    ];
    out['boxShadow.lerpList'] = <Object?>[
      for (final List<BoxShadow>? a in lists)
        for (final List<BoxShadow>? b in lists)
          for (final double t in <double>[0.0, 0.3, 1.0])
            errOr(() => BoxShadow.lerpList(a, b, t)?.map(shadow).toList()),
    ];
    out['boxShadow.scale'] = <Object?>[
      for (final BoxShadow s in ss)
        for (final double f in <double>[0.0, 0.5, 3.0, -1.0]) errOr(() => shadow(s.scale(f))),
    ];
    out['boxShadow.copyWith'] = <Object?>[
      for (final BoxShadow s in ss) ...<Object?>[
        shadow(s.copyWith()),
        shadow(s.copyWith(color: const Color(0xFF445566), blurRadius: 2.0)),
        shadow(s.copyWith(offset: const Offset(1, 1), spreadRadius: 9.0, blurStyle: BlurStyle.outer)),
      ],
    ];
    out['boxShadow.equal'] = <Object?>[
      for (final BoxShadow a in ss)
        for (final BoxShadow b in shadows()) a == b,
    ];
    final List<Object?> toPaint = <Object?>[];
    for (final bool disable in <bool>[false, true]) {
      debugDisableShadows = disable;
      for (final BoxShadow s in ss) {
        toPaint.add(<Object?>[paintOf(s.toPaint()), j(s.blurSigma)]);
      }
    }
    debugDisableShadows = true;
    out['boxShadow.toPaint'] = toPaint;

    // ---- DecorationImage ----
    final ui.Image img100 = await createTestImage(width: 100, height: 100, cache: false);
    final ui.Image img37x23 = await createTestImage(width: 37, height: 23, cache: false);
    final ui.Image img512 = await createTestImage(width: 512, height: 300, cache: false);
    final TestProvider pa = TestProvider('a', img100);
    final TestProvider pb = TestProvider('b', img37x23, 2.0, 'b.png');
    final List<DecorationImage> dis = <DecorationImage>[
      DecorationImage(image: pa),
      DecorationImage(image: pb, fit: BoxFit.cover, alignment: Alignment.topRight, opacity: 0.5),
      DecorationImage(
        image: pa,
        placeholder: pb,
        colorFilter: const ColorFilter.mode(Color(0xFF00FF00), BlendMode.src),
        fit: BoxFit.fill,
        alignment: AlignmentDirectional.bottomStart,
        centerSlice: const Rect.fromLTRB(10, 20, 40, 60),
        repeat: ImageRepeat.repeatX,
        matchTextDirection: true,
        scale: 1.25,
        opacity: 0.75,
        filterQuality: FilterQuality.none,
        invertColors: true,
        isAntiAlias: true,
      ),
      DecorationImage(image: pb, fit: BoxFit.scaleDown, repeat: ImageRepeat.repeat, scale: 0.04),
      DecorationImage(image: pa, fit: BoxFit.fill),
      DecorationImage(image: pa, fit: BoxFit.scaleDown, centerSlice: const Rect.fromLTWH(1, 2, 3, 4)),
    ];
    out['decorationImage.toString'] = dis.map((DecorationImage d) => '$d').toList();
    out['decorationImage.equal'] = <Object?>[
      for (final DecorationImage a in dis)
        for (final DecorationImage b in dis) a == b,
      DecorationImage(image: pa) == DecorationImage(image: TestProvider('a', img37x23)),
    ];
    final List<DecorationImage?> disNull = <DecorationImage?>[null, dis[0], dis[1], dis[2]];
    out['decorationImage.lerp'] = <Object?>[
      for (final DecorationImage? a in disNull)
        for (final DecorationImage? b in disNull)
          for (final double t in <double>[0.0, 0.4, 1.0]) '${DecorationImage.lerp(a, b, t)}',
    ];

    // ---- paintImage ----
    final List<Object?> pis = <Object?>[];
    final List<BoxFit?> fits = <BoxFit?>[null, ...BoxFit.values];
    for (final (Rect, ui.Image) ri in <(Rect, ui.Image)>[
      (const Rect.fromLTWH(10.0, 20.0, 250.0, 120.0), img100),
      (const Rect.fromLTRB(-3.5, 7.25, 41.0, 99.5), img37x23),
    ]) {
      for (final BoxFit? fit in fits) {
        for (final Alignment a in <Alignment>[Alignment.center, Alignment.bottomRight, const Alignment(-0.3, 0.7)]) {
          for (final ImageRepeat repeat in ImageRepeat.values) {
            for (final bool flip in <bool>[false, true]) {
              for (final double scale in <double>[1.0, 2.5]) {
                for (final Rect? slice in <Rect?>[null, const Rect.fromLTRB(5, 6, 20, 18)]) {
                  final RecCanvas c = RecCanvas();
                  final String e = err(() => paintImage(
                        canvas: c,
                        rect: ri.$1,
                        image: ri.$2,
                        fit: fit,
                        alignment: a,
                        repeat: repeat,
                        flipHorizontally: flip,
                        scale: scale,
                        centerSlice: slice,
                      ));
                  pis.add(<Object?>[e, c.calls]);
                }
              }
            }
          }
        }
      }
    }
    out['paintImage'] = pis;
    final List<Object?> pis2 = <Object?>[];
    for (final (double, ColorFilter?, bool, FilterQuality, bool, BlendMode) o
        in <(double, ColorFilter?, bool, FilterQuality, bool, BlendMode)>[
      (0.5, null, false, FilterQuality.medium, false, BlendMode.srcOver),
      (1.7, const ColorFilter.mode(Color(0xFF0000FF), BlendMode.multiply), true, FilterQuality.none, true, BlendMode.plus),
      (-0.2, const ColorFilter.linearToSrgbGamma(), false, FilterQuality.high, true, BlendMode.dstIn),
      (0.123, null, true, FilterQuality.low, false, BlendMode.modulate),
    ]) {
      final RecCanvas c = RecCanvas();
      paintImage(
        canvas: c,
        rect: const Rect.fromLTWH(0, 0, 64, 48),
        image: img37x23,
        opacity: o.$1,
        colorFilter: o.$2,
        invertColors: o.$3,
        filterQuality: o.$4,
        isAntiAlias: o.$5,
        blendMode: o.$6,
      );
      pis2.add(c.calls);
    }
    final RecCanvas emptyRect = RecCanvas();
    paintImage(canvas: emptyRect, rect: const Rect.fromLTWH(5, 5, 0, 10), image: img100);
    pis2.add(emptyRect.calls);
    out['paintImage.paint'] = pis2;

    // The oversized-image path.
    debugInvertOversizedImages = true;
    final FlutterExceptionHandler? oldOnError = FlutterError.onError;
    final List<String> messages = <String>[];
    FlutterError.onError = (FlutterErrorDetails d) =>
        messages.add('${d.exceptionAsString()}|${d.library}|${d.context}');
    final List<Object?> inverted = <Object?>[];
    for (final (Rect, String?, BoxFit) o in <(Rect, String?, BoxFit)>[
      (const Rect.fromLTWH(50.0, 50.0, 10.0, 7.0), 'Big', BoxFit.fill),
      (const Rect.fromLTWH(-5.0, 3.0, 20.5, 11.25), null, BoxFit.contain),
      (const Rect.fromLTWH(0, 0, 400, 290), 'Fits', BoxFit.fill),
      (const Rect.fromLTWH(0, 0, 90, 90), 'Small', BoxFit.cover),
      (const Rect.fromLTWH(0, 0, 130, 120), 'Allowance', BoxFit.fill),
    ]) {
      final RecCanvas c = RecCanvas();
      paintImage(canvas: c, rect: o.$1, image: img512, debugImageLabel: o.$2, fit: o.$3, flipHorizontally: true);
      inverted.add(c.calls);
    }
    inverted.add(messages);
    FlutterError.onError = oldOnError;
    debugInvertOversizedImages = false;
    out['paintImage.inverted'] = inverted;

    // DecorationImage painters.
    final List<Object?> painters = <Object?>[];
    for (final DecorationImage d in dis) {
      for (final TextDirection? dir in <TextDirection?>[TextDirection.ltr, TextDirection.rtl]) {
        final RecCanvas c = RecCanvas();
        final DecorationImagePainter p = d.createPainter(() {});
        final String e = err(() => p.paint(c, const Rect.fromLTWH(2, 3, 120, 80), Path()..addRect(const Rect.fromLTWH(0, 0, 50, 60)),
            ImageConfiguration(textDirection: dir)));
        painters.add(<Object?>[e, c.calls]);
        p.dispose();
      }
    }
    for (final (DecorationImage?, DecorationImage?, double) l in <(DecorationImage?, DecorationImage?, double)>[
      (dis[0], dis[1], 0.3),
      (null, dis[1], 0.6),
      (dis[1], null, 0.25),
      (dis[3], dis[0], 0.5),
    ]) {
      final RecCanvas c = RecCanvas();
      final DecorationImagePainter p = DecorationImage.lerp(l.$1, l.$2, l.$3)!.createPainter(() {});
      p.paint(c, const Rect.fromLTWH(0, 0, 64, 64), null, const ImageConfiguration(textDirection: TextDirection.ltr),
          blend: 0.8, blendMode: BlendMode.multiply);
      painters.add(c.calls);
      p.dispose();
    }
    out['decorationImage.paint'] = painters;

    // ---- Decoration.lerp / FlutterLogoDecoration ----
    const FlutterLogoDecoration l0 = FlutterLogoDecoration();
    const FlutterLogoDecoration l1 = FlutterLogoDecoration(
        textColor: Color(0xFFD4F144), style: FlutterLogoStyle.stacked, margin: EdgeInsets.all(10.0));
    const FlutterLogoDecoration l2 = FlutterLogoDecoration(
        textColor: Color(0xFF81D4FA), style: FlutterLogoStyle.horizontal, margin: EdgeInsets.fromLTRB(1, 2, 3, 4));
    final List<FlutterLogoDecoration?> logos = <FlutterLogoDecoration?>[null, l0, l1, l2];
    Object? logo(Decoration? d) {
      if (d is! FlutterLogoDecoration) {
        return d == null ? null : d.runtimeType.toString();
      }
      return <Object?>[
        col(d.textColor),
        d.style.index,
        js(<double>[d.margin.left, d.margin.top, d.margin.right, d.margin.bottom]),
        d.isComplex,
        '$d',
      ];
    }

    out['flutterLogo.lerp'] = <Object?>[
      for (final FlutterLogoDecoration? a in logos)
        for (final FlutterLogoDecoration? b in logos)
          for (final double t in ts) ...<Object?>[
            errOr(() => logo(FlutterLogoDecoration.lerp(a, b, t))),
            errOr(() => logo(Decoration.lerp(a, b, t))),
          ],
    ];
    out['decoration.lerp.unrelated'] = <Object?>[
      for (final double t in ts) ...<Object?>[
        logo(Decoration.lerp(l1, const PlainDecoration(), t)),
        logo(Decoration.lerp(const PlainDecoration(), l2, t)),
      ],
    ];
    out['flutterLogo.equal'] = <Object?>[
      FlutterLogoDecoration.lerp(null, l0, 0.5) == l0,
      FlutterLogoDecoration.lerp(null, l0, 0.5) == FlutterLogoDecoration.lerp(null, l0, 0.5),
      FlutterLogoDecoration.lerp(l1, null, 0.25) == l1,
      for (final FlutterLogoDecoration? a in logos)
        for (final FlutterLogoDecoration? b in logos) a == b,
    ];
    // The mark only (position 0): the styles with the "Flutter" label need
    // text layout.
    final List<Object?> logoPaint = <Object?>[];
    for (final FlutterLogoDecoration d in <FlutterLogoDecoration>[
      l0,
      const FlutterLogoDecoration(margin: EdgeInsets.fromLTRB(3, 5, 7, 11)),
      FlutterLogoDecoration.lerp(null, l0, 0.4)!,
      FlutterLogoDecoration.lerp(l0, null, 0.9)!,
    ]) {
      for (final (Offset, Size) g in <(Offset, Size)>[
        (Offset.zero, const Size(202, 202)),
        (const Offset(10.5, -3), const Size(64, 40)),
        (const Offset(1, 2), const Size(17.3, 99.9)),
        (Offset.zero, const Size(5, 5)),
      ]) {
        final RecCanvas c = RecCanvas();
        d.createBoxPainter().paint(c, g.$1, ImageConfiguration(size: g.$2));
        logoPaint.add(c.calls);
      }
    }
    out['flutterLogo.paint'] = logoPaint;
    out['flutterLogo.misc'] = <Object?>[
      rect(l0.getClipPath(const Rect.fromLTWH(1, 2, 3, 4), TextDirection.ltr).getBounds()),
      l0.hitTest(const Size(10, 10), const Offset(100, 100)),
      '${l0.padding}',
      l0.toStringShort(),
      FlutterLogoStyle.values.map((FlutterLogoStyle s) => '$s').toList(),
      ImageRepeat.values.map((ImageRepeat s) => '$s').toList(),
    ];

    File('build/painting_gradients.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(jsonEncode(out));
  });
}
