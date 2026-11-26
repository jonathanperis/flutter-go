package services

// Port of packages/flutter/lib/src/services/undo_manager.dart (Flutter
// 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// UndoDirection is the direction in which an undo action should be
// performed, whether undo or redo.
type UndoDirection int

const (
	// UndoDirectionUndo: perform an undo action.
	UndoDirectionUndo UndoDirection = iota
	// UndoDirectionRedo: perform a redo action.
	UndoDirectionRedo
)

// UndoDirectionValues is UndoDirection.values.
var UndoDirectionValues = []UndoDirection{UndoDirectionUndo, UndoDirectionRedo}

// Name is the enum value's name.
func (d UndoDirection) Name() string { return [...]string{"undo", "redo"}[d] }

func (d UndoDirection) String() string { return "UndoDirection." + d.Name() }

// UndoManagerClient is an interface to receive events from a native
// UndoManager (Dart's mixin, whose members are all abstract).
type UndoManagerClient interface {
	// HandlePlatformUndo requests that the client perform an undo or redo
	// operation.
	HandlePlatformUndo(direction UndoDirection)
	// Undo reverts the value to the previous state.
	Undo()
	// Redo updates the value to the next state.
	Redo()
	// CanUndo checks if the value can be reverted.
	CanUndo() bool
	// CanRedo checks if the value can be updated to a later state.
	CanRedo() bool
}

// undoManager is the UndoManager singleton (Dart's private instance).
type undoManager struct {
	channel       *MethodChannel
	currentClient UndoManagerClient
}

var undoManagerSingleton *undoManager

// undoManagerInstance is UndoManager._instance, created (registering its
// handler) on first use, as Dart's lazy static final is.
func undoManagerInstance() *undoManager {
	if undoManagerSingleton == nil {
		m := &undoManager{}
		undoManagerSingleton = m
		m.channel = SystemChannels.UndoManager
		m.channel.SetMethodCallHandler(m.handleUndoManagerInvocation)
	}
	return undoManagerSingleton
}

// undoManagerStatics is the type of UndoManager (a class whose public API is
// static).
type undoManagerStatics struct{}

// UndoManager is a low-level interface to the system's undo manager
// (UndoManager.client is UndoManager.Client(), its setter SetClient...).
var UndoManager undoManagerStatics

// SetChannel sets the MethodChannel used to communicate with the engine
// (@visibleForTesting; debug builds only).
func (undoManagerStatics) SetChannel(newChannel *MethodChannel) {
	if foundation.KDebugMode {
		instance := undoManagerInstance()
		instance.channel = newChannel
		newChannel.SetMethodCallHandler(instance.handleUndoManagerInvocation)
	}
}

// SetClient receives undo and redo events from the platform (nil: none).
func (undoManagerStatics) SetClient(client UndoManagerClient) {
	undoManagerInstance().currentClient = client
}

// Client is the current client, or nil.
func (undoManagerStatics) Client() UndoManagerClient { return undoManagerInstance().currentClient }

// UndoManagerSetUndoStateOptions are setUndoState's named parameters
// (default false).
type UndoManagerSetUndoStateOptions struct{ CanUndo, CanRedo bool }

// SetUndoState sets the current state of the system UndoManager.
func (undoManagerStatics) SetUndoState(opts ...UndoManagerSetUndoStateOptions) {
	var o UndoManagerSetUndoStateOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	undoManagerInstance().setUndoState(o.CanUndo, o.CanRedo)
}

func (m *undoManager) handleUndoManagerInvocation(methodCall MethodCall) async.Future[any] {
	return async.Async(func() any {
		method := methodCall.Method
		args := methodCall.Arguments.([]any)
		if method == "UndoManagerClient.handleUndo" {
			if m.currentClient == nil {
				panic(dart.NewAssertionError("There must be a current UndoManagerClient."))
			}
			m.currentClient.HandlePlatformUndo(m.toUndoDirection(args[0].(string)))
			return nil
		}
		panic(NewMissingPluginException())
	})
}

func (m *undoManager) setUndoState(canUndo, canRedo bool) {
	async.OnComplete(m.channel.InvokeMethod("UndoManager.setUndoState", []dart.MapEntry[string, any]{
		{Key: "canUndo", Value: canUndo},
		{Key: "canRedo", Value: canRedo},
	}), func(any) {}, func(err error) {
		library := "services library"
		foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exceptionOf(err), foundation.FlutterErrorDetailsOptions{
			Library: &library,
			Context: foundation.NewErrorDescription("while sending the UndoManager.setUndoState event"),
		}))
	})
}

func (m *undoManager) toUndoDirection(direction string) UndoDirection {
	switch direction {
	case "undo":
		return UndoDirectionUndo
	case "redo":
		return UndoDirectionRedo
	}
	panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
		foundation.NewErrorSummary("Unknown undo direction: " + direction),
	}))
}

// exceptionOf is the exception a future's error stands for: the value a
// panic carried (async.PanicError), else the error itself.
func exceptionOf(err error) any {
	if p, ok := err.(async.PanicError); ok {
		return p.Value
	}
	return err
}
