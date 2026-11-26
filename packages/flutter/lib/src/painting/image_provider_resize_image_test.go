package painting_test

// Port of packages/flutter/test/painting/image_provider_resize_image_test.dart
// (Flutter 3.50.1): all 50 tests (the 32 policy=fit cases are a table).
// The hashCode halves of the equality tests have no Go counterpart (== is
// Equal / KeysEqual).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
)

func newResizeTester(t *testing.T) *flutter_test.Tester {
	tester := flutter_test.New(t)
	t.Cleanup(func() {
		ImageCacheInstance().Clear()
		ImageCacheInstance().ClearLiveImages()
	})
	return tester
}

func ip(v int) *int { return &v }

func resolveAndGetSize(tester *flutter_test.Tester, provider ImageProvider, configuration ...ImageConfiguration) ui.Size {
	config := ImageConfigurationEmpty
	if len(configuration) > 0 {
		config = configuration[0]
	}
	stream := provider.Resolve(config)
	completer := async.NewCompleter[ui.Size]()
	stream.AddListener(NewImageStreamListener(func(image *ImageInfo, synchronousCall bool) {
		completer.Complete(ui.NewSize(float64(image.Image.Width), float64(image.Image.Height)))
	}))
	return await(tester, completer.Future())
}

func expectImageSize(t *testing.T, tester *flutter_test.Tester, provider ImageProvider, w, h float64) {
	t.Helper()
	if s := resolveAndGetSize(tester, provider); s != ui.NewSize(w, h) {
		t.Errorf("size %v, want %vx%v", s, w, h)
	}
}

func TestResizeImageEquality(t *testing.T) {
	newResizeTester(t)
	bytes := append([]byte{}, kTransparentImage...)
	r1 := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(100), Height: ip(200), Policy: ResizeImagePolicyFit, AllowUpscaling: true})
	r2 := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(100), Height: ip(200), Policy: ResizeImagePolicyFit, AllowUpscaling: true})
	if !r1.Equal(r2) {
		t.Error("identical instances not equal")
	}
	r3 := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(100), Height: ip(200)})
	r4 := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(150), Height: ip(200)})
	if r3.Equal(r4) {
		t.Error("different instances equal")
	}
	if !r3.Equal(r3) {
		t.Error("not reflexive")
	}
}

func TestResizeImageUpscalesToCorrectDimensions(t *testing.T) {
	tester := newResizeTester(t)
	bytes := append([]byte{}, kTransparentImage...)
	expectImageSize(t, tester, NewMemoryImage(bytes), 1, 1)
	resizeDims := ui.NewSize(14, 7)
	resized := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(14), Height: ip(7), AllowUpscaling: true})
	if s := resolveAndGetSize(tester, resized, ImageConfiguration{Size: &resizeDims}); s != resizeDims {
		t.Errorf("size %v", s)
	}
}

func TestResizeImageDownscalesToCorrectDimensions(t *testing.T) {
	tester := newResizeTester(t)
	bytes := append([]byte{}, kBlueSquarePng...)
	expectImageSize(t, tester, NewMemoryImage(bytes), 50, 50)
	resizeDims := ui.NewSize(25, 25)
	resized := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(25), Height: ip(25), AllowUpscaling: true})
	if s := resolveAndGetSize(tester, resized, ImageConfiguration{Size: &resizeDims}); s != resizeDims {
		t.Errorf("size %v", s)
	}
}

func dpr(v float64) ImageConfiguration { return ImageConfiguration{DevicePixelRatio: &v} }

func TestResizeImageUseLogicalSize(t *testing.T) {
	tester := newResizeTester(t)
	bytes := append([]byte{}, kBlueSquarePng...)
	expectImageSize(t, tester, NewMemoryImage(bytes), 50, 50)
	// 25x25 logical pixels on a 2x device: 50x50 physical pixels.
	resized := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(25), Height: ip(25), AllowUpscaling: true, UseLogicalSize: true})
	if s := resolveAndGetSize(tester, resized, dpr(2.0)); s != ui.NewSize(50, 50) {
		t.Errorf("size %v", s)
	}
}

