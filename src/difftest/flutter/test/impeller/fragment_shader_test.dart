// Reference side of flutter-go's fragment shader differential test
// (dart:ui's FragmentProgram, FragmentShader, uniform slots, image sampler
// slots and ImageFilter.shader): uniform reflection (slot indices), slot
// and sampler errors and their messages, ImageFilter.shader's errors,
// toString and ==, and the pixels a few shaders render through
// picture.toImage → toByteData. Run with --enable-impeller (it writes
// nothing otherwise).
// Output: build/fragment_shader.json as {case: value}.
//
// The shaders are copies of the engine's fixtures
// (engine/src/flutter/lib/ui/fixtures/shaders) declared under `shaders:` in
// pubspec.yaml; the tester reads their Vulkan stage, flutter-go their Metal
// stage. Cases whose answer depends on the backend's reflection (sampler
// order of double_sampler_swapped, the Vulkan uniform block's name) are left
// out.
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

String errorType(Object e) {
  if (e is IndexError) {
    return 'IndexError';
  }
  if (e is RangeError) {
    return 'RangeError';
  }
  if (e is ArgumentError) {
    return 'ArgumentError';
  }
  if (e is StateError) {
    return 'StateError';
  }
  if (e is AssertionError) {
    return 'AssertionError';
  }
  if (e is String) {
    return 'String';
  }
  if (e is Exception) {
    return 'Exception';
  }
  return 'other';
}

String errT(Object? Function() f) {
  try {
    f();
    return 'no error';
  } catch (e) {
    // An AssertionError's toString has the assert's source location.
    return '${errorType(e)}: ${e is AssertionError ? e.message : e}';
  }
}

Future<List<int>> pixels(Shader? shader, int w, int h, {ImageFilter? filter, Color color = const Color(0xFFFF0000)}) async {
  final recorder = PictureRecorder();
  final canvas = Canvas(recorder);
  final paint = Paint()..color = color;
  if (shader != null) {
    paint.shader = shader;
  }
  if (filter != null) {
    paint.imageFilter = filter;
  }
  canvas.drawPaint(paint);
  final Picture picture = recorder.endRecording();
  final Image image = await picture.toImage(w, h);
  final ByteData data = (await image.toByteData())!;
  picture.dispose();
  image.dispose();
  return data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes).toList();
}

// A w×h image of solid rects: left half blue, right half green (as
// _createBlueGreenImageSync), or 4 colored quadrants.
Image blueGreenImage() {
  final recorder = PictureRecorder();
  final canvas = Canvas(recorder);
  canvas.drawRect(const Rect.fromLTWH(0, 0, 5, 10), Paint()..color = const Color(0xFF0000FF));
  canvas.drawRect(const Rect.fromLTWH(5, 0, 5, 10), Paint()..color = const Color(0xFF00FF00));
  final Picture picture = recorder.endRecording();
  final Image image = picture.toImageSync(10, 10);
  picture.dispose();
  return image;
}

Image quadrantImage() {
  final recorder = PictureRecorder();
  final canvas = Canvas(recorder);
  canvas.drawRect(const Rect.fromLTWH(0, 0, 2, 2), Paint()..color = const Color(0xFFFF0000));
  canvas.drawRect(const Rect.fromLTWH(2, 0, 2, 2), Paint()..color = const Color(0xFF00FF00));
  canvas.drawRect(const Rect.fromLTWH(0, 2, 2, 2), Paint()..color = const Color(0xFF0000FF));
  canvas.drawRect(const Rect.fromLTWH(2, 2, 2, 2), Paint()..color = const Color(0x80FFFFFF));
  final Picture picture = recorder.endRecording();
  final Image image = picture.toImageSync(4, 4);
  picture.dispose();
  return image;
}

String hashless(String s) => s.replaceAll(RegExp(r'#\d+'), '#H');

