package dart

import "strconv"

// IndexError is dart:core's IndexError (IndexError.withLength, and what
// IndexError.check and the VM's list and typed data index checks throw):
// Index is not a valid index into an indexable object of length Length.
// Name "" stands for a null name; Message "" for the default "Index out of
// range".
type IndexError struct {
	Index, Length int
	Name          string
	Message       string
}

// Error is IndexError.toString (ArgumentError's, with RangeError's name).
func (e IndexError) Error() string {
	s := "RangeError"
	if e.Name != "" {
		s += " (" + e.Name + ")"
	}
	msg := e.Message
	if msg == "" {
		msg = "Index out of range"
	}
	s += ": " + msg
	switch {
	case e.Index < 0:
		s += ": index must not be negative"
	case e.Length == 0:
		s += ": no indices are valid"
	default:
		s += ": index should be less than " + strconv.Itoa(e.Length)
	}
	return s + ": " + strconv.Itoa(e.Index)
}
