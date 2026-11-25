package foundation

// Port of error_dumper.dart (Flutter 3.50.1), which exports
// _error_dumper_io.dart on native platforms: ErrorToConsoleDumper. The web
// variant (console.error plus listeners) doesn't apply to flutter-go, so the
// listener methods do nothing, as on io.

// errorToConsoleDumper is the type of ErrorToConsoleDumper.
type errorToConsoleDumper struct{}

// ErrorToConsoleDumper is ErrorToConsoleDumper, a class of static members
// (see API_MAPPING): ErrorToConsoleDumper.Dump(message).
var ErrorToConsoleDumper errorToConsoleDumper

// Dump dumps an error message to the console (through DebugPrint).
func (errorToConsoleDumper) Dump(message string) { DebugPrint(message) }

// AddWebDumpListener does nothing outside the web.
func (errorToConsoleDumper) AddWebDumpListener(listener func(message string)) {}

// ClearWebDumpListeners does nothing outside the web.
func (errorToConsoleDumper) ClearWebDumpListeners() {}
