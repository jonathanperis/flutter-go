// Reference side of flutter-go's AnimationController differential test.
// Every scene drives controllers (and the animations built on them) with
// tester.pump at irregular steps and logs, in order, every listener and
// status notification, the value/status/velocity/elapsed after each pump,
// TickerFuture completions and toString forms (hash codes masked). Doubles
// are logged with Dart's toString (shortest round-trip form), so the Go
// twin (../animation_controller_test.go) is compared bit for bit.
// Output: build/animation_controller.json as {scene: [lines]}.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/animation.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/physics.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_test/flutter_test.dart';

final Map<String, List<String>> out = <String, List<String>>{};
late List<String> lines;

void log(String s) => lines.add(s);

String mask(Object? o) => '$o'.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH');

String state(AnimationController c) =>
    'value=${c.value} status=${c.status.name} animating=${c.isAnimating} '
    'velocity=${c.velocity} elapsed=${c.lastElapsedDuration?.inMicroseconds}';

void watch(String name, AnimationController c) {
  c.addListener(() => log('$name tick ${c.value} ${c.status.name}'));
  c.addStatusListener((AnimationStatus s) => log('$name status ${s.name} value=${c.value}'));
}

void future(String name, TickerFuture f) {
  f.then((_) => log('$name future complete'));
  f.orCancel.then((_) {}, onError: (Object e) => log('$name orCancel ${e.runtimeType}'));
}

Future<void> steps(WidgetTester tester, AnimationController c, List<int> us) async {
  for (final int step in us) {
    await tester.pump(Duration(microseconds: step));
    log('pump $step ${state(c)}');
  }
}

class MutedVSync implements TickerProvider {
  const MutedVSync();
  @override
  Ticker createTicker(TickerCallback onTick) => Ticker(onTick)..muted = true;
}

const List<int> frame60 = <int>[0, 16667, 16666, 16667, 16667, 16666, 16667, 16667, 16666, 16667, 16667, 16666, 16667,
  16667, 16666, 16667, 16667, 16666, 16667, 16667, 16666, 16667, 16667];

void scene(String name, Future<void> Function(WidgetTester tester) body) {
  testWidgets(name, (WidgetTester tester) async {
    lines = <String>[];
    out[name] = lines;
    await body(tester);
  });
}

