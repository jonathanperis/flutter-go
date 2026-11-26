package services

// Port of packages/flutter/lib/src/services/spell_check.dart (Flutter
// 3.50.1).

import (
	"slices"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// SuggestionSpan is a range of misspelled text and the spell check
// suggestions for it. == is Equal.
type SuggestionSpan struct {
	// Range is the range of the misspelled text.
	Range ui.TextRange
	// Suggestions are the alternate suggestions for the misspelled text.
	Suggestions []string
}

// NewSuggestionSpan is SuggestionSpan(range, suggestions).
func NewSuggestionSpan(textRange ui.TextRange, suggestions []string) SuggestionSpan {
	return SuggestionSpan{Range: textRange, Suggestions: suggestions}
}

// Equal is SuggestionSpan's ==.
func (s SuggestionSpan) Equal(other SuggestionSpan) bool {
	return other.Range.Start == s.Range.Start && other.Range.End == s.Range.End &&
		listEqualsNullable(other.Suggestions, s.Suggestions, func(a, b string) bool { return a == b })
}

func (s SuggestionSpan) String() string {
	return "SuggestionSpan(range: " + s.Range.String() + ", suggestions: [" + strings.Join(s.Suggestions, ", ") + "])"
}

// SpellCheckResults are the spell check results of a particular text.
type SpellCheckResults struct {
	// SpellCheckedText is the text that the spell check results were
	// generated for.
	SpellCheckedText string
	// SuggestionSpans are the spell check results for SpellCheckedText.
	SuggestionSpans []SuggestionSpan
}

// NewSpellCheckResults is SpellCheckResults(spellCheckedText,
// suggestionSpans).
func NewSpellCheckResults(spellCheckedText string, suggestionSpans []SuggestionSpan) SpellCheckResults {
	return SpellCheckResults{SpellCheckedText: spellCheckedText, SuggestionSpans: suggestionSpans}
}

// Equal is SpellCheckResults's ==.
func (r SpellCheckResults) Equal(other SpellCheckResults) bool {
	return other.SpellCheckedText == r.SpellCheckedText &&
		listEqualsNullable(other.SuggestionSpans, r.SuggestionSpans, SuggestionSpan.Equal)
}

func (r SpellCheckResults) String() string {
	spans := make([]string, len(r.SuggestionSpans))
	for i, s := range r.SuggestionSpans {
		spans[i] = s.String()
	}
	return "SpellCheckResults(spellCheckText: " + r.SpellCheckedText + ", suggestionSpans: [" + strings.Join(spans, ", ") + "])"
}

// listEqualsNullable is foundation's listEquals with element equality eq (a
// nil slice is null).
func listEqualsNullable[T any](a, b []T, eq func(T, T) bool) bool {
	if a == nil {
		return b == nil
	}
	if b == nil || len(a) != len(b) {
		return false
	}
	return slices.EqualFunc(a, b, eq)
}

// SpellCheckService determines how spell check results are generated for
// specified text.
type SpellCheckService interface {
	// FetchSpellCheckSuggestions facilitates a spell check request; the
	// future's value is nil when the request was canceled.
	FetchSpellCheckSuggestions(locale ui.Locale, text string) async.Future[[]SuggestionSpan]
}

// DefaultSpellCheckService is the service used by default to fetch spell
// check results for text input (Android and iOS).
type DefaultSpellCheckService struct {
	// LastSavedResults are the results of the last spell check request (nil
	// if none).
	LastSavedResults *SpellCheckResults
	// SpellCheckChannel is the channel used to communicate with the spell
	// checker.
	SpellCheckChannel *MethodChannel
}

// NewDefaultSpellCheckService is DefaultSpellCheckService().
func NewDefaultSpellCheckService() *DefaultSpellCheckService {
	return &DefaultSpellCheckService{SpellCheckChannel: SystemChannels.SpellCheck}
}

// DefaultSpellCheckServiceMergeResults is DefaultSpellCheckService
// .mergeResults: merges two lists of spell check SuggestionSpans, keeping
// the older span where both start at the same index.
func DefaultSpellCheckServiceMergeResults(oldResults, newResults []SuggestionSpan) []SuggestionSpan {
	mergedResults := []SuggestionSpan{}
	oldSpanPointer, newSpanPointer := 0, 0
	for oldSpanPointer < len(oldResults) && newSpanPointer < len(newResults) {
		oldSpan := oldResults[oldSpanPointer]
		newSpan := newResults[newSpanPointer]
		if oldSpan.Range.Start == newSpan.Range.Start {
			mergedResults = append(mergedResults, oldSpan)
			oldSpanPointer++
			newSpanPointer++
		} else if oldSpan.Range.Start < newSpan.Range.Start {
			mergedResults = append(mergedResults, oldSpan)
			oldSpanPointer++
		} else {
			mergedResults = append(mergedResults, newSpan)
			newSpanPointer++
		}
	}
	mergedResults = append(mergedResults, oldResults[oldSpanPointer:]...)
	mergedResults = append(mergedResults, newResults[newSpanPointer:]...)
	return mergedResults
}

// FetchSpellCheckSuggestions sends the text to the platform's spell
// checker and returns its suggestions (nil if the request failed or was
// canceled).
func (s *DefaultSpellCheckService) FetchSpellCheckSuggestions(locale ui.Locale, text string) async.Future[[]SuggestionSpan] {
	return async.Async(func() []SuggestionSpan {
		languageTag := locale.ToLanguageTag()
		rawResults, ok := func() (raw []any, ok bool) {
			defer func() {
				if recover() != nil {
					// Spell check request canceled due to pending request.
					raw, ok = nil, false
				}
			}()
			return s.SpellCheckChannel.InvokeMethod("SpellCheck.initiateSpellCheck", []string{languageTag, text}).Await().([]any), true
		}()
		if !ok {
			return nil
		}
		suggestionSpans := []SuggestionSpan{}
		for _, raw := range rawResults {
			resultMap := raw.(*dart.LinkedMap)
			rawSuggestions := resultMap.Lookup("suggestions").([]any)
			suggestions := make([]string, len(rawSuggestions))
			for i, v := range rawSuggestions {
				suggestions[i] = v.(string)
			}
			suggestionSpans = append(suggestionSpans, SuggestionSpan{
				Range:       ui.NewTextRange(resultMap.Lookup("startIndex").(int), resultMap.Lookup("endIndex").(int)),
				Suggestions: suggestions,
			})
		}
		if s.LastSavedResults != nil {
			// Merge current and previous spell check results if between
			// requests, the text has not changed but the spell check results
			// have.
			textHasNotChanged := s.LastSavedResults.SpellCheckedText == text
			spansHaveChanged := listEqualsNullable(s.LastSavedResults.SuggestionSpans, suggestionSpans, SuggestionSpan.Equal)
			if textHasNotChanged && spansHaveChanged {
				suggestionSpans = DefaultSpellCheckServiceMergeResults(s.LastSavedResults.SuggestionSpans, suggestionSpans)
			}
		}
		s.LastSavedResults = &SpellCheckResults{SpellCheckedText: text, SuggestionSpans: suggestionSpans}
		return suggestionSpans
	})
}
