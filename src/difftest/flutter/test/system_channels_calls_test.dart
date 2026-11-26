// Reference side of flutter-go's system channel calls differential test:
// the platform messages SystemChrome, SystemNavigator, SystemSound,
// HapticFeedback, BrowserContextMenu, DeferredComponent and
// SensitiveContentService send (the exact bytes: JSON text for the JSON
// channels, the standard codec's bytes otherwise), what they return, and
// when (relative to a chain of microtasks). The Go twin is
// ../../system_channels_calls_test.go.
// Output: build/system_channels_calls.json as {"log": [...]}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:ui' show AppLifecycleState;

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

String bytes(ByteData? data) => data == null
    ? 'null'
    : data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes).join(',');

TestDefaultBinaryMessenger get messenger =>
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

// Each channel logs what it receives; replies[channel] (if any) answers.
final Map<String, ByteData? Function(ByteData?)> replies = <String, ByteData? Function(ByteData?)>{};

void logChannel(String name, {required bool json}) {
  messenger.setMockMessageHandler(name, (ByteData? message) async {
    if (json) {
      log.add('$name ${utf8.decode(message!.buffer.asUint8List(message.offsetInBytes, message.lengthInBytes))}');
    } else {
      log.add('$name ${bytes(message)}');
    }
    return replies[name]?.call(message);
  });
}

