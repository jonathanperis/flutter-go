package app

// The host side of the platform channels the window runner implements,
// as the engine's macOS embedder does (FlutterPlatformPlugin for
// flutter/platform's clipboard, FlutterEngine's asset channel for
// flutter/assets). Everything else gets the engine's empty reply (the
// framework reads it as "not implemented"). ENGINE GAP: the other
// FlutterPlatformPlugin methods (SystemSound.play, System.exitApplication,
// SystemNavigator.pop...) and the other plugins (text input, mouse cursor,
// keyboard state, restoration...) are not implemented.

import (
	"github.com/go-gl/glfw/v3.3/glfw"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// HandlePlatformMessage makes the engine a ui.PlatformMessageHandler: the
// framework's messages to the host arrive here (on the UI thread, which
// is the main thread GLFW requires).
func (e *engine) HandlePlatformMessage(message *ui.PlatformMessage) {
	switch message.Channel {
	case "flutter/platform":
		handlePlatformChannel(e.window, message)
	case "flutter/assets":
		handleAssetChannel(message)
	default:
		message.Response.CompleteEmpty()
	}
}

// handlePlatformChannel is FlutterPlatformPlugin's clipboard part
// (Clipboard.getData/setData/hasStrings on the system pasteboard through
// GLFW).
func handlePlatformChannel(window *glfw.Window, message *ui.PlatformMessage) {
	codec := services.JSONMethodCodec{}
	var call services.MethodCall
	if !decodeCall(codec, message.Data, &call) || window == nil {
		message.Response.CompleteEmpty()
		return
	}
	switch call.Method {
	case "Clipboard.setData":
		text := ""
		if args, ok := call.Arguments.(map[string]any); ok {
			text, _ = args["text"].(string)
		}
		window.SetClipboardString(text)
		message.Response.Complete(codec.EncodeSuccessEnvelope(nil))
	case "Clipboard.getData":
		// Only "text/plain" is supported, as on macOS.
		if format, _ := call.Arguments.(string); format != "text/plain" {
			message.Response.Complete(codec.EncodeSuccessEnvelope(nil))
			return
		}
		text := window.GetClipboardString()
		if text == "" {
			message.Response.Complete(codec.EncodeSuccessEnvelope(nil))
			return
		}
		message.Response.Complete(codec.EncodeSuccessEnvelope(map[string]any{"text": text}))
	case "Clipboard.hasStrings":
		message.Response.Complete(codec.EncodeSuccessEnvelope(map[string]any{"value": window.GetClipboardString() != ""}))
	default:
		message.Response.CompleteEmpty()
	}
}

func decodeCall(codec services.MethodCodec, data []byte, call *services.MethodCall) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	*call = codec.DecodeMethodCall(data)
	return true
}

// handleAssetChannel serves flutter/assets (the asset key, UTF-8 and
// URI-encoded, as PlatformAssetBundle sends it) from ui.EngineAssetLoader,
// the embedder's asset manager; a missing asset gets an empty reply.
func handleAssetChannel(message *ui.PlatformMessage) {
	loader := ui.EngineAssetLoader
	if loader == nil {
		message.Response.CompleteEmpty()
		return
	}
	data, err := loader(dart.UTF8Decode(message.Data))
	if err != nil || len(data) == 0 {
		message.Response.CompleteEmpty()
		return
	}
	message.Response.Complete(append([]byte(nil), data...))
}
