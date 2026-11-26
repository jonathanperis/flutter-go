// Reference side of flutter-go's broadcast StreamController differential
// test: each scene logs the order in which the callbacks of broadcast
// controllers (async and sync) and their listeners run, inside
// package:fake_async. Output: build/broadcast_stream.json as
// {scene: [log lines]}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:fake_async/fake_async.dart';
import 'package:flutter_test/flutter_test.dart';

typedef Scene = void Function(FakeAsync fa, void Function(String) log);

String _msg(Object e) => e is StateError ? e.message : e.toString();

StreamController<int> _controller(void Function(String) log, {bool sync = false}) {
  return StreamController<int>.broadcast(
    onListen: () => log('onListen'),
    onCancel: () => log('onCancel'),
    sync: sync,
  );
}

void _basic(FakeAsync fa, void Function(String) log, bool sync) {
  final StreamController<int> c = _controller(log, sync: sync);
  c.add(0); // no listener: dropped
  c.addError(StateError('dropped'));
  log('hasListener=${c.hasListener} isPaused=${c.isPaused}');
  c.stream.listen((int v) => log('a $v'), onDone: () => log('a done'));
  log('after a hasListener=${c.hasListener}');
  c.add(1);
  c.stream.listen((int v) => log('b $v'), onDone: () => log('b done'));
  c.add(2);
  scheduleMicrotask(() => log('micro1'));
  c.addError(StateError('e'));
  c.close().then((_) => log('close future'));
  c.done.then((_) => log('done future'));
  scheduleMicrotask(() => log('micro2'));
  log('sync end isClosed=${c.isClosed}');
  try {
    c.add(4);
  } catch (e) {
    log('add after close: ${_msg(e)}');
  }
  try {
    c.addError(StateError('late'));
  } catch (e) {
    log('addError after close: ${_msg(e)}');
  }
  c.stream.listen((int v) => log('late $v'), onDone: () => log('late done'));
  log('listened after close hasListener=${c.hasListener}');
}

void _cancelInData(FakeAsync fa, void Function(String) log, bool sync) {
  final StreamController<int> c = _controller(log, sync: sync);
  late StreamSubscription<int> a;
  late StreamSubscription<int> b;
  a = c.stream.listen((int v) {
    log('a $v');
    if (v == 2) {
      a.cancel(); // (its _nullFuture is the root zone's: fake_async never runs a then on it)
      log('a canceled hasListener=${c.hasListener}');
    }
  });
  b = c.stream.listen((int v) {
    log('b $v');
    if (v == 3) {
      b.cancel();
      log('b canceled hasListener=${c.hasListener}');
      c.stream.listen((int v) => log('c $v'));
      log('c added');
    }
  });
  c.add(1);
  c.add(2);
  c.add(3);
  c.add(4);
  scheduleMicrotask(() => log('micro'));
  fa.flushMicrotasks();
  c.add(5);
  fa.flushMicrotasks();
  log('end');
}

void _addDuringFiring(FakeAsync fa, void Function(String) log, bool sync) {
  final StreamController<int> c = _controller(log, sync: sync);
  c.stream.listen((int v) {
    log('a $v');
    if (v == 1) {
      try {
        c.add(10);
        log('added 10');
      } catch (e) {
        log('add during firing: ${_msg(e)}');
      }
      c.stream.listen((int v) => log('new $v'));
    }
  });
  c.stream.listen((int v) => log('b $v'));
  c.add(1);
  c.add(2);
  fa.flushMicrotasks();
  log('end');
}

