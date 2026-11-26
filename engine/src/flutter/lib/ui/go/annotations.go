package ui

// Port of engine/src/flutter/lib/ui/annotations.dart (Flutter 3.50.1).

import "github.com/jonathanperis/flutter-go/src/dart"

// KeepToString is dart:ui's keepToString annotation: it asks the Flutter
// tool not to replace the annotated toString in release builds (Locale's
// is). Go has no annotations and no toString stripping, so it only
// documents intent.
var KeepToString = dart.Pragma{Name: "flutter:keep-to-string"}
