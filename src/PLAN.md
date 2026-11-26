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
  in `src/`. Moves were done with `git mv` (history kept), and
  `sync_upstream.sh --check` still finds every upstream blob identical,
  including the three CRLF files.
- **One place can't mirror upstream:** `engine/src/flutter/lib/ui/` holds
  the engine's C++ next to dart:ui's Dart, and Go refuses to build a
  package in a directory with C++ sources (excluding them would mean
  editing upstream files). dart:ui's Go files are in `ui/go/`.
- `internal/dart` became `src/dart`: Go's `internal/` rule would
  forbid `packages/...` from importing it under `src/`.
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
  `src/dart` gained `Mod` and `TruncDiv`, checked against the Dart
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
     exp/log/sin/cos/atan2: `src/dart` wraps libm through cgo with
     those special cases, verified value by value against Dart.
  2. *Go fuses multiply-adds on arm64; Dart never does.* `a*b + c` can
     become one FMA instruction (one rounding instead of two), and whether
     it does depends on inlining, so the same function gave different bits
     at two call sites. `src/cmd/fmacheck` lists every fusable float
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
  `src/dart` gained the dart:core pieces this needs: `Print` /
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
  map to `package:flutter/...`, `src/async` to `dart:async` (so the
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
  used the function name's package; a closure inlined from src/async
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
  `src/dart`: `ToStringAsPrecision` (double-conversion's
  ToPrecision rules: exact ties away from zero, exponential below 1e-6 or at
  ≥ precision digits, "e+", signed -0), `DurationToString`, `JSONEncode`,
  `StateError`/`ArgumentError`/`RangeError`/`UnimplementedError`.
  `src/developer` (dart:developer: registerExtension/postEvent/
  ServiceExtensionResponse, Timeline/TimelineTask/Flow with Dart's argument
  checks and unbalanced-call StateErrors). `src/io`
  (`HttpClientResponse`, `HttpClientResponseCompressionState`).
  `src/async`: a Stream subset (`_BufferingStreamSubscription`'s
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
  `src/developer`'s registry; nothing but `developer.LookupExtension`
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

### animation: curves.dart (and dart:math's Random)
- `packages/flutter/lib/src/animation/curves.go` ports curves.dart in full:
  `ParametricCurve[T]`/`Curve`/`Curve2D` as interfaces with embeddable
  `ParametricCurveBase`/`CurveBase`/`Curve2DBase` (self pattern, so
  `Transform`'s assert and 0/1 shortcut call a subclass's
  `TransformInternal`, and `Curve2D`'s sampling calls an overridden
  `SamplingSeed`), every public class, the private `_Linear`,
  `_DecelerateCurve` and `_Bounce*Curve` (Go types with the Dart names, so
  `toString` matches), and `Curves` as a package-level struct with all 43
  constants. The old minimal `Curve`/`Cubic`/`Linear`... in animation.go
  are gone; `Controller`, widgets and cmd/animate use `Curves.X`.
- **`dart.Random`** (`src/dart/random.go`) ports the VM's `_Random`
  (math_patch.dart): Thomas Wang seed mix, four cranks, multiply-with-carry,
  `nextInt` with the power-of-two fast path and rejection loop, `nextDouble`
  from 26+27 bits, `nextBool`. Checked against `dart run` output for 14
  seeds (incl. min/max int64 and seed 9223367638806167551, found by
  inverting the mix, whose mixed seed is 0 → 0x5a17) × 6 doubles, 14
  `nextInt` maxes and 8 bools, plus Dart's RangeError text: identical on the
  first run. `generateSamples` needs it to match Flutter's samples.
  `src/dart` also gained `ArgumentError`, `RangeError`,
  `UnimplementedError` (Dart's toString texts).
- Flutter's tests: curves_test.dart 22/22, plus own tests (a curve
  embedding `CurveBase` from outside, unbound-self panic, Split's
  base `transformInternal` throwing UnimplementedError, toString forms,
  parameter asserts).
- **Difftest** (`difftest/curves_test.go` + `flutter/test/curves_test.dart`):
  82 curves (all 43 `Curves`, SawTooth ×3, Interval ×7 incl. nested and the
  degenerate `Interval(0.5, 0.5)`, Split ×5, Threshold ×4, Cubic ×3,
  ThreePointCubic, Elastic periods ×6, FlippedCurve ×5, CatmullRomCurve ×5)
  at 124 t values (i/97, edges like 1e-9, 5e-324, the bounce breakpoints)
  plus `toString` and `flipped.toString`; 9 Curve2Ds (CatmullRomSpline with
  handles/tension/precompute/a near-closed loop, a closed circle with and
  without its own seed, a wave) sampled, `generateSamples` with 3 option
  sets (t, point and `Curve2DSample.toString`), `findInverse` at 21 x,
  `samplingSeed`; 23 error cases (assert messages, FlutterError, the
  ArgumentError, the NaN seed's UnsupportedError); `validateControlPoints`
  for 16 point sets × 3 tensions with and without reasons. 16,381 doubles,
  all compared **bit for bit**: identical.
- **Found by the difftest: Go fuses across statements, and across
  inlined calls.** `t *= count; return t - trunc(t)` (SawTooth) compiled
  to one multiply-subtract: `SawTooth(3)` at 1/3 gave -5.6e-17 instead of
  Dart's 0. fmacheck only looked at `a*b + c` inside one expression; it now
  also flags a product stored unrounded (`v := a*b`, `v = a*b`, `v *= x`),
  and `-fix` wraps it. That found and fixed 18 more sites (ui geometry,
  painting, rsuperellipse, friction, flex, objects, text, scroll_simulation,
  velocity; existing difftests unchanged, drag-fling still 2.6e-11).
  `ui.Offset.Mul`/`Scale` now round their products: a mutation removing
  that fails the spline difftest, so the fusion through an inlined method
  is real. fmacheck still can't see a product returned from a function or
  stored in a struct literal and added by an inlining caller; such
  helpers need `float64(...)` by hand.
- Flutter quirks kept: `ThreePointCubic.toString` in debug is just
  `"ThreePointCubic "` (the description is objectRuntimeType's *optimized*
  value, and a trailing space is appended); `Interval(0.5, 0.5)` maps 0.5
  to 1.0 (0/0 is NaN, which `clampDouble` sends to the upper bound);
  asserts without messages report `null`; `Split` overrides `transform`
  (so its `transformInternal` throws UnimplementedError); validate's
  "more than one Y" check happens before the per-sample loop and bails
  unless reasons are collected.
- Mutations: 104 valid, 89 caught, 15 equivalent. Caught only after adding
  cases: Split's range assert, validate's Y tolerance at the ends, the
  X range (above 1 and below 0) and `x >= 1.0` (found by searching control
  points with the Go port; Flutter confirmed). Equivalent: Cubic's `<`
  bound (exact tie unreachable), `estimate <= t` (then |t-estimate| is 0),
  midpoint `start + (end-start)/2` (exact for these dyadics), Interval's
  0/1 shortcut and Split's `t == split` and `<=` (the inner transform or
  the other branch gives the same value), `trunc` vs `floor` (t ≥ 0),
  `<=` at bounceInOut/elasticInOut/ThreePointCubic midpoints (both branches
  give the same bits), isFlat `<=` (exact tie), CatmullRomCurve's search
  `>` (lands on the same sample's y), validate's `x <= 0` (index 1 is
  always (0,0), so monotonicity fails first), Random's `>=` in the
  rejection loop and its power-of-two fast path (both only differ for
  powers of two, where the slow path gives the same result).
- **apiparity bug fixed:** an export with an empty `show` set and an
  unrestricted export printed the same key (`map[]`), so a file first
  reached through a narrow `show` (curves.dart via `animations.dart`) was
  never visited again unfiltered. Totals were undercounted: now 2,885
  declarations (was 2,792); animation 54 (was 28), services 211, gestures
  188. Parity: 175 → 191 exposed, 127 → 143 tested; animation 1 → 17.
- Merge note: the curves and foundation ports each added Dart error types
  to `src/dart`; unified on the value types (`ArgumentError`,
  `RangeError` with an optional message, `UnimplementedError`), with
  `NewArgumentErrorValue`/`NewRangeErrorRange` constructors.

### Layout: `src/` renamed `src/`; `.github/` and `.vscode/` left out
- At the user's request, flutter-go's own directory is now `src/` (import
  paths `github.com/jonathanperis/flutter-go/src/...`), and upstream's
  `.vscode/` is excluded from the mirror like the AI-agent configuration.
- `.github/` was briefly removed, then restored byte for byte: it keeps the
  mirror faithful and lets `.github/README.md` (ours) show on GitHub. Its
  workflows don't fire on pushes here (the push-triggered ones are guarded
  to `flutter/flutter`); five unguarded ones (tree-analyze,
  tool-test-general, mac-verify-binaries, revert, delete-bot-branches) would
  run, and fail, on pull requests to master.
### package:listen (ChangeNotifier, Listenable, ValueListenable, ValueNotifier)
- Flutter 3.50.1's foundation re-exports these four from `package:listen`
  (`flutter/core-packages`), resolved to **1.0.1** by the checkout's
  `pubspec.lock` (sha256 47501a08…); ported from
  `~/.pub-cache/hosted/pub.dev/listen-1.0.1/lib/src/listen.dart` (the
  difftest's own `flutter pub get` resolved the same 1.0.1).
- **Placement rule (API_MAPPING):** third-party packages Flutter re-exports
  live in `src/pub/<package>` (pub's layout, files named after
  `lib/src`), with the whole public API (`ErrorContext`, `ErrorCallback`,
  `ObjectCreated/DisposedCallback`, `VoidCallback`, the statics
  `ListenableOnError`/`ListenableDebugMaybeDispatchCreated/Disposed`,
  `ListenableMerge`...). Go has no re-exports: the re-exporting package
  declares type aliases and constructor/static wrappers in
  `<package>_exports.go` (`foundation/listen_exports.go`); assignable
  statics stay in listen.
- **Design.** `mixin class ChangeNotifier` → interface `ChangeNotifier` +
  embeddable `ChangeNotifierMixin` (self pattern; `NewChangeNotifier()`
  returns `*ChangeNotifierMixin`, printed as Dart's `ChangeNotifier`);
  `ValueNotifier[T]` embeds it, and embedders call `Construct(self,
  value)` (Dart's `super(value)`, which dispatches the creation with the
  outer object). The listener list is Dart's fixed-length list
  (len = Dart's `_listeners.length`), with the same growth (×2), shrink
  thresholds (`count*2 <= length`), nulling during notifications, swap
  compaction, reentrancy counters and no `finally` (an onError that throws
  leaves the call depth incremented, as in Dart). Listener panics are
  recovered with the panic's stack and passed to `ListenableOnError`.
  `assert(() {...}())` blocks run in debug builds (a `debugMode` constant
  from the same build tags as `KDebugMode`); the dispose-during-notify
  assert is plain (always on).
- **Listener identity:** `dart.FunctionsEqual` (new `src/dart/identity.go`,
  with `IdentityHashCode` moved there from foundation so listen's
  `ValueNotifier.toString` and foundation's `shortHash` agree). Same closure
  pointer → equal; otherwise, when both are the compiler's method-value
  wrapper for a pointer receiver (`pkg.(*T).M-fm`, cached by code pointer)
  with the same code pointer, the captured receivers are compared, so
  `n.RemoveListener(s.handle)` removes an earlier `n.AddListener(s.handle)`
  as Dart's tear-off equality does. Foundation's `funcIdentity` (debugPrint
  check, FlutterMemoryAllocations.removeListener) now uses it too.
  Divergences (API_MAPPING): value-receiver and interface method values
  only equal the same closure; a non-capturing func literal is one value.
- **Foundation gaps closed:** `BindingBase.InitInstances` runs
  `_initListenable` (assertion errors → FlutterError: rethrown as is,
  StateError's message, AssertionError's message or toString; listener
  errors → `FlutterErrorReportError` with library "foundation library" and
  context "while dispatching notifications for Listenable");
  `FlutterMemoryAllocations._()` installs the Listenable creation/disposal
  hooks (library "foundation"). `ObjectRuntimeType` now prints generic
  types as Dart does (`ValueNotifier<int>`, was `ValueNotifier[int]`).
- Tests: listen's own `test/listen_test.dart` 25/25 (the default onError's
  stack check dropped: a Go panic carries no stack); Flutter's
  `change_notifier_test.dart` 22/22 (with the binding's onError installed
  and reported errors collected, as tester.takeException);
  memory_allocations "publishers…" now ported without its Picture/Image
  events (7/7). Own tests: Dart's `_listeners` length/layout after growth,
  removeAt and both compaction paths (lengths derived from Dart's
  algorithm), nested-notification compaction, tear-off removal,
  `RemoveListener(nil)` during a notification, creation/disposal hooks,
  ValueNotifier equality (Equal method, NaN, ±0, slices, `any`, pointers),
  toString, overrides through self, the binding's error mapping,
  `FunctionsEqual`/`IdentityHashCode`.
- **Difftest** (`difftest/listen_test.go` + `flutter/test/listen_test.dart`,
  168 log lines, identical on the first run, hashes masked): basic
  add/remove with duplicates, removal of earlier/later listeners and of the
  mutator itself during a notification, duplicates removed once during a
  notification, self removal and clearing everything, three-level nested
  notifications with removal/addition and compaction, listener errors of
  five kinds through a logging onError, an onError that throws (escapes
  the notification), dispose during a notification, every use after
  dispose (with a non-throwing onError the listener is still added and
  called), the default onError (rethrow, then deferred removal and
  dispose's depth assert), ValueNotifier skip rules and toString (double
  incl. NaN/−0.0/∞/1e21, String, List identity, Object?, a class with
  ==, an overridden notifyListeners), Listenable.merge (nulls, a source
  twice, nested toString), tear-off removal, the binding's onError
  (reports, FlutterError.toStringDeep, ValueNotifier set after dispose
  keeps the value) and memory-allocation events. Not compared:
  `ValueNotifier<Object?>.toString` (Go can't print the `?`) and
  `1 == 1.0` in a `ValueNotifier<Object?>` (Go's typed values differ).
- **Found:** with a non-throwing `Listenable.onError`, using a disposed
  notifier only reports: `addListener` still adds (and `notify` then calls)
  the listener — the difftest pins this. A `ValueNotifier` whose `value=`
  throws after dispose has already stored the value.
- Mutations: 59 valid on listen.go, identity.go, binding.go and
  memory_allocations.go; 56 caught, 3 equivalent (the `count == 0` early
  return, starting the swap search at `i` (that slot is null), a reordered
  `count++`). Caught only after adding a test: skipping nil slots in
  `removeListener` (Go can pass nil; Dart can't). Every compaction
  threshold/counter mutation, the dispatch hooks, the onError routing and
  the binding's message mapping are caught.
- apiparity follows `export 'package:listen/...'` into the pub cache (the
  version from pubspec.lock) and counts the shown names for foundation
  (other package: exports — meta's annotations, vector_math, characters —
  are still skipped). Parity: 237/2885 → 241/2889 exposed, 178 → 183
  tested (the 4 listen names, and objectRuntimeType now tested);
  foundation 141/141, 114 tested.
- Not done: the old hand-rolled `widgets.Notifier` (paint.go, integer
  listener ids) and animation's listener lists don't use ChangeNotifier
  yet; release-build (`-tags flutter_release`) runs of the debug-only
  tests fail, as foundation's already did.
### scheduler (P2), and the frame loop moved onto it
- `packages/flutter/lib/src/scheduler` ports binding, debug, priority,
  service_extensions and ticker (3.50.1), one Go file per Dart file:
  SchedulerBinding (interface + `SchedulerBindingMixin`, whose embedded
  `foundation.BindingBase` is Dart's `super`, so mixins stack in Dart's
  order; see API_MAPPING), transient/persistent/post-frame callbacks,
  SchedulerPhase incl. midFrameMicrotasks, scheduleFrame/Forced/WarmUp
  (with the engine frame postponed during a warm-up frame), the epoch and
  timeDilation (Dart's `(us / dilation).round()`, half away from zero),
  endOfFrame, lifecycle and framesEnabled, performance-mode requests,
  timings callbacks (+ `Flutter.Frame` events), the task queue with
  package:collection 1.19.1's `HeapPriorityQueue` ported (its tie order is
  observable and matched), the timeDilation service extension, the frame
  banners; Ticker (self pattern, muted, forceFrames, absorbTicker),
  TickerFuture (a user-defined `async.Future`), TickerCanceled. scheduler:
  0 → 16 of 16; dart:ui 20 → 28 of 195; total 237 → 261 of 2885 (tested
  178 → 195).
- **dart:ui (partial):** PlatformDispatcher gains onBeginFrame,
  onDrawFrame, onReportTimings, scheduleFrame, scheduleWarmUpFrame (two
  `Timer.run`s), requestDartPerformanceMode, frameData/onFrameDataChanged;
  FrameTiming, FramePhase, AppLifecycleState, DartPerformanceMode,
  FrameData, VoidCallback/FrameCallback/TimingsCallback. The embedder side
  (`platform_configuration.go`) ports hooks.dart's `_beginFrame`/
  `_drawFrame`/`_reportTimings` and the engine's
  `PlatformConfiguration::BeginFrame` (begin frame, **drain microtasks**,
  draw frame; time-travel clamp; frame number) and the natives, which go to
  a `PlatformConfigurationClient` the embedder installs. ENGINE GAP: no
  Dart_SetPerformanceMode, no frame timings collected.
- **Migration:** `animation.go`'s ad-hoc Scheduler/Ticker are gone.
  `Controller` takes a `scheduler.TickerProvider` (stop on completion is
  `canceled: false`, interruptions `canceled: true`, as
  AnimationController); widgets get `Element.Vsync()`; the cursor blink is
  an `async.Timer`; `BuildOwner.DrawFrame(size, canvas)` is only the
  rendering pipeline, run from a binding's persistent frame callback.
  - The tester is `flutter_test.AutomatedTestWidgetsFlutterBinding`
    (partial: BindingBase + SchedulerBinding), a process-wide singleton as
    in flutter_test; each `Tester` gets its own `async.Loop` (FakeAsync).
    `Pump` is flutter_test's: elapse; if a frame is scheduled, flush
    microtasks, `handleBeginFrame(2015-01-01 + elapsed)`, flush microtasks,
    `handleDrawFrame`; flush microtasks. Frames are drawn only when
    scheduled (builds and render dirties call `EnsureVisualUpdate`);
    `PumpWidget` = attach + scheduleFrame + pump; a `Size` change forces a
    frame (handleMetricsChanged). After each test it unmounts the tree with
    a warm-up frame and a pump (flutter_test's post-test runApp) and fails
    the test on leftover transient callbacks, pending timers or
    timeDilation ≠ 1. Exceptions caught by the scheduler (reported to
    FlutterError.onError) are re-raised at the end of `Pump`
    (`TakeException` takes them instead).
  - The window runner (`src/app`) installs a
    PlatformConfigurationClient: `scheduleFrame` sets needsFrame; the loop
    advances the main `async.Loop` to the wall clock (timers, microtasks),
    then produces the frame with `PlatformConfiguration.BeginFrame` at
    vsync; resize forces a frame. Checked with cmd/animate (16.7 ms frame
    intervals while animating) and cmd/counter.
- **Found:** the post-test unmount exposed a real leak: a ScrollController
  passed by the user kept its ballistic animation (and ticker) running
  after the list was unmounted; Flutter disposes the position's activity.
  Fixed (`goIdle` on unmount / controller swap). A Dart quirk the difftest
  had to avoid: while a task is refused by the scheduling strategy,
  `handleEventLoopCallback` returns true and `_runTasks` re-arms
  `Timer.run`, so the event loop spins (fake_async's `elapse` never
  returns).
- Bindings are per process; Go runs a package's tests in one process, so
  tests that build their own binding re-run themselves in a child process
  (the foundation binding difftest now does too).
- **Flutter's tests:** scheduler_test 8/9 (the warm-up-interleaving test is
  skipped upstream as flaky; ported as a Go test where the engine frame
  arrives in a microtask between the two warm-up timers), ticker_test 7/7,
  animation_test 2/2, binding_test 5/5, performance_mode_test 4/4,
  debug_test 1/1, priority_test 1/1, time_dilation_test 1/1,
  benchmarks_test 0/1 (LiveTestWidgetsFlutterBinding/benchmarkWidgets not
  ported; its microtask rule is checked on the tester), plus the
  timeDilation test of foundation's service_extensions_test, engine-path
  tests and dart:ui platform_configuration tests.
- **Difftest** (`difftest/scheduler_test.go` + `flutter/test/scheduler_test.dart`):
  a BindingBase+SchedulerBinding in fake_async, 11 scenes, 1000 log lines
  compared **exactly**: callback order and the phase each sees, microtasks
  and futures relative to frames, timers, raw/adjusted time stamps,
  ensureVisualUpdate per phase, cancel semantics, endOfFrame from each
  phase, tickers started in each phase (frame time vs first tick),
  muting (incl. from inside the tick), stop/restart inside onTick,
  TickerFuture complete/cancel/dispose/orCancel/whenCompleteOrCancel/
  await/timeout, absorbTicker, timeDilation with odd microseconds and
  rounding ties, task order by priority (4 tie patterns), strategies,
  locking, lifecycle transitions, forced frames, warm-up frames (and an
  engine frame inside one), errors (library/context/collector), timings,
  performance mode, frame banners. All existing difftests (animations,
  fling, bouncing, drag_fling, lists, editing, repaint, inherited...) pass
  unchanged.
- Mutations: 61 valid, 59 caught, 2 equivalent (`isFirstTask` guard:
  duplicate timers are deduplicated by `_hasRequestedAnEventLoopCallback`;
  never clearing `_removedIds`: ids are never reused). Caught only after
  adding cases: rounding ties of the epoch, the heap's last-left-child tie,
  warm-up resetEpoch/reschedule, the postponed frame's
  hasScheduledFrame reset (a Go engine test; unobservable from Dart),
  inactive enabling frames, duplicate event-loop callbacks, banner
  microsecond padding, muting from inside a tick, orCancel first read after
  a cancel, the tester's clock epoch.
### animation (P3): the rest of the library, and the ad-hoc Controller replaced
- `packages/flutter/lib/src/animation` now ports all of 3.50.1's
  animation library, one Go file per Dart file: animation.dart
  (`Animation[T]` interface + `AnimationBase[T]`, `AnimationStatus` with
  its predicates, `Animation.fromValueListenable`, `drive` as
  `animation.Drive`), animation_controller.dart (`AnimationController`
  with forward/reverse/toggle/animateTo/animateBack/repeat/fling/
  animateWith/animateBackWith/stop/resync/reset/velocity/dispose,
  `unbounded`, `AnimationBehavior`, `_InterpolationSimulation`,
  `_RepeatingSimulation`), animations.dart (kAlwaysComplete/Dismissed,
  AlwaysStoppedAnimation, AnimationWithParentMixin, ProxyAnimation,
  ReverseAnimation, CurvedAnimation, TrainHoppingAnimation,
  CompoundAnimation, AnimationMean/Max/Min with dart:math's max/min),
  listener_helpers.dart (the four mixins, with Dart's HashedObserverList /
  ObserverList semantics and the "animation library" error reports),
  tween.dart (Animatable, Tween with Dart's dynamic `begin + (end - begin)
  * t` through the operator methods, ReverseTween, Color/Size/Rect/Int/
  Step/Constant/CurveTween, chain/animate/evaluate, fromCallback),
  tween_sequence.dart and animation_style.dart (incl. `_LerpedCurve`).
  Divergences are listed in API_MAPPING (self pattern, `Construct` for
  subclasses, options structs, `Tween[*ui.Color]` for nullable tweens,
  `ReverseTween(&x.Tween)`, the `SemanticsDisableAnimations` hook).
- **Migration:** `animation.Controller`/`animate_with.go` are gone.
  widgets' AnimatedBox uses `AnimationController` (`value = 0; forward()`,
  as ImplicitlyAnimatedWidget), ScrollController's ballistic and driven
  activities use `AnimationController.unbounded` + `animateWith` /
  `animateTo`. Every existing difftest (animations, fling, bouncing,
  drag_fling, lists, editing, repaint...) passes with unchanged
  references.
- **Found (behavior the old Controller got wrong, invisible to the old
  difftests):** an interpolation is done only once `elapsed > duration`,
  so a controller reaches its end value at `elapsed == duration` but
  reports completed (and stops its ticker) on the next frame; the old
  controller completed at `t == 1`. Flutter's `animateWith` doesn't notify
  listeners when it starts (the old one did). The scroll difftests are
  unaffected: the extra frame and the initial notification set the same
  offsets. flutter-go's own old test of the reverse timing was corrected
  accordingly.
- **Found by fmacheck:** `_RepeatingSimulation._initialT`'s product was
  stored unrounded and added later (`timeInSeconds + _initialT`); wrapped.
  foundation's `dartToString` prints a `*time.Duration` with Go's format
  ("1s"): AnimationStyle passes the dereferenced Duration to its
  DiagnosticsProperty (Dart prints "0:00:01.000000"). A foundation fix
  (deref nullable Durations) would help other ports.
- Flutter's tests: animation_controller_test 48/48, animations_test
  18/18, tween_test 15/18 (BorderTween, Matrix4Tween, BorderRadiusTween
  skipped: painting/vector_math not ported), listener_helpers_test 2/2,
  animation_controller_listener_test 1/1, animation_from_listener_test
  3/3, iteration_patterns_test 4/4, futures_test 6/6 (asStream not
  ported), animation_style_test 8/8; animation_sheet_test (3) and
  live_binding_test (2) not ported (flutter_test goldens and
  LiveTestWidgetsFlutterBinding). All share one plain SchedulerBinding per
  process; setUp cancels the transient callbacks earlier tests leave (Dart
  gives each file its own isolate). Own tests: listener-list semantics,
  lazy start/stop, error report details, subclassing through `Construct`,
  null transformer results, typedef signatures, compile-time conformance
  of every type to Animation/Animatable/TweenSelf.
- **Difftests**, compared line for line with doubles in Dart's shortest
  round-trip form (so bit for bit), identical on the first run except the
  harness's end-of-body microtask flush:
  `difftest/animation_controller_test.go` + `flutter/test/
  animation_controller_test.dart`, 12 scenes, 1,861 lines through
  flutter_test's pump: forward/reverse/toggle with reverseDuration and
  `from`, animateTo/animateBack with 8 curves (incl. elastic/bounce/
  overshooting), already-at-target and zero durations, interruptions,
  repeat (reverse, min/max, count, period, initial value mid-range, min ==
  max, wider bounds), fling (default, negative, custom spring, wide
  bounds, velocity 0), animateWith/animateBackWith (gravity, spring,
  friction), unbounded controllers, stop(canceled: false), value
  clamping, resync (incl. to a muted ticker: "silenced"), dispose,
  timeDilation 3 and 0.37, AnimationBehavior normal vs preserve under
  reduced motion (animateTo at 5%, fling at 200×, repeat), CurvedAnimation
  with the reverse-curve direction lock and curve changes,
  TrainHoppingAnimation (crossing trains, ties in both modes, equal
  starts, status dedupe after a hop), ProxyAnimation parent switches,
  ReverseAnimation, AnimationMean/Max/Min (incl. ±0.0/NaN for doubles,
  ints), every toString form (hashes masked) and TickerFuture
  completion/orCancel order. `difftest/tween_test.go` + `flutter/test/
  tween_test.dart`, 2,441 lines: Tween<double> (incl. ∞), Tween<Offset>
  and Tween<Size> through the dynamic operators, ColorTween (nulls, alpha,
  Display P3 → sRGB), Size/RectTween with nulls, Int/StepTween rounding
  (negative ranges, extrapolation), ReverseTween, ConstantTween,
  CurveTween, chains, fromCallback, tween.animate/drive, TweenSequence
  (uneven weights incl. 1e-3) and FlippedTweenSequence, a color
  sequence, AnimationStyle.lerp (durations, lerped curves, nulls, equal
  curves, lerps of lerps) and the lerp error messages.
- Mutations: 128 valid on the new code, 127 caught, 1 equivalent
  (dart:math min's `a == 0 && b.isNegative` clause: after the `a == 0.0`
  return it is unreachable for doubles). Caught only after adding
  difftest cases: Duration × fraction rounding (1000us × 2/3), the
  unbounded controller's infinite range, repeat's `>=` exit tie, fling
  with velocity 0, TrainHopping's `<=`/`>=` ties, its status dedupe after
  a hop and its equal-values start, dart:math max/min on ±0, Animatable's
  default toString, AnimationStyle.lerp of equal curves/durations and of
  lerped curves (field-wise ==).
- Parity: 265 → 302 of 2,889 exposed, 200 → 238 tested; animation
  17 → 54 of 54 (all tested).
### semantics (P2), with dart:ui's semantics.dart
- `packages/flutter/lib/src/semantics` ports binding, debug, semantics
  (in four files: semantics.go, semantics_node.go, semantics_owner.go,
  semantics_configuration.go), semantics_event and semantics_service
  (3.50.1): semantics 0 → 36 of 36. The debug role checks, merging
  (getSemanticsData), the traversal sort (vertical groups, horizontal
  groups, knots, sort-key groups), traversal grafting (traversal
  parent/child identifiers and the traversal transform), dirty tracking
  and sendSemanticsUpdate, action routing through merged nodes and by
  position, the configuration absorb/compatibility rules and the debug
  output (toStringDeep in both dump orders, SemanticsData.toString/toJson)
  are all ported. Flutter's quirk that `onMoveCursorForwardByWord=` /
  `onMoveCursorBackwardByWord=` store the handler in the *character*
  getters is kept.
- **dart:ui** `ui/go/semantics.go` ports semantics.dart in full
  (SemanticsAction, SemanticsFlag(s), CheckedState, Tristate, roles,
  input types, validation results, hit-test behavior, StringAttribute and
  subclasses, SemanticsUpdateBuilder/SemanticsUpdate) with the engine's
  `testing/dart/semantics_test.dart` (4/4). `platform_dispatcher_semantics.go`
  adds Locale, SemanticsActionEvent, AccessibilityFeatures, a minimal
  FlutterView (implicit view, id 0; updateSemantics), the dispatcher's
  semantics/accessibility callbacks and the engine entry points
  (`PlatformConfiguration.UpdateSemanticsEnabled/UpdateAccessibilityFeatures/
  DispatchSemanticsAction`); `text.go` adds TextDirection, TextAffinity,
  TextPosition, TextRange. **ENGINE GAP:** no accessibility bridge: the
  native builder records every call and `ui.EngineSemanticsUpdateOf`
  returns it; setSemanticsTreeEnabled/updateSemantics reach an embedder
  client implementing `PlatformConfigurationSemanticsClient`, and are
  recorded (`ui.SemanticsTreeEnabled`).
- **Dependencies ported (partial, outside semantics):** package:vector_math
  2.4.4 Matrix4/Vector3/Vector4 subset in `src/pub/vector_math` (pinned
  2.4.3 is not in the pub cache; the code used is unchanged between them);
  painting's MatrixUtils subset, debugDescribeTransform, TransformProperty;
  services' TextSelection (full), StandardMessageCodec (full, maps as
  ordered `[]dart.MapEntry`), BinaryMessenger, BasicMessageChannel and
  `SystemChannels.Accessibility`. ServicesBinding is not ported: a channel
  without a messenger uses `services.DefaultBinaryMessenger` (nil: the
  send fails, as without a binding in Flutter). `src/dart` gains
  `ListSort` (the VM's Sort.sort: insertion sort up to 33 elements,
  dual-pivot quicksort above; its tie order decides traversal order),
  `CompareDoubles`, `CompareStrings` (UTF-16), `DoubleTryParse`, checked
  against Dart output.
- **Shared fixes:** foundation's EnumProperty/describeEnum printed
  `TextDirection.ltr` for dart:ui/scheduler-style enums (whose `String()`
  is Dart's toString); enums with a `Name()` now print the name (and
  `dartToString` no longer doubles the type). `DiagnosticsProperty.Level`
  now calls an overriding `IsInteresting` through self
  (AttributedStringProperty overrides it).
- **Design notes:** order-observable Sets/Maps keep insertion order
  (slices / ordered entries; see API_MAPPING). Stacking SchedulerBinding and
  SemanticsBinding in one Go binding needs the forwarding pattern in
  API_MAPPING (embedding both mixins makes the BindingBase methods
  ambiguous): an open design question for RendererBinding/WidgetsBinding.
- **Flutter's tests:** semantics_test 45/48 test cases ported (skipped:
  the RenderProxyBox/pumpFrame one; memory events via
  FlutterMemoryAllocations instead of leak_tracker; tags test uses a
  local tag instead of RenderViewport's), custom_semantics_action_test
  1/1, semantics_binding_test 3/6 (3 need widgets), both
  set_semantics_tree_enabled tests, semantics_node_send_event_test 1/1,
  semantics_service_test 1/1 (adapted to a mock messenger and the implicit
  view). Skipped entirely (they pump widgets): semantics_owner_test (5),
  semantics_update_test (5), traversal_order_test (1); their behaviors
  (merged custom-action routing, traversal grafting, hit-test filtering)
  are in the difftest. message_codecs_test's StandardMessageCodec group
  4/4; matrix_utils_test getAsTranslation (adapted). Own tests for the
  binding's action dispatch, accessibility features, events, Locale,
  flags, the builder record, sort/compare/parse.
- **Difftest** (`difftest/semantics_test.go` +
  `flutter/test/semantics_test.dart`, ~300 log entries, ~1800 lines,
  identical on the first run, hashes masked): attributed strings (UTF-16
  offsets), Locale, TextSelection, custom action ids, flag/action/role
  orders, label builder, every event's toMap and its StandardMessageCodec
  bytes, codec edge values; grid/knot/rtl/sort-key/40-child quicksort
  traversal orders, rotated/scaled children; a deep merge (labels across
  text directions, tags, controlsNodes, validation, hint overrides, custom
  actions, blocked actions) with toStringDeep at hidden, data and toJson;
  configuration absorb, copy and 15 compatibility pairs; an owner sending
  updates (every SemanticsUpdateBuilder.updateNode/updateCustomAction
  argument logged through a spy builder the binding returns), dirty
  propagation from merged leaves, detach/reattach; traversal grafting with
  translated/scaled/rotated transforms; 41 debug role check cases; error
  messages; action arguments routed through a merge boundary.
- Mutations: 97 on the semantics, dart:ui, services, painting, vector_math
  and src/dart ports (driver: each replaced once, go vet, the package tests
  plus the semantics difftest). 94 caught, 3 equivalent: the
  common-ancestor walk with `>` instead of `>=` (both sides still climb the
  same nodes in the same order), the `attached` filter on grafted traversal
  children (detached nodes are always removed from the traversal sets
  first) and DoubleTryParse's empty-exponent check (strconv rejects "1e"
  anyway). Caught only after adding cases: toStrings' isFocusable, flag
  merging of one-sided booleans, Locale's '' vs null country (both ways),
  SemanticsData == on rect, identity transforms stored as null, a knot edge
  in (pi/2, 3pi/4), overlapping hit-test siblings, absorb with a
  direction-less child label / heading levels / traversal identifiers, the
  deprecated flags bitmask, the platform's single semantics handle, action
  listeners removed mid-dispatch, the int32 codec boundary, a projective
  matrix inverse, child reordering marking dirty.
- Parity: 265 → 334 of 2889 exposed (tested 200 → 246): semantics 0 → 36
  (24 tested), dart:ui 28 → 51, services 4 → 11, painting 0 → 3.
### services, part 1: codecs, platform channels, binary messenger (P3)
- Ported from 3.50.1 (`packages/flutter/lib/src/services`, beside the old
  text-editing files): message_codec (MessageCodec, MethodCall,
  MethodCodec, PlatformException, MissingPluginException), message_codecs
  (Binary/String/JSONMessage/JSONMethod/StandardMessage/StandardMethod
  codecs; StandardMessageCodec extensible through `self`), binary_messenger,
  platform_channel (BasicMessageChannel, MethodChannel,
  OptionalMethodChannel, EventChannel on a broadcast controller,
  debugProfilePlatformChannels stats with Dart's stable sort and
  toStringAsFixed), system_channels (all 24), debug, service_extensions,
  binding (ServicesBinding **partial**: defaultBinaryMessenger /
  createBinaryMessenger / _DefaultBinaryMessenger, channelBuffers,
  rootIsolateToken, system/accessibility/lifecycle/platform handlers with
  Dart's lifecycle transition generator and its debug check, app exit,
  initializationComplete, the evict/profilePlatformChannels extensions,
  SystemContextMenuClient; not: keyboard, restoration, TextInput, licenses
  from the asset bundle, SystemChrome's hook, view focus) and
  _background_isolate_binary_messenger_io (as
  `background_isolate_binary_messenger_io.go`; ENGINE GAP: no isolates, send
  panics). flutter_test: `test_default_binary_messenger.go`
  (TestDefaultBinaryMessenger, AllMessagesHandler, the
  TestDefaultBinaryMessengerBinding mixin, mock_event_channel's
  MockStreamHandler, `AddTearDown` + `SetCurrentTest`).
- dart:ui: ChannelBuffers in full (control channel decoding, the legacy
  `resize\r` form with the VM's int.parse, overflow warnings through a
  debug-only printDebug, `ChannelUpdateClient`), and platform messages
  (`platform_message.go`: sendPlatformMessage, onPlatformMessage,
  _dispatchPlatformMessage, response ids, RootIsolateToken,
  initialLifecycleState, AppExitResponse/AppExitType, the engine's
  PlatformMessage/PlatformMessageResponse and
  `PlatformConfiguration.DispatchPlatformMessage`). ENGINE GAP: messages
  reach a host only through an embedder implementing
  `ui.PlatformMessageHandler`; otherwise (src/app, tests) they get the
  default PlatformView's empty reply, posted to the sender's loop.
- Support: `src/async` broadcast StreamController (async + sync variants;
  16-scene difftest exact), `src/dart` UTF-8 and JSON (the VM's decoder
  state machine, `_JsonUtf8Parser` and `json.decode` with every error
  message/offset, JsonUtf8Encoder with lone surrogates; 63 encode + 167×2
  decode + 76 UTF-8 cases, exact), `dart.FormatException`/`Exception`/
  `TypeError`/`ToString`. **Found:** `dart.DoubleToString` printed exact
  digits for integral doubles in [2^53, 1e21) (Dart prints the shortest
  round-trip digits padded with zeros): fixed.
- Decisions (API_MAPPING "Platform channel values"): Dart int ↔ Go int
  (decoded 32- and 64-bit ints are both `int`; all Go integer kinds
  encode); decoded maps are `map[any]any` / `map[string]any` (order lost);
  Go maps encode in sorted key order, `[]dart.MapEntry` in its order.
- **Found by the difftest:** a Dart map merges an int key with an equal
  double key (and 0 / 0.0 / -0.0) keeping the first key, but only when
  their hashCodes match too: the VM hashes an integral double as the int it
  casts to, so 2^53+1 and 2^53.0 (==, different hashes) stay apart, and on
  arm64 the cast saturates, so 2^63.0 merges with 2^63-1. Go's decoder
  reproduces this (`dartMapKey`). Also: utf8.decode drops a leading BOM in
  StandardMessageCodec strings; a success envelope ignores trailing bytes,
  an error envelope with any nonzero flag is an error; `ChannelBuffers`
  only drains in a microtask, so a binding constructed outside a running
  loop must have its microtasks flushed (the test helpers do).
- Merge with the semantics port (master b305082): its partial services
  (StandardMessageCodec decoding maps to `[]MapEntry`, BasicMessageChannel,
  SystemChannels.accessibility, `services.DefaultBinaryMessenger`) is
  replaced by this port; StandardMessageCodec's methods take value
  receivers so `services.StandardMessageCodec{}` still works; `MapLookup`
  and its tests and the TextSelection tests are kept; the semantics service
  tests install their mock on a replaced SystemChannels.Accessibility and
  compare decoded maps in any order (as Flutter's `equals`).
- Flutter's tests: message_codecs 14/14 (+ vm 4/4), platform_channel 20/20
  (the two `invokeMethod<List/Map>` TypeError expectations dropped: Go's
  InvokeMethod returns any), platform_messages 1/1,
  default_binary_messenger 2/2, set_method_call_handler 1/1,
  use_platform_channel_without_initialization 1/1, channel_buffers
  (services) 1/1, binding_lifecycle 1/1, binding 2/7 (licenses, memory
  pressure, textInput, keyboard, view focus need unported libraries),
  lifecycle 3/3 (adapted: states sent on flutter/lifecycle; the initial
  state through `PlatformConfiguration.UpdateInitialLifecycleState`), the
  engine's channel_buffers_test 16/16; plus own tests (Go value mapping,
  errors, extension by embedding, lifecycle transitions, system channels,
  context menu, profiling, event close/cancel, platform message dispatch).
- Difftests, exact: `message_codecs` (238 cases: every codec's bytes and
  decoded values, hand-made and corrupt inputs, map key merging, error
  texts, toString), `platform_channels` (≈150 log lines: invoke/handle/
  basic/buffers/events/pending/profiling scenes timed against a microtask
  chain; identical on the first run), `channel_buffers` (11 scenes),
  `broadcast_stream`, `dart_convert`.
- Mutations: services/flutter_test/ui platform messages ≈97 valid, 93
  caught, 4 equivalent (`float64(i) == k` after the hash match;
  `awaitOrNull`'s yield — every use is a coroutine's first await, where
  `Async` already completes in a microtask; the response's double-complete
  assert and the reply copy, both unreachable from Go callers); caught only
  after adding cases: uint64 at MaxInt64, the 2^63 key, Go map int-key
  order across a power of ten, PlatformException details in a handler's
  envelope, the event channel's close await and cancel, the setMethodCall
  assert's message, nil handler futures, profiling (clear, sort, postfix),
  initializationComplete at construction, the initial-state guard, the
  all-messages handler, the keyboard mock. ChannelBuffers 79/74/5,
  broadcast streams 34/33/1, dart:convert 56/54/2 (equivalents in the
  forks' reports).
- Parity: 371 → 400 of 2889 exposed (tested 284 → 309); services 11 → 31,
  dart:ui 51 → 60.

### Bindings: one composition pattern for binding mixins
- Problem: each binding mixin struct embedded `foundation.BindingBase` as
  its "super", so a binding could embed only the topmost mixin; the
  semantics port forwarded SemanticsBinding's methods to a field, and
  ServicesBinding's super was typed `scheduler.SchedulerBinding` (fixed
  order). Go can't embed a type parameter, so a static mixin chain over an
  arbitrary composition is impossible.
- Pattern (`foundation/binding_mixins.go`, API_MAPPING "Bindings"): a
  binding embeds `foundation.BindingBaseBase` and the mixin structs (which
  embed nothing) in Dart's `with` order, calls each mixin's `SetSelf(b)`,
  then `b.Construct(b, &b.M1, &b.M2, ...)`. Methods defined by several
  mixins (`InitInstances`, `InitServiceExtensions`, `Unlocked`, a mixin's
  method a later mixin overrides) are ambiguous in Go, so the binding
  defines them as calls to the last mixin in `with` order that defines
  them; the compiler lists the missing ones. Dart's `super.m()` inside a
  mixin is `foundation.BindingSuper[I](self, mixin)` (the closest mixin
  before it implementing I, else BindingBase) or
  `foundation.BindingBaseSuper(self, mixin)` for BindingBase's methods: a
  dynamic walk of the binding's mixin list, i.e. Dart's linearization,
  whatever the composition. Mixin-on-mixin overrides
  (TestDefaultBinaryMessengerBinding.createBinaryMessenger over
  ServicesBinding's) work the same way.
- Moved onto it: SchedulerBinding, ServicesBinding, SemanticsBinding,
  flutter_test's TestDefaultBinaryMessengerBinding and
  AutomatedTestWidgetsFlutterBinding (now BindingBase with Scheduler,
  Services, Gesture, Semantics, TestDefaultBinaryMessenger, as
  TestWidgetsFlutterBinding orders them; the loop is flushed after
  construction for the channel buffers' microtask and the engine's posted
  reply), src/app's binding, and every test/difftest binding (the
  semantics forwarding wrappers are gone).
- Verified by `foundation/binding_mixins_test.go` (three toy mixins:
  init order, super of a mixin's own method, unlocked order, errors) and
  the `binding_composition` difftest: BindingBase with L0, Gesture, L1,
  Scheduler, L2, Services, L3 (on Services), Semantics, L4 (on
  Scheduler), logging mixins interleaved, compared line by line with Dart:
  initInstances/initServiceExtensions/unlocked nesting, the X.instance
  statics set at each step, createBinaryMessenger and
  handleAppLifecycleStateChanged chains (incl. through the lifecycle
  channel), and the 10 service extensions in registration order.

### gestures (P4), with dart:ui's pointer.dart and key.dart
- `packages/flutter/lib/src/gestures` ports the whole library (3.50.1),
  one Go file per Dart file (hit_test.dart is `hittest.go`): events (all
  PointerEvent classes and their transformed variants), converter,
  hit test, pointer router, signal resolver, arena, team, recognizer
  (GestureRecognizer/OneSequence/PrimaryPointer), eager, tap, multitap
  (DoubleTap/MultiTap/SerialTap), long press, force press, monodrag
  (Vertical/Horizontal/Pan), drag, drag details, multidrag, scale,
  tap_and_drag, velocity tracker (+ iOS/macOS trackers), lsq solver,
  resampler, gesture settings/details, constants, debug, and GestureBinding
  (packet handling, hit tests kept per pointer, dispatch with each entry's
  transform, close/sweep, cancelPointer, locking, signal resolution,
  onHitTest, the touch resampler on the sampling clock,
  FlutterErrorDetailsForPointerEventDispatcher). dart:ui: pointer.dart and
  key.dart in full, PlatformDispatcher.onPointerDataPacket/onKeyData/
  onHitTest, the engine's packet layout (`DispatchPointerDataPacket`,
  `PackPointerDataPacket`), window.dart's GestureSettings and the view's
  devicePixelRatio/gestureSettings (ENGINE GAP: no window metrics; an
  embedder sets them with `UpdateViewGestureMetrics`). `src/dart`
  Stopwatch; vector_math additions (perspectiveTransform, rows/columns).
- Done with five forks in parallel worktrees (base layer; tap family;
  drag family; scale; tap-and-drag + cross-family tests), each with an
  exact difftest, then merged here; the binding, the recognizer bases and
  the test helpers were ported first so the families could build on them.
- flutter_test: `TestPointer` (test_pointer.dart), gesture_tester.dart's
  `testGesture`/`GestureTester` (`RunTestGesture`), and the test binding's
  `DebugSamplingClock` on the test's fake clock (the tester's loop, not an
  inner FakeAsync's, as testWidgets' clock is).
- **Migration:** widgets' ad-hoc `PointerRouter` arena and its
  VelocityTracker/least-squares copy (`velocity.go`) are gone. The test
  binding and src/app's binding (now BindingBase with GestureBinding,
  SchedulerBinding) hit-test the tree through
  `BuildOwner.HitTestGestures`, which adds the gesture targets a
  RenderPointerListener would: a TapGestureRecognizer per tap region and a
  VerticalDragGestureRecognizer per list, configured as ScrollableState
  does (fling limits from the physics, the platform's velocity tracker,
  hold on down, cancel → goBallistic(0), Scrollable's IgnorePointer while
  a scroll activity runs, wheel events through the PointerSignalResolver).
  `widgets.PointerRouter` is now a thin event source (TestPointer-like
  events to `GestureBinding.handlePointerEvent`, microtasks flushed after
  each event as flutter_test's awaits do). The tester's `Fling` ports
  flingFrom's positions exactly (`Offset.lerp`, `offset.distance`).
- **Result:** the drag-fling difftest, which differed by up to 2.6e-11,
  is now **exact** and compares exactly (the real tracker's arithmetic
  plus flingFrom's lerped positions); a new `scroll_gestures` difftest
  drives real gestures on lists on Android, iOS and macOS (each
  platform's tracker and fling limits, the bouncing motion-start
  threshold with event time stamps, holds and cancels under a competing
  outer tap, item taps) and matched on the first run.
- **Found:** Dart dispatches `event.transformed(entry.transform)` to every
  target, and the root entry's transform is the identity, not null, so
  even untransformed targets receive `_TransformedPointerXEvent`s (they
  still pass `is PointerDownEvent`); `_CombiningGestureArenaMember`
  prints as `Instance of '_CombiningGestureArenaMember'` in arena
  diagnostics; scale.dart in 3.50.1 has no pointer-scroll/scale signal
  handling (trackpad scale only through pan/zoom events).
- Difftests, all exact: `binding_composition` (58 lines), `gesture_binding`
  (≈300 lines: dispatch/transforms/arena/cancel/locking/errors/signals/
  resampling/base recognizers/teams/invokeCallback), `pointer_events`
  (4,170 lines: every event class and transform, converter at several
  DPRs, resampler, arena, router, resolver, hit test), `velocity_tracker`
  (2,512 lines: all three trackers, lsq solver degrees 0–3),
  `recognizers_tap` (1,051 lines, 37 scenes), `recognizers_drag` (10,132
  lines, 319 scenes), `recognizers_scale` (607 lines, 27 scenes),
  `recognizers_tap_and_drag`, `scroll_gestures`, and `drag_fling`
  tightened to exact.
  (`recognizers_tap_and_drag`: ≈14,000 lines, 455 scenes.) Every one
  matched Flutter on its first run once its scenes were written; the
  scenes added for mutations matched too.
- Flutter's tests (packages/flutter/test/gestures): arena 11/11,
  debug 5/5, events 25/25, hit_test 3/3, lsq_solver 5/5, pointer_router
  9/9, pointer_signal_resolver 5/5, resampler 10/10, velocity_tracker
  11/11, recognizer 7/7, gesture_binding 20/20, locking 1/1,
  gesture_binding_resample_event 1/1, team 2/2, tap 34/34, double_tap
  23/23, multitap 2/2, serial_tap 16/16, long_press 27/27, force_press
  9/9, drag 43/43, monodrag 5/6, multidrag 5/5, scale 15/15,
  tap_and_drag 30/30, transformed_scale 1/1 (adapted: the hit test's
  transform is put on the event). Skipped because they pump widgets
  (GestureDetector/Listener/Scrollable not ported): monodrag's
  "CustomScrollView" test, transformed_tap 0/3, transformed_double_tap
  0/2, transformed_long_press 0/3, transformed_monodrag 0/8 (a
  non-widget slop-under-scale test stands in), gesture_config_regression
  0/2, gesture_binding_resample_event_on_widget 0/1. Plus
  `foundation/binding_mixins_test.go` and own tests (arena default win,
  stopped boundary, perspective assert, pack/unpack, KeyData,
  GestureSettings, router up-elsewhere).
- Mutations: 334 valid, 309 caught, 25 equivalent (the binding pattern
  and GestureBinding/recognizer bases 47/46/1, the widgets bridge 11/9/2,
  base layer 66/63/3, tap family 45/44/1, drag family 68/63/5, scale
  42/36/6, tap-and-drag 55/48/7; each fork's equivalents are argued in
  its report: e.g. a second resolve of an already-resolved arena entry is
  a no-op, a NaN-free `max`'s -0.0 rule, the iOS bouncing drag start
  always resetting the motion-stop state so its time stamp never
  matters, the bridge adding no transforms so tap local = global).
  Caught only after adding scenes: arena close/sweep through hit-test
  targets, cancelPointer's microtask, a pointer held down across
  resampling, the velocity trackers on the binding's clock, filters on
  device kind vs buttons separately, the deadline at exactly its
  duration, an up beyond the slop, a team's captain winning, the bouncing
  min fling velocity, holds under a competing outer tap.
- Parity: 400 → 603 of 2889 exposed (tested 309 → 432); gestures 0 →
  188/188, dart:ui 60 → 75.

### dart:ui drawing over Impeller (P1 core): painting.dart in full

- **Scope.** Every declaration of painting.dart: Paint (with its 68-byte
  float32/int32 storage reproduced: values read back float32-rounded,
  alpha stored as 1 - a, blend mode XOR srcOver, and the 3.50.1 quirk that
  `strokeMiterLimit` returns the stored limit minus 4, so a new Paint
  reports 0.0 and prints "up to 0.0"), the drawing enums, MaskFilter,
  ColorFilter, ImageFilter (blur/dilate/erode/matrix/compose/shader),
  Shader/Gradient/ImageShader, Canvas (every method), PictureRecorder,
  Picture, Vertices, Shadow, Path/PathMetrics/Tangent, Image/codecs/
  ImmutableBuffer/ImageDescriptor/decoding, FragmentProgram/
  FragmentShader/uniform slots. dart:ui parity 75 → 133 of 195; what is
  left is text.dart, compositing.dart and window/platform pieces.
- **How it maps to the engine.** Canvas calls go to the Impeller SDK's
  display list builder, which is the engine's own DisplayListBuilder: the
  current matrix and save count are read back from it, so getTransform is
  the engine's float math for free. The SDK doesn't expose the builder's
  clip tracking, so `DisplayListMatrixClipState` (cull rects, AA
  round-out, difference cutouts, saveLayer bounds and the filters'
  `get_input_device_bounds`) and the impeller rect/matrix operations it
  needs (Invert, TransformBounds...) are ported in float32. Paths are Go
  (a port of SkPathBuilder) converted to SDK paths when drawn (conics to
  quads at a scale-aware tolerance). Picture.toImage renders the display
  list into an offscreen CAMetalLayer drawable and reads it back
  (src/platform/metal); the SDK hides its command queue, so the commits
  made during the draw are captured (the command buffer class's `commit`
  is wrapped) and waited for before the blit — without that the readback
  raced the GPU. Images are CPU pixels uploaded lazily as SDK textures in
  dart:ui's process-wide Impeller context, which src/app now also
  presents with.
- **Exactness: the engine's compiler.** The reference binary is arm64 and
  the engine builds with clang's default -ffp-contract=on: within one
  expression `a*b + c` becomes an FMA (left product first). Reproducing
  that made getLocalClipBounds after rotations exact (the cofactor
  expansion of Matrix::Invert is fused term by term); the Path helper
  needed an exact float32 FMA (round-to-odd; a float64 FMA double-rounds)
  and found that clang merges sinf+cosf of one value into Apple's
  `__sincosf_stret`, which differs from separate sinf/cosf in the last bit
  for ~2.6% of inputs (16 arc cases failed until it was called via cgo).
- **Difftests.** painting_api (87 cases: Paint state/toString, filters'
  toString/==, saturation matrices, argument errors, Shadow, enums, 40
  Canvas states: transforms, save counts, local/destination clip bounds
  under rotation/skew/perspective/saveLayer filters): exact. path (3408
  cases over 136 paths: bounds, contains grids and probes ±1/4096 off the
  outline, metrics/tangents/extractPath, transforms): exact.
  painting_pixels (39 scenes, Impeller on Metal vs flutter_tester's
  Impeller on Vulkan/SwiftShader): 24 bit-exact (gradients, images, image
  shader, color filters, invertColors, blur/dilate, mask blur ≤1, layers,
  shadow ≤1 once drawShadow gets the tester's device pixel ratio 3.0);
  anti-aliased edges differ by exactly one MSAA sample (64) on a few edge
  pixels (the GPUs' sample positions); circle ≤5; clip.rect is an ENGINE
  GAP (the SDK's clips record is_aa=false). image_codec (Impeller
  reference; the Skia backend fails some cases): PNG/GIF/APNG/BMP pixels
  exact; JPEG ≤4–13 (Go's IDCT vs libjpeg-turbo, scaled IDCTs), Metal's
  bilinear resize ≤2, PNG encoder bytes differ (pixels compared).
  fragment_shader: reflection/errors exact, pixels exact except one case
  at an 8-bit rounding tie (Metal correct, SwiftShader off by 1).
- **Found by the difftests:** saveLayer/draw blurs without a tile mode are
  decal (canvas.cc passes kDecal, drawPaint kClamp); Impeller draws a
  point-mode polygon as separate lines; Flutter's ColorFilter.matrix
  translation is 0..255 and the engine scales it (the SDK header's
  "0..255" comment is wrong: it passes the matrix straight through);
  calling a native method after endRecording throws "native peer has been
  collected"; the blur/dilate/erode NOP thresholds (SK_ScalarNearlyZero).
- **Mutations.** Canvas/Paint/filters: 22 valid, 21 caught (two only after
  adding difftest cases: a paint alpha whose float32 rounding differs,
  rotations 0.5 and 1.3 whose float-vs-double degree conversion differs);
  1 equivalent (Invert's `1.0/det` in double vs float: no difference
  found over 10^5 values). Path: 95 caught, ~25 equivalent, 11 below what
  grids/probes resolve (listed in the Path report). Fragment shaders: 51 of
  52 (1 equivalent). Images/codecs: see image_codec_test.go.
- **ENGINE GAPs (Impeller SDK):** Path.combine (Skia PathOps, 18k lines;
  panics with UnimplementedError); Path.transform with a perspective
  crossing w=0 (SkEdgeClipper); clips are never anti-aliased; no gamma
  color filters, no ColorFilter as ImageFilter, no composition of a
  non-matrix color filter with invertColors; drawVertices ignores colors
  and texture coordinates (triangles as paths); blur `bounds`; sampling is
  nearest/linear only (medium/high = linear); toImageSync renders eagerly
  and ignores targetFormat; Picture.approximateBytesUsed is an estimate;
  image decoding is Go's (JPEG/WebP/ICC/resize differences above); shader
  samplers always linear; Canvas.drawParagraph waits for text.dart
  (`ui.EngineDrawParagraph` meanwhile).
- **src/app** builds each frame with ui.PictureRecorder/ui.Canvas (the
  interim rendering.Canvas is an adapter over it) and presents the
  picture's display list with dart:ui's Impeller context. The counter demo
  ran for 8 s with no errors (window opened; screenshots don't work on
  this machine, so the pixels were not inspected on screen — the offscreen
  pixel difftest covers rendering).

### dart:ui text (P1): text.dart on the engine's own SkParagraph

- **Scope.** Every declaration of text.dart: FontStyle, FontWeight,
  FontFeature (all named constructors), FontVariation, GlyphInfo,
  TextAlign, Hyphens, TextBaseline, TextDecoration, TextDecorationStyle,
  TextLeadingDistribution, TextHeightBehavior, TextStyle/ParagraphStyle/
  StrutStyle with their encodings, toString and ==, TextBox,
  ParagraphConstraints, BoxHeightStyle/BoxWidthStyle,
  PlaceholderAlignment, LineMetrics, Paragraph (every query),
  ParagraphBuilder (styles, placeholders and their scales),
  loadFontFromList (with the fontsChange message), and
  Canvas.drawParagraph (the `ui.EngineDrawParagraph` stand-in is gone).
  TextSelection and FontLoader are services', not dart:ui's. dart:ui parity
  133 → 158 of 195 (no text.dart name missing).
- **How it maps to the engine.** The SDK's typography C API is a small
  subset (no boxes, positions, placeholders, strut, spacing, shadows,
  features or variations), but its ImpellerParagraphBuilder and
  ImpellerParagraph hold the engine's txt::ParagraphBuilder and
  txt::Paragraph — abstract classes whose every method is virtual — over
  SkParagraph, HarfBuzz and ICU compiled into libimpeller.dylib. So the
  bridge (`src/impeller/typography_bridge.cc`) declares layout mirrors of
  the engine headers at the pinned revision (`txt_mirror.h`: txt's
  TextStyle/ParagraphStyle/LineMetrics, Skia's GlyphInfo/LineMetrics,
  DlPaint, the interop objects) and calls through the vtables: no hidden
  symbol, no copied engine code, and the layout is exactly flutter_tester's
  (the same code). The builder is the SDK's: the encoded ParagraphStyle
  is written into an ImpellerParagraphStyle's txt::ParagraphStyle, pushed
  once to make the SDK create its ParagraphBuilderSkia, and popped
  (SkParagraph's builder is then in its constructor's state: one empty
  block with the paragraph's text style — checked in
  ParagraphBuilderImpl.cpp). lib/ui/text's natives (paragraph_builder.cc's
  decoding of the Dart encodings, paragraph.cc's result encodings) are
  ported in the bridge; text.dart is Go. The SDK's libc++ (`std::__fl`,
  LLVM 21) has the system libc++'s object layout and both allocate with
  malloc/free, so strings, vectors, maps and optionals cross the boundary;
  a self-test reads values set through the SDK's public setters back
  through the mirrors (`go test ./src/impeller`).
- **flutter_tester's fonts.** `flutter test` runs flutter_tester with
  --use-test-fonts: FlutterTest, Ahem and Cough (extracted from
  runtime/test_font_data.cc by `src/cmd/gentestfonts`; kCoughFont's array
  holds 4 bytes more than its declared length, which is what the engine
  streams) are registered in a second typography context, and
  txt::TestFontManager (any other family → FlutterTest) is reimplemented
  as an SkFontMgr subclass over its provider (Skia's SkFontMgr vtable at
  the pinned revision, with the new onMatch/onFallback), placed in the
  font collection's test slot with fallback disabled.
  `ui.EngineUseTestFonts()` turns it on; flutter_test's tester does.
- **Difftest (text, exact).** 65 scenes from `difftest/text_scenes.json`
  (shared with Dart), 90 layouts: FlutterTest (wrapping, newlines,
  trailing spaces, RTL Hebrew and bidi, emoji ZWJ/flags/skin tones,
  combining marks, CJK with a supplementary char, tabs, mixed sizes,
  letter/word spacing incl. negative, height/kTextHeightNone/even leading,
  TextHeightBehavior, struts (forced, not forced, leading only, fallback family, Ahem), an empty paragraph whose last style resets the height, placeholders on the baseline with the default offset, maxLines, ellipsis incl.
  RTL, all alignments, every placeholder alignment and scale, soft hyphens
  with both Hyphens, Ahem/Cough/unknown families, locales) and real fonts
  (Roboto: ligatures on/off, tnum/smcp, kerning with spacing, bold/italic
  mixes, strut, ellipsis, placeholders, justify; Noto Naskh Arabic RTL and
  mixed; Roboto Slab's wght axis via fontVariations and fontWeight). Each
  layout dumps the metrics, line metrics (computeLineMetrics and
  getLineMetricsAt −1..n), getLineNumberAt −1..n+1, boxes of every
  one-unit range and of 4 ranges × 12 height/width styles, placeholder
  boxes, getGlyphInfoAt −1..n, positions and closest glyphs on a grid,
  word and line boundaries at every offset with both affinities. Plus 37
  API cases (incl. a family resolved before a font of that name is loaded: loading clears the caches) (toString/== matrices of TextStyle/ParagraphStyle/StrutStyle,
  the encodings' mutation by pushStyle/build, FontWeight/FontVariation
  lerps, every FontFeature constructor, errors: asserts, malformed UTF-16,
  native calls after build/dispose, drawParagraph's checks). Matched on
  the first run. Flutter plain and --enable-impeller write byte-identical
  references (the layout is SkParagraph's on both); Go checks both.
- **Difftest (pixels).** 16 drawParagraph scenes against flutter_tester
  --enable-impeller: 12 bit-exact (FlutterTest glyphs at integer and
  fractional offsets, colors with alpha, background, shadow, gradient and
  stroked foregrounds through the copied DlPaint, RTL ellipsis, Roboto at
  20 and 9 px, no pushed style), a blurred shadow within 1; decorations differ on one anti-aliased row (one MSAA
  sample, as lines in painting_pixels), rotated glyphs by ≤13 on edges.
- **Moved onto ui.Paragraph:** flutter_test's tester and src/app lay text
  out with `rendering.UITextEngine` (dart:ui ParagraphBuilder; test fonts in
  tests), replacing the ad-hoc FixedTextEngine and the SDK-API paragraphs;
  every widget/editing/scene difftest still passes unchanged.
- **Mutations.** 65 (Go encodings/equality/toString/lerps, the builder
  and paragraph wrappers, and the C++ bridge's decoding, test font
  manager and cache clearing): 61 caught. Eight were first missed and
  caught after adding cases (strut forceStrutHeight false, strut with only
  leading, strut fallback families, the placeholder's default baseline
  offset, shadow blur sigma, FontWeight.lerp rounding, kTextHeightNone's
  height override — visible only in an empty paragraph — and the font
  cache clearing). 4 equivalent: StrutStyle == on the view instead of
  the whole buffer (the bytes past the view are always zero); the
  `offset == line.end` term of getLineBoundary's downstream rule (implied
  by the other two); the strut's default `""` family (with test fonts an
  empty family list resolves to FlutterTest too); the initial pop of the
  style the SDK's builder needs (its style equals the base style; it only
  differs for unbalanced pops, which are undefined in the engine). A first
  run filled the disk (each mutation of a cgo file adds a build of every
  dependent package): the mutation helper now cleans the build cache
  below 15 GB free.
- **Divergences:** families loaded with loadFontFromList are matched
  exactly (the engine's dynamic font manager lowercases ASCII names); the
  builder's impeller flag is always set (it selects the painting path
  only).
