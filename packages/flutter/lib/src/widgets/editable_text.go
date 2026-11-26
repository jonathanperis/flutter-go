package widgets

// Interim bridge for packages/flutter/lib/src/widgets/editable_text.dart
// (Flutter 3.50.1): EditableTextState's text-editing actions (the _actions
// map: _DeleteTextAction, _UpdateTextSelectionAction,
// _UpdateTextSelectionVerticallyAction for a single line, _SelectAllAction,
// _CopySelectionAction, _PasteSelectionAction, transpose, replace/update
// selection), its boundary helpers (_characterBoundary, _nextWordBoundary,
// _linebreak, _paragraphBoundary, _documentBoundary, _moveBeyondTextBoundary,
// _moveToTextBoundary), userUpdateTextEditingValue/_formatAndSetValue and
// performSelector, ported from the source onto the interim TextField
// (textfield.go). The EditableText widget itself, its selection overlay,
// scrolling, formatters, undo and obscured text are not ported yet.
//
// editableTextLayout stands in for RenderEditable's text layout: a
// painting.TextPainter of the text laid out on one line (as a single-line
// EditableText lays out), whose WordBoundaries (painting's WordBoundary)
// give the word moves, as RenderEditable's TextPainter does.

import (
	"reflect"
	"strconv"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/characters"
)

// editableTextLayout is the text layout of a single-line field: a
// TextPainter of the text with unbounded width. It is the
// TextLayoutMetrics RenderEditable implements.
type editableTextLayout struct {
	text    string
	painter *painting.TextPainter
}

func newEditableTextLayout(text string) *editableTextLayout {
	ltr := ui.TextDirectionLtr
	p := painting.NewTextPainter(painting.TextPainterOptions{
		Text:          painting.NewTextSpan(painting.TextSpanOptions{Text: &text}),
		TextDirection: &ltr,
	})
	p.Layout()
	return &editableTextLayout{text: text, painter: p}
}

// GetLineAtOffset is RenderEditable.getLineAtOffset: the painter's line
// boundary as a selection.
func (l *editableTextLayout) GetLineAtOffset(position ui.TextPosition) services.TextSelection {
	line := l.painter.GetLineBoundary(position)
	return services.NewTextSelection(line.Start, line.End)
}

// GetWordBoundary is RenderEditable.getWordBoundary.
func (l *editableTextLayout) GetWordBoundary(position ui.TextPosition) ui.TextRange {
	return l.painter.GetWordBoundary(position)
}

// GetTextPositionAbove: a single line has no line above (the start).
func (l *editableTextLayout) GetTextPositionAbove(ui.TextPosition) ui.TextPosition {
	return ui.NewTextPosition(0)
}

// GetTextPositionBelow: a single line has no line below (the end).
func (l *editableTextLayout) GetTextPositionBelow(ui.TextPosition) ui.TextPosition {
	return ui.NewTextPosition(dart.StringLength(l.text))
}

// --- EditableTextState's boundaries.

func (s *textFieldState) value() services.TextEditingValue { return s.controller().value }

func (s *textFieldState) layout() *editableTextLayout {
	text := s.value().Text
	if s.textLayout == nil || s.textLayout.text != text {
		s.textLayout = newEditableTextLayout(text)
	}
	return s.textLayout
}

func (s *textFieldState) characterBoundary() services.TextBoundary {
	return services.NewCharacterBoundary(s.value().Text)
}

func (s *textFieldState) nextWordBoundary() services.TextBoundary {
	return s.layout().painter.WordBoundaries().MoveByWordBoundary()
}

func (s *textFieldState) linebreak() services.TextBoundary {
	return services.NewLineBoundary(s.layout())
}

func (s *textFieldState) paragraphBoundary() services.TextBoundary {
	return services.NewParagraphBoundary(s.value().Text)
}

func (s *textFieldState) documentBoundary() services.TextBoundary {
	return services.NewDocumentBoundary(s.value().Text)
}

type applyTextBoundary func(s *textFieldState, extent ui.TextPosition, forward bool, textBoundary services.TextBoundary) ui.TextPosition

