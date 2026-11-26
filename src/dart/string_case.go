package dart

import (
	"unicode"
	"unicode/utf16"
)

// StringToUpperCase is String.toUpperCase as the Dart VM runs it: each code
// point is mapped with Unicode's simple (one-to-one) case mapping (so 'ß'
// stays 'ß'); lone surrogates are kept. The VM's case tables are older than
// Go's unicode package: the mappings Unicode added since (Georgian
// Mtavruli, Cherokee small letters, Latin Extended-D, Adlam...) are not
// applied (dartNoUpperMapping/dartNoLowerMapping, measured against the
// Dart VM of Flutter 3.50.1 over U+0000..U+1FFFF by the raw_key_event_data
// difftest; no character above U+1FFFF has a case mapping).
func StringToUpperCase(s string) string {
	return mapCodePoints(s, func(r rune) rune {
		if unicode.Is(dartNoUpperMapping, r) {
			return r
		}
		return unicode.ToUpper(r)
	})
}

// StringToLowerCase is String.toLowerCase (simple case mapping, as
// StringToUpperCase).
func StringToLowerCase(s string) string {
	return mapCodePoints(s, func(r rune) rune {
		if unicode.Is(dartNoLowerMapping, r) {
			return r
		}
		return unicode.ToLower(r)
	})
}

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

