package async

// Broadcast stream controllers, ported from the Dart SDK that ships with
// Flutter 3.50.1 (broadcast_stream_controller.dart): StreamController
// .broadcast, both the asynchronous variant (_AsyncBroadcastStreamController:
// each listener gets the events through its own _PendingEvents, one event
// per microtask) and the synchronous one (_SyncBroadcastStreamController:
// events are delivered at once, with the "firing" state that defers the
// removal of listeners canceled while an event is being delivered).
// _AsBroadcastStreamController (Stream.asBroadcastStream) is not ported.

// StreamControllerBroadcastOptions are StreamController.broadcast's
// optional parameters. OnListen runs when the first listener subscribes
// (again after all listeners have left), OnCancel when the last one leaves
// (Dart's onCancel is a `void Function()`: a returned future is ignored).
type StreamControllerBroadcastOptions struct {
	OnListen func()
	OnCancel func()
	Sync     bool
}

const (
	bcEventID   = 1
	bcFiring    = 2
	bcClosed    = 4
	bcAddStream = 8

	bsubEventID           = 1
	bsubFiring            = 2
	bsubRemoveAfterFiring = 4
)

// BroadcastStreamController is dart:async's _BroadcastStreamController<T>
// (the controller StreamController.broadcast returns): any number of
// listeners, events go to the listeners subscribed when they are added.
type BroadcastStreamController[T any] struct {
	OnListen func()
	OnCancel func()

	loop       *Loop
	sync       bool
	state      int
	first      *broadcastSubscription[T]
	last       *broadcastSubscription[T]
	addState   *broadcastAddStreamState[T]
	doneFuture *future[Void]
}

// NewStreamControllerBroadcast is StreamController.broadcast({onListen,
// onCancel, sync}).
func NewStreamControllerBroadcast[T any](opts ...StreamControllerBroadcastOptions) *BroadcastStreamController[T] {
	c := &BroadcastStreamController[T]{loop: current}
	if len(opts) > 0 {
		c.OnListen, c.OnCancel, c.sync = opts[0].OnListen, opts[0].OnCancel, opts[0].Sync
	}
	return c
}

type broadcastStream[T any] struct{ c *BroadcastStreamController[T] }

func (s broadcastStream[T]) Listen(onData func(T), opts ...StreamListenOptions) StreamSubscription[T] {
	return s.c.subscribe(onData, opts)
}

// IsBroadcast is Stream.isBroadcast (true).
func (s broadcastStream[T]) IsBroadcast() bool { return true }

// Stream is the stream this controller controls; it can be listened to
// any number of times.
func (c *BroadcastStreamController[T]) Stream() Stream[T] { return broadcastStream[T]{c} }

// IsClosed reports whether Close was called.
func (c *BroadcastStreamController[T]) IsClosed() bool { return c.state&bcClosed != 0 }

// IsPaused is always false: a broadcast controller never buffers.
func (c *BroadcastStreamController[T]) IsPaused() bool { return false }

// HasListener reports whether there is at least one active listener.
func (c *BroadcastStreamController[T]) HasListener() bool { return !c.isEmpty() }

func (c *BroadcastStreamController[T]) hasOneListener() bool { return c.first == c.last }
func (c *BroadcastStreamController[T]) isFiring() bool       { return c.state&bcFiring != 0 }
func (c *BroadcastStreamController[T]) isAddingStream() bool { return c.state&bcAddStream != 0 }
func (c *BroadcastStreamController[T]) isEmpty() bool        { return c.first == nil }

func (c *BroadcastStreamController[T]) mayAddEvent() bool {
	if c.sync {
		return c.state < bcClosed && !c.isFiring()
	}
	return c.state < bcClosed
}

func (c *BroadcastStreamController[T]) ensureDoneFuture() *future[Void] {
	if c.doneFuture == nil {
		c.doneFuture = newFuture[Void]()
	}
	return c.doneFuture
}

