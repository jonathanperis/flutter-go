package async

// The part of dart:async's Stream that Flutter's foundation uses, ported
// from the SDK that ships with Flutter 3.50.1 (stream.dart,
// stream_impl.dart, stream_controller.dart): single-subscription
// streams with _BufferingStreamSubscription's state machine and
// _PendingEvents' one-event-per-microtask delivery, the asynchronous
// StreamController (with addStream), Stream.multi, Stream.fromIterable,
// Stream.empty and toList. Not ported: broadcast streams, synchronous
// controllers, transformations (map, where...), async* generators.

import "iter"

// Stream is dart:async's Stream<T>, an interface so Go code can implement
// it. Dart's methods built on listen (toList...) are generic functions.
type Stream[T any] interface {
	// Listen adds a subscription: onData (may be nil) receives the data
	// events; StreamListenOptions has onError, onDone and cancelOnError.
	Listen(onData func(T), opts ...StreamListenOptions) StreamSubscription[T]
}

// StreamListenOptions are Stream.listen's optional parameters. A nil
// OnError makes errors uncaught.
type StreamListenOptions struct {
	OnError       func(error)
	OnDone        func()
	CancelOnError bool
}

// StreamSubscription is dart:async's StreamSubscription<T>.
type StreamSubscription[T any] interface {
	// Cancel stops the subscription; the future completes when the
	// stream has cleaned up.
	Cancel() Future[Void]
	// Pause pauses the subscription until Resume (or until resumeSignal,
	// when not nil, completes).
	Pause(resumeSignal Future[Void])
	// Resume undoes one Pause.
	Resume()
	// IsPaused reports whether the subscription is paused.
	IsPaused() bool
}

// nullFuture is Future._nullFuture, the loop's completed Future<void>.
func (l *Loop) nullFuture() *future[Void] {
	if l.nullFut == nil {
		l.nullFut = &future[Void]{loop: l, state: hasValue}
	}
	return l.nullFut
}

func (l *Loop) guardCallback(f func()) { l.guard(f) }

// ---- _PendingEvents ----

type delayedEvent[T any] struct {
	kind  int // 0 data, 1 error, 2 done
	value T
	err   error
	next  *delayedEvent[T]
}

type eventDispatch[T any] interface {
	sendData(T)
	sendError(error)
	sendDone()
}

func (e *delayedEvent[T]) perform(d eventDispatch[T]) {
	switch e.kind {
	case 0:
		d.sendData(e.value)
	case 1:
		d.sendError(e.err)
	default:
		d.sendDone()
	}
}

const (
	pendingUnscheduled = 0
	pendingScheduled   = 1
	pendingCanceled    = 3
)

type pendingEvents[T any] struct {
	state       int
	first, last *delayedEvent[T]
}

func (p *pendingEvents[T]) isScheduled() bool    { return p.state == pendingScheduled }
func (p *pendingEvents[T]) eventScheduled() bool { return p.state >= pendingScheduled }
func (p *pendingEvents[T]) isEmpty() bool        { return p.last == nil }

func (p *pendingEvents[T]) schedule(d eventDispatch[T]) {
	if p.isScheduled() {
		return
	}
	if p.eventScheduled() {
		p.state = pendingScheduled
		return
	}
	ScheduleMicrotask(func() {
		old := p.state
		p.state = pendingUnscheduled
		if old == pendingCanceled {
			return
		}
		p.handleNext(d)
	})
	p.state = pendingScheduled
}

func (p *pendingEvents[T]) cancelSchedule() {
	if p.isScheduled() {
		p.state = pendingCanceled
	}
}

func (p *pendingEvents[T]) add(e *delayedEvent[T]) {
	if p.last == nil {
		p.first, p.last = e, e
	} else {
		p.last.next = e
		p.last = e
	}
}

func (p *pendingEvents[T]) handleNext(d eventDispatch[T]) {
	e := p.first
	p.first = e.next
	if p.first == nil {
		p.last = nil
	}
	e.perform(d)
}

// ---- _BufferingStreamSubscription ----

