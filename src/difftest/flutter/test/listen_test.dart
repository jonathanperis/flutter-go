// Reference side of flutter-go's package:listen differential test
// (ChangeNotifier, ValueNotifier, Listenable.merge as Flutter 3.50.1
// re-exports them from foundation): the order of listener calls while
// listeners are added and removed during notifications (also nested ones),
// duplicates, errors routed through Listenable.onError (custom, default and
// the binding's), dispose and use-after-dispose, ValueNotifier's equality
// skip and toString, tear-off removal, and the memory-allocation events.
// Hashes are masked (#HASH). Output: build/listen.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:listen/listen.dart' show ErrorCallback, ErrorContext;

final List<String> entries = <String>[];
void log(String s) => entries.add(s.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH'));

class TestNotifier extends ChangeNotifier {
  void notify() => notifyListeners();
  bool get listened => hasListeners;
}

class LoggingValueNotifier extends ValueNotifier<int> {
  LoggingValueNotifier(super.value);

  @override
  void notifyListeners() {
    log('override notifyListeners($value)');
    super.notifyListeners();
  }
}

class P {
  P(this.x, this.y);
  final int x;
  final int y;

  @override
  bool operator ==(Object other) => other is P && other.x == x;

  @override
  int get hashCode => x;

  @override
  String toString() => 'P($x, $y)';
}

class MyException implements Exception {
  MyException(this.n);
  final int n;

  @override
  String toString() => 'MyException($n)';
}

class Handler {
  Handler(this.name);
  final String name;
  void handle() => log('handle $name');
}

class ListenBinding extends BindingBase {
  @override
  void registerServiceExtension({required String name, required ServiceExtensionCallback callback}) {}
}

String describe(Object e) {
  if (e is FlutterError) {
    return 'FlutterError: ${e.toStringDeep()}';
  }
  if (e is AssertionError) {
    return 'AssertionError: ${e.message}';
  }
  if (e is ArgumentError) {
    return 'ArgumentError: $e';
  }
  if (e is StateError) {
    return 'StateError: $e';
  }
  if (e is String) {
    return 'String: $e';
  }
  return 'Object: $e';
}

void main() {
  test('listen', () {
    final ErrorCallback defaultOnError = Listenable.onError;
    void loggingOnError(Object error, StackTrace? stack, ErrorContext context) {
      log('onError ${context.name} ${describe(error)} stack=${stack != null}');
    }

    VoidCallback l(String name) => () => log(name);

    // basic: duplicates, removal of one registration, unknown listeners.
    log('--- basic');
    Listenable.onError = loggingOnError;
    final basic = TestNotifier();
    final VoidCallback a = l('a');
    final VoidCallback b = l('b');
    final VoidCallback c = l('c');
    log('hasListeners=${basic.listened}');
    basic.addListener(a);
    basic.addListener(a);
    basic.addListener(b);
    basic.notify();
    basic.removeListener(a);
    basic.notify();
    basic.removeListener(c);
    basic.notify();
    basic.removeListener(a);
    log('hasListeners=${basic.listened}');
    basic.removeListener(b);
    log('hasListeners=${basic.listened}');
    basic.notify();
    for (var i = 0; i < 9; i++) {
      basic.addListener(i.isEven ? a : b);
    }
    basic.removeListener(b);
    basic.removeListener(a);
    basic.removeListener(b);
    basic.notify();

    // mutating: removing earlier and later listeners and adding ones during
    // a notification.
    log('--- mutating');
    final mut = TestNotifier();
    final VoidCallback d = l('d');
    late final VoidCallback mutator;
    mutator = () {
      log('mutator');
      mut.removeListener(a);
      mut.removeListener(c);
      mut.addListener(d);
      mut.addListener(mutator);
      mut.removeListener(mutator);
    };
    mut.addListener(a);
    mut.addListener(mutator);
    mut.addListener(c);
    mut.addListener(b);
    for (var i = 0; i < 3; i++) {
      log('notify $i');
      mut.notify();
    }

    // duplicates removed once during a notification are still called.
    log('--- duplicates');
    final dup = TestNotifier();
    final VoidCallback removeA = () {
      log('removeA');
      dup.removeListener(a);
    };
    dup.addListener(a);
    dup.addListener(removeA);
    dup.addListener(a);
    dup.notify();
    dup.notify();
    final dup2 = TestNotifier();
    final VoidCallback removeB = () {
      log('removeB');
      dup2.removeListener(b);
    };
    dup2.addListener(removeB);
    dup2.addListener(b);
    dup2.addListener(b);
    dup2.notify();
    dup2.notify();
    dup2.removeListener(removeB);
    dup2.notify();

    // self removal and removing everything during a notification.
    log('--- self removal');
    final self = TestNotifier();
    late final VoidCallback selfRemoving;
    selfRemoving = () {
      log('selfRemoving');
      self.removeListener(selfRemoving);
    };
    self.addListener(a);
    self.addListener(selfRemoving);
    self.addListener(b);
    self.notify();
    self.notify();
    final VoidCallback clearAll = () {
      log('clearAll');
      self.removeListener(a);
      self.removeListener(b);
    };
    self.addListener(clearAll);
    self.notify();
    log('hasListeners=${self.listened}');
    self.removeListener(clearAll);
    log('hasListeners=${self.listened}');

    // nested notifications: removal and addition inside the inner one,
    // compaction after the outer one.
    log('--- nested');
    final nest = TestNotifier();
    var depth = 0;
    final VoidCallback e = l('e');
    late final VoidCallback nester;
    nester = () {
      log('nester depth=$depth');
      if (depth < 2) {
        depth++;
        nest.removeListener(a);
        nest.addListener(e);
        nest.notify();
        depth--;
        log('back to depth=$depth');
      } else {
        nest.removeListener(b);
        nest.removeListener(nester);
      }
    };
    nest.addListener(a);
    nest.addListener(nester);
    nest.addListener(b);
    nest.addListener(c);
    nest.notify();
    log('after nested');
    nest.notify();

    // errors with a custom onError: the other listeners still run.
    log('--- errors');
    final err = TestNotifier();
    err.addListener(() => throw 'boom');
    err.addListener(a);
    err.addListener(() => throw ArgumentError());
    err.addListener(() => throw StateError('state'));
    err.addListener(() => throw MyException(1));
    late final VoidCallback throwsAndRemoves;
    throwsAndRemoves = () {
      err.removeListener(throwsAndRemoves);
      throw ArgumentError('removed');
    };
    err.addListener(throwsAndRemoves);
    err.addListener(b);
    err.notify();
    err.notify();

    // onError that throws inside a notification: the exception escapes.
    Listenable.onError = (Object error, StackTrace? stack, ErrorContext context) {
      log('rethrowing onError ${describe(error)}');
      throw MyException(2);
    };
    final escape = TestNotifier();
    escape.addListener(() => throw 'first');
    escape.addListener(a);
    try {
      escape.notify();
    } catch (e) {
      log('caught ${describe(e)}');
    }
    Listenable.onError = loggingOnError;

    // dispose: during a notification, use after dispose.
    log('--- dispose');
    final disp = TestNotifier();
    var finished = false;
    disp.addListener(() {
      disp.dispose();
      finished = true;
    });
    disp.addListener(a);
    disp.notify();
    log('finished=$finished hasListeners=${disp.listened}');
    final disp2 = TestNotifier();
    disp2.addListener(a);
    disp2.dispose();
    log('hasListeners=${disp2.listened}');
    disp2.removeListener(a);
    disp2.addListener(b);
    log('hasListeners=${disp2.listened}');
    disp2.notify();
    log('debugAssertNotDisposed=${ChangeNotifier.debugAssertNotDisposed(disp2)}');
    disp2.dispose();
    log('hasListeners=${disp2.listened}');
    final plain = ChangeNotifier();
    plain.dispose();
    plain.notifyListeners();

    // default onError: rethrows, abandoning the notification (the call depth
    // stays incremented: removals are deferred, dispose asserts).
    log('--- default onError');
    Listenable.onError = defaultOnError;
    final def = TestNotifier();
    def.addListener(() => throw StateError('default'));
    def.addListener(a);
    try {
      def.notify();
    } catch (e) {
      log('caught ${describe(e)}');
    }
    def.removeListener(a);
    log('hasListeners=${def.listened}');
    final VoidCallback removeFirst = () => log('never');
    def.addListener(removeFirst);
    try {
      def.notify();
    } catch (e) {
      log('caught ${describe(e)}');
    }
    try {
      def.dispose();
    } catch (e) {
      log('caught ${describe(e)}');
    }
    final disposed = TestNotifier()..dispose();
    try {
      disposed.notify();
    } catch (e) {
      log('caught ${describe(e)}');
    }
    Listenable.onError = loggingOnError;

    // ValueNotifier: equality skip, toString, overrides.
    log('--- ValueNotifier');
    final vd = ValueNotifier<double>(2.0);
    vd.addListener(() => log('vd ${vd.value}'));
    for (final double v in <double>[3.0, 3.0, double.nan, double.nan, -0.0, 0.0, double.infinity, 1e21, 0.1]) {
      vd.value = v;
    }
    log('$vd');
    final vs = ValueNotifier<String>('a');
    vs.addListener(() => log('vs ${vs.value}'));
    vs.value = 'a';
    vs.value = 'b';
    vs.value = 'b';
    log('$vs');
    final list = <int>[1, 2];
    final vl = ValueNotifier<List<int>>(list);
    vl.addListener(() => log('vl ${vl.value}'));
    vl.value = list;
    vl.value = <int>[1, 2];
    log('$vl');
    final vo = ValueNotifier<Object?>(null);
    vo.addListener(() => log('vo ${vo.value}'));
    vo.value = null;
    vo.value = 'x';
    vo.value = 'x';
    vo.value = 1;
    vo.value = 1;
    vo.value = null;
    // Not logged: $vo prints ValueNotifier<Object?> (Go can't spell the ?).
    final vp = ValueNotifier<P>(P(1, 2));
    vp.addListener(() => log('vp ${vp.value}'));
    vp.value = P(1, 3);
    vp.value = P(2, 3);
    log('$vp');
    final vb = ValueNotifier<bool>(false);
    log('$vb');
    final lv = LoggingValueNotifier(0);
    lv.addListener(() => log('lv listener ${lv.value}'));
    lv.value = 0;
    lv.value = 5;
    log('$lv');
    log('${ChangeNotifier()}');
    log('${TestNotifier()}');

    // Listenable.merge.
    log('--- merge');
    final s1 = TestNotifier();
    final s2 = TestNotifier();
    final vm = ValueNotifier<int>(0);
    final Listenable merged = Listenable.merge(<Listenable?>[s1, null, s2, vm, s1]);
    log('$merged');
    merged.addListener(a);
    s1.notify();
    s2.notify();
    vm.value = 1;
    log('$merged');
    merged.removeListener(a);
    log('s1=${s1.listened} s2=${s2.listened}');
    s1.notify();
    merged.addListener(a);
    s1.removeListener(a);
    s1.notify();
    log('${Listenable.merge(<Listenable?>[])} ${Listenable.merge(<Listenable?>[null])}');
    log('${Listenable.merge(<Listenable>[Listenable.merge(<Listenable>[s1]), ChangeNotifier()])}');

    // tear-offs: x.m == x.m.
    log('--- tear-offs');
    final h1 = Handler('h1');
    final h2 = Handler('h2');
    final tear = TestNotifier();
    tear.addListener(h1.handle);
    tear.addListener(h2.handle);
    tear.addListener(h1.handle);
    tear.notify();
    tear.removeListener(h1.handle);
    tear.notify();
    tear.removeListener(h2.handle);
    tear.removeListener(h1.handle);
    log('hasListeners=${tear.listened}');

    // The binding's onError: listener errors are reported, a disposed
    // notifier throws a FlutterError.
    log('--- binding');
    ListenBinding();
    final FlutterExceptionHandler? originalOnError = FlutterError.onError;
    FlutterError.onError = (FlutterErrorDetails details) {
      log('reported ${describe(details.exception)} library=${details.library} context=${details.context} stack=${details.stack != null}');
    };
    final bind = TestNotifier();
    bind.addListener(() => throw StateError('in listener'));
    bind.addListener(() => throw 'plain');
    bind.addListener(a);
    bind.notify();
    bind.addListener(() => bind.dispose());
    bind.notify();
    bind.dispose();
    for (final void Function() use in <void Function()>[
      () => bind.addListener(a),
      () => bind.notify(),
      () => bind.dispose(),
      () => ChangeNotifier.debugAssertNotDisposed(bind),
    ]) {
      try {
        use();
        log('no error');
      } catch (e) {
        log('caught ${describe(e)}');
      }
    }
    final ValueNotifier<int> vdisp = ValueNotifier<int>(3)..dispose();
    try {
      vdisp.value = 4;
    } catch (e) {
      log('caught ${describe(e)}');
    }
    log('value=${vdisp.value}');
    FlutterError.onError = originalOnError;

    // Memory allocation events.
    log('--- memory allocations');
    final FlutterMemoryAllocations ma = FlutterMemoryAllocations.instance;
    void maListener(ObjectEvent event) {
      final Map<String, Object?> fields = event.toMap().values.single;
      log('${event.runtimeType} ${event.object.runtimeType} ${fields['libraryName']} ${fields['className']} ${fields['eventType']}');
    }

    ma.addListener(maListener);
    final m1 = TestNotifier();
    log('m1 created');
    m1.addListener(a);
    m1.addListener(b);
    final m2 = ValueNotifier<int>(1);
    final m3 = ChangeNotifier();
    final m4 = TestNotifier();
    final m5 = LoggingValueNotifier(2);
    log('all created');
    m1.dispose();
    m2.dispose();
    m3.dispose();
    m4.dispose();
    m5.dispose();
    ma.removeListener(maListener);

    File('build/listen.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': entries}));
  });
}
