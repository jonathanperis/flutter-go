package ui

// Port of painting.dart's FragmentProgram, FragmentShader, UniformType and
// its slots, UniformArray, ImageSamplerSlot (Flutter 3.50.1), with the
// engine's natives (lib/ui/painting/fragment_program.cc,
// fragment_shader.cc) on the Impeller SDK.

import (
	"encoding/binary"
	"math"
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/impeller"
)

// uniformFloatInfo is Dart's _UniformFloatInfo record.
type uniformFloatInfo struct{ index, size int }

// FragmentProgram creates Shader objects (FragmentShader) from a shader
// compiled by impellerc (an "iplr" runtime stage bundle).
//
// For more information, see the website documentation:
// https://docs.flutter.dev/development/ui/advanced/shaders.
type FragmentProgram struct {
	debugName string

	// Set by the engine (Dart's vm:entry-point fields).
	uniformFloatCount int
	samplerCount      int
	uniformInfo       []runtimeUniformDescription

	// The engine's runtime effect: the SDK program (nil without a GPU
	// context).
	program *impeller.FragmentProgram
}

// shaderRegistry is FragmentProgram._shaderRegistry: programs loaded by
// FragmentProgramFromAsset, by encoded asset key.
var shaderRegistry = map[string]*FragmentProgram{}

// FragmentProgramFromAsset is FragmentProgram.fromAsset: creates a fragment
// program from the asset with key assetKey, read through EngineAssetLoader.
//
// The asset must be a file produced as the output of the impellerc
// compiler (with --iplr). The program is cached: later calls with the same
// key complete with the same object. The future fails with a
// dart.Exception when the asset is missing or isn't a runtime stage bundle
// for the engine's backend.
func FragmentProgramFromAsset(assetKey string) async.Future[*FragmentProgram] {
	// The flutter tool converts all asset keys with spaces into URI
	// encoded paths (replacing ' ' with '%20', for example). We perform
	// the same encoding here so that users can load assets with the same
	// key they have written in the pubspec.
	encodedKey := uriPathOf(uriEncodeFull(assetKey))
	if program := shaderRegistry[encodedKey]; program != nil {
		return async.FutureValue(program)
	}
	return async.FutureMicrotask(func() *FragmentProgram {
		program := newFragmentProgramFromAsset(encodedKey)
		shaderRegistry[encodedKey] = program
		return program
	})
}

// newFragmentProgramFromAsset is FragmentProgram._fromAsset.
func newFragmentProgramFromAsset(assetKey string) *FragmentProgram {
	p := &FragmentProgram{}
	if result := p.initFromAsset(assetKey); result != "" {
		panic(dart.NewException(result))
	}
	if debugMode {
		p.debugName = assetKey
	}
	return p
}

// initFromAsset is FragmentProgram::initFromAsset: it returns an error
// message, or "" on success.
func (p *FragmentProgram) initFromAsset(assetName string) string {
	var data []byte
	if EngineAssetLoader != nil {
		if d, err := EngineAssetLoader(assetName); err == nil && d != nil {
			data = d
		}
	}
	if data == nil {
		return "Asset '" + assetName + "' not found"
	}
	stages, err := decodeRuntimeStages(data)
	if err != nil {
		return "Asset '" + assetName + "' manifest could not be decoded: " + err.Error()
	}
	if len(stages) == 0 {
		return "Asset '" + assetName + "' does not contain any shader data."
	}
	backend := engineRuntimeStageBackend
	stage := stages[backend]
	if stage == nil {
		var b strings.Builder
		b.WriteString("Asset '" + assetName + "' does not contain appropriate runtime stage data for current backend (" +
			backend.String() + ").\nFound stages: ")
		for k := runtimeStageBackendSkSL; k <= runtimeStageBackendVulkan; k++ {
			if stages[k] != nil {
				b.WriteString(k.String() + " ")
			}
		}
		return b.String()
	}

	sampledImageCount := 0
	otherUniformsBytes := 0
	for i := range stage.uniforms {
		if stage.uniforms[i].typ == runtimeUniformTypeSampledImage {
			sampledImageCount++
		} else {
			otherUniformsBytes += stage.uniforms[i].dartSize()
		}
	}
	// The engine's runtime effect: the SDK decodes the bundle again and
	// picks the stage of its context's backend (Metal) when a color source
	// or image filter is made.
	if ctx := engineContext(); ctx != nil {
		if p.program != nil {
			p.program.Release()
		}
		p.program = impeller.NewFragmentProgram(data)
	}
	p.samplerCount = sampledImageCount
	p.uniformFloatCount = (otherUniformsBytes + 3) &^ 3 / 4
	p.uniformInfo = stage.uniforms
	return ""
}

