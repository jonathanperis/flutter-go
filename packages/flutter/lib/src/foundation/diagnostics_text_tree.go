package foundation

// Port of diagnostics.dart (Flutter 3.50.1), part 2 of 5: TextTreeConfiguration,
// the named text configurations, _PrefixedStringBuilder (word wrapping) and
// TextTreeRenderer. String lengths and offsets are Dart's UTF-16 code units.

import (
	"strconv"
	"strings"
)

// TextTreeConfiguration describes how a DiagnosticsTreeStyle is rendered as
// text art. Build one with NewTextTreeConfiguration.
type TextTreeConfiguration struct {
	PrefixLineOne                           string
	SuffixLineOne                           string
	PrefixOtherLines                        string
	PrefixLastChildLineOne                  string
	PrefixOtherLinesRootNode                string
	PropertyPrefixIfChildren                string
	PropertyPrefixNoChildren                string
	LinkCharacter                           string
	ChildLinkSpace                          string // spaces, as long as LinkCharacter
	LineBreak                               string
	LineBreakProperties                     bool
	BeforeName                              string
	AfterName                               string
	AfterDescriptionIfBody                  string
	AfterDescription                        string
	BeforeProperties                        string
	AfterProperties                         string
	MandatoryAfterProperties                string
	PropertySeparator                       string
	BodyIndent                              string
	ShowChildren                            bool
	AddBlankLineIfNoChildren                bool
	IsNameOnOwnLine                         bool
	Footer                                  string
	MandatoryFooter                         string
	IsBlankLineBetweenPropertiesAndChildren bool
}

// TextTreeConfigurationOptions are TextTreeConfiguration's named
// parameters. The first seven are required in Dart; nil pointers take the
// Dart defaults (LineBreak "\n", AfterName ":", the bools true).
type TextTreeConfigurationOptions struct {
	PrefixLineOne            string
	PrefixOtherLines         string
	PrefixLastChildLineOne   string
	PrefixOtherLinesRootNode string
	LinkCharacter            string
	PropertyPrefixIfChildren string
	PropertyPrefixNoChildren string

	LineBreak                               *string
	LineBreakProperties                     *bool
	AfterName                               *string
	AfterDescriptionIfBody                  string
	AfterDescription                        string
	BeforeProperties                        string
	AfterProperties                         string
	MandatoryAfterProperties                string
	PropertySeparator                       string
	BodyIndent                              string
	Footer                                  string
	ShowChildren                            *bool
	AddBlankLineIfNoChildren                *bool
	IsNameOnOwnLine                         bool
	IsBlankLineBetweenPropertiesAndChildren *bool
	BeforeName                              string
	SuffixLineOne                           string
	MandatoryFooter                         string
}

// NewTextTreeConfiguration is TextTreeConfiguration({...}).
func NewTextTreeConfiguration(o TextTreeConfigurationOptions) *TextTreeConfiguration {
	lineBreak := "\n"
	if o.LineBreak != nil {
		lineBreak = *o.LineBreak
	}
	afterName := ":"
	if o.AfterName != nil {
		afterName = *o.AfterName
	}
	return &TextTreeConfiguration{
		PrefixLineOne:                           o.PrefixLineOne,
		PrefixOtherLines:                        o.PrefixOtherLines,
		PrefixLastChildLineOne:                  o.PrefixLastChildLineOne,
		PrefixOtherLinesRootNode:                o.PrefixOtherLinesRootNode,
		LinkCharacter:                           o.LinkCharacter,
		PropertyPrefixIfChildren:                o.PropertyPrefixIfChildren,
		PropertyPrefixNoChildren:                o.PropertyPrefixNoChildren,
		LineBreak:                               lineBreak,
		LineBreakProperties:                     boolOr(o.LineBreakProperties, true),
		AfterName:                               afterName,
		AfterDescriptionIfBody:                  o.AfterDescriptionIfBody,
		AfterDescription:                        o.AfterDescription,
		BeforeProperties:                        o.BeforeProperties,
		AfterProperties:                         o.AfterProperties,
		MandatoryAfterProperties:                o.MandatoryAfterProperties,
		PropertySeparator:                       o.PropertySeparator,
		BodyIndent:                              o.BodyIndent,
		Footer:                                  o.Footer,
		ShowChildren:                            boolOr(o.ShowChildren, true),
		AddBlankLineIfNoChildren:                boolOr(o.AddBlankLineIfNoChildren, true),
		IsNameOnOwnLine:                         o.IsNameOnOwnLine,
		IsBlankLineBetweenPropertiesAndChildren: boolOr(o.IsBlankLineBetweenPropertiesAndChildren, true),
		BeforeName:                              o.BeforeName,
		SuffixLineOne:                           o.SuffixLineOne,
		MandatoryFooter:                         o.MandatoryFooter,
		ChildLinkSpace:                          strings.Repeat(" ", len16(o.LinkCharacter)),
	}
}

