// Reference side of flutter-go's dart:ui window/platform differential
// test: the implicit FlutterView and PlatformDispatcher as flutter_tester
// configures them (metrics, display, locales, settings), the deprecated
// window, ViewPadding, ViewConstraints, DisplayFeature, DisplayCornerRadii,
// GestureSettings, ViewFocusEvent, SystemColor, CallbackHandle,
// PluginUtilities, IsolateNameServer, platform isolates, keepToString.
// Output: build/window_api.json as {case: value}.
import 'dart:convert';
import 'dart:io';
import 'dart:isolate';
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';

String err(void Function() f) {
  try {
    f();
  } catch (e) {
    if (e is AssertionError) {
      return 'AssertionError: ${e.message}';
    }
    return '$e';
  }
  return 'no error';
}

@pragma('vm:entry-point')
String topLevelCallback() => 'top';

Object? viewState(FlutterView v) => <String, Object?>{
  'toString': '$v',
  'viewId': v.viewId,
  'devicePixelRatio': v.devicePixelRatio,
  'physicalSize': '${v.physicalSize}',
  'physicalConstraints': '${v.physicalConstraints}',
  'padding': '${v.padding}',
  'viewInsets': '${v.viewInsets}',
  'viewPadding': '${v.viewPadding}',
  'systemGestureInsets': '${v.systemGestureInsets}',
  'gestureSettings': '${v.gestureSettings}',
  'displayFeatures': '${v.displayFeatures}',
  'displayCornerRadii': '${v.displayCornerRadii}',
  'display': '${v.display}',
};

