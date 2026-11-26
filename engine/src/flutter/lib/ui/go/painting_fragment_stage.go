package ui

// The engine side of FragmentProgram: Impeller's runtime stage format
// (impeller/runtime_stage/runtime_stage.fbs, runtime_stage.cc and
// core/runtime_types.cc) decoded in Go, because the SDK's
// ImpellerFragmentProgram exposes no reflection. FragmentProgram reads the
// uniforms of the current backend's stage from it, as
// FragmentProgram::initFromAsset does (lib/ui/painting/fragment_program.cc).

import (
	"encoding/binary"
	"errors"
	"strconv"
)

// runtimeStageBackend is impeller::RuntimeStageBackend (std::map order).
type runtimeStageBackend int

const (
	runtimeStageBackendSkSL runtimeStageBackend = iota
	runtimeStageBackendMetal
	runtimeStageBackendOpenGLES
	runtimeStageBackendOpenGLES3
	runtimeStageBackendVulkan
)

// String is fragment_program.cc's RuntimeStageBackendToString.
func (b runtimeStageBackend) String() string {
	return [...]string{"SkSL", "Metal", "OpenGLES", "OpenGLES3", "Vulkan"}[b]
}

// engineRuntimeStageBackend is UIDartState::GetRuntimeStageBackend: the
// engine renders with Impeller on Metal.
const engineRuntimeStageBackend = runtimeStageBackendMetal

// runtimeUniformType is impeller::RuntimeUniformType.
type runtimeUniformType int

const (
	runtimeUniformTypeFloat runtimeUniformType = iota
	runtimeUniformTypeSampledImage
	runtimeUniformTypeStruct
)

// String is the "type" entry of the engine's uniform info map.
func (t runtimeUniformType) String() string {
	return [...]string{"Float", "SampledImage", "Struct"}[t]
}

type runtimeStructField struct {
	name     string
	byteSize int
}

// runtimeUniformDescription is impeller::RuntimeUniformDescription (the
// fields FragmentProgram uses).
type runtimeUniformDescription struct {
	name          string
	location      uint64
	binding       uint64
	typ           runtimeUniformType
	rows, columns uint64
	bitWidth      uint64
	arrayElements uint64
	paddingLayout []byte // 0: padding, 1: float
	structFields  []runtimeStructField
}

// dartSize is RuntimeUniformDescription::GetDartSize: the uniform's size in
// bytes as Dart sees it (floats only, no padding).
func (u *runtimeUniformDescription) dartSize() int {
	size := 0
	if len(u.paddingLayout) > 0 {
		for _, b := range u.paddingLayout {
			if b == 1 {
				size += 4
			}
		}
	} else {
		size = int(u.rows * u.columns * u.bitWidth / 8)
	}
	if u.arrayElements > 0 {
		size *= int(u.arrayElements)
	}
	return size
}

// runtimeStage is the part of impeller::RuntimeStage dart:ui reads.
type runtimeStage struct {
	entrypoint string
	uniforms   []runtimeUniformDescription
}

// runtimeStagesFormatVersion is fb::RuntimeStagesFormatVersion::kVersion.
const runtimeStagesFormatVersion = 2

// errFlatbuffer reports a buffer that fails the flatbuffers verifier.
var errFlatbuffer = errors.New("flatbuffer out of bounds")

// fbReader reads a flatbuffer, checking every access against its bounds
// (Go's stand-in for flatbuffers::Verifier: an out-of-bounds table, vector
// or string fails decoding).
type fbReader struct{ b []byte }

func (r fbReader) u8(p int) uint8 {
	if p < 0 || p+1 > len(r.b) {
		panic(errFlatbuffer)
	}
	return r.b[p]
}

func (r fbReader) u16(p int) int {
	if p < 0 || p+2 > len(r.b) {
		panic(errFlatbuffer)
	}
	return int(binary.LittleEndian.Uint16(r.b[p:]))
}

func (r fbReader) u32(p int) uint32 {
	if p < 0 || p+4 > len(r.b) {
		panic(errFlatbuffer)
	}
	return binary.LittleEndian.Uint32(r.b[p:])
}

func (r fbReader) u64(p int) uint64 {
	if p < 0 || p+8 > len(r.b) {
		panic(errFlatbuffer)
	}
	return binary.LittleEndian.Uint64(r.b[p:])
}

// field returns the position of a table's field, or -1 when absent.
func (r fbReader) field(table, index int) int {
	vtable := table - int(int32(r.u32(table)))
	vsize := r.u16(vtable)
	if 4+2*index+2 > vsize {
		return -1
	}
	off := r.u16(vtable + 4 + 2*index)
	if off == 0 {
		return -1
	}
	return table + off
}

