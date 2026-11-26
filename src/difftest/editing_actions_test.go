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
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

var actionKeys = map[string]services.LogicalKeyboardKey{
	"pageUp": services.LogicalKeyboardKeyPageUp, "pageDown": services.LogicalKeyboardKeyPageDown,
	"insert": services.LogicalKeyboardKeyInsert, "space": services.LogicalKeyboardKeySpace,
	"enter": services.LogicalKeyboardKeyEnter, "tab": services.LogicalKeyboardKeyTab,
	"escape": services.LogicalKeyboardKeyEscape, "keyB": services.LogicalKeyboardKeyKeyB,
	"keyC": services.LogicalKeyboardKeyKeyC, "keyE": services.LogicalKeyboardKeyKeyE,
	"keyF": services.LogicalKeyboardKeyKeyF, "keyN": services.LogicalKeyboardKeyKeyN,
	"keyP": services.LogicalKeyboardKeyKeyP, "keyT": services.LogicalKeyboardKeyKeyT,
	"keyV": services.LogicalKeyboardKeyKeyV, "keyX": services.LogicalKeyboardKeyKeyX,
}

func actionKey(name string) services.LogicalKeyboardKey {
	if k, ok := editKeys[name]; ok {
		return k
	}
	return actionKeys[name]
}

// actionScenePlatform is the Dart side's platformFor: the scene's prefix,
// android otherwise.
func actionScenePlatform(scene string) *foundation.TargetPlatform {
	for prefix, p := range map[string]foundation.TargetPlatform{"mac_": foundation.TargetPlatformMacOS,
		"ios_": foundation.TargetPlatformIOS, "linux_": foundation.TargetPlatformLinux,
		"windows_": foundation.TargetPlatformWindows} {
		if strings.HasPrefix(scene, prefix) {
			return &p
		}
	}
	return nil
}

// TestEditingActionsMatchFlutter runs editing_actions_scenes.json as
// flutter/test/editing_actions_test.dart does (see it and
// TestTextFieldMatchesFlutter), with every platform's shortcuts and the
// clipboard mocked on flutter/platform, comparing [text, baseOffset,
// extentOffset, affinity] after each step.
func TestEditingActionsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("editing_actions_scenes.json")
	if err != nil {
		t.Fatal(err)
	}
	var scenes map[string][]editOp
	if err := json.Unmarshal(data, &scenes); err != nil {
		t.Fatal(err)
	}
	ref, err := os.ReadFile("flutter/build/editing_actions.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string][][4]any
	if err := json.Unmarshal(ref, &want); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(scenes))
	for n := range scenes {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			foundation.SetDebugDefaultTargetPlatformOverride(actionScenePlatform(name))
			defer foundation.SetDebugDefaultTargetPlatformOverride(nil)
			tester := flutter_test.New(t)
			tester.Owner.Platform = foundation.DefaultTargetPlatform()
			var clipboard *string
			messenger := tester.Binding().DefaultBinaryMessenger().(*flutter_test.TestDefaultBinaryMessenger)
			messenger.SetMockMethodCallHandler(services.SystemChannels.Platform, func(call services.MethodCall) async.Future[any] {
				return async.Async(func() any {
					switch call.Method {
					case "Clipboard.setData":
						s := call.Arguments.(map[string]any)["text"].(string)
						clipboard = &s
					case "Clipboard.getData":
						if clipboard != nil {
							return map[string]any{"text": *clipboard}
						}
					}
					return nil
				})
			})
			defer messenger.SetMockMethodCallHandler(services.SystemChannels.Platform, nil)
			controller := widgets.NewTextEditingController("")
			tester.PumpWidget(widgets.TextField{Controller: controller})
			tester.Tap(flutter_test.ByType[widgets.TextField]())
			tester.Pump()
			if len(want[name]) != len(scenes[name]) {
				t.Fatalf("reference has %d steps, scene %d", len(want[name]), len(scenes[name]))
			}
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
					tester.SendKeyEvent(actionKey(op.Key))
					for j := len(op.Mods) - 1; j >= 0; j-- {
						tester.SendKeyUpEvent(editMods[op.Mods[j]])
					}
				}
				tester.Pump()
				v := controller.Value()
				got := fmt.Sprintf("%q %d %d %s", v.Text, v.Selection.BaseOffset, v.Selection.ExtentOffset, v.Selection.Affinity.Name())
				w := want[name][i]
				exp := fmt.Sprintf("%q %d %d %s", w[0], int(w[1].(float64)), int(w[2].(float64)), w[3])
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
