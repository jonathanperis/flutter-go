package impeller

import "testing"

// The mirrors in txt_mirror.h must match the engine objects the SDK
// builds: values set through the SDK's public setters are read back through
// them (a layout mismatch would read garbage or crash here first).
func TestTypographyMirrorsMatchTheSDK(t *testing.T) {
	if err := TypographySelfCheck(); err != nil {
		t.Fatal(err)
	}
}

// The bridge builds, lays out and queries a paragraph with the test fonts.
func TestTxtParagraphWithTestFonts(t *testing.T) {
	fonts := NewTxtFontCollection(nil)
	b := NewTxtParagraphBuilder(fonts, TxtParagraphBuilderArgs{Encoded: make([]int32, 8)})
	b.AddText([]uint16{'h', 'i'})
	p := b.Build()
	p.Layout(100)
	m := p.Metrics()
	if m.Width != 100 || m.Height <= 0 || m.MaxIntrinsicWidth <= 0 {
		t.Errorf("metrics %+v", m)
	}
	if n := p.NumberOfLines(); n != 1 {
		t.Errorf("%d lines", n)
	}
	p.Dispose()
	p.Dispose() // idempotent
}
