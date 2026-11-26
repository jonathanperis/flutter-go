// Reference side of flutter-go's TextPainter differential test
// (text_painter.dart over dart:ui's Paragraph): the scenes of
// ../painting_text_painter_scenes.json (shared with Go) are laid out by one
// TextPainter per scene at each [minWidth, maxWidth] in turn (so the
// relayout cache is exercised) and queried exhaustively: size, intrinsic
// widths, baselines, line metrics, placeholder boxes, caret offsets and
// heights at every position with both affinities, boxes for every
// selection range, positions and closest glyphs on a grid, word and line
// boundaries, WordBoundary.moveByWordBoundary, getOffsetAfter/Before, and
// the drawParagraph offset of paint. Plus state changes (setters, paint-only
// changes, errors) and the static computeWidth/computeMaxIntrinsicWidth.
// Fonts: flutter_tester's test fonts plus Roboto (loadFontFromList).
// Output: build/painting_text_painter.json, a list of lines.
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/painting.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

String err(Object? Function() f) {
  try {
    return '${f()}';
  } on AssertionError catch (e) {
    return 'AssertionError: ${e.message}';
  } on FlutterError catch (e) {
    return 'FlutterError: $e';
  } catch (e) {
    return 'error: $e';
  }
}

double dbl(Object? v) => v is String ? double.parse(v) : (v! as num).toDouble();

/// A placeholder span as widgets' WidgetSpan builds one: the dimension at
/// the builder's placeholder count.
class TestPlaceholder extends PlaceholderSpan {
  const TestPlaceholder({super.alignment, super.baseline, super.style});

