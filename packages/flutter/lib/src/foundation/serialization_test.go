package foundation

// Port of packages/flutter/test/foundation/serialization_test.dart.

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"
)

func TestBufferRoundTrips(t *testing.T) {
	if n := len(NewWriteBuffer().Done()); n != 0 {
		t.Errorf("empty: %d bytes", n)
	}
	w := NewWriteBuffer()
	w.PutUint8(201)
	b := w.Done()
	if len(b) != 1 || NewReadBuffer(b).GetUint8() != 201 {
		t.Error("single byte")
	}
	for _, e := range []binary.ByteOrder{nil, binary.BigEndian} {
		w = NewWriteBuffer()
		w.PutInt32(-9, e)
		b = w.Done()
		if len(b) != 4 || NewReadBuffer(b).GetInt32(e) != -9 {
			t.Errorf("int32 %v", e)
		}
		w = NewWriteBuffer()
		w.PutInt64(-9000000000000, e)
		b = w.Done()
		if len(b) != 8 || NewReadBuffer(b).GetInt64(e) != -9000000000000 {
			t.Errorf("int64 %v", e)
		}
		w = NewWriteBuffer()
		w.PutFloat64(3.14, e)
		b = w.Done()
		if len(b) != 8 || NewReadBuffer(b).GetFloat64(e) != 3.14 {
			t.Errorf("float64 %v", e)
		}
	}
}

func TestBufferUnalignedLists(t *testing.T) {
	w := NewWriteBuffer()
	w.PutUint8(9)
	w.PutInt32List([]int32{-99, 2, 99})
	b := w.Done()
	r := NewReadBuffer(b)
	r.GetUint8()
	if len(b) != 16 || !slices.Equal(r.GetInt32List(3), []int32{-99, 2, 99}) {
		t.Error("int32 list")
	}
	w = NewWriteBuffer()
	w.PutUint8(9)
	w.PutInt64List([]int64{-99, 2, 99})
	b = w.Done()
	r = NewReadBuffer(b)
	r.GetUint8()
	if len(b) != 32 || !slices.Equal(r.GetInt64List(3), []int64{-99, 2, 99}) {
		t.Error("int64 list")
	}
	w = NewWriteBuffer()
	w.PutUint8(9)
	w.PutFloat32List([]float32{3.14, float32(math.NaN())})
	b = w.Done()
	r = NewReadBuffer(b)
	r.GetUint8()
	f := r.GetFloat32List(2)
	if len(b) != 12 || math.Abs(float64(f[0])-3.14) > 0.0001 || !math.IsNaN(float64(f[1])) {
		t.Error("float32 list")
	}
	w = NewWriteBuffer()
	w.PutUint8(9)
	w.PutFloat64List([]float64{3.14, math.NaN()})
	b = w.Done()
	r = NewReadBuffer(b)
	r.GetUint8()
	d := r.GetFloat64List(2)
	if len(b) != 24 || d[0] != 3.14 || !math.IsNaN(d[1]) {
		t.Error("float64 list")
	}
}

func TestBufferMisuse(t *testing.T) {
	mustPanic := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: want panic", name)
			}
		}()
		f()
	}
	w := NewWriteBuffer()
	w.Done()
	mustPanic("done twice", func() { w.Done() })
	mustPanic("startCapacity 0", func() { NewWriteBuffer(WriteBufferOptions{StartCapacity: 0}) })
	NewWriteBuffer(WriteBufferOptions{StartCapacity: 1}) // size 1 is fine
}