func (p *FragmentProgram) hasUniform(name string) bool {
	for i := range p.uniformInfo {
		e := &p.uniformInfo[i]
		if e.name == name {
			return true
		}
		if e.typ == runtimeUniformTypeStruct {
			for _, f := range e.structFields {
				if f.name == name {
					return true
				}
			}
		}
	}
	return false
}

func (p *FragmentProgram) getImageSamplerIndex(name string) int {
	index := 0
	found := false
	for i := range p.uniformInfo {
		e := &p.uniformInfo[i]
		if e.name == name {
			if e.typ != runtimeUniformTypeSampledImage {
				panic(dart.ArgumentError{Message: `Uniform "` + name + `" is not an image sampler.`})
			}
			found = true
			break
		} else if e.typ == runtimeUniformTypeSampledImage {
			index++
		}
	}
	if !found {
		if p.hasUniform(name) {
			panic(dart.ArgumentError{Message: `Uniform "` + name + `" is not an image sampler.`})
		}
		panic(dart.ArgumentError{Message: `No uniform named "` + name + `".`})
	}
	return index
}

func (p *FragmentProgram) getUniformFloatInfo(name string) uniformFloatInfo {
	offset := 0
	sizeInFloats := 0
	found := false
	const sizeOfFloat = 4
	for i := range p.uniformInfo {
		e := &p.uniformInfo[i]
		if found {
			break
		}
		if e.typ == runtimeUniformTypeStruct {
			for _, f := range e.structFields {
				sizeInFloats = f.byteSize / sizeOfFloat
				if f.name == name {
					found = true
					break
				}
				offset += sizeInFloats
			}
		} else {
			sizeInFloats = e.dartSize() / sizeOfFloat
			if e.name == name {
				found = true
				break
			}
			offset += sizeInFloats
		}
	}
	if !found {
		panic(dart.ArgumentError{Message: `No uniform named "` + name + `".`})
	}
	return uniformFloatInfo{index: offset, size: sizeInFloats}
}

// FragmentShader returns a fresh instance of FragmentShader.
func (p *FragmentProgram) FragmentShader() *FragmentShader {
	return newFragmentShader(p, p.debugName)
}

// UniformType is a binding into a uniform defined in a shader, used to
// restrict the types of UniformArrays that can be created. Dart's class is
// sealed: the slot types of this package are its only implementations.
type UniformType interface{ uniformType() }

// UniformFloatSlot is a binding to a uniform of type float. Calling Set on
// it updates a float uniform's value. FragmentShader.GetUniformFloat
// returns it.
type UniformFloatSlot struct {
	shader      *FragmentShader
	shaderIndex int

	// Name is the name of the bound uniform.
	Name string
	// Index is the offset into the bound uniform. For example, 1 for `.y`
	// or 2 for `.b`.
	Index int
}

func (*UniformFloatSlot) uniformType() {}

// Set sets the float value of the bound uniform.
func (s *UniformFloatSlot) Set(val float64) { s.shader.SetFloat(s.shaderIndex, val) }

// ShaderIndex is the index one would use with FragmentShader.SetFloat for
// this uniform (VisibleForTesting).
func (s *UniformFloatSlot) ShaderIndex() int { return s.shaderIndex }

// UniformVec2Slot is a binding to a uniform of type vec2.
type UniformVec2Slot struct{ xSlot, ySlot *UniformFloatSlot }

func (*UniformVec2Slot) uniformType() {}

// Set sets the float values of the bound uniform.
func (s *UniformVec2Slot) Set(x, y float64) {
	s.xSlot.Set(x)
	s.ySlot.Set(y)
}

// UniformVec3Slot is a binding to a uniform of type vec3.
type UniformVec3Slot struct{ xSlot, ySlot, zSlot *UniformFloatSlot }

func (*UniformVec3Slot) uniformType() {}

// Set sets the float values of the bound uniform.
func (s *UniformVec3Slot) Set(x, y, z float64) {
	s.xSlot.Set(x)
	s.ySlot.Set(y)
	s.zSlot.Set(z)
}

