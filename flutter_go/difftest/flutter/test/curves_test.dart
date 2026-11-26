// Reference side of flutter-go's curves differential test: samples every
// Curves constant and every curve class of package:flutter/animation.dart
// (several parameter sets each) at the same t values, plus toString,
// CatmullRomSpline points, generateSamples output, findInverse,
// samplingSeed, validateControlPoints reasons and assertion messages.
// Output: build/curves.json. Doubles are JSON numbers (Dart's shortest
// round-trip form) or "NaN"/"Infinity"/"-Infinity" strings.
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;

import 'package:flutter/animation.dart';
import 'package:flutter_test/flutter_test.dart';

Object d(double v) => v.isFinite ? v : v.toString();

List<double> get ts => <double>[
  for (int i = 0; i <= 97; i++) i / 97,
  1e-9, 1e-6, 0.001, 0.01, 0.1, 0.2, 0.25, 0.3, 1 / 3, 0.4, 0.5, 0.6, 2 / 3, 0.7, 0.75,
  0.8, 0.9, 0.99, 0.999, 1 - 1e-6, 1 - 1e-9, 5e-324, 0.36363636363636365, 0.7272727272727273,
  0.9090909090909091, 0.9545454545454546,
];

// A closed curve (first point == last point) forces generateSamples to
// subdivide; it overrides samplingSeed.
class Circle extends Curve2D {
  const Circle(this.seed);
  final int seed;
  @override
  Offset transformInternal(double t) =>
      Offset(math.cos(t * 2 * math.pi), math.sin(t * 2 * math.pi));
  @override
  int get samplingSeed => seed;
}

// A periodic open curve with the default samplingSeed (0).
class Wave extends Curve2D {
  const Wave();
  @override
  Offset transformInternal(double t) => Offset(t, math.sin(t * 10) * 0.5);
}

const List<Offset> splinePoints = <Offset>[
  Offset.zero,
  Offset(0.01, 0.25),
  Offset(0.2, 0.25),
  Offset(0.33, 0.25),
  Offset(0.5, 1.0),
  Offset(0.66, 0.75),
  Offset(1.0, 1.0),
];

const List<Offset> curvePoints = <Offset>[
  Offset(0.2, 0.25),
  Offset(0.33, 0.25),
  Offset(0.5, 1.0),
  Offset(0.8, 0.75),
];

