// Reference side of flutter-go's dart:ui painting differential test (Color
// so far): exact component values and printed forms.
// Output: build/painting.json as {case: value}.
import 'dart:convert';
import 'dart:io';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

List<Object> c(Color x) => <Object>[x.a, x.r, x.g, x.b, x.colorSpace.index];

void main() {
  test('painting', () {
    const Color p3red = Color.from(alpha: 1, red: 1, green: 0, blue: 0, colorSpace: ColorSpace.displayP3);
    const Color ext = Color.from(alpha: 0.8, red: 1.2, green: -0.1, blue: 0.5, colorSpace: ColorSpace.extendedSRGB);
    const Color half = Color.from(alpha: 0.5, red: 0.25, green: 0.75, blue: 0.125);
    final Map<String, Object?> out = <String, Object?>{
      'toString': '${const Color(0x12345678)}',
      'toString.p3': '$p3red',
      'fromRGBO': c(const Color.fromRGBO(10, 300, -1, 0.3)),
      'argb.half': const Color.from(alpha: 0.5 / 255, red: 1.5 / 255, green: 2.5 / 255, blue: 254.5 / 255).toARGB32(),
      'alphaFromOpacity': <int>[Color.getAlphaFromOpacity(0.5), Color.getAlphaFromOpacity(0.002), Color.getAlphaFromOpacity(1.5)],
      'withOpacity': c(const Color(0xFF102030).withOpacity(0.5)),
      'withAlpha': c(const Color(0x80102030).withAlpha(0x1ff)),
      'withValues.red': c(half.withValues(red: 0.9)),
      'p3.to.srgb': c(p3red.withValues(colorSpace: ColorSpace.sRGB)),
      'p3.to.ext': c(p3red.withValues(colorSpace: ColorSpace.extendedSRGB)),
      'srgb.to.p3': c(half.withValues(colorSpace: ColorSpace.displayP3)),
      'srgb.to.ext': c(half.withValues(colorSpace: ColorSpace.extendedSRGB)),
      'ext.to.srgb': c(ext.withValues(colorSpace: ColorSpace.sRGB)),
      'ext.to.p3': c(ext.withValues(colorSpace: ColorSpace.displayP3)),
      'ext.values.and.space': c(ext.withValues(alpha: 0.3, colorSpace: ColorSpace.sRGB)),
      'lerp.mixed': c(Color.lerp(half, p3red, 0.3)!),
      'lerp.null': c(Color.lerp(null, half, 0.4)!),
      'lerp.null.b': c(Color.lerp(half, null, 0.4)!),
      'alphaBlend': c(Color.alphaBlend(const Color(0x80336699), const Color(0x40FFCC00))),
      'luminance': <double>[const Color(0xFF336699).computeLuminance(), half.computeLuminance(), p3red.computeLuminance()],
      'opacity': const Color(0x80000000).opacity,
    };
    File('build/painting.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
