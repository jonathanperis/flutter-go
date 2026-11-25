package foundation

// Port of serialization.dart. dart:typed_data maps to Go as: ByteData →
// []byte, Uint8List → []byte, Int32List → []int32, Int64List → []int64,
// Float32List → []float32, Float64List → []float64, and Endian →
// encoding/binary's ByteOrder (omitted means the host's, as Endian.host).

import (
	"encoding/binary"
	"math"
)

func endianOf(e []binary.ByteOrder) binary.ByteOrder {
	if len(e) > 0 && e[0] != nil {
		return e[0]
	}
	return binary.NativeEndian
}

// WriteBuffer builds a byte buffer, with the alignment rules Flutter's
// StandardMessageCodec relies on (8-byte doubles, 4/8-byte typed lists).
type WriteBuffer struct {
	buffer      []byte
	currentSize int
	isDone      bool
	eightBytes  [8]byte
}

// WriteBufferOptions are WriteBuffer's named arguments.
type WriteBufferOptions struct{ StartCapacity int } // default 8

func NewWriteBuffer(opts ...WriteBufferOptions) *WriteBuffer {
	capacity := 8
	if len(opts) > 0 {
		capacity = opts[0].StartCapacity
	}
	if capacity <= 0 {
		panic("foundation: WriteBuffer startCapacity must be > 0")
	}
	return &WriteBuffer{buffer: make([]byte, capacity)}
}

func (w *WriteBuffer) checkNotDone() {
	if w.isDone {
		panic("foundation: WriteBuffer used after done()")
	}
}

func (w *WriteBuffer) add(b byte) {
	if w.currentSize == len(w.buffer) {
		w.resize(0)
	}
	w.buffer[w.currentSize] = b
	w.currentSize++
}

func (w *WriteBuffer) append(other []byte) {
	newSize := w.currentSize + len(other)
	if newSize >= len(w.buffer) {
		w.resize(newSize)
	}
	copy(w.buffer[w.currentSize:newSize], other)
	w.currentSize += len(other)
}

func (w *WriteBuffer) addAll(data []byte, start, end int) {
	newSize := w.currentSize + (end - start)
	if newSize >= len(w.buffer) {
		w.resize(newSize)
	}
	copy(w.buffer[w.currentSize:newSize], data[start:end])
	w.currentSize = newSize
}

func (w *WriteBuffer) resize(requiredLength int) {
	newLength := max(requiredLength, len(w.buffer)*2)
	nb := make([]byte, newLength)
	copy(nb, w.buffer)
	w.buffer = nb
}

func (w *WriteBuffer) PutUint8(b int) { w.checkNotDone(); w.add(byte(b)) }

func (w *WriteBuffer) PutUint16(value int, endian ...binary.ByteOrder) {
	w.checkNotDone()
	endianOf(endian).PutUint16(w.eightBytes[:], uint16(value))
	w.addAll(w.eightBytes[:], 0, 2)
}

func (w *WriteBuffer) PutUint32(value int, endian ...binary.ByteOrder) {
	w.checkNotDone()
	endianOf(endian).PutUint32(w.eightBytes[:], uint32(value))
	w.addAll(w.eightBytes[:], 0, 4)
}

func (w *WriteBuffer) PutInt32(value int, endian ...binary.ByteOrder) {
	w.checkNotDone()
	endianOf(endian).PutUint32(w.eightBytes[:], uint32(int32(value)))
	w.addAll(w.eightBytes[:], 0, 4)
}

func (w *WriteBuffer) PutInt64(value int, endian ...binary.ByteOrder) {
	w.checkNotDone()
	endianOf(endian).PutUint64(w.eightBytes[:], uint64(int64(value)))
	w.addAll(w.eightBytes[:], 0, 8)
}

func (w *WriteBuffer) PutFloat64(value float64, endian ...binary.ByteOrder) {
	w.checkNotDone()
	w.alignTo(8)
	endianOf(endian).PutUint64(w.eightBytes[:], math.Float64bits(value))
	w.addAll(w.eightBytes[:], 0, 8)
}

func (w *WriteBuffer) PutUint8List(list []byte) { w.checkNotDone(); w.append(list) }

func (w *WriteBuffer) PutInt32List(list []int32) {
	w.checkNotDone()
	w.alignTo(4)
	b := make([]byte, 4*len(list))
	for i, v := range list {
		binary.NativeEndian.PutUint32(b[4*i:], uint32(v))
	}
	w.append(b)
}

