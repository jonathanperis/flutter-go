#!/usr/bin/env python3
# Generates recognizers_tap_and_drag_scenes.json (run: python3 recognizers_tap_and_drag_scenes.py recognizers_tap_and_drag_scenes.json), shared by
# the Dart and Go sides of the recognizers_tap_and_drag difftest.
import json, sys

scenes = []
nextptr = [1]


class S:
    def __init__(self, name):
        self.name = name
        self.ops = []
        self.ptrs = {}
        scenes.append({"name": name, "ops": self.ops})

    def new(self, id, type, **kw):
        op = {"op": "new", "id": id, "type": type}
        op.update(kw)
        self.ops.append(op)
        return self

    def ptr(self, p, kind="touch", buttons=1):
        self.kinds = getattr(self, "kinds", {})
        self.kinds[p] = (kind, buttons)
        self.used = getattr(self, "used", set())
        self.used.discard(p)

    def _fresh(self, p):
        kind, buttons = self.kinds[p]
        n = nextptr[0]
        nextptr[0] += 1
        self.ptrs[p] = n
        self.ops.append({"op": "ptr", "p": p, "pointer": n, "kind": kind, "buttons": buttons})

    def down(self, p, x, y, add, ts=0, m=None, buttons=None):
        self._fresh(p)
        op = {"op": "down", "p": p, "x": x, "y": y, "ts": ts, "add": add}
        if m is not None:
            op["m"] = m
        if buttons is not None:
            op["buttons"] = buttons
        self.ops.append(op)

    def rdown(self, p):
        self.ops.append({"op": "routedown", "p": p})

    def close(self, p):
        self.ops.append({"op": "close", "pointer": self.ptrs[p]})

    def sweep(self, p):
        self.ops.append({"op": "sweep", "pointer": self.ptrs[p]})

    def move(self, p, x, y, ts=0, buttons=None):
        op = {"op": "move", "p": p, "x": x, "y": y, "ts": ts}
        if buttons is not None:
            op["buttons"] = buttons
        self.ops.append(op)

    def up(self, p, ts=0):
        self.ops.append({"op": "up", "p": p, "ts": ts})

    def cancel(self, p, ts=0):
        self.ops.append({"op": "cancel", "p": p, "ts": ts})

    def elapse(self, ms):
        self.ops.append({"op": "elapse", "ms": ms})

    def flush(self):
        self.ops.append({"op": "flush"})

    def state(self, id):
        self.ops.append({"op": "state", "id": id})

    def resign(self, id):
        self.ops.append({"op": "resign", "id": id})

    def dispose(self, id):
        self.ops.append({"op": "dispose", "id": id})

    # helpers
    def tap(self, p, x, y, add, ts=0, upts=None, sweep=True, moves=()):
        self.down(p, x, y, add, ts=ts)
        self.close(p)
        self.rdown(p)
        for (mx, my) in moves:
            self.move(p, mx, my, ts=ts)
        self.up(p, ts=upts if upts is not None else ts)
        if sweep:
            self.sweep(p)


def tadtypes():
    return ["hd", "pan", "tad"]


# 1. Consecutive taps, every type, both start behaviors.
for t in tadtypes():
    for dsb in ["down", "start"]:
        s = S(f"consecutive taps {t} {dsb}")
        s.new("r", t, dsb=dsb, max=3)
        for i, (x, y) in enumerate([(10, 10), (12, 12), (13, 11), (12.5, 12.25), (11, 9)]):
            s.ptr(f"p{i}")
            s.tap(f"p{i}", x, y, ["r"], ts=i * 100)
            s.state("r")
            s.elapse(150)
        s.dispose("r")

# 2. Timeouts between taps (just under, at, over kDoubleTapTimeout), far taps.
for gap in [299, 300, 301, 1000]:
    s = S(f"tap timeout gap {gap}")
    s.new("r", "pan", dsb="down", max=5)
    s.ptr("a")
    s.tap("a", 10, 10, ["r"])
    s.elapse(gap)
    s.ptr("b")
    s.tap("b", 11, 11, ["r"])
    s.state("r")
    s.dispose("r")
for dist in [99.0, 100.0, 100.5]:
    s = S(f"tap distance {dist}")
    s.new("r", "hd", max=None)
    s.ptr("a")
    s.tap("a", 0, 0, ["r"])
    s.elapse(100)
    s.ptr("b")
    s.tap("b", dist, 0, ["r"])
    s.state("r")
    s.dispose("r")

