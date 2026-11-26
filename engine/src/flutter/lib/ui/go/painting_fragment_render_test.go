package ui

// Port of fragment_shader_test.dart's rendering tests (Flutter 3.50.1).
//
// Skipped:
//   - 'ImageComparer tests' (FragmentShader renders sampler with filter
//     quality none/low/medium/high): Skia Gold goldens. ENGINE GAP besides:
//     the SDK samples fragment program samplers with linear filtering
//     whatever the filter quality.
//   - 'The ink_sparkle shader is accepted' is ported (it only renders).
//   - the !impellerEnabled op shader blocks (SkSL only).

import (
	"math"
	"math/rand"
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
)

const shaderImageDimension = 4

var (
	greenColor = NewColor(0xFF00FF00)
	blackColor = NewColor(0xFF000000)
)

func imageByteDataFromShader(t *testing.T, shader Shader, dim int) []byte {
	t.Helper()
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	p := NewPaint()
	p.SetShader(shader)
	c.DrawPaint(p)
	pic := rec.EndRecording()
	var data []byte
	runAsyncPosted(t, func() { data = pic.ToImage(dim, dim).Await().ToByteData().Await() })
	return data
}

// runAsyncPosted is runAsync for bodies awaiting posted events
// (Picture.toImage).
func runAsyncPosted(t *testing.T, body func()) {
	t.Helper()
	l := async.NewLoop()
	defer async.SetCurrent(l)()
	f := async.AsyncVoid(body)
	done := false
	var err error
	async.OnComplete(f, func(async.Void) { done = true }, func(e error) { err = e; done = true })
	for !done {
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

func expectShaderRendersColor(t *testing.T, shader Shader, color Color) {
	t.Helper()
	b := imageByteDataFromShader(t, shader, shaderImageDimension)
	for i := 0; i+3 < len(b); i += 4 {
		got := NewColorFromARGB(int(b[i+3]), int(b[i]), int(b[i+1]), int(b[i+2]))
		if got != color {
			t.Fatalf("pixel %d: %v, want %v", i/4, got, color)
		}
	}
}

func TestFragmentProgramsFromDifferentAssetPathsRender(t *testing.T) {
	for _, key := range []string{"no_uniforms.frag.iplr", "no_uniforms_alt.frag.iplr"} {
		shader := loadProgram(t, key).FragmentShader()
		p := NewPaint()
		p.SetShader(shader)
		rec := NewPictureRecorder()
		NewCanvas(rec).DrawRect(NewRectFromLTWH(0, 0, 10, 10), p)
		pic := rec.EndRecording()
		if pic == nil {
			t.Fatal("nil picture")
		}
		pic.Dispose()
		expectShaderRendersColor(t, shader, greenColor)
	}
}

func TestFragmentShaderUniformsRender(t *testing.T) {
	sh := func(key string) *FragmentShader { return loadProgram(t, key).FragmentShader() }
	r := func(c Color) float64 { return c.R }
	g := func(c Color) float64 { return c.G }
	b := func(c Color) float64 { return c.B }
	a := func(c Color) float64 { return c.A }

	s := sh("float_uniform.frag.iplr")
	c := NewColorFromARGB(255, 255, 0, 0)
	s.SetFloat(0, r(c))
	expectShaderRendersColor(t, s, c)
	c = NewColorFromARGB(255, 50, 0, 0)
	s.GetUniformFloat("color_r").Set(r(c))
	expectShaderRendersColor(t, s, c)

	s = sh("vec2_uniform.frag.iplr")
	c = NewColorFromARGB(255, 255, 255, 0)
	s.SetFloat(0, r(c))
	s.SetFloat(1, g(c))
	expectShaderRendersColor(t, s, c)
	c = NewColorFromARGB(255, 50, 50, 0)
	s.GetUniformVec2("color_rg").Set(r(c), g(c))
	expectShaderRendersColor(t, s, c)

	s = sh("vec3_uniform.frag.iplr")
	c = NewColorFromARGB(255, 67, 42, 12)
	s.SetFloat(0, r(c))
	s.SetFloat(1, g(c))
	s.SetFloat(2, b(c))
	expectShaderRendersColor(t, s, c)
	c = NewColorFromARGB(255, 12, 42, 67)
	s.GetUniformVec3("color_rgb").Set(r(c), g(c), b(c))
	expectShaderRendersColor(t, s, c)

	s = sh("vec4_uniform.frag.iplr")
	c = NewColorFromARGB(255, 67, 42, 12)
	for i, v := range []float64{r(c), g(c), b(c), a(c)} {
		s.SetFloat(i, v)
	}
	expectShaderRendersColor(t, s, c)
	c = NewColorFromARGB(255, 12, 42, 67)
	s.GetUniformVec4("color_rgba").Set(r(c), g(c), b(c), a(c))
	expectShaderRendersColor(t, s, c)

	s = sh("mat2_uniform.frag.iplr")
	c = NewColorFromARGB(255, 67, 42, 12)
	for i, v := range []float64{r(c), g(c), b(c), a(c)} {
		s.SetFloat(i, v)
	}
	expectShaderRendersColor(t, s, c)
	c = NewColorFromARGB(255, 12, 42, 67)
	s.GetUniformMat2("color_rgba").Set(r(c), g(c), b(c), a(c))
	expectShaderRendersColor(t, s, c)
}

// 'all uniforms': random colors (Go's generator; Dart's Random(1337)
// sequence isn't needed for this check) set through setFloat and the
// typed getters render as a barcode.
func TestFragmentShaderAllUniformsBarcode(t *testing.T) {
	program := loadProgram(t, "all_uniforms.frag.iplr")
	rnd := rand.New(rand.NewSource(1337))
	n := func() int { return rnd.Intn(255) }
	var colors []Color
	add := func(k int, f func() Color) {
		for range k {
			colors = append(colors, f())
		}
	}
	rgb1 := func() Color { return NewColorFromARGB(255, n(), 0, 0) }
	rgb2 := func() Color { return NewColorFromARGB(255, n(), n(), 0) }
	rgb3 := func() Color { return NewColorFromARGB(255, n(), n(), n()) }
	rgba := func() Color { return NewColorFromARGB(n(), n(), n(), n()) }
	add(1, rgb1)
	add(1, rgb2)
	add(1, rgb3)
	add(1, rgba)
	add(2, rgb2)
	add(3, rgb3)
	add(4, rgba)
	add(10, rgb1)
	add(10, rgb2)
	add(10, rgb3)
	add(10, rgba)
	add(20, rgb2)
	add(30, rgb3)
	add(40, rgba)
	// Component counts per color, in uniform order.
	var comps []int
	push := func(k, w int) {
		for range k {
			comps = append(comps, w)
		}
	}
	push(1, 1)
	push(1, 2)
	push(1, 3)
	push(1, 4)
	push(2, 2)
	push(3, 3)
	push(4, 4)
	push(10, 1)
	push(10, 2)
	push(10, 3)
	push(10, 4)
	push(20, 2)
	push(30, 3)
	push(40, 4)

	shader := program.FragmentShader()
	idx := 0
	for i, c := range colors {
		v := []float64{c.R, c.G, c.B, c.A}
		for j := 0; j < comps[i]; j++ {
			shader.SetFloat(idx, v[j])
			idx++
		}
	}
	expectBarcode(t, shader, colors)
}

func expectBarcode(t *testing.T, shader Shader, colors []Color) {
	t.Helper()
	b := imageByteDataFromShader(t, shader, len(colors))
	for i, want := range colors {
		got := NewColorFromARGB(int(b[i*4+3]), int(b[i*4]), int(b[i*4+1]), int(b[i*4+2]))
		for _, d := range []float64{got.R - want.R, got.G - want.G, got.B - want.B, got.A - want.A} {
			if math.Abs(d) > 0.06 {
				t.Fatalf("color %d: %v, want %v", i, got, want)
			}
		}
	}
}

func TestReusedFragmentShaderSimpleShaderRendersCorrectly(t *testing.T) {
	shader := loadProgram(t, "functions.frag.iplr").FragmentShader()
	shader.SetFloat(0, 1)
	expectShaderRendersColor(t, shader, greenColor)
	shader.SetFloat(0, 0)
	expectShaderRendersColor(t, shader, blackColor)
	shader.Dispose()
}

func TestFragmentShaderBlueGreenImageRendersGreen(t *testing.T) {
	shader := loadProgram(t, "blue_green_sampler.frag.iplr").FragmentShader()
	img := testBlueGreenImage()
	shader.SetImageSampler(0, img)
	expectShaderRendersColor(t, shader, greenColor)
	shader.Dispose()
	img.Dispose()
}

func TestFragmentShaderBlueGreenImageRendersGreenGPUImage(t *testing.T) {
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	blue, green := NewPaint(), NewPaint()
	blue.SetColor(NewColor(0xFF0000FF))
	green.SetColor(NewColor(0xFF00FF00))
	c.DrawRect(NewRectFromLTWH(0, 0, 5, 10), blue)
	c.DrawRect(NewRectFromLTWH(5, 0, 5, 10), green)
	pic := rec.EndRecording()
	img := pic.ToImageSync(10, 10)
	pic.Dispose()
	shader := loadProgram(t, "blue_green_sampler.frag.iplr").FragmentShader()
	shader.SetImageSampler(0, img)
	expectShaderRendersColor(t, shader, greenColor)
	shader.Dispose()
	img.Dispose()
}

func TestFragmentShaderUniformsAreSortedCorrectly(t *testing.T) {
	shader := loadProgram(t, "uniforms_sorted.frag.iplr").FragmentShader()
	for i := 0; i < 32; i++ {
		shader.SetFloat(i, float64(i))
	}
	expectShaderRendersColor(t, shader, greenColor)
	shader.Dispose()
}

func TestFragmentShaderUniformsWithInterleavedTexturesAreSorted(t *testing.T) {
	shader := loadProgram(t, "uniform_ordering.frag.iplr").FragmentShader()
	shader.SetFloat(0, 1)
	shader.SetFloat(1, 2)
	shader.SetFloat(2, 3)
	shader.SetImageSampler(0, testBlueGreenImage())
	expectShaderRendersColor(t, shader, greenColor)
	shader.Dispose()
}

func TestFragmentShaderUserDefinedFunctionsDoNotRedefineBuiltins(t *testing.T) {
	shader := loadProgram(t, "no_builtin_redefinition.frag.iplr").FragmentShader()
	shader.SetFloat(0, 1)
	expectShaderRendersColor(t, shader, greenColor)
	shader.Dispose()
}

func TestFragmentShaderFromAssetAcceptsAShaderWithNoUniforms(t *testing.T) {
	shader := loadProgram(t, "no_uniforms.frag.iplr").FragmentShader()
	expectShaderRendersColor(t, shader, greenColor)
	shader.Dispose()
}

func TestFragmentShaderSimpleShaderRendersCorrectly(t *testing.T) {
	shader := loadProgram(t, "functions.frag.iplr").FragmentShader()
	shader.SetFloat(0, 1)
	expectShaderRendersColor(t, shader, greenColor)
	shader.Dispose()
}

func TestFragmentShaderWithUniformsRendersCorrectly(t *testing.T) {
	shader := loadProgram(t, "uniforms.frag.iplr").FragmentShader()
	for i, v := range []float64{0, 0.25, 0.75, 0, 0, 0, 1} {
		shader.SetFloat(i, v)
	}
	b := imageByteDataFromShader(t, shader, 100)
	const epsilon = 0.5 / 255.0
	for i, want := range []float64{0, 0.25, 0.75, 1} {
		if got := float64(b[i]) / 255; math.Abs(got-want) > epsilon {
			t.Errorf("component %d: %v, want %v", i, got, want)
		}
	}
	shader.Dispose()
}

func TestFragmentShaderShaderWithMat2UniformRendersCorrectly(t *testing.T) {
	shader := loadProgram(t, "uniform_mat2.frag.iplr").FragmentShader()
	for i, v := range []float64{4, 8, 16, 32} {
		shader.SetFloat(i, v)
	}
	expectShaderRendersColor(t, shader, greenColor)
	shader.Dispose()
}

func TestImageFilterShaderCanBeAppliedToCanvasOperations(t *testing.T) {
	shader := loadProgram(t, "filter_shader.frag.iplr").FragmentShader()
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	p := NewPaint()
	p.SetColor(NewColor(0xFFFF0000))
	p.SetImageFilter(NewImageFilterShader(shader))
	c.DrawPaint(p)
	pic := rec.EndRecording()
	var data []byte
	runAsyncPosted(t, func() { data = pic.ToImage(1, 1).Await().ToByteData().Await() })
	if len(data) != 4 {
		t.Fatalf("%d bytes", len(data))
	}
	// filter_shader.frag swaps red and blue: red becomes blue.
	if got := NewColorFromARGB(int(data[3]), int(data[0]), int(data[1]), int(data[2])); got != NewColor(0xFF0000FF) {
		t.Fatalf("%v", got)
	}
}

func TestFragmentShaderTheInkSparkleShaderIsAccepted(t *testing.T) {
	shader := loadProgram(t, "ink_sparkle.frag.iplr").FragmentShader()
	imageByteDataFromShader(t, shader, 100)
	shader.Dispose()
}

// A shader with an unset sampler draws transparent black, as Impeller
// draws a runtime effect with a missing sampler.
func TestFragmentShaderMissingSamplerDrawsTransparent(t *testing.T) {
	shader := loadProgram(t, "blue_green_sampler.frag.iplr").FragmentShader()
	if src := shader.impellerColorSource(); src == nil {
		t.Fatal("no color source")
	}
}