func (c *BroadcastStreamController[T]) addListener(s *broadcastSubscription[T]) {
	s.eventState = c.state & bcEventID
	oldLast := c.last
	c.last = s
	s.next = nil
	s.previous = oldLast
	if oldLast == nil {
		c.first = s
	} else {
		oldLast.next = s
	}
}

func (c *BroadcastStreamController[T]) removeListener(s *broadcastSubscription[T]) {
	previous, next := s.previous, s.next
	if previous == nil {
		c.first = next
	} else {
		previous.next = next
	}
	if next == nil {
		c.last = previous
	} else {
		next.previous = previous
	}
	s.next, s.previous = s, s
}

func (c *BroadcastStreamController[T]) subscribe(onData func(T), opts []StreamListenOptions) StreamSubscription[T] {
	if c.IsClosed() {
		var onDone func()
		if len(opts) > 0 {
			onDone = opts[0].OnDone
		}
		return newDoneStreamSubscription[T](onDone)
	}
	s := &broadcastSubscription[T]{bufferingSubscription: newBufferingSubscription(onData, opts), controller: c}
	s.next, s.previous = s, s
	s.hooks = s
	c.addListener(s)
	if c.first == c.last {
		// Only one listener, so it must be the first listener.
		runGuarded(c.OnListen)
	}
	return s
}

func (c *BroadcastStreamController[T]) recordCancel(s *broadcastSubscription[T]) Future[Void] {
	// If already removed by the stream, don't remove it again.
	if s.next == s {
		return nil
	}
	if s.isFiring() {
		s.eventState |= bsubRemoveAfterFiring
	} else {
		c.removeListener(s)
		// If we are currently firing an event, the empty-check is performed
		// at the end of the listener loop instead of here.
		if !c.isFiring() && c.isEmpty() {
			c.callOnCancel()
		}
	}
	return nil
}

func (c *BroadcastStreamController[T]) addEventError() error {
	if c.sync && c.isFiring() {
		return stateError("Cannot fire new event. Controller is already firing an event")
	}
	if c.IsClosed() {
		return stateError("Cannot add new events after calling close")
	}
	return stateError("Cannot add new events while doing an addStream")
}

// Add sends a data event to the current listeners.
func (c *BroadcastStreamController[T]) Add(data T) {
	if !c.mayAddEvent() {
		panic(c.addEventError())
	}
	c.sendData(data)
}

// AddError sends an error event to the current listeners.
func (c *BroadcastStreamController[T]) AddError(err error) {
	if !c.mayAddEvent() {
		panic(c.addEventError())
	}
	c.sendError(err)
}

// Close sends the done event to the current listeners; the future (Done)
// completes once they are all gone.
func (c *BroadcastStreamController[T]) Close() Future[Void] {
	if c.IsClosed() {
		return c.doneFuture
	}
	if !c.mayAddEvent() {
		panic(c.addEventError())
	}
	c.state |= bcClosed
	doneFuture := c.ensureDoneFuture()
	c.sendDone()
	return doneFuture
}

// Done completes when the controller is closed and has no listeners left.
func (c *BroadcastStreamController[T]) Done() Future[Void] { return c.ensureDoneFuture() }

// AddStream sends source's events to the listeners and completes when
// source is done (with cancelOnError, an error ends it).
func (c *BroadcastStreamController[T]) AddStream(source Stream[T], cancelOnError bool) Future[Void] {
	if !c.mayAddEvent() {
		panic(c.addEventError())
	}
	c.state |= bcAddStream
	st := &broadcastAddStreamState[T]{addStreamFuture: newFuture[Void]()}
	c.addState = st
	onError := c.sendError
	if cancelOnError {
		onError = func(err error) {
			c.sendError(err)
			c.closeAddStream()
		}
	}
	st.addSubscription = source.Listen(c.sendData, StreamListenOptions{
		OnError:       onError,
		OnDone:        c.closeAddStream,
		CancelOnError: cancelOnError,
	})
	return st.addStreamFuture
}

