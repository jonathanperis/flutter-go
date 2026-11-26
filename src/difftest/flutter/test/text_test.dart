// Reference side of flutter-go's dart:ui text differential test.
//
// Paragraphs: the scenes of ../text_scenes.json (shared with Go) are built,
// laid out at each width and queried exhaustively: metrics, line metrics,
// boxes for every one-code-unit range and for whole ranges under every
// height/width style, placeholder boxes, glyph info at every offset and
// closest to a grid of points, positions for the grid, word and line
// boundaries at every offset with both affinities, line numbers. Fonts: the
// test fonts flutter_tester registers (FlutterTest, Ahem, Cough) plus
// Roboto, Noto Naskh Arabic and Roboto Slab (variable) loaded with
// loadFontFromList from the engine's txt/third_party/fonts.
//
// API: toString/==/encoding of TextStyle, ParagraphStyle, StrutStyle and the
// small value types, and the errors the API throws.
//
// Output: build/text.json, or build/text_impeller.json when run with
// --enable-impeller (layout is SkParagraph's in both; Go compares both).
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

Object n(double v) => v.isFinite ? v : '$v';

String err(void Function() f) {
  try {
    f();
  } on AssertionError catch (e) {
    return 'AssertionError: ${e.message}';
  } catch (e) {
    return '$e';
  }
  return 'no error';
}

Color? color(Object? v) => v == null ? null : Color(v as int);

Locale? locale(Object? v) {
  if (v == null) {
    return null;
  }
  final List<Object?> l = v as List<Object?>;
  return l.length > 1 ? Locale(l[0]! as String, l[1]! as String) : Locale(l[0]! as String);
}

T? enumOf<T extends Enum>(List<T> values, Object? name) =>
    name == null ? null : values.byName(name as String);

double? d(Object? v) => v == null ? null : (v as num).toDouble();

Paint? paint(Object? v) => v == null ? null : (Paint()..color = Color(v as int));

TextStyle textStyle(Map<String, Object?> s) {
  return TextStyle(
    color: color(s['color']),
    decoration: s['decoration'] == null
        ? null
        : TextDecoration.combine(<TextDecoration>[
            for (final Object? x in s['decoration']! as List<Object?>)
              switch (x) {
                'underline' => TextDecoration.underline,
                'overline' => TextDecoration.overline,
                'lineThrough' => TextDecoration.lineThrough,
                _ => TextDecoration.none,
              },
          ]),
    decorationColor: color(s['decorationColor']),
    decorationStyle: enumOf(TextDecorationStyle.values, s['decorationStyle']),
    decorationThickness: d(s['decorationThickness']),
    fontWeight: s['fontWeight'] == null ? null : FontWeight(s['fontWeight']! as int),
    fontStyle: enumOf(FontStyle.values, s['fontStyle']),
    textBaseline: enumOf(TextBaseline.values, s['textBaseline']),
    fontFamily: s['fontFamily'] as String?,
    fontFamilyFallback: (s['fontFamilyFallback'] as List<Object?>?)?.cast<String>(),
    fontSize: d(s['fontSize']),
    letterSpacing: d(s['letterSpacing']),
    wordSpacing: d(s['wordSpacing']),
    height: d(s['height']),
    leadingDistribution: enumOf(TextLeadingDistribution.values, s['leadingDistribution']),
    locale: locale(s['locale']),
    background: paint(s['background']),
    foreground: paint(s['foreground']),
    shadows: (s['shadows'] as List<Object?>?)
        ?.map((Object? x) {
          final List<Object?> l = x! as List<Object?>;
          return Shadow(
            color: Color(l[0]! as int),
            offset: Offset(d(l[1])!, d(l[2])!),
            blurRadius: d(l[3])!,
          );
        })
        .toList(),
    fontFeatures: (s['fontFeatures'] as List<Object?>?)
        ?.cast<List<Object?>>()
        .map((List<Object?> x) => FontFeature(x[0]! as String, x[1]! as int))
        .toList(),
    fontVariations: (s['fontVariations'] as List<Object?>?)
        ?.cast<List<Object?>>()
        .map((List<Object?> x) => FontVariation(x[0]! as String, d(x[1])!))
        .toList(),
  );
}

