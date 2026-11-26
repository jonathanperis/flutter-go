// Port of packages/flutter/test/foundation/timeline_test.dart (all 9
// tests), plus tests of the parts Flutter's tests don't reach (toString,
// the assertions of unbalanced calls, aggregation order and sums, nested
// blocks, release-only errors are untested: debug build).
package foundation

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

const kSliceSize = 500

func timelineSetUp(t *testing.T) {
	FlutterTimeline.DebugReset()
	FlutterTimeline.SetDebugCollectionEnabled(false)
	t.Cleanup(func() {
		FlutterTimeline.DebugReset()
		FlutterTimeline.SetDebugCollectionEnabled(false)
	})
}

func expectPanic(t *testing.T, check func(r any) bool, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil || !check(r) {
			t.Fatalf("recovered %#v", r)
		}
	}()
	f()
}

func isStateError(r any) bool { _, ok := r.(dart.StateError); return ok }

func TestTimelineDoesNotCollectWhenCollectionNotEnabled(t *testing.T) {
	timelineSetUp(t)
	FlutterTimeline.StartSync("TEST")
	FlutterTimeline.FinishSync()
	expectPanic(t, isStateError, func() { FlutterTimeline.DebugCollect() })
}

func TestTimelineCollectsWhenCollectionIsEnabled(t *testing.T) {
	timelineSetUp(t)
	FlutterTimeline.SetDebugCollectionEnabled(true)
	FlutterTimeline.StartSync("TEST")
	FlutterTimeline.FinishSync()
	data := FlutterTimeline.DebugCollect()
	equal(t, len(data.TimedBlocks), 1)
	equal(t, len(data.AggregatedBlocks()), 1)

	block := data.GetAggregated("TEST")
	equal(t, block.Name, "TEST")
	equal(t, block.Count, 1)

	// After collection the timeline is reset back to empty.
	data2 := FlutterTimeline.DebugCollect()
	equal(t, len(data2.TimedBlocks), 0)
	equal(t, len(data2.AggregatedBlocks()), 0)
}

func TestTimelineDeletesOldDataWhenReset(t *testing.T) {
	timelineSetUp(t)
	FlutterTimeline.SetDebugCollectionEnabled(true)
	FlutterTimeline.StartSync("TEST")
	FlutterTimeline.FinishSync()
	FlutterTimeline.DebugReset()

	data := FlutterTimeline.DebugCollect()
	equal(t, len(data.TimedBlocks), 0)
	equal(t, len(data.AggregatedBlocks()), 0)
}

func TestTimelineReportsZeroAggregationWhenRequestedMissingBlock(t *testing.T) {
	timelineSetUp(t)
	FlutterTimeline.SetDebugCollectionEnabled(true)
	data := FlutterTimeline.DebugCollect()
	block := data.GetAggregated("MISSING")
	equal(t, block.Name, "MISSING")
	equal(t, block.Count, 0)
	equal(t, block.Duration, 0.0)
}

func TestTimelineMeasuresTheRuntimeOfAFunction(t *testing.T) {
	timelineSetUp(t)
	FlutterTimeline.SetDebugCollectionEnabled(true)
	start := FlutterTimeline.Now() - 1
	FlutterTimelineTimeSync("TEST", func() struct{} {
		watch := time.Now()
		for time.Since(watch) < 5*time.Millisecond {
		}
		return struct{}{}
	})
	end := FlutterTimeline.Now() + 1

	data := FlutterTimeline.DebugCollect()
	equal(t, len(data.TimedBlocks), 1)
	equal(t, len(data.AggregatedBlocks()), 1)

	block := data.TimedBlocks[0]
	equal(t, block.Name, "TEST")
	if block.Start < float64(start) || block.End > float64(end) || !(block.Duration() > 0) {
		t.Fatalf("block %v not within [%d, %d]", block, start, end)
	}

	aggregated := data.GetAggregated("TEST")
	equal(t, aggregated.Name, "TEST")
	equal(t, aggregated.Count, 1)
	equal(t, aggregated.Duration, block.Duration())
}

func TestTimelineInstantSyncDoesNotCollectAnything(t *testing.T) {
	timelineSetUp(t)
	FlutterTimeline.SetDebugCollectionEnabled(true)
	FlutterTimeline.InstantSync("TEST", nil)
	data := FlutterTimeline.DebugCollect()
	equal(t, len(data.TimedBlocks), 0)
	equal(t, len(data.AggregatedBlocks()), 0)
}

func TestTimelineNowReturnsAValue(t *testing.T) {
	timelineSetUp(t)
	FlutterTimeline.SetDebugCollectionEnabled(true)
	_ = FlutterTimeline.Now() // Go ints are never null
}

