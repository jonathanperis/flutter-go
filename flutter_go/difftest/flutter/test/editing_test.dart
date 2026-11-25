// Reference side of flutter-go's text-editing differential tests. Scenes are
// read from ../editing_scenes.json (shared with Go). Typing is delivered the
// way a platform keyboard does (replace the selection, via TextInput);
// every other key goes through Flutter's default text-editing shortcuts.
// After each step we record [text, baseOffset, extentOffset] (UTF-16).
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
  'keyA': LogicalKeyboardKey.keyA,
};

const Map<String, LogicalKeyboardKey> mods = <String, LogicalKeyboardKey>{
  'shift': LogicalKeyboardKey.shiftLeft,
  'control': LogicalKeyboardKey.controlLeft,
  'alt': LogicalKeyboardKey.altLeft,
  'meta': LogicalKeyboardKey.metaLeft,
};

void main() {
  final Map<String, dynamic> scenes =
      jsonDecode(File('../editing_scenes.json').readAsStringSync()) as Map<String, dynamic>;
  final Map<String, List<List<Object>>> out = <String, List<List<Object>>>{};

  for (final MapEntry<String, dynamic> scene in scenes.entries) {
    testWidgets(scene.key, (WidgetTester tester) async {
      // Scenes named mac_* run with macOS shortcuts.
      if (scene.key.startsWith('mac_')) {
        debugDefaultTargetPlatformOverride = TargetPlatform.macOS;
      }
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      final TextEditingController controller = TextEditingController();
      final FocusNode focus = FocusNode();
      // The key bindings come from DefaultTextEditingShortcuts, which
      // WidgetsApp normally provides; without it keys do nothing.
      await tester.pumpWidget(Directionality(
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
      ));
      await tester.showKeyboard(find.byType(EditableText));
      final List<List<Object>> samples = <List<Object>>[];
      for (final dynamic op in scene.value as List<dynamic>) {
        final Map<String, dynamic> o = op as Map<String, dynamic>;
        if (o.containsKey('type')) {
          final TextEditingValue v = controller.value;
          final String text = v.selection.textBefore(v.text) + (o['type'] as String) + v.selection.textAfter(v.text);
          final int caret = v.selection.start + (o['type'] as String).length;
          tester.testTextInput.updateEditingValue(TextEditingValue(
            text: text,
            selection: TextSelection.collapsed(offset: caret),
          ));
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
        samples.add(<Object>[v.text, v.selection.baseOffset, v.selection.extentOffset]);
      }
      out[scene.key] = samples;
    });
  }

  tearDownAll(() {
    File('build/editing.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
