// Port of packages/flutter/test/foundation/annotations_test.dart (all 3
// tests).
package foundation

import (
	"slices"
	"strings"
	"testing"
)

func TestCategoryConstructor(t *testing.T) {
	sections := []string{"First section", "Second section", "Third section"}
	category := Category{Sections: sections}
	if !slices.Equal(category.Sections, sections) {
		t.Fatal(category.Sections)
	}
}

func TestDocumentationIconConstructor(t *testing.T) {
	docIcon := DocumentationIcon{URL: "Test String"}
	if !strings.Contains(docIcon.URL, "Test String") {
		t.Fatal(docIcon.URL)
	}
}

func TestSummaryConstructor(t *testing.T) {
	summary := Summary{Text: "Test String"}
	if !strings.Contains(summary.Text, "Test String") {
		t.Fatal(summary.Text)
	}
}
