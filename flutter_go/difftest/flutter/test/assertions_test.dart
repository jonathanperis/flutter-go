// Reference side of flutter-go's assertions/stack_frame/print differential
// test: FlutterError and FlutterErrorDetails reports (toString, dumped
// console text, summaries, properties, JSON), StackFrame parsing of
// Dart-format traces, the default stack filter and RepetitiveStackFrameFilter,
// DiagnosticsStackTrace, debugPrintStack, debugWordWrap and the
// debugPrintThrottled schedule.
// Output: build/assertions.json as {"<scene>": <string or JSON>}.
// The Go twin is difftest/assertions_test.go; keep the two in step.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:fake_async/fake_async.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

class MyException implements Exception {
  MyException(this.text);
  final String text;
  @override
  String toString() => text;
}

class MyError extends Error {
  MyError(this.text);
  final String text;
  @override
  String toString() => text;
}

class Foo {
  Foo(this.n);
  final int n;
  @override
  String toString() => 'Foo($n)';
}

/// A Diagnosticable used as an expandable property value.
class Widgetish with Diagnosticable {
  Widgetish(this.name);
  final String name;
  @override
  String toStringShort() => 'Widgetish($name)';
  @override
  void debugFillProperties(DiagnosticPropertiesBuilder properties) {
    super.debugFillProperties(properties);
    properties.add(StringProperty('name', name));
    properties.add(DoubleProperty('width', 12.5));
    properties.add(FlagProperty('visible', value: true, ifTrue: 'visible'));
  }
}

Object realAssert(String? message) {
  try {
    assert(false, message);
  } catch (e) {
    return e;
  }
  throw 'asserts are disabled';
}

const String flutterStack = '''
#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)
#1      _AssertionError._throwNew (dart:core-patch/errors_patch.dart:38:5)
#2      new Text (package:flutter/src/widgets/text.dart:287:10)
#3      new SomeWidgetUsingText (package:flutter/src/widgets/text_helper.dart:287:10)
#4      _MyHomePageState.build (package:hello_flutter/main.dart:72:16)
#5      StatefulElement.build (package:flutter/src/widgets/framework.dart:4414:27)
#6      ComponentElement.performRebuild (package:flutter/src/widgets/framework.dart:4303:15)
#7      Element.rebuild (package:flutter/src/widgets/framework.dart:4027:5)
#8      ComponentElement._firstBuild (package:flutter/src/widgets/framework.dart:4286:5)
#9      Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)
#10     Element.updateChild (package:flutter/src/widgets/framework.dart:3070:12)
#11     Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)
#12     Element.updateChild (package:flutter/src/widgets/framework.dart:3070:12)
#13     _rootRun (dart:async/zone.dart:1126:13)
#14     _CustomZone.run (dart:async/zone.dart:1023:19)
#15     _FakeAsync.run (package:fake_async/fake_async.dart:177:54)
#16     Chain.capture (package:stack_trace/src/chain.dart:94:22)
#17     _asyncThenWrapperHelper (dart:async-patch/async_patch.dart:83:23)
#18     _Timer._runTimers (dart:isolate-patch/timer_impl.dart:398:19)
#19     _FrameCallbackEntry.run (package:flutter/src/scheduler/binding.dart:1:1)
<asynchronous suspension>
#20     main (file:///app/main.dart:10:4)
#21     _RawReceivePortImpl._handleMessage (dart:isolate-patch/isolate_patch.dart:174:12)
...
#22     Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)
#23     Element.updateChild (package:flutter/src/widgets/framework.dart:3070:12)
#24     tail (package:app/tail.dart:1:1)''';

const String userStack = '''
#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)
#1      _AssertionError._throwNew (dart:core-patch/errors_patch.dart:38:5)
#2      new Text (package:flutter/src/widgets/text.dart:287:10)
#3      _MyHomePageState.build (package:hello_flutter/main.dart:72:16)
#4      StatefulElement.build (package:flutter/src/widgets/framework.dart:4414:27)''';

