package services

// Partial port of binding.dart (Flutter 3.50.1): ServicesBinding's
// messaging (defaultBinaryMessenger, createBinaryMessenger, channelBuffers,
// rootIsolateToken, _DefaultBinaryMessenger), the system, accessibility,
// lifecycle and platform channel handlers (memory pressure, accessibility
// focus, lifecycle transitions, app exit requests, system UI changes, the
// system context menu client), exitApplication, initializationComplete and
// the service extensions.
//
// The keyboard (HardwareKeyboard, KeyEventManager, the key event channel)
// comes with hardware_keyboard.go. Not ported yet (they come with their
// libraries): the restoration manager, TextInput.ensureInitialized, the licenses read from
// the asset bundle (initLicenses), the root bundle's clear/evict
// (handleMemoryPressure and evict do nothing else), SystemChrome's
// lifecycle hook, and PlatformDispatcher.onViewFocusChange /
// handleViewFocusChanged.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// ServicesBinding listens for platform messages and directs them to the
// DefaultBinaryMessenger; it also handles the system, accessibility,
// lifecycle and platform channels (mixin ServicesBinding on BindingBase,
// SchedulerBinding; partial, see the file comment).
type ServicesBinding interface {
	scheduler.SchedulerBinding

	// DefaultBinaryMessenger is the default instance of BinaryMessenger,
	// used to send messages from the application to the platform and keep
	// track of the handlers registered on each channel.
	DefaultBinaryMessenger() BinaryMessenger
	// ChannelBuffers is the buffers holding platform messages until their
	// handlers are registered (ui.ChannelBuffersInstance()).
	ChannelBuffers() *ui.ChannelBuffers
	// CreateBinaryMessenger creates the default BinaryMessenger (called
	// once, by InitInstances; bindings override it, as flutter_test does).
	CreateBinaryMessenger() BinaryMessenger
	// HandleMemoryPressure is called when the operating system notifies the
	// application of a memory pressure situation; overrides must call it.
	HandleMemoryPressure()
	// HandleSystemMessage handles a message of the system channel (memory
	// pressure); overrides must call it.
	HandleSystemMessage(systemMessage any) async.Future[async.Void]
	// Evict is called in response to the ext.flutter.evict service
	// extension; overrides must call it.
	Evict(asset string)
	// ReadInitialLifecycleStateFromNativeWindow initializes LifecycleState
	// from PlatformDispatcher.InitialLifecycleState, unless it is already
	// set or the initial state is empty.
	ReadInitialLifecycleStateFromNativeWindow()
	// AccessibilityFocus is the id of the semantics node that has
	// accessibility focus on the platform (nil when none).
	AccessibilityFocus() *foundation.ValueNotifier[*int]
	// HandleRequestAppExit is called when the platform asks to exit the
	// application (WidgetsBinding answers with the observers' decision);
	// the default allows it.
	HandleRequestAppExit() async.Future[ui.AppExitResponse]
	// ExitApplication asks the platform to exit the application (Dart's
	// [exitCode = 0] is optional).
	ExitApplication(exitType ui.AppExitType, exitCode ...int) async.Future[ui.AppExitResponse]
	// SetSystemUiChangeCallback sets the callback for changes in the
	// visibility of the system UI overlays (Android's fullscreen modes).
	SetSystemUiChangeCallback(callback SystemUiChangeCallback)
	// InitializationComplete tells the platform that the framework has
	// initialized (it may then remove the splash screen).
	InitializationComplete() async.Future[async.Void]

	// Keyboard is the global HardwareKeyboard (HardwareKeyboard.instance).
	Keyboard() *HardwareKeyboard
	// KeyEventManager dispatches key messages (deprecated).
	KeyEventManager() *KeyEventManager

	servicesBinding() *ServicesBindingMixin
}

// SystemUiChangeCallback is the signature of the callback for changes in
// the visibility of the system UI overlays.
type SystemUiChangeCallback func(systemOverlaysAreVisible bool) async.Future[async.Void]

// SystemContextMenuClient is notified about the system context menu
// (mixin SystemContextMenuClient).
type SystemContextMenuClient interface {
	// HandleSystemHide is called when the system hides its context menu.
	HandleSystemHide()
	// HandleCustomContextMenuAction is called when a custom item of the
	// system context menu is tapped.
	HandleCustomContextMenuAction(actionID string)
}

