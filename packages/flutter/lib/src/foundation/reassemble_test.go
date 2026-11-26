// Port of packages/flutter/test/foundation/reassemble_test.dart (its one
// test).
package foundation

import (
	"testing"

	"github.com/jonathanperis/flutter-go/flutter_go/async"
)

type testFoundationFlutterBinding struct {
	plainBinding // overrides RegisterServiceExtension (see binding_test.go)
	wasLocked    *bool
}

func (b *testFoundationFlutterBinding) PerformReassemble() async.Future[async.Void] {
	return async.Async(func() async.Void {
		locked := b.Locked()
		b.wasLocked = &locked
		return b.plainBinding.PerformReassemble().Await()
	})
}

func TestPointerEventsAreLockedDuringReassemble(t *testing.T) {
	resetBindingStatics(t)
	binding := &testFoundationFlutterBinding{}
	binding.Construct(binding)
	if err := runLoop(t, func() { binding.ReassembleApplication().Await() }); err != nil {
		t.Fatal(err)
	}
	if binding.wasLocked == nil || !*binding.wasLocked {
		t.Fatal("not locked during reassemble")
	}
	equal(t, binding.Locked(), false)
}
