package services

// Port of packages/flutter/lib/src/services/text_editing.dart (Flutter
// 3.50.1): TextSelection.

import (
	"strconv"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// TextSelection is a range of text that represents a selection (a
// dart:ui TextRange with a base, an extent, an affinity and whether it is
// directional). Compare with Equal (Dart's ==).
type TextSelection struct {
	ui.TextRange
	// BaseOffset is the offset at which the selection originates.
	BaseOffset int
	// ExtentOffset is the offset at which the selection terminates.
	ExtentOffset int
	// Affinity: if the text range is collapsed and has more than one visual
	// location (e.g., occurs at a line break), which of the two locations
	// to use when representing this range.
	Affinity ui.TextAffinity
	// IsDirectional is whether this selection has disambiguated its base
	// and extent.
	IsDirectional bool
}

// TextSelectionOptions are TextSelection's optional named parameters
// (Affinity nil: downstream).
type TextSelectionOptions struct {
	Affinity      *ui.TextAffinity
	IsDirectional bool
}

// NewTextSelection is TextSelection(baseOffset:, extentOffset:, affinity:,
// isDirectional:) (the required named offsets are positional).
func NewTextSelection(baseOffset, extentOffset int, opts ...TextSelectionOptions) TextSelection {
	var o TextSelectionOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	affinity := ui.TextAffinityDownstream
	if o.Affinity != nil {
		affinity = *o.Affinity
	}
	start, end := extentOffset, baseOffset
	if baseOffset < extentOffset {
		start, end = baseOffset, extentOffset
	}
	return TextSelection{
		TextRange:  ui.NewTextRange(start, end),
		BaseOffset: baseOffset, ExtentOffset: extentOffset, Affinity: affinity, IsDirectional: o.IsDirectional,
	}
}

// NewTextSelectionCollapsed is TextSelection.collapsed(offset:, affinity:)
// (affinity defaults to downstream).
func NewTextSelectionCollapsed(offset int, affinity ...ui.TextAffinity) TextSelection {
	a := ui.TextAffinityDownstream
	if len(affinity) > 0 {
		a = affinity[0]
	}
	return TextSelection{TextRange: ui.NewTextRangeCollapsed(offset), BaseOffset: offset, ExtentOffset: offset, Affinity: a}
}

// NewTextSelectionFromPosition is TextSelection.fromPosition.
func NewTextSelectionFromPosition(position ui.TextPosition) TextSelection {
	return TextSelection{
		TextRange:  ui.NewTextRangeCollapsed(position.Offset),
		BaseOffset: position.Offset, ExtentOffset: position.Offset, Affinity: position.Affinity,
	}
}

// Base is the position at which the selection originates.
func (s TextSelection) Base() ui.TextPosition {
	var affinity ui.TextAffinity
	switch {
	case !s.IsValid() || s.BaseOffset == s.ExtentOffset:
		affinity = s.Affinity
	case s.BaseOffset < s.ExtentOffset:
		affinity = ui.TextAffinityDownstream
	default:
		affinity = ui.TextAffinityUpstream
	}
	return ui.TextPosition{Offset: s.BaseOffset, Affinity: affinity}
}

// Extent is the position at which the selection terminates.
func (s TextSelection) Extent() ui.TextPosition {
	var affinity ui.TextAffinity
	switch {
	case !s.IsValid() || s.BaseOffset == s.ExtentOffset:
		affinity = s.Affinity
	case s.BaseOffset < s.ExtentOffset:
		affinity = ui.TextAffinityUpstream
	default:
		affinity = ui.TextAffinityDownstream
	}
	return ui.TextPosition{Offset: s.ExtentOffset, Affinity: affinity}
}

func (s TextSelection) String() string {
	typeName := foundation.ObjectRuntimeType(s, "TextSelection")
	if !s.IsValid() {
		return typeName + ".invalid"
	}
	if s.IsCollapsed() {
		return typeName + ".collapsed(offset: " + strconv.Itoa(s.BaseOffset) + ", affinity: " + s.Affinity.String() +
			", isDirectional: " + strconv.FormatBool(s.IsDirectional) + ")"
	}
	return typeName + "(baseOffset: " + strconv.Itoa(s.BaseOffset) + ", extentOffset: " +
		strconv.Itoa(s.ExtentOffset) + ", isDirectional: " + strconv.FormatBool(s.IsDirectional) + ")"
}

// Equal is TextSelection's ==: all invalid selections are equal, and the
// affinity only matters for a collapsed selection.
func (s TextSelection) Equal(other TextSelection) bool {
	if !s.IsValid() {
		return !other.IsValid()
	}
	return other.BaseOffset == s.BaseOffset &&
		other.ExtentOffset == s.ExtentOffset &&
		(!s.IsCollapsed() || other.Affinity == s.Affinity) &&
		other.IsDirectional == s.IsDirectional
}

// TextSelectionCopyWithOptions are TextSelection.copyWith's parameters
// (nil: keep).
type TextSelectionCopyWithOptions struct {
	BaseOffset    *int
	ExtentOffset  *int
	Affinity      *ui.TextAffinity
	IsDirectional *bool
}

// CopyWith creates a new TextSelection based on this one but with the
// given fields replaced.
func (s TextSelection) CopyWith(opts ...TextSelectionCopyWithOptions) TextSelection {
	var o TextSelectionCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	base, extent, affinity, directional := s.BaseOffset, s.ExtentOffset, s.Affinity, s.IsDirectional
	if o.BaseOffset != nil {
		base = *o.BaseOffset
	}
	if o.ExtentOffset != nil {
		extent = *o.ExtentOffset
	}
	if o.Affinity != nil {
		affinity = *o.Affinity
	}
	if o.IsDirectional != nil {
		directional = *o.IsDirectional
	}
	return NewTextSelection(base, extent, TextSelectionOptions{Affinity: &affinity, IsDirectional: directional})
}

// ExpandTo returns the smallest TextSelection that this could expand to in
// order to include the given TextPosition.
func (s TextSelection) ExpandTo(position ui.TextPosition, extentAtIndex ...bool) TextSelection {
	atIndex := len(extentAtIndex) > 0 && extentAtIndex[0]
	// If position is already within in the selection, there's nothing to do.
	if position.Offset >= s.Start && position.Offset <= s.End {
		return s
	}
	normalized := s.BaseOffset <= s.ExtentOffset
	pick := func(cond bool, a, b int) *int {
		if cond {
			return &a
		}
		return &b
	}
	if position.Offset <= s.Start {
		// Here the position is somewhere before the selection: ..|..[...]....
		if atIndex {
			end, off, aff := s.End, position.Offset, position.Affinity
			return s.CopyWith(TextSelectionCopyWithOptions{BaseOffset: &end, ExtentOffset: &off, Affinity: &aff})
		}
		return s.CopyWith(TextSelectionCopyWithOptions{
			BaseOffset:   pick(normalized, position.Offset, s.BaseOffset),
			ExtentOffset: pick(normalized, s.ExtentOffset, position.Offset),
		})
	}
	// Here the position is somewhere after the selection: ....[...]..|..
	if atIndex {
		start, off, aff := s.Start, position.Offset, position.Affinity
		return s.CopyWith(TextSelectionCopyWithOptions{BaseOffset: &start, ExtentOffset: &off, Affinity: &aff})
	}
	return s.CopyWith(TextSelectionCopyWithOptions{
		BaseOffset:   pick(normalized, s.BaseOffset, position.Offset),
		ExtentOffset: pick(normalized, position.Offset, s.ExtentOffset),
	})
}

// ExtendTo keeps the selection's base where it is and moves the extent to
// position.
func (s TextSelection) ExtendTo(position ui.TextPosition) TextSelection {
	// If the selection's extent is at the position already, then nothing
	// happens.
	if s.Extent() == position {
		return s
	}
	off, aff := position.Offset, position.Affinity
	return s.CopyWith(TextSelectionCopyWithOptions{ExtentOffset: &off, Affinity: &aff})
}
