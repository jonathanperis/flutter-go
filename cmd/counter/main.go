// counter runs Flutter's classic counter app with flutter-go.
//
// For unattended checks: -taps=N injects N taps on "+" (one per frame, through
// the same pointer router the mouse uses) and -frames=N exits after N frames.
// The window number is logged for `screencapture -l`.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/jonathanperis/flutter-go/app"
	"github.com/jonathanperis/flutter-go/examples/counter"
	"github.com/jonathanperis/flutter-go/platform/metal"
)

func main() {
	frames := flag.Int("frames", 0, "exit after N frames (0 = until closed)")
	taps := flag.Int("taps", 0, "inject N taps on the + button")
	flag.Parse()
	err := app.Run(counter.CounterApp{Title: "flutter-go counter"}, app.Options{
		Title: "flutter-go: counter", Width: 800, Height: 600, MaxFrames: *frames,
		// The button's center in logical pixels: 800-24-28, 600-24-28.
		InjectTaps: *taps, InjectTapAt: [2]float64{748, 548},
		Window: func(w *glfw.Window) {
			log.Printf("window %d", metal.WindowNumber(w.GetCocoaWindow()))
		},
	})
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
