package foundation

// Port of timeline.dart with _timeline_io.dart (Flutter 3.50.1).

import (
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/developer"
)

// flutterTimeline holds FlutterTimeline's static members.
type flutterTimeline struct{}

// FlutterTimeline measures how long blocks of code take to run. It wraps
// dart:developer's Timeline (whose events, in flutter-go, are dropped: see
// package developer) and, when DebugCollectionEnabled, also records the
// blocks so tests and benchmarks can aggregate them with DebugCollect.
//
// Its generic timeSync is the package-level FlutterTimelineTimeSync.
var FlutterTimeline flutterTimeline

var (
	timelineBuffer            = &blockBuffer{}
	timelineCollectionEnabled = false
)

// DebugCollectionEnabled reports whether blocks are collected.
func (flutterTimeline) DebugCollectionEnabled() bool { return timelineCollectionEnabled }

// SetDebugCollectionEnabled enables or disables collection; changing it
// resets the collected data. It panics with a StateError in release mode.
func (flutterTimeline) SetDebugCollectionEnabled(value bool) {
	if KReleaseMode {
		panic(releaseModeNotSupportedError())
	}
	if value == timelineCollectionEnabled {
		return
	}
	timelineCollectionEnabled = value
	FlutterTimeline.DebugReset()
}

func releaseModeNotSupportedError() dart.StateError {
	return dart.StateError{Message: "FlutterTimeline metric collection not supported in release mode."}
}

// StartSync starts a synchronous block named name (Timeline.startSync).
func (flutterTimeline) StartSync(name string, opts ...developer.TimelineOptions) {
	developer.TimelineStartSync(name, opts...)
	if !KReleaseMode && timelineCollectionEnabled {
		timelineBuffer.startSync(name)
	}
}

// FinishSync finishes the last block started.
func (flutterTimeline) FinishSync() {
	developer.TimelineFinishSync()
	if !KReleaseMode && timelineCollectionEnabled {
		timelineBuffer.finishSync()
	}
}

// InstantSync emits an instant event (never collected).
func (flutterTimeline) InstantSync(name string, arguments map[string]any) {
	developer.TimelineInstantSync(name, arguments)
}

// FlutterTimelineTimeSync is FlutterTimeline.timeSync: runs function inside
// a block named name.
func FlutterTimelineTimeSync[T any](name string, function func() T, opts ...developer.TimelineOptions) T {
	FlutterTimeline.StartSync(name, opts...)
	defer FlutterTimeline.FinishSync()
	return function()
}

// Now is the current timeline timestamp, in microseconds.
func (flutterTimeline) Now() int { return int(performanceTimestamp()) }

// performanceTimestamp is _timeline_io.dart's: Timeline.now as a double.
func performanceTimestamp() float64 { return float64(developer.TimelineNow()) }

// DebugCollect returns the blocks collected since the last collection or
// reset, and resets. It panics with a StateError in release mode or when
// collection is not enabled.
func (flutterTimeline) DebugCollect() *AggregatedTimings {
	if KReleaseMode {
		panic(releaseModeNotSupportedError())
	}
	if !timelineCollectionEnabled {
		panic(dart.StateError{Message: "Timeline metric collection not enabled."})
	}
	result := NewAggregatedTimings(timelineBuffer.computeTimings())
	FlutterTimeline.DebugReset()
	return result
}

// DebugReset forgets the collected blocks. It panics with a StateError in
// release mode.
func (flutterTimeline) DebugReset() {
	if KReleaseMode {
		panic(releaseModeNotSupportedError())
	}
	timelineBuffer = &blockBuffer{}
}

// TimedBlock is a block of code timed by FlutterTimeline: its name and
// start and end timestamps, in microseconds.
type TimedBlock struct {
	Name  string
	Start float64
	End   float64
}

// NewTimedBlock creates a TimedBlock; end must not be before start.
func NewTimedBlock(name string, start, end float64) TimedBlock {
	dartAssert(end >= start, "The start timestamp must not be greater than the end timestamp.")
	return TimedBlock{Name: name, Start: start, End: end}
}

// Duration is End - Start.
func (b TimedBlock) Duration() float64 { return b.End - b.Start }

func (b TimedBlock) String() string {
	return "TimedBlock(" + b.Name + ", " + dart.DoubleToString(b.Start) + ", " +
		dart.DoubleToString(b.End) + ", " + dart.DoubleToString(b.Duration()) + ")"
}