void main() {
  test('window_api', () {
    final Map<String, Object?> out = <String, Object?>{};
    final PlatformDispatcher pd = PlatformDispatcher.instance;
    final FlutterView view = pd.implicitView!;

    // ---- the implicit view and display
    out['view'] = viewState(view);
    out['views'] = pd.views.map((FlutterView v) => '$v').toList();
    out['view.0.identical'] = identical(pd.view(id: 0), view);
    out['view.1'] = '${pd.view(id: 1)}';
    out['displays'] = pd.displays.map((Display d) => '$d').toList();
    out['display.fields'] = <Object>[
      view.display.id,
      view.display.devicePixelRatio,
      '${view.display.size}',
      view.display.refreshRate,
    ];

    // ---- the dispatcher's configuration
    out['dispatcher'] = <String, Object?>{
      'locale': '${pd.locale}',
      'locale.languageTag': pd.locale.toLanguageTag(),
      'locales': pd.locales.map((Locale l) => '$l').toList(),
      'textScaleFactor': pd.textScaleFactor,
      'alwaysUse24HourFormat': pd.alwaysUse24HourFormat,
      'platformBrightness': '${pd.platformBrightness}',
      'systemFontFamily': '${pd.systemFontFamily}',
      'nativeSpellCheckServiceDefined': pd.nativeSpellCheckServiceDefined,
      'supportsShowingSystemContextMenu': pd.supportsShowingSystemContextMenu,
      'brieflyShowPassword': pd.brieflyShowPassword,
      'defaultRouteName': pd.defaultRouteName,
      'semanticsEnabled': pd.semanticsEnabled,
      'accessibilityFeatures': '${pd.accessibilityFeatures}',
      'frameData.frameNumber': pd.frameData.frameNumber,
      'lineHeightScaleFactorOverride': '${pd.lineHeightScaleFactorOverride}',
      'letterSpacingOverride': '${pd.letterSpacingOverride}',
      'wordSpacingOverride': '${pd.wordSpacingOverride}',
      'paragraphSpacingOverride': '${pd.paragraphSpacingOverride}',
      'persistentIsolateData': '${pd.getPersistentIsolateData()}',
      'onMetricsChanged': '${pd.onMetricsChanged}',
      'onLocaleChanged': '${pd.onLocaleChanged}',
      'onError': '${pd.onError}',
      'onViewFocusChange': '${pd.onViewFocusChange}',
      'computePlatformResolvedLocale': '${pd.computePlatformResolvedLocale(const <Locale>[
            Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hans', countryCode: 'CN'),
            Locale('en', 'US'),
          ])}',
      'computePlatformResolvedLocale.empty': '${pd.computePlatformResolvedLocale(const <Locale>[])}',
    };
    out['scaleFontSize'] = <double>[0, 1, 2, 3.4, 14.5, 1000.25]
        .map((double v) => pd.scaleFontSize(v))
        .toList();
    out['scaleFontSize.errors'] = <String>[
      err(() => pd.scaleFontSize(-1)),
      err(() => pd.scaleFontSize(double.infinity)),
    ];
    out['natives'] = <String>[
      err(() => pd.requestViewFocusChange(
            viewId: 0, state: ViewFocusState.focused, direction: ViewFocusDirection.forward)),
      err(() => pd.setApplicationLocale(const Locale('fr', 'CA'))),
      err(() => pd.setIsolateDebugName('difftest')),
      err(() => DartPluginRegistrant.ensureInitialized()),
    ];

    // ---- the deprecated window
    // ignore: deprecated_member_use
    final SingletonFlutterWindow w = window;
    out['window'] = <String, Object?>{
      ...viewState(w)! as Map<String, Object?>,
      'locale': '${w.locale}',
      'locales': '${w.locales}',
      'textScaleFactor': w.textScaleFactor,
      'alwaysUse24HourFormat': w.alwaysUse24HourFormat,
      'platformBrightness': '${w.platformBrightness}',
      'systemFontFamily': '${w.systemFontFamily}',
      'defaultRouteName': w.defaultRouteName,
      'semanticsEnabled': w.semanticsEnabled,
      'accessibilityFeatures': '${w.accessibilityFeatures}',
      'brieflyShowPassword': w.brieflyShowPassword,
      'nativeSpellCheckServiceDefined': w.nativeSpellCheckServiceDefined,
      'supportsShowingSystemContextMenu': w.supportsShowingSystemContextMenu,
      'frameData.frameNumber': w.frameData.frameNumber,
      'computePlatformResolvedLocale': '${w.computePlatformResolvedLocale(const <Locale>[Locale('en')])}',
      'identicalDispatcher': identical(w.platformDispatcher, pd),
    };

    // ---- ViewPadding
    out['viewPadding.zero'] = '${ViewPadding.zero}';
    // ignore: deprecated_member_use
    out['windowPadding.zero'] = '${WindowPadding.zero}';
    out['viewPadding.fields'] = <double>[
      ViewPadding.zero.left,
      ViewPadding.zero.top,
      ViewPadding.zero.right,
      ViewPadding.zero.bottom,
    ];

    // ---- ViewConstraints
    final List<ViewConstraints> constraints = <ViewConstraints>[
      const ViewConstraints(),
      ViewConstraints.tight(const Size(200, 300)),
      const ViewConstraints(minWidth: 100, maxWidth: 200, minHeight: 300, maxHeight: 400),
      const ViewConstraints(minWidth: double.infinity, minHeight: double.infinity),
      const ViewConstraints(minWidth: double.infinity, maxHeight: 10),
      const ViewConstraints(maxWidth: 1.25, maxHeight: 0.05),
      const ViewConstraints(minWidth: 2.35, maxWidth: 2.45, minHeight: 0.15, maxHeight: 0.15),
      const ViewConstraints(minWidth: 0, maxWidth: double.infinity, minHeight: 5),
      const ViewConstraints(maxWidth: 0, maxHeight: 0),
      ViewConstraints.tight(Size.zero),
      const ViewConstraints(minWidth: double.nan, maxWidth: double.nan),
    ];
    out['viewConstraints'] = constraints.map((ViewConstraints c) => <Object?>[
      '$c',
      c.minWidth.toString(),
      c.maxWidth.toString(),
      c.minHeight.toString(),
      c.maxHeight.toString(),
      c.isTight,
      c.isSatisfiedBy(const Size(200, 300)),
      c.isSatisfiedBy(const Size(400, 500)),
      c.isSatisfiedBy(Size.zero),
      '${c / 2}',
      '${c / 3}',
      '${c / 0}',
      c == ViewConstraints.tight(const Size(200, 300)),
      c == const ViewConstraints(),
    ]).toList();

    // ---- DisplayFeature, DisplayCornerRadii
    const DisplayFeature fold = DisplayFeature(
      bounds: Rect.fromLTRB(10, 0, 12, 600),
      type: DisplayFeatureType.fold,
      state: DisplayFeatureState.postureHalfOpened,
    );
    out['displayFeature'] = <Object>[
      '$fold',
      fold == const DisplayFeature(
        bounds: Rect.fromLTRB(10, 0, 12, 600),
        type: DisplayFeatureType.fold,
        state: DisplayFeatureState.postureHalfOpened,
      ),
      fold == const DisplayFeature(
        bounds: Rect.fromLTRB(10, 0, 12, 600),
        type: DisplayFeatureType.hinge,
        state: DisplayFeatureState.postureHalfOpened,
      ),
      err(() => DisplayFeature(
            bounds: Rect.zero,
            type: DisplayFeatureType.values[3],
            state: DisplayFeatureState.values[1],
          )),
      '${const DisplayFeature(bounds: Rect.zero, type: DisplayFeatureType.cutout, state: DisplayFeatureState.unknown)}',
    ];
    out['displayFeature.enums'] = <Object>[
      DisplayFeatureType.values.map((DisplayFeatureType t) => '$t').toList(),
      DisplayFeatureState.values.map((DisplayFeatureState s) => '$s').toList(),
    ];
    const DisplayCornerRadii radii = DisplayCornerRadii(
      topLeft: 1, topRight: 2.5, bottomRight: 0, bottomLeft: 4,
    );
    out['displayCornerRadii'] = <Object>[
      '$radii',
      radii == const DisplayCornerRadii(topLeft: 1, topRight: 2.5, bottomRight: 0, bottomLeft: 4),
      radii == const DisplayCornerRadii(topLeft: 1, topRight: 2.5, bottomRight: 0, bottomLeft: 5),
      err(() => DisplayCornerRadii(topLeft: double.parse('-1'), topRight: 0, bottomRight: 0, bottomLeft: 0)),
    ];

    // ---- GestureSettings
    const GestureSettings gs = GestureSettings(physicalDoubleTapSlop: 2.0, physicalTouchSlop: 1.0);
    out['gestureSettings'] = <Object>[
      '$gs',
      '${const GestureSettings()}',
      '${gs.copyWith()}',
      '${gs.copyWith(physicalTouchSlop: 3)}',
      '${const GestureSettings().copyWith(physicalDoubleTapSlop: 0.5)}',
      gs == const GestureSettings(physicalDoubleTapSlop: 2.0, physicalTouchSlop: 1.0),
      gs == const GestureSettings(physicalTouchSlop: 1.0),
    ];

    // ---- view focus, enums
    out['viewFocusEvent'] = '${const ViewFocusEvent(viewId: 3, state: ViewFocusState.unfocused, direction: ViewFocusDirection.backward)}';
    out['enums'] = <Object>[
      ViewFocusState.values.map((ViewFocusState v) => '$v').toList(),
      ViewFocusDirection.values.map((ViewFocusDirection v) => '$v').toList(),
      Brightness.values.map((Brightness v) => '$v').toList(),
    ];

    // ---- SystemColor
    out['systemColor'] = <Object>[
      SystemColor.platformProvidesSystemColors,
      '${SystemColor.light.brightness}',
      '${SystemColor.dark.brightness}',
      err(() => SystemColor.light.accentColor),
      err(() => SystemColor.dark.visitedText),
      const SystemColor(name: 'x').isSupported,
      const SystemColor(name: 'y', value: Color(0xFF000000)).isSupported,
      const SystemColor(name: 'z').name,
    ];

    // ---- plugins
    final CallbackHandle? hTop = PluginUtilities.getCallbackHandle(topLevelCallback);
    final Function anon = (int a, int b) => a + b; // ignore: prefer_function_declarations_over_variables
    out['plugins'] = <Object?>[
      hTop != null,
      PluginUtilities.getCallbackHandle(topLevelCallback) == hTop,
      (PluginUtilities.getCallbackFromHandle(hTop!)! as String Function())(),
      PluginUtilities.getCallbackHandle(anon) == null,
      CallbackHandle.fromRawHandle(5) == CallbackHandle.fromRawHandle(5),
      CallbackHandle.fromRawHandle(5) == CallbackHandle.fromRawHandle(6),
      CallbackHandle.fromRawHandle(7).toRawHandle(),
      '${PluginUtilities.getCallbackFromHandle(CallbackHandle.fromRawHandle(12345))}',
    ];

    // ---- IsolateNameServer
    const String portName = 'window_api_port';
    final ReceivePort receivePort = ReceivePort();
    final ReceivePort receivePort2 = ReceivePort();
    out['isolateNameServer'] = <Object?>[
      IsolateNameServer.lookupPortByName(portName) == null,
      IsolateNameServer.removePortNameMapping(portName),
      IsolateNameServer.registerPortWithName(receivePort.sendPort, portName),
      IsolateNameServer.lookupPortByName(portName) == receivePort.sendPort,
      IsolateNameServer.registerPortWithName(receivePort2.sendPort, portName),
      IsolateNameServer.lookupPortByName(portName) == receivePort.sendPort,
      IsolateNameServer.removePortNameMapping(portName),
      IsolateNameServer.lookupPortByName(portName) == null,
      IsolateNameServer.registerPortWithName(receivePort2.sendPort, portName),
      IsolateNameServer.lookupPortByName(portName) == receivePort2.sendPort,
      IsolateNameServer.removePortNameMapping(portName),
    ];
    receivePort.close();
    receivePort2.close();

    // ---- platform isolates, annotations
    out['platformIsolate'] = <Object>[
      isRunningOnPlatformThread,
      err(() => runOnPlatformThread<int>(() => 1)),
    ];
    out['keepToString'] = <Object?>[keepToString.name, '${keepToString.options}'];

    Directory('build').createSync(recursive: true);
    File('build/window_api.json').writeAsStringSync(const JsonEncoder.withIndent(' ').convert(out));
  });
}
