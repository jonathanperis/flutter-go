// Reference side of flutter-go's ChannelBuffers differential test
// (dart:ui's channel_buffers.dart): each scene drives a fresh
// ChannelBuffers inside package:fake_async and logs which listener sees
// which message, which response callbacks are called (and with what), and
// in what order relative to microtasks and timers. Output:
// build/channel_buffers.json as {scene: [log lines]}.
// ignore_for_file: deprecated_member_use
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:fake_async/fake_async.dart';
import 'package:flutter_test/flutter_test.dart';

typedef Scene = void Function(FakeAsync fa, void Function(String) log);

ByteData bd(String s) => ByteData.sublistView(utf8.encode(s));

String show(ByteData? d) {
  if (d == null) {
    return 'null';
  }
  if (d.lengthInBytes == 0) {
    return '<empty>';
  }
  return utf8.decode(Uint8List.sublistView(d));
}

ui.PlatformMessageResponseCallback reply(void Function(String) log, String name) =>
    (ByteData? d) => log('reply $name ${show(d)}');

ui.ChannelCallback listener(void Function(String) log, String name, {bool respond = false}) =>
    (ByteData? d, ui.PlatformMessageResponseCallback cb) {
      log('$name got ${show(d)}');
      if (respond) {
        cb(d == null ? null : bd('re:${show(d)}'));
      }
    };

String err(Object e) {
  if (e is Exception) {
    return e.toString();
  }
  if (e is StateError) {
    return 'StateError: ${e.message}';
  }
  if (e is AssertionError) {
    return 'AssertionError null-name=${e.toString().contains('U+0000 NULL')}';
  }
  return 'error (other)';
}

void tryLog(void Function(String) log, String label, void Function() f) {
  try {
    f();
    log('$label ok');
  } catch (e) {
    log('$label: ${err(e)}');
  }
}

ByteData bytes(List<int> b) => ByteData.sublistView(Uint8List.fromList(b));

List<int> str(String s) => <int>[7, utf8.encode(s).length, ...utf8.encode(s)];

class LoggingBuffers extends ui.ChannelBuffers {
  LoggingBuffers(this.log);
  final void Function(String) log;
  @override
  void resize(String name, int newSize) {
    log('resize "$name" $newSize');
    super.resize(name, newSize);
  }

  @override
  void allowOverflow(String name, bool allowed) {
    log('allowOverflow "$name" $allowed');
    super.allowOverflow(name, allowed);
  }
}

