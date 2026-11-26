// Reference side of flutter-go's differential test of the small services
// (UndoManager, Scribe, LiveText, ProcessText, SpellCheck): the exact bytes
// of every message they send, the replies they give the platform, their
// parsed results and errors, timed against a chain of microtasks. The Go
// twin is ../../undo_scribe_misc_test.go. Output:
// build/undo_scribe_misc.json.
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:ui' show Locale;

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];

void ticks(int n) {
  var i = 0;
  void step() {
    log.add('tick $i');
    i++;
    if (i < n) {
      scheduleMicrotask(step);
    }
  }

  scheduleMicrotask(step);
}

String bytes(ByteData? d) =>
    d == null ? 'null' : d.buffer.asUint8List(d.offsetInBytes, d.lengthInBytes).join(',');

String show(Object? v) {
  if (v is PlatformException) {
    return 'PlatformException(${v.code}, ${v.message}, ${v.details}, ${v.stacktrace})';
  }
  if (v is List<ProcessTextAction>) {
    return v.map((ProcessTextAction a) => '${a.id}:${a.label}').join(';');
  }
  return '$v';
}

void track(String name, Future<Object?> f) {
  f.then(
    (Object? v) => log.add('$name -> ${show(v)}'),
    onError: (Object e) => log.add('$name !! ${show(e)}'),
  );
}

Future<void> settle() => Future<void>.delayed(const Duration(milliseconds: 1));

TestDefaultBinaryMessenger get messenger =>
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

final Map<String, List<Object?>> replies = <String, List<Object?>>{};

void installPlatform(String channel, MethodCodec codec) {
  messenger.setMockMessageHandler(channel, (ByteData? message) async {
    final MethodCall call = codec.decodeMethodCall(message);
    log.add('$channel ${call.method} [${bytes(message)}]');
    final List<Object?>? queue = replies[call.method];
    final Object? reply = queue == null || queue.isEmpty ? null : queue.removeAt(0);
    if (reply is PlatformException) {
      return codec.encodeErrorEnvelope(code: reply.code, message: reply.message);
    }
    if (reply is _NoReply) {
      return null;
    }
    return codec.encodeSuccessEnvelope(reply);
  });
}

class _NoReply {
  const _NoReply();
}