// moveBeyondTextBoundary is _moveBeyondTextBoundary: the closest boundary
// to extent but not extent itself, in the direction of forward.
func moveBeyondTextBoundary(s *textFieldState, extent ui.TextPosition, forward bool, textBoundary services.TextBoundary) ui.TextPosition {
	if extent.Offset < 0 {
		panic(assertionFailed("extent.offset >= 0"))
	}
	var newOffset int
	if forward {
		if t := textBoundary.GetTrailingTextBoundaryAt(extent.Offset); t != nil {
			newOffset = *t
		} else {
			newOffset = dart.StringLength(s.value().Text)
		}
	} else if l := textBoundary.GetLeadingTextBoundaryAt(extent.Offset - 1); l != nil {
		newOffset = *l
	}
	return ui.NewTextPosition(newOffset)
}

// moveToTextBoundary is _moveToTextBoundary: the closest boundary to
// extent, extent included (a fixed point), used for line boundaries.
func moveToTextBoundary(s *textFieldState, extent ui.TextPosition, forward bool, textBoundary services.TextBoundary) ui.TextPosition {
	if extent.Offset < 0 {
		panic(assertionFailed("extent.offset >= 0"))
	}
	var caretOffset int
	switch extent.Affinity {
	case ui.TextAffinityUpstream:
		if extent.Offset < 1 && !forward {
			return ui.NewTextPosition(0)
		}
		caretOffset = max(0, extent.Offset-1)
	default:
		caretOffset = extent.Offset
	}
	if forward {
		offset := dart.StringLength(s.value().Text)
		if t := textBoundary.GetTrailingTextBoundaryAt(caretOffset); t != nil {
			offset = *t
		}
		return ui.NewTextPosition(offset, ui.TextAffinityUpstream)
	}
	offset := 0
	if l := textBoundary.GetLeadingTextBoundaryAt(caretOffset); l != nil {
		offset = *l
	}
	return ui.NewTextPosition(offset)
}

var widgetsLibrary = "widgets library"

// --- the actions.

type contextAction struct {
	ActionBase
	enabled func() bool
	invoke  func(Intent) any
}

func (a *contextAction) IsEnabled(Intent) bool {
	if a.enabled == nil {
		return true
	}
	return a.enabled()
}

func (a *contextAction) Invoke(intent Intent) any { return a.invoke(intent) }

func newContextAction(enabled func() bool, invoke func(Intent) any) *contextAction {
	a := &contextAction{enabled: enabled, invoke: invoke}
	a.SetSelf(a)
	return a
}

// invokeIntent is Actions.invoke(context, intent) from the field: the
// field's action for the intent, when enabled.
func (s *textFieldState) invokeIntent(intent Intent) any {
	if a := s.findAction(intent); a != nil && a.IsEnabled(intent) {
		return a.Invoke(intent)
	}
	return nil
}

// findAction is Actions.maybeFind from the field (EditableText's actions).
func (s *textFieldState) findAction(intent Intent) Action {
	return s.actions[IntentType(intent)]
}

// deleteTextAction is _DeleteTextAction.
func (s *textFieldState) deleteTextAction(getTextBoundary func() services.TextBoundary, apply applyTextBoundary) Action {
	return newContextAction(
		func() bool { return !s.Widget().ReadOnly && s.value().Selection.IsValid() },
		func(intent Intent) any {
			forward := intent.(DirectionalTextEditingIntent).Forward()
			value := s.value()
			selection := value.Selection
			if !selection.IsValid() {
				return nil
			}
			// Expands the selection to ensure the range covers full graphemes.
			atomicBoundary := s.characterBoundary()
			textLength := dart.StringLength(value.Text)
			if !selection.IsCollapsed() {
				start, end := textLength, 0
				if l := atomicBoundary.GetLeadingTextBoundaryAt(selection.Start); l != nil {
					start = *l
				}
				if t := atomicBoundary.GetTrailingTextBoundaryAt(selection.End - 1); t != nil {
					end = *t
				}
				return s.invokeIntent(ReplaceTextIntent{CurrentTextEditingValue: value, ReplacementText: "",
					ReplacementRange: ui.NewTextRange(start, end), Cause: services.SelectionChangedCauseKeyboard})
			}
			target := apply(s, selection.Base(), forward, getTextBoundary()).Offset
			var base int
			if forward {
				base = textLength
				if l := atomicBoundary.GetLeadingTextBoundaryAt(selection.BaseOffset); l != nil {
					base = *l
				}
			} else if t := atomicBoundary.GetTrailingTextBoundaryAt(selection.BaseOffset - 1); t != nil {
				base = *t
			}
			rangeToDelete := services.NewTextSelection(base, target)
			return s.invokeIntent(ReplaceTextIntent{CurrentTextEditingValue: value, ReplacementText: "",
				ReplacementRange: rangeToDelete.TextRange, Cause: services.SelectionChangedCauseKeyboard})
		})
}

