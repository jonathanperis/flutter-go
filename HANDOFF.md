# Handoff (2026-11-25)

Start here when continuing flutter-go in a new session.

## Where things stand

- **What it is:** Flutter's widget → element → render-object architecture
  written in Go, drawn by Impeller's standalone C API (no Dart VM, no
  Flutter engine). macOS/Metal only.
- **Done:** stages 0–10 in [PLAN.md](PLAN.md) (render tree, widgets,
  input, layout parity, performance, lazy lists, animations, flings, text
  input) and backlog items **G1–G4** in [GOAL.md](GOAL.md)
  (InheritedWidget/MediaQuery, repaint boundaries and layers, Flutter's
  VelocityTracker and gesture-arena drag rules, bouncing physics).
- **Next:** **G5**, a variable-height lazy list (Flutter's
  `RenderSliverList`), then G6–G10 in order. GOAL.md defines "done" for
  every item.
- **Verification:** 55 scenes compared with real Flutter (layout, lists,
  animations, flings, drags, bouncing, editing, rebuilds, repaints). In
  each area, the key rules were mutation-checked: breaking one makes a
  scene fail.

## Environment

| Need | Where |
|---|---|
| Impeller SDK (not in git) | `./scripts/fetch_impeller_sdk.sh` (pinned by `IMPELLER_SHA`) |
| Flutter SDK + source to port from | `~/Projetos/jonathanperis/flutter` (local clone of upstream at `06f1c24`, matching `IMPELLER_SHA`; don't update it without updating the pin) |
| Reference tests | `FLUTTER=~/Projetos/jonathanperis/flutter/bin/flutter ./scripts/difftest.sh` |
| Go vs Flutter perf | `scripts/perf_compare.sh` (needs Xcode; installed) |

The T3 project `flutter-go` has these as scripts, and fetches the SDK for
new worktrees.

## How each item has been done (keep doing this)

1. Read the behavior in the Flutter clone's source and port it; don't
   reimplement from memory.
2. Write the Flutter side of the comparison first (`difftest/flutter/test/`),
   run it, and **read the output** before trusting it. It was wrong twice:
   missing `DefaultTextEditingShortcuts`, and a "tappable" scene whose child
   wasn't hit-testable.
3. Write the Go twin (`difftest/*_test.go`) and make it match.
4. Mutation-check: break the key rule and confirm a scene fails. Treat a
   mutation that **doesn't compile** as invalid, not as "caught" or "missed".
5. `go vet ./... && go test ./...`, add findings to PLAN.md, mark GOAL.md,
   commit the item on its own, push.

## Gotchas

- Window screenshots (`screencapture -l <window id>`) stopped working on
  this machine mid-session; it's environmental, not a code regression.
- Benchmarks vary about ±30% between runs here; repeat before concluding.
- Equal widgets skip rebuilds (Go's stand-in for `const`), a deliberate and
  tested divergence from Flutter for new-but-equal subtrees.
- Go zero values need deliberate defaults (`Color{}` is transparent,
  `Bold` can't be unset), documented in PLAN.md.