ParagraphStyle paragraphStyle(Map<String, Object?> p) {
  final Map<String, Object?>? thb = p['textHeightBehavior'] as Map<String, Object?>?;
  final Map<String, Object?>? strut = p['strut'] as Map<String, Object?>?;
  return ParagraphStyle(
    textAlign: enumOf(TextAlign.values, p['textAlign']),
    textDirection: enumOf(TextDirection.values, p['textDirection']),
    maxLines: p['maxLines'] as int?,
    fontFamily: p['fontFamily'] as String?,
    fontSize: d(p['fontSize']),
    height: d(p['height']),
    textHeightBehavior: thb == null
        ? null
        : TextHeightBehavior(
            applyHeightToFirstAscent: thb['first']! as bool,
            applyHeightToLastDescent: thb['last']! as bool,
            leadingDistribution:
                enumOf(TextLeadingDistribution.values, thb['leading']) ??
                TextLeadingDistribution.proportional,
          ),
    fontWeight: p['fontWeight'] == null ? null : FontWeight(p['fontWeight']! as int),
    fontStyle: enumOf(FontStyle.values, p['fontStyle']),
    strutStyle: strut == null
        ? null
        : StrutStyle(
            fontFamily: strut['fontFamily'] as String?,
            fontSize: d(strut['fontSize']),
            height: d(strut['height']),
            leadingDistribution: enumOf(TextLeadingDistribution.values, strut['leadingDistribution']),
            leading: d(strut['leading']),
            forceStrutHeight: strut['forceStrutHeight'] as bool?,
          ),
    ellipsis: p['ellipsis'] as String?,
    locale: locale(p['locale']),
    hyphens: enumOf(Hyphens.values, p['hyphens']),
  );
}

// Builds a scene's paragraph; returns it with its length in code units.
(Paragraph, int) build(Map<String, Object?> scene) {
  final ParagraphBuilder b = ParagraphBuilder(
    paragraphStyle((scene['paragraph'] as Map<String, Object?>?) ?? <String, Object?>{}),
  );
  var length = 0;
  for (final Object? opObject in scene['ops']! as List<Object?>) {
    final List<Object?> op = opObject! as List<Object?>;
    switch (op[0]) {
      case 'push':
        b.pushStyle(textStyle(op[1]! as Map<String, Object?>));
      case 'pop':
        b.pop();
      case 'text':
        b.addText(op[1]! as String);
        length += (op[1]! as String).length;
      case 'placeholder':
        final Map<String, Object?> o = op.length > 4 ? op[4]! as Map<String, Object?> : <String, Object?>{};
        b.addPlaceholder(
          d(op[1])!,
          d(op[2])!,
          PlaceholderAlignment.values.byName(op[3]! as String),
          scale: d(o['scale']) ?? 1.0,
          baselineOffset: d(o['baselineOffset']),
          baseline: enumOf(TextBaseline.values, o['baseline']),
        );
        length += 1;
    }
  }
  return (b.build(), length);
}

List<Object> box(TextBox b) => <Object>[n(b.left), n(b.top), n(b.right), n(b.bottom), b.direction.index];
List<Object> boxes(List<TextBox> l) => l.map(box).toList();
Object? glyph(GlyphInfo? g) => g == null
    ? null
    : <Object>[
        n(g.graphemeClusterLayoutBounds.left),
        n(g.graphemeClusterLayoutBounds.top),
        n(g.graphemeClusterLayoutBounds.right),
        n(g.graphemeClusterLayoutBounds.bottom),
        g.graphemeClusterCodeUnitRange.start,
        g.graphemeClusterCodeUnitRange.end,
        g.writingDirection.index,
      ];
Object? line(LineMetrics? m) => m == null
    ? null
    : <Object>[
        m.hardBreak,
        n(m.ascent),
        n(m.descent),
        n(m.unscaledAscent),
        n(m.height),
        n(m.width),
        n(m.left),
        n(m.baseline),
        m.lineNumber,
      ];

