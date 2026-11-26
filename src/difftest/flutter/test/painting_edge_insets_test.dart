// Reference side of flutter-go's edge_insets/border_radius differential
// test: EdgeInsetsGeometry and BorderRadiusGeometry values (all three types
// of each, mixed sums included) through every operator, lerp, resolve,
// toString, inflate/deflate, clamp, copyWith and factory. Output:
// build/painting_edge_insets.json as {case: value}; non-finite doubles are
// written as their toString.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

String err(Object? Function() f) {
  try {
    f();
  } catch (e) {
    return '$e';
  }
  return 'no error';
}

// Whether f throws (for asserts whose message is the file position).
bool throws(Object? Function() f) {
  try {
    f();
  } catch (e) {
    return true;
  }
  return false;
}

List<double> ltrb(EdgeInsets e) => <double>[e.left, e.top, e.right, e.bottom];
List<double> sz(Size s) => <double>[s.width, s.height];
List<double> rc(Rect r) => <double>[r.left, r.top, r.right, r.bottom];
List<double> rad(Radius r) => <double>[r.x, r.y];
List<double> rr(RRect r) => <double>[
  r.left, r.top, r.right, r.bottom,
  r.tlRadiusX, r.tlRadiusY, r.trRadiusX, r.trRadiusY,
  r.brRadiusX, r.brRadiusY, r.blRadiusX, r.blRadiusY,
];
List<double> rse(RSuperellipse r) => <double>[
  r.left, r.top, r.right, r.bottom,
  r.tlRadiusX, r.tlRadiusY, r.trRadiusX, r.trRadiusY,
  r.brRadiusX, r.brRadiusY, r.blRadiusX, r.blRadiusY,
];
List<double> corners(BorderRadius b) => <double>[
  ...rad(b.topLeft), ...rad(b.topRight), ...rad(b.bottomLeft), ...rad(b.bottomRight),
];

// Everything observable about an EdgeInsetsGeometry.
Object ei(EdgeInsetsGeometry? g) {
  if (g == null) {
    return 'null';
  }
  Object res(TextDirection? d) {
    try {
      return ltrb(g.resolve(d));
    } catch (e) {
      return '$e';
    }
  }

  return <Object>[
    '${g.runtimeType}',
    '$g',
    res(TextDirection.ltr),
    res(TextDirection.rtl),
    g.horizontal,
    g.vertical,
    g.isNonNegative,
    sz(g.collapsedSize),
    if (g is EdgeInsetsDirectional) <double>[g.start, g.top, g.end, g.bottom],
  ];
}

// Everything observable about a BorderRadiusGeometry.
Object br(BorderRadiusGeometry? g) {
  if (g == null) {
    return 'null';
  }
  Object res(TextDirection? d) {
    try {
      return corners(g.resolve(d));
    } catch (e) {
      return '$e';
    }
  }

  return <Object>[
    '${g.runtimeType}',
    '$g',
    res(TextDirection.ltr),
    res(TextDirection.rtl),
    if (g is BorderRadiusDirectional)
      <double>[...rad(g.topStart), ...rad(g.topEnd), ...rad(g.bottomStart), ...rad(g.bottomEnd)],
  ];
}

