package services

// Port of packages/flutter/lib/src/services/keyboard_inserted_content.dart
// (Flutter 3.50.1).

import (
	"strconv"
	"strings"
	"unsafe"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// KeyboardInsertedContent is a class representing rich content (such as a
// PNG image) inserted via the system input method. Compare with Equal.
type KeyboardInsertedContent struct {
	// MimeType is the MIME type of the inserted content.
	MimeType string
	// URI is the URI (location) of the inserted content, usually a
	// "content://" URI.
	URI string
	// Data is the bytes of the inserted content (nil: null).
	Data []byte
}

// NewKeyboardInsertedContentFromJSON is KeyboardInsertedContent.fromJson.
func NewKeyboardInsertedContentFromJSON(metadata map[string]any) KeyboardInsertedContent {
	c := KeyboardInsertedContent{MimeType: metadata["mimeType"].(string), URI: metadata["uri"].(string)}
	if metadata["data"] != nil {
		list := metadata["data"].([]any)
		c.Data = make([]byte, len(list))
		for i, v := range list {
			c.Data[i] = byte(v.(int)) // Uint8List.fromList truncates to 8 bits
		}
	}
	return c
}

// HasData is whether the content has bytes.
func (c KeyboardInsertedContent) HasData() bool { return len(c.Data) > 0 }

func (c KeyboardInsertedContent) String() string {
	data := "null"
	if c.Data != nil {
		parts := make([]string, len(c.Data))
		for i, b := range c.Data {
			parts[i] = strconv.Itoa(int(b))
		}
		data = "[" + strings.Join(parts, ", ") + "]"
	}
	return foundation.ObjectRuntimeType(c, "KeyboardInsertedContent") + "(" + c.MimeType + ", " + c.URI + ", " + data + ")"
}

// Equal is KeyboardInsertedContent's == (Data compares by identity, as
// Dart's Uint8List ==).
func (c KeyboardInsertedContent) Equal(other KeyboardInsertedContent) bool {
	sameData := c.Data == nil && other.Data == nil ||
		c.Data != nil && other.Data != nil && len(c.Data) == len(other.Data) && unsafe.SliceData(c.Data) == unsafe.SliceData(other.Data)
	return other.MimeType == c.MimeType && other.URI == c.URI && sameData
}
