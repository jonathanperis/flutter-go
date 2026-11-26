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
- **(Fixed in G3.) Release velocity was not Flutter-exact.** flutter-go estimates it
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

## G2 findings: repaint boundaries and layers (2026-11-25)

- **Retained layers like Flutter's:** a repaint boundary records its subtree
  into a layer made of picture segments and references to child
  boundaries (with the clips and translation active at each). Compositing
  reuses every clean boundary. `markNeedsPaint` marks the nearest boundary
  and stops early if it's already dirty; relayout marks paint, as Flutter's
  `layout()` does. The root view and every list item are boundaries
  (Flutter's `addRepaintBoundaries`), and lists paint only items that
  overlap the visible area, not the cache extent.
- **Paint counts match Flutter frame by frame** in 4 scenes: a repainting
  sibling inside a boundary (only it repaints), the same without a
  boundary (both repaint), nested boundaries (each repaints alone), and
  scrolling a list (only items becoming visible for the first time paint;
  moving painted items costs no repaint). Four mutations each fail.
- **Go vs Flutter rerun** (same grid, profile mode, same machine), now that
  flutter-go also builds a layer tree:

  | Mode | flutter-go p50 / p90 / p99 | Flutter p50 / p90 / p99 |
  |---|---|---|
  | repaint every frame | 4.5 / 5.8 / 7.3 ms | 9.0 / 11.0 / 13.5 ms |
  | relayout every frame | 5.5 / 8.9 / 14.1 ms | 12.3 / 18.0 / 20.6 ms |

  flutter-go still does less per frame than Flutter (no semantics, simpler
  reconciliation), so this remains "Go is not the bottleneck" rather than
  a speed claim; the gap narrowed less than expected because layer
  bookkeeping is cheap next to building 20,000 widgets.
- **A benchmark scare that was mostly noise:** one run suggested layers
  slowed the framework from 2.2 to 3.5 ms per frame. Profiling found a
  real but small cost (`MarkNeedsPaint` walked to the root twice per
  call, 7.5% of time; fixed with Flutter's early exit). Repeated runs then
  gave 2.3–3.2 ms. Single benchmark runs on this machine vary by ±30%.
- **Scrolling** costs ~4 µs per frame (layout + compositing the root layer),
  up from ~1.4 µs without layers, still independent of item count.

## G3 findings: drag velocity and the gesture arena (2026-11-25)

- **Ported Flutter's `VelocityTracker` and `LeastSquaresSolver`**: 20-sample
  ring buffer, 100 ms horizon, "pointer stopped" after 40 ms (measured on a
  separate sampling clock, as Flutter's stopwatch is), quadratic
  Gram–Schmidt fit, plus the vertical recognizer's fling rule (|v| > 50 px/s
  *and* > 18 px covered by the samples, clamped to 8000 px/s).
- **Ported flutter_test's `fling`** event by event (50 moves with synthetic
  timestamps, a frame pumped per 16 ms of event time) and a timestamped
  `DragTimed`, so both sides receive identical input.
- **8 drag scenes match Flutter frame by frame** (≤ 3e-11 px): flings up
  and down at several speeds, too slow to fling, a slow drag that fails the
  minimum-distance rule, the 8000 px/s clamp, an accelerating drag, and a
  list whose items compete for the tap. Five mutations each fail.
- **The comparison exposed a wrong assumption from stage 7:** a drag on a
  list whose items have no tap handlers scrolls the *full* distance. Flutter's
  gesture arena resolves a single-member arena immediately, so the drag is
  accepted on pointer-down; only when a tap competes does it wait for the
  18 px slop, and then (DragStartBehavior.start) it starts where it was
  accepted, not at start + 18 px. Both rules are now ported, and the
  stage-7 unit test's expectation (`200 − 18`) was wrong; it is now 180.
- **A scene that tested nothing:** the first "tappable items" scene used a
  `SizedBox` child, which isn't hit-testable in either framework, so the
  tap detector never joined the arena. Switching to `ColoredBox` made it
  test the competing path (Flutter releases at 276 px = 300 − 24).
- **A flaw in my own mutation harness:** one mutation left a variable
  unused, the package didn't compile, and my filter (which looked only for
  failing test lines) reported it as "not caught". Build failures now count.

## G4 findings: bouncing physics and scroll positions (2026-11-25)

- **Scrolling now goes through physics, like Flutter's `ScrollPosition`**,
  instead of clamping inside the render object. A `ScrollController` owns
  the position and its physics (picked by platform: clamping on Android,
  bouncing on iOS, bouncing with "fast" deceleration on macOS): boundary
  conditions on `setPixels`, rubber-band friction on user drags while
  overscrolled, iOS's 3.5 px drag-start threshold, ballistic activities that
  stop when the physics refuse a position, and settling (`goBallistic(0)`)
  after every activity ends. The mouse wheel always clamps.
- **Ported simulations:** `FrictionSimulation` (including its constant
  deceleration and the 10-iteration Newton solve, whose non-converging case
  is reproduced on purpose), all three spring solutions,
  `ScrollSpringSimulation` and `BouncingScrollSimulation`.
- **8 scenes match Flutter on iOS and macOS** (≤ 3e-13 px): flings within
  range and overshooting either end, an out-of-range jump rubber-banding
  back, and drags pulled past the top (every move, then the spring back).
  Five mutations each fail.
