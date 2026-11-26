// Port of packages/flutter/test/foundation/double_precision_test.dart (both
// tests). The difftest double_precision compares toStringAsPrecision and
// debugFormatDouble with Dart on 1098 values × 21 precisions.
package foundation

import "testing"

func TestDebugFormatDoubleFormatsDoubles(t *testing.T) {
	equal(t, DebugFormatDouble(ptr(1.0)), "1.0")
	equal(t, DebugFormatDouble(ptr(1.1)), "1.1")
	equal(t, DebugFormatDouble(nil), "null")
}

func TestDebugDoublePrecisionControlsPrecision(t *testing.T) {
	DebugDoublePrecision = ptr(3)
	defer func() { DebugDoublePrecision = nil }()
	equal(t, DebugFormatDouble(ptr(1.0)), "1.00")
}
