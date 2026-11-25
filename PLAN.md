# flutter-go: staged validation plan

**Hypothesis:** Flutter's architecture can be written in Go on top of Impeller, with
(a) an API developers can live with, (b) layout identical to real Flutter, and
(c) frame times well within budget.

Each stage is small, ends with something runnable or testable, and has an exit
criterion. If a stage fails its criterion, stop and decide before going on.

| # | Stage | Validates | Exit criterion |
|---|---|---|---|
| 0 | Impeller from Go | cgo + Impeller C API + Metal work | Window shows a rounded rect and text ✅ |
| 1 | Render tree core | Flutter's layout protocol fits Go (no inheritance) | `rendering` package: constraints-down / sizes-up layout, paint to a canvas, hit testing; Box, Padding, Align, Flex, Paragraph; unit tests pass headless ✅ |
| 2 | Widget / Element core | Declarative rebuilds + `setState` work in Go | `widgets` package: stateless, stateful, render-object widgets; element reconciliation by type + key; dirty-only rebuilds; tests pass ✅ |
| 3 | Input + counter app | End-to-end interactivity | Clicking "+" in a real window increments the counter; a Go widget test (`Tester.Tap`) proves it headless ✅ (taps injected through the pointer router; real mouse not automated) |
| 4 | Differential layout tests | Our layout **equals** Flutter's | The same scenes built in Dart (`flutter test`) and in Go produce identical box geometry ✅ 11/11 scenes |
| 5 | Performance | Go is fast enough per frame | Build + layout + paint of 10k widgets measured; GC and cgo cost reported against the 16.6 ms budget ✅ median 4–5 ms; p99 at the budget edge, but below real Flutter's on the same scene |
| 6 | Decision gate | — | Findings below ✅ — recommendation: continue, with two conditions |
| 7 | Lazy scrolling list | Building during layout, element lifecycle off-screen, scroll input | `ListViewBuilder` (fixed item height) builds the same item indices as Flutter, with the same positions, after jumps, clamping, and inside a Column ✅ 6/6 scenes |
| 8 | Animations | Frame scheduling, tickers, curves, implicit animations, animated scroll | `AnimationController`, Flutter's cubic curves, `AnimatedBox` (≈ `AnimatedContainer`) and `ScrollController.AnimateTo` produce the same values as Flutter frame by frame ✅ 4/4 scenes, max difference 5.6e-17 |
| 9 | Fling physics | Ballistic scrolling after a drag, edge handling, interaction with taps | Flutter's `ClampingScrollSimulation` and `ClampingScrollPhysics` rules give the same offsets as Flutter every frame, and stop at the same frame ✅ 4/4 scenes, max difference 7e-12 px |
| 10 | Text input | Focus, keyboard shortcuts, selection, graphemes, cursor, clipboard | Editing behavior (text + selection after every key) matches Flutter's `EditableText` on 8 scenes, 81 steps, Android and macOS shortcut maps, both in the model and through the full `TextField` widget stack ✅ |

## Design rules

- Packages mirror Flutter's layers: `rendering` → `widgets` → `app`.
  Lower layers never import higher ones. Only `impeller/` and `platform/` use cgo.
- `rendering` draws to a `Canvas` interface, so tests run with a recording
  canvas and no GPU.
- Widgets are plain Go structs; the element type is chosen by which interface
  the widget implements (`Build`, `CreateState`, `CreateRenderObject`).
- Do not translate Dart line by line. Reproduce behavior, verified by stage 4.

## Findings (2026-11-25)

Measured on an Apple Silicon Mac (8 cores) with other apps running.

### What worked
- **Impeller is usable from Go with no engine build.** The prebuilt
  `impeller_sdk.zip` (11 MB dylib) is enough; cgo bindings are ~150 lines.
- **Flutter's architecture maps onto Go.** Widgets are plain structs; the
  element kind is picked by interface (`Build`, `CreateState`,
  `CreateRenderObject`, `ApplyParentData`). `StateOf[W]` (generics) gives
  typed `s.Widget()` and `s.SetState(...)`. Keys, state preservation,
  `didUpdateWidget`/`dispose`, parent data and dirty-only rebuilds all work.
- **Layout matches real Flutter exactly** on 11 scenes (Align, Center,
  Padding, SizedBox, Row/Column with every main/cross alignment, Expanded,
  Flexible, text wrapping with the test font). Tolerance 1e-6.
- **The differential test pays for itself.** It caught two bugs the unit
  tests missed: the root gave loose instead of tight constraints (Flutter's
  RenderView is tight), and the test font wrapped per glyph instead of per word.
- **Headless widget tests in Go** (`widgettest`) read like flutter_test:
  pump, find by text/type, tap, assert. The counter smoke test is ported.

