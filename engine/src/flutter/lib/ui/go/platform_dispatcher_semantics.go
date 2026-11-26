package ui

// Partial port of engine/src/flutter/lib/ui/platform_dispatcher.dart and
// window.dart (Flutter 3.50.1): the accessibility and semantics parts
// semantics needs. From platform_dispatcher.dart: Locale,
// SemanticsActionEvent, SemanticsActionEventCallback and PlatformDispatcher's
// semanticsEnabled/onSemanticsEnabledChanged, accessibilityFeatures/
// onAccessibilityFeaturesChanged, onSemanticsActionEvent,
// setSemanticsTreeEnabled, onPlatformConfigurationChanged and implicitView;
// from window.dart: AccessibilityFeatures. From hooks.dart, the
// engine entry points _updateSemanticsEnabled, _updateAccessibilityFeatures
// and _dispatchSemanticsAction, which an embedder calls through
// PlatformConfiguration.
//
// FlutterView and the views are in window.go and
// platform_dispatcher_views.go.

import (
	"strings"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// ---- Locale

// Locale is an identifier used to select a user's language and formatting
// preferences (a language code, optional script and country codes).
// Compare with Equal (Dart's ==, under which deprecated codes and their
// replacements are equal, and an empty country code equals none).
type Locale struct {
	languageCode string
	scriptCode   *string
	countryCode  *string
}

// NewLocale is Locale(languageCode, [countryCode]).
func NewLocale(languageCode string, countryCode ...string) Locale {
	if languageCode == "" {
		panic(dart.NewAssertionError(nil))
	}
	l := Locale{languageCode: languageCode}
	if len(countryCode) > 0 {
		c := countryCode[0]
		l.countryCode = &c
	}
	return l
}

// LocaleFromSubtagsOptions are Locale.fromSubtags' parameters (LanguageCode
// "" is 'und'; nil codes are null).
type LocaleFromSubtagsOptions struct {
	LanguageCode string
	ScriptCode   *string
	CountryCode  *string
}

// NewLocaleFromSubtags is Locale.fromSubtags.
func NewLocaleFromSubtags(opts ...LocaleFromSubtagsOptions) Locale {
	var o LocaleFromSubtagsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.LanguageCode == "" {
		o.LanguageCode = "und"
	}
	if (o.ScriptCode != nil && *o.ScriptCode == "") || (o.CountryCode != nil && *o.CountryCode == "") {
		panic(dart.NewAssertionError(nil))
	}
	return Locale{languageCode: o.LanguageCode, scriptCode: o.ScriptCode, countryCode: o.CountryCode}
}

// deprecatedLanguageSubtagMap is Locale._deprecatedLanguageSubtagMap.
var deprecatedLanguageSubtagMap = map[string]string{
	"in": "id", "iw": "he", "ji": "yi", "jw": "jv", "mo": "ro", "aam": "aas", "adp": "dz", "aue": "ktz",
	"ayx": "nun", "bgm": "bcg", "bjd": "drl", "ccq": "rki", "cjr": "mom", "cka": "cmr", "cmk": "xch",
	"coy": "pij", "cqu": "quh", "drh": "khk", "drw": "prs", "gav": "dev", "gfx": "vaj", "ggn": "gvr",
	"gti": "nyc", "guv": "duz", "hrr": "jal", "ibi": "opa", "ilw": "gal", "jeg": "oyb", "kgc": "tdf",
	"kgh": "kml", "koj": "kwv", "krm": "bmf", "ktr": "dtp", "kvs": "gdj", "kwq": "yam", "kxe": "tvd",
	"kzj": "dtp", "kzt": "dtp", "lii": "raq", "lmm": "rmx", "meg": "cir", "mst": "mry", "mwj": "vaj",
	"myt": "mry", "nad": "xny", "ncp": "kdz", "nnx": "ngv", "nts": "pij", "oun": "vaj", "pcr": "adx",
	"pmc": "huw", "pmu": "phr", "ppa": "bfy", "ppr": "lcq", "pry": "prt", "puz": "pub", "sca": "hle",
	"skk": "oyb", "tdu": "dtp", "thc": "tpo", "thx": "oyb", "tie": "ras", "tkk": "twm", "tlw": "weo",
	"tmp": "tyj", "tne": "kak", "tnf": "prs", "tsf": "taj", "uok": "ema", "xba": "cax", "xia": "acn",
	"xkh": "waw", "xsj": "suj", "ybd": "rki", "yma": "lrr", "ymt": "mtm", "yos": "zom", "yuu": "yug",
}

// deprecatedRegionSubtagMap is Locale._deprecatedRegionSubtagMap.
var deprecatedRegionSubtagMap = map[string]string{
	"BU": "MM", "DD": "DE", "FX": "FR", "TP": "TL", "YD": "YE", "ZR": "CD",
}

// LanguageCode is the primary language subtag (deprecated subtags are
// replaced by their current ones).
func (l Locale) LanguageCode() string {
	if r, ok := deprecatedLanguageSubtagMap[l.languageCode]; ok {
		return r
	}
	return l.languageCode
}

// ScriptCode is the script subtag, or nil.
func (l Locale) ScriptCode() *string { return l.scriptCode }

// CountryCode is the region subtag (deprecated subtags replaced), or nil.
func (l Locale) CountryCode() *string {
	if l.countryCode == nil {
		return nil
	}
	if r, ok := deprecatedRegionSubtagMap[*l.countryCode]; ok {
		return &r
	}
	c := *l.countryCode
	return &c
}

// Equal is Locale's ==.
func (l Locale) Equal(other Locale) bool {
	thisCountryCode := l.CountryCode()
	otherCountryCode := other.CountryCode()
	eqStr := func(a, b *string) bool {
		if a == nil || b == nil {
			return a == nil && b == nil
		}
		return *a == *b
	}
	return other.LanguageCode() == l.LanguageCode() &&
		eqStr(other.scriptCode, l.scriptCode) &&
		(eqStr(otherCountryCode, thisCountryCode) ||
			otherCountryCode != nil && *otherCountryCode == "" && thisCountryCode == nil ||
			thisCountryCode != nil && *thisCountryCode == "" && otherCountryCode == nil)
}

// String is the locale with '_' separators (en_US).
func (l Locale) String() string { return l.rawToString("_") }

// ToLanguageTag is the BCP47 tag (en-US).
func (l Locale) ToLanguageTag() string { return l.rawToString("-") }

func (l Locale) rawToString(separator string) string {
	var out strings.Builder
	out.WriteString(l.LanguageCode())
	if l.scriptCode != nil && *l.scriptCode != "" {
		out.WriteString(separator + *l.scriptCode)
	}
	if l.countryCode != nil && *l.countryCode != "" {
		out.WriteString(separator + *l.CountryCode())
	}
	return out.String()
}

// ---- SemanticsActionEvent

// SemanticsActionEventCallback is the signature of
// PlatformDispatcher.onSemanticsActionEvent.
type SemanticsActionEventCallback func(action SemanticsActionEvent)

// SemanticsActionEvent is an event to request a SemanticsAction of Type to
// be performed on the semantics node NodeID owned by the view ViewID. For
// some actions, Arguments carries additional information ([]byte for the
// engine's encoded arguments; nil is null).
type SemanticsActionEvent struct {
	Type      SemanticsAction
	ViewID    int
	NodeID    int
	Arguments any
}

// SemanticsActionEventCopyWithOptions are SemanticsActionEvent.copyWith's
// parameters: nil keeps the value; Arguments is replaced (even by nil)
// when SetArguments is true (Dart's placeholder default).
type SemanticsActionEventCopyWithOptions struct {
	Type         *SemanticsAction
	ViewID       *int
	NodeID       *int
	Arguments    any
	SetArguments bool
}

// CopyWith is a copy of this event with the given values replaced.
func (e SemanticsActionEvent) CopyWith(opts ...SemanticsActionEventCopyWithOptions) SemanticsActionEvent {
	var o SemanticsActionEventCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r := SemanticsActionEvent{
		Type:      orValue(o.Type, e.Type),
		ViewID:    orValue(o.ViewID, e.ViewID),
		NodeID:    orValue(o.NodeID, e.NodeID),
		Arguments: e.Arguments,
	}
	if o.SetArguments {
		r.Arguments = o.Arguments
	}
	return r
}

// ---- AccessibilityFeatures (window.dart)

// AccessibilityFeatures are additional accessibility features that may be
// enabled by the platform (a bit field). Compare with ==.
type AccessibilityFeatures struct{ index int }

const (
	kAccessibleNavigationIndex     = 1 << 0
	kInvertColorsIndex             = 1 << 1
	kDisableAnimationsIndex        = 1 << 2
	kBoldTextIndex                 = 1 << 3
	kReduceMotionIndex             = 1 << 4
	kHighContrastIndex             = 1 << 5
	kOnOffSwitchLabelsIndex        = 1 << 6
	kNoAnnounceIndex               = 1 << 7
	kNoAutoPlayAnimatedImagesIndex = 1 << 8
	kNoAutoPlayVideosIndex         = 1 << 9
	kDeterministicCursorIndex      = 1 << 10
)

// AccessibleNavigation: a screen reader or switch control is navigating.
func (f AccessibilityFeatures) AccessibleNavigation() bool {
	return kAccessibleNavigationIndex&f.index != 0
}

// InvertColors: the platform is inverting the colors of the application.
func (f AccessibilityFeatures) InvertColors() bool { return kInvertColorsIndex&f.index != 0 }

// DisableAnimations: the platform is requesting that animations be
// disabled or simplified.
func (f AccessibilityFeatures) DisableAnimations() bool { return kDisableAnimationsIndex&f.index != 0 }

// BoldText: the platform is requesting that text be rendered at a bold
// font weight.
func (f AccessibilityFeatures) BoldText() bool { return kBoldTextIndex&f.index != 0 }

// ReduceMotion: the platform is requesting that certain animations be
// simplified and parallax effects removed.
func (f AccessibilityFeatures) ReduceMotion() bool { return kReduceMotionIndex&f.index != 0 }

// HighContrast: the platform is requesting that UI be rendered with darker
// colors.
func (f AccessibilityFeatures) HighContrast() bool { return kHighContrastIndex&f.index != 0 }

// OnOffSwitchLabels: the platform is requesting to show on/off labels
// inside switches.
func (f AccessibilityFeatures) OnOffSwitchLabels() bool { return kOnOffSwitchLabelsIndex&f.index != 0 }

// SupportsAnnounce: the platform supports announcements (inverted bit).
func (f AccessibilityFeatures) SupportsAnnounce() bool { return kNoAnnounceIndex&f.index == 0 }

// AutoPlayAnimatedImages: animated images may play automatically (inverted
// bit).
func (f AccessibilityFeatures) AutoPlayAnimatedImages() bool {
	return kNoAutoPlayAnimatedImagesIndex&f.index == 0
}

// AutoPlayVideos: videos may play automatically (inverted bit).
func (f AccessibilityFeatures) AutoPlayVideos() bool { return kNoAutoPlayVideosIndex&f.index == 0 }

// DeterministicCursor: the platform is requesting a non-blinking cursor.
func (f AccessibilityFeatures) DeterministicCursor() bool {
	return kDeterministicCursorIndex&f.index != 0
}

func (f AccessibilityFeatures) String() string {
	features := []string{}
	add := func(cond bool, name string) {
		if cond {
			features = append(features, name)
		}
	}
	add(f.AccessibleNavigation(), "accessibleNavigation")
	add(f.InvertColors(), "invertColors")
	add(f.DisableAnimations(), "disableAnimations")
	add(f.BoldText(), "boldText")
	add(f.ReduceMotion(), "reduceMotion")
	add(f.HighContrast(), "highContrast")
	add(f.OnOffSwitchLabels(), "onOffSwitchLabels")
	add(f.SupportsAnnounce(), "supportsAnnounce")
	add(f.AutoPlayAnimatedImages(), "autoPlayAnimatedImages")
	add(f.AutoPlayVideos(), "autoPlayVideos")
	add(f.DeterministicCursor(), "deterministicCursor")
	return "AccessibilityFeatures[" + strings.Join(features, ", ") + "]"
}

// ---- PlatformDispatcher (accessibility and semantics)

// platformDispatcherSemantics is the state of PlatformDispatcher's
// accessibility and semantics members.
type platformDispatcherSemantics struct {
	onPlatformConfigurationChanged VoidCallback
	onSemanticsEnabledChanged      VoidCallback
	onAccessibilityFeaturesChanged VoidCallback
	onSemanticsActionEvent         SemanticsActionEventCallback
}

// OnPlatformConfigurationChanged is called when the platform configuration
// changes.
func (p *PlatformDispatcher) OnPlatformConfigurationChanged() VoidCallback {
	return p.semantics.onPlatformConfigurationChanged
}

// SetOnPlatformConfigurationChanged sets OnPlatformConfigurationChanged.
func (p *PlatformDispatcher) SetOnPlatformConfigurationChanged(callback VoidCallback) {
	p.semantics.onPlatformConfigurationChanged = callback
}

// SemanticsEnabled is whether the user has requested that UpdateSemantics
// be called when the semantic contents of a view changes.
func (p *PlatformDispatcher) SemanticsEnabled() bool { return p.configuration.semanticsEnabled }

// OnSemanticsEnabledChanged is invoked when SemanticsEnabled changes.
func (p *PlatformDispatcher) OnSemanticsEnabledChanged() VoidCallback {
	return p.semantics.onSemanticsEnabledChanged
}

// SetOnSemanticsEnabledChanged sets OnSemanticsEnabledChanged.
func (p *PlatformDispatcher) SetOnSemanticsEnabledChanged(callback VoidCallback) {
	p.semantics.onSemanticsEnabledChanged = callback
}

// updateSemanticsEnabled is called from the engine, via hooks.
func (p *PlatformDispatcher) updateSemanticsEnabled(enabled bool) {
	if p.configuration.semanticsEnabled == enabled {
		return
	}
	p.configuration.semanticsEnabled = enabled
	invoke(p.semantics.onPlatformConfigurationChanged)
	invoke(p.semantics.onSemanticsEnabledChanged)
}

// AccessibilityFeatures is the additional accessibility features that may
// be enabled by the platform.
func (p *PlatformDispatcher) AccessibilityFeatures() AccessibilityFeatures {
	return p.configuration.accessibilityFeatures
}

// OnAccessibilityFeaturesChanged is invoked when AccessibilityFeatures
// changes.
func (p *PlatformDispatcher) OnAccessibilityFeaturesChanged() VoidCallback {
	return p.semantics.onAccessibilityFeaturesChanged
}

// SetOnAccessibilityFeaturesChanged sets OnAccessibilityFeaturesChanged.
func (p *PlatformDispatcher) SetOnAccessibilityFeaturesChanged(callback VoidCallback) {
	p.semantics.onAccessibilityFeaturesChanged = callback
}

// updateAccessibilityFeatures is called from the engine, via hooks.
func (p *PlatformDispatcher) updateAccessibilityFeatures(values int) {
	newFeatures := AccessibilityFeatures{index: values}
	if newFeatures == p.configuration.accessibilityFeatures {
		return
	}
	p.configuration.accessibilityFeatures = newFeatures
	invoke(p.semantics.onPlatformConfigurationChanged)
	invoke(p.semantics.onAccessibilityFeaturesChanged)
}

// OnSemanticsActionEvent is called whenever the user requests a
// SemanticsAction be performed on a semantics node.
func (p *PlatformDispatcher) OnSemanticsActionEvent() SemanticsActionEventCallback {
	return p.semantics.onSemanticsActionEvent
}

// SetOnSemanticsActionEvent sets OnSemanticsActionEvent.
func (p *PlatformDispatcher) SetOnSemanticsActionEvent(callback SemanticsActionEventCallback) {
	p.semantics.onSemanticsActionEvent = callback
}

// dispatchSemanticsAction is called from the engine, via hooks.
func (p *PlatformDispatcher) dispatchSemanticsAction(viewID, nodeID, action int, args []byte) {
	if p.semantics.onSemanticsActionEvent == nil {
		return
	}
	var arguments any
	if args != nil {
		arguments = args
	}
	p.semantics.onSemanticsActionEvent(SemanticsActionEvent{
		Type:      *SemanticsActionFromIndex(action),
		NodeID:    nodeID,
		ViewID:    viewID,
		Arguments: arguments,
	})
}

// SetSemanticsTreeEnabled tells the engine whether the framework will
// generate a semantics tree (PlatformConfigurationNativeApi::
// SetSemanticsTreeEnabled).
func (p *PlatformDispatcher) SetSemanticsTreeEnabled(enabled bool) {
	platformConfigurationNativeAPISetSemanticsTreeEnabled(enabled)
}

func invoke(callback VoidCallback) {
	if callback != nil {
		callback()
	}
}

// ---- Engine side

// PlatformConfigurationSemanticsClient is the part of the engine's
// PlatformConfigurationClient that receives semantics: an embedder's
// PlatformConfigurationClient implements it to get them. ENGINE GAP:
// flutter-go's embedders have no accessibility bridge; with no such client,
// the natives only record (see SemanticsTreeEnabled, EngineSemanticsUpdateOf).
type PlatformConfigurationSemanticsClient interface {
	SetSemanticsTreeEnabled(enabled bool)
	UpdateSemantics(viewID int, update SemanticsUpdate)
}

var semanticsTreeEnabled bool

// SemanticsTreeEnabled is the engine's record of the last
// SetSemanticsTreeEnabled call.
func SemanticsTreeEnabled() bool { return semanticsTreeEnabled }

func platformConfigurationNativeAPISetSemanticsTreeEnabled(enabled bool) {
	semanticsTreeEnabled = enabled
	if pc := currentPlatformConfiguration; pc != nil {
		if c, ok := pc.client.(PlatformConfigurationSemanticsClient); ok {
			c.SetSemanticsTreeEnabled(enabled)
		}
	}
}

func platformConfigurationNativeAPIUpdateSemantics(viewID int, update SemanticsUpdate) {
	if pc := currentPlatformConfiguration; pc != nil {
		if c, ok := pc.client.(PlatformConfigurationSemanticsClient); ok {
			c.UpdateSemantics(viewID, update)
		}
	}
}

// UpdateSemanticsEnabled tells the framework whether the platform wants a
// semantics tree (PlatformConfiguration::UpdateSemanticsEnabled, which
// calls hooks.dart's _updateSemanticsEnabled).
func (pc *PlatformConfiguration) UpdateSemanticsEnabled(enabled bool) {
	PlatformDispatcherInstance().updateSemanticsEnabled(enabled)
}

// UpdateAccessibilityFeatures sends the platform's accessibility feature
// flags (PlatformConfiguration::UpdateAccessibilityFeatures, hooks.dart's
// _updateAccessibilityFeatures).
func (pc *PlatformConfiguration) UpdateAccessibilityFeatures(values int) {
	PlatformDispatcherInstance().updateAccessibilityFeatures(values)
}

// DispatchSemanticsAction sends a semantics action from the platform
// (PlatformConfiguration::DispatchSemanticsAction, hooks.dart's
// _dispatchSemanticsAction); args are the action's encoded arguments, or
// nil.
func (pc *PlatformConfiguration) DispatchSemanticsAction(viewID, nodeID, action int, args []byte) {
	PlatformDispatcherInstance().dispatchSemanticsAction(viewID, nodeID, action, args)
}
