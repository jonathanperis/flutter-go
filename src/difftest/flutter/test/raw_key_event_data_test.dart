// Reference side of flutter-go's raw key event data differential test:
// every platform's RawKeyEventData (Android, Fuchsia, macOS, iOS, Linux
// GLFW/GTK, Windows, Web) built from deterministic pseudo-random fields
// (an LCG both sides share, picking from the generated key maps in their
// order), with every getter: keyLabel, physicalKey, logicalKey,
// isModifierPressed for each ModifierKey and KeyboardSide, getModifierSide,
// modifiersPressed, the is*Pressed getters, shouldDispatchEvent, toString,
// == with the previous record, and RawKeyEvent.fromMessage of the same
// fields. Also: the known keys' toString/keyLabel/debugName, String
// toUpperCase/toLowerCase over 0..0x1FFFF, and String.hashCode.
// Output: build/raw_key_event_data.json as {"lines": [...]}.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> lines = <String>[];

int _seed = 12345;
int rnd() {
  _seed = (_seed * 1103515245 + 12345) % 2147483648;
  return _seed;
}

T pick<T>(List<T> list) => list[rnd() % list.length];

// Object identity hashes differ between runs: drop them.
String strip(String s) => s.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#');

String guard(Object? Function() f) {
  try {
    final Object? v = f();
    if (v is String) {
      return jsonEncode(v);
    }
    return '$v';
  } catch (e) {
    return 'ERR';
  }
}

const List<int> codePoints = <int>[
  0, 0x61, 0x41, 0x31, 0x7a, 0x60, 0x20, 0x7f, 0x1b, 0x0d, 0x09, 0xe9, 0xc9, 0xdf, 0xff, 0xb5,
  0x3a3, 0x3c3, 0x80000301, 0x301, 0xf700, 0xf728, 0x1f600, 0x2028, 0xd800, 0x10ffff, 0x110000,
];

const List<String> strings = <String>[
  '', 'a', 'A', 'z', '1', ' ', '\t', '\u001b', 'é', 'É', 'ß', 'ÿ', 'ab', '😀', 'e\u0301', '', '',
  'Ω', 'ω', ' ', '=', '/',
];

const List<int> androidSources = <int>[0, 0x101, 0x201, 0x401, 0x01000010, 0x02000001];

void describe(String tag, RawKeyEventData data, RawKeyEventData? previous, Map<String, Object?> message) {
  final List<String> parts = <String>[tag];
  parts.add('label=${guard(() => data.keyLabel)}');
  parts.add('physical=${guard(() => '0x${data.physicalKey.usbHidUsage.toRadixString(16)}')}');
  parts.add('logical=${guard(() => '0x${data.logicalKey.keyId.toRadixString(16)}')}');
  for (final ModifierKey key in ModifierKey.values) {
    final String pressed = KeyboardSide.values
        .map((KeyboardSide side) => guard(() => data.isModifierPressed(key, side: side) ? 1 : 0))
        .join(',');
    parts.add('${key.name}=[$pressed] side=${guard(() => data.getModifierSide(key)?.name)}');
  }
  parts.add(
    'modifiersPressed=${guard(() {
      final Map<ModifierKey, KeyboardSide> m = data.modifiersPressed;
      return ModifierKey.values.where(m.containsKey).map((ModifierKey k) => '${k.name}:${m[k]!.name}').join(' ');
    })}',
  );
  parts.add(
    'cs am=${guard(() => data.isControlPressed)}${guard(() => data.isShiftPressed)}'
    '${guard(() => data.isAltPressed)}${guard(() => data.isMetaPressed)}',
  );
  parts.add('dispatch=${guard(() => data.shouldDispatchEvent())}');
  parts.add('str=${guard(() => strip(data.toString()))}');
  if (previous != null) {
    parts.add('eqPrev=${data == previous} eqSelf=${data == data}');
  }
  parts.add(
    'msg=${guard(() {
      final RawKeyEvent event = RawKeyEvent.fromMessage(message);
      return '${event.runtimeType} ${event.character == null ? 'null' : jsonEncode(event.character)} '
          '${event.repeat} ${strip(event.toString())}';
    })}',
  );
  lines.add(parts.join(' | '));
}

