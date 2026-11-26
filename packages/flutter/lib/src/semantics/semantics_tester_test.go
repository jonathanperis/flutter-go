package semantics_test

// Test helpers: a binding with SchedulerBinding and SemanticsBinding (the
// semantics tests' TestRenderingFlutterBinding stand-in), the isolation
// helper for tests that need a binding of their own, and assertions.

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/semantics"
	"github.com/jonathanperis/flutter-go/src/async"
)

// testBinding is BindingBase with SchedulerBinding, SemanticsBinding. Go
// can't embed both mixins (their promoted BindingBase methods would be
// ambiguous), so it embeds the scheduler mixin and forwards the
// SemanticsBinding methods to a SemanticsBindingMixin whose super
// (BindingBase) is the scheduler mixin.
type testBinding struct {
	scheduler.SchedulerBindingMixin
	sem semantics.SemanticsBindingMixin

	performedActions []ui.SemanticsActionEvent
	builderFactory   func() ui.SemanticsUpdateBuilder
}

func newTestBinding() *testBinding {
	b := &testBinding{}
	base := &foundation.BindingBaseBase{}
	b.SchedulerBindingMixin.BindingBase = base
	b.sem.BindingBase = &b.SchedulerBindingMixin
	b.SchedulerBindingMixin.SetSelf(b)
	b.sem.SetSelf(b)
	base.Construct(b)
	return b
}

func (b *testBinding) InitInstances() { b.sem.InitInstances() }

func (b *testBinding) SemanticsEnabled() bool { return b.sem.SemanticsEnabled() }
func (b *testBinding) AddSemanticsEnabledListener(l func()) {
	b.sem.AddSemanticsEnabledListener(l)
}
func (b *testBinding) RemoveSemanticsEnabledListener(l func()) {
	b.sem.RemoveSemanticsEnabledListener(l)
}
func (b *testBinding) AddSemanticsActionListener(l func(ui.SemanticsActionEvent)) {
	b.sem.AddSemanticsActionListener(l)
}
func (b *testBinding) RemoveSemanticsActionListener(l func(ui.SemanticsActionEvent)) {
	b.sem.RemoveSemanticsActionListener(l)
}
func (b *testBinding) GetRectOfSemanticsNodeInViewCoordinates(viewID, nodeID int) *ui.Rect {
	return b.sem.GetRectOfSemanticsNodeInViewCoordinates(viewID, nodeID)
}
func (b *testBinding) DebugOutstandingSemanticsHandles() int {
	return b.sem.DebugOutstandingSemanticsHandles()
}
func (b *testBinding) EnsureSemantics() *semantics.SemanticsHandle { return b.sem.EnsureSemantics() }
func (b *testBinding) PerformSemanticsAction(action ui.SemanticsActionEvent) {
	b.performedActions = append(b.performedActions, action)
}
func (b *testBinding) AccessibilityFeatures() ui.AccessibilityFeatures {
	return b.sem.AccessibilityFeatures()
}
func (b *testBinding) HandleAccessibilityFeaturesChanged() {
	b.sem.HandleAccessibilityFeaturesChanged()
}
func (b *testBinding) CreateSemanticsUpdateBuilder() ui.SemanticsUpdateBuilder {
	if b.builderFactory != nil {
		return b.builderFactory()
	}
	return b.sem.CreateSemanticsUpdateBuilder()
}
func (b *testBinding) DisableAnimations() bool { return b.sem.DisableAnimations() }

var (
	sharedBinding     *testBinding
	sharedBindingOnce sync.Once
)

// ensureBinding is TestRenderingFlutterBinding.ensureInitialized for the
// tests sharing the package's process.
func ensureBinding(t testing.TB) *testBinding {
	t.Helper()
	sharedBindingOnce.Do(func() {
		restore := async.SetCurrent(async.NewLoop())
		_ = restore
		sharedBinding = newTestBinding()
	})
	return sharedBinding
}

const isolatedEnv = "FLUTTER_GO_SEMANTICS_ISOLATED_TEST"

// isolated re-runs the test in its own process (each Dart test file has
// its own isolate, and a binding is per process); it returns true in the
// child, where the test body runs.
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

func expectEqual[T comparable](t testing.TB, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got  %v\nwant %v", got, want)
	}
}

func expectString(t testing.TB, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

var hashRE = regexp.MustCompile(`#[0-9a-f]{5}`)

// equalsIgnoringHashCodes is flutter_test's matcher.
func expectIgnoringHashCodes(t testing.TB, got, want string) {
	t.Helper()
	expectString(t, hashRE.ReplaceAllString(got, "#00000"), hashRE.ReplaceAllString(want, "#00000"))
}

// expectPanic runs f and returns what it panicked with (failing when it
// doesn't panic).
func expectPanic(t testing.TB, f func()) (r any) {
	t.Helper()
	defer func() {
		r = recover()
		if r == nil {
			t.Fatalf("expected a panic")
		}
	}()
	f()
	return nil
}

func isAssertionError(r any) bool {
	_, ok := r.(interface{ AssertionMessage() any })
	return ok
}

func ptr[T any](v T) *T { return &v }

func contains(s, sub string) bool { return strings.Contains(s, sub) }

var _ = fmt.Sprint
