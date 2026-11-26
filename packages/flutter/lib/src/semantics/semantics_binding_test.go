package semantics_test

// Port of packages/flutter/test/semantics/semantics_binding_test.dart,
// semantics_binding_set_semantics_tree_enabled_test.dart and
// semantics_binding_set_semantics_tree_enabled_1_test.dart (Flutter
// 3.50.1). Each test builds its own binding, in its own process.
//
// Adapted: flutter_test's TestPlatformDispatcher (semanticsEnabledTestValue
// and the setSemanticsTreeEnabled spy) is not ported; the platform's
// semantics setting is changed through the engine entry point
// (PlatformConfiguration.UpdateSemanticsEnabled, hooks.dart's
// _updateSemanticsEnabled, which is what the test value simulates), and
// setSemanticsTreeEnabled is observed through the engine's record of it
// (ui.SemanticsTreeEnabled) and a PlatformConfigurationSemanticsClient.
//
// Skipped (they need widgets/rendering: WidgetTester, pipelineOwner,
// pumpWidget, Semantics widgets):
//   - "SemanticsBinding.ensureSemantics triggers creation of semantics
//     owner."
//   - "getRectOfSemanticsNodeInViewCoordinates asserts for unknown views
//     and nodes"
//   - "getRectOfSemanticsNodeInViewCoordinates returns transformed logical
//     rect"
// "SemanticsHandle dispatches memory events" checks the dispatch through
// FlutterMemoryAllocations (leak_tracker's memoryEvents is not ported).

import (
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/semantics"
	"github.com/jonathanperis/flutter-go/src/async"
)

// semanticsClient records the natives a PlatformConfigurationSemanticsClient
// receives.
type semanticsClient struct {
	treeEnabled []bool
	updates     []ui.SemanticsUpdate
}

func (c *semanticsClient) ScheduleFrame()                 {}
func (c *semanticsClient) EndWarmUpFrame()                {}
func (c *semanticsClient) SetNeedsReportTimings(bool)     {}
func (c *semanticsClient) SetSemanticsTreeEnabled(e bool) { c.treeEnabled = append(c.treeEnabled, e) }
func (c *semanticsClient) UpdateSemantics(viewID int, update ui.SemanticsUpdate) {
	c.updates = append(c.updates, update)
}

func withEngine(t *testing.T) (*ui.PlatformConfiguration, *semanticsClient) {
	t.Helper()
	t.Cleanup(async.SetCurrent(async.NewLoop()))
	client := &semanticsClient{}
	pc := ui.NewPlatformConfiguration(client)
	t.Cleanup(ui.SetPlatformConfiguration(pc))
	return pc, client
}

func TestListenersCalledWhenSemanticsTurnedOnWithEnsureSemantics(t *testing.T) {
	if !isolated(t) {
		return
	}
	pc, _ := withEngine(t)
	newTestBinding()
	binding := semantics.SemanticsBindingInstance()
	expectEqual(t, binding.SemanticsEnabled(), false)

	var status []bool
	listener := func() { status = append(status, semantics.SemanticsBindingInstance().SemanticsEnabled()) }
	binding.AddSemanticsEnabledListener(listener)
	expectEqual(t, binding.SemanticsEnabled(), false)

	handle1 := binding.EnsureSemantics()
	expectEqual(t, len(status), 1)
	expectEqual(t, status[0], true)
	expectEqual(t, binding.SemanticsEnabled(), true)
	status = nil

	handle2 := binding.EnsureSemantics()
	expectEqual(t, len(status), 0) // Listener didn't fire again.
	expectEqual(t, binding.SemanticsEnabled(), true)

	pd := binding.PlatformDispatcher()
	expectEqual(t, pd.SemanticsEnabled(), false)
	pc.UpdateSemanticsEnabled(true)
	expectEqual(t, pd.SemanticsEnabled(), true)
	pc.UpdateSemanticsEnabled(false)
	expectEqual(t, pd.SemanticsEnabled(), false)
	expectEqual(t, len(status), 0) // Listener didn't fire again.
	expectEqual(t, binding.SemanticsEnabled(), true)

	handle1.Dispose()
	expectEqual(t, len(status), 0) // Listener didn't fire.
	expectEqual(t, binding.SemanticsEnabled(), true)

	handle2.Dispose()
	expectEqual(t, len(status), 1)
	expectEqual(t, status[0], false)
	expectEqual(t, binding.SemanticsEnabled(), false)
}

