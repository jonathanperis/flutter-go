package painting

// Port of image_stream.dart (Flutter 3.50.1): ImageInfo,
// ImageStreamListener, ImageChunkEvent, ImageStream,
// ImageStreamCompleterHandle, ImageStreamCompleter,
// OneFrameImageStreamCompleter and MultiFrameImageStreamCompleter.
//
// ImageInfo is a struct Flutter's tests extend (TestImageInfo overrides
// clone, == and toString): a type embedding it calls SetSelf, and the
// completers clone, compare and print images through Self(). Listeners
// receive the *ImageInfo (the embedded base of an extending type; Self()
// returns the outer value).
//
// ImageStreamCompleter is an interface plus the embeddable
// ImageStreamCompleterBase (self pattern); a completer embeds the base and
// calls InitImageStreamCompleter(self). Dart's private override of
// _maybeDispose (MultiFrameImageStreamCompleter) is reached through self.

import (
	"math"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// ---------------------------------------------------------------------------
// ImageInfo

// ImageInfo is a ui.Image with its scale and an optional debug label.
type ImageInfo struct {
	// Image is the raw image pixels.
	Image *ui.Image
	// Scale is the linear scale factor for drawing this image at its
	// intended size (image pixels per logical pixel).
	Scale float64
	// DebugLabel is a string used for debugging purposes (nil: none).
	DebugLabel *string

	self ImageInfoOverrides
}

// ImageInfoOverrides are ImageInfo's methods a type extending it may
// override (Dart's clone, == and toString).
type ImageInfoOverrides interface {
	Clone() *ImageInfo
	Equal(other *ImageInfo) bool
	String() string
}

// ImageInfoOptions are ImageInfo's named parameters (Scale nil: 1.0).
type ImageInfoOptions struct {
	Scale      *float64
	DebugLabel *string
}

// NewImageInfo is ImageInfo(image: image, scale: ..., debugLabel: ...).
func NewImageInfo(image *ui.Image, opts ...ImageInfoOptions) *ImageInfo {
	i := &ImageInfo{}
	i.Init(i, image, opts...)
	return i
}

// Init sets up an ImageInfo embedded in self (Dart's super constructor):
// self is the outer value, whose Clone/Equal/String override ImageInfo's.
func (i *ImageInfo) Init(self ImageInfoOverrides, image *ui.Image, opts ...ImageInfoOptions) {
	var o ImageInfoOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	i.Image, i.Scale, i.DebugLabel, i.self = image, 1.0, o.DebugLabel, self
	if o.Scale != nil {
		i.Scale = *o.Scale
	}
	dartAssert(foundation.DebugMaybeDispatchCreated("painting", "ImageInfo", i))
}

// SetSelf points the info at the value embedding it.
func (i *ImageInfo) SetSelf(self ImageInfoOverrides) { i.self = self }

// Self is the value this ImageInfo is embedded in (itself when not
// extended).
func (i *ImageInfo) Self() ImageInfoOverrides {
	if i.self == nil {
		return i
	}
	return i.self
}

// Clone creates an ImageInfo with a cloned Image, the same scale and debug
// label. (The completers call Self().Clone(), an extending type's
// override.)
func (i *ImageInfo) Clone() *ImageInfo {
	return NewImageInfo(i.Image.Clone(), ImageInfoOptions{Scale: &i.Scale, DebugLabel: i.DebugLabel})
}

// IsCloneOf reports whether other is a clone of this info: its image is a
// handle to the same data, with the same scale and debug label.
func (i *ImageInfo) IsCloneOf(other *ImageInfo) bool {
	return other.Image.IsCloneOf(i.Image) && other.Scale == i.Scale && strPtrEqual(other.DebugLabel, i.DebugLabel)
}

// SizeBytes is an estimate of the image's memory use (4 bytes per pixel).
func (i *ImageInfo) SizeBytes() int { return i.Image.Height * i.Image.Width * 4 }

// Dispose disposes the image handle.
func (i *ImageInfo) Dispose() {
	if st := i.Image.DebugGetOpenHandleStackTraces(); st != nil && len(st) == 0 {
		panic(dart.NewAssertionError(nil))
	}
	dartAssert(foundation.DebugMaybeDispatchDisposed(i))
	i.Image.Dispose()
}

func (i *ImageInfo) String() string {
	label := ""
	if i.DebugLabel != nil {
		label = *i.DebugLabel + " "
	}
	scale := i.Scale
	return label + i.Image.String() + " @ " + foundation.DebugFormatDouble(&scale) + "x"
}

// Equal is ==: same runtime type, image, scale and debug label.
func (i *ImageInfo) Equal(other *ImageInfo) bool {
	if other == nil || typeName(i.Self()) != typeName(other.Self()) {
		return false
	}
	return other.Image == i.Image && other.Scale == i.Scale && strPtrEqual(other.DebugLabel, i.DebugLabel)
}

func strPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func dartAssert(ok bool) {
	if !ok {
		panic(dart.NewAssertionError(nil))
	}
}

// ---------------------------------------------------------------------------
// Listeners

// ImageListener is called when an image is available (synchronousCall:
// whether it is called during AddListener).
type ImageListener func(image *ImageInfo, synchronousCall bool)

// ImageChunkListener is called with progress while an image loads.
type ImageChunkListener func(event *ImageChunkEvent)

// ImageErrorListener is called when an error occurs while loading an
// image (stackTrace nil: unknown).
type ImageErrorListener func(exception any, stackTrace dart.StackTrace)

// ImageStreamListener is the interface for receiving notifications about
// the loading of an image. Build it with NewImageStreamListener (Dart's
// default reportErrors is true); == is Equal (callbacks compare with
// dart.FunctionsEqual).
type ImageStreamListener struct {
	OnImage      ImageListener
	OnChunk      ImageChunkListener
	OnError      ImageErrorListener
	ReportErrors bool
}

// ImageStreamListenerOptions are ImageStreamListener's named parameters
// (ReportErrors nil: true).
type ImageStreamListenerOptions struct {
	OnChunk      ImageChunkListener
	OnError      ImageErrorListener
	ReportErrors *bool
}

// NewImageStreamListener is ImageStreamListener(onImage, {...}).
func NewImageStreamListener(onImage ImageListener, opts ...ImageStreamListenerOptions) ImageStreamListener {
	var o ImageStreamListenerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return ImageStreamListener{OnImage: onImage, OnChunk: o.OnChunk, OnError: o.OnError,
		ReportErrors: o.ReportErrors == nil || *o.ReportErrors}
}

// Equal is ==: the same callbacks and reportErrors.
func (l ImageStreamListener) Equal(other ImageStreamListener) bool {
	return dart.FunctionsEqual(l.OnImage, other.OnImage) &&
		dart.FunctionsEqual(l.OnChunk, other.OnChunk) &&
		dart.FunctionsEqual(l.OnError, other.OnError) &&
		l.ReportErrors == other.ReportErrors
}

// ImageChunkEvent is an immutable notification of image bytes received
// while loading.
type ImageChunkEvent struct {
	foundation.DiagnosticableMixin
	// CumulativeBytesLoaded is the number of bytes received so far.
	CumulativeBytesLoaded int
	// ExpectedTotalBytes is the total number of bytes expected (nil:
	// unknown).
	ExpectedTotalBytes *int
}

// NewImageChunkEvent is ImageChunkEvent(cumulativeBytesLoaded:,
// expectedTotalBytes:).
func NewImageChunkEvent(cumulativeBytesLoaded int, expectedTotalBytes *int) *ImageChunkEvent {
	dartAssert(cumulativeBytesLoaded >= 0)
	dartAssert(expectedTotalBytes == nil || *expectedTotalBytes >= 0)
	e := &ImageChunkEvent{CumulativeBytesLoaded: cumulativeBytesLoaded, ExpectedTotalBytes: expectedTotalBytes}
	e.SetSelf(e)
	return e
}

func (e *ImageChunkEvent) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	e.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewIntProperty("cumulativeBytesLoaded", e.CumulativeBytesLoaded))
	properties.Add(foundation.NewIntProperty("expectedTotalBytes", e.ExpectedTotalBytes))
}

