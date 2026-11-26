// Lists the public API of dart:ui and package:flutter's libraries as JSON,
// for src/cmd/apiparity (flutter-go's API parity measure).
//
// Usage (from src/cmd/apiparity/dart, after `flutter pub get --offline`):
//
//   dart run bin/dump.dart <flutter checkout> <library>... [--reexport <package>=<library>]...
//
// where a library is `dart:ui` or a package:flutter library name
// (`foundation`). For each library it walks the export namespace the
// analyzer computes (exports, show/hide and parts resolved), keeps the
// declarations the library owns (see [owned]), and lists each one's public
// declared members (constructors, fields, getters, setters, methods,
// operators, enum values) and the parameters of every constructor, method
// and function. Members a class only inherits (from a superclass or a mixin)
// are not listed: they are counted where Dart declares them.
import 'dart:convert';
import 'dart:io';

import 'package:analyzer/dart/analysis/analysis_context_collection.dart';
import 'package:analyzer/dart/analysis/results.dart';
import 'package:analyzer/dart/element/element.dart';
import 'package:path/path.dart' as p;

Future<void> main(List<String> args) async {
  if (args.isEmpty) {
    stderr.writeln('usage: dump.dart <flutter checkout> <library>... [--reexport <package>=<library>]...');
    exit(2);
  }
  final String flutterRoot = p.normalize(p.absolute(args[0]));
  final List<String> libs = <String>[];
  final Map<String, String> reexported = <String, String>{};
  for (int i = 1; i < args.length; i++) {
    if (args[i] == '--reexport') {
      final List<String> kv = args[++i].split('=');
      reexported[kv[0]] = kv[1];
    } else {
      libs.add(args[i]);
    }
  }
  final String self = p.normalize(p.absolute(p.dirname(p.dirname(Platform.script.toFilePath()))));
  final AnalysisContextCollection collection = AnalysisContextCollection(includedPaths: <String>[self]);
  final session = collection.contextFor(self).currentSession;

  Future<LibraryElement> load(String uri) async {
    final SomeLibraryElementResult result = await session.getLibraryByUri(uri);
    if (result is! LibraryElementResult) {
      stderr.writeln('dump.dart: cannot resolve $uri: $result');
      exit(1);
    }
    return result.element;
  }

  String uriOf(String lib) => lib == 'dart:ui' ? 'dart:ui' : 'package:flutter/$lib.dart';

  final List<Object?> out = <Object?>[];
  for (final String lib in libs) {
    final LibraryElement library = await load(uriOf(lib));
    // A declaration belongs to the library whose src/<lib>/ directory
    // declares it: what a library re-exports from another (widgets exports
    // rendering; src files export foundation's names) is counted there.
    // A third-party package named by --reexport counts for the library that
    // re-exports it (foundation's package:listen).
    bool owned(Element e) {
      final Uri u = e.library!.uri;
      if (lib == 'dart:ui') {
        return u.toString() == 'dart:ui';
      }
      if (u.toString().startsWith('package:flutter/src/$lib/')) {
        return true;
      }
      return u.scheme == 'package' && u.pathSegments.first != 'flutter' && reexported[u.pathSegments.first] == lib;
    }

    final List<Object?> decls = <Object?>[];
    final Map<String, Element> names = library.exportNamespace.definedNames2;
    for (final String name in names.keys.toList()..sort()) {
      final Element e = names[name]!;
      if (name.endsWith('=')) {
        // A setter: listed with its getter or variable, unless alone.
        if (names.containsKey(name.substring(0, name.length - 1))) {
          continue;
        }
      }
      if (!owned(e)) {
        continue;
      }
      final Map<String, Object?>? d = describeTopLevel(e, flutterRoot);
      if (d != null) {
        decls.add(d);
      }
    }
    out.add(<String, Object?>{'name': lib, 'decls': decls});
  }
  stdout.write(const JsonEncoder.withIndent(' ').convert(<String, Object?>{'libraries': out}));
  stdout.writeln();
}

String fileOf(Element e, String flutterRoot) {
  final String full = e.firstFragment.libraryFragment!.source.fullName;
  final String skyUi = p.join(flutterRoot, 'bin/cache/pkg/sky_engine/lib/ui');
  if (p.isWithin(skyUi, full)) {
    // dart:ui comes from the Flutter SDK's sky_engine copy, identical to the
    // engine sources the Go port sits beside.
    return p.join('engine/src/flutter/lib/ui', p.relative(full, from: skyUi));
  }
  if (p.isWithin(flutterRoot, full)) {
    return p.relative(full, from: flutterRoot);
  }
  final String u = e.library!.uri.toString();
  final Uri parsed = Uri.parse(u);
  if (parsed.scheme == 'package') {
    // A re-exported package: the file's package URI.
    final int i = full.lastIndexOf('/lib/');
    return 'package:${parsed.pathSegments.first}/${full.substring(i + 5)}';
  }
  return full;
}