# 3. Tap then drag, drag after slop; deltas; several moves; throttle.
for t in tadtypes():
    for dsb in ["down", "start"]:
        for moves in [[(40, 45)], [(15, 10), (30, 12), (60, 20), (90.5, 33.25)], [(10, 25), (10, 60)], [(25, 25)]]:
            s = S(f"drag {t} {dsb} {moves}")
            s.new("r", t, dsb=dsb)
            s.ptr("a")
            s.down("a", 10, 10, ["r"], ts=5)
            s.close("a")
            s.rdown("a")
            for i, (mx, my) in enumerate(moves):
                s.move("a", mx, my, ts=10 + 16 * i)
            s.up("a", ts=200)
            s.sweep("a")
            s.dispose("r")

# 4. Consecutive taps then drag.
for t in ["hd", "pan"]:
    s = S(f"taps then drag {t}")
    s.new("r", t, dsb="down", max=3)
    s.ptr("a")
    s.tap("a", 10, 10, ["r"])
    s.elapse(150)
    s.tap("a", 10, 10, ["r"])
    s.elapse(150)
    s.down("a", 10, 10, ["r"])
    s.close("a")
    s.rdown("a")
    s.move("a", 40, 45)
    s.move("a", 50, 55)
    s.up("a")
    s.state("r")
    s.dispose("r")

# 5. Throttled updates.
for throttle in [16, 50]:
    s = S(f"throttle {throttle}")
    s.new("r", "pan", dsb="down", throttle=throttle)
    s.ptr("a")
    s.down("a", 0, 0, ["r"])
    s.close("a")
    s.rdown("a")
    for i in range(6):
        s.move("a", 10 * (i + 1), 5 * (i + 1), ts=10 * i)
        s.elapse(10)
    s.up("a")
    s.elapse(100)
    s.dispose("r")

# 6. Competition.
competitors = ["tap", "hdrag", "vdrag", "panr", "scale", "lp", "dtap", "eager"]
for t in ["hd", "pan"]:
    for comp in competitors:
        for eager in [True, False]:
            for order in ["first", "second"]:
                for moves in [[], [(25, 25)], [(45, 10)], [(10, 45), (10, 100)], [(40, 45)]]:
                    s = S(f"compete {t} vs {comp} eager={eager} {order} {moves}")
                    s.new("r", t, dsb="down", eager=eager, max=3)
                    s.new("c", comp)
                    add = ["r", "c"] if order == "first" else ["c", "r"]
                    s.ptr("a")
                    s.down("a", 10, 10, add)
                    s.close("a")
                    s.rdown("a")
                    for (mx, my) in moves:
                        s.move("a", mx, my)
                    s.up("a")
                    s.sweep("a")
                    s.elapse(600)
                    s.dispose("r")
                    s.dispose("c")

# 7. Beats long press on a double tap with a delay; double tap vs dtap.
for comp in ["lp", "dtap", "tap"]:
    s = S(f"second tap beats {comp}")
    s.new("r", "pan", dsb="down", max=3)
    s.new("c", comp)
    s.ptr("a")
    s.tap("a", 10, 10, ["r", "c"])
    s.elapse(150)
    s.down("a", 10, 10, ["r", "c"])
    s.close("a")
    s.rdown("a")
    s.elapse(500)
    s.move("a", 40, 45)
    s.up("a")
    s.sweep("a")
    s.elapse(600)
    s.dispose("r")
    s.dispose("c")

# 8. Mouse and other kinds: slops.
for kind in ["mouse", "stylus", "unknown", "invertedStylus"]:
    for t in ["hd", "pan"]:
        for mv in [(12, 10), (13.5, 10), (15, 15), (10, 15)]:
            s = S(f"kind {kind} {t} {mv}")
            s.new("r", t, dsb="down")
            s.new("c", "tap")
            s.ptr("a", kind=kind)
            s.down("a", 10, 10, ["r", "c"])
            s.close("a")
            s.rdown("a")
            s.move("a", mv[0], mv[1])
            s.up("a")
            s.sweep("a")
            s.dispose("r")
            s.dispose("c")

# 9. Buttons: secondary, buttons change, a second pointer.
s = S("secondary button not allowed")
s.new("r", "pan", dsb="down")
s.new("c", "tap")
s.ptr("a", buttons=2)
s.down("a", 10, 10, ["r", "c"])
s.close("a")
s.rdown("a")
s.up("a")
s.sweep("a")
s.dispose("r")
s.dispose("c")