void main() {
  scene('forward_reverse', (WidgetTester tester) async {
    final c = AnimationController(
      duration: const Duration(milliseconds: 300),
      reverseDuration: const Duration(milliseconds: 200),
      vsync: const TestVSync(),
      debugLabel: 'fr',
    );
    watch('c', c);
    log('initial ${state(c)} ${mask(c)}');
    future('forward', c.forward());
    log('after forward ${state(c)} ${mask(c)}');
    await steps(tester, c, <int>[0, 16667, 16666, 33333, 50001, 99999]);
    log('mid ${mask(c)}');
    future('reverse', c.reverse());
    await steps(tester, c, <int>[0, 17000, 23456, 40000, 81000, 1, 2]);
    future('forward from', c.forward(from: 0.25));
    await steps(tester, c, <int>[0, 70000, 13]);
    future('toggle', c.toggle());
    await steps(tester, c, frame60);
    future('toggle from', c.toggle(from: 0.6));
    await steps(tester, c, frame60);
    future('reverse from', c.reverse(from: 0.9));
    await steps(tester, c, <int>[0, 100000, 100000, 100000]);
    log('end ${mask(c)}');
    c.dispose();
    log('disposed ${mask(c)}');
  });

  scene('animate_to_curves', (WidgetTester tester) async {
    final c = AnimationController(duration: const Duration(milliseconds: 400), vsync: const TestVSync());
    watch('c', c);
    future('a', c.animateTo(0.7, curve: Curves.easeInOut));
    await steps(tester, c, frame60);
    future('b', c.animateTo(0.2, duration: const Duration(milliseconds: 250), curve: Curves.fastOutSlowIn));
    await steps(tester, c, frame60);
    future('c', c.animateTo(0.9, curve: Curves.bounceOut));
    await steps(tester, c, <int>[0, 5000, 50000, 77777, 100000, 200000, 300000]);
    future('d', c.animateBack(0.1, curve: Curves.elasticIn));
    await steps(tester, c, frame60);
    future('e', c.animateBack(0.6, duration: const Duration(milliseconds: 123), curve: const Interval(0.2, 0.8, curve: Curves.easeIn)));
    await steps(tester, c, <int>[0, 30000, 30000, 30000, 30000, 30000]);
    future('same', c.animateTo(c.value, duration: const Duration(milliseconds: 100)));
    log('same ${state(c)}');
    future('zero', c.animateTo(0.33, duration: Duration.zero));
    log('zero ${state(c)} transient=${SchedulerBinding.instance.transientCallbackCount}');
    future('interrupted', c.animateTo(1.0, curve: Curves.easeOutBack));
    await steps(tester, c, <int>[0, 40000, 40000]);
    future('interrupter', c.animateTo(0.0, curve: Curves.easeInCirc));
    await steps(tester, c, frame60);
    await steps(tester, c, frame60);
    c.dispose();
  });

  scene('repeat', (WidgetTester tester) async {
    final c = AnimationController(duration: const Duration(milliseconds: 170), vsync: const TestVSync());
    watch('c', c);
    future('r1', c.repeat(reverse: true));
    await steps(tester, c, frame60);
    c.stop();
    log('stopped ${state(c)}');
    future('r2', c.repeat(min: 0.2, max: 0.8, count: 3, period: const Duration(milliseconds: 90)));
    await steps(tester, c, frame60);
    future('r3', c.repeat(reverse: true, count: 2, min: 0.1, max: 0.5));
    await steps(tester, c, frame60);
    c.value = 0.65;
    future('r4', c.repeat(reverse: true, count: 5, period: const Duration(microseconds: 33333)));
    await steps(tester, c, frame60);
    future('r5', c.repeat(min: 0.5, max: 0.5));
    await steps(tester, c, <int>[0, 10000, 10000]);
    c.reset();
    log('reset ${state(c)}');
    final big = AnimationController(duration: const Duration(milliseconds: 100), lowerBound: -3, upperBound: 7, value: 2, vsync: const TestVSync());
    watch('big', big);
    future('big', big.repeat(min: -1, max: 5, reverse: true, count: 3));
    await steps(tester, big, frame60);
    c.dispose();
    big.dispose();
  });

  scene('fling', (WidgetTester tester) async {
    final c = AnimationController(vsync: const TestVSync());
    watch('c', c);
    future('f1', c.fling());
    await steps(tester, c, frame60);
    future('f2', c.fling(velocity: -2.5));
    await steps(tester, c, frame60);
    future('f3', c.fling(velocity: 0.3, springDescription: SpringDescription.withDampingRatio(mass: 0.5, stiffness: 120.0, ratio: 1.4)));
    await steps(tester, c, frame60);
    await steps(tester, c, frame60);
    final large = AnimationController(lowerBound: -30, upperBound: 45, vsync: const TestVSync());
    watch('large', large);
    future('large', large.fling(velocity: 40));
    await steps(tester, large, frame60);
    future('large back', large.fling(velocity: -1));
    await steps(tester, large, frame60);
    await steps(tester, large, frame60);
    c.dispose();
    large.dispose();
  });

  scene('simulations', (WidgetTester tester) async {
    final c = AnimationController(lowerBound: -100, upperBound: 400, vsync: const TestVSync());
    watch('c', c);
    future('gravity', c.animateWith(GravitySimulation(500.0, 0.0, 300.0, 10.0)));
    await steps(tester, c, frame60);
    future('spring back', c.animateBackWith(SpringSimulation(const SpringDescription(mass: 1, stiffness: 180, damping: 12), 300, -20, -50)));
    await steps(tester, c, frame60);
    await steps(tester, c, frame60);
    future('friction', c.animateWith(FrictionSimulation(0.135, 100, 900)));
    await steps(tester, c, frame60);
    await steps(tester, c, <int>[1000000, 1000000]);
    final u = AnimationController.unbounded(value: 50, vsync: const TestVSync());
    watch('u', u);
    log('unbounded ${state(u)} behavior=${u.animationBehavior.name} ${mask(u)}');
    future('u to', u.animateTo(2000, duration: const Duration(milliseconds: 400), curve: Curves.easeOut));
    await steps(tester, u, frame60);
    future('u spring', u.animateWith(SpringSimulation(SpringDescription.withDampingRatio(mass: 0.5, stiffness: 100, ratio: 1.1), u.value, -200, 1500)));
    await steps(tester, u, frame60);
    await steps(tester, u, frame60);
    future('u back', u.animateBack(-1e6, duration: const Duration(milliseconds: 50)));
    await steps(tester, u, <int>[0, 20000, 20000, 20000]);
    c.dispose();
    u.dispose();
  });

  scene('stop_reset_resync', (WidgetTester tester) async {
    final c = AnimationController(duration: const Duration(milliseconds: 500), vsync: const TestVSync(), debugLabel: 'srr');
    watch('c', c);
    future('f', c.forward());
    await steps(tester, c, <int>[0, 50000, 50000]);
    c.stop(canceled: false);
    log('stop canceled false ${state(c)}');
    await steps(tester, c, <int>[50000]);
    future('g', c.forward());
    await steps(tester, c, <int>[0, 50000]);
    c.value = 0.8;
    log('set value ${state(c)}');
    c.value = 2.0;
    log('set value clamped ${state(c)}');
    future('h', c.reverse());
    await steps(tester, c, <int>[0, 60000]);
    c.resync(const TestVSync());
    log('resync ${state(c)}');
    await steps(tester, c, <int>[60000, 60000]);
    c.resync(const MutedVSync());
    log('muted ${mask(c)}');
    await steps(tester, c, <int>[60000, 60000, 400000]);
    log('after muted ${state(c)} ${mask(c)}');
    c.reset();
    log('reset ${state(c)}');
    future('i', c.forward());
    await steps(tester, c, <int>[0, 10000]);
    c.dispose();
    log('disposed ${mask(c)}');
    await tester.pump(const Duration(milliseconds: 100));
    log('after dispose pump');
  });

  scene('time_dilation', (WidgetTester tester) async {
    final c = AnimationController(duration: const Duration(milliseconds: 300), vsync: const TestVSync());
    watch('c', c);
    timeDilation = 3.0;
    future('f', c.forward());
    await steps(tester, c, frame60);
    timeDilation = 0.37;
    future('r', c.reverse());
    await steps(tester, c, <int>[0, 10001, 10001, 10001, 10001, 10001, 10001, 10001, 10001]);
    timeDilation = 1.0;
    await steps(tester, c, frame60);
    c.dispose();
  });

  scene('behavior', (WidgetTester tester) async {
    debugSemanticsDisableAnimations = true;
    final normal = AnimationController(duration: const Duration(milliseconds: 1000), vsync: const TestVSync());
    final preserve = AnimationController(duration: const Duration(milliseconds: 1000), vsync: const TestVSync(), animationBehavior: AnimationBehavior.preserve);
    watch('n', normal);
    watch('p', preserve);
    future('n', normal.forward());
    future('p', preserve.forward());
    await steps(tester, normal, <int>[0, 16667, 16667, 16667, 16667]);
    log('preserve ${state(preserve)}');
    future('nf', normal.fling(velocity: -1));
    future('pf', preserve.fling(velocity: -1, animationBehavior: AnimationBehavior.normal));
    await steps(tester, normal, <int>[0, 16667, 16667, 16667]);
    log('preserve ${state(preserve)}');
    future('nr', normal.repeat(reverse: true));
    await steps(tester, normal, <int>[0, 16667, 16667, 16667]);
    normal.stop();
    debugSemanticsDisableAnimations = null;
    future('n2', normal.forward(from: 0));
    await steps(tester, normal, <int>[0, 16667, 16667]);
    normal.dispose();
    preserve.dispose();
  });

  scene('curved', (WidgetTester tester) async {
    final c = AnimationController(duration: const Duration(milliseconds: 200), reverseDuration: const Duration(milliseconds: 150), vsync: const TestVSync());
    final curved = CurvedAnimation(parent: c, curve: Curves.easeIn, reverseCurve: Curves.elasticOut);
    final plain = CurvedAnimation(parent: c, curve: Curves.decelerate);
    c.addListener(() => log('curved ${curved.value} plain ${plain.value} ${mask(curved)}'));
    log('initial ${mask(curved)} ${mask(plain)}');
    c.forward();
    await steps(tester, c, <int>[0, 16667, 16667, 16667, 16667]);
    c.reverse();
    await steps(tester, c, <int>[0, 16667, 16667]);
    c.forward();
    await steps(tester, c, <int>[0, 16667, 16667]);
    await steps(tester, c, frame60);
    c.reverse();
    await steps(tester, c, frame60);
    curved.reverseCurve = null;
    curved.curve = Curves.slowMiddle;
    c.forward();
    await steps(tester, c, <int>[0, 50000]);
    curved.dispose();
    plain.dispose();
    c.dispose();
  });

  scene('train_hopping', (WidgetTester tester) async {
    final a = AnimationController(duration: const Duration(milliseconds: 300), vsync: const TestVSync(), value: 0.6);
    final b = AnimationController(duration: const Duration(milliseconds: 200), vsync: const TestVSync(), value: 0.1);
    final train = TrainHoppingAnimation(a, b, onSwitchedTrain: () => log('switched'));
    train.addListener(() => log('train ${train.value} ${train.status.name}'));
    train.addStatusListener((AnimationStatus s) => log('train status ${s.name}'));
    log('initial ${mask(train)}');
    a.reverse();
    b.forward();
    await steps(tester, a, frame60);
    log('after ${mask(train)} current=${identical(train.currentTrain, b)}');
    final same = TrainHoppingAnimation(a, AnimationController(vsync: const TestVSync(), value: a.value));
    log('same ${mask(same)}');
    final none = TrainHoppingAnimation(b, null);
    log('none ${mask(none)}');
    same.dispose();
    none.dispose();
    train.dispose();
    log('disposed ${train.currentTrain}');
    a.dispose();
    b.dispose();
  });

  scene('proxy_reverse_compound', (WidgetTester tester) async {
    final a = AnimationController(duration: const Duration(milliseconds: 100), vsync: const TestVSync());
    final b = AnimationController(duration: const Duration(milliseconds: 160), vsync: const TestVSync(), value: 0.75);
    final proxy = ProxyAnimation();
    log('proxy ${mask(proxy)} ${proxy.value} ${proxy.status.name}');
    void pl() => log('proxy tick ${proxy.value}');
    proxy.addListener(pl);
    proxy.addStatusListener((AnimationStatus s) => log('proxy status ${s.name}'));
    proxy.parent = a;
    proxy.parent = b;
    log('proxy ${mask(proxy)}');
    final reverse = ReverseAnimation(proxy);
    reverse.addListener(() => log('reverse tick ${reverse.value}'));
    reverse.addStatusListener((AnimationStatus s) => log('reverse status ${s.name}'));
    log('reverse ${mask(reverse)} ${reverse.status.name}');
    final mean = AnimationMean(left: a, right: b);
    final mx = AnimationMax<double>(a, b);
    final mn = AnimationMin<double>(a, b);
    mean.addListener(() => log('mean ${mean.value}'));
    mx.addListener(() => log('max ${mx.value}'));
    mn.addListener(() => log('min ${mn.value}'));
    mean.addStatusListener((AnimationStatus s) => log('mean status ${s.name}'));
    mx.addStatusListener((AnimationStatus s) => log('max status ${s.name}'));
    log('compound ${mask(mean)} ${mask(mx)} ${mask(mn)} ${mean.status.name}');
    a.forward();
    b.reverse();
    await steps(tester, a, <int>[0, 16667, 16667, 16667, 16667, 16667, 16667, 16667, 16667, 16667, 16667, 16667]);
    proxy.parent = a;
    proxy.parent = null;
    log('proxy null ${mask(proxy)} ${proxy.value} ${proxy.status.name}');
    proxy.removeListener(pl);
    a.reverse();
    await steps(tester, a, <int>[0, 30000, 30000, 30000, 30000]);
    log('stopped ${mask(kAlwaysCompleteAnimation)} ${mask(kAlwaysDismissedAnimation)} ${mask(const AlwaysStoppedAnimation<double>(0.25))} ${mask(const AlwaysStoppedAnimation<int>(3))}');
    a.dispose();
    b.dispose();
  });

  scene('edge_cases', (WidgetTester tester) async {
    // Duration * remainingFraction rounds: 1000us * 2/3 is 667us.
    final c = AnimationController(duration: const Duration(microseconds: 1000), vsync: const TestVSync(), value: 1 / 3);
    watch('c', c);
    future('third', c.forward());
    await steps(tester, c, <int>[0, 333, 333, 1, 1]);
    // An unbounded controller's remaining fraction is 1 (the range is infinite).
    final u = AnimationController.unbounded(duration: const Duration(milliseconds: 300), vsync: const TestVSync());
    watch('u', u);
    future('u', u.animateTo(100));
    await steps(tester, u, <int>[0, 100000, 100000, 100000, 1]);
    // repeat(count: 1) is done exactly at the exit time.
    final r = AnimationController(duration: const Duration(milliseconds: 100), vsync: const TestVSync());
    watch('r', r);
    future('r', r.repeat(count: 1));
    await steps(tester, r, <int>[0, 50000, 50000, 1]);
    // fling(velocity: 0) goes forward.
    r.value = 0.5;
    future('f0', r.fling(velocity: 0.0));
    await steps(tester, r, <int>[0, 16667, 16667, 16667]);
    // TrainHoppingAnimation hops on ties, and dedupes the status after a hop.
    final a = AnimationController(vsync: const TestVSync(), value: 0.5);
    final b = AnimationController(vsync: const TestVSync(), value: 0.25);
    final up = TrainHoppingAnimation(a, b, onSwitchedTrain: () => log('up switched'));
    up.addListener(() => log('up ${up.value}'));
    up.addStatusListener((AnimationStatus s) => log('up status ${s.name}'));
    a.value = 0.6;
    b.value = 0.6;
    log('up ${mask(up)} current=${identical(up.currentTrain, b)}');
    final d = AnimationController(vsync: const TestVSync(), value: 0.7);
    final e = AnimationController(vsync: const TestVSync(), value: 0.2);
    final down = TrainHoppingAnimation(e, d, onSwitchedTrain: () => log('down switched'));
    down.addListener(() => log('down ${down.value}'));
    d.value = 0.2;
    log('down ${mask(down)} current=${identical(down.currentTrain, d)}');
    final f = AnimationController(vsync: const TestVSync(), value: 0.4);
    final g = AnimationController(vsync: const TestVSync(), value: 0.4);
    final same = TrainHoppingAnimation(f, g);
    log('same current=${identical(same.currentTrain, g)}');
    g.value = 0.9;
    log('same ${same.value}');
    // dart:math max/min with signed zeros and NaN.
    const List<double> specials = <double>[0.0, -0.0, double.nan, 1.0];
    for (final double x in specials) {
      for (final double y in specials) {
        log('max/min $x $y ${AnimationMax<double>(AlwaysStoppedAnimation<double>(x), AlwaysStoppedAnimation<double>(y)).value} '
            '${AnimationMin<double>(AlwaysStoppedAnimation<double>(x), AlwaysStoppedAnimation<double>(y)).value}');
      }
    }
    log('ints ${AnimationMax<int>(const AlwaysStoppedAnimation<int>(3), const AlwaysStoppedAnimation<int>(-4)).value} '
        '${AnimationMin<int>(const AlwaysStoppedAnimation<int>(3), const AlwaysStoppedAnimation<int>(-4)).value}');
    up.dispose();
    down.dispose();
    same.dispose();
    for (final AnimationController x in <AnimationController>[c, u, r, a, b, d, e, f, g]) {
      x.dispose();
    }
  });

  tearDownAll(() {
    File('build/animation_controller.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
