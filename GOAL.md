# flutter-go: standing goal

**Goal:** close every gap the stage findings in [PLAN.md](PLAN.md) list as
"not done" that can be verified, one item at a time, without waiting for
approval between items.

## Definition of done (every item)

1. Behavior is ported from the fork's Flutter source (`../flutter`), not
   reimplemented from memory.
2. A differential test compares it with real Flutter (`scripts/difftest.sh`)
   wherever Flutter can produce a reference. The reference output is read
   and sanity-checked before Go is made to match it.
3. A mutation check: breaking the key rule must fail a test. If it doesn't,
   the test is strengthened first.
4. `go vet ./...` and `go test ./...` pass.
5. Findings (including mistakes) are added to PLAN.md, and the item is
   committed on its own.

Items that can't be verified this way say so explicitly and are marked
partial, not done.

## Backlog (in order)

| # | Item | Verified by | Status |
|---|---|---|---|
| G1 | `InheritedWidget` with dependency tracking; `MediaQuery`, `DefaultTextStyle` on top; equal widgets skip rebuilds (Go's stand-in for `const`) | Which widgets rebuild at each step matches Flutter | ✅ |
| G2 | Repaint boundaries and retained layers | Paint counts per frame match the rules; rerun `perf_compare.sh` | |
| G3 | Flutter's `VelocityTracker` (least-squares drag velocity) | `tester.fling` scenes: same release velocity and offsets as Flutter | |
| G4 | `BouncingScrollPhysics` (macOS/iOS overscroll and spring-back) | Fling and overscroll scenes on the iOS/macOS platform | |
| G5 | Variable-height lazy list (`RenderSliverList`) | Built indices, positions and offsets vs Flutter | |
| G6 | `Opacity` via `saveLayer`, `AnimatedOpacity` | Animated values vs Flutter; paint ops checked | |
| G7 | Retina scale detection in the window | Scale equals the display's backing scale (unit test + run) | |
| G8 | Text editing: undo/redo, double-click word select, drag-select | Editing scenes vs Flutter | |
| G9 | Tab focus traversal | Focus order vs Flutter's default traversal | |
| G10 | IME composition display (native `NSTextInputClient`) | Composing-range model tests; visual check needs a person | |

## Out of scope (unless asked)

Accessibility/semantics, right-to-left text, platforms other than macOS,
Material/Cupertino widget libraries, and production packaging. Each is a
project of its own; none is needed to answer whether Flutter's
architecture works in Go.