bool deprecated(Element e) => e.metadata.hasDeprecated;

List<Object?> params(List<FormalParameterElement> ps) => <Object?>[
      for (final FormalParameterElement fp in ps)
        <String, Object?>{
          'name': fp.name ?? '',
          if (fp.isNamed) 'named': true,
          if (fp.isRequired) 'required': true,
        },
    ];

Map<String, Object?>? describeTopLevel(Element e, String flutterRoot) {
  final Map<String, Object?> d = <String, Object?>{
    'name': e.name,
    'file': fileOf(e, flutterRoot),
  };
  final Uri lib = e.library!.uri;
  if (lib.scheme == 'package' && lib.pathSegments.first != 'flutter') {
    d['package'] = lib.pathSegments.first;
  }
  if (deprecated(e)) {
    d['deprecated'] = true;
  }
  switch (e) {
    case ClassElement():
      d['kind'] = 'class';
      if (e.isAbstract) d['abstract'] = true;
      if (e.isSealed) d['sealed'] = true;
      if (e.isMixinClass) d['mixinClass'] = true;
    case MixinElement():
      d['kind'] = 'mixin';
    case EnumElement():
      d['kind'] = 'enum';
    case ExtensionTypeElement():
      d['kind'] = 'extension type';
    case ExtensionElement():
      d['kind'] = 'extension';
    case TypeAliasElement():
      d['kind'] = 'typedef';
    case TopLevelFunctionElement():
      d['kind'] = 'function';
      if (e.typeParameters.isNotEmpty) d['generic'] = true;
      d['params'] = params(e.formalParameters);
      return d;
    case GetterElement():
      final PropertyInducingElement? v = e.variable;
      if (v != null && !e.isOriginDeclaration) {
        d['kind'] = 'variable';
        d['name'] = v.name;
        if (v.setter != null) d['settable'] = true;
        if (deprecated(v)) d['deprecated'] = true;
      } else {
        d['kind'] = 'getter';
      }
      return d;
    case SetterElement():
      d['kind'] = 'setter';
      d['name'] = e.name;
      return d;
    default:
      stderr.writeln('dump.dart: unexpected ${e.runtimeType} ${e.name}');
      return null;
  }
  if (e is InstanceElement) {
    d['members'] = members(e);
  }
  return d;
}

List<Object?> members(InstanceElement e) {
  final List<(int, Map<String, Object?>)> out = <(int, Map<String, Object?>)>[];
  void add(Element m, Map<String, Object?> desc) {
    if (deprecated(m)) desc['deprecated'] = true;
    out.add((m.firstFragment.offset, desc));
  }

  if (e is InterfaceElement && e is! EnumElement) {
    for (final ConstructorElement c in e.constructors) {
      if (!c.isOriginDeclaration || c.isPrivate) {
        continue;
      }
      final String name = c.name == null || c.name == 'new' ? '' : c.name!;
      add(c, <String, Object?>{
        'name': name,
        'kind': 'constructor',
        if (c.isFactory) 'factory': true,
        if (c.isConst) 'const': true,
        'params': params(c.formalParameters),
      });
    }
  }
  for (final FieldElement f in e.fields) {
    if (f.isPrivate || !(f.isOriginDeclaration || f.isEnumConstant)) {
      continue;
    }
    add(f, <String, Object?>{
      'name': f.name,
      'kind': f.isEnumConstant ? 'enumValue' : 'field',
      if (f.isStatic && !f.isEnumConstant) 'static': true,
      if (f.setter != null) 'settable': true,
    });
  }
  for (final GetterElement g in e.getters) {
    if (g.isPrivate || !g.isOriginDeclaration) {
      continue;
    }
    add(g, <String, Object?>{'name': g.name, 'kind': 'getter', if (g.isStatic) 'static': true});
  }
  for (final SetterElement s in e.setters) {
    if (s.isPrivate || !s.isOriginDeclaration) {
      continue;
    }
    add(s, <String, Object?>{'name': s.name, 'kind': 'setter', if (s.isStatic) 'static': true});
  }
  for (final MethodElement m in e.methods) {
    if (m.isPrivate || !m.isOriginDeclaration) {
      continue;
    }
    add(m, <String, Object?>{
      'name': m.name,
      'kind': m.isOperator ? 'operator' : 'method',
      if (m.isStatic) 'static': true,
      if (m.typeParameters.isNotEmpty) 'generic': true,
      'params': params(m.formalParameters),
    });
  }
  out.sort((a, b) => a.$1.compareTo(b.$1));
  return <Object?>[for (final (int, Map<String, Object?>) m in out) m.$2];
}