// ref follows an offset field to its table, vector or string (-1 when the
// field is absent).
func (r fbReader) ref(table, index int) int {
	p := r.field(table, index)
	if p < 0 {
		return -1
	}
	return p + int(r.u32(p))
}

func (r fbReader) str(table, index int) string {
	p := r.ref(table, index)
	if p < 0 {
		return ""
	}
	n := int(r.u32(p))
	if p+4+n > len(r.b) || n < 0 {
		panic(errFlatbuffer)
	}
	return string(r.b[p+4 : p+4+n])
}

func (r fbReader) scalar64(table, index int) uint64 {
	p := r.field(table, index)
	if p < 0 {
		return 0
	}
	return r.u64(p)
}

func (r fbReader) scalar32(table, index int) uint32 {
	p := r.field(table, index)
	if p < 0 {
		return 0
	}
	return r.u32(p)
}

// vector returns the position of a vector's first element and its length.
func (r fbReader) vector(table, index, elemSize int) (int, int) {
	p := r.ref(table, index)
	if p < 0 {
		return -1, 0
	}
	n := int(r.u32(p))
	if n < 0 || p+4+n*elemSize > len(r.b) {
		panic(errFlatbuffer)
	}
	return p + 4, n
}

// tableAt follows the offset stored at p (a vector element or the root).
func (r fbReader) tableAt(p int) int { return p + int(r.u32(p)) }

// decodeRuntimeStages is RuntimeStage::DecodeRuntimeStages: the stage of
// each backend (nil when absent), or an error carrying absl::Status's text.
func decodeRuntimeStages(payload []byte) (stages map[runtimeStageBackend]*runtimeStage, err error) {
	if len(payload) == 0 {
		return nil, errors.New("INVALID_ARGUMENT: Payload is null or empty.")
	}
	if len(payload) < 8 || string(payload[4:8]) != "IPLR" {
		return nil, errors.New("INVALID_ARGUMENT: Payload does not have valid identifier.")
	}
	defer func() {
		if r := recover(); r != nil {
			if r != errFlatbuffer {
				panic(r)
			}
			stages, err = nil, errors.New("INVALID_ARGUMENT: Runtime stages buffer failed verification.")
		}
	}()
	r := fbReader{payload}
	root := r.tableAt(0)
	stages = map[runtimeStageBackend]*runtimeStage{}
	// Read every stage before checking the version, as the verifier checks
	// the whole buffer first.
	for b := runtimeStageBackendSkSL; b <= runtimeStageBackendVulkan; b++ {
		if t := r.ref(root, int(b)); t >= 0 {
			stages[b] = decodeRuntimeStage(r, t)
		} else {
			stages[b] = nil
		}
	}
	if version := r.scalar32(root, 5); version != runtimeStagesFormatVersion {
		return nil, errors.New("INVALID_ARGUMENT: Unsupported runtime stages format version. Expected " +
			strconv.Itoa(runtimeStagesFormatVersion) + ", got " + strconv.FormatUint(uint64(version), 10) + ".")
	}
	return stages, nil
}

// decodeRuntimeStage is RuntimeStage::Create (the reflection part).
func decodeRuntimeStage(r fbReader, t int) *runtimeStage {
	s := &runtimeStage{entrypoint: r.str(t, 1)}
	p, n := r.vector(t, 3, 4)
	for i := range n {
		u := r.tableAt(p + 4*i)
		d := runtimeUniformDescription{
			name:          r.str(u, 0),
			location:      r.scalar64(u, 1),
			binding:       r.scalar64(u, 2),
			typ:           runtimeUniformType(r.scalar32(u, 3)),
			bitWidth:      r.scalar64(u, 4),
			rows:          r.scalar64(u, 5),
			columns:       r.scalar64(u, 6),
			arrayElements: r.scalar64(u, 7),
		}
		if d.typ > runtimeUniformTypeStruct {
			panic(errFlatbuffer) // FML_UNREACHABLE in ToType
		}
		if pp, pn := r.vector(u, 8, 1); pn > 0 {
			d.paddingLayout = make([]byte, pn)
			for j := range pn {
				d.paddingLayout[j] = r.u8(pp + j)
			}
		}
		fp, fn := r.vector(u, 9, 4)
		for j := range fn {
			f := r.tableAt(fp + 4*j)
			d.structFields = append(d.structFields, runtimeStructField{name: r.str(f, 0), byteSize: int(r.scalar64(f, 1))})
		}
		s.uniforms = append(s.uniforms, d)
	}
	return s
}
