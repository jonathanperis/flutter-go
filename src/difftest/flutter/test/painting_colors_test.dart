// Reference side of flutter-go's painting colors/matrix_utils differential
// test: HSVColor/HSLColor conversions, setters, lerps and toString;
// ColorSwatch; ColorProperty; MatrixUtils (transformRect,
// inverseTransformRect, transformPoint, getAsTranslation/getAsScale,
// createCylindricalProjectionTransform, forceToPoint, multiplyInPlace),
// debugDescribeTransform and TransformProperty; paintZigZag's recorded
// calls. Output: build/painting_colors.json as {case: value}; doubles are
// compared exactly (non-finite ones as their toString).
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;
import 'dart:ui' show PathMetric, Tangent;

import 'package:flutter/foundation.dart';
import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vector_math/vector_math_64.dart';

Object j(double v) => v.isFinite ? v : '$v';
List<Object> js(Iterable<double> vs) => vs.map(j).toList();
List<Object> hsv(HSVColor c) => js(<double>[c.alpha, c.hue, c.saturation, c.value]);
List<Object> hsl(HSLColor c) => js(<double>[c.alpha, c.hue, c.saturation, c.lightness]);
List<Object> rect(Rect r) => js(<double>[r.left, r.top, r.right, r.bottom]);
List<Object> col(Color c) => <Object>[c.a, c.r, c.g, c.b, c.colorSpace.index];

String err(void Function() f) {
  try {
    f();
  } catch (e) {
    return e is AssertionError ? 'assertion' : '$e';
  }
  return 'no error';
}

