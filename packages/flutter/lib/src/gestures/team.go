package gestures

// Port of team.dart (Flutter 3.50.1).

import "slices"

// combiningGestureArenaEntry is _CombiningGestureArenaEntry.
type combiningGestureArenaEntry struct {
	combiner *combiningGestureArenaMember
	member   GestureArenaMember
}

func (e *combiningGestureArenaEntry) Resolve(disposition GestureDisposition) {
	e.combiner.resolve(e.member, disposition)
}

// combiningGestureArenaMember is _CombiningGestureArenaMember: the team's
// one member in a pointer's arena.
type combiningGestureArenaMember struct {
	owner    *GestureArenaTeam
	members  []GestureArenaMember
	pointer  int
	resolved bool
	winner   GestureArenaMember
	entry    GestureArenaEntry
}

// String is Object.toString of the private Dart class (the arena
// diagnostics print it).
func (c *combiningGestureArenaMember) String() string {
	return "Instance of '_CombiningGestureArenaMember'"
}

func (c *combiningGestureArenaMember) AcceptGesture(pointer int) {
	dartAssert(c.pointer == pointer, nil)
	dartAssert(c.winner != nil || len(c.members) > 0, nil)
	c.close()
	if c.winner == nil {
		if c.owner.Captain != nil {
			c.winner = c.owner.Captain
		} else {
			c.winner = c.members[0]
		}
	}
	for _, member := range c.members {
		if member != c.winner {
			member.RejectGesture(pointer)
		}
	}
	c.winner.AcceptGesture(pointer)
}

func (c *combiningGestureArenaMember) RejectGesture(pointer int) {
	dartAssert(c.pointer == pointer, nil)
	c.close()
	for _, member := range c.members {
		member.RejectGesture(pointer)
	}
}

func (c *combiningGestureArenaMember) close() {
	dartAssert(!c.resolved, nil)
	c.resolved = true
	combiner := c.owner.combiners[c.pointer]
	delete(c.owner.combiners, c.pointer)
	dartAssert(combiner == c, nil)
}

func (c *combiningGestureArenaMember) add(pointer int, member GestureArenaMember) GestureArenaEntry {
	dartAssert(!c.resolved, nil)
	dartAssert(c.pointer == pointer, nil)
	c.members = append(c.members, member)
	if c.entry == nil {
		c.entry = GestureBindingInstance().GestureArena().Add(pointer, c)
	}
	return &combiningGestureArenaEntry{combiner: c, member: member}
}

func (c *combiningGestureArenaMember) resolve(member GestureArenaMember, disposition GestureDisposition) {
	if c.resolved {
		return
	}
	switch disposition {
	case GestureDispositionAccepted:
		if c.winner == nil {
			if c.owner.Captain != nil {
				c.winner = c.owner.Captain
			} else {
				c.winner = member
			}
		}
		c.entry.Resolve(disposition)
	case GestureDispositionRejected:
		if i := slices.Index(c.members, member); i >= 0 {
			c.members = slices.Delete(c.members, i, i+1)
		}
		member.RejectGesture(c.pointer)
		if len(c.members) == 0 {
			c.entry.Resolve(disposition)
		}
	}
}

// GestureArenaTeam is a group of GestureArenaMembers that compete in the
// arena as one: when one member accepts, the team wins (and Captain, if
// set, or else that member, gets the gesture).
type GestureArenaTeam struct {
	combiners map[int]*combiningGestureArenaMember

	// Captain, if not nil, wins the gesture whenever the team does.
	Captain GestureArenaMember
}

// NewGestureArenaTeam is GestureArenaTeam().
func NewGestureArenaTeam() *GestureArenaTeam {
	return &GestureArenaTeam{combiners: map[int]*combiningGestureArenaMember{}}
}

// Add adds member to the arena of pointer as part of the team.
func (t *GestureArenaTeam) Add(pointer int, member GestureArenaMember) GestureArenaEntry {
	if t.combiners == nil {
		t.combiners = map[int]*combiningGestureArenaMember{}
	}
	combiner, ok := t.combiners[pointer]
	if !ok {
		combiner = &combiningGestureArenaMember{owner: t, pointer: pointer}
		t.combiners[pointer] = combiner
	}
	return combiner.add(pointer, member)
}