### What hurt
- **No inheritance.** `RenderObject → RenderBox → RenderProxyBox` became
  "embed `Box` + implement an interface". Workable, but each proxy object
  repeats ~6 lines of child handling. Go also forbids an embedded field and a
  method with the same name (`Box`/`Box()`), so the accessor is `Base()`.
- **No optional or named parameters.** Flutter's nullable `double? width`
  became `*float64` with a helper: `SizedBox{Width: F(100)}`. Zero values
  must be designed as defaults (center alignment, `MainAxisSize.max`,
  `CrossAxisAlignment.center`), which worked but constrains API design.
- **No `const` widgets.** Flutter skips rebuilding identical const subtrees;
  Go has no identity for struct values, so every rebuild updates everything.
- **Allocation and GC.** A full rebuild of 20,303 elements allocates 1.2 MB
  (40k objects). About half comes from app `Build` code: every widget is a
  struct boxed in an interface. Go's GC is not generational, so this costs
  more than in Dart. Profiling showed ~35% of CPU in GC before optimizing.
- **cgo forces escapes.** Structs passed by pointer to C escape to the
  heap: ~20k extra allocations per frame for 10k draw calls.

### Performance (worst case: all 20,303 elements rebuild every frame)
| Mode | p50 | p90 | p99 | notes |
|---|---|---|---|---|
| repaint only | 4.0–4.3 ms | 7–11 ms | 13–23 ms | GOGC=100, 4 runs |
| relayout every frame | 4.9–5.0 ms | 8–11 ms | 16–17 ms | GOGC=100, 4 runs |
| repaint only, GC off | 4.0 ms | 5.5 ms | 11.6 ms | 1 run |

The median is stable and well inside the 16.6 ms budget. The p99 is not: it
moves between runs and gets worse when other processes compete for the CPU
(23 ms when run alongside the other test packages). With the GC off it is
still 11–13 ms, so the tail is part GC, part machine noise.

Benchmarks (framework only / with Impeller display-list recording):
3.1 ms / 4.6 ms per frame for repaint, 4.4 ms / 8.0 ms for relayout.
Removing per-element temporary slices and maps in the framework cut frame
time by ~45%; the remaining allocations are mostly inherent to the API.

### Go vs real Flutter (same grid, same machine, 2026-11-25)
Flutter 3.51 master (`06f1c24`), **profile mode**, macOS, Impeller/Metal,
measured with Flutter's own `FrameTiming`. Go numbers are the same run of
`scripts/perf_compare.sh`. Both columns are UI-thread work only
(build + layout + paint, no GPU rasterization).

| Mode | Elements | Go p50 / p90 / p99 | Flutter p50 / p90 / p99 |
|---|---|---|---|
| repaint every frame | 20,303 vs 20,415 | 4.1 / 7.1 / 15.8 ms | 9.9 / 14.2 / 25.7 ms |
| relayout every frame | 20,303 vs 20,415 | 4.9 / 8.2 / 14.6 ms | 15.1 / 19.6 / 28.3 ms |

Flutter's GPU raster thread took a further 9.5 ms p50 / 15.3 ms p99.

**Read this carefully: flutter-go does less work per frame.** Flutter also
builds a layer tree for compositing, runs a full `updateChildren` with
slot tracking, looks up `DefaultTextStyle`/`MediaQuery` for each `Text`,
and profile mode keeps timeline tracing on. flutter-go has none of that
yet. So the result is **"Go is not the bottleneck"**, not "Go is 2–3× faster
than Dart". A fair speed claim needs flutter-go to reach feature parity
for this scene (layers, repaint boundaries) and the comparison rerun.

What the comparison does settle: the worry that Go's GC makes this
architecture unworkable is not supported. Go's p99 is below Flutter's on
the same scene, and Flutter itself goes over the 16.6 ms budget here.
The grid is simply an extreme workload for both.

### Recommendation
Continue, with two conditions:
1. ~~**Add the missing comparison**~~ ✅ done: Go's UI-thread time is below
   Flutter's on the same grid, with the caveat that Go does less work.
   Rerun `scripts/perf_compare.sh` whenever layers/repaint boundaries land.
2. **Make the next stage the hard parts** (scrolling ✅ stage 7, animations ✅
   stage 8, text input ✅ stage 10), not more box widgets: scrolling
   (needs viewport + slivers), text input (IME), and implicit animations.
   These stress the frame loop and are where an API in Go will either hold
   up or not.

Gaps that remain: InheritedWidget, repaint boundaries/layers, relayout
boundaries, baseline alignment, RTL, accessibility, Retina scale detection
(the window reported scale 1.0), and every platform except macOS.

## Stage 7 findings: lazy scrolling (2026-11-25)

