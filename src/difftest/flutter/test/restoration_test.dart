// Reference side of flutter-go's restoration differential test: a
// RestorationManager talking to a mock engine on flutter/restoration, and
// sequences of RestorationBucket operations (write/read/remove, claimChild,
// adoptChild, rename, dispose, duplicate claims, data pushed by the engine).
// Every 'put' the manager sends is logged byte for byte (the restoration
// data's encoding, including the maps' insertion order), with the values
// read back and the errors thrown. The Go twin is ../../restoration_test.go.
// Output: build/restoration.json as {"log": [...]}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final List<String> log = <String>[];

String bytes(Object? data) {
  if (data is Uint8List) {
    return data.join(',');
  }
  if (data is ByteData) {
    return data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes).join(',');
  }
  return '$data';
}

TestDefaultBinaryMessenger get messenger =>
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

Uint8List encode(Object? data) {
  final ByteData encoded = const StandardMessageCodec().encodeMessage(data)!;
  return encoded.buffer.asUint8List(encoded.offsetInBytes, encoded.lengthInBytes);
}

Future<void> push(Object? data) async {
  await messenger.handlePlatformMessage(
    'flutter/restoration',
    const StandardMethodCodec().encodeMethodCall(MethodCall('push', data)),
    (_) {},
  );
}