void main() {
  test('painting_edge_insets', () {
    final Map<String, Object?> out = <String, Object?>{};

    // ---- EdgeInsetsGeometry ----
    final Map<String, EdgeInsetsGeometry> insets = <String, EdgeInsetsGeometry>{
      'e1': const EdgeInsets.fromLTRB(5, 7.5, 11.25, 13),
      'e2': const EdgeInsets.all(3.3),
      'e3': const EdgeInsets.symmetric(vertical: 2, horizontal: -4.4),
      'e4': EdgeInsets.zero,
      'e5': const EdgeInsets.only(left: 0.1, bottom: 1e22),
      'e6': const EdgeInsets.fromLTRB(-0.0, 0.05, -0.05, 0.25),
      'e7': const EdgeInsets.all(double.infinity),
      'e8': const EdgeInsets.only(top: 1.25, right: 2.35),
      'd1': const EdgeInsetsDirectional.fromSTEB(1, 2, 3, 4),
      'd2': const EdgeInsetsDirectional.all(2.5),
      'd3': const EdgeInsetsDirectional.symmetric(horizontal: 7, vertical: -1),
      'd4': EdgeInsetsDirectional.zero,
      'd5': const EdgeInsetsDirectional.only(start: -0.0, end: 0.04),
      'd6': const EdgeInsetsDirectional.only(top: 0.15, bottom: -2.65),
      'm1': const EdgeInsetsDirectional.fromSTEB(1, 2, 3, 4).add(const EdgeInsets.fromLTRB(5, 7.5, 11.25, 13)),
      'm2': const EdgeInsets.all(3.3).subtract(const EdgeInsetsDirectional.symmetric(horizontal: 7, vertical: -1)),
      'm3': const EdgeInsets.fromLTRB(5, 7.5, 11.25, 13).add(EdgeInsetsDirectional.zero),
      'm4': const EdgeInsetsDirectional.fromSTEB(1, 2, 3, 4).add(EdgeInsets.zero),
      'm5': EdgeInsetsGeometry.infinity,
      'm6': const EdgeInsetsDirectional.only(start: 1).add(const EdgeInsets.only(top: 3)),
      'm7': const EdgeInsetsDirectional.only(start: 4).add(const EdgeInsets.only(left: 3)),
      'm8': EdgeInsets.zero.add(EdgeInsetsDirectional.zero),
    };
    final List<double> divisors = <double>[1.5, 4, -3, 2.5, 5];
    insets.forEach((String n, EdgeInsetsGeometry g) {
      out['ei.$n'] = ei(g);
      out['ei.$n.resolve.null'] = err(() => g.resolve(null));
      out['ei.$n.along'] = <double>[g.along(Axis.horizontal), g.along(Axis.vertical)];
      out['ei.$n.inflateSize'] = sz(g.inflateSize(const Size(100, 50)));
      out['ei.$n.deflateSize'] = sz(g.deflateSize(const Size(100, 50)));
      out['ei.$n.flipped'] = ei(g.flipped);
      out['ei.$n.neg'] = ei(-g);
      for (final double d in divisors) {
        out['ei.$n.mul.$d'] = ei(g * d);
        out['ei.$n.div.$d'] = ei(g / d);
        out['ei.$n.truncDiv.$d'] = (() {
          try {
            return ei(g ~/ d);
          } catch (e) {
            return '$e';
          }
        })();
        out['ei.$n.mod.$d'] = ei(g % d);
      }
      out['ei.$n.lerp.null.a'] = ei(EdgeInsetsGeometry.lerp(g, null, 0.25));
      out['ei.$n.lerp.null.b'] = ei(EdgeInsetsGeometry.lerp(null, g, 0.25));
      out['ei.$n.lerp.self'] = ei(EdgeInsetsGeometry.lerp(g, g, 0.5));
    });
    final List<String> pairNames = <String>['e1', 'e3', 'e6', 'd1', 'd3', 'm1', 'm2', 'e4', 'd4'];
    final List<double> ts = <double>[0, 0.25, 0.5, 1, 1.5, -0.5, 0.1];
    for (final String a in pairNames) {
      for (final String b in pairNames) {
        final EdgeInsetsGeometry ga = insets[a]!, gb = insets[b]!;
        out['ei.$a.add.$b'] = ei(ga.add(gb));
        out['ei.$a.subtract.$b'] = ei(ga.subtract(gb));
        out['ei.$a.eq.$b'] = ga == gb;
        for (final double t in ts) {
          out['ei.$a.lerp.$b.$t'] = ei(EdgeInsetsGeometry.lerp(ga, gb, t));
        }
      }
    }
    out['ei.lerp.null.null'] = ei(EdgeInsetsGeometry.lerp(null, null, 0.5));
    out['ei.lerp.inf.finite'] = throws(() => EdgeInsetsGeometry.lerp(insets['m5'], insets['e1'], 0.5));
    out['ei.lerp.inf.inf'] = ei(EdgeInsetsGeometry.lerp(insets['e7'], insets['e7'], 0.5));
    out['ei.lerp.inf.inf.0'] = ei(EdgeInsetsGeometry.lerp(insets['m5'], EdgeInsetsGeometry.infinity, 0));
    // Concrete lerps.
    const EdgeInsets ea = EdgeInsets.fromLTRB(5, 7.5, 11.25, 13), eb = EdgeInsets.fromLTRB(-1, 0.3, 2, 100);
    const EdgeInsetsDirectional da = EdgeInsetsDirectional.fromSTEB(1, 2, 3, 4),
        db = EdgeInsetsDirectional.fromSTEB(0.7, -2, 30, 0.1);
    for (final double t in ts) {
      out['edgeInsets.lerp.$t'] = ei(EdgeInsets.lerp(ea, eb, t));
      out['edgeInsets.lerp.nullA.$t'] = ei(EdgeInsets.lerp(null, eb, t));
      out['edgeInsets.lerp.nullB.$t'] = ei(EdgeInsets.lerp(ea, null, t));
      out['directional.lerp.$t'] = ei(EdgeInsetsDirectional.lerp(da, db, t));
      out['directional.lerp.nullA.$t'] = ei(EdgeInsetsDirectional.lerp(null, db, t));
      out['directional.lerp.nullB.$t'] = ei(EdgeInsetsDirectional.lerp(da, null, t));
    }
    out['edgeInsets.lerp.null'] = ei(EdgeInsets.lerp(null, null, 0.5));
    out['directional.lerp.null'] = ei(EdgeInsetsDirectional.lerp(null, null, 0.5));
    // Concrete operators.
    out['edgeInsets.plus'] = ei(ea + eb);
    out['edgeInsets.minus'] = ei(ea - eb);
    out['directional.plus'] = ei(da + db);
    out['directional.minus'] = ei(da - db);
    out['edgeInsets.corners'] = <Object>[
      <double>[ea.topLeft.dx, ea.topLeft.dy, ea.topRight.dx, ea.topRight.dy],
      <double>[ea.bottomLeft.dx, ea.bottomLeft.dy, ea.bottomRight.dx, ea.bottomRight.dy],
    ];
    const Rect rect = Rect.fromLTRB(10, 20, 110, 220);
    for (final String n in <String>['e1', 'e3', 'e6', 'e8']) {
      final EdgeInsets e = insets[n]! as EdgeInsets;
      out['edgeInsets.$n.inflateRect'] = rc(e.inflateRect(rect));
      out['edgeInsets.$n.deflateRect'] = rc(e.deflateRect(rect));
      final RRect r = RRect.fromLTRBAndCorners(
        10, 20, 110, 220,
        topLeft: const Radius.elliptical(5, 6),
        topRight: const Radius.circular(3.3),
        bottomRight: const Radius.elliptical(12, 0.5),
        bottomLeft: Radius.zero,
      );
      out['edgeInsets.$n.inflateRRect'] = rr(e.inflateRRect(r));
      out['edgeInsets.$n.deflateRRect'] = rr(e.deflateRRect(r));
      out['edgeInsets.$n.flippedConcrete'] = ltrb(e.flipped);
    }
    out['edgeInsets.copyWith'] = <Object>[
      ltrb(ea.copyWith()),
      ltrb(ea.copyWith(left: 1, bottom: -2)),
      ltrb(ea.copyWith(top: 9, right: 0.5)),
    ];
    out['directional.copyWith'] = <Object>[
      ei(da.copyWith()),
      ei(da.copyWith(start: 1.5, bottom: -2)),
      ei(da.copyWith(top: 9, end: 0.5)),
    ];
    // Clamp.
    final Map<String, List<EdgeInsetsGeometry>> bounds = <String, List<EdgeInsetsGeometry>>{
      'abs': <EdgeInsetsGeometry>[const EdgeInsets.all(2), const EdgeInsets.all(6)],
      'dir': <EdgeInsetsGeometry>[const EdgeInsetsDirectional.all(1), const EdgeInsetsDirectional.all(3)],
      'inf': <EdgeInsetsGeometry>[-EdgeInsetsGeometry.infinity, EdgeInsetsGeometry.infinity],
      'zeroInf': <EdgeInsetsGeometry>[EdgeInsets.zero, EdgeInsetsGeometry.infinity],
    };
    for (final String n in <String>['e1', 'e3', 'e6', 'd1', 'd3', 'd6', 'm1', 'm2', 'm5']) {
      bounds.forEach((String b, List<EdgeInsetsGeometry> mm) {
        out['ei.$n.clamp.$b'] = (() {
          try {
            return ei(insets[n]!.clamp(mm[0], mm[1]));
          } catch (e) {
            return 'error';
          }
        })();
      });
    }
    // Factories.
    out['ei.factories'] = <Object>[
      ei(const EdgeInsetsGeometry.all(10)),
      ei(const EdgeInsetsGeometry.only(left: 10, top: 20, right: 30, bottom: 40)),
      ei(const EdgeInsetsGeometry.directional(start: 10, top: 20, end: 30, bottom: 40)),
      ei(const EdgeInsetsGeometry.symmetric(horizontal: 10, vertical: 20)),
      ei(const EdgeInsetsGeometry.fromLTRB(10, 20, 30, 40)),
      ei(const EdgeInsetsGeometry.fromSTEB(10, 20, 20, 40)),
      ei(EdgeInsetsGeometry.zero),
      ei(const EdgeInsets.symmetric()),
      ei(const EdgeInsets.symmetric(vertical: 3)),
      ei(const EdgeInsetsDirectional.symmetric(horizontal: 3)),
    ];

    // ---- BorderRadiusGeometry ----
    final Map<String, BorderRadiusGeometry> radii = <String, BorderRadiusGeometry>{
      'b1': const BorderRadius.all(Radius.elliptical(5, 7)),
      'b2': BorderRadius.circular(3.25),
      'b3': const BorderRadius.vertical(top: Radius.circular(2), bottom: Radius.elliptical(4, 1)),
      'b4': const BorderRadius.horizontal(left: Radius.circular(-1), right: Radius.elliptical(0.05, 0.25)),
      'b5': BorderRadius.zero,
      'b6': const BorderRadius.only(topRight: Radius.circular(1), bottomLeft: Radius.elliptical(2, 3)),
      'b7': const BorderRadius.only(topLeft: Radius.circular(1)),
      'b8': const BorderRadius.only(bottomRight: Radius.elliptical(-0.0, 2.35)),
      'b9': const BorderRadius.vertical(bottom: Radius.circular(1e22)),
      'bd1': const BorderRadiusDirectional.all(Radius.elliptical(5, 7)),
      'bd2': BorderRadiusDirectional.circular(2),
      'bd3': const BorderRadiusDirectional.vertical(top: Radius.circular(2), bottom: Radius.elliptical(4, 1)),
      'bd4': const BorderRadiusDirectional.horizontal(start: Radius.circular(-1), end: Radius.elliptical(9, 0.5)),
      'bd5': BorderRadiusDirectional.zero,
      'bd6': const BorderRadiusDirectional.only(topEnd: Radius.circular(1), bottomStart: Radius.elliptical(2, 3)),
      'bd7': const BorderRadiusDirectional.only(topStart: Radius.circular(4), bottomEnd: Radius.circular(6.5)),
      'bm1': const BorderRadius.all(Radius.elliptical(5, 7))
          .add(const BorderRadiusDirectional.only(topEnd: Radius.circular(1), bottomStart: Radius.elliptical(2, 3))),
      'bm2': const BorderRadiusDirectional.all(Radius.elliptical(5, 7))
          .subtract(const BorderRadius.only(topRight: Radius.circular(1), bottomLeft: Radius.elliptical(2, 3))),
      'bm3': BorderRadius.zero.add(BorderRadiusDirectional.zero),
      'bm4': BorderRadius.circular(3.25).add(BorderRadiusDirectional.circular(2)),
      'bm5': const BorderRadius.all(Radius.elliptical(1, 2)).add(const BorderRadiusDirectional.all(Radius.elliptical(3, 4))),
      'bm6': const BorderRadius.only(topLeft: Radius.circular(1)).add(BorderRadiusDirectional.zero),
    };
    radii.forEach((String n, BorderRadiusGeometry g) {
      out['br.$n'] = br(g);
      out['br.$n.resolve.null'] = err(() => g.resolve(null));
      out['br.$n.neg'] = br(-g);
      for (final double d in divisors) {
        out['br.$n.mul.$d'] = br(g * d);
        out['br.$n.div.$d'] = br(g / d);
        out['br.$n.truncDiv.$d'] = (() {
          try {
            return br(g ~/ d);
          } catch (e) {
            return '$e';
          }
        })();
        out['br.$n.mod.$d'] = br(g % d);
      }
      out['br.$n.lerp.null.a'] = br(BorderRadiusGeometry.lerp(g, null, 0.25));
      out['br.$n.lerp.null.b'] = br(BorderRadiusGeometry.lerp(null, g, 0.25));
      out['br.$n.lerp.self'] = br(BorderRadiusGeometry.lerp(g, g, 0.5));
      final BorderRadius ltr = g.resolve(TextDirection.ltr);
      for (final Rect r in <Rect>[const Rect.fromLTRB(19, 23, 29, 31), const Rect.fromLTWH(-5, 0, 100.5, 3)]) {
        out['br.$n.toRRect.${rc(r)}'] = rr(ltr.toRRect(r));
        out['br.$n.toRSuperellipse.${rc(r)}'] = rse(ltr.toRSuperellipse(r));
      }
    });
    final List<String> brPairs = <String>['b1', 'b4', 'b6', 'bd1', 'bd4', 'bd6', 'bm1', 'bm2', 'b5', 'bd5'];
    for (final String a in brPairs) {
      for (final String b in brPairs) {
        final BorderRadiusGeometry ga = radii[a]!, gb = radii[b]!;
        out['br.$a.add.$b'] = br(ga.add(gb));
        out['br.$a.subtract.$b'] = br(ga.subtract(gb));
        out['br.$a.eq.$b'] = ga == gb;
        for (final double t in ts) {
          out['br.$a.lerp.$b.$t'] = br(BorderRadiusGeometry.lerp(ga, gb, t));
        }
      }
    }
    out['br.lerp.null.null'] = br(BorderRadiusGeometry.lerp(null, null, 0.5));
    const BorderRadius ba = BorderRadius.only(
      topLeft: Radius.elliptical(10, 20),
      topRight: Radius.elliptical(30, 40),
      bottomLeft: Radius.elliptical(50, 60),
    );
    const BorderRadius bb = BorderRadius.only(
      topRight: Radius.elliptical(100, 110),
      bottomLeft: Radius.elliptical(120.3, 130),
      bottomRight: Radius.elliptical(-140, 150),
    );
    const BorderRadiusDirectional bda = BorderRadiusDirectional.only(
      topStart: Radius.elliptical(10, 20),
      topEnd: Radius.elliptical(30, 40),
      bottomStart: Radius.elliptical(50, 60),
    );
    const BorderRadiusDirectional bdb = BorderRadiusDirectional.only(
      topEnd: Radius.elliptical(100, 110),
      bottomStart: Radius.elliptical(120.3, 130),
      bottomEnd: Radius.elliptical(-140, 150),
    );
    for (final double t in ts) {
      out['borderRadius.lerp.$t'] = br(BorderRadius.lerp(ba, bb, t));
      out['borderRadius.lerp.nullA.$t'] = br(BorderRadius.lerp(null, bb, t));
      out['borderRadius.lerp.nullB.$t'] = br(BorderRadius.lerp(ba, null, t));
      out['brDirectional.lerp.$t'] = br(BorderRadiusDirectional.lerp(bda, bdb, t));
      out['brDirectional.lerp.nullA.$t'] = br(BorderRadiusDirectional.lerp(null, bdb, t));
      out['brDirectional.lerp.nullB.$t'] = br(BorderRadiusDirectional.lerp(bda, null, t));
    }
    out['borderRadius.lerp.null'] = br(BorderRadius.lerp(null, null, 0.5));
    out['brDirectional.lerp.null'] = br(BorderRadiusDirectional.lerp(null, null, 0.5));
    out['borderRadius.plus'] = br(ba + bb);
    out['borderRadius.minus'] = br(ba - bb);
    out['brDirectional.plus'] = br(bda + bdb);
    out['brDirectional.minus'] = br(bda - bdb);
    out['borderRadius.copyWith'] = <Object>[
      br(ba.copyWith()),
      br(ba.copyWith(topLeft: Radius.zero, bottomRight: const Radius.circular(3))),
      br(ba.copyWith(topRight: const Radius.elliptical(1, 2), bottomLeft: const Radius.circular(0.5))),
    ];
    const Radius r5 = Radius.circular(5), r10 = Radius.elliptical(10, 11);
    out['br.factories'] = <Object>[
      br(const BorderRadiusGeometry.all(r10)),
      br(BorderRadiusGeometry.circular(10)),
      br(BorderRadiusGeometry.horizontal(left: r5, right: r10)),
      br(BorderRadiusGeometry.horizontal(left: r5)),
      br(BorderRadiusGeometry.horizontal(start: r5, end: r10)),
      br(BorderRadiusGeometry.horizontal(end: r10)),
      br(BorderRadiusGeometry.horizontal()),
      throws(() => BorderRadiusGeometry.horizontal(start: r5, left: r10)),
      throws(() => BorderRadiusGeometry.horizontal(end: r5, right: r10)),
      br(const BorderRadiusGeometry.only(topLeft: r5, bottomRight: r10)),
      br(const BorderRadiusGeometry.directional(topStart: r5, bottomEnd: r10)),
      br(const BorderRadiusGeometry.vertical(top: r5, bottom: r10)),
      br(BorderRadiusGeometry.zero),
      br(const BorderRadius.vertical()),
      br(const BorderRadius.horizontal(right: r5)),
      br(const BorderRadiusDirectional.vertical(bottom: r5)),
      br(const BorderRadiusDirectional.horizontal()),
    ];

    File('build/painting_edge_insets.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(JsonEncoder.withIndent(' ', (Object? o) => '$o').convert(out));
  });
}
