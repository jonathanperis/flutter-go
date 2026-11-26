// Reference side of flutter-go's editing actions difftest: more editing
// scenes than editing_test.dart (document, paragraph and line boundaries,
// clipboard, transposition, expansion, deletes, punctuation, graphemes,
// keys left to the platform) on every platform's default text editing
// shortcuts. Scenes are read from ../editing_actions_scenes.json (shared
// with Go); a scene's prefix (mac_, ios_, linux_, windows_) selects the
// platform, android otherwise. Typing is delivered the way a platform
// keyboard does (TextInput's updateEditingValue), and so are selection
// changes (select: [base, extent], affinity) and whole new values
// (reset: text, caret at the end); every other key is
// simulated (on macOS the keys the framework leaves to the platform come
// back as selectors from flutter_test's macOS key handler). The clipboard is
// mocked on flutter/platform. After each step we record [text, baseOffset,
// extentOffset, affinity] (UTF-16).
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

const Map<String, LogicalKeyboardKey> keys = <String, LogicalKeyboardKey>{
  'backspace': LogicalKeyboardKey.backspace,
  'delete': LogicalKeyboardKey.delete,
  'arrowLeft': LogicalKeyboardKey.arrowLeft,
  'arrowRight': LogicalKeyboardKey.arrowRight,
  'arrowUp': LogicalKeyboardKey.arrowUp,
  'arrowDown': LogicalKeyboardKey.arrowDown,
  'home': LogicalKeyboardKey.home,
  'end': LogicalKeyboardKey.end,
  'pageUp': LogicalKeyboardKey.pageUp,
  'pageDown': LogicalKeyboardKey.pageDown,
  'insert': LogicalKeyboardKey.insert,
  'space': LogicalKeyboardKey.space,
  'enter': LogicalKeyboardKey.enter,
  'tab': LogicalKeyboardKey.tab,
  'escape': LogicalKeyboardKey.escape,
  'keyA': LogicalKeyboardKey.keyA,
  'keyB': LogicalKeyboardKey.keyB,
  'keyC': LogicalKeyboardKey.keyC,
  'keyE': LogicalKeyboardKey.keyE,
  'keyF': LogicalKeyboardKey.keyF,
  'keyN': LogicalKeyboardKey.keyN,
  'keyP': LogicalKeyboardKey.keyP,
  'keyT': LogicalKeyboardKey.keyT,
  'keyV': LogicalKeyboardKey.keyV,
  'keyX': LogicalKeyboardKey.keyX,
};

const Map<String, LogicalKeyboardKey> mods = <String, LogicalKeyboardKey>{
  'shift': LogicalKeyboardKey.shiftLeft,
  'control': LogicalKeyboardKey.controlLeft,
  'alt': LogicalKeyboardKey.altLeft,
  'meta': LogicalKeyboardKey.metaLeft,
};

TargetPlatform? platformFor(String scene) {
  if (scene.startsWith('mac_')) {
    return TargetPlatform.macOS;
  }
  if (scene.startsWith('ios_')) {
    return TargetPlatform.iOS;
  }
  if (scene.startsWith('linux_')) {
    return TargetPlatform.linux;
  }
  if (scene.startsWith('windows_')) {
    return TargetPlatform.windows;
  }
  return null;
}

void main() {
  final Map<String, dynamic> scenes =
      jsonDecode(File('../editing_actions_scenes.json').readAsStringSync()) as Map<String, dynamic>;
  final Map<String, List<List<Object>>> out = <String, List<List<Object>>>{};

  for (final MapEntry<String, dynamic> scene in scenes.entries) {
    testWidgets(scene.key, (WidgetTester tester) async {
      debugDefaultTargetPlatformOverride = platformFor(scene.key);
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      String? clipboard;
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(SystemChannels.platform, (
        MethodCall call,
      ) async {
        switch (call.method) {
          case 'Clipboard.setData':
            clipboard = (call.arguments as Map<String, dynamic>)['text'] as String;
          case 'Clipboard.getData':
            return clipboard == null ? null : <String, dynamic>{'text': clipboard};
        }
        return null;
      });
      addTearDown(
        () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(SystemChannels.platform, null),
      );
      final TextEditingController controller = TextEditingController();
      final FocusNode focus = FocusNode();
      await tester.pumpWidget(
        Directionality(
          textDirection: TextDirection.ltr,
          child: DefaultTextEditingShortcuts(
            child: EditableText(
              controller: controller,
              focusNode: focus,
              style: const TextStyle(fontSize: 14),
              cursorColor: const Color(0xff000000),
              backgroundCursorColor: const Color(0xff888888),
            ),
          ),
        ),
      );
      await tester.showKeyboard(find.byType(EditableText));
      final List<List<Object>> samples = <List<Object>>[];
      for (final dynamic op in scene.value as List<dynamic>) {
        final Map<String, dynamic> o = op as Map<String, dynamic>;
        if (o.containsKey('reset')) {
          final String text = o['reset'] as String;
          tester.testTextInput.updateEditingValue(
            TextEditingValue(text: text, selection: TextSelection.collapsed(offset: text.length)),
          );
        } else if (o.containsKey('select')) {
          final List<dynamic> sel = o['select'] as List<dynamic>;
          tester.testTextInput.updateEditingValue(
            controller.value.copyWith(
              selection: TextSelection(
                baseOffset: sel[0] as int,
                extentOffset: sel[1] as int,
                affinity: o['affinity'] == 'upstream' ? TextAffinity.upstream : TextAffinity.downstream,
              ),
            ),
          );
        } else if (o.containsKey('type')) {
          final TextEditingValue v = controller.value;
          final String text =
              v.selection.textBefore(v.text) + (o['type'] as String) + v.selection.textAfter(v.text);
          final int caret = v.selection.start + (o['type'] as String).length;
          tester.testTextInput.updateEditingValue(
            TextEditingValue(text: text, selection: TextSelection.collapsed(offset: caret)),
          );
        } else {
          final List<String> m = ((o['mods'] as List<dynamic>?) ?? <dynamic>[]).cast<String>();
          for (final String k in m) {
            await tester.sendKeyDownEvent(mods[k]!);
          }
          await tester.sendKeyEvent(keys[o['key'] as String]!);
          for (final String k in m.reversed) {
            await tester.sendKeyUpEvent(mods[k]!);
          }
        }
        await tester.pump();
        final TextEditingValue v = controller.value;
        samples.add(<Object>[
          v.text,
          v.selection.baseOffset,
          v.selection.extentOffset,
          v.selection.affinity.name,
        ]);
      }
      out[scene.key] = samples;
      focus.dispose();
      controller.dispose();
    });
  }

  tearDownAll(() {
    File('build/editing_actions.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
