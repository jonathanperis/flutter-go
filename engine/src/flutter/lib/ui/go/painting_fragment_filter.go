package ui

// Port of painting.dart's ImageFilter.shader and _FragmentShaderImageFilter
// (Flutter 3.50.1), with the engine's ImageFilter::initShader and
// ReusableFragmentShader::as_image_filter.

import (
	"bytes"
	"strconv"
	"strings"
	"sync"

	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/impeller"
)

// impellerEnabled is dart:ui's _impellerEnabled: the engine always renders
// with Impeller.
const impellerEnabled = true

// ImageFilterShaderOptions are ImageFilter.shader's named parameters.
type ImageFilterShaderOptions struct {
	// FilterQuality is the quality level used to sample the filter input
	// (default FilterQualityNone).
	FilterQuality FilterQuality
}

// NewImageFilterShader is ImageFilter.shader: an image filter running a
// FragmentShader on its input.
//
// The shader has additional requirements to be used by the engine for
// filtering. The first uniform must be a vec2, which the engine sets to
// the size of the bound texture. There must also be at least one sampler2D
// uniform, the first of which the engine sets to the filter input. It
// panics with a dart.StateError otherwise.
func NewImageFilterShader(shader *FragmentShader, opts ...ImageFilterShaderOptions) ImageFilter {
	var o ImageFilterShaderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if !impellerEnabled {
		panic(dart.UnsupportedError{Message: "ImageFilter.shader only supported with Impeller rendering engine."})
	}
	invalidFloats := len(shader.floats) < 2
	invalidSampler := !shader.validateImageFilter()
	if invalidFloats || invalidSampler {
		var b strings.Builder
		b.WriteString("ImageFilter.shader requires that the first uniform is a vec2 and at " +
			"least one sampler uniform is present.\n")
		if invalidFloats {
			b.WriteString("The shader has fewer than two float uniforms.\n")
		}
		if invalidSampler {
			b.WriteString("The shader is missing a sampler uniform.\n")
		}
		panic(dart.StateError{Message: b.String()})
	}
	return &fragmentShaderImageFilter{shader: shader, filterQuality: o.FilterQuality}
}

// fragmentShaderImageFilter is _FragmentShaderImageFilter.
type fragmentShaderImageFilter struct {
	shader        *FragmentShader
	filterQuality FilterQuality

	// nativeFilter is Dart's `late final nativeFilter`: made on first use,
	// with the shader's uniforms and samplers at that time.
	nativeFilter *shaderNativeFilter
}

// shaderNativeFilter is the DlRuntimeEffectImageFilter the engine makes
// (ReusableFragmentShader::as_image_filter): copies of the uniform data and
// the sampler list, and the input sampling.
type shaderNativeFilter struct {
	program       *FragmentProgram // the runtime effect
	samplers      []*shaderSampler
	uniformData   []byte
	inputSampling FilterQuality
	filter        *impeller.ImageFilter
	made          bool
}

func (f *fragmentShaderImageFilter) native() *shaderNativeFilter {
	if f.nativeFilter == nil {
		s := f.shader
		if s.nativeProgram == nil {
			panic("Check failed: program_") // FML_CHECK in as_image_filter
		}
		f.nativeFilter = &shaderNativeFilter{
			program:       s.nativeProgram,
			samplers:      append([]*shaderSampler(nil), s.samplers...),
			uniformData:   s.uniformBytes(),
			inputSampling: f.filterQuality,
		}
	}
	return f.nativeFilter
}

// String is _FragmentShaderImageFilter.toString.
func (f *fragmentShaderImageFilter) String() string {
	return "ImageFilter.shader(Shader#" + strconv.Itoa(dart.IdentityHashCode(f.shader)) + ", " +
		f.filterQuality.String() + ")"
}

// DebugShortDescription is ImageFilter.debugShortDescription.
func (f *fragmentShaderImageFilter) DebugShortDescription() string { return "shader" }

// equalImageFilter is _FragmentShaderImageFilter's ==: the same shader and
// filter quality, and equal native filters (DlRuntimeEffectImageFilter's
// ==: the same runtime effect, the same sampler objects, the same input
// sampling and uniform bytes).
func (f *fragmentShaderImageFilter) equalImageFilter(other ImageFilter) bool {
	o, ok := other.(*fragmentShaderImageFilter)
	if !ok || o.shader != f.shader || o.filterQuality != f.filterQuality {
		return false
	}
	a, b := f.native(), o.native()
	if a == b {
		return true
	}
	if a.program != b.program || len(a.samplers) != len(b.samplers) ||
		len(a.uniformData) != len(b.uniformData) || a.inputSampling != b.inputSampling {
		return false
	}
	for i := range a.samplers {
		if a.samplers[i] != b.samplers[i] {
			return false
		}
	}
	return bytes.Equal(a.uniformData, b.uniformData)
}

// impellerImageFilter makes the SDK's runtime effect image filter. The SDK
// needs a texture for every sampler, so the first (replaced by the filter
// input) gets a placeholder. ENGINE GAP: the SDK always samples the input
// with nearest-neighbor filtering and the other samplers with linear
// filtering, whatever the filter qualities.
func (f *fragmentShaderImageFilter) impellerImageFilter() *impeller.ImageFilter {
	n := f.native()
	if n.made {
		return n.filter
	}
	n.made = true
	ctx := engineContext()
	if ctx == nil || n.program.program == nil {
		return nil
	}
	textures := make([]*impeller.Texture, len(n.samplers))
	for i, sampler := range n.samplers {
		var tex *impeller.Texture
		if i == 0 {
			tex = placeholderTexture(ctx)
		} else if sampler != nil {
			tex = sampler.image.impellerTexture()
		}
		if tex == nil {
			return nil
		}
		textures[i] = tex
	}
	n.filter = impeller.NewFragmentProgramImageFilter(ctx, n.program.program, textures, n.uniformData)
	return n.filter
}

var (
	placeholderOnce sync.Once
	placeholderTex  *impeller.Texture
)

// placeholderTexture is a 1x1 transparent texture standing for an image
// filter's input sampler.
func placeholderTexture(ctx *impeller.Context) *impeller.Texture {
	placeholderOnce.Do(func() { placeholderTex = impeller.NewTextureRGBA8(ctx, 1, 1, make([]byte, 4)) })
	return placeholderTex
}