// UniformVec4Slot is a binding to a uniform of type vec4.
type UniformVec4Slot struct{ xSlot, ySlot, zSlot, wSlot *UniformFloatSlot }

func (*UniformVec4Slot) uniformType() {}

// Set sets the float values of the bound uniform.
func (s *UniformVec4Slot) Set(x, y, z, w float64) {
	s.xSlot.Set(x)
	s.ySlot.Set(y)
	s.zSlot.Set(z)
	s.wSlot.Set(w)
}

// UniformMat2Slot is a binding to a uniform of type mat2.
type UniformMat2Slot struct{ m [4]*UniformFloatSlot }

func (*UniformMat2Slot) uniformType() {}

// Set sets the float values of the matrix in row-major order.
func (s *UniformMat2Slot) Set(m00, m01, m10, m11 float64) {
	setSlots(s.m[:], m00, m01, m10, m11)
}

// UniformMat3Slot is a binding to a uniform of type mat3.
type UniformMat3Slot struct{ m [9]*UniformFloatSlot }

func (*UniformMat3Slot) uniformType() {}

// Set sets the float values of the matrix in row-major order.
func (s *UniformMat3Slot) Set(m00, m01, m02, m10, m11, m12, m20, m21, m22 float64) {
	setSlots(s.m[:], m00, m01, m02, m10, m11, m12, m20, m21, m22)
}

// UniformMat4Slot is a binding to a uniform of type mat4.
type UniformMat4Slot struct{ m [16]*UniformFloatSlot }

func (*UniformMat4Slot) uniformType() {}

// Set sets the float values of the matrix in row-major order.
func (s *UniformMat4Slot) Set(m00, m01, m02, m03, m10, m11, m12, m13, m20, m21, m22, m23, m30, m31, m32, m33 float64) {
	setSlots(s.m[:], m00, m01, m02, m03, m10, m11, m12, m13, m20, m21, m22, m23, m30, m31, m32, m33)
}

func setSlots(slots []*UniformFloatSlot, values ...float64) {
	for i, v := range values {
		slots[i].Set(v)
	}
}

// UniformArray is an array of bindings to uniforms of the same type T.
// Access elements with At and set them individually.
type UniformArray[T UniformType] struct{ elements []T }

// At is operator [](index): the element at index. An index out of range
// panics with the RangeError Dart's growable list throws.
func (a *UniformArray[T]) At(index int) T {
	if index < 0 || index >= len(a.elements) {
		panic(dart.NewRangeErrorRange(index, 0, len(a.elements)-1, "length", ""))
	}
	return a.elements[index]
}

// Length is the number of uniforms in the array.
func (a *UniformArray[T]) Length() int { return len(a.elements) }

// ImageSamplerSlot is a binding to a shader's image sampler. Calling Set
// on it updates a sampler's bound image.
type ImageSamplerSlot struct {
	shader      *FragmentShader
	shaderIndex int

	// Name is the name of the bound uniform.
	Name string
}

// Set sets the Image value for the bound sampler associated with this
// slot.
func (s *ImageSamplerSlot) Set(val *Image) { s.shader.SetImageSampler(s.shaderIndex, val) }

// ShaderIndex is the index one would use with
// FragmentShader.SetImageSampler for this sampler (VisibleForTesting).
func (s *ImageSamplerSlot) ShaderIndex() int { return s.shaderIndex }

// shaderSampler is the DlColorSource::MakeImage the engine makes for a
// sampler: a new object for each SetImageSampler, compared by identity.
type shaderSampler struct {
	image         *Image
	filterQuality FilterQuality
}

// FragmentShader is a Shader generated from a FragmentProgram.
//
// Instances are obtained from FragmentProgram.FragmentShader. The float
// uniforms list is initialized to the size expected by the shader and is
// zero-filled. Uniforms of float type can then be set by calling
// SetFloat. Sampler uniforms are set by calling SetImageSampler.
//
// A FragmentShader can be re-used, and this is an efficient way to avoid
// allocating and re-initializing the uniform buffer and samplers. However,
// if two FragmentShader objects with different float uniforms or samplers
// are required to exist simultaneously, they must be obtained from two
// different calls to FragmentProgram.FragmentShader.
type FragmentShader struct {
	shaderBase
	program        *FragmentProgram
	debugNameValue string // Dart's _debugName ("" for null)

	// floats is Dart's _floats, a view of the first floatCount floats of
	// the native uniform data.
	floats []float32

	// The native ReusableFragmentShader: the uniform data (floats, then
	// each sampler's width and height), the samplers, and the program
	// (nil once disposed).
	uniformData   []float32
	samplers      []*shaderSampler
	nativeProgram *FragmentProgram

	// The color source made from the current uniforms and samplers.
	source      *impeller.ColorSource
	sourceStale bool
}

