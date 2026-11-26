package ui

// Port of engine/src/flutter/testing/dart/fragment_shader_test.dart
// (Flutter 3.50.1): the tests that don't read rendered pixels. The pixel
// tests are in painting_fragment_render_test.go.
//
// Skipped:
//   - the !impellerEnabled block (supported_glsl_op_shaders /
//     supported_op_shaders rendering through SkSL): flutter-go always
//     renders with Impeller, as the Dart test skips it under Impeller.
//   - nothing else: hot reload (_reinitializeShader) has no test upstream.
//
// Adapted: shaders are compiled with the SDK's impellerc into a temporary
// directory and served through EngineAssetLoader
// (painting_fragment_assets_test.go); the reflection is the Metal stage's
// (the engine's backend), where Flutter's tester reads the Vulkan stage.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// loadProgram is `await FragmentProgram.fromAsset(key)`.
func loadProgram(t *testing.T, key string) *FragmentProgram {
	t.Helper()
	loadFragmentAssets(t)
	var p *FragmentProgram
	runAsync(t, func() { p = FragmentProgramFromAsset(key).Await() })
	return p
}

// loadProgramError is the error `await FragmentProgram.fromAsset(key)`
// throws (nil when it doesn't).
func loadProgramError(t *testing.T, key string) (err error) {
	t.Helper()
	loadFragmentAssets(t)
	runAsync(t, func() {
		r := panics(func() { FragmentProgramFromAsset(key).Await() })
		if r != nil {
			err = r.(error)
		}
	})
	return err
}

func errorText(r any) string {
	switch v := r.(type) {
	case nil:
		return "<no error>"
	case string:
		return v
	case error:
		return v.Error()
	}
	return dart.ToString(r)
}

func expectPanicContains(t *testing.T, name string, f func(), want string) any {
	t.Helper()
	r := panics(f)
	if r == nil || !strings.Contains(errorText(r), want) {
		t.Errorf("%s: got %v, want an error containing %q", name, errorText(r), want)
	}
	return r
}

// testImage is a w×h image of opaque blue (left half) and green (right
// half) pixels, as _createBlueGreenImage makes.
func testBlueGreenImage() *Image {
	const length = 10
	pixels := make([]byte, length*length*4)
	i := 0
	for y := 0; y < length; y++ {
		for x := 0; x < length; x++ {
			if float64(x) < length/2.0 {
				pixels[i+2] = 0xFF
			} else {
				pixels[i+1] = 0xFF
			}
			pixels[i+3] = 0xFF
			i += 4
		}
	}
	return newImageFromRGBA8(length, length, pixels)
}

