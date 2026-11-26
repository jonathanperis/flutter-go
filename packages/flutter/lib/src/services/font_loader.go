package services

// Port of font_loader.dart (Flutter 3.50.1).

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// FontLoaderOverridable is what a type extending FontLoader may override
// (its @protected LoadFont).
type FontLoaderOverridable interface {
	LoadFont(list []byte, family string) async.Future[async.Void]
}

// FontLoader is a class that enables the dynamic loading of fonts at
// runtime. It is extended by embedding (call SetSelf so an override of
// LoadFont runs).
type FontLoader struct {
	// Family is the font family being loaded.
	Family string

	self        FontLoaderOverridable
	loaded      bool
	fontFutures []async.Future[[]byte]
}

// NewFontLoader is FontLoader(family).
func NewFontLoader(family string) *FontLoader {
	l := &FontLoader{Family: family}
	l.self = l
	return l
}

// SetSelf points the loader at the value embedding it, whose LoadFont
// override it calls.
func (l *FontLoader) SetSelf(self FontLoaderOverridable) { l.self = self }

func (l *FontLoader) outer() FontLoaderOverridable {
	if l.self == nil {
		return l
	}
	return l.self
}

// AddFont registers a font asset to be loaded by this font loader. It
// panics with a StateError if the loader was already loaded.
func (l *FontLoader) AddFont(bytes async.Future[[]byte]) {
	if l.loaded {
		panic(dart.StateError{Message: "FontLoader is already loaded"})
	}
	// Dart: bytes.then((data) => Uint8List.view(...)), a view of the same
	// bytes.
	l.fontFutures = append(l.fontFutures, async.Then(bytes, func(data []byte) []byte { return data }))
}

// Load loads this font loader's font Family and all of its associated
// assets into the Flutter engine, making the font available to the
// current application, in the order the fonts were added. It panics with a
// StateError if the loader was already loaded.
func (l *FontLoader) Load() async.Future[async.Void] {
	return async.AsyncVoid(func() {
		if l.loaded {
			panic(dart.StateError{Message: "FontLoader is already loaded"})
		}
		l.loaded = true
		for _, fontFuture := range l.fontFutures {
			l.outer().LoadFont(fontFuture.Await(), l.Family).Await()
		}
	})
}

// LoadFont hooks into font loading (@protected, @visibleForTesting); the
// default loads the font with dart:ui's loadFontFromList.
func (l *FontLoader) LoadFont(list []byte, family string) async.Future[async.Void] {
	return ui.LoadFontFromList(list, ui.LoadFontFromListOptions{FontFamily: &family})
}
