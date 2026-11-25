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
2. **Make the next stage the hard parts,** not more box widgets: scrolling
   (needs viewport + slivers), text input (IME), and implicit animations.
   These stress the frame loop and are where an API in Go will either hold
   up or not.

Gaps that remain: InheritedWidget, repaint boundaries/layers, relayout
boundaries, baseline alignment, RTL, accessibility, Retina scale detection
(the window reported scale 1.0), and every platform except macOS.
