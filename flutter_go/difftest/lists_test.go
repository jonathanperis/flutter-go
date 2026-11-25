package difftest

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"testing"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

type listResult struct {
	Offset float64      `json:"offset"`
	Items  [][5]float64 `json:"items"`
}

func runListScene(t *testing.T, s ListScene) listResult {
	c := widgets.NewScrollController(0)
	tester := flutter_test.New(t)
	tester.PumpWidget(s.Widget(c))
	c.JumpTo(s.JumpTo)
	tester.PumpAndSettle() // out-of-range jumps animate back, as in Flutter

	list := tester.FindAll(flutter_test.ByType[widgets.ListViewBuilder]())[0]
	var items [][5]float64
	for _, child := range list.Children() {
		ro := child.RenderObject()
		o, sz := r.GlobalOffset(ro), ro.Base().Size
		items = append(items, [5]float64{float64(list.LazyIndex(child)), o.X, o.Y, sz.W, sz.H})
	}
	return listResult{c.Offset(), items}
}

func TestListsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/lists.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string]listResult
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(ListScenes))
	for n := range ListScenes {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			ref, ok := want[name]
			if !ok {
				t.Fatal("scene missing from reference")
			}
			got := runListScene(t, ListScenes[name])
			if math.Abs(got.Offset-ref.Offset) > 1e-6 {
				t.Errorf("scroll offset: flutter %v, go %v", ref.Offset, got.Offset)
			}
			if d := compare(ref.Items, got.Items); d != "" {
				t.Errorf("items differ (index, x, y, w, h):\n%s", d)
			} else {
				t.Logf("offset %v, items %v..%v (%d built)", got.Offset, got.Items[0][0], got.Items[len(got.Items)-1][0], len(got.Items))
			}
		})
	}
}
