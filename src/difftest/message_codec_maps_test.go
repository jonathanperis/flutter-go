package difftest

// Go twin of flutter/test/message_codec_maps_test.dart: Dart's Maps keep
// their insertion order through the standard and JSON codecs (decoded maps
// are *dart.LinkedMap). The same maps, in the same pseudo-random orders,
// are encoded, decoded, changed and encoded again; bytes, iteration order
// and toString are compared with Flutter's, case by case.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// linkedOf is a *dart.LinkedMap of a Go map's entries in key order (for
// messages whose order doesn't matter to the code reading them).
func linkedOf(m map[string]any) *dart.LinkedMap {
	out := &dart.LinkedMap{}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		out.Set(k, m[k])
	}
	return out
}

// mapsRand is the Dart side's LCG on 31 bits.
type mapsRand struct{ state int }

func (r *mapsRand) next(n int) int {
	r.state = (r.state*1103515245 + 12345) & 0x7fffffff
	return r.state % n
}

// shuffled is the Dart side's Fisher-Yates shuffle of 0..n-1.
func (r *mapsRand) shuffled(n int) []int {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := r.next(i + 1)
		order[i], order[j] = order[j], order[i]
	}
	return order
}

func mapsAllKeys() []any {
	return []any{"b", "a", "", 3, -1, math.MaxInt64, 1.5, -0.25, true, false, nil,
		[]any{1, "x"}, []byte{9, 8}, []int32{7}, []float64{2.5},
		dart.NewLinkedMapOf("k", 1, "j", []any{})}
}

func mapsValueFor(i int) any {
	switch i % 5 {
	case 0:
		return i
	case 1:
		return "v" + strconv.Itoa(i)
	case 2:
		return float64(i) + 0.5
	case 3:
		return []any{i, nil}
	}
	return dart.NewLinkedMapOf("n"+strconv.Itoa(i), i, "m", nil)
}

func mapsRoundTrip(codec services.MessageCodec[any], message any) string {
	encoded := codec.EncodeMessage(message)
	decoded := codec.DecodeMessage(encoded)
	again := codec.EncodeMessage(decoded)
	return codecHex(encoded) + " -> " + codecDesc(decoded) + " | " + dart.ToString(decoded) + " | again " +
		strconv.FormatBool(codecHex(again) == codecHex(encoded))
}

