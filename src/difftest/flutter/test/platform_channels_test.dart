// Reference side of flutter-go's platform channel differential test: the
// order in which MethodChannel, OptionalMethodChannel, BasicMessageChannel,
// EventChannel, the default binary messenger, the channel buffers and
// flutter_test's TestDefaultBinaryMessenger deliver results, relative to a
// chain of microtasks ("tick N"), and the values and errors they deliver.
// The Go twin is ../../platform_channels_test.go.
// Output: build/platform_channels.json as {"log": [...]}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];

// ticks logs "tick i" from a chain of n microtasks, so that the position of
// other log lines shows how many microtasks they took.
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

String show(Object? v) {
  if (v is ByteData) {
    return 'bytes:${v.buffer.asUint8List(v.offsetInBytes, v.lengthInBytes).join(',')}';
  }
  if (v is PlatformException) {
    return 'PlatformException(${v.code}, ${v.message}, ${v.details}, ${v.stacktrace})';
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

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('platform channels', () async {
    const jsonMethod = JSONMethodCodec();
    const channel = MethodChannel('diff/method', jsonMethod);
    const standardChannel = MethodChannel('diff/standard');
    const optional = OptionalMethodChannel('diff/optional', jsonMethod);

    // ---- invokeMethod against mock handlers.
    log.add('scene invoke');
    messenger.setMockMethodCallHandler(channel, (MethodCall call) async {
      log.add('handler ${call.method} ${call.arguments}');
      switch (call.method) {
        case 'echo':
          return call.arguments;
        case 'fail':
          throw PlatformException(code: 'E', message: 'failed', details: <String, Object?>{'k': 1});
        case 'missing':
          throw MissingPluginException();
        case 'error':
          throw StateError('boom');
        case 'later':
          await Future<void>.delayed(Duration.zero);
          return 'late';
      }
      return null;
    });
    ticks(12);
    track('echo', channel.invokeMethod<Object?>('echo', <Object?>[1, 'a']));
    track('fail', channel.invokeMethod<Object?>('fail'));
    track('missing', channel.invokeMethod<Object?>('missing'));
    track('error', channel.invokeMethod<Object?>('error'));
    track('null', channel.invokeMethod<Object?>('other'));
    log.add('sent');
    await settle();
    track('later', channel.invokeMethod<Object?>('later'));
    await settle();

    // ---- Raw mock message handlers: null replies, sync futures.
    log.add('scene raw');
    messenger.setMockMessageHandler('diff/optional', (ByteData? message) => null);
    messenger.setMockMessageHandler('diff/standard', (ByteData? message) {
      log.add('raw handler ${show(message)}');
      return Future<ByteData?>.value(const StandardMethodCodec().encodeSuccessEnvelope(<Object?>[true]));
    });
    ticks(8);
    track('optional', optional.invokeMethod<Object?>('anything'));
    track('standard', standardChannel.invokeMethod<Object?>('m', 7));
    track('list', standardChannel.invokeListMethod<Object?>('m'));
    log.add('sent');
    await settle();
    messenger.setMockMessageHandler('diff/optional', null);
    messenger.setMockMessageHandler('diff/standard', null);
    messenger.setMockMethodCallHandler(channel, null);

    // ---- setMethodCallHandler + handlePlatformMessage.
    log.add('scene incoming');
    channel.setMethodCallHandler((MethodCall call) async {
      log.add('incoming ${call.method} ${call.arguments}');
      if (call.method == 'throw') {
        throw ArgumentError('bad');
      }
      if (call.method == 'platform') {
        throw PlatformException(code: 'P', message: 'pm');
      }
      if (call.method == 'missing') {
        throw MissingPluginException('nope');
      }
      return '${call.arguments}!';
    });
    ticks(10);
    for (final String m in <String>['hi', 'throw', 'platform', 'missing']) {
      track(
        'handle $m',
        messenger.handlePlatformMessage(
          'diff/method',
          jsonMethod.encodeMethodCall(MethodCall(m, 'x')),
          (ByteData? reply) {
            String decoded;
            try {
              decoded = reply == null ? 'null' : '${jsonMethod.decodeEnvelope(reply)}';
            } catch (e) {
              decoded = show(e);
            }
            log.add('reply $m $decoded');
          },
        ),
      );
    }
    log.add('sent');
    await settle();
    channel.setMethodCallHandler(null);
    track('no handler', messenger.handlePlatformMessage('diff/method', null, (ByteData? r) => log.add('reply none ${show(r)}')));
    await settle();

    // ---- BasicMessageChannel both ways.
    log.add('scene basic');
    const basic = BasicMessageChannel<String?>('diff/basic', StringCodec());
    messenger.setMockDecodedMessageHandler<String?>(basic, (String? message) async {
      log.add('platform got $message');
      return '$message back';
    });
    basic.setMessageHandler((String? message) async {
      log.add('framework got $message');
      return '$message ack';
    });
    ticks(8);
    track('basic send', basic.send('ping'));
    track(
      'basic receive',
      messenger.handlePlatformMessage(
        'diff/basic',
        const StringCodec().encodeMessage('pong'),
        (ByteData? r) => log.add('basic reply ${const StringCodec().decodeMessage(r)}'),
      ),
    );
    log.add('sent');
    await settle();
    messenger.setMockDecodedMessageHandler<String?>(basic, null);
    basic.setMessageHandler(null);

    // ---- Channel buffers: messages pushed before the handler exists.
    log.add('scene buffers');
    ServicesBinding.instance.channelBuffers.push('diff/buffered', utf8.encode('one').buffer.asByteData(), (ByteData? r) {
      log.add('buffered reply 1 ${show(r)}');
    });
    ServicesBinding.instance.channelBuffers.push('diff/buffered', utf8.encode('two').buffer.asByteData(), (ByteData? r) {
      log.add('buffered reply 2 ${show(r)}');
    });
    ticks(10);
    messenger.setMessageHandler('diff/buffered', (ByteData? message) async {
      log.add('buffered got ${utf8.decode(message!.buffer.asUint8List())}');
      return message;
    });
    log.add('handler set');
    await settle();
    ServicesBinding.instance.channelBuffers.resize('diff/buffered2', 2);
    for (final String s in <String>['a', 'b', 'c']) {
      ServicesBinding.instance.channelBuffers.push('diff/buffered2', utf8.encode(s).buffer.asByteData(), (ByteData? r) {
        log.add('buffered2 reply $s ${show(r)}');
      });
    }
    messenger.setMessageHandler('diff/buffered2', (ByteData? message) async {
      final String s = utf8.decode(message!.buffer.asUint8List());
      log.add('buffered2 got $s');
      if (s == 'c') {
        throw StateError('handler failed');
      }
      return null;
    });
    final FlutterExceptionHandler? previous = FlutterError.onError;
    FlutterError.onError = (FlutterErrorDetails d) {
      log.add('reported ${d.exception} | ${d.library} | ${d.context}');
    };
    await settle();
    FlutterError.onError = previous;
    messenger.setMessageHandler('diff/buffered', null);
    messenger.setMessageHandler('diff/buffered2', null);

    // ---- EventChannel.
    log.add('scene events');
    const events = EventChannel('diff/events', jsonMethod);
    messenger.setMockStreamHandler(
      events,
      MockStreamHandler.inline(
        onListen: (Object? arguments, MockStreamHandlerEventSink sink) {
          log.add('listen $arguments');
          sink.success('$arguments 1');
          sink.error(code: 'E', message: 'm', details: 3);
          sink.success('$arguments 2');
          sink.endOfStream();
        },
        onCancel: (Object? arguments) => log.add('cancel $arguments'),
      ),
    );
    ticks(16);
    final Stream<dynamic> stream = events.receiveBroadcastStream('args');
    stream.listen(
      (dynamic e) => log.add('event $e'),
      onError: (Object e) => log.add('event error ${show(e)}'),
      onDone: () => log.add('event done'),
    );
    log.add('listening');
    await settle();
    messenger.setMockStreamHandler(events, null);

    // An event channel whose platform side is missing: the listen failure
    // is reported.
    log.add('scene events missing');
    messenger.setMockMessageHandler('diff/events2', (ByteData? m) async => null);
    FlutterError.onError = (FlutterErrorDetails d) {
      log.add('reported ${d.exception} | ${d.library} | ${d.context}');
    };
    const events2 = EventChannel('diff/events2');
    final StreamSubscription<dynamic> sub = events2.receiveBroadcastStream().listen(
      (dynamic e) => log.add('event2 $e'),
    );
    await settle();
    await sub.cancel();
    await settle();
    FlutterError.onError = previous;
    messenger.setMockMessageHandler('diff/events2', null);

    // ---- Pending messages.
    log.add('scene pending');
    final Completer<ByteData?> completer = Completer<ByteData?>();
    messenger.setMockMessageHandler('diff/pending', (ByteData? m) => completer.future);
    track('pending send', messenger.send('diff/pending', null)!);
    log.add('pending ${messenger.pendingMessageCount}');
    track('finished', messenger.platformMessagesFinished);
    completer.complete(null);
    log.add('pending after complete ${messenger.pendingMessageCount}');
    await settle();
    log.add('pending at end ${messenger.pendingMessageCount}');
    messenger.setMockMessageHandler('diff/pending', null);

    // ---- A null mock reply (await null), before other microtasks.
    log.add('scene await null');
    messenger.setMockMessageHandler('diff/optional', (ByteData? message) => null);
    track('optional first', optional.invokeMethod<Object?>('x'));
    ticks(4);
    await settle();
    ticks(4);
    track('optional second', optional.invokeMethod<Object?>('x'));
    await settle();
    messenger.setMockMessageHandler('diff/optional', null);

    // ---- An event stream closed by the platform: when the close's reply
    // completes, relative to the done event.
    log.add('scene events close');
    messenger.setMockMessageHandler('diff/events3', (ByteData? m) async => jsonMethod.encodeSuccessEnvelope(null));
    const events3 = EventChannel('diff/events3', jsonMethod);
    events3.receiveBroadcastStream().listen((dynamic e) => log.add('event3 $e'), onDone: () => log.add('event3 done'));
    await settle();
    ticks(8);
    track('close', messenger.handlePlatformMessage('diff/events3', null, null));
    await settle();
    messenger.setMockMessageHandler('diff/events3', null);

    // ---- Profiling.
    log.add('scene profiling');
    final DebugPrintCallback previousPrint = debugPrint;
    debugPrint = (String? message, {int? wrapWidth}) => log.add('debugPrint $message');
    debugProfilePlatformChannels = true;
    const profiled = MethodChannel('diff/profiled');
    messenger.setMockMethodCallHandler(profiled, (MethodCall call) async => <Object?>[call.arguments]);
    await profiled.invokeMethod<Object?>('m', 1);
    await profiled.invokeMethod<Object?>('m', 'two');
    await profiled.invokeMethod<Object?>('n');
    await Future<void>.delayed(const Duration(milliseconds: 1100));
    debugProfilePlatformChannels = false;
    debugPrint = previousPrint;
    messenger.setMockMethodCallHandler(profiled, null);

    File('build/platform_channels.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'log': log}));
  });
}
