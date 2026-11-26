// Reference side of flutter-go's BindingBase differential test: a binding
// with Flutter's test hooks (registerServiceExtension into a map, postEvent
// logged) logs its construction, the service extensions it registers, and
// the order of extension results, events, reassembles and lock/unlock
// against microtasks and timers in fake_async. Output: build/binding.json
// as {"log": [...]}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:fake_async/fake_async.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> entries = <String>[];
void log(String s) => entries.add(s);

class TestBinding extends BindingBase {
  final Map<String, ServiceExtensionCallback> extensions = <String, ServiceExtensionCallback>{};

  @override
  void initInstances() {
    log('initInstances');
    super.initInstances();
  }

  @override
  void initServiceExtensions() {
    log('initServiceExtensions');
    super.initServiceExtensions();
  }

  @override
  void registerServiceExtension({required String name, required ServiceExtensionCallback callback}) {
    log('register $name');
    extensions[name] = callback;
  }

  @override
  void postEvent(String eventKind, Map<String, dynamic> eventData) {
    log('event $eventKind ${json.encode(eventData)}');
  }

  @override
  Future<void> performReassemble() {
    log('performReassemble locked=$locked');
    return super.performReassemble();
  }

  @override
  void unlocked() {
    log('unlocked');
    super.unlocked();
  }

  void call(String name, Map<String, String> parameters) {
    log('call $name ${json.encode(parameters)}');
    extensions[name]!(parameters).then(
      (Map<String, dynamic> result) => log('result $name ${json.encode(result)}'),
      onError: (Object e) => log('error $name $e'),
    );
  }
}

void main() {
  test('binding', () {
    FakeAsync().run((FakeAsync fa) {
      final TestBinding binding = TestBinding();
      log('constructed $binding locked=${binding.locked}');
      scheduleMicrotask(() => log('micro 1'));
      binding.call('reassemble', <String, String>{});
      binding.call('platformOverride', <String, String>{});
      binding.call('platformOverride', <String, String>{'value': 'iOS'});
      scheduleMicrotask(() => log('micro 2'));
      fa.flushMicrotasks();
      log('flushed');
      binding.call('platformOverride', <String, String>{'value': 'bogus'});
      binding.call('brightnessOverride', <String, String>{'value': 'Brightness.dark'});
      binding.call('brightnessOverride', <String, String>{});
      binding.call('brightnessOverride', <String, String>{'value': 'x'});
      binding.call('connectedVmServiceUri', <String, String>{});
      binding.call('connectedVmServiceUri', <String, String>{'value': 'http://a/'});
      binding.call('activeDevToolsServerAddress', <String, String>{'value': 'http://b/'});
      fa.flushMicrotasks();
      log('flushed');

      // lockEvents nests; unlocked runs once, when the last lock ends.
      binding.lockEvents(() {
        log('callback 1 locked=${binding.locked}');
        return Future<void>.delayed(const Duration(milliseconds: 2), () => log('lock 1 ends'));
      });
      binding.lockEvents(() {
        log('callback 2');
        return Future<void>.delayed(const Duration(milliseconds: 1), () => log('lock 2 ends'));
      });
      log('locked=${binding.locked}');
      fa.elapse(const Duration(milliseconds: 1));
      log('at 1ms locked=${binding.locked}');
      fa.elapse(const Duration(milliseconds: 1));
      log('at 2ms locked=${binding.locked}');

      // The protected registration helpers.
      bool flag = false;
      double scale = 1;
      // ignore: invalid_use_of_protected_member
      binding.registerBoolServiceExtension(
        name: 'flag',
        getter: () async => flag,
        setter: (bool v) async {
          log('set flag $v');
          flag = v;
        },
      );
      // ignore: invalid_use_of_protected_member
      binding.registerNumericServiceExtension(
        name: 'scale',
        getter: () async => scale,
        setter: (double v) async => scale = v,
      );
      // ignore: invalid_use_of_protected_member
      binding.registerSignalServiceExtension(name: 'signal', callback: () async => log('signal'));
      binding.call('flag', <String, String>{'enabled': 'true'});
      binding.call('flag', <String, String>{});
      binding.call('scale', <String, String>{'scale': '0.1'});
      binding.call('scale', <String, String>{'scale': '1e300'});
      binding.call('signal', <String, String>{});
      fa.flushMicrotasks();
      log('end');
      debugDefaultTargetPlatformOverride = null;
      debugBrightnessOverride = null;
      connectedVmServiceUri = null;
      activeDevToolsServerAddress = null;
    });
    File('build/binding.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object>{'log': entries}));
  });
}
