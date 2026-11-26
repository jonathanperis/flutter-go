package difftest

// Go twin of flutter/test/animation_controller_test.dart: the same scenes,
// step for step, through flutter_test's pump. Every line (values in Dart's
// shortest round-trip form, so bit for bit) must be identical.

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/animation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

var acLines []string

func acLog(s string) { acLines = append(acLines, s) }

var hashRe = regexp.MustCompile(`#[0-9a-f]{5}`)

func mask(o interface{ String() string }) string {
	if reflect.ValueOf(o).Kind() == reflect.Pointer && reflect.ValueOf(o).IsNil() {
		return "null"
	}
	return hashRe.ReplaceAllString(o.String(), "#HASH")
}

func ds(v float64) string { return dart.DoubleToString(v) }

func acState(c *AnimationController) string {
	elapsed := "null"
	if e := c.LastElapsedDuration(); e != nil {
		elapsed = strconv.FormatInt(e.Microseconds(), 10)
	}
	return "value=" + ds(c.Value()) + " status=" + c.Status().Name() + " animating=" +
		strconv.FormatBool(c.IsAnimating()) + " velocity=" + ds(c.Velocity()) + " elapsed=" + elapsed
}

func watch(name string, c *AnimationController) {
	c.AddListener(func() { acLog(name + " tick " + ds(c.Value()) + " " + c.Status().Name()) })
	c.AddStatusListener(func(s AnimationStatus) { acLog(name + " status " + s.Name() + " value=" + ds(c.Value())) })
}

func future(name string, f *scheduler.TickerFuture) {
	async.OnComplete[async.Void](f, func(async.Void) { acLog(name + " future complete") }, nil)
	async.OnComplete(f.OrCancel(), func(async.Void) {},
		func(e error) { acLog(name + " orCancel " + reflect.TypeOf(e).Name()) })
}

func steps(tester *flutter_test.Tester, c *AnimationController, us []int) {
	for _, step := range us {
		tester.Pump(time.Duration(step) * time.Microsecond)
		acLog("pump " + strconv.Itoa(step) + " " + acState(c))
	}
}

type testVSync struct{}

func (testVSync) CreateTicker(onTick scheduler.TickerCallback) *scheduler.Ticker {
	return scheduler.NewTicker(onTick)
}

type mutedVSync struct{}

func (mutedVSync) CreateTicker(onTick scheduler.TickerCallback) *scheduler.Ticker {
	t := scheduler.NewTicker(onTick)
	t.SetMuted(true)
	return t
}

var frame60 = []int{0, 16667, 16666, 16667, 16667, 16666, 16667, 16667, 16666, 16667, 16667, 16666, 16667,
	16667, 16666, 16667, 16667, 16666, 16667, 16667, 16666, 16667, 16667}

func pd(d time.Duration) *time.Duration { return &d }
func pf(v float64) *float64             { return &v }
func pi(v int) *int                     { return &v }

var vs = testVSync{}

