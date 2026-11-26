package dart

// StateError is dart:core's StateError(message).
type StateError struct{ Message string }

func (e StateError) Error() string { return "Bad state: " + e.Message }

// ArgumentError is dart:core's ArgumentError: ArgumentError(message) when
// HasValue is false, ArgumentError.value(value, name, message) otherwise.
type ArgumentError struct {
	Message  string
	Name     string // "" when no argument name is given
	Value    any
	HasValue bool
}

func (e ArgumentError) Error() string {
	if !e.HasValue {
		s := "Invalid argument(s)"
		if e.Name != "" {
			s += " (" + e.Name + ")"
		}
		if e.Message != "" {
			s += ": " + e.Message
		}
		return s
	}
	s := "Invalid argument"
	if e.Name != "" {
		s += " (" + e.Name + ")"
	}
	if e.Message != "" {
		s += ": " + e.Message
	}
	return s + ": " + SafeToString(e.Value)
}
