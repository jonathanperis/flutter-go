package flutter_test

// A minimal stand-in for package:matcher (which flutter_test re-exports):
// the Matcher interface, expect and isNot, enough for flutter_test's own
// matchers (the paints matchers of mock_canvas.dart, isPathThat). The rest
// of package:matcher is not ported.

import (
	"reflect"
	"testing"
)

// Matcher is package:matcher's Matcher: Matches tests item (recording
// details in matchState), Describe describes what it expects, and
// DescribeMismatch explains why item did not match.
type Matcher interface {
	Matches(item any, matchState map[any]any) bool
	Describe() string
	DescribeMismatch(item any, matchState map[any]any) string
}

// TestFailure is what expect throws when a match fails (package:matcher's
// TestFailure); its message is the failure report.
type TestFailure struct{ Message string }

func (f TestFailure) Error() string { return f.Message }

// failureMessage formats a failed expectation as package:matcher does.
func failureMessage(actual any, m Matcher, matchState map[any]any) string {
	msg := "Expected: " + m.Describe() + "\n  Actual: " + describeActual(actual) + "\n"
	if which := m.DescribeMismatch(actual, matchState); which != "" {
		msg += "   Which: " + which + "\n"
	}
	return msg
}

func describeActual(actual any) string {
	if actual != nil && reflect.TypeOf(actual).Kind() == reflect.Func {
		return "<Closure: " + reflect.TypeOf(actual).String() + ">"
	}
	return "<" + dartToString(actual) + ">"
}

// Expect is expect(actual, matcher) inside a Go test: a mismatch is
// reported with t.Error (the test continues, as Go tests do).
func Expect(t testing.TB, actual any, matcher Matcher) {
	t.Helper()
	matchState := map[any]any{}
	if !matcher.Matches(actual, matchState) {
		t.Error(failureMessage(actual, matcher, matchState))
	}
}

// expectOrThrow is expect() where it is called outside a test function
// (inside a matcher): a mismatch panics with a TestFailure.
func expectOrThrow(actual any, matcher Matcher) {
	matchState := map[any]any{}
	if !matcher.Matches(actual, matchState) {
		panic(TestFailure{failureMessage(actual, matcher, matchState)})
	}
}

type isNot struct{ m Matcher }

// IsNot is isNot(matcher): it matches what matcher does not.
func IsNot(matcher Matcher) Matcher { return isNot{matcher} }

func (n isNot) Matches(item any, _ map[any]any) bool { return !n.m.Matches(item, map[any]any{}) }
func (n isNot) Describe() string                     { return "not " + n.m.Describe() }
func (n isNot) DescribeMismatch(any, map[any]any) string {
	return ""
}