- **One mismatch, then a missed override:** only the macOS overshoot
  differed, and only after the spring took over. macOS's "fast" bouncing
  physics replaces the spring (mass 0.3, stiffness 75, ratio 1.3); the iOS
  spring had been used for both.
- **`jumpTo` now matches Flutter:** an out-of-range jump lands there and
  animates back (on clamping physics too, with Flutter's spring). Stage 7's
  list scenes now settle on both sides, and the immediate-clamp divergence
  noted in stage 7 is gone.
- **An ordering detail that matters:** Flutter applies a simulation's final
  value before firing "completed" (which triggers the settle). The Go
  controller fired the status first; fixed.
- **Not ported:** iOS's and macOS's own velocity trackers
  (`IOSScrollViewFlingVelocityTracker`, `MacOSScrollViewFlingVelocityTracker`;
  the scenes release with zero velocity so all trackers agree), carried
  momentum between consecutive flings, overscroll glow, and scrollbars.


## Goal: Flutter 3.50.1 with Go in place of Dart (from 2026-11-25)

The standing goal is now full API parity with Flutter 3.50.1's Dart layer
(see GOAL.md). Progress is measured by `docs/PARITY.md`.

### P0: pin and measure
- **Pinned to the 3.50.1 release tag** (ae9b586; engine 4d35714). The
  local clone had drifted 12 commits past the old pin, which HANDOFF.md
  wrongly described as matching. All 55 difftest scenes, regenerated with
  Flutter 3.50.1, still match.
- **upstream/** holds the 9,677 non-Dart files of 3.50.1. Three of them are
  stored upstream with CRLF line endings, and `git add` would have
  normalized them under upstream's own `* text=auto`, changing their blob
  hashes. The sync script stages upstream's exact blobs; `--check`
  compares hashes, not just file contents.
- **Size of the job:** 2,792 public top-level declarations at 3.50.1
  (widgets 949, material 524, rendering 312, dart:ui 195), about 321k lines
  of Dart excluding comments. flutter-go exposed 46 before this goal.

### physics (P2, done early as the first fully ported library)
- All 12 declarations, ported from 3.50.1 (identical to master for this
  library). The Dart→Go rules in docs/API_MAPPING.md were fixed while doing
  it: `Simulation` is an interface plus an embeddable `SimulationBase` for
  the mutable tolerance; optional named arguments are a variadic options
  struct with pointers where "unset" differs from zero.
- **Flutter's own tests ported** (all of test/physics/, except
  `nearEqual`'s null cases, a documented divergence, and the
  `BouncingScrollSimulation` cases, which belong to widgets).
- **Difftest:** 15 scenes sampled every 50 ms for 7 s plus `finalX`,
  `timeAtX`, spring type and `toString`; max difference 4.5e-13.
- **The difftest was needed:** Flutter's tests use loose tolerances, so two
  key rules survived mutation with them alone: the 10-iteration Newton solve
  for `finalTime` (9 passed) and `BoundedFrictionSimulation`'s lower bound.
  Both are caught now. Also found: `ScrollSpringSimulation.toString` prints
  its own type name (debug-mode `objectRuntimeType`).
- **Not caught, by design:** `math.Exp(x)` instead of Dart's
  `pow(e, x)` differs only in the last bits, inside the comparison's 1e-12
  relative tolerance. Go's `math.Pow` is not the C library's `pow` that
  Dart uses, so exact bit equality is not a goal.

### P0: scroll simulations move to widgets; `ui` and `internal/dart` start
- `ClampingScrollSimulation` and `BouncingScrollSimulation` now live in
  `widgets` (where Flutter declares them) on top of `physics`;
  `animation/simulation.go` is gone and `Controller.AnimateWith` takes a
  `physics.Simulation`. All existing fling, drag and bouncing scenes still
  match.
- **Flutter's own test caught a real bug:** a clamping fling with zero
  velocity has zero duration, so `time / duration` is NaN. Flutter's
  `clampDouble` maps NaN to the upper bound; the Go code used
  `math.Max(0, math.Min(1, …))`, which returns NaN. Fixed by porting
  `clampDouble` itself, which lives in `dart:ui` (foundation only
  re-exports it), so it is `ui.ClampDouble`.
- **A Flutter quirk, kept:** `BouncingScrollSimulation` caps the hand-over
  velocity with `min(dx, 5000)`, which limits overscroll but not
  underscroll (negative velocities pass through). The same fast fling
  overshoots 121 px past the end but 287 px past the start. A mutation that
  "fixes" this fails the difftest.
- **Difftest:** 11 scroll-simulation scenes sampled directly at 60 Hz for
  4 s, max difference 2.7e-12. Before it, two rules survived mutation
  (the transfer-velocity cap, and the strict `position < leadingExtent`
  test for starting out of range).
- **`internal/dart`** reproduces Dart's number formatting
  (`double.toString`, `toStringAsFixed`), checked against the Dart 3.50.1
  SDK. Dart rounds exact ties away from zero (`2.5.toStringAsFixed(0)` is
  "3"); Go's strconv rounds them to even.

### dart:async in Go (`async`), needed by scheduler and animation
- Flutter's `TickerFuture` is a `Future`, and the frame contract depends on
  Dart's event loop (microtasks are drained between `handleBeginFrame` and
  `handleDrawFrame`; flutter_test advances a fake clock). The `async`
  package reproduces the Dart 3 VM's rules, read from the 3.50.1 SDK source
  (`future_impl.dart`, the VM's `async_patch.dart`) and package:fake_async:
  FIFO microtasks; timers by deadline, then creation order; async
  `Completer`/`Future.value` deliver in a microtask; listeners run in
  registration order during completion; `await` on a completed future costs
  one microtask; a throw before the first `await` fails the future in a
  microtask, after it synchronously.
- **`await` without a divergence:** `Async(func)` runs the body as a
  coroutine (a goroutine with explicit hand-off, so only one runs at a
  time), and `future.Await()` really suspends it. Code reads like Dart's.
  Parked coroutines whose future never completes are leaked goroutines
  (Dart would collect them).
- **Difftest:** 9 scenes log callback order inside fake_async in Dart and
  in Go (queue order, completers, chains, async functions, nesting, errors,
  timers, periodic catch-up, flushTimers): identical. 8 mutations of the
  rules above each fail it. One mutation deadlocked a coroutine instead of
  failing: the mutation runner now has a 60 s timeout.

### P0: the repository mirrors upstream's layout
- At the user's request, the repository root is now flutter/flutter@3.50.1
  without Dart (no `upstream/` prefix), so the two trees read side by side.
  Go ports sit beside the Dart files they replace, with the same base name
  (`packages/flutter/lib/src/physics/friction_simulation.go`); flutter-go's
  own code (runner, Impeller binding, `async`, difftests, tools, docs) is
  in `flutter_go/`. Moves were done with `git mv` (history kept), and
  `sync_upstream.sh --check` still finds every upstream blob identical,
  including the three CRLF files.
- **One place can't mirror upstream:** `engine/src/flutter/lib/ui/` holds
  the engine's C++ next to dart:ui's Dart, and Go refuses to build a
  package in a directory with C++ sources (excluding them would mean
  editing upstream files). dart:ui's Go files are in `ui/go/`.
- `internal/dart` became `flutter_go/dart`: Go's `internal/` rule would
  forbid `packages/...` from importing it under `flutter_go/`.
- GitHub shows `.github/README.md` (ours; upstream has none there), so the
  root `README.md` can stay upstream's byte for byte.
- Side effect to know about: upstream's `.claude/`, `.agents/` and
  `.gemini/` agent configuration is now live at the root.
- **A bug in the sync script, found by running it end to end:** 56 upstream
  files have spaces in their names (iOS golden PNGs), and the script split
  `ls-tree` output on whitespace, so `--check` compared truncated paths on
  both sides and couldn't see those files; a full re-sync then staged bogus
  truncated paths. Now the path is split off only at the tab, the CRLF fix
  restages just the blobs whose hash differs from upstream (a full sync
  takes under a minute instead of many), and the script ends by verifying
  the index against the tag. The committed tree was correct throughout;
  only the check was blind.

### dart:ui geometry and lerp (P1 starts)
- `engine/src/flutter/lib/ui/go/geometry.go` and `lerp.go` port
  geometry.dart and lerp.dart in full (`OffsetBase`, `Offset`, `Size`,
  `Rect`, `Radius`, `RRect`, `RSuperellipse`, `RSTransform`, `lerpDouble`),
  with the engine's own Dart tests ported (`testing/dart/geometry_test.dart`,
  `lerp_test.dart`) and a difftest of 56 cases printed by Flutter 3.50.1
  (toString forms, lerps against null, `%`, `~/`, middle rects, contains
  over a grid): all identical.
- Details that only the difftest pins down: `toStringAsFixed` rounds exact
  ties away from zero (`Offset(5.25, …)` prints 5.3), Dart's `%` is never
  negative, `~/` throws `UnsupportedError` on non-finite results, and
  `RSTransform` stores float32 (`0.1` reads back as 0.10000000149011612).
  `flutter_go/dart` gained `Mod` and `TruncDiv`, checked against the Dart
  SDK.
- **ENGINE GAP:** `RSuperellipse.contains` is native
  (`RSuperellipse::contains` in the engine); the prebuilt Impeller SDK has
  no superellipse API. Next step: port the engine's C++ algorithm
  (`impeller/geometry/round_superellipse_param.cc`) to Go and the 7
  skipped engine test cases with it.
- Mutation checks: 10 caught (float32 storage, Flutter's truncated
  1-cos(π/4) constant, lerp form and null branches, contains radii, tie
  rounding, `%` sign, `Size - Size` type, isStadium, lerpDouble's NaN
  shortcut). One rule (elliptical top-right corners in `contains`) was
  only caught after adding a contains grid to the difftest. Two survivors
  are equivalent mutations (`>=` at a boundary where the result can't
  change).
- `rendering` still defines its own Offset/Size/Rect; moving it onto `ui`
  is the next P0/P1 step.
- **ENGINE GAP filled in Go:** `RSuperellipse.contains` now ports the
  engine's C++ (`lib/ui/painting/rsuperellipse.cc` and
  `impeller/geometry/round_superellipse_param.cc`) in float32, including
  `SafeNarrow`, `GetPositive` and the one step C++ computes in double. The
  7 engine test cases pass, and grids over 6 shapes (~5,400 points each,
  plus 25,600 points at 0.05 px over one corner) match the engine's native
  `contains` run by `flutter test`, point for point. Mutations: 8 caught
  (one, a lookup-table entry, only after adding the fine grid); 3 survivors
  are equivalent or below what any grid can resolve (the all-corners-same
  fast path, `seN < 2` where `seN` is 0 or ≥ 2, and `R` rounded to float
  instead of double, a ~1e-6 px shift).
- **rendering and widgets now use dart:ui's Offset, Size and Rect.** The
  old types had other field names and a different Rect (left, top, width,
  height instead of Flutter's left, top, right, bottom). Fields were renamed
  type-aware with `gopls rename` (Alignment also has X/Y fields, so text
  replacement wasn't safe), `RectFrom(o, s)` became `o.And(s)` (Dart's
  `offset & size`), and literals became constructors. All tests and
  difftests pass unchanged. `rendering.Offset/Size/Rect` remain as aliases
  for now (like Dart's re-export); remove them once callers import `ui`.

### dart:ui Color; matching Dart's floating point bit for bit
- `painting.go` starts dart:ui's painting.dart with `Color` and
  `ColorSpace` (deprecated 8-bit accessors included), ported with the
  engine's color tests (except Paint, subclass and 1<<1000000 cases) and a
  21-case difftest compared **exactly**. Flutter quirks kept:
  `_ClampTransform` never calls its child transform (extended sRGB →
  Display P3 only clamps), and the identity transform keeps the source
  color space label (sRGB → extended sRGB stays tagged sRGB).
- **Why exact was hard, and the three rules that fixed it** (each found by
  a failing bit, not assumed):
  1. *Dart's math is libm's, plus VM special cases.* `math.pow` calls C's
     `pow` (Go's `math.Pow` differs in the last bit), except that the VM
     returns `x*x*x` for exponent 3, `x*x` for 2, `sqrt` for 0.5 and so on
     (pow(0.3, 3.0) is 0.027 in Dart, 0.026999999999999996 in C). Same for
     exp/log/sin/cos/atan2: `flutter_go/dart` wraps libm through cgo with
     those special cases, verified value by value against Dart.
  2. *Go fuses multiply-adds on arm64; Dart never does.* `a*b + c` can
     become one FMA instruction (one rounding instead of two), and whether
     it does depends on inlining, so the same function gave different bits
     at two call sites. `flutter_go/cmd/fmacheck` lists every fusable float
     expression in the ports and `-fix` wraps the product in `float64(...)`
     (the spec's way to forbid fusion). 53 sites fixed.
  3. *Go constant expressions are exact; Dart's consts round each step.*
     `9.80665 * 39.37 * 160.0 * 0.84` folded exactly in Go differed from
     Dart's double arithmetic; fmacheck now also flags float constant
     expressions whose exact value differs from step-by-step float64.
- **Result:** physics (15 scenes) and the scroll simulations (11 scenes)
  now match Flutter with **zero** difference, down from 4.5e-13 and
  2.7e-12; the difftests compare them exactly. The drag-fling scenes still
  differ by up to 2.6e-11 (velocity tracker; to be fixed when gestures is
  ported).
- `rendering.Color` is now dart:ui's `Color` (alias); the two unkeyed
  literals (different field order: RGBA vs ARGB) were rewritten as keyed.

### foundation, part 1
- Ported with Flutter's tests: constants (build tags for release/profile),
  object (`objectRuntimeType`), platform (`TargetPlatform`,
  `defaultTargetPlatform` from `runtime.GOOS`, android under
  `FLUTTER_TEST` in debug, the debug override), key (`Key`, `LocalKey`,
  `UniqueKey`, `ValueKey[T]`; Go `==` gives Dart's equality, including
  "different type ⇒ different key"), basic_types (callback typedefs,
  `CachingIterable` over `iter.Seq`, `Factory`, `lerpDuration`),
  observer_list (`HashedObserverList` keeps Dart's insertion order, which a
  Go map wouldn't). Partial: `FlutterError` (message only) and
  `shortHash`/`describeIdentity` (Go has no identityHashCode; objects get a
  stable id on first use).
- Mutations: 6 caught; `lerpDuration` rounding was only caught after adding
  half-microsecond cases (Flutter's tests never land on a fraction).
- **Found: part of foundation's API lives in other packages.** At 3.50.1
  `ChangeNotifier`, `Listenable`, `ValueListenable` and `ValueNotifier` come
  from package:listen (re-exported), `Matrix4` from vector_math, and
  `Characters` from characters. They are API a Flutter developer uses, so
  they need their own items (apiparity doesn't count them yet).
- **Design question for scheduler:** Dart's `Future` is an interface that
  Flutter implements (`SynchronousFuture`, `TickerFuture`); `async.Future`
  is a concrete struct. Resolve before porting those two.
- foundation part 2: collections (`listEquals`/`setEquals`/`mapEquals`
  with nil as Dart's null, `binarySearch`, Flutter's own stable
  `mergeSort` ported as is), bitfield (`BitField[T ~int]`, 62 bits as on
  the Dart VM), unicode (bidi characters as a package-level struct, the
  mapping for classes of static constants). Flutter's tests ported; empty
  vs null cases added (a mutation dropping the nil check survived
  Flutter's tests). One mutation (merge-sort cutoff) is equivalent: a
  stable sort's output is unique.
- foundation serialization (`WriteBuffer`/`ReadBuffer`, the byte layer of
  platform-channel codecs): Flutter's tests ported, and a difftest where a
  mixed sequence (alignment, both endiannesses, growth from capacity 1,
  -0.0, infinities) produces the same 96 bytes as Flutter, which Go's
  ReadBuffer then reads back. dart:typed_data maps to Go slices and
  `binary.ByteOrder`. Mutations: 6 caught; the read-side alignment was
  only caught once the difftest read Flutter's bytes back.
- foundation `PersistentHashMap` (the trie behind inherited-widget lookup):
  ported node for node; keys hash with `maphash.Comparable` or their own
  `HashCode()` (Flutter's tests force collisions that way). Flutter's tests
  ported, plus one that re-reads every intermediate map: a mutation that
  broke persistence in the collision path survived Flutter's tests. The
  16-entry inflation threshold is a performance detail (an equivalent
  mutation).
- foundation diagnostics (`diagnostics.dart`, the text-art and JSON
  descriptions behind every `toString`/`toStringDeep` and the widget
  inspector): ported in full, split over five `diagnostics*.go` files, plus
  `debugFormatDouble` (debug.go, partial) and `dart.MapEntry`. All 39
  public declarations exposed. Overridable methods use one pattern: every
  embeddable base keeps `self` (set by constructors, or `SetSelf` for types
  in other packages) and calls overrides through it; nullable values are the
  nil of the type parameter (see API_MAPPING "Mixins and overridable
  methods" and "Nullable values and options"). Strings are measured in
  UTF-16 units, as Dart does, so wrapping and box widths match with emoji
  and accents.
- Flutter's tests: diagnostics_test.dart 29 of 30 (all goldens copied
  verbatim; "Stack trace test" needs `DiagnosticsStackTrace`, which is
  assertions.dart) and diagnostics_json_test.dart 13 of 15 (the two Element
  tests need widgets' `Element`). Plus a few tests of our own for parts
  Flutter only covers indirectly.
- Difftest (`difftest/diagnostics_test.go` + `flutter/test/diagnostics_test.dart`):
  516 outputs compared exactly (hashes normalized): a tree with 68
  properties of every kind (toStringDeep at 3 widths and with prefixes,
  toString, toStringShallow, every minLevel, timeline arguments, JSON full /
  default / iterative), each property alone (toString, dense, toStringDeep,
  level, showName, JSON), every DiagnosticsTreeStyle as a tree and as a root,
  long wrapped text (accents, emoji, CJK, unbreakable words, non-wrappable
  names and values) in 5 style pairs × 4 widths, truncation, error blocks
  with line prefixes, Diagnosticable vs DiagnosticableTree vs the mixin.
- **Found by the difftest** (the unit tests passed on the first run):
  Dart's `toUpperCase` keeps `ß` (simple case mapping; I had expanded it to
  `SS`); Go enums (named ints) were emitted as JSON numbers and printed
  without Dart's `Type.` prefix as `defaultValue`; Dart prints a top-level
  function as `Closure: () => void from Function 'f': static.` (closures
  omit the suffix; Go tells them apart by the runtime name); a lazy property
  of non-nilable `T` that panicked reported `""` instead of null. And a bug
  in the test itself: Go folds the constant `0.1+0.2` exactly (rule 3
  above). `fmacheck` crashed on constant string concatenation; fixed.
- Flutter quirks reproduced on purpose: `DiagnosticableTree.toStringShallow`
  (the class, not the mixin) starts with the full `toString()`, so the
  properties appear twice; `TextTreeRenderer` truncation appends a `...`
  property when a node has *fewer* properties than the limit; an unnamed
  `truncateChildren` node says "This null had..."; an expandable
  Diagnosticable value prints its properties twice (`Point2#…(x: 3.0)(x:
  3.0)`); `PercentProperty(-0.0)` is `-0.0%`.
- Mutations: 39 tried, 37 caught, 2 equivalent. Caught only after adding
  difftest scenes: wrap ranges measured in bytes instead of UTF-16 units,
  the break-point boundary at the start of a wrappable range, `ifEmpty` on
  an unquoted/raw value. Equivalent: the word wrapper's fast-path `<` →
  `<=` and over-width `>` → `>=` (no line can break at exactly the width;
  confirmed by comparing the original and mutated wrapper on 300k random
  inputs).
- Not ported: `debugDoublePrecision` (needs `toStringAsPrecision`),
  ErrorSummary/ErrorDescription/ErrorHint/ErrorSpacer/DiagnosticsStackTrace
  (assertions.dart; the tests' `FlutterError.fromParts([ErrorSummary(..)])`
  became `panic(NewFlutterError(..))`, same runtime type), and the
  `.lazy` constructor's `defaultValue is T?` assert. Profile-mode branches
  are ported but untested.
- foundation assertions, stack_frame and print (`FlutterError` and its
  static reporting machinery, `FlutterErrorDetails`, the `Error*`
  diagnostics, `DiagnosticsStackTrace`, stack filters, `debugPrintStack`,
  `StackFrame`, `debugPrint*`, `debugWordWrap`, plus `ErrorToConsoleDumper`
  from error_dumper.dart, which assertions.dart dumps through): ported in
  full; all 22 missing top-level names are now exposed (foundation 77 → 99
  of 137; total 153 → 175). The minimal `FlutterError{Message}` is gone:
  `NewFlutterError(message)` now builds Flutter's summary + descriptions,
  and `FlutterError` is a Go `error` and a `dart.AssertionFailure`.
  `flutter_go/dart` gained the dart:core pieces this needs: `Print` /
  `SetPrintHandler`, `AssertionError` / `SafeToString`, `Pattern`,
  `StackTrace`, and the part of `Uri.parse` the frame parser reads
  (ported from the SDK's uri.dart: dot segments, percent normalization).
  `DiagnosticableNode` now calls `Builder()` through `self`, so
  `_FlutterErrorDetailsNode` can override it as in Dart.
- **Design (API_MAPPING "Errors and stack traces"):** exceptions are `any`;
  Dart's `is AssertionError/String/num/Error` map to
  `dart.AssertionFailure`/`string`/builtin numbers/`error`. Flutter reads
  stack traces as text, so Go stacks (`dart.StackTraceCurrent`,
  `dart.StackTraceFromPanic` in a deferred recover) are rendered in the
  Dart VM's format with Dart-style names and URIs: flutter-go's packages
  map to `package:flutter/...`, `flutter_go/async` to `dart:async` (so the
  default filter elides it, like dart:async), other files to `file://`.
  A real nil-dereference panic reported with `FlutterErrorReportError`
  prints `The following errorString was thrown:` / the runtime message /
  `#0      explode (package:flutter/src/foundation/assertions_test.go:780)`
  / callers / `#6      tRunner (file:///usr/local/go/src/testing/testing.go:1792)`.
  Differences: no columns, no async-suspension lines, runtime frames
  dropped, stack overflow is fatal. Statics are `FlutterErrorOnError`,
  `FlutterErrorPresentError`... Output goes through the `DebugPrint`
  variable (tests replace it, as Flutter's do) and `dart.Print` (tests
  capture it with `dart.SetPrintHandler`; global, not zone-scoped).
- Flutter's tests: assertions_test 11/11, stack_frame_test 13/15 (not
  ported: live stack overflow, fatal in Go; the ParallelWaitError test,
  which needs `(f1, f2).wait`), print_test 3/3, error_reporting_test 7/7,
  stack_trace_test 2/2 (assert-generated `_AssertionError`s are rebuilt
  with the VM's toString; Go has no columns; Go frames replace Dart's
  test-runner frames), plus diagnostics_test's skipped "Stack trace test".
  Plus own tests for filters, reporting, Go panics, wrapping and the
  throttle's limits.
- **Found:** three typos in Flutter's stack_frame_test expected data
  (`packagePath`), invisible because `StackFrame.==` ignores packagePath;
  the Go test checks every field and lists them. My first stack mapping
  used the function name's package; a closure inlined from flutter_go/async
  is named after its caller, so `coroutine.go` showed up under
  `package:flutter/src/foundation`. Locations now come from the file path.
- Difftest (`difftest/assertions_test.go` + `flutter/test/assertions_test.dart`):
  339 outputs compared exactly (hashes normalized): FlutterError
  toString/toStringDeep/shallow/JSON, ErrorSummary/Hint/Spacer JSON,
  malformed errors, 21 exception kinds (strings, numbers, AssertionError
  variants, FlutterError, Exception/Error twins, plain objects, real Dart
  asserts) × toString/summary/exceptionAsString/properties/dump/with-stack,
  a full report with library, context, information collector, expandable
  and truncatable nodes, dumpErrorToConsole sequences (Another exception,
  forceReport, silent, null library), copyWith, propertiesTransformers,
  custom stack filters, 30 StackFrame lines (VM, web debug, gaps, odd
  URIs), the default filter and RepetitiveStackFrameFilter (incl.
  overlapping and regex patterns), DiagnosticsStackTrace variants,
  debugPrintStack, debugWordWrap of 12 texts × 10 widths + indent,
  debugPrintSynchronously, and the debugPrintThrottled schedule in
  fake_async (prints and debugPrintDone at each millisecond). All matched
  on the first run.
- Flutter quirks reproduced on purpose: a collapsed run of filtered
  frames reports `index - start + 2` frames (4 frames print "(5 frames)");
  `RepetitiveStackFrameFilter` never matches a run ending at the last
  frame; the frame regex's greedy method makes `#13 paren (file:///dir
  (copy)/x.dart:1:2)` the method `paren (file:///dir`; `debugWordWrap`
  throws on a blank message at least as long as the width and when the
  wrap indent makes the prefix longer than the message;
  `_FlutterErrorDetailsNode` rebuilds (and re-transforms) its properties on
  every access, with a sparse default style.
- Mutations: 116 valid, 113 caught, 3 equivalent: `string` vs
  `AssertionFailure` order in the error-name switch (a Go string can't
  implement it), resuming `inWord` instead of `atBreak` after an early
  break (the index is at a space, so it moves to atBreak at once), and
  breaking at `>= width` (300k random inputs, panics included, give the
  same lines; as in the diagnostics wrapper). Caught only after adding
  tests or scenes: the `position > 2` boundary, dump wrapWidthProperties
  (only visible on indented long lines: max(100, indent + 65)), the
  truncation limit, the lazy-vs-greedy method regex, JavaScript `.` and
  `\s` semantics (U+2028, NBSP), wrap fast path and UTF-16 length, first
  vs last summary, overlapping repetitive filters, `new A.b.c`, the
  throttle's stopwatch, and StackFrame equality fields.
- Not ported: web-only and release/profile branches are ported but
  untested (debug build only); `Uri.parse`'s errors (Go parses leniently).

### Upstream's AI-agent configuration removed (at the user's request)
- The root mirrors upstream byte for byte **except** Flutter's AI-agent
  configuration and skills: `.agents/`, `.claude/`, `.gemini/`,
  `agent-artifacts/`, `CONTRIBUTING.ai.md` and the nested
  `AGENTS.md`/`GEMINI.md` files (50 files). They configured agents working
  here as if this were flutter/flutter (its skills appeared in sessions).
  `sync_upstream.sh` keeps them out (`EXCLUDED`), and `--check` verifies
  everything else as before.
### dart:async: `Future` becomes an interface (resolves the scheduler design question)
- `async.Future[T]` is now an interface, as Dart's is, so Go code implements
  it (foundation's `SynchronousFuture`; scheduler's `TickerFuture` next).
  Methods: `ThenAny` (Dart's `then<R>` with R erased; the generic
  `Then`/`ThenOrError`/`ThenFuture` dispatch to it and restore R),
  `CatchError`, `WhenComplete`, `Timeout` (new, ported from `_Future.timeout`,
  with `TimeoutException`), `Ignore`, `Await`. `OnComplete` became a free
  function. The old struct is the unexported `future[T]` (Dart's `_Future`);
  every constructor returns the interface. See API_MAPPING "Futures".
- User-defined futures follow the VM: `await` on one calls its `then`
  (`AwaitUserDefined`; if the callback runs synchronously the coroutine
  continues without suspending, as the VM resumes within `then`), chaining
  to one listens through `then` and completes synchronously in the callback
  (`_chainForeignFuture`), `ignore` is `then(_ignore, onError: _ignore)`.
  Type erasure (`EraseFuture`, internal casts) keeps identities Dart keeps:
  `SynchronousFuture.then` returning a future returns that very future.
- Fixed on the way (a deviation the old difftest did not exercise):
  `_chainCoreFuture` also completes synchronously when the target has no
  listeners yet and the source holds a value (or an ignored error); Go
  always went through a microtask unless `sync` was set. Visible through
  `Completer.complete(completedFuture)` and `timeout` on a completed future
  (listener order relative to other microtasks). Caught by the new
  synchronous_future difftest scene `completer_chains_completed_core`.
- Checks: the 9 async scenes still match Dart (the Go file only changed
  `*Future` → `Future` and `x.OnComplete(` → `OnComplete(x, `); the 8 async
  mutations re-run after the refactor (LIFO microtasks, timer ties, async
  value delivered synchronously, reversed listeners, late listener run
  synchronously, early throw completed synchronously, periodic catch-up
  once, await of a completed future without a microtask): all caught, as
  before it. Two more were missed before and after (sync chaining of a
  completed source through a microtask, and not reporting an unhandled
  error): the first is now caught by the synchronous_future difftest; the
  second is not observable in a difftest (Dart's zone reports it), noted.

### foundation, finished (binding, SynchronousFuture, licenses, timeline, memory allocations, isolates, ...)
- Ported from 3.50.1, one Go file per Dart file: binding (`BindingBase`
  interface + embeddable `BindingBaseBase`, `Construct(self)` is the Dart
  constructor; checkInstance, debugCheckZone, lock/unlock, reassemble, the
  service-extension registry and the six foundation extensions),
  synchronous_future, licenses, timeline (`FlutterTimeline` as a variable
  of statics, `FlutterTimelineTimeSync`), memory_allocations, isolates
  (`Compute`), consolidate_response, service_extensions, node
  (`AbstractNode` + `AbstractNodeBase`), annotations, capabilities, and the
  rest of debug (`DebugAssertAllFoundationVarsUnset`,
  `DebugInstrumentAction`, `DebugDoublePrecision`, brightness/VM-service
  variables, `DebugMaybeDispatchCreated/Disposed`). foundation: 99 → 137 of
  137 top-level names; total 175 → 221 of 2792 (tested 127 → 162).
- New support code, ported from the Dart SDK shipped with 3.50.1:
  `flutter_go/dart`: `ToStringAsPrecision` (double-conversion's
  ToPrecision rules: exact ties away from zero, exponential below 1e-6 or at
  ≥ precision digits, "e+", signed -0), `DurationToString`, `JSONEncode`,
  `StateError`/`ArgumentError`/`RangeError`/`UnimplementedError`.
  `flutter_go/developer` (dart:developer: registerExtension/postEvent/
  ServiceExtensionResponse, Timeline/TimelineTask/Flow with Dart's argument
  checks and unbalanced-call StateErrors). `flutter_go/io`
  (`HttpClientResponse`, `HttpClientResponseCompressionState`).
  `flutter_go/async`: a Stream subset (`_BufferingStreamSubscription`'s
  state machine, `_PendingEvents` one event per microtask, async and
  internal sync `StreamController` with `addStream`, `Stream.multi`,
  `fromIterable`, `fromFuture`, `empty`, `toList`) and `Loop.Post` /
  `RunPosted` / `WaitPosted` for events from other goroutines (Compute's
  result, like a port message).
- dart:ui (all **partial**, only what foundation reads):
  `platform_dispatcher.go` (`PlatformDispatcher`, `PlatformDispatcherInstance`,
  `PlatformBrightness` with the configuration default light), `window.go`
  (`Brightness`, the deprecated `SingletonFlutterWindow` and `Window` with
  `PlatformDispatcher`/`PlatformBrightness` only), `painting.go` (`Image`
  and `Picture` as opaque types with only their static
  `onCreate`/`onDispose` hooks, `ImageEventCallback`,
  `PictureEventCallback`). dart:ui exposed: 12 → 20 of 195.
- **ENGINE GAP: no VM service.** `registerServiceExtension` registers in
  `flutter_go/developer`'s registry; nothing but `developer.LookupExtension`
  (tests, a future transport) calls a handler; `postEvent` returns as Dart's
  does without a listener on the Extension stream (`extensionStreamHasListener`
  is false); DevTools, `flutter attach`, hot reload's `ext.flutter.reassemble`
  can't reach the app. **ENGINE GAP: no timeline recorder.** Timeline events
  are dropped, as with the Dart stream disabled (FlutterTimeline's own
  debug collection works). Not ported: `_initListenable` and the memory
  allocation hooks into package:listen (not ported), `asStream`,
  async* generators, broadcast streams, dart:io's HTTP client.
- Design: Go has no zones, so the binding's zone is the current
  `async.Loop`; `assert(() {...}())` blocks with side effects are
  `if KDebugMode`; Dart's function identity (`removeListener`,
  `debugPrint != debugPrintOverride`) is the closure pointer; the
  isolate is a goroutine whose result is posted to the caller's loop
  (`ComputeCallback` returns R: a goroutine has no event loop of its own).
  `isCanvasKit/isSkwasm/isSkiaWeb` panic with UnimplementedError, as
  `_capabilities_io.dart` throws (Flutter reads them only under kIsWeb).
- Flutter's tests: annotations 3/3, double_precision 2/2, debug 4/4,
  synchronous_future 1/1 (asStream and the future-returning whenComplete
  action skipped), timeline 9/9, memory_allocations 6/7 (the publishers
  test needs ChangeNotifier and real Pictures/Images), binding 1/1,
  binding_2 1/1, reassemble 1/1, service_extensions 6/31 (the foundation
  extensions; frame checks and the 25 extensions of other bindings
  skipped), isolates "compute()" 8 of 24 expectations (plus the debugLabel call
  without the name check) (Isolate.exit/kill,
  async callbacks, unsendable values, debugName skipped) and 0/4 "closes all
  ports" (they spawn a Dart VM), consolidate_response 10/10, licenses 4/4
  (async* collectors as fromIterable). Plus own tests for every file
  (node, capabilities, developer, checkInstance errors, lockEvents,
  bool/numeric extensions, double.parse, timeline assertions, hooks).
- Difftests, all exact: synchronous_future (10 scenes: then/await/chaining
  of a user-defined future against core futures, timeout, whenComplete,
  ignore), stream (10 scenes), double_precision (1098 values × 21
  precisions, debugFormatDouble × 22, RangeError messages), licenses (418
  texts: Flutter's test text, the repo's LICENSE files incl. the 1.3 MB
  sky_engine LICENSE, 400 random texts, 65 of which make Flutter's parser
  assert; plus the registry's delivery order), binding (75 log lines:
  construction order, registrations, extension results/events/reassembles/
  unlocks against microtasks and timers), memory_allocations (listener
  add/remove during dispatch, hooks, toMap, reported errors, timeline
  aggregation sums). Run the Dart side with
  `flutter test test/<name>_test.dart` in difftest/flutter.
- **Found:** an async function that throws after awaiting a
  SynchronousFuture fails its future synchronously, before anyone can
  listen, so Dart reports the error as uncaught (and delivers it to late
  listeners too) — reproduced. Flutter's checkInstance asserts
  `_debugInitializedType == null` before throwing "Binding mixin instance is
  null but bindings are already initialized.", so in debug builds that
  FlutterError is unreachable (a bare AssertionError). Flutter's
  memory_allocations_test expects `hasListeners` false inside a listener
  that removed the other one; it is true (verified in the difftest), and
  the failed `expect` is swallowed by dispatchObjectEvent's error
  reporting. A canceled subscription's `Future._nullFuture` belongs to the root zone,
  so fake_async never runs its listeners (a stream scene had to drop one;
  Go has no zones and runs them).
- Mutations: 93 valid on the new code, 92 caught, 1
  equivalent (skipping the `\n` of a CRLF in the blank-line state: the
  second break is a no-op). Caught only after adding tests or scenes:
  Duration.toString padding, JSON quoting and double formatting (unit
  tests), a -1 content length kept as a total, addStream not paused before
  the listener comes, pause not cancelling an already scheduled delivery
  (two stream scenes). Plus, on the Future refactor, 11 mutations of the
  foreign-future paths and SynchronousFuture (all caught; the sync-chain
  rule, timeout of a failed future, a value arriving after the timeout and
  SynchronousFuture.timeout needed extra scenes), and the 8 async ones
  (still caught).
