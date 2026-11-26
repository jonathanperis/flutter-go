// Reference side of flutter-go's mouse cursor differential test:
// MouseCursorManager's sessions (the exact bytes of every
// activateSystemCursor call, custom sessions' activate/dispose order, the
// active cursor per device), SystemMouseCursors' kinds and descriptions,
// and MouseTrackerAnnotation's diagnostics (hashes masked). The Go twin is
// ../../mouse_cursor_test.go. Output: build/mouse_cursor.json.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];

String bytes(ByteData? d) =>
    d == null ? 'null' : d.buffer.asUint8List(d.offsetInBytes, d.lengthInBytes).join(',');

String mask(String s) => s.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH');

class _LoggingSession extends MouseCursorSession {
  _LoggingSession(_LoggingCursor super.cursor, super.device);

  @override
  Future<void> activate() async {
    log.add('activate ${cursor.debugDescription} on $device');
  }

  @override
  void dispose() {
    log.add('dispose ${cursor.debugDescription} on $device');
  }
}

class _LoggingCursor extends MouseCursor {
  const _LoggingCursor(this.name);

  final String name;

  @override
  MouseCursorSession createSession(int device) => _LoggingSession(this, device);

  @override
  String get debugDescription => 'logging($name)';
}

const List<SystemMouseCursor> all = <SystemMouseCursor>[
  SystemMouseCursors.none,
  SystemMouseCursors.basic,
  SystemMouseCursors.click,
  SystemMouseCursors.forbidden,
  SystemMouseCursors.wait,
  SystemMouseCursors.progress,
  SystemMouseCursors.contextMenu,
  SystemMouseCursors.help,
  SystemMouseCursors.text,
  SystemMouseCursors.verticalText,
  SystemMouseCursors.cell,
  SystemMouseCursors.precise,
  SystemMouseCursors.move,
  SystemMouseCursors.grab,
  SystemMouseCursors.grabbing,
  SystemMouseCursors.noDrop,
  SystemMouseCursors.alias,
  SystemMouseCursors.copy,
  SystemMouseCursors.disappearing,
  SystemMouseCursors.allScroll,
  SystemMouseCursors.resizeLeftRight,
  SystemMouseCursors.resizeUpDown,
  SystemMouseCursors.resizeUpLeftDownRight,
  SystemMouseCursors.resizeUpRightDownLeft,
  SystemMouseCursors.resizeUp,
  SystemMouseCursors.resizeDown,
  SystemMouseCursors.resizeLeft,
  SystemMouseCursors.resizeRight,
  SystemMouseCursors.resizeUpLeft,
  SystemMouseCursors.resizeUpRight,
  SystemMouseCursors.resizeDownLeft,
  SystemMouseCursors.resizeDownRight,
  SystemMouseCursors.resizeColumn,
  SystemMouseCursors.resizeRow,
  SystemMouseCursors.zoomIn,
  SystemMouseCursors.zoomOut,
];

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('mouse cursor', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMessageHandler(
      'flutter/mousecursor',
      (ByteData? message) async {
        final MethodCall call = const StandardMethodCodec().decodeMethodCall(message);
        log.add('${call.method} [${bytes(message)}]');
        return const StandardMethodCodec().encodeSuccessEnvelope(null);
      },
    );

    log.add('scene manager');
    final manager = MouseCursorManager(SystemMouseCursors.basic);
    const a = _LoggingCursor('a');
    const b = _LoggingCursor('b');
    void update(String what, int device, PointerEvent? event, List<MouseCursor> cursors) {
      log.add('update $what');
      manager.handleDeviceCursorUpdate(device, event, cursors);
      log.add('active $device ${manager.debugDeviceActiveCursor(device)}');
    }

    update('defer first', 1, null, <MouseCursor>[MouseCursor.defer, SystemMouseCursors.click, SystemMouseCursors.text]);
    update('same', 1, null, <MouseCursor>[SystemMouseCursors.click]);
    update('all defer', 1, null, <MouseCursor>[MouseCursor.defer, MouseCursor.defer]);
    update('empty', 1, null, <MouseCursor>[]);
    update('uncontrolled', 2, null, <MouseCursor>[MouseCursor.uncontrolled, SystemMouseCursors.grab]);
    update('grab', 2, null, <MouseCursor>[SystemMouseCursors.grab]);
    update('custom', 2, null, <MouseCursor>[a]);
    update('custom same', 2, const PointerHoverEvent(device: 2), <MouseCursor>[a]);
    update('custom other', 2, null, <MouseCursor>[b]);
    update('back to system', 2, null, <MouseCursor>[SystemMouseCursors.zoomOut]);
    update('removed', 1, const PointerRemovedEvent(device: 1), <MouseCursor>[SystemMouseCursors.wait]);
    update('after removed', 1, null, <MouseCursor>[SystemMouseCursors.wait]);
    update('removed custom', 2, const PointerRemovedEvent(device: 2), <MouseCursor>[]);
    update('device 7', 7, const PointerAddedEvent(device: 7), <MouseCursor>[MouseCursor.defer, a, b]);
    await Future<void>.delayed(Duration.zero);

    log.add('scene uncontrolled fallback');
    final uncontrolled = MouseCursorManager(MouseCursor.uncontrolled);
    uncontrolled.handleDeviceCursorUpdate(0, null, <MouseCursor>[]);
    log.add('active ${uncontrolled.debugDeviceActiveCursor(0)}');
    uncontrolled.handleDeviceCursorUpdate(0, null, <MouseCursor>[SystemMouseCursors.help]);
    await Future<void>.delayed(Duration.zero);

    log.add('scene system cursors');
    for (final cursor in all) {
      log.add(
        '${cursor.kind} | $cursor | ${cursor.debugDescription} | '
        '${mask(cursor.toString(minLevel: DiagnosticLevel.debug))} | '
        '${mask(cursor.toString(minLevel: DiagnosticLevel.fine))} | '
        '${mask(cursor.toStringShort())} | ${cursor == SystemMouseCursors.click} | '
        '${cursor == SystemMouseCursors.text}',
      );
    }
    for (final cursor in <SystemMouseCursor>[SystemMouseCursors.basic, SystemMouseCursors.zoomIn]) {
      for (final level in <DiagnosticLevel>[DiagnosticLevel.debug, DiagnosticLevel.info]) {
        log.add(mask(cursor.toDiagnosticsNode().toStringDeep(minLevel: level)));
      }
    }
    for (final MouseCursor cursor in <MouseCursor>[MouseCursor.defer, MouseCursor.uncontrolled, a]) {
      log.add(
        '$cursor | ${cursor.debugDescription} | ${mask(cursor.toString(minLevel: DiagnosticLevel.debug))} | '
        '${mask(cursor.toStringShort())}',
      );
    }

    log.add('scene annotations');
    for (final annotation in <MouseTrackerAnnotation>[
      const MouseTrackerAnnotation(),
      MouseTrackerAnnotation(onEnter: (_) {}),
      MouseTrackerAnnotation(onExit: (_) {}),
      MouseTrackerAnnotation(onEnter: (_) {}, onExit: (_) {}, cursor: SystemMouseCursors.click),
      const MouseTrackerAnnotation(cursor: MouseCursor.uncontrolled, validForMouseTracker: false),
      const MouseTrackerAnnotation(cursor: MouseCursor.defer),
      const MouseTrackerAnnotation(cursor: a),
    ]) {
      log.add('${mask('$annotation')} | ${annotation.validForMouseTracker} | ${annotation.cursor}');
      log.add(mask(annotation.toStringShort()));
      log.add(mask(annotation.toString(minLevel: DiagnosticLevel.hidden)));
    }
  });

  tearDownAll(() {
    File('build/mouse_cursor.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': log}));
  });
}
