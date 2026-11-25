# How Flutter's Dart API maps to Go

flutter-go exposes Flutter's public API under the same names, so Flutter's
documentation can be read with a short list of rules. These rules are fixed:
every ported library follows them, and `cmd/apiparity` checks the result.

## Libraries

| Dart | Go package (import path under `github.com/jonathanperis/flutter-go/`) |
|---|---|
| `package:flutter/<lib>.dart` | `packages/flutter/lib/src/<lib>` (package `<lib>`) |
| `dart:ui` | `engine/src/flutter/lib/ui/go` (package `ui`) |
| `package:flutter_test` | `packages/flutter_test/lib/src` (package `flutter_test`) |
| `dart:async` (the subset Flutter's API uses) | `flutter_go/async` |

Go files sit beside the Dart files they replace and keep their base name
(`friction_simulation.dart` → `friction_simulation.go`), so the two trees
can be read side by side. dart:ui is the exception: its directory holds the
engine's C++ sources, and Go can't build a package there, so its Go files
are in `ui/go/`.

A name lives in the Go package that mirrors the Dart library declaring it.
Flutter re-exports (for example `widgets` exporting `rendering`) are not
repeated: import both packages.

## Declarations

| Dart | Go | Example |
|---|---|---|
| top-level name `fooBar` | `FooBar` | `nearZero` → `physics.NearZero` |
| class `Foo` | type `Foo` | `Tolerance` → `physics.Tolerance` |
| abstract class or interface implemented by user code | interface `Foo` (+ embeddable `FooBase` when it carries state or default methods) | `Simulation` → `physics.Simulation` + `physics.SimulationBase` |
| mixin `Foo` | interface `Foo` + embeddable struct `FooMixin` | |
| enum `Foo { a, b }` | `type Foo int` with `FooA`, `FooB`; `String()` gives the Dart name | `SpringType.underDamped` → `physics.SpringTypeUnderDamped` |
| typedef of a function | func type | `ValueChanged<T>` → `foundation.ValueChanged[T]` |
| generic `Foo<T>` | generic `Foo[T]` | |
| `toString()` | `String()` | |
| `==` / `hashCode` | `Equal(other) bool` / comparable struct values | |

## Constructors

| Dart | Go |
|---|---|
| `Foo(a, b)` | `NewFoo(a, b)`; a struct literal `Foo{...}` when every field is public and the zero value of each field is its Dart default |
| `Foo(a, {x = 1, y})` (optional named) | `NewFoo(a, FooOptions{X: ..., Y: ...})`, options passed variadically so they can be left out; an unset option takes the Dart default |
| `Foo.named(...)` | `NewFooNamed(...)` (`SpringDescription.withDampingRatio` → `NewSpringDescriptionWithDampingRatio`) |
| widget constructors | struct literals, as before (`widgets.SizedBox{Width: widgets.F(100)}`) |

A nullable option (`double? x`) is a pointer in the options struct, so "not
given" and "zero" stay distinct.

## Members

| Dart | Go |
|---|---|
| method `fooBar()` | method `FooBar()` |
| getter `foo` | method `Foo()` |
| setter `foo=` | method `SetFoo(v)` |
| public final field on a concrete class | exported struct field, when no Go method of the same name is needed |
| static member `Foo.bar` | package-level `FooBar`; `Bar` alone when `bar` already contains `Foo` (`Tolerance.defaultTolerance` → `DefaultTolerance`) |
| class of static members only (`Colors`, `Curves`, `Icons`) | package-level variable of that name holding a struct: `material.Colors.Red` |

## Operators

| Dart | Go method |
|---|---|
| `a + b`, `a - b`, `a * b`, `a / b` | `a.Add(b)`, `a.Sub(b)`, `a.Mul(b)`, `a.Div(b)` |
| unary `-a` | `a.Neg()` |
| `a ~/ b` | `a.TruncDiv(b)` (panics with `dart.UnsupportedError` where Dart throws) |
| `a % b` | `a.Mod(b)` (Dart's semantics: never negative) |
| `a & b` | `a.And(b)` |
| `<`, `<=`, `>`, `>=` | `Less`, `LessEq`, `Greater`, `GreaterEq` |
| `==` on value types | Go's `==` (struct equality; NaN is unequal, as in Dart) |

Value types (`Offset`, `Size`, `Rect`, `Radius`...) are structs with
exported fields. Outside their package, build them with the constructors
(`ui.NewOffset(1, 2)` for Dart's `Offset(1, 2)`): `go vet` rejects unkeyed
literals of imported structs.

## Types

| Dart | Go |
|---|---|
| `double`, `int`, `bool`, `String` | `float64`, `int`, `bool`, `string` |
| `double?` and other nullable values | pointer, or a documented sentinel |
| `Duration` | `time.Duration` |
| `List<T>`, `Map<K,V>`, `Set<T>` | `[]T`, `map[K]V`, `map[T]struct{}` |
| `VoidCallback`, `ValueChanged<T>` | `func()`, `func(T)` |
| `Iterable<T>` | `iter.Seq[T]` (methods named `All()` return it) |
| `Future<T>` | `*async.Future[T]`; `await f` is `f.Await()` inside `async.Async(func() T {...})` |
| `Stream<T>` | `*async.Stream[T]` (not yet ported) |

## Build modes

`kReleaseMode` and `kProfileMode` come from build tags (`-tags
flutter_release`, `-tags flutter_profile`); the default build is debug, as
`flutter run` is.

## Known divergences

Each divergence is listed here when it is introduced.

- `physics.NearEqual` takes `float64`, not `double?`: Dart's null case
  (`nearEqual(null, null) == true`) has no Go equivalent.
- Assertions are always on: where Dart asserts in debug mode (negative
  RRect radii, non-finite lerp inputs), Go panics in every build.
- Equal widgets skip rebuilds (Go's stand-in for `const`), see PLAN.md G1.