var (
	pFalse = new(bool)
	pEmpty = new(string)
)

// SparseTextConfiguration is sparseTextConfiguration, the default style.
var SparseTextConfiguration = NewTextTreeConfiguration(TextTreeConfigurationOptions{
	PrefixLineOne:            "├─",
	PrefixOtherLines:         " ",
	PrefixLastChildLineOne:   "└─",
	LinkCharacter:            "│",
	PropertyPrefixIfChildren: "│ ",
	PropertyPrefixNoChildren: "  ",
	PrefixOtherLinesRootNode: " ",
})

// DashedTextConfiguration is dashedTextConfiguration (offstage children).
var DashedTextConfiguration = NewTextTreeConfiguration(TextTreeConfigurationOptions{
	PrefixLineOne:          "╎╌",
	PrefixLastChildLineOne: "└╌",
	PrefixOtherLines:       " ",
	LinkCharacter:          "╎",
	// Intentionally not set as a dashed line as that would make the properties
	// look like they were disabled.
	PropertyPrefixIfChildren: "│ ",
	PropertyPrefixNoChildren: "  ",
	PrefixOtherLinesRootNode: " ",
})

// DenseTextConfiguration is denseTextConfiguration.
var DenseTextConfiguration = NewTextTreeConfiguration(TextTreeConfigurationOptions{
	PropertySeparator:                       ", ",
	BeforeProperties:                        "(",
	AfterProperties:                         ")",
	LineBreakProperties:                     pFalse,
	PrefixLineOne:                           "├",
	PrefixOtherLines:                        "",
	PrefixLastChildLineOne:                  "└",
	LinkCharacter:                           "│",
	PropertyPrefixIfChildren:                "│",
	PropertyPrefixNoChildren:                " ",
	PrefixOtherLinesRootNode:                "",
	AddBlankLineIfNoChildren:                pFalse,
	IsBlankLineBetweenPropertiesAndChildren: pFalse,
})

// TransitionTextConfiguration is transitionTextConfiguration: a box around
// a leaf node.
var TransitionTextConfiguration = NewTextTreeConfiguration(TextTreeConfigurationOptions{
	PrefixLineOne:                           "╞═╦══ ",
	PrefixLastChildLineOne:                  "╘═╦══ ",
	PrefixOtherLines:                        " ║ ",
	Footer:                                  " ╚═══════════",
	LinkCharacter:                           "│",
	PropertyPrefixIfChildren:                "",
	PropertyPrefixNoChildren:                "",
	PrefixOtherLinesRootNode:                "",
	AfterName:                               strp(" ═══"),
	AfterDescriptionIfBody:                  ":",
	BodyIndent:                              "  ",
	IsNameOnOwnLine:                         true,
	AddBlankLineIfNoChildren:                pFalse,
	IsBlankLineBetweenPropertiesAndChildren: pFalse,
})

// ErrorTextConfiguration is errorTextConfiguration: a box around an error.
var ErrorTextConfiguration = NewTextTreeConfiguration(TextTreeConfigurationOptions{
	PrefixLineOne:                           "╞═╦",
	PrefixLastChildLineOne:                  "╘═╦",
	PrefixOtherLines:                        " ║ ",
	Footer:                                  " ╚═══════════",
	LinkCharacter:                           "│",
	PropertyPrefixIfChildren:                "",
	PropertyPrefixNoChildren:                "",
	PrefixOtherLinesRootNode:                "",
	BeforeName:                              "══╡ ",
	SuffixLineOne:                           " ╞══",
	MandatoryFooter:                         "═════",
	AddBlankLineIfNoChildren:                pFalse,
	IsBlankLineBetweenPropertiesAndChildren: pFalse,
})