// Code points Go maps but the Dart VM leaves as they are.
var dartNoUpperMapping = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x23f, Hi: 0x240, Stride: 1},
		{Lo: 0x252, Hi: 0x252, Stride: 1},
		{Lo: 0x25c, Hi: 0x25c, Stride: 1},
		{Lo: 0x261, Hi: 0x261, Stride: 1},
		{Lo: 0x265, Hi: 0x266, Stride: 1},
		{Lo: 0x26a, Hi: 0x26a, Stride: 1},
		{Lo: 0x26c, Hi: 0x26c, Stride: 1},
		{Lo: 0x282, Hi: 0x282, Stride: 1},
		{Lo: 0x287, Hi: 0x287, Stride: 1},
		{Lo: 0x29d, Hi: 0x29e, Stride: 1},
		{Lo: 0x3f3, Hi: 0x3f3, Stride: 1},
		{Lo: 0x525, Hi: 0x525, Stride: 1},
		{Lo: 0x527, Hi: 0x527, Stride: 1},
		{Lo: 0x529, Hi: 0x529, Stride: 1},
		{Lo: 0x52b, Hi: 0x52b, Stride: 1},
		{Lo: 0x52d, Hi: 0x52d, Stride: 1},
		{Lo: 0x52f, Hi: 0x52f, Stride: 1},
		{Lo: 0x10d0, Hi: 0x10fa, Stride: 1},
		{Lo: 0x10fd, Hi: 0x10ff, Stride: 1},
		{Lo: 0x13f8, Hi: 0x13fd, Stride: 1},
		{Lo: 0x1c80, Hi: 0x1c88, Stride: 1},
		{Lo: 0x1d8e, Hi: 0x1d8e, Stride: 1},
		{Lo: 0x2c5f, Hi: 0x2c5f, Stride: 1},
		{Lo: 0x2cec, Hi: 0x2cec, Stride: 1},
		{Lo: 0x2cee, Hi: 0x2cee, Stride: 1},
		{Lo: 0x2cf3, Hi: 0x2cf3, Stride: 1},
		{Lo: 0x2d27, Hi: 0x2d27, Stride: 1},
		{Lo: 0x2d2d, Hi: 0x2d2d, Stride: 1},
		{Lo: 0xa661, Hi: 0xa661, Stride: 1},
		{Lo: 0xa699, Hi: 0xa699, Stride: 1},
		{Lo: 0xa69b, Hi: 0xa69b, Stride: 1},
		{Lo: 0xa791, Hi: 0xa791, Stride: 1},
		{Lo: 0xa793, Hi: 0xa794, Stride: 1},
		{Lo: 0xa797, Hi: 0xa797, Stride: 1},
		{Lo: 0xa799, Hi: 0xa799, Stride: 1},
		{Lo: 0xa79b, Hi: 0xa79b, Stride: 1},
		{Lo: 0xa79d, Hi: 0xa79d, Stride: 1},
		{Lo: 0xa79f, Hi: 0xa79f, Stride: 1},
		{Lo: 0xa7a1, Hi: 0xa7a1, Stride: 1},
		{Lo: 0xa7a3, Hi: 0xa7a3, Stride: 1},
		{Lo: 0xa7a5, Hi: 0xa7a5, Stride: 1},
		{Lo: 0xa7a7, Hi: 0xa7a7, Stride: 1},
		{Lo: 0xa7a9, Hi: 0xa7a9, Stride: 1},
		{Lo: 0xa7b5, Hi: 0xa7b5, Stride: 1},
		{Lo: 0xa7b7, Hi: 0xa7b7, Stride: 1},
		{Lo: 0xa7b9, Hi: 0xa7b9, Stride: 1},
		{Lo: 0xa7bb, Hi: 0xa7bb, Stride: 1},
		{Lo: 0xa7bd, Hi: 0xa7bd, Stride: 1},
		{Lo: 0xa7bf, Hi: 0xa7bf, Stride: 1},
		{Lo: 0xa7c1, Hi: 0xa7c1, Stride: 1},
		{Lo: 0xa7c3, Hi: 0xa7c3, Stride: 1},
		{Lo: 0xa7c8, Hi: 0xa7c8, Stride: 1},
		{Lo: 0xa7ca, Hi: 0xa7ca, Stride: 1},
		{Lo: 0xa7d1, Hi: 0xa7d1, Stride: 1},
		{Lo: 0xa7d7, Hi: 0xa7d7, Stride: 1},
		{Lo: 0xa7d9, Hi: 0xa7d9, Stride: 1},
		{Lo: 0xa7f6, Hi: 0xa7f6, Stride: 1},
		{Lo: 0xab53, Hi: 0xab53, Stride: 1},
		{Lo: 0xab70, Hi: 0xabbf, Stride: 1},
	},
	R32: []unicode.Range32{
		{Lo: 0x104d8, Hi: 0x104fb, Stride: 1},
		{Lo: 0x10597, Hi: 0x105a1, Stride: 1},
		{Lo: 0x105a3, Hi: 0x105b1, Stride: 1},
		{Lo: 0x105b3, Hi: 0x105b9, Stride: 1},
		{Lo: 0x105bb, Hi: 0x105bc, Stride: 1},
		{Lo: 0x10cc0, Hi: 0x10cf2, Stride: 1},
		{Lo: 0x118c0, Hi: 0x118df, Stride: 1},
		{Lo: 0x16e60, Hi: 0x16e7f, Stride: 1},
		{Lo: 0x1e922, Hi: 0x1e943, Stride: 1},
	},
}

