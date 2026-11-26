package dart

import (
	"unicode"
	"unicode/utf16"
)

// StringToUpperCase is String.toUpperCase as the Dart VM runs it: each code
// point is mapped with Unicode's simple (one-to-one) case mapping (so 'ß'
// stays 'ß'); lone surrogates are kept.
func StringToUpperCase(s string) string { return mapCodePoints(s, unicode.ToUpper) }

// StringToLowerCase is String.toLowerCase (simple case mapping, as
// StringToUpperCase).
func StringToLowerCase(s string) string { return mapCodePoints(s, unicode.ToLower) }

func mapCodePoints(s string, f func(rune) rune) string {
	units := utf16Units(s)
	out := make([]uint16, 0, len(units))
	for i := 0; i < len(units); i++ {
		u := units[i]
		if u >= 0xD800 && u < 0xDC00 && i+1 < len(units) && units[i+1] >= 0xDC00 && units[i+1] < 0xE000 {
			r := f(utf16.DecodeRune(rune(u), rune(units[i+1])))
			out = utf16.AppendRune(out, r)
			i++
			continue
		}
		if u >= 0xD800 && u < 0xE000 {
			out = append(out, u)
			continue
		}
		out = utf16.AppendRune(out, f(rune(u)))
	}
	return StringFromCharCodes(out)
}