func newFragmentShader(program *FragmentProgram, debugName string) *FragmentShader {
	s := &FragmentShader{program: program, debugNameValue: debugName}
	s.floats = s.constructor(program, program.uniformFloatCount, program.samplerCount)
	return s
}

// constructor is ReusableFragmentShader::Create: it returns the view of the
// float uniforms.
func (s *FragmentShader) constructor(program *FragmentProgram, floatCount, samplerCount int) []float32 {
	s.nativeProgram = program
	s.uniformData = make([]float32, floatCount+2*samplerCount)
	s.samplers = make([]*shaderSampler, samplerCount)
	s.sourceStale = true
	return s.uniformData[:floatCount:floatCount]
}

// debugName is Dart's `_debugName ?? ”` (Paint.shader's debug check).
func (s *FragmentShader) debugName() string { return s.debugNameValue }

// String is Object.toString (FragmentShader doesn't override it).
func (s *FragmentShader) String() string { return "Instance of 'FragmentShader'" }

func (s *FragmentShader) getSlotsForUniform(name string, size int) []*UniformFloatSlot {
	info := s.program.getUniformFloatInfo(name)
	if info.size != size {
		panic(dart.ArgumentError{Message: "Uniform `" + name + "` has size " + strconv.Itoa(info.size) +
			", not size " + strconv.Itoa(size) + "."})
	}
	slots := make([]*UniformFloatSlot, size)
	for i := range slots {
		slots[i] = &UniformFloatSlot{shader: s, Name: name, Index: i, shaderIndex: info.index + i}
	}
	return slots
}

// SetFloat sets the float uniform at index to value.
//
// All uniforms defined in a fragment shader that are not samplers must be
// set through this method. This includes floats and vec2, vec3, and vec4.
// The correct index for each uniform is determined by the order of the
// uniforms as defined in the fragment program, ignoring any samplers. For
// data types that are composed of multiple floats such as a vec4, more than
// one call to SetFloat is required. Any float uniforms that are left
// uninitialized default to 0.
func (s *FragmentShader) SetFloat(index int, value float64) {
	if s.debugDisposed {
		panic(dart.NewAssertionError("Tried to accesss uniforms on a disposed Shader: " + s.String()))
	}
	if index < 0 || index >= len(s.floats) {
		panic(newIndexError(index, len(s.floats), ""))
	}
	s.floats[index] = float32(value)
	s.sourceStale = true
}

// GetUniformFloat accesses the float binding for the uniform named name
// with optional offset index (default 0). Example index values: 1 for
// 'foo.y', 2 for 'foo.b'.
func (s *FragmentShader) GetUniformFloat(name string, index ...int) *UniformFloatSlot {
	i := 0
	if len(index) > 0 {
		i = index[0]
	}
	info := s.program.getUniformFloatInfo(name)
	if i < 0 || i >= info.size {
		panic(newIndexError(i, info.size, "Index `"+strconv.Itoa(i)+"` out of bounds for `"+name+"`."))
	}
	return &UniformFloatSlot{shader: s, Name: name, Index: i, shaderIndex: info.index + i}
}

// GetUniformVec2 accesses the float binding for a vec2 uniform named name.
func (s *FragmentShader) GetUniformVec2(name string) *UniformVec2Slot {
	slots := s.getSlotsForUniform(name, 2)
	return &UniformVec2Slot{slots[0], slots[1]}
}

// GetUniformVec3 accesses the float binding for a vec3 uniform named name.
func (s *FragmentShader) GetUniformVec3(name string) *UniformVec3Slot {
	slots := s.getSlotsForUniform(name, 3)
	return &UniformVec3Slot{slots[0], slots[1], slots[2]}
}

// GetUniformVec4 accesses the float binding for a vec4 uniform named name.
func (s *FragmentShader) GetUniformVec4(name string) *UniformVec4Slot {
	slots := s.getSlotsForUniform(name, 4)
	return &UniformVec4Slot{slots[0], slots[1], slots[2], slots[3]}
}

