package widgets_test

import (
	"fmt"
	"testing"

	r "github.com/jonathanperis/flutter-go/rendering"
	. "github.com/jonathanperis/flutter-go/widgets"
	"github.com/jonathanperis/flutter-go/widgettest"
)

func itemText(i int) string { return fmt.Sprintf("item %d", i) }

func textList(count int, c *ScrollController, built *int) ListViewBuilder {
	return ListViewBuilder{
		ItemCount: count, ItemExtent: 50, Controller: c,
		ItemBuilder: func(_ *Element, i int) Widget {
			*built++
			return ColoredBox{Color: r.RGB(0xeeeeee), Child: Text{Text: itemText(i)}}
		},
	}
}

func builtItems(t *widgettest.Tester) []string {
	var out []string
	for _, e := range t.FindAll(widgettest.ByType[Text]()) {
		out = append(out, e.Widget().(Text).Text)
	}
	return out
}

func TestListBuildsOnlyVisiblePlusCache(t *testing.T) {
	built := 0
	c := NewScrollController(0)
	tester := widgettest.New(t)
	tester.PumpWidget(textList(1_000_000, c, &built))

	// 600px viewport + 250px cache below, 50px items → items 0..16.
	items := builtItems(tester)
	if len(items) != 17 || items[0] != itemText(0) || items[16] != itemText(16) {
		t.Fatalf("built %d items: %v", len(items), items)
	}
	if built != 17 {
		t.Fatalf("ItemBuilder called %d times, want 17", built)
	}

	c.JumpTo(500_000 * 50)
	tester.Pump()
	items = builtItems(tester)
	// 250px cache above and below: offset 25,000,000 → items 499995..500016.
	if len(items) != 22 || items[0] != itemText(499_995) || items[21] != itemText(500_016) {
		t.Fatalf("after jump built %d items: %v..%v", len(items), items[0], items[len(items)-1])
	}
}

func TestListClampsToEnd(t *testing.T) {
	built := 0
	c := NewScrollController(0)
	tester := widgettest.New(t)
	tester.PumpWidget(textList(100, c, &built))
	c.JumpTo(1e9)
	// Like Flutter's jumpTo, an out-of-range jump lands there and the physics
	// then spring back into range.
	if c.Offset() != 1e9 {
		t.Fatalf("jump should land out of range first, offset %v", c.Offset())
	}
	tester.PumpAndSettle()
	if c.Offset() != 100*50-600 || c.MaxScrollExtent() != 4400 {
		t.Fatalf("offset %v max %v", c.Offset(), c.MaxScrollExtent())
	}
	if got := tester.Rect(widgettest.Text(itemText(99))); got.Y != 550 {
		t.Fatalf("last item at y=%v, want 550", got.Y)
	}
}

type itemState struct {
	StateOf[statefulItem]
}

type statefulItem struct {
	Index int
	Log   *[]string
}

func (w statefulItem) CreateState() State { return &itemState{} }
func (s *itemState) InitState() {
	*s.Widget().Log = append(*s.Widget().Log, fmt.Sprint("init ", s.Widget().Index))
}
func (s *itemState) Dispose() {
	*s.Widget().Log = append(*s.Widget().Log, fmt.Sprint("dispose ", s.Widget().Index))
}
func (s *itemState) Build(*Element) Widget {
	return Text{Text: itemText(s.Widget().Index)}
}

func TestItemStateIsDisposedOffscreenAndRecreated(t *testing.T) {
	var log []string
	c := NewScrollController(0)
	tester := widgettest.New(t)
	tester.PumpWidget(ListViewBuilder{
		ItemCount: 1000, ItemExtent: 50, Controller: c,
		ItemBuilder: func(_ *Element, i int) Widget { return statefulItem{i, &log} },
	})
	log = nil
	c.JumpTo(1000) // item 0 now 1000px above: beyond the 250px cache
	tester.Pump()
	if !contains(log, "dispose 0") || !contains(log, "init 30") {
		t.Fatalf("expected item 0 disposed and item 30 created, log: %v", log)
	}
	log = nil
	c.JumpTo(0)
	tester.Pump()
	if !contains(log, "init 0") {
		t.Fatalf("item 0 should be created again, log: %v", log)
	}
}

func TestDragScrollsAfterSlopAndCancelsTap(t *testing.T) {
	taps := 0
	c := NewScrollController(0)
	tester := widgettest.New(t)
	tester.PumpWidget(ListViewBuilder{
		ItemCount: 1000, ItemExtent: 50, Controller: c,
		ItemBuilder: func(_ *Element, i int) Widget {
			return GestureDetector{OnTap: func() { taps++ }, Child: Text{Text: itemText(i)}}
		},
	})
	// Items have tap handlers, so the drag competes with a tap and is only
	// accepted past the 18px slop: in 10px steps that's at 20px, and the drag
	// begins there (DragStartBehavior.start), so 200px scrolls 180px.
	tester.Drag(widgettest.Text(itemText(5)), -200)
	tester.Pump()
	if c.Offset() != 180 {
		t.Fatalf("offset after 200px drag = %v, want 180", c.Offset())
	}
	if taps != 0 {
		t.Fatal("a drag must not count as a tap")
	}
	// A small move stays a tap.
	tester.Drag(widgettest.Text(itemText(10)), -5)
	tester.Pump()
	if taps != 1 || c.Offset() != 180 {
		t.Fatalf("taps=%d offset=%v after a 5px move", taps, c.Offset())
	}
}

func TestTapAfterScrollHitsTheRightItem(t *testing.T) {
	var tapped []int
	c := NewScrollController(0)
	tester := widgettest.New(t)
	tester.PumpWidget(ListViewBuilder{
		ItemCount: 1000, ItemExtent: 50, Controller: c,
		ItemBuilder: func(_ *Element, i int) Widget {
			return GestureDetector{OnTap: func() { tapped = append(tapped, i) }, Child: Text{Text: itemText(i)}}
		},
	})
	tester.Scroll(widgettest.ByType[ListViewBuilder](), 1234)
	tester.Pump()
	tester.Tap(widgettest.Text(itemText(30)))
	if len(tapped) != 1 || tapped[0] != 30 {
		t.Fatalf("tapped %v, want [30]", tapped)
	}
}

type prefixed struct{}

func (prefixed) CreateState() State { return &prefixedState{prefix: "a"} }

type prefixedState struct {
	StateOf[prefixed]
	prefix string
}

func (s *prefixedState) Build(*Element) Widget {
	return ListViewBuilder{ItemCount: 100, ItemExtent: 50, ItemBuilder: func(_ *Element, i int) Widget {
		return Text{Text: fmt.Sprint(s.prefix, i)}
	}}
}

func TestParentRebuildUpdatesLiveItems(t *testing.T) {
	tester := widgettest.New(t)
	tester.PumpWidget(prefixed{})
	st := StateOfElement(tester.FindAll(widgettest.ByType[prefixed]())[0]).(*prefixedState)
	st.SetState(func() { st.prefix = "b" })
	tester.Pump()
	if tester.Count(widgettest.Text("a3")) != 0 || tester.Count(widgettest.Text("b3")) != 1 {
		t.Fatal("live items were not rebuilt with the new builder")
	}
}

func TestListInsideColumnNeedsExpanded(t *testing.T) {
	tester := widgettest.New(t)
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for an unbounded-height list, like Flutter's assertion")
		}
	}()
	tester.PumpWidget(Column{Children: []Widget{ListViewBuilder{ItemCount: 1, ItemExtent: 10,
		ItemBuilder: func(*Element, int) Widget { return SizedBox{} }}}})
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
