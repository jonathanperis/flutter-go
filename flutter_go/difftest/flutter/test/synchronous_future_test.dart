// Reference side of flutter-go's SynchronousFuture differential test: each
// scene logs the order in which callbacks run when SynchronousFuture (a
// user-defined Future) meets dart:async's own futures, inside
// package:fake_async. Output: build/synchronous_future.json as
// {scene: [log lines]}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:fake_async/fake_async.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

typedef Scene = void Function(FakeAsync fa, void Function(String) log);

String _msg(Object e) => e is StateError ? e.message : e.toString();

final Map<String, Scene> scenes = <String, Scene>{
  'then_sync': (FakeAsync fa, void Function(String) log) {
    final f = SynchronousFuture<int>(1);
    log('before then');
    f
        .then((int v) {
          log('then $v');
          return v + 1;
        })
        .then((int v) => log('chained $v'));
    log('after then');
    scheduleMicrotask(() => log('micro'));
    f.then((int v) => log('again $v'));
    log('sync end');
  },
  'then_returns_future': (FakeAsync fa, void Function(String) log) {
    SynchronousFuture<int>(1)
        .then((int v) => Future<int>.value(v * 10))
        .then((int v) => log('core result $v'));
    SynchronousFuture<int>(2)
        .then((int v) => SynchronousFuture<int>(v * 10))
        .then((int v) => log('sync result $v'));
    final Completer<int> c = Completer<int>();
    SynchronousFuture<int>(3).then((int v) => c.future).then((int v) => log('completer result $v'));
    scheduleMicrotask(() => log('micro'));
    c.complete(30);
    log('sync end');
  },
  'then_throws': (FakeAsync fa, void Function(String) log) {
    try {
      SynchronousFuture<int>(1).then<int>((int v) => throw StateError('boom'));
      log('not thrown');
    } catch (e) {
      log('thrown synchronously ${_msg(e)}');
    }
  },
  'await': (FakeAsync fa, void Function(String) log) {
    Future<int> f() async {
      log('f start');
      final int v = await SynchronousFuture<int>(5);
      log('f resumed $v');
      final int w = await SynchronousFuture<int>(6);
      log('f resumed2 $w');
      return v + w;
    }

    Future<int> g() async {
      log('g start');
      await SynchronousFuture<int>(1);
      log('g after sync await');
      await Future<void>.value();
      log('g after core await');
      return 2;
    }

    Future<int> h() async {
      await SynchronousFuture<int>(1);
      throw StateError('after sync await');
    }

    Future<int> k() async {
      throw StateError('before await');
    }

    f().then((int v) => log('f result $v'));
    log('after f()');
    g().then((int v) => log('g result $v'));
    log('after g()');
    // h fails synchronously, before anyone can listen: Dart reports the
    // error as uncaught (the zone's handler logs it), then delivers it to the
    // late listener as well.
    runZonedGuarded(() {
      final Future<int> hf = h();
      scheduleMicrotask(() => log('micro'));
      hf.catchError((Object e) {
        log('h caught ${_msg(e)}');
        return 0;
      });
    }, (Object e, StackTrace s) => log('uncaught ${_msg(e)}'));
    final Future<int> kf = k();
    kf.catchError((Object e) {
      log('k caught ${_msg(e)}');
      return 0;
    });
    log('sync end');
  },
  'completer_chains_foreign': (FakeAsync fa, void Function(String) log) {
    final Completer<int> a = Completer<int>();
    final Completer<int> s = Completer<int>.sync();
    a.future.then((int v) => log('async completer $v'));
    s.future.then((int v) => log('sync completer $v'));
    scheduleMicrotask(() => log('micro1'));
    a.complete(SynchronousFuture<int>(1));
    log('after a.complete isCompleted=${a.isCompleted}');
    s.complete(SynchronousFuture<int>(2));
    log('after s.complete');
    Future<int>.value(3)
        .then((int v) => SynchronousFuture<int>(v + 1))
        .then((int v) => log('core then foreign $v'));
    scheduleMicrotask(() => log('micro2'));
    log('sync end');
  },
  'completer_chains_completed_core': (FakeAsync fa, void Function(String) log) {
    final Future<int> done = Future<int>.value(1);
    fa.flushMicrotasks();
    final Completer<int> c = Completer<int>();
    c.complete(done);
    scheduleMicrotask(() => log('micro'));
    c.future.then((int v) => log('completer $v'));
    final Completer<int> d = Completer<int>();
    d.future.then((int v) => log('listened first $v'));
    d.complete(done);
    scheduleMicrotask(() => log('micro2'));
    log('sync end');
  },
  'when_complete': (FakeAsync fa, void Function(String) log) {
    final f = SynchronousFuture<int>(4);
    final Future<int> w = f.whenComplete(() => log('action'));
    log('same future ${identical(w, f)}');
    w.then((int v) => log('after whenComplete $v'));
    f
        .whenComplete(() => throw StateError('action failed'))
        .then((int v) => log('never $v'))
        .catchError((Object e) {
          log('caught ${_msg(e)}');
          return 0;
        });
    scheduleMicrotask(() => log('micro'));
    log('sync end');
  },
  'catch_error_never_completes': (FakeAsync fa, void Function(String) log) {
    SynchronousFuture<int>(1)
        .catchError((Object e) => 0)
        .then((int v) => log('never $v'))
        .whenComplete(() => log('never done'));
    fa.elapse(const Duration(seconds: 1));
    log('end');
  },
  'timeout': (FakeAsync fa, void Function(String) log) {
    SynchronousFuture<int>(7)
        .timeout(const Duration(milliseconds: 5))
        .then((int v) => log('sync timeout value $v'));
    log('after sync timeout then');
    scheduleMicrotask(() => log('micro'));
    final Future<int> done = Future<int>.value(8);
    fa.flushMicrotasks();
    final Future<int> t = done.timeout(const Duration(milliseconds: 5));
    scheduleMicrotask(() => log('micro2'));
    t.then((int v) => log('completed timeout $v'));
    Completer<int>().future
        .timeout(const Duration(milliseconds: 5))
        .catchError((Object e) {
          log('timed out at ${fa.elapsed.inMilliseconds}: $e');
          return 0;
        });
    Completer<int>().future
        .timeout(const Duration(milliseconds: 3), onTimeout: () => 9)
        .then((int v) => log('onTimeout $v at ${fa.elapsed.inMilliseconds}'));
    final Completer<int> late = Completer<int>();
    late.future
        .timeout(const Duration(milliseconds: 10))
        .then((int v) => log('in time $v at ${fa.elapsed.inMilliseconds}'));
    Timer(const Duration(milliseconds: 4), () => late.complete(10));
    final Completer<int> tooLate = Completer<int>();
    final Future<int> tl = tooLate.future.timeout(
      const Duration(milliseconds: 5),
      onTimeout: () => -1,
    );
    Timer(const Duration(milliseconds: 8), () => tooLate.complete(11));
    final Future<int> failed = Future<int>.error(StateError('failed'));
    failed.catchError((Object e) => 0);
    fa.elapse(const Duration(milliseconds: 20));
    tl.then((int v) => log('too late result $v'));
    final Future<int> tf = failed.timeout(const Duration(milliseconds: 5));
    scheduleMicrotask(() => log('micro3'));
    tf.catchError((Object e) {
      log('failed timeout ${_msg(e)}');
      return 0;
    });
    fa.flushMicrotasks();
    final Completer<int> c = Completer<int>();
    c.complete(failed);
    scheduleMicrotask(() => log('micro4'));
    c.future.catchError((Object e) {
      log('completer caught ${_msg(e)}');
      return 0;
    });
    log('end');
  },
  'ignore': (FakeAsync fa, void Function(String) log) {
    SynchronousFuture<int>(1).ignore();
    Future<int>.error(StateError('ignored')).ignore();
    log('end');
  },
};

void main() {
  test('synchronous_future', () {
    final Map<String, List<String>> out = <String, List<String>>{};
    for (final MapEntry<String, Scene> e in scenes.entries) {
      final List<String> log = <String>[];
      FakeAsync().run((FakeAsync fa) {
        e.value(fa, log.add);
        fa.flushMicrotasks();
      });
      out[e.key] = log;
    }
    File('build/synchronous_future.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
