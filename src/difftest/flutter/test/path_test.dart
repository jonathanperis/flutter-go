// Reference side of flutter-go's dart:ui Path differential test. The cases
// are data (../path_cases.json, shared with src/difftest/path_test.go):
// each path is built from a list of operations, then queried (bounds,
// contains grids under both fill types, metrics with tangents and
// extracted segments, transforms, shift). Every double is printed with
// Dart's toString, so the Go side must match bit for bit.
// Output: build/path.json as {case: "printed value"}.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

double n(Object? v) {
  if (v is String) {
    switch (v) {
      case 'inf':
        return double.infinity;
      case '-inf':
        return double.negativeInfinity;
      case 'neg0':
        return -0.0;
      default:
        return double.nan;
    }
  }
  return (v! as num).toDouble();
}

late Map<String, dynamic> spec;

Float64List? matrix(Object? name) =>
    name == null ? null : Float64List.fromList([for (final v in spec['matrices'][name] as List) n(v)]);

Path build(String name) {
  final p = Path();
  final ops = spec['paths'][name]['ops'] as List;
  for (final o in ops) {
    final op = o as List;
    double a(int i) => n(op[i]);
    switch (op[0] as String) {
      case 'M':
        p.moveTo(a(1), a(2));
      case 'm':
        p.relativeMoveTo(a(1), a(2));
      case 'L':
        p.lineTo(a(1), a(2));
      case 'l':
        p.relativeLineTo(a(1), a(2));
      case 'Q':
        p.quadraticBezierTo(a(1), a(2), a(3), a(4));
      case 'q':
        p.relativeQuadraticBezierTo(a(1), a(2), a(3), a(4));
      case 'C':
        p.cubicTo(a(1), a(2), a(3), a(4), a(5), a(6));
      case 'c':
        p.relativeCubicTo(a(1), a(2), a(3), a(4), a(5), a(6));
      case 'K':
        p.conicTo(a(1), a(2), a(3), a(4), a(5));
      case 'k':
        p.relativeConicTo(a(1), a(2), a(3), a(4), a(5));
      case 'arcTo':
        p.arcTo(Rect.fromLTRB(a(1), a(2), a(3), a(4)), a(5), a(6), op[7] == 1);
      case 'arcToPoint':
        p.arcToPoint(Offset(a(1), a(2)),
            radius: Radius.elliptical(a(3), a(4)), rotation: a(5), largeArc: op[6] == 1, clockwise: op[7] == 1);
      case 'relArcToPoint':
        p.relativeArcToPoint(Offset(a(1), a(2)),
            radius: Radius.elliptical(a(3), a(4)), rotation: a(5), largeArc: op[6] == 1, clockwise: op[7] == 1);
      case 'rect':
        p.addRect(Rect.fromLTRB(a(1), a(2), a(3), a(4)));
      case 'oval':
        p.addOval(Rect.fromLTRB(a(1), a(2), a(3), a(4)));
      case 'arc':
        p.addArc(Rect.fromLTRB(a(1), a(2), a(3), a(4)), a(5), a(6));
      case 'poly':
        p.addPolygon(<Offset>[for (int i = 2; i + 1 < op.length; i += 2) Offset(a(i), a(i + 1))], op[1] == 1);
      case 'rrect':
        p.addRRect(RRect.fromLTRBAndCorners(a(1), a(2), a(3), a(4),
            topLeft: Radius.elliptical(a(5), a(6)),
            topRight: Radius.elliptical(a(7), a(8)),
            bottomRight: Radius.elliptical(a(9), a(10)),
            bottomLeft: Radius.elliptical(a(11), a(12))));
      case 'rse':
        p.addRSuperellipse(RSuperellipse.fromLTRBAndCorners(a(1), a(2), a(3), a(4),
            topLeft: Radius.elliptical(a(5), a(6)),
            topRight: Radius.elliptical(a(7), a(8)),
            bottomRight: Radius.elliptical(a(9), a(10)),
            bottomLeft: Radius.elliptical(a(11), a(12))));
      case 'Z':
        p.close();
      case 'fill':
        p.fillType = PathFillType.values[op[1] as int];
      case 'reset':
        p.reset();
      case 'addPath':
        p.addPath(build(op[1] as String), Offset(a(2), a(3)), matrix4: matrix(op[4]));
      case 'extend':
        p.extendWithPath(build(op[1] as String), Offset(a(2), a(3)), matrix4: matrix(op[4]));
      default:
        throw ArgumentError('unknown op ${op[0]}');
    }
  }
  return p;
}

String d(double v) => '$v';
String rect(Rect r) => '${d(r.left)},${d(r.top)},${d(r.right)},${d(r.bottom)}';
String tangent(Tangent? t, {bool angle = true}) => t == null
    ? 'null'
    : '${d(t.position.dx)},${d(t.position.dy)},${d(t.vector.dx)},${d(t.vector.dy)}' +
        (angle ? ',${d(t.angle)}' : '');