// ---------------------------------------------------------------------------
// ImageStream

// ImageStream is a handle to an image resource: listeners added before
// its completer is set are handed to the completer when it is.
type ImageStream struct {
	foundation.DiagnosticableMixin

	completer ImageStreamCompleter
	listeners []ImageStreamListener
	hasList   bool // _listeners != null
}

// NewImageStream is ImageStream().
func NewImageStream() *ImageStream {
	s := &ImageStream{}
	s.SetSelf(s)
	return s
}

// Completer is the completer that has been assigned to this stream (nil
// until SetCompleter).
func (s *ImageStream) Completer() ImageStreamCompleter { return s.completer }

// SetCompleter assigns a particular ImageStreamCompleter to this stream
// (once), and hands it the listeners added so far.
func (s *ImageStream) SetCompleter(value ImageStreamCompleter) {
	dartAssert(s.completer == nil)
	s.completer = value
	if s.hasList {
		initialListeners := s.listeners
		s.listeners, s.hasList = nil, false
		base := s.completer.imageStreamCompleter()
		base.addingInitialListeners = true
		for _, l := range initialListeners {
			s.completer.AddListener(l)
		}
		base.addingInitialListeners = false
	}
}

// AddListener adds a listener to the image stream (or to its completer).
func (s *ImageStream) AddListener(listener ImageStreamListener) {
	if s.completer != nil {
		s.completer.AddListener(listener)
		return
	}
	s.hasList = true
	s.listeners = append(s.listeners, listener)
}

