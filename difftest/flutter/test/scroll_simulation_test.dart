// Reference side of flutter-go's scroll-simulation differential test: samples
// BouncingScrollSimulation and ClampingScrollSimulation directly (no widgets)
// every 1/60 s for 4 s. Output: build/scroll_simulation.json as
// {scene: [[t, x, dx, isDone ? 1 : 0], ...]}.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

final SpringDescription ios = SpringDescription.withDampingRatio(mass: 0.5, stiffness: 100, ratio: 1.1);
final SpringDescription mac = SpringDescription.withDampingRatio(mass: 0.3, stiffness: 75, ratio: 1.3);
const Tolerance dpr3 = Tolerance(velocity: 1 / (0.050 * 3), distance: 1 / 3);

Map<String, Simulation Function()> scenes = <String, Simulation Function()>{
  'bounce_within': () => BouncingScrollSimulation(position: 300, velocity: 800, leadingExtent: 0, trailingExtent: 1000, spring: ios, tolerance: dpr3),
  'bounce_cap_over': () => BouncingScrollSimulation(position: 900, velocity: 12000, leadingExtent: 0, trailingExtent: 1000, spring: ios),
  'bounce_cap_under': () => BouncingScrollSimulation(position: 100, velocity: -12000, leadingExtent: 0, trailingExtent: 1000, spring: ios),
  'bounce_under_start': () => BouncingScrollSimulation(position: -0.5, velocity: 0, leadingExtent: 0, trailingExtent: 1000, spring: ios, tolerance: dpr3),
  'bounce_over_start': () => BouncingScrollSimulation(position: 1000.5, velocity: 300, leadingExtent: 0, trailingExtent: 1000, spring: ios),
  'bounce_mac_fast': () => BouncingScrollSimulation(position: 500, velocity: 4000, leadingExtent: 0, trailingExtent: 1000, spring: mac, constantDeceleration: 1400, tolerance: dpr3),
  'bounce_empty_range': () => BouncingScrollSimulation(position: 0, velocity: 2000, leadingExtent: 0, trailingExtent: 0, spring: ios),
  'clamping_fast': () => ClampingScrollSimulation(position: 0, velocity: 8000),
  'clamping_up': () => ClampingScrollSimulation(position: 100, velocity: -3000, tolerance: dpr3),
  'clamping_slow': () => ClampingScrollSimulation(position: 50, velocity: 0.5),
  'clamping_friction': () => ClampingScrollSimulation(position: 0, velocity: 2500, friction: 0.03),
};

void main() {
  test('scroll simulations', () {
    final Map<String, List<List<double>>> out = <String, List<List<double>>>{};
    for (final MapEntry<String, Simulation Function()> e in scenes.entries) {
      final Simulation s = e.value();
      out[e.key] = <List<double>>[
        for (int i = 0; i <= 240; i++)
          <double>[i / 60, s.x(i / 60), s.dx(i / 60), if (s.isDone(i / 60)) 1 else 0],
      ];
    }
    File('build/scroll_simulation.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
