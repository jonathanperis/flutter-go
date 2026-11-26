package ui

// The rest of engine/src/flutter/lib/ui/platform_dispatcher.dart and
// hooks.dart (Flutter 3.50.1): locales (_updateLocales,
// computePlatformResolvedLocale, setApplicationLocale), the user settings
// (_updateUserSettingsData: textScaleFactor, alwaysUse24HourFormat,
// platformBrightness, systemFontFamily, the spell-check/context-menu/
// password flags, configurationId) with their callbacks, scaleFontSize,
// onError (hooks.dart's _onError), defaultRouteName, the isolate natives
// (setIsolateDebugName, getPersistentIsolateData,
// registerBackgroundIsolate, sendPortPlatformMessage), resetInternalState,
// the deprecated updateSemantics, ErrorCallback, SystemColor and
// SystemColorPalette; and the engine natives behind them
// (PlatformConfigurationNativeApi::Render, ComputePlatformResolvedLocale,
// SetApplicationLocale, DefaultRouteName, GetScaledFontSize...), which go
// to optional interfaces of the embedder's PlatformConfigurationClient;
// without them they answer as the engine's default PlatformView does.

import (
	"math"
	"strconv"

	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/isolate"
)

// ErrorCallback is the signature of PlatformDispatcher.onError: it returns
// whether the error was handled.
type ErrorCallback func(exception any, stackTrace dart.StackTrace) bool

// platformDispatcherSettings is the state of PlatformDispatcher's settings
// members.
type platformDispatcherSettings struct {
	onLocaleChanged                  VoidCallback
	onTextScaleFactorChanged         VoidCallback
	onPlatformBrightnessChanged      VoidCallback
	onSystemFontFamilyChanged        VoidCallback
	onError                          ErrorCallback
	nativeSpellCheckServiceDefined   bool
	supportsShowingSystemContextMenu bool
	brieflyShowPassword              bool
	cachedFontSizes                  map[int]float64
}

// ---- Locales

// Locale is the system's preferred locale: the first of Locales, or
// Locale.fromSubtags() ("und") when there is none.
func (p *PlatformDispatcher) Locale() Locale {
	if len(p.configuration.locales) == 0 {
		return NewLocaleFromSubtags()
	}
	return p.configuration.locales[0]
}

// SetApplicationLocale tells the platform the application's locale
// (PlatformConfigurationNativeApi::SetApplicationLocale, with the locale's
// language tag).
func (p *PlatformDispatcher) SetApplicationLocale(locale Locale) {
	if pc := currentPlatformConfiguration; pc != nil {
		if c, ok := pc.client.(PlatformConfigurationLocaleClient); ok {
			c.SetApplicationLocale(locale.ToLanguageTag())
		}
	}
}

// Locales are the system's preferred locales, most preferred first.
func (p *PlatformDispatcher) Locales() []Locale { return p.configuration.locales }

// ComputePlatformResolvedLocale asks the platform to pick the best of
// supportedLocales for the user (nil when it has no opinion, as on
// platforms without a native locale resolution).
func (p *PlatformDispatcher) ComputePlatformResolvedLocale(supportedLocales []Locale) *Locale {
	data := make([]*string, 0, 3*len(supportedLocales))
	for _, l := range supportedLocales {
		lang := l.LanguageCode()
		data = append(data, &lang, l.CountryCode(), l.ScriptCode())
	}
	result := platformConfigurationNativeAPIComputePlatformResolvedLocale(data)
	if len(result) > 0 {
		o := LocaleFromSubtagsOptions{LanguageCode: result[0]}
		if result[1] != "" {
			o.CountryCode = &result[1]
		}
		if result[2] != "" {
			o.ScriptCode = &result[2]
		}
		l := NewLocaleFromSubtags(o)
		return &l
	}
	return nil
}

// OnLocaleChanged is called when Locales changes.
func (p *PlatformDispatcher) OnLocaleChanged() VoidCallback { return p.settings.onLocaleChanged }

// SetOnLocaleChanged sets OnLocaleChanged.
func (p *PlatformDispatcher) SetOnLocaleChanged(callback VoidCallback) {
	p.settings.onLocaleChanged = callback
}

