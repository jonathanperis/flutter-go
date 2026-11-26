// Reference side of flutter-go's FlutterMemoryAllocations and
// FlutterTimeline differential test: the sequence of listener calls,
// hasListeners and dart:ui hook states while listeners are added and
// removed (also during dispatch), the events' toMap, the errors reported
// for failing listeners, and the aggregation of timed blocks (built from
// fixed timestamps, since real ones are not reproducible). Output:
// build/memory_allocations.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> entries = <String>[];
void log(String s) => entries.add(s);

String sortedJson(Map<String, Object> m) {
  final List<String> keys = m.keys.toList()..sort();
  return json.encode(<String, Object>{for (final String k in keys) k: m[k]!});
}

void main() {
  test('memory_allocations', () {
    final FlutterMemoryAllocations ma = FlutterMemoryAllocations.instance;
    void hooks(String when) => log(
      '$when: hasListeners=${ma.hasListeners} hooks=${ui.Image.onCreate != null},'
      '${ui.Image.onDispose != null},${ui.Picture.onCreate != null},${ui.Picture.onDispose != null}',
    );
    hooks('start');
    ma.dispatchObjectCreated(library: 'l', className: 'C', object: 'nobody');

    void a(ObjectEvent e) => log('a ${e.runtimeType} ${e.object}');
    void b(ObjectEvent e) => log('b ${e.runtimeType} ${e.object}');
    late void Function(ObjectEvent) c;
    c = (ObjectEvent e) {
      log('c ${e.object}: removes b and c, adds a');
      ma.removeListener(b);
      ma.removeListener(c);
      ma.addListener(a);
      log('c sees hasListeners=${ma.hasListeners}');
    };
    ma.addListener(a);
    hooks('a added');
    ma.addListener(b);
    ma.addListener(c);
    ma.addListener(b);
    ma.dispatchObjectCreated(library: 'package:x/y.dart', className: 'Thing', object: 'o1');
    hooks('after dispatch 1');
    ma.dispatchObjectDisposed(object: 'o2');
    ma.removeListener(a);
    hooks('a removed');

    for (final ObjectEvent e in <ObjectEvent>[
      ObjectCreated(library: 'lib', className: 'Cls', object: 'obj'),
      ObjectDisposed(object: 'obj'),
    ]) {
      final Map<Object, Map<String, Object>> m = e.toMap();
      log('toMap ${m.keys.single} ${sortedJson(m.values.single)}');
    }

    final FlutterExceptionHandler? onError = FlutterError.onError;
    FlutterError.onError = (FlutterErrorDetails details) {
      log('reported ${details.exception} library=${details.library} context=${details.context}');
      for (final DiagnosticsNode node in details.informationCollector!()) {
        log('  info $node');
      }
    };
    void bad(ObjectEvent e) => throw StateError('bad listener');
    ma.addListener(bad);
    ma.addListener(a);
    ma.dispatchObjectDisposed(object: 42);
    ma.dispatchObjectDisposed(object: 1.5);
    ma.removeListener(bad);
    ma.removeListener(a);
    FlutterError.onError = onError;
    hooks('end');

    // Flutter's "does not invoke concurrently removed listeners" scenario.
    void l2(ObjectEvent e) => log('l2');
    void l1(ObjectEvent e) {
      log('l1');
      ma.removeListener(l2);
      log('l1 sees hasListeners=${ma.hasListeners}');
    }
    ma.addListener(l1);
    ma.addListener(l2);
    ma.dispatchObjectDisposed(object: 'x');
    ma.removeListener(l1);
    hooks('after l1');

    // Timeline aggregation of fixed blocks.
    final AggregatedTimings timings = AggregatedTimings(<TimedBlock>[
      const TimedBlock(name: 'b', start: 0.1, end: 0.3),
      const TimedBlock(name: 'a', start: 1, end: 1.1),
      const TimedBlock(name: 'b', start: 0.2, end: 0.7),
      const TimedBlock(name: 'c', start: 5, end: 5),
      const TimedBlock(name: 'a', start: 1e15, end: 1e15 + 0.3),
      const TimedBlock(name: 'b', start: 3, end: 3.3),
    ]);
    for (final TimedBlock block in timings.timedBlocks) {
      log('$block');
    }
    for (final AggregatedTimedBlock block in timings.aggregatedBlocks) {
      log('$block');
    }
    log('${timings.getAggregated('missing')}');
    log('${timings.getAggregated('b')}');

    File('build/memory_allocations.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': entries}));
  });
}
