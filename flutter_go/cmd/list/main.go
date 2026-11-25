// list shows a lazily built list of 1,000,000 rows. Scroll with the mouse
// wheel or trackpad; click a row to select it.
//
// -scroll=N injects a wheel scroll of N pixels after the first frame.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/jonathanperis/flutter-go/flutter_go/app"
	"github.com/jonathanperis/flutter-go/flutter_go/platform/metal"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
)

type listApp struct{}

func (listApp) CreateState() State { return &listState{selected: -1} }

type listState struct {
	StateOf[listApp]
	selected int
}

func (s *listState) Build(*Element) Widget {
	header := fmt.Sprint("1,000,000 rows · built lazily")
	if s.selected >= 0 {
		header = fmt.Sprintf("Selected row %d", s.selected)
	}
	return Column{CrossAxisAlignment: r.CrossStretch, Children: []Widget{
		ColoredBox{Color: r.RGB(0x6750a4), Child: Padding{Padding: r.Symmetric(24, 18), Child: Text{
			Text: header, Style: r.TextStyle{FontSize: 20, Color: r.RGB(0xffffff)},
		}}},
		Expanded{Child: ListViewBuilder{ItemCount: 1_000_000, ItemExtent: 56, ItemBuilder: s.row}},
	}}
}

func (s *listState) row(_ *Element, i int) Widget {
	bg := r.RGB(0xffffff)
	if i%2 == 1 {
		bg = r.RGB(0xf3edf7)
	}
	if i == s.selected {
		bg = r.RGB(0xd0bcff)
	}
	return GestureDetector{
		OnTap: func() { s.SetState(func() { s.selected = i }) },
		Child: ColoredBox{Color: bg, Child: Align{Alignment: r.CenterLeft, Child: Padding{
			Padding: r.Symmetric(24, 0),
			Child:   Text{Text: fmt.Sprintf("Row %d", i), Style: r.TextStyle{FontSize: 16, Color: r.RGB(0x1d1b20)}},
		}}},
	}
}

func main() {
	scroll := flag.Float64("scroll", 0, "inject a wheel scroll of N pixels")
	flag.Parse()
	err := app.Run(listApp{}, app.Options{
		Title: "flutter-go: list", Width: 800, Height: 600, InjectScroll: *scroll,
		Window: func(w *glfw.Window) { log.Printf("window %d", metal.WindowNumber(w.GetCocoaWindow())) },
	})
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
