// Reference side of flutter-go's Stream differential test (the subset of
// dart:async streams that foundation uses): each scene logs the order in
// which stream callbacks run, inside package:fake_async. Output:
// build/stream.json as {scene: [log lines]}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:fake_async/fake_async.dart';
import 'package:flutter_test/flutter_test.dart';

typedef Scene = void Function(FakeAsync fa, void Function(String) log);

String _msg(Object e) => e is StateError ? e.message : e.toString();

final Map<String, Scene> scenes = <String, Scene>{
  'controller_basic': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = StreamController<int>(
      onListen: () => log('onListen'),
      onPause: () => log('onPause'),
      onResume: () => log('onResume'),
      onCancel: () => log('onCancel'),
    );
    c.add(1);
    c.add(2);
    log('before listen hasListener=${c.hasListener} isPaused=${c.isPaused}');
    c.stream.listen(
      (int v) => log('data $v'),
      onDone: () => log('done'),
    );
    log('after listen hasListener=${c.hasListener} isPaused=${c.isPaused}');
    scheduleMicrotask(() => log('micro1'));
    c.add(3);
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
      c.stream.listen(null);
    } catch (e) {
      log('second listen: ${_msg(e)}');
    }
  },
  'pause_resume': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = StreamController<int>(
      onPause: () => log('onPause'),
      onResume: () => log('onResume'),
    );
    late StreamSubscription<int> sub;
    sub = c.stream.listen((int v) {
      log('data $v');
      if (v == 1) {
        sub.pause();
        log('paused isPaused=${sub.isPaused} controller.isPaused=${c.isPaused}');
        Timer(const Duration(milliseconds: 5), () {
          log('resuming');
          sub.resume();
        });
      }
      if (v == 3) {
        sub.pause(Future<void>.delayed(const Duration(milliseconds: 3)));
        sub.pause();
        log('double paused');
        Timer(const Duration(milliseconds: 1), () {
          log('resume once');
          sub.resume();
        });
      }
    }, onDone: () => log('done'));
    c.add(1);
    c.add(2);
    c.add(3);
    c.add(4);
    c.close();
    scheduleMicrotask(() => log('micro'));
    fa.elapse(const Duration(milliseconds: 20));
    log('end');
  },
  'cancel': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = StreamController<int>(
      onCancel: () {
        log('onCancel');
        return Future<void>.delayed(const Duration(milliseconds: 2), () => log('onCancel future'));
      },
    );
    late StreamSubscription<int> sub;
    sub = c.stream.listen((int v) {
      log('data $v');
      if (v == 2) {
        sub.cancel().then((_) => log('cancel future'));
        log('canceled hasListener=${c.hasListener}');
      }
    }, onDone: () => log('done (never)'));
    c.add(1);
    c.add(2);
    c.add(3);
    c.close().then((_) => log('close future'));
    fa.elapse(const Duration(milliseconds: 5));
    try {
      c.add(4);
    } catch (e) {
      log('add after close: ${_msg(e)}');
    }
    log('end isClosed=${c.isClosed}');
  },
  'errors': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> a = StreamController<int>();
    a.stream.listen(
      (int v) => log('a data $v'),
      onError: (Object e) => log('a error ${_msg(e)}'),
      onDone: () => log('a done'),
    );
    a.add(1);
    a.addError(StateError('e1'));
    a.add(2);
    a.close();
    final StreamController<int> b = StreamController<int>(
      onCancel: () => Future<void>.delayed(const Duration(milliseconds: 1), () => log('b onCancel future')),
    );
    b.stream.listen(
      (int v) => log('b data $v'),
      onError: (Object e) => log('b error ${_msg(e)}'),
      onDone: () => log('b done (never)'),
      cancelOnError: true,
    );
    b.add(1);
    b.addError(StateError('e2'));
    b.add(2);
    scheduleMicrotask(() => log('micro'));
    fa.elapse(const Duration(milliseconds: 5));
    log('end');
  },
  'from_iterable': (FakeAsync fa, void Function(String) log) {
    late StreamSubscription<int> sub;
    sub = Stream<int>.fromIterable(<int>[1, 2, 3, 4]).listen((int v) {
      log('data $v');
      if (v == 2) {
        sub.pause();
        Timer(const Duration(milliseconds: 1), () {
          log('resume');
          sub.resume();
        });
      }
    }, onDone: () => log('done'));
    scheduleMicrotask(() => log('micro1'));
    scheduleMicrotask(() => scheduleMicrotask(() => log('micro2')));
    log('sync end');
    fa.elapse(const Duration(milliseconds: 2));
    Stream<int>.fromIterable(<int>[5, 6]).listen((int v) {
      log('data $v');
    }).cancel(); // Its future is the root zone's null future: not fake time.
    Stream<int>.fromIterable(<int>[]).listen(null, onDone: () => log('empty iterable done'));
    log('end');
  },
  'empty': (FakeAsync fa, void Function(String) log) {
    Stream<int>.empty().listen(null, onDone: () => log('empty done'));
    final StreamSubscription<int> paused = Stream<int>.empty().listen(
      null,
      onDone: () => log('paused empty done'),
    );
    paused.pause();
    log('paused isPaused=${paused.isPaused}');
    scheduleMicrotask(() => log('micro'));
    Timer(const Duration(milliseconds: 1), () {
      log('resume');
      paused.resume();
    });
    Stream<int>.empty().listen(null, onDone: () => log('never')).cancel();
    log('sync end');
    fa.elapse(const Duration(milliseconds: 2));
  },
  'add_stream': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = StreamController<int>(
      onListen: () => log('onListen'),
      onCancel: () => log('onCancel'),
    );
    c.addStream(Stream<int>.fromIterable(<int>[1, 2])).then((_) {
      log('addStream 1 done');
      c.add(3);
      c
          .addStream(Stream<int>.fromIterable(<int>[4, 5]))
          .then((_) => log('addStream 2 done'))
          .then((_) => c.close())
          .then((_) => log('closed'));
      try {
        c.add(99);
      } catch (e) {
        log('add while adding: ${_msg(e)}');
      }
    });
    log('before listen');
    scheduleMicrotask(() => log('micro'));
    c.stream.listen((int v) => log('data $v'), onDone: () => log('done'));
    log('sync end');
    fa.elapse(const Duration(milliseconds: 1));
    final StreamController<int> errorsEnd = StreamController<int>();
    errorsEnd.stream.listen(
      (int v) => log('ee data $v'),
      onError: (Object e) => log('ee error ${_msg(e)}'),
    );
    final StreamController<int> source = StreamController<int>();
    errorsEnd.addStream(source.stream, cancelOnError: true).then((_) => log('ee addStream done'));
    source.add(7);
    source.addError(StateError('stop'));
    source.add(8);
    fa.elapse(const Duration(milliseconds: 1));
    errorsEnd.add(9);
    fa.flushMicrotasks();
    log('end');
  },
  'pause_while_scheduled': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = StreamController<int>();
    final StreamSubscription<int> sub = c.stream.listen((int v) => log('data $v'));
    c.add(1);
    c.add(2);
    sub.pause();
    scheduleMicrotask(() => log('micro'));
    Timer(const Duration(milliseconds: 1), () {
      log('resume');
      sub.resume();
    });
    fa.elapse(const Duration(milliseconds: 2));
    log('end');
  },
  'add_stream_late_listen': (FakeAsync fa, void Function(String) log) {
    final StreamController<int> c = StreamController<int>();
    c.addStream(Stream<int>.fromIterable(<int>[1, 2, 3])).then((_) => log('addStream done'));
    scheduleMicrotask(() => log('micro'));
    Timer(const Duration(milliseconds: 1), () {
      log('listen');
      c.stream.listen((int v) => log('data $v'));
      scheduleMicrotask(() => log('micro after listen'));
      scheduleMicrotask(() => scheduleMicrotask(() => log('micro 2 after listen')));
    });
    fa.elapse(const Duration(milliseconds: 2));
    log('end');
  },
  'to_list_and_from_future': (FakeAsync fa, void Function(String) log) {
    Stream<int>.fromIterable(<int>[1, 2, 3]).toList().then((List<int> l) => log('list $l'));
    final StreamController<int> c = StreamController<int>();
    c.stream.toList().then((List<int> l) => log('controller list $l'));
    c.add(7);
    c.close();
    Stream<int>.fromFuture(Future<int>.value(5)).listen(
      (int v) => log('fromFuture data $v'),
      onDone: () => log('fromFuture done'),
    );
    Stream<int>.fromFuture(Future<int>.error(StateError('bad'))).listen(
      (int v) => log('never'),
      onError: (Object e) => log('fromFuture error ${_msg(e)}'),
      onDone: () => log('fromFuture error done'),
    );
    final StreamController<int> failing = StreamController<int>();
    failing.stream.toList().then(
      (List<int> l) => log('never $l'),
      onError: (Object e) => log('toList error ${_msg(e)}'),
    );
    failing.add(1);
    failing.addError(StateError('x'));
    scheduleMicrotask(() => log('micro'));
    log('sync end');
  },
};

void main() {
  test('stream', () {
    final Map<String, List<String>> out = <String, List<String>>{};
    for (final MapEntry<String, Scene> e in scenes.entries) {
      final List<String> log = <String>[];
      FakeAsync().run((FakeAsync fa) {
        e.value(fa, log.add);
        fa.flushMicrotasks();
        fa.flushTimers();
      });
      out[e.key] = log;
    }
    File('build/stream.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