Map<String, Curve Function()> curves = <String, Curve Function()>{
  'linear': () => Curves.linear,
  'decelerate': () => Curves.decelerate,
  'fastLinearToSlowEaseIn': () => Curves.fastLinearToSlowEaseIn,
  'fastEaseInToSlowEaseOut': () => Curves.fastEaseInToSlowEaseOut,
  'ease': () => Curves.ease,
  'easeIn': () => Curves.easeIn,
  'easeInToLinear': () => Curves.easeInToLinear,
  'easeInSine': () => Curves.easeInSine,
  'easeInQuad': () => Curves.easeInQuad,
  'easeInCubic': () => Curves.easeInCubic,
  'easeInQuart': () => Curves.easeInQuart,
  'easeInQuint': () => Curves.easeInQuint,
  'easeInExpo': () => Curves.easeInExpo,
  'easeInCirc': () => Curves.easeInCirc,
  'easeInBack': () => Curves.easeInBack,
  'easeOut': () => Curves.easeOut,
  'linearToEaseOut': () => Curves.linearToEaseOut,
  'easeOutSine': () => Curves.easeOutSine,
  'easeOutQuad': () => Curves.easeOutQuad,
  'easeOutCubic': () => Curves.easeOutCubic,
  'easeOutQuart': () => Curves.easeOutQuart,
  'easeOutQuint': () => Curves.easeOutQuint,
  'easeOutExpo': () => Curves.easeOutExpo,
  'easeOutCirc': () => Curves.easeOutCirc,
  'easeOutBack': () => Curves.easeOutBack,
  'easeInOut': () => Curves.easeInOut,
  'easeInOutSine': () => Curves.easeInOutSine,
  'easeInOutQuad': () => Curves.easeInOutQuad,
  'easeInOutCubic': () => Curves.easeInOutCubic,
  'easeInOutCubicEmphasized': () => Curves.easeInOutCubicEmphasized,
  'easeInOutQuart': () => Curves.easeInOutQuart,
  'easeInOutQuint': () => Curves.easeInOutQuint,
  'easeInOutExpo': () => Curves.easeInOutExpo,
  'easeInOutCirc': () => Curves.easeInOutCirc,
  'easeInOutBack': () => Curves.easeInOutBack,
  'fastOutSlowIn': () => Curves.fastOutSlowIn,
  'slowMiddle': () => Curves.slowMiddle,
  'bounceIn': () => Curves.bounceIn,
  'bounceOut': () => Curves.bounceOut,
  'bounceInOut': () => Curves.bounceInOut,
  'elasticIn': () => Curves.elasticIn,
  'elasticOut': () => Curves.elasticOut,
  'elasticInOut': () => Curves.elasticInOut,
  'sawTooth1': () => const SawTooth(1),
  'sawTooth3': () => const SawTooth(3),
  'sawTooth7': () => const SawTooth(7),
  'interval': () => const Interval(0.25, 0.75),
  'intervalFull': () => const Interval(0.0, 1.0),
  'intervalEaseOut': () => const Interval(0.1, 0.9, curve: Curves.easeOut),
  'intervalNested': () => const Interval(0.2, 0.6, curve: Interval(0.25, 0.75, curve: Curves.bounceOut)),
  'intervalElastic': () => const Interval(0.0, 0.5, curve: Curves.elasticOut),
  'intervalDegenerate': () => const Interval(0.5, 0.5),
  'intervalThird': () => const Interval(1 / 3, 2 / 3, curve: Curves.easeInOutBack),
  'split': () => const Split(0.3),
  'splitCurves': () => const Split(0.5, beginCurve: Curves.easeIn, endCurve: Curves.bounceOut),
  'splitZero': () => const Split(0.0),
  'splitOne': () => const Split(1.0, beginCurve: Curves.elasticIn),
  'splitThird': () => const Split(1 / 3, beginCurve: Curves.decelerate, endCurve: Curves.elasticInOut),
  'threshold': () => const Threshold(0.25),
  'thresholdZero': () => const Threshold(0.0),
  'thresholdOne': () => const Threshold(1.0),
  'thresholdHalf': () => const Threshold(0.5),
  'cubic': () => const Cubic(0.1, 0.2, 0.3, 0.4),
  'cubicOvershoot': () => const Cubic(0.0, 1.5, 1.0, -0.5),
  'cubicSteep': () => const Cubic(0.9, 0.0, 0.1, 1.0),
  'threePointCubic': () => const ThreePointCubic(
    Offset(0.1, 0.05),
    Offset(0.2, 0.3),
    Offset(0.4, 0.6),
    Offset(0.6, 0.8),
    Offset(0.8, 0.95),
  ),
  'elasticIn03': () => const ElasticInCurve(0.3),
  'elasticIn075': () => const ElasticInCurve(0.75),
  'elasticOut02': () => const ElasticOutCurve(0.2),
  'elasticOut1': () => const ElasticOutCurve(1.0),
  'elasticInOut05': () => const ElasticInOutCurve(0.5),
  'elasticInOut015': () => const ElasticInOutCurve(0.15),
  'flippedEaseIn': () => const FlippedCurve(Curves.easeIn),
  'flippedBounceOut': () => Curves.bounceOut.flipped,
  'flippedTwice': () => const FlippedCurve(FlippedCurve(Curves.elasticIn)),
  'flippedInterval': () => const Interval(0.2, 0.8, curve: Curves.fastOutSlowIn).flipped,
  'flippedEmphasized': () => Curves.easeInOutCubicEmphasized.flipped,
  'catmullRom': () => CatmullRomCurve(curvePoints),
  'catmullRomPrecompute': () => CatmullRomCurve.precompute(curvePoints),
  'catmullRomTension': () => CatmullRomCurve(curvePoints, tension: 0.5),
  'catmullRomS': () => CatmullRomCurve(const <Offset>[Offset(0.25, 0.1), Offset(0.5, 0.5), Offset(0.75, 0.9)]),
  'catmullRomTwo': () => CatmullRomCurve.precompute(
    const <Offset>[Offset(0.3, 0.6), Offset(0.6, 0.3)],
    tension: 0.25,
  ),
};

