package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

type editOp struct {
	Type *string  `json:"type"`
	Key  string   `json:"key"`
	Mods []string `json:"mods"`
}

var keyNames = map[string]services.Key{
	"backspace": services.KeyBackspace, "delete": services.KeyDelete,
	"arrowLeft": services.KeyLeft, "arrowRight": services.KeyRight,
	"arrowUp": services.KeyUp, "arrowDown": services.KeyDown,
	"home": services.KeyHome, "end": services.KeyEnd, "keyA": services.KeyA,
}

func (o editOp) event() services.KeyEvent {
	ev := services.KeyEvent{Key: keyNames[o.Key]}
	for _, m := range o.Mods {
		switch m {
		case "shift":
			ev.Shift = true
		case "control":
			ev.Ctrl = true
		case "alt":
			ev.Alt = true
		case "meta":
			ev.Meta = true
		}
	}
	return ev
}

func loadEditing(t *testing.T) (map[string][]editOp, map[string][][3]any) {
	data, err := os.ReadFile("editing_scenes.json")
	if err != nil {
		t.Fatal(err)
	}
	var scenes map[string][]editOp
	if err := json.Unmarshal(data, &scenes); err != nil {
		t.Fatal(err)
	}
	ref, err := os.ReadFile("flutter/build/editing.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string][][3]any
	if err := json.Unmarshal(ref, &want); err != nil {
		t.Fatal(err)
	}
	return scenes, want
}

// compareEditing runs each scene through step and compares every state.
func compareEditing(t *testing.T, step func(scene string, op editOp) services.Value) {
	scenes, want := loadEditing(t)
	names := make([]string, 0, len(scenes))
	for n := range scenes {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			for i, op := range scenes[name] {
				v := step(name, op)
				got := fmt.Sprintf("%q %d %d", v.Text, v.Selection.Base, v.Selection.Extent)
				w := want[name][i]
				exp := fmt.Sprintf("%q %d %d", w[0], int(w[1].(float64)), int(w[2].(float64)))
				if got != exp {
					desc := op.Key
					if op.Type != nil {
						desc = "type " + *op.Type
					}
					t.Errorf("step %d (%s %v): flutter %s, go %s", i, desc, op.Mods, exp, got)
				}
			}
		})
	}
}

// platformFor: scenes named mac_* use macOS shortcuts, like the Dart side.
func platformFor(scene string) services.Platform {
	if strings.HasPrefix(scene, "mac_") {
		return services.MacOS
	}
	return services.Android
}

// TestEditingModelMatchesFlutter checks the textedit package on its own.
func TestEditingModelMatchesFlutter(t *testing.T) {
	values := map[string]services.Value{}
	compareEditing(t, func(scene string, op editOp) services.Value {
		v := values[scene]
		if op.Type != nil {
			v = services.InsertText(v, *op.Type)
		} else {
			v, _ = services.HandleKey(v, op.event(), platformFor(scene), nil)
		}
		values[scene] = v
		return v
	})
}

// TestTextFieldMatchesFlutter runs the same scenes through the widget stack:
// a TextField focused by a tap, keys delivered through the focus system.
func TestTextFieldMatchesFlutter(t *testing.T) {
	fields := map[string]*flutter_test.Tester{}
	ctrls := map[string]*widgets.TextEditingController{}
	compareEditing(t, func(scene string, op editOp) services.Value {
		tester, ok := fields[scene]
		if !ok {
			tester = flutter_test.New(t)
			tester.Owner.Platform = platformFor(scene)
			ctrls[scene] = widgets.NewTextEditingController("")
			tester.PumpWidget(widgets.TextField{Controller: ctrls[scene]})
			tester.Tap(flutter_test.ByType[widgets.TextField]())
			tester.Pump()
			fields[scene] = tester
		}
		if op.Type != nil {
			tester.EnterText(*op.Type)
		} else {
			tester.SendKey(op.event())
		}
		return ctrls[scene].Value()
	})
}