/// Lines exercising every branch of StackFrame.fromStackTraceLine.
const List<String> frameLines = <String>[
  '#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)',
  '#1      new Text (package:flutter/src/widgets/text.dart:287:10)',
  '#2      new Text.rich (package:flutter/src/widgets/text.dart:300:1)',
  '#3      new (http://localhost:42191/dart-sdk/lib/async/stream_controller.dart:880:9)',
  '#4      main.<anonymous closure>.<anonymous closure> (file:///temp/path.whatever/listener.dart:47:18)',
  '#5      A.b.c.d (package:x/y/z.dart:1:2)',
  '#6      blah (package:assertions/main.dart:4)',
  '#7      noLine (package:assertions/main.dart)',
  '#8      dots (dart:ui/../ui/./hooks.dart:9:9)',
  '#9      spaces (file:///my%20dir/a%2fb.dart:3:4)',
  '#10     weird (file:///a/b c.dart:3:4)',
  '#11     relative (a/../../b.dart:1:1)',
  '#12     upper (PACKAGE:Foo/Bar.dart:1:1)',
  '#123456 big (package:p/q.dart:12345:67890)',
  '#0      someMethod  (file:///diagnostics_test.dart:42:19)',
  '<asynchronous suspension>',
  '===== asynchronous gap ===========================',
  '...',
  'package:dart-sdk/lib/_internal/js_dev_runtime/private/ddc_runtime/errors.dart 196:49  throw_',
  'package:assertions/main.dart 8:5                                                      main\$',
  'object_test.dart 210:5                                                                performLayout',
  r'test\bindings_async_gap_test.dart 42:17                        main.<fn>.<fn>',
  'not a frame at all',
  '    at Foo.bar (http://x/main.dart.js:1:2)',
  '#13     paren (file:///dir (copy)/x.dart:1:2)',
  '#14     new A.b.c (package:x/y.dart:1:1)',
  'a\u2028b.dart 1:2 lineSeparator',
  'nbsp.dart 1:2\u00a0method',
];

const String loremIpsum =
    'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore '
    'et dolore magna aliqua.  Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut '
    'aliquip ex ea commodo consequat. Supercalifragilisticexpialidocious-is-an-unbreakable-word-here. '
    'Ünïcödé àccents, emoji 😀😀 and CJK 漢字漢字 too.';

const List<String> wrapTexts = <String>[
  loremIpsum,
  '  - a list item whose text goes on for quite a while so that it has to wrap onto further lines',
  '12. a numbered item whose text goes on for quite a while so that it has to wrap onto lines',
  '* bullet     with     many     spaces     between     the     words     of     this     line',
  '   #0      main (file:///a.dart:1:2) frames are never wrapped even when they are long',
  'short',
  'exactly twenty chars',
  '    leading spaces then words that need wrapping because the width is small',
  'trailing spaces after words that need wrapping because the width is small      ',
  '😀😀😀 😀😀😀 😀😀😀 😀😀😀 😀😀😀',
  'a b c d e f g h i j k l m n o p q r s t u v w x y z',
  'anunbreakablewordlongerthananywidthwetryhere and then short words',
];

const List<int> wrapWidths = <int>[0, 1, 3, 7, 10, 15, 20, 21, 40, 80];

FlutterError richError() => FlutterError.fromParts(<DiagnosticsNode>[
  ErrorSummary('Something is wrong with the widget configuration.'),
  ErrorDescription(
    'This description is long enough that the error renderer has to wrap it onto several lines '
    'when it is shown at the default wrap width of a hundred characters.',
  ),
  ErrorSpacer(),
  DiagnosticsProperty<Widgetish>('The offending widget was', Widgetish('Inner'), style: DiagnosticsTreeStyle.errorProperty),
  DiagnosticsProperty<Widgetish>('Expanded', Widgetish('Exp'), expandableValue: true, style: DiagnosticsTreeStyle.whitespace),
  ErrorHint('Try wrapping the widget in something else.\nOr pass a different argument.'),
  ErrorSpacer(),
  ErrorDescription('A final note.'),
  DiagnosticsBlock(
    name: 'Block of details',
    style: DiagnosticsTreeStyle.errorProperty,
    properties: <DiagnosticsNode>[IntProperty('count', 3), StringProperty('label', 'abc')],
  ),
  DiagnosticsStackTrace('Stack of the creation', StackTrace.fromString(userStack)),
]);