// RemoveListener stops listening for events from this stream's completer.
func (s *ImageStream) RemoveListener(listener ImageStreamListener) {
	if s.completer != nil {
		s.completer.RemoveListener(listener)
		return
	}
	dartAssert(s.hasList)
	for i := range s.listeners {
		if s.listeners[i].Equal(listener) {
			s.listeners = append(s.listeners[:i:i], s.listeners[i+1:]...)
			break
		}
	}
}

// Key returns an object identifying the stream's resource: its completer
// once set, the stream itself before.
func (s *ImageStream) Key() any {
	if s.completer != nil {
		return s.completer
	}
	return s
}

func (s *ImageStream) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	s.DiagnosticableMixin.DebugFillProperties(properties)
	var ifPresent *string
	if s.completer != nil {
		v := s.completer.ToStringShort()
		ifPresent = &v
	}
	properties.Add(foundation.NewObjectFlagProperty("completer", s.completer, foundation.ObjectFlagPropertyOptions{
		IfPresent: ifPresent, IfNull: strp("unresolved"),
	}))
	level := foundation.DiagnosticLevelInfo
	if s.completer != nil {
		level = foundation.DiagnosticLevelHidden
	}
	var listeners any
	n := "null"
	if s.hasList {
		listeners = s.listeners
		n = dart.ToString(len(s.listeners))
	}
	plural := "s"
	if s.hasList && len(s.listeners) == 1 {
		plural = ""
	}
	properties.Add(foundation.NewObjectFlagProperty("listeners", listeners, foundation.ObjectFlagPropertyOptions{
		IfPresent: strp(n + " listener" + plural), IfNull: strp("no listeners"), Level: &level,
	}))
	if s.completer != nil {
		s.completer.DebugFillProperties(properties)
	}
}

func strp(s string) *string { return &s }

// ---------------------------------------------------------------------------
// ImageStreamCompleterHandle

// ImageStreamCompleterHandle keeps an ImageStreamCompleter alive (not
// disposed) while it is not disposed.
type ImageStreamCompleterHandle struct {
	completer *ImageStreamCompleterBase
}

func newImageStreamCompleterHandle(c *ImageStreamCompleterBase) *ImageStreamCompleterHandle {
	h := &ImageStreamCompleterHandle{completer: c}
	c.keepAliveHandles++
	dartAssert(foundation.DebugMaybeDispatchCreated("painting", "ImageStreamCompleterHandle", h))
	return h
}

// Dispose releases the handle; the completer may then be disposed.
func (h *ImageStreamCompleterHandle) Dispose() {
	dartAssert(h.completer != nil)
	dartAssert(h.completer.keepAliveHandles > 0)
	dartAssert(!h.completer.disposed)
	h.completer.keepAliveHandles--
	h.completer.outer().maybeDisposeInternal()
	h.completer = nil
	dartAssert(foundation.DebugMaybeDispatchDisposed(h))
}

// ---------------------------------------------------------------------------
// ImageStreamCompleter

// ImageStreamCompleter is the base of the classes that manage loading
// images (abstract in Dart). Embed ImageStreamCompleterBase and call
// InitImageStreamCompleter(self).
type ImageStreamCompleter interface {
	foundation.Diagnosticable
	// AddListener adds a listener; it is called at once (synchronously)
	// with the current image or error, if any.
	AddListener(listener ImageStreamListener)
	// RemoveListener stops listener from receiving events; removing the
	// last one disposes the completer unless a KeepAlive handle is open.
	RemoveListener(listener ImageStreamListener)
	// AddEphemeralErrorListener adds an error listener called once (with
	// the current or next error), unless an image loads first.
	AddEphemeralErrorListener(listener ImageErrorListener)
	// KeepAlive creates a handle preventing the completer's disposal.
	KeepAlive() *ImageStreamCompleterHandle
	// MaybeDispose disposes the completer if it has no listeners and no
	// keep-alive handles.
	MaybeDispose()
	// OnDisposed is called when the completer is disposed (overrides must
	// call the base's).
	OnDisposed()
	// AddOnLastListenerRemovedCallback adds a callback called when the last
	// listener is removed.
	AddOnLastListenerRemovedCallback(callback func())
	// RemoveOnLastListenerRemovedCallback removes such a callback.
	RemoveOnLastListenerRemovedCallback(callback func())
	// SetImage sets the current image and notifies the listeners (Dart:
	// @protected).
	SetImage(image *ImageInfo)
	// ReportError reports an error to the error listeners (and to
	// FlutterError.reportError when none handles it).
	ReportError(exception any, opts ...ImageStreamCompleterReportErrorOptions)
	// ReportImageChunkEvent notifies the chunk listeners (Dart: @protected).
	ReportImageChunkEvent(event *ImageChunkEvent)
	// HasListeners is whether any listener is registered.
	HasListeners() bool
	// DebugLabel is a string identifying the source of the image (nil:
	// none); SetDebugLabel sets it (Dart's public field debugLabel).
	DebugLabel() *string
	SetDebugLabel(label *string)

	imageStreamCompleter() *ImageStreamCompleterBase
	maybeDisposeInternal()
}

