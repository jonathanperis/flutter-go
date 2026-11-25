# How Flutter's Dart API maps to Go

flutter-go exposes Flutter's public API under the same names, so Flutter's
documentation can be read with a short list of rules. These rules are fixed:
every ported library follows them, and `cmd/apiparity` checks the result.

## Libraries

| Dart | Go |
|---|---|
| `package:flutter/<lib>.dart` | `github.com/jonathanperis/flutter-go/<lib>` |
| `dart:ui` | `github.com/jonathanperis/flutter-go/ui` |

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

## Types

| Dart | Go |
|---|---|
| `double`, `int`, `bool`, `String` | `float64`, `int`, `bool`, `string` |
| `double?` and other nullable values | pointer, or a documented sentinel |
| `Duration` | `time.Duration` |
| `List<T>`, `Map<K,V>`, `Set<T>` | `[]T`, `map[K]V`, `map[T]struct{}` |
| `VoidCallback`, `ValueChanged<T>` | `func()`, `func(T)` |
| `Future<T>` | `*foundation.Future[T]` (completes on the framework's thread) |
| `Stream<T>` | `*foundation.Stream[T]` |

## Known divergences

Each divergence is listed here when it is introduced.

- `physics.NearEqual` takes `float64`, not `double?`: Dart's null case
  (`nearEqual(null, null) == true`) has no Go equivalent.
- Equal widgets skip rebuilds (Go's stand-in for `const`), see PLAN.md G1.