type broadcastAddStreamState[T any] struct {
	addStreamFuture *future[Void]
	addSubscription StreamSubscription[T]
}

func (c *BroadcastStreamController[T]) closeAddStream() {
	st := c.addState
	c.addState = nil
	c.state &^= bcAddStream
	st.addStreamFuture.asyncCompleteWithValue(Void{})
}

// forEachListener is _forEachListener: fire an event to the listeners
// subscribed before it (a listener added while firing expects the next
// event), deferring removals and the onCancel check until the end.
func (c *BroadcastStreamController[T]) forEachListener(action func(*bufferingSubscription[T])) {
	if c.isFiring() {
		panic(stateError("Cannot fire new event. Controller is already firing an event"))
	}
	if c.isEmpty() {
		return
	}
	id := c.state & bcEventID
	c.state ^= bcEventID | bcFiring
	s := c.first
	for s != nil {
		if s.expectsEvent(id) {
			s.eventState |= bsubFiring
			action(s.bufferingSubscription)
			s.eventState ^= bsubEventID
			next := s.next
			if s.eventState&bsubRemoveAfterFiring != 0 {
				c.removeListener(s)
			}
			s.eventState &^= bsubFiring
			s = next
		} else {
			s = s.next
		}
	}
	c.state &^= bcFiring
	if c.isEmpty() {
		c.callOnCancel()
	}
}

func (c *BroadcastStreamController[T]) callOnCancel() {
	if c.IsClosed() {
		// When closed, doneFuture is not nil.
		if d := c.doneFuture; d.state == incomplete {
			d.asyncCompleteWithValue(Void{})
		}
	}
	runGuarded(c.OnCancel)
}

func (c *BroadcastStreamController[T]) sendData(data T) {
	if c.sync {
		if c.isEmpty() {
			return
		}
		if c.hasOneListener() {
			c.state |= bcFiring
			c.first.add(data)
			c.state &^= bcFiring
			if c.isEmpty() {
				c.callOnCancel()
			}
			return
		}
		c.forEachListener(func(s *bufferingSubscription[T]) { s.add(data) })
		return
	}
	for s := c.first; s != nil; s = s.next {
		s.addPending(&delayedEvent[T]{kind: 0, value: data})
	}
}

func (c *BroadcastStreamController[T]) sendError(err error) {
	if c.sync {
		if c.isEmpty() {
			return
		}
		c.forEachListener(func(s *bufferingSubscription[T]) { s.addError(err) })
		return
	}
	for s := c.first; s != nil; s = s.next {
		s.addPending(&delayedEvent[T]{kind: 1, err: err})
	}
}

func (c *BroadcastStreamController[T]) sendDone() {
	if c.isEmpty() {
		c.doneFuture.asyncCompleteWithValue(Void{})
		return
	}
	if c.sync {
		c.forEachListener(func(s *bufferingSubscription[T]) { s.close() })
		return
	}
	for s := c.first; s != nil; s = s.next {
		s.addPending(&delayedEvent[T]{kind: 2})
	}
}

// broadcastSubscription is _BroadcastSubscription: a controller
// subscription in the controller's linked list, with its event id and
// firing state.
type broadcastSubscription[T any] struct {
	*bufferingSubscription[T]
	controller *BroadcastStreamController[T]
	eventState int
	next       *broadcastSubscription[T]
	previous   *broadcastSubscription[T]
}

func (s *broadcastSubscription[T]) expectsEvent(id int) bool { return s.eventState&bsubEventID == id }
func (s *broadcastSubscription[T]) isFiring() bool           { return s.eventState&bsubFiring != 0 }

// A broadcast controller doesn't record pauses or resumes.
func (s *broadcastSubscription[T]) onPause()               {}
func (s *broadcastSubscription[T]) onResume()              {}
func (s *broadcastSubscription[T]) onCancel() Future[Void] { return s.controller.recordCancel(s) }
