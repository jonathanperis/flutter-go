package painting

// Port of image_decoder.dart (Flutter 3.50.1).

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
)

// DecodeImageFromList creates an image from a list of bytes (any format
// the engine decodes), through PaintingBinding's
// InstantiateImageCodecWithSize; the future fails if the data can't be
// decoded.
func DecodeImageFromList(bytes []byte) async.Future[*ui.Image] {
	return async.Async(func() *ui.Image {
		buffer := ui.ImmutableBufferFromUint8List(bytes).Await()
		codec := PaintingBindingInstance().InstantiateImageCodecWithSize(buffer).Await()
		var frameInfo *ui.FrameInfo
		func() {
			defer codec.Dispose()
			frameInfo = codec.GetNextFrame().Await()
		}()
		return frameInfo.Image
	})
}