// GetUniformMat2 accesses the float binding for a mat2 uniform named name.
func (s *FragmentShader) GetUniformMat2(name string) *UniformMat2Slot {
	return mat2Slot(s.getSlotsForUniform(name, 4))
}

// GetUniformMat3 accesses the float binding for a mat3 uniform named name.
func (s *FragmentShader) GetUniformMat3(name string) *UniformMat3Slot {
	return mat3Slot(s.getSlotsForUniform(name, 9))
}

// GetUniformMat4 accesses the float binding for a mat4 uniform named name.
func (s *FragmentShader) GetUniformMat4(name string) *UniformMat4Slot {
	return mat4Slot(s.getSlotsForUniform(name, 16))
}

func mat2Slot(c []*UniformFloatSlot) *UniformMat2Slot {
	m := &UniformMat2Slot{}
	copy(m.m[:], c)
	return m
}

func mat3Slot(c []*UniformFloatSlot) *UniformMat3Slot {
	m := &UniformMat3Slot{}
	copy(m.m[:], c)
	return m
}

func mat4Slot(c []*UniformFloatSlot) *UniformMat4Slot {
	m := &UniformMat4Slot{}
	copy(m.m[:], c)
	return m
}

// getUniformArray is FragmentShader._getUniformArray.
func getUniformArray[T UniformType](s *FragmentShader, name string, elementSize int,
	elementFactory func(slots []*UniformFloatSlot) T) *UniformArray[T] {
	info := s.program.getUniformFloatInfo(name)
	if info.size%elementSize != 0 {
		panic(dart.ArgumentError{Message: "Uniform size (" + strconv.Itoa(info.size) + `) for "` + name +
			`" is not a multiple of ` + strconv.Itoa(elementSize) + "."})
	}
	numElements := info.size / elementSize
	elements := make([]T, numElements)
	for i := range elements {
		// As in Dart, each element's slot list has info.size slots, of
		// which the factory uses the first elementSize.
		slots := make([]*UniformFloatSlot, info.size)
		for j := range slots {
			slots[j] = &UniformFloatSlot{shader: s, Name: name, Index: j, shaderIndex: info.index + i*elementSize + j}
		}
		elements[i] = elementFactory(slots)
	}
	return &UniformArray[T]{elements}
}

// GetUniformFloatArray accesses the binding for a float[] uniform named
// name.
func (s *FragmentShader) GetUniformFloatArray(name string) *UniformArray[*UniformFloatSlot] {
	return getUniformArray(s, name, 1, func(c []*UniformFloatSlot) *UniformFloatSlot { return c[0] })
}

// GetUniformVec2Array accesses the binding for a vec2[] uniform named name.
func (s *FragmentShader) GetUniformVec2Array(name string) *UniformArray[*UniformVec2Slot] {
	return getUniformArray(s, name, 2, func(c []*UniformFloatSlot) *UniformVec2Slot {
		return &UniformVec2Slot{c[0], c[1]}
	})
}

// GetUniformVec3Array accesses the binding for a vec3[] uniform named name.
func (s *FragmentShader) GetUniformVec3Array(name string) *UniformArray[*UniformVec3Slot] {
	return getUniformArray(s, name, 3, func(c []*UniformFloatSlot) *UniformVec3Slot {
		return &UniformVec3Slot{c[0], c[1], c[2]}
	})
}

// GetUniformVec4Array accesses the binding for a vec4[] uniform named name.
func (s *FragmentShader) GetUniformVec4Array(name string) *UniformArray[*UniformVec4Slot] {
	return getUniformArray(s, name, 4, func(c []*UniformFloatSlot) *UniformVec4Slot {
		return &UniformVec4Slot{c[0], c[1], c[2], c[3]}
	})
}

// GetUniformMat2Array accesses the binding for a mat2[] uniform named name.
func (s *FragmentShader) GetUniformMat2Array(name string) *UniformArray[*UniformMat2Slot] {
	return getUniformArray(s, name, 4, mat2Slot)
}

// GetUniformMat3Array accesses the binding for a mat3[] uniform named name.
func (s *FragmentShader) GetUniformMat3Array(name string) *UniformArray[*UniformMat3Slot] {
	return getUniformArray(s, name, 9, mat3Slot)
}

