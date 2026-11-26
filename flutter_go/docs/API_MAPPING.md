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
| `dart:developer` (service extensions, timeline) | `flutter_go/developer` |
| `dart:io` (the types Flutter's API mentions) | `flutter_go/io` |

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
| `ByteData`, `Uint8List` | `[]byte` |
| `Int32List`, `Int64List`, `Float32List`, `Float64List` | `[]int32`, `[]int64`, `[]float32`, `[]float64` |
| `Endian` | `encoding/binary.ByteOrder` (omitted: the host's, as `Endian.host`) |
| `Iterable<T>` | `iter.Seq[T]` (methods named `All()` return it) |
| `Future<T>` | `async.Future[T]`, an interface (see "Futures" below); `await f` is `f.Await()` inside `async.Async(func() T {...})` |
| `Stream<T>` | `async.Stream[T]`, an interface whose one method is `Listen`; Dart's other methods are generic functions (`async.StreamToList`). Ported: single-subscription streams, `StreamController`, `Stream.multi/fromIterable/fromFuture/empty` (not broadcast streams, transformations or `async*`) |

## Futures

Dart's `Future<T>` is an interface that Flutter implements
(`SynchronousFuture`, `TickerFuture`), so `async.Future[T]` is a Go
interface. Its methods are Dart's, with three adaptations:

- `f.then<R>(onValue, onError:)` is the generic function
  `async.Then(f, onValue)` / `ThenOrError(f, onValue, onError)` /
  `ThenFuture(f, onValue)` (when `onValue` returns a future), because Go
  methods can't take type parameters. They dispatch to the interface's
  `ThenAny(onValue func(T) any, onError func(error) any) Future[any]`,
  Dart's `then` with `R` erased; a callback result that is a `Future[any]`
  (`async.EraseFuture(f)`) is a `FutureOr<R>` holding a future.
  `then` with callbacks whose result is ignored is `async.OnComplete(f,
  onValue, onError)`.
- `CatchError(onError, test)`, `WhenComplete(action)` and
  `Timeout(timeLimit, onTimeout)` are methods (`asStream` is not ported).
  A `WhenComplete` action can't return a future.
- Dart's `await` and the `ignore()` extension are the methods `Await()` and
  `Ignore()`.

The futures `async` creates (`FutureValue`, `Completer.Future()`, `Async`...)
are one unexported implementation, Dart's `_Future`, and `async` treats them
as Dart treats `_Future` (`_chainCoreFuture`). A user-defined future
implements `ThenAny`, `CatchError`, `WhenComplete` and `Timeout` as its Dart
class does, and returns `async.AwaitUserDefined(f)` from `Await` and calls
`async.IgnoreUserDefined(f)` from `Ignore`: Dart's VM awaits and ignores a
user-defined future through its `then`, and `async` chains to one through
`ThenAny` (`_chainForeignFuture`), so a future whose `then` calls back
synchronously (SynchronousFuture) completes chained futures and resumes an
awaiting coroutine synchronously, as in Dart.

Events from other goroutines (an isolate's result, I/O) reach the event
loop through `Loop.Post`, the one goroutine-safe `Loop` method; the loop's
owner runs them with `RunPosted`/`WaitPosted` (Dart's event loop runs port
messages as they arrive; the fake clock never runs them).

## Mixins and overridable methods

Dart subclasses override methods that the base class's own algorithms call
(`valueToString`, `level`, `debugFillProperties`...). Go's embedding doesn't
dispatch that way, so every embeddable base (`FooBase`, `FooMixin`, and
concrete classes meant to be extended, such as `DiagnosticsProperty`) keeps a
pointer to the outer object, `self`, and calls every overridable method
through it. One pattern, everywhere:

- Constructors in the base's package set `self`. A type in another package
  that embeds a base calls `x.SetSelf(x)` once, when it is created:

  ```go
  type TestTree struct{ foundation.DiagnosticableTreeMixin; ... }

  func NewTestTree() *TestTree {
  	t := &TestTree{}
  	t.SetSelf(t)
  	return t
  }

  type ColorProperty struct{ foundation.DiagnosticsProperty[ui.Color] }

  func NewColorProperty(name string, c ui.Color) *ColorProperty {
  	p := &ColorProperty{DiagnosticsProperty: *foundation.NewDiagnosticsProperty(name, c)}
  	p.SetSelf(p)
  	return p
  }
  ```

- An override calls "super" through the embedded field:
  `t.DiagnosticableTreeMixin.DebugFillProperties(b)`.
- Types embedding a base are used by pointer (a copy still points at the
  original). Using a mixin before `SetSelf` panics with a message saying so.

## Nullable values and options

- A generic value `T?` is a `T` whose nil is Dart's null: use a pointer type
  for a nullable number, bool or struct (`DiagnosticsProperty[*float64]`).
  Constructors of fixed-type classes accept both forms through a type
  parameter (`NewDoubleProperty("x", 1.5)`, `NewDoubleProperty[*float64]("x", nil)`).
- An option whose Dart default is not null but which accepts an explicit
  null takes `foundation.Null` (any-typed options, e.g. `DefaultValue: Null`
  is `defaultValue: null`) or `foundation.NullString` (`*string` options).
  Leaving an option unset always gives the Dart default.
- Methods with optional named parameters take a variadic options struct
  named `<Type><Method>Options` (or `<Method>Options` when only one type has
  the method): `node.ToStringDeep(DiagnosticsNodeToStringDeepOptions{WrapWidth: &w})`.

## Errors and stack traces

- **Exceptions** (Dart's `Object exception`) are `any`: a Go `error`, a
  recovered panic value, a string... Dart's type tests map as: `is
  AssertionError` → `dart.AssertionFailure` (an interface satisfied by
  `*dart.AssertionError` and `*foundation.FlutterError`; Dart's `message`
  getter is `AssertionMessage()`), `is String` → `string`, `is num` → a
  builtin Go number, `is Error || is Exception` → `error`, anything else is
  "an object". A Go error prints its `Error()`; its runtime type is the Go
  type name (a nil dereference is `errorString`, as `runtime.Error`'s
  concrete type is).
- **`throw` / `catch`** are `panic` / `recover`. A failed Dart `assert(c,
  m)` that Flutter relies on is `panic(dart.NewAssertionError(m))`
  (`FlutterError.fromParts` does this for a missing summary).
- **`StackTrace`** is `dart.StackTrace`, an interface whose `String()` is the
  trace text; nil is null. `StackTrace.fromString(s)` is
  `dart.StackTraceFromString(s)`, `StackTrace.current` is
  `dart.StackTraceCurrent()`, and `catch (e, stack)` is `r := recover();
  stack := dart.StackTraceFromPanic()` in the deferred function (the stack
  starts at the function that panicked, as Dart's starts at the throw).
- **Go stacks are rendered in the Dart VM's format**, because Flutter
  parses traces as text (`StackFrame.fromStackString`, the default stack
  filter, the "is this the framework's fault" check):
  `#0      FlutterError.ToString (package:flutter/src/foundation/assertions.go:123)`.
  Methods drop the package qualifier and pointer (`(*T).M` → `T.M`),
  closures are `<anonymous closure>`; files of the Go packages that stand
  for Dart libraries get those libraries' URIs (`packages/<pkg>/lib/...` →
  `package:<pkg>/...`, `flutter_go/async` → `dart:async/...` so the default
  filter elides it like dart:async, `flutter_go/dart` → `dart:core/...`,
  dart:ui's `go/` → `dart:ui/...`), and every other file is a `file://`
  URI. Differences from Dart: no column numbers (`uri:line`, parsed as
  column -1), no `<asynchronous suspension>` lines, the goroutine header and
  `runtime.goexit`/`runtime.main` are dropped, and a stack overflow is fatal
  in Go (no trace).
- **Statics** follow the statics rule: `FlutterError.onError` →
  `foundation.FlutterErrorOnError`, `presentError` →
  `FlutterErrorPresentError`, `reportError` → `FlutterErrorReportError`,
  `demangleStackTrace` → `FlutterErrorDemangleStackTrace`,
  `defaultStackFilter` → `FlutterErrorDefaultStackFilter`, `StackFrame.fromStackString`
  → `StackFrameFromStackString`... Assign the variables to replace them.
- **Console output**: Dart's `print` is `dart.Print`; tests capture it with
  `dart.SetPrintHandler` (Go's stand-in for a Zone's `print`; unlike a zone,
  it is global, so a timer firing after the capture ends prints to whatever
  handler is current then). Flutter's own channel is `foundation.DebugPrint`,
  a variable (`DebugPrintCallback`) tests replace, exactly as Flutter's
  tests replace `debugPrint`; errors are dumped through it
  (`ErrorToConsoleDumper.Dump`). Dart's nullable `String? message` is a Go
  `string`: a null message prints `"null"` in every implementation, so
  `"null"` stands for it. `DebugPrintThrottled` pauses on the current
  `async.Loop`'s timers, so a program must run its loop for long output to
  finish (as a Dart program's event loop must run).
- `StackFrame` compares with `Equal` (Dart's `==`: number, package, line,
  column, className, method and source only). `PartialStackFrame.package`
  is a `dart.Pattern` (`dart.StringPattern("...")` or a `*regexp.Regexp`).

## Bindings, zones and statics

- `BindingBase` is an interface plus the embeddable `BindingBaseBase`. A
  binding embeds it (and its mixins) and its constructor calls
  `b.Construct(b)`, which is Dart's `BindingBase()` constructor: it sets
  `self` and runs `InitInstances` and `InitServiceExtensions` through it.
  Protected members are exported methods (Go has no `@protected`); named
  required parameters (`registerServiceExtension(name:, callback:)`) are
  positional.
- Go has no zones: where Flutter records `Zone.current` (the binding's
  zone, `debugCheckZone`), Go records the current `async.Loop`
  (flutter_test gives each test its own FakeAsync zone, as flutter-go gives
  it its own loop).
- A class of static members whose Dart name must stay visible
  (`FlutterTimeline`, `LicenseRegistry`) is a package-level variable whose
  methods are the statics (`foundation.FlutterTimeline.StartSync`); a
  generic static is a package function (`timeSync` →
  `FlutterTimelineTimeSync`). Statics of other classes follow the statics
  rule (`FlutterMemoryAllocations.instance` →
  `FlutterMemoryAllocationsInstance`, `BindingBase.checkInstance` →
  `BindingBaseCheckInstance`).
- Dart compares functions by identity (`removeListener(listener)`,
  `debugPrint != debugPrintThrottled`); Go compares the function values'
  closure pointers: the same top-level function or the same closure value
  is identical, a method value (`x.M`) is a new closure each time.
- Dart's `assert(() {...}())` blocks with side effects (recording a type,
  registering debug service extensions) run in debug builds only
  (`KDebugMode`); plain assertions panic in every build, as before.

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
- diagnostics: `DiagnosticsNode.Value()` returns `any`; a typed subclass's
  Dart `T value` is `TypedValue()` (Go has no covariant return types).
- diagnostics: a positional property name `""` stands for Dart's null name
  (only JSON shows the difference); `DiagnosticsNode.message` keeps its
  non-null `''`. `Name()` returns `""` for both.
- diagnostics: `FlagProperty`'s required named `value:` is positional.
- diagnostics: `FlagsSummary`'s `Map<String, T?>` is
  `[]dart.MapEntry[string, T]` (Dart maps keep insertion order; Go maps
  don't).
- diagnostics: Dart's `runtimeType`/`toString` are reproduced from Go types
  (`DiagnosticsProperty[*int]` prints `DiagnosticsProperty<int>`, a Go enum
  prints `Fruit.apple`, a package-level func prints Dart's
  `Closure: () => void from Function 'f': static.`). A nullable type
  argument prints without `?` (Go can't tell `T` from `T?` once T is a
  pointer).
- diagnostics: a lazy property's compute callback that panics is recorded as
  the property's exception (Dart catches the throw).
- assertions: `FlutterError` is a Go `error` and a `dart.AssertionFailure`;
  its `message` getter is `Message()`. `DiagnosticsStackTrace.singleFrame`'s
  required named `frame:` is positional. See "Errors and stack traces" for
  exceptions, stack traces and `print`.
- print: `DebugPrintCallback` is `func(message string, opts
  ...DebugPrintOptions)`; Dart's null message is `"null"` (same output).
  `debugWordWrap` returns `iter.Seq[string]`.
- foundation: `consolidateHttpClientResponseBytes`'s `autoUncompress`
  (default true) is the option `DisableAutoUncompress`; its gzip decoder
  decodes at the end instead of chunk by chunk (same bytes, same progress
  reports). `HttpClientResponse` is an interface in `flutter_go/io`.
- foundation: `compute` runs the callback on a goroutine and delivers the
  result through `Loop.Post`; `ComputeCallback` returns `R`, not
  `FutureOr<R>` (the event loop belongs to the calling goroutine), and
  nothing is "unsendable". `ComputeImpl` is a generic func type.
- foundation: `isCanvasKit`/`isSkwasm`/`isSkiaWeb` are the dart:io versions:
  they panic with `UnimplementedError` (flutter-go is never a web build).
- foundation: `LicenseEntry.packages/paragraphs` return `iter.Seq`;
  annotations (`Category`, `DocumentationIcon`, `Summary`) are plain
  structs (Go has no annotations).
- foundation: `AbstractNode` (deprecated) is an interface plus the
  embeddable `AbstractNodeBase`.
- foundation: `SynchronousFuture.asStream` is not ported, and a
  `WhenComplete` action can't return a future (see "Futures").
- foundation: `BindingBase` does not install package:listen's
  `Listenable.onError`, and `FlutterMemoryAllocations` does not hook
  `Listenable.debugMaybeDispatchCreated/Disposed` (package:listen is not
  ported).
- dart:developer: no VM service and no timeline recorder (ENGINE GAP, see
  PLAN.md); `developer.LookupExtension` is how a registered service
  extension is invoked.
