package services

// Port of browser_context_menu.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// BrowserContextMenu controls the browser's context menu on the web
// platform (Dart: a class with a private singleton; its statics are the
// methods of this variable). Go is never a web build, so its methods
// panic with Dart's AssertionError, as Flutter asserts kIsWeb.
var BrowserContextMenu = &browserContextMenu{enabled: true, channel: SystemChannels.ContextMenu}

type browserContextMenu struct {
	enabled bool
	channel *MethodChannel
}

// Enabled is whether the browser's context menu is enabled.
func (m *browserContextMenu) Enabled() bool { return m.enabled }

// DisableContextMenu disables the browser's context menu (web only).
func (m *browserContextMenu) DisableContextMenu() async.Future[async.Void] {
	if !foundation.KIsWeb {
		panic(dart.NewAssertionError("This has no effect on platforms other than web."))
	}
	return async.Then(m.channel.InvokeMethod("disableContextMenu"), func(any) async.Void {
		m.enabled = false
		return async.Void{}
	})
}

// EnableContextMenu enables the browser's context menu (web only).
func (m *browserContextMenu) EnableContextMenu() async.Future[async.Void] {
	if !foundation.KIsWeb {
		panic(dart.NewAssertionError("This has no effect on platforms other than web."))
	}
	return async.Then(m.channel.InvokeMethod("enableContextMenu"), func(any) async.Void {
		m.enabled = true
		return async.Void{}
	})
}