// The query grid: x and y from -5 past the paragraph's extent.
List<double> grid(double extent, double step) {
  final List<double> out = <double>[-5, 0];
  if (!extent.isFinite) {
    extent = 300;
  }
  for (double v = 1.5; v <= extent + 5; v += step) {
    out.add(v);
  }
  return out;
}

Map<String, Object?> dump(Paragraph p, int length) {
  final Map<String, Object?> out = <String, Object?>{};
  out['metrics'] = <Object>[
    n(p.width),
    n(p.height),
    n(p.longestLine),
    n(p.minIntrinsicWidth),
    n(p.maxIntrinsicWidth),
    n(p.alphabeticBaseline),
    n(p.ideographicBaseline),
    p.didExceedMaxLines,
    p.numberOfLines,
  ];
  out['lines'] = p.computeLineMetrics().map(line).toList();
  out['lineAt'] = <Object?>[for (int i = -1; i <= p.numberOfLines; i++) line(p.getLineMetricsAt(i))];
  out['lineNumberAt'] = <Object?>[for (int i = -1; i <= length + 1; i++) p.getLineNumberAt(i)];
  out['boxes1'] = <Object>[for (int i = 0; i < length; i++) boxes(p.getBoxesForRange(i, i + 1))];
  final List<List<int>> ranges = <List<int>>[
    <int>[0, length],
    <int>[length ~/ 3, length - length ~/ 3],
    <int>[length - 2, length + 3],
    <int>[3, 1],
  ];
  final Map<String, Object> styled = <String, Object>{};
  for (final List<int> r in ranges) {
    for (final BoxHeightStyle hs in BoxHeightStyle.values) {
      for (final BoxWidthStyle ws in BoxWidthStyle.values) {
        styled['${r[0]},${r[1]},${hs.index},${ws.index}'] = boxes(
          p.getBoxesForRange(r[0], r[1], boxHeightStyle: hs, boxWidthStyle: ws),
        );
      }
    }
  }
  out['boxes'] = styled;
  out['placeholders'] = boxes(p.getBoxesForPlaceholders());
  out['glyphAt'] = <Object?>[for (int i = -1; i <= length; i++) glyph(p.getGlyphInfoAt(i))];
  final List<Object> positions = <Object>[];
  final List<Object?> closest = <Object?>[];
  for (final double y in grid(p.height, 7)) {
    for (final double x in grid(p.width, 6.25)) {
      final TextPosition pos = p.getPositionForOffset(Offset(x, y));
      positions.add(<Object>[pos.offset, pos.affinity.index]);
      closest.add(glyph(p.getClosestGlyphInfoForOffset(Offset(x, y))));
    }
  }
  out['positions'] = positions;
  out['closest'] = closest;
  final List<Object> words = <Object>[];
  final List<Object> lineBounds = <Object>[];
  for (int i = 0; i <= length; i++) {
    for (final TextAffinity a in TextAffinity.values) {
      final TextRange w = p.getWordBoundary(TextPosition(offset: i, affinity: a));
      words.add(<int>[w.start, w.end]);
      final TextRange l = p.getLineBoundary(TextPosition(offset: i, affinity: a));
      lineBounds.add(<int>[l.start, l.end]);
    }
  }
  out['words'] = words;
  out['lineBounds'] = lineBounds;
  out['string'] = '$p';
  return out;
}

String fontPath(String name) => '../../../engine/src/flutter/txt/third_party/fonts/$name';

Future<void> loadFonts() async {
  await loadFontFromList(File(fontPath('Roboto-Regular.ttf')).readAsBytesSync(), fontFamily: 'Roboto');
  await loadFontFromList(
    File(fontPath('NotoNaskhArabic-Regular.ttf')).readAsBytesSync(),
    fontFamily: 'NotoNaskhArabic',
  );
  await loadFontFromList(
    File(fontPath('RobotoSlab-VariableFont_wght.ttf')).readAsBytesSync(),
    fontFamily: 'RobotoSlab',
  );
}