func TestListenersCalledWhenSemanticsTurnedOnByPlatform(t *testing.T) {
	if !isolated(t) {
		return
	}
	pc, _ := withEngine(t)
	newTestBinding()
	binding := semantics.SemanticsBindingInstance()
	expectEqual(t, binding.SemanticsEnabled(), false)

	var status []bool
	binding.AddSemanticsEnabledListener(func() {
		status = append(status, semantics.SemanticsBindingInstance().SemanticsEnabled())
	})
	expectEqual(t, binding.SemanticsEnabled(), false)

	pc.UpdateSemanticsEnabled(true)
	expectEqual(t, len(status), 1)
	expectEqual(t, status[0], true)
	expectEqual(t, binding.SemanticsEnabled(), true)
	status = nil

	handle := binding.EnsureSemantics()
	handle.Dispose()
	expectEqual(t, len(status), 0) // Listener didn't fire.
	expectEqual(t, binding.SemanticsEnabled(), true)

	pc.UpdateSemanticsEnabled(false)
	expectEqual(t, len(status), 1)
	expectEqual(t, status[0], false)
	expectEqual(t, binding.SemanticsEnabled(), false)
}

func TestSemanticsHandleDispatchesMemoryEvents(t *testing.T) {
	if !isolated(t) {
		return
	}
	withEngine(t)
	newTestBinding()
	var events []foundation.ObjectEvent
	listener := func(e foundation.ObjectEvent) {
		if _, ok := e.Object().(*semantics.SemanticsHandle); ok {
			events = append(events, e)
		}
	}
	foundation.FlutterMemoryAllocationsInstance.AddListener(listener)
	defer foundation.FlutterMemoryAllocationsInstance.RemoveListener(listener)
	semantics.SemanticsBindingInstance().EnsureSemantics().Dispose()
	expectEqual(t, len(events), 2)
	_, created := events[0].(*foundation.ObjectCreated)
	_, disposed := events[1].(*foundation.ObjectDisposed)
	expectEqual(t, created && disposed, true)
}

// semantics_binding_set_semantics_tree_enabled_test.dart
func TestEnsureSemanticsCallsSetSemanticsTreeEnabled(t *testing.T) {
	if !isolated(t) {
		return
	}
	_, client := withEngine(t)
	b := newTestBinding()
	expectEqual(t, ui.SemanticsTreeEnabled(), false)
	handle := b.EnsureSemantics()
	expectEqual(t, ui.SemanticsTreeEnabled(), true)
	handle.Dispose()
	expectEqual(t, ui.SemanticsTreeEnabled(), false)
	expectEqual(t, len(client.treeEnabled), 2)
	expectEqual(t, client.treeEnabled[0], true)
	expectEqual(t, client.treeEnabled[1], false)
}

// semantics_binding_set_semantics_tree_enabled_1_test.dart
func TestSetSemanticsTreeEnabledWhenEnabledBeforeBindingInitialized(t *testing.T) {
	if !isolated(t) {
		return
	}
	pc, client := withEngine(t)
	pc.UpdateSemanticsEnabled(true)
	newTestBinding()
	expectEqual(t, ui.SemanticsTreeEnabled(), true)
	expectEqual(t, len(client.treeEnabled) > 0 && client.treeEnabled[len(client.treeEnabled)-1], true)
}

