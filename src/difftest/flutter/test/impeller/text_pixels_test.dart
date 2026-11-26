// Reference side of flutter-go's Canvas.drawParagraph pixel differential
// test: paragraphs (test fonts and Roboto) drawn with Canvas.drawParagraph,
// rasterized with Picture.toImage and read back with Image.toByteData
// (rawRgba). Run with --enable-impeller; without it the test writes
// nothing. Output: build/text_pixels.json as {scene: base64 bytes}.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

const int kWidth = 96;
const int kHeight = 48;

Paragraph para(
  String text, {
  ParagraphStyle? style,
  TextStyle? textStyle,
  double width = kWidth - 8.0,
  void Function(ParagraphBuilder b)? more,
}) {
  final ParagraphBuilder b = ParagraphBuilder(style ?? ParagraphStyle());
  b.pushStyle(textStyle ?? TextStyle(fontSize: 12, color: const Color(0xFF000000)));
  b.addText(text);
  more?.call(b);
  return b.build()..layout(ParagraphConstraints(width: width));
}

Map<String, void Function(Canvas c)> scenes() {
  return <String, void Function(Canvas c)>{
    'test_font': (Canvas c) => c.drawParagraph(para('Hello'), const Offset(4, 4)),
    'test_font.fractional': (Canvas c) => c.drawParagraph(para('Hi there'), const Offset(4.3, 6.6)),
    'test_font.wrap': (Canvas c) => c.drawParagraph(
      para('ab cd ef gh', textStyle: TextStyle(fontSize: 10, color: const Color(0xFF2040C0)), width: 40),
      const Offset(2, 2),
    ),
    'colors': (Canvas c) => c.drawParagraph(
      para(
        'red',
        textStyle: TextStyle(fontSize: 12, color: const Color(0xFFFF0000)),
        more: (ParagraphBuilder b) {
          b.pushStyle(TextStyle(color: const Color(0x8000AA00), fontSize: 16));
          b.addText('green');
          b.pop();
        },
      ),
      const Offset(2, 8),
    ),
    'decorations': (Canvas c) => c.drawParagraph(
      para(
        'deco',
        textStyle: TextStyle(
          fontSize: 14,
          color: const Color(0xFF000000),
          decoration: TextDecoration.combine(<TextDecoration>[TextDecoration.underline, TextDecoration.lineThrough]),
          decorationColor: const Color(0xFFFF00FF),
          decorationThickness: 2,
        ),
      ),
      const Offset(4, 10),
    ),
    'decorations.wavy': (Canvas c) => c.drawParagraph(
      para(
        'wavy',
        textStyle: TextStyle(
          fontSize: 14,
          color: const Color(0xFF000000),
          decoration: TextDecoration.overline,
          decorationStyle: TextDecorationStyle.wavy,
          decorationColor: const Color(0xFF0080FF),
        ),
      ),
      const Offset(4, 10),
    ),
    'background': (Canvas c) => c.drawParagraph(
      para('bg', textStyle: TextStyle(fontSize: 16, background: Paint()..color = const Color(0xFFFFCC00))),
      const Offset(6, 6),
    ),
    'shadow': (Canvas c) => c.drawParagraph(
      para(
        'sh',
        textStyle: TextStyle(
          fontSize: 16,
          color: const Color(0xFF000000),
          shadows: const <Shadow>[Shadow(color: Color(0xFFFF0000), offset: Offset(3, 2))],
        ),
      ),
      const Offset(6, 6),
    ),
    'foreground.gradient': (Canvas c) => c.drawParagraph(
      para(
        'grad',
        textStyle: TextStyle(
          fontSize: 18,
          foreground: Paint()
            ..shader = Gradient.linear(
              Offset.zero,
              const Offset(72, 0),
              const <Color>[Color(0xFFFF0000), Color(0xFF0000FF)],
            ),
        ),
      ),
      const Offset(4, 4),
    ),
    'foreground.stroke': (Canvas c) => c.drawParagraph(
      para(
        'st',
        textStyle: TextStyle(
          fontSize: 24,
          foreground: Paint()
            ..style = PaintingStyle.stroke
            ..strokeWidth = 2
            ..color = const Color(0xFF008000),
        ),
      ),
      const Offset(4, 4),
    ),
    'transform': (Canvas c) {
      c.translate(20, 4);
      c.rotate(0.3);
      c.scale(1.5, 1);
      c.drawParagraph(para('rot', textStyle: TextStyle(fontSize: 10, color: const Color(0xFF000000))), Offset.zero);
    },
    'rtl_ellipsis': (Canvas c) => c.drawParagraph(
      para(
        'שלום עולם טוב',
        style: ParagraphStyle(textDirection: TextDirection.rtl, maxLines: 1, ellipsis: '...'),
        textStyle: TextStyle(fontSize: 8, color: const Color(0xFF000000)),
        width: 60,
      ),
      const Offset(2, 2),
    ),
    'roboto': (Canvas c) => c.drawParagraph(
      para('Agile', textStyle: TextStyle(fontSize: 20, fontFamily: 'Roboto', color: const Color(0xFF000000))),
      const Offset(4, 4),
    ),
    'roboto.small': (Canvas c) => c.drawParagraph(
      para(
        'Small text, Roboto at 9px',
        textStyle: TextStyle(fontSize: 9, fontFamily: 'Roboto', color: const Color(0xFF203040)),
      ),
      const Offset(2, 2),
    ),
  };
}

void main() {
  test('text_pixels', () async {
    if (Platform.environment['FLUTTER_TEST_IMPELLER'] != 'true') {
      return; // the reference is Impeller's; see the header
    }
    await loadFontFromList(
      File('../../../engine/src/flutter/txt/third_party/fonts/Roboto-Regular.ttf').readAsBytesSync(),
      fontFamily: 'Roboto',
    );
    final Map<String, Object> out = <String, Object>{};
    for (final MapEntry<String, void Function(Canvas)> e in scenes().entries) {
      final PictureRecorder rec = PictureRecorder();
      final Canvas canvas = Canvas(rec);
      canvas.drawColor(const Color(0xFFFFFFFF), BlendMode.src);
      e.value(canvas);
      final Image image = await rec.endRecording().toImage(kWidth, kHeight);
      final ByteData bytes = (await image.toByteData())!;
      out[e.key] = base64Encode(bytes.buffer.asUint8List());
    }
    File('build/text_pixels.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
