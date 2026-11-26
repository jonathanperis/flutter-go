package ui

// Port of engine/src/flutter/lib/ui/channel_buffers.dart (Flutter 3.50.1):
// the buffers that hold messages from the platform until the framework
// registers a listener for their channel.
//
// Dart records the zone each listener and each response callback was
// registered in and invokes them there; Go has no zones, so they are called
// directly (see API_MAPPING "Bindings, zones and statics").

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"unicode/utf16"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// DrainChannelCallback is the signature of the callback of the deprecated
// ChannelBuffers.Drain (migrate to ChannelCallback and SetListener). A nil
// future is treated as an already completed one.
type DrainChannelCallback func(data []byte, callback PlatformMessageResponseCallback) async.Future[async.Void]

// ChannelCallback is the signature for ChannelBuffers.SetListener's
// callback: data is the message's payload and callback must be called (with
// the response, or nil) for every message.
type ChannelCallback func(data []byte, callback PlatformMessageResponseCallback)

// channelCallbackRecord is _ChannelCallbackRecord: a listener (Dart also
// records its zone).
type channelCallbackRecord struct{ callback ChannelCallback }

func (r *channelCallbackRecord) invoke(data []byte, callback PlatformMessageResponseCallback) {
	if r.callback != nil {
		r.callback(data, callback)
	}
}

// storedMessage is _StoredMessage: a message and its response callback.
type storedMessage struct {
	data     []byte
	callback PlatformMessageResponseCallback
}

func (m *storedMessage) invoke(data []byte) {
	if m.callback != nil {
		m.callback(data)
	}
}

// channelBuffer is _Channel: the buffer of one channel, with its listener.
type channelBuffer struct {
	queue                      []*storedMessage // a FIFO (Dart's ListQueue)
	capacity                   int
	debugEnableDiscardWarnings bool
	draining                   bool
	channelCallbackRecord      *channelCallbackRecord
}

func newChannel(capacity ...int) *channelBuffer {
	c := &channelBuffer{capacity: ChannelBuffersKDefaultBufferSize, debugEnableDiscardWarnings: true}
	if len(capacity) > 0 {
		c.capacity = capacity[0]
	}
	// Dart's queue is a ListQueue(capacity), whose table is allocated at
	// once: its size is capacity rounded up to a power of two (at least 8),
	// and the rounding overflows to a negative size, which List.filled
	// rejects, for capacities above 2^62. (Go's queue grows as needed, so
	// large capacities don't allocate.)
	if table := listQueueCapacity(c.capacity); table < 0 {
		panic(dart.NewRangeErrorRange(table, 0, maxInt, "length", ""))
	}
	return c
}

const maxInt = int(^uint(0) >> 1)

// listQueueCapacity is ListQueue._calculateCapacity.
func listQueueCapacity(initialCapacity int) int {
	const initial = 8 // ListQueue._INITIAL_CAPACITY
	if initialCapacity < initial {
		return initial
	}
	if initialCapacity&(initialCapacity-1) == 0 {
		return initialCapacity
	}
	number := initialCapacity<<1 - 1 // ListQueue._nextPowerOf2
	for {
		next := number & (number - 1)
		if next == 0 {
			return number
		}
		number = next
	}
}

func (c *channelBuffer) setCapacity(newSize int) {
	c.capacity = newSize
	c.dropOverflowMessages(newSize)
}

// push adds a message to the buffer, or gives it to the listener directly
// when there is one and the buffer isn't draining. It returns whether a
// message was discarded (and a warning is wanted).
func (c *channelBuffer) push(message *storedMessage) bool {
	if !c.draining && c.channelCallbackRecord != nil {
		if len(c.queue) != 0 {
			panic(dart.NewAssertionError(nil))
		}
		c.channelCallbackRecord.invoke(message.data, message.invoke)
		return false
	}
	if c.capacity <= 0 {
		return c.debugEnableDiscardWarnings
	}
	result := c.dropOverflowMessages(c.capacity - 1)
	c.queue = append(c.queue, message)
	return result
}