const (
	subCancelOnError   = 1 << 0
	subClosed          = 1 << 1
	subInputPaused     = 1 << 2
	subCanceled        = 1 << 3
	subWaitForCancel   = 1 << 4
	subHasErrorHandler = 1 << 5
	subInCallback      = 1 << 6
	subHasPending      = 1 << 7
	subPauseCount      = 1 << 8
)

// subscriptionHooks are _BufferingStreamSubscription's overridable
// _onPause/_onResume/_onCancel (a controller's subscription reports them
// to the controller).
type subscriptionHooks interface {
	onPause()
	onResume()
	onCancel() Future[Void] // nil for none
}

type bufferingSubscription[T any] struct {
	loop      *Loop
	onData    func(T)
	onError   func(error)
	onDone    func()
	state     int
	cancelFut Future[Void]
	pending   *pendingEvents[T]
	hooks     subscriptionHooks
}

func newBufferingSubscription[T any](onData func(T), opts []StreamListenOptions) *bufferingSubscription[T] {
	var o StreamListenOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	s := &bufferingSubscription[T]{loop: current}
	if o.CancelOnError {
		s.state |= subCancelOnError
	}
	if o.OnError != nil {
		s.state |= subHasErrorHandler
	}
	s.onData = onData
	if s.onData == nil {
		s.onData = func(T) {}
	}
	s.onError = o.OnError
	if s.onError == nil {
		l := s.loop
		s.onError = func(err error) { l.reportUncaught(err) }
	}
	s.onDone = o.OnDone
	if s.onDone == nil {
		s.onDone = func() {}
	}
	return s
}

func (s *bufferingSubscription[T]) setPendingEvents(p *pendingEvents[T]) {
	if p == nil {
		return
	}
	s.pending = p
	if !p.isEmpty() {
		s.state |= subHasPending
		p.schedule(s)
	}
}

func (s *bufferingSubscription[T]) isInputPaused() bool { return s.state&subInputPaused != 0 }
func (s *bufferingSubscription[T]) isClosed() bool      { return s.state&subClosed != 0 }
func (s *bufferingSubscription[T]) isCanceled() bool    { return s.state&subCanceled != 0 }
func (s *bufferingSubscription[T]) waitsForCancel() bool {
	return s.state&subWaitForCancel != 0
}
func (s *bufferingSubscription[T]) inCallback() bool { return s.state&subInCallback != 0 }
func (s *bufferingSubscription[T]) hasPending() bool { return s.state&subHasPending != 0 }
func (s *bufferingSubscription[T]) isPaused() bool   { return s.state >= subPauseCount }
func (s *bufferingSubscription[T]) canFire() bool    { return s.state < subInCallback }
func (s *bufferingSubscription[T]) cancelOnError() bool {
	return s.state&subCancelOnError != 0
}
func (s *bufferingSubscription[T]) mayResumeInput() bool {
	return !s.isPaused() && (s.pending == nil || s.pending.isEmpty())
}

func (s *bufferingSubscription[T]) IsPaused() bool { return s.isPaused() }

func (s *bufferingSubscription[T]) Pause(resumeSignal Future[Void]) {
	if s.isCanceled() {
		return
	}
	wasPaused := s.isPaused()
	wasInputPaused := s.isInputPaused()
	s.state = (s.state + subPauseCount) | subInputPaused
	if resumeSignal != nil {
		resumeSignal.WhenComplete(s.Resume)
	}
	if !wasPaused && s.pending != nil {
		s.pending.cancelSchedule()
	}
	if !wasInputPaused && !s.inCallback() {
		s.guardCallback(s.callOnPause)
	}
}

func (s *bufferingSubscription[T]) Resume() {
	if s.isCanceled() {
		return
	}
	if s.isPaused() {
		s.state -= subPauseCount
		if !s.isPaused() {
			if s.hasPending() && !s.pending.isEmpty() {
				s.pending.schedule(s)
			} else {
				s.state &^= subInputPaused
				if !s.inCallback() {
					s.guardCallback(s.callOnResume)
				}
			}
		}
	}
}

func (s *bufferingSubscription[T]) Cancel() Future[Void] {
	s.state &^= subWaitForCancel
	if !s.isCanceled() {
		s.cancel()
	}
	if s.cancelFut != nil {
		return s.cancelFut
	}
	return s.loop.nullFuture()
}

