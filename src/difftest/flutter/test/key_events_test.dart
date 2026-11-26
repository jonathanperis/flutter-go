// Reference side of flutter-go's key event differential test: key event
// conversion and synthesis through KeyEventManager, per platform and in
// both transit modes (raw key messages only, and key data then raw key
// messages), as flutter_test's simulateKeyDownEvent & co. deliver them;
// plus hand-made raw messages whose modifier flags disagree with the
// pressed keys (synthesized events), and key data sequences (synthesized
// key data, empty key data). Each step logs what the HardwareKeyboard
// handler, the RawKeyboard listener and the keyMessageHandler receive, the
// step's result, and HardwareKeyboard's and RawKeyboard's state.
// Output: build/key_events.json as {"lines": [...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> lines = <String>[];

String strip(String s) => s.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#');

String hex(int v) => '0x${v.toRadixString(16)}';

void logState() {
  final HardwareKeyboard hk = HardwareKeyboard.instance;
  lines.add(
    '  state hw=[${hk.physicalKeysPressed.map((PhysicalKeyboardKey k) => hex(k.usbHidUsage)).join(',')}]'
    ' [${hk.logicalKeysPressed.map((LogicalKeyboardKey k) => hex(k.keyId)).join(',')}]'
    ' locks=[${hk.lockModesEnabled.map((KeyboardLockMode m) => m.name).join(',')}]'
    ' raw=[${RawKeyboard.instance.physicalKeysPressed.map((PhysicalKeyboardKey k) => hex(k.usbHidUsage)).join(',')}]'
    ' [${RawKeyboard.instance.keysPressed.map((LogicalKeyboardKey k) => hex(k.keyId)).join(',')}]'
    ' ctrl=${hk.isControlPressed} shift=${hk.isShiftPressed} alt=${hk.isAltPressed} meta=${hk.isMetaPressed}',
  );
}

void rawListener(RawKeyEvent event) {
  lines.add('  raw ${event.runtimeType} ${strip(event.data.toString())} char=${jsonEncode(event.character)} repeat=${event.repeat}');
}

void install() {
  HardwareKeyboard.instance.addHandler((KeyEvent event) {
    lines.add('  handler ${strip(event.toString())}');
    return event.logicalKey == LogicalKeyboardKey.keyB;
  });
  RawKeyboard.instance.addListener(rawListener);
  ServicesBinding.instance.keyEventManager.keyMessageHandler = (KeyMessage message) {
    lines.add('  message ${strip(message.toString())} raw=${message.rawEvent?.runtimeType}');
    return message.events.any((KeyEvent e) => e.logicalKey == LogicalKeyboardKey.keyC);
  };
}

Future<void> step(String desc, Future<bool> Function() f) async {
  lines.add('step $desc');
  final bool result = await f();
  lines.add('  -> $result');
  logState();
}

void reset() {
  RawKeyboard.instance.removeListener(rawListener);
  RawKeyboard.instance.clearKeysPressed();
  HardwareKeyboard.instance.clearState();
  ServicesBinding.instance.keyEventManager.clearState();
}

const List<String> platforms = <String>['android', 'fuchsia', 'macos', 'ios', 'linux', 'windows', 'web'];

// The modifier flags meaning "left shift is down" in each platform's raw
// message.
Map<String, Object> shiftFlags(String platform) {
  return switch (platform) {
    'android' => <String, Object>{
      'metaState': RawKeyEventDataAndroid.modifierLeftShift | RawKeyEventDataAndroid.modifierShift,
    },
    'fuchsia' => <String, Object>{'modifiers': RawKeyEventDataFuchsia.modifierLeftShift},
    'macos' => <String, Object>{
      'modifiers': RawKeyEventDataMacOs.modifierLeftShift | RawKeyEventDataMacOs.modifierShift,
    },
    'ios' => <String, Object>{'modifiers': RawKeyEventDataIos.modifierLeftShift | RawKeyEventDataIos.modifierShift},
    'linux' => <String, Object>{'modifiers': GLFWKeyHelper.modifierShift},
    'windows' => <String, Object>{
      'modifiers': RawKeyEventDataWindows.modifierLeftShift | RawKeyEventDataWindows.modifierShift,
    },
    _ => <String, Object>{'metaState': RawKeyEventDataWeb.modifierShift},
  };
}

