// Reference side of flutter-go's painting text pixel differential test:
// text painted by painting's TextPainter (alignment, direction, wrapping,
// ellipsis, styled spans, decorations, struts, scalers, placeholders) and
// the Flutter logo with its "Flutter" label (horizontal and stacked, at rest
// and mid-transition, with a margin, fading), drawn into 128x64 pictures on
// white, rasterized with Picture.toImage and read back as rawRgba. Fonts:
// flutter_tester's test fonts plus Roboto (loadFontFromList), which the
// logo's label asks for. Run with --enable-impeller; without it the test
// writes nothing. Output: build/painting_text_pixels.json as
// {scene: base64 bytes}.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

const int kWidth = 128;
const int kHeight = 64;

class Gap extends PlaceholderSpan {
  const Gap();

  @override
  void build(
    ui.ParagraphBuilder builder, {
    TextScaler textScaler = TextScaler.noScaling,
    List<PlaceholderDimensions>? dimensions,
  }) {
    final PlaceholderDimensions d = dimensions![builder.placeholderCount];
    builder.addPlaceholder(d.size.width, d.size.height, d.alignment,
        baseline: d.baseline, baselineOffset: d.baselineOffset);
  }

  @override
  bool visitChildren(InlineSpanVisitor visitor) => visitor(this);

  @override
  bool visitDirectChildren(InlineSpanVisitor visitor) => true;

  @override
  InlineSpan? getSpanForPositionVisitor(TextPosition position, Accumulator offset) {
    offset.increment(1);
    return null;
  }

  @override
  int? codeUnitAtVisitor(int index, Accumulator offset) {
    final bool expectedOffset = index - offset.value == 0;
    offset.increment(1);
    return expectedOffset ? PlaceholderSpan.placeholderCodeUnit : null;
  }

  @override
  RenderComparison compareTo(InlineSpan other) => RenderComparison.layout;

  @override
  bool debugAssertIsValid() => true;
}

void Function(Canvas) painted(
  InlineSpan text, {
  TextAlign textAlign = TextAlign.start,
  TextDirection textDirection = TextDirection.ltr,
  double maxWidth = 120,
  int? maxLines,
  String? ellipsis,
  StrutStyle? strutStyle,
  TextScaler textScaler = TextScaler.noScaling,
  List<PlaceholderDimensions>? placeholders,
  Offset offset = const Offset(4, 4),
}) {
  return (Canvas c) {
    final TextPainter p = TextPainter(
      text: text,
      textAlign: textAlign,
      textDirection: textDirection,
      maxLines: maxLines,
      ellipsis: ellipsis,
      strutStyle: strutStyle,
      textScaler: textScaler,
    );
    if (placeholders != null) {
      p.setPlaceholderDimensions(placeholders);
    }
    p.layout(maxWidth: maxWidth);
    p.paint(c, offset);
    p.dispose();
  };
}

void Function(Canvas) logo(FlutterLogoDecoration d, Offset o, Size s) =>
    (Canvas c) => d.createBoxPainter().paint(c, o, ImageConfiguration(size: s));

Map<String, void Function(Canvas c)> scenes() {
  const TextStyle black = TextStyle(fontSize: 10, color: Color(0xFF000000));
  return <String, void Function(Canvas c)>{
    'plain': painted(const TextSpan(text: 'Hello world', style: black), offset: const Offset(4.3, 6.6)),
    'wrap.center': painted(const TextSpan(text: 'one two three four five', style: black),
        textAlign: TextAlign.center, maxWidth: 70),
    'rtl.right': painted(const TextSpan(text: 'שלום עולם abc', style: black),
        textDirection: TextDirection.rtl, textAlign: TextAlign.right),
    'ellipsis': painted(const TextSpan(text: 'the quick brown fox jumps over', style: black),
        maxLines: 1, ellipsis: '…', maxWidth: 90),
    'spans': painted(const TextSpan(style: black, children: <InlineSpan>[
      TextSpan(text: 'red ', style: TextStyle(color: Color(0xFFFF0000), fontSize: 14)),
      TextSpan(text: 'bg', style: TextStyle(backgroundColor: Color(0xFF80C0FF))),
      TextSpan(text: ' under', style: TextStyle(decoration: TextDecoration.underline, decorationColor: Color(0xFF00A000))),
      TextSpan(text: ' big', style: TextStyle(fontSize: 18, color: Color(0x80000080))),
    ])),
    'strut': painted(const TextSpan(text: 'strut a\nb c', style: black),
        strutStyle: const StrutStyle(fontSize: 14, height: 1.4, forceStrutHeight: true)),
    'roboto.scaled': painted(const TextSpan(text: 'Roboto text', style: TextStyle(fontFamily: 'Roboto', fontSize: 12, color: Color(0xFF202020))),
        textScaler: const TextScaler.linear(1.5)),
    'placeholders': painted(const TextSpan(style: black, children: <InlineSpan>[
      TextSpan(text: 'a'), Gap(), TextSpan(text: 'b'), Gap(), TextSpan(text: 'c'),
    ]), placeholders: const <PlaceholderDimensions>[
      PlaceholderDimensions(size: Size(20, 8), alignment: ui.PlaceholderAlignment.middle),
      PlaceholderDimensions(size: Size(6, 20), alignment: ui.PlaceholderAlignment.bottom),
    ]),
    'logo.horizontal': logo(const FlutterLogoDecoration(style: FlutterLogoStyle.horizontal), Offset.zero, const Size(128, 64)),
    'logo.stacked': logo(const FlutterLogoDecoration(style: FlutterLogoStyle.stacked, textColor: Color(0xFF8000C0)),
        const Offset(32, 0), const Size(64, 64)),
    'logo.horizontal.transition': logo(
        FlutterLogoDecoration.lerp(const FlutterLogoDecoration(), const FlutterLogoDecoration(style: FlutterLogoStyle.horizontal), 0.5)!,
        Offset.zero, const Size(128, 64)),
    'logo.stacked.transition': logo(
        FlutterLogoDecoration.lerp(const FlutterLogoDecoration(), const FlutterLogoDecoration(style: FlutterLogoStyle.stacked), 0.4)!,
        const Offset(32, 0), const Size(64, 64)),
    'logo.horizontal.margin.fading': logo(
        FlutterLogoDecoration.lerp(null,
            const FlutterLogoDecoration(style: FlutterLogoStyle.horizontal, margin: EdgeInsets.fromLTRB(6, 4, 2, 8)), 0.7)!,
        const Offset(2, 3), const Size(120, 56)),
  };
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('painting_text_pixels', () async {
    if (Platform.environment['FLUTTER_TEST_IMPELLER'] != 'true') {
      return; // the reference is Impeller's; see the header
    }
    await ui.loadFontFromList(
      File('../../../engine/src/flutter/txt/third_party/fonts/Roboto-Regular.ttf').readAsBytesSync(),
      fontFamily: 'Roboto',
    );
    final Map<String, Object> out = <String, Object>{};
    for (final MapEntry<String, void Function(Canvas)> e in scenes().entries) {
      final ui.PictureRecorder rec = ui.PictureRecorder();
      final Canvas canvas = Canvas(rec);
      canvas.drawColor(const Color(0xFFFFFFFF), BlendMode.src);
      e.value(canvas);
      final ui.Image image = await rec.endRecording().toImage(kWidth, kHeight);
      final ByteData bytes = (await image.toByteData())!;
      out[e.key] = base64Encode(bytes.buffer.asUint8List());
    }
    File('build/painting_text_pixels.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
