package ui

// Port of engine/src/flutter/testing/dart/semantics_test.dart (Flutter
// 3.50.1), all 4 tests, plus own tests of the value types, the builder's
// record, Locale, AccessibilityFeatures and the platform dispatcher's
// semantics hooks.

import (
	"strings"
	"testing"
)

func TestSemanticsFlagValuesRefersToAllFlags(t *testing.T) {
	// This must match the number of flags in lib/ui/semantics.dart
	const numSemanticsFlags = 31
	if n := len(SemanticsFlagValues()); n != numSemanticsFlags {
		t.Fatalf("got %d flags", n)
	}
	for index := 0; index < numSemanticsFlags; index++ {
		flag := 1 << index
		f := SemanticsFlagFromIndex(flag)
		if f == nil || !strings.HasPrefix(f.String(), "SemanticsFlag.") {
			t.Fatalf("flag %d: %v", flag, f)
		}
	}
}

func TestSemanticsActionValuesRefersToAllActions(t *testing.T) {
	// This must match the number of actions in lib/ui/semantics.dart
	const numSemanticsActions = 26
	if n := len(SemanticsActionValues()); n != numSemanticsActions {
		t.Fatalf("got %d actions", n)
	}
	for index := 0; index < numSemanticsActions; index++ {
		flag := 1 << index
		a := SemanticsActionFromIndex(flag)
		if a == nil || !strings.HasPrefix(a.String(), "SemanticsAction.") {
			t.Fatalf("action %d: %v", flag, a)
		}
	}
}

func TestSpellOutStringAttributeToString(t *testing.T) {
	got := NewSpellOutStringAttribute(NewTextRange(2, 5)).String()
	if got != "SpellOutStringAttribute(TextRange(start: 2, end: 5))" {
		t.Fatal(got)
	}
}

func TestLocaleStringAttributeToString(t *testing.T) {
	got := NewLocaleStringAttribute(NewTextRange(2, 5), NewLocale("test")).String()
	if got != "LocaleStringAttribute(TextRange(start: 2, end: 5), test)" {
		t.Fatal(got)
	}
}

func TestSemanticsFlagsMergeAndStrings(t *testing.T) {
	a := SemanticsFlags{IsChecked: CheckedStateIsFalse, IsButton: true, IsFocused: TristateIsTrue}
	b := SemanticsFlags{IsChecked: CheckedStateMixed, IsSelected: TristateIsFalse, IsAccessibilityFocusBlocked: true}
	m := a.Merge(b)
	want := SemanticsFlags{IsChecked: CheckedStateMixed, IsSelected: TristateIsFalse, IsButton: true,
		IsFocused: TristateIsTrue, IsAccessibilityFocusBlocked: true}
	if m != want {
		t.Fatalf("merge %+v", m)
	}
	if got := strings.Join(m.ToStrings(), ","); got !=
		"hasCheckedState,isButton,isFocused,isFocusable,isAccessibilityFocusBlocked,isCheckStateMixed,hasSelectedState" {
		t.Fatal(got)
	}
	if !a.HasConflictingFlags(b) || a.HasRepeatedFlags(SemanticsFlags{IsHeader: true}) ||
		!a.HasConflictingFlags(SemanticsFlags{IsAccessibilityFocusBlocked: true}) ||
		a.HasRepeatedFlags(SemanticsFlags{IsAccessibilityFocusBlocked: true}) {
		t.Fatal("conflicts")
	}
	if got := strings.Join(SemanticsFlags{IsFocused: TristateIsFalse}.ToStrings(), ","); got != "isFocusable" {
		t.Fatal(got)
	}
	if m := (SemanticsFlags{IsKeyboardKey: true}).Merge(SemanticsFlags{}); !m.IsKeyboardKey {
		t.Fatal("keyboard key merge")
	}
	yes := true
	c := a.CopyWith(SemanticsFlagsCopyWithOptions{IsHeader: &yes})
	if !c.IsHeader || !c.IsButton || a.IsHeader {
		t.Fatal("copyWith")
	}
	if CheckedStateNone.Merge(CheckedStateIsFalse) != CheckedStateIsFalse ||
		CheckedStateIsFalse.Merge(CheckedStateIsTrue) != CheckedStateIsTrue ||
		TristateIsFalse.Merge(TristateIsTrue) != TristateIsTrue || TristateNone.ToBoolOrNull() != nil ||
		!*TristateIsTrue.ToBoolOrNull() || *TristateIsFalse.ToBoolOrNull() {
		t.Fatal("tristates")
	}
}

