package ui

// The engine side of painting.dart's natives: dart:ui's drawing objects
// are backed by the prebuilt Impeller SDK (src/impeller), as Flutter's are
// backed by the engine's DisplayList and Impeller. Objects that need a GPU
// context (textures, offscreen rendering, fragment programs) share one
// Impeller context per process, created on first use, as the engine shares
// its IO/raster context.

import (
	"sync"

	"github.com/jonathanperis/flutter-go/src/impeller"
)

var (
	engineCtxMu   sync.Mutex
	engineCtx     *impeller.Context
	engineCtxErr  error
	engineCtxDone bool
)

// EngineImpellerContext returns the process's Impeller context (Metal),
// creating it on first use; nil when no GPU context can be created. The
// embedder (src/app) renders frames with it, so textures created by dart:ui
// (images) can be drawn on screen.
func EngineImpellerContext() *impeller.Context {
	engineCtxMu.Lock()
	defer engineCtxMu.Unlock()
	if !engineCtxDone {
		engineCtxDone = true
		engineCtx, engineCtxErr = impeller.NewMetalContext()
	}
	return engineCtx
}

// engineContext is EngineImpellerContext for dart:ui's own use.
func engineContext() *impeller.Context { return EngineImpellerContext() }

// EngineAssetLoader is the embedder's asset manager (the engine's
// AssetManager): it returns the bytes of the asset with the given key, or
// an error when there is none. ImmutableBuffer.fromAsset and
// FragmentProgram.fromAsset read through it. The default finds no assets;
// embedders and tests install their own.
var EngineAssetLoader func(key string) ([]byte, error)