// ServicesBindingMixin is ServicesBinding's state and methods, to embed in
// a binding after foundation.BindingBaseBase and scheduler's
// SchedulerBindingMixin (Dart: `on BindingBase, SchedulerBinding`; see
// foundation/binding_mixins.go):
//
//	type MyBinding struct {
//		foundation.BindingBaseBase
//		scheduler.SchedulerBindingMixin
//		services.ServicesBindingMixin
//	}
//
//	func NewMyBinding() *MyBinding {
//		b := &MyBinding{}
//		b.SchedulerBindingMixin.SetSelf(b)
//		b.ServicesBindingMixin.SetSelf(b)
//		b.Construct(b, &b.SchedulerBindingMixin, &b.ServicesBindingMixin)
//		return b
//	}
//
//	func (b *MyBinding) InitInstances()         { b.ServicesBindingMixin.InitInstances() }
//	func (b *MyBinding) InitServiceExtensions() { b.ServicesBindingMixin.InitServiceExtensions() }
//	func (b *MyBinding) Unlocked()              { b.SchedulerBindingMixin.Unlocked() }
//
// ServicesBinding's channel handlers are registered in InitInstances, and
// ChannelBuffers drains the messages buffered meanwhile in a microtask:
// flush the event loop after constructing the binding.
type ServicesBindingMixin struct {
	self ServicesBinding

	defaultBinaryMessenger  BinaryMessenger
	accessibilityFocus      *foundation.ValueNotifier[*int]
	systemUiChangeCallback  SystemUiChangeCallback
	systemContextMenuClient SystemContextMenuClient
	keyboard                *HardwareKeyboard
	keyEventManager         *KeyEventManager
}

// SetSelf points the mixin at the binding embedding it, whose methods it
// calls so overrides run. Call it once, before constructing the binding.
func (m *ServicesBindingMixin) SetSelf(self ServicesBinding) {
	m.self = self
	m.accessibilityFocus = foundation.NewValueNotifier[*int](nil)
}

func (m *ServicesBindingMixin) outer() ServicesBinding {
	if m.self == nil {
		panic("services: ServicesBindingMixin used before SetSelf; call SetSelf with the binding when creating it")
	}
	return m.self
}

func (m *ServicesBindingMixin) servicesBinding() *ServicesBindingMixin { return m }

var servicesBindingInstance ServicesBinding

// ServicesBindingInstance is ServicesBinding.instance: the current
// ServicesBinding (it panics with an explanation if none was created).
func ServicesBindingInstance() ServicesBinding {
	return foundation.BindingBaseCheckInstance(servicesBindingInstance)
}

func (m *ServicesBindingMixin) InitInstances() {
	foundation.BindingBaseSuper(m.outer(), m).InitInstances()
	self := m.outer()
	servicesBindingInstance = self
	m.defaultBinaryMessenger = self.CreateBinaryMessenger()
	m.initKeyboard()
	// The handlers' Future<void> results are encoded as null.
	SystemChannels.System.SetMessageHandler(func(message any) async.Future[any] {
		return voidToNull(self.HandleSystemMessage(message))
	})
	SystemChannels.Accessibility.SetMessageHandler(func(message any) async.Future[any] {
		return voidToNull(m.handleAccessibilityMessage(message))
	})
	SystemChannels.Lifecycle.SetMessageHandler(m.handleLifecycleMessage)
	SystemChannels.Platform.SetMethodCallHandler(m.handlePlatformMessage)
	self.ReadInitialLifecycleStateFromNativeWindow()
	self.InitializationComplete()
}

// voidToNull is a Future<void> used where Dart expects a Future<Object?>:
// its value is null.
func voidToNull(f async.Future[async.Void]) async.Future[any] {
	return async.Async(func() any {
		f.Await()
		return nil
	})
}

func (m *ServicesBindingMixin) Keyboard() *HardwareKeyboard { return m.keyboard }

func (m *ServicesBindingMixin) KeyEventManager() *KeyEventManager { return m.keyEventManager }

// initKeyboard is _initKeyboard: the key handlers are installed once the
// platform has answered getKeyboardState.
func (m *ServicesBindingMixin) initKeyboard() {
	m.keyboard = NewHardwareKeyboard()
	m.keyEventManager = NewKeyEventManager(m.keyboard, RawKeyboardInstance)
	async.OnComplete(m.keyboard.SyncKeyboardState(), func(async.Void) {
		m.outer().PlatformDispatcher().SetOnKeyData(m.keyEventManager.HandleKeyData)
		SystemChannels.KeyEvent.SetMessageHandler(func(message any) async.Future[any] {
			return async.Then(m.keyEventManager.HandleRawKeyMessage(message), func(r map[string]any) any { return r })
		})
	}, nil)
}