Map<String, Curve2D Function()> curves2D = <String, Curve2D Function()>{
  'splineHandles': () => CatmullRomSpline(
    splinePoints,
    startHandle: const Offset(0.0, -0.3),
    endHandle: const Offset(1.3, 1.3),
  ),
  'splineDefault': () => CatmullRomSpline(splinePoints),
  'splineTension': () => CatmullRomSpline(splinePoints, tension: 0.3),
  'splinePrecompute': () => CatmullRomSpline.precompute(splinePoints, tension: 0.7, endHandle: const Offset(2, 0)),
  'splineLoop': () => CatmullRomSpline(const <Offset>[
    Offset(0, 0), Offset(1, 0), Offset(1, 1), Offset(0, 1), Offset(0, 0.0001), Offset(0.5, -0.5),
  ], tension: 0.1),
  'splineTensionOne': () => CatmullRomSpline(splinePoints, tension: 1.0),
  'circle': () => const Circle(0),
  'circleSeed': () => const Circle(12345),
  'wave': () => const Wave(),
};

Object? errorMessage(void Function() f) {
  try {
    f();
  } on AssertionError catch (e) {
    return 'AssertionError: ${e.message}';
  } catch (e) {
    return '${e.runtimeType}: $e';
  }
  return null;
}

void main() {
  test('curves', () {
    final Map<String, Object> out = <String, Object>{};
    out['ts'] = <Object>[for (final double t in ts) d(t)];
    final Map<String, Object> one = <String, Object>{};
    for (final MapEntry<String, Curve Function()> e in curves.entries) {
      final Curve c = e.value();
      one[e.key] = <String, Object>{
        'values': <Object>[for (final double t in ts) d(c.transform(t))],
        'string': c.toString().replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH'),
        'flipped': c.flipped.toString().replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH'),
      };
    }
    out['curves'] = one;

    final Map<String, Object> two = <String, Object>{};
    for (final MapEntry<String, Curve2D Function()> e in curves2D.entries) {
      final Curve2D c = e.value();
      final Map<String, Object> r = <String, Object>{};
      r['values'] = <Object>[
        for (final double t in ts) <Object>[d(c.transform(t).dx), d(c.transform(t).dy)],
      ];
      List<Object> samples(Iterable<Curve2DSample> s) => <Object>[
        for (final Curve2DSample x in s) <Object>[d(x.t), d(x.value.dx), d(x.value.dy), x.toString()],
      ];
      r['samples'] = samples(c.generateSamples());
      r['samplesTol'] = samples(c.generateSamples(tolerance: 1e-4));
      r['samplesRange'] = samples(c.generateSamples(start: 0.2, end: 0.7, tolerance: 1e-6));
      r['inverse'] = <Object>[
        for (int i = 0; i <= 20; i++) d(c.findInverse(i / 20)),
      ];
      // ignore: invalid_use_of_protected_member
      r['seed'] = c.samplingSeed;
      r['string'] = c.toString();
      two[e.key] = r;
    }
    out['curves2D'] = two;

    final Map<String, Object?> errors = <String, Object?>{
      'cubicNaN': errorMessage(() => Curves.ease.transformInternal(double.nan)),
      'cubicInternalLow': Curves.ease.transformInternal(-1.0),
      'cubicInternalHigh': Curves.ease.transformInternal(2.0),
      'transformHigh': errorMessage(() => Curves.ease.transform(1.5)),
      'transformNaN': errorMessage(() => Curves.ease.transform(double.nan)),
      'transform2D': errorMessage(() => CatmullRomSpline(splinePoints).transform(-0.5)),
      'splitHigh': errorMessage(() => const Split(0.5).transform(1.5)),
      'splitBadRange': errorMessage(() => const Split(1.5).transform(0.5)),
      'intervalBad': errorMessage(() => const Interval(0.6, 0.4).transform(0.5)),
      'thresholdBad': errorMessage(() => const Threshold(1.5).transform(0.5)),
      'splineTension': errorMessage(() => CatmullRomSpline(splinePoints, tension: 1.5)),
      'splineNegTension': errorMessage(() => CatmullRomSpline(splinePoints, tension: -0.5)),
      'splineFew': errorMessage(() => CatmullRomSpline(const <Offset>[Offset.zero])),
      'splineInfinite': errorMessage(
        () => CatmullRomSpline.precompute(const <Offset>[
          Offset.zero, Offset(1, 1), Offset(double.infinity, 2), Offset(3, 3),
        ]),
      ),
      'splineStartHandle': errorMessage(
        () => CatmullRomSpline.precompute(splinePoints, startHandle: const Offset(double.nan, 0)),
      ),
      'splineEndHandle': errorMessage(
        () => CatmullRomSpline.precompute(splinePoints, endHandle: const Offset(0, double.infinity)),
      ),
      'generateBadRange': errorMessage(() => const Wave().generateSamples(start: 0.5, end: 0.5)),
      'splineNaNSeed': errorMessage(() => CatmullRomSpline(const <Offset>[
        Offset.zero, Offset.zero, Offset.zero, Offset.zero,
      ]).generateSamples()),
      'curveFew': errorMessage(() => CatmullRomCurve(const <Offset>[Offset(0.5, 0.5)])),
      'curveMulti': errorMessage(() => CatmullRomCurve(const <Offset>[Offset(0.5, 0.05), Offset(0.5, 0.95)])),
      'curveRange': errorMessage(() => CatmullRomCurve.precompute(const <Offset>[Offset(0.2, 0.25), Offset(1.01, 0.25)])),
      'curveY0': errorMessage(() => CatmullRomCurve(const <Offset>[
        Offset(0.05, 0.50), Offset(0.50, 0.50), Offset(0.75, 0.75),
      ])),
      'curveY1': errorMessage(() => CatmullRomCurve(const <Offset>[
        Offset(0.25, 0.25), Offset(0.50, 0.50), Offset(0.95, 0.51),
      ])),
    };
    out['errors'] = errors;

    final List<List<Offset>?> validate = <List<Offset>?>[
      null,
      const <Offset>[],
      const <Offset>[Offset(0.5, 0.5)],
      curvePoints,
      const <Offset>[Offset(0.2, 0.25), Offset(0.01, 0.25)],
      const <Offset>[Offset(0.2, 0.25), Offset(1.01, 0.25)],
      const <Offset>[Offset(0.0, 0.25), Offset(0.5, 0.25)],
      const <Offset>[Offset(0.05, 0.50), Offset(0.50, 0.50), Offset(0.75, 0.75)],
      const <Offset>[Offset(0.25, 0.25), Offset(0.50, 0.50), Offset(0.95, 0.51)],
      const <Offset>[Offset(0.5, 0.05), Offset(0.5, 0.95)],
      const <Offset>[Offset(0.1, 0.9), Offset(0.15, 0.1), Offset(0.2, 0.9), Offset(0.9, 0.1)],
      const <Offset>[Offset(0.3, 0.6), Offset(0.6, 0.3)],
      // First sample's y between 1e-3 and 1e-2; X slightly below 0; X slightly above 1.
      const <Offset>[Offset(0.001, 0.01), Offset(0.5, 0.5)],
      const <Offset>[Offset(0.031, -1.0), Offset(0.5, 0.5)],
      const <Offset>[Offset(0.5, 0.5), Offset(0.951, -1.0)],
      const <Offset>[Offset(0.5, 0.5), Offset(1.0, 0.5)],
    ];
    final List<Object> validations = <Object>[];
    for (final List<Offset>? points in validate) {
      for (final double tension in <double>[0.0, 0.5, 1.0]) {
        final List<String> reasons = <String>[];
        final bool withReasons = CatmullRomCurve.validateControlPoints(points, tension: tension, reasons: reasons);
        final bool without = CatmullRomCurve.validateControlPoints(points, tension: tension);
        validations.add(<Object>[withReasons, without, reasons]);
      }
    }
    out['validate'] = validations;

    final Directory dir = Directory('build')..createSync(recursive: true);
    File('${dir.path}/curves.json').writeAsStringSync(jsonEncode(out));
  });
}