// updateLocales is _updateLocales: four strings per locale (language,
// country, script, variant).
func (p *PlatformDispatcher) updateLocales(locales []string) {
	const stringsPerLocale = 4
	numLocales := len(locales) / stringsPerLocale
	previous := p.configuration
	newLocales := []Locale{}
	localesDiffer := numLocales != len(previous.locales)
	for i := 0; i < numLocales; i++ {
		countryCode := locales[i*stringsPerLocale+1]
		scriptCode := locales[i*stringsPerLocale+2]
		o := LocaleFromSubtagsOptions{LanguageCode: locales[i*stringsPerLocale]}
		if countryCode != "" {
			o.CountryCode = &countryCode
		}
		if scriptCode != "" {
			o.ScriptCode = &scriptCode
		}
		newLocales = append(newLocales, NewLocaleFromSubtags(o))
		if !localesDiffer && !newLocales[i].Equal(previous.locales[i]) {
			localesDiffer = true
		}
	}
	if !localesDiffer {
		return
	}
	p.configuration.locales = newLocales
	invoke(p.semantics.onPlatformConfigurationChanged)
	invoke(p.settings.onLocaleChanged)
}

// localeClosure is _localeClosure (the engine reads the locale with it).
func (p *PlatformDispatcher) localeClosure() string { return p.Locale().String() }

// ---- User settings

// AlwaysUse24HourFormat is whether the user prefers a 24-hour clock.
func (p *PlatformDispatcher) AlwaysUse24HourFormat() bool {
	return p.configuration.alwaysUse24HourFormat
}

// LineHeightScaleFactorOverride is the platform's line height override
// (nil: none; no embedder sets it in 3.50.1).
func (p *PlatformDispatcher) LineHeightScaleFactorOverride() *float64 {
	return p.configuration.lineHeightScaleFactorOverride
}

// LetterSpacingOverride is the platform's letter spacing override (nil:
// none).
func (p *PlatformDispatcher) LetterSpacingOverride() *float64 {
	return p.configuration.letterSpacingOverride
}

// WordSpacingOverride is the platform's word spacing override (nil: none).
func (p *PlatformDispatcher) WordSpacingOverride() *float64 {
	return p.configuration.wordSpacingOverride
}

// ParagraphSpacingOverride is the platform's paragraph spacing override
// (nil: none).
func (p *PlatformDispatcher) ParagraphSpacingOverride() *float64 {
	return p.configuration.paragraphSpacingOverride
}

// TextScaleFactor is the system's text scale factor (1.0 by default).
func (p *PlatformDispatcher) TextScaleFactor() float64 { return p.configuration.textScaleFactor }

// OnTextScaleFactorChanged is called when TextScaleFactor changes.
func (p *PlatformDispatcher) OnTextScaleFactorChanged() VoidCallback {
	return p.settings.onTextScaleFactorChanged
}

// SetOnTextScaleFactorChanged sets OnTextScaleFactorChanged.
func (p *PlatformDispatcher) SetOnTextScaleFactorChanged(callback VoidCallback) {
	p.settings.onTextScaleFactorChanged = callback
}

// NativeSpellCheckServiceDefined is whether the platform has a spell check
// service.
func (p *PlatformDispatcher) NativeSpellCheckServiceDefined() bool {
	return p.settings.nativeSpellCheckServiceDefined
}

// SupportsShowingSystemContextMenu is whether the platform can show a
// system context menu.
func (p *PlatformDispatcher) SupportsShowingSystemContextMenu() bool {
	return p.settings.supportsShowingSystemContextMenu
}

// BrieflyShowPassword is whether the last typed character of an obscured
// text field is briefly shown (true by default).
func (p *PlatformDispatcher) BrieflyShowPassword() bool { return p.settings.brieflyShowPassword }

// OnPlatformBrightnessChanged is called when PlatformBrightness changes.
func (p *PlatformDispatcher) OnPlatformBrightnessChanged() VoidCallback {
	return p.settings.onPlatformBrightnessChanged
}

