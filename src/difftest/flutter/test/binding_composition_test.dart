// Reference side of flutter-go's binding composition differential test:
// a binding composed like WidgetsFlutterBinding's lower layers
// (BindingBase with GestureBinding, SchedulerBinding, ServicesBinding,
// SemanticsBinding), with logging mixins interleaved between them, logs
// the order Dart's mixin linearization runs initInstances,
// initServiceExtensions, unlocked, createBinaryMessenger and
// handleAppLifecycleStateChanged in, which `X.instance` statics are set at
// each step, and the service extensions registered, in order.
// The Go twin is ../../binding_composition_test.go.
// Output: build/binding_composition.json as {"log": [...]}.
import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];

String instances() {
  final parts = <String>[];
  void probe(String name, Object Function() get) {
    try {
      get();
      parts.add(name);
    } catch (_) {
      parts.add('-');
    }
  }

  probe('G', () => GestureBinding.instance);
  probe('S', () => SchedulerBinding.instance);
  probe('V', () => ServicesBinding.instance);
  probe('M', () => SemanticsBinding.instance);
  return parts.join();
}

void logInit(String name, void Function() superCall) {
  log.add('$name.initInstances before ${instances()}');
  superCall();
  log.add('$name.initInstances after ${instances()}');
}

mixin L0 on BindingBase {
  @override
  void initInstances() => logInit('L0', super.initInstances);
  @override
  void initServiceExtensions() {
    log.add('L0.initServiceExtensions before');
    super.initServiceExtensions();
    log.add('L0.initServiceExtensions after');
  }

  @override
  void unlocked() {
    log.add('L0.unlocked before');
    super.unlocked();
    log.add('L0.unlocked after');
  }
}

mixin L1 on BindingBase {
  @override
  void initInstances() => logInit('L1', super.initInstances);
  @override
  void initServiceExtensions() {
    log.add('L1.initServiceExtensions before');
    super.initServiceExtensions();
    log.add('L1.initServiceExtensions after');
  }

  @override
  void unlocked() {
    log.add('L1.unlocked before');
    super.unlocked();
    log.add('L1.unlocked after');
  }
}

mixin L2 on BindingBase {
  @override
  void initInstances() => logInit('L2', super.initInstances);
  @override
  void initServiceExtensions() {
    log.add('L2.initServiceExtensions before');
    super.initServiceExtensions();
    log.add('L2.initServiceExtensions after');
  }

  @override
  void unlocked() {
    log.add('L2.unlocked before');
    super.unlocked();
    log.add('L2.unlocked after');
  }
}

// After ServicesBinding: overrides its createBinaryMessenger.
mixin L3 on BindingBase, ServicesBinding {
  @override
  void initInstances() => logInit('L3', super.initInstances);
  @override
  void initServiceExtensions() {
    log.add('L3.initServiceExtensions before');
    super.initServiceExtensions();
    log.add('L3.initServiceExtensions after');
  }

  @override
  BinaryMessenger createBinaryMessenger() {
    log.add('L3.createBinaryMessenger ${instances()}');
    return super.createBinaryMessenger();
  }

  @override
  void handleAppLifecycleStateChanged(ui.AppLifecycleState state) {
    log.add('L3.handleAppLifecycleStateChanged ${state.name} before');
    super.handleAppLifecycleStateChanged(state);
    log.add('L3.handleAppLifecycleStateChanged ${state.name} after ${lifecycleState?.name}');
  }
}

mixin L4 on BindingBase, SchedulerBinding {
  @override
  void initInstances() => logInit('L4', super.initInstances);
  @override
  void initServiceExtensions() {
    log.add('L4.initServiceExtensions before');
    super.initServiceExtensions();
    log.add('L4.initServiceExtensions after');
  }

  @override
  void unlocked() {
    log.add('L4.unlocked before');
    super.unlocked();
    log.add('L4.unlocked after');
  }

  @override
  void handleAppLifecycleStateChanged(ui.AppLifecycleState state) {
    log.add('L4.handleAppLifecycleStateChanged ${state.name}');
    super.handleAppLifecycleStateChanged(state);
  }
}

class ComposedBinding extends BindingBase
    with
        L0,
        GestureBinding,
        L1,
        SchedulerBinding,
        L2,
        ServicesBinding,
        L3,
        SemanticsBinding,
        L4 {
  @override
  void initInstances() => logInit('ComposedBinding', super.initInstances);

  @override
  void registerServiceExtension({
    required String name,
    required ServiceExtensionCallback callback,
  }) {
    log.add('registerServiceExtension $name ${instances()}');
    super.registerServiceExtension(name: name, callback: callback);
  }

  @override
  BinaryMessenger createBinaryMessenger() {
    log.add('ComposedBinding.createBinaryMessenger');
    return super.createBinaryMessenger();
  }

  @override
  void performSemanticsAction(ui.SemanticsActionEvent action) {}
}

void main() {
  test('binding composition', () async {
    final binding = ComposedBinding();
    log.add('constructed ${instances()}');
    log.add('debugBindingType ${BindingBase.debugBindingType()}');
    log.add('toString $binding');
    log.add('same instances ${identical(GestureBinding.instance, binding)} '
        '${identical(SchedulerBinding.instance, binding)} '
        '${identical(ServicesBinding.instance, binding)} '
        '${identical(SemanticsBinding.instance, binding)}');
    final pd = binding.platformDispatcher;
    log.add('dispatcher pointer=${pd.onPointerDataPacket != null} '
        'semanticsEnabled=${pd.onSemanticsEnabledChanged != null} '
        'semanticsAction=${pd.onSemanticsActionEvent != null} '
        'a11y=${pd.onAccessibilityFeaturesChanged != null} '
        'timings=${pd.onReportTimings != null} '
        'beginFrame=${pd.onBeginFrame != null}');

    // unlocked runs down the chain when lockEvents' future completes.
    log.add('lockEvents');
    await binding.lockEvents(() async {
      log.add('locked ${binding.locked}');
    });
    log.add('after lockEvents locked=${binding.locked}');

    // The lifecycle channel's handler (ServicesBinding) calls
    // handleAppLifecycleStateChanged through the binding.
    log.add('lifecycle message');
    await binding.defaultBinaryMessenger.handlePlatformMessage(
      'flutter/lifecycle',
      const StringCodec().encodeMessage('AppLifecycleState.paused'),
      (ByteData? _) {},
    );
    log.add('lifecycleState ${binding.lifecycleState?.name} framesEnabled ${binding.framesEnabled}');
    binding.handleAppLifecycleStateChanged(ui.AppLifecycleState.resumed);
    log.add('lifecycleState ${binding.lifecycleState?.name} framesEnabled ${binding.framesEnabled}');

    File('build/binding_composition.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'log': log}));
  });
}
