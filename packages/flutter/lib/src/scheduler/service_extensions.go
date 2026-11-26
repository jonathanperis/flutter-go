package scheduler

// Port of service_extensions.dart (Flutter 3.50.1).

// SchedulerServiceExtensions are the service extensions the scheduler
// library registers (as ext.flutter.<name>).
type SchedulerServiceExtensions int

const (
	// SchedulerServiceExtensionsTimeDilation reads and sets timeDilation
	// (registered by SchedulerBinding.InitServiceExtensions).
	SchedulerServiceExtensionsTimeDilation SchedulerServiceExtensions = iota
)

// SchedulerServiceExtensionsValues is SchedulerServiceExtensions.values.
var SchedulerServiceExtensionsValues = []SchedulerServiceExtensions{SchedulerServiceExtensionsTimeDilation}

// Name is the enum value's name (Dart's EnumName.name).
func (e SchedulerServiceExtensions) Name() string {
	return [...]string{"timeDilation"}[e]
}

func (e SchedulerServiceExtensions) String() string {
	return "SchedulerServiceExtensions." + e.Name()
}
