# flutter-go: standing goal

Set 2026-11-25. Supersedes the G1–G10 backlog (G1–G4 done; G5–G10 folded
into the phases below).

```text
GOAL: Flutter 3.50.1 with Go in place of Dart. flutter-go is Flutter 3.50.1 with exactly one difference: the Dart is gone and Go does its job. A Go developer must be able to do everything a Dart developer does with Flutter 3.50.1, through the same APIs.

PIN: RELEASE TAG 3.50.1, NOT MASTER
- FLUTTER_TAG = 3.50.1 (commit ae9b586). IMPELLER_SHA = its engine (bin/internal/engine.version, 4d35714). The local Flutter clone stays checked out at the tag; port from it and run reference tests with it.
- Moving to a newer tag is a separate, deliberate item: re-sync, re-run every difftest, record what changed.

UPSTREAM, BYTE FOR BYTE
- upstream/ holds every non-Dart file of flutter/flutter@3.50.1 at its upstream path (engine C++, Impeller, embedders, shaders, fonts, tools, CI, docs), never edited. scripts/sync_upstream.sh creates it; `--check` proves every blob matches the tag.
- So the repo's real difference from Flutter is: Dart files left out, Go added. Nothing else.

THE DART LAYER WE REPLACE
1. package:flutter: foundation, physics, scheduler, semantics, animation, services, gestures, painting, rendering, widgets, cupertino, material.
2. dart:ui: canvas, paths, paint, images, shaders, text, scene/layers, PlatformDispatcher, views, pointer/key events, semantics, platform messages.
3. flutter_test, as Go's widgettest.
4. The Dart VM's job (running the app, frame loop, isolates), done by the Go runtime.

ENGINE: UNMODIFIED IN THIS GOAL
- Use engine code only as upstream ships it: the prebuilt Impeller SDK for 4d35714, or upstream/ sources compiled unchanged. Never patch engine sources.
- Where dart:ui needs engine parts that only work with the Dart VM (shell threads, embedders, platform channels, accessibility bridge, image decoding), write the narrowest Go equivalent and log it as an ENGINE GAP in PLAN.md. A later goal swaps the engine's Dart VM binding (tonic/DartIsolate) for a Go one.

MEASURE
- Each public declaration exists in the Go package of the same name, per docs/API_MAPPING.md (divergences listed there).
- `go run ./cmd/apiparity` writes docs/PARITY.md. Extend it to count members, constructors and parameters. Done = 100% exposed and tested, minus documented Go impossibilities.

DONE MEANS (every item)
1. Behavior ported from the 3.50.1 source, not from memory.
2. Flutter's own tests for it (packages/flutter/test/<lib>) ported to Go, plus a difftest against real Flutter 3.50.1 where it can give a reference. Read the reference before trusting it.
3. Key rules mutation-checked: breaking one fails a test (a non-compiling mutation doesn't count).
4. go vet ./... and go test ./... pass; sync_upstream.sh --check passes; PARITY.md regenerated.
5. Findings and mistakes in PLAN.md; each item committed on its own and pushed.
Anything not verifiable this way is partial, never done.

ORDER (bottom-up)
P0 Pin to 3.50.1, re-verify all existing difftests on it; restructure Go packages to mirror Flutter's libraries; tests stay green.
P1 dart:ui → package ui over Impeller.
P2 foundation, physics, scheduler, semantics.
P3 animation, services (codecs, channels, keyboard, text input, clipboard, assets).
P4 gestures, painting.
P5 rendering (slivers, opacity/layers, RTL, semantics).
P6 widgets (navigation, focus, text editing, scrolling, all of widgets.dart).
P7 cupertino, material, with fonts, icons, localizations.
P8 flutter_test parity in widgettest.
Fold open items G5–G10 (SliverList, Opacity, Retina scale, editing, focus traversal, IME) into their phase.

PLATFORMS
macOS first. 3.50.1's Impeller SDK also ships for Linux, Windows and Android arm64; add each once P1 runs there. Platforms needing embedders wait for the engine goal.

RULES
- Work item by item without pausing. Ask only for my decisions: API divergences, dropping a feature, touching the engine, changing the tag.
- Keep HANDOFF.md current.
- Report progress as PARITY.md numbers plus what was verified, never estimates.
```

## Amendment (2026-11-25, at the user's request)

The layout changed after the goal was set: instead of `upstream/`, the
**repository root is the upstream tree** (every non-Dart file at its
upstream path), each Go port sits **beside the Dart file it replaces**
(same directory, same base name), and everything flutter-go adds without
an upstream counterpart is in `flutter_go/`. dart:ui's Go files are in
`engine/src/flutter/lib/ui/go/` because Go can't build a package in a
directory holding C++ sources. `flutter_go/scripts/sync_upstream.sh
--check` verifies the root against the tag. Read `upstream/` in the goal
text as "the repository root", and `scripts/`, `cmd/`, `docs/` as
`flutter_go/scripts/`, `flutter_go/cmd/`, `flutter_go/docs/`.

## Phase status

| Phase | Status |
|---|---|
| P0 pin to 3.50.1, re-verify difftests | ✅ 55/55 scenes match 3.50.1 |
| P0 upstream byte for byte, at upstream's paths | ✅ 9,677 files at the root, blob hashes checked; Go beside the Dart it replaces |
| P0 packages mirror Flutter's libraries | in progress (`physics` done; `animation` still holds scheduler code, `widgets` holds gestures and services code, `rendering` holds dart:ui geometry) |
| P2 physics | ✅ 12/12, Flutter's tests ported, difftest, mutation-checked |
| everything else | not started; see docs/PARITY.md |
