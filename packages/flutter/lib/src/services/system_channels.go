package services

// Port of system_channels.dart (Flutter 3.50.1): the platform channels
// Flutter uses to talk to the engine and the host platform.

// systemChannels is the type of SystemChannels (Dart's abstract final
// class of static constants).
type systemChannels struct {
	// Navigation: "flutter/navigation", JSON (popRoute, pushRoute,
	// pushRouteInformation; routeUpdated, routeInformationUpdated...).
	Navigation *MethodChannel
	// BackGesture: "flutter/backgesture", standard codec (predictive back:
	// startBackGesture, updateBackGestureProgress, commitBackGesture,
	// cancelBackGesture).
	BackGesture *MethodChannel
	// Platform: "flutter/platform", JSON (clipboard, haptics, sounds,
	// system chrome, app exit, initializationComplete...).
	Platform *MethodChannel
	// StatusBar: "flutter/status_bar", JSON (handleScrollToTop).
	StatusBar *OptionalMethodChannel
	// ProcessText: "flutter/processtext", standard codec.
	ProcessText *MethodChannel
	// TextInput: "flutter/textinput", JSON.
	TextInput *MethodChannel
	// Scribe: "flutter/scribe", JSON (Android stylus handwriting).
	Scribe *MethodChannel
	// SpellCheck: "flutter/spellcheck", standard codec.
	SpellCheck *MethodChannel
	// UndoManager: "flutter/undomanager", JSON.
	UndoManager *MethodChannel
	// KeyEvent: "flutter/keyevent", JSON messages (raw key events).
	KeyEvent *BasicMessageChannel[any]
	// Lifecycle: "flutter/lifecycle", string messages
	// ("AppLifecycleState.resumed"...).
	Lifecycle *BasicMessageChannel[*string]
	// System: "flutter/system", JSON messages ({"type": "memoryPressure"}).
	System *BasicMessageChannel[any]
	// Accessibility: "flutter/accessibility", standard codec messages.
	Accessibility *BasicMessageChannel[any]
	// PlatformViews: "flutter/platform_views", standard codec.
	PlatformViews *MethodChannel
	// PlatformViews2: "flutter/platform_views_2", standard codec.
	PlatformViews2 *MethodChannel
	// Skia: "flutter/skia", JSON (Skia.setResourceCacheMaxBytes).
	Skia *MethodChannel
	// MouseCursor: "flutter/mousecursor", standard codec.
	MouseCursor *MethodChannel
	// Restoration: "flutter/restoration", standard codec.
	Restoration *MethodChannel
	// DeferredComponent: "flutter/deferredcomponent", standard codec.
	DeferredComponent *MethodChannel
	// Localization: "flutter/localization", JSON.
	Localization *MethodChannel
	// Menu: "flutter/menu", standard codec (platform menus).
	Menu *MethodChannel
	// ContextMenu: "flutter/contextmenu", JSON (browser context menu).
	ContextMenu *MethodChannel
	// Keyboard: "flutter/keyboard", standard codec (getKeyboardState).
	Keyboard *MethodChannel
	// SensitiveContent: "flutter/sensitivecontent", standard codec.
	SensitiveContent *MethodChannel
}

func optionalChannel(name string, codec ...MethodCodec) *MethodChannel {
	var opts MethodChannelOptions
	if len(codec) > 0 {
		opts.Codec = codec[0]
	}
	return &NewOptionalMethodChannel(name, opts).MethodChannel
}

// SystemChannels holds the platform channels Flutter uses (Dart's static
// constants of SystemChannels; the ones Dart declares as MethodChannel but
// creates as OptionalMethodChannel behave as such).
var SystemChannels = systemChannels{
	Navigation:  optionalChannel("flutter/navigation", JSONMethodCodec{}),
	BackGesture: optionalChannel("flutter/backgesture"),
	Platform:    optionalChannel("flutter/platform", JSONMethodCodec{}),
	StatusBar: NewOptionalMethodChannel("flutter/status_bar",
		MethodChannelOptions{Codec: JSONMethodCodec{}}),
	ProcessText:       optionalChannel("flutter/processtext"),
	TextInput:         optionalChannel("flutter/textinput", JSONMethodCodec{}),
	Scribe:            optionalChannel("flutter/scribe", JSONMethodCodec{}),
	SpellCheck:        optionalChannel("flutter/spellcheck"),
	UndoManager:       optionalChannel("flutter/undomanager", JSONMethodCodec{}),
	KeyEvent:          NewBasicMessageChannel[any]("flutter/keyevent", JSONMessageCodec{}),
	Lifecycle:         NewBasicMessageChannel[*string]("flutter/lifecycle", StringCodec{}),
	System:            NewBasicMessageChannel[any]("flutter/system", JSONMessageCodec{}),
	Accessibility:     NewBasicMessageChannel[any]("flutter/accessibility", NewStandardMessageCodec()),
	PlatformViews:     NewMethodChannel("flutter/platform_views"),
	PlatformViews2:    NewMethodChannel("flutter/platform_views_2"),
	Skia:              NewMethodChannel("flutter/skia", MethodChannelOptions{Codec: JSONMethodCodec{}}),
	MouseCursor:       optionalChannel("flutter/mousecursor"),
	Restoration:       optionalChannel("flutter/restoration"),
	DeferredComponent: optionalChannel("flutter/deferredcomponent"),
	Localization:      optionalChannel("flutter/localization", JSONMethodCodec{}),
	Menu:              optionalChannel("flutter/menu"),
	ContextMenu:       optionalChannel("flutter/contextmenu", JSONMethodCodec{}),
	Keyboard:          optionalChannel("flutter/keyboard"),
	SensitiveContent:  optionalChannel("flutter/sensitivecontent"),
}