// GetUniformMat4Array accesses the binding for a mat4[] uniform named name.
func (s *FragmentShader) GetUniformMat4Array(name string) *UniformArray[*UniformMat4Slot] {
	return getUniformArray(s, name, 16, mat4Slot)
}

// GetImageSampler accesses the ImageSamplerSlot binding associated with the
// sampler named name. Its index is the index of the sampler uniform in the
// fragment program, excluding all non-sampler uniforms.
func (s *FragmentShader) GetImageSampler(name string) *ImageSamplerSlot {
	index := s.program.getImageSamplerIndex(name)
	return &ImageSamplerSlot{shader: s, Name: name, shaderIndex: index}
}

// SetImageSamplerOptions are FragmentShader.setImageSampler's named
// parameters.
type SetImageSamplerOptions struct {
	// FilterQuality is the quality level used to sample the image
	// (default FilterQualityNone).
	FilterQuality FilterQuality
}

// SetImageSampler sets the sampler uniform at index to image. The index is
// the index of the sampler uniform defined in the fragment program,
// excluding all non-sampler uniforms. All the sampler uniforms that a
// shader expects must be provided or the results will be undefined.
func (s *FragmentShader) SetImageSampler(index int, image *Image, opts ...SetImageSamplerOptions) {
	var o SetImageSamplerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if s.debugDisposed {
		panic(dart.NewAssertionError("Tried to access uniforms on a disposed Shader: " + s.String()))
	}
	if imageDebugDisposed(image) {
		panic(dart.NewAssertionError("Image has been disposed"))
	}
	s.setImageSampler(index, image, o.FilterQuality)
}

// setImageSampler is ReusableFragmentShader::SetImageSampler (its errors are
// thrown strings).
func (s *FragmentShader) setImageSampler(index int, image *Image, filterQuality FilterQuality) {
	if index < 0 || index >= len(s.samplers) {
		panic("Sampler index out of bounds")
	}
	if image == nil || imageEngineDisposed(image) {
		panic("Image has been disposed")
	}
	s.samplers[index] = &shaderSampler{image: image, filterQuality: filterQuality}
	floatCount := len(s.uniformData) - 2*len(s.samplers)
	w, h := imageSize(image)
	s.uniformData[floatCount+2*index] = float32(w)
	s.uniformData[floatCount+2*index+1] = float32(h)
	s.sourceStale = true
}

// validateSamplers is ReusableFragmentShader::ValidateSamplers: whether
// every sampler is set (Paint.shader's debug check).
func (s *FragmentShader) validateSamplers() bool {
	for _, sampler := range s.samplers {
		if sampler == nil {
			return false
		}
	}
	return true
}

// validateImageFilter is ReusableFragmentShader::ValidateImageFilter: image
// filters need at least one sampler, the first of which (the filter input)
// need not be set.
func (s *FragmentShader) validateImageFilter() bool {
	if len(s.samplers) < 1 {
		return false
	}
	for _, sampler := range s.samplers[1:] {
		if sampler == nil {
			return false
		}
	}
	return true
}

// Dispose releases the native resources held by the FragmentShader.
//
// After this method is called, calling methods on the shader, or attaching
// it to a Paint object, will fail with an exception. Calling Dispose twice
// will also panic.
func (s *FragmentShader) Dispose() {
	s.shaderBase.dispose()
	s.floats = []float32{}
	// ReusableFragmentShader::Dispose.
	s.uniformData = nil
	s.nativeProgram = nil
	s.samplers = nil
	if s.source != nil {
		s.source.Release()
		s.source = nil
	}
}

// uniformBytes is a copy of the native uniform data, as the engine hands
// it to the DisplayList.
func (s *FragmentShader) uniformBytes() []byte {
	b := make([]byte, 4*len(s.uniformData))
	for i, f := range s.uniformData {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(f))
	}
	return b
}

// impellerColorSource is ReusableFragmentShader::shader: a runtime effect
// color source with a copy of the current uniforms and samplers, remade
// when they change.
func (s *FragmentShader) impellerColorSource() *impeller.ColorSource {
	if s.nativeProgram == nil {
		return nil
	}
	if s.source != nil && !s.sourceStale {
		return s.source
	}
	if s.source != nil {
		s.source.Release()
		s.source = nil
	}
	ctx := engineContext()
	if ctx == nil || s.nativeProgram.program == nil {
		return nil
	}
	textures := make([]*impeller.Texture, len(s.samplers))
	for i, sampler := range s.samplers {
		var tex *impeller.Texture
		if sampler != nil {
			tex = sampler.image.impellerTexture()
		}
		if tex == nil {
			// Impeller draws a runtime effect with a missing sampler as
			// transparent black (paint.cc).
			s.source = transparentColorSource()
			s.sourceStale = false
			return s.source
		}
		textures[i] = tex
	}
	s.source = impeller.NewFragmentProgramColorSource(ctx, s.nativeProgram.program, textures, s.uniformBytes())
	s.sourceStale = false
	return s.source
}