void flush(RestorationManager manager, String label) {
  log.add('flush $label');
  try {
    manager.flushData();
  } on FlutterError catch (e) {
    log.add('FlutterError: ${e.message}');
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('restoration', () async {
    Object? getReply = <String, Object?>{
      'enabled': true,
      'data': encode(<String, Object?>{
        'v': <String, Object?>{'count': 7, 'name': 'Ada', 'list': <Object?>[1, 'two', 3.5, null]},
        'c': <String, Object?>{
          'page': <String, Object?>{
            'v': <String, Object?>{'scroll': 120.5, 'flag': true},
            'c': <String, Object?>{
              'field': <String, Object?>{
                'v': <String, Object?>{'text': 'hello', 'map': <Object?, Object?>{'b': 1, 'a': 2}},
              },
            },
          },
          'other': <String, Object?>{'v': <String, Object?>{'x': 1}},
        },
      }),
    };
    messenger.setMockMethodCallHandler(SystemChannels.restoration, (MethodCall call) async {
      if (call.method == 'put') {
        log.add('put ${bytes(call.arguments)}');
        return null;
      }
      log.add('engine ${call.method}');
      return getReply;
    });

    final manager = RestorationManager();
    var notifications = 0;
    manager.addListener(() => notifications++);

    log.add('scene root');
    final RestorationBucket root = (await manager.rootBucket)!;
    log.add('root ${root.restorationId} replacing ${root.isReplacing}');
    for (final key in <String>['count', 'name', 'list', 'missing']) {
      log.add('read $key ${root.read<Object?>(key)} contains ${root.contains(key)}');
    }

    log.add('scene values');
    root.write<int>('count', 7); // unchanged: nothing to serialize
    flush(manager, 'unchanged');
    root.write<int>('count', 8);
    root.write<String>('added', 'new');
    root.write<Object?>('nothing', null);
    root.write<double>('pi', 3.14159);
    root.write<List<Object?>>('nested', <Object?>[<String, Object?>{'z': 1, 'y': <int>[1, 2]}, false]);
    flush(manager, 'values');
    log.add('remove name ${root.remove<String>('name')}');
    log.add('remove missing ${root.remove<String>('missing')}');
    root.write<int>('count', 9);
    flush(manager, 'remove');

    log.add('scene children');
    final RestorationBucket page = root.claimChild('page', debugOwner: 'pageOwner');
    log.add('page scroll ${page.read<double>('scroll')} flag ${page.read<bool>('flag')}');
    final RestorationBucket field = page.claimChild('field', debugOwner: 'fieldOwner');
    log.add('field text ${field.read<String>('text')} map ${field.read<Object?>('map')}');
    flush(manager, 'claimed existing');
    field.write<String>('text', 'world');
    final RestorationBucket fresh = page.claimChild('fresh', debugOwner: 'freshOwner');
    fresh.write<int>('a', 1);
    final RestorationBucket deep = fresh.claimChild('deep', debugOwner: 'deepOwner');
    deep.write<String>('d', 'deep');
    flush(manager, 'new children');

    log.add('scene rename');
    fresh.rename('renamed');
    flush(manager, 'rename');
    fresh.rename('renamed');
    flush(manager, 'rename same');

    log.add('scene adopt');
    root.adoptChild(deep);
    flush(manager, 'adopt');
    final empty = RestorationBucket.empty(restorationId: 'adopted', debugOwner: 'emptyOwner');
    empty.write<String>('e', 'mpty');
    page.adoptChild(empty);
    flush(manager, 'adopt empty');

    log.add('scene dispose');
    field.dispose();
    flush(manager, 'field disposed');
    page.dispose();
    flush(manager, 'page disposed');
    log.add('remove all ${root.remove<int>('count')} ${root.remove<String>('added')} '
        '${root.remove<Object?>('nothing')} ${root.remove<double>('pi')} '
        '${root.remove<Object?>('nested')} ${root.remove<Object?>('list')}');
    flush(manager, 'values removed');

    log.add('scene engine push');
    final dupKeys = WriteBuffer()
      ..putUint8(13) // map of 3: 'v' -> {1: 'a', 1.0: 'b', 'k': 'c'}
      ..putUint8(1)
      ..putUint8(7)
      ..putUint8(1)
      ..putUint8(118)
      ..putUint8(13)
      ..putUint8(3)
      ..putUint8(3)
      ..putInt32(1)
      ..putUint8(7)
      ..putUint8(1)
      ..putUint8(97)
      ..putUint8(6)
      ..putFloat64(1.0)
      ..putUint8(7)
      ..putUint8(1)
      ..putUint8(98)
      ..putUint8(7)
      ..putUint8(1)
      ..putUint8(107)
      ..putUint8(7)
      ..putUint8(1)
      ..putUint8(99);
    final ByteData dupData = dupKeys.done();
    await push(<String, Object?>{
      'enabled': true,
      'data': dupData.buffer.asUint8List(dupData.offsetInBytes, dupData.lengthInBytes),
    });
    final RestorationBucket pushed = (await manager.rootBucket)!;
    log.add('notifications $notifications same ${identical(pushed, root)} replacing ${manager.isReplacing}');
    log.add('pushed 1 ${pushed.read<Object?>('1')} k ${pushed.read<String>('k')}');
    pushed.write<String>('k', 'C');
    pushed.write<int>('z', 26);
    flush(manager, 'pushed');
    await push(<String, Object?>{'enabled': false, 'data': null});
    log.add('notifications $notifications root ${await manager.rootBucket}');
    await push(<String, Object?>{'enabled': true, 'data': null});
    final RestorationBucket blank = (await manager.rootBucket)!;
    log.add('notifications $notifications blank ${blank.contains('k')}');
    blank.write<int>('only', 1);
    flush(manager, 'blank');

    log.add('scene second manager');
    getReply = null;
    final manager2 = RestorationManager();
    log.add('manager2 root ${await manager2.rootBucket}');
    getReply = <String, Object?>{'enabled': true, 'data': null};
    final manager3 = RestorationManager();
    final RestorationBucket root3 = (await manager3.rootBucket)!;
    final RestorationBucket a = root3.claimChild('a', debugOwner: null);
    final RestorationBucket b = root3.claimChild('b', debugOwner: null);
    b.write<int>('bv', 2);
    a.write<int>('av', 1);
    root3.write<int>('rv', 0);
    flush(manager3, 'insertion order');
    b.rename('a2');
    a.rename('b2');
    flush(manager3, 'renamed order');
    log.add('scene duplicate claims');
    final manager4 = RestorationManager();
    final RestorationBucket root4 = (await manager4.rootBucket)!;
    final RestorationBucket other = root4.claimChild('other', debugOwner: 'otherOwner');
    other.write<int>('x', 1);
    flush(manager4, 'other');
    final RestorationBucket dup1 = root4.claimChild('other', debugOwner: 'firstOther');
    log.add('dup1 x ${dup1.read<int>('x')}');
    final RestorationBucket dup2 = root4.claimChild('other', debugOwner: 'secondOther');
    log.add('dup2 x ${dup2.read<int>('x')}');
    dup2.write<int>('y', 2);
    other.dispose();
    flush(manager4, 'two pending, one given up');
    try {
      manager4.flushData();
    } on AssertionError catch (e) {
      log.add('wedged: AssertionError(${e.message})');
    }
    manager2.dispose();
    manager3.dispose();
    manager.dispose();

    File('build/restoration.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'log': log}));
  });
}
