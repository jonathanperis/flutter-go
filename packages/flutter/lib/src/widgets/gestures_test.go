package widgets_test

import (
	"testing"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// The router moves the pointer to where it is lifted (the engine
// synthesizes that move), so a drag released elsewhere scrolls there.
func TestPointerRouterUpElsewhereMovesFirst(t *testing.T) {
	tester := flutter_test.New(t)
	c := NewScrollController(0)
	tester.PumpWidget(ListViewBuilder{ItemCount: 1000, ItemExtent: 50, Controller: c,
		ItemBuilder: func(*Element, int) Widget { return SizedBox{} }})
	router := NewPointerRouter(tester.Owner)
	router.AfterEvent = func() { tester.Loop().FlushMicrotasks() }
	router.PointerDown(r.Offset{Dx: 400, Dy: 300})
	router.PointerUp(r.Offset{Dx: 400, Dy: 200})
	if c.Offset() != 100 {
		t.Fatalf("offset %v, want 100", c.Offset())
	}
	tester.PumpAndSettle()
}