// ImageStreamCompleterReportErrorOptions are reportError's named
// parameters (Dart's required exception is ReportError's positional
// parameter).
type ImageStreamCompleterReportErrorOptions struct {
	Context              foundation.DiagnosticsNode
	Stack                dart.StackTrace
	InformationCollector foundation.InformationCollector
	Silent               bool
}

// ImageStreamCompleterBase is ImageStreamCompleter's state and methods.
type ImageStreamCompleterBase struct {
	foundation.DiagnosticableMixin

	self                           ImageStreamCompleter
	listeners                      []ImageStreamListener
	ephemeralErrorListeners        []ImageErrorListener
	currentImage                   *ImageInfo
	currentError                   *foundation.FlutterErrorDetails
	debugLabel                     *string
	addingInitialListeners         bool
	hadErrorListener               bool
	keepAliveHandles               int
	disposed                       bool
	onLastListenerRemovedCallbacks []func()
}

// InitImageStreamCompleter points the base at the completer embedding it
// (call once, when creating it).
func (c *ImageStreamCompleterBase) InitImageStreamCompleter(self ImageStreamCompleter) {
	c.self = self
	c.SetSelf(self)
}

func (c *ImageStreamCompleterBase) outer() ImageStreamCompleter {
	if c.self == nil {
		panic("painting: ImageStreamCompleterBase used before InitImageStreamCompleter")
	}
	return c.self
}

func (c *ImageStreamCompleterBase) imageStreamCompleter() *ImageStreamCompleterBase { return c }

// HasListeners is whether any listener is registered.
func (c *ImageStreamCompleterBase) HasListeners() bool { return len(c.listeners) > 0 }

func (c *ImageStreamCompleterBase) DebugLabel() *string         { return c.debugLabel }
func (c *ImageStreamCompleterBase) SetDebugLabel(label *string) { c.debugLabel = label }

// CurrentImage is the image currently held (nil: none); a flutter-go
// accessor for Dart's private _currentImage, which tests read through the
// diagnostics.
func (c *ImageStreamCompleterBase) CurrentImage() *ImageInfo { return c.currentImage }

// Disposed reports whether the completer was disposed.
func (c *ImageStreamCompleterBase) Disposed() bool { return c.disposed }

// tryCatch runs f, returning what it panicked with and where (Dart's
// catch (exception, stack)).
func tryCatch(f func()) (exception any, stack dart.StackTrace, threw bool) {
	func() {
		defer func() {
			if r := recover(); r != nil {
				exception, stack, threw = r, dart.StackTraceFromPanic(), true
			}
		}()
		f()
	}()
	return
}

// sameObject is Dart's != on exceptions (identity or ==, never panicking
// on uncomparable values).
func sameObject(a, b any) (same bool) {
	defer func() {
		if recover() != nil {
			same = false
		}
	}()
	return a == b
}

func reportImageError(exception any, stack dart.StackTrace, context string) {
	lib := "image resource service"
	foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
		Library: &lib, Context: foundation.NewErrorDescription(context), Stack: stack,
	}))
}

// AddListener adds listener; it is called at once with the current image
// (synchronousCall true unless the stream is handing over its initial
// listeners) or error.
func (c *ImageStreamCompleterBase) AddListener(listener ImageStreamListener) {
	c.checkDisposed()
	self := c.outer()
	// Track that a listener opted out of error reporting.
	if !listener.ReportErrors {
		c.hadErrorListener = true
	}
	c.listeners = append(c.listeners, listener)
	if c.currentImage != nil {
		if exception, stack, threw := tryCatch(func() {
			listener.OnImage(c.currentImage.Self().Clone(), !c.addingInitialListeners)
		}); threw {
			self.ReportError(exception, ImageStreamCompleterReportErrorOptions{
				Context: foundation.NewErrorDescription("by a synchronously-called image listener"),
				Stack:   stack,
			})
		}
	}
	if c.currentError != nil && listener.OnError != nil {
		if newException, newStack, threw := tryCatch(func() {
			listener.OnError(c.currentError.Exception, c.currentError.Stack)
		}); threw {
			if !sameObject(newException, c.currentError.Exception) {
				reportImageError(newException, newStack, "by a synchronously-called image error listener")
			}
		}
	}
}

