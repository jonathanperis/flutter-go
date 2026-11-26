package gestures

// Port of eager.dart (Flutter 3.50.1).

// EagerGestureRecognizer accepts every pointer it is given (EagerGestureRecognizer
// is useful to stop other recognizers from recognizing gestures).
type EagerGestureRecognizer struct {
	OneSequenceGestureRecognizerBase
}

// NewEagerGestureRecognizer is EagerGestureRecognizer({supportedDevices,
// allowedButtonsFilter}).
func NewEagerGestureRecognizer(opts ...GestureRecognizerOptions) *EagerGestureRecognizer {
	r := &EagerGestureRecognizer{}
	var o GestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r.Construct(r, GestureRecognizerOptions{SupportedDevices: o.SupportedDevices, AllowedButtonsFilter: o.AllowedButtonsFilter})
	return r
}

func (r *EagerGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	r.OneSequenceGestureRecognizerBase.AddAllowedPointer(event)
	r.Resolve(GestureDispositionAccepted)
	r.StopTrackingPointer(event.Pointer())
}

func (r *EagerGestureRecognizer) DebugDescription() string { return "eager" }

func (r *EagerGestureRecognizer) DidStopTrackingLastPointer(pointer int) {}

func (r *EagerGestureRecognizer) HandleEvent(event PointerEvent) {}
