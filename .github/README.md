# flutter-go

**Flutter 3.50.1 with Go in place of Dart.** This repository is
[flutter/flutter@3.50.1](https://github.com/flutter/flutter/tree/3.50.1)
with exactly one difference: the Dart is gone, and Go does its job.

- Every non-Dart file of the release (engine C++, Impeller, embedders,
  shaders, fonts, tools, CI, docs) is here at its upstream path, unchanged,
  except Flutter's AI-agent configuration (`.agents/`, `.claude/`,
  `.gemini/` and similar) and its `.vscode/` folder, which are left out.
  `src/scripts/sync_upstream.sh --check` proves it blob for blob.
- Each Go file sits beside the Dart file it replaces:
  `packages/flutter/lib/src/physics/friction_simulation.go` is the port of
  `friction_simulation.dart`, which lives at the same path upstream. The
  one exception is `dart:ui`: its Go files are in
  `engine/src/flutter/lib/ui/go/`, because Go can't build a package in a
  directory holding C++ sources.
- Everything without an upstream counterpart is in [`src/`](../src):
  the window runner, the Impeller binding, `dart:async` in Go, the
  differential tests against real Flutter, tools, and docs.

The repository root's `README.md` is upstream's.

## Status

Progress is measured as API parity with Flutter's Dart layer:
[src/docs/PARITY.md](../src/docs/PARITY.md). The goal and the
rules every port follows are in [src/GOAL.md](../src/GOAL.md)
and [src/docs/API_MAPPING.md](../src/docs/API_MAPPING.md);
findings, including mistakes, are in [src/PLAN.md](../src/PLAN.md).

## Run (macOS)

```sh
./src/scripts/fetch_impeller_sdk.sh   # the engine's prebuilt Impeller SDK for 3.50.1
go run ./src/cmd/counter              # the classic counter app
go run ./src/cmd/list                 # 1,000,000-row lazy list
go run ./src/cmd/animate              # implicit animation + animated scroll
go run ./src/cmd/textfield            # to-do list with a text field
go test ./...                                # unit, widget and difftests
FLUTTER=/path/to/flutter-3.50.1/bin/flutter ./src/scripts/difftest.sh   # compare with real Flutter
```

Requires Go 1.24+ and Xcode command-line tools.

## Import paths

```go
import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"   // package:flutter/widgets.dart
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"   // package:flutter/physics.dart
	"github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"       // dart:ui (package ui)
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"      // package:flutter_test (package flutter_test)
)
```

Flutter's own license covers the upstream files and the ported code; see
[LICENSE](../LICENSE) and [src/THIRD_PARTY_NOTICES.md](../src/THIRD_PARTY_NOTICES.md).
