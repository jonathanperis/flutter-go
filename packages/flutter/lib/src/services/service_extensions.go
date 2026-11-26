package services

// Port of service_extensions.dart (Flutter 3.50.1).

// ServicesServiceExtensions are the service extensions the services
// library registers (as ext.flutter.<name>).
type ServicesServiceExtensions int

const (
	// ServicesServiceExtensionsProfilePlatformChannels reads and sets
	// DebugProfilePlatformChannels (non-release builds).
	ServicesServiceExtensionsProfilePlatformChannels ServicesServiceExtensions = iota
	// ServicesServiceExtensionsEvict evicts an asset from the root bundle's
	// cache (debug builds).
	ServicesServiceExtensionsEvict
)

// ServicesServiceExtensionsValues is ServicesServiceExtensions.values.
var ServicesServiceExtensionsValues = []ServicesServiceExtensions{
	ServicesServiceExtensionsProfilePlatformChannels, ServicesServiceExtensionsEvict,
}

// Name is the enum value's name (Dart's EnumName.name).
func (e ServicesServiceExtensions) Name() string {
	return [...]string{"profilePlatformChannels", "evict"}[e]
}

func (e ServicesServiceExtensions) String() string {
	return "ServicesServiceExtensions." + e.Name()
}
