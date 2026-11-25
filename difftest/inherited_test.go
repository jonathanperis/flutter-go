package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	. "github.com/jonathanperis/flutter-go/widgets"
	"github.com/jonathanperis/flutter-go/widgettest"
)

var buildLog []string

type value struct {
	Value int
	Child Widget
}

func (w value) ChildWidget() Widget                { return w.Child }
func (w value) UpdateShouldNotify(old Widget) bool { return old.(value).Value != w.Value }

type leaf struct{ Name string }

func (w leaf) Build(*Element) Widget { buildLog = append(buildLog, w.Name); return SizedBox{} }

type dependent struct{ Name string }

func (w dependent) Build(ctx *Element) Widget {
	v, _ := DependOn[value](ctx)
	buildLog = append(buildLog, fmt.Sprintf("%s=%d", w.Name, v.Value))
	return SizedBox{}
}

type holder struct {
	Name  string
	Child Widget
}

func (w holder) Build(*Element) Widget { buildLog = append(buildLog, w.Name); return w.Child }

type root struct{ Child Widget }

func (root) CreateState() State { return &rootState{} }

type rootState struct {
	StateOf[root]
	v, c int
}

func (s *rootState) set(v, c int) { s.SetState(func() { s.v, s.c = v, c }) }
func (s *rootState) Build(*Element) Widget {
	buildLog = append(buildLog, fmt.Sprintf("Root v=%d c=%d", s.v, s.c))
	return value{Value: s.v, Child: s.Widget().Child}
}

func subtree() Widget {
	return Column{Children: []Widget{leaf{"A"}, dependent{"B"}, holder{"C", dependent{"D"}}}}
}

// goSteps runs the same steps as flutter/test/inherited_test.dart.
func goSteps(t *testing.T) [][]string {
	tester := widgettest.New(t)
	var steps [][]string
	step := func(f func()) {
		buildLog = nil
		f()
		steps = append(steps, append([]string{}, buildLog...))
	}
	child := subtree()
	state := func() *rootState {
		return StateOfElement(tester.FindAll(widgettest.ByType[root]())[0]).(*rootState)
	}
	step(func() { tester.PumpWidget(root{child}) })
	step(func() { state().set(1, 0); tester.Pump() })
	step(func() { state().set(1, 1); tester.Pump() })
	step(func() { tester.PumpWidget(root{child}) })
	step(func() { state().set(2, 1); tester.Pump() })
	step(func() { tester.PumpWidget(root{subtree()}) })
	return steps
}

func TestInheritedRebuildsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/inherited.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want [][]string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	got := goSteps(t)
	for i := 0; i < 5; i++ { // steps 0–4 must match exactly
		if !reflect.DeepEqual(nonNil(got[i]), nonNil(want[i])) {
			t.Errorf("step %d: flutter %v, go %v", i, want[i], got[i])
		}
	}
	// Step 5 is a deliberate divergence: a new subtree equal to the old one.
	// Flutter compares widgets by identity and rebuilds all of it; flutter-go
	// compares by value and rebuilds nothing below Root.
	if !reflect.DeepEqual(got[5], []string{"Root v=2 c=1"}) {
		t.Errorf("step 5 (equal subtree): go %v, want only Root (flutter: %v)", got[5], want[5])
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
