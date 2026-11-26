// Port of packages/flutter/test/foundation/binding_2_test.dart (its one
// test). Dart's `Zone.current.fork().run(...)` is a new current
// async.Loop (the binding remembers the loop it was initialized in).
package foundation

import (
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
)

const zoneMismatchPrefix = "Zone mismatch.\n" +
	"The Flutter bindings were initialized in a different zone than is now being used. " +
	"This will likely cause confusion and bugs as any zone-specific configuration will " +
	"inconsistently use the configuration of the original binding initialization zone " +
	"or this zone based on hard-to-predict factors such as which zone was active when " +
	"a particular callback was set.\n" +
	"It is important to use the same zone when calling `ensureInitialized` on the " +
	"binding as when calling `"

func TestBindingBaseDebugCheckZone(t *testing.T) {
	resetBindingStatics(t)
	binding := newPlainBinding() // TestBinding extends BindingBase
	binding.DebugCheckZone("test1")
	BindingBaseDebugZoneErrorsAreFatal = true
	func() {
		defer async.SetCurrent(async.NewLoop())()
		defer func() {
			err, ok := recover().(*FlutterError)
			if !ok {
				t.Fatal("expected an exception")
			}
			equal(t, err.Error(), zoneMismatchPrefix+"test2` later.\n"+
				"To make this error non-fatal, set BindingBase.debugZoneErrorsAreFatal to false "+
				"before the bindings are initialized (i.e. as the first statement in `void main() { }`).")
		}()
		binding.DebugCheckZone("test2")
	}()
	BindingBaseDebugZoneErrorsAreFatal = false
	func() {
		defer async.SetCurrent(async.NewLoop())()
		sawError := false
		lastHandler := FlutterErrorOnError
		FlutterErrorOnError = func(details *FlutterErrorDetails) {
			err, ok := details.Exception.(*FlutterError)
			if !ok {
				t.Fatalf("exception %#v", details.Exception)
			}
			equal(t, err.Error(), zoneMismatchPrefix+"test3` later.\n"+
				"To make this warning fatal, set BindingBase.debugZoneErrorsAreFatal to true "+
				"before the bindings are initialized (i.e. as the first statement in `void main() { }`).")
			equal(t, details.Context.String(), "during test3")
			sawError = true
		}
		binding.DebugCheckZone("test3")
		equal(t, sawError, true)
		FlutterErrorOnError = lastHandler
	}()
}