type updateSelectionOptions struct {
	ignoreNonCollapsedSelection, isExpand, extentAtIndex bool
}

// isAtWordwrapUpstream / isAtWordwrapDownstream are
// _UpdateTextSelectionAction's wrap checks.
func (s *textFieldState) isAtWordwrapUpstream(position ui.TextPosition) bool {
	end := ui.NewTextPosition(s.layout().GetLineAtOffset(position).End, ui.TextAffinityUpstream)
	text := s.value().Text
	return end == position && end.Offset != dart.StringLength(text) && dart.CodeUnitAt(text, position.Offset) != 10
}

func (s *textFieldState) isAtWordwrapDownstream(position ui.TextPosition) bool {
	start := ui.NewTextPosition(s.layout().GetLineAtOffset(position).Start)
	return start == position && start.Offset != 0 && dart.CodeUnitAt(s.value().Text, position.Offset-1) != 10
}

// updateTextSelectionAction is _UpdateTextSelectionAction.
func (s *textFieldState) updateTextSelectionAction(getTextBoundary func() services.TextBoundary, apply applyTextBoundary, o updateSelectionOptions) Action {
	return newContextAction(
		func() bool { return s.value().Selection.IsValid() },
		func(i Intent) any {
			intent := i.(DirectionalCaretMovementIntent)
			value := s.value()
			selection := value.Selection
			if !selection.IsValid() {
				panic(assertionFailed("selection.isValid"))
			}
			collapseSelection := intent.CollapseSelection() || !s.selectionEnabled()
			if !selection.IsCollapsed() && !o.ignoreNonCollapsedSelection && collapseSelection {
				offset := selection.Start
				if intent.Forward() {
					offset = selection.End
				}
				return s.invokeIntent(UpdateSelectionIntent{CurrentTextEditingValue: value,
					NewSelection: services.NewTextSelectionCollapsed(offset), Cause: services.SelectionChangedCauseKeyboard})
			}
			extent := selection.Extent()
			// If continuesAtWrap is true extent and is at the relevant
			// wordwrap, then move it just to the other side of the wordwrap.
			if intent.ContinuesAtWrap() {
				if intent.Forward() && s.isAtWordwrapUpstream(extent) {
					extent = ui.NewTextPosition(extent.Offset)
				} else if !intent.Forward() && s.isAtWordwrapDownstream(extent) {
					extent = ui.NewTextPosition(extent.Offset, ui.TextAffinityUpstream)
				}
			}
			shouldTargetBase := o.isExpand && (intent.Forward() && selection.BaseOffset > selection.ExtentOffset ||
				!intent.Forward() && selection.BaseOffset < selection.ExtentOffset)
			from := extent
			if shouldTargetBase {
				from = selection.Base()
			}
			newExtent := apply(s, from, intent.Forward(), getTextBoundary())
			var newSelection services.TextSelection
			switch {
			case collapseSelection || (!o.isExpand && newExtent.Offset == selection.BaseOffset):
				newSelection = services.NewTextSelectionFromPosition(newExtent)
			case o.isExpand:
				newSelection = selection.ExpandTo(newExtent, o.extentAtIndex || selection.IsCollapsed())
			default:
				newSelection = selection.ExtendTo(newExtent)
			}
			shouldCollapseToBase := intent.CollapseAtReversal() &&
				(selection.BaseOffset-selection.ExtentOffset)*(selection.BaseOffset-newSelection.ExtentOffset) < 0
			newRange := newSelection
			if shouldCollapseToBase {
				newRange = services.NewTextSelectionFromPosition(selection.Base())
			}
			return s.invokeIntent(UpdateSelectionIntent{CurrentTextEditingValue: value, NewSelection: newRange,
				Cause: services.SelectionChangedCauseKeyboard})
		})
}

