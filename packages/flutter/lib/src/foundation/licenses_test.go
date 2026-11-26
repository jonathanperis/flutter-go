// Port of packages/flutter/test/foundation/licenses_test.dart (all 4
// tests). The `async*` collectors of "LicenseRegistry" are
// Stream.fromIterable streams (async* generators are not ported); the
// difftest licenses compares paragraphs and the registry's delivery order
// with Flutter's.
package foundation

import (
	"slices"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/flutter_go/async"
)

func paragraphsOf(text string) []LicenseParagraph {
	return slices.Collect(NewLicenseEntryWithLineBreaks(nil, text).Paragraphs())
}

func TestLicenseEntryWithLineBreaksMostCases(t *testing.T) {
	// There's some trailing spaces in this string; they are escaped as
	// \u0020 in Flutter's test.
	paragraphs := paragraphsOf(`A
A
A
  B
B
B
   C
C
C
                  D
D
D

E
E
 F
  G
 G
G

[H
 H
 H]
` + "\u0020\u0020" + `
I` + "\u000c" + `J
` + "\u000c" + `K
K
` + "\u000c" + `
L
L L
L  L
L   L
L    L
L     L

   M
M` + "\u0020\u0020\u0020" + `
M` + "\u0020\u0020\u0020\u0020" + `

N

O
O


P



QQQ

RR RRR RRRR RRRRR
R

S

   T

      U
         V

        W

       X
` + "\u0020\u0020\u0020\u0020\u0020\u0020" + `
      Y`)
	want := []LicenseParagraph{
		{"A A A", 0}, {"B B B", 0}, {"C C C", 1}, {"D D D", LicenseParagraphCenteredIndent},
		{"E E", 0}, {"F", 0}, {"G G G", 0}, {"[H H H]", 0}, {"I", 0}, {"J", 0}, {"K K", 0},
		{"L L L L  L L   L L    L L     L", 0}, {"M M    M    ", 1}, {"N", 0}, {"O O", 0},
		{"P", 0}, {"QQQ", 0}, {"RR RRR RRRR RRRRR R", 0}, {"S", 0}, {"T", 1}, {"U", 2},
		{"V", 3}, {"W", 2}, {"X", 2}, {"Y", 2},
	}
	for i := range max(len(want), len(paragraphs)) {
		if i >= len(want) || i >= len(paragraphs) || paragraphs[i] != want[i] {
			t.Fatalf("paragraph %d:\n got  %q\n want %q", i, paragraphs, want)
		}
	}
}

func TestLicenseEntryWithLineBreaksLeadingAndTrailingWhitespace(t *testing.T) {
	equal(t, len(paragraphsOf("    \n\n    ")), 0)
	equal(t, len(paragraphsOf("    \r\n\r\n    ")), 0)
	for _, text := range []string{"    \nA\n    ", "\n\n\nA\n\n\n"} {
		p := paragraphsOf(text)
		if len(p) != 1 || p[0] != (LicenseParagraph{"A", 0}) {
			t.Fatalf("%q: %q", text, p)
		}
	}
}

func TestLicenseEntryWithLineBreaksCRLFInsideParagraphs(t *testing.T) {
	p := paragraphsOf("A\r\nB\r\nC\r\n\r\nD\r\nE")
	if len(p) != 2 || p[0] != (LicenseParagraph{"A B C", 0}) || p[1] != (LicenseParagraph{"D E", 0}) {
		t.Fatalf("%q", p)
	}
	if strings.Contains(p[0].Text, "\r") || strings.Contains(p[1].Text, "\r") {
		t.Fatal("CR kept")
	}
}

func TestLicenseRegistry(t *testing.T) {
	LicenseRegistry.Reset()
	defer LicenseRegistry.Reset()
	entries := func(texts ...string) LicenseEntryCollector {
		return func() async.Stream[LicenseEntry] {
			var es []LicenseEntry
			for _, text := range texts {
				es = append(es, NewLicenseEntryWithLineBreaks(nil, text))
			}
			return async.StreamFromIterable(slices.Values(es))
		}
	}
	var first, second, licenses []LicenseEntry
	err := runLoop(t, func() {
		first = async.StreamToList(LicenseRegistry.Licenses()).Await()
		LicenseRegistry.AddLicense(entries("A", "B"))
		LicenseRegistry.AddLicense(entries("C", "D"))
		second = async.StreamToList(LicenseRegistry.Licenses()).Await()
		licenses = async.StreamToList(LicenseRegistry.Licenses()).Await()
	})
	if err != nil {
		t.Fatal(err)
	}
	equal(t, len(first), 0)
	equal(t, len(second), 4)
	equal(t, len(licenses), 4)
	for i, want := range []string{"A", "B", "C", "D"} {
		p := slices.Collect(licenses[i].Paragraphs())
		if len(p) != 1 || p[0].Text != want {
			t.Fatalf("license %d: %q", i, p)
		}
	}
}
