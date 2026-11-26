package impeller

// Bindings for the SDK's fragment programs (runtime effects): programs
// compiled by impellerc, and the color sources and image filters that run
// them. Used by dart:ui's FragmentProgram, FragmentShader and
// ImageFilter.shader.

/*
#include <stdlib.h>
#include "impeller.h"

static ImpellerFragmentProgram fg_fragment_program_new(const void* bytes, size_t n) {
  // No release callback: the SDK copies the bytes.
  ImpellerMapping m = {(const uint8_t*)bytes, n, NULL};
  return ImpellerFragmentProgramNew(&m, NULL);
}
*/
import "C"

import (
	"unsafe"
)

// FragmentProgram is an ImpellerFragmentProgram: the runtime stages of a
// shader compiled by impellerc with --iplr. The SDK picks the stage of the
// context's backend when a color source or image filter is created.
type FragmentProgram struct{ h C.ImpellerFragmentProgram }

// NewFragmentProgram decodes an impellerc runtime stage bundle (the bytes
// are copied). It returns nil when the SDK refuses the data (not a runtime
// stage bundle, wrong format version, no stages).
func NewFragmentProgram(data []byte) *FragmentProgram {
	if len(data) == 0 {
		return nil
	}
	h := C.fg_fragment_program_new(unsafe.Pointer(&data[0]), C.size_t(len(data)))
	if h == nil {
		return nil
	}
	return &FragmentProgram{h}
}

// Release releases the program.
func (p *FragmentProgram) Release() { C.ImpellerFragmentProgramRelease(p.h) }

// samplersC converts textures to the SDK's sampler array (nil for none).
// Every texture must be non-nil: the SDK dereferences each one.
func samplersC(samplers []*Texture) (*C.ImpellerTexture, C.size_t) {
	if len(samplers) == 0 {
		return nil, 0
	}
	hs := make([]C.ImpellerTexture, len(samplers))
	for i, t := range samplers {
		hs[i] = t.h
	}
	return &hs[0], C.size_t(len(hs))
}

func bytesC(data []byte) (*C.uint8_t, C.size_t) {
	if len(data) == 0 {
		return nil, 0
	}
	return (*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data))
}

// NewFragmentProgramColorSource is ImpellerColorSourceCreateFragmentProgramNew:
// a color source shading each pixel with the program, given its sampler
// textures (sampled with clamp tiling and linear filtering) and its uniform
// bytes (copied). Nil when the context's backend has no stage in the
// program.
func NewFragmentProgramColorSource(ctx *Context, p *FragmentProgram, samplers []*Texture, uniforms []byte) *ColorSource {
	s, n := samplersC(samplers)
	d, dn := bytesC(uniforms)
	return colorSource(C.ImpellerColorSourceCreateFragmentProgramNew(ctx.h, p.h, s, n, d, dn))
}

// NewFragmentProgramImageFilter is ImpellerImageFilterCreateFragmentProgramNew:
// an image filter running the program on its input, which replaces the
// first sampler (whose texture is ignored but must be given). The input is
// sampled with nearest-neighbor filtering.
func NewFragmentProgramImageFilter(ctx *Context, p *FragmentProgram, samplers []*Texture, uniforms []byte) *ImageFilter {
	s, n := samplersC(samplers)
	d, dn := bytesC(uniforms)
	return imageFilter(C.ImpellerImageFilterCreateFragmentProgramNew(ctx.h, p.h, s, n, d, dn))
}