// WhitespaceTextConfiguration is whitespaceTextConfiguration: children
// indented two spaces, no line art.
var WhitespaceTextConfiguration = NewTextTreeConfiguration(TextTreeConfigurationOptions{
	PrefixLineOne:                           "",
	PrefixLastChildLineOne:                  "",
	PrefixOtherLines:                        " ",
	PrefixOtherLinesRootNode:                "  ",
	PropertyPrefixIfChildren:                "",
	PropertyPrefixNoChildren:                "",
	LinkCharacter:                           " ",
	AddBlankLineIfNoChildren:                pFalse,
	AfterDescriptionIfBody:                  ":",
	IsBlankLineBetweenPropertiesAndChildren: pFalse,
})

// FlatTextConfiguration is flatTextConfiguration: children not indented.
var FlatTextConfiguration = NewTextTreeConfiguration(TextTreeConfigurationOptions{
	PrefixLineOne:                           "",
	PrefixLastChildLineOne:                  "",
	PrefixOtherLines:                        "",
	PrefixOtherLinesRootNode:                "",
	PropertyPrefixIfChildren:                "",
	PropertyPrefixNoChildren:                "",
	LinkCharacter:                           "",
	AddBlankLineIfNoChildren:                pFalse,
	AfterDescriptionIfBody:                  ":",
	IsBlankLineBetweenPropertiesAndChildren: pFalse,
})

// SingleLineTextConfiguration is singleLineTextConfiguration: one line, no
// children.
var SingleLineTextConfiguration = NewTextTreeConfiguration(TextTreeConfigurationOptions{
	PropertySeparator:        ", ",
	BeforeProperties:         "(",
	AfterProperties:          ")",
	PrefixLineOne:            "",
	PrefixOtherLines:         "",
	PrefixLastChildLineOne:   "",
	LineBreak:                pEmpty,
	LineBreakProperties:      pFalse,
	AddBlankLineIfNoChildren: pFalse,
	ShowChildren:             pFalse,
	PropertyPrefixIfChildren: "  ",
	PropertyPrefixNoChildren: "  ",
	LinkCharacter:            "",
	PrefixOtherLinesRootNode: "",
})

// ErrorPropertyTextConfiguration is errorPropertyTextConfiguration: the
// name on its own line, then the description and properties.
var ErrorPropertyTextConfiguration = NewTextTreeConfiguration(TextTreeConfigurationOptions{
	PropertySeparator:        ", ",
	BeforeProperties:         "(",
	AfterProperties:          ")",
	PrefixLineOne:            "",
	PrefixOtherLines:         "",
	PrefixLastChildLineOne:   "",
	LineBreakProperties:      pFalse,
	AddBlankLineIfNoChildren: pFalse,
	ShowChildren:             pFalse,
	PropertyPrefixIfChildren: "  ",
	PropertyPrefixNoChildren: "  ",
	LinkCharacter:            "",
	PrefixOtherLinesRootNode: "",
	IsNameOnOwnLine:          true,
})

// ShallowTextConfiguration is shallowTextConfiguration: properties on
// several lines, no children.
var ShallowTextConfiguration = NewTextTreeConfiguration(TextTreeConfigurationOptions{
	PrefixLineOne:                           "",
	PrefixLastChildLineOne:                  "",
	PrefixOtherLines:                        " ",
	PrefixOtherLinesRootNode:                "  ",
	PropertyPrefixIfChildren:                "",
	PropertyPrefixNoChildren:                "",
	LinkCharacter:                           " ",
	AddBlankLineIfNoChildren:                pFalse,
	AfterDescriptionIfBody:                  ":",
	IsBlankLineBetweenPropertiesAndChildren: pFalse,
	ShowChildren:                            pFalse,
})

type wordWrapParseMode int

const (
	wordWrapInSpace wordWrapParseMode = iota
	wordWrapInWord
	wordWrapAtBreak
)

// prefixedStringBuilder is _PrefixedStringBuilder: builds a string with one
// prefix for the first line and another for the rest, word wrapping the
// spans written with allowWrap.
type prefixedStringBuilder struct {
	prefixLineOne        string
	prefixOtherLines_    string
	nextPrefixOtherLines *string
	wrapWidth            int
	hasWrapWidth         bool

	buffer          strings.Builder
	currentLine     strings.Builder
	currentLineLen  int   // UTF-16 length of currentLine
	wrappableRanges []int // start/end pairs within currentLine
	numLines        int
}