func (s *bufferingSubscription[T]) cancel() {
	s.state |= subCanceled
	if s.hasPending() {
		s.pending.cancelSchedule()
	}
	if !s.inCallback() {
		s.pending = nil
	}
	s.cancelFut = s.callOnCancel()
}

func (s *bufferingSubscription[T]) callOnPause() {
	if s.hooks != nil {
		s.hooks.onPause()
	}
}

func (s *bufferingSubscription[T]) callOnResume() {
	if s.hooks != nil {
		s.hooks.onResume()
	}
}

func (s *bufferingSubscription[T]) callOnCancel() Future[Void] {
	if s.hooks != nil {
		return s.hooks.onCancel()
	}
	return nil
}

func (s *bufferingSubscription[T]) add(data T) {
	if s.isCanceled() {
		return
	}
	if s.canFire() {
		s.sendData(data)
	} else {
		s.addPending(&delayedEvent[T]{kind: 0, value: data})
	}
}

func (s *bufferingSubscription[T]) addError(err error) {
	if s.isCanceled() {
		return
	}
	if s.canFire() {
		s.sendError(err)
	} else {
		s.addPending(&delayedEvent[T]{kind: 1, err: err})
	}
}

func (s *bufferingSubscription[T]) close() {
	if s.isCanceled() {
		return
	}
	s.state |= subClosed
	if s.canFire() {
		s.sendDone()
	} else {
		s.addPending(&delayedEvent[T]{kind: 2})
	}
}

func (s *bufferingSubscription[T]) addPending(e *delayedEvent[T]) {
	if s.pending == nil {
		s.pending = &pendingEvents[T]{}
	}
	s.pending.add(e)
	if !s.hasPending() {
		s.state |= subHasPending
		if !s.isPaused() {
			s.pending.schedule(s)
		}
	}
}

func (s *bufferingSubscription[T]) sendData(data T) {
	wasInputPaused := s.isInputPaused()
	s.state |= subInCallback
	s.loop.guardCallback(func() { s.onData(data) })
	s.state &^= subInCallback
	s.checkState(wasInputPaused)
}

func (s *bufferingSubscription[T]) sendError(err error) {
	wasInputPaused := s.isInputPaused()
	send := func() {
		if s.isCanceled() && !s.waitsForCancel() {
			return
		}
		s.state |= subInCallback
		s.loop.guardCallback(func() { s.onError(err) })
		s.state &^= subInCallback
	}
	if s.cancelOnError() {
		s.state |= subWaitForCancel
		s.cancel()
		if s.cancelFut != nil && s.cancelFut != Future[Void](s.loop.nullFuture()) {
			s.cancelFut.WhenComplete(send)
		} else {
			send()
		}
	} else {
		send()
		s.checkState(wasInputPaused)
	}
}

func (s *bufferingSubscription[T]) sendDone() {
	send := func() {
		if !s.waitsForCancel() {
			return
		}
		s.state |= subCanceled | subClosed | subInCallback
		s.loop.guardCallback(s.onDone)
		s.state &^= subInCallback
	}
	s.cancel()
	s.state |= subWaitForCancel
	if s.cancelFut != nil && s.cancelFut != Future[Void](s.loop.nullFuture()) {
		s.cancelFut.WhenComplete(send)
	} else {
		send()
	}
}

func (s *bufferingSubscription[T]) guardCallback(callback func()) {
	wasInputPaused := s.isInputPaused()
	s.state |= subInCallback
	callback()
	s.state &^= subInCallback
	s.checkState(wasInputPaused)
}

func (s *bufferingSubscription[T]) checkState(wasInputPaused bool) {
	if s.hasPending() && s.pending.isEmpty() {
		s.state &^= subHasPending
		if s.isInputPaused() && s.mayResumeInput() {
			s.state &^= subInputPaused
		}
	}
	for {
		if s.isCanceled() {
			s.pending = nil
			return
		}
		isInputPaused := s.isInputPaused()
		if wasInputPaused == isInputPaused {
			break
		}
		s.state ^= subInCallback
		if isInputPaused {
			s.callOnPause()
		} else {
			s.callOnResume()
		}
		s.state &^= subInCallback
		wasInputPaused = isInputPaused
	}
	if s.hasPending() && !s.isPaused() {
		s.pending.schedule(s)
	}
}