// updateTextSelectionVerticallyAction is
// _UpdateTextSelectionVerticallyAction for a single line: the vertical
// caret movement run has no line above or below, so the extent moves to
// the start or the end of the text.
func (s *textFieldState) updateTextSelectionVerticallyAction() Action {
	return newContextAction(
		func() bool { return s.value().Selection.IsValid() },
		func(i Intent) any {
			intent := i.(DirectionalCaretMovementIntent)
			collapseSelection := intent.CollapseSelection() || !s.selectionEnabled()
			value := s.value()
			if !value.Selection.IsValid() {
				return nil
			}
			newExtent := ui.NewTextPosition(0)
			if intent.Forward() {
				newExtent = ui.NewTextPosition(dart.StringLength(value.Text))
			}
			newSelection := value.Selection.ExtendTo(newExtent)
			if collapseSelection {
				newSelection = services.NewTextSelectionFromPosition(newExtent)
			}
			s.invokeIntent(UpdateSelectionIntent{CurrentTextEditingValue: value, NewSelection: newSelection,
				Cause: services.SelectionChangedCauseKeyboard})
			return nil
		})
}

// selectionEnabled is EditableText.selectionEnabled (enableInteractiveSelection
// and not obscured: always, in the bridge).
func (s *textFieldState) selectionEnabled() bool { return true }

// replaceText is _replaceText.
func (s *textFieldState) replaceText(intent ReplaceTextIntent) {
	newValue := intent.CurrentTextEditingValue.Replaced(intent.ReplacementRange, intent.ReplacementText)
	s.UserUpdateTextEditingValue(newValue, intent.Cause)
}

// updateSelection is _updateSelection.
func (s *textFieldState) updateSelection(intent UpdateSelectionIntent) {
	n := dart.StringLength(intent.CurrentTextEditingValue.Text)
	if intent.NewSelection.Start > n || intent.NewSelection.End > n {
		panic(assertionFailed("invalid selection: " + intent.NewSelection.String() +
			": it must not exceed the current text length " + strconv.Itoa(n)))
	}
	sel := intent.NewSelection
	s.UserUpdateTextEditingValue(intent.CurrentTextEditingValue.CopyWith(services.TextEditingValueCopyWithOptions{Selection: &sel}), intent.Cause)
}

// transposeCharacters is _transposeCharacters.
func (s *textFieldState) transposeCharacters(TransposeCharactersIntent) {
	value := s.value()
	if characters.NewCharacters(value.Text).Length() <= 1 || !value.Selection.IsCollapsed() || value.Selection.BaseOffset == 0 {
		return
	}
	text := value.Text
	atEnd := value.Selection.BaseOffset == dart.StringLength(text)
	transposing := characters.NewCharacterRangeAt(text, value.Selection.BaseOffset)
	if atEnd {
		transposing.MoveBack(2)
	} else {
		transposing.MoveBack()
		transposing.ExpandNext()
	}
	current := transposing.CurrentCharacters()
	if current.Length() != 2 {
		panic(assertionFailed("transposing.currentCharacters.length == 2"))
	}
	sel := services.NewTextSelectionCollapsed(transposing.StringBeforeLength() + dart.StringLength(transposing.Current()))
	cause := services.SelectionChangedCauseKeyboard
	s.UserUpdateTextEditingValue(services.NewTextEditingValue(services.TextEditingValueOptions{
		Text:      dart.Concat(transposing.StringBefore(), current.Last(), current.First(), transposing.StringAfter()),
		Selection: &sel,
	}), cause)
}