func (w *WriteBuffer) PutInt64List(list []int64) {
	w.checkNotDone()
	w.alignTo(8)
	b := make([]byte, 8*len(list))
	for i, v := range list {
		binary.NativeEndian.PutUint64(b[8*i:], uint64(v))
	}
	w.append(b)
}

func (w *WriteBuffer) PutFloat32List(list []float32) {
	w.checkNotDone()
	w.alignTo(4)
	b := make([]byte, 4*len(list))
	for i, v := range list {
		binary.NativeEndian.PutUint32(b[4*i:], math.Float32bits(v))
	}
	w.append(b)
}

func (w *WriteBuffer) PutFloat64List(list []float64) {
	w.checkNotDone()
	w.alignTo(8)
	b := make([]byte, 8*len(list))
	for i, v := range list {
		binary.NativeEndian.PutUint64(b[8*i:], math.Float64bits(v))
	}
	w.append(b)
}

var zeroBuffer [8]byte

func (w *WriteBuffer) alignTo(alignment int) {
	if mod := w.currentSize % alignment; mod != 0 {
		w.addAll(zeroBuffer[:], 0, alignment-mod)
	}
}

// Done returns the bytes written. Calling it twice panics (Dart throws a
// StateError).
func (w *WriteBuffer) Done() []byte {
	if w.isDone {
		panic("foundation: done() must not be called more than once on the same WriteBuffer.")
	}
	result := w.buffer[:w.currentSize:w.currentSize]
	w.buffer = nil
	w.isDone = true
	return result
}

// ReadBuffer reads values back in the order and alignment WriteBuffer wrote
// them.
type ReadBuffer struct {
	Data     []byte
	position int
}

func NewReadBuffer(data []byte) *ReadBuffer { return &ReadBuffer{Data: data} }

func (r *ReadBuffer) HasRemaining() bool { return r.position < len(r.Data) }

func (r *ReadBuffer) GetUint8() int {
	v := r.Data[r.position]
	r.position++
	return int(v)
}

func (r *ReadBuffer) GetUint16(endian ...binary.ByteOrder) int {
	v := endianOf(endian).Uint16(r.Data[r.position:])
	r.position += 2
	return int(v)
}

func (r *ReadBuffer) GetUint32(endian ...binary.ByteOrder) int {
	v := endianOf(endian).Uint32(r.Data[r.position:])
	r.position += 4
	return int(v)
}

func (r *ReadBuffer) GetInt32(endian ...binary.ByteOrder) int {
	v := int32(endianOf(endian).Uint32(r.Data[r.position:]))
	r.position += 4
	return int(v)
}

func (r *ReadBuffer) GetInt64(endian ...binary.ByteOrder) int {
	v := int64(endianOf(endian).Uint64(r.Data[r.position:]))
	r.position += 8
	return int(v)
}

func (r *ReadBuffer) GetFloat64(endian ...binary.ByteOrder) float64 {
	r.alignTo(8)
	v := math.Float64frombits(endianOf(endian).Uint64(r.Data[r.position:]))
	r.position += 8
	return v
}

// GetUint8List returns a view of the next length bytes (no copy, as Dart's
// asUint8List).
func (r *ReadBuffer) GetUint8List(length int) []byte {
	v := r.Data[r.position : r.position+length : r.position+length]
	r.position += length
	return v
}

func (r *ReadBuffer) GetInt32List(length int) []int32 {
	r.alignTo(4)
	out := make([]int32, length)
	for i := range out {
		out[i] = int32(binary.NativeEndian.Uint32(r.Data[r.position+4*i:]))
	}
	r.position += 4 * length
	return out
}

func (r *ReadBuffer) GetInt64List(length int) []int64 {
	r.alignTo(8)
	out := make([]int64, length)
	for i := range out {
		out[i] = int64(binary.NativeEndian.Uint64(r.Data[r.position+8*i:]))
	}
	r.position += 8 * length
	return out
}

func (r *ReadBuffer) GetFloat32List(length int) []float32 {
	r.alignTo(4)
	out := make([]float32, length)
	for i := range out {
		out[i] = math.Float32frombits(binary.NativeEndian.Uint32(r.Data[r.position+4*i:]))
	}
	r.position += 4 * length
	return out
}

func (r *ReadBuffer) GetFloat64List(length int) []float64 {
	r.alignTo(8)
	out := make([]float64, length)
	for i := range out {
		out[i] = math.Float64frombits(binary.NativeEndian.Uint64(r.Data[r.position+8*i:]))
	}
	r.position += 8 * length
	return out
}

func (r *ReadBuffer) alignTo(alignment int) {
	if mod := r.position % alignment; mod != 0 {
		r.position += alignment - mod
	}
}
