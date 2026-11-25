// Reference side of flutter-go's dart:ui geometry differential test: each
// case prints a value with Dart's own toString/number formatting.
// Output: build/geometry.json as {case: "printed value"}.
import 'dart:convert';
import 'dart:io';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

String err(Object Function() f) {
  try {
    return '${f()}';
  } on UnsupportedError {
    return 'UnsupportedError';
  }
}

void main() {
  test('geometry', () {
    const Offset o = Offset(5.25, -3.5);
    const Size s = Size(3, 4);
    const Rect r = Rect.fromLTRB(1, 2, 11, 22);
    final RRect rr = RRect.fromRectAndCorners(r,
        topLeft: const Radius.circular(2), topRight: const Radius.elliptical(3, 4),
        bottomRight: const Radius.circular(9), bottomLeft: const Radius.circular(12));
    final RRect uni = RRect.fromLTRBXY(0, 0, 10, 10, 2, 3);
    final Map<String, Object?> out = <String, Object?>{
      'offset.toString': '$o',
      'offset.neg': '${-o}',
      'offset.mod': '${o % 2}',
      'offset.mod.neg': '${const Offset(-7.5, -0.25) % 2}',
      'offset.truncdiv': '${o ~/ 2}',
      'offset.truncdiv.zero': err(() => o ~/ 0),
      'offset.distance': o.distance,
      'offset.and': '${o & s}',
      'offset.lerp.a.null': '${Offset.lerp(null, o, 0.25)}',
      'offset.lerp.b.null': '${Offset.lerp(o, null, 0.25)}',
      'offset.lerp': '${Offset.lerp(o, const Offset(1, 1), 0.3)}',
      'offset.lerp.x': Offset.lerp(o, const Offset(1, 1), 0.3)!.dx,
      'size.toString': '$s',
      'size.minus.size': '${s - const Size(1, 1)}',
      'size.minus.offset': '${s - const Offset(1, 1)}',
      'size.mod': '${const Size(-5, 7.5) % 4}',
      'size.flipped': '${s.flipped}',
      'size.center': '${s.center(o)}',
      'size.lerp': '${Size.lerp(s, const Size(10, 0), 0.7)}',
      'size.lerp.null': '${Size.lerp(null, s, 0.5)}',
      'rect.toString': '$r',
      'rect.inflate': '${r.inflate(1.5)}',
      'rect.overlaps': '${r.overlaps(const Rect.fromLTRB(11, 0, 12, 30))}',
      'rect.contains.edge': '${r.contains(const Offset(11, 5))}',
      'rect.fromPoints': '${Rect.fromPoints(const Offset(5, 1), const Offset(-1, 9))}',
      'rect.fromCircle': '${Rect.fromCircle(center: o, radius: 2.5)}',
      'rect.lerp': '${Rect.lerp(r, Rect.zero, 0.25)}',
      'rect.lerp.null': '${Rect.lerp(null, r, 0.25)}',
      'rect.isEmpty': '${const Rect.fromLTRB(1, 1, 1, 5).isEmpty}',
      'radius.toString.circular': '${const Radius.circular(2.25)}',
      'radius.toString.elliptical': '${const Radius.elliptical(2, 3)}',
      'radius.mod': '${const Radius.elliptical(-5, 5) % 3}',
      'radius.lerp': '${Radius.lerp(const Radius.circular(1), const Radius.elliptical(3, 9), 0.5)}',
      'radius.lerp.null': '${Radius.lerp(null, const Radius.circular(4), 0.5)}',
      'rrect.toString.corners': '$rr',
      'rrect.toString.xy': '$uni',
      'rrect.toString.r': '${RRect.fromLTRBR(0, 0, 4, 4, const Radius.circular(1))}',
      'rrect.safeInnerRect': '${rr.safeInnerRect}',
      'rrect.safeInnerRect.left': rr.safeInnerRect.left,
      'rrect.middleRect': '${rr.middleRect}',
      'rrect.wideMiddleRect': '${rr.wideMiddleRect}',
      'rrect.tallMiddleRect': '${rr.tallMiddleRect}',
      'rrect.scaleRadii': '${rr.scaleRadii()}',
      'rrect.isStadium': '${RRect.fromLTRBR(0, 0, 10, 4, const Radius.circular(2)).isStadium}',
      'rrect.isEllipse': '${RRect.fromLTRBXY(0, 0, 10, 4, 5, 2).isEllipse}',
      'rrect.isCircle': '${RRect.fromLTRBR(0, 0, 4, 4, const Radius.circular(2)).isCircle}',
      'rrect.isRect': '${RRect.fromLTRBXY(0, 0, 4, 4, 0, 3).isRect}',
      'rrect.lerp': '${RRect.lerp(rr, uni, 0.4)}',
      'rrect.lerp.null': '${RRect.lerp(null, rr, 0.4)}',
      'rrect.lerp.null.b': '${RRect.lerp(rr, null, 0.4)}',
      'rrect.contains.corner': '${rr.contains(const Offset(1.5, 2.5))}',
      'rrect.contains.inside': '${rr.contains(const Offset(6, 12))}',
      'rrect.contains.grid': <String>[
        for (double y = 1.75; y <= 22.25; y += 0.5)
          <String>[for (double x = 0.75; x <= 11.25; x += 0.5) if (rr.contains(Offset(x, y))) '1' else '0'].join(),
      ].join('|'),
      for (final MapEntry<String, RSuperellipse> e in <String, RSuperellipse>{
        'uniform': RSuperellipse.fromLTRBXY(0, 0, 40, 30, 9, 9),
        'elliptical': RSuperellipse.fromLTRBXY(0, 0, 40, 30, 6, 13),
        'mixed': RSuperellipse.fromLTRBAndCorners(0, 0, 40, 30, topLeft: const Radius.circular(2),
            topRight: const Radius.elliptical(15, 6), bottomRight: const Radius.circular(25), bottomLeft: Radius.zero),
        'inverted': RSuperellipse.fromLTRBXY(40, 30, 0, 0, 7, 7),
        'ratio27': RSuperellipse.fromLTRBXY(0, 0, 30, 30, 11, 11),
        'ratio15': RSuperellipse.fromLTRBXY(0, 0, 40, 30, 2, 2),
      }.entries)
        'rse.contains.${e.key}': <String>[
          for (double y = -0.75; y <= 30.75; y += 0.5)
            <String>[for (double x = -0.75; x <= 40.75; x += 0.5) if (e.value.contains(Offset(x, y))) '1' else '0'].join(),
        ].join('|'),
      'rse.contains.ratio27.corner.fine': <String>[
        for (int j = 0; j < 160; j++)
          <String>[
            for (int i = 0; i < 160; i++)
              if (RSuperellipse.fromLTRBXY(0, 0, 30, 30, 11, 11).contains(Offset(22.025 + i * 0.05, 0.025 + j * 0.05))) '1' else '0',
          ].join(),
      ].join('|'),
      'rse.toString': '${RSuperellipse.fromRectAndCorners(r, topLeft: const Radius.circular(2))}',
      'rse.lerp': '${RSuperellipse.lerp(RSuperellipse.fromLTRBXY(0, 0, 10, 10, 2, 3), null, 0.5)}',
      'rstransform': <double>[
        for (final RSTransform t in <RSTransform>[
          RSTransform(0.1, 0.2, 0.3, 1e-8),
          RSTransform.fromComponents(rotation: 0.7, scale: 1.3, anchorX: 2, anchorY: 3, translateX: 10, translateY: -4),
        ]) ...<double>[t.scos, t.ssin, t.tx, t.ty],
      ],
    };
    File('build/geometry.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
