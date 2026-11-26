// Port of packages/flutter/test/gestures/arena_test.dart (Flutter 3.50.1):
// all 11 tests. No skips. The test's GestureTester is arenaTester here
// (flutter_test's GestureTester is another class).
package gestures_test

import (
	"testing"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/src/async"
)

const primaryKey = 4

type testGestureArenaMember struct {
	t         *testing.T
	acceptRan bool
	rejectRan bool
}

func (m *testGestureArenaMember) AcceptGesture(key int) {
	if key != primaryKey {
		m.t.Errorf("accept key %d", key)
	}
	m.acceptRan = true
}

func (m *testGestureArenaMember) RejectGesture(key int) {
	if key != primaryKey {
		m.t.Errorf("reject key %d", key)
	}
	m.rejectRan = true
}

type arenaTester struct {
	t                       *testing.T
	arena                   *GestureArenaManager
	first, second           *testGestureArenaMember
	firstEntry, secondEntry GestureArenaEntry
}

func newArenaTester(t *testing.T) *arenaTester {
	return &arenaTester{t: t, arena: NewGestureArenaManager(),
		first: &testGestureArenaMember{t: t}, second: &testGestureArenaMember{t: t}}
}

func (g *arenaTester) addFirst()  { g.firstEntry = g.arena.Add(primaryKey, g.first) }
func (g *arenaTester) addSecond() { g.secondEntry = g.arena.Add(primaryKey, g.second) }

func (g *arenaTester) expect(fa, fr, sa, sr bool) {
	g.t.Helper()
	if g.first.acceptRan != fa || g.first.rejectRan != fr || g.second.acceptRan != sa || g.second.rejectRan != sr {
		g.t.Fatalf("first accept/reject %v/%v second %v/%v, want %v/%v %v/%v",
			g.first.acceptRan, g.first.rejectRan, g.second.acceptRan, g.second.rejectRan, fa, fr, sa, sr)
	}
}

func (g *arenaTester) expectNothing()   { g.t.Helper(); g.expect(false, false, false, false) }
func (g *arenaTester) expectFirstWin()  { g.t.Helper(); g.expect(true, false, false, true) }
func (g *arenaTester) expectSecondWin() { g.t.Helper(); g.expect(false, true, true, false) }

func TestArenaShouldWinByAccepting(t *testing.T) {
	g := newArenaTester(t)
	g.addFirst()
	g.addSecond()
	g.arena.Close(primaryKey)
	g.expectNothing()
	g.firstEntry.Resolve(GestureDispositionAccepted)
	g.expectFirstWin()
}

func TestArenaShouldWinBySweep(t *testing.T) {
	g := newArenaTester(t)
	g.addFirst()
	g.addSecond()
	g.arena.Close(primaryKey)
	g.expectNothing()
	g.arena.Sweep(primaryKey)
	g.expectFirstWin()
}

func TestArenaShouldWinOnReleaseAfterHoldSweepRelease(t *testing.T) {
	g := newArenaTester(t)
	g.addFirst()
	g.addSecond()
	g.arena.Close(primaryKey)
	g.expectNothing()
	g.arena.Hold(primaryKey)
	g.expectNothing()
	g.arena.Sweep(primaryKey)
	g.expectNothing()
	g.arena.Release(primaryKey)
	g.expectFirstWin()
}

func TestArenaShouldWinOnSweepAfterHoldReleaseSweep(t *testing.T) {
	g := newArenaTester(t)
	g.addFirst()
	g.addSecond()
	g.arena.Close(primaryKey)
	g.expectNothing()
	g.arena.Hold(primaryKey)
	g.expectNothing()
	g.arena.Release(primaryKey)
	g.expectNothing()
	g.arena.Sweep(primaryKey)
	g.expectFirstWin()
}

func TestArenaOnlyFirstWinnerShouldWin(t *testing.T) {
	g := newArenaTester(t)
	g.addFirst()
	g.addSecond()
	g.arena.Close(primaryKey)
	g.expectNothing()
	g.firstEntry.Resolve(GestureDispositionAccepted)
	g.secondEntry.Resolve(GestureDispositionAccepted)
	g.expectFirstWin()
}

func TestArenaOnlyFirstWinnerShouldWinRegardlessOfOrder(t *testing.T) {
	g := newArenaTester(t)
	g.addFirst()
	g.addSecond()
	g.arena.Close(primaryKey)
	g.expectNothing()
	g.secondEntry.Resolve(GestureDispositionAccepted)
	g.firstEntry.Resolve(GestureDispositionAccepted)
	g.expectSecondWin()
}

func TestArenaWinBeforeCloseIsDelayedToClose(t *testing.T) {
	g := newArenaTester(t)
	g.addFirst()
	g.addSecond()
	g.expectNothing()
	g.firstEntry.Resolve(GestureDispositionAccepted)
	g.expectNothing()
	g.arena.Close(primaryKey)
	g.expectFirstWin()
}

func TestArenaWinBeforeCloseOnlyFirstWinner(t *testing.T) {
	g := newArenaTester(t)
	g.addFirst()
	g.addSecond()
	g.expectNothing()
	g.firstEntry.Resolve(GestureDispositionAccepted)
	g.secondEntry.Resolve(GestureDispositionAccepted)
	g.expectNothing()
	g.arena.Close(primaryKey)
	g.expectFirstWin()
}

func TestArenaWinBeforeCloseOnlyFirstWinnerRegardlessOfOrder(t *testing.T) {
	g := newArenaTester(t)
	g.addFirst()
	g.addSecond()
	g.expectNothing()
	g.secondEntry.Resolve(GestureDispositionAccepted)
	g.firstEntry.Resolve(GestureDispositionAccepted)
	g.expectNothing()
	g.arena.Close(primaryKey)
	g.expectSecondWin()
}

func TestArenaEagerWinnerClearedWhenItRejectsWhileOpen(t *testing.T) {
	arena := NewGestureArenaManager()
	memberA := &testGestureArenaMember{t: t}
	memberB := &testGestureArenaMember{t: t}
	memberC := &testGestureArenaMember{t: t}
	entryA := arena.Add(primaryKey, memberA)
	arena.Add(primaryKey, memberB)
	arena.Add(primaryKey, memberC)

	entryA.Resolve(GestureDispositionAccepted)
	if memberA.acceptRan || memberA.rejectRan {
		t.Fatal("A resolved while open")
	}
	entryA.Resolve(GestureDispositionRejected)
	if !memberA.rejectRan || memberA.acceptRan {
		t.Fatal("A not rejected")
	}
	arena.Close(primaryKey)
	if memberA.acceptRan || memberB.rejectRan || memberC.rejectRan {
		t.Fatal("stale eager winner resolved the arena")
	}
	arena.Sweep(primaryKey)
	if !memberB.acceptRan || memberB.rejectRan || !memberC.rejectRan || memberC.acceptRan {
		t.Fatal("B should win the sweep")
	}
}

// Not in Flutter's tests: a single remaining member wins by default in a
// microtask after the arena closes.
func TestArenaSingleMemberWinsByDefaultInMicrotask(t *testing.T) {
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	g := newArenaTester(t)
	g.addFirst()
	g.arena.Close(primaryKey)
	if g.first.acceptRan {
		t.Fatal("accepted synchronously")
	}
	loop.FlushMicrotasks()
	if !g.first.acceptRan {
		t.Fatal("not accepted by default")
	}
}
