// Reference side of flutter-go's painting borders differential test:
// borders.dart (BorderSide, ShapeBorder, OutlinedBorder, the compound
// border, paintBorder), box_border.dart, circle/oval/rounded rectangle/
// rounded superellipse/stadium/beveled/continuous/star/linear borders and
// notched_shapes.dart. Every value is printed as a string with Dart's
// toString (doubles round-trip exactly), so the Go side
// (src/difftest/painting_borders_test.go) must match bit for bit.
// Paths are compared by bounds, metrics (lengths, tangents at fixed
// fractions) and contains() over a grid; paint() calls through a
// recording canvas. Output: build/painting_borders.json as {case: value}.
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

String d(double v) => '$v';

String col(Color c) => '${d(c.a)},${d(c.r)},${d(c.g)},${d(c.b)}';

String sideF(BorderSide s) => '${col(s.color)}|${d(s.width)}|${s.style}|${d(s.strokeAlign)}';

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

// Path signature: bounds, metrics, and contains() over an n×n grid
// covering area inflated by 3.
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

String paintF(Paint p) => '${col(p.color)}/${d(p.strokeWidth)}/${p.style}';

class Rec implements Canvas {
  Rec(this.area);
  final Rect area;
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
      return paintF(a);
    }
    if (a is Offset) {
      return off(a);
    }
    if (a is double) {
      return d(a);
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

BorderSide bs({int color = 0xFF000000, double width = 1.0, BorderStyle style = BorderStyle.solid,
        double strokeAlign = BorderSide.strokeAlignInside}) =>
    BorderSide(color: Color(color), width: width, style: style, strokeAlign: strokeAlign);

List<BorderSide> sides() => <BorderSide>[
  const BorderSide(),
  BorderSide.none,
  bs(color: 0xFF2196F3, width: 3),
  bs(color: 0x802196F3, width: 0.5, strokeAlign: BorderSide.strokeAlignCenter),
  bs(color: 0xFFE91E63, width: 7, strokeAlign: BorderSide.strokeAlignOutside),
  bs(color: 0xFF4CAF50, width: 2, style: BorderStyle.none),
  bs(color: 0xFFFF9800, width: 0),
  bs(color: 0x33123456, width: 12.75, strokeAlign: -0.25),
];

const List<double> ts = <double>[-0.5, 0, 0.1, 0.25, 0.5, 0.75, 1, 1.5];

List<ShapeBorder> shapes() {
  final List<BorderSide> s = sides();
  return <ShapeBorder>[
    Border.all(width: 3, color: const Color(0xFF2196F3)),
    Border(top: s[2], right: s[4], bottom: s[6], left: s[3]),
    Border(left: s[2]),
    Border.symmetric(vertical: s[3], horizontal: s[2]),
    BorderDirectional(top: s[2], start: s[4], end: s[3], bottom: s[7]),
    BorderDirectional(start: s[2]),
    CircleBorder(side: s[2]),
    CircleBorder(side: s[4], eccentricity: 0.5),
    OvalBorder(side: s[3]),
    OvalBorder(side: s[2], eccentricity: 0.25),
    RoundedRectangleBorder(side: s[2], borderRadius: const BorderRadius.all(Radius.circular(12))),
    RoundedRectangleBorder(
      side: s[3],
      borderRadius: const BorderRadiusDirectional.only(topStart: Radius.circular(20), bottomEnd: Radius.elliptical(8, 30)),
    ),
    const RoundedRectangleBorder(),
    RoundedSuperellipseBorder(side: s[2], borderRadius: const BorderRadius.all(Radius.circular(18))),
    RoundedSuperellipseBorder(side: s[4]),
    StadiumBorder(side: s[2]),
    StadiumBorder(side: s[7]),
    BeveledRectangleBorder(side: s[2], borderRadius: const BorderRadius.all(Radius.circular(10))),
    BeveledRectangleBorder(
      side: s[4],
      borderRadius: const BorderRadiusDirectional.only(topStart: Radius.circular(30), bottomStart: Radius.circular(5)),
    ),
    ContinuousRectangleBorder(side: s[2], borderRadius: const BorderRadius.all(Radius.circular(24))),
    ContinuousRectangleBorder(side: s[3], borderRadius: const BorderRadius.only(topLeft: Radius.elliptical(60, 20))),
    StarBorder(side: s[2]),
    StarBorder(side: s[3], points: 7.5, innerRadiusRatio: 0.6, pointRounding: 0.3, valleyRounding: 0.2, rotation: 27, squash: 0.5),
    StarBorder.polygon(side: s[4], sides: 6, pointRounding: 0.4, rotation: 90, squash: 1),
    StarBorder.polygon(side: s[2], sides: 3.5),
    LinearBorder.start(side: s[2], alignment: -0.5, size: 0.75),
    LinearBorder(side: s[4], top: const LinearBorderEdge(size: 0.5, alignment: 1), bottom: const LinearBorderEdge()),
    LinearBorder.none,
  ];
}

final List<Rect> rects = <Rect>[
  const Rect.fromLTRB(10, 20, 110, 70),
  const Rect.fromLTRB(5.5, 3.25, 45.75, 120.5),
  const Rect.fromLTWH(-20, -20, 40, 40),
];

void main() {
  test('painting_borders', () {
    final out = <String, String>{};
    final List<BorderSide> s = sides();

    // BorderSide.
    for (int i = 0; i < s.length; i++) {
      final BorderSide a = s[i];
      out['side.$i'] =
          '$a|${sideF(a)}|${d(a.strokeInset)}|${d(a.strokeOutset)}|${d(a.strokeOffset)}|${paintF(a.toPaint())}';
      for (final double t in <double>[-1, 0, 0.5, 2]) {
        out['side.$i.scale.$t'] = tryS(() => sideF(a.scale(t)));
      }
      for (int j = 0; j < s.length; j++) {
        final BorderSide b = s[j];
        out['side.$i.$j.canMerge'] = '${BorderSide.canMerge(a, b)}';
        out['side.$i.$j.merge'] = tryS(() => sideF(BorderSide.merge(a, b)));
        out['side.$i.$j.=='] = '${a == b}';
        for (final double t in ts) {
          out['side.$i.$j.lerp.$t'] = tryS(() => sideF(BorderSide.lerp(a, b, t)));
        }
      }
    }
    out['side.copyWith'] = sideF(
      s[2].copyWith(color: const Color(0xFF00FF00), width: 4, style: BorderStyle.none, strokeAlign: 0.5),
    );
    out['side.copyWith.neg'] = tryS(() => sideF(s[2].copyWith(width: -1)));

    // Shape borders.
    final List<ShapeBorder> b = shapes();
    String desc(ShapeBorder? x) {
      if (x == null) {
        return 'null';
      }
      return '${x.runtimeType}|$x|${tryS(() => insets(x.dimensions, TextDirection.ltr))}|'
          '${tryS(() => insets(x.dimensions, TextDirection.rtl))}|${x.preferPaintInterior}|'
          '${tryS(() => pathSig(x.getOuterPath(rects[0], textDirection: TextDirection.ltr), rects[0], n: 7))}|'
          '${tryS(() => pathSig(x.getInnerPath(rects[0], textDirection: TextDirection.rtl), rects[0], n: 7))}';
    }

    for (int i = 0; i < b.length; i++) {
      final ShapeBorder x = b[i];
      out['shape.$i'] = desc(x);
      for (final double t in <double>[0, 0.5, 2]) {
        out['shape.$i.scale.$t'] = tryS(() => desc(x.scale(t)));
      }
      for (final double t in <double>[0, 0.3, 1]) {
        out['shape.$i.lerpNull.$t'] = tryS(() => desc(ShapeBorder.lerp(x, null, t)));
        out['shape.$i.nullLerp.$t'] = tryS(() => desc(ShapeBorder.lerp(null, x, t)));
      }
      for (int r = 0; r < rects.length; r++) {
        for (final TextDirection td in TextDirection.values) {
          final String k = 'shape.$i.r$r.${td.name}';
          out['$k.outer'] = tryS(() => pathSig(x.getOuterPath(rects[r], textDirection: td), rects[r]));
          out['$k.inner'] = tryS(() => pathSig(x.getInnerPath(rects[r], textDirection: td), rects[r]));
          out['$k.hit'] = tryS(() {
            final buf = StringBuffer();
            final Rect a = rects[r].inflate(2);
            for (int p = 0; p < 9; p++) {
              for (int q = 0; q < 9; q++) {
                buf.write(x.hitTest(rects[r], Offset(a.left + a.width * p / 8, a.top + a.height * q / 8), textDirection: td) ? '1' : '0');
              }
            }
            return buf.toString();
          });
          out['$k.paint'] = tryS(() {
            final rec = Rec(rects[r]);
            x.paint(rec, rects[r], textDirection: td);
            return rec.log.join('\n');
          });
          out['$k.paintInterior'] = tryS(() {
            final rec = Rec(rects[r]);
            x.paintInterior(rec, rects[r], Paint()..color = const Color(0xFF123456), textDirection: td);
            return rec.log.join('\n');
          });
        }
      }
      for (int j = 0; j < b.length; j++) {
        final ShapeBorder y = b[j];
        out['shape.$i.$j.add'] = tryS(() => desc(x + y));
        for (final double t in <double>[-0.25, 0, 0.2, 0.5, 0.9, 1, 1.25]) {
          out['shape.$i.$j.lerp.$t'] = tryS(() => desc(ShapeBorder.lerp(x, y, t)));
        }
        if (x is OutlinedBorder && y is OutlinedBorder) {
          out['shape.$i.$j.olerp.0.4'] = tryS(() => desc(OutlinedBorder.lerp(x, y, 0.4)));
        }
        if (x is BoxBorder && y is BoxBorder) {
          for (final double t in <double>[-0.5, 0.25, 0.5, 0.75, 1.5]) {
            out['shape.$i.$j.boxlerp.$t'] = tryS(() => desc(BoxBorder.lerp(x, y, t)));
          }
        }
      }
    }

    // Lerps through the private intermediates, lerped again.
    final rr = RoundedRectangleBorder(side: s[2], borderRadius: const BorderRadius.all(Radius.circular(12)));
    final rs = RoundedSuperellipseBorder(side: s[3], borderRadius: const BorderRadius.all(Radius.circular(18)));
    final st = StadiumBorder(side: s[4]);
    final ci = CircleBorder(side: s[7], eccentricity: 0.4);
    final ov = OvalBorder(side: s[2]);
    final inter = <String, ShapeBorder>{
      'rr-ci': ShapeBorder.lerp(rr, ci, 0.3)!,
      'ci-rr': ShapeBorder.lerp(ci, rr, 0.6)!,
      'rs-ov': ShapeBorder.lerp(rs, ov, 0.45)!,
      'st-ci': ShapeBorder.lerp(st, ci, 0.35)!,
      'st-rr': ShapeBorder.lerp(st, rr, 0.65)!,
      'rr-st': ShapeBorder.lerp(rr, st, 0.2)!,
      'compound': (b[0] + b[2]) + b[6],
    };
    final others = <ShapeBorder>[rr, rs, st, ci, ov, ...inter.values];
    for (final MapEntry<String, ShapeBorder> e in inter.entries) {
      out['inter.${e.key}'] = desc(e.value);
      out['inter.${e.key}.copyWithSide'] = tryS(() => desc((e.value as OutlinedBorder).copyWith(side: s[6])));
      for (int r = 0; r < rects.length; r++) {
        for (final TextDirection td in TextDirection.values) {
          final String k = 'inter.${e.key}.r$r.${td.name}';
          out['$k.outer'] = tryS(() => pathSig(e.value.getOuterPath(rects[r], textDirection: td), rects[r]));
          out['$k.inner'] = tryS(() => pathSig(e.value.getInnerPath(rects[r], textDirection: td), rects[r]));
          out['$k.paint'] = tryS(() {
            final rec = Rec(rects[r]);
            e.value.paint(rec, rects[r], textDirection: td);
            return rec.log.join('\n');
          });
        }
      }
      for (int j = 0; j < others.length; j++) {
        for (final double t in <double>[0.15, 0.5, 0.85]) {
          out['inter.${e.key}.$j.lerp.$t'] = tryS(() => desc(ShapeBorder.lerp(e.value, others[j], t)));
          out['inter.${e.key}.$j.rlerp.$t'] = tryS(() => desc(ShapeBorder.lerp(others[j], e.value, t)));
        }
      }
    }

    // Box borders painted with shapes and radii.
    final boxes = <BoxBorder>[
      for (final ShapeBorder x in b)
        if (x is BoxBorder) x,
      Border(top: s[2], right: s[2].copyWith(width: 6), bottom: s[2], left: s[2].copyWith(width: 1)),
      Border(top: s[2], left: s[5]),
      Border(bottom: s[6]),
      BorderDirectional(start: s[2], top: s[2].copyWith(width: 9), end: s[2]),
      Border.all(color: const Color(0xFF00FF00), width: 4, strokeAlign: BorderSide.strokeAlignCenter),
      Border.all(color: const Color(0xFF00FF00), width: 0),
      Border.all(style: BorderStyle.none),
    ];
    for (int i = 0; i < boxes.length; i++) {
      out['box.$i'] = '${boxes[i]}|${boxes[i].isUniform}|${sideF(boxes[i].top)}|${sideF(boxes[i].bottom)}';
      for (final BoxShape shape in BoxShape.values) {
        for (final BorderRadius? radius in <BorderRadius?>[
          null,
          BorderRadius.zero,
          const BorderRadius.all(Radius.circular(8)),
          const BorderRadius.only(topLeft: Radius.elliptical(20, 10), bottomRight: Radius.circular(4)),
        ]) {
          for (final TextDirection? td in <TextDirection?>[TextDirection.ltr, TextDirection.rtl, null]) {
            out['box.$i.$shape.${radius == null ? 'null' : rrect(radius.toRRect(rects[0]))}.$td'] = tryS(() {
              final rec = Rec(rects[0]);
              boxes[i].paint(rec, rects[0], textDirection: td, shape: shape, borderRadius: radius);
              return rec.log.join('\n');
            });
          }
        }
      }
    }

    // paintBorder.
    for (int i = 0; i < s.length; i++) {
      for (int j = 0; j < s.length; j++) {
        out['paintBorder.$i.$j'] = tryS(() {
          final rec = Rec(rects[1]);
          paintBorder(rec, rects[1], top: s[i], right: s[j], bottom: s[(i + j) % s.length], left: s[(i + 2 * j) % s.length]);
          return rec.log.join('\n');
        });
      }
    }

    // StarBorder parameters.
    int k = 0;
    for (final double points in <double>[2, 2.5, 3, 4.25, 5, 8]) {
      for (final double inner in <double>[0, 0.35, 1]) {
        for (final (double pr, double vr) in <(double, double)>[(0, 0), (0.5, 0.2), (1, 0), (0, 1)]) {
          for (final double rot in <double>[0, 27, 270]) {
            for (final double squash in <double>[0, 0.5, 1]) {
              k++;
              if (k % 3 != 0) {
                continue;
              }
              final star = StarBorder(
                points: points,
                innerRadiusRatio: inner,
                pointRounding: pr,
                valleyRounding: vr,
                rotation: rot,
                squash: squash,
              );
              out['star.$points.$inner.$pr.$vr.$rot.$squash'] =
                  '$star|${d(star.rotation)}|${pathSig(star.getOuterPath(rects[k % 3]), rects[k % 3])}';
            }
          }
        }
      }
    }
    for (final double sides in <double>[2, 3, 3.5, 6, 11.75]) {
      for (final double pr in <double>[0, 0.5, 1]) {
        final poly = StarBorder.polygon(sides: sides, pointRounding: pr, rotation: 45, squash: 0.25);
        out['polygon.$sides.$pr'] = '$poly|${d(poly.innerRadiusRatio)}|${pathSig(poly.getOuterPath(rects[0]), rects[0])}';
        out['polygon.$sides.$pr.copy'] = '${poly.copyWith(points: sides + 1)}';
      }
    }
    for (final ShapeBorder other in <ShapeBorder>[
      const CircleBorder(),
      const CircleBorder(eccentricity: 0.6),
      const StadiumBorder(),
      const RoundedRectangleBorder(),
      const RoundedRectangleBorder(borderRadius: BorderRadius.only(bottomLeft: Radius.circular(10))),
      const StarBorder(points: 6, pointRounding: 0.5, valleyRounding: 0.5, innerRadiusRatio: 0.5, rotation: 90),
    ]) {
      for (final StarBorder from in <StarBorder>[const StarBorder(), const StarBorder.polygon(sides: 2.2)]) {
        for (final double t in <double>[0, 0.2, 0.3, 0.5, 0.7, 0.9, 1]) {
          final String key = 'starlerp.$from.$other.$t';
          out['$key.to'] = tryS(() => desc(from.lerpTo(other, t)));
          out['$key.from'] = tryS(() => desc(from.lerpFrom(other, t)));
        }
      }
    }

    // LinearBorderEdge.
    final edges = <LinearBorderEdge?>[
      null,
      const LinearBorderEdge(),
      const LinearBorderEdge(size: 0.25, alignment: -1),
      const LinearBorderEdge(size: 0, alignment: 0.5),
    ];
    for (int i = 0; i < edges.length; i++) {
      out['edge.$i'] = '${edges[i]}';
      for (int j = 0; j < edges.length; j++) {
        for (final double t in <double>[0, 0.4, 1]) {
          out['edge.$i.$j.$t'] = tryS(() {
            final LinearBorderEdge? e = LinearBorderEdge.lerp(edges[i], edges[j], t);
            return e == null ? 'null' : '$e|${d(e.size)}|${d(e.alignment)}';
          });
        }
      }
    }

    // Notched shapes.
    const host = Rect.fromLTRB(0, 100, 300, 300);
    for (final bool inverted in <bool>[false, true]) {
      for (final Rect? guest in <Rect?>[
        null,
        const Rect.fromLTWH(50, 50, 10, 10),
        const Rect.fromLTRB(190, 85, 210, 105),
        const Rect.fromLTRB(190, 95, 210, 115),
        const Rect.fromLTRB(190, 285, 210, 305),
        const Rect.fromLTRB(190, 295, 210, 315),
        const Rect.fromLTRB(100.5, 70.25, 160.5, 130.25),
      ]) {
        out['notch.$inverted.${guest == null ? 'null' : rect(guest)}'] = tryS(
          () => pathSig(CircularNotchedRectangle(inverted: inverted).getOuterPath(host, guest), host, n: 15),
        );
      }
    }
    out['notch.auto.noguest'] = pathSig(
      AutomaticNotchedShape(b[10], b[6]).getOuterPath(rects[1], null),
      rects[1],
    );

    // Equality.
    for (int i = 0; i < b.length; i++) {
      final buf = StringBuffer();
      for (int j = 0; j < b.length; j++) {
        buf.write(b[i] == b[j] ? '1' : '0');
      }
      out['eq.$i'] = buf.toString();
    }

    File('build/painting_borders.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
