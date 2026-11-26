// Reference side of flutter-go's painting text-style differential test
// (text_style.dart, strut_style.dart, inline_span.dart, text_span.dart,
// placeholder_span.dart): TextStyle and StrutStyle toString/diagnostics,
// copyWith/apply/merge/lerp/==/compareTo over many pairs, the dart:ui
// TextStyle and ParagraphStyle they produce (their toString and the layout
// of a paragraph built with them), lerpFontVariations, and span trees
// (toPlainText, codeUnitAt, getSpanForPosition, visitChildren, semantics
// information, combineSemanticsInfo, ==, compareTo, toStringDeep).
// Output: build/painting_text_style.json, a list of lines.
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/painting.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

String err(Object? Function() f) {
  try {
    return '${f()}';
  } on AssertionError catch (e) {
    try {
      return 'AssertionError: ${e.message}';
    } on AssertionError {
      return 'AssertionError: (message asserts)';
    }
  } catch (e) {
    return 'error: $e';
  }
}

class TestPlaceholder extends PlaceholderSpan {
  const TestPlaceholder({super.alignment, super.baseline, super.style});

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
    if (position.offset == offset.value) {
      return this;
    }
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
  RenderComparison compareTo(InlineSpan other) =>
      identical(this, other) ? RenderComparison.identical : RenderComparison.layout;

  @override
  bool debugAssertIsValid() => true;

  @override
  String toStringShort() => 'TestPlaceholder';
}

String layoutOf(ui.ParagraphStyle ps, ui.TextStyle? ts) {
  final ui.ParagraphBuilder b = ui.ParagraphBuilder(ps);
  if (ts != null) {
    b.pushStyle(ts);
  }
  b.addText('Ag xy\nz');
  final ui.Paragraph p = b.build()..layout(const ui.ParagraphConstraints(width: 200));
  return '${p.width} ${p.height} ${p.longestLine} ${p.alphabeticBaseline} ${p.computeLineMetrics()}';
}

String props(Diagnosticable d, String prefix) {
  final DiagnosticPropertiesBuilder b = DiagnosticPropertiesBuilder();
  if (d is TextStyle) {
    d.debugFillProperties(b, prefix: prefix);
  } else if (d is StrutStyle) {
    d.debugFillProperties(b, prefix: prefix);
  }
  return b.properties.map((DiagnosticsNode n) => '${n.name}=${n.toDescription()}@${n.level.name}').join('; ');
}

String spanName(InlineSpan? s) => switch (s) {
  null => 'null',
  final TextSpan t => 'TextSpan(${t.text})',
  _ => 'TestPlaceholder',
};

String info(InlineSpanSemanticsInformation i) =>
    '{${i.text}|${i.semanticsLabel}|${i.semanticsIdentifier}|${i.recognizer != null}|${i.isPlaceholder}|'
    '${i.requiresOwnNode}|${i.stringAttributes}}';