func newPrefixedStringBuilder(prefixLineOne, prefixOtherLines string, wrapWidth int) *prefixedStringBuilder {
	return &prefixedStringBuilder{prefixLineOne: prefixLineOne, prefixOtherLines_: prefixOtherLines,
		wrapWidth: wrapWidth, hasWrapWidth: true}
}

func (b *prefixedStringBuilder) prefixOtherLines() string {
	if b.nextPrefixOtherLines != nil {
		return *b.nextPrefixOtherLines
	}
	return b.prefixOtherLines_
}

func (b *prefixedStringBuilder) setPrefixOtherLines(prefix string) {
	b.prefixOtherLines_ = prefix
	b.nextPrefixOtherLines = nil
}

func (b *prefixedStringBuilder) incrementPrefixOtherLines(suffix string, updateCurrentLine bool) {
	if b.currentLineLen == 0 || updateCurrentLine {
		b.prefixOtherLines_ = b.prefixOtherLines() + suffix
		b.nextPrefixOtherLines = nil
	} else {
		next := b.prefixOtherLines() + suffix
		b.nextPrefixOtherLines = &next
	}
}

func (b *prefixedStringBuilder) requiresMultipleLines() bool {
	return b.numLines > 1 ||
		(b.numLines == 1 && b.currentLineLen != 0) ||
		(b.currentLineLen+len16(b.getCurrentPrefix(true)) > b.wrapWidth)
}

func (b *prefixedStringBuilder) isCurrentLineEmpty() bool { return b.currentLineLen == 0 }

func (b *prefixedStringBuilder) finalizeLine(addTrailingLineBreak bool) {
	firstLine := b.buffer.Len() == 0
	text := b.currentLine.String()
	b.currentLine.Reset()
	b.currentLineLen = 0

	if len(b.wrappableRanges) == 0 {
		// Fast path. There were no wrappable spans of text.
		b.writeLine(text, addTrailingLineBreak, firstLine)
		return
	}
	startOffset := len16(b.prefixOtherLines_)
	if firstLine {
		startOffset = len16(b.prefixLineOne)
	}
	lines := wordWrapLine(text, b.wrappableRanges, b.wrapWidth, startOffset, len16(b.prefixOtherLines_))
	for i, line := range lines {
		b.writeLine(line, addTrailingLineBreak || i+1 < len(lines), firstLine)
	}
	b.wrappableRanges = b.wrappableRanges[:0]
}

// wordWrapLine is _PrefixedStringBuilder._wordWrapLine: wraps message at
// width, breaking only at spaces inside wrapRanges.
func wordWrapLine(messageStr string, wrapRanges []int, width, startOffset, otherLineOffset int) []string {
	message := toUTF16(messageStr)
	if len(message)+startOffset < width {
		// Nothing to do. The line doesn't wrap.
		return []string{messageStr}
	}
	var wrappedLine []string
	startForLengthCalculations := -startOffset
	addPrefix := false
	index := 0
	mode := wordWrapInSpace
	var lastWordStart int
	lastWordEnd, hasLastWordEnd := 0, false
	start := 0

	currentChunk := 0

	// This helper is called with increasing indexes.
	noWrap := func(index int) bool {
		for {
			if currentChunk >= len(wrapRanges) {
				return true
			}
			if index < wrapRanges[currentChunk+1] {
				break // Found nearest chunk.
			}
			currentChunk += 2
		}
		return index < wrapRanges[currentChunk]
	}

	for {
		switch mode {
		case wordWrapInSpace: // at start of break point (or start of line); can't break until next break
			for index < len(message) && message[index] == ' ' {
				index++
			}
			lastWordStart = index
			mode = wordWrapInWord
		case wordWrapInWord: // looking for a good break point. Treat all text
			for index < len(message) && (message[index] != ' ' || noWrap(index)) {
				index++
			}
			mode = wordWrapAtBreak
		case wordWrapAtBreak: // at start of break point
			if index-startForLengthCalculations > width || index == len(message) {
				// we are over the width line, so break
				if index-startForLengthCalculations <= width || !hasLastWordEnd {
					// we should use this point, because either it doesn't actually go over the
					// end (last line), or it does, but there was no earlier break point
					lastWordEnd, hasLastWordEnd = index, true
				}
				wrappedLine = append(wrappedLine, fromUTF16(message[start:lastWordEnd]))
				addPrefix = true
				if lastWordEnd >= len(message) {
					return wrappedLine
				}
				// just yielded a line
				if lastWordEnd == index {
					// we broke at current position
					// eat all the wrappable spaces, then set our start point
					// Even if some of the spaces are not wrappable that is ok.
					for index < len(message) && message[index] == ' ' {
						index++
					}
					start = index
					mode = wordWrapInWord
				} else {
					// we broke at the previous break point, and we're at the start of a new one
					if lastWordStart <= lastWordEnd {
						panic("foundation: word wrap: lastWordStart must be > lastWordEnd")
					}
					start = lastWordStart
					mode = wordWrapAtBreak
				}
				startForLengthCalculations = start - otherLineOffset
				_ = addPrefix
				hasLastWordEnd = false
			} else {
				// save this break point, we're not yet over the line width
				lastWordEnd, hasLastWordEnd = index, true
				// skip to the end of this break point
				mode = wordWrapInSpace
			}
		}
	}
}