void main() {
  test('raw key event data', () {
    const int n = 300;
    // Android
    RawKeyEventData? previous;
    for (int i = 0; i < n; i++) {
      final int keyCode = rnd() % 3 == 0 ? rnd() % 400 : pick(kAndroidToLogicalKey.keys.toList());
      final int scanCode = rnd() % 4 == 0 ? rnd() % 800 : pick(kAndroidToPhysicalKey.keys.toList());
      final int plain = pick(codePoints);
      final int code = pick(codePoints);
      final int meta = rnd() % 0x800000;
      final int source = pick(androidSources);
      final int flags = rnd() % 8;
      final RawKeyEventDataAndroid data = RawKeyEventDataAndroid(
        flags: flags,
        codePoint: code,
        plainCodePoint: plain,
        keyCode: keyCode,
        scanCode: scanCode,
        metaState: meta,
        eventSource: source,
      );
      describe('android $i', data, previous, <String, Object?>{
        'type': i.isEven ? 'keydown' : 'keyup',
        'keymap': 'android',
        'flags': flags,
        'codePoint': code,
        'plainCodePoint': plain,
        'keyCode': keyCode,
        'scanCode': scanCode,
        'metaState': meta,
        'source': source,
        if (i % 3 == 0) 'character': plain == 0 ? null : 'c',
      });
      previous = data;
    }
    // Fuchsia
    previous = null;
    for (int i = 0; i < n; i++) {
      final int hid = rnd() % 4 == 0 ? rnd() % 0x80000 : pick(kFuchsiaToPhysicalKey.keys.toList());
      final int code = pick(codePoints);
      final int modifiers = rnd() % 0x200;
      final RawKeyEventDataFuchsia data = RawKeyEventDataFuchsia(hidUsage: hid, codePoint: code, modifiers: modifiers);
      describe('fuchsia $i', data, previous, <String, Object?>{
        'type': i.isEven ? 'keydown' : 'keyup',
        'keymap': 'fuchsia',
        'hidUsage': hid,
        'codePoint': code,
        'modifiers': modifiers,
      });
      previous = data;
    }
    // macOS and iOS
    for (final String platform in <String>['macos', 'ios']) {
      previous = null;
      final List<int> keyCodes = platform == 'macos'
          ? <int>[
              ...kMacOsToPhysicalKey.keys,
              ...kMacOsToLogicalKey.keys,
              ...kMacOsNumPadMap.keys,
              ...kMacOsFunctionKeyMap.keys,
            ]
          : <int>[...kIosToPhysicalKey.keys, ...kIosToLogicalKey.keys, ...kIosNumPadMap.keys];
      final List<String> labels = <String>[
        ...strings,
        if (platform == 'ios') ...kIosSpecialLogicalMap.keys,
      ];
      for (int i = 0; i < n; i++) {
        final int keyCode = rnd() % 5 == 0 ? rnd() % 300 : pick(keyCodes);
        final String chars = pick(labels);
        final String ignoring = pick(labels);
        final int modifiers = rnd() % 0x1000000;
        final int masked = rnd() % 2 == 0 ? modifiers & 0xffff0000 : modifiers;
        final int? specified = platform == 'macos' && rnd() % 6 == 0
            ? pick(LogicalKeyboardKey.knownLogicalKeys.toList()).keyId
            : null;
        final RawKeyEventData data = platform == 'macos'
            ? RawKeyEventDataMacOs(
                characters: chars,
                charactersIgnoringModifiers: ignoring,
                keyCode: keyCode,
                modifiers: masked,
                specifiedLogicalKey: specified,
              )
            : RawKeyEventDataIos(
                characters: chars,
                charactersIgnoringModifiers: ignoring,
                keyCode: keyCode,
                modifiers: masked,
              );
        describe('$platform $i', data, previous, <String, Object?>{
          'type': i.isEven ? 'keydown' : 'keyup',
          'keymap': platform,
          'characters': chars,
          'charactersIgnoringModifiers': ignoring,
          'keyCode': keyCode,
          'modifiers': masked,
          if (specified != null) 'specifiedLogicalKey': specified,
        });
        previous = data;
      }
    }
    // Linux (GLFW and GTK)
    for (final String toolkit in <String>['glfw', 'gtk']) {
      previous = null;
      final List<int> keyCodes = toolkit == 'glfw'
          ? <int>[...kGlfwToLogicalKey.keys, ...kGlfwNumpadMap.keys, 340, 344, 341, 342, 343, 280, 282]
          : <int>[...kGtkToLogicalKey.keys, ...kGtkNumpadMap.keys, 0xffe1, 0xffe5, 0xffe6, 0xffe9, 0xff7f];
      const List<int> gtkBits = <int>[1, 2, 4, 8, 16, 1 << 26];
      for (int i = 0; i < n; i++) {
        final int keyCode = rnd() % 5 == 0 ? rnd() % 0x10000 : pick(keyCodes);
        final int scanCode = rnd() % 4 == 0 ? rnd() % 300 : pick(kLinuxToPhysicalKey.keys.toList());
        final int usv = pick(codePoints);
        int modifiers;
        if (toolkit == 'glfw') {
          modifiers = rnd() % 0x40;
        } else {
          modifiers = 0;
          for (final int bit in gtkBits) {
            if (rnd() % 2 == 0) {
              modifiers |= bit;
            }
          }
        }
        final bool isDown = rnd() % 2 == 0;
        final int? specified = rnd() % 6 == 0 ? pick(LogicalKeyboardKey.knownLogicalKeys.toList()).keyId : null;
        final RawKeyEventDataLinux data = RawKeyEventDataLinux(
          keyHelper: KeyHelper(toolkit),
          unicodeScalarValues: usv,
          scanCode: scanCode,
          keyCode: keyCode,
          modifiers: modifiers,
          isDown: isDown,
          specifiedLogicalKey: specified,
        );
        describe('linux-$toolkit $i', data, previous, <String, Object?>{
          'type': isDown ? 'keydown' : 'keyup',
          'keymap': 'linux',
          'toolkit': toolkit,
          'unicodeScalarValues': usv,
          'keyCode': keyCode,
          'scanCode': scanCode,
          'modifiers': modifiers,
          if (specified != null) 'specifiedLogicalKey': specified,
        });
        previous = data;
      }
    }
    // Windows
    previous = null;
    final List<int> windowsKeyCodes = <int>[...kWindowsToLogicalKey.keys, ...kWindowsNumPadMap.keys, 0xe5];
    for (int i = 0; i < n; i++) {
      final int keyCode = rnd() % 5 == 0 ? rnd() % 300 : pick(windowsKeyCodes);
      final int scanCode = rnd() % 4 == 0 ? rnd() % 0x200 : pick(kWindowsToPhysicalKey.keys.toList());
      final int cp = pick(codePoints);
      final int modifiers = rnd() % 0x4000;
      final RawKeyEventDataWindows data = RawKeyEventDataWindows(
        keyCode: keyCode,
        scanCode: scanCode,
        characterCodePoint: cp,
        modifiers: modifiers,
      );
      describe('windows $i', data, previous, <String, Object?>{
        'type': i.isEven ? 'keydown' : 'keyup',
        'keymap': 'windows',
        'keyCode': keyCode,
        'scanCode': scanCode,
        'characterCodePoint': cp,
        'modifiers': modifiers,
      });
      previous = data;
    }
    // Web
    previous = null;
    final List<String> webCodes = <String>[...kWebToPhysicalKey.keys, 'UnknownCode', 'Foo', ''];
    final List<String> webKeys = <String>[
      ...kWebToLogicalKey.keys,
      ...kWebLocationMap.keys,
      ...strings,
      'Unidentified',
    ];
    for (int i = 0; i < n; i++) {
      final String code = pick(webCodes);
      final String key = pick(webKeys);
      final int location = rnd() % 4;
      final int metaState = rnd() % 0x80;
      final int keyCode = rnd() % 200;
      final RawKeyEventDataWeb data = RawKeyEventDataWeb(
        code: code,
        key: key,
        location: location,
        metaState: metaState,
        keyCode: keyCode,
      );
      describe('web $i', data, previous, <String, Object?>{
        'type': i.isEven ? 'keydown' : 'keyup',
        'keymap': 'web',
        'code': code,
        'key': key,
        'location': location,
        'metaState': metaState,
        'keyCode': keyCode,
      });
      previous = data;
    }
    // Unknown keymap and type.
    lines.add('unknown keymap ${guard(() => RawKeyEvent.fromMessage(<String, Object?>{'type': 'keydown', 'keymap': 'beos'}))}');
    lines.add(
      'unknown type ${guard(() => RawKeyEvent.fromMessage(<String, Object?>{'type': 'keypress', 'keymap': 'android'}))}',
    );

    // The known keys.
    for (final LogicalKeyboardKey key in LogicalKeyboardKey.knownLogicalKeys) {
      lines.add(
        'logical ${strip(key.toString())} ${jsonEncode(key.keyLabel)} ${key.debugName} '
        '${key.isAutogenerated} ${key.synonyms.map((LogicalKeyboardKey k) => k.keyId).join(',')}',
      );
    }
    for (final PhysicalKeyboardKey key in PhysicalKeyboardKey.knownPhysicalKeys) {
      lines.add('physical ${strip(key.toString())} ${key.debugName}');
    }
    for (final int id in <int>[0x41, 0xd9, 0xf9, 0x1f600, 0x100012345, 0x1100000042, 0x10ffff, 0xd800]) {
      final LogicalKeyboardKey key = LogicalKeyboardKey(id);
      lines.add('logical $id ${guard(() => key.keyLabel)} ${guard(() => key.debugName)} ${guard(() => strip('$key'))}');
    }
    // String case mapping, as the VM does it.
    for (int c = 0; c < 0x20000; c++) {
      final String s = String.fromCharCode(c);
      final String up = s.toUpperCase();
      final String low = s.toLowerCase();
      if (up != s || low != s) {
        lines.add('case $c ${up.codeUnits.join(',')} ${low.codeUnits.join(',')}');
      }
    }
    // String.hashCode.
    for (final String s in <String>[...webCodes, ...strings, 'hello world', 'a' * 100]) {
      lines.add('hash ${jsonEncode(s)} ${s.hashCode}');
    }
    // runeToLowerCase.
    for (final int c in codePoints.where((int c) => c <= 0x10ffff)) {
      lines.add('runeToLowerCase $c ${guard(() => runeToLowerCase(c))}');
    }
  });

  tearDownAll(() {
    File('build/raw_key_event_data.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'lines': lines}));
  });
}
