// Reference side of flutter-go's painting image differential test: the
// observable behavior of the image cache (LRU eviction, sizes, statuses),
// image streams (listener and error order, reported errors), the
// multi-frame completer's frame timing, AssetImage's variant choice,
// ResizeImage's keys and decode sizes, decodeImageFromList, and the
// toStrings of the image types. Each scene logs lines; hash codes are
// normalized to #00000. Output: build/painting_images.json as
// {scene: [lines]}.
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/painting.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

final Map<String, List<String>> out = <String, List<String>>{};

String norm(Object? o) => o.toString().replaceAll(RegExp(r'#[0-9a-f]{5}'), '#00000');

String st(ImageCacheStatus s) => 'p${s.pending ? 1 : 0}k${s.keepAlive ? 1 : 0}l${s.live ? 1 : 0}';

class MockCodec implements ui.Codec {
  MockCodec(this.frameCount, this.repetitionCount);
  @override
  final int frameCount;
  @override
  final int repetitionCount;
  int numFramesAsked = 0;
  bool disposed = false;
  Completer<ui.FrameInfo> _next = Completer<ui.FrameInfo>();

  @override
  Future<ui.FrameInfo> getNextFrame() {
    numFramesAsked += 1;
    return _next.future;
  }

  void completeNextFrame(ui.FrameInfo f) {
    _next.complete(f);
    _next = Completer<ui.FrameInfo>();
  }

  @override
  void dispose() {
    disposed = true;
  }
}

class FakeFrame implements ui.FrameInfo {
  FakeFrame(this.duration, this.image);
  @override
  final Duration duration;
  @override
  final ui.Image image;
}

class TestCompleter extends ImageStreamCompleter {
  void set(ui.Image image) => setImage(ImageInfo(image: image.clone()));
}

class ManifestBundle extends CachingAssetBundle {
  ManifestBundle(this.manifest);
  final Map<String, List<Object>> manifest;
  @override
  Future<ByteData> load(String key) async {
    if (key == 'AssetManifest.bin') {
      return const StandardMessageCodec().encodeMessage(manifest)!;
    }
    throw FlutterError('key not found');
  }
}

/// An inner provider for ResizeImage: loadImage calls decode at once with a
/// buffer and records the getTargetSize the ResizeImage passes.
class CapturingProvider extends ImageProvider<CapturingProvider> {
  CapturingProvider(this.buffer);
  final ui.ImmutableBuffer buffer;
  @override
  Future<CapturingProvider> obtainKey(ImageConfiguration configuration) =>
      SynchronousFuture<CapturingProvider>(this);
  @override
  ImageStreamCompleter loadImage(CapturingProvider key, ImageDecoderCallback decode) {
    decode(buffer);
    return TestCompleter();
  }
}