final Map<String, Scene> scenes = <String, Scene>{
  'push_default_capacity': (FakeAsync fa, void Function(String) log) {
    final b = ui.ChannelBuffers();
    b.push('a', bd('1'), reply(log, '1'));
    b.push('a', bd('2'), reply(log, '2'));
    b.push('a', bd('3'), reply(log, '3'));
    b.push('b', null, reply(log, 'b-null'));
    b.push('b', bd(''), reply(log, 'b-empty'));
    log('pushed');
    b.setListener('a', listener(log, 'A', respond: true));
    b.setListener('b', listener(log, 'B', respond: true));
    scheduleMicrotask(() => log('micro'));
    log('listeners set');
  },
  'resize_sequences': (FakeAsync fa, void Function(String) log) {
    final b = ui.ChannelBuffers();
    b.resize('a', 4);
    for (var i = 1; i <= 6; i++) {
      b.push('a', bd('$i'), reply(log, '$i'));
    }
    log('resize 2');
    b.resize('a', 2);
    log('resize 5');
    b.resize('a', 5);
    b.push('a', bd('7'), reply(log, '7'));
    log('resize 0');
    b.resize('a', 0);
    b.push('a', bd('8'), reply(log, '8'));
    log('resize 1');
    b.resize('a', 1);
    b.push('a', bd('9'), reply(log, '9'));
    b.push('a', bd('10'), reply(log, '10'));
    tryLog(log, 'resize -1', () => b.resize('a', -1));
    b.push('a', bd('11'), reply(log, '11'));
    b.resize('a', 3);
    b.push('a', bd('12'), reply(log, '12'));
    b.setListener('a', listener(log, 'A'));
  },
  'drain_interleaved': (FakeAsync fa, void Function(String) log) {
    final b = ui.ChannelBuffers();
    b.resize('a', 10);
    for (var i = 1; i <= 4; i++) {
      b.push('a', bd('$i'), reply(log, '$i'));
    }
    scheduleMicrotask(() => log('m1'));
    b.setListener('a', (ByteData? d, ui.PlatformMessageResponseCallback cb) {
      log('L1 got ${show(d)}');
      if (show(d) == '1') {
        b.push('a', bd('5'), reply(log, '5'));
        scheduleMicrotask(() => log('m-from-L1'));
      }
      if (show(d) == '2') {
        b.setListener('a', (ByteData? d, ui.PlatformMessageResponseCallback cb) {
          log('L2 got ${show(d)}');
          cb(bd('L2'));
          if (show(d) == '4') {
            b.clearListener('a');
            b.push('a', bd('6'), reply(log, '6'));
            log('cleared');
          }
        });
      }
    });
    scheduleMicrotask(() => log('m2'));
    log('sync end');
    Timer.run(() {
      log('timer: relisten');
      b.setListener('a', listener(log, 'L3', respond: true));
      b.push('a', bd('7'), reply(log, '7'));
      log('timer end');
    });
  },
  'listener_set_twice_before_drain': (FakeAsync fa, void Function(String) log) {
    final b = ui.ChannelBuffers();
    b.resize('a', 3);
    b.push('a', bd('1'), reply(log, '1'));
    b.push('a', bd('2'), reply(log, '2'));
    b.setListener('a', listener(log, 'X'));
    b.clearListener('a');
    b.setListener('a', listener(log, 'Y', respond: true));
    b.clearListener('nope');
    scheduleMicrotask(() {
      log('m');
      b.clearListener('a');
      b.push('a', bd('3'), reply(log, '3'));
      b.push('a', bd('4'), reply(log, '4'));
      b.setListener('a', listener(log, 'Z', respond: true));
      b.push('a', bd('5'), reply(log, '5'));
    });
  },
  'replace_listener_idle': (FakeAsync fa, void Function(String) log) {
    final b = ui.ChannelBuffers();
    b.setListener('a', listener(log, 'P'));
    scheduleMicrotask(() {
      log('idle');
      b.setListener('a', listener(log, 'Q', respond: true));
      b.push('a', bd('1'), reply(log, '1'));
      b.push('a', bd('2'), reply(log, '2'));
      log('pushed');
    });
  },
  'deprecated_drain': (FakeAsync fa, void Function(String) log) {
    final b = ui.ChannelBuffers();
    b.resize('a', 5);
    for (var i = 1; i <= 3; i++) {
      b.push('a', bd('$i'), reply(log, '$i'));
    }
    b
        .drain('a', (ByteData? d, ui.PlatformMessageResponseCallback cb) async {
          log('drain got ${show(d)}');
          if (show(d) == '1') {
            b.push('a', bd('4'), reply(log, '4'));
            await Future<void>.delayed(const Duration(milliseconds: 5));
            log('after delay');
          }
          cb(bd('d${show(d)}'));
        })
        .then((_) => log('drain done'));
    scheduleMicrotask(() => log('micro'));
    b.drain('empty', (ByteData? d, ui.PlatformMessageResponseCallback cb) async {
      log('never');
    }).then((_) => log('empty drain done'));
    log('sync end');
  },
  'allow_overflow': (FakeAsync fa, void Function(String) log) {
    final b = ui.ChannelBuffers();
    b.allowOverflow('a', true);
    b.push('a', bd('1'), reply(log, '1'));
    b.push('a', bd('2'), reply(log, '2'));
    b.allowOverflow('a', false);
    b.allowOverflow('b', false);
    b.push('b', bd('3'), reply(log, '3'));
    b.resize('z', 0);
    b.allowOverflow('z', true);
    b.push('z', bd('4'), reply(log, '4'));
    b.setListener('a', listener(log, 'A'));
    b.setListener('b', listener(log, 'B'));
    b.setListener('z', listener(log, 'Z'));
  },
  'handle_message': (FakeAsync fa, void Function(String) log) {
    final b = LoggingBuffers(log);
    final cases = <String, List<int>>{
      'resize ok': <int>[...str('resize'), 12, 2, ...str('chan'), 3, 2, 0, 0, 0],
      'resize extra args': <int>[...str('resize'), 12, 3, ...str('chan'), 3, 1, 1, 0, 0, 1],
      'resize big': <int>[...str('resize'), 12, 2, ...str('x'), 3, 0xff, 0xff, 0xff, 0x7f],
      'resize unicode name': <int>[...str('resize'), 12, 2, ...str('ç✓'), 3, 7, 0, 0, 0],
      'overflow true': <int>[...str('overflow'), 12, 2, ...str('chan'), 1],
      'overflow false': <int>[...str('overflow'), 12, 2, ...str('chan'), 2],
      'method too long': <int>[7, 254, 0, 0],
      'unknown method': str('flush'),
      'resize not list': <int>[...str('resize'), 13, 2],
      'resize one arg': <int>[...str('resize'), 12, 1, ...str('a')],
      'resize name not string': <int>[...str('resize'), 12, 2, 3, 0, 0, 0, 0],
      'resize name too long': <int>[...str('resize'), 12, 2, 7, 254, 0, 0],
      'resize null in name': <int>[...str('resize'), 12, 2, ...str('a\u0000b'), 3, 1, 0, 0, 0],
      'resize size not int32': <int>[...str('resize'), 12, 2, ...str('a'), 4, 1, 0, 0, 0, 0, 0, 0, 0],
      'overflow not list': <int>[...str('overflow'), 13],
      'overflow one arg': <int>[...str('overflow'), 12, 1],
      'overflow name not string': <int>[...str('overflow'), 12, 2, 0, 1],
      'overflow name too long': <int>[...str('overflow'), 12, 2, 7, 255],
      'overflow flag not bool': <int>[...str('overflow'), 12, 2, ...str('a'), 0],
      'overflow flag int': <int>[...str('overflow'), 12, 2, ...str('a'), 3],
      'overflow null in name': <int>[...str('overflow'), 12, 2, ...str('a\u0000'), 1],
      'truncated': <int>[7],
      'truncated resize': <int>[...str('resize'), 12, 2, ...str('a'), 3, 1],
    };
    for (final MapEntry<String, List<int>> c in cases.entries) {
      tryLog(log, c.key, () => b.handleMessage(bytes(c.value)));
    }
    final legacy = <String>[
      'resize\ra\r3',
      'resize\r\r0',
      'resize\rb\r -0x1F ',
      'resize\rc\r+00042',
      'resize\rd\r9223372036854775807',
      'resize\ra',
      'resize\ra\r1\r2',
      'Resize\ra\r1',
      'resize\ra\rbar',
      'resize\ra\r',
      'resize\ra\r   ',
      'resize\ra\r-',
      'resize\ra\r+',
      'resize\ra\r0x',
      'resize\ra\r0xg',
      'resize\ra\r 0x1g',
      'resize\ra\r00012a',
      'resize\ra\r1.5',
      'resize\ra\r1234567890123456789x',
      'resize\ra\r12345678901234567890123456789012345678x',
      'resize\ra\r12345678901234567890',
      'resize\ra\r9223372036854775808',
      'resize\ra\r-9223372036854775808',
      'resize\ra\r-9223372036854775809',
      'resize\ra\r0xFFFFFFFFFFFFFFFF',
      'resize\ra\r0x10000000000000000',
      'resize\ra\r-0x8000000000000001',
      'resize\ra\r 7　',
      'resize\ra\r١',
      '\ufeffresize\re\r2',
      '\ufeff\ufeffresize\re\r2',
      'resize\ra\r0x@',
      'resize\ra\r0x1`',
      'resize\ra\r0xaF',
      'resize\ra\r\u1680\u00a0\u0085 5\u202f\u205f\u2028\u2029\ufeff\u200a',
      'resize\ra\r\u2000\u200b5',
      'resize\ra\r5\u0084',
      'resize\rlong\r${'1' * 70}x${'2' * 30}',
      'resize\rlong\r${'1' * 100}x${'2' * 100}',
      'resize\rlong\r${'1' * 10}x${'2' * 100}',
      'resize\rnul\u0000l\r1',
      '',
    ];
    for (final s in legacy) {
      tryLog(log, 'legacy ${jsonEncode(s)}', () => b.handleMessage(bd(s)));
    }
    tryLog(log, 'bad utf8', () => b.handleMessage(bytes(<int>[0xff, 0x61])));
    tryLog(log, 'utf8 ext', () => b.handleMessage(bytes(<int>[0x61, 0x80])));
    tryLog(log, 'utf8 unfinished', () => b.handleMessage(bytes(<int>[0x61, 0xe2, 0x82])));
    tryLog(log, 'utf8 missing ext', () => b.handleMessage(bytes(<int>[0x61, 0xe2, 0x41])));
    tryLog(log, 'utf8 overlong', () => b.handleMessage(bytes(<int>[0x61, 0xc0, 0x80])));
    tryLog(log, 'utf8 surrogate', () => b.handleMessage(bytes(<int>[0x61, 0xed, 0xa0, 0x80])));
    tryLog(log, 'utf8 range', () => b.handleMessage(bytes(<int>[0x61, 0xf4, 0x90, 0x80, 0x80])));
    tryLog(log, 'utf8 C1', () => b.handleMessage(bytes(<int>[0x61, 0xc1, 0xbf])));
    tryLog(log, 'utf8 F5', () => b.handleMessage(bytes(<int>[0x61, 0xf5, 0x80, 0x80, 0x80])));
    tryLog(log, 'utf8 overlong3', () => b.handleMessage(bytes(<int>[0x61, 0xe0, 0x80, 0x80])));
    tryLog(log, 'utf8 overlong4', () => b.handleMessage(bytes(<int>[0xf0, 0x80, 0x80, 0x80])));
    tryLog(log, 'utf8 bom only', () => b.handleMessage(bytes(<int>[0xef, 0xbb, 0xbf])));
    tryLog(log, 'utf8 partial bom', () => b.handleMessage(bytes(<int>[0xef, 0xbb])));
    tryLog(log, 'empty', () => b.handleMessage(bytes(<int>[])));
    // The resized channels behave as resized.
    for (var i = 1; i <= 3; i++) {
      b.push('chan', bd('c$i'), reply(log, 'c$i'));
    }
    b.setListener('chan', listener(log, 'C'));
  },
  'null_names': (FakeAsync fa, void Function(String) log) {
    final b = ui.ChannelBuffers();
    tryLog(log, 'push', () => b.push('a\u0000', bd('1'), reply(log, '1')));
    tryLog(log, 'setListener', () => b.setListener('a\u0000', listener(log, 'A')));
    tryLog(log, 'resize new', () => b.resize('a\u0000', 2));
    tryLog(log, 'allowOverflow new', () => b.allowOverflow('a\u0000', true));
    tryLog(log, 'allowOverflow false', () => b.allowOverflow('a\u0000', false));
    tryLog(log, 'clearListener', () => b.clearListener('a\u0000'));
  },
  'responses': (FakeAsync fa, void Function(String) log) {
    final b = ui.ChannelBuffers();
    b.setListener('a', (ByteData? d, ui.PlatformMessageResponseCallback cb) {
      log('A got ${show(d)}');
      if (show(d) == 'late') {
        Timer(const Duration(milliseconds: 3), () => cb(bd('late reply')));
      } else if (show(d) == 'twice') {
        cb(bd('first'));
        cb(null);
      } else if (show(d) != 'never') {
        cb(d);
      }
    });
    log('direct push:');
    b.push('a', bd('now'), reply(log, 'now'));
    b.push('a', bd('late'), reply(log, 'late'));
    b.push('a', bd('twice'), reply(log, 'twice'));
    b.push('a', bd('never'), reply(log, 'never'));
    b.push('a', null, reply(log, 'null'));
    b.push('a', bd(''), reply(log, 'empty'));
    log('sync end');
  },
  'channel_buffers_singleton': (FakeAsync fa, void Function(String) log) {
    log('kDefaultBufferSize ${ui.ChannelBuffers.kDefaultBufferSize}');
    log('kControlChannelName ${ui.ChannelBuffers.kControlChannelName}');
    log('identical ${identical(ui.channelBuffers, ui.channelBuffers)}');
  },
};

void main() {
  test('channel_buffers', () {
    final out = <String, List<String>>{};
    for (final MapEntry<String, Scene> e in scenes.entries) {
      final log = <String>[];
      FakeAsync().run((FakeAsync fa) {
        e.value(fa, log.add);
        fa.flushMicrotasks();
        fa.flushTimers();
      });
      out[e.key] = log;
    }
    File('build/channel_buffers.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
