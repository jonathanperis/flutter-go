// Reference side of flutter-go's text input differential test
// (services/text_input.dart, autofill.dart, keyboard_inserted_content.dart,
// clipboard.dart): TextInputConfiguration/TextInputType/TextInputStyle/
// AutofillConfiguration encodings and toString, TextEditingValue JSON and
// replaced, the bytes TextInputConnection and TextInputControl send on
// flutter/textinput, the client callbacks and replies for every incoming
// platform call, the system context menu's messages and Clipboard's.
// The Go twin is ../../text_input_test.go.
// Output: build/text_input.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:ui' show Locale;

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];

const JSONMethodCodec jsonMethod = JSONMethodCodec();
const JSONMessageCodec jsonMessage = JSONMessageCodec();

String bytes(ByteData? data) =>
    data == null ? 'null' : utf8.decode(data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes));

TestDefaultBinaryMessenger get messenger =>
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

class LogClient with TextInputClient implements DeltaTextInputClient {
  LogClient(this.name, this.value);
  final String name;
  TextEditingValue value;

  void add(String s) => log.add('$name.$s');

  @override
  TextEditingValue? get currentTextEditingValue => value;
  @override
  AutofillScope? get currentAutofillScope => null;
  @override
  void updateEditingValue(TextEditingValue value) => add('updateEditingValue $value');
  @override
  void updateEditingValueWithDeltas(List<TextEditingDelta> deltas) {
    for (final TextEditingDelta d in deltas) {
      add('delta ${d.runtimeType} ${d.oldText} ${d.selection} ${d.composing} -> ${d.apply(value)}');
    }
  }

  @override
  void performAction(TextInputAction action) => add('performAction $action');
  @override
  void insertContent(KeyboardInsertedContent content) =>
      add('insertContent $content ${content.hasData}');
  @override
  void performPrivateCommand(String action, Map<String, dynamic> data) =>
      add('performPrivateCommand $action ${jsonEncode(data)}');
  @override
  void updateFloatingCursor(RawFloatingCursorPoint point) =>
      add('updateFloatingCursor ${point.state} ${point.offset} ${point.startLocation}');
  @override
  void showAutocorrectionPromptRect(int start, int end) => add('showAutocorrectionPromptRect $start $end');
  @override
  bool onFocusReceived() {
    add('onFocusReceived');
    return true;
  }

  @override
  void connectionClosed() => add('connectionClosed');
  @override
  void didChangeInputControl(TextInputControl? oldControl, TextInputControl? newControl) =>
      add('didChangeInputControl ${oldControl is LogControl ? 'log' : oldControl == null ? 'null' : 'platform'} '
          '${newControl is LogControl ? 'log' : newControl == null ? 'null' : 'platform'}');
  @override
  void showToolbar() => add('showToolbar');
  @override
  void insertTextPlaceholder(Size size) => add('insertTextPlaceholder $size');
  @override
  void removeTextPlaceholder() => add('removeTextPlaceholder');
  @override
  void performSelector(String selectorName) => add('performSelector $selectorName');
}

class LogControl with TextInputControl {
  void add(String s) => log.add('control.$s');
  @override
  void attach(TextInputClient client, TextInputConfiguration configuration) =>
      add('attach ${configuration.inputType}');
  @override
  void detach(TextInputClient client) => add('detach');
  @override
  void show() => add('show');
  @override
  void hide() => add('hide');
  @override
  void updateConfig(TextInputConfiguration configuration) => add('updateConfig ${configuration.inputType}');
  @override
  void setEditingState(TextEditingValue value) => add('setEditingState $value');
  @override
  void setEditableSizeAndTransform(Size editableBoxSize, Matrix4 transform) =>
      add('setEditableSizeAndTransform $editableBoxSize');
  @override
  void setComposingRect(Rect rect) => add('setComposingRect $rect');
  @override
  void setCaretRect(Rect rect) => add('setCaretRect $rect');
  @override
  void setSelectionRects(List<SelectionRect> selectionRects) => add('setSelectionRects $selectionRects');
  @override
  void updateStyle(TextInputStyle style) => add('updateStyle ${jsonEncode(style.toJson())}');
  @override
  void requestAutofill() => add('requestAutofill');
  @override
  void finishAutofillContext({bool shouldSave = true}) => add('finishAutofillContext $shouldSave');
}