// ---- _DoneStreamSubscription and Stream.empty ----

type doneStreamSubscription[T any] struct {
	loop   *Loop
	state  int
	onDone func()
}

const (
	doneReadyToSend = 0
	doneScheduled   = 1
	doneDone        = -1
	donePausedOnce  = 2
)

func newDoneStreamSubscription[T any](onDone func()) *doneStreamSubscription[T] {
	s := &doneStreamSubscription[T]{loop: current, state: doneScheduled, onDone: onDone}
	ScheduleMicrotask(s.onMicrotask)
	return s
}

func (s *doneStreamSubscription[T]) IsPaused() bool { return s.state >= donePausedOnce }

func (s *doneStreamSubscription[T]) Pause(resumeSignal Future[Void]) {
	if s.state >= 0 {
		s.state += donePausedOnce
		if resumeSignal != nil {
			resumeSignal.WhenComplete(s.Resume)
		}
	}
}

func (s *doneStreamSubscription[T]) Resume() {
	resumeState := s.state - donePausedOnce
	if resumeState < 0 {
		return
	}
	if resumeState == doneReadyToSend {
		s.state = doneScheduled
		ScheduleMicrotask(s.onMicrotask)
	} else {
		s.state = resumeState
	}
}

func (s *doneStreamSubscription[T]) Cancel() Future[Void] {
	s.state = doneDone
	s.onDone = nil
	return s.loop.nullFuture()
}

func (s *doneStreamSubscription[T]) onMicrotask() {
	unscheduled := s.state - doneScheduled
	if unscheduled == doneReadyToSend {
		s.state = doneDone
		if h := s.onDone; h != nil {
			s.onDone = nil
			s.loop.guardCallback(h)
		}
	} else {
		s.state = unscheduled
	}
}

type emptyStream[T any] struct{}

// StreamEmpty is Stream.empty(): a stream that is done as soon as it is
// listened to (in a microtask).
func StreamEmpty[T any]() Stream[T] { return emptyStream[T]{} }

func (emptyStream[T]) Listen(onData func(T), opts ...StreamListenOptions) StreamSubscription[T] {
	var onDone func()
	if len(opts) > 0 {
		onDone = opts[0].OnDone
	}
	return newDoneStreamSubscription[T](onDone)
}

// ---- StreamController ----

const (
	ctrlInitial          = 0
	ctrlSubscribed       = 1
	ctrlCanceled         = 2
	ctrlSubscriptionMask = 3
	ctrlClosed           = 4
	ctrlAddStream        = 8
)

// StreamControllerOptions are StreamController's optional callbacks.
type StreamControllerOptions struct {
	OnListen func()
	OnPause  func()
	OnResume func()
	// OnCancel may return a future to wait for (nil for none).
	OnCancel func() Future[Void]
}

// StreamController is dart:async's (asynchronous, single-subscription)
// StreamController<T>. (Synchronous controllers are used internally only,
// by StreamFromFuture.)
type StreamController[T any] struct {
	OnListen func()
	OnPause  func()
	OnResume func()
	OnCancel func() Future[Void]

	loop         *Loop
	sync         bool // _SyncStreamControllerDispatch
	state        int
	pendingEvs   *pendingEvents[T]
	subscription *controllerSubscription[T]
	addState     *addStreamState[T]
	doneFuture   *future[Void]
}

// NewStreamController is StreamController({onListen, onPause, onResume,
// onCancel}).
func NewStreamController[T any](opts ...StreamControllerOptions) *StreamController[T] {
	c := &StreamController[T]{loop: current}
	if len(opts) > 0 {
		c.OnListen, c.OnPause, c.OnResume, c.OnCancel = opts[0].OnListen, opts[0].OnPause, opts[0].OnResume, opts[0].OnCancel
	}
	return c
}

// Stream is the stream this controller controls.
func (c *StreamController[T]) Stream() Stream[T] { return controllerStream[T]{c} }

