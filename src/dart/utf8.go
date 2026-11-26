package dart

// dart:convert's UTF-8 codec as the Dart VM runs it (convert/utf.dart's
// _Utf8Decoder state machine and the VM patch's convertSingle/
// convertChunked, in the SDK shipped with Flutter 3.50.1).
//
// Go strings vs Dart strings: a Dart String is a sequence of UTF-16 code
// units and may hold unpaired surrogates; a Go string is bytes, normally
// UTF-8. flutter-go reads a Go string as WTF-8: well-formed UTF-8 is the
// text, a three-byte encoding of a surrogate (ED A0..BF 80..BF) is that
// lone surrogate code unit (what Dart's string would hold), and any other
// byte that isn't valid UTF-8 is one U+FFFD. Strings produced by the
// decoders are always valid UTF-8: a lone surrogate (from a JSON \u
// escape) becomes U+FFFD.

import (
	"unicode/utf16"
	"unicode/utf8"
)

// utf16Units returns the UTF-16 code units of a Go string read as WTF-8
// (see above).
func utf16Units(s string) []uint16 {
	units := make([]uint16, 0, len(s))
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			units = append(units, uint16(c))
			i++
			continue
		}
		if c == 0xED && i+2 < len(s) && s[i+1] >= 0xA0 && s[i+1] <= 0xBF && s[i+2] >= 0x80 && s[i+2] <= 0xBF {
			units = append(units, 0xD000|uint16(s[i+1]&0x3F)<<6|uint16(s[i+2]&0x3F))
			i += 3
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		units = utf16.AppendRune(units, r)
		i += size
	}
	return units
}

// unitsToString converts UTF-16 code units to a Go string; unpaired
// surrogates become U+FFFD.
func unitsToString(units []uint16) string { return string(utf16.Decode(units)) }

// UTF8Encode is utf8.encode(s): the UTF-8 bytes of s, where a lone
// surrogate (see the package comment: WTF-8) is encoded as U+FFFD, as
// Dart's Utf8Encoder does.
func UTF8Encode(s string) []byte {
	if utf8.ValidString(s) {
		return []byte(s)
	}
	return []byte(unitsToString(utf16Units(s)))
}

// The decoder's state machine (Björn Höhrmann's DFA as Dart builds it).
const (
	utf8TypeMask        = 0x1F
	utf8ShiftedByteMask = 0xF0FE

	utf8StateIA = 0x00 // Initial / Accept
	utf8StateBB = 0x10 // Before BOM (unused by the VM decoder)
	utf8StateAB = 0x20 // After BOM (unused by the VM decoder)
	utf8StateX1 = 0x30
	utf8StateX2 = 0x3A
	utf8StateX3 = 0x44
	utf8StateTO = 0x4E
	utf8StateTS = 0x58
	utf8StateQO = 0x62
	utf8StateQR = 0x6C
	utf8StateB1 = 0x76
	utf8StateB2 = 0x80
	utf8StateE1 = 0x41 // Error: Missing extension byte
	utf8StateE2 = 0x43 // Error: Unexpected extension byte
	utf8StateE3 = 0x45 // Error: Invalid byte
	utf8StateE4 = 0x47 // Error: Overlong encoding
	utf8StateE5 = 0x49 // Error: Out of range
	utf8StateE6 = 0x4B // Error: Surrogate
	utf8StateE7 = 0x4D // Error: Unfinished
)

const utf8TypeTable = "" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" + // 00-1F
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" + // 20-3F
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" + // 40-5F
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" + // 60-7F
	"FFFFFFFFFFFFFFFFGGGGGGGGGGGGGGGG" + // 80-9F
	"HHHHHHHHHHHHHHHHHHHHHHHHHHHIHHHJ" + // A0-BF
	"EEBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB" + // C0-DF
	"KCCCCCCCCCCCCDCLONNNMEEEEEEEEEEE" // E0-FF

