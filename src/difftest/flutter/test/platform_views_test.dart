// Reference side of flutter-go's platform views differential test: every
// message PlatformViewsService and its controllers send on
// flutter/platform_views and flutter/platform_views_2 (logged as the exact
// bytes of the encoded method call), the controllers' states, the Android
// motion events built from PointerEvents, the replies and errors, timed
// against a chain of microtasks ("tick N"). The Go twin is
// ../../platform_views_test.go. Output: build/platform_views.json.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/gestures.dart';
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
  if (v is Size) {
    return 'Size(${v.width}, ${v.height})';
  }
  return '$v';
}

void track(String name, Future<Object?> f) {
  f.then(
    (Object? v) => log.add('$name -> ${show(v)}'),
    onError: (Object e) => log.add('$name !! ${show(e)}'),
  );
}

Object? guard(Object? Function() f) {
  try {
    return f();
  } catch (e) {
    return 'threw ${show(e)}';
  }
}

Future<void> settle() => Future<void>.delayed(const Duration(milliseconds: 1));

TestDefaultBinaryMessenger get messenger =>
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

// The platform side: replies queued per method (default null).
final Map<String, List<Object?>> replies = <String, List<Object?>>{};

void installPlatform(String channel) {
  const codec = StandardMethodCodec();
  messenger.setMockMessageHandler(channel, (ByteData? message) async {
    final MethodCall call = codec.decodeMethodCall(message);
    log.add('$channel ${call.method} [${bytes(message)}]');
    final List<Object?>? queue = replies[call.method];
    final Object? reply = queue == null || queue.isEmpty ? null : queue.removeAt(0);
    if (reply is PlatformException) {
      return codec.encodeErrorEnvelope(code: reply.code, message: reply.message);
    }
    return codec.encodeSuccessEnvelope(reply);
  });
}

