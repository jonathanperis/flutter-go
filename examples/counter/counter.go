// Package counter is Flutter's classic counter app, written with flutter-go.
package counter

import (
	"fmt"

	r "github.com/jonathanperis/flutter-go/rendering"
	. "github.com/jonathanperis/flutter-go/widgets"
)

var (
	purple = r.RGB(0x6750a4)
	ink    = r.RGB(0x1d1b20)
	white  = r.RGB(0xffffff)
)

type CounterApp struct{ Title string }

func (CounterApp) CreateState() State { return &counterState{} }

type counterState struct {
	StateOf[CounterApp]
	count int
}

func (s *counterState) increment() { s.SetState(func() { s.count++ }) }

func (s *counterState) Build(*Element) Widget {
	return ColoredBox{Color: r.RGB(0xfef7ff), Child: Column{
		Children: []Widget{
			ColoredBox{Color: purple, Child: SizedBox{Height: F(64), Child: Align{
				Alignment: r.CenterLeft,
				Child: Padding{Padding: r.Symmetric(24, 0), Child: Text{
					Text: s.Widget().Title, Style: r.TextStyle{FontSize: 22, Color: white},
				}},
			}}},
			Expanded{Child: Center{Child: Column{
				MainAxisSize: r.MainMin,
				Children: []Widget{
					Text{Text: "You have pushed the button this many times:", Style: r.TextStyle{FontSize: 16, Color: ink}},
					SizedBox{Height: F(8)},
					Text{Text: fmt.Sprint(s.count), Style: r.TextStyle{FontSize: 48, Color: ink, Bold: true}},
				},
			}}},
			Align{Alignment: r.BottomRight, Child: Padding{Padding: r.All(24), Child: IncrementButton{OnTap: s.increment}}},
		},
	}}
}

// IncrementButton is a stateless "+" button.
type IncrementButton struct{ OnTap func() }

func (b IncrementButton) Build(*Element) Widget {
	return GestureDetector{OnTap: b.OnTap, Child: DecoratedBox{
		Color: purple, BorderRadius: 16,
		Child: SizedBox{Width: F(56), Height: F(56), Child: Center{
			Child: Text{Text: "+", Style: r.TextStyle{FontSize: 28, Color: white}},
		}},
	}}
}
