// Reference side of flutter-go's painting alignment differential test:
// alignment.dart (Alignment, AlignmentDirectional, the mixed alignment,
// AlignmentGeometry, TextAlignVertical), fractional_offset.dart,
// box_fit.dart and geometry.dart — lerps, operators, resolve, toString,
// equality, alongOffset/alongSize/withinRect/inscribe, applyBoxFit and
// positionDependentBox. Output: build/painting_alignment.json as
// {case: value}; doubles that JSON can't hold (and -0.0) are strings.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

Object n(double v) {
  if (v.isNaN || v.isInfinite) {
    return '$v';
  }
  if (v == 0.0 && v.isNegative) {
    return '-0.0';
  }
  return v;
}

List<Object> ns(List<double> vs) => vs.map(n).toList();

String err(void Function() f) {
  try {
    f();
  } catch (e) {
    return '$e';
  }
  return 'no error';
}

String kind(AlignmentGeometry g) => switch (g) {
  FractionalOffset() => 'FractionalOffset',
  Alignment() => 'Alignment',
  AlignmentDirectional() => 'AlignmentDirectional',
  _ => '_MixedAlignment',
};

Object desc(AlignmentGeometry? g) {
  if (g == null) {
    return 'null';
  }
  final List<Object> out = <Object>[kind(g), g.toString()];
  switch (g) {
    case FractionalOffset():
      out.addAll(ns(<double>[g.x, g.y, g.dx, g.dy]));
    case Alignment():
      out.addAll(ns(<double>[g.x, g.y]));
    case AlignmentDirectional():
      out.addAll(ns(<double>[g.start, g.y]));
    default:
      final Alignment l = g.resolve(TextDirection.ltr);
      final Alignment r = g.resolve(TextDirection.rtl);
      out.addAll(ns(<double>[l.x, r.x, l.y]));
  }
  return out;
}

List<Object> off(Offset o) => ns(<double>[o.dx, o.dy]);
List<Object> sz(Size s) => ns(<double>[s.width, s.height]);
List<Object> rc(Rect r) => ns(<double>[r.left, r.top, r.right, r.bottom]);

final List<AlignmentGeometry?> geoms = <AlignmentGeometry?>[
  Alignment.topLeft,
  Alignment.center,
  Alignment.bottomRight,
  const Alignment(0.3, -0.7),
  const Alignment(-0.0, 0.5),
  const Alignment(1e-3, 2.5),
  const Alignment(-1.25, 0.1),
  const FractionalOffset(0.25, 0.8),
  FractionalOffset.center,
  const FractionalOffset(0.1, 0.7),
  AlignmentDirectional.topStart,
  const AlignmentDirectional(0.35, -0.2),
  const AlignmentDirectional(-2.5, 1.5),
  const Alignment(0.1, 0.2).add(const AlignmentDirectional(0.3, 0.4)),
  Alignment.bottomRight.add(AlignmentDirectional.bottomEnd),
  const Alignment(0.5, 0.5).add(const AlignmentDirectional(0.0, 0.25)),
  const AlignmentDirectional(0.7, 0.5).add(const Alignment(0.0, 0.25)),
  const Alignment(0.3, -0.7).add(const AlignmentDirectional(0.2, 0.0)),
  const Alignment(-0.6, 0.3).add(const AlignmentDirectional(0.6, 0.0)),
  null,
];

const List<double> ts = <double>[0.0, 0.25, 0.5, 1.0, -0.5, 1.5, 0.1, 1.0 / 3.0];
const List<double> scalars = <double>[2.0, 0.5, -3.0, 0.3, 1.0 / 3.0, 7.0, 0.0, -0.25];

