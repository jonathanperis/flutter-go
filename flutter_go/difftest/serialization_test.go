package difftest

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"slices"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// Same sequence as flutter/test/serialization_test.dart.
func TestWriteBufferMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/serialization.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Bytes []byte }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	w := foundation.NewWriteBuffer(foundation.WriteBufferOptions{StartCapacity: 1})
	w.PutUint8(7)
	w.PutFloat64(1.5)
	w.PutUint16(0xBEEF)
	w.PutUint16(0xBEEF, binary.BigEndian)
	w.PutInt32(-2)
	w.PutUint32(0xDEADBEEF, binary.BigEndian)
	w.PutUint8(1)
	w.PutInt64List([]int64{1, -1})
	w.PutUint8(2)
	w.PutFloat32List([]float32{0.1})
	w.PutUint8List([]byte{9, 8, 7})
	w.PutFloat64(math.Copysign(0, -1), binary.BigEndian)
	w.PutInt64(1 << 40)
	w.PutInt32List([]int32{5})
	w.PutFloat64List([]float64{math.Inf(1)})
	if got := w.Done(); !slices.Equal(got, ref.Bytes) {
		t.Errorf("bytes differ:\n Go:      %v\n Flutter: %v", got, ref.Bytes)
	}

	// Read Flutter's bytes back with ReadBuffer, in the same order.
	r := foundation.NewReadBuffer(ref.Bytes)
	be := binary.BigEndian
	ok := r.GetUint8() == 7 && r.GetFloat64() == 1.5 && r.GetUint16() == 0xBEEF && r.GetUint16(be) == 0xBEEF &&
		r.GetInt32() == -2 && r.GetUint32(be) == 0xDEADBEEF && r.GetUint8() == 1 &&
		slices.Equal(r.GetInt64List(2), []int64{1, -1}) && r.GetUint8() == 2 &&
		slices.Equal(r.GetFloat32List(1), []float32{0.1}) && slices.Equal(r.GetUint8List(3), []byte{9, 8, 7})
	negZero := r.GetFloat64(be)
	ok = ok && negZero == 0 && math.Signbit(negZero) && r.GetInt64() == 1<<40 &&
		slices.Equal(r.GetInt32List(1), []int32{5}) && math.IsInf(r.GetFloat64List(1)[0], 1) && !r.HasRemaining()
	if !ok {
		t.Error("ReadBuffer doesn't read Flutter's bytes back")
	}
}