func TestResizeImageUseLogicalSizeDoesNotUpscaleByDefault(t *testing.T) {
	tester := newResizeTester(t)
	bytes := append([]byte{}, kBlueSquarePng...)
	resized := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(25), Height: ip(25), UseLogicalSize: true})
	if s := resolveAndGetSize(tester, resized, dpr(3.0)); s != ui.NewSize(50, 50) {
		t.Errorf("size %v", s)
	}
}

func TestResizeImageWithoutUseLogicalSizeIgnoresDevicePixelRatio(t *testing.T) {
	tester := newResizeTester(t)
	bytes := append([]byte{}, kBlueSquarePng...)
	resized := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(25), Height: ip(25)})
	if s := resolveAndGetSize(tester, resized, dpr(2.0)); s != ui.NewSize(25, 25) {
		t.Errorf("size %v", s)
	}
}

func TestResizeImageEqualKeysWhenEffectiveDimensionsMatch(t *testing.T) {
	tester := newResizeTester(t)
	bytes := append([]byte{}, kBlueSquarePng...)
	logicalKey := await(tester, NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(25), Height: ip(25), UseLogicalSize: true}).ObtainKey(dpr(2.0)))
	physicalKey := await(tester, NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(50), Height: ip(50)}).ObtainKey(dpr(2.0)))
	if !KeysEqual(logicalKey, physicalKey) {
		t.Error("keys differ")
	}
}

func TestResizeImageDifferentKeysWhenDevicePixelRatioDiffers(t *testing.T) {
	tester := newResizeTester(t)
	bytes := append([]byte{}, kBlueSquarePng...)
	resized := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(25), Height: ip(25), UseLogicalSize: true})
	at1x := await(tester, resized.ObtainKey(dpr(1.0)))
	at2x := await(tester, resized.ObtainKey(dpr(2.0)))
	if KeysEqual(at1x, at2x) {
		t.Error("keys equal")
	}
}

func TestResizeImageUseLogicalSizeWithFit(t *testing.T) {
	tester := newResizeTester(t)
	resized := NewResizeImage(NewMemoryImage(append([]byte{}, kBlueSquarePng...)),
		ResizeImageOptions{Width: ip(12), Height: ip(25), Policy: ResizeImagePolicyFit, UseLogicalSize: true})
	if s := resolveAndGetSize(tester, resized, dpr(2.0)); s != ui.NewSize(24, 24) {
		t.Errorf("size %v", s)
	}
}

func TestResizeImageRefusesUpscaling(t *testing.T) {
	tester := newResizeTester(t)
	bytes := append([]byte{}, kTransparentImage...)
	expectImageSize(t, tester, NewMemoryImage(bytes), 1, 1)
	resizeDims := ui.NewSize(50, 50)
	resized := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: ip(50), Height: ip(50)})
	if s := resolveAndGetSize(tester, resized, ImageConfiguration{Size: &resizeDims}); s != ui.NewSize(1, 1) {
		t.Errorf("size %v", s)
	}
}