void main() {
  late ui.Image img8, img10, img16, img20, img40;
  late ui.ImmutableBuffer buffer;

  setUpAll(() async {
    Future<ui.Image> make(int w, int h) {
      final Completer<ui.Image> c = Completer<ui.Image>();
      ui.decodeImageFromPixels(Uint8List(w * h * 4), w, h, ui.PixelFormat.rgba8888, c.complete);
      return c.future;
    }

    img8 = await make(8, 8);
    img10 = await make(10, 10);
    img16 = await make(16, 16);
    img20 = await make(20, 10);
    img40 = await make(40, 40);
    buffer = await ui.ImmutableBuffer.fromUint8List(Uint8List(4));
  });

  tearDownAll(() {
    File('build/painting_images.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });

  void scene(String name, Future<void> Function(WidgetTester tester, void Function(String) log) body) {
    testWidgets(name, (WidgetTester tester) async {
      final List<String> lines = out[name] = <String>[];
      await body(tester, lines.add);
      imageCache
        ..clear()
        ..clearLiveImages()
        ..maximumSize = 1000
        ..maximumSizeBytes = 100 << 20;
    });
  }

  scene('cache_lru', (WidgetTester tester, void Function(String) log) async {
    final ImageCache c = imageCache;
    final List<ui.Image> images = <ui.Image>[img8, img10, img16, img20, img40];
    void dump(String what) {
      final List<String> s = <String>[for (int k = 0; k < 7; k++) st(c.statusForKey(k))];
      log('$what: size ${c.currentSize} bytes ${c.currentSizeBytes} live ${c.liveImageCount} '
          'pending ${c.pendingImageCount} ${s.join(' ')} '
          'contains ${<int>[for (int k = 0; k < 7; k++) if (c.containsKey(k)) k]}');
    }

    c.maximumSize = 3;
    for (int k = 0; k < 5; k++) {
      final TestCompleter completer = TestCompleter()..set(images[k]);
      c.putIfAbsent(k, () => completer);
      dump('put $k');
    }
    c.putIfAbsent(2, () => throw StateError('loader called'));
    dump('touch 2');
    c.maximumSizeBytes = 1200;
    dump('maxBytes 1200');
    c.maximumSizeBytes = 100 << 20;
    c.maximumSize = 4;
    final TestCompleter pending = TestCompleter();
    c.putIfAbsent(5, () => pending);
    dump('pending 5');
    final ImageStreamListener l = ImageStreamListener((ImageInfo i, bool s) => i.dispose());
    pending.addListener(l);
    dump('listen 5');
    pending.set(img40);
    dump('complete 5');
    log('evict 3 ${c.evict(3)}');
    log('evict 6 ${c.evict(6)}');
    log('evict 5 includeLive:false ${c.evict(5, includeLive: false)}');
    dump('evicted');
    c.putIfAbsent(5, () => throw StateError('loader called'));
    dump('re-put live 5');
    pending.removeListener(l);
    dump('unlisten 5');
    c.maximumSize = 1;
    dump('max 1');
    c.clear();
    dump('clear');
    c.clearLiveImages();
    dump('clearLive');
    c.maximumSize = 0;
    final TestCompleter disabled = TestCompleter();
    c.putIfAbsent(6, () => disabled);
    dump('disabled put');
    disabled.set(img8);
    dump('disabled complete');
    ImageStreamCompleter? r = c.putIfAbsent(7, () => throw StateError('x'), onError: (Object e, StackTrace? s) {
      log('onError ${e is StateError ? e.message : e}');
    });
    log('result $r');
    await tester.pump();
  });

  scene('stream_listeners', (WidgetTester tester, void Function(String) log) async {
    final TestCompleter c = TestCompleter();
    final ImageStream stream = ImageStream();
    log(norm(stream));
    late ImageStreamListener l2;
    final ImageStreamListener l1 = ImageStreamListener((ImageInfo i, bool sync) {
      log('l1 ${norm(i)} sync $sync');
      i.dispose();
    });
    l2 = ImageStreamListener((ImageInfo i, bool sync) {
      log('l2 ${norm(i)} sync $sync');
      i.dispose();
      stream.removeListener(l2);
    }, onChunk: (ImageChunkEvent e) => log('chunk ${norm(e)}'));
    stream.addListener(l1);
    stream.addListener(l2);
    log(norm(stream));
    stream.setCompleter(c);
    log(norm(stream));
    log(norm(c));
    c.reportImageChunkEvent(const ImageChunkEvent(cumulativeBytesLoaded: 1, expectedTotalBytes: 3));
    c.reportImageChunkEvent(const ImageChunkEvent(cumulativeBytesLoaded: 3, expectedTotalBytes: null));
    c.set(img8);
    log(norm(c));
    c.set(img10);
    stream.addListener(ImageStreamListener((ImageInfo i, bool sync) {
      log('late ${norm(i)} sync $sync');
      i.dispose();
    }));
    c.addEphemeralErrorListener((Object e, StackTrace? s) => log('ephemeral (not called)'));
    log(norm(c));
    log('key is completer ${identical(stream.key, c)}');
    final ImageStreamCompleterHandle h = c.keepAlive();
    stream.removeListener(l1);
    log(norm(c));
    h.dispose();
    log(norm(c));
  });

  scene('stream_errors', (WidgetTester tester, void Function(String) log) async {
    final List<FlutterErrorDetails> reported = <FlutterErrorDetails>[];
    final FlutterExceptionHandler? old = FlutterError.onError;
    FlutterError.onError = reported.add;
    void dumpReported() {
      for (final FlutterErrorDetails d in reported) {
        log('reported ${d.exception} library ${d.library} context ${d.context} silent ${d.silent} '
            'info ${d.informationCollector?.call().map(norm).toList()}');
      }
      reported.clear();
    }

    final TestCompleter c = TestCompleter();
    c.addEphemeralErrorListener((Object e, StackTrace? s) => log('ephemeral1 $e'));
    c.addListener(ImageStreamListener((ImageInfo i, bool s) {}, onError: (Object e, StackTrace? s) {
      log('listener1 $e');
      throw 'listener1 threw';
    }));
    c.addListener(ImageStreamListener((ImageInfo i, bool s) {}, onError: (Object e, StackTrace? s) {
      log('listener2 $e');
      throw e;
    }));
    c.reportError(exception: 'boom', context: ErrorDescription('testing'), silent: true,
        informationCollector: () => <DiagnosticsNode>[ErrorDescription('extra info')]);
    dumpReported();
    c.addEphemeralErrorListener((Object e, StackTrace? s) => log('ephemeral2 $e'));
    c.addListener(ImageStreamListener((ImageInfo i, bool s) {}, onError: (Object e, StackTrace? s) {
      log('listener3 $e');
      throw 'listener3 threw';
    }));
    dumpReported();
    log(norm(c));

    final TestCompleter unheard = TestCompleter();
    unheard.reportError(exception: 'nobody listens');
    dumpReported();

    final TestCompleter imageThrows = TestCompleter();
    imageThrows.addListener(ImageStreamListener((ImageInfo i, bool s) {
      i.dispose();
      throw 'onImage threw';
    }, onError: (Object e, StackTrace? s) => log('onError $e')));
    imageThrows.set(img8);
    dumpReported();
    imageThrows.addListener(ImageStreamListener((ImageInfo i, bool s) {
      i.dispose();
      throw 'sync onImage threw';
    }));
    dumpReported();

    // OneFrame / MultiFrame completers whose futures fail; a provider whose
    // obtainKey throws; a provider whose load throws.
    final Completer<ImageInfo> one = Completer<ImageInfo>();
    OneFrameImageStreamCompleter(one.future, informationCollector: () => <DiagnosticsNode>[ErrorDescription('one')]);
    one.completeError('one failed');
    final Completer<ui.Codec> codec = Completer<ui.Codec>();
    MultiFrameImageStreamCompleter(codec: codec.future, scale: 2.0, debugLabel: 'multi');
    codec.completeError('codec failed');
    await tester.idle();
    dumpReported();
    FlutterError.onError = old;
  });

  scene('multi_frame', (WidgetTester tester, void Function(String) log) async {
    for (final double dilation in <double>[1.0, 2.0, 0.5]) {
      timeDilation = dilation;
      final MockCodec codec = MockCodec(3, 1);
      final Completer<ui.Codec> cc = Completer<ui.Codec>();
      final MultiFrameImageStreamCompleter c =
          MultiFrameImageStreamCompleter(codec: cc.future, scale: 2.0, debugLabel: 'anim');
      final Duration start = SchedulerBinding.instance.currentSystemFrameTimeStamp;
      final List<Duration> durations = <Duration>[
        const Duration(milliseconds: 100),
        const Duration(milliseconds: 250),
        const Duration(milliseconds: 40),
      ];
      final List<ui.Image> frames = <ui.Image>[img8, img10, img16];
      int t = 0;
      final ImageStreamListener l = ImageStreamListener((ImageInfo i, bool sync) {
        log('dilation $dilation t=$t frame ${i.image.width} ${norm(i)}');
        i.dispose();
      });
      c.addListener(l);
      cc.complete(codec);
      int served = 0;
      for (int step = 0; step < 40; step++) {
        if (codec.numFramesAsked > served) {
          codec.completeNextFrame(FakeFrame(durations[served % 3], frames[served % 3].clone()));
          served++;
        }
        await tester.idle();
        await tester.pump(const Duration(milliseconds: 30));
        t += 30;
        if (step % 5 == 0) {
          log('dilation $dilation t=$t asked ${codec.numFramesAsked} disposed ${codec.disposed} '
              'transient ${SchedulerBinding.instance.transientCallbackCount}');
        }
      }
      log('end ${codec.numFramesAsked} disposed ${codec.disposed} ${start >= Duration.zero}');
      c.removeListener(l);
      log('removed: disposed ${codec.disposed}');
    }
    timeDilation = 1.0;
  });

  scene('asset_variants', (WidgetTester tester, void Function(String) log) async {
    final List<List<double?>> manifests = <List<double?>>[
      <double?>[null, 1.5, 2.0, 3.0, 4.0],
      <double?>[2.0, 3.0],
      <double?>[0.5, null],
      <double?>[null, 3.0],
      <double?>[2.0, 2.0, 1.0],
      <double?>[],
    ];
    final List<double?> ratios = <double?>[
      null, 0.25, 0.5, 0.75, 1.0, 1.2, 1.5, 1.74, 1.75, 1.76, 1.99, 2.0, 2.24, 2.25, 2.5, 2.51, 2.6, 3.0,
      3.4, 3.5, 3.51, 4.0, 5.0,
    ];
    for (int m = 0; m < manifests.length; m++) {
      final ManifestBundle bundle = ManifestBundle(<String, List<Object>>{
        'img.png': <Object>[
          for (final (int i, double? d) in manifests[m].indexed)
            if (d == null) <String, Object>{'asset': 'img.png'} else <String, Object>{'asset': 'v$i/${d}x/img.png', 'dpr': d},
        ],
      });
      final AssetImage image = AssetImage('img.png', bundle: bundle);
      final List<String> chosen = <String>[];
      for (final double? r in ratios) {
        final Completer<AssetBundleImageKey> k = Completer<AssetBundleImageKey>();
        image.obtainKey(ImageConfiguration(devicePixelRatio: r)).then(k.complete);
        await tester.idle();
        final AssetBundleImageKey key = await k.future;
        chosen.add('$r:${key.name}@${key.scale}');
      }
      log('manifest $m ${chosen.join(' ')}');
    }
    final AssetImage pkg = AssetImage('a.png', package: 'pkg');
    log('package keyName ${pkg.keyName} ${norm(pkg)}');
  });

  scene('resize', (WidgetTester tester, void Function(String) log) async {
    final List<(int, int)> intrinsic = <(int, int)>[(50, 50), (50, 100), (100, 50), (33, 7), (7, 33), (1, 1), (640, 480), (3, 1000)];
    final List<(int?, int?)> dims = <(int?, int?)>[(25, 50), (50, 25), (25, null), (null, 25), (120, 100), (12, 25), (100, 200), (1, 1), (999, 3)];
    for (final ResizeImagePolicy policy in ResizeImagePolicy.values) {
      for (final bool up in <bool>[false, true]) {
        for (final (int?, int?) d in dims) {
          final ResizeImage r = ResizeImage(CapturingProvider(buffer), width: d.$1, height: d.$2, policy: policy, allowUpscaling: up);
          final ResizeImageKey key = await r.obtainKey(ImageConfiguration.empty);
          ui.TargetImageSizeCallback? got;
          final ImageStreamCompleter c = r.loadImage(key, (ui.ImmutableBuffer b, {ui.TargetImageSizeCallback? getTargetSize}) {
            got = getTargetSize;
            return Completer<ui.Codec>().future;
          });
          final List<String> sizes = <String>[];
          for (final (int, int) i in intrinsic) {
            try {
              final ui.TargetImageSize t = got!(i.$1, i.$2);
              sizes.add('${t.width}x${t.height}');
            } catch (e) {
              sizes.add('error');
            }
          }
          log('${policy.name} up $up ${d.$1}x${d.$2} ${c.debugLabel}: ${sizes.join(' ')}');
        }
      }
    }
    for (final double? dpr in <double?>[null, 1.0, 1.5, 2.0, 2.625, 3.0, 0.3]) {
      for (final bool logical in <bool>[false, true]) {
        final ResizeImage r = ResizeImage(CapturingProvider(buffer), width: 33, height: 17, useLogicalSize: logical);
        final ResizeImageKey key = await r.obtainKey(ImageConfiguration(devicePixelRatio: dpr));
        final ImageStreamCompleter c = r.loadImage(key, (ui.ImmutableBuffer b, {ui.TargetImageSizeCallback? getTargetSize}) {
          return Completer<ui.Codec>().future;
        });
        log('dpr $dpr logical $logical ${c.debugLabel}');
      }
    }
  });

  scene('to_strings', (WidgetTester tester, void Function(String) log) async {
    log(ImageConfiguration.empty.toString());
    log(const ImageConfiguration(
      devicePixelRatio: 1.25,
      locale: Locale('en', 'US'),
      textDirection: TextDirection.rtl,
      size: Size(10, 20.5),
      platform: TargetPlatform.iOS,
    ).toString());
    log(const ImageConfiguration(devicePixelRatio: 2.75).toString());
    log(const ImageConfiguration(size: Size.zero, platform: TargetPlatform.android).toString());
    log(const ImageConfiguration(devicePixelRatio: 1.0).copyWith(textDirection: TextDirection.ltr).toString());
    log('${const ImageConfiguration(devicePixelRatio: 1.0) == const ImageConfiguration(devicePixelRatio: 1.0)}');
    log(const NetworkImage('https://x/y.png', scale: 2.5, headers: <String, String>{'a': 'b'}).toString());
    log(const NetworkImage('u').toString());
    log(const ExactAssetImage('e.png', scale: 3.0, package: 'p').toString());
    log(FileImage(File('/tmp/f.png'), scale: 0.5).toString());
    log(norm(MemoryImage(Uint8List(3), scale: 1.05)));
    log(norm(AssetBundleImageKey(bundle: ManifestBundle(const <String, List<Object>>{}), name: 'n', scale: 1.5)));
    log(norm(imageCache.statusForKey('nothing')));
    log(norm(const ImageChunkEvent(cumulativeBytesLoaded: 10, expectedTotalBytes: null)));
    log(norm(ImageInfo(image: img20, scale: 1.5)));
    log(norm(ImageInfo(image: img20, scale: 3, debugLabel: 'label')));
    log(NetworkImageLoadException(statusCode: 404, uri: Uri.parse('https://a/b')).toString());
    const ImageSizeInfo info = ImageSizeInfo(source: 'src', displaySize: Size(10, 20), imageSize: Size(33, 7));
    log('$info ${info.displaySizeInBytes} ${info.decodedSizeInBytes} ${jsonEncode(info.toJson())}');
    const ImageSizeInfo info2 = ImageSizeInfo(displaySize: Size(1.5, 2.5), imageSize: Size(1000, 999));
    log('$info2 ${info2.displaySizeInBytes} ${info2.decodedSizeInBytes} ${jsonEncode(info2.toJson())}');
    log(norm(ImageStream()));
    final OneFrameImageStreamCompleter one =
        OneFrameImageStreamCompleter(SynchronousFuture<ImageInfo>(ImageInfo(image: img8.clone(), scale: 2)));
    log(norm(one));
    log(ResizeImagePolicy.fit.toString());
    log(WebHtmlElementStrategy.prefer.toString());
  });

  test('decode', () async {
    final List<String> lines = out['decode'] = <String>[];
    final List<List<int>> data = <List<int>>[
      // PNGs 1x1, 3x2, 17x5; GIF 1x1.
      base64Decode('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg=='),
      base64Decode('iVBORw0KGgoAAAANSUhEUgAAAAMAAAACCAYAAACddGYaAAAAEUlEQVR4nGP4z8DwH4YZkDkAm34L9XKwuTwAAAAASUVORK5CYII='),
      base64Decode('iVBORw0KGgoAAAANSUhEUgAAABEAAAAFCAYAAACjKgd3AAAAFElEQVR4nGP4z8Dwn1LMMGoIBgYAInqpV+p9ZlMAAAAASUVORK5CYII='),
      base64Decode('R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7'),
    ];
    for (final List<int> d in data) {
      final ui.Image image = await decodeImageFromList(Uint8List.fromList(d));
      lines.add('${image.width}x${image.height}');
    }
    try {
      await decodeImageFromList(Uint8List.fromList(<int>[1, 2, 3]));
      lines.add('no error');
    } catch (e) {
      lines.add('error ${e.runtimeType}');
    }
  });
}
