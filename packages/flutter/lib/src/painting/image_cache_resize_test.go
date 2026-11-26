package painting_test

// Port of packages/flutter/test/painting/image_cache_resize_test.dart,
// image_cache_clearing_test.dart and image_cache_binding_test.dart
// (Flutter 3.50.1): all 4 tests.

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
)

func TestImageCacheResizingBasedOnCount(t *testing.T) {
	tester := newImageTester(t)
	c := ImageCacheInstance()
	c.SetMaximumSize(2)
	check := func(key, value, want int) {
		t.Helper()
		if got := resolveValue(tester, key, value); got != want {
			t.Errorf("TestImageProvider(%d, %d): %d, want %d", key, value, got, want)
		}
	}
	check(1, 1, 1)
	check(2, 2, 2)
	check(3, 3, 3)
	check(1, 4, 4)
	c.SetMaximumSize(0)
	check(1, 5, 5)
	check(1, 6, 6)
	c.SetMaximumSize(3)
	check(1, 7, 7)
	check(1, 8, 7)
}

func TestImageCacheResizingBasedOnSize(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8}) // 256 B.
	c := ImageCacheInstance()
	c.SetMaximumSizeBytes(256 * 2)
	check := func(key, value, want int) {
		t.Helper()
		info := await(tester, extractOneFrame(NewTestImageProvider(key, value, testImage).Resolve(ImageConfigurationEmpty)))
		if got := info.Self().(*TestImageInfo).Value; got != want {
			t.Errorf("TestImageProvider(%d, %d): %d, want %d", key, value, got, want)
		}
	}
	check(1, 1, 1)
	check(2, 2, 2)
	check(3, 3, 3)
	check(1, 4, 4)
	c.SetMaximumSizeBytes(0)
	check(1, 5, 5)
	check(1, 6, 6)
	c.SetMaximumSizeBytes(256 * 3)
	check(1, 7, 7)
	check(1, 8, 7)
}

func TestImageCacheClearingWhilePendingDoesNotCrash(t *testing.T) {
	tester := newImageTester(t)
	memoryImage := NewMemoryImage(append([]byte{}, kTransparentImage...))
	stream := memoryImage.Resolve(ImageConfigurationEmpty)
	completer := async.NewCompleter[async.Void]()
	old := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		completer.CompleteError(fmt.Errorf("%v", details.Exception))
	}
	defer func() { foundation.FlutterErrorOnError = old }()
	stream.AddListener(NewImageStreamListener(func(*ImageInfo, bool) { completer.Complete(async.Void{}) }))
	ImageCacheInstance().ClearLiveImages()
	await(tester, completer.Future())
}

// paintingTestBinding is image_cache_binding_test.dart's
// TestPaintingBinding: BindingBase with SchedulerBinding, ServicesBinding,
// PaintingBinding, recording handleMemoryPressure.
type paintingTestBinding struct {
	foundation.BindingBaseBase
	scheduler.SchedulerBindingMixin
	services.ServicesBindingMixin
	PaintingBindingMixin
	handled bool
}

func (b *paintingTestBinding) InitInstances()         { b.PaintingBindingMixin.InitInstances() }
func (b *paintingTestBinding) InitServiceExtensions() { b.ServicesBindingMixin.InitServiceExtensions() }
func (b *paintingTestBinding) Unlocked()              { b.SchedulerBindingMixin.Unlocked() }
func (b *paintingTestBinding) Evict(asset string)     { b.PaintingBindingMixin.Evict(asset) }
func (b *paintingTestBinding) HandleSystemMessage(m any) async.Future[async.Void] {
	return b.PaintingBindingMixin.HandleSystemMessage(m)
}
func (b *paintingTestBinding) HandleMemoryPressure() {
	b.PaintingBindingMixin.HandleMemoryPressure()
	b.handled = true
}

const isolatedEnv = "FLUTTER_GO_PAINTING_ISOLATED_TEST"

// isolated re-runs the test in its own process (a Dart test file has its
// own isolate, so its own binding); it returns true in that process.
func isolated(t *testing.T) bool {
	t.Helper()
	if os.Getenv(isolatedEnv) == t.Name() {
		return true
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), isolatedEnv+"="+t.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s in its own process: %v\n%s", t.Name(), err, out)
	}
	return false
}

func TestPaintingBindingWithMemoryPressureBeforeInitInstances(t *testing.T) {
	if !isolated(t) {
		return
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	ui.ChannelBuffersInstance().Push(services.SystemChannels.System.Name,
		services.SystemChannels.System.Codec.EncodeMessage(map[string]any{"type": "memoryPressure"}),
		func([]byte) {})
	b := &paintingTestBinding{}
	b.SchedulerBindingMixin.SetSelf(b)
	b.ServicesBindingMixin.SetSelf(b)
	b.PaintingBindingMixin.SetSelf(b)
	b.Construct(b, &b.SchedulerBindingMixin, &b.ServicesBindingMixin, &b.PaintingBindingMixin)
	if b.handled {
		t.Error("handled during initInstances")
	}
	if b.ImageCache() == nil || b.ImageCache().CurrentSize() != 0 {
		t.Error("imageCache")
	}
	loop.FlushMicrotasks() // allow microtasks to run
	if !b.handled {
		t.Error("not handled")
	}
}