String grid(Path p, List<Object?> g) {
  final double x0 = n(g[0]), y0 = n(g[1]), step = n(g[2]);
  final int nx = g[3]! as int, ny = g[4]! as int;
  return <String>[
    for (int j = 0; j < ny; j++)
      <String>[for (int i = 0; i < nx; i++) p.contains(Offset(x0 + i * step, y0 + j * step)) ? '1' : '0'].join(),
  ].join('|');
}

// Contains just off the outline: points at distances eps along the normal
// of 64 points per contour (the on-curve tolerance is 1/4096).
String probes(Path p, Path q) {
  const eps = <double>[0, 1e-4, -1e-4, 1 / 4096, -1 / 4096, 2.5e-4, -2.5e-4, 1e-3, -1e-3];
  final out = <String>[];
  for (final PathMetric m in p.computeMetrics()) {
    final sb = StringBuffer();
    for (int k = 0; k < 64; k++) {
      final Tangent? t = m.getTangentForOffset(m.length * k / 64);
      if (t == null) {
        sb.write('n');
        continue;
      }
      for (final double e in eps) {
        sb.write(q.contains(Offset(t.position.dx - t.vector.dy * e, t.position.dy + t.vector.dx * e)) ? '1' : '0');
      }
    }
    out.add(sb.toString());
  }
  return out.join('|');
}

String metrics(Path p, bool forceClosed) {
  final out = <String>[];
  for (final PathMetric m in p.computeMetrics(forceClosed: forceClosed)) {
    final double len = m.length;
    out.add('${d(len)},${m.isClosed},${m.contourIndex}');
    final ds = <double>[-1, 0, 1e-3, for (int k = 0; k <= 8; k++) len * k / 8, len + 1, double.nan];
    out.add(<String>[for (final double x in ds) tangent(m.getTangentForOffset(x))].join(';'));
    final se = <List<double>>[
      <double>[0, len / 3],
      <double>[len / 4, len * 3 / 4],
      <double>[len / 2, len],
      <double>[-5, len + 5],
      <double>[len / 2, len / 2],
      <double>[3, 2],
      <double>[len * 0.37, len * 0.38],
    ];
    for (final List<double> s in se) {
      for (final bool mv in <bool>[true, false]) {
        final Path q = m.extractPath(s[0], s[1], startWithMoveTo: mv);
        final parts = <String>[rect(q.getBounds())];
        for (final PathMetric qm in q.computeMetrics()) {
          parts.add('${d(qm.length)},${qm.isClosed},${tangent(qm.getTangentForOffset(qm.length / 2), angle: false)}');
        }
        out.add(parts.join(';'));
      }
    }
  }
  return out.join('|');
}

String summary(Path p) {
  final out = <String>[rect(p.getBounds())];
  for (final PathMetric m in p.computeMetrics()) {
    out.add('${d(m.length)},${m.isClosed}');
    out.add(<String>[for (int k = 0; k <= 4; k++) tangent(m.getTangentForOffset(m.length * k / 4))].join(';'));
  }
  return out.join('|');
}

void main() {
  test('path', () {
    spec = jsonDecode(File('../path_cases.json').readAsStringSync()) as Map<String, dynamic>;
    final out = <String, String>{};
    final paths = spec['paths'] as Map<String, dynamic>;
    for (final String name in paths.keys) {
      final Path p = build(name);
      out['$name.bounds'] = rect(p.getBounds());
      out['$name.fillType'] = '${p.fillType}';
      final grids = (paths[name]['grids'] ?? spec['defaultGrids']) as List;
      for (int i = 0; i < grids.length; i++) {
        final g = grids[i] as List<Object?>;
        out['$name.grid$i.nz'] = grid(Path.from(p)..fillType = PathFillType.nonZero, g);
        out['$name.grid$i.eo'] = grid(Path.from(p)..fillType = PathFillType.evenOdd, g);
      }
      out['$name.probes.nz'] = probes(p, Path.from(p)..fillType = PathFillType.nonZero);
      out['$name.probes.eo'] = probes(p, Path.from(p)..fillType = PathFillType.evenOdd);
      out['$name.grid.own'] = grid(p, grids[0] as List<Object?>);
      out['$name.metrics.open'] = metrics(p, false);
      out['$name.metrics.closed'] = metrics(p, true);
      out['$name.metrics.toString'] = '${p.computeMetrics()}';
      final skip = (paths[name]['skip'] ?? <Object?>[]) as List;
      for (final Object? t in spec['transforms'] as List) {
        if (skip.contains(t)) {
          continue;
        }
        out['$name.transform.$t'] = summary(p.transform(matrix(t)!));
      }
      out['$name.shift'] = summary(p.shift(const Offset(3.25, -1.5)));
    }
    File('build/path.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
