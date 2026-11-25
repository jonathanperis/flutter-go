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
	t := reflect.TypeOf(object)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}