class _UndoClient with UndoManagerClient {
  @override
  void handlePlatformUndo(UndoDirection direction) => log.add('handlePlatformUndo $direction');
  @override
  void undo() {}
  @override
  void redo() {}
  @override
  bool get canUndo => true;
  @override
  bool get canRedo => false;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('undo scribe misc', () async {
    installPlatform('flutter/undomanager', const JSONMethodCodec());
    installPlatform('flutter/scribe', const JSONMethodCodec());
    installPlatform('flutter/platform', const JSONMethodCodec());
    installPlatform('flutter/textinput', const JSONMethodCodec());
    installPlatform('flutter/processtext', const StandardMethodCodec());
    installPlatform('flutter/spellcheck', const StandardMethodCodec());

    // ---- UndoManager.
    log.add('scene undo');
    ticks(6);
    UndoManager.setUndoState();
    UndoManager.setUndoState(canUndo: true);
    UndoManager.setUndoState(canRedo: true);
    UndoManager.setUndoState(canUndo: true, canRedo: true);
    log.add('sent');
    await settle();
    UndoManager.client = _UndoClient();
    log.add('client ${UndoManager.client?.canUndo} ${UndoManager.client?.canRedo}');
    for (final Map<String, Object?> message in <Map<String, Object?>>[
      <String, Object?>{'method': 'UndoManagerClient.handleUndo', 'args': <Object?>['undo']},
      <String, Object?>{'method': 'UndoManagerClient.handleUndo', 'args': <Object?>['redo']},
      <String, Object?>{'method': 'UndoManagerClient.handleUndo', 'args': <Object?>['sideways']},
      <String, Object?>{'method': 'UndoManagerClient.other', 'args': <Object?>[]},
    ]) {
      ticks(4);
      await messenger.handlePlatformMessage(
        'flutter/undomanager',
        const JSONMessageCodec().encodeMessage(message),
        (ByteData? reply) => log.add('reply [${bytes(reply)}]'),
      );
      await settle();
    }
    UndoManager.client = null;
    // A failing platform: the error is reported.
    final FlutterExceptionHandler? oldOnError = FlutterError.onError;
    FlutterError.onError = (FlutterErrorDetails details) {
      log.add('reported ${show(details.exception)} | ${details.context} | ${details.library}');
    };
    replies['UndoManager.setUndoState'] = <Object?>[
      PlatformException(code: 'error', message: 'Channel failed'),
    ];
    ticks(6);
    UndoManager.setUndoState(canUndo: true);
    await settle();
    FlutterError.onError = oldOnError;

    // ---- Scribe.
    log.add('scene scribe');
    replies['Scribe.isFeatureAvailable'] = <Object?>[true, false, null];
    replies['Scribe.isStylusHandwritingAvailable'] = <Object?>[false, true, null];
    for (var i = 0; i < 3; i++) {
      ticks(6);
      track('isFeatureAvailable', Scribe.isFeatureAvailable());
      track('isStylusHandwritingAvailable', Scribe.isStylusHandwritingAvailable());
      await settle();
    }
    ticks(6);
    track('startStylusHandwriting', Scribe.startStylusHandwriting());
    await settle();
    await messenger.handlePlatformMessage(
      'flutter/scribe',
      const JSONMessageCodec().encodeMessage(<String, Object?>{'method': 'Scribe.unsupportedMessage'}),
      (ByteData? reply) => log.add('scribe reply [${bytes(reply)}]'),
    );

    // ---- LiveText.
    log.add('scene live text');
    replies['LiveText.isLiveTextInputAvailable'] = <Object?>[null, true, false];
    for (var i = 0; i < 3; i++) {
      ticks(6);
      track('isLiveTextInputAvailable', LiveText.isLiveTextInputAvailable());
      await settle();
    }
    ticks(6);
    track('startLiveTextInput', LiveText.startLiveTextInput());
    await settle();

    // ---- ProcessText.
    log.add('scene process text');
    final service = DefaultProcessTextService();
    replies['ProcessText.queryTextActions'] = <Object?>[
      <String, String>{'com.a.Translate': 'Translate'},
      <String, String>{'com.a.Define': 'Define', 'com.b.Search': 'Search', 'com.c.Upper': 'UPPER'},
      // The platform's order, not sorted: Dart lists the actions in it.
      <String, String>{'com.z.Zap': 'Zap', 'com.a.Alpha': 'Alpha', 'com.m.Mid': 'Mid', 'com.b.Beta': 'Beta'},
      <Object?, Object?>{'com.y.Second': 'Second', 'com.x.First': 'First'},
      null,
      PlatformException(code: 'busy', message: 'pending'),
      <Object?>['not', 'a', 'map'],
      const _NoReply(),
    ];
    for (var i = 0; i < 8; i++) {
      ticks(8);
      track('queryTextActions', service.queryTextActions());
      await settle();
    }
    replies['ProcessText.processTextAction'] = <Object?>['FLUTTER', null, PlatformException(code: 'x')];
    for (final readOnly in <bool>[false, true, false]) {
      ticks(6);
      track('processTextAction', service.processTextAction('com.c.Upper', 'flutter 🐦', readOnly));
      await settle();
    }

    // ---- SpellCheck.
    log.add('scene spell check');
    final spell = DefaultSpellCheckService();
    Map<String, Object?> span(int start, int end, List<String> suggestions) =>
        <String, Object?>{'startIndex': start, 'endIndex': end, 'suggestions': suggestions};
    replies['SpellCheck.initiateSpellCheck'] = <Object?>[
      <Object?>[span(0, 3, <String>['the', 'tea']), span(4, 8, <String>['word'])],
      <Object?>[span(0, 3, <String>['the', 'tea']), span(4, 8, <String>['word'])],
      <Object?>[span(2, 3, <String>['x'])],
      <Object?>[],
      PlatformException(code: 'pending'),
      'not a list',
      const _NoReply(),
    ];
    for (final (Locale locale, String text) in <(Locale, String)>[
      (const Locale('en', 'US'), 'teh wrod'),
      (const Locale('en', 'US'), 'teh wrod'),
      (const Locale('pt', 'BR'), 'teh wrod'),
      (const Locale('fr'), 'bonjour'),
      (const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant', countryCode: 'TW'), 'x'),
      (const Locale('en'), 'y'),
      (const Locale('en'), 'z'),
    ]) {
      ticks(8);
      track('fetch', spell.fetchSpellCheckSuggestions(locale, text));
      await settle();
      log.add('saved ${spell.lastSavedResults}');
    }
    SuggestionSpan s(int start, int end, [String w = 'w']) => SuggestionSpan(TextRange(start: start, end: end), <String>[w]);
    for (final (List<SuggestionSpan> a, List<SuggestionSpan> b) in <(List<SuggestionSpan>, List<SuggestionSpan>)>[
      (<SuggestionSpan>[], <SuggestionSpan>[]),
      (<SuggestionSpan>[s(0, 1, 'old')], <SuggestionSpan>[]),
      (<SuggestionSpan>[], <SuggestionSpan>[s(0, 1, 'new')]),
      (<SuggestionSpan>[s(0, 1, 'old'), s(5, 6, 'old')], <SuggestionSpan>[s(0, 2, 'new'), s(3, 4, 'new'), s(9, 9, 'new')]),
      (<SuggestionSpan>[s(4, 5, 'old')], <SuggestionSpan>[s(1, 2, 'new'), s(4, 9, 'new'), s(7, 8, 'new')]),
    ]) {
      log.add('merge ${DefaultSpellCheckService.mergeResults(a, b)}');
    }
    log.add('equal ${s(1, 2) == s(1, 2)} ${s(1, 2) == s(1, 3)} ${s(1, 2, 'a') == s(1, 2, 'b')}');
    log.add(
      'results ${SpellCheckResults('t', <SuggestionSpan>[s(1, 2)]) == SpellCheckResults('t', <SuggestionSpan>[s(1, 2)])} '
      '${SpellCheckResults('t', <SuggestionSpan>[s(1, 2)]) == SpellCheckResults('u', <SuggestionSpan>[s(1, 2)])}',
    );
    log.add('${SpellCheckResults('multi\nline', <SuggestionSpan>[s(0, 1), s(2, 3, 'two words')])}');
  });

  tearDownAll(() {
    File('build/undo_scribe_misc.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': log}));
  });
}