// SetOnPlatformBrightnessChanged sets OnPlatformBrightnessChanged.
func (p *PlatformDispatcher) SetOnPlatformBrightnessChanged(callback VoidCallback) {
	p.settings.onPlatformBrightnessChanged = callback
}

// SystemFontFamily is the system's font family (nil: the platform default).
func (p *PlatformDispatcher) SystemFontFamily() *string { return p.configuration.systemFontFamily }

// OnSystemFontFamilyChanged is called when SystemFontFamily changes.
func (p *PlatformDispatcher) OnSystemFontFamilyChanged() VoidCallback {
	return p.settings.onSystemFontFamilyChanged
}

// SetOnSystemFontFamilyChanged sets OnSystemFontFamilyChanged.
func (p *PlatformDispatcher) SetOnSystemFontFamilyChanged(callback VoidCallback) {
	p.settings.onSystemFontFamilyChanged = callback
}

// jsonNullCheck is Dart's `x!` on a decoded JSON value.
func jsonNullCheck(v any) any {
	if v == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	return v
}

// jsonAs is Dart's `x as T?` on a decoded JSON value: nil stays nil, a
// value of another type is a TypeError.
func jsonAs[T any](v any, typeName string) *T {
	if v == nil {
		return nil
	}
	t, ok := v.(T)
	if !ok {
		panic(dart.TypeError{Message: "type '" + jsonTypeName(v) + "' is not a subtype of type '" + typeName + "' in type cast"})
	}
	return &t
}

func jsonTypeName(v any) string {
	switch v.(type) {
	case string:
		return "String"
	case bool:
		return "bool"
	case int:
		return "int"
	case float64:
		return "double"
	case []any:
		return "List<dynamic>"
	case *dart.LinkedMap:
		return "_Map<String, dynamic>"
	}
	return "Object"
}

// updateUserSettingsData is _updateUserSettingsData: the settings as JSON.
func (p *PlatformDispatcher) updateUserSettingsData(jsonData string) {
	decoded := dart.JSONDecode(jsonData)
	data, ok := decoded.(*dart.LinkedMap)
	if !ok {
		panic(dart.TypeError{Message: "type '" + jsonTypeName(decoded) + "' is not a subtype of type 'Map<String, Object?>' in type cast"})
	}
	if data.IsEmpty() {
		return
	}
	var textScaleFactor float64
	switch n := jsonNullCheck(data.Lookup("textScaleFactor")).(type) {
	case int:
		textScaleFactor = float64(n)
	case float64:
		textScaleFactor = n
	default:
		panic(dart.TypeError{Message: "type '" + jsonTypeName(n) + "' is not a subtype of type 'num' in type cast"})
	}
	alwaysUse24HourFormat := *jsonAs[bool](jsonNullCheck(data.Lookup("alwaysUse24HourFormat")), "bool")
	if v := jsonAs[bool](data.Lookup("nativeSpellCheckServiceDefined"), "bool?"); v != nil {
		p.settings.nativeSpellCheckServiceDefined = *v
	} else {
		p.settings.nativeSpellCheckServiceDefined = false
	}
	if v := jsonAs[bool](data.Lookup("supportsShowingSystemContextMenu"), "bool?"); v != nil {
		p.settings.supportsShowingSystemContextMenu = *v
	} else {
		p.settings.supportsShowingSystemContextMenu = false
	}
	// This field is optional.
	if v := jsonAs[bool](data.Lookup("brieflyShowPassword"), "bool?"); v != nil {
		p.settings.brieflyShowPassword = *v
	}
	var platformBrightness Brightness
	switch data.Lookup("platformBrightness") {
	case "dark":
		platformBrightness = BrightnessDark
	case "light":
		platformBrightness = BrightnessLight
	default:
		panic(dart.StateError{Message: dart.ToString(data.Lookup("platformBrightness")) + " is not a valid platformBrightness."})
	}
	systemFontFamily := jsonAs[string](data.Lookup("systemFontFamily"), "String?")
	configurationID := jsonAs[int](data.Lookup("configurationId"), "int?")
	previous := p.configuration
	platformBrightnessChanged := previous.platformBrightness != platformBrightness
	textScaleFactorChanged := previous.textScaleFactor != textScaleFactor
	alwaysUse24HourFormatChanged := previous.alwaysUse24HourFormat != alwaysUse24HourFormat
	systemFontFamilyChanged := !nullableStringEqual(previous.systemFontFamily, systemFontFamily)
	if !platformBrightnessChanged && !textScaleFactorChanged && !alwaysUse24HourFormatChanged &&
		!systemFontFamilyChanged && configurationID == nil {
		return
	}
	// _PlatformConfiguration.copyWith: a null argument keeps the previous
	// value (so a systemFontFamily going back to null is reported as a
	// change but keeps the old family, as in Dart).
	next := previous
	next.textScaleFactor = textScaleFactor
	next.alwaysUse24HourFormat = alwaysUse24HourFormat
	next.platformBrightness = platformBrightness
	if systemFontFamily != nil {
		next.systemFontFamily = systemFontFamily
	}
	if configurationID != nil {
		next.configurationID = configurationID
	}
	p.configuration = next
	invoke(p.semantics.onPlatformConfigurationChanged)
	if textScaleFactorChanged {
		p.settings.cachedFontSizes = nil
		invoke(p.settings.onTextScaleFactorChanged)
	}
	if platformBrightnessChanged {
		invoke(p.settings.onPlatformBrightnessChanged)
	}
	if systemFontFamilyChanged {
		invoke(p.settings.onSystemFontFamilyChanged)
	}
}

func nullableStringEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// ScaleFontSize computes the scaled font size of unscaledFontSize with the
// platform's text scaling: linear (×TextScaleFactor) unless the platform
// reported a configuration id, whose nonlinear scaling the engine computes
// (PlatformConfigurationClient's GetScaledFontSize), interpolated between
// integer sizes.
func (p *PlatformDispatcher) ScaleFontSize(unscaledFontSize float64) float64 {
	assertNoMessage(unscaledFontSize >= 0)
	assertNoMessage(!math.IsInf(unscaledFontSize, 0) && !math.IsNaN(unscaledFontSize))
	if p.TextScaleFactor() == 1.0 {
		return unscaledFontSize
	}
	unscaledFloor := int(math.Floor(unscaledFontSize))
	unscaledCeil := int(math.Ceil(unscaledFontSize))
	if unscaledFloor == unscaledCeil {
		// No need to interpolate if the input value is an integer.
		if v := p.scaleAndMemoize(unscaledFloor); v != nil {
			return *v
		}
		return unscaledFontSize * p.TextScaleFactor()
	}
	dartAssert(unscaledCeil-unscaledFloor == 1, "Unexpected interpolation range: "+
		strconv.Itoa(unscaledFloor)+" - "+strconv.Itoa(unscaledCeil)+".")
	lower, upper := p.scaleAndMemoize(unscaledFloor), p.scaleAndMemoize(unscaledCeil)
	if lower == nil || upper == nil {
		return unscaledFontSize * p.TextScaleFactor()
	}
	return *lower + float64((*upper-*lower)*(unscaledFontSize-float64(unscaledFloor)))
}

// scaleAndMemoize is _scaleAndMemoize: nil on an error.
func (p *PlatformDispatcher) scaleAndMemoize(unscaledFontSize int) *float64 {
	if p.configuration.configurationID == nil {
		// The platform uses linear scaling, or hasn't sent a configuration.
		return nil
	}
	configurationID := *p.configuration.configurationID
	if cached, ok := p.settings.cachedFontSizes[unscaledFontSize]; ok {
		assertNoMessage(cached >= 0)
		return &cached
	}
	fontSize := platformConfigurationNativeAPIGetScaledFontSize(float64(unscaledFontSize), configurationID)
	if fontSize >= 0 {
		if p.settings.cachedFontSizes == nil {
			p.settings.cachedFontSizes = map[int]float64{}
		}
		p.settings.cachedFontSizes[unscaledFontSize] = fontSize
		return &fontSize
	}
	if fontSize == -1 {
		// Invalid configuration id. This error can be unrecoverable as
		// _getScaledFontSize can be destructive.
		dartAssert(false, "Flutter Error: incorrect configuration id: "+strconv.Itoa(configurationID)+".")
	} else {
		dartAssert(false, "Unknown error: GetScaledFontSize failed with "+dart.DoubleToString(fontSize)+".")
	}
	return nil
}