// AddEphemeralErrorListener calls listener with the current error, or
// keeps it for the next error unless an image has loaded.
func (c *ImageStreamCompleterBase) AddEphemeralErrorListener(listener ImageErrorListener) {
	c.checkDisposed()
	if c.currentError != nil {
		// immediately fire the listener, and no need to add to
		// _ephemeralErrorListeners
		if newException, newStack, threw := tryCatch(func() {
			listener(c.currentError.Exception, c.currentError.Stack)
		}); threw {
			if !sameObject(newException, c.currentError.Exception) {
				reportImageError(newException, newStack, "by a synchronously-called image error listener")
			}
		}
	} else if c.currentImage == nil {
		// add to _ephemeralErrorListeners to wait for the error, only if no
		// image has been loaded
		c.ephemeralErrorListeners = append(c.ephemeralErrorListeners, listener)
	}
}

// KeepAlive creates a handle that keeps the completer from being disposed
// while it is open.
func (c *ImageStreamCompleterBase) KeepAlive() *ImageStreamCompleterHandle {
	c.checkDisposed()
	return newImageStreamCompleterHandle(c)
}

// RemoveListener removes the first listener equal to listener; when none
// is left, the last-listener callbacks run and the completer may be
// disposed.
func (c *ImageStreamCompleterBase) RemoveListener(listener ImageStreamListener) {
	c.checkDisposed()
	for i := range c.listeners {
		if c.listeners[i].Equal(listener) {
			c.listeners = append(c.listeners[:i:i], c.listeners[i+1:]...)
			break
		}
	}
	if len(c.listeners) == 0 {
		callbacks := append([]func(){}, c.onLastListenerRemovedCallbacks...)
		for _, callback := range callbacks {
			callback()
		}
		c.onLastListenerRemovedCallbacks = nil
		c.outer().maybeDisposeInternal()
	}
}

// OnDisposed is called when the completer is disposed.
func (c *ImageStreamCompleterBase) OnDisposed() {}

// MaybeDispose disposes the completer if it has no listeners and no
// keep-alive handles.
func (c *ImageStreamCompleterBase) MaybeDispose() { c.outer().maybeDisposeInternal() }

func (c *ImageStreamCompleterBase) maybeDisposeInternal() {
	if c.disposed || len(c.listeners) > 0 || c.keepAliveHandles != 0 {
		return
	}
	c.ephemeralErrorListeners = nil
	if c.currentImage != nil {
		c.currentImage.Dispose()
	}
	c.currentImage = nil
	c.disposed = true
	c.outer().OnDisposed()
}

func (c *ImageStreamCompleterBase) checkDisposed() {
	if c.disposed {
		panic(dart.StateError{Message: "Stream has been disposed.\n" +
			"An ImageStream is considered disposed once at least one listener has " +
			"been added and subsequently all listeners have been removed and no " +
			"handles are outstanding from the keepAlive method.\n" +
			"To resolve this error, maintain at least one listener on the stream, " +
			"or create an ImageStreamCompleterHandle from the keepAlive " +
			"method, or create a new stream for the image."})
	}
}

// AddOnLastListenerRemovedCallback adds a callback called when the last
// listener is removed.
func (c *ImageStreamCompleterBase) AddOnLastListenerRemovedCallback(callback func()) {
	c.checkDisposed()
	c.onLastListenerRemovedCallbacks = append(c.onLastListenerRemovedCallbacks, callback)
}

// RemoveOnLastListenerRemovedCallback removes the first callback equal to
// callback (dart.FunctionsEqual).
func (c *ImageStreamCompleterBase) RemoveOnLastListenerRemovedCallback(callback func()) {
	c.checkDisposed()
	for i, cb := range c.onLastListenerRemovedCallbacks {
		if dart.FunctionsEqual(cb, callback) {
			c.onLastListenerRemovedCallbacks = append(c.onLastListenerRemovedCallbacks[:i:i], c.onLastListenerRemovedCallbacks[i+1:]...)
			break
		}
	}
}

