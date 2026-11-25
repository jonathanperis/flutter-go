// Reference side of flutter-go's diagnostics differential test: text art,
// toString and JSON of diagnostic trees built with every property kind,
// every DiagnosticsTreeStyle, long word-wrapped text and error styles.
// Output: build/diagnostics.json as {"<scene>": <string or JSON>}.
// The Go twin is difftest/diagnostics_test.go; keep the two in step.
import 'dart:convert';
import 'dart:io';
import 'dart:ui' show Rect;

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

enum Fruit { apple, bananaSplit }

/// A Diagnosticable without children.
class Point2 with Diagnosticable {
  Point2(this.x, this.y);
  final double x;
  final double y;

  @override
  void debugFillProperties(DiagnosticPropertiesBuilder properties) {
    super.debugFillProperties(properties);
    properties.add(DoubleProperty('x', x));
    properties.add(DoubleProperty('y', y, defaultValue: 0.0));
  }
}

/// A DiagnosticableTreeMixin tree.
class SceneTree with DiagnosticableTreeMixin {
  SceneTree(
    this.label, {
    this.style,
    this.children = const <SceneTree>[],
    this.properties = const <DiagnosticsNode>[],
    this.emptyBody,
  });

  final String label;
  final DiagnosticsTreeStyle? style;
  final List<SceneTree> children;
  final List<DiagnosticsNode> properties;
  final String? emptyBody;

  @override
  List<DiagnosticsNode> debugDescribeChildren() => <DiagnosticsNode>[
    for (final SceneTree child in children)
      child.toDiagnosticsNode(name: child.label, style: child.style),
  ];

  @override
  void debugFillProperties(DiagnosticPropertiesBuilder properties) {
    super.debugFillProperties(properties);
    if (style != null) {
      properties.defaultDiagnosticsTreeStyle = style!;
    }
    if (emptyBody != null) {
      properties.emptyBodyDescription = emptyBody;
    }
    this.properties.forEach(properties.add);
  }
}

/// A subclass of the abstract DiagnosticableTree (not the mixin): its
/// toStringShallow starts with toString(), not toStringShort().
class PlainTree extends DiagnosticableTree {
  PlainTree(this.properties);
  final List<DiagnosticsNode> properties;

  @override
  void debugFillProperties(DiagnosticPropertiesBuilder properties) {
    super.debugFillProperties(properties);
    this.properties.forEach(properties.add);
  }
}

void onTap() {}
void onDrag() {}