// ---- Errors

// OnError is the callback for uncaught errors of the root isolate (Go:
// reported by the embedder with PlatformConfiguration.DispatchError); it
// returns whether the error was handled.
func (p *PlatformDispatcher) OnError() ErrorCallback { return p.settings.onError }

// SetOnError sets OnError.
func (p *PlatformDispatcher) SetOnError(callback ErrorCallback) { p.settings.onError = callback }

// dispatchError is _dispatchError.
func (p *PlatformDispatcher) dispatchError(err any, stackTrace dart.StackTrace) bool {
	if p.settings.onError == nil {
		return false
	}
	return p.settings.onError(err, stackTrace)
}

// DispatchError hands an uncaught error to PlatformDispatcher.onError
// (hooks.dart's _onError, which the engine calls for unhandled
// exceptions; a nil stack trace is StackTrace.empty) and returns whether
// it was handled.
func (pc *PlatformConfiguration) DispatchError(err any, stackTrace dart.StackTrace) bool {
	if stackTrace == nil {
		stackTrace = dart.StackTraceFromString("")
	}
	return PlatformDispatcherInstance().dispatchError(err, stackTrace)
}

// ---- Misc natives

// DefaultRouteName is the route the embedder asked the app to start at
// ("/" unless the embedder's PlatformConfigurationRouteClient says
// otherwise).
func (p *PlatformDispatcher) DefaultRouteName() string {
	if pc := currentPlatformConfiguration; pc != nil {
		if c, ok := pc.client.(PlatformConfigurationRouteClient); ok {
			return c.DefaultRouteName()
		}
	}
	return "/"
}

// SetIsolateDebugName sets the isolate's debug name (UIDartState::
// SetDebugName; see EngineIsolateDebugName).
func (p *PlatformDispatcher) SetIsolateDebugName(name string) { isolateDebugName = name }

var isolateDebugName = "main"

// EngineIsolateDebugName is the root isolate's debug name, as set with
// PlatformDispatcher.SetIsolateDebugName ("main" by default).
func EngineIsolateDebugName() string { return isolateDebugName }

// GetPersistentIsolateData is the data the embedder attached to the root
// isolate for its whole lifetime (nil when none; see
// PlatformConfigurationPersistentDataClient).
func (p *PlatformDispatcher) GetPersistentIsolateData() []byte {
	if pc := currentPlatformConfiguration; pc != nil {
		if c, ok := pc.client.(PlatformConfigurationPersistentDataClient); ok {
			return c.GetPersistentIsolateData()
		}
	}
	return nil
}

// ResetInternalState resets the engine's internal state for this
// dispatcher (the embedder's PlatformConfigurationResetClient; tests use
// it between runs).
func (p *PlatformDispatcher) ResetInternalState() {
	if pc := currentPlatformConfiguration; pc != nil {
		if c, ok := pc.client.(PlatformConfigurationResetClient); ok {
			c.ResetInternalState()
		}
	}
}

// RegisterBackgroundIsolate registers a background isolate with the root
// isolate's platform message handler. ENGINE GAP: a Go program is one
// isolate (background work runs on goroutines that already share the
// root's messenger), so this only ensures DartPluginRegistrant is
// initialized.
func (p *PlatformDispatcher) RegisterBackgroundIsolate(token *RootIsolateToken) {
	DartPluginRegistrantEnsureInitialized()
}

