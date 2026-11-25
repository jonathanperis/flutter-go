package difftest

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	r "github.com/jonathanperis/flutter-go/rendering"
	"github.com/jonathanperis/flutter-go/widgettest"
)

const referencePath = "flutter/build/layout.json"

// layout returns depth, x, y, w, h for every render box of a scene.
func layout(t *testing.T, name string) [][5]float64 {
	tester := widgettest.New(t)
	tester.PumpWidget(Scenes[name])
	root := tester.Owner.Root().Children()[0].RenderObject()
	var out [][5]float64
	r.Walk(root, func(b r.RenderBox, depth int) {
		o, s := r.GlobalOffset(b), b.Base().Size
		out = append(out, [5]float64{float64(depth), o.X, o.Y, s.W, s.H})
	})
	return out
}

func TestLayoutMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile(referencePath)
	if err != nil {
		t.Skipf("no reference layouts (%v); run scripts/difftest.sh", err)
	}
	var want map[string][][5]float64
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(Scenes))
	for n := range Scenes {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			ref, ok := want[name]
			if !ok {
				t.Fatalf("scene missing from %s", referencePath)
			}
			if diff := compare(ref, layout(t, name)); diff != "" {
				t.Errorf("layout differs from Flutter:\n%s", diff)
			}
		})
	}
}

func compare(want, got [][5]float64) string {
	var b strings.Builder
	n := max(len(want), len(got))
	bad := false
	for i := 0; i < n; i++ {
		var w, g string
		if i < len(want) {
			w = fmt.Sprint(want[i])
		}
		if i < len(got) {
			g = fmt.Sprint(got[i])
		}
		mark := "  "
		if i >= len(want) || i >= len(got) || !near(want[i], got[i]) {
			mark, bad = "✗ ", true
		}
		fmt.Fprintf(&b, "%sflutter %-32s go %s\n", mark, w, g)
	}
	if !bad {
		return ""
	}
	return b.String()
}

func near(a, b [5]float64) bool {
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-6 {
			return false
		}
	}
	return true
}
