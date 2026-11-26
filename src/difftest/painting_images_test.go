package difftest

// Go side of flutter/test/painting_images_test.dart: the same scenes, in
// flutter_test's binding (tester.idle() is a zero elapse of the fake clock,
// tester.pump a pump), compared line by line with Flutter's output.

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/io"
)

func imgNorm(o any) string { return hashRE.ReplaceAllString(fmt.Sprint(o), "#00000") }

func imgSt(s ImageCacheStatus) string {
	b := func(v bool) int {
		if v {
			return 1
		}
		return 0
	}
	return fmt.Sprintf("p%dk%dl%d", b(s.Pending), b(s.KeepAlive), b(s.Live))
}

type MockCodec struct {
	frameCount, repetitionCount int
	numFramesAsked              int
	disposed                    bool
	next                        *async.Completer[*ui.FrameInfo]
}

func (c *MockCodec) FrameCount() int      { return c.frameCount }
func (c *MockCodec) RepetitionCount() int { return c.repetitionCount }
func (c *MockCodec) GetNextFrame() async.Future[*ui.FrameInfo] {
	c.numFramesAsked++
	return c.next.Future()
}
func (c *MockCodec) completeNextFrame(f *ui.FrameInfo) {
	c.next.Complete(f)
	c.next = async.NewCompleter[*ui.FrameInfo]()
}
func (c *MockCodec) Dispose() { c.disposed = true }

type TestCompleter struct{ ImageStreamCompleterBase }

func newTestCompleter() *TestCompleter {
	c := &TestCompleter{}
	c.InitImageStreamCompleter(c)
	return c
}

func (c *TestCompleter) set(image *ui.Image) { c.SetImage(NewImageInfo(image.Clone())) }

type ManifestBundle struct {
	services.CachingAssetBundle
	manifest map[string]any
}

func newManifestBundleD(manifest map[string]any) *ManifestBundle {
	b := &ManifestBundle{manifest: manifest}
	b.SetSelf(b)
	return b
}

func (b *ManifestBundle) Load(key string) async.Future[[]byte] {
	return async.Async(func() []byte {
		if key == "AssetManifest.bin" {
			return services.NewStandardMessageCodec().EncodeMessage(b.manifest)
		}
		panic(foundation.NewFlutterError("key not found"))
	})
}

type CapturingProvider struct {
	ImageProviderBase
	buffer *ui.ImmutableBuffer
}

func newCapturingProvider(buffer *ui.ImmutableBuffer) *CapturingProvider {
	p := &CapturingProvider{buffer: buffer}
	p.SetSelf(p)
	return p
}

func (p *CapturingProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

func (p *CapturingProvider) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	decode(p.buffer)
	return newTestCompleter()
}

type imgFixture struct {
	img8, img10, img16, img20, img40 *ui.Image
	buffer                           *ui.ImmutableBuffer
}

func makeImage(tester *flutter_test.Tester, w, h int) *ui.Image {
	c := async.NewCompleter[*ui.Image]()
	ui.DecodeImageFromPixels(make([]byte, w*h*4), w, h, ui.PixelFormatRgba8888, func(i *ui.Image) { c.Complete(i) })
	return flutter_test.AwaitFuture(tester, c.Future())
}

func imgOpt[T any](v T) *T { return &v }