func (c *channelBuffer) pop() *storedMessage {
	if len(c.queue) == 0 {
		panic(dart.StateError{Message: "No element"})
	}
	m := c.queue[0]
	c.queue[0] = nil
	c.queue = c.queue[1:]
	return m
}

// dropOverflowMessages discards the oldest messages until at most
// lengthLimit remain, answering each with an empty reply.
func (c *channelBuffer) dropOverflowMessages(lengthLimit int) bool {
	result := false
	for len(c.queue) > lengthLimit {
		message := c.pop()
		message.invoke(nil) // send empty reply to the plugin side
		result = true
	}
	return result
}

func (c *channelBuffer) setListener(callback ChannelCallback) {
	needDrain := c.channelCallbackRecord == nil
	c.channelCallbackRecord = &channelCallbackRecord{callback}
	if needDrain && !c.draining {
		c.drain()
	}
}

func (c *channelBuffer) clearListener() { c.channelCallbackRecord = nil }

// drain delivers the buffered messages to the listener, one per microtask.
func (c *channelBuffer) drain() {
	if c.draining {
		panic(dart.NewAssertionError(nil))
	}
	c.draining = true
	async.ScheduleMicrotask(c.drainStep)
}

func (c *channelBuffer) drainStep() {
	if !c.draining {
		panic(dart.NewAssertionError(nil))
	}
	if len(c.queue) != 0 && c.channelCallbackRecord != nil {
		message := c.pop()
		c.channelCallbackRecord.invoke(message.data, message.invoke)
		async.ScheduleMicrotask(c.drainStep)
	} else {
		c.draining = false
	}
}

// ChannelBuffersOverridable are the ChannelBuffers methods its own methods
// call through self, so that a type embedding ChannelBuffers can override
// them (Dart subclasses override resize and allowOverflow, which
// HandleMessage calls, and sendChannelUpdate).
type ChannelBuffersOverridable interface {
	Resize(name string, newSize int)
	AllowOverflow(name string, allowed bool)
	SendChannelUpdate(name string, listening bool)
}

// ChannelBuffers is the storage for messages from the platform to the
// framework, held until a listener is registered for their channel. Each
// channel has a fixed-size buffer (ChannelBuffersKDefaultBufferSize, one
// message, unless resized); when it is full, the oldest message is
// discarded and answered with a nil reply.
//
// A type embedding ChannelBuffers calls SetSelf(outer) so that its
// overrides of the ChannelBuffersOverridable methods are used.
type ChannelBuffers struct {
	channels map[string]*channelBuffer
	self     ChannelBuffersOverridable
}

// ChannelBuffersKDefaultBufferSize is ChannelBuffers.kDefaultBufferSize, the
// capacity of a channel's buffer unless it is resized.
const ChannelBuffersKDefaultBufferSize = 1

// ChannelBuffersKControlChannelName is ChannelBuffers.kControlChannelName,
// the channel the platform uses to configure the buffers (HandleMessage).
const ChannelBuffersKControlChannelName = "dev.flutter/channel-buffers"

// NewChannelBuffers is ChannelBuffers(), a new set of buffers. (Messages from
// the platform go to the singleton, ChannelBuffersInstance.)
func NewChannelBuffers() *ChannelBuffers {
	b := &ChannelBuffers{}
	b.self = b
	return b
}

// SetSelf makes b call self's overrides of the ChannelBuffersOverridable
// methods.
func (b *ChannelBuffers) SetSelf(self ChannelBuffersOverridable) { b.self = self }

func (b *ChannelBuffers) overridable() ChannelBuffersOverridable {
	if b.self == nil {
		return b
	}
	return b.self
}

func (b *ChannelBuffers) putIfAbsent(name string) *channelBuffer {
	if b.channels == nil {
		b.channels = map[string]*channelBuffer{}
	}
	c := b.channels[name]
	if c == nil {
		c = newChannel()
		b.channels[name] = c
	}
	return c
}

