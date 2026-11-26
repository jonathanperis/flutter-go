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
| `dart:async` (the subset Flutter's API uses) | `src/async` |
| `dart:developer` (service extensions, timeline) | `src/developer` |
| `dart:io` (the types Flutter's API mentions) | `src/io` |
| `dart:core`, `dart:math` (the parts Flutter needs) | `src/dart` (`dart.Random`, `dart.Pow`, `dart.Sin`, `dart.ArgumentError`, `dart.DoubleToString`...) |
| `dart:async` (the subset Flutter's API uses) | `src/async` |
| `dart:developer` (service extensions, timeline) | `src/developer` |
| `dart:io` (the types Flutter's API mentions) | `src/io` |
| `dart:core`, `dart:math` (the parts Flutter needs) | `src/dart` (`dart.Random`, `dart.Pow`, `dart.Sin`, `dart.ArgumentError`, `dart.DoubleToString`...) |
| Third-party Dart packages Flutter re-exports (`package:listen`) | `src/pub/<package>` (package `<package>`), files named after the package's `lib/src` files |
| `package:vector_math/vector_math_64.dart` (re-exported `Matrix4`) | `src/pub/vector_math` (package `vector_math`, partial) |

Go files sit beside the Dart files they replace and keep their base name
(`friction_simulation.dart` → `friction_simulation.go`), so the two trees
can be read side by side. dart:ui is the exception: its directory holds the
engine's C++ sources, and Go can't build a package there, so its Go files
are in `ui/go/`.

A name lives in the Go package that mirrors the Dart library declaring it.
Flutter re-exports (for example `widgets` exporting `rendering`) are not
repeated: import both packages.

**Third-party Dart packages Flutter re-exports: `src/pub/<package>`.**
Their code is not in the flutter/flutter tree, so the port mirrors pub's
layout in flutter-go's own area (`src/pub/listen/listen.go` is
`package:listen`'s `lib/src/listen.dart`), ported from the version Flutter's
`pubspec.lock` pins (in the pub cache). Such a package holds the whole
public API of the Dart package. Because Flutter users reach the re-exported
names through Flutter's library (`foundation.dart`'s `export
'package:listen/listen.dart' show ChangeNotifier, Listenable,
ValueListenable, ValueNotifier`), the re-exporting Go package declares
them too, in `<package>_exports.go` (`foundation/listen_exports.go`): type
aliases (`foundation.ChangeNotifier = listen.ChangeNotifier`) and wrappers
of the constructors and static functions (`foundation.NewValueNotifier`,
`foundation.ListenableMerge`). Assignable statics can't be aliased (a Go
variable has one name) and stay in the package: `Listenable.onError` is
`listen.ListenableOnError`. `cmd/apiparity` counts the shown names for the
re-exporting library.

## Declarations

| Dart | Go | Example |
|---|---|---|
| top-level name `fooBar` | `FooBar` | `nearZero` → `physics.NearZero` |
| class `Foo` | type `Foo` | `Tolerance` → `physics.Tolerance` |
| abstract class or interface implemented by user code | interface `Foo` (+ embeddable `FooBase` when it carries state or default methods) | `Simulation` → `physics.Simulation` + `physics.SimulationBase` |
| mixin `Foo` | interface `Foo` + embeddable struct `FooMixin` | |
| `mixin class Foo` (a mixin that is also a concrete class) | interface `Foo` + embeddable struct `FooMixin`; `Foo()` is `NewFoo() *FooMixin` | `ChangeNotifier` → `listen.ChangeNotifier` + `listen.ChangeNotifierMixin` |
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
| `ByteData`, `Uint8List` | `[]byte` (a nil slice is null; an empty non-nil slice is a zero-length ByteData) |
| `Int32List`, `Int64List`, `Float32List`, `Float64List` | `[]int32`, `[]int64`, `[]float32`, `[]float64` |
| `Endian` | `encoding/binary.ByteOrder` (omitted: the host's, as `Endian.host`) |
| `Iterable<T>` | `iter.Seq[T]` (methods named `All()` return it) |
| `Future<T>` | `async.Future[T]`, an interface (see "Futures" below); `await f` is `f.Await()` inside `async.Async(func() T {...})` |
| `Stream<T>` | `async.Stream[T]`, an interface whose one method is `Listen`; Dart's other methods are generic functions (`async.StreamToList`). Ported: single-subscription streams, `StreamController`, `StreamController.broadcast` → `NewStreamControllerBroadcast` (async and sync variants, `*BroadcastStreamController[T]`; its `onCancel` is a `func()`, its result ignored as in Dart), `Stream.multi/fromIterable/fromFuture/empty` (not `asBroadcastStream`, transformations or `async*`) |

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
  `package:<pkg>/...`, `src/async` → `dart:async/...` so the default
  filter elides it like dart:async, `src/dart` → `dart:core/...`,
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

## Bindings

Flutter composes its bindings as mixins on `BindingBase`
(`WidgetsFlutterBinding extends BindingBase with GestureBinding,
SchedulerBinding, ServicesBinding, PaintingBinding, SemanticsBinding,
RendererBinding, WidgetsBinding`), and Dart's mixin linearization decides
which override runs and where each `super` call goes. Go has no mixins, so
one pattern (`foundation/binding_mixins.go`) stands for it, for every
binding and every combination of mixins:

- `BindingBase` is an interface plus the embeddable `BindingBaseBase`. A
  binding mixin (`mixin SchedulerBinding on BindingBase`) is an interface
  (`scheduler.SchedulerBinding`, embedding `foundation.BindingBase` and the
  interfaces of its other `on` types) plus an embeddable struct
  (`scheduler.SchedulerBindingMixin`) with the mixin's state and methods.
  The struct embeds nothing: it keeps `self`, the binding it is mixed into,
  set by `SetSelf`, and calls every overridable method through it.
- A binding class embeds `foundation.BindingBaseBase`, then the mixin
  structs in Dart's `with` order; its constructor points each mixin at the
  binding (`SetSelf`) and calls `Construct(self, mixins...)` with pointers
  to the embedded mixins in `with` order. `Construct` is Dart's
  `BindingBase()` constructor: it records the mixins and runs
  `InitInstances` and `InitServiceExtensions` through `self`.
- A method that several mixins define (or a mixin and `BindingBase`:
  `InitInstances`, `InitServiceExtensions`, `Unlocked`, a mixin's method a
  later mixin overrides) is ambiguous in Go and isn't promoted; the binding
  class defines it, calling the last mixin in `with` order that defines it
  (Dart's most-derived override). The compiler lists them: the binding
  doesn't implement the mixins' interfaces until they are all defined.
- Dart's `super.m()` inside a mixin is `foundation.BindingSuper[I](m.self,
  m).M()`: the closest mixin before `m` in `with` order implementing `I`
  (an interface with `M`), else `BindingBaseBase`. For `BindingBase`'s own
  methods, `foundation.BindingBaseSuper(m.self, m)` is a `BindingBase`
  whose every method goes to `super`. So the chain follows the binding's
  actual composition, as Dart's linearization does: `initInstances` runs
  `BindingBase`'s first, then each mixin's in `with` order (each calls
  super first), and so do `initServiceExtensions` and `unlocked`. A binding
  class overriding a method calls the most-derived mixin's method as its
  super (or `BindingSuper` with a nil mixin).
- `X.instance` is `XInstance()` (`scheduler.SchedulerBindingInstance()`),
  set in the mixin's `InitInstances` to `self`, and checked with
  `foundation.BindingBaseCheckInstance`.
- A Go binding with a fixed composition that other bindings extend (Dart:
  `class TestBinding extends WidgetsFlutterBinding`) is a struct embedding
  the base and the mixins, with the forwarding methods; the extending
  binding embeds it, overrides at depth 0, and calls `SetSelf`/`Construct`
  with itself.

  ```go
  type MyBinding struct {
  	foundation.BindingBaseBase
  	gestures.GestureBindingMixin
  	scheduler.SchedulerBindingMixin
  	services.ServicesBindingMixin
  	semantics.SemanticsBindingMixin
  }

  func NewMyBinding() *MyBinding {
  	b := &MyBinding{}
  	b.GestureBindingMixin.SetSelf(b)
  	b.SchedulerBindingMixin.SetSelf(b)
  	b.ServicesBindingMixin.SetSelf(b)
  	b.SemanticsBindingMixin.SetSelf(b)
  	b.Construct(b, &b.GestureBindingMixin, &b.SchedulerBindingMixin,
  		&b.ServicesBindingMixin, &b.SemanticsBindingMixin)
  	return b
  }

  // Defined by several mixins: the last one in `with` order.
  func (b *MyBinding) InitInstances()         { b.SemanticsBindingMixin.InitInstances() }
  func (b *MyBinding) InitServiceExtensions() { b.ServicesBindingMixin.InitServiceExtensions() }
  func (b *MyBinding) Unlocked()              { b.SchedulerBindingMixin.Unlocked() }

  // Abstract in SemanticsBinding (RendererBinding implements it).
  func (b *MyBinding) PerformSemanticsAction(action ui.SemanticsActionEvent) {}
  ```

  and inside a mixin:

  ```go
  func (m *ServicesBindingMixin) InitInstances() {
  	foundation.BindingBaseSuper(m.outer(), m).InitInstances() // super.initInstances()
  	servicesBindingInstance = m.outer()                       // _instance = this
  	...
  }
  ```
- ServicesBinding registers its channel handlers in `InitInstances`, and
  `ChannelBuffers` drains what was buffered meanwhile in a microtask: flush
  the event loop after constructing a binding with it.

## Zones and statics

- A binding can be constructed once per process, as in Dart. Go runs a
  package's tests in one process, so tests that need a binding of their
  own (each Dart test file has its own isolate) re-run themselves in a
  child process (`isolated(t)` in the scheduler tests and difftests).
- Protected members are exported methods (Go has no `@protected`); named
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
- **Listeners and function identity.** Listeners are plain Go funcs
  (`AddListener(func())`, returning nothing, as in Dart), and Dart's `==`
  on functions (`removeListener(listener)`, `debugPrint !=
  debugPrintThrottled`) is `dart.FunctionsEqual`: the same top-level
  function or the same closure value is equal; a method value of a
  pointer receiver equals another of the same receiver and method, as
  Dart's tear-offs do (`n.AddListener(s.handleChange)` ...
  `n.RemoveListener(s.handleChange)` works, though Go allocates a new
  closure for each `s.handleChange`; it compares the wrapper's code
  pointer and the receiver it captured). Divergences: a method value of a
  value receiver or of an interface (`i.M`) equals only the same closure
  value (keep it in a variable to remove it); a func literal that captures
  nothing is a single value however often it is evaluated (Dart creates a
  new closure each time); a method promoted from an embedded pointer
  (`outer.M`, `M` of the embedded `*Inner`) is the embedded value's
  method value.
- Dart's `assert(() {...}())` blocks with side effects (recording a type,
  registering debug service extensions) run in debug builds only
  (`KDebugMode`); plain assertions panic in every build, as before.

## Build modes

`kReleaseMode` and `kProfileMode` come from build tags (`-tags
flutter_release`, `-tags flutter_profile`); the default build is debug, as
`flutter run` is.


## Platform channel values (services codecs)

A Dart `int` is a Go `int` (64-bit on every platform flutter-go targets):
the standard codec decodes both its 32- and 64-bit integers as `int`, the
JSON codec decodes integer literals as `int`; every Go integer type is
accepted when encoding (a `uint64` above `math.MaxInt64` throws
`ArgumentError`, as no Dart int holds it). `double` is `float64` (a `float32`
is encoded as the double it widens to). Typed lists are `[]byte`,
`[]int32`, `[]int64`, `[]float32`, `[]float64`; any other slice or array is a
`List` (decoded: `[]any`); a nil slice or map is an empty list or map, only
a nil interface or nil pointer is null (a non-nil pointer is its value). A
`Map` is a Go map or a `[]dart.MapEntry[K, V]` (an ordered map, written in its
order); a Go map is written in sorted key order (null, bools, numbers,
strings, then the rest by their text). Decoded maps are `map[any]any`
(standard codec) or `map[string]any` (JSON): their insertion order is lost.
The standard codec merges keys as Dart's map does (an int and a double of the
same value, 0.0 and -0.0; the first key is kept); a key Go can't hash (a
list, typed data, a map) panics with `UnsupportedError`.

## Known divergences

Each divergence is listed here when it is introduced.

- scheduler: `scheduleTask<T>` (a generic method) is the function
  `scheduler.ScheduleTask[T](binding, task, priority, opts)`, and
  `ScheduleTaskFuture` for a task returning a future (Dart's
  `FutureOr<T>`). `SchedulingStrategy`'s required named parameters are
  positional; `schedulingStrategy` is `SchedulingStrategy()` /
  `SetSchedulingStrategy`. `handleBeginFrame(Duration?)` takes a
  `*time.Duration`; `lifecycleState` and `debugGetRequestedPerformanceMode`
  return pointers (nil is null).
- scheduler: `Ticker` is a struct (Flutter extends it): a type embedding it
  calls `SetSelf`, and `TickerProvider.CreateTicker` returns `*Ticker` (the
  embedded one). `toString({debugIncludeStack})` is
  `ToString(TickerToStringOptions{...})`; a `""` `DebugLabel` stands for
  null (both print nothing), as does `""` for `addPostFrameCallback`'s
  `debugLabel` (meaning the default, `"callback"`).
- dart:ui: PlatformDispatcher callbacks are called directly (Go has no
  zones to run them in). The `external` natives (`scheduleFrame`,
  `endWarmUpFrame`, `setNeedsReportTimings`, `requestDartPerformanceMode`)
  go to the embedder's `ui.PlatformConfigurationClient`, installed with
  `ui.SetPlatformConfiguration`; the engine's frame
  (`PlatformConfiguration::BeginFrame`: begin frame, drain microtasks, draw
  frame) is `PlatformConfiguration.BeginFrame`. ENGINE GAP: no
  `Dart_SetPerformanceMode` (the mode is only recorded) and no frame
  timings are collected.

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
  reports). `HttpClientResponse` is an interface in `src/io`.
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
- foundation: `FlutterMemoryAllocationsInstance` is created when the
  package is initialized (Dart creates it on first access), so its
  constructor installs package:listen's creation/disposal hooks before any
  use; they dispatch only while a listener is registered, so this is not
  observable.
- listen: `ChangeNotifier` is an interface (plus `ChangeNotifierMixin`,
  see `mixin class` above) whose unexported method only types embedding
  `ChangeNotifierMixin` have (Dart's statics read its private state). A
  type embedding it calls `SetSelf(outer)`; one embedding `ValueNotifier`
  calls `Construct(outer, value)` (Dart's `super(value)`, which also
  dispatches the creation). Overrides of `NotifyListeners` and `Value` are
  reached through `self`.
- listen: `Listenable.merge`'s `Iterable<Listenable?>` is
  `iter.Seq[Listenable]`; a nil entry, or a nil pointer in the interface,
  is Dart's null.
- listen: the default `ListenableOnError` rethrows the error, but a Go
  panic carries no stack trace, so the original stack is not preserved
  (Dart's `Error.throwWithStackTrace`).
- listen: `AddListener(nil)` panics (Dart's `VoidCallback` is
  non-nullable); `RemoveListener(nil)` does nothing.
- listen: `ValueNotifier`'s `==` is the value's `Equal(T) bool` method when
  it has one, Go's `==` otherwise; slices, maps and funcs compare by
  identity (Dart's `List ==`); a struct with uncomparable fields and no
  `Equal` is never equal (a Go value has no identity); in a
  `ValueNotifier[any]`, `1` and `1.0` differ (Dart's `1 == 1.0` is true).
  As in diagnostics, a nullable type argument prints without `?`
  (`ValueNotifier[*int]` is `ValueNotifier<int>`, Dart says `<int?>`).
- dart:developer: no VM service and no timeline recorder (ENGINE GAP, see
  PLAN.md); `developer.LookupExtension` is how a registered service
  extension is invoked.
- curves: curves are used by pointer and built with constructors
  (`animation.NewCubic(0.25, 0.1, 0.25, 1.0)`, not struct literals: the
  embedded `CurveBase` needs `self`). Dart's `const` curves are the fields
  of `animation.Curves`. Two curves built with the same arguments are
  distinct pointers (Dart canonicalizes equal `const` instances, which only
  `identical` can observe: Curve doesn't override `==`).
  `ElasticInCurve([period])` is `NewElasticInCurve(period ...float64)`;
  `Curve2D.generateSamples` returns `iter.Seq[Curve2DSample]`.
- dart:ui: the top-level `channelBuffers` is `ui.ChannelBuffersInstance()`
  (the type has the variable's Go name). `ChannelBuffers` is overridable
  (Flutter's tests override `resize`/`allowOverflow`): a type embedding it
  calls `SetSelf`. `RootIsolateToken.instance` is
  `ui.RootIsolateTokenInstance()`, never nil (a Go program is one isolate).
  ENGINE GAP: platform messages reach the embedder only through a client
  implementing `ui.PlatformMessageHandler`; otherwise (src/app, tests) every
  message gets an empty reply, posted to the sender's loop as the engine's
  default PlatformView does. Replies are loop events (`Loop.RunPosted`).
- dart:convert (`src/dart`): Go strings are read as WTF-8, so a lone
  surrogate encodes as Dart's does (EF BF BD in UTF-8, `\udXXX` in JSON); a
  lone surrogate decoded from a JSON `\u` escape becomes U+FFFD.
- services: `MessageCodec[T]`'s `T?` is T's nil (`StringCodec` is a
  `MessageCodec[*string]`). `StandardMessageCodec` is used by pointer and is
  extended by embedding it (`SetSelf`; overridable `WriteValue`,
  `ReadValue`, `ReadValueOfType`, `WriteSize`, `ReadSize`).
  `PlatformException` and `MissingPluginException` are pointer errors
  (thrown as panics, delivered as futures' errors; `errors.As`). Optional
  positional parameters, when more than one (`MethodChannel(name, [codec,
  binaryMessenger])`), are an options struct (`MethodChannelOptions`). A
  channel is used by pointer; `OptionalMethodChannel` embeds `MethodChannel`
  and its `&c.MethodChannel` behaves optionally (as `SystemChannels.platform`,
  declared `MethodChannel`, does). `invokeMethod<T>` returns `Future[any]`
  (the cast is the caller's); `invokeListMethod<T>`/`invokeMapMethod<K, V>`
  are `MethodChannelInvokeListMethod[T]`/`MethodChannelInvokeMapMethod[K,
  V]` and convert eagerly (Dart's `cast` is lazy). `SystemChannels` is a
  package variable of a struct of channels. `BinaryMessenger.
  HandlePlatformMessage` returns `Future[[]byte]` for every messenger (Dart:
  `Future<void>`, which TestDefaultBinaryMessenger overrides as
  `Future<ByteData?>`). ServicesBinding is partial (no keyboard,
  restoration, text input, licenses from the asset bundle, view focus).
  `BackgroundIsolateBinaryMessenger.send` panics (ENGINE GAP: no isolates).
  The int/double key merge follows the VM on arm64, where the double 2^63
  hashes as 2^63-1.
- flutter_test: `TestDefaultBinaryMessengerBinding`'s covariant
  `defaultBinaryMessenger` is `DefaultBinaryMessenger().(*TestDefaultBinaryMessenger)`;
  the generic `setMockDecodedMessageHandler<T>` is
  `TestDefaultBinaryMessengerSetMockDecodedMessageHandler[T]`. Go can't tell
  which test is running, so `AddTearDown` (used by `setMockStreamHandler`)
  registers with the test given to `SetCurrentTest(t)`.
- curves: a curve that overrides `transform` itself (as `Split` does)
  defines `Transform` on its own type; `TransformInternal` overrides are
  reached through `self`.
- animation: `Animation<T>` is an interface plus the embeddable
  `AnimationBase[T]` (self pattern); the generic method `drive<U>` is the
  function `animation.Drive[U](animation, child)`. The listener mixins
  (`AnimationLazyListenerMixin`, `AnimationEagerListenerMixin`,
  `AnimationLocalListenersMixin`, `AnimationLocalStatusListenersMixin`,
  `AnimationWithParentMixin`) are embeddable structs with the Dart names
  (their Dart names already end in Mixin); each calls the members it leaves
  abstract through `SetSelf`. Local listener lists keep Dart's
  HashedObserverList (value listeners: one call per notification however
  often added) and ObserverList (status listeners: duplicates called
  twice) semantics, comparing with `dart.FunctionsEqual`.
- animation: `AnimationController` is a struct Flutter extends: a type
  extending it calls `c.Construct(c, vsync, opts)` (or
  `ConstructUnbounded`); `NewAnimationController(vsync, opts)` and
  `NewAnimationControllerUnbounded` are the two constructors (vsync,
  required, is positional). The setter `value=` is `SetValue`; nullable
  `duration`/`reverseDuration` are `*time.Duration` fields; methods with
  named parameters take `ForwardOptions`, `ReverseOptions`,
  `ToggleOptions`, `AnimateToOptions`, `AnimateBackOptions`,
  `RepeatOptions`, `FlingOptions`, `StopOptions` (nil `Canceled` is
  Dart's default true). The mixins' @protected members (`NotifyListeners`,
  `NotifyStatusListeners`, `ClearListeners`, `DidRegisterListener`...)
  are exported. The controller's own algorithms
  call its members directly (only `toString` and the Animation getters go
  through self).
- animation: `AnimationBehavior.normal` consults
  `SemanticsBinding.instance.disableAnimations`; the semantics library is
  not ported, so it reads the hook `animation.SemanticsDisableAnimations`
  (false by default; tests set it where Flutter sets
  `debugSemanticsDisableAnimations`).
- animation: `Animatable<T>` is an interface plus `AnimatableBase[T]`;
  `Animatable.fromCallback` is `NewAnimatableFromCallback`. `Tween<T>` is a
  struct (Flutter extends it): `Begin`/`End` are `T`, so a nullable tween
  has a pointer type argument (`ColorTween` is `Tween[*ui.Color]`; nil is
  null), and a non-pointer `Tween[float64]` has no null ends. A type
  extending Tween overrides `Lerp` and calls `SetSelf`.
  `ReverseTween(parent)` takes the parent's embedded `*Tween[T]`
  (`&intTween.Tween`, as `TickerProvider.CreateTicker` returns the
  embedded `*Ticker`). `Tween.lerp`'s dynamic `begin + (end - begin) * t`
  calls the types' `Add`/`Sub`/`Mul` operator methods by reflection
  (float64 and int natively); a missing method or a result of the wrong
  type is reported as Flutter reports Dart's NoSuchMethodError/TypeError
  (the same FlutterError in debug builds).
- animation: `AlwaysStoppedAnimation.value`, `CurvedAnimation.parent` and
  `TrainHoppingAnimation.currentTrain` are methods (`Value()`, `Parent()`,
  `CurrentTrain()`), since the Animation interfaces need them; the other
  public fields (`Curve`, `ReverseCurve`, `IsDisposed`, `OnSwitchedTrain`,
  `First`/`Next`, `ReverseAnimation.Parent`) are fields. `CompoundAnimation`
  is embedded and set up with `Construct(self, first, next)`; `AnimationMax`
  and `AnimationMin` take `T ~int | ~float64` (Dart's `num`) and use
  dart:math's `max`/`min` rules (NaN, ±0). `kAlwaysCompleteAnimation` is
  `KAlwaysCompleteAnimation`. `TweenSequenceItem`'s required named
  parameters are positional in `NewTweenSequenceItem(tween, weight)`.
- animation: `AnimationStyle` is used by pointer (it is Diagnosticable);
  `==` is `Equal`, `copyWith` takes `CopyWithOptions`, `AnimationStyle.lerp`
  and `AnimationStyle.noAnimation` are `AnimationStyleLerp` and
  `AnimationStyleNoAnimation`. Curves compare by identity, except the
  private `_LerpedCurve` (field-wise, as in Dart).
- enums: a Go enum may have a `Name()` method (Dart's `.name`), in which
  case its `String()` is Dart's whole `toString()` ("TextDirection.ltr");
  foundation's `EnumProperty`/`describeEnum` use `Name()` and `dartToString`
  uses `String()` as is. Enums without `Name()` (foundation's own) keep
  returning the bare name from `String()`.
- semantics: `Set`s and `Map`s whose iteration order is observable keep
  Dart's insertion order: `SemanticsNode.Tags`, `controlsNodes` and
  `tagsForChildren` are slices without duplicates (nil is null); a custom
  action map is `[]dart.MapEntry[CustomSemanticsAction, func()]`
  (`SetCustomSemanticsActions` replaces it; mutating the slice the getter
  returns doesn't reach the configuration). A tag set shared by reference
  and mutated later is not seen by the node (assign it again).
- semantics: `SemanticsTag` and `SemanticsSortKey` compare by identity
  (Dart's classes have no `==`): use them by pointer.
  `CustomSemanticsAction` is a comparable value: `Label`/`Hint` `""` and the
  zero `ui.SemanticsAction` stand for null (Dart asserts the strings are
  not empty).
- semantics: `AttributedString`'s `string` is the field `Text` (`String()`
  is `toString`); `+` is `Add`; `==` is `Equal` (attributes by identity, as
  Dart's `listEquals` on `StringAttribute`s). `Uri` is `*url.URL`
  (compared by its `String()`). `SemanticsData`, `SemanticsProperties` and
  `SemanticsHintOverrides` embed their `...Options` struct (the fields);
  build them with `NewSemanticsData(SemanticsDataOptions{...})` etc.
- semantics: `SemanticsNode.toStringDeep/toDiagnosticsNode/
  debugDescribeChildren({childOrder})` keep the DiagnosticableTree
  signatures (traversal order) and add `ToStringDeepInOrder`,
  `ToDiagnosticsNodeInOrder`, `DebugDescribeChildrenInOrder` taking the
  `DebugSemanticsDumpOrder`. `updateWith({config, childrenInInversePaintOrder})`
  is `UpdateWith(SemanticsNodeUpdateWithOptions{...})`. A Dart list passed
  again after being mutated in place (the "list was mutated" check) is a Go
  slice sharing its backing array.
- semantics: `SemanticsService` is a variable of static methods
  (`SemanticsService.SendAnnouncement(view, ...)`); `SemanticsEvent` is an
  interface plus the embeddable `SemanticsEventBase`; `toMap`/`getDataMap`
  return ordered maps (`[]dart.MapEntry[string, any]`).
- dart:ui semantics: `SemanticsAction`/`SemanticsFlag` are comparable
  values (zero: none); `values` and `fromIndex` are
  `SemanticsActionValues()`/`SemanticsActionFromIndex(i) *SemanticsAction`.
  `SemanticsFlags` is a struct literal (every zero value is Dart's
  default). `SemanticsUpdateBuilder.updateNode`'s named parameters are
  `SemanticsUpdateBuilderUpdateNodeOptions`. ENGINE GAP: no accessibility
  bridge consumes a built `SemanticsUpdate`; `ui.EngineSemanticsUpdateOf`
  returns what was built, `ui.SemanticsTreeEnabled` the last
  `setSemanticsTreeEnabled`, and an embedder's
  `PlatformConfigurationClient` that also implements
  `PlatformConfigurationSemanticsClient` receives both.
- services (partial): `BasicMessageChannel` without a messenger uses
  `services.DefaultBinaryMessenger`, the stand-in for
  `ServicesBinding.instance.defaultBinaryMessenger` until ServicesBinding
  is ported (nil: the channel's send fails as Flutter fails without a
  binding). `StandardMessageCodec` decodes maps as
  `[]dart.MapEntry[any, any]` (insertion order) and encodes those, string
  keyed entries and Go maps (sorted by key); `services.MapLookup` reads any
  of them. Subclassing the codec (overriding `writeValue`) is not
  supported yet.
- dart:core `List.sort` is `dart.ListSort` (the VM's insertion sort /
  dual-pivot quicksort, whose tie order is observable);
  `double.compareTo`, `String.compareTo` (UTF-16 code units) and
  `double.tryParse` are `dart.CompareDoubles`, `dart.CompareStrings`,
  `dart.DoubleTryParse`.
- gestures: `PointerEvent` is an interface; each event class is a struct
  used by pointer (`*PointerDownEvent`), so `event is PointerDownEvent` is
  a type switch. Dart's `_TransformedPointerXEvent` wrappers are values of
  the same struct with `Original()` set (so the switch matches them too);
  only `ToStringShort`/`toString` show the `_Transformed...` name.
  `copyWith` is `CopyWith(PointerEventCopyWithOptions)` and `transformed`
  is `Transformed(*Matrix4)`, both returning `PointerEvent` (no covariant
  returns). Constructors take per-class options whose non-zero Dart
  defaults are pointers (`Buttons`, `Pressure`, `PressureMin/Max`, signal
  events' `Kind`, `Scale`). `PointerSignalEvent` is an interface.
  PointerEvent's statics are functions (`PointerEventTransformPosition`,
  `PointerEventTransformDeltaViaPositions`, `ComputeHitSlop`...).
- gestures: hit_test.dart is `hittest.go` and dart:ui's onHitTest is
  `platform_dispatcher_hittest.go` (a `_test.go` name would make them test
  files). `HitTestEntry` is an interface plus `HitTestEntryBase`
  (`NewHitTestEntry`, `InitHitTestEntry` for subclasses); `HitTestEntry(t)`
  prints `HitTestEntry<HitTestTarget>` as Flutter's does; `Path()` is an
  `iter.Seq`. `PointerEventConverter.expand` is
  `PointerEventConverter.Expand(iter.Seq[ui.PointerData], getter)` (a variable of statics), lazy as
  Dart's Iterable; `DevicePixelRatioGetter` returns `*float64` (nil: the
  view is gone). `PointerRouter.addRoute`'s optional transform is
  variadic; routes compare with `dart.FunctionsEqual`.
- gestures: `GestureArenaEntry` is an interface (team.dart implements it);
  `GestureArenaMember` is an interface.
- gestures: the recognizer classes are interfaces plus embeddable bases
  (`GestureRecognizer` + `GestureRecognizerBase`,
  `OneSequenceGestureRecognizer` + `...Base`,
  `PrimaryPointerGestureRecognizer` + `...Base`); a recognizer embeds the
  base of its Dart superclass (or the concrete recognizer it extends) and
  calls `Construct(self, options)` once (Dart's super constructor); every
  overridable method goes through `self`, and overrides call "super"
  through the embedded field. Public mutable fields (`GestureSettings`,
  `SupportedDevices`, callbacks such as `OnTap`, `OnStart`) are struct
  fields; @protected methods are exported. The generic
  `invokeCallback<T>` is `InvokeCallback[T](recognizer, name, callback,
  debugReport...)` (T's zero value is Dart's null result) and
  `InvokeVoidCallback`. `PrimaryPointerGestureRecognizer`'s slop
  tolerances are `*float64` options whose nil is Dart's default (the touch
  slop); `NoPreAcceptSlopTolerance`/`NoPostAcceptSlopTolerance` pass
  Dart's explicit null.
- gestures: `GestureBinding` is a binding mixin (see "Bindings"); its
  public fields `pointerRouter`, `gestureArena`, `pointerSignalResolver`
  are methods, `resamplingEnabled`/`samplingOffset` are getter/setter
  pairs. `SamplingClock` is an interface (`Now`, `Stopwatch()
  *dart.Stopwatch`); the velocity trackers read
  `GestureBinding.instance.samplingClock` through a hook GestureBinding
  sets in InitInstances (the wall clock without a binding).
  `FlutterError.reportError` takes `*foundation.FlutterErrorDetails`, so
  `details is FlutterErrorDetailsForPointerEventDispatcher` is
  `gestures.AsFlutterErrorDetailsForPointerEventDispatcher(details)`.
  Only the implicit view exists: the device pixel ratio of any other view
  id is unknown (null).
- gestures: `VelocityTracker` is a struct the iOS/macOS trackers embed
  (self pattern); `getVelocityEstimate` returns `*VelocityEstimate` (nil:
  null). `Velocity`, `VelocityEstimate`, `DeviceGestureSettings` and
  `OffsetPair` are comparable values. Drag/scale details are used by
  pointer and built with `NewXDetails(XDetailsOptions{...})` (nil options:
  Dart's defaults; `ScaleEndDetails`'s `Velocity` defaults to the zero
  value, which is `VelocityZero`). `pointerCount` is `PointerCount()`.
- dart:ui (pointer.dart, key.dart): enums with `Name()`/`String()` and
  `...Values`; `PointerData` is a struct literal (`SignalKind` is a
  pointer; nil is null); `PlatformDispatcher.onPointerDataPacket`/
  `onKeyData`/`onHitTest` are getter/setter pairs; the engine side is
  `PlatformConfiguration.DispatchPointerDataPacket(bytes)` and
  `PlatformConfiguration.HitTest(viewID, x, y)`. ENGINE GAP: views have no
  metrics: the device pixel ratio is 1.0 and the gesture settings empty
  unless an embedder sets them with
  `PlatformConfiguration.UpdateViewGestureMetrics` (a stand-in for
  `_updateWindowMetrics`).
- flutter_test: `TestPointer` is ported (`NewTestPointer(opts)`, events'
  named parameters as options); `TestGesture` is not. The gesture tests'
  helper `testGesture`/`GestureTester` (test/gestures/gesture_tester.dart)
  is `flutter_test.RunTestGesture`/`GestureTester`.
- gestures (recognizers): tap, long press and tap-and-drag details are
  Diagnosticable pointers built with `NewXDetails(XDetailsOptions)`;
  positions are methods, other fields struct fields, nullable values
  pointers (`*ui.PointerDeviceKind`, `*time.Duration`, `*float64`).
  `BaseTapGestureRecognizer`'s handlers take positional parameters
  (`HandleTapUp(down, up)`). `DragGestureRecognizer` and
  `BaseTapAndDragGestureRecognizer` (sealed in Dart) are interfaces plus
  `...Base` structs whose private abstract members are unexported
  interface methods; the `_TapStatusTrackerMixin` members belong to the
  tap-and-drag Base. Fling limits, `maxConsecutiveTap`,
  `dragUpdateThrottleFrequency` and `DelayedMultiDragGestureRecognizer`'s
  `Delay` are pointer fields/options (nil: null). `GestureVelocityTrackerBuilder`
  returns `*VelocityTracker` (an iOS/macOS tracker's embedded pointer).
  `Drag` is an interface plus `DragBase`; a nil `Drag` from
  `GestureMultiDragStartCallback` is null. `MultiDragPointerState` is an
  interface plus a Base (`Construct(self, initialPosition, kind, settings)`).
  LongPress's nil post-accept slop is its Dart default (null).
- widgets (until GestureDetector/Scrollable are ported): the test binding
  and src/app hit-test through `BuildOwner.HitTestGestures`, which adds a
  TapGestureRecognizer per tap region and a VerticalDragGestureRecognizer
  per list; `widgets.PointerRouter` only builds pointer events for
  `GestureBinding.HandlePointerEvent` (it moves the pointer to where it is
  lifted first, as the engine synthesizes that move). src/app sends touch
  events for the mouse.
- dart:ui drawing: Paint is `*Paint` (`NewPaint`, `Paint.from` →
  `NewPaintFrom`), getters/setters as methods; `strokeMiterLimit` returns
  the stored value as 3.50.1 does (limit − 4). Nullable named arguments
  are pointers in options structs (`CanvasClipRectOptions{ClipOp,
  DoAntiAlias}`, `ImageFilterBlurOptions{TileMode, Bounds}`,
  `ImageShaderOptions{FilterQuality}`...); optional positional arguments
  are options structs (`GradientLinearOptions{ColorStops, TileMode,
  Matrix4}`, `GradientRadialOptions{..., Focal *Offset, FocalRadius}`,
  `GradientSweepOptions{..., EndAngle *float64}`); `Canvas.scale(sx, [sy])`
  is `Scale(sx, sy...)`. `Canvas` is an interface (Flutter's tests
  implement it), `NewCanvas(recorder, cullRect...)` returns the engine's;
  `ImageFilter` and `Shader` are interfaces only dart:ui implements;
  `ColorFilter` is `*ColorFilter` with `Equal`; `ImageFilterEqual(a, b)` is
  `==` on image filters; `MaskFilter` and `Shadow` are comparable values.
  `Picture.toImage` returns `async.Future[*Image]` completed in a later
  event of the current loop. Engine hooks: `ui.EngineImpellerContext()`
  (the process's Impeller context), `ui.EnginePictureDisplayList(p)`,
  `ui.EngineDevicePixelRatio` (drawShadow), `ui.EngineAssetLoader`
  (assets), `ui.EngineDrawParagraph` (until Paragraph is ported). ENGINE
  GAPs: see PLAN.md "dart:ui drawing over Impeller".
- dart:ui Path: used by pointer (`ui.NewPath()`, `Path.from` →
  `NewPathFrom`, `Path.combine` → `PathCombine`); `Float64List matrix4` is
  `[]float64` (nil is null). Named parameters are `ArcToPointOptions`
  (`RelativeArcToPointOptions` is the same type; nil `Clockwise` is Dart's
  default true), `AddPathOptions` (`ExtendWithPathOptions`),
  `ComputeMetricsOptions` and `ExtractPathOptions` (nil `StartWithMoveTo`
  is true). ENGINE GAPs: `PathCombine` (Skia PathOps) and `Transform` with
  a perspective whose w=0 plane crosses the path (SkEdgeClipper) panic with
  `dart.UnimplementedError`. `Reset` also resets the fill type to nonZero
  (SkPathBuilder::reset).
- dart:ui PathMetrics: Dart's Iterable is `All()` (iter.Seq) plus
  `Iterator()`, which, as in Dart, always returns the same
  PathMetricIterator, so iterating (or `String()`) consumes it; use
  `slices.Collect(m.All())` for `toList()`/`first`.
  `PathMetricIterator.Current` panics with an error printing Dart's
  "RangeError: …" text. `PathMetric.length/isClosed/contourIndex` are
  fields; `getTangentForOffset` returns `*Tangent` (nil is null); `Tangent`
  is a comparable struct (Dart's class has no ==).
- dart:ui images: `Image` is `*Image` (fields `Width`/`Height`);
  `toByteData` returns `async.Future[[]byte]` (nil is null); codecs,
  `ImmutableBuffer` and `ImageDescriptor` follow the Futures rules. Images
  are decoded in Go (golang.org/x/image for WebP/BMP); see PLAN.md for the
  measured differences from Skia's codecs.
- dart:ui runtime effects: `FragmentProgram.fromAsset` reads impellerc
  runtime-stage bundles through `ui.EngineAssetLoader`; `ImageFilter.shader`
  is `NewImageFilterShader`. ENGINE GAPs: the SDK samples shader samplers
  linearly and the filter input nearest, whatever the FilterQuality.
