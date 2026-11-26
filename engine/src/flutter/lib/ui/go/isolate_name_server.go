package ui

// Port of engine/src/flutter/lib/ui/isolate_name_server.dart and the
// engine's IsolateNameServer (lib/ui/isolate_name_server/, Flutter
// 3.50.1): a process-wide registry of named send ports, so code running
// elsewhere (Dart: another isolate; Go: another goroutine) can find a port
// by name. Ports are src/isolate's SendPort.

import (
	"sync"

	"github.com/jonathanperis/flutter-go/src/isolate"
)

// isolateNameServer is IsolateNameServer, a class of static members.
type isolateNameServer struct {
	mu          sync.Mutex
	portMapping map[string]isolate.SendPort
}

// IsolateNameServer is the registry of named ports
// (IsolateNameServer.LookupPortByName(name)...). It is safe to use from
// any goroutine.
var IsolateNameServer = &isolateNameServer{portMapping: map[string]isolate.SendPort{}}

// LookupPortByName returns the port registered as name, or nil.
func (s *isolateNameServer) LookupPortByName(name string) isolate.SendPort {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.portMapping[name]
}

// RegisterPortWithName registers port as name; false (and no change) when
// the name is already registered. A nil port panics, as Dart's non-null
// SendPort parameter can't be null.
func (s *isolateNameServer) RegisterPortWithName(port isolate.SendPort, name string) bool {
	if port == nil {
		panic(argumentError("port must not be null"))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.portMapping[name]; ok {
		// Name is already registered.
		return false
	}
	s.portMapping[name] = port
	return true
}

// RemovePortNameMapping removes the port registered as name; false when
// there is none.
func (s *isolateNameServer) RemovePortNameMapping(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.portMapping[name]; !ok {
		return false
	}
	delete(s.portMapping, name)
	return true
}