// write writes s; with allowWrap the text may be wrapped at wrapWidth.
func (b *prefixedStringBuilder) write(s string, allowWrap bool) {
	if s == "" {
		return
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i > 0 {
			b.finalizeLine(true)
			b.updatePrefix()
		}
		if line != "" {
			if allowWrap && b.hasWrapWidth {
				wrapStart := b.currentLineLen
				wrapEnd := wrapStart + len16(line)
				if n := len(b.wrappableRanges); n > 0 && b.wrappableRanges[n-1] == wrapStart {
					// Extend last range.
					b.wrappableRanges[n-1] = wrapEnd
				} else {
					b.wrappableRanges = append(b.wrappableRanges, wrapStart, wrapEnd)
				}
			}
			b.currentLine.WriteString(line)
			b.currentLineLen += len16(line)
		}
	}
}

func (b *prefixedStringBuilder) updatePrefix() {
	if b.nextPrefixOtherLines != nil {
		b.prefixOtherLines_ = *b.nextPrefixOtherLines
		b.nextPrefixOtherLines = nil
	}
}

func (b *prefixedStringBuilder) writeLine(line string, includeLineBreak, firstLine bool) {
	line = b.getCurrentPrefix(firstLine) + line
	b.buffer.WriteString(dartTrimRight(line))
	if includeLineBreak {
		b.buffer.WriteString("\n")
	}
	b.numLines++
}

func (b *prefixedStringBuilder) getCurrentPrefix(firstLine bool) string {
	if b.buffer.Len() == 0 {
		return b.prefixLineOne
	}
	return b.prefixOtherLines_
}

// writeRawLines writes lines that already carry their prefixes, ending
// with a line break.
func (b *prefixedStringBuilder) writeRawLines(lines string) {
	if lines == "" {
		return
	}
	if b.currentLineLen != 0 {
		b.finalizeLine(true)
	}
	b.buffer.WriteString(lines)
	if !strings.HasSuffix(lines, "\n") {
		b.buffer.WriteString("\n")
	}
	b.numLines++
	b.updatePrefix()
}

// writeStretched finishes the current line with text, repeating its last
// character up to targetLineLength.
func (b *prefixedStringBuilder) writeStretched(text string, targetLineLength int) {
	b.write(text, false)
	currentLineLength := b.currentLineLen + len16(b.getCurrentPrefix(b.buffer.Len() == 0))
	targetLength := targetLineLength - currentLineLength
	if targetLength > 0 {
		u := toUTF16(text)
		lastChar := fromUTF16(u[len(u)-1:])
		b.currentLine.WriteString(strings.Repeat(lastChar, targetLength))
		b.currentLineLen += targetLength * len16(lastChar)
	}
	// Mark the entire line as not wrappable.
	b.wrappableRanges = b.wrappableRanges[:0]
}

func (b *prefixedStringBuilder) build() string {
	if b.currentLineLen != 0 {
		b.finalizeLine(false)
	}
	return b.buffer.String()
}

func isSingleLine(style DiagnosticsTreeStyle) bool { return style == DiagnosticsTreeStyleSingleLine }

