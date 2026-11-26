package foundation

import "reflect"

// ObjectRuntimeType is objectRuntimeType: the object's runtime type name in
// debug builds, optimizedValue otherwise (Dart only spends the reflection in
// debug mode).
func ObjectRuntimeType(object any, optimizedValue string) string {
	if !KDebugMode || object == nil {
		if object == nil && KDebugMode {
			return "Null"
		}
		return optimizedValue
	}
	// Dart's runtimeType.toString(): a generic type prints its arguments
	// with <> and Dart's names (ValueNotifier<int>, not ValueNotifier[int]).
	return dartTypeName(reflect.TypeOf(object))
}