func TestResizeImagePolicyFit(t *testing.T) {
	rawSizes := map[*byte][2]float64{&kBlueSquarePng[0]: {50, 50}, &kBluePortraitPng[0]: {50, 100}, &kBlueLandscapePng[0]: {100, 50}}
	cases := []struct {
		name           string
		raw            []byte
		width, height  int // 0: null
		allowUpscaling bool
		wantW, wantH   float64
		group          bool
	}{
		{"constrains square image to bounded portrait rect", kBlueSquarePng, 25, 50, false, 25, 25, false},
		{"constrains square image to bounded landscape rect", kBlueSquarePng, 50, 25, false, 25, 25, false},
		{"constrains square image to bounded square", kBlueSquarePng, 25, 25, false, 25, 25, false},
		{"constrains square image to bounded width", kBlueSquarePng, 25, 0, false, 25, 25, false},
		{"constrains square image to bounded height", kBlueSquarePng, 0, 25, false, 25, 25, false},
		{"constrains portrait image to bounded portrait rect", kBluePortraitPng, 25, 60, false, 25, 50, false},
		{"constrains portrait image to bounded landscape rect", kBluePortraitPng, 60, 25, false, 12, 25, false},
		{"constrains portrait image to bounded square", kBluePortraitPng, 25, 25, false, 12, 25, false},
		{"constrains portrait image to bounded width", kBluePortraitPng, 25, 0, false, 25, 50, false},
		{"constrains portrait image to bounded height", kBluePortraitPng, 0, 25, false, 12, 25, false},
		{"constrains landscape image to bounded portrait rect", kBlueLandscapePng, 25, 60, false, 25, 12, false},
		{"constrains landscape image to bounded landscape rect", kBlueLandscapePng, 60, 25, false, 50, 25, false},
		{"constrains landscape image to bounded square", kBlueLandscapePng, 25, 25, false, 25, 12, false},
		{"constrains landscape image to bounded width", kBlueLandscapePng, 25, 0, false, 25, 12, false},
		{"constrains landscape image to bounded height", kBlueLandscapePng, 0, 25, false, 50, 25, false},
		{"leaves image as-is if constraints are bigger than image", kBlueSquarePng, 120, 100, false, 50, 50, false},
		{"constrains square image to bounded portrait rect", kBlueSquarePng, 100, 200, true, 100, 100, true},
		{"constrains square image to bounded landscape rect", kBlueSquarePng, 200, 100, true, 100, 100, true},
		{"constrains square image to bounded square", kBlueSquarePng, 100, 100, true, 100, 100, true},
		{"constrains square image to bounded width", kBlueSquarePng, 100, 0, true, 100, 100, true},
		{"constrains square image to bounded height", kBlueSquarePng, 0, 100, true, 100, 100, true},
		{"constrains portrait image to bounded portrait rect", kBluePortraitPng, 100, 250, true, 100, 200, true},
		{"constrains portrait image to bounded landscape rect", kBluePortraitPng, 400, 200, true, 100, 200, true},
		{"constrains portrait image to bounded square", kBluePortraitPng, 200, 200, true, 100, 200, true},
		{"constrains portrait image to bounded width", kBluePortraitPng, 100, 0, true, 100, 200, true},
		{"constrains portrait image to bounded height", kBluePortraitPng, 0, 200, true, 100, 200, true},
		{"constrains landscape image to bounded portrait rect", kBlueLandscapePng, 200, 400, true, 200, 100, true},
		{"constrains landscape image to bounded landscape rect", kBlueLandscapePng, 250, 100, true, 200, 100, true},
		{"constrains landscape image to bounded square", kBlueLandscapePng, 200, 200, true, 200, 100, true},
		{"constrains landscape image to bounded width", kBlueLandscapePng, 200, 0, true, 200, 100, true},
		{"constrains landscape image to bounded height", kBlueLandscapePng, 0, 100, true, 200, 100, true},
		{"shrinks image if constraints are smaller than image", kBlueSquarePng, 25, 30, true, 25, 25, true},
	}
	for _, c := range cases {
		group := "allowResize=false"
		if c.group {
			group = "allowResize=true"
		}
		t.Run(group+"/"+c.name, func(t *testing.T) {
			tester := newResizeTester(t)
			rawImage := NewMemoryImage(append([]byte{}, c.raw...))
			raw := rawSizes[&c.raw[0]]
			expectImageSize(t, tester, rawImage, raw[0], raw[1])
			var w, h *int
			if c.width != 0 {
				w = ip(c.width)
			}
			if c.height != 0 {
				h = ip(c.height)
			}
			resized := NewResizeImage(rawImage, ResizeImageOptions{Width: w, Height: h, Policy: ResizeImagePolicyFit, AllowUpscaling: c.allowUpscaling})
			expectImageSize(t, tester, resized, c.wantW, c.wantH)
		})
	}
}

func TestResizeImageDoesNotResizeWhenNoSizeIsPassed(t *testing.T) {
	tester := newResizeTester(t)
	imageProvider := NewMemoryImage(append([]byte{}, kTransparentImage...))
	expectImageSize(t, tester, imageProvider, 1, 1)
	resized := ResizeImageResizeIfNeeded(nil, nil, imageProvider)
	expectImageSize(t, tester, resized, 1, 1)
}

