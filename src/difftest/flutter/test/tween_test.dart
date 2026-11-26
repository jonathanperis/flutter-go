// Reference side of flutter-go's tween differential test: every Tween type
// (Tween<double>, Tween<Offset>, Tween<Size> through the dynamic operators,
// ColorTween, SizeTween, RectTween with nulls, IntTween, StepTween,
// ReverseTween, ConstantTween, CurveTween), chains, Animatable.fromCallback,
// tween.animate, TweenSequence/FlippedTweenSequence, AnimationStyle.lerp,
// evaluated at many t, plus every toString and the lerp error messages.
// Doubles use Dart's toString (shortest round-trip form): the Go twin
// (../tween_test.go) is compared bit for bit.
// Output: build/tween.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/animation.dart';
import 'dart:ui' show ColorSpace;

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> lines = <String>[];
void log(String s) => lines.add(s);

String mask(Object? o) => '$o'.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH');

String col(Color? c) => c == null ? 'null' : '${c.a} ${c.r} ${c.g} ${c.b} ${c.colorSpace.name}';
String sz(Size? s) => s == null ? 'null' : '${s.width} ${s.height}';
String rc(Rect? r) => r == null ? 'null' : '${r.left} ${r.top} ${r.right} ${r.bottom}';
String off(Offset o) => '${o.dx} ${o.dy}';

final List<double> ts = <double>[
  0.0, 1.0, 0.5, 0.25, 0.75, 0.1, 0.2, 0.3, 0.7, 0.9, 1 / 3, 2 / 3, 0.123456789, 0.999999, 1e-9, 0.4999999999,
  -0.5, 1.5, -2.0, 3.25, for (int i = 1; i < 40; i++) i / 41,
];

final List<double> unit = ts.where((double t) => t >= 0 && t <= 1).toList();

String errorOf(void Function() f) {
  try {
    f();
    return 'no error';
  } on FlutterError catch (e) {
    return e.diagnostics.map((DiagnosticsNode n) => n.toString()).join(' | ');
  } catch (e) {
    return '${e.runtimeType}: $e';
  }
}

