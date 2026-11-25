package widgets_test

import (
	"testing"
	"time"

	. "github.com/jonathanperis/flutter-go/widgets"
	"github.com/jonathanperis/flutter-go/widgettest"
)

func TestFastDragReleaseFlingsThenSettles(t *testing.T) {
	c := NewScrollController(0)
	tester := widgettest.New(t)
	tester.PumpWidget(textList(1000, c, new(int)))
	tester.FlingGesture(widgettest.Text(itemText(5)), -300, 100*time.Millisecond) // ~3000 px/s
	afterRelease := c.Offset()
	tester.PumpAndSettle()
	if c.Offset() <= afterRelease+100 {
		t.Fatalf("expected the list to keep moving after release: %v → %v", afterRelease, c.Offset())
	}
	if c.IsAnimating() {
		t.Fatal("fling should have settled")
	}
}

func TestSlowDragReleaseDoesNotFling(t *testing.T) {
	c := NewScrollController(0)
	tester := widgettest.New(t)
	tester.PumpWidget(textList(1000, c, new(int)))
	// 40px over ~1s is 40 px/s, under kMinFlingVelocity (50).
	tester.FlingGesture(widgettest.Text(itemText(5)), -40, time.Second)
	afterRelease := c.Offset()
	tester.PumpAndSettle()
	if c.Offset() != afterRelease {
		t.Fatalf("slow drag flung: %v → %v", afterRelease, c.Offset())
	}
}

func TestTouchingAFlingingListStopsItWithoutTapping(t *testing.T) {
	taps := 0
	c := NewScrollController(0)
	tester := widgettest.New(t)
	tester.PumpWidget(ListViewBuilder{ItemCount: 1000, ItemExtent: 50, Controller: c,
		ItemBuilder: func(_ *Element, i int) Widget {
			return GestureDetector{OnTap: func() { taps++ }, Child: Text{Text: itemText(i)}}
		}})
	c.Fling(3000)
	tester.Pump()
	tester.Pump(100 * time.Millisecond)
	tester.TapAt(400, 300)
	stoppedAt := c.Offset()
	tester.PumpAndSettle()
	if taps != 0 {
		t.Fatal("the touch that stops a fling must not tap an item (Flutter ignores pointers during scroll activities)")
	}
	if c.Offset() != stoppedAt || c.IsAnimating() {
		t.Fatalf("fling should stop where touched: %v → %v", stoppedAt, c.Offset())
	}
	tester.TapAt(400, 300) // now idle: a normal tap
	if taps != 1 {
		t.Fatalf("taps = %d after tapping an idle list", taps)
	}
}