List<DiagnosticsNode> allKinds() => <DiagnosticsNode>[
  StringProperty('string', 'hello'),
  StringProperty('unquoted', 'plain', quoted: false),
  StringProperty('multiline', 'one\ntwo'),
  StringProperty('empty', '', ifEmpty: '<empty>'),
  StringProperty('nullString', null),
  StringProperty('nullHidden', null, defaultValue: null),
  StringProperty('described', 'x', description: 'custom', tooltip: 'tip'),
  DoubleProperty('double', 1.25),
  DoubleProperty('tie', 0.25),
  DoubleProperty('belowTie', 0.35),
  DoubleProperty('negZero', -0.04),
  DoubleProperty('big', 1e21),
  DoubleProperty('nan', double.nan),
  DoubleProperty('inf', double.negativeInfinity, unit: 'px'),
  DoubleProperty('nullDouble', null, ifNull: 'unset'),
  DoubleProperty('defaultDouble', 1.0, defaultValue: 1.0),
  DoubleProperty('tooltipDouble', 2.5, tooltip: 'logical pixels'),
  DoubleProperty.lazy('lazy', () => 3.75, unit: 'ms'),
  DoubleProperty.lazy('throws', () => throw FlutterError('nope')),
  IntProperty('int', 42),
  IntProperty('intDefault', 42, defaultValue: 42.0),
  IntProperty('intUnit', -7, unit: 'px'),
  IntProperty('nullInt', null),
  PercentProperty('percent', 0.1234),
  PercentProperty('percentClamped', 1.5, unit: 'opacity'),
  PercentProperty('percentNegZero', -0.0),
  PercentProperty('percentTie', 0.0025, showName: false),
  FlagProperty('visible', value: true, ifTrue: 'visible', ifFalse: 'hidden'),
  FlagProperty('enabled', value: false, ifTrue: 'enabled'),
  FlagProperty('checked', value: false, ifFalse: 'unchecked', showName: true),
  FlagProperty('nullFlag', value: null, ifTrue: 'on'),
  IterableProperty<int>('ints', <int>[1, 2, 3]),
  IterableProperty<double>('doubles', <double>[0.25, 1.0 / 3.0, 2]),
  IterableProperty<String>('emptyList', <String>[]),
  IterableProperty<String>('emptyNoIfEmpty', <String>[], ifEmpty: null),
  IterableProperty<String>('nullList', null),
  IterableProperty<String>('lines', <String>['a', 'b'], style: DiagnosticsTreeStyle.whitespace),
  EnumProperty<Fruit>('fruit', Fruit.bananaSplit),
  EnumProperty<Fruit>('defaultFruit', Fruit.apple, defaultValue: Fruit.apple),
  EnumProperty<Fruit>('nullFruit', null),
  ObjectFlagProperty<Function>('onTap', onTap, ifPresent: 'tappable'),
  ObjectFlagProperty<Function>('onDrag', null, ifNull: 'not draggable'),
  ObjectFlagProperty<Function>('onHidden', onDrag, ifNull: 'no drag'),
  ObjectFlagProperty<Function>.has('callback', onTap),
  FlagsSummary<Function>('listeners', <String, Function?>{'tap': onTap, 'drag': null, 'hover': onDrag}),
  FlagsSummary<Function>('noListeners', <String, Function?>{'tap': null}, ifEmpty: '<none>'),
  MessageProperty('note', 'a message'),
  DiagnosticsNode.message('a bare message'),
  DiagnosticsProperty<bool>('bool', true),
  DiagnosticsProperty<bool>('nullBool', null, missingIfNull: true),
  DiagnosticsProperty<Object>('object', null, ifNull: 'nothing'),
  DiagnosticsProperty<double>('rawDouble', 0.1 + 0.2),
  DiagnosticsProperty<double>('tinyDouble', 1.5e-7),
  DiagnosticsProperty<int>('rawInt', 10),
  DiagnosticsProperty<List<int>>('list', <int>[4, 5]),
  DiagnosticsProperty<Rect>('rect', const Rect.fromLTRB(1, 2, 3.5, 4)),
  DiagnosticsProperty<Point2>('point', Point2(1, 2)),
  DiagnosticsProperty<Point2>('expanded', Point2(3, 0), expandableValue: true),
  DiagnosticsProperty<SceneTree>('treeValue', SceneTree('inner')),
  DiagnosticsProperty<String>('noSeparator', 'value', showSeparator: false),
  DiagnosticsProperty<String>('warn', 'careful', level: DiagnosticLevel.warning),
  DiagnosticsProperty<String>('fine', 'detail', level: DiagnosticLevel.fine),
  DiagnosticsProperty<String>('debugOnly', 'debug', level: DiagnosticLevel.debug),
  DiagnosticsProperty<String>('hidden', 'secret', level: DiagnosticLevel.hidden),
  DiagnosticsProperty<String>('err', 'bad', level: DiagnosticLevel.error),
  DiagnosticsProperty<String>.lazy('lazyThrows', () => throw FlutterError('x'), description: 'had description'),
  StringProperty('emptyUnquoted', '', quoted: false, ifEmpty: '<none>'),
  DiagnosticsProperty<String>('emptyRaw', '', ifEmpty: '<blank>'),
];

