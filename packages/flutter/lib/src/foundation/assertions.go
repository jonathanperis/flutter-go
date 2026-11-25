package foundation

// Partial port of assertions.dart: only the message-carrying FlutterError
// so far. FlutterErrorDetails, reportError and the diagnostics-based
// FlutterError.fromParts come with diagnostics.dart (see flutter_go/PLAN.md).

// FlutterError is an error in the use of the framework.
type FlutterError struct{ Message string }

// NewFlutterError is FlutterError(message).
func NewFlutterError(message string) *FlutterError { return &FlutterError{message} }

func (e *FlutterError) Error() string  { return e.Message }
func (e *FlutterError) String() string { return e.Message }