func TestSemanticsUpdateBuilderRecordsUpdates(t *testing.T) {
	b := NewSemanticsUpdateBuilder()
	identity := []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	rtl := TextDirectionRtl
	loc := NewLocale("en", "US")
	b.UpdateNode(SemanticsUpdateBuilderUpdateNodeOptions{ID: 3, Transform: identity, TextDirection: &rtl,
		Locale: &loc, Role: SemanticsRoleTab, InputType: SemanticsInputTypeEmail})
	label := "l"
	b.UpdateCustomAction(1, SemanticsUpdateBuilderUpdateCustomActionOptions{Label: &label})
	u := b.Build()
	e := EngineSemanticsUpdateOf(u)
	if e == nil || len(e.Nodes) != 1 || len(e.CustomActions) != 1 {
		t.Fatalf("%+v", e)
	}
	n := e.Nodes[0]
	if n.ID != 3 || n.TextDirectionIndex != 1 || n.LocaleTag != "en-US" || n.RoleIndex != 1 || n.InputTypeIndex != 5 {
		t.Fatalf("%+v", n)
	}
	if a := e.CustomActions[0]; a.ID != 1 || a.Label != "l" || a.Hint != "" || a.OverrideID != -1 {
		t.Fatalf("%+v", a)
	}
	u.Dispose()
	if EngineSemanticsUpdateOf(u) != nil {
		t.Fatal("disposed update still readable")
	}
	for _, bad := range [][]float64{identity[:15], append(append([]float64{}, identity[:15]...), 1.0/zero())} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid transform accepted")
				}
			}()
			NewSemanticsUpdateBuilder().UpdateNode(SemanticsUpdateBuilderUpdateNodeOptions{Transform: bad})
		}()
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("heading level 7 accepted")
			}
		}()
		NewSemanticsUpdateBuilder().UpdateNode(SemanticsUpdateBuilderUpdateNodeOptions{Transform: identity, HeadingLevel: 7})
	}()
}

func zero() float64 { return 0 }

func TestLocale(t *testing.T) {
	cases := []struct {
		l           Locale
		str, tag    string
		lang        string
		equalsPlain bool
	}{
		{NewLocale("en"), "en", "en", "en", true},
		{NewLocale("en", "US"), "en_US", "en-US", "en", false},
		{NewLocale("iw", "DD"), "he_DE", "he-DE", "he", false},
		{NewLocale("en", ""), "en", "en", "en", true},
		{NewLocaleFromSubtags(LocaleFromSubtagsOptions{ScriptCode: ptrS("Hant"), CountryCode: ptrS("TW")}), "und_Hant_TW", "und-Hant-TW", "und", false},
	}
	for _, c := range cases {
		if c.l.String() != c.str || c.l.ToLanguageTag() != c.tag || c.l.LanguageCode() != c.lang ||
			c.l.Equal(NewLocale(c.lang)) != c.equalsPlain {
			t.Fatalf("%v: %q %q %q %v", c.str, c.l.String(), c.l.ToLanguageTag(), c.l.LanguageCode(), c.l.Equal(NewLocale(c.lang)))
		}
	}
	if !NewLocale("en").Equal(NewLocale("en", "")) || !NewLocale("en", "").Equal(NewLocale("en")) {
		t.Fatal("empty country")
	}
	if !NewLocale("iw", "DD").Equal(NewLocale("he", "DE")) {
		t.Fatal("deprecated codes")
	}
}

func ptrS(s string) *string { return &s }

func TestAccessibilityFeaturesString(t *testing.T) {
	if got := (AccessibilityFeatures{}).String(); got !=
		"AccessibilityFeatures[supportsAnnounce, autoPlayAnimatedImages, autoPlayVideos]" {
		t.Fatal(got)
	}
	all := AccessibilityFeatures{index: 1<<11 - 1}
	if got := all.String(); got !=
		"AccessibilityFeatures[accessibleNavigation, invertColors, disableAnimations, boldText, reduceMotion, highContrast, onOffSwitchLabels, deterministicCursor]" {
		t.Fatal(got)
	}
}

func TestPlatformDispatcherSemanticsHooks(t *testing.T) {
	pd := PlatformDispatcherInstance()
	pc := NewPlatformConfiguration(nil)
	var log []string
	pd.SetOnPlatformConfigurationChanged(func() { log = append(log, "config") })
	pd.SetOnSemanticsEnabledChanged(func() { log = append(log, "semantics") })
	pd.SetOnAccessibilityFeaturesChanged(func() { log = append(log, "features") })
	var events []SemanticsActionEvent
	pd.SetOnSemanticsActionEvent(func(e SemanticsActionEvent) { events = append(events, e) })
	defer func() {
		pc.UpdateSemanticsEnabled(false)
		pc.UpdateAccessibilityFeatures(0)
		pd.SetOnPlatformConfigurationChanged(nil)
		pd.SetOnSemanticsEnabledChanged(nil)
		pd.SetOnAccessibilityFeaturesChanged(nil)
		pd.SetOnSemanticsActionEvent(nil)
	}()
	pc.UpdateSemanticsEnabled(true)
	pc.UpdateSemanticsEnabled(true) // unchanged: no callbacks
	pc.UpdateAccessibilityFeatures(4)
	pc.UpdateAccessibilityFeatures(4)
	if got := strings.Join(log, ","); got != "config,semantics,config,features" || !pd.SemanticsEnabled() ||
		!pd.AccessibilityFeatures().DisableAnimations() {
		t.Fatal(got)
	}
	pc.DispatchSemanticsAction(2, 9, kTapIndex, []byte{1})
	pc.DispatchSemanticsAction(2, 9, kFocusIndex, nil)
	if len(events) != 2 || events[0].Type != SemanticsActionTap || events[0].ViewID != 2 || events[0].NodeID != 9 ||
		events[1].Arguments != nil || events[1].Type != SemanticsActionFocus {
		t.Fatalf("%+v", events)
	}
	c := events[0].CopyWith(SemanticsActionEventCopyWithOptions{SetArguments: true})
	if c.Arguments != nil || c.Type != SemanticsActionTap {
		t.Fatalf("%+v", c)
	}
	if v := pd.ImplicitView(); v.ViewID() != 0 || v.String() != "FlutterView(id: 0)" || v.PlatformDispatcher() != pd {
		t.Fatal("implicit view")
	}
}