void main() {
  test('fragment_shader', () async {
    // Impeller only: the Skia pass of the difftest run must not overwrite
    // the reference.
    if (Platform.environment['FLUTTER_TEST_IMPELLER'] != 'true') {
      return;
    }
    final out = <String, Object?>{};
    Future<FragmentProgram> load(String name) => FragmentProgram.fromAsset('shaders/$name.frag');

    // Uniform reflection: the shader index of every float of every
    // uniform, and the sampler indices.
    final reflect = <String, Object?>{};
    final Map<String, List<String>> uniforms = <String, List<String>>{
      'uniforms': <String>['iFloatUniform', 'iVec2Uniform', 'iMat2Uniform', 'iVec3Uniform', 'iVec4Uniform',
        'iFloatArrayUniform', 'iVec2ArrayUniform', 'iVec3ArrayUniform', 'iVec4ArrayUniform'],
      'all_uniforms': <String>['uFloat', 'uVec2', 'uVec3', 'uVec4', 'uMat2', 'uMat3', 'uMat4', 'uFloatArray',
        'uVec2Array', 'uVec3Array', 'uVec4Array', 'uMat2Array', 'uMat3Array', 'uMat4Array'],
      'uniform_ordering': <String>['a', 'b', 'c', 'u_texture'],
      'children_and_uniforms': <String>['child1', 'a', 'child2', 'b'],
      'texture': <String>['u_size', 'u_texture'],
      'mat3_array_uniform': <String>['colors'],
    };
    for (final MapEntry<String, List<String>> e in uniforms.entries) {
      final FragmentShader s = (await load(e.key)).fragmentShader();
      for (final String u in e.value) {
        final row = <Object?>[];
        for (var i = -1; i < 20; i++) {
          try {
            row.add(s.getUniformFloat(u, i).shaderIndex);
          } catch (err) {
            row.add('${errorType(err)}: $err');
          }
        }
        try {
          row.add(s.getImageSampler(u).shaderIndex);
        } catch (err) {
          row.add('${errorType(err)}: $err');
        }
        reflect['${e.key}.$u'] = row;
      }
      reflect['${e.key}.nope'] = errT(() => s.getUniformFloat('nope'));
      reflect['${e.key}.nope.sampler'] = errT(() => s.getImageSampler('nope'));
    }
    out['reflect'] = reflect;

    // Typed getters and arrays.
    final FragmentShader all = (await load('all_uniforms')).fragmentShader();
    final FragmentShader vec3 = (await load('vec3_uniform')).fragmentShader();
    final FragmentShader mat3a = (await load('mat3_array_uniform')).fragmentShader();
    final getters = <String, Object? Function(FragmentShader, String)>{
      'vec2': (s, n) => s.getUniformVec2(n),
      'vec3': (s, n) => s.getUniformVec3(n),
      'vec4': (s, n) => s.getUniformVec4(n),
      'mat2': (s, n) => s.getUniformMat2(n),
      'mat3': (s, n) => s.getUniformMat3(n),
      'mat4': (s, n) => s.getUniformMat4(n),
      'floatArray': (s, n) => s.getUniformFloatArray(n).length,
      'vec2Array': (s, n) => s.getUniformVec2Array(n).length,
      'vec3Array': (s, n) => s.getUniformVec3Array(n).length,
      'vec4Array': (s, n) => s.getUniformVec4Array(n).length,
      'mat2Array': (s, n) => s.getUniformMat2Array(n).length,
      'mat3Array': (s, n) => s.getUniformMat3Array(n).length,
      'mat4Array': (s, n) => s.getUniformMat4Array(n).length,
    };
    final typed = <String, Object?>{};
    void typedCase(String key, Object? Function() f) {
      try {
        final Object? r = f();
        typed[key] = r is int ? r : 'ok';
      } catch (e) {
        typed[key] = '${errorType(e)}: $e';
      }
    }

    for (final MapEntry<String, Object? Function(FragmentShader, String)> g in getters.entries) {
      for (final String u in uniforms['all_uniforms']!) {
        typedCase('${g.key}.$u', () => g.value(all, u));
      }
      typedCase('${g.key}.vec3.color_rgb', () => g.value(vec3, 'color_rgb'));
      typedCase('${g.key}.mat3a.colors', () => g.value(mat3a, 'colors'));
    }
    out['typed'] = typed;

    // Elements of arrays: the shader index of their slots, through set.
    final FragmentShader arr = (await load('all_uniforms')).fragmentShader();
    final UniformArray<UniformVec3Slot> v3a = arr.getUniformVec3Array('uVec3Array');
    out['array.index.10'] = errT(() => v3a[10]);
    out['array.index.-1'] = errT(() => v3a[-1]);
    out['array.length'] = v3a.length;
    out['array.empty'] = errT(() => (vec3.getUniformFloatArray('color_rgb'), arr.getUniformMat4Array('uMat4')[1]));

    // setFloat and errors.
    final FragmentShader ord = (await load('uniform_ordering')).fragmentShader();
    out['setFloat.3'] = errT(() => ord.setFloat(3, 1));
    out['setFloat.-1'] = errT(() => ord.setFloat(-1, 1));
    final Image bg = blueGreenImage();
    out['setImageSampler.1'] = errT(() => ord.setImageSampler(1, bg));
    out['setImageSampler.-1'] = errT(() => ord.setImageSampler(-1, bg));
    out['paint.missingSampler'] = errT(() => Paint().shader = (ord..setFloat(0, 1)));
    final FragmentShader disposed = (await load('uniform_ordering')).fragmentShader()..dispose();
    out['disposed.setFloat'] = errT(() => disposed.setFloat(0, 1));
    out['disposed.setImageSampler'] = errT(() => disposed.setImageSampler(0, bg));
    out['disposed.dispose'] = errT(disposed.dispose);
    out['disposed.paint'] = errT(() => Paint().shader = disposed);
    out['disposed.debugDisposed'] = disposed.debugDisposed;
    out['disposed.getUniformFloat'] = errT(() => disposed.getUniformFloat('b').shaderIndex);

    // fromAsset errors.
    Future<String> loadErr(String key) async {
      try {
        await FragmentProgram.fromAsset(key);
        return 'no error';
      } catch (e) {
        return '${errorType(e)}: $e';
      }
    }

    out['fromAsset.missing'] = await loadErr('shaders/missing.frag');
    out['fromAsset.invalid'] = await loadErr('<invalid> #?');
    out['fromAsset.cached'] = identical(await load('texture'), await load('texture'));

    // ImageFilter.shader.
    for (final String name in <String>['no_uniforms', 'missing_size', 'missing_texture', 'filter_shader', 'children_and_uniforms']) {
      final FragmentShader s = (await load(name)).fragmentShader();
      out['filter.$name'] = errT(() => ImageFilter.shader(s));
    }
    final FragmentShader fs = (await load('filter_shader')).fragmentShader();
    final filter = ImageFilter.shader(fs);
    final filter2 = ImageFilter.shader(fs);
    final low = ImageFilter.shader(fs, filterQuality: FilterQuality.low);
    out['filter.toString'] = hashless(filter.toString());
    out['filter.toString.low'] = hashless(low.toString());
    out['filter.toString.hashIsShader'] = filter.toString().contains('#${fs.hashCode},');
    out['filter.debugShortDescription'] = (filter as dynamic).debugShortDescription;
    out['filter.eq'] = <bool>[
      filter == filter,
      filter == filter2,
      filter == low,
      low == ImageFilter.shader(fs, filterQuality: FilterQuality.low),
      filter == ImageFilter.blur(),
    ];
    fs.setFloat(0, 1);
    final filter3 = ImageFilter.shader(fs);
    out['filter.eq.afterSet'] = <bool>[filter == filter3, filter2 == filter, filter3 == ImageFilter.shader(fs)];
    out['isShaderFilterSupported'] = ImageFilter.isShaderFilterSupported;

    // Pixels.
    final FragmentShader u = (await load('uniforms')).fragmentShader()
      ..setFloat(0, 0.0)
      ..setFloat(1, 0.25)
      ..setFloat(2, 0.75)
      ..setFloat(3, 0.0)
      ..setFloat(4, 0.0)
      ..setFloat(5, 0.0)
      ..setFloat(6, 1.0);
    out['px.uniforms'] = await pixels(u, 4, 4);
    final FragmentShader u2 = (await load('uniforms')).fragmentShader()
      ..getUniformFloat('iFloatUniform').set(0.1)
      ..getUniformVec2('iVec2Uniform').set(0.3, 0.55)
      ..getUniformMat2('iMat2Uniform').set(0, 0, 0, 0.9);
    out['px.uniforms.slots'] = await pixels(u2, 2, 2);

    final FragmentShader bar = (await load('all_uniforms')).fragmentShader();
    for (var i = 0; i < 429; i++) {
      bar.setFloat(i, ((i * 37) % 256) / 255.0);
    }
    out['px.all_uniforms'] = await pixels(bar, 143, 1);

    final FragmentShader green = (await load('blue_green_sampler')).fragmentShader()..setImageSampler(0, bg);
    out['px.blue_green_sampler'] = await pixels(green, 4, 4);

    final Image quad = quadrantImage();
    final FragmentShader tex = (await load('texture')).fragmentShader()
      ..setFloat(0, 4)
      ..setFloat(1, 4)
      ..setImageSampler(0, quad);
    out['px.texture'] = await pixels(tex, 4, 4);

    final FragmentShader ordered = (await load('uniform_ordering')).fragmentShader()
      ..setFloat(0, 1)
      ..setFloat(1, 2)
      ..setFloat(2, 3)
      ..setImageSampler(0, bg);
    out['px.uniform_ordering'] = await pixels(ordered, 2, 2);

    final FragmentShader children = (await load('children_and_uniforms')).fragmentShader()
      ..setFloat(0, 0)
      ..setFloat(1, 1)
      ..setImageSampler(0, bg)
      ..setImageSampler(1, bg);
    out['px.children_and_uniforms'] = await pixels(children, 2, 2);

    final FragmentShader sdf = (await load('circle_sdf')).fragmentShader()
      ..setFloat(0, 8)
      ..setFloat(1, 8);
    out['px.circle_sdf'] = await pixels(sdf, 8, 8);

    final FragmentShader mat = (await load('uniform_mat2')).fragmentShader()
      ..setFloat(0, 4)
      ..setFloat(1, 8)
      ..setFloat(2, 16)
      ..setFloat(3, 32);
    out['px.uniform_mat2'] = await pixels(mat, 2, 2);

    final FragmentShader fn = (await load('functions')).fragmentShader()..setFloat(0, 1);
    out['px.functions'] = await pixels(fn, 2, 2);
    fn.setFloat(0, 0);
    out['px.functions.reused'] = await pixels(fn, 2, 2);

    out['px.no_uniforms'] = await pixels((await load('no_uniforms')).fragmentShader(), 2, 2);

    final FragmentShader swap = (await load('filter_shader')).fragmentShader();
    out['px.filter.red'] = await pixels(null, 2, 2, filter: ImageFilter.shader(swap));
    out['px.filter.color'] = await pixels(null, 2, 2, filter: ImageFilter.shader(swap), color: const Color(0x80336699));

    File('build/fragment_shader.json')
      ..createSync(recursive: true)
      ..writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
