package counter

import (
	"testing"

	"github.com/jonathanperis/flutter-go/widgettest"
)

// Mirrors the smoke test `flutter create` generates for the counter app.
func TestCounterIncrementsSmokeTest(t *testing.T) {
	tester := widgettest.New(t)
	tester.PumpWidget(CounterApp{Title: "flutter-go"})

	if tester.Count(widgettest.Text("0")) != 1 || tester.Count(widgettest.Text("1")) != 0 {
		t.Fatal("expected 0 and not 1")
	}

	tester.Tap(widgettest.Text("+"))
	tester.Pump()

	if tester.Count(widgettest.Text("0")) != 0 || tester.Count(widgettest.Text("1")) != 1 {
		t.Fatal("expected 1 and not 0 after tapping +")
	}
}

func TestOnlyTheStatefulSubtreeRebuilds(t *testing.T) {
	tester := widgettest.New(t)
	tester.PumpWidget(CounterApp{Title: "flutter-go"})
	before := tester.Owner.Builds

	tester.Tap(widgettest.Text("+"))
	tester.Pump()

	// One rebuild of counterState plus the IncrementButton below it.
	if got := tester.Owner.Builds - before; got != 2 {
		t.Fatalf("builds per tap = %d, want 2", got)
	}
}

func TestButtonGeometry(t *testing.T) {
	tester := widgettest.New(t)
	tester.PumpWidget(CounterApp{Title: "flutter-go"})
	got := tester.Rect(widgettest.ByType[IncrementButton]())
	// 56×56 with 24px padding from the bottom-right of 800×600.
	if got.X != 720 || got.Y != 520 || got.W != 56 || got.H != 56 {
		t.Fatalf("button rect = %+v", got)
	}
}
