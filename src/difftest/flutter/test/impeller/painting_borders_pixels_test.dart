// Reference side of flutter-go's painting borders pixel differential test:
// each border of painting (Border, BorderDirectional, the shape borders,
// compound borders and lerp intermediates) painted into a 64×64 picture,
// rasterized with Picture.toImage and read back as rawRgba. Run with
// --enable-impeller (Impeller on Vulkan/SwiftShader); without it the test
// writes nothing (see painting_pixels_test.dart).
// Output: build/painting_borders_pixels.json as {scene: base64 bytes}.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

const int kSize = 64;
const Rect box = Rect.fromLTRB(10, 12, 54, 50);

BorderSide bs(int color, double width, [double strokeAlign = BorderSide.strokeAlignInside]) =>
    BorderSide(color: Color(color), width: width, strokeAlign: strokeAlign);

Map<String, void Function(Canvas c)> scenes() {
  final red = bs(0xFFE53935, 4);
  final green = bs(0xFF43A047, 6);
  final blue = bs(0xFF1E88E5, 2);
  final black = bs(0xFF000000, 1);
  void shape(Canvas c, ShapeBorder s, {TextDirection td = TextDirection.ltr, bool fill = false}) {
    if (fill) {
      s.paintInterior(c, box, Paint()..color = const Color(0x8000BCD4), textDirection: td);
    }
    s.paint(c, box, textDirection: td);
  }

  return <String, void Function(Canvas c)>{
    'border.uniform': (Canvas c) => shape(c, Border.all(color: const Color(0xFFE53935), width: 4)),
    'border.uniform.center': (Canvas c) =>
        shape(c, Border.all(color: const Color(0xFFE53935), width: 6, strokeAlign: BorderSide.strokeAlignCenter)),
    'border.uniform.outside': (Canvas c) =>
        shape(c, Border.all(color: const Color(0xFFE53935), width: 5, strokeAlign: BorderSide.strokeAlignOutside)),
    'border.uniform.radius': (Canvas c) => Border.all(color: const Color(0xFF1E88E5), width: 5)
        .paint(c, box, borderRadius: const BorderRadius.all(Radius.circular(12))),
    'border.uniform.circle': (Canvas c) =>
        Border.all(color: const Color(0xFF1E88E5), width: 5).paint(c, box, shape: BoxShape.circle),
    'border.hairline': (Canvas c) => shape(c, Border.all(color: const Color(0xFF000000), width: 0)),
    'border.nonuniform': (Canvas c) => shape(c, Border(top: red, right: green, bottom: blue, left: black)),
    'border.nonuniform.radius': (Canvas c) => Border(
      top: bs(0xFF8E24AA, 2),
      right: bs(0xFF8E24AA, 8),
      bottom: bs(0xFF8E24AA, 4),
      left: bs(0xFF8E24AA, 1),
    ).paint(c, box, borderRadius: const BorderRadius.only(topLeft: Radius.circular(16), bottomRight: Radius.elliptical(20, 8))),
    'border.nonuniform.circle': (Canvas c) => Border(
      top: bs(0xFF8E24AA, 2),
      right: bs(0xFF8E24AA, 8, BorderSide.strokeAlignCenter),
      bottom: bs(0xFF8E24AA, 4),
      left: bs(0xFF8E24AA, 1),
    ).paint(c, box, shape: BoxShape.circle),
    'border.someNone': (Canvas c) => shape(c, Border(top: red, left: green.copyWith(style: BorderStyle.none), bottom: blue)),
    'directional.ltr': (Canvas c) => shape(c, BorderDirectional(top: red, start: green, end: blue, bottom: black)),
    'directional.rtl': (Canvas c) =>
        shape(c, BorderDirectional(top: red, start: green, end: blue, bottom: black), td: TextDirection.rtl),
    'circle': (Canvas c) => shape(c, CircleBorder(side: red), fill: true),
    'circle.eccentric': (Canvas c) => shape(c, CircleBorder(side: green, eccentricity: 0.5), fill: true),
    'circle.outside': (Canvas c) => shape(c, CircleBorder(side: bs(0xFF1E88E5, 5, BorderSide.strokeAlignOutside))),
    'oval': (Canvas c) => shape(c, OvalBorder(side: blue), fill: true),
    'rrect': (Canvas c) =>
        shape(c, RoundedRectangleBorder(side: red, borderRadius: const BorderRadius.all(Radius.circular(10))), fill: true),
    'rrect.hairline': (Canvas c) =>
        shape(c, RoundedRectangleBorder(side: bs(0xFF000000, 0), borderRadius: const BorderRadius.all(Radius.circular(10)))),
    'rrect.directional.rtl': (Canvas c) => shape(
      c,
      RoundedRectangleBorder(
        side: bs(0xFF43A047, 3, BorderSide.strokeAlignCenter),
        borderRadius: const BorderRadiusDirectional.only(topStart: Radius.circular(18), bottomEnd: Radius.circular(6)),
      ),
      td: TextDirection.rtl,
    ),
    'superellipse': (Canvas c) => shape(
      c,
      RoundedSuperellipseBorder(side: red, borderRadius: const BorderRadius.all(Radius.circular(16))),
      fill: true,
    ),
    'superellipse.zero': (Canvas c) => shape(c, RoundedSuperellipseBorder(side: green), fill: true),
    'stadium': (Canvas c) => shape(c, StadiumBorder(side: blue), fill: true),
    'stadium.outside': (Canvas c) => shape(c, StadiumBorder(side: bs(0xFFE53935, 4, BorderSide.strokeAlignOutside))),
    'beveled': (Canvas c) =>
        shape(c, BeveledRectangleBorder(side: red, borderRadius: const BorderRadius.all(Radius.circular(12)))),
    'continuous': (Canvas c) =>
        shape(c, ContinuousRectangleBorder(side: green, borderRadius: const BorderRadius.all(Radius.circular(20)))),
    'star': (Canvas c) => shape(c, StarBorder(side: blue)),
    'star.rounded': (Canvas c) => shape(
      c,
      StarBorder(side: red, points: 6, innerRadiusRatio: 0.5, pointRounding: 0.4, valleyRounding: 0.3, rotation: 15, squash: 0.5),
    ),
    'star.polygon': (Canvas c) => shape(c, StarBorder.polygon(side: green, sides: 5, pointRounding: 0.2, squash: 1)),
    'linear.start': (Canvas c) => shape(c, LinearBorder.start(side: red, size: 0.5, alignment: -1)),
    'linear.start.rtl': (Canvas c) => shape(c, LinearBorder.start(side: red, size: 0.5), td: TextDirection.rtl),
    'linear.topBottom': (Canvas c) => shape(
      c,
      LinearBorder(side: blue, top: const LinearBorderEdge(size: 0.75, alignment: 1), bottom: const LinearBorderEdge()),
    ),
    'compound': (Canvas c) => shape(c, Border.all(color: const Color(0xFFE53935), width: 3) + Border.all(color: const Color(0xFF1E88E5), width: 4)),
    'lerp.rrect.circle': (Canvas c) => shape(
      c,
      ShapeBorder.lerp(RoundedRectangleBorder(side: red, borderRadius: const BorderRadius.all(Radius.circular(4))), CircleBorder(side: blue), 0.4)!,
      fill: true,
    ),
    'lerp.superellipse.oval': (Canvas c) => shape(
      c,
      ShapeBorder.lerp(RoundedSuperellipseBorder(side: red, borderRadius: const BorderRadius.all(Radius.circular(8))), OvalBorder(side: blue), 0.6)!,
      fill: true,
    ),
    'lerp.stadium.circle': (Canvas c) =>
        shape(c, ShapeBorder.lerp(StadiumBorder(side: green), const CircleBorder(eccentricity: 0.3), 0.5)!, fill: true),
    'lerp.stadium.rrect': (Canvas c) => shape(
      c,
      ShapeBorder.lerp(StadiumBorder(side: green), RoundedRectangleBorder(side: red), 0.3)!,
      fill: true,
    ),
    'lerp.star.rrect': (Canvas c) => shape(
      c,
      ShapeBorder.lerp(StarBorder(side: blue), const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(8))), 0.5)!,
    ),
  };
}

void main() {
  test('painting_borders_pixels', () async {
    if (Platform.environment['FLUTTER_TEST_IMPELLER'] != 'true') {
      return; // the reference is Impeller's; see the header
    }
    final Map<String, Object> out = <String, Object>{};
    for (final MapEntry<String, void Function(Canvas)> e in scenes().entries) {
      final rec = ui.PictureRecorder();
      e.value(Canvas(rec));
      final ui.Image image = await rec.endRecording().toImage(kSize, kSize);
      final ByteData bytes = (await image.toByteData())!;
      out[e.key] = base64Encode(bytes.buffer.asUint8List());
    }
    File('build/painting_borders_pixels.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
