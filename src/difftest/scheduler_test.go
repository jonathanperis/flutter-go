package difftest

// Go twin of flutter/test/scheduler_test.dart: the same scenes, step for
// step, on a BindingBase with SchedulerBinding driven by the same pump in
// a fake event loop. The logs must be identical.

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type schedBinding struct {
	foundation.BindingBaseBase
	SchedulerBindingMixin
}

func (b *schedBinding) InitInstances()         { b.SchedulerBindingMixin.InitInstances() }
func (b *schedBinding) InitServiceExtensions() { b.SchedulerBindingMixin.InitServiceExtensions() }

var (
	sLog     []string
	sBinding *schedBinding
	sLoop    *async.Loop
)

const sEpoch = int64(1420070400000000) // DateTime.utc(2015).microsecondsSinceEpoch

func slog(s string) { sLog = append(sLog, s) }

func (b *schedBinding) HandleBeginFrame(rawTimeStamp *time.Duration) {
	raw := "null"
	if rawTimeStamp != nil {
		raw = strconv.FormatInt(rawTimeStamp.Microseconds()-sEpoch, 10)
	}
	slog("handleBeginFrame " + raw + " phase=" + sPhase())
	b.SchedulerBindingMixin.HandleBeginFrame(rawTimeStamp)
	slog("handleBeginFrame done phase=" + sPhase())
}

func (b *schedBinding) HandleDrawFrame() {
	slog("handleDrawFrame phase=" + sPhase())
	b.SchedulerBindingMixin.HandleDrawFrame()
	slog("handleDrawFrame done phase=" + sPhase())
}

func (b *schedBinding) PostEvent(eventKind string, eventData map[string]any) {
	// Dart's map literal keeps insertion order.
	var parts []string
	for _, k := range []string{"number", "startTime", "elapsed", "build", "raster", "vsyncOverhead"} {
		parts = append(parts, fmt.Sprintf("%q:%v", k, eventData[k]))
	}
	slog("postEvent " + eventKind + " {" + strings.Join(parts, ",") + "}")
}

func (b *schedBinding) Unlocked() {
	slog("unlocked")
	b.SchedulerBindingMixin.Unlocked()
}

func sPhase() string            { return sBinding.SchedulerPhase().Name() }
func us(d time.Duration) string { return strconv.FormatInt(d.Microseconds(), 10) }
func boolStr(b bool) string     { return strconv.FormatBool(b) }
func itoa(i int) string         { return strconv.Itoa(i) }
func rawNow() time.Duration     { return time.Duration(sEpoch)*time.Microsecond + sLoop.Elapsed() }
func sysTS() string {
	return strconv.FormatInt(sBinding.CurrentSystemFrameTimeStamp().Microseconds()-sEpoch, 10)
}
func dur(d time.Duration) *time.Duration { return &d }

// pump is flutter_test's pump.
func pump(duration *time.Duration) {
	if duration == nil {
		slog("pump null")
	} else {
		slog("pump " + us(*duration))
		sLoop.Elapse(*duration)
	}
	if sBinding.HasScheduledFrame() {
		sLoop.FlushMicrotasks()
		raw := rawNow()
		sBinding.HandleBeginFrame(&raw)
		sLoop.FlushMicrotasks()
		sBinding.HandleDrawFrame()
	}
	sLoop.FlushMicrotasks()
	slog("pumped hasScheduledFrame=" + boolStr(sBinding.HasScheduledFrame()) + " transient=" + itoa(sBinding.TransientCallbackCount()))
}

func msd(n int) *time.Duration { return dur(time.Duration(n) * time.Millisecond) }
func usd(n int) *time.Duration { return dur(time.Duration(n) * time.Microsecond) }

var tickerFutureHash = regexp.MustCompile(`^TickerFuture#[0-9a-f]{5}`)

func futureState(f *TickerFuture) string {
	return tickerFutureHash.ReplaceAllString(f.String(), "TickerFuture")
}

func dartTypeOf(e any) string {
	t := reflect.TypeOf(e)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}

func schedListen(name string, f async.Future[async.Void]) {
	async.OnComplete(f, func(async.Void) { slog(name + " complete phase=" + sPhase()) },
		func(e error) { slog(name + " error " + dartTypeOf(e) + " " + e.Error() + " phase=" + sPhase()) })
}

func microtask(f func()) { async.ScheduleMicrotask(f) }

