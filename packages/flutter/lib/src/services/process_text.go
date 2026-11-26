package services

// Port of packages/flutter/lib/src/services/process_text.dart (Flutter
// 3.50.1).

import (
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
// empty list if it returns null or fails), in the order of the platform's
// map.
func (s *DefaultProcessTextService) QueryTextActions() async.Future[[]ProcessTextAction] {
	return async.Async(func() []ProcessTextAction {
		rawResults, ok := func() (raw *dart.LinkedMap, ok bool) {
			defer func() {
				if recover() != nil {
					raw, ok = nil, false
				}
			}()
			result := s.processTextChannel.InvokeMethod("ProcessText.queryTextActions").Await()
			if result == nil {
				return nil, false
			}
			return result.(*dart.LinkedMap), true
		}()
		if !ok {
			return []ProcessTextAction{}
		}
		actions := make([]ProcessTextAction, 0, rawResults.Len())
		for id, label := range rawResults.Entries() {
			actions = append(actions, ProcessTextAction{ID: id.(string), Label: label.(string)})
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
