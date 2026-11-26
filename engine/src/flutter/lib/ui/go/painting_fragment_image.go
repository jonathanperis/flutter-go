package ui

// What FragmentShader reads from an Image (Dart's image.width/height, and
// whether the native image is gone).

func imageSize(img *Image) (width, height int) { return img.Width, img.Height }

// imageEngineDisposed is the engine's `!image->image()`: the image data's
// last handle was disposed.
func imageEngineDisposed(img *Image) bool { return img.image == nil || img.image.released }
