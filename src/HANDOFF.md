# Handoff (2026-11-25)

Start here when continuing flutter-go in a new session.

## Where things stand

- **Goal:** [GOAL.md](GOAL.md): Flutter 3.50.1 with Go in place of Dart,
  measured by full API parity. Progress: [docs/PARITY.md](docs/PARITY.md) (in src/)
  (regenerate with `go run ./src/cmd/apiparity -flutter <clone> -o src/docs/PARITY.md`).
- **Pin:** `FLUTTER_TAG` (3.50.1) and `IMPELLER_SHA` (its engine, 4d35714).
  The repository root is every non-Dart file of the tag at its upstream
  path, never edited; Go files sit beside the Dart files they replace
  (dart:ui's in `engine/src/flutter/lib/ui/go/`), and everything else of
  flutter-go is in `src/`. Verify with
  `./src/scripts/sync_upstream.sh --check`.
- **Naming rules:** [docs/API_MAPPING.md](docs/API_MAPPING.md). Every
  divergence from Flutter's API gets listed there.
- **Done:** P0 pin, layout (upstream tree at the root, Go beside the Dart
  it replaces) and `physics`; `src/async` (dart:async semantics);
  dart:ui `math`, `lerp` and `geometry` (P1 started); the `animation`
  library in full (curves with `dart.Random`, the VM's generator). Stages 0–10 and G1–G4
  (see PLAN.md) are verified on 3.50.1. Parity: see src/docs/PARITY.md.
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
     `async.Future` interface, a Stream subset, `src/developer` and
     `src/io`. Still to do for foundation users: package:listen
     (ChangeNotifier, Listenable, ValueNotifier; then BindingBase's
     `_initListenable` and the memory-allocation hooks) and vector_math's
     Matrix4. `ui.PlatformDispatcher` is minimal (platformBrightness only):
     `async.Future` interface, a Stream subset, `src/developer` and
     `src/io`. Done: package:listen 1.0.1 in `src/pub/listen`
     (ChangeNotifier, Listenable, ValueListenable, ValueNotifier, re-exported
     by foundation as aliases; `_initListenable` and the memory-allocation
     hooks). Still to do for foundation users: vector_math's Matrix4; move
     the hand-rolled notifiers (`widgets.Notifier` in paint.go, animation's
     listeners) onto `foundation.ChangeNotifier`. `ui.PlatformDispatcher` is minimal (platformBrightness only):
     the scheduler port adds frame scheduling and `AppLifecycleState`.
  4. Port `SchedulerBinding`/`Ticker`/`TickerFuture` into
     `packages/flutter/lib/src/scheduler` on `src/async`; move
     `animation` onto it (it still holds the old scheduler).
     Matrix4.
  4. Done: scheduler, in full (SchedulerBinding as `SchedulerBindingMixin`
     over `foundation.BindingBase`, Ticker, TickerFuture, Priority,
     timeDilation), with dart:ui's frame entry points (PlatformDispatcher
     onBeginFrame/onDrawFrame/scheduleFrame/onReportTimings and the engine's
     `PlatformConfiguration.BeginFrame`, partial). Widgets, the tester
     (`flutter_test.AutomatedTestWidgetsFlutterBinding`) and the window
     runner (`src/app`) run on it. Next for it:
     TickerProviderStateMixin/TickerMode in widgets.
  4b. Done: the animation library in full, 54/54 (AnimationController,
     Animation/AnimationStatus, the combinators, listener mixins, tweens,
     TweenSequence, AnimationStyle; curves were already done). The ad-hoc
     `animation.Controller` is gone: AnimatedBox and ScrollController's
     ballistic/driven activities use `AnimationController`. Still to do for
     its users: `AnimationBehavior.normal` reads the hook
     `animation.SemanticsDisableAnimations` until the semantics library
     (SemanticsBinding.disableAnimations, debugSemanticsDisableAnimations)
     is ported; ScrollController still ends its activities from a status
     listener (Flutter: `TickerFuture.whenComplete`), and widgets'
     AnimatedBox lerps the flutter-go `rendering.Color` by hand (move it to
     `ColorTween`/`SizeTween` with ui.Color when rendering switches).
  5. Split `widgets` (gestures code) and `services` (text-editing model
  5. Done: semantics, 36/36 (SemanticsNode/Owner/Configuration/Data,
     SemanticsBinding, events, SemanticsService) with dart:ui's
     semantics.dart in full (ENGINE GAP: no accessibility bridge; built
     updates are inspectable). Partial dependencies it brought:
     `src/pub/vector_math` (Matrix4 subset), painting's MatrixUtils subset
     and TransformProperty, services' TextSelection, StandardMessageCodec,
     BasicMessageChannel and SystemChannels.accessibility (no
     ServicesBinding yet: `services.DefaultBinaryMessenger` stands in).
     Next for it: port ServicesBinding (default messenger, channel
     buffers), the rest of MatrixUtils/Matrix4, and the semantics tests
     that pump widgets once rendering/widgets exist; decide the Go pattern
     for bindings stacking several mixins (see API_MAPPING).
  6. Done (P3, first part): services' messaging — message codecs (all
     five, byte-exact), MethodChannel/OptionalMethodChannel/
     BasicMessageChannel/EventChannel with profiling, BinaryMessenger,
     SystemChannels (all), ServicesBinding (partial: messaging, system/
     accessibility/lifecycle/platform handlers, app exit,
     initializationComplete, service extensions), flutter_test's
     TestDefaultBinaryMessenger(+Binding) and MockStreamHandler; dart:ui
     ChannelBuffers (full) and platform messages; `src/async` broadcast
     controllers; `src/dart` JSON/UTF-8. `services.DefaultBinaryMessenger`
     is gone (channels use ServicesBinding.instance). Next for it: put
     ServicesBinding + TestDefaultBinaryMessengerBinding into
     `flutter_test.AutomatedTestWidgetsFlutterBinding` (flush the loop
     after construction: ChannelBuffers drains in a microtask), route
     src/app's platform messages to a host (ENGINE GAP), then the rest of
     services (keyboard, text input, restoration, asset bundle, system
     chrome...).
  7. Done: bindings compose as Dart's mixins do (`foundation/
     binding_mixins.go`, API_MAPPING "Bindings": BindingBaseBase + mixin
     structs in `with` order, `BindingSuper` for the linearized `super`);
     scheduler, services, semantics and flutter_test's bindings use it, and
     flutter_test's AutomatedTestWidgetsFlutterBinding is BindingBase with
     Scheduler, Services, Gesture, Semantics, TestDefaultBinaryMessenger.
  8. Done (P4): gestures in full, with dart:ui's pointer.dart/key.dart,
     flutter_test's TestPointer and the gesture tests' helper; widgets'
     ad-hoc pointer router/velocity tracker replaced by the real
     recognizers (drag-fling difftest now exact). Next for it: port
     rendering's hit testing (RenderPointerListener, BoxHitTestResult) and
     widgets' GestureDetector/RawGestureDetector/Listener/Scrollable, then
     drop `BuildOwner.HitTestGestures` and `widgets.PointerRouter` (the
     thin bridge) and port the widget-based gesture tests skipped so far
     (transformed_*, resample_event_on_widget); flutter_test's TestGesture
     and WidgetController (tester.fling/drag on real TestGestures);
     src/app's mouse as a mouse pointer kind (it sends touch events today).
  9. Done (P1): dart:ui's drawing API over the Impeller SDK — everything
     painting.dart declares: Paint, BlendMode & co., MaskFilter,
     ColorFilter, ImageFilter, Shader/Gradient/ImageShader, Canvas,
     PictureRecorder, Picture (toImage/toImageSync through an offscreen
     Metal render + readback), Vertices, Shadow, Path/PathMetrics (Skia's
     SkPathBuilder/contains/SkContourMeasure ported bit for bit), Image,
     codecs and decoding (Go decoders), FragmentProgram/FragmentShader
     (impellerc runtime stages). src/app records frames with
     ui.PictureRecorder/ui.Canvas and presents with dart:ui's Impeller
     context. compositing.dart is done (item 12) and text.dart (item 11); next: move rendering off its interim
     `rendering.Canvas` onto ui.Canvas (painting library: BoxPainter,
     decorations, image providers). ENGINE GAPs to revisit: Path.combine
     (Skia PathOps — the Flutter SDK's prebuilt libpath_ops.dylib exposes
     an Op for lines/cubics only), AA clips (the SDK records is_aa=false),
     gamma color filters and ColorFilter-as-ImageFilter, drawVertices'
     colors/texture coordinates, blur bounds, mipmap/cubic sampling.
  10. Done: the old text-editing model (services' textedit.go/keys.go) is
     gone; RendererBinding/PaintingBinding/WidgetsBinding go into the
     bindings as their libraries are ported.
  12. Done (P3): services in full, 211/211 (keyboard with generated keys
     and maps, HardwareKeyboard/KeyEventManager/RawKeyboard and all
     platform data; text input, autofill, deltas, formatters, boundaries,
     clipboard; assets, restoration, system chrome/navigator/sound,
     haptics, platform views, mouse cursor, undo/scribe/live text/process
     text/spell check...), package:characters (`src/pub/characters`),
     `dart.RegExp`, flutter_test's key simulation, TestTextInput (+ macOS
     key handler), TestRestorationManager and fake platform views. The
     interim TextField runs on the real types through an editing bridge in
     widgets (actions.go, shortcuts.go, focus_manager.go,
     text_editing_intents.go, default_text_editing_shortcuts.go,
     editable_text.go). src/app delivers real key events (the macOS
     embedder's key responders ported) and hosts Clipboard, assets and a
     text-input stand-in. Next for it: the widgets port proper (FocusNode/
     FocusManager, Shortcuts/Actions widgets, EditableText with formatters,
     undo history and multi-line VerticalCaretMovementRun) replaces the
     bridge; painting's TextPainter.wordBoundaries replaces
     editable_text.go's interim WordBoundary; src/app: a real text input
     plugin (NSTextInputContext for IME), SystemSound/SystemNavigator/
     mouse cursor/platform views hosts (ENGINE GAPs); widgets re-exports
     package:characters.
  11. Done (P1): dart:ui text.dart in full (styles, Paragraph,
     ParagraphBuilder, placeholders, loadFontFromList,
     Canvas.drawParagraph) on the engine's own txt/SkParagraph inside the
     Impeller SDK, reached through its vtables (`src/impeller/
     txt_mirror.h`, `typography_bridge.cc`: re-check the mirrors whenever
     IMPELLER_SHA moves — `go test ./src/impeller` does). flutter_tester's
     test fonts are `ui.EngineUseTestFonts()`; flutter_test's tester and
     src/app lay text out with `rendering.UITextEngine` (ui.Paragraph),
     `FixedTextEngine` is left for benchmarks. Next for it: painting's
     TextPainter/TextSpan/StrutStyle/TextStyle on ui.ParagraphBuilder (the
     painting agent), then drop rendering's interim text engine;
     loadFontFromList's family names are case-sensitive (the engine
     lowercases them; see API_MAPPING).

  12. Done (P1): dart:ui in full (194/194): compositing.dart (SceneBuilder,
     Scene with toImage/toImageSync, EngineLayer and the typed layers with
     the retained-layer debug checks) over a Go port of the engine's layer
     tree (flow/layers + LayerStateStack, `compositing_layers.go`), the
     views and metrics of window.dart/platform_dispatcher.dart (FlutterView
     full API, Display, ViewPadding, ViewConstraints, DisplayFeature,
     DisplayCornerRadii, view focus, locales, user settings, scaleFontSize,
     onError, SystemColor, the deprecated window) with hooks.dart's entry
     points as `PlatformConfiguration` methods, plugins.dart,
     isolate_name_server.dart (on the new `src/isolate`), natives.dart's
     DartPluginRegistrant, platform_isolate.dart, annotations.dart. src/app
     reports real metrics (G7: the backing scale factor is the view's
     device pixel ratio, so gestures and drawShadow see 2.0 on Retina) and
     presents frames through SceneBuilder → FlutterView.render →
     `EngineLayerTree.Flatten`. Next for it: a real RenderView in
     rendering (its TransformLayer/OffsetLayer/PictureLayer tree →
     SceneBuilder, `addToScene`) replacing src/app's one-picture scene;
     route src/app's mouse/keys through PlatformDispatcher's packets;
     ENGINE GAPs to revisit: picture bounds (needed for culling and exact
     saveLayer bounds), texture registry, platform views, performance
     overlay, superellipse clips (path approximation, ≤2 MSAA samples).

  13. Done (P4): painting except text, 121/134 (geometry values,
     borders and every shape border, notched shapes, gradients, shadows,
     Decoration/BoxDecoration/ShapeDecoration, DecorationImage/paintImage,
     FlutterLogo, the image subsystem with ImageCache/ImageProvider/
     AssetImage/NetworkImage, PaintingBinding (in the test binding),
     ShaderWarmUp, debug, MatrixUtils, colors, TextScaler); flutter_test's
     TestRecordingCanvas and `paints` matcher; rendering's EdgeInsets/
     Alignment/Axis placeholders replaced by painting's. Next for it: the
     text item — TextPainter, TextStyle, StrutStyle, TextSpan/InlineSpan/
     PlaceholderSpan (Accumulator, InlineSpanSemanticsInformation,
     combineSemanticsInfo, PlaceholderDimensions, WordBoundary — then
     drop widgets' editable_text.go stand-in — lerpFontVariations) and
     FlutterLogoDecoration's "Flutter" label; rendering on painting
     (RenderDecoratedBox with BoxPainter, RenderImage, the `...Geometry`
     types resolved with a text direction in RenderPadding/
     RenderPositionedBox, TestRecordingPaintingContext and `paints` on
     RenderObjects); fmacheck should flag products stored in struct
     fields or passed as arguments that fuse after inlining (found by the
     paintImage difftest); ENGINE GAPs: AutomaticNotchedShape with a
     guest (Path.combine), invertColors on translucent image paints.
  14. Re-verify the dart:ui compositing mutations (steps 1-8 of its
     mutation run). A painting worker's `mut.sh` in the shared scratchpad
     ran against the dart:ui worktree while that run was in progress, so
     those results are unreliable. Re-run them on master. Agents now
     prefix scratch files uniquely (rule added to the shared brief).
  15. Done: `cmd/apiparity` counts members, constructors and parameters
     (Dart analyzer via `src/cmd/apiparity/dart`, go/packages on the Go
     side, API_MAPPING's rules, resolved test uses). Worklists per library
     in `src/docs/parity/<lib>.md`; documented renames and Go
     impossibilities in `src/docs/api_divergences.json` (each cites its
     API_MAPPING section; the tool fails on stale or unneeded entries, so
     add one with every new documented divergence the rules don't reach).
     Gaps left in the "100%" libraries (see the worklists): dart:ui
     `OffsetBase` (constructor, `==`, `hashCode`, `toString`: an interface
     in Go), `UniformArray[]` (Go `At`, undocumented),
     `instantiateImageCodec`/`instantiateImageCodecFromBuffer`/
     `decodeImageFromPixels`'s `allowUpscaling`; foundation `BitField`
     `[]`/`[]=` and `PersistentHashMap[]` (Go `Get`/`Set`, undocumented),
     `CachingIterable`'s Iterable overrides (`map`, `where`, `expand`,
     `take`, `takeWhile`, `skip`, `skipWhile`), `Key.empty`,
     `ObjectEvent()`, `SynchronousFuture.asStream`; scheduler
     `TickerFuture.asStream`; physics `Simulation.toString` (SimulationBase
     has no `String`); services `KeyEvent()`/`RawKeyEvent()` (no base
     set-up for subclasses), `TextInput.scribbleInProgress` (an instance
     getter of a private singleton); gestures `PointerSignalEvent()`,
     `PositionedGestureDetails()`, `BaseTapAndDragGestureRecognizer`'s
     `eagerVictoryOnDrag` constructor parameter. Either port them or
     document their Go form in API_MAPPING (then a divergence entry). The
     larger gap is tests: ~40% of exposed members of these libraries no Go
     test refers to (services' generated key constants account for most).

## Environment

| Need | Where |
|---|---|
| Impeller SDK (not in git) | `./src/scripts/fetch_impeller_sdk.sh` (into `src/third_party/`) |
| Flutter SDK + source to port from | `~/Projetos/jonathanperis/flutter`, checked out at tag 3.50.1 (detached). Don't move it without changing the pin. |
| Reference tests | `FLUTTER=~/Projetos/jonathanperis/flutter/bin/flutter ./src/scripts/difftest.sh` |
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
- Engine float code (Skia, DisplayList, impeller geometry) is compiled by
  clang for arm64 with -ffp-contract=on: `a*b + c` in one expression is a
  fused multiply-add, and sinf+cosf of one value is one `__sincosf_stret`
  call. Ports of that C++ must reproduce both (see PLAN, dart:ui drawing).
- Pixel references come from `flutter test --enable-impeller`
  (`src/difftest/flutter/test/impeller/`, written only in that mode); plain
  `flutter test` is Skia's software renderer.
- Window screenshots (`screencapture -l`) stopped working on this machine;
  environmental.
- Benchmarks vary about ±30% between runs here.
- Equal widgets skip rebuilds (Go's stand-in for `const`), a documented
  divergence.