const nullCharacterAssertion = "Channel names must not contain U+0000 NULL characters."

// Push adds a message (data) on the named channel, with the callback to
// call with the response. If the channel has a listener, it receives the
// message at once; otherwise the message is buffered, and if the buffer
// overflows the oldest message is discarded (its callback gets nil) and a
// warning is printed in debug builds.
func (b *ChannelBuffers) Push(name string, data []byte, callback PlatformMessageResponseCallback) {
	if strings.ContainsRune(name, 0) {
		panic(dart.NewAssertionError(nullCharacterAssertion))
	}
	channel := b.putIfAbsent(name)
	if channel.push(&storedMessage{data, callback}) {
		s := "s"
		if channel.capacity == 1 {
			s = ""
		}
		printDebug("A message on the " + name + " channel was discarded before it could be handled.\n" +
			"This happens when a plugin sends messages to the framework side before the " +
			"framework has had an opportunity to register a listener. See the ChannelBuffers " +
			"API documentation for details on how to configure the channel to expect more " +
			"messages, or to expect messages to get discarded:\n" +
			"  https://api.flutter.dev/flutter/dart-ui/ChannelBuffers-class.html\n" +
			"The capacity of the " + name + " channel is " + itoa(channel.capacity) + " message" + s + ".")
	}
}

// SetListener makes callback the listener of the named channel, replacing
// any previous one. Buffered messages are delivered to it one per
// microtask, before newer messages.
func (b *ChannelBuffers) SetListener(name string, callback ChannelCallback) {
	if strings.ContainsRune(name, 0) {
		panic(dart.NewAssertionError(nullCharacterAssertion))
	}
	channel := b.putIfAbsent(name)
	channel.setListener(callback)
	b.overridable().SendChannelUpdate(name, true)
}

// ClearListener removes the named channel's listener: its messages are
// buffered again.
func (b *ChannelBuffers) ClearListener(name string) {
	if channel := b.channels[name]; channel != nil {
		channel.clearListener()
		b.overridable().SendChannelUpdate(name, false)
	}
}

// ChannelUpdateClient is implemented by a PlatformConfigurationClient that
// wants to know when channels gain or lose a listener
// (PlatformConfigurationClient::SendChannelUpdate in the engine).
type ChannelUpdateClient interface {
	SendChannelUpdate(name string, listening bool)
}

// SendChannelUpdate tells the engine whether the named channel has a
// listener (Dart's required named listening: is positional).
func (b *ChannelBuffers) SendChannelUpdate(name string, listening bool) {
	sendChannelUpdate(name, listening)
}

// sendChannelUpdate is the native
// PlatformConfigurationNativeApi::SendChannelUpdate.
func sendChannelUpdate(name string, listening bool) {
	if pc := currentPlatformConfiguration; pc != nil {
		if c, ok := pc.client.(ChannelUpdateClient); ok {
			c.SendChannelUpdate(name, listening)
		}
	}
}

// Drain removes the named channel's buffered messages and passes them to
// callback one at a time, waiting for each returned future (deprecated in
// Flutter: use SetListener).
func (b *ChannelBuffers) Drain(name string, callback DrainChannelCallback) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		channel := b.channels[name]
		for channel != nil && len(channel.queue) != 0 {
			message := channel.pop()
			f := callback(message.data, message.invoke)
			if f == nil {
				f = async.FutureValue(async.Void{})
			}
			f.Await()
		}
	})
}