func (c *StreamController[T]) isCanceled() bool { return c.state&ctrlCanceled != 0 }
func (c *StreamController[T]) isInitialState() bool {
	return c.state&ctrlSubscriptionMask == ctrlInitial
}
func (c *StreamController[T]) isAddingStream() bool { return c.state&ctrlAddStream != 0 }
func (c *StreamController[T]) mayAddEvent() bool    { return c.state < ctrlClosed }

// HasListener reports whether the stream is listened to (and not
// canceled).
func (c *StreamController[T]) HasListener() bool { return c.state&ctrlSubscribed != 0 }

// IsClosed reports whether Close was called.
func (c *StreamController[T]) IsClosed() bool { return c.state&ctrlClosed != 0 }

// IsPaused reports whether the subscription would buffer events.
func (c *StreamController[T]) IsPaused() bool {
	if c.HasListener() {
		return c.subscription.isInputPaused()
	}
	return !c.isCanceled()
}

func (c *StreamController[T]) ensurePendingEvents() *pendingEvents[T] {
	if c.pendingEvs == nil {
		c.pendingEvs = &pendingEvents[T]{}
	}
	return c.pendingEvs
}

func (c *StreamController[T]) badEventState() error {
	if c.IsClosed() {
		return stateError("Cannot add event after closing")
	}
	return stateError("Cannot add event while adding a stream")
}

type stateError string

func (e stateError) Error() string { return "Bad state: " + string(e) }

// AddStream sends source's events through the controller and completes
// when source is done (cancelOnError: an error ends it).
func (c *StreamController[T]) AddStream(source Stream[T], cancelOnError bool) Future[Void] {
	if !c.mayAddEvent() {
		panic(c.badEventState())
	}
	if c.isCanceled() {
		f := newFuture[Void]()
		f.asyncCompleteWithValue(Void{})
		return f
	}
	c.addState = newAddStreamState(c, source, cancelOnError)
	c.state |= ctrlAddStream
	return c.addState.addStreamFuture
}

// Done completes when the controller has sent its done event.
func (c *StreamController[T]) Done() Future[Void] { return c.ensureDoneFuture() }

func (c *StreamController[T]) ensureDoneFuture() Future[Void] {
	if c.doneFuture == nil {
		if c.isCanceled() {
			return c.loop.nullFuture()
		}
		c.doneFuture = newFuture[Void]()
	}
	return c.doneFuture
}

// Add sends a data event.
func (c *StreamController[T]) Add(value T) {
	if !c.mayAddEvent() {
		panic(c.badEventState())
	}
	c.add(value)
}

// AddError sends an error event.
func (c *StreamController[T]) AddError(err error) {
	if !c.mayAddEvent() {
		panic(c.badEventState())
	}
	c.addError(err)
}

// Close sends the done event; the future is Done().
func (c *StreamController[T]) Close() Future[Void] {
	if c.IsClosed() {
		return c.ensureDoneFuture()
	}
	if !c.mayAddEvent() {
		panic(c.badEventState())
	}
	c.closeUnchecked()
	return c.ensureDoneFuture()
}

func (c *StreamController[T]) closeUnchecked() {
	c.state |= ctrlClosed
	if c.HasListener() {
		c.sendDone()
	} else if c.isInitialState() {
		c.ensurePendingEvents().add(&delayedEvent[T]{kind: 2})
	}
}

func (c *StreamController[T]) add(value T) {
	if c.HasListener() {
		c.sendData(value)
	} else if c.isInitialState() {
		c.ensurePendingEvents().add(&delayedEvent[T]{kind: 0, value: value})
	}
}

func (c *StreamController[T]) addError(err error) {
	if c.HasListener() {
		c.sendError(err)
	} else if c.isInitialState() {
		c.ensurePendingEvents().add(&delayedEvent[T]{kind: 1, err: err})
	}
}

// sendData, sendError and sendDone are the controller's dispatch: an
// asynchronous controller queues the event on the subscription, a
// synchronous one delivers it at once.
func (c *StreamController[T]) sendData(value T) {
	if c.sync {
		c.subscription.add(value)
		return
	}
	c.subscription.addPending(&delayedEvent[T]{kind: 0, value: value})
}

