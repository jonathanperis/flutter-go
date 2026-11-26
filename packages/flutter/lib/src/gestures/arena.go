package gestures

// Port of arena.dart (Flutter 3.50.1).

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// GestureDisposition is whether the gesture was accepted or rejected.
type GestureDisposition int

const (
	// GestureDispositionAccepted: this gesture was accepted as the
	// interpretation of the user's input.
	GestureDispositionAccepted GestureDisposition = iota
	// GestureDispositionRejected: this gesture was rejected as the
	// interpretation of the user's input.
	GestureDispositionRejected
)

// Name is the enum value's name (Dart's .name).
func (d GestureDisposition) Name() string { return [...]string{"accepted", "rejected"}[d] }

func (d GestureDisposition) String() string { return "GestureDisposition." + d.Name() }

// GestureArenaMember represents an object participating in an arena.
// Members compare by identity: use pointers.
type GestureArenaMember interface {
	// AcceptGesture is called when this member wins the arena for pointer.
	AcceptGesture(pointer int)
	// RejectGesture is called when this member loses the arena for pointer.
	RejectGesture(pointer int)
}

// GestureArenaEntry is an interface to pass information to an arena (a
// member's entry in the arena of a pointer). Dart's class is implemented by
// team.dart's combining entry, so it is an interface.
type GestureArenaEntry interface {
	// Resolve calls the member's AcceptGesture or RejectGesture (now or when
	// the arena resolves).
	Resolve(disposition GestureDisposition)
}

// gestureArenaEntry is the arena's GestureArenaEntry.
type gestureArenaEntry struct {
	arena   *GestureArenaManager
	pointer int
	member  GestureArenaMember
}

// Resolve calls the member's AcceptGesture or RejectGesture (now or when
// the arena resolves).
func (e *gestureArenaEntry) Resolve(disposition GestureDisposition) {
	e.arena.resolve(e.pointer, e.member, disposition)
}

// gestureArena is _GestureArena.
type gestureArena struct {
	members         []GestureArenaMember
	isOpen          bool
	isHeld          bool
	hasPendingSweep bool
	// eagerWinner is the member that tried to win while the arena was open
	// (nil: none).
	eagerWinner GestureArenaMember
}

func (a *gestureArena) add(member GestureArenaMember) {
	dartAssert(a.isOpen, nil)
	a.members = append(a.members, member)
}

func (a *gestureArena) String() string {
	var buffer strings.Builder
	if len(a.members) == 0 {
		buffer.WriteString("<empty>")
	} else {
		parts := make([]string, len(a.members))
		for i, member := range a.members {
			if member == a.eagerWinner {
				parts[i] = dart.ToString(member) + " (eager winner)"
			} else {
				parts[i] = dart.ToString(member)
			}
		}
		buffer.WriteString(strings.Join(parts, ", "))
	}
	if a.isOpen {
		buffer.WriteString(" [open]")
	}
	if a.isHeld {
		buffer.WriteString(" [held]")
	}
	if a.hasPendingSweep {
		buffer.WriteString(" [hasPendingSweep]")
	}
	return buffer.String()
}

// GestureArenaManager is the central manager for gesture arenas used by
// the gesture recognition machinery: one arena per pointer.
type GestureArenaManager struct {
	arenas map[int]*gestureArena
}

// NewGestureArenaManager creates a manager with no arenas.
func NewGestureArenaManager() *GestureArenaManager {
	return &GestureArenaManager{arenas: map[int]*gestureArena{}}
}

// Add adds a new member (e.g., gesture recognizer) to the arena of
// pointer, opening it if needed.
func (m *GestureArenaManager) Add(pointer int, member GestureArenaMember) GestureArenaEntry {
	state, ok := m.arenas[pointer]
	if !ok {
		m.debugLogDiagnostic(pointer, "★ Opening new gesture arena.", nil)
		state = &gestureArena{isOpen: true}
		m.arenas[pointer] = state
	}
	state.add(member)
	if foundation.KDebugMode && DebugPrintGestureArenaDiagnostics {
		m.debugLogDiagnostic(pointer, "Adding: "+dart.ToString(member), nil)
	}
	return &gestureArenaEntry{arena: m, pointer: pointer, member: member}
}

// Close prevents new members from entering the arena; it is called after
// the framework has finished dispatching the pointer down event.
func (m *GestureArenaManager) Close(pointer int) {
	state, ok := m.arenas[pointer]
	if !ok {
		return // This arena either never existed or has been resolved.
	}
	state.isOpen = false
	m.debugLogDiagnostic(pointer, "Closing", state)
	m.tryToResolveArena(pointer, state)
}

// Sweep forces resolution of the arena, giving the win to the first
// member; it is called after the pointer up event (unless the arena is
// held, then the sweep happens on Release).
func (m *GestureArenaManager) Sweep(pointer int) {
	state, ok := m.arenas[pointer]
	if !ok {
		return // This arena either never existed or has been resolved.
	}
	dartAssert(!state.isOpen, nil)
	if state.isHeld {
		state.hasPendingSweep = true
		m.debugLogDiagnostic(pointer, "Delaying sweep", state)
		return // This arena is being held for a long-lived member.
	}
	m.debugLogDiagnostic(pointer, "Sweeping", state)
	delete(m.arenas, pointer)
	if len(state.members) > 0 {
		// First member wins.
		if foundation.KDebugMode && DebugPrintGestureArenaDiagnostics {
			m.debugLogDiagnostic(pointer, "Winner: "+dart.ToString(state.members[0]), nil)
		}
		state.members[0].AcceptGesture(pointer)
		// Give all the other members the bad news.
		for i := 1; i < len(state.members); i++ {
			state.members[i].RejectGesture(pointer)
		}
	}
}