// HandleMessage handles a message on the control channel
// (ChannelBuffersKControlChannelName): a "resize" or "overflow" method call
// in the standard method codec's encoding, or the legacy
// "resize\r<channel>\r<size>" UTF-8 string. Invalid messages panic with a
// dart.Exception (or the FormatException of a malformed UTF-8 or integer
// text).
func (b *ChannelBuffers) HandleMessage(data []byte) {
	// We hard-code the deserialization here because the StandardMethodCodec
	// class is part of the framework, not dart:ui.
	bytes := data
	if bytes[0] == 0x07 { // 7 = value code for string
		methodNameLength := int(bytes[1])
		if methodNameLength >= 254 { // lengths greater than 253 have more elaborate encoding
			panic(dart.NewException("Unrecognized message sent to " + ChannelBuffersKControlChannelName + " (method name too long)"))
		}
		index := 2 // where we are in reading the bytes
		methodName := dart.UTF8Decode(bytes[index : index+methodNameLength])
		index += methodNameLength
		switch methodName {
		case "resize":
			if bytes[index] != 0x0C { // 12 = value code for list
				panic(dart.NewException("Invalid arguments for 'resize' method sent to " + ChannelBuffersKControlChannelName + " (arguments must be a two-element list, channel name and new capacity)"))
			}
			index++
			if bytes[index] < 0x02 { // We ignore extra arguments, in case we need to support them in the future, hence <2 rather than !=2.
				panic(dart.NewException("Invalid arguments for 'resize' method sent to " + ChannelBuffersKControlChannelName + " (arguments must be a two-element list, channel name and new capacity)"))
			}
			index++
			if bytes[index] != 0x07 { // 7 = value code for string
				panic(dart.NewException("Invalid arguments for 'resize' method sent to " + ChannelBuffersKControlChannelName + " (first argument must be a string)"))
			}
			index++
			channelNameLength := int(bytes[index])
			if channelNameLength >= 254 { // lengths greater than 253 have more elaborate encoding
				panic(dart.NewException("Invalid arguments for 'resize' method sent to " + ChannelBuffersKControlChannelName + " (channel name must be less than 254 characters long)"))
			}
			index++
			channelName := dart.UTF8Decode(bytes[index : index+channelNameLength])
			if strings.ContainsRune(channelName, 0) {
				panic(dart.NewException("Invalid arguments for 'resize' method sent to " + ChannelBuffersKControlChannelName + " (channel name must not contain any null bytes)"))
			}
			index += channelNameLength
			if bytes[index] != 0x03 { // 3 = value code for uint32
				panic(dart.NewException("Invalid arguments for 'resize' method sent to " + ChannelBuffersKControlChannelName + " (second argument must be an integer in the range 0 to 2147483647)"))
			}
			index++
			b.overridable().Resize(channelName, int(binary.LittleEndian.Uint32(data[index:index+4])))
		case "overflow":
			if bytes[index] != 0x0C { // 12 = value code for list
				panic(dart.NewException("Invalid arguments for 'overflow' method sent to " + ChannelBuffersKControlChannelName + " (arguments must be a two-element list, channel name and flag state)"))
			}
			index++
			if bytes[index] < 0x02 { // We ignore extra arguments, in case we need to support them in the future, hence <2 rather than !=2.
				panic(dart.NewException("Invalid arguments for 'overflow' method sent to " + ChannelBuffersKControlChannelName + " (arguments must be a two-element list, channel name and flag state)"))
			}
			index++
			if bytes[index] != 0x07 { // 7 = value code for string
				panic(dart.NewException("Invalid arguments for 'overflow' method sent to " + ChannelBuffersKControlChannelName + " (first argument must be a string)"))
			}
			index++
			channelNameLength := int(bytes[index])
			if channelNameLength >= 254 { // lengths greater than 253 have more elaborate encoding
				panic(dart.NewException("Invalid arguments for 'overflow' method sent to " + ChannelBuffersKControlChannelName + " (channel name must be less than 254 characters long)"))
			}
			index++
			channelName := dart.UTF8Decode(bytes[index : index+channelNameLength])
			index += channelNameLength
			if bytes[index] != 0x01 && bytes[index] != 0x02 { // 1 = value code for true, 2 = value code for false
				panic(dart.NewException("Invalid arguments for 'overflow' method sent to " + ChannelBuffersKControlChannelName + " (second argument must be a boolean)"))
			}
			b.overridable().AllowOverflow(channelName, bytes[index] == 0x01)
		default:
			panic(dart.NewException("Unrecognized method '" + methodName + "' sent to " + ChannelBuffersKControlChannelName))
		}
	} else {
		parts := strings.Split(dart.UTF8Decode(bytes), "\r")
		if len(parts) == 1+2 /*arity*/ && parts[0] == "resize" {
			b.overridable().Resize(parts[1], dartIntParse(parts[2]))
		} else {
			// If the message couldn't be decoded as UTF-8, a FormatException
			// will have been thrown by dart.UTF8Decode above.
			panic(dart.NewException("Unrecognized message " + dart.ToString(parts) + " sent to " + ChannelBuffersKControlChannelName + "."))
		}
	}
}