// SendPortPlatformMessage sends a platform message whose reply is posted to
// port as Dart's engine posts it: a []any{identifier, data} pair for a
// reply with data, nil for an empty reply.
func (p *PlatformDispatcher) SendPortPlatformMessage(name string, data []byte, identifier int, port isolate.SendPort) {
	if port == nil {
		panic(dart.NewException("Invalid port specified"))
	}
	response := &platformMessageResponseDartPort{port: port, identifier: identifier, channel: name}
	message := &PlatformMessage{Channel: name, Data: data, Response: response}
	if pc := currentPlatformConfiguration; pc != nil {
		if handler, ok := pc.client.(PlatformMessageHandler); ok {
			handler.HandlePlatformMessage(message)
			return
		}
	}
	response.CompleteEmpty()
}

// platformMessageResponseDartPort is the engine's
// PlatformMessageResponseDartPort.
type platformMessageResponseDartPort struct {
	port       isolate.SendPort
	identifier int
	channel    string
	complete   bool
}

func (r *platformMessageResponseDartPort) Complete(data []byte) {
	r.complete = true
	r.port.Send([]any{r.identifier, append([]byte{}, data...)})
}

func (r *platformMessageResponseDartPort) CompleteEmpty() {
	r.complete = true
	r.port.Send(nil)
}

func (r *platformMessageResponseDartPort) IsComplete() bool { return r.complete }

// UpdateSemantics changes the implicit view's semantics tree (deprecated:
// use FlutterView.UpdateSemantics).
func (p *PlatformDispatcher) UpdateSemantics(update SemanticsUpdate) {
	platformConfigurationNativeAPIUpdateSemantics(kImplicitViewID, update)
}

// ---- SystemColor

// SystemColor is a color the platform defines (named after the CSS system
// colors); Value is nil when the platform doesn't support it.
type SystemColor struct {
	// Name is the color's name.
	Name string
	// Value is the color, or nil when unsupported.
	Value *Color
}

// IsSupported is whether the platform supports this color.
func (c SystemColor) IsSupported() bool { return c.Value != nil }

// SystemColorPlatformProvidesSystemColors is
// SystemColor.platformProvidesSystemColors: false except on the web.
func SystemColorPlatformProvidesSystemColors() bool { return false }

// SystemColorPalette is the platform's system colors for a brightness.
// Outside the web every getter panics with UnsupportedError.
type SystemColorPalette struct {
	// Brightness is the brightness of the palette.
	Brightness Brightness
}

// SystemColorLight and SystemColorDark are SystemColor.light and
// SystemColor.dark.
var (
	SystemColorLight = &SystemColorPalette{Brightness: BrightnessLight}
	SystemColorDark  = &SystemColorPalette{Brightness: BrightnessDark}
)

func systemColorUnsupportedError() dart.UnsupportedError {
	return dart.UnsupportedError{Message: "SystemColor not supported on the current platform."}
}

func (*SystemColorPalette) AccentColor() SystemColor      { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) AccentColorText() SystemColor  { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) ActiveText() SystemColor       { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) ButtonBorder() SystemColor     { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) ButtonFace() SystemColor       { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) ButtonText() SystemColor       { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) Canvas() SystemColor           { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) CanvasText() SystemColor       { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) Field() SystemColor            { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) FieldText() SystemColor        { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) GrayText() SystemColor         { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) Highlight() SystemColor        { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) HighlightText() SystemColor    { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) LinkText() SystemColor         { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) Mark() SystemColor             { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) MarkText() SystemColor         { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) SelectedItem() SystemColor     { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) SelectedItemText() SystemColor { panic(systemColorUnsupportedError()) }
func (*SystemColorPalette) VisitedText() SystemColor      { panic(systemColorUnsupportedError()) }

// ---- Engine side

// UpdateLocales reports the platform's preferred locales, four strings per
// locale: language, country, script and variant codes ("" for none)
// (PlatformConfiguration::UpdateLocales, hooks.dart's _updateLocales).
func (pc *PlatformConfiguration) UpdateLocales(locales []string) {
	PlatformDispatcherInstance().updateLocales(locales)
}

// UpdateUserSettingsData reports the platform's user settings as JSON
// (PlatformConfiguration::UpdateUserSettingsData, hooks.dart's
// _updateUserSettingsData): textScaleFactor, alwaysUse24HourFormat and
// platformBrightness are required; nativeSpellCheckServiceDefined,
// supportsShowingSystemContextMenu, brieflyShowPassword, systemFontFamily
// and configurationId are optional.
func (pc *PlatformConfiguration) UpdateUserSettingsData(data string) {
	PlatformDispatcherInstance().updateUserSettingsData(data)
}

