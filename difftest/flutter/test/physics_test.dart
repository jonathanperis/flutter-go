// Reference side of flutter-go's physics differential test: samples every
// package:flutter/physics simulation at fixed times. Output: build/physics.json
// as {scene: {"samples": [[t, x, dx, isDone ? 1 : 0], ...], "scalars": {...}}}.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/physics.dart';
import 'package:flutter_test/flutter_test.dart';

Map<String, Simulation Function()> scenes = <String, Simulation Function()>{
  'friction_pos': () => FrictionSimulation(0.135, 0, 1000),
  'friction_neg': () => FrictionSimulation(0.135, 100, -1000),
  'friction_decel': () => FrictionSimulation(0.135, 0, 3000, constantDeceleration: 1400),
  'friction_tol': () => FrictionSimulation(0.3, 100, 400, tolerance: const Tolerance(velocity: 1)),
  'friction_through': () => FrictionSimulation.through(10, 300, 600, 20),
  'bounded_max': () => BoundedFrictionSimulation(0.3, 100, 400, 50, 150),
  'bounded_min': () => BoundedFrictionSimulation(0.3, 100, -400, 50, 150),
  'gravity': () => GravitySimulation(-10, 0, 6, 10),
  'clamped': () => ClampedSimulation(GravitySimulation(9.81, 10, 0, 0), xMin: 20, xMax: 100, dxMin: 7, dxMax: 11),
  'spring_crit': () => SpringSimulation(SpringDescription.withDampingRatio(mass: 1, stiffness: 100), 0, 500, 0),
  'spring_under': () => SpringSimulation(SpringDescription.withDampingRatio(mass: 1, stiffness: 100, ratio: 0.25), 0, 300, 50),
  'spring_over': () => SpringSimulation(SpringDescription.withDampingRatio(mass: 1, stiffness: 100, ratio: 1.25), 0, 500, -200),
  'spring_snap': () => SpringSimulation(SpringDescription.withDampingRatio(mass: 1, stiffness: 400), 0, 1, 0,
      snapToEnd: true, tolerance: const Tolerance(distance: 0.1, velocity: 0.1)),
  'spring_bounce': () => SpringSimulation(SpringDescription.withDurationAndBounce(bounce: 0.3), 0, 1, 0),
  'scroll_spring': () => ScrollSpringSimulation(SpringDescription.withDampingRatio(mass: 0.5, stiffness: 100, ratio: 1.1), 0, 100, 0,
      tolerance: const Tolerance(distance: 0.5, velocity: 5)),
};

void main() {
  test('physics', () {
    final Map<String, Object> out = <String, Object>{};
    for (final MapEntry<String, Simulation Function()> e in scenes.entries) {
      final Simulation s = e.value();
      final List<List<double>> samples = <List<double>>[];
      for (int i = 0; i <= 140; i++) {
        final double t = i * 0.05;
        samples.add(<double>[t, s.x(t), s.dx(t), if (s.isDone(t)) 1 else 0]);
      }
      final Map<String, double> scalars = <String, double>{};
      if (s is FrictionSimulation) {
        scalars['finalX'] = s.finalX;
        for (final double at in <double>[0.1, 0.5, 2.0]) {
          final double x = s.x(at);
          if (x.isFinite) {
            scalars['timeAtX($at)'] = s.timeAtX(x);
          }
        }
      }
      if (s is SpringSimulation) {
        scalars['type'] = s.type.index.toDouble();
      }
      out[e.key] = <String, Object>{'samples': samples, 'scalars': scalars, 'string': s.toString()};
    }
    final SpringDescription d = SpringDescription.withDurationAndBounce(
        duration: const Duration(milliseconds: 350), bounce: -0.2);
    out['spring_description'] = <String, Object>{
      'scalars': <String, double>{
        'mass': d.mass, 'stiffness': d.stiffness, 'damping': d.damping,
        'bounce': d.bounce, 'durationMs': d.duration.inMilliseconds.toDouble(),
      },
    };
    File('build/physics.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