- **Building during layout works in Go.** The render object asks a
  `ChildManager` for items while it lays out; the element layer builds
  them on demand. This is the same inversion Flutter uses
  (`invokeLayoutCallback`), and it needed one new element kind, ~150 lines.
- **Matches Flutter on 6 list scenes**: same set of built indices (visible
  plus the 250 px cache on each side), same positions, same final offset,
  including fractional item heights, a list inside `Column`/`Expanded`, a
  list shorter than the viewport, and jumping past the end. Changing the
  cache extent to 200 px makes 5 of the 6 scenes fail, so the test really
  checks the laziness rule.
- **A behavior difference the test surfaced:** Flutter's `jumpTo` accepts
  out-of-range offsets and *animates* back with the scroll physics; flutter-go
  clamps immediately. The comparison uses Flutter's settled state. Scroll
  physics (fling, overscroll, the bounce on macOS) need an animation system,
  which flutter-go does not have yet.
- **Item state** is disposed when an item leaves the cache area and created
  again when it comes back, as in Flutter without keep-alive.
- **Drag** starts after an 18 px slop that is not applied to the scroll
  (Flutter's `DragStartBehavior.start`) and cancels taps. This is unit
  tested in Go but not yet compared with Flutter's gesture arena.
- **Cost is independent of item count:** ~1.4 µs per scroll frame (layout
  only, no paint) for both 1,000 and 1,000,000 items.

Not done: variable-height items (Flutter's `RenderSliverList`, which must
estimate extents), horizontal lists, slivers in general (`CustomScrollView`),
keep-alive, scrollbars, and scroll physics/animations.

## Stage 8 findings: animations (2026-11-25)

- **Values match Flutter frame by frame**, to floating-point precision, in
  4 scenes: an `AnimatedContainer` retargeted mid-animation, a
  `fastOutSlowIn` animation sampled at 16 ms (not a divisor of 250 ms), and
  two `animateTo` scrolls. This required copying Flutter's exact arithmetic:
  the cubic-curve bisection with its 0.001 error bound, `lerpDouble`'s
  `a*(1-t) + b*t` form, duration scaling rounded to whole microseconds.
- **Ticker start time has two rules**, and each has a scene that fails
  when it is broken: a ticker started *during* a frame starts at that
  frame's timestamp; one started *outside* a frame (a button handler)
  starts at its first tick. Getting the second wrong shifts every animation
  by one frame in a real app.
- **The first version of this test was too weak.** Both rules could be
  broken without any scene failing, because the Go tester applied
  `PumpWidget` outside a frame (Flutter applies it inside the next frame)
  and the scroll scene never let time pass before starting. Fixing the
  harness to defer root updates into the frame, as Flutter's
  `RootWidget.attach` does, made the tests discriminating. Lesson: run a
  mutation check on every new differential test.
- **Real window:** while animating, frames arrive every 16.7 ms (p95
  17.4 ms) on a 60 Hz display, paced by Metal presentation, with ~0.2 ms of
  CPU per frame. The loop no longer sleeps between frames while a ticker
  is active.
- **Another zero-value trap:** `Color{}` is transparent, so `Text` without
  a color was invisible. Unset text color now means opaque black.

Not done: bouncing overscroll physics (macOS/iOS), explicit-animation widgets (`AnimatedBuilder`,
transitions), `Hero`, opacity (needs `saveLayer`), and staggered/sequenced
animations.

## Stage 9 findings: fling physics (2026-11-25)

- **Ported from the fork's own source** (`scroll_simulation.dart`,
  `scroll_physics.dart`, `scroll_activity.dart`): the Android-derived
  `ClampingScrollSimulation`, the velocity tolerance `1/(0.05·dpr)`, "no
  fling toward an edge you are already at", and stopping when the
  simulation overscrolls. Offsets match Flutter to 7e-12 px over 70 frames
  (Dart's and Go's `pow` differ in the last bits).
- **Checking positions alone was not enough.** Removing the stop-at-edge
  rule kept every position identical (the offset is clamped anyway) while
  the animation kept running and burning frames. Recording whether a frame
  is still scheduled made that mutation fail.
- **That check surfaced two Flutter behaviors.** The overscroll glow keeps
  frames running after a fling hits the edge (decoration, turned off on the
  Flutter side since flutter-go has no glow). And Flutter puts an
  `IgnorePointer` over the list during scroll activities: **a touch on a
  flinging list only stops it, it does not tap the item underneath.**
  flutter-go now does the same. Flutter's extra frame to lift that
  `IgnorePointer` is allowed by the test rather than copied.
