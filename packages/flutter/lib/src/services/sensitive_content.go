package services

// Port of sensitive_content.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// ContentSensitivity is the content sensitivity setting of a Flutter view
// (Android API 35+).
type ContentSensitivity int

const (
	// ContentSensitivityAutoSensitive: the view's content is sensitive
	// only if it contains sensitive content (not supported yet).
	ContentSensitivityAutoSensitive ContentSensitivity = iota
	// ContentSensitivitySensitive: the view's content is sensitive and
	// should be obscured during screen projection.
	ContentSensitivitySensitive
	// ContentSensitivityNotSensitive: the view's content is not sensitive.
	ContentSensitivityNotSensitive
	// contentSensitivityUnknown is Dart's private _unknown: a mode the
	// platform reports that Flutter doesn't recognize.
	contentSensitivityUnknown
)

// ContentSensitivityValues is ContentSensitivity.values (with the private
// _unknown last, as in Dart).
var ContentSensitivityValues = []ContentSensitivity{ContentSensitivityAutoSensitive,
	ContentSensitivitySensitive, ContentSensitivityNotSensitive, contentSensitivityUnknown}

// Name is the enum value's Dart name.
func (c ContentSensitivity) Name() string {
	return [...]string{"autoSensitive", "sensitive", "notSensitive", "_unknown"}[c]
}

func (c ContentSensitivity) String() string { return "ContentSensitivity." + c.Name() }

// SensitiveContentService sets the content sensitivity of the native
// Android View that contains the app's Flutter view.
type SensitiveContentService struct {
	// SensitiveContentChannel is the channel to the platform
	// (SystemChannels.SensitiveContent).
	SensitiveContentChannel *MethodChannel
}

// NewSensitiveContentService is SensitiveContentService().
func NewSensitiveContentService() *SensitiveContentService {
	return &SensitiveContentService{SensitiveContentChannel: SystemChannels.SensitiveContent}
}

// SetContentSensitivity sets the content sensitivity of the Flutter view.
func (s *SensitiveContentService) SetContentSensitivity(contentSensitivity ContentSensitivity) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		s.SensitiveContentChannel.InvokeMethod("SensitiveContent.setContentSensitivity", int(contentSensitivity)).Await()
	})
}

// GetContentSensitivity returns the content sensitivity of the Flutter
// view; it fails with an UnsupportedError for a mode Flutter doesn't
// recognize.
func (s *SensitiveContentService) GetContentSensitivity() async.Future[ContentSensitivity] {
	return async.Async(func() ContentSensitivity {
		result := s.SensitiveContentChannel.InvokeMethod("SensitiveContent.getContentSensitivity").Await()
		if result == nil {
			panic(dart.TypeError{Message: "Null check operator used on a null value"})
		}
		index := castElement[int](result)
		if index < 0 || index >= len(ContentSensitivityValues) {
			panic(dart.IndexError{Index: index, Length: len(ContentSensitivityValues), Name: "index"})
		}
		contentSensitivity := ContentSensitivityValues[index]
		if contentSensitivity == contentSensitivityUnknown {
			panic(dart.UnsupportedError{Message: "Android Flutter View has a content sensitivity mode " +
				"that is not recognized by Flutter. If you see this error, it " +
				"is possible that the View uses a new mode that Flutter needs to " +
				"support; please file an issue."})
		}
		return contentSensitivity
	})
}

// IsSupported is whether setting content sensitivity is supported on the
// device (Android API 35+ only).
func (s *SensitiveContentService) IsSupported() async.Future[bool] {
	return async.Async(func() bool {
		if foundation.DefaultTargetPlatform() != foundation.TargetPlatformAndroid {
			return false
		}
		result := s.SensitiveContentChannel.InvokeMethod("SensitiveContent.isSupported").Await()
		if result == nil {
			panic(dart.TypeError{Message: "Null check operator used on a null value"})
		}
		return castElement[bool](result)
	})
}
