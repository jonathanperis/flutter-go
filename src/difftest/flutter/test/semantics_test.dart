// Reference side of flutter-go's semantics differential test (Flutter
// 3.50.1's package:flutter/semantics.dart and dart:ui's semantics.dart):
// SemanticsNode trees built directly (no widgets) with configurations,
// merging, tags, custom actions, transforms, sort keys and text directions,
// printed with toStringDeep in both DebugSemanticsDumpOrders (and at
// minLevel hidden), SemanticsData.toString/toJson, every
// SemanticsUpdateBuilder call SemanticsOwner.sendSemanticsUpdate makes
// (through a spy builder the binding returns), traversal grafting,
// action routing (performAction / performActionAt), the debug role checks,
// configuration absorb/compatibility, AttributedString, SemanticsLabelBuilder,
// SemanticsEvent.toMap and its StandardMessageCodec bytes, Locale and
// TextSelection. Hashes are masked (#HASH). Output: build/semantics.json as
// {"log": [...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> entries = <String>[];
void log(String s) => entries.add(s.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH'));

String d(double v) => v.toString();

String ints(Int32List l) => '[${l.join(',')}]';

String doubles(Float64List l) => '[${l.map(d).join(',')}]';

String attrs(List<StringAttribute>? l) => l == null ? 'null' : '[${l.join(', ')}]';

class SpyBuilder extends Fake implements ui.SemanticsUpdateBuilder {
  final ui.SemanticsUpdateBuilder _builder = ui.SemanticsUpdateBuilder();

  @override
  void updateNode({
    required int id,
    required ui.SemanticsFlags flags,
    required int actions,
    required int maxValueLength,
    required int currentValueLength,
    required int textSelectionBase,
    required int textSelectionExtent,
    required int platformViewId,
    required int scrollChildren,
    required int scrollIndex,
    required int traversalParent,
    required double scrollPosition,
    required double scrollExtentMax,
    required double scrollExtentMin,
    required Rect rect,
    required String identifier,
    required String label,
    required List<StringAttribute> labelAttributes,
    required String value,
    required List<StringAttribute> valueAttributes,
    required String increasedValue,
    required List<StringAttribute> increasedValueAttributes,
    required String decreasedValue,
    required List<StringAttribute> decreasedValueAttributes,
    required String hint,
    required List<StringAttribute> hintAttributes,
    required String tooltip,
    required TextDirection? textDirection,
    required Float64List transform,
    required Float64List hitTestTransform,
    required Int32List childrenInTraversalOrder,
    required Int32List childrenInHitTestOrder,
    required Int32List additionalActions,
    int headingLevel = 0,
    String linkUrl = '',
    ui.SemanticsRole role = ui.SemanticsRole.none,
    required List<String>? controlsNodes,
    ui.SemanticsValidationResult validationResult = ui.SemanticsValidationResult.none,
    ui.SemanticsHitTestBehavior hitTestBehavior = ui.SemanticsHitTestBehavior.defer,
    required ui.SemanticsInputType inputType,
    required ui.Locale? locale,
    required String minValue,
    required String maxValue,
  }) {
    log(
      'updateNode id=$id flags=${flags.toStrings()} actions=$actions rect=$rect '
      'identifier="$identifier" label="$label" ${attrs(labelAttributes)} value="$value" ${attrs(valueAttributes)} '
      'increased="$increasedValue" ${attrs(increasedValueAttributes)} decreased="$decreasedValue" ${attrs(decreasedValueAttributes)} '
      'hint="$hint" ${attrs(hintAttributes)} tooltip="$tooltip" textDirection=$textDirection '
      'selection=$textSelectionBase,$textSelectionExtent platformViewId=$platformViewId '
      'maxValueLength=$maxValueLength currentValueLength=$currentValueLength '
      'scrollChildren=$scrollChildren scrollIndex=$scrollIndex scroll=${d(scrollPosition)},${d(scrollExtentMax)},${d(scrollExtentMin)} '
      'transform=${doubles(transform)} hitTestTransform=${doubles(hitTestTransform)} '
      'traversal=${ints(childrenInTraversalOrder)} hitTest=${ints(childrenInHitTestOrder)} '
      'additionalActions=${ints(additionalActions)} traversalParent=$traversalParent '
      'headingLevel=$headingLevel linkUrl="$linkUrl" role=$role controlsNodes=$controlsNodes '
      'validationResult=$validationResult hitTestBehavior=$hitTestBehavior inputType=$inputType '
      'locale=$locale minValue="$minValue" maxValue="$maxValue"',
    );
  }

  @override
  void updateCustomAction({required int id, String? label, String? hint, int overrideId = -1}) {
    log('updateCustomAction id=$id label=$label hint=$hint overrideId=$overrideId');
    _builder.updateCustomAction(id: id, label: label, hint: hint, overrideId: overrideId);
  }

  @override
  ui.SemanticsUpdate build() => _builder.build();
}

class SemanticsDiffBinding extends AutomatedTestWidgetsFlutterBinding {
  @override
  ui.SemanticsUpdateBuilder createSemanticsUpdateBuilder() => SpyBuilder();
}

SemanticsNode nodeAt(Rect rect) => SemanticsNode()..rect = rect;

SemanticsConfiguration boundaryConfig({bool merging = false}) => SemanticsConfiguration()
  ..isSemanticBoundary = true
  ..isMergingSemanticsOfDescendants = merging;

void dump(String name, SemanticsNode node) {
  log('$name traversal:\n${node.toStringDeep()}');
  log('$name inverseHitTest:\n${node.toStringDeep(childOrder: DebugSemanticsDumpOrder.inverseHitTest)}');
}

void dumpHidden(String name, SemanticsNode node) {
  log('$name hidden:\n${node.toStringDeep(minLevel: DiagnosticLevel.hidden)}');
}

void data(String name, SemanticsNode node) {
  final SemanticsData data = node.getSemanticsData();
  log('$name data: $data');
  log('$name data deep:\n${data.toDiagnosticsNode().toStringDeep(minLevel: DiagnosticLevel.hidden)}');
  log('$name json: ${jsonEncode(node.toJson())}');
}

void guard(String name, void Function() f) {
  try {
    f();
    log('$name: ok');
  } on FlutterError catch (e) {
    log('$name: FlutterError: ${e.message}');
  } on AssertionError catch (e) {
    log('$name: AssertionError: ${e.message}');
  }
}

String hex(ByteData? data) {
  if (data == null) {
    return 'null';
  }
  final buffer = StringBuffer();
  for (var i = 0; i < data.lengthInBytes; i++) {
    buffer.write(data.getUint8(i).toRadixString(16).padLeft(2, '0'));
  }
  return buffer.toString();
}

SemanticsOwner newOwner(String name) {
  return SemanticsOwner(
    onSemanticsUpdate: (ui.SemanticsUpdate update) {
      log('$name: update built');
      update.dispose();
    },
  );
}

void sceneValues() {
  log('--- values');
  final string1 = AttributedString(
    'string1',
    attributes: <StringAttribute>[SpellOutStringAttribute(range: const TextRange(start: 0, end: 4))],
  );
  final string2 = AttributedString(
    'abc\ndef',
    attributes: <StringAttribute>[
      LocaleStringAttribute(locale: const ui.Locale('es', 'MX'), range: const TextRange(start: 1, end: 3)),
    ],
  );
  log('${string1 + string2}');
  log('${AttributedString('') + string2}');
  log('${string1 + AttributedString('')}');
  log('equal ${string1 == AttributedString('string1', attributes: string1.attributes)} '
      '${string1 == AttributedString('string1', attributes: <StringAttribute>[SpellOutStringAttribute(range: const TextRange(start: 0, end: 4))])} '
      '${AttributedString('x') == AttributedString('x')}');
  log('${AttributedString('😀a', attributes: <StringAttribute>[SpellOutStringAttribute(range: const TextRange(start: 2, end: 3))]) + string1}');
  log(AttributedStringProperty('label', string2).toString());
  log(AttributedStringProperty('label', string2).toString(parentConfiguration: singleLineTextConfiguration));
  log(AttributedStringProperty('label', AttributedString('')).toString(minLevel: DiagnosticLevel.hidden));
  log('interesting ${AttributedStringProperty('label', AttributedString('')).isInteresting} '
      '${AttributedStringProperty('label', AttributedString(''), showWhenEmpty: true).isInteresting} '
      '${AttributedStringProperty('label', null, defaultValue: null).isInteresting}');

  for (final locale in <ui.Locale>[
    const ui.Locale('en'),
    const ui.Locale('en', 'US'),
    const ui.Locale('iw', 'DD'),
    const ui.Locale('en', ''),
    const ui.Locale.fromSubtags(scriptCode: 'Hant', countryCode: 'TW'),
    const ui.Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hans'),
  ]) {
    log('locale $locale ${locale.toLanguageTag()} ${locale.languageCode} ${locale.countryCode} '
        '${locale == const ui.Locale('en')} ${locale == ui.Locale(locale.languageCode, locale.countryCode)}');
  }

  for (final s in <TextSelection>[
    const TextSelection(baseOffset: 1, extentOffset: 4),
    const TextSelection(baseOffset: 4, extentOffset: 1, isDirectional: true),
    const TextSelection.collapsed(offset: 3, affinity: TextAffinity.upstream),
    const TextSelection(baseOffset: -1, extentOffset: -1),
  ]) {
    log('selection $s start=${s.start} end=${s.end} base=${s.base} extent=${s.extent} valid=${s.isValid}');
  }

  CustomSemanticsAction.resetForTests();
  const a1 = CustomSemanticsAction(label: 'one');
  const a2 = CustomSemanticsAction.overridingAction(hint: 'two', action: SemanticsAction.tap);
  log('$a1 $a2');
  log('ids ${CustomSemanticsAction.getIdentifier(a2)} ${CustomSemanticsAction.getIdentifier(a1)} '
      '${CustomSemanticsAction.getIdentifier(const CustomSemanticsAction(label: 'one'))}');
  log('$a1 $a2 ${CustomSemanticsAction.getAction(0)} ${CustomSemanticsAction.getAction(7)}');

  log('flags ${SemanticsFlag.values.map((SemanticsFlag f) => f.name).join(',')}');
  log('actions ${SemanticsAction.values.map((SemanticsAction a) => '${a.name}=${a.index}').join(',')}');
  log('roles ${SemanticsRole.values.join(',')}');

  for (final builder in <SemanticsLabelBuilder>[
    SemanticsLabelBuilder(),
    SemanticsLabelBuilder(textDirection: TextDirection.ltr, separator: ', '),
    SemanticsLabelBuilder(textDirection: TextDirection.rtl, separator: ''),
  ]) {
    builder
      ..addPart('one')
      ..addPart('')
      ..addPart('two', textDirection: TextDirection.rtl)
      ..addPart('three', textDirection: TextDirection.ltr)
      ..addPart('four');
    log('label "${builder.build()}" ${builder.length}');
  }

  final events = <SemanticsEvent>[
    const AnnounceSemanticsEvent('hello', TextDirection.rtl, 2),
    const AnnounceSemanticsEvent('né 😀', TextDirection.ltr, 0, assertiveness: Assertiveness.assertive),
    const TooltipSemanticsEvent('tip'),
    const LongPressSemanticsEvent(),
    const TapSemanticEvent(),
    const FocusSemanticEvent(),
  ];
  for (final event in events) {
    log('event $event ${jsonEncode(event.toMap())} ${jsonEncode(event.toMap(nodeId: 70000))}');
    log('bytes ${hex(const StandardMessageCodec().encodeMessage(event.toMap(nodeId: 5)))}');
  }
  log('codec ${hex(const StandardMessageCodec().encodeMessage(<Object?>[null, true, false, 1, -1, 2147483648, -2147483649, 1.5, 'a' * 300, Uint8List.fromList(<int>[1, 2]), Int32List.fromList(<int>[3]), Int64List.fromList(<int>[4]), Float64List.fromList(<double>[0.5]), Float32List.fromList(<double>[0.25]), <String, Object?>{'k': <Object?>[]}]))}');
  log('decoded ${const StandardMessageCodec().decodeMessage(const StandardMessageCodec().encodeMessage(<String, Object?>{'base': 1, 'extent': 3, 'x': <Object?>[1.5, 'y']}))}');
}

void sceneTreesAndOrder() {
  log('--- trees and traversal order');
  debugResetSemanticsIdCounter();
  // Children in a grid, paint order shuffled: geometric order in ltr/rtl.
  SemanticsNode grid(TextDirection? direction) {
    final children = <SemanticsNode>[
      nodeAt(const Rect.fromLTRB(100, 0, 150, 40)),
      nodeAt(const Rect.fromLTRB(0, 50, 40, 90)),
      nodeAt(const Rect.fromLTRB(0, 0, 40, 40)),
      nodeAt(const Rect.fromLTRB(50, 0, 90, 40)),
      nodeAt(const Rect.fromLTRB(50, 50, 90, 90)),
      nodeAt(const Rect.fromLTRB(20, 20, 70, 70)), // a knot
      nodeAt(const Rect.fromLTRB(100, 45, 150, 95)),
    ];
    final root = nodeAt(const Rect.fromLTRB(0, 0, 200, 100));
    final config = SemanticsConfiguration();
    if (direction != null) {
      config.textDirection = direction;
    }
    root.updateWith(config: config, childrenInInversePaintOrder: children);
    return root;
  }

  dump('grid null', grid(null));
  dump('grid ltr', grid(TextDirection.ltr));
  dump('grid rtl', grid(TextDirection.rtl));

  // Sort keys: groups of compatible keys sort, the rest keep their place.
  final keyed = <SemanticsNode>[];
  final keys = <SemanticsSortKey?>[
    const OrdinalSortKey(3),
    const OrdinalSortKey(1),
    null,
    const OrdinalSortKey(2, name: 'b'),
    const OrdinalSortKey(0, name: 'b'),
    const OrdinalSortKey(5, name: 'a'),
    const OrdinalSortKey(-1),
    const OrdinalSortKey(-1),
  ];
  for (var i = 0; i < keys.length; i++) {
    final config = SemanticsConfiguration()..label = 'k$i'..textDirection = TextDirection.ltr;
    if (keys[i] != null) {
      config.sortKey = keys[i];
    }
    keyed.add(nodeAt(Rect.fromLTWH(i * 10.0, 0, 10, 10))..updateWith(config: config));
  }
  final keyedRoot = nodeAt(const Rect.fromLTRB(0, 0, 100, 10))
    ..updateWith(
      config: SemanticsConfiguration()..textDirection = TextDirection.ltr,
      childrenInInversePaintOrder: keyed,
    );
  dump('keys', keyedRoot);

  // 40 children with identical rects and ties everywhere: Dart's quicksort
  // decides their order.
  final many = <SemanticsNode>[
    for (var i = 0; i < 40; i++) nodeAt(Rect.fromLTWH((i * 7 % 5) * 10.0, (i * 3 % 4) * 10.0, 10, 10)),
  ];
  final manyRoot = nodeAt(const Rect.fromLTRB(0, 0, 100, 100))
    ..updateWith(
      config: SemanticsConfiguration()..textDirection = TextDirection.ltr,
      childrenInInversePaintOrder: many,
    );
  log('many traversal ${manyRoot.debugListChildrenInOrder(DebugSemanticsDumpOrder.traversalOrder).map((SemanticsNode n) => n.id).join(',')}');
  final manyKeys = <SemanticsNode>[
    for (var i = 0; i < 40; i++)
      nodeAt(Rect.fromLTWH(i * 1.0, 0, 10, 10))
        ..updateWith(config: SemanticsConfiguration()..sortKey = OrdinalSortKey((i * 5 % 7).toDouble())),
  ];
  final manyKeysRoot = nodeAt(const Rect.fromLTRB(0, 0, 100, 100))
    ..updateWith(config: SemanticsConfiguration(), childrenInInversePaintOrder: manyKeys);
  log('many keys ${manyKeysRoot.debugListChildrenInOrder(DebugSemanticsDumpOrder.traversalOrder).map((SemanticsNode n) => n.id).join(',')}');

  // Transformed children: rotated and scaled ones sort by transformed
  // corners.
  final rotated = nodeAt(const Rect.fromLTRB(0, 0, 30, 10))..transform = Matrix4.rotationZ(0.3);
  final rotated2 = nodeAt(const Rect.fromLTRB(0, 0, 30, 10))
    ..transform = (Matrix4.translationValues(40, 5, 0)..multiply(Matrix4.rotationZ(-2.5)));
  final scaledChild = nodeAt(const Rect.fromLTRB(0, 0, 10, 10))..transform = Matrix4.diagonal3Values(3, 3, 1);
  final translated = nodeAt(const Rect.fromLTRB(0, 0, 10, 10))..transform = Matrix4.translationValues(60, 1, 0);
  final identityChild = nodeAt(const Rect.fromLTRB(5, 5, 15, 15))..transform = Matrix4.identity();
  final transformedRoot = nodeAt(const Rect.fromLTRB(0, 0, 100, 100))
    ..updateWith(
      config: SemanticsConfiguration()..textDirection = TextDirection.rtl,
      childrenInInversePaintOrder: <SemanticsNode>[rotated, rotated2, scaledChild, translated, identityChild],
    );
  dump('transformed', transformedRoot);
  dumpHidden('transformed', transformedRoot);
  log('identity child transform ${identityChild.transform}');
  data('rotated', rotated);
}

void sceneMerging() {
  log('--- merging');
  debugResetSemanticsIdCounter();
  CustomSemanticsAction.resetForTests();
  const tagA = SemanticsTag('tagA');
  const tagB = SemanticsTag('tagB');
  final leaf1 = nodeAt(const Rect.fromLTRB(0, 0, 10, 10))
    ..tags = <SemanticsTag>{tagB}
    ..updateWith(
      config: SemanticsConfiguration()
        ..label = 'first'
        ..textDirection = TextDirection.rtl
        ..hint = 'h1'
        ..value = 'v1'
        ..increasedValue = 'v2'
        ..onIncrease = () {}
        ..isChecked = true
        ..isToggled = false
        ..controlsNodes = <String>{'x', 'y'}
        ..validationResult = SemanticsValidationResult.valid
        ..tooltip = 'tip'
        ..headingLevel = 3
        ..customSemanticsActions = <CustomSemanticsAction, VoidCallback>{
          const CustomSemanticsAction(label: 'custom1'): () {},
        }
        ..hintOverrides = const SemanticsHintOverrides(onTapHint: 'tap me')
        ..minValue = '1',
    );
  final leaf2 = nodeAt(const Rect.fromLTRB(10, 0, 20, 10))
    ..tags = <SemanticsTag>{tagA, tagB}
    ..updateWith(
      config: SemanticsConfiguration()
        ..attributedLabel = AttributedString(
          'second',
          attributes: <StringAttribute>[SpellOutStringAttribute(range: const TextRange(start: 0, end: 3))],
        )
        ..textDirection = TextDirection.ltr
        ..hint = 'h2'
        ..isBlockingUserActions = true
        ..onTap = () {}
        ..onDidGainAccessibilityFocus = () {}
        ..isSelected = true
        ..isEnabled = true
        ..controlsNodes = <String>{'y', 'z'}
        ..validationResult = SemanticsValidationResult.invalid
        ..role = SemanticsRole.tab
        ..inputType = ui.SemanticsInputType.search
        ..maxValue = '9'
        ..identifier = 'id2'
        ..scrollPosition = 3.5
        ..scrollExtentMax = 10
        ..scrollExtentMin = 0
        ..scrollChildCount = 4
        ..scrollIndex = 1
        ..customSemanticsActions = <CustomSemanticsAction, VoidCallback>{
          const CustomSemanticsAction(label: 'custom2'): () {},
          const CustomSemanticsAction(label: 'custom1'): () {},
        },
    );
  final middle = nodeAt(const Rect.fromLTRB(0, 0, 20, 10))
    ..updateWith(config: SemanticsConfiguration(), childrenInInversePaintOrder: <SemanticsNode>[leaf1, leaf2]);
  final mergeRoot = nodeAt(const Rect.fromLTRB(0, 0, 30, 30))
    ..tags = <SemanticsTag>{tagA}
    ..updateWith(
      config: boundaryConfig(merging: true)
        ..label = 'root'
        ..textDirection = TextDirection.ltr
        ..locale = const ui.Locale('fr', 'CA')
        ..isButton = true,
      childrenInInversePaintOrder: <SemanticsNode>[middle],
    );
  dump('merge root', mergeRoot);
  dumpHidden('merge root', mergeRoot);
  data('merge root', mergeRoot);
  data('leaf2', leaf2);
  log('merged flags ${leaf1.isMergedIntoParent} ${leaf2.isMergedIntoParent} ${middle.isMergedIntoParent} ${mergeRoot.isPartOfNodeMerging}');
  log('equal data ${mergeRoot.getSemanticsData() == mergeRoot.getSemanticsData()} ${leaf1.getSemanticsData() == leaf2.getSemanticsData()}');
  log('hint overrides ${const SemanticsHintOverrides(onTapHint: 'a') == const SemanticsHintOverrides(onTapHint: 'a')} '
      '${const SemanticsHintOverrides(onTapHint: 'a', onLongPressHint: 'b').toStringDeep()}');
}

void sceneConfiguration() {
  log('--- configuration');
  SemanticsConfiguration describe(String name, SemanticsConfiguration c) {
    log('$name: annotated=${c.hasBeenAnnotated} label="${c.label}" ${c.attributedLabel} hint="${c.hint}" '
        'value="${c.value}" flags=${c.isButton},${c.isChecked},${c.isCheckStateMixed},${c.isSelected},${c.isEnabled},'
        '${c.isToggled},${c.isExpanded},${c.isFocused},${c.isRequired},${c.isLink},${c.isHeader},${c.isTextField} '
        'textDirection=${c.textDirection} heading=${c.headingLevel} role=${c.role} validation=${c.validationResult} '
        'hitTest=${c.hitTestBehavior} controls=${c.controlsNodes} focusBlock=${c.accessibilityFocusBlockType} '
        'traversal=${c.traversalParentIdentifier},${c.traversalChildIdentifier} min=${c.minValue} max=${c.maxValue} '
        'tooltip="${c.tooltip}" identifier="${c.identifier}" sortKey=${c.sortKey} actions=${<String>[for (final SemanticsAction a in SemanticsAction.values) if (c.getActionHandler(a) != null) a.name]}');
    return c;
  }

  final parent = SemanticsConfiguration()
    ..label = 'parent'
    ..textDirection = TextDirection.ltr
    ..isButton = true
    ..onTap = () {}
    ..controlsNodes = <String>{'a'}
    ..validationResult = SemanticsValidationResult.valid
    ..traversalParentIdentifier = 'p';
  final child = SemanticsConfiguration()
    ..label = 'child'
    ..textDirection = TextDirection.rtl
    ..hint = 'child hint'
    ..isChecked = false
    ..isCheckStateMixed = true
    ..onLongPress = () {}
    ..onScrollUp = () {}
    ..isBlockingUserActions = true
    ..onDidLoseAccessibilityFocus = () {}
    ..controlsNodes = <String>{'b', 'a'}
    ..validationResult = SemanticsValidationResult.invalid
    ..accessibilityFocusBlockType = AccessibilityFocusBlockType.blockNode
    ..headingLevel = 2
    ..sortKey = const OrdinalSortKey(4)
    ..hitTestBehavior = ui.SemanticsHitTestBehavior.opaque
    ..minValue = '0'
    ..tooltip = 'tt'
    ..identifier = 'cid';
  describe('parent', parent);
  describe('child', child);
  log('compatible ${parent.isCompatibleWith(child)} ${child.isCompatibleWith(parent)} '
      '${parent.isCompatibleWith(SemanticsConfiguration())} ${parent.isCompatibleWith(null)}');
  final pairs = <List<SemanticsConfiguration>>[
    <SemanticsConfiguration>[SemanticsConfiguration()..onTap = () {}, SemanticsConfiguration()..onTap = () {}],
    <SemanticsConfiguration>[SemanticsConfiguration()..isButton = true, SemanticsConfiguration()..isButton = true],
    <SemanticsConfiguration>[SemanticsConfiguration()..isButton = true, SemanticsConfiguration()..isHeader = true],
    <SemanticsConfiguration>[SemanticsConfiguration()..value = 'a', SemanticsConfiguration()..value = 'b'],
    <SemanticsConfiguration>[SemanticsConfiguration()..isTextField = true, SemanticsConfiguration()..isImage = true],
    <SemanticsConfiguration>[SemanticsConfiguration()..isTextField = true, SemanticsConfiguration()..label = 'x'],
    <SemanticsConfiguration>[SemanticsConfiguration()..role = SemanticsRole.tab, SemanticsConfiguration()..isLink = true],
    <SemanticsConfiguration>[SemanticsConfiguration()..label = 'a', SemanticsConfiguration()..hitTestBehavior = ui.SemanticsHitTestBehavior.transparent],
    <SemanticsConfiguration>[SemanticsConfiguration()..platformViewId = 1, SemanticsConfiguration()..platformViewId = 2],
    <SemanticsConfiguration>[SemanticsConfiguration()..localeForSubtree = const ui.Locale('en'), SemanticsConfiguration()..label = 'x'],
    <SemanticsConfiguration>[SemanticsConfiguration()..traversalChildIdentifier = 1, SemanticsConfiguration()..label = 'x'],
    <SemanticsConfiguration>[SemanticsConfiguration(), SemanticsConfiguration()..traversalChildIdentifier = 1],
    <SemanticsConfiguration>[SemanticsConfiguration()..accessibilityFocusBlockType = AccessibilityFocusBlockType.blockSubtree, SemanticsConfiguration()..label = 'y'],
    <SemanticsConfiguration>[SemanticsConfiguration()..minValue = '1', SemanticsConfiguration()..minValue = '2'],
    <SemanticsConfiguration>[SemanticsConfiguration()..isFocused = true, SemanticsConfiguration()..isFocused = false],
  ];
  for (var i = 0; i < pairs.length; i++) {
    log('pair $i ${pairs[i][0].isCompatibleWith(pairs[i][1])}');
  }
  parent.absorb(child);
  describe('absorbed', parent);
  final copy = parent.copy();
  describe('copy', copy);
  final unannotated = SemanticsConfiguration()..label = 'keep';
  unannotated.absorb(SemanticsConfiguration());
  describe('absorb unannotated', unannotated);
  final focus = SemanticsConfiguration()
    ..isFocusable = true;
  describe('focusable', focus);
  focus.isFocused = true;
  focus.isFocusable = false;
  describe('unfocusable', focus);
  final values = SemanticsConfiguration()
    ..isChecked = null
    ..isCheckStateMixed = false
    ..isExpanded = false
    ..isRequired = true
    ..linkUrl = Uri.parse('https://flutter.dev/a?b=1')
    ..isLink = true
    ..indexInParent = 3
    ..scrollIndex = 2
    ..scrollChildCount = 2;
  describe('values', values);
  final cursor = SemanticsConfiguration();
  cursor.onMoveCursorForwardByWord = (bool b) => log('word forward $b');
  log('cursor ${cursor.onMoveCursorForwardByWord} ${cursor.onMoveCursorForwardByCharacter != null}');
}

void sceneOwner() {
  log('--- owner');
  debugResetSemanticsIdCounter();
  CustomSemanticsAction.resetForTests();
  final owner = newOwner('owner');
  final root = SemanticsNode.root(owner: owner, showOnScreen: () => log('root showOnScreen'))
    ..rect = const Rect.fromLTRB(0, 0, 400, 300);
  final a = nodeAt(const Rect.fromLTRB(0, 0, 100, 100));
  final b = nodeAt(const Rect.fromLTRB(100, 0, 200, 100))..transform = Matrix4.translationValues(50, 20, 0);
  final b1 = nodeAt(const Rect.fromLTRB(0, 0, 20, 20));
  final b2 = nodeAt(const Rect.fromLTRB(20, 0, 40, 20));
  final c = nodeAt(const Rect.fromLTRB(0, 100, 300, 200));
  b.updateWith(
    config: boundaryConfig(merging: true)
      ..label = 'b'
      ..textDirection = TextDirection.ltr
      ..onTap = () { log('b tapped'); },
    childrenInInversePaintOrder: <SemanticsNode>[b1, b2],
  );
  b1.updateWith(
    config: SemanticsConfiguration()
      ..label = 'b1'
      ..textDirection = TextDirection.ltr
      ..onLongPress = () { log('b1 long pressed'); }
      ..customSemanticsActions = <CustomSemanticsAction, VoidCallback>{
        const CustomSemanticsAction(label: 'b1 custom'): () { log('b1 custom'); },
      },
  );
  b2.updateWith(
    config: SemanticsConfiguration()
      ..label = 'b2'
      ..textDirection = TextDirection.ltr
      ..onScrollToOffset = (Offset o) { log('b2 scroll to $o'); }
      ..onSetSelection = (TextSelection s) { log('b2 selection $s'); }
      ..onSetText = (String s) { log('b2 text $s'); }
      ..customSemanticsActions = <CustomSemanticsAction, VoidCallback>{
        const CustomSemanticsAction(label: 'b2 custom'): () { log('b2 custom'); },
      },
  );
  a.updateWith(
    config: SemanticsConfiguration()
      ..label = 'a'
      ..textDirection = TextDirection.ltr
      ..onTap = () { log('a tapped'); }
      ..textSelection = const TextSelection(baseOffset: 2, extentOffset: 1)
      ..platformViewId = 7
      ..maxValueLength = 10
      ..currentValueLength = 3
      ..linkUrl = Uri.parse('https://example.com/x')
      ..isLink = true
      ..headingLevel = 1
      ..locale = const ui.Locale('en', 'GB')
      ..indexInParent = 0,
  );
  c.updateWith(
    config: SemanticsConfiguration()
      ..label = 'c'
      ..textDirection = TextDirection.rtl
      ..value = '50'
      ..increasedValue = '60'
      ..decreasedValue = '40'
      ..onIncrease = () { log('c increase'); }
      ..onDecrease = () { log('c decrease'); }
      ..hintOverrides = const SemanticsHintOverrides(onLongPressHint: 'hold'),
  );
  root.updateWith(
    config: SemanticsConfiguration()..textDirection = TextDirection.ltr,
    childrenInInversePaintOrder: <SemanticsNode>[a, b, c],
  );
  owner.sendSemanticsUpdate();
  log('dirty after ${root.debugIsDirty} ${b1.debugIsDirty}');
  owner.sendSemanticsUpdate(); // nothing dirty: no update

  // Change a leaf of the merged node: the merge root is sent again.
  b2.updateWith(
    config: SemanticsConfiguration()
      ..label = 'b2 changed'
      ..textDirection = TextDirection.rtl,
  );
  log('b2 dirty ${b2.debugIsDirty}');
  owner.sendSemanticsUpdate();

  // Reorder and drop children; mark several dirty.
  c.rect = const Rect.fromLTRB(0, 100, 290, 200);
  a.rect = const Rect.fromLTRB(0, 0, 90, 100);
  root.updateWith(
    config: SemanticsConfiguration()..textDirection = TextDirection.rtl,
    childrenInInversePaintOrder: <SemanticsNode>[c, a],
  );
  log('b attached ${b.attached} ${b1.attached}');
  dump('owner root', root);
  owner.sendSemanticsUpdate();

  // Re-add b: it was detached, so it is sent again.
  root.updateWith(
    config: SemanticsConfiguration()..textDirection = TextDirection.rtl,
    childrenInInversePaintOrder: <SemanticsNode>[c, a, b],
  );
  owner.sendSemanticsUpdate();

  // Actions.
  owner.performAction(a.id, SemanticsAction.tap);
  owner.performAction(b.id, SemanticsAction.tap);
  owner.performAction(b.id, SemanticsAction.longPress);
  owner.performAction(b.id, SemanticsAction.customAction, CustomSemanticsAction.getIdentifier(const CustomSemanticsAction(label: 'b1 custom')));
  owner.performAction(b2.id, SemanticsAction.setText, 'typed');
  owner.performAction(b.id, SemanticsAction.scrollToOffset, Float64List.fromList(<double>[1.5, 2.5]));
  owner.performAction(b.id, SemanticsAction.setSelection, <dynamic, dynamic>{'base': 4, 'extent': 2});
  owner.performAction(c.id, SemanticsAction.increase);
  owner.performAction(c.id, SemanticsAction.decrease);
  owner.performAction(c.id, SemanticsAction.tap);
  owner.performAction(root.id, SemanticsAction.showOnScreen);
  owner.performAction(999, SemanticsAction.tap);
  owner.performActionAt(const Offset(10, 10), SemanticsAction.tap);
  owner.performActionAt(const Offset(160, 30), SemanticsAction.tap); // b is translated by (50, 20)
  owner.performActionAt(const Offset(160, 30), SemanticsAction.longPress);
  owner.performActionAt(const Offset(149, 30), SemanticsAction.tap);
  owner.performActionAt(const Offset(150, 150), SemanticsAction.increase);
  owner.performActionAt(const Offset(500, 500), SemanticsAction.tap);
  dumpHidden('owner b', b);
  data('owner a', a);
  data('owner c', c);
  log('root data ${root.getSemanticsData()}');
  log('owner node ${owner.getSemanticsNode(b.id) == b} ${owner.rootSemanticsNode == root}');
  owner.dispose();
}

void sceneTraversalGrafting() {
  log('--- traversal grafting');
  debugResetSemanticsIdCounter();
  final owner = newOwner('graft');
  final root = SemanticsNode.root(owner: owner)..rect = const Rect.fromLTRB(0, 0, 400, 400);
  final anchor = nodeAt(const Rect.fromLTRB(10, 10, 60, 60))..transform = Matrix4.translationValues(5, 5, 0);
  final anchorChild = nodeAt(const Rect.fromLTRB(0, 0, 20, 20));
  final overlay = nodeAt(const Rect.fromLTRB(100, 100, 300, 300))
    ..transform = (Matrix4.translationValues(10, 0, 0)..multiply(Matrix4.diagonal3Values(2, 2, 1)));
  final menu = nodeAt(const Rect.fromLTRB(0, 0, 50, 50));
  final orphan = nodeAt(const Rect.fromLTRB(60, 0, 90, 50));
  anchor.updateWith(
    config: SemanticsConfiguration()
      ..label = 'anchor'
      ..textDirection = TextDirection.ltr
      ..traversalParentIdentifier = 'portal',
    childrenInInversePaintOrder: <SemanticsNode>[anchorChild],
  );
  anchorChild.updateWith(config: SemanticsConfiguration()..label = 'anchor child'..textDirection = TextDirection.ltr);
  menu.updateWith(
    config: SemanticsConfiguration()
      ..label = 'menu item'
      ..textDirection = TextDirection.ltr
      ..traversalChildIdentifier = 'portal',
  );
  orphan.updateWith(
    config: SemanticsConfiguration()
      ..label = 'orphan'
      ..textDirection = TextDirection.ltr
      ..traversalChildIdentifier = 'nobody',
  );
  overlay.updateWith(config: SemanticsConfiguration(), childrenInInversePaintOrder: <SemanticsNode>[menu, orphan]);
  root.updateWith(
    config: SemanticsConfiguration()..textDirection = TextDirection.ltr,
    childrenInInversePaintOrder: <SemanticsNode>[anchor, overlay],
  );
  owner.sendSemanticsUpdate();
  log('menu traversal parent ${menu.traversalParent?.id}');
  dump('graft root', root);
  // A traversal parent without hit-test children.
  final lone = nodeAt(const Rect.fromLTRB(300, 0, 350, 50));
  final loneChild = nodeAt(const Rect.fromLTRB(0, 0, 10, 10))..transform = Matrix4.rotationZ(math.pi / 6);
  lone.updateWith(config: SemanticsConfiguration()..traversalParentIdentifier = 7);
  loneChild.updateWith(config: SemanticsConfiguration()..traversalChildIdentifier = 7);
  final holder = nodeAt(const Rect.fromLTRB(300, 300, 400, 400))
    ..updateWith(config: SemanticsConfiguration(), childrenInInversePaintOrder: <SemanticsNode>[loneChild]);
  root.updateWith(
    config: SemanticsConfiguration()..textDirection = TextDirection.ltr,
    childrenInInversePaintOrder: <SemanticsNode>[anchor, overlay, lone, holder],
  );
  owner.sendSemanticsUpdate();
  // Remove the overlay: the anchor's traversal children update.
  root.updateWith(
    config: SemanticsConfiguration()..textDirection = TextDirection.ltr,
    childrenInInversePaintOrder: <SemanticsNode>[anchor, lone, holder],
  );
  owner.sendSemanticsUpdate();
  dump('graft root after', root);
  owner.dispose();
}

void sceneRoleChecks() {
  log('--- role checks');
  SemanticsConfiguration role(SemanticsRole r) => SemanticsConfiguration()
    ..role = r
    ..textDirection = TextDirection.ltr;
  final cases = <String, List<Object>>{
    'tab no selected': <Object>[role(SemanticsRole.tab)],
    'tab no tap': <Object>[role(SemanticsRole.tab)..isSelected = false],
    'tab ok': <Object>[role(SemanticsRole.tab)..isSelected = true..onTap = () {}],
    'tab disabled': <Object>[role(SemanticsRole.tab)..isSelected = true..isEnabled = false],
    'empty tab bar': <Object>[role(SemanticsRole.tabBar)],
    'tab bar child': <Object>[role(SemanticsRole.tabBar), role(SemanticsRole.cell)],
    'table child': <Object>[role(SemanticsRole.table), role(SemanticsRole.cell)],
    'row outside table': <Object>[role(SemanticsRole.row)],
    'cell outside row': <Object>[role(SemanticsRole.cell)],
    'column header': <Object>[role(SemanticsRole.columnHeader)],
    'radio group': <Object>[
      role(SemanticsRole.radioGroup),
      SemanticsConfiguration()..isInMutuallyExclusiveGroup = true..isChecked = true,
      SemanticsConfiguration()..isInMutuallyExclusiveGroup = true..isChecked = true,
    ],
    'radio group ok': <Object>[
      role(SemanticsRole.radioGroup),
      SemanticsConfiguration()..isInMutuallyExclusiveGroup = true..isChecked = true,
      SemanticsConfiguration()..isInMutuallyExclusiveGroup = true..isChecked = false,
    ],
    'empty menu': <Object>[role(SemanticsRole.menu)],
    'empty menu bar': <Object>[role(SemanticsRole.menuBar)],
    'menu item': <Object>[role(SemanticsRole.menuItem)],
    'menu item checkbox': <Object>[role(SemanticsRole.menuItemCheckbox)],
    'menu item checkbox in menu': <Object>[role(SemanticsRole.menu), role(SemanticsRole.menuItemCheckbox)..isChecked = true],
    'menu item radio': <Object>[role(SemanticsRole.menuItemRadio)..isChecked = false],
    'alert live region': <Object>[role(SemanticsRole.alert)..liveRegion = true],
    'list item': <Object>[role(SemanticsRole.listItem)],
    'list item in list': <Object>[role(SemanticsRole.list), role(SemanticsRole.listItem)],
    'region': <Object>[role(SemanticsRole.region)],
    'region labelled': <Object>[role(SemanticsRole.region)..label = 'r'],
    'main in navigation': <Object>[role(SemanticsRole.navigation)..label = 'n', role(SemanticsRole.main)],
    'complementary': <Object>[role(SemanticsRole.complementary)],
    'progress bar missing': <Object>[role(SemanticsRole.progressBar)..value = '5'],
    'progress bar bad number': <Object>[role(SemanticsRole.progressBar)..value = 'x'..minValue = '0'..maxValue = '10'],
    'progress bar min max': <Object>[role(SemanticsRole.progressBar)..value = '5'..minValue = '10'..maxValue = '1e1'],
    'progress bar out of range': <Object>[role(SemanticsRole.progressBar)..value = ' 11 '..minValue = '0'..maxValue = '10'],
    'progress bar percent': <Object>[role(SemanticsRole.progressBar)..value = '150%'..minValue = '0'..maxValue = '10'],
    'progress bar ok': <Object>[role(SemanticsRole.progressBar)..value = '50%'..minValue = '-1.5'..maxValue = 'Infinity'],
    'slider missing': <Object>[role(SemanticsRole.slider)..value = '1'..minValue = '0'],
    'slider bad': <Object>[role(SemanticsRole.slider)..value = '1'..minValue = '0x1'..maxValue = '2'],
    'slider percent': <Object>[role(SemanticsRole.slider)..value = '1%0%'..minValue = '0'..maxValue = '200'],
    'slider range': <Object>[role(SemanticsRole.slider)..value = '-.5'..minValue = '0'..maxValue = '2'],
    'slider ok': <Object>[role(SemanticsRole.slider)..value = '1.'..minValue = '0'..maxValue = '2'],
    'drag handle': <Object>[role(SemanticsRole.dragHandle)],
    'expanded with expand': <Object>[SemanticsConfiguration()..isExpanded = true..onExpand = () {}],
    'collapsed with collapse': <Object>[SemanticsConfiguration()..isExpanded = false..onCollapse = () {}],
    'expand and collapse': <Object>[SemanticsConfiguration()..isExpanded = true..onExpand = () {}..onCollapse = () {}],
    'focus blocked': <Object>[SemanticsConfiguration()..isFocused = false..accessibilityFocusBlockType = AccessibilityFocusBlockType.blockNode],
  };
  for (final MapEntry<String, List<Object>> entry in cases.entries) {
    debugResetSemanticsIdCounter();
    final owner = newOwner(entry.key);
    final root = SemanticsNode.root(owner: owner)..rect = const Rect.fromLTRB(0, 0, 100, 100);
    final configs = entry.value.cast<SemanticsConfiguration>();
    final parent = nodeAt(const Rect.fromLTRB(0, 0, 50, 50));
    final children = <SemanticsNode>[
      for (var i = 1; i < configs.length; i++)
        nodeAt(Rect.fromLTWH(i * 5.0, 0, 5, 5))..updateWith(config: configs[i]),
    ];
    parent.updateWith(config: configs[0], childrenInInversePaintOrder: children);
    root.updateWith(config: SemanticsConfiguration(), childrenInInversePaintOrder: <SemanticsNode>[parent]);
    guard(entry.key, owner.sendSemanticsUpdate);
    owner.dispose();
  }
}

void sceneErrors() {
  log('--- errors');
  debugResetSemanticsIdCounter();
  guard('non-finite rect', () => SemanticsNode()..rect = const Rect.fromLTRB(0, 0, double.infinity, 1));
  guard('increase without increasedValue', () {
    SemanticsNode().updateWith(
      config: SemanticsConfiguration()
        ..value = 'v'
        ..textDirection = TextDirection.ltr
        ..onIncrease = () {},
    );
  });
  guard('data with label and no direction', () {
    (SemanticsNode()..updateWith(config: SemanticsConfiguration()..label = 'x')).getSemanticsData();
  });
  guard('merging without boundary', () => SemanticsConfiguration()..isMergingSemanticsOfDescendants = true);
  guard('attributed range', () => AttributedString('ab', attributes: <StringAttribute>[SpellOutStringAttribute(range: const TextRange(start: 0, end: 3))]));
  guard('duplicate builder', () {
    final config = SemanticsConfiguration();
    (ChildSemanticsConfigurationsResultBuilder()
          ..markAsMergeUp(config)
          ..markAsSiblingMergeGroup(<SemanticsConfiguration>[config]))
        .build();
  });
  final result = (ChildSemanticsConfigurationsResultBuilder()
        ..markAsMergeUp(SemanticsConfiguration())
        ..markAsSiblingMergeGroup(<SemanticsConfiguration>[SemanticsConfiguration(), SemanticsConfiguration()]))
      .build();
  log('result ${result.mergeUp.length} ${result.siblingMergeGroups.length} ${result.siblingMergeGroups.first.length}');
  final owner = newOwner('graft errors');
  final root = SemanticsNode.root(owner: owner)..rect = const Rect.fromLTRB(0, 0, 100, 100);
  final child = nodeAt(const Rect.fromLTRB(0, 0, 50, 50));
  final grandChild = nodeAt(const Rect.fromLTRB(0, 0, 10, 10));
  child.updateWith(
    config: SemanticsConfiguration()..traversalChildIdentifier = 'loop',
    childrenInInversePaintOrder: <SemanticsNode>[grandChild],
  );
  grandChild.updateWith(config: SemanticsConfiguration()..traversalParentIdentifier = 'loop');
  root.updateWith(config: SemanticsConfiguration(), childrenInInversePaintOrder: <SemanticsNode>[child]);
  guard('graft cycle', owner.sendSemanticsUpdate);
}

void sceneActionArgs() {
  log('--- action args');
  debugResetSemanticsIdCounter();
  CustomSemanticsAction.resetForTests();
  final owner = newOwner('args');
  final root = SemanticsNode.root(owner: owner)..rect = const Rect.fromLTRB(0, 0, 100, 100);
  final field = nodeAt(const Rect.fromLTRB(0, 0, 50, 50));
  final config = SemanticsConfiguration()
    ..label = 'field'
    ..textDirection = TextDirection.ltr
    ..onScrollToOffset = (Offset o) {
      log('scroll to $o');
    }
    ..onSetSelection = (TextSelection s) {
      log('selection $s');
    }
    ..onSetText = (String s) {
      log('text $s');
    }
    ..onMoveCursorForwardByCharacter = (bool b) {
      log('forward char $b');
    }
    ..onMoveCursorBackwardByWord = (bool b) {
      log('backward word $b');
    }
    ..onCopy = () {
      log('copy');
    }
    ..onDismiss = () {
      log('dismiss');
    }
    ..customSemanticsActions = <CustomSemanticsAction, VoidCallback>{
      const CustomSemanticsAction(label: 'x'): () {
        log('custom x');
      },
    };
  field.updateWith(config: config);
  final mergeParent = nodeAt(const Rect.fromLTRB(0, 0, 100, 100))
    ..updateWith(
      config: boundaryConfig(merging: true)
        ..customSemanticsActions = <CustomSemanticsAction, VoidCallback>{
          const CustomSemanticsAction(label: 'y'): () {
            log('custom y');
          },
        },
      childrenInInversePaintOrder: <SemanticsNode>[field],
    );
  root.updateWith(config: SemanticsConfiguration(), childrenInInversePaintOrder: <SemanticsNode>[mergeParent]);
  owner.sendSemanticsUpdate();
  final int x = CustomSemanticsAction.getIdentifier(const CustomSemanticsAction(label: 'x'));
  final int y = CustomSemanticsAction.getIdentifier(const CustomSemanticsAction(label: 'y'));
  log('ids $x $y');
  final int id = mergeParent.id;
  owner.performAction(id, SemanticsAction.scrollToOffset, Float64List.fromList(<double>[3.0, 4.5]));
  owner.performAction(id, SemanticsAction.setSelection, <dynamic, dynamic>{'base': 5, 'extent': 1});
  owner.performAction(id, SemanticsAction.setText, 'hello');
  owner.performAction(id, SemanticsAction.moveCursorForwardByCharacter, true);
  owner.performAction(id, SemanticsAction.moveCursorBackwardByWord, false);
  owner.performAction(id, SemanticsAction.copy);
  owner.performAction(id, SemanticsAction.dismiss);
  owner.performAction(id, SemanticsAction.customAction, x);
  owner.performAction(id, SemanticsAction.customAction, y);
  owner.performAction(id, SemanticsAction.customAction, 99);
  owner.performActionAt(const Offset(10, 10), SemanticsAction.customAction, x);
  owner.performActionAt(const Offset(60, 60), SemanticsAction.customAction, y);
  owner.performActionAt(const Offset(60, 60), SemanticsAction.setText, 'at');
  log('handlers ${config.onMoveCursorBackwardByWord} ${config.onMoveCursorBackwardByCharacter != null} '
      '${config.getActionHandler(SemanticsAction.moveCursorBackwardByWord) != null}');
  data('merge parent', mergeParent);
  owner.dispose();
}

void sceneExtra() {
  log('--- extra');
  debugResetSemanticsIdCounter();
  // A knot whose second center is below-left of the first (direction
  // between pi/2 and 3pi/4).
  final knotA = nodeAt(const Rect.fromLTRB(30, 30, 70, 70));
  final knotB = nodeAt(const Rect.fromLTRB(20, 42, 60, 82));
  final knotC = nodeAt(const Rect.fromLTRB(45, 35, 85, 75));
  for (final dir in <TextDirection>[TextDirection.ltr, TextDirection.rtl]) {
    final knotRoot = nodeAt(const Rect.fromLTRB(0, 0, 100, 100))
      ..updateWith(
        config: SemanticsConfiguration()..textDirection = dir,
        childrenInInversePaintOrder: <SemanticsNode>[knotA, knotB, knotC],
      );
    log('knot $dir ${knotRoot.debugListChildrenInOrder(DebugSemanticsDumpOrder.traversalOrder).map((SemanticsNode n) => n.id).join(',')}');
  }

  // A knot edge from a lower node to a taller one starting higher.
  final knotX = nodeAt(const Rect.fromLTRB(40, 40, 60, 50));
  final knotY = nodeAt(const Rect.fromLTRB(30, 30, 50, 88));
  final knot2 = nodeAt(const Rect.fromLTRB(0, 0, 100, 100))
    ..updateWith(
      config: SemanticsConfiguration()..textDirection = TextDirection.ltr,
      childrenInInversePaintOrder: <SemanticsNode>[knotX, knotY],
    );
  log('knot2 ${knot2.debugListChildrenInOrder(DebugSemanticsDumpOrder.traversalOrder).map((SemanticsNode n) => n.id).join(',')}');

  // Overlapping siblings: the last painted one is hit first.
  final owner = newOwner('extra');
  final root = SemanticsNode.root(owner: owner)..rect = const Rect.fromLTRB(0, 0, 100, 100);
  final under = nodeAt(const Rect.fromLTRB(0, 0, 60, 60))
    ..updateWith(config: SemanticsConfiguration()..onTap = () {
      log('under tapped');
    });
  final over = nodeAt(const Rect.fromLTRB(40, 40, 100, 100))
    ..updateWith(config: SemanticsConfiguration()..onTap = () {
      log('over tapped');
    });
  root.updateWith(config: SemanticsConfiguration(), childrenInInversePaintOrder: <SemanticsNode>[under, over]);
  owner.performActionAt(const Offset(50, 50), SemanticsAction.tap);
  owner.performActionAt(const Offset(10, 10), SemanticsAction.tap);
  owner.dispose();

  // Absorb: a child label without a direction, heading levels, traversal
  // identifiers.
  final parent = SemanticsConfiguration()
    ..label = 'p'
    ..textDirection = TextDirection.ltr
    ..headingLevel = 2
    ..traversalChildIdentifier = 'c';
  final child = SemanticsConfiguration()
    ..label = 'q'
    ..headingLevel = 3
    ..traversalParentIdentifier = 'p';
  parent.absorb(child);
  log('absorbed ${parent.attributedLabel} heading=${parent.headingLevel} '
      'traversal=${parent.traversalParentIdentifier},${parent.traversalChildIdentifier}');

  // Deprecated flags bit mask, through toJson.
  final focusable = nodeAt(const Rect.fromLTRB(0, 0, 10, 10))
    ..updateWith(config: SemanticsConfiguration()..isFocused = false..isRequired = false..isExpanded = true);
  log('focusable json ${jsonEncode(focusable.toJson())} flags=${focusable.getSemanticsData().flags}');
}

void main() {
  SemanticsDiffBinding();

  test('semantics', () {
    sceneValues();
    sceneTreesAndOrder();
    sceneMerging();
    sceneConfiguration();
    sceneOwner();
    sceneTraversalGrafting();
    sceneRoleChecks();
    sceneErrors();
    sceneActionArgs();
    sceneExtra();

    File('build/semantics.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': entries}));
  });
}