func (m *ServicesBindingMixin) DefaultBinaryMessenger() BinaryMessenger {
	return m.defaultBinaryMessenger
}

// ServicesBindingRootIsolateToken is ServicesBinding.rootIsolateToken: the
// RootIsolateToken of the current isolate (never nil in Go, see
// ui.RootIsolateTokenInstance).
func ServicesBindingRootIsolateToken() *ui.RootIsolateToken { return ui.RootIsolateTokenInstance() }

func (m *ServicesBindingMixin) ChannelBuffers() *ui.ChannelBuffers {
	return ui.ChannelBuffersInstance()
}

func (m *ServicesBindingMixin) CreateBinaryMessenger() BinaryMessenger {
	return defaultBinaryMessenger{}
}

func (m *ServicesBindingMixin) HandleMemoryPressure() {
	// rootBundle.clear(): the asset bundle is not ported yet.
}

func (m *ServicesBindingMixin) HandleSystemMessage(systemMessage any) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		message := systemMessage.(map[string]any)
		typ := message["type"].(string)
		switch typ {
		case "memoryPressure":
			m.outer().HandleMemoryPressure()
		}
	})
}

func (m *ServicesBindingMixin) InitServiceExtensions() {
	foundation.BindingBaseSuper(m.outer(), m).InitServiceExtensions()
	self := m.outer()
	if foundation.KDebugMode {
		self.RegisterStringServiceExtension(ServicesServiceExtensionsEvict.Name(),
			func() async.Future[string] { return async.Async(func() string { return "" }) },
			func(value string) async.Future[async.Void] {
				return async.AsyncVoid(func() { self.Evict(value) })
			})
	}
	if !foundation.KReleaseMode {
		self.RegisterBoolServiceExtension(ServicesServiceExtensionsProfilePlatformChannels.Name(),
			func() async.Future[bool] { return async.Async(func() bool { return DebugProfilePlatformChannels }) },
			func(value bool) async.Future[async.Void] {
				return async.AsyncVoid(func() { DebugProfilePlatformChannels = value })
			})
	}
}

func (m *ServicesBindingMixin) Evict(asset string) {
	// rootBundle.evict(asset): the asset bundle is not ported yet.
}

func (m *ServicesBindingMixin) ReadInitialLifecycleStateFromNativeWindow() {
	self := m.outer()
	if self.LifecycleState() != nil || self.PlatformDispatcher().InitialLifecycleState() == "" {
		return
	}
	initial := self.PlatformDispatcher().InitialLifecycleState()
	m.handleLifecycleMessage(&initial)
}

func (m *ServicesBindingMixin) handleLifecycleMessage(message *string) async.Future[*string] {
	return async.Async(func() *string {
		if message == nil {
			panic(dart.TypeError{Message: "Null check operator used on a null value"})
		}
		state := parseAppLifecycleMessage(*message)
		if state == nil {
			panic(dart.TypeError{Message: "Null check operator used on a null value"})
		}
		generated := m.generateStateTransitions(m.outer().LifecycleState(), *state)
		for _, stateChange := range generated {
			m.outer().HandleAppLifecycleStateChanged(stateChange)
			// SystemChrome.handleAppLifecycleStateChanged: not ported yet.
		}
		return nil
	})
}

