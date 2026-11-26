package services_test

// Ports of packages/flutter/test/services/font_loader_test.dart (2/2),
// flutter_version_test.dart (6/6), app_version_test.dart (2/2) and
// predictive_back_event_test.dart (8/8) (Flutter 3.50.1). The version
// tests compare with the undefined defines (Go builds without
// `-ldflags -X` have none, as `flutter test` without --dart-define).

import (
	"bytes"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type testFontLoader struct {
	*services.FontLoader
	fontAssets [][]byte
}

func newTestFontLoader(family string) *testFontLoader {
	l := &testFontLoader{FontLoader: services.NewFontLoader(family)}
	l.SetSelf(l)
	return l
}

func (l *testFontLoader) LoadFont(list []byte, family string) async.Future[async.Void] {
	return async.AsyncVoid(func() { l.fontAssets = append(l.fontAssets, list) })
}

func TestFontLoader(t *testing.T) {
	runAsync(t, func() {
		tfl := newTestFontLoader("TestFamily")
		expectedAssets := [][]byte{{100}, {10, 20, 30}, {200}}
		for _, asset := range expectedAssets {
			tfl.AddFont(async.FutureValue(asset))
		}
		tfl.Load().Await()
		expect(len(tfl.fontAssets) == 3, "assets %v", tfl.fontAssets)
		for i := range expectedAssets {
			expect(bytes.Equal(tfl.fontAssets[i], expectedAssets[i]), "assets %v", tfl.fontAssets)
		}
		// A loaded loader can't load or add again.
		expect(awaitError(tfl.Load()) != nil, "second load")
		func() {
			defer func() {
				_, ok := recover().(dart.StateError)
				expect(ok, "addFont after load")
			}()
			tfl.AddFont(async.FutureValue([]byte{1}))
		}()
	})
}

func TestFontLoaderLoadsFontsInOrder(t *testing.T) {
	runAsync(t, func() {
		tfl := newTestFontLoader("TestFamily")
		expectedAssets := [][]byte{{1}, {2}}
		completer := async.NewCompleter[async.Void]()
		// Deliberately delay the load of the first font.
		tfl.AddFont(async.Async(func() []byte {
			completer.Future().Await()
			return expectedAssets[0]
		}))
		tfl.AddFont(async.FutureValue(expectedAssets[1]))
		fontLoadComplete := tfl.Load()
		completer.Complete(async.Void{})
		fontLoadComplete.Await()
		expect(len(tfl.fontAssets) == 2 && tfl.fontAssets[0][0] == 1 && tfl.fontAssets[1][0] == 2, "assets %v", tfl.fontAssets)
	})
}

func TestFlutterVersionDefines(t *testing.T) {
	v := services.FlutterVersion
	for name, value := range map[string]*string{"version": v.Version, "channel": v.Channel, "gitUrl": v.GitURL,
		"frameworkRevision": v.FrameworkRevision, "engineRevision": v.EngineRevision, "dartVersion": v.DartVersion} {
		if value != nil {
			t.Errorf("FlutterVersion.%s = %q without a define", name, *value)
		}
	}
	if services.AppBuildName != nil || services.AppBuildNumber != nil || services.AppFlavor != nil {
		t.Error("app build name/number/flavor set without a define")
	}
}

func backEvent(m map[any]any) services.PredictiveBackEvent {
	return services.NewPredictiveBackEventFromMap(m)
}

func TestPredictiveBackEventFromMap(t *testing.T) {
	left := backEvent(map[any]any{"touchOffset": []any{0.0, 100.0}, "progress": 0.0, "swipeEdge": 0})
	if left.SwipeEdge != services.SwipeEdgeLeft || left.IsButtonEvent() {
		t.Error(left)
	}
	right := backEvent(map[any]any{"touchOffset": []float64{0.0, 100.0}, "progress": 0.0, "swipeEdge": 1})
	if right.SwipeEdge != services.SwipeEdgeRight || right.IsButtonEvent() {
		t.Error(right)
	}
	if e := backEvent(map[any]any{"touchOffset": []any{0.0, 0.0}, "progress": 0.0, "swipeEdge": 1}); !e.IsButtonEvent() {
		t.Error("zero position", e)
	}
	if e := backEvent(map[any]any{"touchOffset": nil, "progress": 0.0, "swipeEdge": 1}); !e.IsButtonEvent() {
		t.Error("null position", e)
	}
	if e := backEvent(map[any]any{"touchOffset": []any{0.0, 0.0}, "progress": 0.5, "swipeEdge": 1}); e.IsButtonEvent() {
		t.Error("zero position with progress", e)
	}
	func() {
		defer func() {
			if _, ok := recover().(*dart.AssertionError); !ok {
				t.Error("progress 2.0 doesn't assert")
			}
		}()
		backEvent(map[any]any{"touchOffset": []any{0.0, 100.0}, "progress": 2.0, "swipeEdge": 1})
	}()
	func() {
		defer func() {
			if _, ok := recover().(dart.IndexError); !ok {
				t.Error("swipeEdge 2 doesn't throw a RangeError")
			}
		}()
		backEvent(map[any]any{"touchOffset": []any{0.0, 100.0}, "progress": 0.0, "swipeEdge": 2})
	}()
}

func TestPredictiveBackEventEquality(t *testing.T) {
	a := backEvent(map[any]any{"touchOffset": []any{0.0, 100.0}, "progress": 0.0, "swipeEdge": 0})
	b := backEvent(map[any]any{"touchOffset": []any{0.0, 100.0}, "progress": 0.0, "swipeEdge": 0})
	if !a.Equal(b) || a.String() != b.String() {
		t.Error("equal events", a, b)
	}
	c := backEvent(map[any]any{"touchOffset": []any{1.0, 100.0}, "progress": 0.0, "swipeEdge": 0})
	if a.Equal(c) || a.String() == c.String() {
		t.Error("different events", a, c)
	}
	if got := a.String(); got != "PredictiveBackEvent{touchOffset: Offset(0.0, 100.0), progress: 0.0, swipeEdge: SwipeEdge.left}" {
		t.Error(got)
	}
	o := ui.NewOffset(0, 100)
	if !a.Equal(services.PredictiveBackEvent{TouchOffset: &o, SwipeEdge: services.SwipeEdgeLeft}) {
		t.Error("literal")
	}
}