Future<void> sendRaw(Map<String, dynamic> data) async {
  await TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.handlePlatformMessage(
    SystemChannels.keyEvent.name,
    SystemChannels.keyEvent.codec.encodeMessage(data),
    (ByteData? data) {},
  );
}

void main() {
  for (final KeyDataTransitMode mode in KeyDataTransitMode.values) {
    for (final String platform in platforms) {
      testWidgets('$platform ${mode.name}', (WidgetTester tester) async {
        debugKeyEventSimulatorTransitModeOverride = mode;
        addTearDown(() => debugKeyEventSimulatorTransitModeOverride = null);
        lines.add('scene $platform ${mode.name}');
        install();
        Future<void> down(LogicalKeyboardKey k, {PhysicalKeyboardKey? physical, String? character}) => step(
          'down ${k.keyLabel} ${physical?.usbHidUsage} $character',
          () => simulateKeyDownEvent(k, platform: platform, physicalKey: physical, character: character),
        );
        Future<void> up(LogicalKeyboardKey k, {PhysicalKeyboardKey? physical}) => step(
          'up ${k.keyLabel} ${physical?.usbHidUsage}',
          () => simulateKeyUpEvent(k, platform: platform, physicalKey: physical),
        );
        Future<void> repeat(LogicalKeyboardKey k) =>
            step('repeat ${k.keyLabel}', () => simulateKeyRepeatEvent(k, platform: platform));

        await down(LogicalKeyboardKey.shiftLeft);
        await down(LogicalKeyboardKey.keyA);
        if (mode == KeyDataTransitMode.keyDataThenRawKeyData) {
          await repeat(LogicalKeyboardKey.keyA);
        }
        await up(LogicalKeyboardKey.keyA);
        await up(LogicalKeyboardKey.shiftLeft);
        await down(LogicalKeyboardKey.keyB);
        await up(LogicalKeyboardKey.keyB);
        await down(LogicalKeyboardKey.keyC, character: 'x');
        await up(LogicalKeyboardKey.keyC);
        await down(LogicalKeyboardKey.capsLock);
        await up(LogicalKeyboardKey.capsLock);
        await down(LogicalKeyboardKey.capsLock);
        await up(LogicalKeyboardKey.capsLock);
        if (platform != 'macos' && platform != 'ios' && platform != 'fuchsia') {
          await down(LogicalKeyboardKey.numLock);
          await up(LogicalKeyboardKey.numLock);
        }
        await down(LogicalKeyboardKey.controlLeft);
        await down(LogicalKeyboardKey.altLeft);
        await down(LogicalKeyboardKey.metaLeft);
        await down(LogicalKeyboardKey.digit1);
        await up(LogicalKeyboardKey.digit1);
        await up(LogicalKeyboardKey.metaLeft);
        await up(LogicalKeyboardKey.altLeft);
        await up(LogicalKeyboardKey.controlLeft);
        await down(LogicalKeyboardKey.shiftRight);
        await down(LogicalKeyboardKey.arrowLeft);
        await up(LogicalKeyboardKey.arrowLeft);
        await up(LogicalKeyboardKey.shiftRight);
        if (platform != 'linux' && platform != 'windows' && platform != 'ios' && platform != 'web') {
          await down(LogicalKeyboardKey.fn);
          await down(LogicalKeyboardKey.f12);
          await up(LogicalKeyboardKey.fn);
          await up(LogicalKeyboardKey.f12);
        }
        await down(LogicalKeyboardKey.enter);
        await up(LogicalKeyboardKey.enter);
        await down(LogicalKeyboardKey.numpad1);
        await up(LogicalKeyboardKey.numpad1);
        if (mode == KeyDataTransitMode.keyDataThenRawKeyData) {
          await down(LogicalKeyboardKey.keyB, physical: PhysicalKeyboardKey.keyA);
          await up(LogicalKeyboardKey.keyB, physical: PhysicalKeyboardKey.keyA);
        }

        // Raw messages whose modifier flags disagree with the pressed keys.
        if (mode == KeyDataTransitMode.rawKeyData) {
          reset();
          install();
          await step('raw keyA with shift flags', () async {
            final Map<String, dynamic> data = KeyEventSimulator.getKeyData(LogicalKeyboardKey.keyA, platform: platform)
              ..addAll(shiftFlags(platform));
            await sendRaw(data);
            return false;
          });
          await step('raw keyA up without flags', () async {
            await sendRaw(KeyEventSimulator.getKeyData(LogicalKeyboardKey.keyA, platform: platform, isDown: false));
            return false;
          });
          await step('raw shiftLeft down without flags', () async {
            await sendRaw(
              KeyEventSimulator.getKeyData(LogicalKeyboardKey.keyA, platform: platform),
            );
            final Map<String, dynamic> data = KeyEventSimulator.getKeyData(
              LogicalKeyboardKey.shiftLeft,
              platform: platform,
            );
            data['metaState'] = 0;
            data['modifiers'] = 0;
            await sendRaw(data);
            return false;
          });
          await step('raw keyA up', () async {
            await sendRaw(KeyEventSimulator.getKeyData(LogicalKeyboardKey.keyA, platform: platform, isDown: false));
            return false;
          });
        }
        reset();
      });
    }
  }

  testWidgets('key data sequences', (WidgetTester tester) async {
    lines.add('scene key data');
    install();
    final KeyEventManager manager = ServicesBinding.instance.keyEventManager;
    ui.KeyData data(ui.KeyEventType type, int physical, int logical, {String? character, bool synthesized = false, int time = 0}) =>
        ui.KeyData(
          type: type,
          physical: physical,
          logical: logical,
          timeStamp: Duration(microseconds: time),
          character: character,
          synthesized: synthesized,
        );
    Map<String, dynamic> macRaw(String type, int keyCode, int modifiers, [String chars = '']) => <String, dynamic>{
      'type': type,
      'keymap': 'macos',
      'keyCode': keyCode,
      'characters': chars,
      'charactersIgnoringModifiers': chars,
      'modifiers': modifiers,
    };
    // Empty key data first: no event, but the mode is inferred.
    await step('empty key data', () async {
      lines.add('  keydata ${manager.handleKeyData(data(ui.KeyEventType.down, 0, 0, character: 'a'))}');
      final Map<String, dynamic> r = await manager.handleRawKeyMessage(macRaw('keydown', 0x00, 0, 'a'));
      lines.add('  reply $r');
      return false;
    });
    // A solitary synthesized event is dispatched at once.
    await step('synthesized solitary', () async {
      lines.add(
        '  keydata ${manager.handleKeyData(data(ui.KeyEventType.up, PhysicalKeyboardKey.keyA.usbHidUsage, LogicalKeyboardKey.keyA.keyId, synthesized: true, time: 5))}',
      );
      return false;
    });
    // CapsLock tap on macOS: a down event and a synthesized up event, then
    // the raw message.
    await step('caps lock tap', () async {
      lines.add(
        '  keydata ${manager.handleKeyData(data(ui.KeyEventType.down, PhysicalKeyboardKey.capsLock.usbHidUsage, LogicalKeyboardKey.capsLock.keyId, time: 10))}',
      );
      lines.add(
        '  keydata ${manager.handleKeyData(data(ui.KeyEventType.up, PhysicalKeyboardKey.capsLock.usbHidUsage, LogicalKeyboardKey.capsLock.keyId, synthesized: true, time: 10))}',
      );
      final Map<String, dynamic> r = await manager.handleRawKeyMessage(macRaw('keydown', 0x39, 0x10000));
      lines.add('  reply $r');
      return false;
    });
    // Several key data before one raw message, characters and repeats.
    await step('batch', () async {
      lines.add(
        '  keydata ${manager.handleKeyData(data(ui.KeyEventType.down, PhysicalKeyboardKey.shiftLeft.usbHidUsage, LogicalKeyboardKey.shiftLeft.keyId, synthesized: true, time: 20))}',
      );
      lines.add(
        '  keydata ${manager.handleKeyData(data(ui.KeyEventType.down, PhysicalKeyboardKey.keyQ.usbHidUsage, LogicalKeyboardKey.keyQ.keyId, character: 'Q', time: 21))}',
      );
      lines.add(
        '  keydata ${manager.handleKeyData(data(ui.KeyEventType.repeat, PhysicalKeyboardKey.keyQ.usbHidUsage, 0x51, character: 'Q', time: 22))}',
      );
      lines.add(
        '  keydata ${manager.handleKeyData(data(ui.KeyEventType.down, 0x123456, 0x1400000099, time: 23))}',
      );
      final Map<String, dynamic> r = await manager.handleRawKeyMessage(
        macRaw('keydown', 0x0c, RawKeyEventDataMacOs.modifierLeftShift | RawKeyEventDataMacOs.modifierShift, 'Q'),
      );
      lines.add('  reply $r');
      return false;
    });
    await step('batch up', () async {
      for (final List<int> k in <List<int>>[
        <int>[PhysicalKeyboardKey.keyQ.usbHidUsage, 0x51],
        <int>[0x123456, 0x1400000099],
        <int>[PhysicalKeyboardKey.shiftLeft.usbHidUsage, LogicalKeyboardKey.shiftLeft.keyId],
      ]) {
        lines.add('  keydata ${manager.handleKeyData(data(ui.KeyEventType.up, k[0], k[1], time: 30))}');
      }
      final Map<String, dynamic> r = await manager.handleRawKeyMessage(macRaw('keyup', 0x0c, 0));
      lines.add('  reply $r');
      return false;
    });
    // Fn on macOS: the raw event is skipped, along with its key up.
    await step('fn skipped', () async {
      lines.add(
        '  keydata ${manager.handleKeyData(data(ui.KeyEventType.down, PhysicalKeyboardKey.fn.usbHidUsage, LogicalKeyboardKey.fn.keyId, time: 40))}',
      );
      lines.add('  reply ${await manager.handleRawKeyMessage(macRaw('keydown', 0x3f, 0x800000))}');
      lines.add(
        '  keydata ${manager.handleKeyData(data(ui.KeyEventType.up, PhysicalKeyboardKey.fn.usbHidUsage, LogicalKeyboardKey.fn.keyId, time: 41))}',
      );
      lines.add('  reply ${await manager.handleRawKeyMessage(macRaw('keyup', 0x3f, 0))}');
      return false;
    });
    reset();
  });

  testWidgets('getKeyData', (WidgetTester tester) async {
    for (final String platform in platforms) {
      for (final LogicalKeyboardKey key in <LogicalKeyboardKey>[
        LogicalKeyboardKey.keyA,
        LogicalKeyboardKey.shift,
        LogicalKeyboardKey.controlRight,
        LogicalKeyboardKey.enter,
        LogicalKeyboardKey.numpad5,
        LogicalKeyboardKey.backquote,
        LogicalKeyboardKey.arrowUp,
        LogicalKeyboardKey.escape,
      ]) {
        lines.add('getKeyData $platform ${key.keyLabel} ${jsonEncode(KeyEventSimulator.getKeyData(key, platform: platform))}');
        lines.add(
          'getKeyData $platform ${key.keyLabel} up ${jsonEncode(KeyEventSimulator.getKeyData(key, platform: platform, isDown: false))}',
        );
      }
    }
  });

  tearDownAll(() {
    File('build/key_events.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'lines': lines}));
  });
}
