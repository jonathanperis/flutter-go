package gestures

// Port of hit_test.dart (Flutter 3.50.1). The file is hittest.go, not
// hit_test.go: Go treats files ending in _test.go as tests.

import (
	"iter"
	"math"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// HitTestable is an object that can hit-test pointers.
type HitTestable interface {
	// HitTest checks whether the given position hits this object, adding
	// the entries it hits to result.
	//
	// Deprecated: use HitTestInView and specify the view to hit test
	// (deprecated after v3.11.0-20.0.pre).
	HitTest(result *HitTestResult, position ui.Offset)
	// HitTestInView fills result with the objects hit at position in the
	// view viewID.
	HitTestInView(result *HitTestResult, position ui.Offset, viewID int)
}

// HitTestDispatcher is an object that can dispatch events.
type HitTestDispatcher interface {
	// DispatchEvent dispatches event to the targets in result's path.
	DispatchEvent(event PointerEvent, result *HitTestResult)
}

// HitTestTarget is an object that can handle events.
type HitTestTarget interface {
	// HandleEvent overrides this method to receive events.
	HandleEvent(event PointerEvent, entry HitTestEntry)
}

// NativeHitTestTarget marks a HitTestTarget backed by a native view (the
// mixin NativeHitTestTarget; embed NativeHitTestTargetMixin).
type NativeHitTestTarget interface {
	nativeHitTestTarget()
}

// NativeHitTestTargetMixin is the NativeHitTestTarget mixin, to embed.
type NativeHitTestTargetMixin struct{}

func (NativeHitTestTargetMixin) nativeHitTestTarget() {}

// HitTestEntry is data collected during a hit test about a specific
// HitTestTarget (Dart's HitTestEntry<T extends HitTestTarget>: subclasses
// such as rendering's BoxHitTestEntry embed HitTestEntryBase and narrow
// Target themselves).
type HitTestEntry interface {
	// Target is the HitTestTarget encountered during the hit test.
	Target() HitTestTarget
	// Transform describes the transform from the global coordinate system
	// to the target's, set when the entry is added to a HitTestResult.
	Transform() *vm.Matrix4
	String() string

	hitTestEntry() *HitTestEntryBase
}

// HitTestEntryBase is HitTestEntry's state, to use as is (NewHitTestEntry)
// or to embed (InitHitTestEntry).
type HitTestEntryBase struct {
	self      HitTestEntry
	target    HitTestTarget
	transform *vm.Matrix4
}

// NewHitTestEntry is HitTestEntry(target).
func NewHitTestEntry(target HitTestTarget) *HitTestEntryBase {
	e := &HitTestEntryBase{}
	e.InitHitTestEntry(e, target)
	return e
}

// InitHitTestEntry is the HitTestEntry constructor for a type embedding
// HitTestEntryBase.
func (e *HitTestEntryBase) InitHitTestEntry(self HitTestEntry, target HitTestTarget) {
	e.self = self
	e.target = target
}

func (e *HitTestEntryBase) hitTestEntry() *HitTestEntryBase { return e }

// Target is the HitTestTarget encountered during the hit test.
func (e *HitTestEntryBase) Target() HitTestTarget { return e.target }

// Transform is the transform from the global coordinate system to the
// target's (nil until the entry is added to a result).
func (e *HitTestEntryBase) Transform() *vm.Matrix4 { return e.transform }

// String is '${describeIdentity(this)}($target)'. A HitTestEntry made by
// NewHitTestEntry prints as Flutter's HitTestEntry(target) does:
// HitTestEntry<HitTestTarget>.
func (e *HitTestEntryBase) String() string {
	self := e.self
	if self == nil {
		self = e
	}
	var name string
	if _, isBase := self.(*HitTestEntryBase); isBase {
		name = "HitTestEntry<HitTestTarget>#" + foundation.ShortHash(self)
	} else {
		name = foundation.DescribeIdentity(self)
	}
	return name + "(" + dart.ToString(e.target) + ")"
}

// transformPart is _TransformPart: data that can be applied to a matrix by
// left-multiplication.
type transformPart interface {
	multiply(rhs *vm.Matrix4) *vm.Matrix4
}

// matrixTransformPart is _MatrixTransformPart.
type matrixTransformPart struct{ matrix *vm.Matrix4 }

func (p matrixTransformPart) multiply(rhs *vm.Matrix4) *vm.Matrix4 { return p.matrix.Multiplied(rhs) }

// offsetTransformPart is _OffsetTransformPart.
type offsetTransformPart struct{ offset ui.Offset }

func (p offsetTransformPart) multiply(rhs *vm.Matrix4) *vm.Matrix4 {
	m := rhs.Clone()
	m.LeftTranslateByDouble(p.offset.Dx, p.offset.Dy, 0, 1)
	return m
}

// hitTestState is a HitTestResult's lists, shared by results wrapping it
// (Dart's HitTestResult.wrap shares the list objects).
type hitTestState struct {
	path []HitTestEntry
	// A stack of transform parts, stored in two parts: transforms are
	// globalized matrices (relative to the global coordinate space),
	// localTransforms are local parts relative to the parent's coordinate
	// space, globalized and moved to transforms only when used.
	transforms      []*vm.Matrix4
	localTransforms []transformPart
}

// HitTestResult is the result of performing a hit test. Use it by pointer;
// subclasses (BoxHitTestResult...) embed it, built with
// NewHitTestResultWrap.
type HitTestResult struct {
	state *hitTestState
}

// NewHitTestResult creates an empty hit test result.
func NewHitTestResult() *HitTestResult {
	return &HitTestResult{state: &hitTestState{transforms: []*vm.Matrix4{vm.NewMatrix4Identity()}}}
}

// NewHitTestResultWrap is HitTestResult.wrap: a result sharing result's
// path and transforms (adding to one adds to the other).
func NewHitTestResultWrap(result *HitTestResult) *HitTestResult {
	return &HitTestResult{state: result.state}
}

// Path is the entries recorded during the hit test, from the most
// specific (deepest) to the least.
func (r *HitTestResult) Path() iter.Seq[HitTestEntry] {
	return func(yield func(HitTestEntry) bool) {
		for _, e := range r.state.path {
			if !yield(e) {
				return
			}
		}
	}
}

func (r *HitTestResult) globalizeTransforms() {
	s := r.state
	if len(s.localTransforms) == 0 {
		return
	}
	last := s.transforms[len(s.transforms)-1]
	for _, part := range s.localTransforms {
		last = part.multiply(last)
		s.transforms = append(s.transforms, last)
	}
	s.localTransforms = s.localTransforms[:0]
}

func (r *HitTestResult) lastTransform() *vm.Matrix4 {
	r.globalizeTransforms()
	dartAssert(len(r.state.localTransforms) == 0, nil)
	return r.state.transforms[len(r.state.transforms)-1]
}

// Add adds an entry to the end of the path, recording the current
// transform in it. Entries go from the most specific to the least.
func (r *HitTestResult) Add(entry HitTestEntry) {
	base := entry.hitTestEntry()
	dartAssert(base.transform == nil, nil)
	base.transform = r.lastTransform()
	r.state.path = append(r.state.path, entry)
}

// PushTransform pushes a new transform matrix (protected), applied to the
// entries added after it until PopTransform. Its third row and column must
// be (0, 0, 1, 0) (see PointerEventRemovePerspectiveTransform).
func (r *HitTestResult) PushTransform(transform *vm.Matrix4) {
	if foundation.KDebugMode {
		zZero := vm.NewVector4(0, 0, 1, 0)
		if !(debugVectorMoreOrLessEquals(transform.GetRow(2), zZero) &&
			debugVectorMoreOrLessEquals(transform.GetColumn(2), zZero)) {
			panic(dart.NewAssertionError("The third row and third column of a transform matrix for pointer " +
				"events must be Vector4(0, 0, 1, 0) to ensure that a transformed " +
				"point is directly under the pointing device. Did you forget to run the paint " +
				"matrix through PointerEvent.removePerspectiveTransform? " +
				"The provided matrix is:\n" + transform.String()))
		}
	}
	r.state.localTransforms = append(r.state.localTransforms, matrixTransformPart{transform})
}

// PushOffset pushes a new translation offset (protected), applied to the
// entries added after it until PopTransform.
func (r *HitTestResult) PushOffset(offset ui.Offset) {
	r.state.localTransforms = append(r.state.localTransforms, offsetTransformPart{offset})
}

// PopTransform removes the last transform added by PushTransform or
// PushOffset (protected).
func (r *HitTestResult) PopTransform() {
	s := r.state
	if len(s.localTransforms) > 0 {
		s.localTransforms = s.localTransforms[:len(s.localTransforms)-1]
	} else {
		s.transforms = s.transforms[:len(s.transforms)-1]
	}
	dartAssert(len(s.transforms) > 0, nil)
}

func debugVectorMoreOrLessEquals(a, b *vm.Vector4) bool {
	difference := a.Sub(b)
	for _, component := range difference.Storage() {
		if !(math.Abs(component) < foundation.PrecisionErrorTolerance) {
			return false
		}
	}
	return true
}

func (r *HitTestResult) String() string {
	if len(r.state.path) == 0 {
		return "HitTestResult(<empty path>)"
	}
	parts := make([]string, len(r.state.path))
	for i, e := range r.state.path {
		parts[i] = e.String()
	}
	return "HitTestResult(" + strings.Join(parts, ", ") + ")"
}
