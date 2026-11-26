package ui

// Ports of engine/src/flutter/testing/dart/plugin_utilities_test.dart and
// isolate_name_server_test.dart (Flutter 3.50.1). Dart's isolates are
// goroutines here (isolate.ReceivePort delivers to the creating loop).
//
// Adapted: Dart's static method Foo.getInt has no Go equivalent (a Go
// method expression takes its receiver), so a second package-level
// function stands for it; Dart's instance method tear-off foo.getDouble is
// a Go method value.

import (
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/isolate"
)

func top() string { return "top" }

func getInt() int { return 1 }

type foo struct{}

func (foo) getDouble() float64 { return 1.0 }

func TestPluginUtilitiesCallbackHandles(t *testing.T) {
	// Top level callback.
	hTop := PluginUtilities.GetCallbackHandle(top)
	if hTop == nil || hTop.ToRawHandle() == 0 {
		t.Fatal("top")
	}
	if *PluginUtilities.GetCallbackHandle(top) != *hTop {
		t.Fatal("handle not stable")
	}
	if PluginUtilities.GetCallbackFromHandle(*hTop).(func() string)() != "top" {
		t.Fatal("top closure")
	}
	// "Static" callback.
	hGetInt := PluginUtilities.GetCallbackHandle(getInt)
	if hGetInt == nil || hGetInt.ToRawHandle() == 0 || *hGetInt == *hTop {
		t.Fatal("getInt")
	}
	if *PluginUtilities.GetCallbackHandle(getInt) != *hGetInt ||
		PluginUtilities.GetCallbackFromHandle(*hGetInt).(func() int)() != 1 {
		t.Fatal("getInt closure")
	}
	// Instance method callbacks cannot be looked up.
	if PluginUtilities.GetCallbackHandle(foo{}.getDouble) != nil {
		t.Fatal("method value")
	}
	if PluginUtilities.GetCallbackHandle(foo.getDouble) != nil {
		t.Fatal("method expression")
	}
	// Anonymous closures cannot be looked up.
	anon := func(a, b int) int { return a + b }
	if PluginUtilities.GetCallbackHandle(anon) != nil {
		t.Fatal("closure")
	}
	if PluginUtilities.GetCallbackHandle(42) != nil || PluginUtilities.GetCallbackHandle(nil) != nil {
		t.Fatal("non-functions")
	}
	if PluginUtilities.GetCallbackFromHandle(NewCallbackHandleFromRawHandle(12345)) != nil {
		t.Fatal("unknown handle")
	}
	if !NewCallbackHandleFromRawHandle(5).Equal(NewCallbackHandleFromRawHandle(5)) ||
		NewCallbackHandleFromRawHandle(5) == NewCallbackHandleFromRawHandle(6) {
		t.Fatal("==")
	}
}

func TestSimpleIsolateNameServer(t *testing.T) {
	const portName = "foobar1"
	defer IsolateNameServer.RemovePortNameMapping(portName)
	if IsolateNameServer.LookupPortByName(portName) != nil || IsolateNameServer.RemovePortNameMapping(portName) {
		t.Fatal("unset")
	}
	receivePort := isolate.NewReceivePort()
	sendPort := receivePort.SendPort()
	if !IsolateNameServer.RegisterPortWithName(sendPort, portName) || IsolateNameServer.LookupPortByName(portName) != sendPort {
		t.Fatal("register")
	}
	receivePort2 := isolate.NewReceivePort()
	sendPort2 := receivePort2.SendPort()
	if IsolateNameServer.RegisterPortWithName(sendPort2, portName) || IsolateNameServer.LookupPortByName(portName) != sendPort {
		t.Fatal("double register")
	}
	if !IsolateNameServer.RemovePortNameMapping(portName) || IsolateNameServer.LookupPortByName(portName) != nil {
		t.Fatal("remove")
	}
	if !IsolateNameServer.RegisterPortWithName(sendPort2, portName) || IsolateNameServer.LookupPortByName(portName) != sendPort2 {
		t.Fatal("re-register")
	}
	receivePort.Close()
	receivePort2.Close()
	expectPanic(t, "port must not be null", func() { IsolateNameServer.RegisterPortWithName(nil, "x") })
}

const (
	kErrorCode   = -1
	kStartCode   = 0
	kCloseCode   = 1
	kDeletedCode = 2
)

// isolateSpawnEntrypoint is the spawned isolate's entry point, on a
// goroutine.
func isolateSpawnEntrypoint(port isolate.SendPort, portName string) {
	sendHelper := func(code int, message string) { port.Send([]any{code, message}) }
	shared := IsolateNameServer.LookupPortByName(portName)
	if shared == nil {
		sendHelper(kErrorCode, "Could not find port: "+portName)
		return
	}
	sendHelper(kStartCode, "")
	shared.Send(portName)
	sendHelper(kCloseCode, "")
	// The ReceivePort's callback must not run more than once.
	shared.Send("garbage")
	if IsolateNameServer.RemovePortNameMapping(portName) {
		sendHelper(kDeletedCode, "")
	} else {
		sendHelper(kErrorCode, "Was unable to remove mapping for "+portName)
	}
}

func TestIsolateNameServerMultiIsolate(t *testing.T) {
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	const portName = "foobar2"
	defer IsolateNameServer.RemovePortNameMapping(portName)
	receivePort := isolate.NewReceivePort()
	if !IsolateNameServer.RegisterPortWithName(receivePort.SendPort(), portName) {
		t.Fatal("register")
	}
	testReceivePort := isolate.NewReceivePort()
	testDone, portDone := false, false
	responses, messages := 0, 0
	testReceivePort.Listen(func(response any) {
		responses++
		r := response.([]any)
		switch r[0].(int) {
		case kStartCode:
		case kCloseCode:
			receivePort.Close()
		case kDeletedCode:
			if IsolateNameServer.LookupPortByName(portName) != nil {
				t.Error("mapping still present")
			}
			testReceivePort.Close()
		case kErrorCode:
			t.Error(r[1])
		}
	}, func() { testDone = true })
	receivePort.Listen(func(message any) {
		messages++
		if message != portName {
			t.Errorf("message %v", message)
		}
	}, func() { portDone = true })
	go isolateSpawnEntrypoint(testReceivePort.SendPort(), portName)
	for !testDone || !portDone {
		loop.WaitPosted()
		loop.RunPosted()
	}
	if responses != 3 || messages != 1 {
		t.Fatalf("responses %d, messages %d", responses, messages)
	}
}
