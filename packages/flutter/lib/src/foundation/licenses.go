package foundation

// Port of licenses.dart (Flutter 3.50.1).

import (
	"iter"
	"slices"
	"strings"

	"github.com/jonathanperis/flutter-go/src/async"
)

// LicenseEntryCollector returns a stream of license entries; it is called
// each time LicenseRegistry.Licenses is listened to.
type LicenseEntryCollector func() async.Stream[LicenseEntry]

// LicenseParagraph is a string with an indentation level: a paragraph of a
// license.
type LicenseParagraph struct {
	// Text is the text of the paragraph, with no line breaks.
	Text string
	// Indent is how many steps to indent the paragraph, or
	// LicenseParagraphCenteredIndent to center it.
	Indent int
}

// LicenseParagraphCenteredIndent is LicenseParagraph.centeredIndent: an
// Indent meaning "centered".
const LicenseParagraphCenteredIndent = -1

// LicenseEntry is the license text of one or more packages.
type LicenseEntry interface {
	// Packages are the names of the packages this license applies to.
	Packages() iter.Seq[string]
	// Paragraphs are the paragraphs of the license.
	Paragraphs() iter.Seq[LicenseParagraph]
}

type licenseParserState int

const (
	licenseBeforeParagraph licenseParserState = iota
	licenseInParagraph
)

// LicenseEntryWithLineBreaks is a license whose paragraphs are separated
// by blank lines (or form feeds) and whose lines are wrapped with line
// breaks; indentation is guessed from leading spaces and tabs.
type LicenseEntryWithLineBreaks struct {
	packages []string
	// Text is the license text.
	Text string
}

// NewLicenseEntryWithLineBreaks is LicenseEntryWithLineBreaks(packages, text).
func NewLicenseEntryWithLineBreaks(packages []string, text string) *LicenseEntryWithLineBreaks {
	return &LicenseEntryWithLineBreaks{packages: packages, Text: text}
}

func (e *LicenseEntryWithLineBreaks) Packages() iter.Seq[string] { return slices.Values(e.packages) }

// Paragraphs splits Text into paragraphs (the whole list is computed on
// each call, as in Dart).
//
// The text is scanned byte by byte where Dart scans UTF-16 code units: the
// only characters the parser looks at are ASCII, which never occur inside
// a multi-byte UTF-8 sequence, so the paragraphs are the same.
func (e *LicenseEntryWithLineBreaks) Paragraphs() iter.Seq[LicenseParagraph] {
	return slices.Values(e.paragraphs())
}

func (e *LicenseEntryWithLineBreaks) paragraphs() []LicenseParagraph {
	text := e.Text
	lineStart := 0
	currentPosition := 0
	lastLineIndent := 0
	currentLineIndent := 0
	var currentParagraphIndentation *int
	state := licenseBeforeParagraph
	var lines []string
	result := []LicenseParagraph{}

	addLine := func() {
		lineEnd := currentPosition
		// Drop a trailing CR so CRLF line endings are treated like LF line
		// endings.
		if text[lineEnd-1] == '\r' {
			lineEnd--
		}
		dartAssert(lineStart < lineEnd, nil)
		lines = append(lines, text[lineStart:lineEnd])
	}
	getParagraph := func() LicenseParagraph {
		dartAssert(len(lines) > 0, nil)
		dartAssert(currentParagraphIndentation != nil, nil)
		p := LicenseParagraph{Text: strings.Join(lines, " "), Indent: *currentParagraphIndentation}
		dartAssert(strings.TrimLeftFunc(p.Text, isDartWhitespace) == p.Text, nil)
		dartAssert(p.Text != "", nil)
		lines = lines[:0]
		return p
	}

	for currentPosition < len(text) {
		switch state {
		case licenseBeforeParagraph:
			dartAssert(lineStart == currentPosition, nil)
			switch c := text[currentPosition]; c {
			case ' ':
				lineStart = currentPosition + 1
				currentLineIndent++
				state = licenseBeforeParagraph
			case '\t':
				lineStart = currentPosition + 1
				currentLineIndent += 8
				state = licenseBeforeParagraph
			case '\r', '\n', '\f':
				if len(lines) > 0 {
					result = append(result, getParagraph())
				}
				if c == '\r' && currentPosition < len(text)-1 && text[currentPosition+1] == '\n' {
					currentPosition++
				}
				lastLineIndent = 0
				currentLineIndent = 0
				currentParagraphIndentation = nil
				lineStart = currentPosition + 1
				state = licenseBeforeParagraph
			default:
				if c == '[' {
					// This is a bit of a hack for the LGPL 2.1, which does
					// something like this:
					//
					//   [this is a
					//    single paragraph]
					//
					// ...near the top.
					currentLineIndent++
				}
				// startParagraph:
				if len(lines) > 0 && currentLineIndent > lastLineIndent {
					result = append(result, getParagraph())
					currentParagraphIndentation = nil
				}
				// The following is a wild heuristic for guessing the
				// indentation level. It happens to work for common variants
				// of the BSD and LGPL licenses.
				if currentParagraphIndentation == nil {
					indent := currentLineIndent / 3
					if currentLineIndent > 10 {
						indent = LicenseParagraphCenteredIndent
					}
					currentParagraphIndentation = &indent
				}
				state = licenseInParagraph
			}
		case licenseInParagraph:
			switch text[currentPosition] {
			case '\n':
				addLine()
				lastLineIndent = currentLineIndent
				currentLineIndent = 0
				lineStart = currentPosition + 1
				state = licenseBeforeParagraph
			case '\f':
				addLine()
				result = append(result, getParagraph())
				lastLineIndent = 0
				currentLineIndent = 0
				currentParagraphIndentation = nil
				lineStart = currentPosition + 1
				state = licenseBeforeParagraph
			default:
				state = licenseInParagraph
			}
		}
		currentPosition++
	}
	switch state {
	case licenseBeforeParagraph:
		if len(lines) > 0 {
			result = append(result, getParagraph())
		}
	case licenseInParagraph:
		addLine()
		result = append(result, getParagraph())
	}
	return result
}

// licenseRegistry holds LicenseRegistry's static members.
type licenseRegistry struct{}

// LicenseRegistry is the registry of the licenses of the packages the app
// uses (shown by the about box's license page).
var LicenseRegistry licenseRegistry

var licenseCollectors []LicenseEntryCollector

// AddLicense adds a collector, which is called each time Licenses is
// listened to.
func (licenseRegistry) AddLicense(collector LicenseEntryCollector) {
	if licenseCollectors == nil {
		licenseCollectors = []LicenseEntryCollector{}
	}
	licenseCollectors = append(licenseCollectors, collector)
}

// Licenses is a stream of every license entry: each collector's stream in
// turn (an empty stream if there is no collector).
func (licenseRegistry) Licenses() async.Stream[LicenseEntry] {
	if licenseCollectors == nil {
		return async.StreamEmpty[LicenseEntry]()
	}
	var controller *async.StreamController[LicenseEntry]
	controller = async.NewStreamController[LicenseEntry](async.StreamControllerOptions{
		OnListen: func() {
			async.AsyncVoid(func() {
				for _, collector := range licenseCollectors {
					controller.AddStream(collector(), false).Await()
				}
				controller.Close().Await()
			})
		},
	})
	return controller.Stream()
}

// Reset forgets every collector (for tests).
func (licenseRegistry) Reset() { licenseCollectors = nil }