func TestResizeImageStoresValues(t *testing.T) {
	newResizeTester(t)
	memoryImage := NewMemoryImage(append([]byte{}, kTransparentImage...))
	memoryImage.Resolve(ImageConfigurationEmpty)
	resizeImage := NewResizeImage(memoryImage, ResizeImageOptions{Width: ip(10), Height: ip(20)})
	if *resizeImage.Width != 10 || *resizeImage.Height != 20 || resizeImage.ImageProvider != memoryImage {
		t.Error("values")
	}
	if memoryImage.Resolve(ImageConfigurationEmpty) == resizeImage.Resolve(ImageConfigurationEmpty) {
		t.Error("same stream")
	}
}

func TestResizeImageTakesOneDim(t *testing.T) {
	newResizeTester(t)
	memoryImage := NewMemoryImage(append([]byte{}, kTransparentImage...))
	resizeImage := NewResizeImage(memoryImage, ResizeImageOptions{Width: ip(10)})
	if *resizeImage.Width != 10 || resizeImage.Height != nil || resizeImage.ImageProvider != memoryImage {
		t.Error("values")
	}
	if memoryImage.Resolve(ImageConfigurationEmpty) == resizeImage.Resolve(ImageConfigurationEmpty) {
		t.Error("same stream")
	}
}

func TestResizeImageFormsClosure(t *testing.T) {
	tester := newResizeTester(t)
	memoryImage := NewMemoryImage(append([]byte{}, kTransparentImage...))
	resizeImage := NewResizeImage(memoryImage, ResizeImageOptions{Width: ip(123), Height: ip(321), AllowUpscaling: true})
	called := false
	decode := func(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecWithSizeOptions) async.Future[ui.Codec] {
		return PaintingBindingInstance().InstantiateImageCodecWithSize(buffer, InstantiateImageCodecWithSizeOptions{
			GetTargetSize: func(intrinsicWidth, intrinsicHeight int) ui.TargetImageSize {
				called = true
				if len(opts) == 0 || opts[0].GetTargetSize == nil {
					t.Fatal("no getTargetSize")
				}
				targetSize := opts[0].GetTargetSize(intrinsicWidth, intrinsicHeight)
				if *targetSize.Width != 123 || *targetSize.Height != 321 {
					t.Errorf("target %v", targetSize)
				}
				return targetSize
			},
		})
	}
	completer := resizeImage.LoadImage(await(tester, resizeImage.ObtainKey(ImageConfigurationEmpty)), decode)
	done := async.NewCompleter[async.Void]()
	completer.AddListener(NewImageStreamListener(func(info *ImageInfo, _ bool) { info.Dispose(); done.Complete(async.Void{}) }))
	await(tester, done.Future())
	if !called {
		t.Error("getTargetSize not called")
	}
}

func TestResizeImageHandlesSyncObtainKey(t *testing.T) {
	newResizeTester(t)
	memoryImage := NewMemoryImage(append([]byte{}, kTransparentImage...))
	resizeImage := NewResizeImage(memoryImage, ResizeImageOptions{Width: ip(123), Height: ip(321)})
	isAsync, keyObtained := false, false
	async.Then(resizeImage.ObtainKey(ImageConfigurationEmpty), func(any) any {
		keyObtained = true
		if isAsync {
			t.Error("async")
		}
		return nil
	})
	isAsync = true
	if !keyObtained {
		t.Error("key not obtained synchronously")
	}
}

// asyncKeyMemoryImage's ObtainKey never completes synchronously.
type asyncKeyMemoryImage struct{ MemoryImage }

func (p *asyncKeyMemoryImage) ObtainKey(ImageConfiguration) async.Future[any] {
	return async.NewFuture(func() any { return &p.MemoryImage })
}

func TestResizeImageHandlesAsyncObtainKey(t *testing.T) {
	tester := newResizeTester(t)
	memoryImage := &asyncKeyMemoryImage{MemoryImage{Bytes: append([]byte{}, kTransparentImage...), Scale: 1}}
	memoryImage.SetSelf(memoryImage)
	resizeImage := NewResizeImage(memoryImage, ResizeImageOptions{Width: ip(123), Height: ip(321)})
	isAsync := false
	completer := async.NewCompleter[async.Void]()
	async.Then(resizeImage.ObtainKey(ImageConfigurationEmpty), func(any) any {
		if !isAsync {
			t.Error("sync")
		}
		completer.Complete(async.Void{})
		return nil
	})
	isAsync = true
	await(tester, completer.Future())
}