func TestImpellercProducesReasonableJSONEncodedIPLRFiles(t *testing.T) {
	loadFragmentAssets(t)
	dir := t.TempDir()
	src := filepath.Join(repoRoot, "engine", "src", "flutter", "impeller", "fixtures", "ink_sparkle.frag")
	data, err := compileShader(src, dir, []string{"--sksl", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	// The key order of the JSON object.
	dec := json.NewDecoder(strings.NewReader(string(data)))
	var keys []string
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		t.Fatalf("not a JSON object: %v %v", tok, err)
	}
	for dec.More() {
		tok, _ := dec.Token()
		keys = append(keys, tok.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(keys, []string{"format_version", "sksl"}) {
		t.Fatalf("keys = %v", keys)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	sksl, ok := raw["sksl"].(map[string]any)
	if !ok {
		t.Fatal("sksl is not a map")
	}
	uniforms, ok := sksl["uniforms"].([]any)
	if !ok {
		t.Fatal("uniforms is not a list")
	}
	u, ok := uniforms[0].(map[string]any)
	if !ok {
		t.Fatal("uniform is not a map")
	}
	if loc, ok := u["location"].(float64); !ok || loc != float64(int(loc)) {
		t.Fatalf("location = %v", u["location"])
	}
	_ = os.Remove(dir)
}

func TestFragmentProgramObjectsAreCached(t *testing.T) {
	a := loadProgram(t, "blue_green_sampler.frag.iplr")
	b := loadProgram(t, "blue_green_sampler.frag.iplr")
	if a != b {
		t.Fatal("not identical")
	}
}

func TestFragmentProgramsFromDifferentAssetPathsDoNotCollide(t *testing.T) {
	a := loadProgram(t, "no_uniforms.frag.iplr")
	b := loadProgram(t, "no_uniforms_alt.frag.iplr")
	if a == b {
		t.Fatal("identical")
	}
	if a.FragmentShader() == nil || b.FragmentShader() == nil {
		t.Fatal("nil shader")
	}
	// The drawing half is in painting_fragment_render_test.go.
}

func TestFragmentProgramUniformInfo(t *testing.T) {
	shader := loadProgram(t, "uniforms.frag.iplr").FragmentShader()
	slots := []*UniformFloatSlot{
		shader.GetUniformFloat("iFloatUniform"),
		shader.GetUniformFloat("iVec2Uniform", 0),
		shader.GetUniformFloat("iVec2Uniform", 1),
		shader.GetUniformFloat("iMat2Uniform", 0),
		shader.GetUniformFloat("iMat2Uniform", 1),
		shader.GetUniformFloat("iMat2Uniform", 2),
		shader.GetUniformFloat("iMat2Uniform", 3),
	}
	for i, s := range slots {
		if s.ShaderIndex() != i {
			t.Errorf("slot %d: shaderIndex %d", i, s.ShaderIndex())
		}
	}
	if slots[2].Name != "iVec2Uniform" || slots[2].Index != 1 {
		t.Errorf("slot 2: %q %d", slots[2].Name, slots[2].Index)
	}
	// Later uniforms (offsets add up the Dart sizes, vec3 without padding).
	for name, want := range map[string]int{"iVec3Uniform": 7, "iVec4Uniform": 10, "iFloatArrayUniform": 14,
		"iVec2ArrayUniform": 24, "iVec3ArrayUniform": 30, "iVec4ArrayUniform": 39} {
		if got := shader.GetUniformFloat(name).ShaderIndex(); got != want {
			t.Errorf("%s: %d, want %d", name, got, want)
		}
	}
	if got := shader.program.uniformFloatCount; got != 51 {
		t.Errorf("uniformFloatCount %d", got)
	}
}

// The error cases of the 'FragmentShader uniforms' group (the rendering
// cases are in painting_fragment_render_test.go).
func TestFragmentShaderUniformErrors(t *testing.T) {
	sh := func(key string) *FragmentShader { return loadProgram(t, key).FragmentShader() }
	float := sh("float_uniform.frag.iplr")
	r := expectPanicContains(t, "getUniformFloat offset overflow", func() { float.GetUniformFloat("color_r", 2) },
		"Index `2` out of bounds for `color_r`.")
	if _, ok := r.(dart.IndexError); !ok {
		t.Errorf("overflow: %T, want IndexError", r)
	}
	if got := errorText(r); got != "RangeError (index): Index `2` out of bounds for `color_r`.: index should be less than 1: 2" {
		t.Errorf("overflow text %q", got)
	}
	r = expectPanicContains(t, "getUniformFloat offset underflow", func() { float.GetUniformFloat("color_r", -1) },
		"Index `-1` out of bounds for `color_r`.")
	if _, ok := r.(dart.IndexError); !ok {
		t.Errorf("underflow: %T, want IndexError", r)
	}
	vec3 := sh("vec3_uniform.frag.iplr")
	cases := []struct {
		name string
		f    func()
		want string
	}{
		{"vec2 wrong datatype", func() { vec3.GetUniformVec2("color_rgb") }, "`color_rgb` has size 3, not size 2."},
		{"vec3 wrong datatype", func() { sh("vec2_uniform.frag.iplr").GetUniformVec3("color_rg") }, "`color_rg` has size 2, not size 3."},
		{"vec4 wrong datatype", func() { vec3.GetUniformVec4("color_rgb") }, "`color_rgb` has size 3, not size 4."},
		{"mat2 wrong datatype", func() { vec3.GetUniformMat2("color_rgb") }, "`color_rgb` has size 3, not size 4."},
		{"mat3 wrong datatype", func() { vec3.GetUniformMat3("color_rgb") }, "`color_rgb` has size 3, not size 9."},
		{"mat4 wrong datatype", func() { vec3.GetUniformMat4("color_rgb") }, "`color_rgb` has size 3, not size 16."},
		{"vec2 array wrong datatype", func() { vec3.GetUniformVec2Array("color_rgb") }, `Uniform size (3) for "color_rgb" is not a multiple of 2.`},
		{"vec3 array wrong datatype", func() { sh("vec2_uniform.frag.iplr").GetUniformVec3Array("color_rg") }, `Uniform size (2) for "color_rg" is not a multiple of 3.`},
		{"vec4 array wrong datatype", func() { vec3.GetUniformVec4Array("color_rgb") }, `Uniform size (3) for "color_rgb" is not a multiple of 4.`},
		{"mat2 array wrong datatype", func() { vec3.GetUniformMat2Array("color_rgb") }, `Uniform size (3) for "color_rgb" is not a multiple of 4.`},
		{"mat4 array wrong datatype", func() { vec3.GetUniformMat4Array("color_rgb") }, `Uniform size (3) for "color_rgb" is not a multiple of 16.`},
		{"mat3 array wrong datatype", func() { vec3.GetUniformMat3Array("color_rgb") }, `Uniform size (3) for "color_rgb" is not a multiple of 9.`},
		{"no uniform", func() { vec3.GetUniformFloat("nope") }, `No uniform named "nope".`},
	}
	for _, c := range cases {
		r := expectPanicContains(t, c.name, c.f, c.want)
		if e, ok := r.(dart.ArgumentError); !ok || !strings.HasSuffix(e.Message, c.want) {
			t.Errorf("%s: %#v, want ArgumentError", c.name, r)
		}
	}
	if got := errorText(panics(func() { vec3.GetUniformVec2("color_rgb") })); got != "Invalid argument(s): Uniform `color_rgb` has size 3, not size 2." {
		t.Errorf("text %q", got)
	}
}

// Slot indices of every getter, on the shaders the rendering tests use.
func TestFragmentShaderSlotIndices(t *testing.T) {
	all := loadProgram(t, "all_uniforms.frag.iplr").FragmentShader()
	idx := func(fs ...*UniformFloatSlot) []int {
		var out []int
		for _, f := range fs {
			out = append(out, f.ShaderIndex())
		}
		return out
	}
	check := func(name string, got []int, from int) {
		t.Helper()
		for i, g := range got {
			if g != from+i {
				t.Errorf("%s: %v, want from %d", name, got, from)
				return
			}
		}
	}
	v2 := all.GetUniformVec2("uVec2")
	check("vec2", idx(v2.xSlot, v2.ySlot), 1)
	v3 := all.GetUniformVec3("uVec3")
	check("vec3", idx(v3.xSlot, v3.ySlot, v3.zSlot), 3)
	v4 := all.GetUniformVec4("uVec4")
	check("vec4", idx(v4.xSlot, v4.ySlot, v4.zSlot, v4.wSlot), 6)
	check("mat2", idx(all.GetUniformMat2("uMat2").m[:]...), 10)
	check("mat3", idx(all.GetUniformMat3("uMat3").m[:]...), 14)
	check("mat4", idx(all.GetUniformMat4("uMat4").m[:]...), 23)
	fa := all.GetUniformFloatArray("uFloatArray")
	if fa.Length() != 10 || fa.At(9).ShaderIndex() != 48 || fa.At(0).Index != 0 {
		t.Errorf("float array: %d %d", fa.Length(), fa.At(9).ShaderIndex())
	}
	v2a := all.GetUniformVec2Array("uVec2Array")
	check("vec2 array[3]", idx(v2a.At(3).xSlot, v2a.At(3).ySlot), 49+6)
	v3a := all.GetUniformVec3Array("uVec3Array")
	check("vec3 array[9]", idx(v3a.At(9).xSlot, v3a.At(9).ySlot, v3a.At(9).zSlot), 69+27)
	v4a := all.GetUniformVec4Array("uVec4Array")
	check("vec4 array[1]", idx(v4a.At(1).xSlot, v4a.At(1).ySlot, v4a.At(1).zSlot, v4a.At(1).wSlot), 99+4)
	m2a := all.GetUniformMat2Array("uMat2Array")
	check("mat2 array[2]", idx(m2a.At(2).m[:]...), 139+8)
	m3a := all.GetUniformMat3Array("uMat3Array")
	check("mat3 array[1]", idx(m3a.At(1).m[:]...), 179+9)
	m4a := all.GetUniformMat4Array("uMat4Array")
	if m4a.Length() != 10 {
		t.Errorf("mat4 array length %d", m4a.Length())
	}
	check("mat4 array[9]", idx(m4a.At(9).m[:]...), 269+144)
	if got := all.program.uniformFloatCount; got != 429 {
		t.Errorf("uniformFloatCount %d", got)
	}
	// Elements' slots: index j counts from the element's start, name is
	// the array's.
	if s := m4a.At(1).m[5]; s.Index != 5 || s.Name != "uMat4Array" {
		t.Errorf("element slot %q %d", s.Name, s.Index)
	}
	r := expectPanicContains(t, "array index", func() { fa.At(10) }, "RangeError (length): Invalid value: Not in inclusive range 0..9: 10")
	if _, ok := r.(dart.RangeError); !ok {
		t.Errorf("%T", r)
	}
	expectPanicContains(t, "array negative index", func() { fa.At(-1) }, "RangeError (length): Invalid value: Not in inclusive range 0..9: -1")
	expectPanicContains(t, "one element", func() { all.GetUniformMat4Array("uMat4").At(1) }, "RangeError (length): Invalid value: Only valid value is 0: 1")

	// Setting through slots writes the float at their shader index.
	v3a.At(1).Set(1, 2, 3)
	if got := all.floats[72:75]; !reflect.DeepEqual(got, []float32{1, 2, 3}) {
		t.Errorf("vec3 array set: %v", got)
	}
	m2a.At(0).Set(5, 6, 7, 8)
	if got := all.floats[139:143]; !reflect.DeepEqual(got, []float32{5, 6, 7, 8}) {
		t.Errorf("mat2 array set: %v", got)
	}
	all.GetUniformMat3("uMat3").Set(1, 2, 3, 4, 5, 6, 7, 8, 9)
	if got := all.floats[14:23]; !reflect.DeepEqual(got, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9}) {
		t.Errorf("mat3 set: %v", got)
	}
	all.GetUniformMat4("uMat4").Set(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16)
	if got := all.floats[23:39]; !reflect.DeepEqual(got, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}) {
		t.Errorf("mat4 set: %v", got)
	}
	all.GetUniformVec4("uVec4").Set(0.1, 0.2, 0.3, 0.4)
	if got := all.floats[6:10]; !reflect.DeepEqual(got, []float32{0.1, 0.2, 0.3, 0.4}) {
		t.Errorf("vec4 set: %v", got)
	}
	all.GetUniformFloat("uFloat").Set(1e40) // float32 rounding: +Inf
	if got := all.floats[0]; !(got > 3.4e38) {
		t.Errorf("float32 overflow: %v", got)
	}
}

func TestFragmentProgramGetImageSampler(t *testing.T) {
	shader := loadProgram(t, "uniform_ordering.frag.iplr").FragmentShader()
	img := testBlueGreenImage()
	slot := shader.GetImageSampler("u_texture")
	slot.Set(img)
	if slot.ShaderIndex() != 0 || slot.Name != "u_texture" {
		t.Fatalf("shaderIndex %d", slot.ShaderIndex())
	}
	if shader.samplers[0] == nil || shader.samplers[0].image != img {
		t.Fatal("sampler not set")
	}
	// The sampler's size follows the float uniforms.
	if got := shader.uniformData; !reflect.DeepEqual(got, []float32{0, 0, 0, 10, 10}) {
		t.Fatalf("uniform data %v", got)
	}
	// Float slots skip the sampler.
	if got := shader.GetUniformFloat("b").ShaderIndex(); got != 1 {
		t.Fatalf("b: %d", got)
	}
	// Sampler indices count samplers only, in the stage's order.
	two := loadProgram(t, "children_and_uniforms.frag.iplr").FragmentShader()
	if got := two.GetImageSampler("child2").ShaderIndex(); got != 1 {
		t.Fatalf("child2: %d", got)
	}
}

func TestFragmentProgramGetImageSamplerUnknown(t *testing.T) {
	shader := loadProgram(t, "uniform_ordering.frag.iplr").FragmentShader()
	expectPanicContains(t, "unknown", func() { shader.GetImageSampler("unknown") }, `No uniform named "unknown".`)
}

func TestFragmentProgramGetImageSamplerWrongType(t *testing.T) {
	shader := loadProgram(t, "uniform_ordering.frag.iplr").FragmentShader()
	r := expectPanicContains(t, "wrong type", func() { shader.GetImageSampler("b") }, `Uniform "b" is not an image sampler.`)
	if errorText(r) != `Invalid argument(s): Uniform "b" is not an image sampler.` {
		t.Errorf("%q", errorText(r))
	}
}

func TestFragmentShaderSetSamplerThrowsWithOutOfBoundsIndex(t *testing.T) {
	shader := loadProgram(t, "blue_green_sampler.frag.iplr").FragmentShader()
	img := testBlueGreenImage()
	defer shader.Dispose()
	r := panics(func() { shader.SetImageSampler(1, img) })
	if s, ok := r.(string); !ok || !strings.Contains(s, "Sampler index out of bounds") {
		t.Fatalf("%#v", r)
	}
	if r := panics(func() { shader.SetImageSampler(-1, img) }); r != "Sampler index out of bounds" {
		t.Fatalf("negative: %#v", r)
	}
}

func TestFragmentShaderWithSamplerAssertsIfSamplerIsMissingWhenAssignedToPaint(t *testing.T) {
	shader := loadProgram(t, "blue_green_sampler.frag.iplr").FragmentShader()
	defer shader.Dispose()
	r := expectPanicContains(t, "missing sampler", func() { NewPaint().SetShader(shader) },
		"Invalid FragmentShader blue_green_sampler.frag.iplr")
	if errorText(r) != "Exception: Invalid FragmentShader blue_green_sampler.frag.iplr: missing sampler" {
		t.Errorf("%q", errorText(r))
	}
	// With the sampler set, the shader is accepted.
	shader.SetImageSampler(0, testBlueGreenImage())
	NewPaint().SetShader(shader)
}

func TestDisposedFragmentShaderOnPaint(t *testing.T) {
	shader := loadProgram(t, "blue_green_sampler.frag.iplr").FragmentShader()
	shader.SetImageSampler(0, testBlueGreenImage())
	shader.Dispose()
	r := expectPanicContains(t, "disposed on paint", func() { NewPaint().SetShader(shader) }, "Attempted to set a disposed shader")
	if _, ok := r.(*dart.AssertionError); !ok {
		t.Errorf("%T", r)
	}
	if !shader.DebugDisposed() {
		t.Error("debugDisposed")
	}
}

func TestDisposedFragmentShaderSetFloat(t *testing.T) {
	shader := loadProgram(t, "uniforms.frag.iplr").FragmentShader()
	shader.SetFloat(0, 0)
	shader.Dispose()
	r := expectPanicContains(t, "setFloat", func() { shader.SetFloat(0, 0) }, "Tried to accesss uniforms on a disposed Shader")
	if a, ok := r.(*dart.AssertionError); !ok || a.AssertionMessage() != "Tried to accesss uniforms on a disposed Shader: Instance of 'FragmentShader'" {
		t.Errorf("%#v", r)
	}
	// Slots go through setFloat.
	expectPanicContains(t, "slot", func() { shader.GetUniformFloat("iFloatUniform").Set(1) }, "disposed Shader")
}

func TestDisposedFragmentShaderSetImageSampler(t *testing.T) {
	shader := loadProgram(t, "blue_green_sampler.frag.iplr").FragmentShader()
	img := testBlueGreenImage()
	shader.SetImageSampler(0, img)
	shader.Dispose()
	r := expectPanicContains(t, "setImageSampler", func() { shader.SetImageSampler(0, img) }, "Tried to access uniforms on a disposed Shader")
	if a, ok := r.(*dart.AssertionError); !ok || a.AssertionMessage() != "Tried to access uniforms on a disposed Shader: Instance of 'FragmentShader'" {
		t.Errorf("%#v", r)
	}
}

func TestDisposedFragmentShaderDispose(t *testing.T) {
	shader := loadProgram(t, "uniforms.frag.iplr").FragmentShader()
	shader.SetFloat(0, 0)
	shader.Dispose()
	expectPanicContains(t, "dispose", shader.Dispose, "Shader cannot be disposed more than once")
	// The native shader is gone: no uniforms, no samplers, no color source.
	if len(shader.floats) != 0 || shader.uniformData != nil || shader.nativeProgram != nil || shader.impellerColorSource() != nil {
		t.Error("native state kept")
	}
	if !shader.validateSamplers() {
		t.Error("validateSamplers after dispose")
	}
}

func TestFragmentShaderSetFloatIndexErrors(t *testing.T) {
	shader := loadProgram(t, "uniform_ordering.frag.iplr").FragmentShader()
	// _floats is the float uniforms only (3), not the sampler sizes.
	r := expectPanicContains(t, "past floats", func() { shader.SetFloat(3, 1) }, "RangeError (index): Index out of range: index should be less than 3: 3")
	if _, ok := r.(dart.IndexError); !ok {
		t.Errorf("%T", r)
	}
	expectPanicContains(t, "negative", func() { shader.SetFloat(-1, 1) }, "index must not be negative: -1")
	shader.SetFloat(2, 0.1)
	if shader.floats[2] != float32(0.1) || shader.uniformData[2] != float32(0.1) {
		t.Error("setFloat")
	}
}

func TestFromAssetThrowsAnExceptionOnInvalidAssetKey(t *testing.T) {
	err := loadProgramError(t, "<invalid>")
	var e dart.Exception
	if !errors.As(err, &e) {
		t.Fatalf("%v", err)
	}
	if got := err.Error(); got != "Exception: Asset '%3Cinvalid%3E' not found" {
		t.Fatalf("%q", got)
	}
	if _, ok := shaderRegistry["%3Cinvalid%3E"]; ok {
		t.Fatal("failed program registered")
	}
}

func TestFromAssetThrowsAnExceptionOnInvalidData(t *testing.T) {
	err := loadProgramError(t, "DashInNooglerHat.jpg")
	if err == nil || err.Error() != "Exception: Asset 'DashInNooglerHat.jpg' manifest could not be decoded: INVALID_ARGUMENT: Payload does not have valid identifier." {
		t.Fatalf("%v", err)
	}
}

func TestFragmentProgramDebugName(t *testing.T) {
	shader := loadProgram(t, "blue_green_sampler.frag.iplr").FragmentShader()
	if got := shader.debugName(); got != "blue_green_sampler.frag.iplr" {
		t.Fatalf("%q", got)
	}
}

func TestImageFilterShaderErrorsIfShaderDoesNotHaveCorrectUniformLayout(t *testing.T) {
	cases := []struct {
		file                     string
		floatError, samplerError bool
	}{
		{"no_uniforms.frag.iplr", true, true},
		{"missing_size.frag.iplr", true, false},
		{"missing_texture.frag.iplr", false, true},
	}
	for _, c := range cases {
		shader := loadProgram(t, c.file).FragmentShader()
		r := panics(func() { NewImageFilterShader(shader) })
		e, ok := r.(dart.StateError)
		if !ok {
			t.Fatalf("%s: %#v", c.file, r)
		}
		want := "Bad state: ImageFilter.shader requires that the first uniform is a vec2 and at least one sampler uniform is present.\n"
		if c.floatError {
			want += "The shader has fewer than two float uniforms.\n"
		}
		if c.samplerError {
			want += "The shader is missing a sampler uniform.\n"
		}
		if e.Error() != want {
			t.Errorf("%s: %q", c.file, e.Error())
		}
	}
	// One float uniform (no fixture has one with a sampler): too few.
	one := &FragmentShader{floats: make([]float32, 1), samplers: make([]*shaderSampler, 1)}
	expectPanicContains(t, "one float", func() { NewImageFilterShader(one) }, "The shader has fewer than two float uniforms.")
	// Samplers after the first must be set.
	two := loadProgram(t, "children_and_uniforms.frag.iplr").FragmentShader()
	expectPanicContains(t, "second sampler", func() { NewImageFilterShader(two) }, "The shader is missing a sampler uniform.")
	two.SetImageSampler(1, testBlueGreenImage())
	NewImageFilterShader(two)
}

func TestImageFilterShaderEqualityChecksConsiderUniformValues(t *testing.T) {
	shader := loadProgram(t, "filter_shader.frag.iplr").FragmentShader()
	filter := NewImageFilterShader(shader)
	if !ImageFilterEqual(filter, filter) {
		t.Fatal("filter != filter")
	}
	filter2 := NewImageFilterShader(shader)
	if !ImageFilterEqual(filter, filter2) || filter == filter2 {
		t.Fatal("filter vs filter_2")
	}
	filterLowQuality := NewImageFilterShader(shader, ImageFilterShaderOptions{FilterQuality: FilterQualityLow})
	if ImageFilterEqual(filter, filterLowQuality) {
		t.Fatal("filter == low quality")
	}
	if !ImageFilterEqual(filterLowQuality, NewImageFilterShader(shader, ImageFilterShaderOptions{FilterQuality: FilterQualityLow})) {
		t.Fatal("low != low")
	}
	shader.SetFloat(0, 1)
	filter3 := NewImageFilterShader(shader)
	if ImageFilterEqual(filter, filter3) {
		t.Fatal("filter == filter_3")
	}
	// filter_2's native filter was made before the change: still equal to
	// filter; a new one made after it equals filter_3.
	if !ImageFilterEqual(filter2, filter) || !ImageFilterEqual(filter3, NewImageFilterShader(shader)) {
		t.Fatal("snapshots")
	}
	// A different shader of the same program is never equal.
	if ImageFilterEqual(filter, NewImageFilterShader(loadProgram(t, "filter_shader.frag.iplr").FragmentShader())) {
		t.Fatal("other shader")
	}
	if ImageFilterEqual(filter, NewImageFilterBlur()) || ImageFilterEqual(filter, nil) {
		t.Fatal("other filter")
	}
}

func TestImageFilterShaderSamplerIdentity(t *testing.T) {
	shader := loadProgram(t, "children_and_uniforms.frag.iplr").FragmentShader()
	img := testBlueGreenImage()
	shader.SetImageSampler(1, img)
	a := NewImageFilterShader(shader)
	b := NewImageFilterShader(shader)
	if !ImageFilterEqual(a, b) {
		t.Fatal("same sampler")
	}
	// Setting the same image again makes a new sampler (DlColorSource).
	shader.SetImageSampler(1, img)
	if ImageFilterEqual(a, NewImageFilterShader(shader)) {
		t.Fatal("new sampler object compared equal")
	}
	// The first sampler (the input) is part of the comparison too.
	c := NewImageFilterShader(shader)
	ImageFilterEqual(c, c) // makes c's native filter now (Dart's late final)
	shader.SetImageSampler(0, img)
	if ImageFilterEqual(c, NewImageFilterShader(shader)) {
		t.Fatal("input sampler ignored")
	}
}

func TestImageFilterShaderToString(t *testing.T) {
	shader := loadProgram(t, "filter_shader.frag.iplr").FragmentShader()
	f := NewImageFilterShader(shader, ImageFilterShaderOptions{FilterQuality: FilterQualityMedium})
	want := "ImageFilter.shader(Shader#" + itoa(dart.IdentityHashCode(shader)) + ", FilterQuality.medium)"
	if f.String() != want {
		t.Fatalf("%q, want %q", f.String(), want)
	}
	if f.DebugShortDescription() != "shader" {
		t.Fatal(f.DebugShortDescription())
	}
	if !ImageFilterIsShaderFilterSupported() {
		t.Fatal("isShaderFilterSupported")
	}
}

func TestFragmentProgramAssetKeyEncoding(t *testing.T) {
	cases := map[string]string{
		"simple.frag.iplr":   "simple.frag.iplr",
		"a b.frag":           "a%20b.frag",
		"shaders/x y/z.frag": "shaders/x%20y/z.frag",
		"a#b?c":              "a%23b%3Fc",
		"100%.frag":          "100%25.frag",
		"é.frag":             "%C3%A9.frag",
		"a@b[c]":             "a%40b%5Bc%5D",
		"./a/../b/./c":       "b/c",
		"a/..":               "./",
		"../a":               "../a",
		"../../a":            "../../a",
		"a/../../b":          "../b",
		"/x/../y":            "/y",
		"/../a":              "/a",
		"/x/./":              "/x/",
		"c:/x":               "c%3A/x",
		"c:/./x":             "c%3A/x",
		"1:x":                "1:x",
		".//a":               ".//a",
		"":                   "",
	}
	for in, want := range cases {
		if got := uriPathOf(uriEncodeFull(in)); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

func TestDecodeRuntimeStagesErrors(t *testing.T) {
	assets := loadFragmentAssets(t)
	good := assets["simple.frag.iplr"]
	check := func(name string, data []byte, want string) {
		t.Helper()
		_, err := decodeRuntimeStages(data)
		if got := errorText(err); got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
	check("empty", []byte{}, "INVALID_ARGUMENT: Payload is null or empty.")
	check("short", []byte("IPLR"), "INVALID_ARGUMENT: Payload does not have valid identifier.")
	bad := append([]byte(nil), good...)
	bad[0] = 0xF0 // root offset past the end
	bad[1] = 0xFF
	check("bad root", bad, "INVALID_ARGUMENT: Runtime stages buffer failed verification.")
	check("truncated", good[:len(good)/2], "INVALID_ARGUMENT: Runtime stages buffer failed verification.")

	// The format version check: patch the version field.
	r := fbReader{good}
	root := r.tableAt(0)
	p := r.field(root, 5)
	if p < 0 {
		t.Fatal("no version field")
	}
	v3 := append([]byte(nil), good...)
	v3[p] = 3
	check("version", v3, "INVALID_ARGUMENT: Unsupported runtime stages format version. Expected 2, got 3.")

	// Every backend's stage is present in the fixtures.
	stages, err := decodeRuntimeStages(good)
	if err != nil {
		t.Fatal(err)
	}
	for b := runtimeStageBackendSkSL; b <= runtimeStageBackendVulkan; b++ {
		if stages[b] == nil {
			t.Errorf("%v missing", b)
		}
	}
	if stages[runtimeStageBackendMetal].entrypoint == "" {
		t.Error("no entrypoint")
	}
}

func TestFragmentProgramMissingBackendStage(t *testing.T) {
	loadFragmentAssets(t)
	dir := t.TempDir()
	src := filepath.Join(repoRoot, "engine", "src", "flutter", "lib", "ui", "fixtures", "shaders", "general_shaders", "simple.frag")
	data, err := compileShader(src, dir, []string{"--sksl", "--runtime-stage-vulkan"})
	if err != nil {
		t.Fatal(err)
	}
	p := &FragmentProgram{}
	prev := EngineAssetLoader
	defer func() { EngineAssetLoader = prev }()
	EngineAssetLoader = func(string) ([]byte, error) { return data, nil }
	if got := p.initFromAsset("vk.iplr"); got != "Asset 'vk.iplr' does not contain appropriate runtime stage data for current backend (Metal).\nFound stages: SkSL Vulkan " {
		t.Fatalf("%q", got)
	}
	EngineAssetLoader = func(string) ([]byte, error) { return nil, errors.New("x") }
	if got := p.initFromAsset("k"); got != "Asset 'k' not found" {
		t.Fatalf("%q", got)
	}
}

func TestFragmentShaderSetImageSamplerAssertsIfImageIsDisposed(t *testing.T) {
	shader := loadProgram(t, "blue_green_sampler.frag.iplr").FragmentShader()
	img := testBlueGreenImage()
	img.Dispose()
	r := expectPanicContains(t, "disposed image", func() { shader.SetImageSampler(0, img) }, "Image has been disposed")
	if _, ok := r.(*dart.AssertionError); !ok {
		t.Errorf("%T", r)
	}
	shader.Dispose()
}

// The reflection of a Vulkan stage (one uniform block, a "Struct"), as
// Flutter's tester reads it: the same float offsets, and a block field
// asked for as a sampler is "not an image sampler".
func TestFragmentProgramVulkanStructReflection(t *testing.T) {
	assets := loadFragmentAssets(t)
	stages, err := decodeRuntimeStages(assets["uniform_ordering.frag.iplr"])
	if err != nil {
		t.Fatal(err)
	}
	p := &FragmentProgram{uniformInfo: stages[runtimeStageBackendVulkan].uniforms}
	if got := p.getUniformFloatInfo("c"); got != (uniformFloatInfo{index: 2, size: 1}) {
		t.Fatalf("%+v", got)
	}
	if got := p.getImageSamplerIndex("u_texture"); got != 0 {
		t.Fatalf("%d", got)
	}
	expectPanicContains(t, "struct field", func() { p.getImageSamplerIndex("b") }, `Uniform "b" is not an image sampler.`)
	expectPanicContains(t, "unknown", func() { p.getImageSamplerIndex("zz") }, `No uniform named "zz".`)
	all, err := decodeRuntimeStages(assets["all_uniforms.frag.iplr"])
	if err != nil {
		t.Fatal(err)
	}
	vk := &FragmentProgram{uniformInfo: all[runtimeStageBackendVulkan].uniforms}
	mtl := &FragmentProgram{uniformInfo: all[runtimeStageBackendMetal].uniforms}
	for _, n := range []string{"uFloat", "uVec3", "uMat3", "uVec3Array", "uMat3Array", "uMat4Array"} {
		if a, b := vk.getUniformFloatInfo(n), mtl.getUniformFloatInfo(n); a != b {
			t.Errorf("%s: vulkan %+v, metal %+v", n, a, b)
		}
	}
}

// The engine appends each sampler's width and height to the uniform data
// (fragment_shader.cc's SetImageSampler).
func TestFragmentShaderSamplerSizes(t *testing.T) {
	shader := loadProgram(t, "children_and_uniforms.frag.iplr").FragmentShader()
	shader.SetImageSampler(1, newImageFromRGBA8(3, 2, make([]byte, 24)))
	shader.SetImageSampler(0, newImageFromRGBA8(5, 7, make([]byte, 140)))
	if got := shader.uniformData; !reflect.DeepEqual(got, []float32{0, 0, 5, 7, 3, 2}) {
		t.Fatalf("%v", got)
	}
	if !shader.validateSamplers() || !shader.validateImageFilter() {
		t.Fatal("samplers set")
	}
	// The uniform bytes handed to Impeller are the little-endian floats.
	shader.SetFloat(1, 1)
	if got := shader.uniformBytes(); !reflect.DeepEqual(got[4:8], []byte{0, 0, 0x80, 0x3f}) || len(got) != 24 {
		t.Fatalf("% x", got)
	}
	// A disposed image's data can't be bound (the engine's check, reached
	// in release builds).
	img := newImageFromRGBA8(1, 1, make([]byte, 4))
	img.Dispose()
	if r := panics(func() { shader.setImageSampler(0, img, FilterQualityNone) }); r != "Image has been disposed" {
		t.Fatalf("%#v", r)
	}
	// One of two handles disposed: the data is still there.
	img2 := newImageFromRGBA8(1, 1, make([]byte, 4))
	clone := img2.Clone()
	img2.Dispose()
	shader.setImageSampler(0, clone, FilterQualityNone)
}