// TextTreeRenderer renders trees of DiagnosticsNode as text art.
type TextTreeRenderer struct {
	wrapWidth                     int
	wrapWidthProperties           int
	minLevel                      DiagnosticLevel
	maxDescendentsTruncatableNode int
}

// TextTreeRendererOptions are TextTreeRenderer's named parameters (nil:
// minLevel debug, wrapWidth 100, wrapWidthProperties 65,
// maxDescendentsTruncatableNode -1).
type TextTreeRendererOptions struct {
	MinLevel                      *DiagnosticLevel
	WrapWidth                     *int
	WrapWidthProperties           *int
	MaxDescendentsTruncatableNode *int
}

// NewTextTreeRenderer is TextTreeRenderer({...}).
func NewTextTreeRenderer(opts ...TextTreeRendererOptions) *TextTreeRenderer {
	var o TextTreeRendererOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return &TextTreeRenderer{
		minLevel:                      levelOr(o.MinLevel, DiagnosticLevelDebug),
		wrapWidth:                     intOr(o.WrapWidth, 100),
		wrapWidthProperties:           intOr(o.WrapWidthProperties, 65),
		maxDescendentsTruncatableNode: intOr(o.MaxDescendentsTruncatableNode, -1),
	}
}

func (r *TextTreeRenderer) childTextConfiguration(child DiagnosticsNode, textStyle *TextTreeConfiguration) *TextTreeConfiguration {
	childStyle := child.Style()
	if isSingleLine(childStyle) || childStyle == DiagnosticsTreeStyleErrorProperty {
		return textStyle
	}
	return child.TextTreeConfiguration()
}

// TextTreeRendererRenderOptions are render's named parameters.
type TextTreeRendererRenderOptions struct {
	PrefixLineOne       string
	PrefixOtherLines    *string // nil: PrefixLineOne
	ParentConfiguration *TextTreeConfiguration
}

// Render renders node to a string ("" in release builds).
func (r *TextTreeRenderer) Render(node DiagnosticsNode, opts ...TextTreeRendererRenderOptions) string {
	if KReleaseMode {
		return ""
	}
	var o TextTreeRendererRenderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return r.debugRender(node, o.PrefixLineOne, o.PrefixOtherLines, o.ParentConfiguration)
}

func (r *TextTreeRenderer) render(node DiagnosticsNode, prefixLineOne string, prefixOtherLines *string, parentConfiguration *TextTreeConfiguration) string {
	if KReleaseMode {
		return ""
	}
	return r.debugRender(node, prefixLineOne, prefixOtherLines, parentConfiguration)
}

