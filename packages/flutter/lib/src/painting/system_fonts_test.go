package painting_test

// packages/flutter/test/painting/system_fonts_test.dart (Flutter 3.50.1):
// all 11 tests are skipped: they check that render objects and widgets
// (RenderParagraph, RenderEditable, Banner, CupertinoDatePicker,
// RangeSlider, Slider, TimePicker, RelayoutWhenSystemFontsChangeMixin)
// relayout on a fontsChange message, and rendering/widgets/material/
// cupertino are not ported. What they rely on from painting —
// PaintingBinding.systemFonts notifying on a "fontsChange" system message —
// is checked here.

import (
	"testing"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

func TestSystemFontsNotifiesOnFontsChange(t *testing.T) {
	tester := newImageTester(t)
	notified := 0
	listener := func() { notified++ }
	fonts := PaintingBindingInstance().SystemFonts()
	fonts.AddListener(listener)
	fonts.AddListener(listener) // a set: added once
	send := func(typ string) {
		message := services.SystemChannels.System.Codec.EncodeMessage(map[string]any{"type": typ})
		await(tester, tester.Binding().DefaultBinaryMessenger().HandlePlatformMessage("flutter/system", message, func([]byte) {}))
	}
	send("fontsChange")
	if notified != 1 {
		t.Errorf("notified %d", notified)
	}
	send("memoryPressure")
	if notified != 1 {
		t.Errorf("notified %d", notified)
	}
	fonts.RemoveListener(listener)
	send("fontsChange")
	if notified != 1 {
		t.Errorf("notified %d after removal", notified)
	}
}