func sceneFrameOrder() {
	b := sBinding
	slog("== frame order")
	b.AddPersistentFrameCallback(func(ts time.Duration) {
		slog("persistent A ts=" + us(ts) + " phase=" + sPhase() + " current=" + us(b.CurrentFrameTimeStamp()) + " system=" + sysTS())
	})
	addedNested := false
	b.AddPersistentFrameCallback(func(time.Duration) {
		slog("persistent B phase=" + sPhase())
		if !addedNested {
			addedNested = true
			b.AddPersistentFrameCallback(func(time.Duration) { slog("persistent C (added during B)") })
			b.AddPostFrameCallback(func(time.Duration) {
				slog("post 1 (added in persistent) phase=" + sPhase())
				b.AddPostFrameCallback(func(time.Duration) { slog("post 3 (added in post 1) phase=" + sPhase()) })
				microtask(func() { slog("microtask from post 1 phase=" + sPhase()) })
			})
		}
	})
	slog("hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	pump(nil)
	b.AddPostFrameCallback(func(ts time.Duration) { slog("post 2 (added before frame) ts=" + us(ts)) }, AddPostFrameCallbackOptions{DebugLabel: "two"})
	id1 := b.ScheduleFrameCallback(func(ts time.Duration) {
		slog("transient 1 ts=" + us(ts) + " phase=" + sPhase())
		microtask(func() { slog("microtask from transient 1 phase=" + sPhase()) })
		c := async.NewCompleter[async.Void]()
		async.OnComplete(c.Future(), func(async.Void) { slog("future from transient 1 phase=" + sPhase()) }, nil)
		c.Complete(async.Void{})
		b.ScheduleFrameCallback(func(ts time.Duration) { slog("transient 3 (added in 1) ts=" + us(ts) + " phase=" + sPhase()) })
		slog("in transient: hasScheduledFrame=" + boolStr(b.HasScheduledFrame()) + " count=" + itoa(b.TransientCallbackCount()))
	})
	id2 := b.ScheduleFrameCallback(func(ts time.Duration) { slog("transient 2 ts=" + us(ts)) })
	slog("ids " + itoa(id1) + " " + itoa(id2) + " count=" + itoa(b.TransientCallbackCount()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	microtask(func() { slog("microtask before frame phase=" + sPhase()) })
	async.NewTimer(5*time.Millisecond, func() { slog("timer 5ms phase=" + sPhase()) })
	async.NewTimer(16*time.Millisecond, func() { slog("timer 16ms phase=" + sPhase()) })
	pump(msd(16))
	pump(usd(16667))
	pump(msd(1))
	b.ScheduleFrame()
	pump(msd(3))
	var idB, idSelf int
	b.ScheduleFrameCallback(func(time.Duration) {
		slog("transient cancel A")
		b.CancelFrameCallbackWithId(idB)
	})
	idB = b.ScheduleFrameCallback(func(time.Duration) { slog("transient cancel B (must not run)") })
	idC := b.ScheduleFrameCallback(func(time.Duration) { slog("transient cancel C (must not run)") })
	idSelf = b.ScheduleFrameCallback(func(time.Duration) {
		slog("transient self")
		b.CancelFrameCallbackWithId(idSelf)
	})
	b.CancelFrameCallbackWithId(idC)
	slog("count after cancel=" + itoa(b.TransientCallbackCount()))
	pump(msd(10))
	no := false
	b.ScheduleFrameCallback(func(ts time.Duration) { slog("transient no new frame ts=" + us(ts)) }, ScheduleFrameCallbackOptions{ScheduleNewFrame: &no})
	slog("hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	pump(msd(10))
	b.EnsureVisualUpdate()
	pump(msd(10))
	b.ScheduleFrameCallback(func(time.Duration) {
		b.EnsureVisualUpdate()
		slog("ensureVisualUpdate in transient: " + boolStr(b.HasScheduledFrame()))
		microtask(func() {
			b.EnsureVisualUpdate()
			slog("ensureVisualUpdate in midFrameMicrotasks: " + boolStr(b.HasScheduledFrame()))
		})
	})
	once := false
	b.AddPersistentFrameCallback(func(time.Duration) {
		if !once {
			once = true
			b.EnsureVisualUpdate()
			slog("ensureVisualUpdate in persistent: " + boolStr(b.HasScheduledFrame()))
			b.AddPostFrameCallback(func(time.Duration) {
				b.EnsureVisualUpdate()
				slog("ensureVisualUpdate in post: " + boolStr(b.HasScheduledFrame()))
			})
		}
	})
	pump(msd(10))
	pump(msd(10))
	b.ScheduleFrameCallback(func(ts time.Duration) { slog("transient warm ts=" + us(ts)) })
	b.HandleBeginFrame(nil)
	sLoop.FlushMicrotasks()
	b.HandleDrawFrame()
	pump(msd(7))
}

func sceneEndOfFrame() {
	b := sBinding
	slog("== endOfFrame")
	schedListen("endOfFrame idle", b.EndOfFrame())
	slog("identical=" + boolStr(b.EndOfFrame() == b.EndOfFrame()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	pump(msd(4))
	b.ScheduleFrameCallback(func(time.Duration) { schedListen("endOfFrame in transient", b.EndOfFrame()) })
	done := false
	b.AddPersistentFrameCallback(func(time.Duration) {
		if !done {
			done = true
			b.AddPostFrameCallback(func(time.Duration) {
				schedListen("endOfFrame in post", b.EndOfFrame())
				slog("in post hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
			})
		}
	})
	pump(msd(4))
	pump(msd(4))
	pump(msd(4))
}

func sceneTickers() {
	b := sBinding
	slog("== tickers")
	make := func(name string) *Ticker {
		return NewTicker(func(elapsed time.Duration) { slog(name + " tick " + us(elapsed) + " phase=" + sPhase()) }, TickerOptions{DebugLabel: name})
	}
	outside := make("outside")
	slog(outside.String() + " isActive=" + boolStr(outside.IsActive()) + " isTicking=" + boolStr(outside.IsTicking()))
	f1 := outside.Start()
	schedListen("outside future", f1)
	f1.WhenCompleteOrCancel(func() { slog("outside whenCompleteOrCancel") })
	slog(futureState(f1) + " isActive=" + boolStr(outside.IsActive()) + " isTicking=" + boolStr(outside.IsTicking()))
	pump(msd(10))
	var inTransient, inMicrotask, inPersistent, inPost *Ticker
	b.ScheduleFrameCallback(func(time.Duration) {
		inTransient = make("inTransient")
		inTransient.Start()
		microtask(func() { inMicrotask = make("inMicrotask"); inMicrotask.Start() })
	})
	started := false
	b.AddPersistentFrameCallback(func(time.Duration) {
		if !started {
			started = true
			inPersistent = make("inPersistent")
			inPersistent.Start()
			b.AddPostFrameCallback(func(time.Duration) { inPost = make("inPost"); inPost.Start() })
		}
	})
	pump(msd(7))
	pump(msd(16))
	pump(msd(16))
	inTransient.SetMuted(true)
	slog("muted: isActive=" + boolStr(inTransient.IsActive()) + " isTicking=" + boolStr(inTransient.IsTicking()))
	pump(msd(16))
	inTransient.SetMuted(false)
	var selfMuter *Ticker
	muterTicks := 0
	selfMuter = NewTicker(func(elapsed time.Duration) {
		slog("selfMuter tick " + us(elapsed))
		muterTicks++
		if muterTicks == 2 {
			selfMuter.SetMuted(true)
		}
	})
	selfMuter.Start()
	pump(msd(16))
	var restarter *Ticker
	restarts := 0
	restarter = NewTicker(func(elapsed time.Duration) {
		slog("restarter tick " + us(elapsed))
		if restarts < 2 {
			restarts++
			restarter.Stop()
			schedListen("restarter future "+itoa(restarts), restarter.Start())
		} else {
			restarts++
		}
	})
	restarter.Start()
	pump(msd(5))
	pump(msd(5))
	pump(msd(5))
	pump(msd(5))
	f2 := NewTickerFutureComplete()
	slog("complete(): " + futureState(f2))
	schedListen("complete() orCancel", f2.OrCancel())
	outside.Stop()
	slog("after stop " + futureState(f1))
	schedListen("outside orCancel after stop", f1.OrCancel())
	inPersistent.Stop(TickerStopOptions{Canceled: true})
	pump(msd(1))
	c := make("canceled")
	fc := c.Start()
	schedListen("canceled future", fc)
	schedListen("canceled orCancel", fc.OrCancel())
	fc.WhenCompleteOrCancel(func() { slog("canceled whenCompleteOrCancel") })
	c.Stop(TickerStopOptions{Canceled: true})
	slog(futureState(fc))
	schedListen("canceled orCancel late", fc.OrCancel())
	c2 := make("canceled2")
	fc2 := c2.Start()
	c2.Stop(TickerStopOptions{Canceled: true})
	schedListen("orCancel first read after cancel", fc2.OrCancel())
	d := make("disposed")
	fd := d.Start()
	schedListen("disposed orCancel", fd.OrCancel())
	pump(msd(2))
	d.Dispose()
	slog("disposed " + futureState(fd) + " transient=" + itoa(b.TransientCallbackCount()))
	t2 := NewTicker(func(time.Duration) {})
	fd2 := t2.Start()
	fd2.Ignore()
	slog("fd2 " + futureState(fd2))
	t2.Dispose()
	slog("fd2 after dispose " + futureState(fd2))
	w := make("awaited")
	fw := w.Start()
	async.AsyncVoid(func() {
		slog("awaiting")
		fw.Await()
		slog("await returned phase=" + sPhase())
	})
	fw.WhenComplete(func() { slog("whenComplete phase=" + sPhase()) })
	async.OnComplete(fw.Timeout(100*time.Millisecond, nil), func(async.Void) { slog("timeout result complete") },
		func(e error) { slog("timeout error " + dartTypeOf(e)) })
	pump(msd(3))
	w.Stop()
	slog("stopped awaited")
	pump(msd(3))
	original := make("original")
	fo := original.Start()
	schedListen("original future", fo)
	pump(msd(4))
	pump(msd(4))
	successor := make("successor")
	successor.AbsorbTicker(original)
	slog("absorbed: successor active=" + boolStr(successor.IsActive()) + " original active=" + boolStr(original.IsActive()) + " transient=" + itoa(b.TransientCallbackCount()))
	pump(msd(4))
	successor.Stop()
	pump(msd(4))
	idleOriginal := make("idleOriginal")
	make("idleSuccessor").AbsorbTicker(idleOriginal)
	twice := make("twice")
	twice.Start()
	func() {
		defer func() {
			if e, ok := recover().(*foundation.FlutterError); ok {
				slog("started twice: " + itoa(len(e.Diagnostics)) + " " + e.Diagnostics[0].String() + " | " + e.Diagnostics[1].String() + " | " + e.Diagnostics[2].Name())
			}
		}()
		twice.Start()
	}()
	twice.Dispose()
	slog("selfMuter active=" + boolStr(selfMuter.IsActive()) + " ticking=" + boolStr(selfMuter.IsTicking()))
	for _, t := range []*Ticker{inTransient, inMicrotask, inPost, restarter, selfMuter} {
		t.Dispose()
	}
	pump(msd(1))
	slog(TickerCanceled{}.Error() + " | " + TickerCanceled{Ticker: make("x")}.Error())
}

func sceneTimeDilation() {
	b := sBinding
	slog("== timeDilation")
	t := NewTicker(func(elapsed time.Duration) { slog("dilated tick " + us(elapsed)) })
	t.Start()
	tsLog := func(ts time.Duration) { slog("ts " + us(ts)) }
	b.ScheduleFrameCallback(tsLog)
	pump(msd(10))
	pump(msd(10))
	SetTimeDilation(2.0)
	b.ScheduleFrameCallback(tsLog)
	pump(msd(10))
	pump(usd(16667))
	SetTimeDilation(3.0)
	pump(usd(16667))
	pump(usd(16667))
	pump(usd(1))
	pump(usd(2))
	SetTimeDilation(3.0)
	pump(usd(5))
	SetTimeDilation(0.3)
	pump(usd(7))
	pump(usd(16667))
	tie := NewTicker(func(elapsed time.Duration) { slog("tie tick " + us(elapsed)) })
	tie.Start()
	SetTimeDilation(2.0)
	pump(usd(1))
	pump(usd(1)) // 1 / 2 = 0.5
	pump(usd(2)) // 3 / 2 = 1.5
	pump(usd(2)) // 5 / 2 = 2.5
	SetTimeDilation(1.0)
	tie.Dispose()
	b.ScheduleFrameCallback(tsLog)
	pump(msd(10))
	t.Dispose()
	b.ResetEpoch()
	b.ScheduleFrameCallback(func(ts time.Duration) { slog("after resetEpoch ts=" + us(ts) + " system=" + sysTS()) })
	pump(msd(10))
}

func sceneTasks() {
	b := sBinding
	slog("== tasks")
	schedule := func(name string, p Priority, throws bool) {
		async.OnComplete(ScheduleTask(b, func() string {
			slog("task " + name + " phase=" + sPhase() + " transient=" + itoa(b.TransientCallbackCount()))
			if throws {
				panic(dart.StateError{Message: "task " + name + " failed"})
			}
			return name
		}, p, ScheduleTaskOptions{DebugLabel: &name}), func(v string) { slog("task result " + v) }, nil)
	}
	schedule("idle", PriorityIdle, false)
	schedule("touch", PriorityTouch, false)
	schedule("animation", PriorityAnimation, false)
	schedule("animation+1", PriorityAnimation.Add(1), false)
	schedule("animation-1", PriorityAnimation.Sub(1), false)
	schedule("idle+20000", PriorityIdle.Add(20000), false)
	schedule("tie a", PriorityIdle.Add(7), false)
	schedule("tie b", PriorityIdle.Add(7), false)
	schedule("tie c", PriorityIdle.Add(7), false)
	schedule("tie d", PriorityIdle.Add(7), false)
	schedule("touch-99999", PriorityTouch.Sub(99999), false)
	schedule("thrower", PriorityAnimation.Add(5), true)
	slog("scheduled; timers=" + itoa(len(sLoop.PendingTimers())))
	id := b.ScheduleFrameCallback(func(time.Duration) { slog("transient during tasks") })
	for i := 0; i < 7; i++ {
		slog("handleEventLoopCallback=" + boolStr(b.HandleEventLoopCallback()))
	}
	b.CancelFrameCallbackWithId(id)
	slog("transient=" + itoa(b.TransientCallbackCount()))
	ScheduleTask(b, func() async.Void { slog("task late"); return async.Void{} }, PriorityIdle.Add(3))
	sLoop.Elapse(0)
	pump(msd(1))
	sLoop.Elapse(0)
	slog("after drain")
	async.OnComplete(ScheduleTaskFuture(b, func() async.Future[int] {
		return async.Async(func() int {
			slog("async task start")
			async.FutureValue[any](nil).Await()
			slog("async task after await")
			return 42
		})
	}, PriorityTouch), func(v int) { slog("async task result " + itoa(v)) }, nil)
	sLoop.Elapse(0)
	lock := async.NewCompleter[async.Void]()
	b.LockEvents(func() async.Future[async.Void] { return lock.Future() })
	ScheduleTask(b, func() async.Void { slog("task while locked"); return async.Void{} }, PriorityTouch)
	slog("locked; handleEventLoopCallback=" + boolStr(b.HandleEventLoopCallback()))
	sLoop.Elapse(0)
	slog("still locked")
	lock.Complete(async.Void{})
	sLoop.FlushMicrotasks()
	slog("lock released")
	sLoop.Elapse(0)
	for _, pattern := range [][]int{
		{5, 5, 3, 5, 3, 3, 5, 1, 5, 3, 1, 5, 5, 3},
		{1, 2, 2, 1, 2, 1, 1, 2, 2, 2, 1},
		{4, 4, 4, 4, 4, 4, 4, 4, 4, 4},
		{9, 1, 9, 1, 9, 1, 9, 1},
	} {
		var order []string
		for n, p := range pattern {
			name := itoa(p) + "#" + itoa(n)
			ScheduleTask(b, func() async.Void { order = append(order, name); return async.Void{} }, PriorityIdle.Add(p))
		}
		for b.HandleEventLoopCallback() {
		}
		slog("heap order " + strings.Join(order, " "))
		sLoop.Elapse(0)
	}
	ScheduleTask(b, func() async.Void { slog("task before unlock"); return async.Void{} }, PriorityTouch)
	b.LockEvents(func() async.Future[async.Void] { return async.FutureValue(async.Void{}) })
	sLoop.FlushMicrotasks()
	slog("timers after unlock=" + itoa(len(sLoop.PendingTimers())))
	sLoop.Elapse(0)
	b.SetSchedulingStrategy(func(priority int, scheduler SchedulerBinding) bool {
		slog("strategy asked " + itoa(priority))
		return priority > 5
	})
	schedule("s3", PriorityIdle.Add(3), false)
	schedule("s9", PriorityIdle.Add(9), false)
	for i := 0; i < 3; i++ {
		slog("handleEventLoopCallback=" + boolStr(b.HandleEventLoopCallback()))
	}
	b.SetSchedulingStrategy(DefaultSchedulingStrategy)
	sLoop.Elapse(0)
	slog("handleEventLoopCallback empty=" + boolStr(b.HandleEventLoopCallback()))
}

func stateStr(s *ui.AppLifecycleState) string {
	if s == nil {
		return "null"
	}
	return s.String()
}

func sceneLifecycle() {
	b := sBinding
	slog("== lifecycle")
	slog("state=" + stateStr(b.LifecycleState()) + " framesEnabled=" + boolStr(b.FramesEnabled()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	t := NewTicker(func(e time.Duration) { slog("lifecycle tick " + us(e)) })
	for _, s := range []ui.AppLifecycleState{ui.AppLifecycleStateResumed, ui.AppLifecycleStateInactive, ui.AppLifecycleStateHidden} {
		b.HandleAppLifecycleStateChanged(s)
		slog(s.String() + " framesEnabled=" + boolStr(b.FramesEnabled()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	}
	t.Start()
	slog("started while hidden: isTicking=" + boolStr(t.IsTicking()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()) + " transient=" + itoa(b.TransientCallbackCount()))
	pump(msd(10))
	b.ScheduleForcedFrame()
	slog("forced: hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	pump(msd(10))
	forced := NewTicker(func(e time.Duration) { slog("forced tick " + us(e)) })
	forced.ForceFrames = true
	forced.Start()
	slog("forceFrames ticker: hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	pump(msd(10))
	pump(msd(10))
	forced.Dispose()
	b.HandleAppLifecycleStateChanged(ui.AppLifecycleStatePaused)
	b.HandleAppLifecycleStateChanged(ui.AppLifecycleStatePaused)
	slog("paused framesEnabled=" + boolStr(b.FramesEnabled()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	b.HandleAppLifecycleStateChanged(ui.AppLifecycleStateResumed)
	slog("resumed framesEnabled=" + boolStr(b.FramesEnabled()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	pump(msd(10))
	pump(msd(10))
	b.HandleAppLifecycleStateChanged(ui.AppLifecycleStateDetached)
	slog("detached framesEnabled=" + boolStr(b.FramesEnabled()))
	b.HandleAppLifecycleStateChanged(ui.AppLifecycleStateHidden)
	slog("hidden framesEnabled=" + boolStr(b.FramesEnabled()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	pump(msd(10))
	b.HandleAppLifecycleStateChanged(ui.AppLifecycleStateInactive)
	slog("inactive framesEnabled=" + boolStr(b.FramesEnabled()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	pump(msd(10))
	b.ResetInternalState()
	slog("reset state=" + stateStr(b.LifecycleState()) + " framesEnabled=" + boolStr(b.FramesEnabled()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	pump(msd(10))
	t.Dispose()
	pump(msd(10))
}

func sceneWarmUp() {
	b := sBinding
	slog("== warm-up")
	t := NewTicker(func(e time.Duration) { slog("warm tick " + us(e) + " phase=" + sPhase()) })
	t.Start()
	b.ScheduleFrameCallback(func(ts time.Duration) {
		slog("warm transient ts=" + us(ts))
		microtask(func() { slog("warm microtask phase=" + sPhase()) })
	})
	slog("before warm-up hasScheduledFrame=" + boolStr(b.HasScheduledFrame()) + " locked=" + boolStr(b.Locked()) + " timers=" + itoa(len(sLoop.PendingTimers())))
	b.ScheduleWarmUpFrame()
	b.ScheduleWarmUpFrame()
	slog("warm-up scheduled locked=" + boolStr(b.Locked()) + " timers=" + itoa(len(sLoop.PendingTimers())) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	ScheduleTask(b, func() async.Void { slog("task after warm-up"); return async.Void{} }, PriorityTouch)
	microtask(func() { slog("microtask before warm-up timers") })
	pd := ui.PlatformDispatcherInstance()
	engineFrame := func(when string) {
		slog("engine frame " + when)
		b.EnsureFrameCallbacksRegistered()
		pd.OnBeginFrame()(rawNow())
		pd.OnDrawFrame()()
		slog("engine frame returned hasScheduledFrame=" + boolStr(b.HasScheduledFrame()) + " phase=" + sPhase())
	}
	b.ScheduleFrameCallback(func(time.Duration) { microtask(func() { engineFrame("during warm-up") }) })
	async.NewTimer(2*time.Millisecond, func() { engineFrame("after warm-up") })
	sLoop.Elapse(3 * time.Millisecond)
	slog("after warm-up locked=" + boolStr(b.Locked()) + " hasScheduledFrame=" + boolStr(b.HasScheduledFrame()))
	pump(msd(16))
	pump(msd(16))
	b.ScheduleFrameCallback(func(ts time.Duration) { slog("engine transient ts=" + us(ts)) })
	pd.OnBeginFrame()(rawNow() + 5*time.Microsecond)
	sLoop.FlushMicrotasks()
	pd.OnDrawFrame()()
	t.Dispose()
	pump(msd(1))
	b.ScheduleFrame()
	b.ScheduleWarmUpFrame()
	sLoop.Elapse(0)
	slog("second warm-up: hasScheduledFrame=" + boolStr(b.HasScheduledFrame()) + " locked=" + boolStr(b.Locked()))
	pump(msd(1))
	pd.SetOnBeginFrame(nil)
	pd.SetOnDrawFrame(nil)
}

func libStr(d *foundation.FlutterErrorDetails) string {
	if d.Library == nil {
		return "null"
	}
	return *d.Library
}

func describeContext(d *foundation.FlutterErrorDetails) string {
	if d.Context == nil {
		return "null"
	}
	return d.Context.ToDescription()
}

func sceneErrors() {
	b := sBinding
	slog("== errors")
	previous := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) {
		slog("error: " + d.ExceptionAsString() + " | library=" + libStr(d) + " | context=" + describeContext(d) + " | collector=" + boolStr(d.InformationCollector != nil))
	}
	b.ScheduleFrameCallback(func(time.Duration) { panic(dart.StateError{Message: "transient boom"}) })
	b.ScheduleFrameCallback(func(time.Duration) { slog("transient after boom") })
	once := false
	b.AddPersistentFrameCallback(func(time.Duration) {
		if !once {
			once = true
			b.AddPostFrameCallback(func(time.Duration) { panic(dart.ArgumentError{Message: "post boom"}) })
			b.AddPostFrameCallback(func(time.Duration) { slog("post after boom") })
			panic("persistent boom")
		}
	})
	pump(msd(1))
	ScheduleTask(b, func() async.Void { panic("task boom") }, PriorityTouch)
	sLoop.Elapse(0)
	cb := func([]ui.FrameTiming) { panic("timings boom") }
	b.AddTimingsCallback(cb)
	b.PlatformDispatcher().OnReportTimings()([]ui.FrameTiming{})
	b.RemoveTimingsCallback(cb)
	t := NewTicker(func(time.Duration) {})
	t.Start()
	func() {
		defer func() {
			if e, ok := recover().(*foundation.FlutterError); ok {
				slog("rescheduling outside a callback: " + e.Diagnostics[0].String())
			}
		}()
		b.ScheduleFrameCallback(func(time.Duration) {}, ScheduleFrameCallbackOptions{Rescheduling: true})
	}()
	t.Dispose()
	b.ScheduleFrameCallback(func(time.Duration) {
		id := b.ScheduleFrameCallback(func(time.Duration) { slog("rescheduled ran") }, ScheduleFrameCallbackOptions{Rescheduling: true})
		slog("rescheduling inside a callback ok, id>0=" + boolStr(id > 0))
	})
	pump(msd(1))
	pump(msd(1))
	b.ScheduleFrameCallback(func(time.Duration) {})
	b.DebugAssertNoTransientCallbacks("leftover")
	b.ScheduleFrameCallback(func(time.Duration) {})
	b.DebugAssertNoTransientCallbacks("leftovers")
	pump(msd(1))
	b.DebugAssertNoTransientCallbacks("none")
	foundation.FlutterErrorOnError = previous
}

func sceneTimings() {
	b := sBinding
	slog("== timings")
	a := func(timings []ui.FrameTiming) {
		parts := make([]string, len(timings))
		for i, t := range timings {
			parts[i] = t.String()
		}
		slog("timings a " + itoa(len(timings)) + " " + strings.Join(parts, " | "))
	}
	var bcb ui.TimingsCallback
	bcb = func(timings []ui.FrameTiming) {
		slog("timings b " + itoa(len(timings)))
		b.RemoveTimingsCallback(a)
	}
	b.AddTimingsCallback(a)
	b.AddTimingsCallback(bcb)
	b.AddTimingsCallback(a)
	seven := 7
	timings := []ui.FrameTiming{
		ui.NewFrameTiming(1000, 1500, 9300, 9500, 12345, 99999, ui.FrameTimingOptions{FrameNumber: &seven}),
		ui.NewFrameTiming(0, 1, 2, 3, 4, 5, ui.FrameTimingOptions{LayerCacheCount: 2, LayerCacheBytes: 3145728, PictureCacheCount: 1, PictureCacheBytes: 1048577}),
	}
	slog(dart.DoubleToString(timings[1].LayerCacheMegabytes()) + " " + dart.DoubleToString(timings[1].PictureCacheMegabytes()) + " " + itoa(timings[0].TimestampInMicroseconds(ui.FramePhaseRasterFinishWallTime)))
	b.PlatformDispatcher().OnReportTimings()(timings)
	b.PlatformDispatcher().OnReportTimings()(timings)
	b.RemoveTimingsCallback(bcb)
	slog("onReportTimings null=" + boolStr(b.PlatformDispatcher().OnReportTimings() == nil))
}

func modeStr(m *ui.DartPerformanceMode) string {
	if m == nil {
		return "null"
	}
	return m.String()
}

func scenePerformanceMode() {
	b := sBinding
	slog("== performance mode")
	h1 := b.RequestPerformanceMode(ui.DartPerformanceModeLatency)
	h2 := b.RequestPerformanceMode(ui.DartPerformanceModeThroughput)
	h3 := b.RequestPerformanceMode(ui.DartPerformanceModeLatency)
	slog(boolStr(h1 != nil) + " " + boolStr(h2 != nil) + " " + boolStr(h3 != nil) + " " + modeStr(b.DebugGetRequestedPerformanceMode()))
	h1.Dispose()
	slog(modeStr(b.DebugGetRequestedPerformanceMode()))
	h3.Dispose()
	slog(modeStr(b.DebugGetRequestedPerformanceMode()))
	func() {
		defer func() {
			if e := recover(); e != nil {
				slog(fmt.Sprintf("pending error %v", e))
			}
		}()
		b.DebugAssertNoPendingPerformanceModeRequests("pending")
		slog("no pending")
	}()
}

func sceneBanners() {
	b := sBinding
	slog("== banners")
	previous := foundation.DebugPrint
	foundation.DebugPrint = func(message string, opts ...foundation.DebugPrintOptions) { slog("debugPrint " + message) }
	DebugPrintBeginFrameBanner = true
	DebugPrintEndFrameBanner = true
	b.ScheduleFrame()
	pump(msd(1))
	b.ScheduleFrame()
	pump(dur(24*time.Hour + 2*time.Hour + 3*time.Minute + 4*time.Second + 5*time.Millisecond + 6*time.Microsecond))
	b.ScheduleFrame()
	pump(usd(40))
	b.ScheduleFrame()
	pump(usd(843))
	b.ScheduleFrame()
	pump(usd(50))
	b.HandleBeginFrame(nil)
	b.HandleDrawFrame()
	DebugPrintBeginFrameBanner = false
	b.ScheduleFrame()
	pump(msd(2))
	DebugPrintEndFrameBanner = false
	SchedulerBindingDebugPrintTransientCallbackRegistrationStack()
	foundation.DebugPrint = previous
}

func TestSchedulerMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/scheduler.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	if !isolated(t) { // it constructs its own binding
		return
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	sLoop = async.NewLoop()
	defer async.SetCurrent(sLoop)()
	previousOnError := foundation.FlutterErrorOnError
	defer func() { foundation.FlutterErrorOnError = previousOnError }()
	foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) {
		slog("reported: " + d.ExceptionAsString() + " | " + libStr(d) + " | " + describeContext(d))
	}
	b := &schedBinding{}
	b.SchedulerBindingMixin.SetSelf(b)
	sBinding = b
	b.Construct(b, &b.SchedulerBindingMixin)
	slog("constructed locked=" + boolStr(b.Locked()) + " phase=" + sPhase() + " timeDilation=" + dart.DoubleToString(TimeDilation()))
	sceneFrameOrder()
	sceneEndOfFrame()
	sceneTickers()
	sceneTimeDilation()
	sceneTasks()
	sceneLifecycle()
	sceneWarmUp()
	sceneErrors()
	sceneTimings()
	scenePerformanceMode()
	sceneBanners()
	sLoop.FlushMicrotasks()
	slog("end timers=" + itoa(len(sLoop.PendingTimers())) + " transient=" + itoa(b.TransientCallbackCount()))

	got := sLog
	for i := 0; i < max(len(got), len(ref.Log)); i++ {
		var w, g string
		if i < len(ref.Log) {
			w = ref.Log[i]
		}
		if i < len(got) {
			g = got[i]
		}
		if w != g {
			lo := max(0, i-5)
			t.Fatalf("log differs at line %d:\nflutter: %q\ngo:      %q\ncontext (flutter):\n%s\ncontext (go):\n%s", i, w, g,
				strings.Join(ref.Log[lo:min(len(ref.Log), i+3)], "\n"), strings.Join(got[lo:min(len(got), i+3)], "\n"))
		}
	}
	t.Logf("%d log lines identical", len(got))
}
