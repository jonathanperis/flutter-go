package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/flutter_go/async"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

type TestBinding struct {
	foundation.BindingBaseBase
	extensions map[string]foundation.ServiceExtensionCallback
	log        func(string)
}

func (b *TestBinding) InitInstances() {
	b.log("initInstances")
	b.BindingBaseBase.InitInstances()
}

func (b *TestBinding) InitServiceExtensions() {
	b.log("initServiceExtensions")
	b.BindingBaseBase.InitServiceExtensions()
}

func (b *TestBinding) RegisterServiceExtension(name string, callback foundation.ServiceExtensionCallback) {
	b.log("register " + name)
	b.extensions[name] = callback
}

func (b *TestBinding) PostEvent(eventKind string, eventData map[string]any) {
	b.log("event " + eventKind + " " + dart.JSONEncode(eventData))
}

func (b *TestBinding) PerformReassemble() async.Future[async.Void] {
	b.log(fmt.Sprintf("performReassemble locked=%v", b.Locked()))
	return b.BindingBaseBase.PerformReassemble()
}

func (b *TestBinding) Unlocked() {
	b.log("unlocked")
	b.BindingBaseBase.Unlocked()
}

func (b *TestBinding) call(name string, parameters map[string]string) {
	b.log("call " + name + " " + dart.JSONEncode(parameters))
	async.OnComplete(b.extensions[name](parameters),
		func(result map[string]any) { b.log("result " + name + " " + dart.JSONEncode(result)) },
		func(err error) { b.log("error " + name + " " + err.Error()) })
}

// Same steps as flutter/test/binding_test.dart.
func TestBindingMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/binding.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FLUTTER_TEST", "true") // defaultTargetPlatform is android under flutter test
	l := async.NewLoop()
	defer async.SetCurrent(l)()
	var got []string
	log := func(s string) { got = append(got, s) }
	defer func() {
		foundation.SetDebugDefaultTargetPlatformOverride(nil)
		foundation.DebugBrightnessOverride = nil
		foundation.ConnectedVmServiceUri = nil
		foundation.ActiveDevToolsServerAddress = nil
	}()

	binding := &TestBinding{extensions: map[string]foundation.ServiceExtensionCallback{}, log: log}
	binding.Construct(binding)
	log(fmt.Sprintf("constructed %s locked=%v", binding, binding.Locked()))
	async.ScheduleMicrotask(func() { log("micro 1") })
	binding.call("reassemble", map[string]string{})
	binding.call("platformOverride", map[string]string{})
	binding.call("platformOverride", map[string]string{"value": "iOS"})
	async.ScheduleMicrotask(func() { log("micro 2") })
	l.FlushMicrotasks()
	log("flushed")
	binding.call("platformOverride", map[string]string{"value": "bogus"})
	binding.call("brightnessOverride", map[string]string{"value": "Brightness.dark"})
	binding.call("brightnessOverride", map[string]string{})
	binding.call("brightnessOverride", map[string]string{"value": "x"})
	binding.call("connectedVmServiceUri", map[string]string{})
	binding.call("connectedVmServiceUri", map[string]string{"value": "http://a/"})
	binding.call("activeDevToolsServerAddress", map[string]string{"value": "http://b/"})
	l.FlushMicrotasks()
	log("flushed")

	binding.LockEvents(func() async.Future[async.Void] {
		log(fmt.Sprintf("callback 1 locked=%v", binding.Locked()))
		return async.FutureDelayed(2*time.Millisecond, func() async.Void { log("lock 1 ends"); return async.Void{} })
	})
	binding.LockEvents(func() async.Future[async.Void] {
		log("callback 2")
		return async.FutureDelayed(1*time.Millisecond, func() async.Void { log("lock 2 ends"); return async.Void{} })
	})
	log(fmt.Sprintf("locked=%v", binding.Locked()))
	l.Elapse(time.Millisecond)
	log(fmt.Sprintf("at 1ms locked=%v", binding.Locked()))
	l.Elapse(time.Millisecond)
	log(fmt.Sprintf("at 2ms locked=%v", binding.Locked()))

	flag := false
	scale := 1.0
	binding.RegisterBoolServiceExtension("flag",
		func() async.Future[bool] { return async.Async(func() bool { return flag }) },
		func(v bool) async.Future[async.Void] {
			return async.AsyncVoid(func() { log(fmt.Sprintf("set flag %v", v)); flag = v })
		})
	binding.RegisterNumericServiceExtension("scale",
		func() async.Future[float64] { return async.Async(func() float64 { return scale }) },
		func(v float64) async.Future[async.Void] { return async.AsyncVoid(func() { scale = v }) })
	binding.RegisterSignalServiceExtension("signal", func() async.Future[async.Void] {
		return async.AsyncVoid(func() { log("signal") })
	})
	binding.call("flag", map[string]string{"enabled": "true"})
	binding.call("flag", map[string]string{})
	binding.call("scale", map[string]string{"scale": "0.1"})
	binding.call("scale", map[string]string{"scale": "1e300"})
	binding.call("signal", map[string]string{})
	l.FlushMicrotasks()
	log("end")

	if !reflect.DeepEqual(got, ref.Log) {
		for i := range max(len(got), len(ref.Log)) {
			var g, w string
			if i < len(got) {
				g = got[i]
			}
			if i < len(ref.Log) {
				w = ref.Log[i]
			}
			if g != w {
				t.Errorf("first difference at %d:\n Go:   %q\n Dart: %q", i, g, w)
				break
			}
		}
		t.Errorf("Go:   %q\nDart: %q", got, ref.Log)
	}
}