String state(AndroidViewController c) {
  final Object? texture = guard(() => c.textureId);
  return 'id ${c.viewId} created ${c.isCreated} awaiting ${c.awaitingCreation} '
      'composition ${c.requiresViewComposition} texture $texture';
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('platform views', () async {
    installPlatform('flutter/platform_views');
    installPlatform('flutter/platform_views_2');
    const params = StandardMessageCodec();

    log.add('scene registry');
    final int first = platformViewsRegistry.getNextPlatformViewId();
    log.add('next ${platformViewsRegistry.getNextPlatformViewId() - first}');

    // ---- Texture view: create, resize, offset, direction, focus, dispose.
    log.add('scene texture');
    replies['create'] = <Object?>[7];
    replies['resize'] = <Object?>[<String, Object?>{'width': 250.5, 'height': 120.0}];
    final AndroidViewController texture = PlatformViewsService.initAndroidView(
      id: 1,
      viewType: 'webview',
      layoutDirection: TextDirection.rtl,
      creationParams: <String, Object?>{'url': 'https://flutter.dev', 'zoom': 1.5, 'list': <int>[1, 2]},
      creationParamsCodec: params,
      onFocus: () => log.add('focus 1'),
    );
    final createdLog = <int>[];
    void onCreated(int id) => createdLog.add(id);
    texture.addOnPlatformViewCreatedListener(onCreated);
    texture.addOnPlatformViewCreatedListener((int id) => log.add('created callback $id'));
    log.add(state(texture));
    track('offset before create', texture.setOffset(const Offset(1, 2)));
    track('clearFocus before create', texture.clearFocus());
    track('direction before create', texture.setLayoutDirection(TextDirection.ltr));
    ticks(10);
    track('create without size', texture.create());
    log.add('sent');
    await settle();
    log.add(state(texture));
    ticks(12);
    track('create', texture.create(size: const Size(200, 100), position: const Offset(10, 20)));
    log.add(state(texture));
    await settle();
    log.add(state(texture));
    log.add('created $createdLog');
    texture.removeOnPlatformViewCreatedListener(onCreated);
    log.add('callbacks ${texture.createdCallbacks.length}');
    ticks(10);
    track('setSize', texture.setSize(const Size(250, 120)));
    track('setOffset', texture.setOffset(const Offset(3.25, -4)));
    track('setOffset same', texture.setOffset(const Offset(3.25, -4)));
    track('direction', texture.setLayoutDirection(TextDirection.rtl));
    track('direction same', texture.setLayoutDirection(TextDirection.rtl));
    track('clearFocus', texture.clearFocus());
    track('reject', texture.rejectGesture());
    track('reject id', texture.rejectGesture(gestureId: 77));
    await settle();
    // The platform reports focus.
    const standardMethod = StandardMethodCodec();
    for (final call in <MethodCall>[
      const MethodCall('viewFocused', 1),
      const MethodCall('viewFocused', 99),
      const MethodCall('unknown', 1),
    ]) {
      await messenger.handlePlatformMessage(
        'flutter/platform_views',
        standardMethod.encodeMethodCall(call),
        (ByteData? reply) => log.add('reply ${call.method} [${bytes(reply)}]'),
      );
    }
    ticks(6);
    track('dispose', texture.dispose());
    log.add(state(texture));
    await settle();
    track('dispose again', texture.dispose());
    await settle();

    // ---- Motion events through a texture view.
    log.add('scene motion');
    replies['create'] = <Object?>[3];
    final AndroidViewController touch = PlatformViewsService.initAndroidView(
      id: 2,
      viewType: 'webview',
      layoutDirection: TextDirection.ltr,
    );
    touch.pointTransformer = (Offset o) => o - const Offset(5, 7.5);
    await touch.create(size: const Size(100, 100));
    final events = <PointerEvent>[
      const PointerDownEvent(
        timeStamp: Duration(microseconds: 1500),
        pointer: 11,
        position: Offset(10, 20),
        pressure: 0.75,
        size: 0.125,
        radiusMajor: 3,
        radiusMinor: 2,
        orientation: 0.5,
        embedderId: 900,
      ),
      const PointerMoveEvent(timeStamp: Duration(milliseconds: 3), pointer: 11, position: Offset(12, 22)),
      const PointerHoverEvent(pointer: 11, position: Offset(1, 1)),
      const PointerDownEvent(
        timeStamp: Duration(milliseconds: 4),
        pointer: 12,
        kind: PointerDeviceKind.stylus,
        position: Offset(50, 60),
      ),
      const PointerMoveEvent(
        timeStamp: Duration(milliseconds: 5),
        pointer: 12,
        position: Offset(51, 61),
        platformData: 1,
      ),
      const PointerMoveEvent(
        timeStamp: Duration(milliseconds: 5),
        pointer: 11,
        position: Offset(13, 23),
        platformData: 2 | (2 << 8),
      ),
      const PointerMoveEvent(
        timeStamp: Duration(milliseconds: 5),
        pointer: 12,
        kind: PointerDeviceKind.stylus,
        position: Offset(52, 62),
        platformData: 2 | (2 << 8),
      ),
      const PointerUpEvent(timeStamp: Duration(milliseconds: 6), pointer: 11, position: Offset(13, 23)),
      const PointerDownEvent(
        timeStamp: Duration(milliseconds: 7),
        pointer: 13,
        kind: PointerDeviceKind.invertedStylus,
        position: Offset(70, 80),
      ),
      const PointerCancelEvent(timeStamp: Duration(milliseconds: 8), pointer: 12),
      const PointerUpEvent(timeStamp: Duration(milliseconds: 9), pointer: 13, kind: PointerDeviceKind.invertedStylus),
      const PointerDownEvent(
        timeStamp: Duration(milliseconds: 20),
        pointer: 14,
        kind: PointerDeviceKind.mouse,
        position: Offset(-1, -2),
      ),
      const PointerUpEvent(timeStamp: Duration(milliseconds: 21), pointer: 14, kind: PointerDeviceKind.mouse),
      const PointerDownEvent(
        timeStamp: Duration(milliseconds: 22),
        pointer: 15,
        kind: PointerDeviceKind.unknown,
      ),
      const PointerUpEvent(timeStamp: Duration(milliseconds: 23), pointer: 15, kind: PointerDeviceKind.unknown),
    ];
    for (final PointerEvent event in events) {
      ticks(4);
      track('dispatch ${event.runtimeType} ${event.pointer}', touch.dispatchPointerEvent(event));
      await settle();
    }
    // Transformed events, as the render object dispatches them.
    await touch.dispatchPointerEvent(
      const PointerDownEvent(pointer: 16, position: Offset(1, 1)).transformed(Matrix4.translationValues(9, 9, 0)),
    );
    await touch.dispatchPointerEvent(const PointerUpEvent(pointer: 16, position: Offset(2, 2)));
    await touch.dispose();
    await settle();

    // ---- Texture view falling back to hybrid composition (create => null).
    log.add('scene texture fallback');
    final AndroidViewController fallback = PlatformViewsService.initAndroidView(
      id: 3,
      viewType: 'webview',
      layoutDirection: TextDirection.ltr,
    );
    await fallback.setSize(const Size(30, 40));
    log.add(state(fallback));
    track('setSize', fallback.setSize(const Size(31, 41)));
    track('setOffset', fallback.setOffset(const Offset(1, 1)));
    await settle();
    fallback.pointTransformer = (Offset o) => o;
    await fallback.dispatchPointerEvent(const PointerDownEvent(pointer: 1, position: Offset(4, 4)));
    await fallback.dispose();
    await settle();

    // ---- Surface view: texture, then hybrid fallback.
    log.add('scene surface');
    replies['create'] = <Object?>[12];
    final AndroidViewController surface = PlatformViewsService.initSurfaceAndroidView(
      id: 4,
      viewType: 'map',
      layoutDirection: TextDirection.ltr,
      creationParams: 'params',
      creationParamsCodec: params,
    );
    await surface.create(size: const Size(10, 10));
    log.add(state(surface));
    await surface.dispose();
    final AndroidViewController surface2 = PlatformViewsService.initSurfaceAndroidView(
      id: 5,
      viewType: 'map',
      layoutDirection: TextDirection.rtl,
    );
    await surface2.create(size: const Size(10, 10), position: const Offset(1, 2));
    log.add(state(surface2));
    track('setSize', surface2.setSize(const Size(20, 20)));
    log.add('setOffset ${guard(() => surface2.setOffset(Offset.zero))}');
    track('reject', surface2.rejectGesture(gestureId: 3));
    await settle();
    await surface2.dispose();
    await settle();

    // ---- Expensive (hybrid) view.
    log.add('scene expensive');
    final AndroidViewController expensive = PlatformViewsService.initExpensiveAndroidView(
      id: 6,
      viewType: 'map',
      layoutDirection: TextDirection.rtl,
      creationParams: <Object?>[1, 'two', null],
      creationParamsCodec: params,
    );
    ticks(8);
    track('create', expensive.create(position: const Offset(4, 5)));
    await settle();
    log.add(state(expensive));
    track('setSize', expensive.setSize(const Size(1, 1)));
    log.add('setOffset ${guard(() => expensive.setOffset(Offset.zero))}');
    expensive.pointTransformer = (Offset o) => o * 2;
    await expensive.dispatchPointerEvent(const PointerDownEvent(pointer: 1, position: Offset(4, 4)));
    await expensive.dispatchPointerEvent(const PointerUpEvent(pointer: 1, position: Offset(4, 4)));
    await expensive.dispose();
    await settle();

    // ---- Hybrid (platform_views_2) view.
    log.add('scene hybrid');
    replies['isSurfaceControlEnabled'] = <Object?>[true];
    track('supported', HybridAndroidViewController.checkIfSupported());
    final AndroidViewController hybrid = PlatformViewsService.initHybridAndroidView(
      id: 7,
      viewType: 'map',
      layoutDirection: TextDirection.ltr,
    );
    await hybrid.create(size: const Size(5, 5));
    log.add(state(hybrid));
    hybrid.pointTransformer = (Offset o) => o;
    await hybrid.dispatchPointerEvent(const PointerDownEvent(pointer: 1, position: Offset(4, 4)));
    await hybrid.dispatchPointerEvent(const PointerCancelEvent(pointer: 1));
    await hybrid.rejectGesture();
    await hybrid.rejectGesture(gestureId: 8);
    await hybrid.clearFocus();
    await hybrid.setLayoutDirection(TextDirection.rtl);
    await hybrid.dispose();
    await settle();

    // ---- Create and dispose at once.
    log.add('scene create+dispose');
    replies['create'] = <Object?>[1];
    final AndroidViewController racing = PlatformViewsService.initAndroidView(
      id: 8,
      viewType: 'webview',
      layoutDirection: TextDirection.ltr,
    );
    racing.addOnPlatformViewCreatedListener((int id) => log.add('racing created $id'));
    ticks(10);
    track('create', racing.create(size: const Size(1, 1)));
    track('dispose', racing.dispose());
    log.add(state(racing));
    await settle();
    log.add(state(racing));

    final AndroidViewController never = PlatformViewsService.initAndroidView(
      id: 10,
      viewType: 'webview',
      layoutDirection: TextDirection.ltr,
    );
    track('dispose uncreated', never.dispose());
    await settle();
    log.add(state(never));

    // ---- Platform errors.
    log.add('scene errors');
    replies['create'] = <Object?>[PlatformException(code: 'error', message: 'nope')];
    final AndroidViewController failing = PlatformViewsService.initAndroidView(
      id: 9,
      viewType: 'x',
      layoutDirection: TextDirection.ltr,
    );
    ticks(10);
    track('create', failing.create(size: const Size(1, 1)));
    await settle();
    log.add(state(failing));

    // ---- UiKit and AppKit.
    log.add('scene darwin');
    for (final UiKitViewGestureBlockingPolicy policy in UiKitViewGestureBlockingPolicy.values) {
      ticks(6);
      track(
        'uikit $policy',
        PlatformViewsService.initUiKitView(
          id: 20 + policy.index,
          viewType: 'ios',
          layoutDirection: TextDirection.ltr,
          gestureBlockingPolicy: policy,
        ).then((UiKitViewController c) => c.id),
      );
      await settle();
    }
    final UiKitViewController uikit = await PlatformViewsService.initUiKitView(
      id: 30,
      viewType: 'ios',
      layoutDirection: TextDirection.ltr,
      creationParams: <String, Object?>{'a': true},
      creationParamsCodec: params,
      onFocus: () => log.add('focus 30'),
    );
    await messenger.handlePlatformMessage(
      'flutter/platform_views',
      standardMethod.encodeMethodCall(const MethodCall('viewFocused', 30)),
      (ByteData? reply) => log.add('reply [${bytes(reply)}]'),
    );
    ticks(6);
    track('accept', uikit.acceptGesture());
    track('reject', uikit.rejectGesture());
    track('direction', uikit.setLayoutDirection(TextDirection.rtl));
    track('dispose', uikit.dispose());
    await settle();
    final UiKitViewController uikitDefault = await PlatformViewsService.initUiKitView(
      id: 31,
      viewType: 'ios',
      layoutDirection: TextDirection.rtl,
    );
    log.add('uikit default ${uikitDefault.id}');
    final AppKitViewController appkit = await PlatformViewsService.initAppKitView(
      id: 40,
      viewType: 'mac',
      layoutDirection: TextDirection.ltr,
      creationParams: 42,
      creationParamsCodec: params,
      onFocus: () => log.add('focus 40'),
    );
    await messenger.handlePlatformMessage(
      'flutter/platform_views',
      standardMethod.encodeMethodCall(const MethodCall('viewFocused', 40)),
      null,
    );
    await appkit.acceptGesture();
    await appkit.dispose();
    await messenger.handlePlatformMessage(
      'flutter/platform_views',
      standardMethod.encodeMethodCall(const MethodCall('viewFocused', 40)),
      null,
    );
    await PlatformViewsService.initAppKitView(id: 41, viewType: 'mac', layoutDirection: TextDirection.rtl);
    await settle();

    // ---- toString.
    log.add('scene toString');
    log.add('${const AndroidPointerProperties(id: 3, toolType: AndroidPointerProperties.kToolTypeStylus)}');
    log.add(
      '${const AndroidPointerCoords(orientation: 0.1, pressure: 1e21, size: -0.0, toolMajor: 1 / 3, toolMinor: 2.5e-7, touchMajor: 100, touchMinor: double.infinity, x: double.nan, y: -12345.678)}',
    );
    log.add(
      '${AndroidMotionEvent(downTime: 1, eventTime: 2, action: AndroidViewController.pointerAction(3, AndroidViewController.kActionPointerUp), pointerCount: 1, pointerProperties: const <AndroidPointerProperties>[AndroidPointerProperties(id: 0, toolType: 1)], pointerCoords: const <AndroidPointerCoords>[AndroidPointerCoords(orientation: 0, pressure: 1, size: 0, toolMajor: 0, toolMinor: 0, touchMajor: 0, touchMinor: 0, x: 1.5, y: 2)], metaState: 0, buttonState: 0, xPrecision: 1, yPrecision: 1, deviceId: 0, edgeFlags: 0, source: AndroidViewController.kInputDeviceSourceTouchScreen, flags: 0, motionEventId: 5)}',
    );
    for (final int pointer in <int>[0, 1, 255, 256, 300]) {
      log.add('pointerAction $pointer ${AndroidViewController.pointerAction(pointer, 6)}');
    }
    log.add('${UiKitViewGestureBlockingPolicy.values}');
  });

  tearDownAll(() {
    File('build/platform_views.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': log}));
  });
}