Future<void> step(String name, Future<Object?> Function() f) async {
  try {
    final Object? result = await f();
    log.add('$name -> $result');
  } on AssertionError catch (e) {
    // Go's AssertionError carries the message only (not the VM's text).
    log.add('$name !! AssertionError(${e.message})');
  } catch (e) {
    log.add('$name !! ${e.runtimeType}: $e');
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('system channels calls', () async {
    logChannel('flutter/platform', json: true);
    logChannel('flutter/navigation', json: true);
    logChannel('flutter/deferredcomponent', json: false);
    logChannel('flutter/sensitivecontent', json: false);
    logChannel('flutter/contextmenu', json: true);

    log.add('scene orientations');
    await step('none', () => SystemChrome.setPreferredOrientations(<DeviceOrientation>[]));
    await step('up', () => SystemChrome.setPreferredOrientations(<DeviceOrientation>[DeviceOrientation.portraitUp]));
    await step('all', () => SystemChrome.setPreferredOrientations(DeviceOrientation.values));

    log.add('scene switcher');
    await step('both', () => SystemChrome.setApplicationSwitcherDescription(
        const ApplicationSwitcherDescription(label: 'Example "label"', primaryColor: 0xFF00FF00)));
    await step('none', () => SystemChrome.setApplicationSwitcherDescription(const ApplicationSwitcherDescription()));
    await step('label', () => SystemChrome.setApplicationSwitcherDescription(
        const ApplicationSwitcherDescription(label: 'ünïcødé ✓')));
    await step('color', () => SystemChrome.setApplicationSwitcherDescription(
        const ApplicationSwitcherDescription(primaryColor: -1)));

    log.add('scene ui mode');
    for (final SystemUiMode mode in SystemUiMode.values) {
      if (mode != SystemUiMode.manual) {
        await step('$mode', () => SystemChrome.setEnabledSystemUIMode(mode));
      }
    }
    await step('manual none', () => SystemChrome.setEnabledSystemUIMode(SystemUiMode.manual, overlays: <SystemUiOverlay>[]));
    await step('manual top', () => SystemChrome.setEnabledSystemUIMode(SystemUiMode.manual, overlays: <SystemUiOverlay>[SystemUiOverlay.top]));
    await step('manual both', () => SystemChrome.setEnabledSystemUIMode(SystemUiMode.manual,
        overlays: <SystemUiOverlay>[SystemUiOverlay.bottom, SystemUiOverlay.top]));
    await step('manual null', () => SystemChrome.setEnabledSystemUIMode(SystemUiMode.manual));
    await step('restore', SystemChrome.restoreSystemUIOverlays);

    log.add('scene ui change callback');
    await step('null callback', () => SystemChrome.setSystemUIChangeCallback(null));
    await step('callback', () => SystemChrome.setSystemUIChangeCallback((bool visible) async {
      log.add('systemUIChange $visible');
    }));
    await messenger.handlePlatformMessage('flutter/platform',
        const JSONMethodCodec().encodeMethodCall(const MethodCall('SystemChrome.systemUIChange', <Object?>[true])),
        (ByteData? reply) => log.add('systemUIChange reply ${bytes(reply)}'));
    await messenger.handlePlatformMessage('flutter/platform',
        const JSONMethodCodec().encodeMethodCall(const MethodCall('SystemChrome.systemUIChange', <Object?>[false])),
        (ByteData? reply) => log.add('systemUIChange reply ${bytes(reply)}'));
    await step('clear callback', () => SystemChrome.setSystemUIChangeCallback(null));

    log.add('scene overlay style');
    // Make sure no style is recorded as sent.
    SystemChrome.handleAppLifecycleStateChanged(AppLifecycleState.detached);
    await Future<void>.delayed(Duration.zero);
    ticks(4);
    SystemChrome.setSystemUIOverlayStyle(SystemUiOverlayStyle.dark);
    log.add('latest ${SystemChrome.latestStyle == SystemUiOverlayStyle.dark}');
    SystemChrome.setSystemUIOverlayStyle(SystemUiOverlayStyle.light);
    log.add('set twice');
    await Future<void>.delayed(Duration.zero);
    log.add('latest ${SystemChrome.latestStyle == SystemUiOverlayStyle.light}');
    ticks(3);
    SystemChrome.setSystemUIOverlayStyle(SystemUiOverlayStyle.light);
    log.add('same again');
    await Future<void>.delayed(Duration.zero);
    const custom = SystemUiOverlayStyle(
      systemNavigationBarColor: Color(0x80123456),
      systemNavigationBarDividerColor: Color(0x00000000),
      systemNavigationBarIconBrightness: Brightness.dark,
      systemNavigationBarContrastEnforced: false,
      statusBarColor: Color(0xFFFFFFFF),
      statusBarBrightness: Brightness.light,
      statusBarIconBrightness: Brightness.dark,
      systemStatusBarContrastEnforced: true,
    );
    SystemChrome.setSystemUIOverlayStyle(custom);
    SystemChrome.setSystemUIOverlayStyle(SystemUiOverlayStyle.light);
    SystemChrome.setSystemUIOverlayStyle(custom);
    await Future<void>.delayed(Duration.zero);
    SystemChrome.setSystemUIOverlayStyle(custom.copyWith(statusBarColor: const Color(0xFF000001)));
    await Future<void>.delayed(Duration.zero);
    SystemChrome.setSystemUIOverlayStyle(custom.copyWith(statusBarColor: const Color(0xFF000001)));
    await Future<void>.delayed(Duration.zero);
    log.add('detached');
    SystemChrome.handleAppLifecycleStateChanged(AppLifecycleState.detached);
    SystemChrome.setSystemUIOverlayStyle(custom.copyWith(statusBarColor: const Color(0xFF000001)));
    await Future<void>.delayed(Duration.zero);
    SystemChrome.setSystemUIOverlayStyle(custom.copyWith(statusBarColor: const Color(0xFF000001)));
    await Future<void>.delayed(Duration.zero);
    log.add('resumed');
    SystemChrome.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    SystemChrome.setSystemUIOverlayStyle(const SystemUiOverlayStyle());
    await Future<void>.delayed(Duration.zero);
    log.add('back to latest');
    SystemChrome.setSystemUIOverlayStyle(SystemUiOverlayStyle.dark);
    SystemChrome.setSystemUIOverlayStyle(const SystemUiOverlayStyle());
    await Future<void>.delayed(Duration.zero);

    log.add('scene navigator');
    await step('pop', () => SystemNavigator.pop());
    await step('pop animated', () => SystemNavigator.pop(animated: true));
    await step('pop not animated', () => SystemNavigator.pop(animated: false));
    for (final TargetPlatform platform in TargetPlatform.values) {
      debugDefaultTargetPlatformOverride = platform;
      await step('handles back $platform', () => SystemNavigator.setFrameworkHandlesBack(true));
      await step('no back $platform', () => SystemNavigator.setFrameworkHandlesBack(false));
    }
    debugDefaultTargetPlatformOverride = null;
    await step('single', SystemNavigator.selectSingleEntryHistory);
    await step('multi', SystemNavigator.selectMultiEntryHistory);
    await step('location', () => SystemNavigator.routeInformationUpdated(location: '/a/b'));
    await step('location state', () => SystemNavigator.routeInformationUpdated(
        location: 'a', state: <String, Object?>{'k': <Object?>[1, 2.5, null, 'x']}, replace: true));
    await step('uri', () => SystemNavigator.routeInformationUpdated(uri: Uri.parse('https://flutter.dev/docs?q=1#frag')));
    await step('uri path', () => SystemNavigator.routeInformationUpdated(uri: Uri.parse('/p/q?x=y'), state: 42));
    await step('both', () => SystemNavigator.routeInformationUpdated(location: 'a', uri: Uri.parse('b')));

    log.add('scene sounds');
    for (final SystemSoundType type in SystemSoundType.values) {
      await step('$type', () => SystemSound.play(type));
    }

    log.add('scene haptics');
    await step('vibrate', HapticFeedback.vibrate);
    await step('light', HapticFeedback.lightImpact);
    await step('medium', HapticFeedback.mediumImpact);
    await step('heavy', HapticFeedback.heavyImpact);
    await step('selection', HapticFeedback.selectionClick);
    await step('success', HapticFeedback.successNotification);
    await step('warning', HapticFeedback.warningNotification);
    await step('error', HapticFeedback.errorNotification);

    log.add('scene context menu');
    await step('disable', BrowserContextMenu.disableContextMenu);
    await step('enable', BrowserContextMenu.enableContextMenu);
    log.add('enabled ${BrowserContextMenu.enabled}');

    log.add('scene deferred components');
    await step('install', () => DeferredComponent.installDeferredComponent(componentName: 'comp'));
    await step('uninstall', () => DeferredComponent.uninstallDeferredComponent(componentName: 'ünï'));

    log.add('scene sensitive content');
    final service = SensitiveContentService();
    for (final ContentSensitivity sensitivity in <ContentSensitivity>[
      ContentSensitivity.autoSensitive,
      ContentSensitivity.sensitive,
      ContentSensitivity.notSensitive,
    ]) {
      await step('set $sensitivity', () => service.setContentSensitivity(sensitivity));
    }
    for (var i = 0; i < 4; i++) {
      replies['flutter/sensitivecontent'] = (ByteData? _) => const StandardMethodCodec().encodeSuccessEnvelope(i);
      await step('get $i', service.getContentSensitivity);
    }
    replies['flutter/sensitivecontent'] = (ByteData? _) => const StandardMethodCodec().encodeSuccessEnvelope(true);
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    await step('supported android', service.isSupported);
    debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
    await step('supported iOS', service.isSupported);
    debugDefaultTargetPlatformOverride = null;
    replies.remove('flutter/sensitivecontent');

    log.add('scene replies');
    replies['flutter/platform'] = (ByteData? _) =>
        const JSONMethodCodec().encodeErrorEnvelope(code: 'denied', message: 'no haptics');
    await step('haptic error', HapticFeedback.vibrate);
    await step('sound error', () => SystemSound.play(SystemSoundType.alert));
    replies['flutter/platform'] = (ByteData? _) => const JSONMethodCodec().encodeSuccessEnvelope(<String, Object?>{'ok': 1});
    await step('chrome value', SystemChrome.restoreSystemUIOverlays);
    replies.remove('flutter/platform');

    File('build/system_channels_calls.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'log': log}));
  });
}