SceneTree bigTree() => SceneTree(
  'root',
  properties: allKinds(),
  children: <SceneTree>[
    SceneTree('a', properties: <DiagnosticsNode>[IntProperty('depth', 1)]),
    SceneTree(
      'b',
      style: DiagnosticsTreeStyle.offstage,
      children: <SceneTree>[
        SceneTree('b1', properties: <DiagnosticsNode>[StringProperty('s', 'v')]),
        SceneTree('b2', style: DiagnosticsTreeStyle.dense, properties: <DiagnosticsNode>[IntProperty('n', 2), FlagProperty('f', value: true, ifTrue: 'yes')]),
      ],
    ),
    SceneTree('c', emptyBody: '<no props>'),
  ],
);

const List<DiagnosticsTreeStyle> treeStyles = <DiagnosticsTreeStyle>[
  DiagnosticsTreeStyle.sparse,
  DiagnosticsTreeStyle.offstage,
  DiagnosticsTreeStyle.dense,
  DiagnosticsTreeStyle.transition,
  DiagnosticsTreeStyle.error,
  DiagnosticsTreeStyle.whitespace,
  DiagnosticsTreeStyle.flat,
  DiagnosticsTreeStyle.singleLine,
  DiagnosticsTreeStyle.errorProperty,
  DiagnosticsTreeStyle.shallow,
  DiagnosticsTreeStyle.truncateChildren,
];

SceneTree styledTree(DiagnosticsTreeStyle style) => SceneTree(
  'styled',
  style: style,
  properties: <DiagnosticsNode>[
    StringProperty('name', 'Größe straße'),
    DoubleProperty('width', 10.5, unit: 'px'),
    StringProperty('multi', 'first\nsecond', quoted: false),
  ],
  children: <SceneTree>[
    SceneTree('first', style: style, properties: <DiagnosticsNode>[IntProperty('i', 1)]),
    SceneTree(
      'second',
      style: style,
      children: <SceneTree>[
        SceneTree('leaf1', style: style),
        SceneTree('leaf2', style: DiagnosticsTreeStyle.sparse, properties: <DiagnosticsNode>[FlagProperty('on', value: true, ifTrue: 'on')]),
      ],
    ),
    SceneTree('third', style: DiagnosticsTreeStyle.transition, properties: <DiagnosticsNode>[MessageProperty('m', 'transition child')]),
  ],
);

const String longText =
    'The overflowing RenderFlex has an orientation of Axis.horizontal. '
    'The edge of the RenderFlex that is overflowing has been marked in the rendering '
    'with a yellow and black striped pattern. Café — naïve 😀 emoji 😀😀 and 中文字符 text '
    'to check UTF-16 widths. Averyveryveryveryveryveryveryveryverylongwordthatcannotwrap here.';

DiagnosticsNode wrapTree(DiagnosticsTreeStyle rootStyle, DiagnosticsTreeStyle propertyStyle) => DiagnosticsBlock(
  name: 'Wrapping straße',
  style: rootStyle,
  description: 'A block with long text',
  properties: <DiagnosticsNode>[
    DiagnosticsNode.message(longText, style: propertyStyle),
    DiagnosticsProperty<String>('long property name that also wraps when it is long enough', longText, style: propertyStyle),
    DiagnosticsProperty<String>(null, 'no wrap: $longText', allowWrap: false, style: propertyStyle),
    StringProperty('quoted', 'short', style: propertyStyle),
    // A wrappable non-ASCII name before a value that must not wrap: the
    // wrappable range ends where the name ends, in UTF-16 units.
    DiagnosticsProperty<String>('Größe 😀😀😀 straße', 'a b c d e f g h no wrap: $longText', allowWrap: false, style: propertyStyle),
    // A name that must not wrap before a value starting with a space: the
    // space opening the wrappable range is a break point.
    DiagnosticsProperty<String>(
      'a long property name that is not allowed to wrap at all',
      ' value with one leading space and more words to wrap',
      allowNameWrap: false,
      style: propertyStyle,
    ),
    DiagnosticsNode.message('  indented line\n    more indented\nback', style: propertyStyle),
  ],
  children: <DiagnosticsNode>[
    DiagnosticsBlock(
      name: 'child block',
      style: DiagnosticsTreeStyle.error,
      properties: <DiagnosticsNode>[DiagnosticsNode.message(longText, style: DiagnosticsTreeStyle.flat)],
    ),
    DiagnosticsNode.message('final child message that is fairly long and will need to be wrapped somewhere', style: propertyStyle),
  ],
);