// SetImage disposes the previous image, keeps image and calls every
// listener with a clone of it.
func (c *ImageStreamCompleterBase) SetImage(image *ImageInfo) {
	c.checkDisposed()
	if c.currentImage != nil {
		c.currentImage.Dispose()
	}
	c.currentImage = image
	c.ephemeralErrorListeners = nil
	if len(c.listeners) == 0 {
		return
	}
	// Make a copy to allow for concurrent modification.
	localListeners := append([]ImageStreamListener(nil), c.listeners...)
	self := c.outer()
	for _, listener := range localListeners {
		if exception, stack, threw := tryCatch(func() {
			listener.OnImage(image.Self().Clone(), false)
		}); threw {
			self.ReportError(exception, ImageStreamCompleterReportErrorOptions{
				Context: foundation.NewErrorDescription("by an image listener"),
				Stack:   stack,
			})
		}
	}
}

// ReportError records the error, calls the error listeners and the
// ephemeral ones, and reports it to FlutterError.reportError if none
// handled it (and no listener opted out of error reporting).
func (c *ImageStreamCompleterBase) ReportError(exception any, opts ...ImageStreamCompleterReportErrorOptions) {
	var o ImageStreamCompleterReportErrorOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	lib := "image resource service"
	c.currentError = foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
		Stack: o.Stack, Library: &lib, Context: o.Context, InformationCollector: o.InformationCollector, Silent: o.Silent,
	})
	// Make a copy to allow for concurrent modification.
	var localErrorListeners []ImageErrorListener
	for _, l := range c.listeners {
		if l.OnError != nil {
			localErrorListeners = append(localErrorListeners, l.OnError)
		}
	}
	localErrorListeners = append(localErrorListeners, c.ephemeralErrorListeners...)
	c.ephemeralErrorListeners = nil
	handled := false
	for _, errorListener := range localErrorListeners {
		if newException, newStack, threw := tryCatch(func() {
			errorListener(exception, o.Stack)
			handled = true
		}); threw {
			if !sameObject(newException, exception) {
				reportImageError(newException, newStack, "when reporting an error to an image listener")
			}
		}
	}
	if !handled {
		// If a listener with reportErrors=false was previously registered,
		// the error was intended to be handled. Skip reporting to
		// FlutterError.onError after the widget is disposed.
		if c.hadErrorListener {
			return
		}
		foundation.FlutterErrorReportError(c.currentError)
	}
}

// ReportImageChunkEvent calls every listener's OnChunk with event.
func (c *ImageStreamCompleterBase) ReportImageChunkEvent(event *ImageChunkEvent) {
	c.checkDisposed()
	if c.outer().HasListeners() {
		// Make a copy to allow for concurrent modification.
		var localListeners []ImageChunkListener
		for _, l := range c.listeners {
			if l.OnChunk != nil {
				localListeners = append(localListeners, l.OnChunk)
			}
		}
		for _, listener := range localListeners {
			listener(event)
		}
	}
}

func (c *ImageStreamCompleterBase) DebugFillProperties(description *foundation.DiagnosticPropertiesBuilder) {
	c.DiagnosticableMixin.DebugFillProperties(description)
	showName := false
	description.Add(foundation.NewDiagnosticsProperty("current", c.currentImage, foundation.DiagnosticsPropertyOptions{
		IfNull: strp("unresolved"), ShowName: &showName,
	}))
	plural := func(n int, word string) string {
		s := dart.ToString(n) + " " + word
		if n != 1 {
			s += "s"
		}
		return s
	}
	description.Add(foundation.NewObjectFlagProperty("listeners", append([]ImageStreamListener{}, c.listeners...), foundation.ObjectFlagPropertyOptions{
		IfPresent: strp(plural(len(c.listeners), "listener")),
	}))
	description.Add(foundation.NewObjectFlagProperty("ephemeralErrorListeners", append([]ImageErrorListener{}, c.ephemeralErrorListeners...), foundation.ObjectFlagPropertyOptions{
		IfPresent: strp(plural(len(c.ephemeralErrorListeners), "ephemeralErrorListener")),
	}))
	description.Add(foundation.NewFlagProperty("disposed", c.disposed, foundation.FlagPropertyOptions{IfTrue: strp("<disposed>")}))
}

// futureErrorValue is the object a Dart future failed with: Go futures
// carry errors, and a non-error Dart throws (a string...) is wrapped in
// async.PanicError.
func futureErrorValue(err error) any {
	if p, ok := err.(async.PanicError); ok {
		return p.Value
	}
	return err
}

// ---------------------------------------------------------------------------
// OneFrameImageStreamCompleter

// OneFrameImageStreamCompleter manages the loading of a single-frame
// image.
type OneFrameImageStreamCompleter struct {
	ImageStreamCompleterBase
}