var acScenes = map[string]func(tester *flutter_test.Tester){
	"forward_reverse": func(tester *flutter_test.Tester) {
		c := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(300 * time.Millisecond),
			ReverseDuration: pd(200 * time.Millisecond), DebugLabel: "fr"})
		watch("c", c)
		acLog("initial " + acState(c) + " " + mask(c))
		future("forward", c.Forward())
		acLog("after forward " + acState(c) + " " + mask(c))
		steps(tester, c, []int{0, 16667, 16666, 33333, 50001, 99999})
		acLog("mid " + mask(c))
		future("reverse", c.Reverse())
		steps(tester, c, []int{0, 17000, 23456, 40000, 81000, 1, 2})
		future("forward from", c.Forward(ForwardOptions{From: pf(0.25)}))
		steps(tester, c, []int{0, 70000, 13})
		future("toggle", c.Toggle())
		steps(tester, c, frame60)
		future("toggle from", c.Toggle(ToggleOptions{From: pf(0.6)}))
		steps(tester, c, frame60)
		future("reverse from", c.Reverse(ReverseOptions{From: pf(0.9)}))
		steps(tester, c, []int{0, 100000, 100000, 100000})
		acLog("end " + mask(c))
		c.Dispose()
		acLog("disposed " + mask(c))
	},
	"animate_to_curves": func(tester *flutter_test.Tester) {
		c := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(400 * time.Millisecond)})
		watch("c", c)
		future("a", c.AnimateTo(0.7, AnimateToOptions{Curve: Curves.EaseInOut}))
		steps(tester, c, frame60)
		future("b", c.AnimateTo(0.2, AnimateToOptions{Duration: pd(250 * time.Millisecond), Curve: Curves.FastOutSlowIn}))
		steps(tester, c, frame60)
		future("c", c.AnimateTo(0.9, AnimateToOptions{Curve: Curves.BounceOut}))
		steps(tester, c, []int{0, 5000, 50000, 77777, 100000, 200000, 300000})
		future("d", c.AnimateBack(0.1, AnimateBackOptions{Curve: Curves.ElasticIn}))
		steps(tester, c, frame60)
		future("e", c.AnimateBack(0.6, AnimateBackOptions{Duration: pd(123 * time.Millisecond),
			Curve: NewInterval(0.2, 0.8, IntervalOptions{Curve: Curves.EaseIn})}))
		steps(tester, c, []int{0, 30000, 30000, 30000, 30000, 30000})
		future("same", c.AnimateTo(c.Value(), AnimateToOptions{Duration: pd(100 * time.Millisecond)}))
		acLog("same " + acState(c))
		future("zero", c.AnimateTo(0.33, AnimateToOptions{Duration: pd(0)}))
		acLog("zero " + acState(c) + " transient=" + strconv.Itoa(scheduler.SchedulerBindingInstance().TransientCallbackCount()))
		future("interrupted", c.AnimateTo(1.0, AnimateToOptions{Curve: Curves.EaseOutBack}))
		steps(tester, c, []int{0, 40000, 40000})
		future("interrupter", c.AnimateTo(0.0, AnimateToOptions{Curve: Curves.EaseInCirc}))
		steps(tester, c, frame60)
		steps(tester, c, frame60)
		c.Dispose()
	},
	"repeat": func(tester *flutter_test.Tester) {
		c := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(170 * time.Millisecond)})
		watch("c", c)
		future("r1", c.Repeat(RepeatOptions{Reverse: true}))
		steps(tester, c, frame60)
		c.Stop()
		acLog("stopped " + acState(c))
		future("r2", c.Repeat(RepeatOptions{Min: pf(0.2), Max: pf(0.8), Count: pi(3), Period: pd(90 * time.Millisecond)}))
		steps(tester, c, frame60)
		future("r3", c.Repeat(RepeatOptions{Reverse: true, Count: pi(2), Min: pf(0.1), Max: pf(0.5)}))
		steps(tester, c, frame60)
		c.SetValue(0.65)
		future("r4", c.Repeat(RepeatOptions{Reverse: true, Count: pi(5), Period: pd(33333 * time.Microsecond)}))
		steps(tester, c, frame60)
		future("r5", c.Repeat(RepeatOptions{Min: pf(0.5), Max: pf(0.5)}))
		steps(tester, c, []int{0, 10000, 10000})
		c.Reset()
		acLog("reset " + acState(c))
		big := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(100 * time.Millisecond),
			LowerBound: pf(-3), UpperBound: pf(7), Value: pf(2)})
		watch("big", big)
		future("big", big.Repeat(RepeatOptions{Min: pf(-1), Max: pf(5), Reverse: true, Count: pi(3)}))
		steps(tester, big, frame60)
		c.Dispose()
		big.Dispose()
	},
	"fling": func(tester *flutter_test.Tester) {
		c := NewAnimationController(vs)
		watch("c", c)
		future("f1", c.Fling())
		steps(tester, c, frame60)
		future("f2", c.Fling(FlingOptions{Velocity: pf(-2.5)}))
		steps(tester, c, frame60)
		spring := physics.NewSpringDescriptionWithDampingRatio(0.5, 120.0, 1.4)
		future("f3", c.Fling(FlingOptions{Velocity: pf(0.3), SpringDescription: &spring}))
		steps(tester, c, frame60)
		steps(tester, c, frame60)
		large := NewAnimationController(vs, AnimationControllerOptions{LowerBound: pf(-30), UpperBound: pf(45)})
		watch("large", large)
		future("large", large.Fling(FlingOptions{Velocity: pf(40)}))
		steps(tester, large, frame60)
		future("large back", large.Fling(FlingOptions{Velocity: pf(-1)}))
		steps(tester, large, frame60)
		steps(tester, large, frame60)
		c.Dispose()
		large.Dispose()
	},
	"simulations": func(tester *flutter_test.Tester) {
		c := NewAnimationController(vs, AnimationControllerOptions{LowerBound: pf(-100), UpperBound: pf(400)})
		watch("c", c)
		future("gravity", c.AnimateWith(physics.NewGravitySimulation(500.0, 0.0, 300.0, 10.0)))
		steps(tester, c, frame60)
		future("spring back", c.AnimateBackWith(physics.NewSpringSimulation(
			physics.SpringDescription{Mass: 1, Stiffness: 180, Damping: 12}, 300, -20, -50)))
		steps(tester, c, frame60)
		steps(tester, c, frame60)
		future("friction", c.AnimateWith(physics.NewFrictionSimulation(0.135, 100, 900)))
		steps(tester, c, frame60)
		steps(tester, c, []int{1000000, 1000000})
		u := NewAnimationControllerUnbounded(vs, AnimationControllerUnboundedOptions{Value: 50})
		watch("u", u)
		acLog("unbounded " + acState(u) + " behavior=" + u.AnimationBehavior.Name() + " " + mask(u))
		future("u to", u.AnimateTo(2000, AnimateToOptions{Duration: pd(400 * time.Millisecond), Curve: Curves.EaseOut}))
		steps(tester, u, frame60)
		future("u spring", u.AnimateWith(physics.NewSpringSimulation(
			physics.NewSpringDescriptionWithDampingRatio(0.5, 100, 1.1), u.Value(), -200, 1500)))
		steps(tester, u, frame60)
		steps(tester, u, frame60)
		future("u back", u.AnimateBack(-1e6, AnimateBackOptions{Duration: pd(50 * time.Millisecond)}))
		steps(tester, u, []int{0, 20000, 20000, 20000})
		c.Dispose()
		u.Dispose()
	},
	"stop_reset_resync": func(tester *flutter_test.Tester) {
		c := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(500 * time.Millisecond), DebugLabel: "srr"})
		watch("c", c)
		future("f", c.Forward())
		steps(tester, c, []int{0, 50000, 50000})
		canceled := false
		c.Stop(StopOptions{Canceled: &canceled})
		acLog("stop canceled false " + acState(c))
		steps(tester, c, []int{50000})
		future("g", c.Forward())
		steps(tester, c, []int{0, 50000})
		c.SetValue(0.8)
		acLog("set value " + acState(c))
		c.SetValue(2.0)
		acLog("set value clamped " + acState(c))
		future("h", c.Reverse())
		steps(tester, c, []int{0, 60000})
		c.Resync(testVSync{})
		acLog("resync " + acState(c))
		steps(tester, c, []int{60000, 60000})
		c.Resync(mutedVSync{})
		acLog("muted " + mask(c))
		steps(tester, c, []int{60000, 60000, 400000})
		acLog("after muted " + acState(c) + " " + mask(c))
		c.Reset()
		acLog("reset " + acState(c))
		future("i", c.Forward())
		steps(tester, c, []int{0, 10000})
		c.Dispose()
		acLog("disposed " + mask(c))
		tester.Pump(100 * time.Millisecond)
		acLog("after dispose pump")
	},
	"time_dilation": func(tester *flutter_test.Tester) {
		c := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(300 * time.Millisecond)})
		watch("c", c)
		scheduler.SetTimeDilation(3.0)
		future("f", c.Forward())
		steps(tester, c, frame60)
		scheduler.SetTimeDilation(0.37)
		future("r", c.Reverse())
		steps(tester, c, []int{0, 10001, 10001, 10001, 10001, 10001, 10001, 10001, 10001})
		scheduler.SetTimeDilation(1.0)
		steps(tester, c, frame60)
		c.Dispose()
	},
	"behavior": func(tester *flutter_test.Tester) {
		previous := SemanticsDisableAnimations
		SemanticsDisableAnimations = func() bool { return true }
		normal := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(1000 * time.Millisecond)})
		preserve := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(1000 * time.Millisecond),
			AnimationBehavior: AnimationBehaviorPreserve})
		watch("n", normal)
		watch("p", preserve)
		future("n", normal.Forward())
		future("p", preserve.Forward())
		steps(tester, normal, []int{0, 16667, 16667, 16667, 16667})
		acLog("preserve " + acState(preserve))
		future("nf", normal.Fling(FlingOptions{Velocity: pf(-1)}))
		normalBehavior := AnimationBehaviorNormal
		future("pf", preserve.Fling(FlingOptions{Velocity: pf(-1), AnimationBehavior: &normalBehavior}))
		steps(tester, normal, []int{0, 16667, 16667, 16667})
		acLog("preserve " + acState(preserve))
		future("nr", normal.Repeat(RepeatOptions{Reverse: true}))
		steps(tester, normal, []int{0, 16667, 16667, 16667})
		normal.Stop()
		SemanticsDisableAnimations = previous
		future("n2", normal.Forward(ForwardOptions{From: pf(0)}))
		steps(tester, normal, []int{0, 16667, 16667})
		normal.Dispose()
		preserve.Dispose()
	},
	"curved": func(tester *flutter_test.Tester) {
		c := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(200 * time.Millisecond),
			ReverseDuration: pd(150 * time.Millisecond)})
		curved := NewCurvedAnimation(c, Curves.EaseIn, CurvedAnimationOptions{ReverseCurve: Curves.ElasticOut})
		plain := NewCurvedAnimation(c, Curves.Decelerate)
		c.AddListener(func() {
			acLog("curved " + ds(curved.Value()) + " plain " + ds(plain.Value()) + " " + mask(curved))
		})
		acLog("initial " + mask(curved) + " " + mask(plain))
		c.Forward()
		steps(tester, c, []int{0, 16667, 16667, 16667, 16667})
		c.Reverse()
		steps(tester, c, []int{0, 16667, 16667})
		c.Forward()
		steps(tester, c, []int{0, 16667, 16667})
		steps(tester, c, frame60)
		c.Reverse()
		steps(tester, c, frame60)
		curved.ReverseCurve = nil
		curved.Curve = Curves.SlowMiddle
		c.Forward()
		steps(tester, c, []int{0, 50000})
		curved.Dispose()
		plain.Dispose()
		c.Dispose()
	},
	"train_hopping": func(tester *flutter_test.Tester) {
		a := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(300 * time.Millisecond), Value: pf(0.6)})
		b := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(200 * time.Millisecond), Value: pf(0.1)})
		train := NewTrainHoppingAnimation(a, b, TrainHoppingAnimationOptions{OnSwitchedTrain: func() { acLog("switched") }})
		train.AddListener(func() { acLog("train " + ds(train.Value()) + " " + train.Status().Name()) })
		train.AddStatusListener(func(s AnimationStatus) { acLog("train status " + s.Name()) })
		acLog("initial " + mask(train))
		a.Reverse()
		b.Forward()
		steps(tester, a, frame60)
		acLog("after " + mask(train) + " current=" + strconv.FormatBool(train.CurrentTrain() == Animation[float64](b)))
		same := NewTrainHoppingAnimation(a, NewAnimationController(vs, AnimationControllerOptions{Value: pf(a.Value())}))
		acLog("same " + mask(same))
		none := NewTrainHoppingAnimation(b, nil)
		acLog("none " + mask(none))
		same.Dispose()
		none.Dispose()
		train.Dispose()
		current := "null"
		if train.CurrentTrain() != nil {
			current = train.CurrentTrain().String()
		}
		acLog("disposed " + current)
		a.Dispose()
		b.Dispose()
	},
	"proxy_reverse_compound": func(tester *flutter_test.Tester) {
		a := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(100 * time.Millisecond)})
		b := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(160 * time.Millisecond), Value: pf(0.75)})
		proxy := NewProxyAnimation()
		acLog("proxy " + mask(proxy) + " " + ds(proxy.Value()) + " " + proxy.Status().Name())
		pl := func() { acLog("proxy tick " + ds(proxy.Value())) }
		proxy.AddListener(pl)
		proxy.AddStatusListener(func(s AnimationStatus) { acLog("proxy status " + s.Name()) })
		proxy.SetParent(a)
		proxy.SetParent(b)
		acLog("proxy " + mask(proxy))
		reverse := NewReverseAnimation(proxy)
		reverse.AddListener(func() { acLog("reverse tick " + ds(reverse.Value())) })
		reverse.AddStatusListener(func(s AnimationStatus) { acLog("reverse status " + s.Name()) })
		acLog("reverse " + mask(reverse) + " " + reverse.Status().Name())
		mean := NewAnimationMean(a, b)
		mx := NewAnimationMax[float64](a, b)
		mn := NewAnimationMin[float64](a, b)
		mean.AddListener(func() { acLog("mean " + ds(mean.Value())) })
		mx.AddListener(func() { acLog("max " + ds(mx.Value())) })
		mn.AddListener(func() { acLog("min " + ds(mn.Value())) })
		mean.AddStatusListener(func(s AnimationStatus) { acLog("mean status " + s.Name()) })
		mx.AddStatusListener(func(s AnimationStatus) { acLog("max status " + s.Name()) })
		acLog("compound " + mask(mean) + " " + mask(mx) + " " + mask(mn) + " " + mean.Status().Name())
		a.Forward()
		b.Reverse()
		steps(tester, a, []int{0, 16667, 16667, 16667, 16667, 16667, 16667, 16667, 16667, 16667, 16667, 16667})
		proxy.SetParent(a)
		proxy.SetParent(nil)
		acLog("proxy null " + mask(proxy) + " " + ds(proxy.Value()) + " " + proxy.Status().Name())
		proxy.RemoveListener(pl)
		a.Reverse()
		steps(tester, a, []int{0, 30000, 30000, 30000, 30000})
		acLog("stopped " + mask(KAlwaysCompleteAnimation) + " " + mask(KAlwaysDismissedAnimation) + " " +
			mask(NewAlwaysStoppedAnimation(0.25)) + " " + mask(NewAlwaysStoppedAnimation(3)))
		a.Dispose()
		b.Dispose()
	},
	"edge_cases": func(tester *flutter_test.Tester) {
		c := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(1000 * time.Microsecond), Value: pf(1.0 / 3)})
		watch("c", c)
		future("third", c.Forward())
		steps(tester, c, []int{0, 333, 333, 1, 1})
		u := NewAnimationControllerUnbounded(vs, AnimationControllerUnboundedOptions{Duration: pd(300 * time.Millisecond)})
		watch("u", u)
		future("u", u.AnimateTo(100))
		steps(tester, u, []int{0, 100000, 100000, 100000, 1})
		r := NewAnimationController(vs, AnimationControllerOptions{Duration: pd(100 * time.Millisecond)})
		watch("r", r)
		future("r", r.Repeat(RepeatOptions{Count: pi(1)}))
		steps(tester, r, []int{0, 50000, 50000, 1})
		r.SetValue(0.5)
		future("f0", r.Fling(FlingOptions{Velocity: pf(0.0)}))
		steps(tester, r, []int{0, 16667, 16667, 16667})
		a := NewAnimationController(vs, AnimationControllerOptions{Value: pf(0.5)})
		b := NewAnimationController(vs, AnimationControllerOptions{Value: pf(0.25)})
		up := NewTrainHoppingAnimation(a, b, TrainHoppingAnimationOptions{OnSwitchedTrain: func() { acLog("up switched") }})
		up.AddListener(func() { acLog("up " + ds(up.Value())) })
		up.AddStatusListener(func(s AnimationStatus) { acLog("up status " + s.Name()) })
		a.SetValue(0.6)
		b.SetValue(0.6)
		acLog("up " + mask(up) + " current=" + strconv.FormatBool(up.CurrentTrain() == Animation[float64](b)))
		d := NewAnimationController(vs, AnimationControllerOptions{Value: pf(0.7)})
		e := NewAnimationController(vs, AnimationControllerOptions{Value: pf(0.2)})
		down := NewTrainHoppingAnimation(e, d, TrainHoppingAnimationOptions{OnSwitchedTrain: func() { acLog("down switched") }})
		down.AddListener(func() { acLog("down " + ds(down.Value())) })
		d.SetValue(0.2)
		acLog("down " + mask(down) + " current=" + strconv.FormatBool(down.CurrentTrain() == Animation[float64](d)))
		f := NewAnimationController(vs, AnimationControllerOptions{Value: pf(0.4)})
		g := NewAnimationController(vs, AnimationControllerOptions{Value: pf(0.4)})
		same := NewTrainHoppingAnimation(f, g)
		acLog("same current=" + strconv.FormatBool(same.CurrentTrain() == Animation[float64](g)))
		g.SetValue(0.9)
		acLog("same " + ds(same.Value()))
		zero := 0.0
		specials := []float64{0.0, math.Copysign(0, -1), zero / zero, 1.0}
		for _, x := range specials {
			for _, y := range specials {
				acLog("max/min " + ds(x) + " " + ds(y) + " " +
					ds(NewAnimationMax[float64](NewAlwaysStoppedAnimation(x), NewAlwaysStoppedAnimation(y)).Value()) + " " +
					ds(NewAnimationMin[float64](NewAlwaysStoppedAnimation(x), NewAlwaysStoppedAnimation(y)).Value()))
			}
		}
		acLog("ints " + strconv.Itoa(NewAnimationMax[int](NewAlwaysStoppedAnimation(3), NewAlwaysStoppedAnimation(-4)).Value()) +
			" " + strconv.Itoa(NewAnimationMin[int](NewAlwaysStoppedAnimation(3), NewAlwaysStoppedAnimation(-4)).Value()))
		up.Dispose()
		down.Dispose()
		same.Dispose()
		for _, x := range []*AnimationController{c, u, r, a, b, d, e, f, g} {
			x.Dispose()
		}
	},
}

func TestAnimationControllerMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/animation_controller.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string][]string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(acScenes) {
		t.Fatalf("flutter has %d scenes, go %d", len(want), len(acScenes))
	}
	for name, run := range acScenes {
		t.Run(name, func(t *testing.T) {
			ref, ok := want[name]
			if !ok {
				t.Fatal("scene missing from the reference")
			}
			acLines = nil
			tester := flutter_test.New(t)
			run(tester)
			tester.Loop().FlushMicrotasks() // the end of the async test body
			compareLines(t, ref, acLines)
		})
	}
}

// compareLines reports the first difference between two logs.
func compareLines(t *testing.T, want, got []string) {
	t.Helper()
	for i := 0; i < max(len(want), len(got)); i++ {
		var w, g string
		if i < len(want) {
			w = want[i]
		}
		if i < len(got) {
			g = got[i]
		}
		if w != g {
			lo := max(0, i-3)
			t.Fatalf("line %d differs (flutter %d lines, go %d)\nflutter: %q\ngo:      %q\ncontext (flutter): %q",
				i, len(want), len(got), w, g, want[lo:min(len(want), i+1)])
		}
	}
	t.Logf("%d lines identical", len(want))
}