// Resize changes the capacity of the named channel's buffer (creating the
// channel if needed), discarding the oldest messages that no longer fit
// (their callbacks get nil).
func (b *ChannelBuffers) Resize(name string, newSize int) {
	channel := b.channels[name]
	if channel == nil {
		if strings.ContainsRune(name, 0) {
			panic(dart.NewAssertionError(nullCharacterAssertion))
		}
		channel = newChannel(newSize)
		if b.channels == nil {
			b.channels = map[string]*channelBuffer{}
		}
		b.channels[name] = channel
	} else {
		channel.setCapacity(newSize)
	}
}

// AllowOverflow toggles whether the named channel warns when it discards
// messages (debug builds only: in release builds this does nothing).
func (b *ChannelBuffers) AllowOverflow(name string, allowed bool) {
	if debugMode {
		channel := b.channels[name]
		if channel == nil && allowed {
			if strings.ContainsRune(name, 0) {
				panic(dart.NewAssertionError(nullCharacterAssertion))
			}
			channel = newChannel()
			if b.channels == nil {
				b.channels = map[string]*channelBuffer{}
			}
			b.channels[name] = channel
		}
		if channel != nil {
			channel.debugEnableDiscardWarnings = !allowed
		}
	}
}

var channelBuffersInstance = NewChannelBuffers()

// ChannelBuffersInstance is dart:ui's top-level channelBuffers: the buffers
// that messages from the platform go to (a function because the type has
// the variable's Go name).
func ChannelBuffersInstance() *ChannelBuffers { return channelBuffersInstance }

// printDebug is natives.dart's _printDebug, the engine's
// DartRuntimeHooks::Logger_PrintDebugString: the message is logged (to
// standard output) in debug builds only.
var printDebug = func(message string) {
	if debugMode {
		fmt.Fprintln(os.Stdout, message)
	}
}

// dartIsWhitespace is the VM's String._isTwoByteWhitespace (what trim and
// int.parse skip).
func dartIsWhitespace(c uint16) bool {
	if c <= 32 {
		return c == 32 || (c <= 13 && c >= 9)
	}
	if c < 0x85 {
		return false
	}
	if c == 0x85 || c == 0xA0 {
		return true
	}
	if c <= 0x200A {
		return c == 0x1680 || 0x2000 <= c
	}
	return c == 0x2028 || c == 0x2029 || c == 0x202F || c == 0x205F || c == 0x3000 || c == 0xFEFF
}