// transparentColorSource is a color source that paints transparent black.
func transparentColorSource() *impeller.ColorSource {
	clear := impeller.Color{}
	return impeller.NewLinearGradient(impeller.Point{}, impeller.Point{X: 1}, []impeller.Color{clear, clear},
		[]float32{0, 1}, 0, nil)
}

// newIndexError is IndexError.check's error (the VM's typed data and list
// index errors are the same, with the default message): name "index".
func newIndexError(index, length int, message string) dart.IndexError {
	if message == "" {
		message = "Index out of range"
	}
	return dart.IndexError{Index: index, Length: length, Name: "index", Message: message}
}

// uriEncodeFull is Uri.encodeFull for the ASCII and UTF-8 text of a Go
// string: every byte outside [A-Za-z0-9!#$&'()*+,\-./:;=?_~] is
// percent-encoded.
func uriEncodeFull(s string) string {
	return uriEncode(s, "!#$&'()*+,-./:;=?_~")
}

func uriEncode(s, allowed string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x80 && (c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.IndexByte(allowed, c) >= 0) {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		}
	}
	return b.String()
}

// uriPathOf is Uri(path: p).path for a p made by uriEncodeFull (so every
// '%' starts a valid uppercase escape of a character that isn't
// unreserved): the characters not allowed in a path ('#', '?') are
// escaped, then dot segments are normalized as for a URI with neither
// scheme nor authority.
func uriPathOf(p string) string {
	p = strings.NewReplacer("#", "%23", "?", "%3F").Replace(p)
	if strings.HasPrefix(p, "/") {
		return removeDotSegments(p)
	}
	return normalizeRelativePath(p)
}

func mayContainDotSegments(p string) bool {
	return strings.HasPrefix(p, ".") || strings.Contains(p, "/.")
}

// removeDotSegments is _Uri._removeDotSegments.
func removeDotSegments(p string) string {
	if !mayContainDotSegments(p) {
		return p
	}
	var output []string
	appendSlash := false
	for _, segment := range strings.Split(p, "/") {
		appendSlash = false
		switch segment {
		case "..":
			if len(output) > 0 {
				output = output[:len(output)-1]
				if len(output) == 0 {
					output = append(output, "")
				}
			}
			appendSlash = true
		case ".":
			appendSlash = true
		default:
			output = append(output, segment)
		}
	}
	if appendSlash {
		output = append(output, "")
	}
	return strings.Join(output, "/")
}

// normalizeRelativePath is _Uri._normalizeRelativePath(path, false).
func normalizeRelativePath(p string) string {
	if !mayContainDotSegments(p) {
		return escapeScheme(p)
	}
	var output []string
	appendSlash := false
	for _, segment := range strings.Split(p, "/") {
		switch segment {
		case "..":
			appendSlash = true
			if len(output) > 0 && output[len(output)-1] != ".." {
				output = output[:len(output)-1]
			} else {
				output = append(output, "..")
			}
		case ".":
			appendSlash = true
		default:
			appendSlash = false
			if segment == "" && len(output) == 0 {
				segment = "./"
			}
			output = append(output, segment)
		}
	}
	if len(output) == 0 {
		return "./"
	}
	if appendSlash {
		output = append(output, "")
	}
	output[0] = escapeScheme(output[0])
	return strings.Join(output, "/")
}

// escapeScheme is _Uri._escapeScheme: a leading "scheme:" has its colon
// escaped.
func escapeScheme(p string) string {
	if len(p) >= 2 && (p[0] >= 'a' && p[0] <= 'z' || p[0] >= 'A' && p[0] <= 'Z') {
		for i := 1; i < len(p); i++ {
			c := p[i]
			if c == ':' {
				return p[:i] + "%3A" + p[i+1:]
			}
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.') {
				break
			}
		}
	}
	return p
}