s = S("secondary pointer after primary")
s.new("r", "pan", dsb="down")
s.ptr("a")
s.ptr("b", buttons=2)
s.down("a", 10, 10, ["r"])
s.close("a")
s.rdown("a")
s.down("b", 20, 20, ["r"])
s.close("b")
s.rdown("b")
s.move("a", 40, 40)
s.up("b")
s.sweep("b")
s.up("a")
s.sweep("a")
s.dispose("r")

for order in ["fifo", "filo"]:
    for accepted in [False, True]:
        s = S(f"two pointers {order} accepted={accepted}")
        s.new("r", "pan", dsb="down")
        s.ptr("a")
        s.ptr("b")
        s.down("a", 10, 10, ["r"])
        if not accepted:
            s.down("b", 12, 12, ["r"])
        s.close("a")
        s.rdown("a")
        if accepted:
            s.down("b", 12, 12, ["r"])
        s.close("b")
        s.rdown("b")
        first, second = ("a", "b") if order == "fifo" else ("b", "a")
        s.up(first)
        s.sweep(first)
        s.up(second)
        s.sweep(second)
        s.dispose("r")

# 10. Cancel at every stage.
for stage in ["down", "afterdeadline", "dragging", "possible"]:
    s = S(f"cancel {stage}")
    s.new("r", "pan", dsb="down")
    s.new("c", "tap")
    s.ptr("a")
    s.down("a", 10, 10, ["r", "c"])
    s.close("a")
    s.rdown("a")
    if stage == "afterdeadline":
        s.elapse(150)
    if stage == "dragging":
        s.move("a", 50, 50)
    if stage == "possible":
        s.move("a", 22, 22)
    s.cancel("a")
    s.sweep("a")
    s.elapse(400)
    s.ptr("b")
    s.tap("b", 10, 10, ["r", "c"])
    s.state("r")
    s.dispose("r")
    s.dispose("c")

# 11. Deadline: down held past kPressTimeout with a competitor, then up.
for comp in ["tap", "hdrag"]:
    for hold in [99, 100, 101, 300]:
        s = S(f"deadline {comp} {hold}")
        s.new("r", "pan", dsb="down")
        s.new("c", comp)
        s.ptr("a")
        s.down("a", 10, 10, ["r", "c"])
        s.close("a")
        s.rdown("a")
        s.elapse(hold)
        s.up("a")
        s.sweep("a")
        s.dispose("r")
        s.dispose("c")

# 12. Gesture settings slop, supported devices.
for slop in [5.0, 30.0]:
    s = S(f"touch slop {slop}")
    s.new("r", "hd", dsb="down", slop=slop)
    s.new("c", "tap")
    s.ptr("a")
    s.down("a", 10, 10, ["r", "c"])
    s.close("a")
    s.rdown("a")
    s.move("a", 18, 10)
    s.move("a", 50, 10)
    s.up("a")
    s.sweep("a")
    s.dispose("r")
    s.dispose("c")
s = S("supported devices mouse only")
s.new("r", "pan", dsb="down", devices=["mouse"])
s.ptr("a")
s.down("a", 10, 10, ["r"])
s.close("a")
s.rdown("a")
s.up("a")
s.sweep("a")
s.ptr("m", kind="mouse")
s.tap("m", 10, 10, ["r"])
s.dispose("r")

# 13. Transforms.
mats = [
    [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, -100, -50, 0, 1],
    [2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1],
    [0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1, 0, 5, 7, 0, 1],
    [0.5, 0.25, 0, 0, -0.25, 0.5, 0, 0, 0, 0, 1, 0, 3, -4, 0, 1],
]
for i, m in enumerate(mats):
    for t in ["hd", "pan"]:
        for dsb in ["down", "start"]:
            s = S(f"transform {i} {t} {dsb}")
            s.new("r", t, dsb=dsb)
            s.ptr("a")
            s.down("a", 10, 10, ["r"], m=m)
            s.close("a")
            s.rdown("a")
            s.move("a", 30, 25)
            s.move("a", 60, 33)
            s.up("a")
            s.sweep("a")
            s.dispose("r")