var dartNoLowerMapping = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x37f, Hi: 0x37f, Stride: 1},
		{Lo: 0x524, Hi: 0x524, Stride: 1},
		{Lo: 0x526, Hi: 0x526, Stride: 1},
		{Lo: 0x528, Hi: 0x528, Stride: 1},
		{Lo: 0x52a, Hi: 0x52a, Stride: 1},
		{Lo: 0x52c, Hi: 0x52c, Stride: 1},
		{Lo: 0x52e, Hi: 0x52e, Stride: 1},
		{Lo: 0x10c7, Hi: 0x10c7, Stride: 1},
		{Lo: 0x10cd, Hi: 0x10cd, Stride: 1},
		{Lo: 0x13a0, Hi: 0x13f5, Stride: 1},
		{Lo: 0x1c90, Hi: 0x1cba, Stride: 1},
		{Lo: 0x1cbd, Hi: 0x1cbf, Stride: 1},
		{Lo: 0x2c2f, Hi: 0x2c2f, Stride: 1},
		{Lo: 0x2c70, Hi: 0x2c70, Stride: 1},
		{Lo: 0x2c7e, Hi: 0x2c7f, Stride: 1},
		{Lo: 0x2ceb, Hi: 0x2ceb, Stride: 1},
		{Lo: 0x2ced, Hi: 0x2ced, Stride: 1},
		{Lo: 0x2cf2, Hi: 0x2cf2, Stride: 1},
		{Lo: 0xa660, Hi: 0xa660, Stride: 1},
		{Lo: 0xa698, Hi: 0xa698, Stride: 1},
		{Lo: 0xa69a, Hi: 0xa69a, Stride: 1},
		{Lo: 0xa78d, Hi: 0xa78d, Stride: 1},
		{Lo: 0xa790, Hi: 0xa790, Stride: 1},
		{Lo: 0xa792, Hi: 0xa792, Stride: 1},
		{Lo: 0xa796, Hi: 0xa796, Stride: 1},
		{Lo: 0xa798, Hi: 0xa798, Stride: 1},
		{Lo: 0xa79a, Hi: 0xa79a, Stride: 1},
		{Lo: 0xa79c, Hi: 0xa79c, Stride: 1},
		{Lo: 0xa79e, Hi: 0xa79e, Stride: 1},
		{Lo: 0xa7a0, Hi: 0xa7a0, Stride: 1},
		{Lo: 0xa7a2, Hi: 0xa7a2, Stride: 1},
		{Lo: 0xa7a4, Hi: 0xa7a4, Stride: 1},
		{Lo: 0xa7a6, Hi: 0xa7a6, Stride: 1},
		{Lo: 0xa7a8, Hi: 0xa7a8, Stride: 1},
		{Lo: 0xa7aa, Hi: 0xa7ae, Stride: 1},
		{Lo: 0xa7b0, Hi: 0xa7b4, Stride: 1},
		{Lo: 0xa7b6, Hi: 0xa7b6, Stride: 1},
		{Lo: 0xa7b8, Hi: 0xa7b8, Stride: 1},
		{Lo: 0xa7ba, Hi: 0xa7ba, Stride: 1},
		{Lo: 0xa7bc, Hi: 0xa7bc, Stride: 1},
		{Lo: 0xa7be, Hi: 0xa7be, Stride: 1},
		{Lo: 0xa7c0, Hi: 0xa7c0, Stride: 1},
		{Lo: 0xa7c2, Hi: 0xa7c2, Stride: 1},
		{Lo: 0xa7c4, Hi: 0xa7c7, Stride: 1},
		{Lo: 0xa7c9, Hi: 0xa7c9, Stride: 1},
		{Lo: 0xa7d0, Hi: 0xa7d0, Stride: 1},
		{Lo: 0xa7d6, Hi: 0xa7d6, Stride: 1},
		{Lo: 0xa7d8, Hi: 0xa7d8, Stride: 1},
		{Lo: 0xa7f5, Hi: 0xa7f5, Stride: 1},
	},
	R32: []unicode.Range32{
		{Lo: 0x104b0, Hi: 0x104d3, Stride: 1},
		{Lo: 0x10570, Hi: 0x1057a, Stride: 1},
		{Lo: 0x1057c, Hi: 0x1058a, Stride: 1},
		{Lo: 0x1058c, Hi: 0x10592, Stride: 1},
		{Lo: 0x10594, Hi: 0x10595, Stride: 1},
		{Lo: 0x10c80, Hi: 0x10cb2, Stride: 1},
		{Lo: 0x118a0, Hi: 0x118bf, Stride: 1},
		{Lo: 0x16e40, Hi: 0x16e5f, Stride: 1},
		{Lo: 0x1e900, Hi: 0x1e921, Stride: 1},
	},
}