func (c *StreamController[T]) sendError(err error) {
	if c.sync {
		c.subscription.addError(err)
		return
	}
	c.subscription.addPending(&delayedEvent[T]{kind: 1, err: err})
}

func (c *StreamController[T]) sendDone() {
	if c.sync {
		c.subscription.close()
		return
	}
	c.subscription.addPending(&delayedEvent[T]{kind: 2})
}

// closeAddStream is _close: the added stream is done.
func (c *StreamController[T]) closeAddStream() {
	st := c.addState
	c.addState = nil
	c.state &^= ctrlAddStream
	st.complete()
}

func (c *StreamController[T]) subscribe(onData func(T), opts []StreamListenOptions) StreamSubscription[T] {
	if !c.isInitialState() {
		panic(stateError("Stream has already been listened to."))
	}
	sub := &controllerSubscription[T]{bufferingSubscription: newBufferingSubscription(onData, opts), controller: c}
	sub.hooks = sub
	pending := c.pendingEvs
	c.pendingEvs = nil
	c.state |= ctrlSubscribed
	c.subscription = sub
	if c.isAddingStream() {
		c.addState.resume()
	}
	sub.setPendingEvents(pending)
	sub.guardCallback(func() { runGuarded(c.OnListen) })
	return sub
}

func runGuarded(handler func()) {
	if handler == nil {
		return
	}
	current.guard(handler)
}

func (c *StreamController[T]) recordCancel() Future[Void] {
	var result Future[Void]
	if c.isAddingStream() {
		result = c.addState.cancel()
	}
	c.subscription = nil
	c.addState = nil
	c.state = (c.state &^ (ctrlSubscribed | ctrlAddStream)) | ctrlCanceled
	if c.OnCancel != nil {
		if result == nil {
			r, err := call(func() Future[Void] { return c.OnCancel() })
			if err != nil {
				f := newFuture[Void]()
				f.asyncCompleteError(err)
				result = f
			} else {
				result = r
			}
		} else {
			result = result.WhenComplete(func() { c.OnCancel() })
		}
	}
	complete := func() {
		if d := c.doneFuture; d != nil && d.state == incomplete {
			d.asyncCompleteWithValue(Void{})
		}
	}
	if result != nil {
		result = result.WhenComplete(complete)
	} else {
		complete()
	}
	return result
}

func (c *StreamController[T]) recordPause() {
	if c.isAddingStream() {
		c.addState.pause()
	}
	runGuarded(c.OnPause)
}

func (c *StreamController[T]) recordResume() {
	if c.isAddingStream() {
		c.addState.resume()
	}
	runGuarded(c.OnResume)
}

type controllerStream[T any] struct{ c *StreamController[T] }

func (s controllerStream[T]) Listen(onData func(T), opts ...StreamListenOptions) StreamSubscription[T] {
	return s.c.subscribe(onData, opts)
}

type controllerSubscription[T any] struct {
	*bufferingSubscription[T]
	controller *StreamController[T]
}

func (s *controllerSubscription[T]) onCancel() Future[Void] { return s.controller.recordCancel() }
func (s *controllerSubscription[T]) onPause()               { s.controller.recordPause() }
func (s *controllerSubscription[T]) onResume()              { s.controller.recordResume() }

// sendData etc. make the subscription an eventDispatch for its own
// pending events (the embedded methods).

type addStreamState[T any] struct {
	addStreamFuture *future[Void]
	addSubscription StreamSubscription[T]
}

func newAddStreamState[T any](c *StreamController[T], source Stream[T], cancelOnError bool) *addStreamState[T] {
	st := &addStreamState[T]{addStreamFuture: newFuture[Void]()}
	onError := c.addError
	if cancelOnError {
		onError = func(err error) {
			c.addError(err)
			c.closeAddStream()
		}
	}
	st.addSubscription = source.Listen(c.add, StreamListenOptions{
		OnError:       onError,
		OnDone:        c.closeAddStream,
		CancelOnError: cancelOnError,
	})
	if c.IsPaused() {
		st.addSubscription.Pause(nil)
	}
	return st
}

func (st *addStreamState[T]) pause()  { st.addSubscription.Pause(nil) }
func (st *addStreamState[T]) resume() { st.addSubscription.Resume() }

