package semantics

// Port of packages/flutter/lib/src/semantics/binding.dart (Flutter 3.50.1):
// the SemanticsBinding mixin and SemanticsHandle.

import (
	"slices"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// SemanticsBinding is the glue between the semantics layer and the Flutter
// engine (a mixin on BindingBase). A binding embeds SemanticsBindingMixin
// (see foundation/binding_mixins.go), calls SetSelf, passes it to
// Construct, and implements PerformSemanticsAction (abstract in Dart).
// SemanticsBinding uses SchedulerBinding.instance, so a binding with it
// also has the scheduler binding.
type SemanticsBinding interface {
	foundation.BindingBase

	SemanticsEnabled() bool
	AddSemanticsEnabledListener(listener func())
	RemoveSemanticsEnabledListener(listener func())
	AddSemanticsActionListener(listener func(ui.SemanticsActionEvent))
	RemoveSemanticsActionListener(listener func(ui.SemanticsActionEvent))
	GetRectOfSemanticsNodeInViewCoordinates(viewID, nodeID int) *ui.Rect
	DebugOutstandingSemanticsHandles() int
	EnsureSemantics() *SemanticsHandle
	// PerformSemanticsAction is called whenever the platform requests an
	// action to be performed on a SemanticsNode (protected, abstract).
	PerformSemanticsAction(action ui.SemanticsActionEvent)
	AccessibilityFeatures() ui.AccessibilityFeatures
	// HandleAccessibilityFeaturesChanged is called when the platform
	// accessibility features change (protected; overrides must call the
	// mixin's).
	HandleAccessibilityFeaturesChanged()
	CreateSemanticsUpdateBuilder() ui.SemanticsUpdateBuilder
	DisableAnimations() bool
}

// SemanticsBindingMixin is the SemanticsBinding mixin: embed it, call
// SetSelf with the binding and pass it to Construct (see
// foundation/binding_mixins.go).
type SemanticsBindingMixin struct {
	self SemanticsBinding

	semanticsEnabled         *foundation.ValueNotifier[bool]
	semanticsActionListeners []func(ui.SemanticsActionEvent)
	outstandingHandles       int
	semanticsHandle          *SemanticsHandle // handle for semantics request from the platform
	accessibilityFeatures    ui.AccessibilityFeatures
}

// SetSelf points the mixin at the binding it is mixed into.
func (m *SemanticsBindingMixin) SetSelf(self SemanticsBinding) { m.self = self }

func (m *SemanticsBindingMixin) outer() SemanticsBinding {
	if m.self == nil {
		panic("semantics: SemanticsBindingMixin used before SetSelf; call SetSelf with the binding when creating it")
	}
	return m.self
}

var semanticsBindingInstance SemanticsBinding

// SemanticsBindingInstance is SemanticsBinding.instance: the current
// SemanticsBinding, if one has been created.
func SemanticsBindingInstance() SemanticsBinding {
	return foundation.BindingBaseCheckInstance(semanticsBindingInstance)
}

// InitInstances sets the instance, reads the accessibility features and
// installs the platform dispatcher's semantics callbacks.
func (m *SemanticsBindingMixin) InitInstances() {
	foundation.BindingBaseSuper(m.outer(), m).InitInstances()
	semanticsBindingInstance = m.outer()
	pd := m.outer().PlatformDispatcher()
	m.accessibilityFeatures = pd.AccessibilityFeatures()
	pd.SetOnSemanticsEnabledChanged(m.handleSemanticsEnabledChanged)
	pd.SetOnSemanticsActionEvent(m.handleSemanticsActionEvent)
	pd.SetOnAccessibilityFeaturesChanged(func() {
		// TODO(chunhtai): Web should not notify accessibility feature changes
		// during updateSemantics https://github.com/flutter/flutter/issues/158399
		if scheduler.SchedulerBindingInstance().SchedulerPhase() == scheduler.SchedulerPhasePersistentCallbacks {
			scheduler.SchedulerBindingInstance().AddPostFrameCallback(func(time.Duration) {
				m.outer().HandleAccessibilityFeaturesChanged()
			}, scheduler.AddPostFrameCallbackOptions{DebugLabel: "SemanticsBinding.handleAccessibilityFeaturesChanged"})
		} else {
			m.outer().HandleAccessibilityFeaturesChanged()
		}
	})
	m.handleSemanticsEnabledChanged()
	m.AddSemanticsEnabledListener(m.handleFrameworkSemanticsEnabledChanged)
	// Ensure the initial value is set.
	if m.SemanticsEnabled() {
		m.handleFrameworkSemanticsEnabledChanged()
	}
}

// semanticsEnabledNotifier is the late final _semanticsEnabled, created on
// first use from the platform's value.
func (m *SemanticsBindingMixin) semanticsEnabledNotifier() *foundation.ValueNotifier[bool] {
	if m.semanticsEnabled == nil {
		m.semanticsEnabled = foundation.NewValueNotifier(m.outer().PlatformDispatcher().SemanticsEnabled())
	}
	return m.semanticsEnabled
}

// SemanticsEnabled is whether semantics information must be collected
// (true while a SemanticsHandle is outstanding).
func (m *SemanticsBindingMixin) SemanticsEnabled() bool {
	dartAssert(m.semanticsEnabledNotifier().Value() == (m.outstandingHandles > 0), nil)
	return m.semanticsEnabledNotifier().Value()
}

// AddSemanticsEnabledListener adds a listener called when SemanticsEnabled
// changes.
func (m *SemanticsBindingMixin) AddSemanticsEnabledListener(listener func()) {
	m.semanticsEnabledNotifier().AddListener(listener)
}

// RemoveSemanticsEnabledListener removes a listener added with
// AddSemanticsEnabledListener.
func (m *SemanticsBindingMixin) RemoveSemanticsEnabledListener(listener func()) {
	m.semanticsEnabledNotifier().RemoveListener(listener)
}

// AddSemanticsActionListener adds a listener called for every semantics
// action event (before PerformSemanticsAction).
func (m *SemanticsBindingMixin) AddSemanticsActionListener(listener func(ui.SemanticsActionEvent)) {
	m.semanticsActionListeners = append(m.semanticsActionListeners, listener)
}

// RemoveSemanticsActionListener removes a listener added with
// AddSemanticsActionListener (the first equal one, as ObserverList does).
func (m *SemanticsBindingMixin) RemoveSemanticsActionListener(listener func(ui.SemanticsActionEvent)) {
	for i, l := range m.semanticsActionListeners {
		if dart.FunctionsEqual(l, listener) {
			m.semanticsActionListeners = slices.Delete(slices.Clone(m.semanticsActionListeners), i, i+1)
			return
		}
	}
}

func (m *SemanticsBindingMixin) containsActionListener(listener func(ui.SemanticsActionEvent)) bool {
	for _, l := range m.semanticsActionListeners {
		if dart.FunctionsEqual(l, listener) {
			return true
		}
	}
	return false
}

// GetRectOfSemanticsNodeInViewCoordinates is the rect of the node nodeID
// in the view viewID, in the view's coordinates (nil: unknown; the
// rendering binding overrides it).
func (m *SemanticsBindingMixin) GetRectOfSemanticsNodeInViewCoordinates(viewID, nodeID int) *ui.Rect {
	return nil
}

// DebugOutstandingSemanticsHandles is the number of clients registered to
// listen for semantics.
func (m *SemanticsBindingMixin) DebugOutstandingSemanticsHandles() int { return m.outstandingHandles }

// EnsureSemantics creates a new SemanticsHandle and requests the
// collection of semantics information; dispose the handle when done.
func (m *SemanticsBindingMixin) EnsureSemantics() *SemanticsHandle {
	dartAssert(m.outstandingHandles >= 0, nil)
	m.outstandingHandles++
	dartAssert(m.outstandingHandles > 0, nil)
	m.semanticsEnabledNotifier().SetValue(true)
	return newSemanticsHandle(m.didDisposeSemanticsHandle)
}

func (m *SemanticsBindingMixin) didDisposeSemanticsHandle() {
	dartAssert(m.outstandingHandles > 0, nil)
	m.outstandingHandles--
	dartAssert(m.outstandingHandles >= 0, nil)
	m.semanticsEnabledNotifier().SetValue(m.outstandingHandles > 0)
}

func (m *SemanticsBindingMixin) handleSemanticsEnabledChanged() {
	if m.outer().PlatformDispatcher().SemanticsEnabled() {
		if m.semanticsHandle == nil {
			m.semanticsHandle = m.outer().EnsureSemantics()
		}
	} else {
		if m.semanticsHandle != nil {
			m.semanticsHandle.Dispose()
		}
		m.semanticsHandle = nil
	}
}

func (m *SemanticsBindingMixin) handleSemanticsActionEvent(action ui.SemanticsActionEvent) {
	decodedAction := action
	if arguments, isByteData := action.Arguments.([]byte); isByteData {
		decodedAction = action.CopyWith(ui.SemanticsActionEventCopyWithOptions{
			Arguments: services.StandardMessageCodec{}.DecodeMessage(arguments), SetArguments: true,
		})
	}
	// Listeners may get added/removed while the iteration is in progress.
	// Since the list cannot be modified while iterating, we are creating a
	// local copy for the iteration.
	localListeners := slices.Clone(m.semanticsActionListeners)
	for _, listener := range localListeners {
		if m.containsActionListener(listener) {
			listener(decodedAction)
		}
	}
	m.outer().PerformSemanticsAction(decodedAction)
}

func (m *SemanticsBindingMixin) handleFrameworkSemanticsEnabledChanged() {
	m.outer().PlatformDispatcher().SetSemanticsTreeEnabled(m.SemanticsEnabled())
}

// AccessibilityFeatures is the currently active set of
// AccessibilityFeatures.
func (m *SemanticsBindingMixin) AccessibilityFeatures() ui.AccessibilityFeatures {
	return m.accessibilityFeatures
}

// HandleAccessibilityFeaturesChanged is called when the platform
// accessibility features change; overrides must call this one.
func (m *SemanticsBindingMixin) HandleAccessibilityFeaturesChanged() {
	m.accessibilityFeatures = m.outer().PlatformDispatcher().AccessibilityFeatures()
}

// CreateSemanticsUpdateBuilder creates an empty semantics update builder
// (tests override it to record the updates).
func (m *SemanticsBindingMixin) CreateSemanticsUpdateBuilder() ui.SemanticsUpdateBuilder {
	return ui.NewSemanticsUpdateBuilder()
}

// DisableAnimations is whether the platform requests that animations be
// disabled (DebugSemanticsDisableAnimations overrides it in debug builds).
func (m *SemanticsBindingMixin) DisableAnimations() bool {
	value := m.accessibilityFeatures.DisableAnimations()
	if foundation.KDebugMode && DebugSemanticsDisableAnimations != nil {
		value = *DebugSemanticsDisableAnimations
	}
	return value
}

// SemanticsHandle is a reference to the semantics information generated
// by the framework: semantics are collected while a handle is
// outstanding. Dispose it when done.
type SemanticsHandle struct {
	onDispose func()
}

func newSemanticsHandle(onDispose func()) *SemanticsHandle {
	h := &SemanticsHandle{onDispose: onDispose}
	if foundation.KDebugMode {
		foundation.DebugMaybeDispatchCreated("semantics", "SemanticsHandle", h)
	}
	return h
}

// Dispose closes the semantics request.
func (h *SemanticsHandle) Dispose() {
	if foundation.KDebugMode {
		foundation.DebugMaybeDispatchDisposed(h)
	}
	h.onDispose()
}
