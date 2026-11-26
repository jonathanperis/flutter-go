package ui

// Tests of platform_message.go: sending platform messages (with and without
// an embedder handling them), the engine's dispatch of incoming messages
// and their responses, the initial lifecycle state, and the enums.

import (
	"reflect"
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
)

type messageClient struct {
	noopClient
	messages []*PlatformMessage
}

func (c *messageClient) HandlePlatformMessage(message *PlatformMessage) {
	c.messages = append(c.messages, message)
}

// noopClient implements PlatformConfigurationClient's natives as no-ops.
type noopClient struct{}

func (noopClient) ScheduleFrame()             {}
func (noopClient) EndWarmUpFrame()            {}
func (noopClient) SetNeedsReportTimings(bool) {}

func withLoop(t *testing.T) *async.Loop {
	loop := async.NewLoop()
	t.Cleanup(async.SetCurrent(loop))
	return loop
}

func TestSendPlatformMessageWithoutHandlerRepliesEmptyAsAnEvent(t *testing.T) {
	loop := withLoop(t)
	for _, pc := range []*PlatformConfiguration{nil, NewPlatformConfiguration(noopClient{})} {
		restore := SetPlatformConfiguration(pc)
		var replies [][]byte
		called := 0
		PlatformDispatcherInstance().SendPlatformMessage("ch", []byte{1}, func(data []byte) {
			replies = append(replies, data)
			called++
		})
		PlatformDispatcherInstance().SendPlatformMessage("ch", nil, nil) // no callback: nothing to do
		loop.FlushMicrotasks()
		if called != 0 {
			t.Fatal("the reply must arrive as an event, not synchronously or in a microtask")
		}
		if n := loop.RunPosted(); n != 1 {
			t.Fatalf("%d events posted, want 1", n)
		}
		if called != 1 || replies[0] != nil {
			t.Fatalf("replies %v", replies)
		}
		restore()
	}
}

func TestSendPlatformMessageToTheEmbedder(t *testing.T) {
	loop := withLoop(t)
	client := &messageClient{}
	defer SetPlatformConfiguration(NewPlatformConfiguration(client))()
	var log []string
	PlatformDispatcherInstance().SendPlatformMessage("a", []byte{1, 2}, func(data []byte) {
		log = append(log, "a "+string(rune('0'+len(data))))
		if data == nil {
			log = append(log, "a nil")
		}
	})
	PlatformDispatcherInstance().SendPlatformMessage("b", nil, func(data []byte) {
		if data == nil {
			log = append(log, "b nil")
		} else {
			log = append(log, "b empty")
		}
	})
	PlatformDispatcherInstance().SendPlatformMessage("c", []byte{}, nil)
	if len(client.messages) != 3 {
		t.Fatalf("%d messages", len(client.messages))
	}
	a, b, c := client.messages[0], client.messages[1], client.messages[2]
	if a.Channel != "a" || !reflect.DeepEqual(a.Data, []byte{1, 2}) || a.Response == nil {
		t.Fatalf("a: %+v", a)
	}
	if b.Channel != "b" || b.Data != nil || b.Response == nil {
		t.Fatalf("b: %+v", b)
	}
	if c.Response != nil || c.Data == nil {
		t.Fatalf("c: %+v", c)
	}
	// Replies, in any order, from any goroutine; each response completes
	// once.
	done := make(chan struct{})
	go func() {
		b.Response.Complete(nil) // an empty ByteData, not null
		a.Response.CompleteEmpty()
		close(done)
	}()
	<-done
	a.Response.Complete([]byte{9}) // ignored: already completed
	loop.RunPosted()
	if want := []string{"b empty", "a 0", "a nil"}; !reflect.DeepEqual(log, want) {
		t.Fatalf("log %v, want %v", log, want)
	}
}

type recordingResponse struct{ log *[]string }

func (r recordingResponse) Complete(data []byte) { *r.log = append(*r.log, "complete "+string(data)) }
func (r recordingResponse) CompleteEmpty()       { *r.log = append(*r.log, "empty") }