// LocaleClosure is the engine's locale getter (hooks.dart's
// _getLocaleClosure): the current locale's toString.
func (pc *PlatformConfiguration) LocaleClosure() string {
	return PlatformDispatcherInstance().localeClosure()
}

// PlatformConfigurationRenderClient is the part of the engine's
// PlatformConfigurationClient (RuntimeController's Render) that receives
// the scenes FlutterView.Render shows: the layer tree of the scene, sized
// in physical pixels, and the view's device pixel ratio. An embedder
// presents it (EngineLayerTree.Flatten).
type PlatformConfigurationRenderClient interface {
	Render(viewID int, layerTree *EngineLayerTree, devicePixelRatio float64)
}

// PlatformConfigurationLocaleClient is the part of the engine's client that
// answers the locale natives (the default resolves no locale and ignores
// the application locale).
type PlatformConfigurationLocaleClient interface {
	SetApplicationLocale(locale string)
	// ComputePlatformResolvedLocale gets three strings per supported
	// locale (language, country, script; nil for none) and returns the
	// chosen one's three strings, or none.
	ComputePlatformResolvedLocale(supportedLocalesData []*string) []string
}

// PlatformConfigurationRouteClient gives the route the app starts at
// (Engine::DefaultRouteName; "/" without it).
type PlatformConfigurationRouteClient interface{ DefaultRouteName() string }

// PlatformConfigurationPersistentDataClient gives the root isolate's
// persistent data (nil without it).
type PlatformConfigurationPersistentDataClient interface{ GetPersistentIsolateData() []byte }

// PlatformConfigurationResetClient receives
// PlatformDispatcher.resetInternalState.
type PlatformConfigurationResetClient interface{ ResetInternalState() }

// PlatformConfigurationFontScaleClient computes the platform's nonlinear
// text scaling (GetScaledFontSize; -1 for an unknown configuration id,
// which is also the answer without it).
type PlatformConfigurationFontScaleClient interface {
	GetScaledFontSize(unscaledFontSize float64, configurationID int) float64
}

func platformConfigurationNativeAPIComputePlatformResolvedLocale(data []*string) []string {
	if pc := currentPlatformConfiguration; pc != nil {
		if c, ok := pc.client.(PlatformConfigurationLocaleClient); ok {
			return c.ComputePlatformResolvedLocale(data)
		}
	}
	return nil
}

func platformConfigurationNativeAPIGetScaledFontSize(unscaledFontSize float64, configurationID int) float64 {
	if pc := currentPlatformConfiguration; pc != nil {
		if c, ok := pc.client.(PlatformConfigurationFontScaleClient); ok {
			return c.GetScaledFontSize(unscaledFontSize, configurationID)
		}
	}
	return -1
}

// platformConfigurationNativeAPIRender is PlatformConfigurationNativeApi::
// Render with RuntimeController::Render: a view the engine has no metrics
// for drops the frame; otherwise the scene's layer tree goes to the
// embedder.
func platformConfigurationNativeAPIRender(viewID int, scene *nativeScene, width, height float64) {
	pc := currentPlatformConfiguration
	if pc == nil {
		return
	}
	metrics := pc.GetMetrics(viewID)
	if metrics == nil {
		return
	}
	tree := scene.takeLayerTree(doubleToUint64(width), doubleToUint64(height))
	if tree == nil {
		return // Engine::Render ignores a disposed scene's null tree
	}
	if c, ok := pc.client.(PlatformConfigurationRenderClient); ok {
		c.Render(viewID, tree, metrics.DevicePixelRatio)
	}
}

// doubleToUint64 is C++'s double → uint64_t conversion (truncation; values
// out of range are 0 here, where C++ is undefined).
func doubleToUint64(v float64) uint64 {
	if !(v >= 0) || v >= 1<<64 {
		return 0
	}
	return uint64(v)
}