func joinIndexes(order []int) string {
	parts := make([]string, len(order))
	for i, v := range order {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

func messageCodecMapsCases() []codecCase {
	c := &codecRecorder{}
	// Dart's record catches with 'throws <runtimeType>: <error>'; any throw is
	// a mismatch here.
	standard := services.NewStandardMessageCodec()
	standardMethod := services.NewStandardMethodCodec()
	jsonCodec := services.JSONMessageCodec{}
	jsonMethod := services.JSONMethodCodec{}
	jsonBytes := func(s string) []byte { return dart.UTF8Encode(s) }

	// ---- Every key type, in many insertion orders.
	r := &mapsRand{state: 20251126}
	for round := range 40 {
		keys := mapsAllKeys()
		count := 1 + r.next(len(keys))
		order := r.shuffled(len(keys))[:count]
		m := &dart.LinkedMap{}
		for _, i := range order {
			m.Set(keys[i], mapsValueFor(i))
		}
		c.record(fmt.Sprintf("standard order %d %s", round, joinIndexes(order)), func() string {
			return mapsRoundTrip(standard, m)
		})
	}

	// ---- All 24 orders of four keys of different types.
	four := []any{"z", 2, nil, 0.5}
	var permute func(prefix, rest []int)
	permute = func(prefix, rest []int) {
		if len(rest) == 0 {
			m, j := &dart.LinkedMap{}, &dart.LinkedMap{}
			for _, i := range prefix {
				m.Set(four[i], i)
				j.Set(dart.ToString(four[i]), i)
			}
			c.record("standard permutation "+joinIndexes(prefix), func() string { return mapsRoundTrip(standard, m) })
			c.record("json permutation "+joinIndexes(prefix), func() string {
				encoded := jsonCodec.EncodeMessage(j)
				decoded := jsonCodec.DecodeMessage(encoded)
				return string(encoded) + " -> " + codecDesc(decoded) + " | " + dart.ToString(decoded)
			})
			return
		}
		for i := range rest {
			permute(append(slices.Clone(prefix), rest[i]), append(slices.Clone(rest[:i]), rest[i+1:]...))
		}
	}
	permute(nil, []int{0, 1, 2, 3})

	// ---- Nested maps, each level in its own order.
	r = &mapsRand{state: 7}
	for round := range 12 {
		var build func(depth int) *dart.LinkedMap
		build = func(depth int) *dart.LinkedMap {
			order := r.shuffled(5)
			m := &dart.LinkedMap{}
			for _, i := range order[:2+r.next(4)] {
				key := []any{"k" + strconv.Itoa(i), i, float64(i) * 1.25, i%2 == 0, nil}[r.next(5)]
				if depth == 0 || r.next(3) == 0 {
					m.Set(key, mapsValueFor(i))
				} else {
					m.Set(key, build(depth-1))
				}
			}
			if r.next(2) == 0 {
				key := []any{depth}
				m.Set(key, []any{build(0), build(0)})
			}
			return m
		}
		m := build(2)
		c.record(fmt.Sprintf("standard nested %d", round), func() string { return mapsRoundTrip(standard, m) })
		c.record(fmt.Sprintf("standard nested %d method", round), func() string {
			call := standardMethod.EncodeMethodCall(services.NewMethodCall("m", m))
			decoded := standardMethod.DecodeMethodCall(call)
			envelope := standardMethod.DecodeEnvelope(standardMethod.EncodeSuccessEnvelope(m))
			return codecHex(call) + " -> " + codecDesc(decoded.Arguments) + " | " + decoded.String() + " | " + codecDesc(envelope)
		})
	}

	// ---- Keys the decoded map merges, in both orders, around other keys.
	for _, mc := range []struct {
		name string
		kv   []any
	}{
		{"int double", []any{"a", 0, 1, 1, "b", 2, 1.0, 3, "c", 4}},
		{"double int", []any{1.0, 0, "a", 1, 1, 2, "b", 3}},
		{"zeros", []any{math.Copysign(0, -1), 0, "x", 1, 0, 2, 0.0, 3}},
		{"strings", []any{"b", 0, "a", 1, "b", 2, "c", 3, "a", 4}},
		{"nulls", []any{nil, 0, "n", 1, nil, 2}},
		{"equal lists", []any{[]any{1}, 0, "l", 1, []any{1}, 2}},
		{"empty lists", []any{[]any{}, 0, []any{}, 1, []byte{}, 2, []byte{}, 3, []int32{}, 4, []int32{}, 5}},
		{"equal maps", []any{dart.NewLinkedMapOf("a", 1), 0, dart.NewLinkedMapOf("a", 1), 1}},
		{"nan", []any{dartNaN, 0, "x", 1, dartNaN, 2}},
		{"big ints", []any{math.MaxInt64, 0, 9223372036854775808.0, 1, 9007199254740993, 2, 9007199254740992.0, 3}},
	} {
		c.record("standard merging "+mc.name, func() string {
			buffer := foundation.NewWriteBuffer()
			buffer.PutUint8(13)
			standard.WriteSize(buffer, len(mc.kv)/2)
			for _, v := range mc.kv {
				standard.WriteValue(buffer, v)
			}
			b := buffer.Done()
			decoded := standard.DecodeMessage(b)
			return codecHex(b) + " -> " + codecDesc(decoded) + " | " + dart.ToString(decoded) + " | " +
				codecHex(standard.EncodeMessage(decoded))
		})
	}

	// ---- Changing a decoded map.
	c.record("standard mutate", func() string {
		m := standard.DecodeMessage(standard.EncodeMessage(dart.NewLinkedMapOf("a", 1, "b", 2, 3, "c", nil, "d", 1.5, "e"))).(*dart.LinkedMap)
		var log []string
		removedB, _ := m.Remove("b")
		log = append(log, "remove b "+dart.ToString(removedB))
		removedX, _ := m.Remove("x")
		log = append(log, "remove x "+dart.ToString(removedX))
		m.Set("b", 20)
		m.Set("a", 10)
		m.Set(3.0, "C")
		log = append(log, "putIfAbsent "+dart.ToString(m.PutIfAbsent("e", func() any { return 5 }))+" "+
			dart.ToString(m.PutIfAbsent("a", func() any { return 0 })))
		m.Remove(nil)
		m.Set(nil, "D")
		log = append(log, fmt.Sprintf("lookups %s %s %s %s %v %v", dart.ToString(m.Lookup(3)), dart.ToString(m.Lookup(3.0)),
			dart.ToString(m.Lookup(1.5)), dart.ToString(m.Lookup("zz")), m.ContainsKey(3.0), m.ContainsKey("b")))
		log = append(log, "keys "+dart.ToString(slices.Collect(m.Keys()))+" values "+dart.ToString(slices.Collect(m.Values()))+
			" length "+strconv.Itoa(m.Len()))
		return strings.Join(log, "; ") + " | " + m.String() + " | " + codecHex(standard.EncodeMessage(m))
	})
	c.record("standard mutate all", func() string {
		m := standard.DecodeMessage(standard.EncodeMessage(dart.NewLinkedMapOf("a", 1, "b", 2))).(*dart.LinkedMap)
		m.Remove("a")
		m.Remove("b")
		m.Set("b", 3)
		m.AddAll(dart.NewLinkedMapOf("c", 4, "b", 5, "a", 6))
		return m.String() + " | " + codecHex(standard.EncodeMessage(m))
	})

	// ---- JSON.
	for _, text := range []string{
		`{"z":1,"a":2,"m":3}`,
		`{"b":{"y":1,"x":[{"q":1,"p":2}]},"a":null}`,
		`{"a":1,"b":2,"a":3}`,
		`{"c":1,"b":2,"c":3,"a":4,"b":5}`,
		`[{"2":0,"10":1,"1":2},{}]`,
		`{"é":1,"e":2,"":3,"😂":4,"E":5}`,
	} {
		c.record("json decode "+text, func() string {
			decoded := jsonCodec.DecodeMessage(jsonBytes(text))
			return codecDesc(decoded) + " | " + dart.ToString(decoded) + " | " + string(jsonCodec.EncodeMessage(decoded))
		})
	}
	c.record("json method call", func() string {
		call := jsonMethod.DecodeMethodCall(jsonBytes(`{"args":{"z":1,"a":[{"d":4,"c":3}]},"method":"m"}`))
		return call.String() + " | " + string(jsonMethod.EncodeMethodCall(call))
	})
	c.record("json envelope", func() string {
		result := jsonMethod.DecodeEnvelope(jsonBytes(`[{"b":1,"a":{"d":1,"c":2}}]`))
		return codecDesc(result) + " | " + string(jsonMethod.EncodeSuccessEnvelope(result))
	})
	platformError := func(f func()) (out string) {
		defer func() {
			if e, ok := recover().(*services.PlatformException); ok {
				out = codecDesc(e.Details) + " | " + e.Error()
			}
		}()
		f()
		return "no error"
	}
	c.record("json error envelope", func() string {
		return platformError(func() { jsonMethod.DecodeEnvelope(jsonBytes(`["code","message",{"y":1,"x":2}]`)) })
	})
	c.record("standard error envelope", func() string {
		return platformError(func() {
			standardMethod.DecodeEnvelope(standardMethod.EncodeErrorEnvelope("c", services.EncodeErrorEnvelopeOptions{
				Details: dart.NewLinkedMapOf("y", 1, "x", dart.NewLinkedMapOf(2, 1, 1, 2))}))
		})
	})
	return c.cases
}

func TestMessageCodecMapsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/message_codec_maps.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Cases []codecCase }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	got := messageCodecMapsCases()
	if len(got) != len(ref.Cases) {
		t.Errorf("%d cases, Flutter has %d", len(got), len(ref.Cases))
	}
	failures := 0
	for i := range min(len(got), len(ref.Cases)) {
		g, w := got[i], ref.Cases[i]
		if g != w {
			t.Errorf("case %d:\n Go:      %s => %s\n Flutter: %s => %s", i, g.Name, g.Out, w.Name, w.Out)
			if failures++; failures > 8 {
				t.Fatal("too many failures")
			}
		}
	}
}