void main() {
  test('painting_alignment', () {
    final Map<String, Object> out = <String, Object>{};

    for (int i = 0; i < geoms.length; i++) {
      final AlignmentGeometry? g = geoms[i];
      out['desc.$i'] = desc(g);
      if (g == null) {
        continue;
      }
      out['resolve.$i.ltr'] = desc(g.resolve(TextDirection.ltr));
      out['resolve.$i.rtl'] = desc(g.resolve(TextDirection.rtl));
      out['resolve.$i.null'] = err(() => g.resolve(null));
      out['neg.$i'] = desc(-g);
      for (int k = 0; k < scalars.length; k++) {
        final double s = scalars[k];
        out['mul.$i.$k'] = desc(g * s);
        out['div.$i.$k'] = desc(g / s);
        out['mod.$i.$k'] = desc(g % s);
        out['truncDiv.$i.$k'] = err(() => out['truncDiv.$i.$k.value'] = desc(g ~/ s));
      }
      if (g is Alignment) {
        for (final Offset o in const <Offset>[Offset(10, 20), Offset(-3.3, 7.7), Offset(0.1, 1e6)]) {
          out['alongOffset.$i.$o'] = off(g.alongOffset(o));
        }
        for (final Size s in const <Size>[Size(10, 20), Size(3.3, 7.7), Size.zero]) {
          out['alongSize.$i.$s'] = off(g.alongSize(s));
          for (final Rect r in const <Rect>[
            Rect.fromLTWH(5, 6, 100, 50),
            Rect.fromLTRB(-1.1, 2.2, 3.3, 4.4),
            Rect.fromLTWH(0, 0, 1, 1),
          ]) {
            out['inscribe.$i.$s.$r'] = rc(g.inscribe(s, r));
          }
        }
        for (final Rect r in const <Rect>[
          Rect.fromLTWH(5, 6, 100, 50),
          Rect.fromLTRB(-1.1, 2.2, 3.3, 4.4),
        ]) {
          out['withinRect.$i.$r'] = off(g.withinRect(r));
        }
      }
    }

    for (int i = 0; i < geoms.length; i++) {
      for (int j = 0; j < geoms.length; j++) {
        final AlignmentGeometry? a = geoms[i];
        final AlignmentGeometry? b = geoms[j];
        for (int k = 0; k < ts.length; k++) {
          out['lerp.$i.$j.$k'] = desc(AlignmentGeometry.lerp(a, b, ts[k]));
          if ((a == null || a is Alignment) && (b == null || b is Alignment)) {
            out['alignmentLerp.$i.$j.$k'] = desc(Alignment.lerp(a as Alignment?, b as Alignment?, ts[k]));
          }
          if ((a == null || a is AlignmentDirectional) && (b == null || b is AlignmentDirectional)) {
            out['directionalLerp.$i.$j.$k'] = desc(
              AlignmentDirectional.lerp(a as AlignmentDirectional?, b as AlignmentDirectional?, ts[k]),
            );
          }
          if ((a == null || a is FractionalOffset) && (b == null || b is FractionalOffset)) {
            out['fractionalLerp.$i.$j.$k'] = desc(
              FractionalOffset.lerp(a as FractionalOffset?, b as FractionalOffset?, ts[k]),
            );
          }
        }
        if (a == null || b == null) {
          continue;
        }
        out['add.$i.$j'] = desc(a.add(b));
        out['eq.$i.$j'] = a == b;
        if (a is Alignment && b is Alignment) {
          out['plus.$i.$j'] = desc(a + b);
          out['minus.$i.$j'] = desc(a - b);
        }
        if (a is AlignmentDirectional && b is AlignmentDirectional) {
          out['plus.$i.$j'] = desc(a + b);
          out['minus.$i.$j'] = desc(a - b);
        }
      }
    }

    // Named constants.
    final Map<String, AlignmentGeometry> named = <String, AlignmentGeometry>{
      'Alignment.topLeft': Alignment.topLeft,
      'Alignment.topCenter': Alignment.topCenter,
      'Alignment.topRight': Alignment.topRight,
      'Alignment.centerLeft': Alignment.centerLeft,
      'Alignment.center': Alignment.center,
      'Alignment.centerRight': Alignment.centerRight,
      'Alignment.bottomLeft': Alignment.bottomLeft,
      'Alignment.bottomCenter': Alignment.bottomCenter,
      'Alignment.bottomRight': Alignment.bottomRight,
      'AlignmentDirectional.topStart': AlignmentDirectional.topStart,
      'AlignmentDirectional.topCenter': AlignmentDirectional.topCenter,
      'AlignmentDirectional.topEnd': AlignmentDirectional.topEnd,
      'AlignmentDirectional.centerStart': AlignmentDirectional.centerStart,
      'AlignmentDirectional.center': AlignmentDirectional.center,
      'AlignmentDirectional.centerEnd': AlignmentDirectional.centerEnd,
      'AlignmentDirectional.bottomStart': AlignmentDirectional.bottomStart,
      'AlignmentDirectional.bottomCenter': AlignmentDirectional.bottomCenter,
      'AlignmentDirectional.bottomEnd': AlignmentDirectional.bottomEnd,
      'FractionalOffset.topLeft': FractionalOffset.topLeft,
      'FractionalOffset.topCenter': FractionalOffset.topCenter,
      'FractionalOffset.topRight': FractionalOffset.topRight,
      'FractionalOffset.centerLeft': FractionalOffset.centerLeft,
      'FractionalOffset.center': FractionalOffset.center,
      'FractionalOffset.centerRight': FractionalOffset.centerRight,
      'FractionalOffset.bottomLeft': FractionalOffset.bottomLeft,
      'FractionalOffset.bottomCenter': FractionalOffset.bottomCenter,
      'FractionalOffset.bottomRight': FractionalOffset.bottomRight,
      'AlignmentGeometry.topStart': AlignmentGeometry.topStart,
      'AlignmentGeometry.bottomEnd': AlignmentGeometry.bottomEnd,
      'AlignmentGeometry.xy': const AlignmentGeometry.xy(0.25, -4),
      'AlignmentGeometry.directional': const AlignmentGeometry.directional(0.25, -4),
    };
    named.forEach((String k, AlignmentGeometry v) => out['named.$k'] = desc(v));

    // toString of odd values.
    for (final List<double> v in const <List<double>>[
      <double>[-0.0, -1.0],
      <double>[0.05, 0.15],
      <double>[0.25, 0.35],
      <double>[-0.05, -0.15],
      <double>[1e21, 123456.789],
      <double>[double.infinity, double.negativeInfinity],
      <double>[double.nan, 0.0],
    ]) {
      out['toString.${v[0]}.${v[1]}'] = <String>[
        Alignment(v[0], v[1]).toString(),
        AlignmentDirectional(v[0], v[1]).toString(),
        FractionalOffset(v[0], v[1]).toString(),
        Alignment(v[0], v[1]).add(AlignmentDirectional(v[1], v[0])).toString(),
      ];
    }

    // ~/ clamps to int64 and gives +0.0 for a negative zero quotient.
    out['truncDiv.huge'] = <Object>[
      desc(const Alignment(1e300, -1e300) ~/ 1.0),
      desc(const FractionalOffset(1e300, -0.3) ~/ 0.5),
      desc(const AlignmentDirectional(-0.25, 0.5) ~/ 2.0),
      desc(const Alignment(-0.0, -0.75) ~/ 1.0),
      err(() => const Alignment(double.infinity, 0.5) ~/ 2.0),
    ];

    // FractionalOffset factories.
    for (final Offset o in const <Offset>[Offset(100, 100), Offset(-3.3, 7.1), Offset(0.1, 0.2)]) {
      for (final Size s in const <Size>[Size(200, 400), Size(3, 7), Size(0.3, 0.7)]) {
        out['fromOffsetAndSize.$o.$s'] = desc(FractionalOffset.fromOffsetAndSize(o, s));
      }
      for (final Rect r in const <Rect>[Rect.fromLTWH(50, 20, 200, 400), Rect.fromLTRB(-1.1, 2.2, 3.3, 4.4)]) {
        out['fromOffsetAndRect.$o.$r'] = desc(FractionalOffset.fromOffsetAndRect(o, r));
      }
    }

    // TextAlignVertical.
    out['textAlignVertical'] = <String>[
      TextAlignVertical.top.toString(),
      TextAlignVertical.center.toString(),
      TextAlignVertical.bottom.toString(),
      const TextAlignVertical(y: 0.3).toString(),
      const TextAlignVertical(y: -0.0).toString(),
    ];

    // applyBoxFit.
    const List<Size> sizes = <Size>[
      Size(100, 1000),
      Size(200, 2000),
      Size(300, 3000),
      Size(2000, 400),
      Size(1000, 100),
      Size(1000, 300),
      Size(400, 2000),
      Size(33.3, 77.7),
      Size(0.7, 0.1),
      Size(64, 64),
      Size(0, 10),
      Size(10, 0),
      Size(-5, 10),
      Size(1e-5, 1e5),
      Size(100, 100),
      Size(10, 100.5),
      Size(3, 7),
      Size(0.5, 7.7),
      Size(4, 7.7),
    ];
    for (final BoxFit fit in BoxFit.values) {
      out['boxFit.name.${fit.index}'] = fit.toString();
      for (int i = 0; i < sizes.length; i++) {
        for (int j = 0; j < sizes.length; j++) {
          final FittedSizes f = applyBoxFit(fit, sizes[i], sizes[j]);
          out['boxFit.${fit.index}.$i.$j'] = <Object>[...sz(f.source), ...sz(f.destination)];
        }
      }
    }

    // positionDependentBox.
    const List<Size> outer = <Size>[Size(100, 100), Size(800, 600), Size(37.5, 300.3)];
    const List<Size> child = <Size>[Size(20, 10), Size(200, 10), Size(50, 33.3), Size(10, 590)];
    const List<Offset> targets = <Offset>[
      Offset(50, 50),
      Offset(0, 50),
      Offset(100, 5),
      Offset(400.5, 590),
      Offset(-10, 300),
      Offset(18.7, 150.15),
    ];
    for (int a = 0; a < outer.length; a++) {
      for (int b = 0; b < child.length; b++) {
        for (int c = 0; c < targets.length; c++) {
          for (final bool below in <bool>[false, true]) {
            out['pdb.$a.$b.$c.$below'] = off(
              positionDependentBox(size: outer[a], childSize: child[b], target: targets[c], preferBelow: below),
            );
            for (final double vo in <double>[0.0, 24.0, 7.5]) {
              for (final double m in <double>[0.0, 10.0, 20.0, 3.3]) {
                out['pdb.$a.$b.$c.$below.$vo.$m'] = off(
                  positionDependentBox(
                    size: outer[a],
                    childSize: child[b],
                    target: targets[c],
                    preferBelow: below,
                    verticalOffset: vo,
                    margin: m,
                  ),
                );
              }
            }
          }
        }
      }
    }

    // basic_types.
    out['basicTypes'] = <Object>[
      for (final Axis a in Axis.values) <Object>[a.toString(), flipAxis(a).toString()],
      for (final AxisDirection d in AxisDirection.values)
        <Object>[
          d.toString(),
          axisDirectionToAxis(d).toString(),
          flipAxisDirection(d).toString(),
          axisDirectionIsReversed(d),
        ],
      for (final TextDirection d in TextDirection.values) textDirectionToAxisDirection(d).toString(),
      for (final VerticalDirection d in VerticalDirection.values) d.toString(),
      for (final RenderComparison c in RenderComparison.values) c.toString(),
    ];

    File('build/painting_alignment.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
