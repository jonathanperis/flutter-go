package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	r "github.com/jonathanperis/flutter-go/rendering"
	. "github.com/jonathanperis/flutter-go/widgets"
	"github.com/jonathanperis/flutter-go/widgettest"
)

var paints = map[string]int{}

type counter struct{ name string }

func (c *counter) Paint(r.Canvas, r.Size) { paints[c.name]++ }

func counted(name string, n *Notifier) Widget {
	return CustomPaint{Painter: &counter{name}, Repaint: n, Size: r.Size{W: 10, H: 10}}
}

func runRepaint(t *testing.T, w Widget, steps ...func()) []map[string]int {
	tester := widgettest.New(t)
	snap := func() map[string]int {
		m := map[string]int{}
		for k, v := range paints {
			m[k] = v
		}
		return m
	}
	paints = map[string]int{}
	tester.PumpWidget(w)
	frames := []map[string]int{snap()}
	for _, s := range steps {
		paints = map[string]int{}
		s()
		tester.Pump()
		frames = append(frames, snap())
	}
	return frames
}

var repaintScenes = map[string]func(t *testing.T) []map[string]int{
	"sibling_in_boundary": func(t *testing.T) []map[string]int {
		a, b := &Notifier{}, &Notifier{}
		return runRepaint(t, Column{Children: []Widget{counted("A", a), RepaintBoundary{Child: counted("B", b)}}},
			b.Notify, a.Notify, func() {})
	},
	"no_boundary": func(t *testing.T) []map[string]int {
		a, b := &Notifier{}, &Notifier{}
		return runRepaint(t, Column{Children: []Widget{counted("A", a), counted("B", b)}}, b.Notify, a.Notify)
	},
	"nested_boundaries": func(t *testing.T) []map[string]int {
		a, b, c := &Notifier{}, &Notifier{}, &Notifier{}
		return runRepaint(t, Column{Children: []Widget{
			counted("A", a),
			RepaintBoundary{Child: Column{Children: []Widget{counted("B", b), RepaintBoundary{Child: counted("C", c)}}}},
		}}, c.Notify, b.Notify, a.Notify)
	},
	"list_scroll": func(t *testing.T) []map[string]int {
		ctrl := NewScrollController(0)
		return runRepaint(t, ListViewBuilder{ItemCount: 100, ItemExtent: 50, Controller: ctrl,
			ItemBuilder: func(_ *Element, i int) Widget { return CustomPaint{Painter: &counter{fmt.Sprint("item", i)}} }},
			func() { ctrl.JumpTo(20) }, func() { ctrl.JumpTo(40) }, func() { ctrl.JumpTo(1000) })
	},
}

func TestRepaintsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/repaint.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string][]map[string]int
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	for name, run := range repaintScenes {
		t.Run(name, func(t *testing.T) {
			got := run(t)
			for i := range want[name] {
				if !reflect.DeepEqual(got[i], want[name][i]) {
					t.Errorf("frame %d: flutter %v\n            go      %v", i, want[name][i], got[i])
				}
			}
		})
	}
}
