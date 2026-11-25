package widgets_test

import (
	"testing"

	r "github.com/jonathanperis/flutter-go/rendering"
	. "github.com/jonathanperis/flutter-go/widgets"
	"github.com/jonathanperis/flutter-go/widgettest"
)

// lifecycle records State lifecycle calls.
type lifecycle struct {
	Label string
	log   *[]string
}

func (w lifecycle) CreateState() State { return &lifecycleState{} }

type lifecycleState struct {
	StateOf[lifecycle]
	builds int
}

func (s *lifecycleState) log(m string) { *s.Widget().log = append(*s.Widget().log, m) }
func (s *lifecycleState) InitState()   { s.log("init " + s.Widget().Label) }
func (s *lifecycleState) DidUpdateWidget(old Widget) {
	s.log("update " + old.(lifecycle).Label + "->" + s.Widget().Label)
}
func (s *lifecycleState) Dispose() { s.log("dispose " + s.Widget().Label) }
func (s *lifecycleState) Build(*Element) Widget {
	s.builds++
	return Text{Text: s.Widget().Label}
}

func TestStateIsPreservedAcrossRebuilds(t *testing.T) {
	var log []string
	tester := widgettest.New(t)
	tester.PumpWidget(Center{Child: lifecycle{"a", &log}})
	tester.PumpWidget(Center{Child: lifecycle{"b", &log}})
	tester.PumpWidget(Center{Child: Text{Text: "gone"}})

	want := []string{"init a", "update a->b", "dispose b"}
	if len(log) != len(want) {
		t.Fatalf("log = %v, want %v", log, want)
	}
	for i := range want {
		if log[i] != want[i] {
			t.Fatalf("log = %v, want %v", log, want)
		}
	}
}

func TestKeysMoveStateWithReorderedChildren(t *testing.T) {
	var log []string
	item := func(k string) Widget { return KeyedSubtree{Key: k, Child: lifecycle{k, &log}} }
	tester := widgettest.New(t)
	tester.PumpWidget(Column{Children: []Widget{item("x"), item("y")}})
	stateX := tester.FindAll(widgettest.ByType[lifecycle]())[0]

	tester.PumpWidget(Column{Children: []Widget{item("y"), item("x")}})
	after := tester.FindAll(widgettest.ByType[lifecycle]())
	if after[1] != stateX {
		t.Fatal("keyed element for x should have moved, not been recreated")
	}
	if len(log) != 4 { // init x, init y, update x->x, update y->y
		t.Fatalf("unexpected lifecycle: %v", log)
	}
	// Render children follow the new order: y above x.
	if tester.Rect(widgettest.Text("y")).Y >= tester.Rect(widgettest.Text("x")).Y {
		t.Fatal("render order did not follow reordered widgets")
	}
}

func TestExpandedParentDataReachesRenderObject(t *testing.T) {
	tester := widgettest.New(t)
	tester.PumpWidget(Row{Children: []Widget{
		SizedBox{Width: F(200)},
		Expanded{Flex: 1, Child: ColoredBox{Color: r.RGB(0xff0000)}},
		Expanded{Flex: 2, Child: ColoredBox{Color: r.RGB(0x00ff00)}},
	}})
	boxes := tester.FindAll(widgettest.ByType[ColoredBox]())
	w := func(i int) float64 { return boxes[i].RenderObject().Base().Size.W }
	if w(0) != 200 || w(1) != 400 {
		t.Fatalf("expanded widths = %v, %v; want 200, 400", w(0), w(1))
	}

	// Changing flex in place must relayout.
	tester.PumpWidget(Row{Children: []Widget{
		SizedBox{Width: F(200)},
		Expanded{Flex: 2, Child: ColoredBox{Color: r.RGB(0xff0000)}},
		Expanded{Flex: 1, Child: ColoredBox{Color: r.RGB(0x00ff00)}},
	}})
	if w(0) != 400 || w(1) != 200 {
		t.Fatalf("after flex swap: %v, %v; want 400, 200", w(0), w(1))
	}
}

type toggler struct{}

func (toggler) CreateState() State { return &togglerState{} }

type togglerState struct {
	StateOf[toggler]
	on bool
}

func (s *togglerState) Build(*Element) Widget {
	if s.on {
		return SizedBox{Width: F(10), Height: F(10)}
	}
	return Text{Text: "off"}
}

func TestStatefulChildSwappingRenderObjectUpdatesParent(t *testing.T) {
	tester := widgettest.New(t)
	tester.PumpWidget(Row{Children: []Widget{toggler{}}})
	el := tester.FindAll(widgettest.ByType[toggler]())[0]
	ts := findState(el)
	ts.SetState(func() { ts.on = true })
	tester.Pump()

	row := tester.FindAll(widgettest.ByType[Row]())[0].RenderObject()
	kids := row.Children()
	if len(kids) != 1 {
		t.Fatalf("row has %d render children", len(kids))
	}
	if _, ok := kids[0].(*r.RenderConstrainedBox); !ok {
		t.Fatalf("row child is %T, want the new SizedBox render object", kids[0])
	}
}

// findState digs the State out via a test-only accessor.
func findState(e *Element) *togglerState { return StateOfElement(e).(*togglerState) }
