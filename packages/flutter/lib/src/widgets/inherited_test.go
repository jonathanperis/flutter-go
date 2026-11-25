package widgets_test

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"testing"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

type sizeReader struct{ seen *[]r.Size }

func (w sizeReader) Build(ctx *Element) Widget {
	*w.seen = append(*w.seen, MediaQueryOf(ctx).Size)
	return SizedBox{}
}

type plain struct{ builds *int }

func (w plain) Build(*Element) Widget { *w.builds++; return SizedBox{} }

func TestResizeRebuildsOnlyMediaQueryDependents(t *testing.T) {
	var seen []r.Size
	builds := 0
	tester := flutter_test.New(t)
	tester.PumpWidget(Column{Children: []Widget{sizeReader{&seen}, plain{&builds}}})
	tester.Size = ui.Size{Width: 1024, Height: 768}
	tester.Pump()
	if len(seen) != 2 || seen[0] != (ui.Size{Width: 800, Height: 600}) || seen[1] != tester.Size {
		t.Fatalf("sizes seen: %v", seen)
	}
	if builds != 1 {
		t.Fatalf("a widget that doesn't depend on MediaQuery rebuilt (%d builds)", builds)
	}
}

func TestDefaultTextStyleIsInheritedAndOverridable(t *testing.T) {
	tester := flutter_test.New(t)
	tester.PumpWidget(DefaultTextStyle{Style: r.TextStyle{FontSize: 20}, Child: Column{Children: []Widget{
		Text{Text: "inherits"},
		Text{Text: "own", Style: r.TextStyle{FontSize: 10}},
	}}})
	if h := tester.Rect(flutter_test.Text("inherits")).Height(); h != 20 {
		t.Fatalf("inherited font size: height %v", h)
	}
	if h := tester.Rect(flutter_test.Text("own")).Height(); h != 10 {
		t.Fatalf("explicit font size: height %v", h)
	}
	tester.PumpWidget(DefaultTextStyle{Style: r.TextStyle{FontSize: 30}, Child: Column{Children: []Widget{
		Text{Text: "inherits"},
		Text{Text: "own", Style: r.TextStyle{FontSize: 10}},
	}}})
	if h := tester.Rect(flutter_test.Text("inherits")).Height(); h != 30 {
		t.Fatalf("after change: height %v (dependent did not rebuild)", h)
	}
}
