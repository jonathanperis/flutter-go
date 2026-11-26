package services

// Port of deferred_component.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type deferredComponent struct{}

// DeferredComponent manages the installation and loading of deferred
// components (a class of static methods). Deferred components are an
// Android feature; Go programs have no deferred libraries, so these only
// send the platform messages.
var DeferredComponent deferredComponent

// InstallDeferredComponent requests that an assets-only deferred component
// identified by componentName be downloaded and installed (Dart's
// required named componentName is positional).
func (deferredComponent) InstallDeferredComponent(componentName string) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.DeferredComponent.InvokeMethod("installDeferredComponent", []dart.MapEntry[string, any]{
			{Key: "loadingUnitId", Value: -1}, {Key: "componentName", Value: componentName},
		}).Await()
	})
}

// UninstallDeferredComponent requests that a deferred component
// identified by componentName be uninstalled.
func (deferredComponent) UninstallDeferredComponent(componentName string) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.DeferredComponent.InvokeMethod("uninstallDeferredComponent", []dart.MapEntry[string, any]{
			{Key: "loadingUnitId", Value: -1}, {Key: "componentName", Value: componentName},
		}).Await()
	})
}
