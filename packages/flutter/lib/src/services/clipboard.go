package services

// Port of packages/flutter/lib/src/services/clipboard.dart (Flutter
// 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// ClipboardData is data stored on the system clipboard.
type ClipboardData struct {
	// Text is plain text variant of this clipboard data (nil: null; always
	// set for data built with NewClipboardData).
	Text *string
}

// NewClipboardData is ClipboardData({required String text}).
func NewClipboardData(text string) ClipboardData { return ClipboardData{Text: &text} }

// Clipboard is the utility methods for interacting with the system's
// clipboard (Dart's abstract final class of statics: a package variable
// whose methods are the statics).
var Clipboard clipboard

type clipboard struct{}

// ClipboardKTextPlain is Clipboard.kTextPlain, the plain text format for
// GetData.
const ClipboardKTextPlain = "text/plain"

// SetData stores the given clipboard data on the clipboard.
func (clipboard) SetData(data ClipboardData) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.Platform.InvokeMethod("Clipboard.setData",
			[]dart.MapEntry[string, any]{{Key: "text", Value: data.Text}}).Await()
	})
}

// GetData retrieves data from the clipboard that matches the given format
// (nil: null, when there is none).
func (clipboard) GetData(format string) async.Future[*ClipboardData] {
	return async.Async(func() *ClipboardData {
		r := SystemChannels.Platform.InvokeMethod("Clipboard.getData", format).Await()
		if r == nil {
			return nil
		}
		result := r.(map[string]any)
		text, ok := result["text"].(string)
		if !ok {
			panic(dart.TypeError{Message: "type 'Null' is not a subtype of type 'String' in type cast"})
		}
		data := NewClipboardData(text)
		return &data
	})
}

// HasStrings returns a future that resolves to true, if (and only if) the
// clipboard contains string data.
func (clipboard) HasStrings() async.Future[bool] {
	return async.Async(func() bool {
		r := SystemChannels.Platform.InvokeMethod("Clipboard.hasStrings", ClipboardKTextPlain).Await()
		if r == nil {
			return false
		}
		return r.(map[string]any)["value"].(bool)
	})
}