var utf8TransitionTable = func() string {
	const (
		ia = "\x00"
		x1 = "\x30"
		x2 = "\x3A"
		x3 = "\x44"
		to = "\x4E"
		ts = "\x58"
		qo = "\x62"
		qr = "\x6C"
		b1 = "\x76"
		b2 = "\x80"
		ab = "\x20"
		e1 = "\x41"
		e2 = "\x43"
		e3 = "\x45"
		e4 = "\x47"
		e5 = "\x49"
		e6 = "\x4B"
	)
	return " " +
		// A  B   C   D   E   F   G   H   I   J   K   L   M   N   O
		ia + x1 + x2 + ts + e3 + e2 + e2 + e2 + e2 + e2 + to + x2 + qr + x3 + qo + " " + // IA
		ia + x1 + x2 + ts + e3 + e2 + e2 + e2 + e2 + e2 + to + b1 + qr + x3 + qo + " " + // BB
		ia + x1 + x2 + ts + e3 + e2 + e2 + e2 + e2 + e2 + to + x2 + qr + x3 + qo + " " + // AB
		e1 + e1 + e1 + e1 + e1 + ia + ia + ia + ia + ia + // X1
		e1 + e1 + e1 + e1 + e1 + x1 + x1 + x1 + x1 + x1 + // X2
		e1 + e1 + e1 + e1 + e1 + x2 + x2 + x2 + x2 + x2 + // X3
		e1 + e1 + e1 + e1 + e1 + e4 + e4 + x1 + x1 + x1 + // TO
		e1 + e1 + e1 + e1 + e1 + x1 + x1 + e6 + e6 + e6 + // TS
		e1 + e1 + e1 + e1 + e1 + e4 + x2 + x2 + x2 + x2 + // QO
		e1 + e1 + e1 + e1 + e1 + x2 + e5 + e5 + e5 + e5 + // QR
		e1 + e1 + e1 + e1 + e1 + x1 + x1 + x1 + b2 + x1 + // B1
		e1 + e1 + e1 + e1 + e1 + ia + ia + ia + ia + ab + e1 + e1 + e1 + e1 + e1 // B2
}()

func utf8IsErrorState(state int) bool { return state&1 != 0 }

func utf8ErrorDescription(state int) string {
	switch state {
	case utf8StateE1:
		return "Missing extension byte"
	case utf8StateE2:
		return "Unexpected extension byte"
	case utf8StateE3:
		return "Invalid UTF-8 byte"
	case utf8StateE4:
		return "Overlong encoding"
	case utf8StateE5:
		return "Out of unicode range"
	case utf8StateE6:
		return "Encoded surrogate"
	case utf8StateE7:
		return "Unfinished UTF-8 octet sequence"
	}
	return ""
}

// utf8Decoder is _Utf8Decoder (allowMalformed false) without the BOM
// handling, which its callers do.
type utf8Decoder struct {
	state int
	char  int
}

// decode appends the code units of bytes[start:end] to out, continuing
// from the decoder's state. On malformed input it returns the error state
// and the index of the byte that made the input malformed (Dart's
// _charOrIndex).
func (d *utf8Decoder) decode(bytes []byte, start, end int, out []uint16) (_ []uint16, errState, errIndex int) {
	state, char := d.state, d.char
	for i := start; i < end; i++ {
		b := int(bytes[i])
		typ := int(utf8TypeTable[b]) & utf8TypeMask
		if state == utf8StateIA {
			char = b & (utf8ShiftedByteMask >> typ)
		} else {
			char = (b & 0x3F) | (char << 6)
		}
		state = int(utf8TransitionTable[state+typ])
		if state == utf8StateIA {
			out = utf16.AppendRune(out, rune(char))
		} else if utf8IsErrorState(state) {
			d.state = utf8StateIA
			return out, state, i
		}
	}
	d.state, d.char = state, char
	return out, 0, 0
}

// flush is _Utf8Decoder.flush: an unfinished sequence is an error
// (FormatException without source or offset).
func (d *utf8Decoder) flush() {
	state := d.state
	d.state = utf8StateIA
	if state <= utf8StateAB {
		return
	}
	panic(FormatException{Message: utf8ErrorDescription(utf8StateE7)})
}

// UTF8Decode is utf8.decode(bytes) (utf8.decoder.convert, allowMalformed
// false): a leading byte order mark (EF BB BF) is skipped, as the VM does;
// malformed input panics with Dart's FormatException (its message, bytes
// as the source, and the offset of the offending byte, or the input's
// length for an unfinished sequence).
func UTF8Decode(bytes []byte) string {
	start := 0
	if len(bytes) >= 3 && bytes[0] == 0xEF && bytes[1] == 0xBB && bytes[2] == 0xBF {
		start = 3
	}
	if utf8.Valid(bytes[start:]) {
		return string(bytes[start:])
	}
	var d utf8Decoder
	units, errState, errIndex := d.decode(bytes, start, len(bytes), nil)
	if errState == 0 && d.state != utf8StateIA {
		errState, errIndex = utf8StateE7, len(bytes)
	}
	if errState != 0 {
		panic(FormatException{Message: utf8ErrorDescription(errState), Source: bytes, Offset: errIndex, HasOffset: true})
	}
	return unitsToString(units)
}