var imageScenes = map[string]func(tester *flutter_test.Tester, f *imgFixture, log func(string)){
	"cache_lru": func(tester *flutter_test.Tester, f *imgFixture, log func(string)) {
		c := ImageCacheInstance()
		images := []*ui.Image{f.img8, f.img10, f.img16, f.img20, f.img40}
		dump := func(what string) {
			var s []string
			var contains []string
			for k := 0; k < 7; k++ {
				s = append(s, imgSt(c.StatusForKey(k)))
				if c.ContainsKey(k) {
					contains = append(contains, dart.ToString(k))
				}
			}
			log(fmt.Sprintf("%s: size %d bytes %d live %d pending %d %s contains [%s]", what, c.CurrentSize(), c.CurrentSizeBytes(),
				c.LiveImageCount(), c.PendingImageCount(), strings.Join(s, " "), strings.Join(contains, ", ")))
		}
		loaderCalled := func() ImageStreamCompleter { panic(dart.StateError{Message: "loader called"}) }
		c.SetMaximumSize(3)
		for k := 0; k < 5; k++ {
			completer := newTestCompleter()
			completer.set(images[k])
			c.PutIfAbsent(k, func() ImageStreamCompleter { return completer })
			dump(fmt.Sprintf("put %d", k))
		}
		c.PutIfAbsent(2, loaderCalled)
		dump("touch 2")
		c.SetMaximumSizeBytes(1200)
		dump("maxBytes 1200")
		c.SetMaximumSizeBytes(100 << 20)
		c.SetMaximumSize(4)
		pending := newTestCompleter()
		c.PutIfAbsent(5, func() ImageStreamCompleter { return pending })
		dump("pending 5")
		l := NewImageStreamListener(func(i *ImageInfo, s bool) { i.Dispose() })
		pending.AddListener(l)
		dump("listen 5")
		pending.set(f.img40)
		dump("complete 5")
		log(fmt.Sprintf("evict 3 %t", c.Evict(3)))
		log(fmt.Sprintf("evict 6 %t", c.Evict(6)))
		log(fmt.Sprintf("evict 5 includeLive:false %t", c.Evict(5, ImageCacheEvictOptions{IncludeLive: imgOpt(false)})))
		dump("evicted")
		c.PutIfAbsent(5, loaderCalled)
		dump("re-put live 5")
		pending.RemoveListener(l)
		dump("unlisten 5")
		c.SetMaximumSize(1)
		dump("max 1")
		c.Clear()
		dump("clear")
		c.ClearLiveImages()
		dump("clearLive")
		c.SetMaximumSize(4)
		c.SetMaximumSizeBytes(256 + 400)
		for k, i := range []*ui.Image{f.img8, f.img10} {
			e := newTestCompleter()
			e.set(i)
			c.PutIfAbsent(k, func() ImageStreamCompleter { return e })
		}
		dump("exactly full")
		c.Clear()
		c.SetMaximumSizeBytes(256)
		e8 := newTestCompleter()
		e8.set(f.img8)
		c.PutIfAbsent(0, func() ImageStreamCompleter { return e8 })
		dump("image of max bytes")
		c.SetMaximumSizeBytes(0)
		c.SetMaximumSizeBytes(0)
		noBytes := newTestCompleter()
		c.PutIfAbsent(1, func() ImageStreamCompleter { return noBytes })
		dump("no bytes pending")
		noBytes.set(f.img8)
		dump("no bytes complete")
		c.ClearLiveImages()
		c.SetMaximumSizeBytes(100 << 20)
		c.SetMaximumSize(0)
		disabled := newTestCompleter()
		c.PutIfAbsent(6, func() ImageStreamCompleter { return disabled })
		dump("disabled put")
		disabled.set(f.img8)
		dump("disabled complete")
		r := c.PutIfAbsent(7, func() ImageStreamCompleter { panic(dart.StateError{Message: "x"}) }, ImageCachePutIfAbsentOptions{
			OnError: func(e any, _ dart.StackTrace) {
				if se, ok := e.(dart.StateError); ok {
					log("onError " + se.Message)
				} else {
					log(fmt.Sprint("onError ", e))
				}
			}})
		if r == nil {
			log("result null")
		} else {
			log(fmt.Sprint("result ", r))
		}
		tester.Pump()
	},
	"stream_listeners": func(tester *flutter_test.Tester, f *imgFixture, log func(string)) {
		pre := NewImageStream()
		var preListeners []ImageStreamListener
		for _, n := range []string{"a", "b", "c"} {
			preListeners = append(preListeners, NewImageStreamListener(func(i *ImageInfo, sync bool) {
				log(fmt.Sprintf("pre %s sync %t", n, sync))
				i.Dispose()
			}))
		}
		for _, l := range preListeners {
			pre.AddListener(l)
		}
		pre.RemoveListener(preListeners[1])
		log(imgNorm(pre))
		preC := newTestCompleter()
		preC.set(f.img8)
		pre.SetCompleter(preC)
		c := newTestCompleter()
		stream := NewImageStream()
		log(imgNorm(stream))
		var l2 ImageStreamListener
		l1 := NewImageStreamListener(func(i *ImageInfo, sync bool) {
			log(fmt.Sprintf("l1 %s sync %t", imgNorm(i), sync))
			i.Dispose()
		})
		l2 = NewImageStreamListener(func(i *ImageInfo, sync bool) {
			log(fmt.Sprintf("l2 %s sync %t", imgNorm(i), sync))
			i.Dispose()
			stream.RemoveListener(l2)
		}, ImageStreamListenerOptions{OnChunk: func(e *ImageChunkEvent) { log("chunk " + imgNorm(e)) }})
		stream.AddListener(l1)
		stream.AddListener(l2)
		log(imgNorm(stream))
		stream.SetCompleter(c)
		log(imgNorm(stream))
		log(imgNorm(c))
		c.ReportImageChunkEvent(NewImageChunkEvent(1, imgOpt(3)))
		c.ReportImageChunkEvent(NewImageChunkEvent(3, nil))
		c.set(f.img8)
		log(imgNorm(c))
		c.set(f.img10)
		stream.AddListener(NewImageStreamListener(func(i *ImageInfo, sync bool) {
			log(fmt.Sprintf("late %s sync %t", imgNorm(i), sync))
			i.Dispose()
		}))
		c.AddEphemeralErrorListener(func(any, dart.StackTrace) { log("ephemeral (not called)") })
		log(imgNorm(c))
		log(fmt.Sprintf("key is completer %t", stream.Key() == ImageStreamCompleter(c)))
		h := c.KeepAlive()
		stream.RemoveListener(l1)
		log(imgNorm(c))
		h.Dispose()
		log(imgNorm(c))
	},
	"stream_errors": func(tester *flutter_test.Tester, f *imgFixture, log func(string)) {
		var reported []*foundation.FlutterErrorDetails
		old := foundation.FlutterErrorOnError
		foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) { reported = append(reported, d) }
		dumpReported := func() {
			for _, d := range reported {
				info := "null"
				if d.InformationCollector != nil {
					var parts []string
					for n := range d.InformationCollector() {
						parts = append(parts, imgNorm(n))
					}
					info = "[" + strings.Join(parts, ", ") + "]"
				}
				ctx := "null"
				if d.Context != nil {
					ctx = d.Context.String()
				}
				lib := "null"
				if d.Library != nil {
					lib = *d.Library
				}
				log(fmt.Sprintf("reported %v library %s context %s silent %t info %s", d.Exception, lib, ctx, d.Silent, info))
			}
			reported = nil
		}
		c := newTestCompleter()
		c.AddEphemeralErrorListener(func(e any, _ dart.StackTrace) { log(fmt.Sprint("ephemeral1 ", e)) })
		c.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}, ImageStreamListenerOptions{OnError: func(e any, _ dart.StackTrace) {
			log(fmt.Sprint("listener1 ", e))
			panic("listener1 threw")
		}}))
		c.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}, ImageStreamListenerOptions{OnError: func(e any, _ dart.StackTrace) {
			log(fmt.Sprint("listener2 ", e))
			panic(e)
		}}))
		c.ReportError("boom", ImageStreamCompleterReportErrorOptions{Context: foundation.NewErrorDescription("testing"), Silent: true,
			InformationCollector: func() iter.Seq[foundation.DiagnosticsNode] {
				return func(yield func(foundation.DiagnosticsNode) bool) { yield(foundation.NewErrorDescription("extra info")) }
			}})
		dumpReported()
		c.AddEphemeralErrorListener(func(e any, _ dart.StackTrace) { log(fmt.Sprint("ephemeral2 ", e)) })
		c.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}, ImageStreamListenerOptions{OnError: func(e any, _ dart.StackTrace) {
			log(fmt.Sprint("listener3 ", e))
			panic("listener3 threw")
		}}))
		dumpReported()
		log(imgNorm(c))
		unheard := newTestCompleter()
		unheard.ReportError("nobody listens")
		dumpReported()
		imageThrows := newTestCompleter()
		imageThrows.AddListener(NewImageStreamListener(func(i *ImageInfo, s bool) {
			i.Dispose()
			panic("onImage threw")
		}, ImageStreamListenerOptions{OnError: func(e any, _ dart.StackTrace) { log(fmt.Sprint("onError ", e)) }}))
		imageThrows.set(f.img8)
		dumpReported()
		imageThrows.AddListener(NewImageStreamListener(func(i *ImageInfo, s bool) {
			i.Dispose()
			panic("sync onImage threw")
		}))
		dumpReported()
		one := async.NewCompleter[*ImageInfo]()
		NewOneFrameImageStreamCompleter(one.Future(), OneFrameImageStreamCompleterOptions{
			InformationCollector: func() iter.Seq[foundation.DiagnosticsNode] {
				return func(yield func(foundation.DiagnosticsNode) bool) { yield(foundation.NewErrorDescription("one")) }
			}})
		one.CompleteError(async.PanicError{Value: "one failed"})
		codec := async.NewCompleter[ui.Codec]()
		NewMultiFrameImageStreamCompleter(codec.Future(), 2.0, MultiFrameImageStreamCompleterOptions{DebugLabel: imgOpt("multi")})
		codec.CompleteError(async.PanicError{Value: "codec failed"})
		tester.Loop().Elapse(0)
		dumpReported()
		foundation.FlutterErrorOnError = old
	},
	"multi_frame": func(tester *flutter_test.Tester, f *imgFixture, log func(string)) {
		for _, dilation := range []float64{1.0, 2.0, 0.5} {
			scheduler.SetTimeDilation(dilation)
			codec := &MockCodec{frameCount: 3, repetitionCount: 1, next: async.NewCompleter[*ui.FrameInfo]()}
			cc := async.NewCompleter[ui.Codec]()
			c := NewMultiFrameImageStreamCompleter(cc.Future(), 2.0, MultiFrameImageStreamCompleterOptions{DebugLabel: imgOpt("anim")})
			start := scheduler.SchedulerBindingInstance().CurrentSystemFrameTimeStamp()
			durations := []time.Duration{100 * time.Millisecond, 250 * time.Millisecond, 40 * time.Millisecond}
			frames := []*ui.Image{f.img8, f.img10, f.img16}
			t := 0
			d := dart.DoubleToString(dilation)
			l := NewImageStreamListener(func(i *ImageInfo, sync bool) {
				log(fmt.Sprintf("dilation %s t=%d frame %d %s", d, t, i.Image.Width, imgNorm(i)))
				i.Dispose()
			})
			c.AddListener(l)
			cc.Complete(codec)
			served := 0
			for step := 0; step < 40; step++ {
				if codec.numFramesAsked > served {
					codec.completeNextFrame(&ui.FrameInfo{Duration: durations[served%3], Image: frames[served%3].Clone()})
					served++
				}
				tester.Loop().Elapse(0)
				tester.Pump(30 * time.Millisecond)
				t += 30
				if step%5 == 0 {
					log(fmt.Sprintf("dilation %s t=%d asked %d disposed %t transient %d", d, t, codec.numFramesAsked, codec.disposed,
						scheduler.SchedulerBindingInstance().TransientCallbackCount()))
				}
			}
			log(fmt.Sprintf("end %d disposed %t %t", codec.numFramesAsked, codec.disposed, start >= 0))
			c.RemoveListener(l)
			log(fmt.Sprintf("removed: disposed %t", codec.disposed))
		}
		scheduler.SetTimeDilation(1.0)
		// A restarted decode completing before the pending frame callback
		// ran: one frame callback only.
		codec := &MockCodec{frameCount: 2, repetitionCount: -1, next: async.NewCompleter[*ui.FrameInfo]()}
		c := NewMultiFrameImageStreamCompleter(foundation.NewSynchronousFuture[ui.Codec](codec), 1.0)
		l := NewImageStreamListener(func(i *ImageInfo, sync bool) { i.Dispose() })
		c.AddListener(l)
		codec.completeNextFrame(&ui.FrameInfo{Image: f.img8.Clone()})
		tester.Loop().Elapse(0)
		b := scheduler.SchedulerBindingInstance()
		log(fmt.Sprintf("scheduled %d", b.TransientCallbackCount()))
		h := c.KeepAlive()
		c.RemoveListener(l)
		c.AddListener(l)
		codec.completeNextFrame(&ui.FrameInfo{Image: f.img10.Clone()})
		tester.Loop().Elapse(0)
		log(fmt.Sprintf("rescheduled %d asked %d", b.TransientCallbackCount(), codec.numFramesAsked))
		tester.Pump()
		c.RemoveListener(l)
		h.Dispose()
	},
	"asset_variants": func(tester *flutter_test.Tester, f *imgFixture, log func(string)) {
		manifests := [][]*float64{
			{nil, imgOpt(1.5), imgOpt(2.0), imgOpt(3.0), imgOpt(4.0)},
			{imgOpt(2.0), imgOpt(3.0)},
			{imgOpt(0.5), nil},
			{nil, imgOpt(3.0)},
			{imgOpt(2.0), imgOpt(2.0), imgOpt(1.0)},
			{},
		}
		ratios := []*float64{nil, imgOpt(0.25), imgOpt(0.5), imgOpt(0.75), imgOpt(1.0), imgOpt(1.2), imgOpt(1.5), imgOpt(1.74),
			imgOpt(1.75), imgOpt(1.76), imgOpt(1.99), imgOpt(2.0), imgOpt(2.24), imgOpt(2.25), imgOpt(2.5), imgOpt(2.51), imgOpt(2.6),
			imgOpt(3.0), imgOpt(3.4), imgOpt(3.5), imgOpt(3.51), imgOpt(4.0), imgOpt(5.0)}
		dbl := func(p *float64) string {
			if p == nil {
				return "null"
			}
			return dart.DoubleToString(*p)
		}
		for m, manifest := range manifests {
			variants := []any{}
			for i, d := range manifest {
				if d == nil {
					variants = append(variants, map[string]any{"asset": "img.png"})
				} else {
					variants = append(variants, map[string]any{"asset": fmt.Sprintf("v%d/%sx/img.png", i, dbl(d)), "dpr": *d})
				}
			}
			bundle := newManifestBundleD(map[string]any{"img.png": variants})
			image := NewAssetImage("img.png", AssetImageOptions{Bundle: bundle})
			var chosen []string
			for _, r := range ratios {
				var key *AssetBundleImageKey
				async.Then(image.ObtainKey(ImageConfiguration{DevicePixelRatio: r}), func(k any) any {
					kk := k.(AssetBundleImageKey)
					key = &kk
					return nil
				})
				tester.Loop().Elapse(0)
				chosen = append(chosen, fmt.Sprintf("%s:%s@%s", dbl(r), key.Name, dart.DoubleToString(key.Scale)))
			}
			log(fmt.Sprintf("manifest %d %s", m, strings.Join(chosen, " ")))
		}
		pkg := NewAssetImage("a.png", AssetImageOptions{Package: imgOpt("pkg")})
		log(fmt.Sprintf("package keyName %s %s", pkg.KeyName(), imgNorm(pkg)))
	},
	"resize": func(tester *flutter_test.Tester, f *imgFixture, log func(string)) {
		intrinsic := [][2]int{{50, 50}, {50, 100}, {100, 50}, {33, 7}, {7, 33}, {1, 1}, {640, 480}, {3, 1000}}
		dims := [][2]*int{{imgOpt(25), imgOpt(50)}, {imgOpt(50), imgOpt(25)}, {imgOpt(25), nil}, {nil, imgOpt(25)},
			{imgOpt(120), imgOpt(100)}, {imgOpt(12), imgOpt(25)}, {imgOpt(100), imgOpt(200)}, {imgOpt(1), imgOpt(1)}, {imgOpt(999), imgOpt(3)}}
		is := func(p *int) string {
			if p == nil {
				return "null"
			}
			return dart.ToString(*p)
		}
		label := func(c ImageStreamCompleter) string {
			if l := c.DebugLabel(); l != nil {
				return *l
			}
			return "null"
		}
		for _, policy := range ResizeImagePolicyValues {
			for _, up := range []bool{false, true} {
				for _, d := range dims {
					r := NewResizeImage(newCapturingProvider(f.buffer), ResizeImageOptions{Width: d[0], Height: d[1], Policy: policy, AllowUpscaling: up})
					key := flutter_test.AwaitFuture(tester, r.ObtainKey(ImageConfigurationEmpty))
					var got ui.TargetImageSizeCallback
					c := r.LoadImage(key, func(b *ui.ImmutableBuffer, opts ...InstantiateImageCodecWithSizeOptions) async.Future[ui.Codec] {
						if len(opts) > 0 {
							got = opts[0].GetTargetSize
						}
						return async.NewCompleter[ui.Codec]().Future()
					})
					var sizes []string
					for _, i := range intrinsic {
						func() {
							defer func() {
								if recover() != nil {
									sizes = append(sizes, "error")
								}
							}()
							ts := got(i[0], i[1])
							sizes = append(sizes, is(ts.Width)+"x"+is(ts.Height))
						}()
					}
					log(fmt.Sprintf("%s up %t %sx%s %s: %s", policy.Name(), up, is(d[0]), is(d[1]), label(c), strings.Join(sizes, " ")))
				}
			}
		}
		for _, dpr := range []*float64{nil, imgOpt(1.0), imgOpt(1.5), imgOpt(2.0), imgOpt(2.625), imgOpt(3.0), imgOpt(0.3)} {
			for _, logical := range []bool{false, true} {
				r := NewResizeImage(newCapturingProvider(f.buffer), ResizeImageOptions{Width: imgOpt(33), Height: imgOpt(17), UseLogicalSize: logical})
				key := flutter_test.AwaitFuture(tester, r.ObtainKey(ImageConfiguration{DevicePixelRatio: dpr}))
				c := r.LoadImage(key, func(*ui.ImmutableBuffer, ...InstantiateImageCodecWithSizeOptions) async.Future[ui.Codec] {
					return async.NewCompleter[ui.Codec]().Future()
				})
				ds := "null"
				if dpr != nil {
					ds = dart.DoubleToString(*dpr)
				}
				log(fmt.Sprintf("dpr %s logical %t %s", ds, logical, label(c)))
			}
		}
	},
	"to_strings": func(tester *flutter_test.Tester, f *imgFixture, log func(string)) {
		log(ImageConfigurationEmpty.String())
		ios, android := foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid
		rtl, ltr := ui.TextDirectionRtl, ui.TextDirectionLtr
		locale := ui.NewLocale("en", "US")
		log(ImageConfiguration{DevicePixelRatio: imgOpt(1.25), Locale: &locale, TextDirection: &rtl,
			Size: imgOpt(ui.NewSize(10, 20.5)), Platform: &ios}.String())
		log(ImageConfiguration{DevicePixelRatio: imgOpt(2.75)}.String())
		log(ImageConfiguration{Size: imgOpt(ui.NewSize(0, 0)), Platform: &android}.String())
		log(ImageConfiguration{DevicePixelRatio: imgOpt(1.0)}.CopyWith(ImageConfiguration{TextDirection: &ltr}).String())
		log(fmt.Sprint(ImageConfiguration{DevicePixelRatio: imgOpt(1.0)}.Equal(ImageConfiguration{DevicePixelRatio: imgOpt(1.0)})))
		log(NewNetworkImage("https://x/y.png", NetworkImageOptions{Scale: imgOpt(2.5), Headers: map[string]string{"a": "b"}}).String())
		log(NewNetworkImage("u").String())
		log(NewExactAssetImage("e.png", ExactAssetImageOptions{Scale: imgOpt(3.0), Package: imgOpt("p")}).String())
		log(NewFileImage(io.NewFile("/tmp/f.png"), FileImageOptions{Scale: imgOpt(0.5)}).String())
		log(imgNorm(NewMemoryImage(make([]byte, 3), MemoryImageOptions{Scale: imgOpt(1.05)})))
		bytes := make([]byte, 2)
		log(fmt.Sprintf("%t %t %t", NewMemoryImage(bytes).Equal(NewMemoryImage(bytes)),
			NewMemoryImage(bytes).Equal(NewMemoryImage(bytes, MemoryImageOptions{Scale: imgOpt(2.0)})),
			NewMemoryImage(bytes).Equal(NewMemoryImage(make([]byte, 2)))))
		log(imgNorm(AssetBundleImageKey{Bundle: newManifestBundleD(map[string]any{}), Name: "n", Scale: 1.5}))
		log(imgNorm(ImageCacheInstance().StatusForKey("nothing")))
		log(imgNorm(NewImageChunkEvent(10, nil)))
		log(imgNorm(NewImageInfo(f.img20, ImageInfoOptions{Scale: imgOpt(1.5)})))
		log(imgNorm(NewImageInfo(f.img20, ImageInfoOptions{Scale: imgOpt(3.0), DebugLabel: imgOpt("label")})))
		u, _ := url.Parse("https://a/b")
		log(NewNetworkImageLoadException(404, u).String())
		info := ImageSizeInfo{Source: "src", DisplaySize: ui.NewSize(10, 20), ImageSize: ui.NewSize(33, 7)}
		log(fmt.Sprintf("%v %d %d %s", info, info.DisplaySizeInBytes(), info.DecodedSizeInBytes(), dart.JSONEncode(info.ToJson())))
		info2 := ImageSizeInfo{DisplaySize: ui.NewSize(1.5, 2.5), ImageSize: ui.NewSize(1000, 999)}
		log(fmt.Sprintf("%v %d %d %s", info2, info2.DisplaySizeInBytes(), info2.DecodedSizeInBytes(), dart.JSONEncode(info2.ToJson())))
		log(imgNorm(NewImageStream()))
		one := NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(NewImageInfo(f.img8.Clone(), ImageInfoOptions{Scale: imgOpt(2.0)})))
		log(imgNorm(one))
		log(ResizeImagePolicyFit.String())
		log(WebHtmlElementStrategyPrefer.String())
	},
}

func TestPaintingImagesMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_images.json")
	if err != nil {
		t.Fatalf("no reference (%v); run the Flutter side: flutter test test/painting_images_test.dart", err)
	}
	var want map[string][]string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	for name, scene := range imageScenes {
		t.Run(name, func(t *testing.T) {
			tester := flutter_test.New(t)
			// Start from an empty cache: other difftests of this package share the
			// process-wide cache (each Dart test file has its own isolate).
			ImageCacheInstance().Clear()
			ImageCacheInstance().ClearLiveImages()
			f := &imgFixture{
				img8: makeImage(tester, 8, 8), img10: makeImage(tester, 10, 10), img16: makeImage(tester, 16, 16),
				img20: makeImage(tester, 20, 10), img40: makeImage(tester, 40, 40),
			}
			f.buffer = flutter_test.AwaitFuture(tester, ui.ImmutableBufferFromUint8List(make([]byte, 4)))
			var got []string
			scene(tester, f, func(s string) { got = append(got, s) })
			c := ImageCacheInstance()
			c.Clear()
			c.ClearLiveImages()
			c.SetMaximumSize(1000)
			c.SetMaximumSizeBytes(100 << 20)
			compareImageLines(t, want[name], got)
		})
	}
	t.Run("decode", func(t *testing.T) {
		tester := flutter_test.New(t)
		var got []string
		for _, b64 := range []string{
			"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg==",
			"iVBORw0KGgoAAAANSUhEUgAAAAMAAAACCAYAAACddGYaAAAAEUlEQVR4nGP4z8DwH4YZkDkAm34L9XKwuTwAAAAASUVORK5CYII=",
			"iVBORw0KGgoAAAANSUhEUgAAABEAAAAFCAYAAACjKgd3AAAAFElEQVR4nGP4z8Dwn1LMMGoIBgYAInqpV+p9ZlMAAAAASUVORK5CYII=",
			"R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7",
		} {
			d, _ := base64.StdEncoding.DecodeString(b64)
			image := flutter_test.AwaitFuture(tester, DecodeImageFromList(d))
			got = append(got, fmt.Sprintf("%dx%d", image.Width, image.Height))
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					var e dart.Exception
					if err, ok := r.(error); ok && errors.As(err, &e) {
						got = append(got, "error _Exception")
					} else {
						got = append(got, fmt.Sprintf("error %T", r))
					}
				}
			}()
			flutter_test.AwaitFuture(tester, DecodeImageFromList([]byte{1, 2, 3}))
			got = append(got, "no error")
		}()
		compareImageLines(t, want["decode"], got)
	})
}

func compareImageLines(t *testing.T, want, got []string) {
	t.Helper()
	if len(want) == 0 {
		t.Fatal("scene missing from the reference")
	}
	n := max(len(want), len(got))
	for i := 0; i < n; i++ {
		var w, g string
		if i < len(want) {
			w = want[i]
		}
		if i < len(got) {
			g = got[i]
		}
		if w != g {
			t.Errorf("line %d:\nflutter %s\ngo      %s", i, w, g)
		}
	}
}