  @override
  void build(
    ui.ParagraphBuilder builder, {
    TextScaler textScaler = TextScaler.noScaling,
    List<PlaceholderDimensions>? dimensions,
  }) {
    final bool hasStyle = style != null;
    if (hasStyle) {
      builder.pushStyle(style!.getTextStyle(textScaler: textScaler));
    }
    final PlaceholderDimensions d = dimensions![builder.placeholderCount];
    builder.addPlaceholder(d.size.width, d.size.height, d.alignment,
        baseline: d.baseline, baselineOffset: d.baselineOffset);
    if (hasStyle) {
      builder.pop();
    }
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

TextStyle? styleOf(Map<String, Object?>? s) {
  if (s == null) {
    return null;
  }
  return TextStyle(
    fontSize: s['size'] == null ? null : dbl(s['size']),
    fontFamily: s['family'] as String?,
    letterSpacing: s['ls'] == null ? null : dbl(s['ls']),
    wordSpacing: s['ws'] == null ? null : dbl(s['ws']),
    height: s['height'] == null ? null : dbl(s['height']),
    leadingDistribution: s['lead'] == null ? null : TextLeadingDistribution.values.byName(s['lead']! as String),
  );
}

InlineSpan span(Map<String, Object?> s) {
  if (s['ph'] == true) {
    return const TestPlaceholder();
  }
  return TextSpan(
    text: s['text'] as String?,
    style: styleOf(s['style'] as Map<String, Object?>?),
    semanticsLabel: s['label'] as String?,
    children: (s['children'] as List<Object?>?)?.map((Object? c) => span(c! as Map<String, Object?>)).toList(),
  );
}

TextPainter painter(Map<String, Object?> scene) {
  final Map<String, Object?> p = (scene['painter'] as Map<String, Object?>?) ?? <String, Object?>{};
  TextScaler? scaler;
  final List<Object?>? sc = p['scaler'] as List<Object?>?;
  if (sc != null) {
    scaler = sc[0] == 'linear'
        ? TextScaler.linear(dbl(sc[1]))
        : TextScaler.linear(dbl(sc[1])).clamp(minScaleFactor: dbl(sc[2]), maxScaleFactor: dbl(sc[3]));
  }
  StrutStyle? strut;
  final Map<String, Object?>? st = p['strut'] as Map<String, Object?>?;
  if (st != null) {
    strut = st['disabled'] == true
        ? StrutStyle.disabled
        : StrutStyle(
            fontFamily: st['family'] as String?,
            fontSize: st['size'] == null ? null : dbl(st['size']),
            height: st['height'] == null ? null : dbl(st['height']),
            leading: st['leading'] == null ? null : dbl(st['leading']),
            forceStrutHeight: st['force'] as bool?,
          );
  }
  final Map<String, Object?>? thb = p['thb'] as Map<String, Object?>?;
  final List<Object?>? loc = p['locale'] as List<Object?>?;
  final TextPainter painter;
  if (p['scaleFactor'] != null) {
    painter = TextPainter(
      text: span(scene['span']! as Map<String, Object?>),
      textDirection: TextDirection.values.byName((p['dir'] as String?) ?? 'ltr'),
      // ignore: deprecated_member_use
      textScaleFactor: dbl(p['scaleFactor']),
    );
  } else {
    painter = TextPainter(
      text: span(scene['span']! as Map<String, Object?>),
      textAlign: TextAlign.values.byName((p['align'] as String?) ?? 'start'),
      textDirection: TextDirection.values.byName((p['dir'] as String?) ?? 'ltr'),
      textScaler: scaler ?? TextScaler.noScaling,
      maxLines: p['maxLines'] as int?,
      ellipsis: p['ellipsis'] as String?,
      locale: loc == null ? null : Locale(loc[0]! as String, loc[1] as String?),
      strutStyle: strut,
      textWidthBasis: TextWidthBasis.values.byName((p['basis'] as String?) ?? 'parent'),
      textHeightBehavior: thb == null
          ? null
          : TextHeightBehavior(
              applyHeightToFirstAscent: thb['first']! as bool,
              applyHeightToLastDescent: thb['last']! as bool,
            ),
      hyphens: Hyphens.values.byName((p['hyphens'] as String?) ?? 'manual'),
    );
  }
  final List<Object?>? phs = scene['placeholders'] as List<Object?>?;
  if (phs != null) {
    painter.setPlaceholderDimensions(<PlaceholderDimensions>[
      for (final List<Object?> ph in phs.cast<List<Object?>>())
        PlaceholderDimensions(
          size: Size(dbl(ph[0]), dbl(ph[1])),
          alignment: ui.PlaceholderAlignment.values.byName(ph[2]! as String),
          baseline: ph[3] == null ? null : TextBaseline.values.byName(ph[3]! as String),
          baselineOffset: ph[4] == null ? null : dbl(ph[4]),
        ),
    ]);
  }
  return painter;
}

String glyph(ui.GlyphInfo? g) =>
    g == null ? 'null' : '${g.graphemeClusterLayoutBounds} ${g.graphemeClusterCodeUnitRange} ${g.writingDirection}';

void dump(List<String> out, String key, TextPainter p) {
  final String plain = p.plainText;
  final int n = plain.length;
  out.add('$key size ${err(() => p.size)} min ${p.minIntrinsicWidth} max ${p.maxIntrinsicWidth} '
      'exceeded ${p.didExceedMaxLines} alpha ${p.computeDistanceToActualBaseline(TextBaseline.alphabetic)} '
      'ideo ${p.computeDistanceToActualBaseline(TextBaseline.ideographic)} preferred ${p.preferredLineHeight}');
  out.add('$key lines ${p.computeLineMetrics()}');
  out.add('$key placeholders ${p.inlinePlaceholderBoxes}');
  const Rect caret = Rect.fromLTWH(0, 0, 2, 10);
  for (int i = 0; i <= n + 1; i++) {
    for (final TextAffinity a in TextAffinity.values) {
      final TextPosition pos = TextPosition(offset: i, affinity: a);
      out.add('$key caret $i ${a.name} ${err(() => p.getOffsetForCaret(pos, caret))} '
          '${err(() => p.getFullHeightForCaret(pos, caret))}');
    }
  }
  final bool full = n <= 24;
  for (int i = 0; i <= n; i++) {
    for (int j = i; j <= n; j++) {
      if (!full && j - i > 3 && j != n) {
        continue;
      }
      out.add('$key boxes $i $j ${p.getBoxesForSelection(TextSelection(baseOffset: i, extentOffset: j))}');
    }
  }
  if (n > 1) {
    for (final ui.BoxHeightStyle h in ui.BoxHeightStyle.values) {
      for (final ui.BoxWidthStyle w in ui.BoxWidthStyle.values) {
        out.add('$key boxes ${h.name} ${w.name} '
            '${p.getBoxesForSelection(TextSelection(baseOffset: 1, extentOffset: n), boxHeightStyle: h, boxWidthStyle: w)} '
            '${p.getBoxesForSelection(TextSelection(baseOffset: n, extentOffset: 0), boxHeightStyle: h, boxWidthStyle: w)}');
      }
    }
  }
  final double width = p.width.isFinite ? p.width : 100;
  for (int yi = 0; yi <= 6; yi++) {
    final double y = -4 + (p.height + 8) * yi / 6;
    final List<String> row = <String>[];
    for (int xi = 0; xi <= 8; xi++) {
      final double x = -4 + (width + 8) * xi / 8;
      final Offset o = Offset(x, y);
      row.add('${p.getPositionForOffset(o)} ${glyph(p.getClosestGlyphForOffset(o))}');
    }
    out.add('$key grid $yi ${row.join(' | ')}');
  }
  final WordBoundary words = p.wordBoundaries;
  final TextBoundary move = words.moveByWordBoundary;
  for (int i = -1; i <= n + 1; i++) {
    out.add('$key words $i ${err(() => p.getWordBoundary(TextPosition(offset: i)))} '
        '${err(() => p.getWordBoundary(TextPosition(offset: i, affinity: TextAffinity.upstream)))} '
        '${err(() => p.getLineBoundary(TextPosition(offset: i)))} '
        '${err(() => p.getLineBoundary(TextPosition(offset: i, affinity: TextAffinity.upstream)))} '
        '${err(() => words.getTextBoundaryAt(i))} '
        '${err(() => move.getLeadingTextBoundaryAt(i))} ${err(() => move.getTrailingTextBoundaryAt(i))} '
        '${err(() => p.getOffsetAfter(i))} ${err(() => p.getOffsetBefore(i))}');
  }
  final TestRecordingCanvas canvas = TestRecordingCanvas();
  p.paint(canvas, const Offset(3, 5));
  out.add('$key paint ${canvas.invocations.map((RecordedInvocation i) => '${i.invocation.memberName} ${i.invocation.positionalArguments.skip(1).toList()}').toList()}');
}

void main() {
  test('painting_text_painter', () async {
    final List<String> out = <String>[];
    String fontPath(String name) => '../../../engine/src/flutter/txt/third_party/fonts/$name';
    await ui.loadFontFromList(File(fontPath('Roboto-Regular.ttf')).readAsBytesSync(), fontFamily: 'Roboto');

    final List<Object?> scenes =
        jsonDecode(File('../painting_text_painter_scenes.json').readAsStringSync()) as List<Object?>;
    for (final Object? s in scenes) {
      final Map<String, Object?> scene = s! as Map<String, Object?>;
      final TextPainter p = painter(scene);
      out.add('${scene['name']} plain ${jsonEncode(p.plainText)}');
      for (final List<Object?> w in (scene['widths']! as List<Object?>).cast<List<Object?>>()) {
        final double minW = dbl(w[0]);
        final double maxW = dbl(w[1]);
        final String key = '${scene['name']} [$minW,$maxW]';
        final String e = err(() {
          p.layout(minWidth: minW, maxWidth: maxW);
          return 'ok';
        });
        if (e != 'ok') {
          out.add('$key layout $e');
          continue;
        }
        dump(out, key, p);
      }
      p.dispose();
    }

    // State changes.
    final TextPainter p = TextPainter(
      text: const TextSpan(text: 'state', style: TextStyle(fontSize: 10, color: Color(0xFF000000))),
      textDirection: TextDirection.ltr,
    );
    out.add('never laid out: ${err(() => p.width)}');
    out.add('paint before layout: ${err(() => p.paint(TestRecordingCanvas(), Offset.zero))}');
    p.layout();
    out.add('size ${p.size}');
    p.text = const TextSpan(text: 'state', style: TextStyle(fontSize: 10, color: Color(0xFFFF0000)));
    out.add('after color change size ${p.size}');
    final TestRecordingCanvas c1 = TestRecordingCanvas();
    p.paint(c1, Offset.zero);
    out.add('painted ${c1.invocations.length}');
    p.textWidthBasis = TextWidthBasis.longestLine;
    out.add('width after basis change: ${err(() => p.width)}');
    out.add('height after basis change: ${err(() => p.height)}');
    p.layout(maxWidth: 30);
    out.add('size ${p.size}');
    p.textAlign = TextAlign.right;
    out.add('after align: ${err(() => p.width).split('\n').take(2).join('\n')}');
    p.layout(maxWidth: 100);
    out.add('right ${p.size} ${p.getOffsetForCaret(const TextPosition(offset: 0), Rect.zero)}');
    p.text = const TextSpan(text: 'state longer', style: TextStyle(fontSize: 10));
    p.layout(maxWidth: 100);
    out.add('text change ${p.size} ${p.plainText}');
    p.textScaler = const TextScaler.linear(2);
    p.layout(maxWidth: 100);
    out.add('scaler ${p.size} ${p.preferredLineHeight} ${p.textScaler}');
    // ignore: deprecated_member_use
    p.textScaleFactor = 1.5;
    p.layout(maxWidth: 100);
    // ignore: deprecated_member_use
    out.add('scale factor ${p.size} ${p.textScaleFactor}');
    p.maxLines = 1;
    p.ellipsis = '…';
    p.layout(maxWidth: 40);
    out.add('ellipsis ${p.size} ${p.didExceedMaxLines}');
    p.strutStyle = const StrutStyle(fontSize: 30, forceStrutHeight: true);
    p.layout(maxWidth: 40);
    out.add('strut ${p.size} ${p.getFullHeightForCaret(const TextPosition(offset: 1), Rect.zero)}');
    p.textDirection = TextDirection.rtl;
    p.layout(maxWidth: 200);
    out.add('rtl ${p.size} ${p.getOffsetForCaret(const TextPosition(offset: 2), Rect.zero)}');
    p.textDirection = null;
    out.add('null direction: ${err(() => p.layout())}');
    p.textDirection = TextDirection.ltr;
    p.text = null;
    out.add('null text: ${err(() => p.layout())}');
    out.add('maxLines 0: ${err(() => p.maxLines = 0)}');
    out.add('empty ellipsis: ${err(() => p.ellipsis = '')}');
    p.dispose();
    out.add('debugDisposed ${p.debugDisposed}');

    final TextPainter c = TextPainter(
      text: const TextSpan(text: 'centered', style: TextStyle(fontSize: 10, color: Color(0xFF000000))),
      textDirection: TextDirection.ltr,
      textAlign: TextAlign.center,
    );
    out.add('center pref ${c.preferredLineHeight}');
    c.layout();
    c.text = const TextSpan(text: 'centered', style: TextStyle(fontSize: 10, color: Color(0xFFFF0000)));
    out.add('center paint ${err(() {
      final TestRecordingCanvas rc = TestRecordingCanvas();
      c.paint(rc, Offset.zero);
      return rc.invocations.map((RecordedInvocation i) => i.invocation.positionalArguments[1]).toList();
    })} ${c.size}');
    c.text = const TextSpan(text: 'centered', style: TextStyle(fontSize: 20, color: Color(0xFFFF0000)));
    out.add('center pref2 ${c.preferredLineHeight}');
    c.layout(maxWidth: 300);
    out.add('center relayout ${c.size} ${c.getOffsetForCaret(const TextPosition(offset: 3), Rect.zero)}');
    c.dispose();

    for (final double max in <double>[double.infinity, 50, 20]) {
      const TextSpan t = TextSpan(text: 'compute me now', style: TextStyle(fontSize: 10));
      out.add('compute $max ${TextPainter.computeWidth(text: t, textDirection: TextDirection.ltr, maxWidth: max)} '
          '${TextPainter.computeMaxIntrinsicWidth(text: t, textDirection: TextDirection.rtl, maxWidth: max, minWidth: 5)} '
          '${TextPainter.computeWidth(text: t, textDirection: TextDirection.ltr, maxWidth: max, textScaler: const TextScaler.linear(1.5), textWidthBasis: TextWidthBasis.longestLine)}');
    }
    out.add('surrogates ${TextPainter.isHighSurrogate(0xD83D)} ${TextPainter.isLowSurrogate(0xDE00)} '
        '${TextPainter.isHighSurrogate(0x41)} ${err(() => TextPainter.isLowSurrogate(0x100000))}');
    out.add('dims ${PlaceholderDimensions.empty} '
        '${const PlaceholderDimensions(size: Size(1, 2), alignment: ui.PlaceholderAlignment.baseline, baselineOffset: 3)} '
        '${const PlaceholderDimensions(size: Size(1, 2), alignment: ui.PlaceholderAlignment.baseline)} '
        '${const PlaceholderDimensions(size: Size(1, 2), alignment: ui.PlaceholderAlignment.middle) == const PlaceholderDimensions(size: Size(1, 2), alignment: ui.PlaceholderAlignment.middle)}');

    File('build/painting_text_painter.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