void main() {
  test('tween', () {
    final doubles = <Tween<double>>[
      Tween<double>(begin: 0.3, end: 0.5),
      Tween<double>(begin: -100, end: 250.75),
      Tween<double>(begin: 1e10, end: -3e-7),
      Tween<double>(begin: 0.1, end: 0.1),
      Tween<double>(begin: 5, end: double.infinity),
    ];
    for (final Tween<double> tw in doubles) {
      log('$tw');
      for (final double t in ts) {
        log('  $t ${tw.transform(t)} ${tw.lerp(t)}');
      }
    }
    final offsets = Tween<Offset>(begin: const Offset(1.5, -2), end: const Offset(-30.25, 7.125));
    log('$offsets');
    for (final double t in ts) {
      log('  $t ${off(offsets.transform(t))}');
    }
    final sizes = Tween<Size>(begin: const Size(10, 20), end: const Size(3.5, 99.25));
    log('$sizes');
    for (final double t in ts) {
      log('  $t ${sz(sizes.transform(t))}');
    }
    final colors = <ColorTween>[
      ColorTween(begin: const Color(0xff000000), end: const Color(0xffffffff)),
      ColorTween(begin: const Color(0x802196f3), end: const Color(0xff4caf50)),
      ColorTween(begin: const Color(0xff2196f3)),
      ColorTween(end: const Color(0x40ff0000)),
      ColorTween(),
      ColorTween(begin: const Color.from(alpha: 1, red: 0.1, green: 0.2, blue: 0.3, colorSpace: ColorSpace.displayP3), end: const Color(0xffff8800)),
    ];
    for (final ColorTween tw in colors) {
      log('$tw');
      for (final double t in unit) {
        log('  $t ${col(tw.transform(t))}');
      }
    }
    final sizeTweens = <SizeTween>[
      SizeTween(begin: Size.zero, end: const Size(20, 30)),
      SizeTween(begin: const Size(1.25, 7), end: const Size(-3, 1e5)),
      SizeTween(begin: const Size(4, 8)),
      SizeTween(end: const Size(4, 8)),
    ];
    for (final SizeTween tw in sizeTweens) {
      log('$tw');
      for (final double t in ts) {
        log('  $t ${sz(tw.transform(t))}');
      }
    }
    final rectTweens = <RectTween>[
      RectTween(begin: const Rect.fromLTWH(5, 3, 7, 11), end: const Rect.fromLTWH(8, 12, 14, 18)),
      RectTween(begin: const Rect.fromLTRB(-1.5, 0, 2.25, 100), end: Rect.zero),
      RectTween(begin: const Rect.fromLTWH(10, 10, 10, 10)),
      RectTween(end: const Rect.fromLTWH(10, 10, 10, 10)),
    ];
    for (final RectTween tw in rectTweens) {
      log('$tw');
      for (final double t in ts) {
        log('  $t ${rc(tw.transform(t))}');
      }
    }
    final ints = <Tween<int>>[
      IntTween(begin: 5, end: 9),
      IntTween(begin: -7, end: 12),
      IntTween(begin: 100, end: -100),
      StepTween(begin: 5, end: 9),
      StepTween(begin: -7, end: 12),
      StepTween(begin: 100, end: -100),
      ReverseTween<int>(IntTween(begin: 5, end: 9)),
      ReverseTween<int>(StepTween(begin: -3, end: 4)),
      ConstantTween<int>(42),
    ];
    for (final Tween<int> tw in ints) {
      log('$tw');
      for (final double t in ts) {
        log('  $t ${tw.transform(t)}');
      }
    }
    final reverseDouble = ReverseTween<double>(Tween<double>(begin: 0.25, end: 4));
    log('$reverseDouble ${reverseDouble.begin} ${reverseDouble.end}');
    for (final double t in ts) {
      log('  $t ${reverseDouble.transform(t)}');
    }
    final constant = ConstantTween<double>(100.0);
    log('$constant ${constant.transform(0.3)}');
    final curves = <CurveTween>[
      CurveTween(curve: Curves.easeIn),
      CurveTween(curve: Curves.elasticInOut),
      CurveTween(curve: const Interval(0.25, 0.75, curve: Curves.bounceOut)),
      CurveTween(curve: const SawTooth(4)),
    ];
    for (final CurveTween tw in curves) {
      log('$tw');
      for (final double t in unit) {
        log('  $t ${tw.transform(t)}');
      }
    }
    final Animatable<double> chain = Tween<double>(begin: 10, end: 20).chain(CurveTween(curve: Curves.easeOutCubic));
    final Animatable<Offset> chain2 = Tween<Offset>(begin: Offset.zero, end: const Offset(5, 9))
        .chain(Tween<double>(begin: 0.2, end: 0.6).chain(CurveTween(curve: Curves.fastOutSlowIn)));
    log('$chain');
    log('$chain2');
    for (final double t in unit) {
      log('  $t ${chain.transform(t)} ${off(chain2.transform(t))}');
    }
    final Animatable<Offset> callback = Animatable<Offset>.fromCallback((double v) => Offset(v * 3, -v));
    log('callback ${off(callback.transform(0.25))} ${callback.runtimeType}');
    log('callback $callback');

    final controller = AnimationController(vsync: const TestVSync(), value: 0.37);
    final Animation<double> animated = Tween<double>(begin: 0.3, end: 0.5).animate(controller);
    final Animation<Size?> driven = controller.drive(SizeTween(begin: const Size(1, 2), end: const Size(3, 5)).chain(CurveTween(curve: Curves.ease)));
    log('animated ${mask(animated)} | ${animated.toStringDetails()} | ${animated.value} ${animated.status.name}');
    log('driven ${sz(driven.value)} ${driven.status.name}');
    for (final double v in unit) {
      controller.value = v;
      log('  $v ${animated.value} ${sz(driven.value)} ${chain.evaluate(controller)}');
    }
    controller.dispose();

    final seqs = <TweenSequence<double>>[
      TweenSequence<double>(<TweenSequenceItem<double>>[
        TweenSequenceItem<double>(tween: Tween<double>(begin: 5.0, end: 10.0), weight: 4.0),
        TweenSequenceItem<double>(tween: ConstantTween<double>(10.0), weight: 2.0),
        TweenSequenceItem<double>(tween: Tween<double>(begin: 10.0, end: 5.0), weight: 4.0),
      ]),
      TweenSequence<double>(<TweenSequenceItem<double>>[
        TweenSequenceItem<double>(tween: Tween<double>(begin: 0, end: 1).chain(CurveTween(curve: Curves.easeIn)), weight: 0.7),
        TweenSequenceItem<double>(tween: Tween<double>(begin: 1, end: -1), weight: 1e-3),
        TweenSequenceItem<double>(tween: Tween<double>(begin: -1, end: 0.5), weight: 3),
        TweenSequenceItem<double>(tween: CurveTween(curve: Curves.bounceIn), weight: 1 / 3),
      ]),
      FlippedTweenSequence(<TweenSequenceItem<double>>[
        TweenSequenceItem<double>(tween: Tween<double>(begin: 0, end: 0.3), weight: 1),
        TweenSequenceItem<double>(tween: Tween<double>(begin: 0.3, end: 1).chain(CurveTween(curve: Curves.decelerate)), weight: 2.5),
      ]),
    ];
    for (final TweenSequence<double> s in seqs) {
      log('$s');
      for (final double t in unit) {
        log('  $t ${s.transform(t)}');
      }
    }
    final colorSeq = TweenSequence<Color?>(<TweenSequenceItem<Color?>>[
      TweenSequenceItem<Color?>(tween: ColorTween(begin: const Color(0xffff0000), end: const Color(0xff00ff00)), weight: 1),
      TweenSequenceItem<Color?>(tween: ColorTween(begin: const Color(0xff00ff00), end: const Color(0x000000ff)), weight: 2),
    ]);
    for (final double t in unit) {
      log('  color $t ${col(colorSeq.transform(t))}');
    }

    const a = AnimationStyle(curve: Curves.ease, duration: Duration(seconds: 1), reverseCurve: Curves.bounceIn, reverseDuration: Duration(milliseconds: 333));
    const b = AnimationStyle(curve: Curves.linear, duration: Duration(microseconds: 2000001));
    for (final double t in <double>[0, 0.1, 0.25, 1 / 3, 0.5, 0.77, 1, 1.5]) {
      final AnimationStyle s = AnimationStyle.lerp(a, b, t)!;
      log('style $t ${s.duration?.inMicroseconds} ${s.reverseDuration?.inMicroseconds} ${s.curve} ${s.reverseCurve}');
      for (final double u in <double>[0.1, 0.5, 0.9]) {
        log('  $u ${s.curve?.transform(u)} ${s.reverseCurve?.transform(u)}');
      }
    }
    log('style null ${AnimationStyle.lerp(null, b, 0.5)!.duration?.inMicroseconds} ${AnimationStyle.lerp(a, null, 0.25)!.curve}');
    final c1 = AnimationStyle(curve: Curves.ease, duration: const Duration(seconds: 1)); // not identical
    final c2 = AnimationStyle(curve: Curves.ease, duration: const Duration(seconds: 1)); // not identical
    final AnimationStyle same = AnimationStyle.lerp(c1, c2, 0.4)!;
    log('style same ${same.curve} ${same.duration?.inMicroseconds} ${same.reverseCurve} ${same.reverseDuration?.inMicroseconds}');
    final AnimationStyle other = AnimationStyle.lerp(AnimationStyle.lerp(a, b, 0.5), AnimationStyle.lerp(const AnimationStyle(curve: Curves.easeIn), b, 0.5), 0.3)!;
    log('style other first ${other.curve}');
    for (final List<double> p in <List<double>>[<double>[0.5, 0.5], <double>[0.5, 0.25]]) {
      final AnimationStyle l = AnimationStyle.lerp(AnimationStyle.lerp(a, b, p[0]), AnimationStyle.lerp(a, b, p[1]), 0.3)!;
      log('style lerped ${p[0]} ${p[1]} ${l.curve} ${l.reverseCurve} ${AnimationStyle.lerp(a, b, p[0]) == AnimationStyle.lerp(a, b, p[1])}');
    }

    log(errorOf(() => Tween<Color>(begin: const Color(0xff000000), end: const Color(0xffffffff)).transform(0.1)));
    log(errorOf(() => Tween<Rect>(begin: const Rect.fromLTWH(0, 0, 10, 10), end: const Rect.fromLTWH(2, 2, 2, 2)).transform(0.1)));
    log(errorOf(() => Tween<int>(begin: 0, end: 1).transform(0.1)));
    log(errorOf(() => Tween<int>(begin: 0, end: 1).transform(1.0)));
  });

  tearDownAll(() {
    File('build/tween.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': lines}));
  });
}