class LogScribble implements ScribbleClient {
  LogScribble(this.elementIdentifier, this.bounds, {this.always = false});
  final bool always;
  @override
  final String elementIdentifier;
  @override
  final Rect bounds;
  @override
  void onScribbleFocus(Offset offset) => log.add('scribble.$elementIdentifier.focus $offset');
  @override
  bool isInScribbleRect(Rect rect) => always || bounds.overlaps(rect);
}

class LogAutofillClient implements AutofillClient {
  LogAutofillClient(this.autofillId, this.textInputConfiguration);
  @override
  final String autofillId;
  @override
  final TextInputConfiguration textInputConfiguration;
  @override
  void autofill(TextEditingValue newEditingValue) => log.add('autofill.$autofillId $newEditingValue');
}

class LogScope with AutofillScopeMixin implements AutofillScope {
  final Map<String, AutofillClient> clients = <String, AutofillClient>{};
  @override
  Iterable<AutofillClient> get autofillClients => clients.values;
  @override
  AutofillClient? getAutofillClient(String autofillId) => clients[autofillId];
}

class AutofillLogClient extends LogClient {
  AutofillLogClient(super.name, super.value, this.scope);
  final AutofillScope scope;
  @override
  AutofillScope? get currentAutofillScope => scope;
}

Future<void> platform(String method, Object? args) async {
  final ByteData message = jsonMessage.encodeMessage(<String, Object?>{'method': method, 'args': args})!;
  await messenger.handlePlatformMessage('flutter/textinput', message, (ByteData? reply) {
    log.add('reply $method ${bytes(reply)}');
  });
}

