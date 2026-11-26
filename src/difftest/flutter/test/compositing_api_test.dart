// Reference side of flutter-go's dart:ui compositing-API differential
// test: SceneBuilder/Scene/EngineLayer behavior that is computed without
// rasterizing — toString, runtime types, the retained-layer debug checks
// and their messages (layer hashes masked), argument checks, use after
// build/dispose, FlutterView.render outside a frame.
// Output: build/compositing_api.json as {case: value}.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

String err(void Function() f) {
  try {
    f();
  } catch (e) {
    if (e is AssertionError) {
      return 'AssertionError: ${e.message}'.replaceAll(RegExp(r'#[0-9a-f]{5}'), '#HASH');
    }
    return '$e';
  }
  return 'no error';
}

typedef Push = EngineLayer Function(SceneBuilder b, EngineLayer? old);

Picture picture() {
  final PictureRecorder rec = PictureRecorder();
  Canvas(rec).drawRect(const Rect.fromLTRB(0, 0, 4, 4), Paint()..color = const Color(0xFF00FF00));
  return rec.endRecording();
}

void main() {
  test('compositing_api', () {
    final Map<String, Object?> out = <String, Object?>{};
    final Float64List identity = Float64List.fromList(<double>[1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]);
    final Map<String, Push> pushes = <String, Push>{
      'pushOffset': (SceneBuilder b, EngineLayer? old) => b.pushOffset(1, 2, oldLayer: old as OffsetEngineLayer?),
      'pushTransform': (SceneBuilder b, EngineLayer? old) =>
          b.pushTransform(identity, oldLayer: old as TransformEngineLayer?),
      'pushClipRect': (SceneBuilder b, EngineLayer? old) =>
          b.pushClipRect(Rect.zero, oldLayer: old as ClipRectEngineLayer?),
      'pushClipRRect': (SceneBuilder b, EngineLayer? old) =>
          b.pushClipRRect(RRect.zero, oldLayer: old as ClipRRectEngineLayer?),
      'pushClipRSuperellipse': (SceneBuilder b, EngineLayer? old) =>
          b.pushClipRSuperellipse(RSuperellipse.zero, oldLayer: old as ClipRSuperellipseEngineLayer?),
      'pushClipPath': (SceneBuilder b, EngineLayer? old) =>
          b.pushClipPath(Path(), oldLayer: old as ClipPathEngineLayer?),
      'pushOpacity': (SceneBuilder b, EngineLayer? old) => b.pushOpacity(100, oldLayer: old as OpacityEngineLayer?),
      'pushColorFilter': (SceneBuilder b, EngineLayer? old) => b.pushColorFilter(
            const ColorFilter.mode(Color(0x80000000), BlendMode.srcIn),
            oldLayer: old as ColorFilterEngineLayer?,
          ),
      'pushImageFilter': (SceneBuilder b, EngineLayer? old) =>
          b.pushImageFilter(ImageFilter.blur(sigmaX: 2), oldLayer: old as ImageFilterEngineLayer?),
      'pushBackdropFilter': (SceneBuilder b, EngineLayer? old) =>
          b.pushBackdropFilter(ImageFilter.blur(sigmaX: 2), oldLayer: old as BackdropFilterEngineLayer?),
      'pushShaderMask': (SceneBuilder b, EngineLayer? old) => b.pushShaderMask(
            Gradient.linear(Offset.zero, const Offset(1, 0), const <Color>[Color(0xFF000000), Color(0x00000000)]),
            Rect.zero,
            BlendMode.dstIn,
            oldLayer: old as ShaderMaskEngineLayer?,
          ),
    };

    // ---- each typed layer: its runtime type and the debug checks.
    pushes.forEach((String name, Push push) {
      final List<String> log = <String>[];
      final SceneBuilder b1 = SceneBuilder();
      final EngineLayer layer = push(b1, null);
      final OpacityEngineLayer child = b1.pushOpacity(10);
      b1.pop();
      b1.pop();
      b1.build();
      log.add('${layer.runtimeType}');
      // Retained then pushed as oldLayer in the same scene.
      final SceneBuilder b2 = SceneBuilder();
      log.add(err(() => b2.addRetained(layer)));
      log.add(err(() => push(b2, layer)));
      log.add(err(() => b2.pushOpacity(1, oldLayer: child)));
      log.add(err(() => b2.addRetained(layer)));
      b2.build();
      // Pushed as oldLayer, then retained, then pushed again.
      final SceneBuilder b3 = SceneBuilder();
      log.add(err(() => push(b3, layer)));
      log.add(err(() => b3.addRetained(layer)));
      log.add(err(() => push(b3, layer)));
      b3.build();
      log.add(err(() => SceneBuilder().addRetained(layer)));
      log.add(err(() => push(SceneBuilder(), layer)));
      // Disposing.
      final SceneBuilder b4 = SceneBuilder();
      final EngineLayer fresh = push(b4, null);
      b4.build();
      log.add(err(fresh.dispose));
      log.add(err(fresh.dispose));
      log.add(err(() => push(SceneBuilder(), fresh)));
      log.add(err(() => SceneBuilder().addRetained(fresh)));
      out['layer.$name'] = log;
    });

    // ---- the builder and the scene
    final SceneBuilder builder = SceneBuilder();
    out['sceneBuilder.toString'] = '$builder';
    out['sceneBuilder.matrix'] = <String>[
      err(() => builder.pushTransform(Float64List(15))),
      err(() => builder.pushTransform(Float64List.fromList(<double>[1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, double.nan]))),
      err(() => builder.pushTransform(Float64List.fromList(<double>[1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, double.infinity, 0, 0, 1]))),
      err(() => builder.pushTransform(Float64List(16))),
    ];
    out['sceneBuilder.clipNone'] = <String>[
      err(() => builder.pushClipRect(Rect.zero, clipBehavior: Clip.none)),
      err(() => builder.pushClipRRect(RRect.zero, clipBehavior: Clip.none)),
      err(() => builder.pushClipRSuperellipse(RSuperellipse.zero, clipBehavior: Clip.none)),
      err(() => builder.pushClipPath(Path(), clipBehavior: Clip.none)),
      err(() => builder.pushClipRect(Rect.zero, clipBehavior: Clip.hardEdge)),
    ];
    final Picture disposed = picture()..dispose();
    out['sceneBuilder.leaves'] = <String>[
      err(() => builder.addPicture(Offset.zero, disposed)),
      err(() => builder.addPicture(const Offset(1, 2), picture(), isComplexHint: true, willChangeHint: true)),
      err(() => builder.addTexture(3, offset: const Offset(1, 1), width: 2, height: 2, freeze: true)),
      err(() => builder.addPlatformView(4, width: 5, height: 6)),
      err(() => builder.addPerformanceOverlay(15, const Rect.fromLTRB(0, 0, 10, 10))),
      err(() => builder.pop()),
      err(() => builder.pop()),
      err(() => builder.pop()),
    ];
    final Scene scene = builder.build();
    out['scene.toString'] = '$scene';
    out['sceneBuilder.afterBuild'] = <String>[
      err(() => builder.pushOffset(0, 0)),
      err(() => builder.pop()),
      err(() => builder.addPicture(Offset.zero, picture())),
      err(() => builder.build()),
      '$builder',
    ];
    out['scene.errors'] = <String>[
      err(() => scene.toImageSync(0, 1)),
      err(() => scene.toImageSync(1, -1)),
      err(() => scene.toImage(0, 0)),
    ];
    final Image image = scene.toImageSync(3, 2);
    out['scene.toImageSync'] = <int>[image.width, image.height];
    out['render.outsideFrame'] = <String>[
      err(() => PlatformDispatcher.instance.implicitView!.render(scene)),
      err(() => PlatformDispatcher.instance.implicitView!.render(scene, size: const Size(10, 10))),
    ];
    scene.dispose();
    out['scene.afterDispose'] = <String>[
      err(() => scene.toImageSync(1, 1)),
      err(() => scene.toImage(1, 1)),
      err(() => scene.dispose()),
      err(() => scene.toImageSync(0, 1)),
      '$scene',
    ];

    Directory('build').createSync(recursive: true);
    File('build/compositing_api.json').writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
