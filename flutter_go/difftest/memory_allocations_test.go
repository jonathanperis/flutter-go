package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

func eventType(e foundation.ObjectEvent) string {
	switch e.(type) {
	case *foundation.ObjectCreated:
		return "ObjectCreated"
	case *foundation.ObjectDisposed:
		return "ObjectDisposed"
	}
	return "?"
}

// Same steps as flutter/test/memory_allocations_test.dart.
func TestMemoryAllocationsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/memory_allocations.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	var got []string
	log := func(s string) { got = append(got, s) }
	ma := foundation.FlutterMemoryAllocationsInstance
	hooks := func(when string) {
		log(fmt.Sprintf("%s: hasListeners=%v hooks=%v,%v,%v,%v", when, ma.HasListeners(),
			ui.ImageOnCreate != nil, ui.ImageOnDispose != nil, ui.PictureOnCreate != nil, ui.PictureOnDispose != nil))
	}
	hooks("start")
	ma.DispatchObjectCreated("l", "C", "nobody")

	a := func(e foundation.ObjectEvent) { log(fmt.Sprintf("a %s %v", eventType(e), e.Object())) }
	b := func(e foundation.ObjectEvent) { log(fmt.Sprintf("b %s %v", eventType(e), e.Object())) }
	var c func(foundation.ObjectEvent)
	c = func(e foundation.ObjectEvent) {
		log(fmt.Sprintf("c %v: removes b and c, adds a", e.Object()))
		ma.RemoveListener(b)
		ma.RemoveListener(c)
		ma.AddListener(a)
		log(fmt.Sprintf("c sees hasListeners=%v", ma.HasListeners()))
	}
	ma.AddListener(a)
	hooks("a added")
	ma.AddListener(b)
	ma.AddListener(c)
	ma.AddListener(b)
	ma.DispatchObjectCreated("package:x/y.dart", "Thing", "o1")
	hooks("after dispatch 1")
	ma.DispatchObjectDisposed("o2")
	ma.RemoveListener(a)
	hooks("a removed")

	for _, e := range []foundation.ObjectEvent{
		foundation.NewObjectCreated("lib", "Cls", "obj"),
		foundation.NewObjectDisposed("obj"),
	} {
		for k, v := range e.ToMap() {
			log(fmt.Sprintf("toMap %v %s", k, dart.JSONEncode(v)))
		}
	}

	onError := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		log(fmt.Sprintf("reported %v library=%s context=%s", details.Exception, *details.Library, details.Context))
		for node := range details.InformationCollector() {
			log(fmt.Sprintf("  info %s", node))
		}
	}
	bad := func(foundation.ObjectEvent) { panic(dart.StateError{Message: "bad listener"}) }
	ma.AddListener(bad)
	ma.AddListener(a)
	ma.DispatchObjectDisposed(42)
	ma.DispatchObjectDisposed(1.5)
	ma.RemoveListener(bad)
	ma.RemoveListener(a)
	foundation.FlutterErrorOnError = onError
	hooks("end")

	l2 := func(foundation.ObjectEvent) { log("l2") }
	l1 := func(foundation.ObjectEvent) {
		log("l1")
		ma.RemoveListener(l2)
		log(fmt.Sprintf("l1 sees hasListeners=%v", ma.HasListeners()))
	}
	ma.AddListener(l1)
	ma.AddListener(l2)
	ma.DispatchObjectDisposed("x")
	ma.RemoveListener(l1)
	hooks("after l1")

	timings := foundation.NewAggregatedTimings([]foundation.TimedBlock{
		foundation.NewTimedBlock("b", 0.1, 0.3),
		foundation.NewTimedBlock("a", 1, 1.1),
		foundation.NewTimedBlock("b", 0.2, 0.7),
		foundation.NewTimedBlock("c", 5, 5),
		foundation.NewTimedBlock("a", 1e15, 1e15+0.3),
		foundation.NewTimedBlock("b", 3, 3.3),
	})
	for _, block := range timings.TimedBlocks {
		log(block.String())
	}
	for _, block := range timings.AggregatedBlocks() {
		log(block.String())
	}
	log(timings.GetAggregated("missing").String())
	log(timings.GetAggregated("b").String())

	if !reflect.DeepEqual(got, ref.Log) {
		t.Errorf("Go:   %q\nDart: %q", got, ref.Log)
	}
}