// CopySelection is copySelection: the selection goes to the clipboard.
func (s *textFieldState) CopySelection(cause services.SelectionChangedCause) {
	value := s.value()
	if value.Selection.IsCollapsed() {
		return
	}
	async.OnComplete(services.Clipboard.SetData(services.NewClipboardData(value.Selection.TextInside(value.Text))), func(async.Void) {},
		s.reportClipboardError("while copying selection to clipboard"))
}

// CutSelection is cutSelection.
func (s *textFieldState) CutSelection(cause services.SelectionChangedCause) {
	if s.Widget().ReadOnly {
		return
	}
	value := s.value()
	if value.Selection.IsCollapsed() {
		return
	}
	async.OnComplete(services.Clipboard.SetData(services.NewClipboardData(value.Selection.TextInside(value.Text))), func(async.Void) {},
		s.reportClipboardError("while cutting selection to clipboard"))
	s.replaceText(ReplaceTextIntent{CurrentTextEditingValue: value, ReplacementText: "",
		ReplacementRange: value.Selection.TextRange, Cause: cause})
}

func (s *textFieldState) reportClipboardError(context string) func(error) {
	return func(err error) {
		foundation.FlutterErrorReportError(&foundation.FlutterErrorDetails{Exception: err,
			Library: &widgetsLibrary, Context: foundation.NewErrorDescription(context)})
	}
}

func (s *textFieldState) allowPaste() bool {
	return !s.Widget().ReadOnly && s.value().Selection.IsValid()
}

// PasteText is pasteText: the clipboard's plain text replaces the
// selection, once the platform answers.
func (s *textFieldState) PasteText(cause services.SelectionChangedCause) async.Future[async.Void] {
	if !s.allowPaste() {
		return async.FutureValue(async.Void{})
	}
	return async.Then(services.Clipboard.GetData(services.ClipboardKTextPlain), func(data *services.ClipboardData) async.Void {
		if data != nil && data.Text != nil {
			s.pasteText(cause, *data.Text)
		}
		return async.Void{}
	})
}

func (s *textFieldState) pasteText(cause services.SelectionChangedCause, text string) {
	if !s.allowPaste() {
		return
	}
	value := s.value()
	selection := value.Selection
	collapsed := services.NewTextSelectionCollapsed(max(selection.BaseOffset, selection.ExtentOffset))
	s.UserUpdateTextEditingValue(value.CopyWith(services.TextEditingValueCopyWithOptions{Selection: &collapsed}).
		Replaced(selection.TextRange, text), cause)
}

