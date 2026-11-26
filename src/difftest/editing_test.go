package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type editOp struct {
	Type *string  `json:"type"`
	Key  string   `json:"key"`
	Mods []string `json:"mods"`
	// Select (editing_actions only) is a selection change from the
	// platform: [base, extent] with Affinity.
	Select   []int   `json:"select"`
	Affinity string  `json:"affinity"`
	Reset    *string `json:"reset"`
}

var editKeys = map[string]services.LogicalKeyboardKey{
	"backspace": services.LogicalKeyboardKeyBackspace, "delete": services.LogicalKeyboardKeyDelete,
	"arrowLeft": services.LogicalKeyboardKeyArrowLeft, "arrowRight": services.LogicalKeyboardKeyArrowRight,
	"arrowUp": services.LogicalKeyboardKeyArrowUp, "arrowDown": services.LogicalKeyboardKeyArrowDown,
	"home": services.LogicalKeyboardKeyHome, "end": services.LogicalKeyboardKeyEnd, "keyA": services.LogicalKeyboardKeyKeyA,
}

var editMods = map[string]services.LogicalKeyboardKey{
	"shift": services.LogicalKeyboardKeyShiftLeft, "control": services.LogicalKeyboardKeyControlLeft,
	"alt": services.LogicalKeyboardKeyAltLeft, "meta": services.LogicalKeyboardKeyMetaLeft,
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

// TestTextFieldMatchesFlutter runs the editing scenes as the Dart side
// does: a focused text field (EditableText under DefaultTextEditingShortcuts
// there; the interim TextField with the ported shortcuts and EditableText
// actions here), typing delivered by the platform's text input
// (TestTextInput.updateEditingValue), every other key simulated as key
// events (simulateKeyDownEvent/UpEvent through HardwareKeyboard, the focus
// system's key message handler and the shortcuts; on macOS the keys the
// framework leaves to the platform come back as selectors from
// TestTextInput's macOS key handler). After each step it compares [text,
// baseOffset, extentOffset] (UTF-16).
func TestTextFieldMatchesFlutter(t *testing.T) {
	scenes, want := loadEditing(t)
	names := make([]string, 0, len(scenes))
	for n := range scenes {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			// Scenes named mac_* run with macOS shortcuts.
			if strings.HasPrefix(name, "mac_") {
				mac := foundation.TargetPlatformMacOS
				foundation.SetDebugDefaultTargetPlatformOverride(&mac)
				defer foundation.SetDebugDefaultTargetPlatformOverride(nil)
			}
			tester := flutter_test.New(t)
			tester.Owner.Platform = foundation.DefaultTargetPlatform()
			controller := widgets.NewTextEditingController("")
			tester.PumpWidget(widgets.TextField{Controller: controller})
			tester.Tap(flutter_test.ByType[widgets.TextField]())
			tester.Pump()
			for i, op := range scenes[name] {
				if op.Type != nil {
					v := controller.Value()
					text := dart.Concat(v.Selection.TextBefore(v.Text), *op.Type, v.Selection.TextAfter(v.Text))
					caret := services.NewTextSelectionCollapsed(v.Selection.Start + dart.StringLength(*op.Type))
					tester.Binding().TestTextInput().UpdateEditingValue(services.NewTextEditingValue(services.TextEditingValueOptions{
						Text: text, Selection: &caret}))
					tester.Loop().FlushMicrotasks()
				} else {
					for _, m := range op.Mods {
						tester.SendKeyDownEvent(editMods[m])
					}
					tester.SendKeyEvent(editKeys[op.Key])
					for j := len(op.Mods) - 1; j >= 0; j-- {
						tester.SendKeyUpEvent(editMods[op.Mods[j]])
					}
				}
				tester.Pump()
				v := controller.Value()
				got := fmt.Sprintf("%q %d %d", v.Text, v.Selection.BaseOffset, v.Selection.ExtentOffset)
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
