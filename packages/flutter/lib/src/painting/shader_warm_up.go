package painting

// Port of shader_warm_up.dart (Flutter 3.50.1): ShaderWarmUp, an abstract
// class user code extends: an interface plus the embeddable
// ShaderWarmUpBase (self pattern). The "Warm-up shader" timeline task is
// not recorded (no timeline recorder).

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
)

// ShaderWarmUp draws a picture offscreen, before the first frame, so that
// the shaders it needs are compiled.
type ShaderWarmUp interface {
	// Size is the size of the warm-up image (100×100 by default).
	Size() ui.Size
	// WarmUpOnCanvas draws the warm-up picture (Dart: @protected).
	WarmUpOnCanvas(canvas ui.Canvas) async.Future[async.Void]
	// Execute draws with WarmUpOnCanvas and rasterizes the picture.
	Execute() async.Future[async.Void]
}

// ShaderWarmUpBase is ShaderWarmUp's default methods; a type embedding it
// implements WarmUpOnCanvas and calls SetSelf.
type ShaderWarmUpBase struct {
	self ShaderWarmUp
}

// SetSelf points the base at the warm-up embedding it.
func (b *ShaderWarmUpBase) SetSelf(self ShaderWarmUp) { b.self = self }

// Size is 100×100.
func (b *ShaderWarmUpBase) Size() ui.Size { return ui.NewSize(100.0, 100.0) }

// Execute records WarmUpOnCanvas into a picture and rasterizes it at
// Size (rounded up), passing both to the debug capture callbacks.
func (b *ShaderWarmUpBase) Execute() async.Future[async.Void] {
	return ShaderWarmUpExecute(b.self)
}

// ShaderWarmUpExecute is ShaderWarmUp.execute for w.
func ShaderWarmUpExecute(w ShaderWarmUp) async.Future[async.Void] {
	if w == nil {
		panic("painting: ShaderWarmUpBase used before SetSelf")
	}
	return async.AsyncVoid(func() {
		recorder := ui.NewPictureRecorder()
		canvas := ui.NewCanvas(recorder)
		w.WarmUpOnCanvas(canvas).Await()
		picture := recorder.EndRecording()
		dartAssert(DebugCaptureShaderWarmUpPicture(picture))
		size := w.Size()
		image := picture.ToImage(int(math.Ceil(size.Width)), int(math.Ceil(size.Height))).Await()
		dartAssert(DebugCaptureShaderWarmUpImage(image))
		image.Dispose()
		picture.Dispose()
	})
}