// Records drawPath/translate/rotate calls (paintZigZag's output).
class _ZigZagCanvas implements Canvas {
  final List<Object> calls = <Object>[];
  @override
  void save() => calls.add('save');
  @override
  void restore() => calls.add('restore');
  @override
  void translate(double dx, double dy) => calls.add(<Object>['translate', j(dx), j(dy)]);
  @override
  void rotate(double radians) => calls.add(<Object>['rotate', j(radians)]);
  @override
  void drawPath(Path path, Paint paint) {
    final List<Object> metrics = <Object>[];
    for (final PathMetric m in path.computeMetrics()) {
      metrics.add(<Object>[j(m.length), m.isClosed]);
      for (double d = 0; d <= m.length; d += m.length / 8) {
        final Tangent? t = m.getTangentForOffset(d);
        metrics.add(js(<double>[t!.position.dx, t.position.dy]));
      }
    }
    calls.add(<Object>['drawPath', rect(path.getBounds()), metrics]);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

List<Matrix4> matrices() {
  final Matrix4 persp = Matrix4.identity()..setEntry(3, 0, 0.002)..setEntry(3, 1, -0.001);
  final Matrix4 skew = Matrix4.identity()..setEntry(0, 1, 0.3)..setEntry(1, 0, -0.7);
  final Matrix4 general = Matrix4(
      2, 1, 0.5, 0.001, -1, 3, 0.25, 0.002, 0.3, -0.7, 4, 0.05, 5, -2, 1, 1.5);
  final Matrix4 rotated = Matrix4.identity()
    ..rotateZ(0.3)
    ..rotateX(0.2)
    ..rotateY(-0.4)
    ..translate(10.0, 20.0)
    ..scale(1.5, 0.5);
  return <Matrix4>[
    Matrix4.identity(),
    Matrix4.translationValues(10.5, -3.25, 0),
    Matrix4.translationValues(1, 2, 3),
    Matrix4.diagonal3Values(2, 2, 1),
    Matrix4.diagonal3Values(-1.5, 0.75, 1),
    Matrix4.rotationZ(0.3),
    Matrix4.rotationZ(math.pi / 2),
    Matrix4.rotationX(1.1),
    Matrix4.rotationY(-0.6),
    persp,
    skew,
    general,
    rotated,
    MatrixUtils.createCylindricalProjectionTransform(radius: 100, angle: math.pi / 3),
    MatrixUtils.createCylindricalProjectionTransform(radius: 10, angle: math.pi / 8, perspective: 0.3),
    MatrixUtils.createCylindricalProjectionTransform(
        radius: 50, angle: -0.7, perspective: 0.05, orientation: Axis.horizontal),
    MatrixUtils.forceToPoint(const Offset(20, -30)),
    Matrix4.zero(),
  ];
}

const List<Rect> rects = <Rect>[
  Rect.fromLTRB(10, 20, 30, 40),
  Rect.fromLTRB(-5.5, 3.25, 100.125, 7),
  Rect.fromLTRB(0, 0, 0, 0),
  Rect.fromLTRB(30, 40, 10, 20),
  Rect.fromLTRB(0.1, 0.2, 0.3, 0.7),
  Rect.fromLTRB(-1e6, -2e6, 3e6, 4e6),
  Rect.fromLTRB(0, -1.7976931348623157e+308, 800, 1.7976931348623157e+308),
  Rect.fromLTRB(double.negativeInfinity, 0, 10, 10),
  Rect.fromLTRB(0, 0, double.infinity, 2),
  Rect.largest,
];

void main() {
  test('painting_colors', () {
    final Map<String, Object?> out = <String, Object?>{};

    // ---- HSV/HSL from Color over a grid ----
    final List<Object> fromColor = <Object>[];
    for (int r = 0; r < 256; r += 17) {
      for (int g = 0; g < 256; g += 17) {
        for (int b = 0; b < 256; b += 17) {
          final Color c = Color.fromARGB(255, r, g, b);
          final HSVColor v = HSVColor.fromColor(c);
          final HSLColor l = HSLColor.fromColor(c);
          fromColor.add(<Object>[...hsv(v), ...hsl(l), v.toColor().value, l.toColor().value]);
        }
      }
    }
    out['fromColor.grid'] = fromColor;
    final List<Object> odd = <Object>[];
    for (final int v in <int>[
      0x00000000, 0x80FF0000, 0x01020304, 0xFF7F7F80, 0xFFFE0000, 0xFF00FE01, 0xC0123456,
      0x7F808080, 0xFFFF3398, 0xFF00FFFF, 0xFFBFBF00, 0xFF0D0E0F, 0xFFFFFFFE, 0xFF010000,
      0x33A1B2C3, 0xFFEB3F7B,
    ]) {
      final HSVColor a = HSVColor.fromColor(Color(v));
      final HSLColor b = HSLColor.fromColor(Color(v));
      odd.add(<Object>[...hsv(a), ...hsl(b), a.toColor().value, b.toColor().value]);
    }
    out['fromColor.odd'] = odd;
    // Non-8-bit colors (Color.from) go through the 8-bit accessors.
    out['fromColor.float'] = <Object>[
      for (final Color c in <Color>[
        const Color.from(alpha: 0.5, red: 0.1234, green: 0.5678, blue: 0.9),
        const Color.from(alpha: 1, red: 0.333, green: 0.333, blue: 0.334),
      ]) <Object>[...hsv(HSVColor.fromColor(c)), ...hsl(HSLColor.fromColor(c))],
    ];

    // ---- toColor over an HSV/HSL grid ----
    const List<double> units = <double>[0, 0.1, 0.25, 1 / 3, 0.5, 0.77, 0.999, 1];
    final List<Object> toColor = <Object>[];
    for (double h = 0; h <= 360; h += 7.5) {
      for (final double s in units) {
        for (final double x in units) {
          toColor.add(<int>[
            HSVColor.fromAHSV(0.6, h, s, x).toColor().value,
            HSLColor.fromAHSL(0.6, h, s, x).toColor().value,
          ]);
        }
      }
    }
    out['toColor.grid'] = toColor;
    out['toColor.hues'] = <Object>[
      for (final double h in <double>[59.999999, 60, 60.5, 119.9999999999, 120.5, 179.5, 180.5, 239.99, 240.5, 299.0000001, 300.5, 359.9999, 360, 1e-9])
        <int>[HSVColor.fromAHSV(1, h, 0.8, 0.9).toColor().value, HSLColor.fromAHSL(1, h, 0.8, 0.4).toColor().value],
    ];
    out['toColor.alpha'] = <Object>[
      for (final double a in <double>[0, 0.001, 0.0019, 0.002, 0.5, 0.501, 0.998, 1])
        <int>[HSVColor.fromAHSV(a, 10, 0.5, 0.5).toColor().value, HSLColor.fromAHSL(a, 10, 0.5, 0.5).toColor().value],
    ];

    // ---- with*, toString, ranges ----
    const HSVColor v0 = HSVColor.fromAHSV(0.7, 28.0, 0.3, 0.6);
    const HSLColor l0 = HSLColor.fromAHSL(0.7, 28.0, 0.3, 0.6);
    out['with'] = <Object>[
      hsv(v0.withAlpha(0.25)), hsv(v0.withHue(359.5)), hsv(v0.withSaturation(1)), hsv(v0.withValue(0)),
      hsl(l0.withAlpha(0.25)), hsl(l0.withHue(359.5)), hsl(l0.withSaturation(1)), hsl(l0.withLightness(0)),
    ];
    out['toString'] = <String>[
      '$v0', '$l0',
      '${const HSVColor.fromAHSV(1, 1e-7, 0.1 + 0.2, 1 / 3)}',
      '${const HSLColor.fromAHSL(0, 360, 1e-21, 0.5)}',
      '${HSVColor.fromColor(const Color(0xFF123456))}',
      '${HSLColor.fromColor(const Color(0xFF123456))}',
    ];
    out['asserts'] = <String>[
      err(() => HSVColor.fromAHSV(1.1, 0, 0, 0)),
      err(() => HSVColor.fromAHSV(1, 360.5, 0, 0)),
      err(() => HSLColor.fromAHSL(1, 0, -0.1, 0)),
      err(() => HSLColor.fromAHSL(1, 0, 0, 1.0000001)),
      err(() => HSVColor.fromAHSV(1, 0, 0, 1)),
    ];

    // ---- lerps ----
    const List<double> ts = <double>[-0.5, -0.1, 0, 0.1, 0.25, 0.5, 0.75, 0.9, 1, 1.1, 1.5, 2];
    final List<(HSVColor, HSVColor)> hsvPairs = <(HSVColor, HSVColor)>[
      (v0, const HSVColor.fromAHSV(0.3, 128.0, 0.7, 0.2)),
      (const HSVColor.fromAHSV(1, 300, 1, 1), const HSVColor.fromAHSV(1, 60, 1, 1)),
      (const HSVColor.fromAHSV(1, 10, 0.5, 0.5), const HSVColor.fromAHSV(0, 350, 0.1, 0.9)),
      (const HSVColor.fromAHSV(1, 0, 1, 1), const HSVColor.fromAHSV(1, 180, 1, 1)),
      (const HSVColor.fromAHSV(1, 180, 1, 1), const HSVColor.fromAHSV(1, 0, 1, 1)),
      (const HSVColor.fromAHSV(1, 0, 0, 0), const HSVColor.fromAHSV(1, 360, 1, 1)),
      (const HSVColor.fromAHSV(0.123, 33.3, 0.456, 0.789), const HSVColor.fromAHSV(0.987, 213.7, 0.654, 0.321)),
    ];
    out['hsv.lerp'] = <Object>[
      for (final (HSVColor, HSVColor) p in hsvPairs)
        for (final double t in ts) <Object>[...hsv(HSVColor.lerp(p.$1, p.$2, t)!), HSVColor.lerp(p.$1, p.$2, t)!.toColor().value],
    ];
    out['hsv.lerp.null'] = <Object>[
      for (final double t in <double>[0, 0.3, 0.5, 1]) ...<Object>[
        hsv(HSVColor.lerp(null, v0, t)!),
        hsv(HSVColor.lerp(v0, null, t)!),
      ],
      err(() => HSVColor.lerp(null, v0, 1.5)),
      err(() => HSVColor.lerp(v0, null, -0.5)),
    ];
    out['hsl.lerp'] = <Object>[
      for (final (HSVColor, HSVColor) p in hsvPairs)
        for (final double t in ts) () {
          final HSLColor a = HSLColor.fromAHSL(p.$1.alpha, p.$1.hue, p.$1.saturation, p.$1.value);
          final HSLColor b = HSLColor.fromAHSL(p.$2.alpha, p.$2.hue, p.$2.saturation, p.$2.value);
          final HSLColor r = HSLColor.lerp(a, b, t)!;
          return <Object>[...hsl(r), r.toColor().value];
        }(),
    ];
    out['hsl.lerp.null'] = <Object>[
      for (final double t in <double>[0, 0.3, 0.5, 1]) ...<Object>[
        hsl(HSLColor.lerp(null, l0, t)!),
        hsl(HSLColor.lerp(l0, null, t)!),
      ],
    ];

    // ---- ColorSwatch ----
    const ColorSwatch<int> swA = ColorSwatch<int>(0xFF102030, <int, Color>{
      50: Color(0xFFFFEEDD), 100: Color(0x80112233), 500: Color(0xFF102030),
    });
    const ColorSwatch<int> swB = ColorSwatch<int>(0x40FFFFFF, <int, Color>{
      100: Color(0xFF000000), 500: Color(0xFFABCDEF), 900: Color(0xFF010101),
    });
    out['swatch'] = <Object>[
      swA.value,
      col(swA),
      col(swA[100]!),
      '${swA[900]}',
      swA.keys.toList(),
      '$swA',
      swA == const ColorSwatch<int>(0xFF102030, <int, Color>{
        500: Color(0xFF102030), 50: Color(0xFFFFEEDD), 100: Color(0x80112233),
      }),
      swA == swB,
    ];
    out['swatch.lerp'] = <Object>[
      for (final double t in <double>[-0.2, 0, 0.3, 0.5, 1, 1.4]) ...<Object>[
        () {
          final ColorSwatch<int> s = ColorSwatch.lerp(swA, swB, t)!;
          return <Object>[col(s), s.keys.toList(), for (final int k in s.keys) col(s[k]!)];
        }(),
        () {
          final ColorSwatch<int> s = ColorSwatch.lerp(null, swB, t)!;
          return <Object>[col(s), s.keys.toList(), for (final int k in s.keys) col(s[k]!)];
        }(),
        () {
          final ColorSwatch<int> s = ColorSwatch.lerp(swA, null, t)!;
          return <Object>[col(s), s.keys.toList(), for (final int k in s.keys) col(s[k]!)];
        }(),
      ],
    ];

    // ---- ColorProperty ----
    final ColorProperty cp = ColorProperty('foo', const Color.fromARGB(10, 20, 30, 40));
    final ColorProperty cpNull = ColorProperty('bar', null);
    final ColorProperty cpDefault = ColorProperty(
      'baz', const Color(0xFF00FF00), defaultValue: const Color(0xFF00FF00), showName: false);
    out['colorProperty'] = <Object?>[
      '$cp', '$cpNull', '$cpDefault', cpDefault.level.name,
      cp.toJsonMap(const DiagnosticsSerializationDelegate()),
      cpNull.toJsonMap(const DiagnosticsSerializationDelegate()),
    ];

    // ---- MatrixUtils ----
    final List<Matrix4> ms = matrices();
    out['matrices'] = <Object>[for (final Matrix4 m in ms) js(m.storage)];
    out['transformRect'] = <Object>[
      for (final Matrix4 m in ms)
        for (final Rect r in rects) rect(MatrixUtils.transformRect(m, r)),
    ];
    out['inverseTransformRect'] = <Object>[
      for (final Matrix4 m in ms.take(ms.length - 2)) // the last two are singular
        for (final Rect r in rects.take(6)) rect(MatrixUtils.inverseTransformRect(m, r)),
    ];
    const List<Offset> points = <Offset>[
      Offset.zero, Offset(1, 1), Offset(-20, 30), Offset(-1.2344, 1422434.23), Offset(0.1, 0.7),
    ];
    out['transformPoint'] = <Object>[
      for (final Matrix4 m in ms)
        for (final Offset p in points) () {
          final Offset o = MatrixUtils.transformPoint(m, p);
          return js(<double>[o.dx, o.dy]);
        }(),
    ];
    out['getAs'] = <Object?>[
      for (final Matrix4 m in ms) <Object?>[
        () {
          final Offset? o = MatrixUtils.getAsTranslation(m);
          return o == null ? null : js(<double>[o.dx, o.dy]);
        }(),
        () {
          final double? s = MatrixUtils.getAsScale(m);
          return s == null ? null : j(s);
        }(),
        MatrixUtils.isIdentity(m),
      ],
    ];
    out['matrixEquals'] = <Object>[
      for (final Matrix4 a in ms) for (final Matrix4 b in ms) MatrixUtils.matrixEquals(a, b),
      MatrixUtils.matrixEquals(null, Matrix4.identity()),
      MatrixUtils.matrixEquals(ms[1], null),
    ];
    out['cylindrical'] = <Object>[
      for (final double radius in <double>[0, 1, 37.5, 1000000])
        for (final double angle in <double>[0, 0.1, math.pi / 3, -2.5, math.pi])
          for (final double perspective in <double>[0, 0.001, 0.3, 1])
            for (final Axis axis in Axis.values)
              js(MatrixUtils.createCylindricalProjectionTransform(
                radius: radius, angle: angle, perspective: perspective, orientation: axis).storage),
      err(() => MatrixUtils.createCylindricalProjectionTransform(radius: 1, angle: 1, perspective: 1.5)),
    ];
    out['forceToPoint'] = <Object>[
      js(MatrixUtils.forceToPoint(const Offset(20, -30)).storage),
      js(MatrixUtils.forceToPoint(const Offset(-0.5, 1e10)).storage),
    ];
    out['multiplyInPlace'] = <Object>[
      for (final Matrix4 a in ms.take(14))
        for (final Matrix4 b in ms.skip(5).take(9)) () {
          final Matrix4 bb = b.clone();
          MatrixUtils.multiplyInPlace(a, bb);
          return js(bb.storage);
        }(),
    ];
    out['debugDescribeTransform'] = <Object>[
      for (final Matrix4 m in ms) debugDescribeTransform(m),
      debugDescribeTransform(null),
    ];
    out['transformProperty'] = <Object>[
      '${TransformProperty('transform', ms[12])}',
      TransformProperty('transform', ms[12]).toString(parentConfiguration: singleLineTextConfiguration),
      '${TransformProperty('t', null)}',
      '${TransformProperty('t', Matrix4.identity(), showName: false, defaultValue: Matrix4.identity())}',
    ];

    // ---- paintZigZag ----
    final List<Object> zig = <Object>[];
    for (final (Offset, Offset, int, double) z in <(Offset, Offset, int, double)>[
      (const Offset(0, 0), const Offset(100, 0), 5, 3),
      (const Offset(10, 20), const Offset(-30, 70), 7, 2.5),
      (const Offset(1.5, 2.5), const Offset(1.5, 50), 1, 10),
      (const Offset(5, 5), const Offset(5, 5), 2, 1),
    ]) {
      final _ZigZagCanvas canvas = _ZigZagCanvas();
      paintZigZag(canvas, Paint(), z.$1, z.$2, z.$3, z.$4);
      zig.add(canvas.calls);
    }
    zig.add(err(() => paintZigZag(_ZigZagCanvas(), Paint(), Offset.zero, const Offset(1, 1), 0, 1)));
    out['paintZigZag'] = zig;

    File('build/painting_colors.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(jsonEncode(out));
  });
}
