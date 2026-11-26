package painting

// Port of paint_utilities.dart (Flutter 3.50.1): paintZigZag.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// PaintZigZag draws a line in a zig-zag pattern from start to end: zigs
// peaks on each side, each width away from the line, stroked with paint.
func PaintZigZag(canvas ui.Canvas, paint *ui.Paint, start, end ui.Offset, zigs int, width float64) {
	if !(zigs > 0) {
		panic(dart.NewAssertionError("zigs > 0"))
	}
	canvas.Save()
	canvas.Translate(start.Dx, start.Dy)
	end = end.Sub(start)
	canvas.Rotate(dart.Atan2(end.Dy, end.Dx))
	length := end.Distance()
	spacing := length / (float64(zigs) * 2.0)
	path := ui.NewPath()
	path.MoveTo(0.0, 0.0)
	for index := 0; index < zigs; index++ {
		x := float64((float64(float64(index)*2.0) + 1.0) * spacing)
		y := float64(width * (float64(dart.Mod(float64(index), 2.0)*2.0) - 1.0))
		path.LineTo(x, y)
	}
	path.LineTo(length, 0.0)
	canvas.DrawPath(path, paint)
	canvas.Restore()
}
