// Package isolate is the subset of dart:isolate that Flutter's API
// mentions: SendPort and ReceivePort, the ends of a port that carries
// messages to an event loop (dart:ui's IsolateNameServer registers send
// ports, PlatformDispatcher.sendPortPlatformMessage replies through one).
//
// A Go program is one isolate whose concurrent work runs on goroutines
// (foundation's compute does), so a ReceivePort delivers each message as
// an event of the async.Loop that created it (Loop.Post, the loop's
// goroutine-safe entry), and a SendPort may be used from any goroutine,
// as a Dart SendPort may be sent to and used from any isolate. Messages
// are passed by reference (Go shares memory: nothing is copied or
// "unsendable").
package isolate

import (
	"sync"

	"github.com/jonathanperis/flutter-go/src/async"
)

// SendPort sends messages to a ReceivePort. Two SendPorts are equal (Go's
// ==) when they send to the same port, as Dart's are.
type SendPort interface {
	// Send sends message to the port's ReceivePort; it is dropped when the
	// port is closed.
	Send(message any)
}

// ReceivePort receives the messages sent to its SendPort, as events of
// the async.Loop that created it. Use it by pointer.
type ReceivePort struct {
	mu      sync.Mutex
	loop    *async.Loop
	handler func(message any)
	pending []any
	closed  bool
	onDone  func()
	port    *sendPort
}

type sendPort struct{ r *ReceivePort }

func (s *sendPort) Send(message any) { s.r.post(message) }

// NewReceivePort is ReceivePort(): a port delivering to the current
// async.Loop.
func NewReceivePort() *ReceivePort {
	r := &ReceivePort{loop: async.Current()}
	r.port = &sendPort{r}
	return r
}

// SendPort is the port's send port.
func (r *ReceivePort) SendPort() SendPort { return r.port }

// Listen sets the handler messages are delivered to (a ReceivePort is a
// single-subscription stream: messages sent before Listen are buffered),
// and onDone, called when the port is closed (nil for none).
func (r *ReceivePort) Listen(onData func(message any), onDone ...func()) {
	r.mu.Lock()
	r.handler = onData
	if len(onDone) > 0 {
		r.onDone = onDone[0]
	}
	pending := r.pending
	r.pending = nil
	r.mu.Unlock()
	for _, m := range pending {
		r.post(m)
	}
}

func (r *ReceivePort) post(message any) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	if r.handler == nil {
		r.pending = append(r.pending, message)
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()
	r.loop.Post(func() {
		r.mu.Lock()
		h, closed := r.handler, r.closed
		r.mu.Unlock()
		if !closed && h != nil {
			h(message)
		}
	})
}

// Close closes the port: later messages are dropped, and the onDone given
// to Listen runs (as an event of the port's loop).
func (r *ReceivePort) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	done := r.onDone
	r.mu.Unlock()
	if done != nil {
		r.loop.Post(done)
	}
}
