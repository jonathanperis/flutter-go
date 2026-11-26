package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

func goParagraphs(text string) (out any) {
	defer func() {
		if r := recover(); r != nil {
			out = "ERROR"
		}
	}()
	list := []any{}
	for p := range foundation.NewLicenseEntryWithLineBreaks(nil, text).Paragraphs() {
		list = append(list, []any{p.Text, float64(p.Indent)})
	}
	return list
}

// Same inputs and registry scene as flutter/test/licenses_test.dart (the
// inputs are read from its output).
func TestLicensesMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/licenses.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct {
		Inputs     []string
		Paragraphs []any
		Registry   []string
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	mismatches := 0
	for i, text := range ref.Inputs {
		if got := goParagraphs(text); !reflect.DeepEqual(got, ref.Paragraphs[i]) {
			if mismatches++; mismatches <= 5 {
				g, _ := json.Marshal(got)
				w, _ := json.Marshal(ref.Paragraphs[i])
				t.Errorf("input %d %q:\n Go:   %.500s\n Dart: %.500s", i, text[:min(len(text), 80)], g, w)
			}
		}
	}

	l := async.NewLoop()
	restore := async.SetCurrent(l)
	defer restore()
	defer foundation.LicenseRegistry.Reset()
	var registry []string
	log := func(s string) { registry = append(registry, s) }
	entry := func(packages []string, text string) foundation.LicenseEntry {
		return foundation.NewLicenseEntryWithLineBreaks(packages, text)
	}
	foundation.LicenseRegistry.Licenses().Listen(func(foundation.LicenseEntry) { log("never") },
		async.StreamListenOptions{OnDone: func() { log("empty registry done") }})
	foundation.LicenseRegistry.AddLicense(func() async.Stream[foundation.LicenseEntry] {
		return async.StreamFromIterable(slices.Values([]foundation.LicenseEntry{
			entry([]string{"a"}, "A"), entry([]string{"b"}, "B\n\nB2"),
		}))
	})
	foundation.LicenseRegistry.AddLicense(func() async.Stream[foundation.LicenseEntry] {
		c := async.NewStreamController[foundation.LicenseEntry]()
		async.NewTimer(2*time.Millisecond, func() {
			log("timer adds C")
			c.Add(entry([]string{"c", "c2"}, "C"))
			c.Close()
		})
		return c.Stream()
	})
	foundation.LicenseRegistry.AddLicense(func() async.Stream[foundation.LicenseEntry] {
		return async.StreamEmpty[foundation.LicenseEntry]()
	})
	foundation.LicenseRegistry.AddLicense(func() async.Stream[foundation.LicenseEntry] {
		return async.StreamFromIterable(slices.Values([]foundation.LicenseEntry{entry(nil, "D")}))
	})
	for _, name := range []string{"first", "second"} {
		foundation.LicenseRegistry.Licenses().Listen(func(e foundation.LicenseEntry) {
			var texts []string
			for p := range e.Paragraphs() {
				texts = append(texts, p.Text)
			}
			log(fmt.Sprintf("%s: %s %s", name, strings.Join(slices.Collect(e.Packages()), ","), strings.Join(texts, "|")))
		}, async.StreamListenOptions{OnDone: func() { log(name + " done") }})
		async.ScheduleMicrotask(func() { log("micro after " + name) })
	}
	log("sync end")
	l.Elapse(5 * time.Millisecond)
	if !reflect.DeepEqual(registry, ref.Registry) {
		t.Errorf("registry:\n Go:   %q\n Dart: %q", registry, ref.Registry)
	}
	if mismatches > 0 {
		t.Errorf("%d of %d inputs differ", mismatches, len(ref.Inputs))
	}
}