DiagnosticsNode truncTree() {
  DiagnosticsNode deep(int level, int width) => DiagnosticsBlock(
    name: 'level $level',
    style: DiagnosticsTreeStyle.sparse,
    allowTruncate: true,
    properties: <DiagnosticsNode>[IntProperty('level', level)],
    children: level == 0 ? const <DiagnosticsNode>[] : <DiagnosticsNode>[
      for (int i = 0; i < width; i++) deep(level - 1, width),
    ],
  );
  return deep(3, 4);
}

String _jsonOf(Object? v) => jsonEncode(v);

void main() {
  test('diagnostics', () {
    final out = <String, Object?>{};

    final SceneTree tree = bigTree();
    out['big.toStringDeep'] = tree.toStringDeep();
    out['big.toStringDeep.wrap40'] = tree.toStringDeep(wrapWidth: 40);
    out['big.toStringDeep.prefixed'] = tree.toStringDeep(prefixLineOne: '>> ', prefixOtherLines: '.. ');
    out['big.toString'] = tree.toString();
    out['big.toString.debug'] = tree.toString(minLevel: DiagnosticLevel.debug);
    out['big.toStringShallow'] = tree.toStringShallow();
    out['big.toStringShallow.joiner'] = tree.toStringShallow(joiner: '\n', minLevel: DiagnosticLevel.hidden);
    out['big.toStringShort'] = tree.toStringShort();
    for (final DiagnosticLevel level in DiagnosticLevel.values) {
      out['big.minLevel.${level.name}'] = tree.toStringDeep(minLevel: level);
    }
    out['big.timeline'] = tree.toDiagnosticsNode().toTimelineArguments();
    out['big.json.default'] = jsonDecode(_jsonOf(tree.toDiagnosticsNode().toJsonMap(const DiagnosticsSerializationDelegate())));
    out['big.json.full'] = jsonDecode(_jsonOf(tree.toDiagnosticsNode().toJsonMap(
      const DiagnosticsSerializationDelegate(subtreeDepth: 3, includeProperties: true),
    )));
    out['big.json.iterative'] = jsonDecode(_jsonOf(tree.toDiagnosticsNode().toJsonMapIterative(
      const DiagnosticsSerializationDelegate(subtreeDepth: 5),
    )));

    final List<DiagnosticsNode> kinds = allKinds();
    for (var i = 0; i < kinds.length; i++) {
      final DiagnosticsNode p = kinds[i];
      out['prop.$i.toString'] = p.toString();
      out['prop.$i.dense'] = p.toString(parentConfiguration: denseTextConfiguration);
      out['prop.$i.toStringDeep'] = p.toStringDeep(minLevel: DiagnosticLevel.hidden);
      out['prop.$i.level'] = p.level.name;
      out['prop.$i.showName'] = p.showName;
      out['prop.$i.json'] = jsonDecode(_jsonOf(p.toJsonMap(const DiagnosticsSerializationDelegate(includeProperties: true))));
    }

    for (final DiagnosticsTreeStyle style in treeStyles) {
      final SceneTree t = styledTree(style);
      out['style.${style.name}.deep'] = t.toStringDeep();
      out['style.${style.name}.asRoot'] = t.toDiagnosticsNode(name: 'named root', style: style).toStringDeep();
      out['style.${style.name}.toString'] = t.toDiagnosticsNode(style: style).toString();
      out['style.${style.name}.json'] = jsonDecode(_jsonOf(t.toDiagnosticsNode().toJsonMap(
        const DiagnosticsSerializationDelegate(subtreeDepth: 2),
      )));
    }

    const List<int> widths = <int>[20, 40, 65, 100];
    const List<(DiagnosticsTreeStyle, DiagnosticsTreeStyle)> wrapStyles = <(DiagnosticsTreeStyle, DiagnosticsTreeStyle)>[
      (DiagnosticsTreeStyle.error, DiagnosticsTreeStyle.singleLine),
      (DiagnosticsTreeStyle.error, DiagnosticsTreeStyle.errorProperty),
      (DiagnosticsTreeStyle.sparse, DiagnosticsTreeStyle.flat),
      (DiagnosticsTreeStyle.whitespace, DiagnosticsTreeStyle.singleLine),
      (DiagnosticsTreeStyle.transition, DiagnosticsTreeStyle.shallow),
    ];
    for (final (DiagnosticsTreeStyle root, DiagnosticsTreeStyle prop) in wrapStyles) {
      for (final int w in widths) {
        out['wrap.${root.name}.${prop.name}.$w'] =
            TextTreeRenderer(wrapWidth: w, wrapWidthProperties: w ~/ 2).render(wrapTree(root, prop));
      }
      out['wrap.${root.name}.${prop.name}.deep'] = wrapTree(root, prop).toStringDeep();
    }

    for (final int n in <int>[0, 1, 2, 5]) {
      out['trunc.$n'] = TextTreeRenderer(maxDescendentsTruncatableNode: n).render(truncTree());
    }
    out['truncateChildren'] = DiagnosticsBlock(
      name: 'Block',
      style: DiagnosticsTreeStyle.truncateChildren,
      children: <DiagnosticsNode>[truncTree(), DiagnosticsNode.message('tail')],
    ).toStringDeep();
    out['truncateChildren.oneChild'] = DiagnosticsBlock(
      style: DiagnosticsTreeStyle.truncateChildren,
      children: <DiagnosticsNode>[DiagnosticsNode.message('only')],
    ).toStringDeep();
    out['truncateChildren.none'] = DiagnosticsBlock(name: 'Empty', style: DiagnosticsTreeStyle.truncateChildren).toStringDeep();

    final block = DiagnosticsBlock(
      name: 'Errors',
      style: DiagnosticsTreeStyle.error,
      linePrefix: '| ',
      value: 7,
      description: 'Something went wrong',
      level: DiagnosticLevel.error,
      properties: <DiagnosticsNode>[
        DiagnosticsNode.message('Summary line.', level: DiagnosticLevel.summary),
        DiagnosticsBlock(name: 'Hint', style: DiagnosticsTreeStyle.whitespace, properties: <DiagnosticsNode>[DiagnosticsNode.message('Try this instead.', level: DiagnosticLevel.hint)]),
      ],
      children: <DiagnosticsNode>[SceneTree('culprit', properties: <DiagnosticsNode>[IntProperty('id', 3)]).toDiagnosticsNode(name: 'The culprit')],
    );
    out['block.deep'] = block.toStringDeep();
    out['block.toString'] = block.toString();
    out['block.json'] = jsonDecode(_jsonOf(block.toJsonMap(const DiagnosticsSerializationDelegate(subtreeDepth: 2, includeProperties: true))));
    out['block.noName'] = DiagnosticsBlock(description: 'no name', properties: <DiagnosticsNode>[IntProperty('x', 1)]).toStringDeep();

    final plain = PlainTree(<DiagnosticsNode>[IntProperty('a', 1), StringProperty('b', 'two')]);
    out['plain.toString'] = plain.toString();
    out['plain.toStringShallow'] = plain.toStringShallow();
    out['plain.toStringDeep'] = plain.toStringDeep();
    final Point2 point = Point2(1.5, 0);
    out['point.toString'] = point.toString();
    out['point.toString.fine'] = point.toString(minLevel: DiagnosticLevel.fine);
    out['point.deep'] = point.toDiagnosticsNode(name: 'pt').toStringDeep();
    out['point.json'] = jsonDecode(_jsonOf(point.toDiagnosticsNode().toJsonMap(const DiagnosticsSerializationDelegate(includeProperties: true))));

    File('build/diagnostics.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