// OneFrameImageStreamCompleterOptions are OneFrameImageStreamCompleter's
// named parameters.
type OneFrameImageStreamCompleterOptions struct {
	InformationCollector foundation.InformationCollector
}

// NewOneFrameImageStreamCompleter is OneFrameImageStreamCompleter(image,
// {informationCollector}): it sets the image image completes with, or
// reports its error (silently).
func NewOneFrameImageStreamCompleter(image async.Future[*ImageInfo], opts ...OneFrameImageStreamCompleterOptions) *OneFrameImageStreamCompleter {
	var o OneFrameImageStreamCompleterOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	c := &OneFrameImageStreamCompleter{}
	c.InitImageStreamCompleter(c)
	async.OnComplete(image, func(info *ImageInfo) { c.outer().SetImage(info) }, func(err error) {
		c.outer().ReportError(futureErrorValue(err), ImageStreamCompleterReportErrorOptions{
			Context:              foundation.NewErrorDescription("resolving a single-frame image stream"),
			InformationCollector: o.InformationCollector,
			Silent:               true,
		})
	})
	return c
}

// ---------------------------------------------------------------------------
// MultiFrameImageStreamCompleter

// MultiFrameImageStreamCompleter manages the decoding and scheduling of
// image frames (animated images), one frame per app frame at most, in time
// with the frames' durations (scaled by timeDilation).
type MultiFrameImageStreamCompleter struct {
	ImageStreamCompleterBase

	chunkSubscription      async.StreamSubscription[*ImageChunkEvent]
	codec                  ui.Codec
	scale                  float64
	informationCollector   foundation.InformationCollector
	nextFrame              *ui.FrameInfo
	shownTimestamp         time.Duration
	frameDuration          *time.Duration
	framesEmitted          int
	timer                  *async.Timer
	frameCallbackScheduled bool
}

// MultiFrameImageStreamCompleterOptions are MultiFrameImageStreamCompleter's
// optional named parameters (codec and scale, required, are positional).
type MultiFrameImageStreamCompleterOptions struct {
	DebugLabel           *string
	ChunkEvents          async.Stream[*ImageChunkEvent]
	InformationCollector foundation.InformationCollector
}

// NewMultiFrameImageStreamCompleter is MultiFrameImageStreamCompleter(codec:,
// scale:, ...).
func NewMultiFrameImageStreamCompleter(codec async.Future[ui.Codec], scale float64, opts ...MultiFrameImageStreamCompleterOptions) *MultiFrameImageStreamCompleter {
	var o MultiFrameImageStreamCompleterOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	c := &MultiFrameImageStreamCompleter{scale: scale, informationCollector: o.InformationCollector}
	c.InitImageStreamCompleter(c)
	c.debugLabel = o.DebugLabel
	async.OnComplete(codec, c.handleCodecReady, func(err error) {
		c.outer().ReportError(futureErrorValue(err), ImageStreamCompleterReportErrorOptions{
			Context:              foundation.NewErrorDescription("resolving an image codec"),
			InformationCollector: o.InformationCollector,
			Silent:               true,
		})
	})
	if o.ChunkEvents != nil {
		c.chunkSubscription = o.ChunkEvents.Listen(func(e *ImageChunkEvent) { c.outer().ReportImageChunkEvent(e) },
			async.StreamListenOptions{OnError: func(err error) {
				c.outer().ReportError(futureErrorValue(err), ImageStreamCompleterReportErrorOptions{
					Context:              foundation.NewErrorDescription("loading an image"),
					InformationCollector: o.InformationCollector,
					Silent:               true,
				})
			}})
	}
	return c
}

func (c *MultiFrameImageStreamCompleter) handleCodecReady(codec ui.Codec) {
	c.codec = codec
	dartAssert(c.codec != nil)
	if c.outer().HasListeners() {
		c.decodeNextFrameAndSchedule()
	}
}

func (c *MultiFrameImageStreamCompleter) handleAppFrame(timestamp time.Duration) {
	c.frameCallbackScheduled = false
	if !c.outer().HasListeners() {
		return
	}
	// A listener can be re-added after this callback was scheduled (for
	// example when TickerMode toggles during a route transition). That
	// restarts decoding in addListener, which discards _nextFrame. The
	// in-flight decode schedules a new callback when it completes.
	if c.nextFrame == nil {
		return
	}
	if c.isFirstFrame() || c.hasFrameDurationPassed(timestamp) {
		c.emitFrame(NewImageInfo(c.nextFrame.Image.Clone(), ImageInfoOptions{Scale: &c.scale, DebugLabel: c.debugLabel}))
		c.shownTimestamp = timestamp
		d := c.nextFrame.Duration
		c.frameDuration = &d
		c.nextFrame.Image.Dispose()
		c.nextFrame = nil
		if c.codec == nil {
			// codec was disposed during _emitFrame
			return
		}
		completedCycles := c.framesEmitted / c.codec.FrameCount()
		if c.codec.RepetitionCount() == -1 || completedCycles <= c.codec.RepetitionCount() {
			c.decodeNextFrameAndSchedule()
			return
		}
		c.codec.Dispose()
		c.codec = nil
		return
	}
	delay := *c.frameDuration - (timestamp - c.shownTimestamp)
	c.timer = async.NewTimer(durationTimes(delay, scheduler.TimeDilation()), func() {
		c.scheduleAppFrame()
	})
}