# 14. maxConsecutiveTap values; dispose mid-gesture.
for mx in [1, 2]:
    s = S(f"max consecutive {mx}")
    s.new("r", "pan", dsb="down", max=mx)
    s.ptr("a")
    for i in range(4):
        s.tap("a", 10, 10, ["r"])
        s.state("r")
        s.elapse(50)
    s.dispose("r")
s = S("dispose while dragging")
s.new("r", "pan", dsb="down")
s.ptr("a")
s.down("a", 10, 10, ["r"])
s.close("a")
s.rdown("a")
s.move("a", 50, 50)
s.dispose("r")
s.up("a")
s.sweep("a")

# 15. Mutation-driven scenes.
# A move past the hit slop (not the pan slop) ends the tap series.
for back in [False, True]:
    s = S(f"slop move ends series back={back}")
    s.new("r", "pan", dsb="down", max=5)
    s.ptr("a")
    s.down("a", 10, 10, ["r"])
    s.close("a")
    s.rdown("a")
    s.move("a", 28.5, 10)
    if back:
        s.move("a", 10, 10)
    s.up("a")
    s.sweep("a")
    s.elapse(50)
    s.tap("a", 10, 10, ["r"])
    s.state("r")
    s.dispose("r")
# Past the slop and back without reaching the drag distance: still a drag.
for t in ["hd", "pan"]:
    s = S(f"slop and back {t}")
    s.new("r", t, dsb="down")
    s.ptr("a")
    s.down("a", 10, 10, ["r"])
    s.close("a")
    s.rdown("a")
    if t == "hd":
        s.move("a", 10, 28.5)
        s.move("a", 10, 19)
    else:
        s.move("a", 28.5, 10)
        s.move("a", 19, 10)
    s.up("a")
    s.sweep("a")
    s.dispose("r")
# A recognizer without callbacks doesn't take part.
s = S("no callbacks")
s.new("r", "tadnone")
s.new("c", "tap")
s.ptr("a")
s.tap("a", 10, 10, ["r", "c"])
s.dispose("r")
s.dispose("c")
# A secondary pointer before the primary pointer won.
s = S("secondary pointer before win")
s.new("r", "pan", dsb="down")
s.new("c", "tap")
s.ptr("a")
s.ptr("b", buttons=2)
s.down("a", 10, 10, ["r", "c"])
s.close("a")
s.rdown("a")
s.down("b", 20, 20, ["r"])
s.close("b")
s.rdown("b")
s.up("b")
s.sweep("b")
s.up("a")
s.sweep("a")
s.dispose("r")
s.dispose("c")
# Moves before the arena closes (eager victory while open).
for t in ["hd", "pan"]:
    for comp in ["tap", "passive", "hdrag"]:
        s = S(f"move before close {t} {comp}")
        s.new("r", t, dsb="down")
        s.new("c", comp)
        s.ptr("a")
        s.down("a", 10, 10, ["r", "c"])
        s.rdown("a")
        s.move("a", 40, 45)
        s.close("a")
        s.move("a", 50, 55)
        s.up("a")
        s.sweep("a")
        s.dispose("r")
        s.dispose("c")
# Without eager victory, against a member that resigns later: the drag
# starts at the first move past the slop.
for t in ["hd", "pan"]:
    for dsb in ["down", "start"]:
        s = S(f"late win {t} {dsb}")
        s.new("r", t, dsb=dsb, eager=False)
        s.new("c", "passive")
        s.ptr("a")
        s.down("a", 10, 10, ["r", "c"])
        s.close("a")
        s.rdown("a")
        s.move("a", 40, 12, ts=10)
        s.move("a", 60, 14, ts=20)
        s.move("a", 80, 20, ts=30)
        s.resign("c")
        s.move("a", 90, 25, ts=40)
        s.up("a", ts=50)
        s.sweep("a")
        s.dispose("r")
        s.dispose("c")
# Default drag start behavior (start), eagerVictoryOnDrag (true).
for t in ["hd", "pan", "tad"]:
    s = S(f"defaults {t}")
    s.new("r", t)
    s.new("c", "tap")
    s.ptr("a")
    s.down("a", 10, 10, ["r", "c"])
    s.close("a")
    s.rdown("a")
    s.move("a", 40, 45)
    s.move("a", 50, 46)
    s.up("a")
    s.sweep("a")
    s.dispose("r")
    s.dispose("c")

json.dump(scenes, open(sys.argv[1], "w"), indent=0)
print(len(scenes), "scenes", sum(len(s["ops"]) for s in scenes), "ops")
