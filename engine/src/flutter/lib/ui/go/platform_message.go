package ui

// Partial port of engine/src/flutter/lib/ui/platform_dispatcher.dart
// (Flutter 3.50.1): platform messages (sendPlatformMessage,
// onPlatformMessage and the dispatch of incoming messages, hooks.dart's
// _dispatchPlatformMessage), RootIsolateToken, initialLifecycleState,
// AppExitResponse and AppExitType; and of the engine side
// (lib/ui/window/platform_configuration.cc, platform_message.h,
// platform_message_response_dart.cc): PlatformMessage,
// PlatformMessageResponse, PlatformConfiguration.DispatchPlatformMessage and
// the natives SendPlatformMessage and RespondToPlatformMessage.
//
// A Dart ByteData is a Go []byte; a nil slice is Dart's null, and an empty
// non-nil slice is a zero-length ByteData.
//
// ENGINE GAP: messages the framework sends reach the embedder only if its
// PlatformConfigurationClient implements PlatformMessageHandler; src/app's
// window runner doesn't (there are no host-platform plugins), so every
// message is answered with an empty (null) reply, as the engine's default
// PlatformView::HandlePlatformMessage and flutter_tester do.

import (
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// PlatformMessageResponseCallback is the signature for responses to
// platform messages (PlatformDispatcher.SendPlatformMessage,
// PlatformDispatcher.OnPlatformMessage, ChannelBuffers).
type PlatformMessageResponseCallback func(data []byte)

// PlatformMessageCallback is the signature of
// PlatformDispatcher.OnPlatformMessage (deprecated in Flutter: migrate to
// ChannelBuffers.SetListener).
type PlatformMessageCallback func(name string, data []byte, callback PlatformMessageResponseCallback)

// SendPlatformMessage sends a message to a platform-specific plugin. The
// name selects the plugin, data is the payload (nil for none), and
// callback, when not nil, receives the plugin's reply. The reply arrives as
// an event of the current async.Loop (the engine posts it to the UI task
// runner), so the loop's owner must run posted events (Loop.RunPosted /
// WaitPosted) for it to be delivered; a panic in the callback is an
// uncaught error of that loop (Dart's runUnaryGuarded in the
// registration zone).
//
// The engine's native (PlatformConfigurationNativeApi::SendPlatformMessage)
// fails, and Dart throws, when called from a background isolate; a Go
// program is always on the root isolate.
func (p *PlatformDispatcher) SendPlatformMessage(name string, data []byte, callback PlatformMessageResponseCallback) {
	var response PlatformMessageResponse
	if callback != nil {
		response = &platformMessageResponseDart{callback: callback, loop: async.Current()}
	}
	message := &PlatformMessage{Channel: name, Data: data, Response: response}
	if pc := currentPlatformConfiguration; pc != nil {
		if handler, ok := pc.client.(PlatformMessageHandler); ok {
			handler.HandlePlatformMessage(message)
			return
		}
	}
	// PlatformView::HandlePlatformMessage, the default when the embedder
	// handles no messages (flutter_tester, src/app): an empty reply.
	if message.Response != nil {
		message.Response.CompleteEmpty()
	}
}

// OnPlatformMessage is called whenever this platform dispatcher receives a
// message from a platform-specific plugin (deprecated in Flutter: migrate to
// ChannelBuffers.SetListener). The handler must call its callback, with
// nil when it has no response.
func (p *PlatformDispatcher) OnPlatformMessage() PlatformMessageCallback { return p.onPlatformMessage }

// SetOnPlatformMessage sets OnPlatformMessage.
func (p *PlatformDispatcher) SetOnPlatformMessage(callback PlatformMessageCallback) {
	p.onPlatformMessage = callback
}

// dispatchPlatformMessage is _dispatchPlatformMessage: a message from the
// platform goes to the channel buffers' control channel, to
// OnPlatformMessage when set, or to the channel buffers.
func (p *PlatformDispatcher) dispatchPlatformMessage(name string, data []byte, responseID int) {
	if name == ChannelBuffersKControlChannelName {
		defer respondToPlatformMessage(responseID, nil)
		ChannelBuffersInstance().HandleMessage(data)
	} else if p.onPlatformMessage != nil {
		p.onPlatformMessage(name, data, func(responseData []byte) {
			respondToPlatformMessage(responseID, responseData)
		})
	} else {
		ChannelBuffersInstance().Push(name, data, func(responseData []byte) {
			respondToPlatformMessage(responseID, responseData)
		})
	}
}

// respondToPlatformMessage is __respondToPlatformMessage
// (PlatformConfigurationNativeApi::RespondToPlatformMessage).
func respondToPlatformMessage(responseID int, data []byte) {
	pc := currentPlatformConfiguration
	if pc == nil {
		return
	}
	if data == nil {
		pc.completePlatformMessageEmptyResponse(responseID)
	} else {
		pc.completePlatformMessageResponse(responseID, data)
	}
}

// InitialLifecycleState is the lifecycle state immediately after the
// isolate was initialized ("" when the embedder reported none). It is not
// updated as the lifecycle changes, and stops following the embedder once
// read.
func (p *PlatformDispatcher) InitialLifecycleState() string {
	p.initialLifecycleStateAccessed = true
	return p.initialLifecycleState
}

// updateInitialLifecycleState is called from the engine, via hooks.go.
func (p *PlatformDispatcher) updateInitialLifecycleState(state string) {
	// We do not update the state if the state has already been used to
	// initialize the lifecycleState.
	if !p.initialLifecycleStateAccessed {
		p.initialLifecycleState = state
	}
}

// RootIsolateToken is a token that represents a root isolate.
type RootIsolateToken struct{ token int }

var rootIsolateToken = &RootIsolateToken{token: 1}

// RootIsolateTokenInstance is RootIsolateToken.instance, the token of the
// root isolate executing this code (nil on other isolates). A Go program
// has a single isolate, so it is never nil (goroutines, such as
// foundation.Compute's, share it).
func RootIsolateTokenInstance() *RootIsolateToken { return rootIsolateToken }

// AppExitResponse is a possible response to a request to exit the
// application.
type AppExitResponse int

const (
	// AppExitResponseExit: exiting the application can proceed.
	AppExitResponseExit AppExitResponse = iota
	// AppExitResponseCancel: cancel the exit; do not exit the application.
	AppExitResponseCancel
)

// AppExitResponseValues is AppExitResponse.values.
var AppExitResponseValues = []AppExitResponse{AppExitResponseExit, AppExitResponseCancel}

// Name is the enum value's name ("exit").
func (r AppExitResponse) Name() string { return [...]string{"exit", "cancel"}[r] }

func (r AppExitResponse) String() string { return "AppExitResponse." + r.Name() }

// AppExitType is the type of application exit to perform when calling
// ServicesBinding.ExitApplication.
type AppExitType int

const (
	// AppExitTypeCancelable requests an orderly exit that the framework may
	// cancel (WidgetsBinding answers the request).
	AppExitTypeCancelable AppExitType = iota
	// AppExitTypeRequired is a non-cancelable orderly exit.
	AppExitTypeRequired
)

// AppExitTypeValues is AppExitType.values.
var AppExitTypeValues = []AppExitType{AppExitTypeCancelable, AppExitTypeRequired}

// Name is the enum value's name ("cancelable").
func (t AppExitType) Name() string { return [...]string{"cancelable", "required"}[t] }

func (t AppExitType) String() string { return "AppExitType." + t.Name() }

// ---- engine side

// PlatformMessage is the engine's PlatformMessage: a message on a channel,
// with its data (nil: no data) and the response through which the receiver
// replies (nil when the sender expects no reply).
type PlatformMessage struct {
	Channel  string
	Data     []byte
	Response PlatformMessageResponse
}

// PlatformMessageResponse is the engine's PlatformMessageResponse. Exactly
// one of its methods is called, at most once, from any goroutine.
type PlatformMessageResponse interface {
	// Complete replies with data (a ByteData, possibly empty).
	Complete(data []byte)
	// CompleteEmpty replies with no data (null).
	CompleteEmpty()
}

// PlatformMessageHandler is implemented by a PlatformConfigurationClient
// that delivers the framework's messages to the host platform
// (PlatformConfigurationClient::HandlePlatformMessage). A client without it
// answers every message with an empty reply.
type PlatformMessageHandler interface {
	HandlePlatformMessage(message *PlatformMessage)
}

// platformMessageResponseDart is PlatformMessageResponseDart: the reply is
// posted to the loop of the isolate that sent the message (the UI task
// runner) and calls the framework's callback there.
type platformMessageResponseDart struct {
	callback   PlatformMessageResponseCallback
	loop       *async.Loop
	isComplete bool
}

func (r *platformMessageResponseDart) postCompletion(result []byte) {
	callback := r.callback
	if callback == nil {
		return
	}
	r.callback = nil // the persistent handle is released
	if r.isComplete {
		panic(dart.NewAssertionError("platform message response completed twice"))
	}
	r.isComplete = true
	r.loop.Post(func() { callback(result) })
}

func (r *platformMessageResponseDart) Complete(data []byte) {
	if data == nil {
		data = []byte{}
	}
	r.postCompletion(data)
}

func (r *platformMessageResponseDart) CompleteEmpty() { r.postCompletion(nil) }

// DispatchPlatformMessage delivers a message from the host platform to the
// framework (PlatformConfiguration::DispatchPlatformMessage, then
// hooks.dart's _dispatchPlatformMessage). The embedder calls it on the UI
// loop; the framework's reply completes message.Response.
func (pc *PlatformConfiguration) DispatchPlatformMessage(message *PlatformMessage) {
	responseID := 0
	if message.Response != nil {
		if pc.pendingResponses == nil {
			pc.pendingResponses = map[int]PlatformMessageResponse{}
			pc.nextResponseID = 1
		}
		responseID = pc.nextResponseID
		pc.nextResponseID++
		pc.pendingResponses[responseID] = message.Response
	}
	PlatformDispatcherInstance().dispatchPlatformMessage(message.Channel, message.Data, responseID)
}

func (pc *PlatformConfiguration) takePendingResponse(responseID int) PlatformMessageResponse {
	if responseID == 0 {
		return nil
	}
	response, ok := pc.pendingResponses[responseID]
	if !ok {
		return nil
	}
	delete(pc.pendingResponses, responseID)
	return response
}

func (pc *PlatformConfiguration) completePlatformMessageEmptyResponse(responseID int) {
	if response := pc.takePendingResponse(responseID); response != nil {
		response.CompleteEmpty()
	}
}

func (pc *PlatformConfiguration) completePlatformMessageResponse(responseID int, data []byte) {
	if response := pc.takePendingResponse(responseID); response != nil {
		response.Complete(append([]byte{}, data...)) // the engine copies the data
	}
}

// UpdateInitialLifecycleState reports the lifecycle state at startup
// (hooks.dart's _updateInitialLifecycleState).
func (pc *PlatformConfiguration) UpdateInitialLifecycleState(state string) {
	PlatformDispatcherInstance().updateInitialLifecycleState(state)
}