// AggregatedTimings are the timings collected by FlutterTimeline.
type AggregatedTimings struct {
	// TimedBlocks are the blocks, in the order they finished.
	TimedBlocks []TimedBlock

	aggregated []AggregatedTimedBlock
	computed   bool
}

// NewAggregatedTimings is AggregatedTimings(timedBlocks).
func NewAggregatedTimings(timedBlocks []TimedBlock) *AggregatedTimings {
	return &AggregatedTimings{TimedBlocks: timedBlocks}
}

// AggregatedBlocks sums the blocks by name, in the order names first
// appear (computed once).
func (t *AggregatedTimings) AggregatedBlocks() []AggregatedTimedBlock {
	if !t.computed {
		t.aggregated = t.computeAggregatedBlocks()
		t.computed = true
	}
	return t.aggregated
}

func (t *AggregatedTimings) computeAggregatedBlocks() []AggregatedTimedBlock {
	index := map[string]int{}
	var durations []float64
	var counts []int
	var names []string
	for _, block := range t.TimedBlocks {
		i, ok := index[block.Name]
		if !ok {
			i = len(names)
			index[block.Name] = i
			names = append(names, block.Name)
			durations = append(durations, 0)
			counts = append(counts, 0)
		}
		durations[i] += block.Duration()
		counts[i]++
	}
	result := make([]AggregatedTimedBlock, len(names))
	for i, name := range names {
		result[i] = NewAggregatedTimedBlock(name, durations[i], counts[i])
	}
	return result
}

// GetAggregated returns the aggregation for name, or one with zero duration
// and count if no block had that name.
func (t *AggregatedTimings) GetAggregated(name string) AggregatedTimedBlock {
	var found *AggregatedTimedBlock
	for _, block := range t.AggregatedBlocks() {
		if block.Name == name {
			if found != nil {
				panic(dart.StateError{Message: "Too many elements"})
			}
			found = &block
		}
	}
	if found == nil {
		return NewAggregatedTimedBlock(name, 0, 0)
	}
	return *found
}

// AggregatedTimedBlock is the total duration and count of the blocks with
// one name.
type AggregatedTimedBlock struct {
	Name     string
	Duration float64
	Count    int
}

// NewAggregatedTimedBlock creates an AggregatedTimedBlock; duration must
// not be negative.
func NewAggregatedTimedBlock(name string, duration float64, count int) AggregatedTimedBlock {
	dartAssert(duration >= 0, nil)
	return AggregatedTimedBlock{Name: name, Duration: duration, Count: count}
}

func (b AggregatedTimedBlock) String() string {
	return "AggregatedTimedBlock(" + b.Name + ", " + dart.DoubleToString(b.Duration) + ", " +
		strconv.Itoa(b.Count) + ")"
}

// blockBuffer is _BlockBuffer. Its stack of open blocks is static in Dart
// (shared by every buffer, so a reset keeps open blocks open); the chained
// fixed-size slices that store finished blocks are an allocation detail,
// here plain slices.
type blockBuffer struct {
	starts, finishes []float64
	names            []string
}

const blockStackDepth = 1000

var (
	blockStartStack   [blockStackDepth]float64
	blockNameStack    [blockStackDepth]string
	blockStackPointer = 0
)

func (b *blockBuffer) computeTimings() []TimedBlock {
	if blockStackPointer != 0 {
		panic(dart.NewAssertionError("Invalid sequence of `startSync` and `finishSync`.\n" +
			"The operation stack was not empty. The following operations are still " +
			"waiting to be finished via the `finishSync` method:\n" +
			strings.Join(blockNameStack[:blockStackPointer], ", ")))
	}
	result := make([]TimedBlock, 0, len(b.finishes))
	for i := range b.finishes {
		result = append(result, NewTimedBlock(b.names[i], b.starts[i], b.finishes[i]))
	}
	return result
}

func (b *blockBuffer) startSync(name string) {
	blockStartStack[blockStackPointer] = performanceTimestamp()
	blockNameStack[blockStackPointer] = name
	blockStackPointer++
}

func (b *blockBuffer) finishSync() {
	dartAssert(blockStackPointer > 0, "Invalid sequence of `startSync` and `finishSync`.\n"+
		"Attempted to finish timing a block of code, but there are no pending "+
		"`startSync` calls.")
	finishTime := performanceTimestamp()
	startTime := blockStartStack[blockStackPointer-1]
	name := blockNameStack[blockStackPointer-1]
	blockStackPointer--
	b.starts = append(b.starts, startTime)
	b.finishes = append(b.finishes, finishTime)
	b.names = append(b.names, name)
}
