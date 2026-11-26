package services

// Port of system_navigator.dart (Flutter 3.50.1).

import (
	"net/url"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type systemNavigator struct{}

// SystemNavigator controls specific aspects of the system navigation
// stack (a class of static methods).
var SystemNavigator systemNavigator

// SetFrameworkHandlesBack informs the platform of whether or not the
// Flutter framework will handle back events (only relevant on Android).
func (systemNavigator) SetFrameworkHandlesBack(frameworkHandlesBack bool) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		// Currently, this method call is only relevant on Android.
		if foundation.KIsWeb {
			return
		}
		switch foundation.DefaultTargetPlatform() {
		case foundation.TargetPlatformAndroid:
			SystemChannels.Platform.InvokeMethod("SystemNavigator.setFrameworkHandlesBack", frameworkHandlesBack).Await()
		}
	})
}

// SystemNavigatorPopOptions are SystemNavigator.pop's named parameters.
type SystemNavigatorPopOptions struct {
	// Animated (nil: null) is whether the pop is animated (iOS).
	Animated *bool
}

// Pop removes the topmost Flutter instance, presenting what was before
// it.
func (systemNavigator) Pop(opts ...SystemNavigatorPopOptions) async.Future[async.Void] {
	var o SystemNavigatorPopOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return async.AsyncVoid(func() {
		SystemChannels.Platform.InvokeMethod("SystemNavigator.pop", o.Animated).Await()
	})
}

// SelectSingleEntryHistory selects the single-entry history mode (web).
func (systemNavigator) SelectSingleEntryHistory() async.Future[async.Void] {
	return asVoid(SystemChannels.Navigation.InvokeMethod("selectSingleEntryHistory"))
}

// SelectMultiEntryHistory selects the multiple-entry history mode (web).
func (systemNavigator) SelectMultiEntryHistory() async.Future[async.Void] {
	return asVoid(SystemChannels.Navigation.InvokeMethod("selectMultiEntryHistory"))
}

// RouteInformationUpdatedOptions are routeInformationUpdated's named
// parameters: exactly one of Location (deprecated) and URI must be given.
type RouteInformationUpdatedOptions struct {
	Location *string
	URI      *url.URL
	State    any
	Replace  bool
}

// RouteInformationUpdated notifies the platform of a route change.
func (systemNavigator) RouteInformationUpdated(opts RouteInformationUpdatedOptions) async.Future[async.Void] {
	if (opts.Location != nil) == (opts.URI != nil) {
		panic(dart.NewAssertionError("One of uri or location must be provided, but not both."))
	}
	uri := opts.URI
	if uri == nil {
		u, err := url.Parse(*opts.Location)
		if err != nil {
			panic(dart.NewFormatException(err.Error(), *opts.Location))
		}
		uri = u
	}
	return asVoid(SystemChannels.Navigation.InvokeMethod("routeInformationUpdated", []dart.MapEntry[string, any]{
		{Key: "uri", Value: uri.String()},
		{Key: "state", Value: opts.State},
		{Key: "replace", Value: opts.Replace},
	}))
}

// asVoid is a Future<void> of f (Dart returns invokeMethod<void>'s future
// as is).
func asVoid(f async.Future[any]) async.Future[async.Void] {
	return async.Then(f, func(any) async.Void { return async.Void{} })
}