// Hold prevents the arena from being swept (members that wait for a
// longer gesture, such as double taps, hold it).
func (m *GestureArenaManager) Hold(pointer int) {
	state, ok := m.arenas[pointer]
	if !ok {
		return // This arena either never existed or has been resolved.
	}
	state.isHeld = true
	m.debugLogDiagnostic(pointer, "Holding", state)
}

// Release releases a hold, sweeping the arena if a sweep was delayed.
func (m *GestureArenaManager) Release(pointer int) {
	state, ok := m.arenas[pointer]
	if !ok {
		return // This arena either never existed or has been resolved.
	}
	state.isHeld = false
	m.debugLogDiagnostic(pointer, "Releasing", state)
	if state.hasPendingSweep {
		m.Sweep(pointer)
	}
}

// resolve is _resolve: reject or accept a gesture recognizer (called by
// GestureArenaEntry.Resolve).
func (m *GestureArenaManager) resolve(pointer int, member GestureArenaMember, disposition GestureDisposition) {
	state, ok := m.arenas[pointer]
	if !ok {
		return // This arena has already resolved.
	}
	dartAssert(slices.Contains(state.members, member), nil)
	switch disposition {
	case GestureDispositionAccepted:
		if foundation.KDebugMode && DebugPrintGestureArenaDiagnostics {
			m.debugLogDiagnostic(pointer, "Accepting: "+dart.ToString(member), nil)
		}
		if state.isOpen {
			if state.eagerWinner == nil {
				state.eagerWinner = member
			}
		} else {
			if foundation.KDebugMode && DebugPrintGestureArenaDiagnostics {
				m.debugLogDiagnostic(pointer, "Self-declared winner: "+dart.ToString(member), nil)
			}
			m.resolveInFavorOf(pointer, state, member)
		}
	case GestureDispositionRejected:
		if foundation.KDebugMode && DebugPrintGestureArenaDiagnostics {
			m.debugLogDiagnostic(pointer, "Rejecting: "+dart.ToString(member), nil)
		}
		if state.eagerWinner == member {
			state.eagerWinner = nil
		}
		if i := slices.Index(state.members, member); i >= 0 {
			state.members = slices.Delete(state.members, i, i+1)
		}
		member.RejectGesture(pointer)
		if !state.isOpen {
			m.tryToResolveArena(pointer, state)
		}
	}
}

func (m *GestureArenaManager) tryToResolveArena(pointer int, state *gestureArena) {
	dartAssert(m.arenas[pointer] == state, nil)
	dartAssert(!state.isOpen, nil)
	if len(state.members) == 1 {
		async.ScheduleMicrotask(func() { m.resolveByDefault(pointer, state) })
	} else if len(state.members) == 0 {
		delete(m.arenas, pointer)
		m.debugLogDiagnostic(pointer, "Arena empty.", nil)
	} else if state.eagerWinner != nil {
		if foundation.KDebugMode && DebugPrintGestureArenaDiagnostics {
			m.debugLogDiagnostic(pointer, "Eager winner: "+dart.ToString(state.eagerWinner), nil)
		}
		m.resolveInFavorOf(pointer, state, state.eagerWinner)
	}
}

func (m *GestureArenaManager) resolveByDefault(pointer int, state *gestureArena) {
	if _, ok := m.arenas[pointer]; !ok {
		return // This arena has already resolved.
	}
	dartAssert(m.arenas[pointer] == state, nil)
	dartAssert(!state.isOpen, nil)
	dartAssert(len(state.members) == 1, nil)
	delete(m.arenas, pointer)
	if foundation.KDebugMode && DebugPrintGestureArenaDiagnostics {
		m.debugLogDiagnostic(pointer, "Default winner: "+dart.ToString(state.members[0]), nil)
	}
	state.members[0].AcceptGesture(pointer)
}

func (m *GestureArenaManager) resolveInFavorOf(pointer int, state *gestureArena, member GestureArenaMember) {
	dartAssert(state == m.arenas[pointer], nil)
	dartAssert(state.eagerWinner == nil || state.eagerWinner == member, nil)
	dartAssert(!state.isOpen, nil)
	delete(m.arenas, pointer)
	for _, rejectedMember := range state.members {
		if rejectedMember != member {
			rejectedMember.RejectGesture(pointer)
		}
	}
	member.AcceptGesture(pointer)
}

// debugLogDiagnostic is _debugLogDiagnostic (debug builds, when
// DebugPrintGestureArenaDiagnostics is set).
func (m *GestureArenaManager) debugLogDiagnostic(pointer int, message string, state *gestureArena) {
	if !foundation.KDebugMode || !DebugPrintGestureArenaDiagnostics {
		return
	}
	suffix := ""
	if state != nil {
		count := len(state.members)
		s := ""
		if count != 1 {
			s = "s"
		}
		suffix = " with " + strconv.Itoa(count) + " member" + s + "."
	}
	p := strconv.Itoa(pointer)
	for len(p) < 4 {
		p += " "
	}
	foundation.DebugPrint("Gesture arena " + p + " ❙ " + message + suffix)
}
