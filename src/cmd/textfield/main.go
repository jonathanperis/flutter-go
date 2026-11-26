// textfield is a small to-do app: type in the field, press Enter to add.
//
// -type=TEXT types TEXT after the first frame (unattended runs).
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/src/app"
	"github.com/jonathanperis/flutter-go/src/platform/metal"
)

type todos struct{ initial []string }

func (t todos) CreateState() State {
	return &todoState{items: t.initial, input: NewTextEditingController("")}
}

type todoState struct {
	StateOf[todos]
	items []string
	input *TextEditingController
}

func (s *todoState) add(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	s.SetState(func() { s.items = append(s.items, text) })
	s.input.SetText("")
}

func (s *todoState) Build(*Element) Widget {
	rows := []Widget{
		ColoredBox{Color: r.RGB(0x6750a4), Child: Padding{Padding: painting.NewEdgeInsetsSymmetric(painting.EdgeInsetsSymmetricOptions{Horizontal: 24, Vertical: 18}), Child: Text{
			Text: fmt.Sprintf("To do · %d items", len(s.items)), Style: r.TextStyle{FontSize: 20, Color: r.RGB(0xffffff)}}}},
		Padding{Padding: painting.NewEdgeInsetsAll(16), Child: DecoratedBox{Color: r.RGB(0xece6f0), BorderRadius: 12, Child: Padding{
			Padding: painting.NewEdgeInsetsSymmetric(painting.EdgeInsetsSymmetricOptions{Horizontal: 14, Vertical: 12}),
			Child:   TextField{Controller: s.input, Autofocus: true, Style: r.TextStyle{FontSize: 18}, OnSubmitted: s.add},
		}}},
	}
	for i, it := range s.items {
		rows = append(rows, Padding{Padding: painting.NewEdgeInsetsSymmetric(painting.EdgeInsetsSymmetricOptions{Horizontal: 24, Vertical: 8}), Child: Text{
			Text: fmt.Sprintf("%d. %s", i+1, it), Style: r.TextStyle{FontSize: 16}}})
	}
	return ColoredBox{Color: r.RGB(0xfef7ff), Child: Column{CrossAxisAlignment: r.CrossStretch, Children: rows}}
}

func main() {
	typ := flag.String("type", "", "text to type after the first frame")
	flag.Parse()
	err := app.Run(todos{initial: []string{"Port the render tree", "Match Flutter's layout", "Lazy lists and flings"}}, app.Options{
		Title: "flutter-go: text field", Width: 640, Height: 420, InjectText: *typ,
		Window: func(w *glfw.Window) { log.Printf("window %d", metal.WindowNumber(w.GetCocoaWindow())) },
	})
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