Map<String, Object Function()> exceptionKinds() => <String, Object Function()>{
  'string': () => 'a thrown string',
  'multilineString': () => '  indented first line\nsecond line\n\n',
  'int': () => 42,
  'double': () => 2.5,
  'doubleWhole': () => 2.0,
  'bool': () => true,
  'assertion': () => AssertionError('Test assertion'),
  'assertionNoMessage': () => AssertionError(),
  'assertionNumber': () => AssertionError(7),
  'assertionQuoted': () => AssertionError('quote " and \\ and\nnewline\ttab'),
  'assertionFlutterError': () => AssertionError(FlutterError('Wrapped summary.\nWrapped description.')),
  'flutterError': () => FlutterError('Plain summary.\nFirst description.\nSecond description.'),
  'richFlutterError': richError,
  'myException': () => MyException('MyException: went wrong'),
  'myExceptionNoPrefix': () => MyException('went wrong without prefix'),
  'myError': () => MyError('Custom error text\nwith a second line'),
  'emptyError': () => MyError(''),
  'foo': () => Foo(1),
  'realAssert': () => realAssert('Message goes here.'),
  'realAssertNoMessage': () => realAssert(null),
  'realAssertLong': () => realAssert('word ' * 30),
};

List<String> captureDebugPrint(void Function() fn) {
  final log = <String>[];
  final DebugPrintCallback saved = debugPrint;
  debugPrint = (String? message, {int? wrapWidth}) {
    log.add('$message');
  };
  try {
    fn();
  } finally {
    debugPrint = saved;
  }
  return log;
}

List<String> capturePrint(void Function() fn) {
  final log = <String>[];
  runZoned<void>(
    fn,
    zoneSpecification: ZoneSpecification(
      print: (Zone self, ZoneDelegate parent, Zone zone, String line) {
        log.add(line);
      },
    ),
  );
  return log;
}

Map<String, Object?> frameJson(StackFrame f) => <String, Object?>{
  'number': f.number,
  'packageScheme': f.packageScheme,
  'package': f.package,
  'packagePath': f.packagePath,
  'line': f.line,
  'column': f.column,
  'className': f.className,
  'method': f.method,
  'isConstructor': f.isConstructor,
  'source': f.source,
  'toString': f.toString(),
};

List<String> propertyStrings(FlutterErrorDetails details) {
  final builder = DiagnosticPropertiesBuilder();
  details.debugFillProperties(builder);
  return <String>[
    for (final DiagnosticsNode p in builder.properties)
      '${p.runtimeType} ${p.level.name} ${p.style?.name}: ${p.toString()}',
  ];
}

