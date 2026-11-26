package difftest

// Go side of flutter/test/impeller/fragment_shader_test.dart: uniform
// reflection, slot and sampler errors, ImageFilter.shader's errors,
// toString and ==, and rendered pixels, compared exactly with Flutter run
// with --enable-impeller. The shaders (flutter/shaders/*.frag) are
// compiled with the Impeller SDK's impellerc for a macOS app (--sksl
// --runtime-stage-metal, as the flutter tool does), and flutter-go reads
// the Metal stage where the tester reads the Vulkan one.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	. "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func fragErrorType(r any) string {
	switch e := r.(type) {
	case dart.IndexError:
		return "IndexError: " + e.Error()
	case dart.RangeError:
		return "RangeError: " + e.Error()
	case dart.ArgumentError:
		return "ArgumentError: " + e.Error()
	case dart.StateError:
		return "StateError: " + e.Error()
	case *dart.AssertionError:
		return "AssertionError: " + dart.ToString(e.AssertionMessage())
	case string:
		return "String: " + e
	case dart.Exception:
		return "Exception: " + e.Error()
	case error:
		return "other: " + e.Error()
	}
	return "other: " + dart.ToString(r)
}

func errT(f func()) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = fragErrorType(r)
		}
	}()
	f()
	return "no error"
}

// compileDifftestShaders compiles flutter/shaders/*.frag and returns them
// by asset key ("shaders/<name>.frag").
func compileDifftestShaders(t *testing.T) map[string][]byte {
	impellerc := filepath.Join("..", "third_party", "impeller_sdk", "bin", "impellerc")
	if _, err := os.Stat(impellerc); err != nil {
		t.Skipf("no impellerc (%v); run src/scripts/fetch_impeller_sdk.sh", err)
	}
	srcs, _ := filepath.Glob(filepath.Join("flutter", "shaders", "*.frag"))
	dir := t.TempDir()
	out := map[string][]byte{}
	for _, src := range srcs {
		sl := filepath.Join(dir, filepath.Base(src)+".iplr")
		args := func(targets ...string) []string {
			return append(targets, "--iplr", "--sl="+sl, "--spirv="+sl+".spirv", "--input="+src, "--input-type=frag",
				"--include="+filepath.Dir(src),
				"--include="+filepath.Join("..", "..", "engine", "src", "flutter", "impeller", "compiler", "shader_lib"))
		}
		msg, err := exec.Command(impellerc, args("--sksl", "--runtime-stage-metal")...).CombinedOutput()
		if err != nil {
			msg, err = exec.Command(impellerc, args("--runtime-stage-metal")...).CombinedOutput()
		}
		if err != nil {
			t.Fatalf("%s: %v\n%s", src, err, msg)
		}
		data, err := os.ReadFile(sl)
		if err != nil {
			t.Fatal(err)
		}
		out["shaders/"+filepath.Base(src)] = data
	}
	return out
}