func (m *ServicesBindingMixin) generateStateTransitions(previousState *ui.AppLifecycleState, state ui.AppLifecycleState) []ui.AppLifecycleState {
	if previousState != nil && *previousState == state {
		return []ui.AppLifecycleState{}
	}
	stateChanges := []ui.AppLifecycleState{}
	if previousState == nil {
		// If there was no previous state, just jump directly to the new
		// state.
		stateChanges = append(stateChanges, state)
	} else {
		previousStateIndex := int(*previousState)
		stateIndex := int(state)
		if state == ui.AppLifecycleStateDetached {
			for i := previousStateIndex + 1; i < len(ui.AppLifecycleStateValues); i++ {
				stateChanges = append(stateChanges, ui.AppLifecycleStateValues[i])
			}
			stateChanges = append(stateChanges, ui.AppLifecycleStateDetached)
		} else if previousStateIndex > stateIndex {
			for i := stateIndex; i < previousStateIndex; i++ {
				stateChanges = append([]ui.AppLifecycleState{ui.AppLifecycleStateValues[i]}, stateChanges...)
			}
		} else {
			for i := previousStateIndex + 1; i <= stateIndex; i++ {
				stateChanges = append(stateChanges, ui.AppLifecycleStateValues[i])
			}
		}
	}
	if foundation.KDebugMode {
		starting := previousState
		for _, ending := range stateChanges {
			if !debugVerifyLifecycleChange(starting, ending) {
				panic(dart.NewAssertionError("Invalid lifecycle state transition generated from " +
					lifecycleString(previousState) + " to " + state.String() + " (generated " + dart.ToString(stateChanges) + ")"))
			}
			ending := ending
			starting = &ending
		}
	}
	return stateChanges
}

func lifecycleString(s *ui.AppLifecycleState) string {
	if s == nil {
		return "null"
	}
	return s.String()
}

func debugVerifyLifecycleChange(starting *ui.AppLifecycleState, ending ui.AppLifecycleState) bool {
	if starting == nil {
		// Any transition from null is fine, since it is initializing the
		// state.
		return true
	}
	if *starting == ending {
		// Any transition to itself shouldn't happen.
		return false
	}
	switch *starting {
	case ui.AppLifecycleStateResumed:
		// Can't go from resumed to detached directly (must go through
		// paused).
		return ending == ui.AppLifecycleStateInactive
	case ui.AppLifecycleStateDetached:
		return ending == ui.AppLifecycleStateResumed || ending == ui.AppLifecycleStatePaused
	case ui.AppLifecycleStateInactive:
		return ending == ui.AppLifecycleStateResumed || ending == ui.AppLifecycleStateHidden
	case ui.AppLifecycleStateHidden:
		return ending == ui.AppLifecycleStatePaused || ending == ui.AppLifecycleStateInactive
	default: // paused
		return ending == ui.AppLifecycleStateHidden || ending == ui.AppLifecycleStateDetached
	}
}

func (m *ServicesBindingMixin) AccessibilityFocus() *foundation.ValueNotifier[*int] {
	return m.accessibilityFocus
}

func (m *ServicesBindingMixin) handleAccessibilityMessage(accessibilityMessage any) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		message := accessibilityMessage.(map[any]any)
		typ := message["type"].(string)
		switch typ {
		case "didGainFocus":
			nodeID := message["nodeId"].(int)
			m.accessibilityFocus.SetValue(&nodeID)
		}
	})
}

func (m *ServicesBindingMixin) handlePlatformMessage(methodCall MethodCall) async.Future[any] {
	return async.Async(func() any {
		method := methodCall.Method
		switch method {
		// Called when the system dismisses the system context menu, such as
		// when the user taps outside the menu. Not called when Flutter shows
		// a new system context menu while an old one is still visible.
		case "ContextMenu.onDismissSystemContextMenu":
			if m.systemContextMenuClient == nil {
				panic(dart.NewAssertionError("Platform sent onDismissSystemContextMenu when no SystemContextMenuClient was registered."))
			}
			m.systemContextMenuClient.HandleSystemHide()
			m.systemContextMenuClient = nil
		case "ContextMenu.onPerformCustomAction":
			if m.systemContextMenuClient == nil {
				panic(dart.NewAssertionError("Platform sent onPerformCustomAction when no SystemContextMenuClient was registered. " +
					"ServicesBinding.systemContextMenuClient shouldn't be cleared unless the menu is hidden."))
			}
			args := methodCall.Arguments.([]any)
			callbackID := args[1].(string)
			m.systemContextMenuClient.HandleCustomContextMenuAction(callbackID)
		case "SystemChrome.systemUIChange":
			args := methodCall.Arguments.([]any)
			if m.systemUiChangeCallback != nil {
				awaitOrNull(m.systemUiChangeCallback(args[0].(bool)))
			}
		case "System.requestAppExit":
			return map[string]any{"response": m.outer().HandleRequestAppExit().Await().Name()}
		default:
			panic(dart.NewAssertionError(`Method "` + method + `" not handled.`))
		}
		return nil
	})
}