// Own tests: the action event dispatch, its ByteData decoding and the
// action listeners; accessibility features (and their post-frame update
// during persistent callbacks); disableAnimations.
func TestSemanticsActionEventDispatch(t *testing.T) {
	if !isolated(t) {
		return
	}
	pc, _ := withEngine(t)
	b := newTestBinding()
	var heard []any
	listener := func(e ui.SemanticsActionEvent) { heard = append(heard, e.Arguments) }
	b.AddSemanticsActionListener(listener)
	var removed bool
	var selfRemoving func(ui.SemanticsActionEvent)
	selfRemoving = func(ui.SemanticsActionEvent) {
		removed = true
		b.RemoveSemanticsActionListener(listener)
	}
	b.AddSemanticsActionListener(selfRemoving)
	var late []any
	lateListener := func(e ui.SemanticsActionEvent) { late = append(late, e.Arguments) }
	b.AddSemanticsActionListener(lateListener)
	b.AddSemanticsActionListener(func(ui.SemanticsActionEvent) { b.RemoveSemanticsActionListener(lateListener) })
	// Added after the remover: removed during the dispatch, so skipped.
	later := func(e ui.SemanticsActionEvent) { late = append(late, "later") }
	b.AddSemanticsActionListener(func(ui.SemanticsActionEvent) { b.RemoveSemanticsActionListener(later) })
	b.AddSemanticsActionListener(later)

	args := (semanticsCodec{}).encode(true)
	pc.DispatchSemanticsAction(0, 7, ui.SemanticsActionMoveCursorForwardByCharacter.Index(), args)
	expectEqual(t, len(b.performedActions), 1)
	got := b.performedActions[0]
	expectEqual(t, got.Type, ui.SemanticsActionMoveCursorForwardByCharacter)
	expectEqual(t, got.NodeID, 7)
	expectEqual(t, got.Arguments, any(true)) // decoded with StandardMessageCodec
	expectEqual(t, len(heard), 1)
	expectEqual(t, removed, true)
	expectEqual(t, len(late), 1) // lateListener ran; later was removed first

	pc.DispatchSemanticsAction(0, 8, ui.SemanticsActionTap.Index(), nil)
	expectEqual(t, len(heard), 1) // removed during the previous dispatch
	expectEqual(t, b.performedActions[1].Arguments, nil)
}

// The platform's request holds one handle however often it is repeated.
func TestPlatformSemanticsHandleHeldOnce(t *testing.T) {
	if !isolated(t) {
		return
	}
	pc, _ := withEngine(t)
	b := newTestBinding()
	pc.UpdateSemanticsEnabled(true)
	b.PlatformDispatcher().OnSemanticsEnabledChanged()()
	expectEqual(t, b.DebugOutstandingSemanticsHandles(), 1)
	pc.UpdateSemanticsEnabled(false)
	expectEqual(t, b.DebugOutstandingSemanticsHandles(), 0)
	expectEqual(t, b.SemanticsEnabled(), false)
}

func TestAccessibilityFeaturesAndDisableAnimations(t *testing.T) {
	if !isolated(t) {
		return
	}
	pc, _ := withEngine(t)
	pc.UpdateAccessibilityFeatures(1 << 2) // disableAnimations
	b := newTestBinding()
	expectEqual(t, b.AccessibilityFeatures().DisableAnimations(), true)
	expectEqual(t, b.DisableAnimations(), true)
	semantics.DebugSemanticsDisableAnimations = ptr(false)
	expectEqual(t, b.DisableAnimations(), false)
	semantics.DebugSemanticsDisableAnimations = nil

	pc.UpdateAccessibilityFeatures(1 << 3) // boldText
	expectEqual(t, b.AccessibilityFeatures().BoldText(), true)
	expectEqual(t, b.AccessibilityFeatures().String(),
		"AccessibilityFeatures[boldText, supportsAnnounce, autoPlayAnimatedImages, autoPlayVideos]")

	// During persistent callbacks the update waits for the post-frame
	// callbacks.
	var during, after bool
	b.AddPersistentFrameCallback(func(time.Duration) {
		if b.AccessibilityFeatures().HighContrast() {
			return
		}
		pc.UpdateAccessibilityFeatures(1 << 5) // highContrast
		during = b.AccessibilityFeatures().HighContrast()
	})
	b.AddPostFrameCallback(func(time.Duration) {})
	b.HandleBeginFrame(ptr(time.Second))
	b.HandleDrawFrame()
	after = b.AccessibilityFeatures().HighContrast()
	expectEqual(t, during, false)
	expectEqual(t, after, true)
}