- **Release velocity is not Flutter-exact yet.** flutter-go estimates it
  with a straight line over the last 100 ms; Flutter's `VelocityTracker`
  uses a least-squares polynomial fit. The comparison therefore starts
  flings with fixed velocities (Flutter's `goBallistic`), and the gesture
  path is covered by Go tests only (fast release flings, slow release
  under 50 px/s does not).

Not done: `BouncingScrollPhysics` (the macOS/iOS default, with overscroll
and spring-back), Flutter's `VelocityTracker`, the overscroll glow, and
scrollbars.

## Stage 10 findings: text input (2026-11-25)

- **The Flutter reference was wrong at first, and only reading it caught
  that.** In a bare `EditableText`, no key did anything: Flutter's editing
  shortcuts come from `DefaultTextEditingShortcuts`, which `WidgetsApp`
  normally provides. Building Go to match that output would have "passed".
  Lesson: inspect a new reference before trusting it, not just the diff.
- **Ported from the fork's source,** not reimplemented from memory:
  `_DeleteTextAction`, `_UpdateTextSelectionAction` and the text boundaries
  (`CharacterBoundary`, `moveByWordBoundary` with its
  skip-spaces-and-punctuation rule). Details that would have been easy to
  get wrong: character moves collapse a selection first but word and line
  moves don't; a move landing on the base collapses; deleting a selection
  widens it to whole grapheme clusters; `_` joins words, so Ctrl+Backspace
  after `_new` deletes `brave _new`.
- **Offsets are UTF-16 code units, like Flutter**, although Go strings are
  UTF-8. Matching Flutter's unit made the port exact; conversion happens
  only at the edges (Impeller's glyph APIs also use UTF-16).
- **Grapheme clusters** (via `rivo/uniseg`): Backspace removes 👍🏽 or 🇧🇷 whole
  and steps over `é` written as e + combining accent.
- **Results:** 8 scenes, 81 steps, identical text and selection after every
  step, for the editing model alone and through `TextField` + focus +
  tester. Includes a macOS scene (Alt for words, Cmd for line ends and
  select all, Home/End don't move the caret). Three mutations (code points
  instead of graphemes, no punctuation skipping, word moves collapsing a
  selection) each fail a scene; the last needed a new scene to be caught.
- **Cursor blink uses timers, not frames,** as Flutter's does: an idle
  focused field draws two frames a second, not sixty. The window loop now
  sleeps until the next timer.
- **The window needed GLFW's refresh callback** so the system can request a
  redraw (first show, uncovering). The old loop's 0.25 s polling had hidden
  that it was missing.
- **Not verified visually in this stage:** window screenshots stopped
  working on this machine midway (a build from the previous commit fails
  the same way), so the text field demo was checked by tests and by
  running it, not by a screenshot.

Not done: IME composition display. GLFW delivers committed text (so
Chinese/Japanese input works once committed) but has no preedit support;
showing the composing underline needs a native `NSTextInputClient` view.
Also: multi-line fields, mouse drag-selection, double-click word select,
selection handles/toolbar, undo/redo, password obscuring, and
accessibility. Tab focus traversal is not implemented.

## G1 findings: InheritedWidget (2026-11-25)

- **`InheritedWidget` with dependency tracking**, as in Flutter: each element
  carries a map from inherited type to the providing ancestor (copied only
  where a provider adds itself), `DependOn[T](ctx)` registers a dependency,
  and a provider rebuilt with `UpdateShouldNotify` true marks exactly its
  dependents dirty, even when everything in between is skipped.
  `MediaQuery` (view size and pixel ratio, updated on resize) and
  `DefaultTextStyle` are built on it; `Text` is now a stateless widget that
  merges the inherited style, like Flutter's.
- **Rebuilds match Flutter step by step** (which widgets build, in order)
  for a value change, a parent rebuild without a change, and re-pumping the
  same subtree. Three mutations (always notify, never notify, no rebuild
  skipping) each fail.
- **Go's stand-in for `const`: value equality.** Flutter skips a subtree when
  the new widget is *identical* to the old one, which const widgets and the
  `widget.child` pattern produce. Go structs have no identity, but
  comparable ones have `==`, so flutter-go skips a child whose widget is
  equal to the previous one. Widgets holding slices, maps or funcs can't be
  compared and always update. This **diverges from Flutter on purpose**:
  a *new but equal* subtree rebuilds in Flutter and doesn't in flutter-go;
  the test asserts that divergence explicitly. It relies on the same
  contract Flutter's identity check does: `Build` depends only on the
  widget's fields and on inherited widgets.
- **The test found a framework bug:** the allocation-free fast path in list
  reconciliation called `update` directly and bypassed the skip.
- **Effect on performance:** the grid benchmark went from 3.1 ms to 2.2 ms
  per frame (framework only), because unchanged widgets such as the row
  labels no longer rebuild. This answers part of the earlier finding
  "No const widgets".
- **Another zero-value limit:** `TextStyle.Merge` can't let a child turn
  `Bold` off, because Go has no "unset" for `bool`.