void configurations() {
  final List<TextInputConfiguration> configs = <TextInputConfiguration>[
    const TextInputConfiguration(),
    const TextInputConfiguration(viewId: 3, readOnly: true, obscureText: true, autocorrect: false),
    const TextInputConfiguration(obscureText: true, smartDashesType: SmartDashesType.enabled),
    const TextInputConfiguration(smartQuotesType: SmartQuotesType.disabled, enableSuggestions: false),
    const TextInputConfiguration(enableInteractiveSelection: false, actionLabel: 'Go!'),
    const TextInputConfiguration(inputAction: TextInputAction.newline, keyboardAppearance: Brightness.dark),
    const TextInputConfiguration(textCapitalization: TextCapitalization.sentences),
    const TextInputConfiguration(enableIMEPersonalizedLearning: false, allowedMimeTypes: <String>['image/png', 'image/gif']),
    const TextInputConfiguration(enableDeltaModel: true, enableInlinePrediction: true),
    const TextInputConfiguration(enableInlinePrediction: false, hintLocales: null),
    TextInputConfiguration(hintLocales: <Locale>[const Locale('en', 'US'), const Locale('pt'), const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant', countryCode: 'TW')]),
    const TextInputConfiguration(
      autofillConfiguration: AutofillConfiguration(
        uniqueIdentifier: 'u1',
        autofillHints: <String>[AutofillHints.email, AutofillHints.username],
        currentEditingValue: TextEditingValue(text: 'me@x', selection: TextSelection.collapsed(offset: 4)),
        hintText: 'Your mail',
      ),
    ),
    const TextInputConfiguration(
      autofillConfiguration: AutofillConfiguration(
        uniqueIdentifier: 'u2',
        autofillHints: <String>[],
        currentEditingValue: TextEditingValue.empty,
      ),
    ),
  ];
  for (final TextInputType type in TextInputType.values) {
    configs.add(TextInputConfiguration(inputType: type));
  }
  for (final bool s in <bool>[false, true]) {
    for (final bool d in <bool>[false, true]) {
      for (final bool p in <bool>[false, true]) {
        configs.add(TextInputConfiguration(
          inputType: TextInputType.numberWithOptions(signed: s, decimal: d, password: p),
        ));
      }
    }
  }
  for (final TextInputAction action in TextInputAction.values) {
    configs.add(TextInputConfiguration(inputAction: action));
  }
  for (final TextCapitalization c in TextCapitalization.values) {
    configs.add(TextInputConfiguration(textCapitalization: c));
  }
  for (final TextInputConfiguration c in configs) {
    log.add('json ${jsonEncode(c.toJson())}');
    log.add('str $c');
  }
  final TextInputConfiguration base = configs[1];
  log.add('copy ${base.copyWith()}');
  log.add('copy ${base.copyWith(viewId: 9, inputType: TextInputType.phone, readOnly: false, obscureText: false, autocorrect: true, smartDashesType: SmartDashesType.enabled, smartQuotesType: SmartQuotesType.enabled, enableSuggestions: false, enableInteractiveSelection: false, actionLabel: 'a', inputAction: TextInputAction.search, keyboardAppearance: Brightness.dark, textCapitalization: TextCapitalization.words, enableIMEPersonalizedLearning: false, allowedMimeTypes: <String>['x'], enableDeltaModel: true, hintLocales: <Locale>[const Locale('fr')], enableInlinePrediction: false)}');
  log.add('eq ${base == base.copyWith()} ${base == base.copyWith(readOnly: false)} ${const TextInputConfiguration() == const TextInputConfiguration()}');
  for (final TextInputType t in TextInputType.values) {
    log.add('type $t ${jsonEncode(t.toJson())}');
  }
  const TextInputStyle style = TextInputStyle(
    fontFamily: 'Roboto',
    fontSize: 16.5,
    fontWeight: FontWeight.w300,
    textDirection: TextDirection.rtl,
    textAlign: TextAlign.end,
    letterSpacing: 0.1,
    wordSpacing: 2,
    lineHeight: 1.25,
  );
  log.add('style ${jsonEncode(style.toJson())}');
  log.add('style ${jsonEncode(const TextInputStyle(textDirection: TextDirection.ltr, textAlign: TextAlign.justify).toJson())}');
  for (final FontWeight w in FontWeight.values) {
    log.add('weight ${jsonEncode(TextInputStyle(fontWeight: w, textDirection: TextDirection.ltr, textAlign: TextAlign.start).toJson())}');
  }
  log.add('autofill ${configs[11].autofillConfiguration}');
  log.add('autofill ${AutofillConfiguration.disabled} ${AutofillConfiguration.disabled.toJson()}');
}

void editingValues() {
  final List<TextEditingValue> values = <TextEditingValue>[
    TextEditingValue.empty,
    const TextEditingValue(text: 'hello', selection: TextSelection(baseOffset: 1, extentOffset: 4)),
    const TextEditingValue(
      text: 'a😀b👍🏽c',
      selection: TextSelection(baseOffset: 5, extentOffset: 1, affinity: TextAffinity.upstream, isDirectional: true),
      composing: TextRange(start: 1, end: 3),
    ),
    const TextEditingValue(text: 'x', selection: TextSelection.collapsed(offset: 1, affinity: TextAffinity.upstream)),
  ];
  for (final TextEditingValue v in values) {
    final String json = jsonEncode(v.toJSON());
    log.add('value $v $json ${v.isComposingRangeValid}');
    log.add('fromJSON ${TextEditingValue.fromJSON(jsonDecode(json) as Map<String, dynamic>)}');
  }
  for (final String json in <String>[
    '{"text": "abc"}',
    '{"text": "abc", "selectionBase": 1}',
    '{"text": "abc", "selectionBase": 1, "selectionExtent": 2, "selectionAffinity": "TextAffinity.upstream", "selectionIsDirectional": true, "composingBase": 0, "composingExtent": 3}',
    '{"text": "abc", "selectionBase": 2, "selectionExtent": 2, "selectionAffinity": "bogus"}',
    '{"text": "\\ud83d\\ude00z", "selectionBase": 2, "selectionExtent": 3, "composingBase": 0, "composingExtent": 2}',
  ]) {
    try {
      final TextEditingValue v = TextEditingValue.fromJSON(jsonDecode(json) as Map<String, dynamic>);
      log.add('fromJSON $json -> $v ${jsonEncode(v.toJSON())}');
    } on AssertionError catch (e) {
      log.add('fromJSON $json throws ${e.message}');
    }
  }
  const String text = 'From a 😀 false 👍🏽 proposition';
  final List<TextSelection> selections = <TextSelection>[
    const TextSelection.collapsed(offset: 0),
    const TextSelection.collapsed(offset: 7),
    const TextSelection(baseOffset: 5, extentOffset: 13),
    const TextSelection(baseOffset: 13, extentOffset: 5),
    const TextSelection(baseOffset: 20, extentOffset: 30),
    const TextSelection(baseOffset: 8, extentOffset: 8, affinity: TextAffinity.upstream, isDirectional: true),
    const TextSelection(baseOffset: 6, extentOffset: 10, isDirectional: true),
  ];
  final List<TextRange> ranges = <TextRange>[
    const TextRange(start: 0, end: 0),
    const TextRange(start: 4, end: 5),
    const TextRange(start: 5, end: 6),
    const TextRange(start: 7, end: 9),
    const TextRange(start: 8, end: 9),
    const TextRange(start: 12, end: 13),
    const TextRange(start: 13, end: 14),
    const TextRange(start: 16, end: 20),
    const TextRange(start: 0, end: 31),
    TextRange.empty,
  ];
  for (final TextSelection s in selections) {
    for (final TextRange r in ranges) {
      for (final String with_ in <String>['', 'AA', '😀', 'xyz']) {
        final TextEditingValue v = TextEditingValue(
          text: text,
          selection: s,
          composing: const TextRange(start: 2, end: 9),
        ).replaced(r, with_);
        log.add('replaced $s $r "$with_" -> ${v.text.length} ${v.selection} ${v.composing} ${jsonEncode(v.text)}');
      }
    }
  }
}

Future<void> connectionScene() async {
  TextInputConnection.debugResetId(to: 5);
  final client = LogClient('c1', const TextEditingValue(text: 'abc', selection: TextSelection.collapsed(offset: 3)));
  final TextInputConnection connection = TextInput.attach(client, const TextInputConfiguration(actionLabel: 'ok'));
  log.add('attached ${connection.attached}');
  connection.show();
  connection.setEditingState(const TextEditingValue(text: 'abcd', selection: TextSelection(baseOffset: 0, extentOffset: 4)));
  final Matrix4 m = Matrix4.identity()..translateByDouble(10.0, 20.5, 0.0, 1.0)..scaleByDouble(2.0, 3.0, 1.0, 1.0);
  connection.setEditableSizeAndTransform(const Size(100.5, 30), m);
  connection.setEditableSizeAndTransform(const Size(100.5, 30), m); // cached
  connection.setEditableSizeAndTransform(const Size(100.5, 31), m);
  connection.setEditableSizeAndTransform(const Size(100.5, 31), Matrix4.identity());
  connection.setComposingRect(const Rect.fromLTWH(1, 2, 3, 4));
  connection.setComposingRect(const Rect.fromLTWH(1, 2, 3, 4)); // cached
  connection.setComposingRect(const Rect.fromLTRB(0, 0, double.infinity, 5));
  connection.setCaretRect(const Rect.fromLTWH(5.5, 6, 1, 14));
  connection.setCaretRect(const Rect.fromLTWH(5.5, 6, 1, 14)); // cached
  connection.setCaretRect(const Rect.fromLTWH(double.nan, 6, 1, 14));
  connection.setSelectionRects(const <SelectionRect>[]); // equal to the initial empty list
  connection.setSelectionRects(const <SelectionRect>[
    SelectionRect(position: 0, bounds: Rect.fromLTWH(0, 0, 7.5, 14)),
    SelectionRect(position: 1, bounds: Rect.fromLTWH(7.5, 0, 8, 14), direction: TextDirection.rtl),
  ]);
  connection.setSelectionRects(const <SelectionRect>[
    SelectionRect(position: 0, bounds: Rect.fromLTWH(0, 0, 7.5, 14)),
    SelectionRect(position: 1, bounds: Rect.fromLTWH(7.5, 0, 8, 14), direction: TextDirection.rtl),
  ]); // equal
  connection.updateStyle(const TextInputStyle(fontFamily: 'Ahem', fontSize: 14, fontWeight: FontWeight.bold, textDirection: TextDirection.ltr, textAlign: TextAlign.center, letterSpacing: 1, wordSpacing: 0.5, lineHeight: 1.5));
  // ignore: deprecated_member_use
  connection.setStyle(fontFamily: null, fontSize: 12, fontWeight: null, textDirection: TextDirection.rtl, textAlign: TextAlign.right);
  connection.requestAutofill();
  connection.updateConfig(const TextInputConfiguration(inputType: TextInputType.multiline, inputAction: TextInputAction.newline));
  TextInput.finishAutofillContext(shouldSave: false);
  TextInput.finishAutofillContext();
  TextInput.updateEditingValue(const TextEditingValue(text: 'zz'));

  // A custom input control: both receive requests; the platform gets the
  // none input type.
  final control = LogControl();
  TextInput.setInputControl(control);
  connection.show();
  connection.updateConfig(const TextInputConfiguration(inputType: TextInputType.emailAddress));
  connection.setEditingState(const TextEditingValue(text: 'q'));
  connection.setComposingRect(const Rect.fromLTWH(9, 9, 9, 9));
  connection.setCaretRect(const Rect.fromLTWH(8, 8, 8, 8));
  connection.setEditableSizeAndTransform(const Size(1, 1), Matrix4.identity());
  connection.setSelectionRects(const <SelectionRect>[SelectionRect(position: 4, bounds: Rect.fromLTWH(1, 1, 1, 1))]);
  connection.updateStyle(const TextInputStyle(textDirection: TextDirection.ltr, textAlign: TextAlign.left));
  connection.requestAutofill();
  TextInput.finishAutofillContext();
  TextInput.updateEditingValue(const TextEditingValue(text: 'from control'));
  final client2 = LogClient('c2', TextEditingValue.empty);
  final TextInputConnection connection2 = TextInput.attach(client2, const TextInputConfiguration(inputType: TextInputType.url));
  log.add('attached ${connection.attached} ${connection2.attached}');
  await platform('TextInputClient.updateEditingState', <Object?>[6, TextEditingValue.empty.toJSON()]);
  TextInput.setInputControl(null);
  connection2.show();
  TextInput.restorePlatformInputControl();
  connection2.show();
  connection2.close();
  log.add('closed ${connection2.attached}');
  connection2.close();
  await Future<void>.delayed(Duration.zero);
  log.add('after microtask');
  // Re-attach then close twice before the hide runs: one hide.
  final TextInputConnection connection3 = TextInput.attach(client2, const TextInputConfiguration());
  connection3.close();
  final TextInputConnection connection4 = TextInput.attach(client2, const TextInputConfiguration());
  connection4.close();
  await Future<void>.delayed(Duration.zero);
  log.add('after microtask 2');
  final TextInputConnection connection5 = TextInput.attach(client2, const TextInputConfiguration());
  connection5.close();
  final TextInputConnection connection6 = TextInput.attach(client2, const TextInputConfiguration());
  await Future<void>.delayed(Duration.zero);
  log.add('after microtask 3 ${connection6.attached}');
  connection6.connectionClosedReceived();
  log.add('received ${connection6.attached}');

  // Autofill scope: every client's configuration goes in "fields".
  final scope = LogScope();
  scope.clients['a'] = LogAutofillClient('a', const TextInputConfiguration(
    autofillConfiguration: AutofillConfiguration(uniqueIdentifier: 'a', autofillHints: <String>['username'], currentEditingValue: TextEditingValue(text: 'joe')),
  ));
  scope.clients['b'] = LogAutofillClient('b', const TextInputConfiguration(
    obscureText: true,
    autofillConfiguration: AutofillConfiguration(uniqueIdentifier: 'b', autofillHints: <String>['password'], currentEditingValue: TextEditingValue.empty, hintText: 'pw'),
  ));
  final trigger = AutofillLogClient('t', TextEditingValue.empty, scope);
  final TextInputConnection autofillConnection = scope.attach(trigger, const TextInputConfiguration(
    viewId: 2,
    readOnly: true,
    enableDeltaModel: true,
    inputType: TextInputType.name,
    actionLabel: 'x',
    autofillConfiguration: AutofillConfiguration(uniqueIdentifier: 'a', autofillHints: <String>['username'], currentEditingValue: TextEditingValue(text: 'joe')),
  ));
  await platform('TextInputClient.updateEditingStateWithTag', <Object?>[
    0,
    <String, Object?>{
      'a': const TextEditingValue(text: 'jim', selection: TextSelection.collapsed(offset: 3)).toJSON(),
      'zzz': TextEditingValue.empty.toJSON(),
    },
  ]);
  autofillConnection.close();
  await Future<void>.delayed(Duration.zero);
}

Future<void> incomingScene() async {
  TextInputConnection.debugResetId();
  final client = LogClient('in', const TextEditingValue(text: 'hello', selection: TextSelection.collapsed(offset: 5)));
  final TextInputConnection connection = TextInput.attach(client, const TextInputConfiguration());
  final int id = 1;
  await platform('TextInputClient.updateEditingState', <Object?>[id, const TextEditingValue(text: 'hi😀', selection: TextSelection(baseOffset: 2, extentOffset: 4), composing: TextRange(start: 0, end: 2)).toJSON()]);
  await platform('TextInputClient.updateEditingState', <Object?>[id + 1, TextEditingValue.empty.toJSON()]); // other client: ignored
  await platform('TextInputClient.updateEditingState', <Object?>[-1, const TextEditingValue(text: 'any').toJSON()]);
  await platform('TextInputClient.updateEditingStateWithDeltas', <Object?>[
    id,
    <String, Object?>{
      'deltas': <Object?>[
        <String, Object?>{'oldText': 'hello', 'deltaText': '!', 'deltaStart': 5, 'deltaEnd': 5, 'selectionBase': 6, 'selectionExtent': 6, 'composingBase': -1, 'composingExtent': -1},
        <String, Object?>{'oldText': 'hello', 'deltaText': '', 'deltaStart': 3, 'deltaEnd': 5, 'selectionBase': 3, 'selectionExtent': 3},
        <String, Object?>{'oldText': 'hello', 'deltaText': 'p', 'deltaStart': 0, 'deltaEnd': 1, 'selectionBase': 1, 'selectionExtent': 1, 'selectionAffinity': 'TextAffinity.upstream', 'selectionIsDirectional': true},
        <String, Object?>{'oldText': 'hello', 'deltaText': '', 'deltaStart': -1, 'deltaEnd': -1, 'selectionBase': 0, 'selectionExtent': 2, 'composingBase': 0, 'composingExtent': 5},
      ],
    },
  ]);
  for (final TextInputAction action in TextInputAction.values) {
    await platform('TextInputClient.performAction', <Object?>[id, action.toString()]);
  }
  await platform('TextInputClient.performAction', <Object?>[id, 'TextInputAction.bogus']);
  await platform('TextInputClient.performAction', <Object?>[
    id,
    'TextInputAction.commitContent',
    <String, Object?>{'mimeType': 'image/gif', 'data': <int>[0, 1, 255, 256, 513], 'uri': 'content://x/y.gif'},
  ]);
  await platform('TextInputClient.performAction', <Object?>[
    id,
    'TextInputAction.commitContent',
    <String, Object?>{'mimeType': 'image/png', 'uri': 'content://z'},
  ]);
  await platform('TextInputClient.performAction', <Object?>[
    id,
    'TextInputAction.commitContent',
    <String, Object?>{'mimeType': 'image/webp', 'data': <int>[], 'uri': 'content://e'},
  ]);
  await platform('TextInputClient.performSelectors', <Object?>[id, <String>['deleteBackward:', 'insertNewline:']]);
  await platform('TextInputClient.performPrivateCommand', <Object?>[id, <String, Object?>{'action': 'a1', 'data': <String, Object?>{'k': <Object?>[1, 2.5, 'x']}}]);
  await platform('TextInputClient.performPrivateCommand', <Object?>[id, <String, Object?>{'action': 'a2'}]);
  await platform('TextInputClient.updateFloatingCursor', <Object?>[id, 'FloatingCursorDragState.start', <String, Object?>{'X': 1, 'Y': 2}]);
  await platform('TextInputClient.updateFloatingCursor', <Object?>[id, 'FloatingCursorDragState.update', <String, Object?>{'X': 3, 'Y': 4.5}]);
  await platform('TextInputClient.updateFloatingCursor', <Object?>[id, 'FloatingCursorDragState.end', <String, Object?>{'X': 0, 'Y': 0}]);
  await platform('TextInputClient.updateFloatingCursor', <Object?>[id, 'FloatingCursorDragState.bogus', <String, Object?>{'X': 0, 'Y': 0}]);
  await platform('TextInputClient.showAutocorrectionPromptRect', <Object?>[id, 2, 4]);
  await platform('TextInputClient.showToolbar', <Object?>[id]);
  await platform('TextInputClient.insertTextPlaceholder', <Object?>[id, 10, 20.5]);
  await platform('TextInputClient.removeTextPlaceholder', <Object?>[id]);
  await platform('TextInputClient.requestExistingInputState', null);
  await platform('TextInputClient.onFocusReceived', <Object?>[id]);
  await platform('TextInputClient.onFocusReceived', <Object?>[99]);
  await platform('TextInputClient.unknownMethod', <Object?>[id]);
  await platform('TextInputClient.unknownMethod', <Object?>[42]);

  // Scribble.
  TextInput.registerScribbleElement('e1', LogScribble('e1', const Rect.fromLTWH(0, 0, 50, 50)));
  TextInput.registerScribbleElement('e2', LogScribble('e2', const Rect.fromLTWH(40, 40, 50, 50)));
  TextInput.registerScribbleElement('e3', LogScribble('e3', Rect.zero));
  TextInput.registerScribbleElement('e0', LogScribble('e0', const Rect.fromLTWH(45, 0, 10, 10)));
  TextInput.registerScribbleElement('e4', LogScribble('e4', Rect.zero, always: true));
  TextInput.registerScribbleElement('e5', LogScribble('e5', const Rect.fromLTRB(0, 0, double.infinity, 10), always: true));
  TextInput.registerScribbleElement('e6', LogScribble('e6', const Rect.fromLTRB(0, 0, double.nan, 10), always: true));
  TextInput.registerScribbleElement('e7', LogScribble('e7', const Rect.fromLTWH(500, 500, 1, 1), always: true));
  await platform('TextInputClient.focusElement', <Object?>['e2', 3, 4.5]);
  await platform('TextInputClient.focusElement', <Object?>['nope', 3, 4.5]);
  await platform('TextInputClient.requestElementsInRect', <Object?>[30, 0, 20, 60.5]);
  await platform('TextInputClient.requestElementsInRect', <Object?>[0, 0, 1000, 1000]);
  TextInput.unregisterScribbleElement('e1');
  await platform('TextInputClient.requestElementsInRect', <Object?>[0, 0, 1000, 1000]);
  await platform('TextInputClient.scribbleInteractionBegan', <Object?>[id]);
  log.add('scribble ${connection.scribbleInProgress}');
  await platform('TextInputClient.scribbleInteractionFinished', <Object?>[id]);
  log.add('scribble ${connection.scribbleInProgress}');
  TextInput.unregisterScribbleElement('e2');
  TextInput.unregisterScribbleElement('e3');
  TextInput.unregisterScribbleElement('e0');
  TextInput.unregisterScribbleElement('e4');
  TextInput.unregisterScribbleElement('e5');
  TextInput.unregisterScribbleElement('e6');
  TextInput.unregisterScribbleElement('e7');

  await platform('TextInputClient.onConnectionClosed', <Object?>[id]);
  connection.connectionClosedReceived();
  // With no connection: only the connection-independent methods act.
  await platform('TextInputClient.updateEditingState', <Object?>[id, TextEditingValue.empty.toJSON()]);
  await platform('TextInputClient.onFocusReceived', <Object?>[id]);
  await platform('TextInputClient.requestExistingInputState', null);
}

Future<void> platformScene() async {
  final List<String> platformCalls = <String>[];
  Map<String, Object?>? clipboard;
  messenger.setMockMessageHandler('flutter/platform', (ByteData? message) async {
    log.add('platform ${bytes(message)}');
    final MethodCall call = jsonMethod.decodeMethodCall(message);
    platformCalls.add(call.method);
    switch (call.method) {
      case 'Clipboard.setData':
        clipboard = (call.arguments as Map<String, Object?>);
      case 'Clipboard.getData':
        return jsonMethod.encodeSuccessEnvelope(clipboard);
      case 'Clipboard.hasStrings':
        return jsonMethod.encodeSuccessEnvelope(<String, Object?>{'value': clipboard?['text'] != null});
    }
    return jsonMethod.encodeSuccessEnvelope(null);
  });
  log.add('hasStrings ${await Clipboard.hasStrings()}');
  log.add('getData ${(await Clipboard.getData(Clipboard.kTextPlain))?.text}');
  await Clipboard.setData(const ClipboardData(text: 'copied 😀'));
  log.add('hasStrings ${await Clipboard.hasStrings()}');
  log.add('getData ${(await Clipboard.getData('text/plain'))?.text}');

  TextInputConnection.debugResetId();
  final TextInputConnection connection = TextInput.attach(LogClient('m', TextEditingValue.empty), const TextInputConfiguration());
  final controller = SystemContextMenuController(onSystemHide: () => log.add('onSystemHide'));
  await controller.showWithItems(const Rect.fromLTWH(1, 2, 30.5, 40), const <IOSSystemContextMenuItemData>[
    IOSSystemContextMenuItemDataCut(),
    IOSSystemContextMenuItemDataCopy(),
    IOSSystemContextMenuItemDataPaste(),
    IOSSystemContextMenuItemDataSelectAll(),
    IOSSystemContextMenuItemDataLookUp(title: 'Look Up'),
    IOSSystemContextMenuItemDataSearchWeb(title: 'Search Web'),
    IOSSystemContextMenuItemDataShare(title: 'Share… 😀'),
    IOSSystemContextMenuItemDataLiveText(),
  ]);
  log.add('visible ${controller.isVisible}');
  await controller.showWithItems(const Rect.fromLTWH(1, 2, 30.5, 40), const <IOSSystemContextMenuItemData>[
    IOSSystemContextMenuItemDataCut(),
    IOSSystemContextMenuItemDataCopy(),
    IOSSystemContextMenuItemDataPaste(),
    IOSSystemContextMenuItemDataSelectAll(),
    IOSSystemContextMenuItemDataLookUp(title: 'Look Up'),
    IOSSystemContextMenuItemDataSearchWeb(title: 'Search Web'),
    IOSSystemContextMenuItemDataShare(title: 'Share… 😀'),
    IOSSystemContextMenuItemDataLiveText(),
  ]); // the same: nothing sent
  await controller.showWithItems(const Rect.fromLTWH(1, 2, 30.5, 40), const <IOSSystemContextMenuItemData>[
    IOSSystemContextMenuItemDataCopy(),
  ]);
  // ignore: deprecated_member_use
  await controller.show(const Rect.fromLTWH(0, 0, 1, 1));
  await messenger.handlePlatformMessage(
    'flutter/platform',
    jsonMethod.encodeMethodCall(const MethodCall('ContextMenu.onDismissSystemContextMenu')),
    (ByteData? reply) => log.add('reply ${bytes(reply)}'),
  );
  log.add('visible ${controller.isVisible}');
  await controller.hide();
  await controller.showWithItems(const Rect.fromLTWH(5, 5, 5, 5), const <IOSSystemContextMenuItemData>[
    IOSSystemContextMenuItemDataSearchWeb(title: ''),
  ]);
  await controller.hide();
  log.add('visible ${controller.isVisible}');
  controller.dispose();
  connection.close();
  log.add('platform calls $platformCalls');
  messenger.setMockMessageHandler('flutter/platform', null);
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('text input', () async {
    FlutterError.onError = (FlutterErrorDetails details) {
      final Object e = details.exception;
      final String shown = e is MissingPluginException || e is FlutterError || e is PlatformException
          ? e.toString()
          : e.runtimeType.toString();
      log.add('error $shown | ${details.context} | ${details.library}');
    };
    messenger.setMockMessageHandler('flutter/textinput', (ByteData? message) async {
      log.add('out ${bytes(message)}');
      return jsonMethod.encodeSuccessEnvelope(null);
    });
    configurations();
    editingValues();
    await connectionScene();
    await incomingScene();

    // Failed platform calls are reported with their context.
    messenger.setMockMessageHandler('flutter/textinput', (ByteData? message) async {
      final MethodCall call = jsonMethod.decodeMethodCall(message);
      log.add('out ${call.method}');
      return jsonMethod.encodeErrorEnvelope(code: 'E${call.method.length}', message: 'failed ${call.method}', details: <int>[1]);
    });
    TextInputConnection.debugResetId();
    final TextInputConnection c = TextInput.attach(LogClient('e', TextEditingValue.empty), const TextInputConfiguration());
    c.show();
    c.setEditingState(TextEditingValue.empty);
    c.updateConfig(const TextInputConfiguration());
    c.setCaretRect(Rect.zero);
    c.requestAutofill();
    c.close();
    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);
    messenger.setMockMessageHandler('flutter/textinput', null);

    await platformScene();
    FlutterError.onError = FlutterError.presentError;
  });

  tearDownAll(() {
    File('build/text_input.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': log}));
  });
}