final Map<String, Scene> scenes = <String, Scene>{
  'basic_async': (FakeAsync fa, void Function(String) log) => _basic(fa, log, false),
  'basic_sync': (FakeAsync fa, void Function(String) log) => _basic(fa, log, true),
  'cancel_in_data_async': (FakeAsync fa, void Function(String) log) =>
      _cancelInData(fa, log, false),
  'cancel_in_data_sync': (FakeAsync fa, void Function(String) log) =>
      _cancelInData(fa, log, true),
  'add_during_firing_async': (FakeAsync fa, void Function(String) log) =>
      _addDuringFiring(fa, log, false),
  'add_during_firing_sync': (FakeAsync fa, void Function(String) log) =>
      _addDuringFiring(fa, log, true),
  'add_during_firing_sync_single': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = _controller(log, sync: true);
    c.stream.listen((int v) {
      log('a $v');
      if (v == 1) {
        try {
          c.add(10);
          log('added 10');
        } catch (e) {
          log('add during firing: ${_msg(e)}');
        }
        try {
          c.close();
        } catch (e) {
          log('close during firing: ${_msg(e)}');
        }
      }
    });
    c.add(1);
    c.add(2);
    log('end');
  },
  'relisten': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = _controller(log);
    StreamSubscription<int> s1 = c.stream.listen((int v) => log('s1 $v'));
    StreamSubscription<int> s2 = c.stream.listen((int v) => log('s2 $v'));
    c.add(1);
    s1.cancel();
    log('s1 canceled hasListener=${c.hasListener}');
    s2.cancel();
    log('s2 canceled hasListener=${c.hasListener}');
    s2.cancel();
    log('s2 canceled twice');
    c.add(2);
    fa.flushMicrotasks();
    c.stream.listen((int v) => log('s3 $v'), onDone: () => log('s3 done'));
    c.add(3);
    c.close().then((_) => log('close future'));
    fa.flushMicrotasks();
    log('end');
  },
  'pause_one_listener': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = _controller(log);
    late StreamSubscription<int> a;
    a = c.stream.listen((int v) {
      log('a $v');
      if (v == 1) {
        a.pause();
        log('a paused isPaused=${a.isPaused} controller.isPaused=${c.isPaused}');
        Timer(const Duration(milliseconds: 3), () {
          log('resume a');
          a.resume();
        });
      }
    }, onDone: () => log('a done'));
    c.stream.listen((int v) => log('b $v'), onDone: () => log('b done'));
    c.add(1);
    c.add(2);
    c.add(3);
    c.close().then((_) => log('close future'));
    fa.elapse(const Duration(milliseconds: 1));
    log('after 1ms');
    fa.elapse(const Duration(milliseconds: 5));
    log('end');
  },
  'pause_with_resume_signal': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = _controller(log);
    final StreamSubscription<int> a = c.stream.listen((int v) => log('a $v'));
    a.pause(Future<void>.delayed(const Duration(milliseconds: 2)));
    c.add(1);
    c.add(2);
    fa.flushMicrotasks();
    log('paused flushed');
    fa.elapse(const Duration(milliseconds: 3));
    a.cancel();
    log('end');
  },
  'close_without_listeners': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = _controller(log);
    c.done.then((_) => log('done future'));
    c.close().then((_) => log('close future'));
    log('closed');
    final Future<void> again = c.close();
    log('close again identical=${identical(again, c.done)}');
    scheduleMicrotask(() => log('micro'));
    c.stream.listen((int v) => log('never $v'), onDone: () => log('late done'));
    fa.flushMicrotasks();
    log('end');
  },
  'errors': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = _controller(log);
    c.stream.listen(
      (int v) => log('a $v'),
      onError: (Object e) => log('a error ${_msg(e)}'),
      onDone: () => log('a done'),
    );
    c.stream.listen(
      (int v) => log('b $v'),
      onError: (Object e) => log('b error ${_msg(e)}'),
      onDone: () => log('b done'),
      cancelOnError: true,
    );
    c.add(1);
    c.addError(StateError('boom'));
    c.add(2);
    c.close().then((_) => log('close future'));
    fa.flushMicrotasks();
    log('end');
  },
  'to_list': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = _controller(log);
    c.stream.toList().then((List<int> l) => log('list1 $l'));
    c.add(1);
    c.stream.toList().then((List<int> l) => log('list2 $l'));
    c.add(2);
    c.close();
    final StreamController<int> f = _controller(log);
    f.stream.toList().then(
      (List<int> l) => log('never $l'),
      onError: (Object e) => log('toList error ${_msg(e)}'),
    );
    f.add(1);
    f.addError(StateError('x'));
    f.add(2);
    log('sync end');
  },
  'add_stream': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = _controller(log);
    c.stream.listen((int v) => log('a $v'), onError: (Object e) => log('a error ${_msg(e)}'));
    c.addStream(Stream<int>.fromIterable(<int>[1, 2, 3])).then((_) => log('addStream done'));
    try {
      c.add(9);
    } catch (e) {
      log('add during addStream: ${_msg(e)}');
    }
    try {
      c.close();
    } catch (e) {
      log('close during addStream: ${_msg(e)}');
    }
    fa.flushMicrotasks();
    final StreamController<int> src = StreamController<int>();
    c
        .addStream(src.stream, cancelOnError: true)
        .then((_) => log('addStream2 done'));
    src.add(4);
    src.addError(StateError('stop'));
    src.add(5);
    fa.flushMicrotasks();
    c.add(6);
    c.close().then((_) => log('close future'));
    fa.flushMicrotasks();
    log('end');
  },
  'cancel_on_done_sync_onCancel_order': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = _controller(log, sync: true);
    c.stream.listen((int v) => log('a $v'), onDone: () => log('a done'));
    c.stream.listen((int v) => log('b $v'), onDone: () => log('b done'));
    c.done.then((_) => log('done future'));
    c.add(1);
    log('after add');
    c.close().then((_) => log('close future'));
    log('after close hasListener=${c.hasListener}');
    fa.flushMicrotasks();
    log('end');
  },
};

void main() {
  test('broadcast_stream', () {
    final Map<String, List<String>> out = <String, List<String>>{};
    for (final MapEntry<String, Scene> e in scenes.entries) {
      final List<String> lines = <String>[];
      FakeAsync().run((FakeAsync fa) {
        runZonedGuarded(() {
          e.value(fa, lines.add);
          fa.flushMicrotasks();
          fa.flushTimers();
        }, (Object error, StackTrace stack) => lines.add('uncaught ${_msg(error)}'));
      });
      out[e.key] = lines;
    }
    File('build/broadcast_stream.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
