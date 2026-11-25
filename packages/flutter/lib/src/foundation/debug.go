package foundation

// Partial port of debug.dart (Flutter 3.50.1): debugFormatDouble, which
// diagnostics.dart uses. debugDoublePrecision (toStringAsPrecision) and the
// debugPrint family are not ported yet.

import "github.com/jonathanperis/flutter-go/flutter_go/dart"

// DebugFormatDouble is debugFormatDouble: "null" or the value with one
// decimal (Dart's toStringAsFixed(1)).
func DebugFormatDouble(value *float64) string {
	if value == nil {
		return "null"
	}
	return dart.ToStringAsFixed(*value, 1)
}
