package services_test

// Port of packages/flutter/test/services/clipboard_test.dart (Flutter
// 3.50.1) with widgets/clipboard_utils.dart's MockClipboard; nothing
// skipped.

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func mapEntry(key string, value any) dart.MapEntry[string, any] {
	return dart.MapEntry[string, any]{Key: key, Value: value}
}

// mockClipboard is clipboard_utils.dart's MockClipboard.
type mockClipboard struct {
	hasStringsThrows bool
	clipboardData    any // map[string]any or nil
}

func (m *mockClipboard) handleMethodCall(methodCall services.MethodCall) async.Future[any] {
	return async.Async(func() any {
		switch methodCall.Method {
		case "Clipboard.getData":
			return m.clipboardData
		case "Clipboard.hasStrings":
			if m.hasStringsThrows {
				panic(services.NewPlatformException("clipboard error"))
			}
			data, _ := m.clipboardData.(map[string]any)
			text, _ := data["text"].(string)
			return map[string]any{"value": text != ""}
		case "Clipboard.setData":
			m.clipboardData = methodCall.Arguments
		}
		return nil
	})
}

func clipboardTest(t *testing.T, body func(mock *mockClipboard)) {
	ensureInitialized()
	mock := &mockClipboard{}
	messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, mock.handleMethodCall)
	defer messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, nil)
	runAsync(t, func() { body(mock) })
}

func TestClipboardGetDataReturnsText(t *testing.T) {
	clipboardTest(t, func(mock *mockClipboard) {
		mock.clipboardData = map[string]any{"text": "Hello world"}
		data := services.Clipboard.GetData(services.ClipboardKTextPlain).Await()
		expect(data != nil && *data.Text == "Hello world", "%v", data)
	})
}

func TestClipboardGetDataReturnsNull(t *testing.T) {
	clipboardTest(t, func(mock *mockClipboard) {
		mock.clipboardData = nil
		expect(services.Clipboard.GetData(services.ClipboardKTextPlain).Await() == nil, "not null")
	})
}

func TestClipboardGetDataThrowsIfTextIsMissingOrNull(t *testing.T) {
	for _, data := range []map[string]any{{}, {"text": nil}} {
		clipboardTest(t, func(mock *mockClipboard) {
			mock.clipboardData = data
			err := awaitError(services.Clipboard.GetData(services.ClipboardKTextPlain))
			var typeError dart.TypeError
			expect(errors.As(err, &typeError), "err %v", err)
		})
	}
}

func TestClipboardSetDataSetsText(t *testing.T) {
	clipboardTest(t, func(mock *mockClipboard) {
		services.Clipboard.SetData(services.NewClipboardData("Hello world")).Await()
		expect(reflect.DeepEqual(mock.clipboardData, map[string]any{"text": "Hello world"}), "%v", mock.clipboardData)
		// Own: hasStrings.
		expect(services.Clipboard.HasStrings().Await(), "no strings")
	})
}

// Own: a null reply to hasStrings is false.
func TestClipboardHasStringsNullReply(t *testing.T) {
	ensureInitialized()
	messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, func(services.MethodCall) async.Future[any] {
		return async.FutureValue[any](nil)
	})
	defer messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, nil)
	runAsync(t, func() { expect(!services.Clipboard.HasStrings().Await(), "true") })
}
