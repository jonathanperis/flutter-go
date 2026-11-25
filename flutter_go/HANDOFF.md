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
- **Done:** P0 pin + upstream; `physics` fully ported. The original stages
  0–10 and G1–G4 (see PLAN.md) are verified on 3.50.1.
- **Next:** finish P0's split by library. `packages/flutter/lib/src/animation`
  still holds scheduler code (port Flutter's `SchedulerBinding`/`Ticker` into
  `packages/flutter/lib/src/scheduler`, on `flutter_go/async`), `widgets`
  holds gestures code (`gestures.go`, `velocity.go`), `services` holds the
  text-editing model under non-Flutter names, and `rendering` holds dart:ui
  geometry. Then P1 (`ui` over Impeller).

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
- Go's `math.Pow` isn't C's `pow`: compare with Flutter to ~1e-12 relative,
  not bit for bit.
- Window screenshots (`screencapture -l`) stopped working on this machine;
  environmental.
- Benchmarks vary about ±30% between runs here.
- Equal widgets skip rebuilds (Go's stand-in for `const`), a documented
  divergence.