func TestTimelineCanCollectMoreThanOneSliceOfData(t *testing.T) {
	timelineSetUp(t)
	FlutterTimeline.SetDebugCollectionEnabled(true)
	for range 10 * kSliceSize {
		FlutterTimeline.StartSync("TEST")
		FlutterTimeline.FinishSync()
	}
	data := FlutterTimeline.DebugCollect()
	equal(t, len(data.TimedBlocks), 10*kSliceSize)
	equal(t, len(data.AggregatedBlocks()), 1)
	block := data.GetAggregated("TEST")
	equal(t, block.Name, "TEST")
	equal(t, block.Count, 10*kSliceSize)
}

func TestTimelineCollectsBlocksInACorrectOrder(t *testing.T) {
	timelineSetUp(t)
	FlutterTimeline.SetDebugCollectionEnabled(true)
	const testCount = 7 * kSliceSize / 2
	for i := range testCount {
		FlutterTimeline.StartSync(fmt.Sprint("TEST", i))
		FlutterTimeline.FinishSync()
	}
	data := FlutterTimeline.DebugCollect()
	equal(t, len(data.TimedBlocks), testCount)
	var names, want []string
	for i, b := range data.TimedBlocks {
		names = append(names, b.Name)
		want = append(want, fmt.Sprint("TEST", i))
	}
	if !slices.Equal(names, want) {
		t.Fatal("order")
	}
}

// Not in Flutter's tests.

func TestTimelineNestedBlocksAndAggregation(t *testing.T) {
	timelineSetUp(t)
	FlutterTimeline.SetDebugCollectionEnabled(true)
	FlutterTimeline.StartSync("outer")
	FlutterTimeline.StartSync("inner")
	FlutterTimeline.FinishSync()
	FlutterTimeline.StartSync("other")
	FlutterTimeline.FinishSync()
	FlutterTimeline.StartSync("inner")
	FlutterTimeline.FinishSync()
	FlutterTimeline.FinishSync()
	data := FlutterTimeline.DebugCollect()
	var names []string
	for _, b := range data.TimedBlocks {
		names = append(names, b.Name)
	}
	// In the order the blocks finish.
	if !slices.Equal(names, []string{"inner", "other", "inner", "outer"}) {
		t.Fatal(names)
	}
	outer := data.TimedBlocks[3]
	if outer.Start > data.TimedBlocks[0].Start || outer.End < data.TimedBlocks[2].End {
		t.Fatal("outer block must contain the inner ones")
	}
	var aggNames []string
	for _, b := range data.AggregatedBlocks() {
		aggNames = append(aggNames, b.Name)
	}
	// In the order names first appear.
	if !slices.Equal(aggNames, []string{"inner", "other", "outer"}) {
		t.Fatal(aggNames)
	}
	inner := data.GetAggregated("inner")
	equal(t, inner.Count, 2)
	equal(t, inner.Duration, data.TimedBlocks[0].Duration()+data.TimedBlocks[2].Duration())
}

func TestTimelineToStringAndAssertions(t *testing.T) {
	equal(t, NewTimedBlock("a", 1, 2.5).String(), "TimedBlock(a, 1.0, 2.5, 1.5)")
	equal(t, NewAggregatedTimedBlock("a", 3, 2).String(), "AggregatedTimedBlock(a, 3.0, 2)")
	isAssertion := func(r any) bool { _, ok := r.(*dart.AssertionError); return ok }
	expectPanic(t, isAssertion, func() { NewTimedBlock("a", 2, 1) })
	expectPanic(t, isAssertion, func() { NewAggregatedTimedBlock("a", -1, 0) })

	timelineSetUp(t)
	// Timeline.finishSync without startSync is a StateError.
	expectPanic(t, isStateError, func() { FlutterTimeline.FinishSync() })
	// Collecting with a block still open is an assertion naming it.
	FlutterTimeline.SetDebugCollectionEnabled(true)
	FlutterTimeline.StartSync("open1")
	FlutterTimeline.StartSync("open2")
	expectPanic(t, func(r any) bool {
		e, ok := r.(*dart.AssertionError)
		return ok && e.AssertionMessage() == "Invalid sequence of `startSync` and `finishSync`.\n"+
			"The operation stack was not empty. The following operations are still "+
			"waiting to be finished via the `finishSync` method:\nopen1, open2"
	}, func() { FlutterTimeline.DebugCollect() })
	FlutterTimeline.FinishSync()
	FlutterTimeline.FinishSync()
	// Enabling collection with a block open: its finish has no start in
	// the buffer, an assertion.
	FlutterTimeline.SetDebugCollectionEnabled(false)
	FlutterTimeline.StartSync("before")
	FlutterTimeline.SetDebugCollectionEnabled(true)
	expectPanic(t, isAssertion, func() { FlutterTimeline.FinishSync() })
	// Setting the same value doesn't reset.
	FlutterTimeline.StartSync("kept")
	FlutterTimeline.FinishSync()
	FlutterTimeline.SetDebugCollectionEnabled(true)
	equal(t, len(FlutterTimeline.DebugCollect().TimedBlocks), 1)
}