// buildActions is EditableTextState._actions.
func (s *textFieldState) buildActions() map[reflect.Type]Action {
	f := false
	vertical := s.updateTextSelectionVerticallyAction()
	return map[reflect.Type]Action{
		IntentTypeOf[DoNothingAndStopPropagationTextIntent](): NewDoNothingAction(DoNothingActionOptions{ConsumesKey: &f}),
		IntentTypeOf[ReplaceTextIntent]():                     NewCallbackAction(func(i ReplaceTextIntent) any { s.replaceText(i); return nil }),
		IntentTypeOf[UpdateSelectionIntent](): NewCallbackAction(func(i UpdateSelectionIntent) any {
			s.updateSelection(i)
			return nil
		}),
		IntentTypeOf[DeleteCharacterIntent]():          s.deleteTextAction(s.characterBoundary, moveBeyondTextBoundary),
		IntentTypeOf[DeleteToNextWordBoundaryIntent](): s.deleteTextAction(s.nextWordBoundary, moveBeyondTextBoundary),
		IntentTypeOf[DeleteToLineBreakIntent]():        s.deleteTextAction(s.linebreak, moveToTextBoundary),
		IntentTypeOf[ExtendSelectionByCharacterIntent](): s.updateTextSelectionAction(s.characterBoundary, moveBeyondTextBoundary,
			updateSelectionOptions{}),
		IntentTypeOf[ExtendSelectionToNextWordBoundaryIntent](): s.updateTextSelectionAction(s.nextWordBoundary, moveBeyondTextBoundary,
			updateSelectionOptions{ignoreNonCollapsedSelection: true}),
		IntentTypeOf[ExtendSelectionToNextParagraphBoundaryIntent](): s.updateTextSelectionAction(s.paragraphBoundary, moveBeyondTextBoundary,
			updateSelectionOptions{ignoreNonCollapsedSelection: true}),
		IntentTypeOf[ExtendSelectionToLineBreakIntent](): s.updateTextSelectionAction(s.linebreak, moveToTextBoundary,
			updateSelectionOptions{ignoreNonCollapsedSelection: true}),
		IntentTypeOf[ExtendSelectionVerticallyToAdjacentLineIntent](): vertical,
		IntentTypeOf[ExtendSelectionVerticallyToAdjacentPageIntent](): vertical,
		IntentTypeOf[ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent](): s.updateTextSelectionAction(s.paragraphBoundary,
			moveBeyondTextBoundary, updateSelectionOptions{ignoreNonCollapsedSelection: true}),
		IntentTypeOf[ExtendSelectionToDocumentBoundaryIntent](): s.updateTextSelectionAction(s.documentBoundary, moveBeyondTextBoundary,
			updateSelectionOptions{ignoreNonCollapsedSelection: true}),
		IntentTypeOf[ExtendSelectionToNextWordBoundaryOrCaretLocationIntent](): s.updateTextSelectionAction(s.nextWordBoundary,
			moveBeyondTextBoundary, updateSelectionOptions{ignoreNonCollapsedSelection: true}),
		// Scrolling the field (bringIntoView) is not ported: a single-line
		// field that fits does not scroll.
		IntentTypeOf[ScrollToDocumentBoundaryIntent](): NewCallbackAction(func(ScrollToDocumentBoundaryIntent) any { return nil }),
		IntentTypeOf[ExpandSelectionToLineBreakIntent](): s.updateTextSelectionAction(s.linebreak, moveToTextBoundary,
			updateSelectionOptions{ignoreNonCollapsedSelection: true, isExpand: true}),
		IntentTypeOf[ExpandSelectionToDocumentBoundaryIntent](): s.updateTextSelectionAction(s.documentBoundary, moveToTextBoundary,
			updateSelectionOptions{ignoreNonCollapsedSelection: true, isExpand: true, extentAtIndex: true}),
		IntentTypeOf[SelectAllTextIntent](): newContextAction(nil, func(i Intent) any {
			if !s.selectionEnabled() {
				return nil
			}
			value := s.value()
			return s.invokeIntent(UpdateSelectionIntent{CurrentTextEditingValue: value,
				NewSelection: services.NewTextSelection(0, dart.StringLength(value.Text)), Cause: i.(SelectAllTextIntent).Cause})
		}),
		IntentTypeOf[CopySelectionTextIntent](): newContextAction(nil, func(i Intent) any {
			intent := i.(CopySelectionTextIntent)
			if !s.value().Selection.IsValid() || s.value().Selection.IsCollapsed() || !s.selectionEnabled() {
				return nil
			}
			if intent.CollapseSelection {
				s.CutSelection(intent.Cause)
			} else {
				s.CopySelection(intent.Cause)
			}
			return nil
		}),
		IntentTypeOf[PasteTextIntent](): newContextAction(nil, func(i Intent) any {
			if !s.selectionEnabled() {
				return nil
			}
			async.OnComplete(s.PasteText(i.(PasteTextIntent).Cause), func(async.Void) {}, s.reportClipboardError("while pasting text"))
			return nil
		}),
		IntentTypeOf[TransposeCharactersIntent](): NewCallbackAction(func(i TransposeCharactersIntent) any {
			s.transposeCharacters(i)
			return nil
		}),
	}
}

// PerformSelector is EditableTextState.performSelector: a macOS selector
// is invoked as its intent.
func (s *textFieldState) PerformSelector(selectorName string) {
	if intent := IntentForMacOSSelector(selectorName); intent != nil {
		s.invokeIntent(intent)
	}
}