// runFrag runs body as an async test body on a fresh loop, running posted
// events (Picture.toImage completes in one) until it's done.
func runFrag(t *testing.T, body func()) {
	l := async.NewLoop()
	defer async.SetCurrent(l)()
	f := async.AsyncVoid(body)
	done := false
	var err error
	async.OnComplete(f, func(async.Void) { done = true }, func(e error) { err = e; done = true })
	for i := 0; !done && i < 100000; i++ {
		l.FlushMicrotasks()
		if !done && l.RunPosted() == 0 {
			l.FlushMicrotasks()
			if !done {
				l.WaitPosted()
			}
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}

func fragPixels(shader Shader, w, h int, filter ImageFilter, color Color) []int {
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	p := NewPaint()
	p.SetColor(color)
	if shader != nil {
		p.SetShader(shader)
	}
	if filter != nil {
		p.SetImageFilter(filter)
	}
	c.DrawPaint(p)
	pic := rec.EndRecording()
	img := pic.ToImage(w, h).Await()
	data := img.ToByteData().Await()
	pic.Dispose()
	img.Dispose()
	out := make([]int, len(data))
	for i, b := range data {
		out[i] = int(b)
	}
	return out
}

func rectsImage(w, h int, rects []Rect, colors []Color) *Image {
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	for i, r := range rects {
		p := NewPaint()
		p.SetColor(colors[i])
		c.DrawRect(r, p)
	}
	pic := rec.EndRecording()
	img := pic.ToImageSync(w, h)
	pic.Dispose()
	return img
}

func fragBlueGreenImage() *Image {
	return rectsImage(10, 10, []Rect{NewRectFromLTWH(0, 0, 5, 10), NewRectFromLTWH(5, 0, 5, 10)},
		[]Color{NewColor(0xFF0000FF), NewColor(0xFF00FF00)})
}

func fragQuadrantImage() *Image {
	return rectsImage(4, 4, []Rect{NewRectFromLTWH(0, 0, 2, 2), NewRectFromLTWH(2, 0, 2, 2), NewRectFromLTWH(0, 2, 2, 2), NewRectFromLTWH(2, 2, 2, 2)},
		[]Color{NewColor(0xFFFF0000), NewColor(0xFF00FF00), NewColor(0xFF0000FF), NewColor(0x80FFFFFF)})
}

var fragHashRE = regexp.MustCompile(`#\d+`)

func fragmentShaderCases(t *testing.T) map[string]any {
	assets := compileDifftestShaders(t)
	prev := EngineAssetLoader
	t.Cleanup(func() { EngineAssetLoader = prev })
	EngineAssetLoader = func(key string) ([]byte, error) {
		if d, ok := assets[key]; ok {
			return d, nil
		}
		return nil, errors.New("not found")
	}
	out := map[string]any{}
	runFrag(t, func() {
		load := func(name string) *FragmentProgram {
			return FragmentProgramFromAsset("shaders/" + name + ".frag").Await()
		}

		reflectOut := map[string]any{}
		uniforms := []dart.MapEntry[string, []string]{
			{Key: "uniforms", Value: []string{"iFloatUniform", "iVec2Uniform", "iMat2Uniform", "iVec3Uniform", "iVec4Uniform",
				"iFloatArrayUniform", "iVec2ArrayUniform", "iVec3ArrayUniform", "iVec4ArrayUniform"}},
			{Key: "all_uniforms", Value: []string{"uFloat", "uVec2", "uVec3", "uVec4", "uMat2", "uMat3", "uMat4", "uFloatArray",
				"uVec2Array", "uVec3Array", "uVec4Array", "uMat2Array", "uMat3Array", "uMat4Array"}},
			{Key: "uniform_ordering", Value: []string{"a", "b", "c", "u_texture"}},
			{Key: "children_and_uniforms", Value: []string{"child1", "a", "child2", "b"}},
			{Key: "texture", Value: []string{"u_size", "u_texture"}},
			{Key: "mat3_array_uniform", Value: []string{"colors"}},
		}
		for _, e := range uniforms {
			s := load(e.Key).FragmentShader()
			for _, u := range e.Value {
				var row []any
				for i := -1; i < 20; i++ {
					var v any
					if r := errT(func() { v = s.GetUniformFloat(u, i).ShaderIndex() }); r != "no error" {
						v = r
					}
					row = append(row, v)
				}
				var v any
				if r := errT(func() { v = s.GetImageSampler(u).ShaderIndex() }); r != "no error" {
					v = r
				}
				reflectOut[e.Key+"."+u] = append(row, v)
			}
			reflectOut[e.Key+".nope"] = errT(func() { s.GetUniformFloat("nope") })
			reflectOut[e.Key+".nope.sampler"] = errT(func() { s.GetImageSampler("nope") })
		}
		out["reflect"] = reflectOut

		all := load("all_uniforms").FragmentShader()
		vec3 := load("vec3_uniform").FragmentShader()
		mat3a := load("mat3_array_uniform").FragmentShader()
		getters := []dart.MapEntry[string, func(*FragmentShader, string) any]{
			{Key: "vec2", Value: func(s *FragmentShader, n string) any { return s.GetUniformVec2(n) }},
			{Key: "vec3", Value: func(s *FragmentShader, n string) any { return s.GetUniformVec3(n) }},
			{Key: "vec4", Value: func(s *FragmentShader, n string) any { return s.GetUniformVec4(n) }},
			{Key: "mat2", Value: func(s *FragmentShader, n string) any { return s.GetUniformMat2(n) }},
			{Key: "mat3", Value: func(s *FragmentShader, n string) any { return s.GetUniformMat3(n) }},
			{Key: "mat4", Value: func(s *FragmentShader, n string) any { return s.GetUniformMat4(n) }},
			{Key: "floatArray", Value: func(s *FragmentShader, n string) any { return s.GetUniformFloatArray(n).Length() }},
			{Key: "vec2Array", Value: func(s *FragmentShader, n string) any { return s.GetUniformVec2Array(n).Length() }},
			{Key: "vec3Array", Value: func(s *FragmentShader, n string) any { return s.GetUniformVec3Array(n).Length() }},
			{Key: "vec4Array", Value: func(s *FragmentShader, n string) any { return s.GetUniformVec4Array(n).Length() }},
			{Key: "mat2Array", Value: func(s *FragmentShader, n string) any { return s.GetUniformMat2Array(n).Length() }},
			{Key: "mat3Array", Value: func(s *FragmentShader, n string) any { return s.GetUniformMat3Array(n).Length() }},
			{Key: "mat4Array", Value: func(s *FragmentShader, n string) any { return s.GetUniformMat4Array(n).Length() }},
		}
		typed := map[string]any{}
		typedCase := func(key string, f func() any) {
			var v any
			if r := errT(func() { v = f() }); r != "no error" {
				typed[key] = r
			} else if n, ok := v.(int); ok {
				typed[key] = n
			} else {
				typed[key] = "ok"
			}
		}
		for _, g := range getters {
			for _, u := range uniforms[1].Value {
				typedCase(g.Key+"."+u, func() any { return g.Value(all, u) })
			}
			typedCase(g.Key+".vec3.color_rgb", func() any { return g.Value(vec3, "color_rgb") })
			typedCase(g.Key+".mat3a.colors", func() any { return g.Value(mat3a, "colors") })
		}
		out["typed"] = typed

		arr := load("all_uniforms").FragmentShader()
		v3a := arr.GetUniformVec3Array("uVec3Array")
		out["array.index.10"] = errT(func() { v3a.At(10) })
		out["array.index.-1"] = errT(func() { v3a.At(-1) })
		out["array.length"] = v3a.Length()
		out["array.empty"] = errT(func() {
			vec3.GetUniformFloatArray("color_rgb")
			arr.GetUniformMat4Array("uMat4").At(1)
		})

		ord := load("uniform_ordering").FragmentShader()
		out["setFloat.3"] = errT(func() { ord.SetFloat(3, 1) })
		out["setFloat.-1"] = errT(func() { ord.SetFloat(-1, 1) })
		bg := fragBlueGreenImage()
		out["setImageSampler.1"] = errT(func() { ord.SetImageSampler(1, bg) })
		out["setImageSampler.-1"] = errT(func() { ord.SetImageSampler(-1, bg) })
		out["paint.missingSampler"] = errT(func() { ord.SetFloat(0, 1); NewPaint().SetShader(ord) })
		disposed := load("uniform_ordering").FragmentShader()
		disposed.Dispose()
		out["disposed.setFloat"] = errT(func() { disposed.SetFloat(0, 1) })
		out["disposed.setImageSampler"] = errT(func() { disposed.SetImageSampler(0, bg) })
		out["disposed.dispose"] = errT(disposed.Dispose)
		out["disposed.paint"] = errT(func() { NewPaint().SetShader(disposed) })
		out["disposed.debugDisposed"] = disposed.DebugDisposed()
		out["disposed.getUniformFloat"] = errT(func() { disposed.GetUniformFloat("b").ShaderIndex() })

		loadErr := func(key string) string {
			return errT(func() { FragmentProgramFromAsset(key).Await() })
		}
		out["fromAsset.missing"] = loadErr("shaders/missing.frag")
		out["fromAsset.invalid"] = loadErr("<invalid> #?")
		out["fromAsset.cached"] = load("texture") == load("texture")

		for _, name := range []string{"no_uniforms", "missing_size", "missing_texture", "filter_shader", "children_and_uniforms"} {
			s := load(name).FragmentShader()
			out["filter."+name] = errT(func() { NewImageFilterShader(s) })
		}
		fs := load("filter_shader").FragmentShader()
		filter := NewImageFilterShader(fs)
		filter2 := NewImageFilterShader(fs)
		low := NewImageFilterShader(fs, ImageFilterShaderOptions{FilterQuality: FilterQualityLow})
		out["filter.toString"] = fragHashRE.ReplaceAllString(filter.String(), "#H")
		out["filter.toString.low"] = fragHashRE.ReplaceAllString(low.String(), "#H")
		out["filter.toString.hashIsShader"] = strings.Contains(filter.String(), fmt.Sprintf("#%d,", dart.IdentityHashCode(fs)))
		out["filter.debugShortDescription"] = filter.DebugShortDescription()
		out["filter.eq"] = []any{
			ImageFilterEqual(filter, filter),
			ImageFilterEqual(filter, filter2),
			ImageFilterEqual(filter, low),
			ImageFilterEqual(low, NewImageFilterShader(fs, ImageFilterShaderOptions{FilterQuality: FilterQualityLow})),
			ImageFilterEqual(filter, NewImageFilterBlur()),
		}
		fs.SetFloat(0, 1)
		filter3 := NewImageFilterShader(fs)
		out["filter.eq.afterSet"] = []any{ImageFilterEqual(filter, filter3), ImageFilterEqual(filter2, filter),
			ImageFilterEqual(filter3, NewImageFilterShader(fs))}
		out["isShaderFilterSupported"] = ImageFilterIsShaderFilterSupported()

		red := NewColor(0xFFFF0000)
		u := load("uniforms").FragmentShader()
		for i, v := range []float64{0, 0.25, 0.75, 0, 0, 0, 1} {
			u.SetFloat(i, v)
		}
		out["px.uniforms"] = fragPixels(u, 4, 4, nil, red)
		u2 := load("uniforms").FragmentShader()
		u2.GetUniformFloat("iFloatUniform").Set(0.1)
		u2.GetUniformVec2("iVec2Uniform").Set(0.3, 0.55)
		u2.GetUniformMat2("iMat2Uniform").Set(0, 0, 0, 0.9)
		out["px.uniforms.slots"] = fragPixels(u2, 2, 2, nil, red)

		bar := load("all_uniforms").FragmentShader()
		for i := 0; i < 429; i++ {
			bar.SetFloat(i, float64((i*37)%256)/255.0)
		}
		out["px.all_uniforms"] = fragPixels(bar, 143, 1, nil, red)

		green := load("blue_green_sampler").FragmentShader()
		green.SetImageSampler(0, bg)
		out["px.blue_green_sampler"] = fragPixels(green, 4, 4, nil, red)

		tex := load("texture").FragmentShader()
		tex.SetFloat(0, 4)
		tex.SetFloat(1, 4)
		tex.SetImageSampler(0, fragQuadrantImage())
		out["px.texture"] = fragPixels(tex, 4, 4, nil, red)

		ordered := load("uniform_ordering").FragmentShader()
		ordered.SetFloat(0, 1)
		ordered.SetFloat(1, 2)
		ordered.SetFloat(2, 3)
		ordered.SetImageSampler(0, bg)
		out["px.uniform_ordering"] = fragPixels(ordered, 2, 2, nil, red)

		children := load("children_and_uniforms").FragmentShader()
		children.SetFloat(0, 0)
		children.SetFloat(1, 1)
		children.SetImageSampler(0, bg)
		children.SetImageSampler(1, bg)
		out["px.children_and_uniforms"] = fragPixels(children, 2, 2, nil, red)

		sdf := load("circle_sdf").FragmentShader()
		sdf.SetFloat(0, 8)
		sdf.SetFloat(1, 8)
		out["px.circle_sdf"] = fragPixels(sdf, 8, 8, nil, red)

		mat := load("uniform_mat2").FragmentShader()
		for i, v := range []float64{4, 8, 16, 32} {
			mat.SetFloat(i, v)
		}
		out["px.uniform_mat2"] = fragPixels(mat, 2, 2, nil, red)

		fn := load("functions").FragmentShader()
		fn.SetFloat(0, 1)
		out["px.functions"] = fragPixels(fn, 2, 2, nil, red)
		fn.SetFloat(0, 0)
		out["px.functions.reused"] = fragPixels(fn, 2, 2, nil, red)

		out["px.no_uniforms"] = fragPixels(load("no_uniforms").FragmentShader(), 2, 2, nil, red)

		swap := load("filter_shader").FragmentShader()
		out["px.filter.red"] = fragPixels(nil, 2, 2, NewImageFilterShader(swap), red)
		out["px.filter.color"] = fragPixels(nil, 2, 2, NewImageFilterShader(swap), NewColor(0x80336699))
	})
	return out
}

func TestFragmentShaderMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/fragment_shader.json")
	if err != nil {
		t.Skipf("no reference (%v); run flutter test --enable-impeller test/impeller/fragment_shader_test.dart", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	cases := fragmentShaderCases(t)
	// Normalize Go values to JSON's.
	enc, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(enc, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(ref) {
		t.Errorf("reference has %d cases, Go has %d", len(ref), len(got))
	}
	keys := make([]string, 0, len(ref))
	for k := range ref {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		want, g := ref[k], got[k]
		if wm, ok := want.(map[string]any); ok {
			gm, _ := g.(map[string]any)
			if len(gm) != len(wm) {
				t.Errorf("%s: %d entries, want %d", k, len(gm), len(wm))
			}
			for kk, wv := range wm {
				if !reflect.DeepEqual(gm[kk], wv) {
					t.Errorf("%s.%s:\n Go:      %v\n Flutter: %v", k, kk, gm[kk], wv)
				}
			}
			continue
		}
		if exact, ok := fragGPURoundingCases[k]; ok {
			// The float→unorm8 conversion of the GPU: Metal rounds
			// float32(0.3)*255 = 76.500003 to 77 and float32(0.9)*255 =
			// 229.499994 to 229 (correctly); SwiftShader's Vulkan gives 76
			// and 230. Go must match the correctly rounded bytes, and be
			// within 1 of Flutter's.
			if !reflect.DeepEqual(g, exact) {
				t.Errorf("%s: Go %v, want the correctly rounded %v", k, g, exact)
			}
			gl, _ := g.([]any)
			wl, _ := want.([]any)
			if len(gl) != len(wl) {
				t.Errorf("%s: %d bytes, Flutter %d", k, len(gl), len(wl))
				continue
			}
			for i := range gl {
				if d := gl[i].(float64) - wl[i].(float64); d > 1 || d < -1 {
					t.Errorf("%s[%d]: Go %v, Flutter %v", k, i, gl[i], wl[i])
				}
			}
			continue
		}
		if !reflect.DeepEqual(g, want) {
			t.Errorf("%s:\n Go:      %v\n Flutter: %v", k, g, want)
		}
	}
}

// fragGPURoundingCases are the pixel cases whose shader outputs land within
// 1e-5 of a rounding tie of the 8-bit conversion, where the GPUs differ,
// with the correctly rounded bytes (as Metal renders them).
var fragGPURoundingCases = map[string]any{
	// vec4(0.1, 0.3, 0.55, 0.9): 25.5000004, 76.500003, 140.250003, 229.499994.
	"px.uniforms.slots": []any{26.0, 77.0, 140.0, 229.0, 26.0, 77.0, 140.0, 229.0, 26.0, 77.0, 140.0, 229.0, 26.0, 77.0, 140.0, 229.0},
}
