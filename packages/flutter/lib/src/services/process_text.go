package services

// Port of packages/flutter/lib/src/services/process_text.dart (Flutter
// 3.50.1).

import (
	"sort"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// ProcessTextAction is a data structure describing text processing actions
// (a comparable value: Dart's == compares id and label).
type ProcessTextAction struct {
	// ID is the action unique id.
	ID string
	// Label is the action localized label.
	Label string
}

// NewProcessTextAction is ProcessTextAction(id, label).
func NewProcessTextAction(id, label string) ProcessTextAction {
	return ProcessTextAction{ID: id, Label: label}
}

// ProcessTextService determines how to interact with the text processing
// feature.
type ProcessTextService interface {
	// QueryTextActions returns a list of text processing actions.
	QueryTextActions() async.Future[[]ProcessTextAction]
	// ProcessTextAction requests that the action identified by id process
	// text; the future's value is the processed text, or nil.
	ProcessTextAction(id, text string, readOnly bool) async.Future[*string]
}

// DefaultProcessTextService is the service used by default by the text
// processing feature.
type DefaultProcessTextService struct {
	processTextChannel *MethodChannel
}

// NewDefaultProcessTextService is DefaultProcessTextService().
func NewDefaultProcessTextService() *DefaultProcessTextService {
	return &DefaultProcessTextService{processTextChannel: SystemChannels.ProcessText}
}

// SetChannel sets the MethodChannel used to communicate with the engine
// (@visibleForTesting; debug builds only).
func (s *DefaultProcessTextService) SetChannel(newChannel *MethodChannel) {
	if foundation.KDebugMode {
		s.processTextChannel = newChannel
	}
}

// QueryTextActions asks the platform for its text processing actions (an
// empty list if it returns null or fails). Divergence: Dart lists the
// actions in the order of the platform's map; Go's decoded map has no
// order, so they are sorted by id.
func (s *DefaultProcessTextService) QueryTextActions() async.Future[[]ProcessTextAction] {
	return async.Async(func() []ProcessTextAction {
		rawResults, ok := func() (raw map[any]any, ok bool) {
			defer func() {
				if recover() != nil {
					raw, ok = nil, false
				}
			}()
			result := s.processTextChannel.InvokeMethod("ProcessText.queryTextActions").Await()
			if result == nil {
				return nil, false
			}
			return result.(map[any]any), true
		}()
		if !ok {
			return []ProcessTextAction{}
		}
		ids := make([]string, 0, len(rawResults))
		for id := range rawResults {
			ids = append(ids, id.(string))
		}
		sort.Slice(ids, func(i, j int) bool { return dart.CompareStrings(ids[i], ids[j]) < 0 })
		actions := make([]ProcessTextAction, 0, len(ids))
		for _, id := range ids {
			actions = append(actions, ProcessTextAction{ID: id, Label: rawResults[id].(string)})
		}
		return actions
	})
}

// ProcessTextAction requests the platform to process text with the action
// id.
func (s *DefaultProcessTextService) ProcessTextAction(id, text string, readOnly bool) async.Future[*string] {
	return async.Async(func() *string {
		processedText := s.processTextChannel.InvokeMethod("ProcessText.processTextAction", []any{id, text, readOnly}).Await()
		if processedText == nil {
			return nil
		}
		str := processedText.(string)
		return &str
	})
}
