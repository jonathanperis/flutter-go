// Reference side of flutter-go's velocity tracker differential test:
// VelocityTracker, IOSScrollViewFlingVelocityTracker and
// MacOSScrollViewFlingVelocityTracker estimates for many sample sets (the
// gestures tests' recorded flings, generated motions with gaps, horizons,
// equal time stamps, ring wrap-around), the "pointer stopped" rule on the
// test's fake sampling clock, Velocity/VelocityEstimate operations and
// toString, and LeastSquaresSolver fits of degrees 0-3 (weighted,
// degenerate). Doubles use Dart's toString: the Go twin
// (../velocity_tracker_test.go) is compared exactly.
// Output: build/velocity_tracker.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;

import 'package:flutter/gestures.dart';
import 'package:flutter_test/flutter_test.dart';

import 'velocity_tracker_data.dart';

final List<String> lines = <String>[];
void log(String s) => lines.add(s);

String off(Offset o) => '${o.dx} ${o.dy}';

String est(VelocityEstimate? e) => e == null
    ? 'null'
    : '${off(e.pixelsPerSecond)} ${e.confidence} ${e.duration.inMicroseconds} ${off(e.offset)} | $e';

typedef Sample = (int us, double x, double y);

List<VelocityTracker> trackers() => <VelocityTracker>[
  VelocityTracker.withKind(PointerDeviceKind.touch),
  IOSScrollViewFlingVelocityTracker(PointerDeviceKind.touch),
  MacOSScrollViewFlingVelocityTracker(PointerDeviceKind.mouse),
];

// Generated motions: (name, samples).
List<(String, List<Sample>)> motions() {
  final out = <(String, List<Sample>)>[];
  List<Sample> gen(int n, int Function(int) t, double Function(int) x, double Function(int) y) =>
      <Sample>[for (int i = 0; i < n; i++) (t(i), x(i), y(i))];
  out.add(('single', <Sample>[(1000, 10.0, 20.0)]));
  out.add(('two', <Sample>[(1000, 10.0, 20.0), (9000, 13.5, 27.25)]));
  out.add(('three linear', gen(3, (i) => 1000 + 8000 * i, (i) => 10.0 * i, (i) => -5.0 * i)));
  out.add(('uniform 8ms', gen(12, (i) => 16000 + 8000 * i, (i) => 3.0 * i, (i) => 100 + 7.5 * i)));
  out.add(('accelerating', gen(15, (i) => 5000 + 7000 * i, (i) => 0.5 * i * i, (i) => 2.0 * i * i - i)));
  out.add(('decelerating', gen(15, (i) => 7000 * i, (i) => 100 - (14 - i) * (14 - i) * 0.3, (i) => 50.0 * math.sqrt(i))));
  out.add(('irregular times', gen(14, (i) => 6000 * i + (i * i * 731) % 5000, (i) => 1.7 * i + math.sin(i), (i) => 400 - 9.1 * i)));
  out.add(('gap over 40ms', <Sample>[
    ...gen(5, (i) => 8000 * i, (i) => 4.0 * i, (i) => 2.0 * i),
    ...gen(5, (i) => 32000 + 41000 + 8000 * i, (i) => 20 + 6.0 * i, (i) => 10 + 1.0 * i),
  ]));
  out.add(('gap exactly 40ms', <Sample>[
    ...gen(5, (i) => 8000 * i, (i) => 4.0 * i, (i) => 2.0 * i),
    ...gen(5, (i) => 32000 + 40000 + 8000 * i, (i) => 20 + 6.0 * i, (i) => 10 + 1.0 * i),
  ]));
  out.add(('past horizon', gen(20, (i) => 9000 * i, (i) => 3.0 * i, (i) => -2.0 * i)));
  out.add(('horizon edge', gen(11, (i) => 10000 * i, (i) => 1.0 * i * i, (i) => 3.0 * i)));
  out.add(('wrap 25', gen(25, (i) => 4000 * i, (i) => 2.5 * i + (i % 3), (i) => 7.0 * i - (i % 4))));
  out.add(('wrap 47', gen(47, (i) => 3000 * i, (i) => 100 * math.cos(i / 7), (i) => 100 * math.sin(i / 5))));
  out.add(('equal times', <Sample>[(1000, 1.0, 1.0), (1000, 2.0, 3.0), (9000, 4.0, 4.0), (9000, 6.0, 9.0), (17000, 7.0, 10.0)]));
  out.add(('stationary', gen(10, (i) => 8000 * i, (i) => 50.0, (i) => 60.0)));
  out.add(('vertical only', gen(10, (i) => 8000 * i, (i) => 0.0, (i) => -13.25 * i)));
  out.add(('large values', gen(10, (i) => 1000000000 + 8000 * i, (i) => 1e6 + 123.456 * i, (i) => -1e5 - 0.001 * i * i)));
  out.add(('tiny steps', gen(10, (i) => 8000 * i, (i) => 1e-7 * i, (i) => 3e-9 * i * i)));
  out.add(('fast fling', gen(8, (i) => 4000 * i, (i) => 0.0, (i) => 300 - 42.0 * i)));
  out.add(('micro times', gen(9, (i) => 1 + i * 1337, (i) => i * 0.75, (i) => i * -1.25)));
  return out;
}