// durationTimes is Dart's Duration * num: the microseconds times factor,
// rounded (half away from zero).
func durationTimes(d time.Duration, factor float64) time.Duration {
	return time.Duration(math.Round(float64(d.Microseconds())*factor)) * time.Microsecond
}

func (c *MultiFrameImageStreamCompleter) isFirstFrame() bool { return c.frameDuration == nil }

func (c *MultiFrameImageStreamCompleter) hasFrameDurationPassed(timestamp time.Duration) bool {
	return timestamp-c.shownTimestamp >= *c.frameDuration
}

func (c *MultiFrameImageStreamCompleter) decodeNextFrameAndSchedule() {
	async.AsyncVoid(func() {
		// This will be null if we gave it away. If not, it's still ours and
		// it must be disposed of.
		if c.nextFrame != nil {
			c.nextFrame.Image.Dispose()
		}
		c.nextFrame = nil
		codec := c.codec
		var frame *ui.FrameInfo
		if exception, stack, threw := tryCatch(func() { frame = codec.GetNextFrame().Await() }); threw {
			if err, ok := exception.(error); ok {
				exception = futureErrorValue(err)
			}
			c.outer().ReportError(exception, ImageStreamCompleterReportErrorOptions{
				Context:              foundation.NewErrorDescription("resolving an image frame"),
				Stack:                stack,
				InformationCollector: c.informationCollector,
				Silent:               true,
			})
			return
		}
		c.nextFrame = frame
		if c.codec == nil {
			// codec was disposed during getNextFrame
			return
		}
		if c.codec.FrameCount() == 1 {
			// ImageStreamCompleter listeners removed while waiting for next
			// frame to be decoded. There's no reason to emit the frame
			// without active listeners.
			if !c.outer().HasListeners() {
				return
			}
			// This is not an animated image, just return it and don't
			// schedule more frames.
			c.emitFrame(NewImageInfo(c.nextFrame.Image.Clone(), ImageInfoOptions{Scale: &c.scale, DebugLabel: c.debugLabel}))
			c.nextFrame.Image.Dispose()
			c.nextFrame = nil
			if c.codec != nil {
				c.codec.Dispose()
			}
			c.codec = nil
			return
		}
		c.scheduleAppFrame()
	})
}

func (c *MultiFrameImageStreamCompleter) scheduleAppFrame() {
	if c.frameCallbackScheduled {
		return
	}
	c.frameCallbackScheduled = true
	scheduler.SchedulerBindingInstance().ScheduleFrameCallback(c.handleAppFrame)
}

func (c *MultiFrameImageStreamCompleter) emitFrame(imageInfo *ImageInfo) {
	c.outer().SetImage(imageInfo)
	c.framesEmitted++
}

// AddListener starts decoding when the first listener is added (for an
// animated image, or before the single frame was emitted).
func (c *MultiFrameImageStreamCompleter) AddListener(listener ImageStreamListener) {
	if !c.outer().HasListeners() && c.codec != nil && (c.currentImage == nil || c.codec.FrameCount() > 1) {
		c.decodeNextFrameAndSchedule()
	}
	c.ImageStreamCompleterBase.AddListener(listener)
}

// RemoveListener cancels the frame timer when the last listener goes.
func (c *MultiFrameImageStreamCompleter) RemoveListener(listener ImageStreamListener) {
	c.ImageStreamCompleterBase.RemoveListener(listener)
	if !c.outer().HasListeners() {
		if c.timer != nil {
			c.timer.Cancel()
		}
		c.timer = nil
	}
}

func (c *MultiFrameImageStreamCompleter) maybeDisposeInternal() {
	c.ImageStreamCompleterBase.maybeDisposeInternal()
	if c.disposed {
		if c.chunkSubscription != nil {
			c.chunkSubscription.Cancel()
		}
		c.chunkSubscription = nil
		if c.codec != nil {
			c.codec.Dispose()
		}
		c.codec = nil
	}
}
