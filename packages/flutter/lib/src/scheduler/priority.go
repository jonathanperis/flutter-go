package scheduler

// Port of priority.dart (Flutter 3.50.1).

// Priority is a task priority, as passed to ScheduleTask
// (SchedulerBinding.scheduleTask).
type Priority struct {
	value int
}

// Value is the integer that describes this priority.
func (p Priority) Value() int { return p.value }

var (
	// PriorityIdle is for a task to run after all other tasks, when no
	// animations are running.
	PriorityIdle = Priority{0}
	// PriorityAnimation is for a task to run even when animations are
	// running.
	PriorityAnimation = Priority{100000}
	// PriorityTouch is for a task to run even when the user is interacting
	// with the device.
	PriorityTouch = Priority{200000}
)

// PriorityKMaxOffset is the maximum offset by which to clamp relative
// priorities (Priority.kMaxOffset). It is still possible to have
// priorities that are offset by more than this amount by repeatedly taking
// relative offsets, but that is generally discouraged.
const PriorityKMaxOffset = 10000

// Add is Dart's operator +: a priority relative to this one (a positive
// offset is a higher priority), the offset clamped to ±PriorityKMaxOffset.
func (p Priority) Add(offset int) Priority {
	if abs(offset) > PriorityKMaxOffset {
		// Clamp the input offset.
		offset = PriorityKMaxOffset * sign(offset)
	}
	return Priority{p.value + offset}
}

// Sub is Dart's operator -: a priority relative to this one (a positive
// offset is a lower priority), the offset clamped to ±PriorityKMaxOffset.
func (p Priority) Sub(offset int) Priority { return p.Add(-offset) }

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// sign is int.sign.
func sign(i int) int {
	switch {
	case i > 0:
		return 1
	case i < 0:
		return -1
	}
	return 0
}
