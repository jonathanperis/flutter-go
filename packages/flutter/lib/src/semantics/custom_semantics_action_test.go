package semantics_test

// Port of packages/flutter/test/semantics/custom_semantics_action_test.dart
// (Flutter 3.50.1). No skips.

import (
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/semantics"
)

func TestCustomSemanticsActionCanonicalID(t *testing.T) {
	action1 := semantics.NewCustomSemanticsAction("test")
	action2 := semantics.NewCustomSemanticsAction("test")
	action3 := semantics.NewCustomSemanticsAction("not test")
	id1 := semantics.CustomSemanticsActionGetIdentifier(action1)
	id2 := semantics.CustomSemanticsActionGetIdentifier(action2)
	id3 := semantics.CustomSemanticsActionGetIdentifier(action3)
	expectEqual(t, id1, id2)
	if id2 == id3 {
		t.Fatal("ids equal")
	}
	expectEqual(t, *semantics.CustomSemanticsActionGetAction(id1), action1)
	expectEqual(t, *semantics.CustomSemanticsActionGetAction(id2), action1)
	expectEqual(t, *semantics.CustomSemanticsActionGetAction(id3), action3)
}