Map<String, Object?> apiCases() {
  final Map<String, Object?> out = <String, Object?>{};
  // FontWeight.
  out['fontWeight.values'] = FontWeight.values.map((FontWeight w) => '$w').toList();
  out['fontWeight.odd'] = <String>['${const FontWeight(1)}', '${const FontWeight(450)}', '${const FontWeight(1000)}'];
  // ignore: deprecated_member_use
  out['fontWeight.index'] = <int>[const FontWeight(1).index, const FontWeight(450).index, const FontWeight(1000).index];
  out['fontWeight.lerp'] = <String>[
    for (final double t in <double>[-1, 0, 0.25, 0.5, 0.55, 1, 2])
      '${FontWeight.lerp(FontWeight.w100, FontWeight.w900, t)}|${FontWeight.lerp(null, FontWeight.w100, t)}|${FontWeight.lerp(FontWeight.w300, null, t)}|${FontWeight.lerp(null, null, t)}',
  ];
  out['fontWeight.errors'] = <String>[err(() => FontWeight(0)), err(() => FontWeight(1001))];
  out['fontWeight.equal'] = <bool>[
    const FontWeight(400) == FontWeight.normal,
    FontWeight(700) == FontWeight.bold,
    FontWeight(701) == FontWeight.bold,
  ];
  // FontFeature.
  out['fontFeature.ctors'] = <String>[
    '${const FontFeature('liga')}',
    '${const FontFeature('liga', 0)}',
    '${const FontFeature.enable('smcp')}',
    '${const FontFeature.disable('kern')}',
    '${const FontFeature.alternative(3)}',
    '${const FontFeature.alternativeFractions()}',
    '${const FontFeature.contextualAlternates()}',
    '${const FontFeature.caseSensitiveForms()}',
    '${FontFeature.characterVariant(7)}',
    '${FontFeature.characterVariant(42)}',
    '${const FontFeature.denominator()}',
    '${const FontFeature.fractions()}',
    '${const FontFeature.historicalForms()}',
    '${const FontFeature.historicalLigatures()}',
    '${const FontFeature.liningFigures()}',
    '${const FontFeature.localeAware()}',
    '${const FontFeature.localeAware(enable: false)}',
    '${const FontFeature.notationalForms()}',
    '${const FontFeature.notationalForms(4)}',
    '${const FontFeature.numerators()}',
    '${const FontFeature.oldstyleFigures()}',
    '${const FontFeature.ordinalForms()}',
    '${const FontFeature.proportionalFigures()}',
    '${const FontFeature.randomize()}',
    '${const FontFeature.stylisticAlternates()}',
    '${const FontFeature.scientificInferiors()}',
    '${FontFeature.stylisticSet(3)}',
    '${FontFeature.stylisticSet(20)}',
    '${const FontFeature.subscripts()}',
    '${const FontFeature.superscripts()}',
    '${const FontFeature.swash()}',
    '${const FontFeature.swash(2)}',
    '${const FontFeature.tabularFigures()}',
    '${const FontFeature.slashedZero()}',
  ];
  out['fontFeature.errors'] = <String>[
    err(() => FontFeature('abc')),
    err(() => FontFeature('abcde')),
    err(() => FontFeature('liga', -1)),
    err(() => FontFeature.characterVariant(0)),
    err(() => FontFeature.characterVariant(100)),
    err(() => FontFeature.stylisticSet(21)),
  ];
  out['fontFeature.equal'] = <bool>[
    const FontFeature('liga') == const FontFeature.enable('liga'),
    const FontFeature('liga') == const FontFeature('liga', 2),
  ];
  // FontVariation.
  out['fontVariation.ctors'] = <String>[
    '${const FontVariation('wght', 500)}',
    '${const FontVariation('wdth', -1.5)}',
    '${const FontVariation.italic(0.5)}',
    '${const FontVariation.opticalSize(12)}',
    '${const FontVariation.slant(-10)}',
    '${const FontVariation.width(75)}',
    '${const FontVariation.weight(1000)}',
  ];
  out['fontVariation.errors'] = <String>[
    err(() => FontVariation('wg', 1)),
    err(() => FontVariation('wght', 32768)),
    err(() => FontVariation('wght', -32769)),
    err(() => FontVariation.italic(2)),
    err(() => FontVariation.slant(90)),
    err(() => FontVariation.weight(0)),
  ];
  out['fontVariation.lerp'] = <String>[
    for (final double t in <double>[-1, 0, 0.3, 0.5, 1, 1e6])
      '${FontVariation.lerp(const FontVariation('wght', 100), const FontVariation('wght', 900), t)}|'
          '${FontVariation.lerp(const FontVariation('wght', 100), const FontVariation('wdth', 900), t)}|'
          '${FontVariation.lerp(null, const FontVariation('wdth', 9), t)}|'
          '${FontVariation.lerp(null, null, t)}',
  ];
  // TextDecoration.
  out['decoration'] = <Object>[
    '${TextDecoration.none}',
    '${TextDecoration.underline}',
    '${TextDecoration.overline}',
    '${TextDecoration.lineThrough}',
    '${TextDecoration.combine(<TextDecoration>[TextDecoration.underline, TextDecoration.lineThrough])}',
    '${TextDecoration.combine(<TextDecoration>[TextDecoration.underline, TextDecoration.overline, TextDecoration.lineThrough])}',
    '${TextDecoration.combine(<TextDecoration>[])}',
    TextDecoration.combine(<TextDecoration>[TextDecoration.underline, TextDecoration.overline]).contains(TextDecoration.overline),
    TextDecoration.underline.contains(TextDecoration.overline),
    TextDecoration.underline.contains(TextDecoration.none),
    TextDecoration.combine(<TextDecoration>[TextDecoration.underline, TextDecoration.overline]).maskValue,
    TextDecoration.combine(<TextDecoration>[TextDecoration.underline]) == TextDecoration.underline,
  ];
  // TextHeightBehavior.
  out['textHeightBehavior'] = <Object>[
    '${const TextHeightBehavior()}',
    '${const TextHeightBehavior(applyHeightToFirstAscent: false, leadingDistribution: TextLeadingDistribution.even)}',
    const TextHeightBehavior() == const TextHeightBehavior(applyHeightToLastDescent: true),
    const TextHeightBehavior() == const TextHeightBehavior(applyHeightToLastDescent: false),
  ];
  // Enums.
  out['enums'] = <String>[
    ...FontStyle.values.map((FontStyle e) => '$e'),
    ...TextAlign.values.map((TextAlign e) => '$e'),
    ...Hyphens.values.map((Hyphens e) => '$e'),
    ...TextBaseline.values.map((TextBaseline e) => '$e'),
    ...TextDecorationStyle.values.map((TextDecorationStyle e) => '$e'),
    ...TextLeadingDistribution.values.map((TextLeadingDistribution e) => '$e'),
    ...BoxHeightStyle.values.map((BoxHeightStyle e) => '$e'),
    ...BoxWidthStyle.values.map((BoxWidthStyle e) => '$e'),
    ...PlaceholderAlignment.values.map((PlaceholderAlignment e) => '$e'),
  ];
  // Small value types.
  out['valueTypes'] = <Object>[
    '${const TextBox.fromLTRBD(1.25, 2.05, 3.35, 4.45, TextDirection.rtl)}',
    const TextBox.fromLTRBD(1, 2, 3, 4, TextDirection.rtl).start,
    const TextBox.fromLTRBD(1, 2, 3, 4, TextDirection.rtl).end,
    const TextBox.fromLTRBD(1, 2, 3, 4, TextDirection.ltr).start,
    '${const TextBox.fromLTRBD(1, 2, 3, 4, TextDirection.ltr).toRect()}',
    '${const ParagraphConstraints(width: 12.5)}',
    '${const ParagraphConstraints(width: double.infinity)}',
    '${LineMetrics(hardBreak: true, ascent: 1, descent: 2.5, unscaledAscent: 3, height: 4, width: 5, left: 6, baseline: 7, lineNumber: 8)}',
    '${GlyphInfo(const Rect.fromLTRB(1, 2, 3, 4), const TextRange(start: 5, end: 6), TextDirection.rtl)}',
    GlyphInfo(const Rect.fromLTRB(1, 2, 3, 4), const TextRange(start: 5, end: 6), TextDirection.rtl) ==
        GlyphInfo(const Rect.fromLTRB(1, 2, 3, 4), const TextRange(start: 5, end: 6), TextDirection.rtl),
  ];
  // TextStyle.
  final Paint bg = Paint()..color = const Color(0xFF00FF00);
  final List<TextStyle> styles = <TextStyle>[
    TextStyle(),
    TextStyle(color: const Color(0xFF123456), fontSize: 12.5),
    TextStyle(
      decoration: TextDecoration.combine(<TextDecoration>[TextDecoration.underline, TextDecoration.overline]),
      decorationColor: const Color(0x80FF0000),
      decorationStyle: TextDecorationStyle.dashed,
      decorationThickness: 2,
      fontWeight: const FontWeight(450),
      fontStyle: FontStyle.italic,
      textBaseline: TextBaseline.ideographic,
      fontFamily: 'Roboto',
      fontFamilyFallback: <String>['Noto', 'Emoji'],
      fontSize: 10,
      letterSpacing: 1.5,
      wordSpacing: -2,
      height: 1.25,
      leadingDistribution: TextLeadingDistribution.even,
      locale: const Locale('fr', 'CA'),
      background: bg,
      foreground: Paint()..color = const Color(0xFF0000FF),
      shadows: const <Shadow>[Shadow(color: Color(0xFF00FF00), offset: Offset(1, 2), blurRadius: 3)],
      fontFeatures: const <FontFeature>[FontFeature.tabularFigures(), FontFeature('liga', 0)],
      fontVariations: const <FontVariation>[FontVariation('wght', 300.5)],
    ),
    TextStyle(height: kTextHeightNone),
    TextStyle(fontFamily: ''),
    TextStyle(fontFamilyFallback: <String>[]),
    TextStyle(fontFamilyFallback: <String>['Only']),
    TextStyle(shadows: const <Shadow>[], fontFeatures: const <FontFeature>[], fontVariations: const <FontVariation>[]),
    TextStyle(background: bg),
    TextStyle(background: bg),
    TextStyle(background: Paint()..color = const Color(0xFF00FF00)),
    TextStyle(fontSize: 12.5, color: const Color(0xFF123456)),
  ];
  out['textStyle.string'] = styles.map((TextStyle s) => '$s').toList();
  out['textStyle.equal'] = <Object>[
    for (int i = 0; i < styles.length; i++)
      <bool>[for (int j = 0; j < styles.length; j++) styles[i] == styles[j]],
  ];
  out['textStyle.errors'] = <String>[
    err(() => TextStyle(color: const Color(0xFF000000), foreground: Paint())),
  ];
  // Pushing a style records the paragraph's leading distribution in it.
  final TextStyle pushed = TextStyle(fontSize: 10);
  final TextStyle twin = TextStyle(fontSize: 10);
  final ParagraphBuilder evenBuilder = ParagraphBuilder(
    ParagraphStyle(textHeightBehavior: const TextHeightBehavior(leadingDistribution: TextLeadingDistribution.even)),
  );
  out['textStyle.pushMutates.before'] = pushed == twin;
  evenBuilder.pushStyle(pushed);
  out['textStyle.pushMutates.after'] = <Object>[pushed == twin, '$pushed'];
  evenBuilder.build().dispose();
  // ParagraphStyle.
  final StrutStyle strut = StrutStyle(fontSize: 10, height: 1.5);
  final List<ParagraphStyle> pstyles = <ParagraphStyle>[
    ParagraphStyle(),
    ParagraphStyle(
      textAlign: TextAlign.justify,
      textDirection: TextDirection.rtl,
      maxLines: 3,
      fontFamily: 'Roboto',
      fontSize: 13,
      height: 1.5,
      textHeightBehavior: const TextHeightBehavior(
        applyHeightToLastDescent: false,
        leadingDistribution: TextLeadingDistribution.even,
      ),
      fontWeight: FontWeight.w600,
      fontStyle: FontStyle.italic,
      strutStyle: strut,
      ellipsis: '…',
      locale: const Locale('en', 'US'),
      hyphens: Hyphens.hidden,
    ),
    ParagraphStyle(height: kTextHeightNone),
    ParagraphStyle(height: 0.5),
    ParagraphStyle(maxLines: 0, ellipsis: ''),
    ParagraphStyle(strutStyle: StrutStyle(fontSize: 10, height: 1.5)),
    ParagraphStyle(strutStyle: strut),
    ParagraphStyle(strutStyle: StrutStyle()),
    ParagraphStyle(textHeightBehavior: const TextHeightBehavior()),
  ];
  out['paragraphStyle.string'] = pstyles.map((ParagraphStyle s) => '$s').toList();
  out['paragraphStyle.equal'] = <Object>[
    for (int i = 0; i < pstyles.length; i++)
      <bool>[for (int j = 0; j < pstyles.length; j++) pstyles[i] == pstyles[j]],
  ];
  // StrutStyle.
  final List<StrutStyle> struts = <StrutStyle>[
    StrutStyle(),
    StrutStyle(height: kTextHeightNone),
    StrutStyle(forceStrutHeight: false),
    StrutStyle(fontFamily: 'A'),
    StrutStyle(fontFamilyFallback: <String>['A']),
    StrutStyle(fontFamilyFallback: <String>[]),
    StrutStyle(fontWeight: FontWeight.w300, fontStyle: FontStyle.italic, fontSize: 12, height: 2, leading: 0.5),
    StrutStyle(fontWeight: FontWeight.w300, fontStyle: FontStyle.italic, fontSize: 12, height: 2, leading: 0.5),
    StrutStyle(leadingDistribution: TextLeadingDistribution.even),
    StrutStyle(fontSize: 12.000001),
    StrutStyle(fontSize: 12.0000001),
  ];
  out['strutStyle.equal'] = <Object>[
    for (int i = 0; i < struts.length; i++)
      <bool>[for (int j = 0; j < struts.length; j++) struts[i] == struts[j]],
  ];
  out['strutStyle.string'] = '${struts[0]}';
  // Building with a strut records the leading distribution in it.
  final StrutStyle usedStrut = StrutStyle(fontSize: 10);
  final StrutStyle twinStrut = StrutStyle(fontSize: 10);
  ParagraphBuilder(
    ParagraphStyle(
      strutStyle: usedStrut,
      textHeightBehavior: const TextHeightBehavior(leadingDistribution: TextLeadingDistribution.even),
    ),
  ).build().dispose();
  out['strutStyle.buildMutates'] = usedStrut == twinStrut;
  // ParagraphBuilder and Paragraph API.
  final ParagraphBuilder builder = ParagraphBuilder(ParagraphStyle());
  out['builder.string'] = '$builder';
  builder.addPlaceholder(10, 10, PlaceholderAlignment.middle, scale: 2);
  builder.addPlaceholder(10, 10, PlaceholderAlignment.top);
  out['builder.placeholders'] = <Object>[builder.placeholderCount, builder.placeholderScales];
  out['builder.errors'] = <String>[
    err(() => builder.addText('bad \uD800 surrogate')),
    err(() => builder.addText('bad \uDC00')),
    err(() => builder.addText('trailing \uD83D')),
    err(() => builder.addPlaceholder(1, 1, PlaceholderAlignment.baseline)),
    err(() => builder.addPlaceholder(1, 1, PlaceholderAlignment.aboveBaseline)),
    err(() => builder.addPlaceholder(1, 1, PlaceholderAlignment.belowBaseline)),
    err(() => builder.addText('ok 👍')),
  ];
  final Paragraph paragraph = builder.build();
  out['paragraph.string'] = <String>['$paragraph'];
  paragraph.layout(const ParagraphConstraints(width: 100));
  (out['paragraph.string']! as List<String>).add('$paragraph');
  out['paragraph.debugDisposed'] = paragraph.debugDisposed;
  paragraph.dispose();
  (out['paragraph.string']! as List<String>).add('$paragraph');
  out['paragraph.disposed'] = <Object>[paragraph.debugDisposed, err(paragraph.dispose)];
  final Paragraph dirty = ParagraphBuilder(ParagraphStyle()).build();
  dirty.dispose();
  out['paragraph.disposedDirty'] = '$dirty';
  // The native peers are gone after build/dispose.
  final ParagraphBuilder built = ParagraphBuilder(ParagraphStyle());
  built.build();
  final TextStyle afterBuildStyle = TextStyle(fontSize: 3);
  out['builder.afterBuild'] = <Object>[
    err(() => built.pushStyle(afterBuildStyle)),
    '$afterBuildStyle',
    err(built.pop),
    err(() => built.addText('x')),
    err(() => built.addText('')),
    err(() => built.addPlaceholder(1, 1, PlaceholderAlignment.top)),
    built.placeholderCount,
    err(built.build),
  ];
  final Paragraph gone = ParagraphBuilder(ParagraphStyle()).build()
    ..layout(const ParagraphConstraints(width: 10))
    ..dispose();
  out['paragraph.afterDispose'] = <String>[
    err(() => gone.width),
    err(() => gone.didExceedMaxLines),
    err(() => gone.layout(const ParagraphConstraints(width: 10))),
    err(() => gone.getBoxesForRange(0, 1)),
    err(gone.getBoxesForPlaceholders),
    err(() => gone.getPositionForOffset(Offset.zero)),
    err(() => gone.getGlyphInfoAt(0)),
    err(() => gone.getClosestGlyphInfoForOffset(Offset.zero)),
    err(() => gone.getWordBoundary(const TextPosition(offset: 0))),
    err(() => gone.getLineBoundary(const TextPosition(offset: 0))),
    err(gone.computeLineMetrics),
    err(() => gone.getLineMetricsAt(0)),
    err(() => gone.numberOfLines),
    err(() => gone.getLineNumberAt(0)),
  ];
  // drawParagraph's checks.
  final PictureRecorder recorder = PictureRecorder();
  final Canvas canvas = Canvas(recorder);
  final Paragraph unlaid = ParagraphBuilder(ParagraphStyle()).build();
  out['drawParagraph.errors'] = <String>[
    err(() => canvas.drawParagraph(unlaid, Offset.zero)),
    err(() => canvas.drawParagraph(unlaid..layout(const ParagraphConstraints(width: 10)), const Offset(double.nan, 0))),
    err(() => canvas.drawParagraph(unlaid, const Offset(1, 2))),
    err(() => canvas.drawParagraph(unlaid..dispose(), Offset.zero)),
  ];
  recorder.endRecording();
  return out;
}

void main() {
  test('text', () async {
    await loadFonts();
    final List<Object?> scenes = jsonDecode(File('../text_scenes.json').readAsStringSync()) as List<Object?>;
    final Map<String, Object?> out = <String, Object?>{};
    for (final Object? sceneObject in scenes) {
      final Map<String, Object?> scene = sceneObject! as Map<String, Object?>;
      final (Paragraph p, int length) = build(scene);
      final List<Object?> perWidth = <Object?>[];
      for (final Object? w in scene['widths']! as List<Object?>) {
        final double width = w == 'Infinity' ? double.infinity : (w! as num).toDouble();
        p.layout(ParagraphConstraints(width: width));
        perWidth.add(dump(p, length));
      }
      p.dispose();
      out['scene.${scene['name']}'] = perWidth;
    }
    out.addAll(apiCases());
    final bool impeller = Platform.environment['FLUTTER_TEST_IMPELLER'] == 'true';
    File(impeller ? 'build/text_impeller.json' : 'build/text.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