void main() {
  test('painting_text_style', () {
    final List<String> out = <String>[];
    final Paint fg = Paint()..color = const Color(0xFF112233);
    final Paint bg = Paint()..color = const Color(0xFF445566);
    final List<TextStyle?> styles = <TextStyle?>[
      null,
      const TextStyle(),
      const TextStyle(inherit: false),
      const TextStyle(
        color: Color(0xFF00FF00),
        fontSize: 12,
        fontWeight: FontWeight.w700,
        fontStyle: FontStyle.italic,
        debugLabel: 'a',
      ),
      const TextStyle(
        backgroundColor: Color(0x80FF0000),
        letterSpacing: 1.5,
        wordSpacing: -2,
        textBaseline: TextBaseline.ideographic,
        height: 1.25,
        leadingDistribution: TextLeadingDistribution.even,
        locale: Locale('fr', 'CA'),
      ),
      TextStyle(foreground: fg, background: bg, fontSize: 20),
      const TextStyle(
        shadows: <Shadow>[Shadow(color: Color(0xFF000000), offset: Offset(1, 2), blurRadius: 3)],
        fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
        fontVariations: <FontVariation>[FontVariation('wght', 300), FontVariation('wdth', 80)],
      ),
      const TextStyle(
        decoration: TextDecoration.underline,
        decorationColor: Color(0xFF0000FF),
        decorationStyle: TextDecorationStyle.dashed,
        decorationThickness: 2,
      ),
      const TextStyle(decorationThickness: 0.5),
      const TextStyle(
        fontFamily: 'Roboto',
        fontFamilyFallback: <String>['Noto', 'Emoji'],
        package: 'pkg',
        overflow: TextOverflow.ellipsis,
        debugLabel: 'pkg',
      ),
      const TextStyle(
        fontFamily: 'Ahem',
        fontFamilyFallback: <String>['A'],
        inherit: false,
        fontSize: 8,
        color: Color(0xFF123456),
        height: 0,
        fontWeight: FontWeight.w100,
      ),
      const TextStyle(
        shadows: <Shadow>[],
        fontFeatures: <FontFeature>[],
        fontVariations: <FontVariation>[FontVariation('wght', 900), FontVariation('opsz', 12), FontVariation('slnt', -5)],
        fontSize: 30,
        letterSpacing: 0,
        wordSpacing: 4,
        height: 2,
        decorationThickness: 1,
        color: Color(0xFFABCDEF),
      ),
      const TextStyle(
        inherit: false,
        backgroundColor: Color(0xFF00FFFF),
        fontSize: 14,
        letterSpacing: 1,
        wordSpacing: 1,
        height: 1,
        decorationColor: Color(0xFF111111),
        decorationThickness: 3,
        color: Color(0x01020304),
      ),
      const TextStyle(package: 'only', decoration: TextDecoration.lineThrough),
    ];

    for (int i = 0; i < styles.length; i++) {
      final TextStyle? s = styles[i];
      if (s == null) {
        continue;
      }
      out.add('style $i: ${err(() => s)}');
      out.add('style $i deep: ${err(() => s.toDiagnosticsNode().toStringDeep())}');
      out.add('style $i fine: ${err(() => s.toDiagnosticsNode().toStringDeep(minLevel: DiagnosticLevel.fine))}');
      out.add('style $i props: ${err(() => props(s, 'p.'))}');
      out.add('style $i fallback: ${s.fontFamilyFallback} family: ${s.fontFamily}');
      out.add('style $i ui: ${s.getTextStyle()}');
      out.add('style $i ui x2: ${s.getTextStyle(textScaler: const TextScaler.linear(2))}');
      // ignore: deprecated_member_use
      out.add('style $i ui f1.5: ${s.getTextStyle(textScaleFactor: 1.5)}');
      // ignore: deprecated_member_use
      out.add('style $i ui both: ${err(() => s.getTextStyle(textScaleFactor: 1.5, textScaler: const TextScaler.linear(2)))}');
      out.add('style $i para: ${s.getParagraphStyle()}');
      final ui.ParagraphStyle ps = s.getParagraphStyle(
        textAlign: TextAlign.center,
        textDirection: TextDirection.rtl,
        textScaler: const TextScaler.linear(1.5),
        ellipsis: '…',
        maxLines: 3,
        textHeightBehavior: const TextHeightBehavior(applyHeightToFirstAscent: false),
        hyphens: ui.Hyphens.hidden,
        locale: const Locale('ja'),
        fontFamily: 'Ahem',
        fontSize: 9,
        fontWeight: FontWeight.w500,
        fontStyle: FontStyle.italic,
        height: 1.7,
        strutStyle: const StrutStyle(fontSize: 10, height: 2, leading: 0.5, forceStrutHeight: true),
      );
      out.add('style $i para2: $ps');
      out.add('style $i para3: ${s.getParagraphStyle(strutStyle: const StrutStyle(fontFamily: 'Ahem', fontSize: 20, leadingDistribution: TextLeadingDistribution.even, height: 1), textScaler: const TextScaler.linear(0.5))}');
      out.add('style $i maxLines0: ${err(() => s.getParagraphStyle(maxLines: 0))}');
      out.add('style $i layout: ${layoutOf(s.getParagraphStyle(), s.getTextStyle())}');
      out.add('style $i layout2: ${layoutOf(ps, s.getTextStyle(textScaler: const TextScaler.linear(1.5)))}');
      out.add('style $i layout3: ${layoutOf(s.getParagraphStyle(strutStyle: const StrutStyle(fontFamily: 'Ahem', fontSize: 20, leadingDistribution: TextLeadingDistribution.even, height: 1, forceStrutHeight: true)), null)}');
      out.add('style $i copy: ${err(() => s.copyWith())}');
      out.add('style $i copy2: ${err(() => s.copyWith(color: const Color(0xFF0000AA), backgroundColor: const Color(0xFF00AA00), fontSize: 33, debugLabel: 'cw'))}');
      out.add('style $i copy3: ${err(() => s.copyWith(foreground: bg, background: fg, inherit: false, fontFamily: 'Z', package: 'q', fontFamilyFallback: <String>['f'], overflow: TextOverflow.fade))}');
      out.add('style $i copy4: ${err(() => s.copyWith(color: const Color(0xFF000000), foreground: fg))}');
      out.add('style $i apply: ${err(() => s.apply())}');
      out.add('style $i apply2: ${err(() => s.apply(color: const Color(0xFF999999), backgroundColor: const Color(0xFF888888), decoration: TextDecoration.overline, decorationColor: const Color(0xFF777777), decorationStyle: TextDecorationStyle.wavy, fontFamily: 'F', fontFamilyFallback: <String>['G'], fontStyle: FontStyle.normal, textBaseline: TextBaseline.alphabetic, leadingDistribution: TextLeadingDistribution.proportional, locale: const Locale('es'), shadows: const <Shadow>[Shadow(blurRadius: 1)], fontFeatures: const <FontFeature>[FontFeature.enable('smcp')], fontVariations: const <FontVariation>[FontVariation('wght', 1)], package: 'ap', overflow: TextOverflow.visible))}');
      out.add('style $i apply3: ${err(() => s.apply(fontSizeFactor: 1.1, fontSizeDelta: 0.3))}');
      out.add('style $i apply4: ${err(() => s.apply(fontWeightDelta: 3))}');
      out.add('style $i apply5: ${err(() => s.apply(fontWeightDelta: -10))}');
      out.add('style $i apply6: ${err(() => s.apply(letterSpacingFactor: 0.3, letterSpacingDelta: 0.1, wordSpacingFactor: 3, wordSpacingDelta: -0.7))}');
      out.add('style $i apply7: ${err(() => s.apply(heightFactor: 1.1, heightDelta: 0.2))}');
      out.add('style $i apply8: ${err(() => s.apply(decorationThicknessFactor: 0.7, decorationThicknessDelta: 0.1))}');
    }
    const List<double> ts = <double>[0, 0.3, 0.5, 0.75, 1];
    for (int i = 0; i < styles.length; i++) {
      for (int j = 0; j < styles.length; j++) {
        final TextStyle? a = styles[i];
        final TextStyle? b = styles[j];
        final String pre = 'pair $i $j';
        if (a != null) {
          out.add('$pre merge: ${err(() => a.merge(b))}');
          if (b != null) {
            out.add('$pre eq: ${a == b} cmp: ${a.compareTo(b)}');
          }
        }
        for (final double t in ts) {
          out.add('$pre lerp $t: ${err(() => TextStyle.lerp(a, b, t))}');
        }
      }
    }
    // compareTo of single-field changes.
    const TextStyle base = TextStyle(fontSize: 10, color: Color(0xFF000000));
    for (final TextStyle other in <TextStyle>[
      base.copyWith(),
      base.copyWith(color: const Color(0xFF000001)),
      base.copyWith(backgroundColor: const Color(0xFF000001)),
      base.copyWith(decoration: TextDecoration.underline),
      base.copyWith(decorationColor: const Color(0xFF000001)),
      base.copyWith(decorationStyle: TextDecorationStyle.dotted),
      base.copyWith(decorationThickness: 2),
      base.copyWith(fontSize: 11),
      base.copyWith(fontWeight: FontWeight.w300),
      base.copyWith(fontStyle: FontStyle.italic),
      base.copyWith(letterSpacing: 1),
      base.copyWith(wordSpacing: 1),
      base.copyWith(textBaseline: TextBaseline.alphabetic),
      base.copyWith(height: 1),
      base.copyWith(leadingDistribution: TextLeadingDistribution.even),
      base.copyWith(locale: const Locale('en')),
      base.copyWith(shadows: const <Shadow>[]),
      base.copyWith(fontFeatures: const <FontFeature>[]),
      base.copyWith(fontVariations: const <FontVariation>[]),
      base.copyWith(fontFamily: 'f'),
      base.copyWith(fontFamilyFallback: const <String>[]),
      base.copyWith(overflow: TextOverflow.clip),
      base.copyWith(inherit: false),
      base.copyWith(debugLabel: 'x'),
      base.copyWith(package: 'p'),
    ]) {
      out.add('cmp ${base.compareTo(other)} ${other.compareTo(base)} ${base == other}');
    }

    // lerpFontVariations.
    final List<List<FontVariation>?> variations = <List<FontVariation>?>[
      null,
      <FontVariation>[],
      <FontVariation>[const FontVariation('wght', 100)],
      <FontVariation>[const FontVariation('wght', 200), const FontVariation('wdth', 50)],
      <FontVariation>[const FontVariation('wdth', 50), const FontVariation('wght', 200)],
      <FontVariation>[
        const FontVariation('opsz', 10),
        const FontVariation('slnt', -10),
        const FontVariation('wght', 400),
        const FontVariation('ital', 1),
      ],
      <FontVariation>[
        const FontVariation('wght', 900),
        const FontVariation('GRAD', 50),
        const FontVariation('abcd', 3),
        const FontVariation('efgh', 4),
        const FontVariation('ijkl', 5),
        const FontVariation('mnop', 6),
        const FontVariation('qrst', 7),
      ],
      <FontVariation>[const FontVariation('wght', 300), const FontVariation('opsz', 20)],
      <FontVariation>[const FontVariation('xxxx', 1), const FontVariation('xxxx', 2), const FontVariation('yyyy', 3)],
    ];
    for (int i = 0; i < variations.length; i++) {
      for (int j = 0; j < variations.length; j++) {
        for (final double t in <double>[0, 0.25, 0.5, 0.6, 1]) {
          out.add('fv $i $j $t: ${lerpFontVariations(variations[i], variations[j], t)}');
        }
      }
    }

    // StrutStyle.
    final List<StrutStyle?> struts = <StrutStyle?>[
      null,
      const StrutStyle(),
      StrutStyle.disabled,
      const StrutStyle(
        fontFamily: 'Roboto',
        fontSize: 12,
        height: 1.5,
        leadingDistribution: TextLeadingDistribution.even,
        leading: 0.25,
        fontWeight: FontWeight.w600,
        fontStyle: FontStyle.italic,
        forceStrutHeight: true,
        debugLabel: 's',
      ),
      const StrutStyle(fontFamily: 'A', fontFamilyFallback: <String>['B', 'C'], package: 'pk'),
      const StrutStyle(forceStrutHeight: false),
      const StrutStyle(height: 2),
      const StrutStyle(leadingDistribution: TextLeadingDistribution.proportional),
      const StrutStyle(fontFamilyFallback: <String>['X'], package: 'p'),
      const StrutStyle(height: 2, leadingDistribution: TextLeadingDistribution.proportional),
    ];
    for (int i = 0; i < struts.length; i++) {
      final StrutStyle? s = struts[i];
      if (s == null) {
        continue;
      }
      out.add('strut $i: $s');
      out.add('strut $i deep: ${err(() => s.toDiagnosticsNode().toStringDeep())}');
      out.add('strut $i props: ${props(s, 'q.')}');
      out.add('strut $i fallback: ${s.fontFamilyFallback} family: ${s.fontFamily}');
      for (int k = 0; k < styles.length; k++) {
        out.add('strut $i inherit $k: ${err(() => s.inheritFromTextStyle(styles[k]))}');
      }
    }
    for (int i = 0; i < struts.length; i++) {
      for (int j = 0; j < struts.length; j++) {
        final StrutStyle? a = struts[i];
        final StrutStyle? b = struts[j];
        if (a != null) {
          out.add('strut pair $i $j merge: ${a.merge(b)}${b == null ? '' : ' eq: ${a == b} cmp: ${a.compareTo(b)}'}');
        }
      }
    }
    for (int k = 1; k < styles.length; k++) {
      out.add('fromTextStyle $k: ${err(() => StrutStyle.fromTextStyle(styles[k]!))}');
      out.add('fromTextStyle2 $k: ${err(() => StrutStyle.fromTextStyle(styles[k]!, fontFamily: 'F', fontFamilyFallback: <String>['G'], fontSize: 7, height: 3, leadingDistribution: TextLeadingDistribution.even, leading: 1, fontWeight: FontWeight.w900, fontStyle: FontStyle.normal, forceStrutHeight: true, debugLabel: 'ft', package: 'fp'))}');
    }
    out.add('strut errors: ${err(() => StrutStyle(fontSize: 0))} ${err(() => StrutStyle(leading: -1))} '
        '${err(() => StrutStyle(package: 'p'))}');
    out.add('style errors: ${err(() => TextStyle(color: const Color(0xFF000000), foreground: fg))} '
        '${err(() => TextStyle(backgroundColor: const Color(0xFF000000), background: bg))} '
        '${err(() => TextStyle(height: double.nan))}');

    // Spans.
    final TapGestureRecognizer tap = TapGestureRecognizer();
    void enter(PointerEnterEvent e) {}
    final List<InlineSpan> spans = <InlineSpan>[
      TextSpan(
        text: 'Hello ',
        style: const TextStyle(fontSize: 10),
        children: <InlineSpan>[
          const TextSpan(text: 'world', semanticsLabel: 'WORLD', locale: Locale('en'), spellOut: true),
          const TestPlaceholder(),
          TextSpan(children: <InlineSpan>[TextSpan(text: '😀x', recognizer: tap)]),
          const TextSpan(text: '', spellOut: false),
        ],
      ),
      const TextSpan(),
      const TextSpan(
        text: 'a',
        semanticsIdentifier: 'id',
        locale: Locale('ja', 'JP'),
        children: <InlineSpan>[
          TextSpan(text: 'b', spellOut: true, children: <InlineSpan>[TextSpan(text: 'c', locale: Locale('de'))]),
          TextSpan(text: 'd', semanticsLabel: 'DD'),
        ],
      ),
      TextSpan(text: 'hover', onEnter: enter, mouseCursor: SystemMouseCursors.text, semanticsLabel: ''),
      const TextSpan(text: 'x', style: TextStyle(color: Color(0xFF000000)), children: <InlineSpan>[TextSpan(text: 'y')]),
      const TextSpan(text: 'x', style: TextStyle(color: Color(0xFF000001)), children: <InlineSpan>[TextSpan(text: 'y')]),
      const TextSpan(text: 'x', style: TextStyle(fontSize: 3), children: <InlineSpan>[TextSpan(text: 'y')]),
      const TextSpan(text: 'x', children: <InlineSpan>[TextSpan(text: 'y')]),
      const TextSpan(text: 'x', style: TextStyle(color: Color(0xFF000000)), children: <InlineSpan>[
        TextSpan(text: 'y', style: TextStyle(decoration: TextDecoration.underline)),
      ]),
      TextSpan(text: 'x', style: const TextStyle(color: Color(0xFF000000)), recognizer: tap,
          children: const <InlineSpan>[TextSpan(text: 'y')]),
      const TextSpan(text: 'x', style: TextStyle(color: Color(0xFF000000)), children: <InlineSpan>[TextSpan(text: 'z')]),
      const TextSpan(text: 'x', style: TextStyle(color: Color(0xFF000000)), children: <InlineSpan>[TextSpan(text: 'y'), TextSpan(text: 'y')]),
      const TestPlaceholder(),
      const TextSpan(text: 'x', style: TextStyle(color: Color(0xFF000000)), children: <InlineSpan>[TextSpan(text: 'y')]),
      const TextSpan(text: '\u{1F600}\u{1F600}', children: <InlineSpan>[TextSpan(text: 'q', semanticsLabel: 'Q', spellOut: true)]),
    ];
    for (int i = 0; i < spans.length; i++) {
      final InlineSpan s = spans[i];
      final String plain = s.toPlainText();
      out.add('span $i plain: ${jsonEncode(plain)} ${jsonEncode(s.toPlainText(includeSemanticsLabels: false))} '
          '${jsonEncode(s.toPlainText(includePlaceholders: false))} '
          '${jsonEncode(s.toPlainText(includeSemanticsLabels: false, includePlaceholders: false))}');
      final int n = s.toPlainText(includeSemanticsLabels: false).length;
      out.add('span $i units: ${<String>[for (int k = -1; k <= n + 1; k++) '${s.codeUnitAt(k)}'].join(',')}');
      out.add('span $i at: ${<String>[
        for (int k = 0; k <= n + 1; k++)
          for (final TextAffinity a in TextAffinity.values)
            spanName(s.getSpanForPosition(TextPosition(offset: k, affinity: a))),
      ].join(',')}');
      final List<String> visited = <String>[];
      out.add('span $i visit: ${s.visitChildren((InlineSpan c) {
        visited.add(spanName(c));
        return true;
      })} $visited');
      final List<String> visited2 = <String>[];
      out.add('span $i visit2: ${s.visitChildren((InlineSpan c) {
        visited2.add(spanName(c));
        return visited2.length < 2;
      })} $visited2');
      final List<String> direct = <String>[];
      out.add('span $i direct: ${s.visitDirectChildren((InlineSpan c) {
        direct.add(spanName(c));
        return true;
      })} $direct');
      final List<InlineSpanSemanticsInformation> sem = s.getSemanticsInformation();
      out.add('span $i semantics: ${sem.map(info).join(' ')}');
      out.add('span $i combined: ${combineSemanticsInfo(sem).map(info).join(' ')}');
      out.add('span $i deep: ${s.toStringDeep()}');
      out.add('span $i valid: ${s.debugAssertIsValid()}');
    }
    for (int i = 0; i < spans.length; i++) {
      out.add('span eq $i: ${<String>[for (final InlineSpan o in spans) spans[i] == o ? '1' : '0'].join()}');
      out.add('span cmp $i: ${<String>[for (final InlineSpan o in spans) spans[i].compareTo(o).index.toString()].join()}');
    }
    final Accumulator acc = Accumulator(3);
    acc.increment(4);
    out.add('accumulator ${Accumulator().value} ${acc.value} ${err(() => acc.increment(-1))}');
    out.add('placeholder info ${InlineSpanSemanticsInformation.placeholder} ${info(InlineSpanSemanticsInformation.placeholder)}');
    final List<InlineSpanSemanticsInformation> infos = <InlineSpanSemanticsInformation>[
      InlineSpanSemanticsInformation('ab', semanticsLabel: 'AB', stringAttributes: <ui.StringAttribute>[
        ui.SpellOutStringAttribute(range: const TextRange(start: 0, end: 2)),
      ]),
      const InlineSpanSemanticsInformation('cd'),
      InlineSpanSemanticsInformation('ef', stringAttributes: <ui.StringAttribute>[
        ui.LocaleStringAttribute(range: const TextRange(start: 1, end: 2), locale: const Locale('fr')),
      ]),
      const InlineSpanSemanticsInformation('id', semanticsIdentifier: 'i'),
      InlineSpanSemanticsInformation.placeholder,
      InlineSpanSemanticsInformation('gh', semanticsLabel: 'G', stringAttributes: <ui.StringAttribute>[
        ui.SpellOutStringAttribute(range: const TextRange(start: 0, end: 1)),
      ]),
      const InlineSpanSemanticsInformation('😀'),
    ];
    out.add('combine ${combineSemanticsInfo(infos).map((InlineSpanSemanticsInformation i) => '$i ${info(i)}').join(' ')}');
    out.add('combine empty ${combineSemanticsInfo(<InlineSpanSemanticsInformation>[]).map(info).join(' ')}');
    out.add('info eq ${const InlineSpanSemanticsInformation('a') == const InlineSpanSemanticsInformation('a')} '
        '${const InlineSpanSemanticsInformation('a', semanticsLabel: 'b') == const InlineSpanSemanticsInformation('a')} '
        '${err(() => InlineSpanSemanticsInformation('a', isPlaceholder: true))}');
    out.add('placeholder assert: ${err(() => const _BarePlaceholder().toPlainText())}');

    File('build/painting_text_style.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}

class _BarePlaceholder extends PlaceholderSpan {
  const _BarePlaceholder();

  @override
  void build(ui.ParagraphBuilder builder,
      {TextScaler textScaler = TextScaler.noScaling, List<PlaceholderDimensions>? dimensions}) {}

  @override
  bool visitChildren(InlineSpanVisitor visitor) => true;

  @override
  bool visitDirectChildren(InlineSpanVisitor visitor) => true;

  @override
  InlineSpan? getSpanForPositionVisitor(TextPosition position, Accumulator offset) => null;

  @override
  int? codeUnitAtVisitor(int index, Accumulator offset) => null;

  @override
  RenderComparison compareTo(InlineSpan other) => RenderComparison.layout;

  @override
  String toPlainText({bool includeSemanticsLabels = true, bool includePlaceholders = true}) {
    assert(debugAssertIsValid());
    return super.toPlainText();
  }
}
