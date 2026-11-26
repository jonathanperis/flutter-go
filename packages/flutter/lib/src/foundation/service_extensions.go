package foundation

// Port of service_extensions.dart (Flutter 3.50.1).

// FoundationServiceExtensions are the service extensions BindingBase
// registers (each as "ext.flutter.<name>").
type FoundationServiceExtensions int

const (
	// FoundationServiceExtensionsReassemble: name of the reassemble service
	// extension (BindingBase.reassembleApplication).
	FoundationServiceExtensionsReassemble FoundationServiceExtensions = iota
	// FoundationServiceExtensionsExit: name of the exit service extension
	// (not registered on the web).
	FoundationServiceExtensionsExit
	// FoundationServiceExtensionsConnectedVmServiceUri: name of the service
	// extension that reads and writes connectedVmServiceUri.
	FoundationServiceExtensionsConnectedVmServiceUri
	// FoundationServiceExtensionsActiveDevToolsServerAddress: name of the
	// service extension that reads and writes activeDevToolsServerAddress.
	FoundationServiceExtensionsActiveDevToolsServerAddress
	// FoundationServiceExtensionsPlatformOverride: name of the service
	// extension that overrides the target platform (debug builds).
	FoundationServiceExtensionsPlatformOverride
	// FoundationServiceExtensionsBrightnessOverride: name of the service
	// extension that overrides the platform brightness (debug builds).
	FoundationServiceExtensionsBrightnessOverride
)

var foundationServiceExtensionNames = [...]string{
	"reassemble", "exit", "connectedVmServiceUri", "activeDevToolsServerAddress",
	"platformOverride", "brightnessOverride",
}

// Name is the enum value's name (Dart's EnumName.name).
func (e FoundationServiceExtensions) Name() string { return foundationServiceExtensionNames[e] }

func (e FoundationServiceExtensions) String() string {
	return "FoundationServiceExtensions." + e.Name()
}