func (r *TextTreeRenderer) debugRender(node DiagnosticsNode, prefixLineOne string, prefixOtherLinesP *string, parentConfiguration *TextTreeConfiguration) string {
	isSingle := isSingleLine(node.Style()) && !(parentConfiguration != nil && parentConfiguration.LineBreakProperties)
	prefixOtherLines := prefixLineOne
	if prefixOtherLinesP != nil {
		prefixOtherLines = *prefixOtherLinesP
	}
	if lp := node.LinePrefix(); lp != "" {
		prefixLineOne += lp
		prefixOtherLines += lp
	}

	config := node.TextTreeConfiguration()
	if prefixOtherLines == "" {
		prefixOtherLines += config.PrefixOtherLinesRootNode
	}

	if node.Style() == DiagnosticsTreeStyleTruncateChildren {
		// This style is different enough that it isn't worthwhile to reuse the
		// existing logic.
		var descendants []string
		const maxDepth = 5
		depth := 0
		const maxLines = 25
		lines := 0
		var visitor func(node DiagnosticsNode)
		visitor = func(node DiagnosticsNode) {
			for _, child := range node.GetChildren() {
				if lines < maxLines {
					depth++
					descendants = append(descendants, prefixOtherLines+strings.Repeat("  ", depth)+child.String())
					if depth < maxDepth {
						visitor(child)
					}
					depth--
				} else if lines == maxLines {
					descendants = append(descendants,
						prefixOtherLines+"  ...(descendants list truncated after "+strconv.Itoa(lines)+" lines)")
				}
				lines++
			}
		}
		visitor(node)
		var information strings.Builder
		information.WriteString(prefixLineOne)
		name := nameOrNull(node)
		if lines > 1 {
			information.WriteString("This " + name + " had the following descendants (showing up to depth " + strconv.Itoa(maxDepth) + "):\n")
		} else if len(descendants) == 1 {
			information.WriteString("This " + name + " had the following child:\n")
		} else {
			information.WriteString("This " + name + " has no descendants.\n")
		}
		information.WriteString(strings.Join(descendants, "\n"))
		return information.String()
	}
	builder := newPrefixedStringBuilder(prefixLineOne, prefixOtherLines,
		max(r.wrapWidth, len16(prefixOtherLines)+r.wrapWidthProperties))

	children := node.GetChildren()

	description := node.ToDescription(ToDescriptionOptions{ParentConfiguration: parentConfiguration})
	if config.BeforeName != "" {
		builder.write(config.BeforeName, false)
	}
	wrapName := !isSingle && node.AllowNameWrap()
	wrapDescription := !isSingle && node.AllowWrap()
	uppercaseTitle := node.Style() == DiagnosticsTreeStyleError
	name, hasName := nodeName(node)
	if uppercaseTitle && hasName {
		name = dartToUpperCase(name)
	}
	if description == "" {
		if node.ShowName() && hasName {
			builder.write(name, wrapName)
		}
	} else {
		includeName := false
		if hasName && name != "" && node.ShowName() {
			includeName = true
			builder.write(name, wrapName)
			if node.ShowSeparator() {
				builder.write(config.AfterName, wrapName)
			}
			sep := " "
			if config.IsNameOnOwnLine || strings.Contains(description, "\n") {
				sep = "\n"
			}
			builder.write(sep, wrapName)
		}
		if !isSingle && builder.requiresMultipleLines() && !builder.isCurrentLineEmpty() {
			// Make sure there is a break between the current line and next one if
			// there is not one already.
			builder.write("\n", false)
		}
		if includeName {
			p := config.PropertyPrefixIfChildren
			if len(children) == 0 {
				p = config.PropertyPrefixNoChildren
			}
			builder.incrementPrefixOtherLines(p, true)
		}

		if uppercaseTitle {
			description = dartToUpperCase(description)
		}
		builder.write(dartTrimRight(description), wrapDescription)

		if !includeName {
			p := config.PropertyPrefixIfChildren
			if len(children) == 0 {
				p = config.PropertyPrefixNoChildren
			}
			builder.incrementPrefixOtherLines(p, false)
		}
	}
	if config.SuffixLineOne != "" {
		builder.writeStretched(config.SuffixLineOne, builder.wrapWidth)
	}

	var propertiesIterable []DiagnosticsNode
	for _, n := range node.GetProperties() {
		if !n.IsFiltered(r.minLevel) {
			propertiesIterable = append(propertiesIterable, n)
		}
	}
	var properties []DiagnosticsNode
	if r.maxDescendentsTruncatableNode >= 0 && node.AllowTruncate() {
		if len(propertiesIterable) < r.maxDescendentsTruncatableNode {
			properties = append(properties, propertiesIterable[:min(len(propertiesIterable), r.maxDescendentsTruncatableNode)]...)
			properties = append(properties, NewDiagnosticsNodeMessage("..."))
		} else {
			properties = propertiesIterable
		}
		if r.maxDescendentsTruncatableNode < len(children) {
			children = append(children[:r.maxDescendentsTruncatableNode:r.maxDescendentsTruncatableNode], NewDiagnosticsNodeMessage("..."))
		}
	} else {
		properties = propertiesIterable
	}

	// If the node does not show a separator and there is no description then
	// we should not place a separator between the name and the value.
	// Essentially in this case the properties are treated a bit like a value.
	if (len(properties) != 0 || len(children) != 0 || node.EmptyBodyDescription() != nil) &&
		(node.ShowSeparator() || description != "") {
		builder.write(config.AfterDescriptionIfBody, false)
	}

	if config.LineBreakProperties {
		builder.write(config.LineBreak, false)
	}

	if len(properties) != 0 {
		builder.write(config.BeforeProperties, false)
	}

	builder.incrementPrefixOtherLines(config.BodyIndent, false)

	if ebd := node.EmptyBodyDescription(); ebd != nil && len(properties) == 0 && len(children) == 0 && prefixLineOne != "" {
		builder.write(*ebd, false)
		if config.LineBreakProperties {
			builder.write(config.LineBreak, false)
		}
	}

	for i, property := range properties {
		if i > 0 {
			builder.write(config.PropertySeparator, false)
		}

		propertyStyle := property.TextTreeConfiguration()
		if isSingleLine(property.Style()) {
			// We have to treat single line properties slightly differently to deal
			// with cases where a single line properties output may not have single
			// linebreak.
			other := propertyStyle.ChildLinkSpace + propertyStyle.PrefixOtherLines
			propertyRender := r.render(property, propertyStyle.PrefixLineOne, &other, config)
			propertyLines := strings.Split(propertyRender, "\n")
			if len(propertyLines) == 1 && !config.LineBreakProperties {
				builder.write(propertyLines[0], false)
			} else {
				builder.write(propertyRender, false)
				if !strings.HasSuffix(propertyRender, "\n") {
					builder.write("\n", false)
				}
			}
		} else {
			other := builder.prefixOtherLines() + propertyStyle.ChildLinkSpace + propertyStyle.PrefixOtherLines
			propertyRender := r.render(property, builder.prefixOtherLines()+propertyStyle.PrefixLineOne, &other, config)
			builder.writeRawLines(propertyRender)
		}
	}
	if len(properties) != 0 {
		builder.write(config.AfterProperties, false)
	}

	builder.write(config.MandatoryAfterProperties, false)

	if !config.LineBreakProperties {
		builder.write(config.LineBreak, false)
	}

	prefixChildren := config.BodyIndent
	prefixChildrenRaw := prefixOtherLines + prefixChildren
	if len(children) == 0 &&
		config.AddBlankLineIfNoChildren &&
		builder.requiresMultipleLines() &&
		dartTrimRight(builder.prefixOtherLines()) != "" {
		builder.write(config.LineBreak, false)
	}

	if len(children) != 0 && config.ShowChildren {
		if config.IsBlankLineBetweenPropertiesAndChildren &&
			len(properties) != 0 &&
			children[0].TextTreeConfiguration().IsBlankLineBetweenPropertiesAndChildren {
			builder.write(config.LineBreak, false)
		}

		builder.setPrefixOtherLines(prefixOtherLines)

		for i, child := range children {
			childConfig := r.childTextConfiguration(child, config)
			if i == len(children)-1 {
				lastChildPrefixLineOne := prefixChildrenRaw + childConfig.PrefixLastChildLineOne
				childPrefixOtherLines := prefixChildrenRaw + childConfig.ChildLinkSpace + childConfig.PrefixOtherLines
				builder.writeRawLines(r.render(child, lastChildPrefixLineOne, &childPrefixOtherLines, config))
				if childConfig.Footer != "" {
					builder.setPrefixOtherLines(prefixChildrenRaw)
					builder.write(childConfig.ChildLinkSpace+childConfig.Footer, false)
					if childConfig.MandatoryFooter != "" {
						builder.writeStretched(childConfig.MandatoryFooter,
							max(builder.wrapWidth, r.wrapWidthProperties+len16(childPrefixOtherLines)))
					}
					builder.write(config.LineBreak, false)
				}
			} else {
				nextChildStyle := r.childTextConfiguration(children[i+1], config)
				childPrefixLineOne := prefixChildrenRaw + childConfig.PrefixLineOne
				childPrefixOtherLines := prefixChildrenRaw + nextChildStyle.LinkCharacter + childConfig.PrefixOtherLines
				builder.writeRawLines(r.render(child, childPrefixLineOne, &childPrefixOtherLines, config))
				if childConfig.Footer != "" {
					builder.setPrefixOtherLines(prefixChildrenRaw)
					builder.write(childConfig.LinkCharacter+childConfig.Footer, false)
					if childConfig.MandatoryFooter != "" {
						builder.writeStretched(childConfig.MandatoryFooter,
							max(builder.wrapWidth, r.wrapWidthProperties+len16(childPrefixOtherLines)))
					}
					builder.write(config.LineBreak, false)
				}
			}
		}
	}
	if parentConfiguration == nil && config.MandatoryFooter != "" {
		builder.writeStretched(config.MandatoryFooter, builder.wrapWidth)
		builder.write(config.LineBreak, false)
	}
	return builder.build()
}