func (st *addStreamState[T]) cancel() Future[Void] {
	c := st.addSubscription.Cancel()
	if c == nil {
		st.addStreamFuture.asyncCompleteWithValue(Void{})
		return current.nullFuture()
	}
	return c.WhenComplete(func() { st.addStreamFuture.asyncCompleteWithValue(Void{}) })
}

func (st *addStreamState[T]) complete() { st.addStreamFuture.asyncCompleteWithValue(Void{}) }

// ---- Stream.multi and Stream.fromIterable ----

// MultiStreamController is the controller Stream.multi gives each
// listener.
type MultiStreamController[T any] struct {
	*StreamController[T]
}

// AddSync sends a data event to the listener now, if it can receive one.
func (m MultiStreamController[T]) AddSync(data T) {
	if !m.mayAddEvent() {
		panic(m.badEventState())
	}
	if m.HasListener() {
		m.subscription.add(data)
	}
}

// AddErrorSync sends an error event now, if possible.
func (m MultiStreamController[T]) AddErrorSync(err error) {
	if !m.mayAddEvent() {
		panic(m.badEventState())
	}
	if m.HasListener() {
		m.subscription.addError(err)
	}
}

// CloseSync sends the done event now, if possible.
func (m MultiStreamController[T]) CloseSync() {
	if m.IsClosed() {
		return
	}
	if !m.mayAddEvent() {
		panic(m.badEventState())
	}
	m.state |= ctrlClosed
	if m.HasListener() {
		m.subscription.close()
	}
}

type multiStream[T any] struct {
	onListen func(MultiStreamController[T])
}

// StreamMulti is Stream.multi(onListen): each listener gets its own
// controller, passed to onListen.
func StreamMulti[T any](onListen func(MultiStreamController[T])) Stream[T] {
	return multiStream[T]{onListen}
}

func (s multiStream[T]) Listen(onData func(T), opts ...StreamListenOptions) StreamSubscription[T] {
	c := MultiStreamController[T]{NewStreamController[T]()}
	c.OnListen = func() { s.onListen(c) }
	return c.subscribe(onData, opts)
}

// StreamFromIterable is Stream.fromIterable(elements): one element per
// microtask while the listener is not paused.
func StreamFromIterable[T any](elements iter.Seq[T]) Stream[T] {
	return StreamMulti(func(controller MultiStreamController[T]) {
		next, stop := iter.Pull(elements)
		isScheduled := true
		var step func()
		step = func() {
			if !controller.HasListener() || controller.IsPaused() {
				isScheduled = false
				return
			}
			value, hasNext, err := pullNext(next)
			if err != nil {
				stop()
				controller.AddErrorSync(err)
				controller.CloseSync()
				return
			}
			if hasNext {
				controller.AddSync(value)
				if controller.HasListener() && !controller.IsPaused() {
					ScheduleMicrotask(step)
				} else {
					isScheduled = false
				}
			} else {
				stop()
				controller.CloseSync()
			}
		}
		controller.OnResume = func() {
			if !isScheduled {
				isScheduled = true
				ScheduleMicrotask(step)
			}
		}
		ScheduleMicrotask(step)
	})
}

func pullNext[T any](next func() (T, bool)) (v T, ok bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = asError(r)
		}
	}()
	v, ok = next()
	return
}

// StreamToList is stream.toList(): a future of every data event, completed
// when the stream is done (or failed with its first error).
func StreamToList[T any](s Stream[T]) Future[[]T] {
	result := []T{}
	f := newFuture[[]T]()
	s.Listen(func(data T) { result = append(result, data) }, StreamListenOptions{
		OnError:       f.completeError,
		OnDone:        func() { f.completeWithValue(result) },
		CancelOnError: true,
	})
	return f
}

// StreamFromFuture is Stream.fromFuture(future): a stream of the future's
// value (or error), then done.
func StreamFromFuture[T any](f Future[T]) Stream[T] {
	c := NewStreamController[T]()
	c.sync = true
	OnComplete(f, func(value T) {
		c.add(value)
		c.closeUnchecked()
	}, func(err error) {
		c.addError(err)
		c.closeUnchecked()
	})
	return c.Stream()
}
