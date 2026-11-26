package services

// Port of flutter_version.dart (Flutter 3.50.1).

// The FLUTTER_* compile-time defines the flutter tool passes (set with
// `-ldflags "-X <this package>.flutterVersionDefine=..."`;
// undefinedDefine means not defined).
var (
	flutterVersionDefine           = undefinedDefine
	flutterChannelDefine           = undefinedDefine
	flutterGitURLDefine            = undefinedDefine
	flutterFrameworkRevisionDefine = undefinedDefine
	flutterEngineRevisionDefine    = undefinedDefine
	flutterDartVersionDefine       = undefinedDefine
)

// flutterVersion is the type of FlutterVersion (Dart's abstract final class
// of static constants).
type flutterVersion struct {
	// Version is the Flutter version used to compile the app (nil when
	// not defined).
	Version *string
	// Channel is the Flutter channel used to compile the app.
	Channel *string
	// GitURL is the URL of the Git repository Flutter was obtained from
	// (Dart: gitUrl).
	GitURL *string
	// FrameworkRevision is the Flutter framework revision, as a (short)
	// Git commit ID.
	FrameworkRevision *string
	// EngineRevision is the Flutter engine revision.
	EngineRevision *string
	// DartVersion is the Dart version used to compile the app.
	DartVersion *string
}

// FlutterVersion holds the details of the Flutter version this app was
// compiled with, from the defines the flutter tool passes (nil when not
// defined, as when the app isn't built by the flutter tool).
var FlutterVersion = flutterVersion{
	Version:           fromEnvironment(flutterVersionDefine),
	Channel:           fromEnvironment(flutterChannelDefine),
	GitURL:            fromEnvironment(flutterGitURLDefine),
	FrameworkRevision: fromEnvironment(flutterFrameworkRevisionDefine),
	EngineRevision:    fromEnvironment(flutterEngineRevisionDefine),
	DartVersion:       fromEnvironment(flutterDartVersionDefine),
}