void main() {
  test('assertions', () {
    final out = <String, Object?>{};

    // The throttle first, while its state is fresh (Go's is too).
    FakeAsync().run((FakeAsync async) {
      final events = <String>[];
      runZoned<void>(
        () {
          void log(String what) => events.add('${async.elapsed.inMilliseconds}ms $what');
          debugPrintDone.then((_) => log('done (idle)'));
          async.flushMicrotasks();
          debugPrintThrottled('${'a' * 5000}\n${'b' * 5000}\n${'c' * 5000}\n${'d' * 3000}');
          debugPrintDone.then((_) => log('done 1'));
          debugPrintThrottled('${'e' * 20000}\nf\ng');
          debugPrintThrottled('h ' * 30, wrapWidth: 20);
          async.elapse(const Duration(milliseconds: 500));
          log('half a second');
          async.elapse(const Duration(milliseconds: 500));
          async.elapse(const Duration(seconds: 3));
          debugPrintDone.then((_) => log('done (idle again)'));
          async.flushMicrotasks();
          debugPrintThrottled('${'i' * 13000}\nj');
          async.elapse(const Duration(seconds: 2));
        },
        zoneSpecification: ZoneSpecification(
          print: (Zone self, ZoneDelegate parent, Zone zone, String line) {
            events.add('${async.elapsed.inMilliseconds}ms print ${line.length} ${line.substring(0, line.length < 3 ? line.length : 3)}');
          },
        ),
      );
      out['throttle.events'] = events;
    });

    // FlutterError.
    final FlutterError plain = FlutterError('Plain summary.\nFirst description.\n\nThird description.');
    out['flutterError.plain.toString'] = plain.toString();
    out['flutterError.plain.toStringDeep'] = plain.toStringDeep();
    out['flutterError.plain.toStringShort'] = plain.toStringShort();
    out['flutterError.plain.message'] = plain.message;
    final FlutterError rich = richError();
    out['flutterError.rich.toString'] = rich.toString();
    out['flutterError.rich.toStringDeep'] = rich.toStringDeep();
    out['flutterError.rich.toStringDeep.wrap40'] = rich.toStringDeep(wrapWidth: 40);
    out['flutterError.rich.toStringShallow'] = rich.toStringShallow(joiner: ' | ');
    out['flutterError.rich.diagnostics'] = <String>[
      for (final DiagnosticsNode d in rich.diagnostics) '${d.runtimeType} ${d.level.name}: $d',
    ];
    out['flutterError.rich.json'] = jsonDecode(jsonEncode(rich.toDiagnosticsNode().toJsonMap(
      const DiagnosticsSerializationDelegate(includeProperties: true, subtreeDepth: 1),
    )));
    out['errorSummary.json'] = jsonDecode(jsonEncode(ErrorSummary('A summary').toJsonMap(const DiagnosticsSerializationDelegate())));
    out['errorHint.json'] = jsonDecode(jsonEncode(ErrorHint('A hint').toJsonMap(const DiagnosticsSerializationDelegate())));
    out['errorSpacer.json'] = jsonDecode(jsonEncode(ErrorSpacer().toJsonMap(const DiagnosticsSerializationDelegate())));
    out['errorSpacer.toString'] = ErrorSpacer().toString();
    out['errorDescription.dense'] = ErrorDescription('dense desc').toString(parentConfiguration: denseTextConfiguration);

    // Malformed errors.
    final malformed = <String, List<DiagnosticsNode>>{
      'empty': <DiagnosticsNode>[],
      'noSummary': <DiagnosticsNode>[ErrorDescription('Error description without a summary')],
      'twoSummaries': <DiagnosticsNode>[
        ErrorSummary('Error Summary A'),
        ErrorDescription('Some descriptionA'),
        ErrorSummary('Error Summary B'),
        ErrorHint('A hint B'),
      ],
      'summaryLast': <DiagnosticsNode>[ErrorDescription('Some description'), ErrorSummary('Error summary')],
    };
    for (final MapEntry<String, List<DiagnosticsNode>> m in malformed.entries) {
      Object error;
      try {
        throw FlutterError.fromParts(m.value);
      } on AssertionError catch (e) {
        error = e;
      }
      out['malformed.${m.key}.details'] = FlutterErrorDetails(exception: error).toString();
      out['malformed.${m.key}.summary'] = FlutterErrorDetails(exception: error).summary.toString();
    }

    // Exception kinds.
    for (final MapEntry<String, Object Function()> k in exceptionKinds().entries) {
      final Object e = k.value();
      if (k.key.startsWith('realAssert')) {
        out['exception.${k.key}.raw'] = e.toString();
      }
      final details = FlutterErrorDetails(exception: e);
      out['exception.${k.key}.toString'] = details.toString();
      out['exception.${k.key}.summary'] = details.summary.toString();
      out['exception.${k.key}.exceptionAsString'] = details.exceptionAsString();
      out['exception.${k.key}.properties'] = propertyStrings(details);
      out['exception.${k.key}.dump'] = captureDebugPrint(() {
        FlutterError.resetErrorCount();
        FlutterError.dumpErrorToConsole(details);
        FlutterError.dumpErrorToConsole(details);
      });
      final withStack = FlutterErrorDetails(
        exception: e,
        stack: StackTrace.fromString(flutterStack),
        library: 'kinds library',
        context: ErrorDescription('while testing ${k.key}'),
      );
      out['exception.${k.key}.withStack'] = withStack.toString();
      out['exception.${k.key}.withStack.properties'] = propertyStrings(withStack);
    }
    FlutterError.resetErrorCount();

    // Full reports.
    final full = FlutterErrorDetails(
      exception: AssertionError('A failed assertion message that is fairly long, long enough to wrap when it is dumped.'),
      stack: StackTrace.fromString(flutterStack),
      library: 'widgets library',
      context: ErrorDescription('while building MyWidget(dirty, state: _MyWidgetState#12345)'),
      informationCollector: () => <DiagnosticsNode>[
        DiagnosticsProperty<Widgetish>('The relevant error-causing widget was', Widgetish('Relevant')),
        ErrorHint('Line one of a hint.\nLine two of a hint.'),
        ErrorDescription('word ' * 40),
        DiagnosticsStackTrace.singleFrame('Created at', frame: '#0      creator (package:app/creator.dart:1:2)'),
        DiagnosticsProperty<Widgetish>('Long', Widgetish('a ' * 80), expandableValue: true, style: DiagnosticsTreeStyle.whitespace),
        DiagnosticsBlock(
          name: 'Many children',
          allowTruncate: true,
          children: <DiagnosticsNode>[for (var i = 0; i < 9; i++) DiagnosticsNode.message('child $i')],
        ),
      ],
    );
    out['full.toString'] = full.toString();
    out['full.toString.debug'] = full.toString(minLevel: DiagnosticLevel.debug);
    out['full.toStringShort'] = full.toStringShort();
    out['full.summary'] = full.summary.toString();
    out['full.dump'] = captureDebugPrint(() {
      FlutterError.resetErrorCount();
      FlutterError.dumpErrorToConsole(full);
      FlutterError.dumpErrorToConsole(full);
      FlutterError.dumpErrorToConsole(full, forceReport: true);
      FlutterError.dumpErrorToConsole(FlutterErrorDetails(exception: richError()));
      FlutterError.resetErrorCount();
      FlutterError.dumpErrorToConsole(FlutterErrorDetails(exception: richError(), library: null, silent: true));
    });
    out['full.json'] = jsonDecode(jsonEncode(full.toDiagnosticsNode().toJsonMap(
      const DiagnosticsSerializationDelegate(includeProperties: true),
    )));
    out['full.properties'] = propertyStrings(full);
    final userFault = FlutterErrorDetails(exception: AssertionError('user'), stack: StackTrace.fromString(userStack));
    out['userFault.toString'] = userFault.toString();
    out['userFault.properties'] = propertyStrings(userFault);
    out['customFilter.toString'] = FlutterErrorDetails(
      exception: 'filtered',
      stack: StackTrace.fromString(flutterStack),
      stackFilter: (Iterable<String> lines) => lines.where((String l) => l.contains('Element')).map((String l) => '> $l'),
    ).toString();
    out['emptyStack.toString'] = FlutterErrorDetails(exception: 'x', stack: StackTrace.fromString('')).toString();
    final summaryFirst = FlutterErrorDetails(
      exception: FlutterError('The real summary.\nIts description.'),
      informationCollector: () => <DiagnosticsNode>[ErrorSummary('A later summary.')],
    );
    out['summaryFirst.summary'] = summaryFirst.summary.toString();
    final copied = full.copyWith(exception: 'copied', library: 'copy lib');
    out['copyWith.toString'] = copied.toString();

    FlutterErrorDetails.propertiesTransformers.add(
      (Iterable<DiagnosticsNode> properties) => <DiagnosticsNode>[
        ErrorDescription('(transformed)'),
        ...properties.where((DiagnosticsNode p) => p is! ErrorSpacer),
      ],
    );
    out['transformed.toString'] = full.toString();
    FlutterErrorDetails.propertiesTransformers.clear();

    // StackFrame parsing.
    out['frames.lines'] = <Object?>[
      for (final String line in frameLines)
        if (StackFrame.fromStackTraceLine(line) case final StackFrame f) frameJson(f) else null,
    ];
    out['frames.flutterStack'] = <Object?>[
      for (final StackFrame f in StackFrame.fromStackString(flutterStack)) frameJson(f),
    ];

    // Stack filters and DiagnosticsStackTrace.
    out['filter.flutterStack'] = FlutterError.defaultStackFilter(flutterStack.split('\n')).toList();
    out['filter.oneElided'] = FlutterError.defaultStackFilter(<String>[
      '#0      a (package:app/a.dart:1:1)',
      '#1      _rootRun (dart:async/zone.dart:1:1)',
    ]).toList();
    out['filter.twoKinds'] = FlutterError.defaultStackFilter(<String>[
      '#0      _rootRun (dart:async/zone.dart:1:1)',
      '#1      a (package:app/a.dart:1:1)',
      '#2      b (dart:async-patch/x.dart:1:1)',
      '#3      c (dart:async-patch/x.dart:1:1)',
    ]).toList();
    out['filter.nothing'] = FlutterError.defaultStackFilter(<String>['#0      a (package:app/a.dart:1:1)', 'not a frame', '']).toList();
    out['stackTrace.deep'] = DiagnosticsStackTrace('Stack trace', StackTrace.fromString(flutterStack)).toStringDeep();
    out['stackTrace.noSeparator'] = DiagnosticsStackTrace('-- callback --', StackTrace.fromString(userStack), showSeparator: false).toStringDeep();
    out['stackTrace.null'] = DiagnosticsStackTrace('Nothing', null).toStringDeep();
    out['stackTrace.singleFrame'] = DiagnosticsStackTrace.singleFrame('One', frame: '#0      f (package:a/b.dart:1:2)').toStringDeep();
    out['stackTrace.json'] = jsonDecode(jsonEncode(DiagnosticsStackTrace('S', StackTrace.fromString(userStack)).toJsonMap(
      const DiagnosticsSerializationDelegate(includeProperties: true),
    )));
    out['debugPrintStack'] = captureDebugPrint(() {
      debugPrintStack(stackTrace: StackTrace.fromString(flutterStack), label: 'The label', maxFrames: 15);
      debugPrintStack(stackTrace: StackTrace.fromString(userStack));
    });

    // Registered filters last: addDefaultStackFilter can't be undone.
    FlutterError.addDefaultStackFilter(
      const RepetitiveStackFrameFilter(
        frames: <PartialStackFrame>[
          PartialStackFrame(package: 'package:flutter/src/widgets/framework.dart', className: 'Element', method: 'inflateWidget'),
          PartialStackFrame(package: 'package:flutter/src/widgets/framework.dart', className: 'Element', method: 'updateChild'),
        ],
        replacement: '...     Normal element mounting',
      ),
    );
    FlutterError.addDefaultStackFilter(
      RepetitiveStackFrameFilter(
        frames: <PartialStackFrame>[
          PartialStackFrame(package: RegExp(r'^package:hello_'), className: '_MyHomePageState', method: 'build'),
        ],
        replacement: '...     my build',
      ),
    );
    FlutterError.addDefaultStackFilter(
      const RepetitiveStackFrameFilter(
        frames: <PartialStackFrame>[PartialStackFrame.asynchronousSuspension],
        replacement: '...     async gap',
      ),
    );
    FlutterError.addDefaultStackFilter(
      const RepetitiveStackFrameFilter(
        frames: <PartialStackFrame>[
          PartialStackFrame(package: 'package:ov/', className: 'Ov', method: 'a'),
          PartialStackFrame(package: 'package:ov/', className: 'Ov', method: 'b'),
          PartialStackFrame(package: 'package:ov/', className: 'Ov', method: 'a'),
        ],
        replacement: '...     overlapping',
      ),
    );
    out['repetitive.overlapping'] = FlutterError.defaultStackFilter(<String>[
      '#0      Ov.a (package:ov/o.dart:1:1)',
      '#1      Ov.b (package:ov/o.dart:2:1)',
      '#2      Ov.a (package:ov/o.dart:1:1)',
      '#3      Ov.b (package:ov/o.dart:2:1)',
      '#4      Ov.a (package:ov/o.dart:1:1)',
      '#5      Ov.b (package:ov/o.dart:2:1)',
      '#6      Ov.a (package:ov/o.dart:1:1)',
      '#7      tail (package:app/tail.dart:1:1)',
      '#8      tail (package:app/tail.dart:1:1)',
    ]).toList();
    out['repetitive.flutterStack'] = FlutterError.defaultStackFilter(flutterStack.split('\n')).toList();
    out['repetitive.details'] = FlutterErrorDetails(exception: 'with filters', stack: StackTrace.fromString(flutterStack)).toString();

    // debugWordWrap and debugPrint wrapping.
    for (var t = 0; t < wrapTexts.length; t++) {
      for (final int w in wrapWidths) {
        out['wrap.$t.$w'] = debugWordWrap(wrapTexts[t], w).toList();
      }
      out['wrap.$t.indent'] = debugWordWrap(wrapTexts[t], 25, wrapIndent: '>> ').toList();
    }
    // Flutter quirks: a prefix longer than the message, and a blank message
    // at least as long as the width, throw (RangeError).
    for (final (String text, int w, String indent) in <(String, int, String)>[('1. a', 0, '  '), ('   ', 3, ''), ('   ', 4, '')]) {
      try {
        out['wrap.throws.${jsonEncode(text)}.$w'] = debugWordWrap(text, w, wrapIndent: indent).toList();
      } on RangeError {
        out['wrap.throws.${jsonEncode(text)}.$w'] = 'throws';
      }
    }
    out['debugPrintSynchronously'] = capturePrint(() {
      debugPrintSynchronously(wrapTexts.join('\n'), wrapWidth: 30);
      debugPrintSynchronously(loremIpsum);
      debugPrintSynchronously('Hello,   world', wrapWidth: 0);
    });

    File('build/assertions.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
