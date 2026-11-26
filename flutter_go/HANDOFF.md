# Handoff (2026-11-25)

Start here when continuing flutter-go in a new session.

## Where things stand

- **Goal:** [GOAL.md](GOAL.md): Flutter 3.50.1 with Go in place of Dart,
  measured by full API parity. Progress: [docs/PARITY.md](docs/PARITY.md) (in flutter_go/)
  (regenerate with `go run ./flutter_go/cmd/apiparity -flutter <clone> -o flutter_go/docs/PARITY.md`).
- **Pin:** `FLUTTER_TAG` (3.50.1) and `IMPELLER_SHA` (its engine, 4d35714).
  The repository root is every non-Dart file of the tag at its upstream
  path, never edited; Go files sit beside the Dart files they replace
  (dart:ui's in `engine/src/flutter/lib/ui/go/`), and everything else of
  flutter-go is in `flutter_go/`. Verify with
  `./flutter_go/scripts/sync_upstream.sh --check`.
- **Naming rules:** [docs/API_MAPPING.md](docs/API_MAPPING.md). Every
  divergence from Flutter's API gets listed there.
- **Done:** P0 pin, layout (upstream tree at the root, Go beside the Dart
  it replaces) and `physics`; `flutter_go/async` (dart:async semantics);
  dart:ui `math`, `lerp` and `geometry` (P1 started). Stages 0–10 and G1–G4
  (see PLAN.md) are verified on 3.50.1. Parity: see flutter_go/docs/PARITY.md.
- **Next, in order:**
  1. Done: `RSuperellipse.Contains` (engine C++ ported to Go).
  2. Done: `rendering`/`widgets` use `ui.Offset/Size/Rect` (aliases left in
     `rendering/geometry.go`; switch callers to `ui` and drop them). Color,
     BoxConstraints, EdgeInsets and Alignment still have flutter-go names:
     port them as `ui.Color`, `rendering.BoxConstraints`,
     `painting.EdgeInsets`/`painting.Alignment`.
  3. Done: foundation, 137/137 top-level names (binding, SynchronousFuture,
     licenses, timeline, memory_allocations, isolates, consolidate_response,
     service_extensions, node, annotations, capabilities, debug), on an
     `async.Future` interface, a Stream subset, `flutter_go/developer` and
     `flutter_go/io`. Still to do for foundation users: package:listen
     (ChangeNotifier, Listenable, ValueNotifier; then BindingBase's
     `_initListenable` and the memory-allocation hooks) and vector_math's
     Matrix4. `ui.PlatformDispatcher` is minimal (platformBrightness only):
     the scheduler port adds frame scheduling and `AppLifecycleState`.
  4. Port `SchedulerBinding`/`Ticker`/`TickerFuture` into
     `packages/flutter/lib/src/scheduler` on `flutter_go/async`; move
     `animation` onto it (it still holds the old scheduler).
  5. Split `widgets` (gestures code) and `services` (text-editing model
     under non-Flutter names) by library.

## Environment

| Need | Where |
|---|---|
| Impeller SDK (not in git) | `./flutter_go/scripts/fetch_impeller_sdk.sh` (into `flutter_go/third_party/`) |
| Flutter SDK + source to port from | `~/Projetos/jonathanperis/flutter`, checked out at tag 3.50.1 (detached). Don't move it without changing the pin. |
| Reference tests | `FLUTTER=~/Projetos/jonathanperis/flutter/bin/flutter ./flutter_go/scripts/difftest.sh` |
| Go vs Flutter perf | `scripts/perf_compare.sh` (needs Xcode) |

## How each item is done (keep doing this)

1. Read the behavior in the 3.50.1 source and port it; don't reimplement
   from memory.
2. Port Flutter's own tests for it (`packages/flutter/test/<lib>`).
3. Write the Flutter side of a difftest (`difftest/flutter/test/`), run it,
   and **read the output** before trusting it.
4. Write the Go twin (`difftest/*_test.go`) and make it match.
5. Mutation-check the key rules (`perl -0pi` the rule, run the tests,
   restore). A mutation that doesn't compile is invalid, not "caught".
   Flutter's own tests often use loose tolerances; physics needed the
   difftest to catch two rules.
6. `go vet ./... && go test ./...`, `sync_upstream.sh --check`, regenerate
   PARITY.md, add findings to PLAN.md, commit the item on its own, push.

## Gotchas

- zsh: never name a shell variable `path` (it is tied to `PATH`).
- Bit-exact parity with Flutter is achievable (physics, scroll simulations,
  geometry and color are exact); see the exactness rules above.
- Window screenshots (`screencapture -l`) stopped working on this machine;
  environmental.
- Benchmarks vary about ±30% between runs here.
- Equal widgets skip rebuilds (Go's stand-in for `const`), a documented
  divergence.
