package dart

// Pragma is dart:core's pragma: a hint to tools (the compiler, the
// runtime) attached to a declaration. Go has no annotations, so a pragma
// is a value that documents intent (dart:ui's KeepToString).
type Pragma struct {
	// Name is the pragma's name ("vm:entry-point").
	Name string
	// Options are its options (nil for none).
	Options any
}

func (p Pragma) String() string { return "Instance of 'pragma'" }
