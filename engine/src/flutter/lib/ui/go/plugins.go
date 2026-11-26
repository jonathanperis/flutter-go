package ui

// Port of engine/src/flutter/lib/ui/plugins.dart (Flutter 3.50.1):
// CallbackHandle and PluginUtilities, with natives.dart's
// _getCallbackHandle/_getCallbackFromHandle and the engine's
// DartCallbackCache (lib/ui/plugins/callback_cache.cc).
//
// Dart looks a callback up by its name, class and library, so a handle
// works in another isolate (or a later run) of the same program. In Go the
// handle of a package-level function is the hash of its qualified name
// (as the engine hashes name, class and library); ENGINE GAP: the cache is
// not saved to disk, so GetCallbackFromHandle only finds callbacks whose
// handle was taken in this process.

import (
	"hash/fnv"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

// CallbackHandle is an opaque handle to a callback (a top-level or static
// function) that can be passed between isolates and back to the function.
// It is a comparable value (Dart's == compares the raw handles).
type CallbackHandle struct{ handle int }

// NewCallbackHandleFromRawHandle is CallbackHandle.fromRawHandle.
func NewCallbackHandleFromRawHandle(handle int) CallbackHandle { return CallbackHandle{handle} }

// ToRawHandle is the handle's integer value.
func (h CallbackHandle) ToRawHandle() int { return h.handle }

// Equal is ==.
func (h CallbackHandle) Equal(other CallbackHandle) bool { return h == other }

func (h CallbackHandle) String() string { return "Instance of 'CallbackHandle'" }

// pluginUtilities is PluginUtilities, a class of static members.
type pluginUtilities struct {
	mu            sync.Mutex
	forwardCache  map[string]*CallbackHandle
	backwardCache map[CallbackHandle]any
}

// PluginUtilities holds functionality for plugins (PluginUtilities'
// statics: PluginUtilities.GetCallbackHandle(f)).
var PluginUtilities = &pluginUtilities{forwardCache: map[string]*CallbackHandle{},
	backwardCache: map[CallbackHandle]any{}}

// GetCallbackHandle returns a handle for callback, a package-level
// function (Dart: a top-level or static function); nil for anything else
// (closures, method values, non-functions), as Dart returns null for
// closures and instance methods.
func (u *pluginUtilities) GetCallbackHandle(callback any) *CallbackHandle {
	name, ok := callbackName(callback)
	if !ok {
		return nil
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if h, ok := u.forwardCache[name]; ok {
		return h
	}
	h := &CallbackHandle{callbackCache.getCallbackHandle(name, callback)}
	u.forwardCache[name] = h
	return h
}

// GetCallbackFromHandle returns the function handle stands for, or nil.
func (u *pluginUtilities) GetCallbackFromHandle(handle CallbackHandle) any {
	u.mu.Lock()
	defer u.mu.Unlock()
	if f, ok := u.backwardCache[handle]; ok {
		return f
	}
	f := callbackCache.getCallbackFromHandle(handle.handle)
	u.backwardCache[handle] = f
	return f
}

// callbackName is DartRuntimeHooks::GetCallbackHandle's check: the
// qualified name of a package-level function, or false for a closure, a
// method value or a non-function.
func callbackName(callback any) (string, bool) {
	v := reflect.ValueOf(callback)
	if !v.IsValid() || v.Kind() != reflect.Func || v.IsNil() {
		return "", false
	}
	fn := runtime.FuncForPC(v.Pointer())
	if fn == nil {
		return "", false
	}
	name := fn.Name()
	last := name[strings.LastIndex(name, "/")+1:]
	// pkg.f is top-level; pkg.f.func1 is a closure, pkg.(*T).M-fm a method
	// value, pkg.T.M or pkg.(*T).M a method expression.
	if strings.Count(last, ".") != 1 || strings.ContainsAny(last, "()-") {
		return "", false
	}
	return name, true
}

// dartCallbackCache is the engine's DartCallbackCache (in memory).
type dartCallbackCache struct {
	mu    sync.Mutex
	cache map[int]any
}

var callbackCache = &dartCallbackCache{cache: map[int]any{}}

func (c *dartCallbackCache) getCallbackHandle(name string, callback any) int {
	h := fnv.New64a()
	h.Write([]byte(name))
	handle := int(h.Sum64() >> 1)
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.cache[handle]; !ok {
		c.cache[handle] = callback
	}
	return handle
}

func (c *dartCallbackCache) getCallbackFromHandle(handle int) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cache[handle]
}
