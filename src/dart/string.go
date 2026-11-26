package dart

// Dart String operations on Go strings, at the UTF-16 level Dart strings
// have. A Go string is read as WTF-8 (see utf8.go): its code units are the
// UTF-16 encoding of its text, and a three-byte encoded surrogate is that
// lone surrogate. The strings these functions return are WTF-8 too: a
// substring that splits a surrogate pair keeps each half as a lone
// surrogate, so lengths and offsets round-trip exactly as in Dart.

import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// StringLength is String.length: the number of UTF-16 code units.
func StringLength(s string) int {
	n := 0
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			n++
			i++
			continue
		}
		if isWTF8Surrogate(s, i) {
			n++
			i += 3
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		n += utf16.RuneLen(r)
		i += size
	}
	return n
}

func isWTF8Surrogate(s string, i int) bool {
	return s[i] == 0xED && i+2 < len(s) && s[i+1] >= 0xA0 && s[i+1] <= 0xBF && s[i+2] >= 0x80 && s[i+2] <= 0xBF
}

// StringCodeUnits is String.codeUnits.
func StringCodeUnits(s string) []uint16 { return utf16Units(s) }

// StringFromCharCodes is String.fromCharCodes for UTF-16 code units: the
// WTF-8 string holding them (an unpaired surrogate stays a lone surrogate).
func StringFromCharCodes(units []uint16) string {
	var b strings.Builder
	b.Grow(len(units))
	for i := 0; i < len(units); i++ {
		u := units[i]
		switch {
		case u < 0x80:
			b.WriteByte(byte(u))
		case u >= 0xD800 && u < 0xDC00 && i+1 < len(units) && units[i+1] >= 0xDC00 && units[i+1] < 0xE000:
			b.WriteRune(utf16.DecodeRune(rune(u), rune(units[i+1])))
			i++
		case u >= 0xD800 && u < 0xE000:
			// WTF-8: the generalized UTF-8 encoding of the surrogate.
			b.WriteByte(0xED)
			b.WriteByte(byte(0x80 | (u>>6)&0x3F))
			b.WriteByte(byte(0x80 | u&0x3F))
		default:
			b.WriteRune(rune(u))
		}
	}
	return b.String()
}

// StringFromCharCode is String.fromCharCode(code): a code point (above
// 0xFFFF it becomes a surrogate pair) or a lone surrogate.
func StringFromCharCode(code int) string {
	if code < 0 || code > 0x10FFFF {
		panic(NewRangeErrorRange(code, 0, 0x10FFFF, "", ""))
	}
	if code <= 0xFFFF {
		return StringFromCharCodes([]uint16{uint16(code)})
	}
	return string(rune(code))
}

// CodeUnitAt is String.codeUnitAt(index); it panics with Dart's RangeError
// (an IndexError) when index is out of range.
func CodeUnitAt(s string, index int) int {
	units := utf16Units(s)
	if index < 0 || index >= len(units) {
		panic(IndexError{Index: index, Length: len(units), Name: "index"})
	}
	return int(units[index])
}

// Substring is String.substring(start, [end]) (end defaults to the
// length); it panics with Dart's RangeError on bad bounds.
func Substring(s string, start int, end ...int) string {
	units := utf16Units(s)
	e := len(units)
	if len(end) > 0 {
		e = end[0]
	}
	if start < 0 || start > len(units) {
		panic(NewRangeErrorRange(start, 0, len(units), "start", ""))
	}
	if e < start || e > len(units) {
		panic(NewRangeErrorRange(e, start, len(units), "end", ""))
	}
	return StringFromCharCodes(units[start:e])
}

// ReplaceRange is String.replaceRange(start, end, replacement).
func ReplaceRange(s string, start, end int, replacement string) string {
	units := utf16Units(s)
	if start < 0 || start > len(units) {
		panic(NewRangeErrorRange(start, 0, len(units), "start", ""))
	}
	if end < start || end > len(units) {
		panic(NewRangeErrorRange(end, start, len(units), "end", ""))
	}
	out := make([]uint16, 0, len(units)+len(replacement))
	out = append(out, units[:start]...)
	out = append(out, utf16Units(replacement)...)
	out = append(out, units[end:]...)
	return StringFromCharCodes(out)
}

// Concat is Dart's `a + b + ...`: Go's + on strings whose ends are lone
// surrogates (halves of a split pair) leaves them as two WTF-8 surrogates,
// which reads as the same code units but isn't equal (==) to the UTF-8 of
// the pair; Concat joins the halves back as Dart's string holds them.
func Concat(parts ...string) string {
	var units []uint16
	for _, p := range parts {
		units = append(units, utf16Units(p)...)
	}
	return StringFromCharCodes(units)
}
