package impeller

// Reference counting for display lists shared between owners (dart:ui's
// Picture and the compositor's layers hold the same display list, as the
// engine's sk_sp<DisplayList> does).

/*
#include "impeller.h"
*/
import "C"

// Retain adds a reference to the display list; each Retain is balanced by a
// Release.
func (d *DisplayList) Retain() { C.ImpellerDisplayListRetain(d.h) }