func TestDispatchPlatformMessage(t *testing.T) {
	loop := withLoop(t)
	pc := NewPlatformConfiguration(noopClient{})
	defer SetPlatformConfiguration(pc)()
	var log []string
	pd := PlatformDispatcherInstance()

	// To the channel buffers: the listener's reply completes the response.
	ChannelBuffersInstance().SetListener("dispatch", func(data []byte, callback PlatformMessageResponseCallback) {
		log = append(log, "listener "+string(data))
		callback([]byte("reply"))
	})
	defer ChannelBuffersInstance().ClearListener("dispatch")
	loop.FlushMicrotasks()
	pc.DispatchPlatformMessage(&PlatformMessage{Channel: "dispatch", Data: []byte("hi"), Response: recordingResponse{&log}})
	// No response expected: the reply goes nowhere.
	pc.DispatchPlatformMessage(&PlatformMessage{Channel: "dispatch", Data: []byte("x")})
	// A null reply completes empty; the response data is copied.
	ChannelBuffersInstance().SetListener("dispatch2", func(data []byte, callback PlatformMessageResponseCallback) {
		callback(nil)
	})
	defer ChannelBuffersInstance().ClearListener("dispatch2")
	loop.FlushMicrotasks()
	pc.DispatchPlatformMessage(&PlatformMessage{Channel: "dispatch2", Response: recordingResponse{&log}})
	if want := []string{"listener hi", "complete reply", "listener x", "empty"}; !reflect.DeepEqual(log, want) {
		t.Fatalf("log %v, want %v", log, want)
	}

	// onPlatformMessage takes precedence over the channel buffers.
	log = nil
	pd.SetOnPlatformMessage(func(name string, data []byte, callback PlatformMessageResponseCallback) {
		log = append(log, "onPlatformMessage "+name)
		callback([]byte("r2"))
		callback([]byte("r3")) // the response is gone: ignored
	})
	if pd.OnPlatformMessage() == nil {
		t.Fatal("OnPlatformMessage")
	}
	pc.DispatchPlatformMessage(&PlatformMessage{Channel: "dispatch", Response: recordingResponse{&log}})
	pd.SetOnPlatformMessage(nil)
	if want := []string{"onPlatformMessage dispatch", "complete r2"}; !reflect.DeepEqual(log, want) {
		t.Fatalf("log %v, want %v", log, want)
	}

	// The control channel goes to ChannelBuffers.HandleMessage, and is
	// answered with null even when it throws.
	log = nil
	pc.DispatchPlatformMessage(&PlatformMessage{Channel: ChannelBuffersKControlChannelName,
		Data: []byte("resize\rdispatch3\r3"), Response: recordingResponse{&log}})
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a bad control message must throw")
			}
		}()
		pc.DispatchPlatformMessage(&PlatformMessage{Channel: ChannelBuffersKControlChannelName,
			Data: []byte("bogus"), Response: recordingResponse{&log}})
	}()
	if want := []string{"empty", "empty"}; !reflect.DeepEqual(log, want) {
		t.Fatalf("log %v, want %v", log, want)
	}
	// Response ids are unique and freed.
	if len(pc.pendingResponses) != 0 {
		t.Fatalf("pending responses %v", pc.pendingResponses)
	}
}

func TestRespondWithoutConfiguration(t *testing.T) {
	// No configuration installed: responding does nothing.
	respondToPlatformMessage(1, []byte{1})
	respondToPlatformMessage(1, nil)
}

func TestInitialLifecycleState(t *testing.T) {
	pd := &PlatformDispatcher{}
	pd.updateInitialLifecycleState("AppLifecycleState.paused")
	pd.updateInitialLifecycleState("AppLifecycleState.resumed")
	if got := pd.InitialLifecycleState(); got != "AppLifecycleState.resumed" {
		t.Fatalf("got %q", got)
	}
	pd.updateInitialLifecycleState("AppLifecycleState.hidden")
	if got := pd.InitialLifecycleState(); got != "AppLifecycleState.resumed" {
		t.Fatalf("got %q after being read", got)
	}
	if (&PlatformDispatcher{}).InitialLifecycleState() != "" {
		t.Fatal("default")
	}
}

func TestPlatformMessageEnums(t *testing.T) {
	if RootIsolateTokenInstance() == nil || RootIsolateTokenInstance() != RootIsolateTokenInstance() {
		t.Fatal("root isolate token")
	}
	if AppExitResponseExit.String() != "AppExitResponse.exit" || AppExitResponseCancel.Name() != "cancel" ||
		len(AppExitResponseValues) != 2 {
		t.Fatal("AppExitResponse")
	}
	if AppExitTypeCancelable.String() != "AppExitType.cancelable" || AppExitTypeRequired.Name() != "required" ||
		len(AppExitTypeValues) != 2 {
		t.Fatal("AppExitType")
	}
}