// dartIntParse is the VM's int.parse(source) (no radix): surrounding
// whitespace, a sign and a 0x prefix are accepted; errors panic with
// Dart's FormatException (message, source and offset in UTF-16 code
// units).
func dartIntParse(src string) int {
	source := utf16.Encode([]rune(src))
	fail := func(index int, hasIndex bool, radix int, message string) int {
		if message == "" {
			if radix == 0 {
				message = "Invalid number"
			} else {
				message = "Invalid radix-" + itoa(radix) + " number"
			}
		}
		e := dart.FormatException{Message: message, Source: src, Offset: index, HasOffset: hasIndex}
		panic(e)
	}
	if len(source) == 0 {
		return fail(0, true, 0, "")
	}
	end := len(source) - 1
	for ; end >= 0 && dartIsWhitespace(source[end]); end-- {
	}
	end++
	if end == 0 {
		return fail(len(source), true, 0, "")
	}
	start := 0
	for ; start < len(source) && dartIsWhitespace(source[start]); start++ {
	}
	first := source[start]
	sign := 1
	if first == '+' || first == '-' {
		sign = 0x2c - int(first)
		start++
		if start == end {
			return fail(end, true, 0, "")
		}
		first = source[start]
	}
	if first == '0' {
		start++
		if start == end {
			return 0
		}
		first = source[start]
		if first|0x20 == 'x' {
			start++
			if start == end {
				return fail(start, true, 0, "")
			}
			return parseRadix(source, 16, start, end, sign, sign > 0, fail)
		}
	}
	return parseRadix(source, 10, start, end, sign, false, fail)
}

// parseLimits64 and overflowLimits64 are the VM's tables for radix 10 and 16.
var parseLimits64 = map[int][2]int{10: {18, 1000000000000000000}, 16: {15, 1152921504606846976}}
var overflowLimits64 = map[int][4]int{
	10: {0x9, -0x9, 0x31993af1d7bffff, 0x31993af1d7c0000},
	16: {0x7, -0x8, 0xfffffffffffffff, 0x0},
}

// parseRadix is the VM's int._parseRadix.
func parseRadix(source []uint16, radix, start, end, sign int, allowU64 bool, fail func(int, bool, int, string) int) int {
	for start < end && source[start] == '0' {
		start++
	}
	length := end - start
	if length == 0 {
		return 0
	}
	blockSize, multiplier := parseLimits64[radix][0], parseLimits64[radix][1]
	if length <= blockSize {
		if smi, ok := parseBlock(source, radix, start, end); ok {
			return sign * smi
		}
		return fail(start, true, radix, "")
	}
	result := 0
	smallBlockSize := length % blockSize
	if smallBlockSize > 0 {
		blockEnd := start + smallBlockSize
		smi, ok := parseBlock(source, radix, start, blockEnd)
		if !ok {
			return fail(start, true, radix, "")
		}
		result = sign * smi
		start = blockEnd
	}
	limits := overflowLimits64[radix]
	positiveOverflowLimit, negativeOverflowLimit := limits[0], limits[1]
	blockEnd := start + blockSize
	for {
		smi, ok := parseBlock(source, radix, start, blockEnd)
		if !ok {
			return fail(start, true, radix, "")
		}
		if result >= positiveOverflowLimit {
			// May overflow when adding a smi.
			if result > positiveOverflowLimit || smi > limits[2] {
				if allowU64 && blockEnd == end && result <= 0xF {
					return result*multiplier + smi
				}
				return fail(0, false, radix, "Positive input exceeds the limit of integer")
			}
		} else if result <= negativeOverflowLimit {
			if result < negativeOverflowLimit || smi > limits[3] {
				return fail(0, false, radix, "Negative input exceeds the limit of integer")
			}
		}
		result = result*multiplier + sign*smi
		start = blockEnd
		blockEnd = start + blockSize
		if blockEnd > end {
			return result
		}
	}
}

// parseBlock is the VM's int._parseBlock: the digits in [start, end), or
// false if one isn't a digit of radix.
func parseBlock(source []uint16, radix, start, end int) (int, bool) {
	result := 0
	for i := start; i < end; i++ {
		char := int(source[i])
		digit := char ^ 0x30
		if radix > 10 && digit > 9 {
			digit = (char | 0x20) - (0x61 - 10)
			if digit < 10 {
				return 0, false
			}
		}
		if digit >= radix {
			return 0, false
		}
		result = radix*result + digit
	}
	return result, true
}