func parseAppLifecycleMessage(message string) *ui.AppLifecycleState {
	var s ui.AppLifecycleState
	switch message {
	case "AppLifecycleState.resumed":
		s = ui.AppLifecycleStateResumed
	case "AppLifecycleState.inactive":
		s = ui.AppLifecycleStateInactive
	case "AppLifecycleState.hidden":
		s = ui.AppLifecycleStateHidden
	case "AppLifecycleState.paused":
		s = ui.AppLifecycleStatePaused
	case "AppLifecycleState.detached":
		s = ui.AppLifecycleStateDetached
	default:
		return nil
	}
	return &s
}

func (m *ServicesBindingMixin) HandleRequestAppExit() async.Future[ui.AppExitResponse] {
	return async.Async(func() ui.AppExitResponse { return ui.AppExitResponseExit })
}

func (m *ServicesBindingMixin) ExitApplication(exitType ui.AppExitType, exitCode ...int) async.Future[ui.AppExitResponse] {
	code := 0
	if len(exitCode) > 0 {
		code = exitCode[0]
	}
	return async.Async(func() ui.AppExitResponse {
		result := SystemChannels.Platform.InvokeMethod("System.exitApplication",
			[]dart.MapEntry[string, any]{{Key: "type", Value: exitType.Name()}, {Key: "exitCode", Value: code}}).Await()
		if result == nil {
			return ui.AppExitResponseCancel
		}
		switch castElement[map[string]any](result)["response"] {
		case "cancel":
			return ui.AppExitResponseCancel
		default:
			// In practice, this will never get returned, because the
			// application will have exited before it returns.
			return ui.AppExitResponseExit
		}
	})
}

func (m *ServicesBindingMixin) SetSystemUiChangeCallback(callback SystemUiChangeCallback) {
	m.systemUiChangeCallback = callback
}

func (m *ServicesBindingMixin) InitializationComplete() async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.Platform.InvokeMethod("System.initializationComplete").Await()
	})
}

// ServicesBindingSetSystemContextMenuClient is the static setter
// ServicesBinding.systemContextMenuClient: the client notified about the
// system context menu (nil when none is shown).
func ServicesBindingSetSystemContextMenuClient(client SystemContextMenuClient) {
	ServicesBindingInstance().servicesBinding().systemContextMenuClient = client
}

// defaultBinaryMessenger is _DefaultBinaryMessenger: messages go to the
// engine (ui.PlatformDispatcher.SendPlatformMessage), handlers are
// ChannelBuffers listeners.
type defaultBinaryMessenger struct{}

func (defaultBinaryMessenger) HandlePlatformMessage(channel string, message []byte, callback ui.PlatformMessageResponseCallback) async.Future[[]byte] {
	// Dart's Future<void> completes with null.
	return async.Async(func() []byte {
		ui.ChannelBuffersInstance().Push(channel, message, func(data []byte) {
			if callback != nil {
				callback(data)
			}
		})
		return nil
	})
}

func (defaultBinaryMessenger) Send(channel string, message []byte) async.Future[[]byte] {
	completer := async.NewCompleter[[]byte]()
	// ui.PlatformDispatcherInstance() is used directly instead of
	// ServicesBinding.instance.platformDispatcher because this method might
	// be invoked before any binding is initialized.
	ui.PlatformDispatcherInstance().SendPlatformMessage(channel, message, func(reply []byte) {
		defer func() {
			if exception := recover(); exception != nil {
				stack := dart.StackTraceFromPanic()
				library := "services library"
				foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
					Stack:   stack,
					Library: &library,
					Context: foundation.NewErrorDescription("during a platform message response callback"),
				}))
			}
		}()
		completer.Complete(reply)
	})
	return completer.Future()
}

func (defaultBinaryMessenger) SetMessageHandler(channel string, handler MessageHandler) {
	if handler == nil {
		ui.ChannelBuffersInstance().ClearListener(channel)
		return
	}
	ui.ChannelBuffersInstance().SetListener(channel, func(data []byte, callback ui.PlatformMessageResponseCallback) {
		async.AsyncVoid(func() {
			var response []byte
			defer func() { callback(response) }()
			func() {
				defer func() {
					if exception := recover(); exception != nil {
						stack := dart.StackTraceFromPanic()
						library := "services library"
						foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
							Stack:   stack,
							Library: &library,
							Context: foundation.NewErrorDescription("during a platform message callback"),
						}))
					}
				}()
				response = awaitOrNull(handler(data))
			}()
		})
	})
}