List<(String, List<Sample>)> recorded() {
  final out = <(String, List<Sample>)>[];
  for (final (String name, List<PointerEvent> data) in <(String, List<PointerEvent>)>[
    ('velocityEventData', velocityEventData),
    ('interruptedVelocityEventData', interruptedVelocityEventData),
  ]) {
    var current = <Sample>[];
    var n = 0;
    for (final event in data) {
      if (event is PointerDownEvent) {
        current = <Sample>[];
      }
      if (event is PointerDownEvent || event is PointerMoveEvent) {
        current.add((event.timeStamp.inMicroseconds, event.position.dx, event.position.dy));
      }
      if (event is PointerUpEvent) {
        out.add(('$name ${n++}', current));
      }
    }
  }
  return out;
}

void main() {
  testWidgets('velocity tracker', (WidgetTester tester) async {
    for (final (String name, List<Sample> samples) in <(String, List<Sample>)>[...recorded(), ...motions()]) {
      log('motion $name (${samples.length})');
      for (final tracker in trackers()) {
        // Estimates after each sample, then the velocity.
        final ests = <String>[];
        for (final (int us, double x, double y) in samples) {
          tracker.addPosition(Duration(microseconds: us), Offset(x, y));
          ests.add(est(tracker.getVelocityEstimate()));
        }
        log('  ${tracker.runtimeType}');
        for (final e in ests) {
          log('    $e');
        }
        final Velocity v = tracker.getVelocity();
        log('    velocity ${off(v.pixelsPerSecond)} $v');
        for (final (double lo, double hi) in <(double, double)>[(0, 8000), (50, 8000), (0, 100), (500, 600), (0, 0)]) {
          log('    clamp $lo $hi ${off(v.clampMagnitude(lo, hi).pixelsPerSecond)}');
        }
      }
    }

    // The stopped rule: time passes on the fake sampling clock.
    for (final int waitUs in <int>[0, 39999, 40000, 40999, 41000, 41001, 100000]) {
      final ts = trackers();
      for (final tracker in ts) {
        tracker.addPosition(Duration.zero, Offset.zero);
        tracker.addPosition(const Duration(milliseconds: 8), const Offset(5, 10));
        tracker.addPosition(const Duration(milliseconds: 16), const Offset(12, 20));
      }
      await tester.pump(Duration(microseconds: waitUs));
      for (final tracker in ts) {
        log('stopped $waitUs ${tracker.runtimeType} ${est(tracker.getVelocityEstimate())}');
      }
      // A new sample restarts the stopwatch.
      for (final tracker in ts) {
        tracker.addPosition(const Duration(milliseconds: 24), const Offset(20, 31));
        log('restarted $waitUs ${tracker.runtimeType} ${est(tracker.getVelocityEstimate())}');
      }
    }

    // Empty trackers.
    log('empty ${est(VelocityTracker.withKind(PointerDeviceKind.touch).getVelocityEstimate())} '
        '${VelocityTracker.withKind(PointerDeviceKind.touch).getVelocity()}');

    // Velocity operations and toString.
    const a = Velocity(pixelsPerSecond: Offset(3.25, -4.75));
    const b = Velocity(pixelsPerSecond: Offset(-0.05, 0.15));
    log('ops ${off((-a).pixelsPerSecond)} ${off((a + b).pixelsPerSecond)} ${off((a - b).pixelsPerSecond)} '
        '${a == const Velocity(pixelsPerSecond: Offset(3.25, -4.75))} ${a == b} $a $b ${Velocity.zero}');
    for (final Velocity v in <Velocity>[
      a, b, const Velocity(pixelsPerSecond: Offset(3000, 4000)), const Velocity(pixelsPerSecond: Offset(1e-3, -2e-3)),
      const Velocity(pixelsPerSecond: Offset(0.05, -0.05)), const Velocity(pixelsPerSecond: Offset(-0.25, 0.35)),
    ]) {
      for (final (double lo, double hi) in <(double, double)>[(0, 5), (5, 10), (1, 1), (0.001, 0.002), (6000, 9000)]) {
        log('clamp ${off(v.pixelsPerSecond)} $lo $hi ${off(v.clampMagnitude(lo, hi).pixelsPerSecond)}');
      }
    }
    log(const VelocityEstimate(
      pixelsPerSecond: Offset(1.25, -3.35),
      confidence: 0.45,
      duration: Duration(microseconds: 12345),
      offset: Offset(7.5, 8.25),
    ).toString());

    // LeastSquaresSolver.
    final xs = <double>[for (int i = 0; i < 9; i++) -i * 7.5 + (i % 2) * 0.3];
    final sets = <(String, List<double>, List<double>, List<double>)>[
      ('flat', xs, <double>[for (int i = 0; i < 9; i++) 1.0], <double>[for (int i = 0; i < 9; i++) 1.0]),
      ('line', xs, <double>[for (final x in xs) 2.5 * x - 3], <double>[for (int i = 0; i < 9; i++) 1.0]),
      ('noisy', xs, <double>[for (int i = 0; i < 9; i++) 0.3 * xs[i] * xs[i] + math.sin(i * 1.3) * 4], <double>[for (int i = 0; i < 9; i++) 1.0]),
      ('weighted', xs, <double>[for (int i = 0; i < 9; i++) math.cos(i / 2) * 10], <double>[for (int i = 0; i < 9; i++) 0.5 + i / 3]),
      ('two points', <double>[0, 1], <double>[3, 5], <double>[1, 1]),
      ('same x', <double>[2, 2, 2, 2], <double>[1, 2, 3, 4], <double>[1, 1, 1, 1]),
      ('zero weights', <double>[0, 1, 2], <double>[1, 2, 3], <double>[0, 0, 0]),
      ('one point', <double>[4], <double>[9], <double>[2]),
      ('cubic', <double>[-3, -2, -1, 0, 1, 2, 3, 4], <double>[-27, -8, -1, 0, 1, 8, 27, 64.5], <double>[1, 1, 1, 1, 1, 1, 1, 1]),
    ];
    for (final (String name, List<double> x, List<double> y, List<double> w) in sets) {
      for (int degree = 0; degree <= 3; degree++) {
        final PolynomialFit? fit = LeastSquaresSolver(x, y, w).solve(degree);
        log('lsq $name $degree ${fit == null ? 'null' : '${fit.coefficients.join(' ')} ${fit.confidence} $fit'}');
      }
    }
  });

  tearDownAll(() {
    File('build/velocity_tracker.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': lines}));
  });
}
