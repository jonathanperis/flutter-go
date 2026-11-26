package semantics

// Port of packages/flutter/lib/src/semantics/semantics_service.dart
// (Flutter 3.50.1): SemanticsService, which sends events through
// SystemChannels.accessibility (see services' platform_channel.go for how
// that channel reaches a messenger before ServicesBinding is ported).

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
)

type semanticsService struct{}

// SemanticsService allows access to the platform's accessibility services
// (a class of static methods: SemanticsService.Announce(...)).
var SemanticsService semanticsService

// SemanticsServiceAnnounceOptions are announce's and sendAnnouncement's
// named parameters (Assertiveness defaults to polite).
type SemanticsServiceAnnounceOptions struct{ Assertiveness Assertiveness }

// Announce sends a semantic announcement to the implicit view
// (Deprecated: use SendAnnouncement; incompatible with multiple windows).
func (semanticsService) Announce(message string, textDirection ui.TextDirection, opts ...SemanticsServiceAnnounceOptions) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		var o SemanticsServiceAnnounceOptions
		if len(opts) > 0 {
			o = opts[0]
		}
		view := ui.PlatformDispatcherInstance().ImplicitView()
		dartAssert(view != nil, "SemanticsService.announce is incompatible with multiple windows. "+
			"Use SemanticsService.sendAnnouncement instead.")
		event := NewAnnounceSemanticsEvent(message, textDirection, view.ViewID(),
			AnnounceSemanticsEventOptions{Assertiveness: o.Assertiveness})
		services.SystemChannels.Accessibility.Send(event.ToMap()).Await()
	})
}

// SendAnnouncement sends a semantic announcement for view.
func (semanticsService) SendAnnouncement(view *ui.FlutterView, message string, textDirection ui.TextDirection, opts ...SemanticsServiceAnnounceOptions) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		var o SemanticsServiceAnnounceOptions
		if len(opts) > 0 {
			o = opts[0]
		}
		event := NewAnnounceSemanticsEvent(message, textDirection, view.ViewID(),
			AnnounceSemanticsEventOptions{Assertiveness: o.Assertiveness})
		services.SystemChannels.Accessibility.Send(event.ToMap()).Await()
	})
}

// Tooltip sends a semantic announcement of a tooltip (Android only).
func (semanticsService) Tooltip(message string) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		event := NewTooltipSemanticsEvent(message)
		services.SystemChannels.Accessibility.Send(event.ToMap()).Await()
	})
}
