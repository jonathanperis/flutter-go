package services

// Port of app_version.dart (Flutter 3.50.1).

// The FLUTTER_BUILD_NAME and FLUTTER_BUILD_NUMBER compile-time defines
// (Dart's bool.hasEnvironment/String.fromEnvironment): set them with
// `-ldflags "-X <this package>.appBuildNameDefine=..."`. Go's -X sets
// strings only, so undefinedDefine (a value no define can hold) stands for
// "not defined".
var (
	appBuildNameDefine   = undefinedDefine
	appBuildNumberDefine = undefinedDefine
)

// undefinedDefine marks a compile-time define as not given.
const undefinedDefine = "\x00undefined"

func fromEnvironment(define string) *string {
	if define == undefinedDefine {
		return nil
	}
	return &define
}

// AppBuildName is appBuildName: the application's build name
// (FLUTTER_BUILD_NAME, from --build-name), nil when not defined.
var AppBuildName = fromEnvironment(appBuildNameDefine)

// AppBuildNumber is appBuildNumber: the application's build number
// (FLUTTER_BUILD_NUMBER, from --build-number), nil when not defined.
var AppBuildNumber = fromEnvironment(appBuildNumberDefine)
