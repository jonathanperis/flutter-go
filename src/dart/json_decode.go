package dart

// json.decode and utf8.decoder.fuse(json.decoder) as the Dart VM runs them:
// a port of the VM's _ChunkedJsonParser with its _JsonStringParser and
// _JsonUtf8Parser front ends and _JsonListener (lib/_internal/vm/lib/
// convert_patch.dart in the SDK shipped with Flutter 3.50.1), for the
// single-chunk conversions (convert), so the partial-value states only
// matter at the end of the input.
//
// Values: a JSON object is a map[string]any (Go maps don't keep Dart's
// insertion order; with duplicate keys the last value wins, as in Dart),
// an array []any, an integer literal an int (Dart's 64-bit int; literals
// out of its range are doubles, as in Dart), other numbers float64,
// strings string (a lone surrogate from a \u escape becomes U+FFFD: Go
// strings are UTF-8), true/false bool and null nil. Errors panic with
// Dart's FormatException (message, source, offset).

import (
	"math"
	"strconv"
	"unicode/utf16"
)

// JSONDecode is json.decode(source) (jsonDecode). Offsets in errors count
// UTF-16 code units, as Dart's do.
func JSONDecode(source string) any {
	p := &jsonParser{units: utf16.Encode([]rune(source)), isUTF16: true, source: source}
	p.end = len(p.units)
	p.parse(0)
	p.close()
	return p.result()
}

// JSONUTF8Decode is utf8.decoder.fuse(json.decoder).convert(bytes), the
// fused decoder Flutter's JSONMessageCodec uses: a leading byte order mark
// is skipped, and malformed UTF-8 inside strings is reported by the UTF-8
// decoder (its messages, with the bytes as the source).
func JSONUTF8Decode(bytes []byte) any {
	p := &jsonParser{bytes: bytes, source: bytes}
	p.end = len(bytes)
	// Starts out checking for an optional BOM (KWD_BOM, count = 0).
	p.partialState = jsonPartialKeyword | jsonKwdBOM
	p.parse(0)
	p.close()
	return p.result()
}

const (
	jsonTopLevel        = 0
	jsonInsideArray     = 1
	jsonInsideObject    = 2
	jsonAfterColon      = 3
	jsonAllowStringMask = 8
	jsonAllowValueMask  = 4
	jsonAllowValue      = 0
	jsonStringOnly      = 4
	jsonNoValues        = 12
	jsonEmpty           = 0
	jsonNonEmpty        = 16

	jsonStateInitial     = jsonTopLevel | jsonEmpty | jsonAllowValue
	jsonStateEnd         = jsonTopLevel | jsonNonEmpty | jsonNoValues
	jsonStateArrayEmpty  = jsonInsideArray | jsonEmpty | jsonAllowValue
	jsonStateArrayValue  = jsonInsideArray | jsonNonEmpty | jsonNoValues
	jsonStateArrayComma  = jsonInsideArray | jsonNonEmpty | jsonAllowValue
	jsonStateObjectEmpty = jsonInsideObject | jsonEmpty | jsonStringOnly
	jsonStateObjectKey   = jsonInsideObject | jsonNonEmpty | jsonNoValues
	jsonStateObjectColon = jsonAfterColon | jsonNonEmpty | jsonAllowValue
	jsonStateObjectValue = jsonAfterColon | jsonNonEmpty | jsonNoValues
	jsonStateObjectComma = jsonInsideObject | jsonNonEmpty | jsonStringOnly
	jsonValueReadBits    = jsonNonEmpty | jsonNoValues

	jsonNoPartial      = 0
	jsonPartialString  = 1
	jsonPartialNumeral = 2
	jsonPartialKeyword = 3
	jsonMaskPartial    = 3

	jsonNumSign     = 0
	jsonNumZero     = 4
	jsonNumDigit    = 8
	jsonNumDot      = 12
	jsonNumDotDigit = 16
	jsonNumE        = 20
	jsonNumESign    = 24
	jsonNumEDigit   = 28

	jsonKwdNull       = 0
	jsonKwdTrue       = 4
	jsonKwdFalse      = 8
	jsonKwdBOM        = 12
	jsonKwdTypeMask   = 12
	jsonKwdTypeShift  = 2
	jsonKwdCountShift = 4

	jsonCharSimpleStringEnd = 1
	jsonCharWhitespace      = 2
)

// jsonCharacterAttributes is _characterAttributes (offset by ' ').
const jsonCharacterAttributes = "" +
	"!!!!!!!!!##!!#!!!!!!!!!!!!!!!!!!\" !                             " +
	"                            !                                   " +
	"                                                                " +
	"                                                                "

var (
	negativeZero     = math.Copysign(0, -1)
	positiveInfinity = math.Inf(1)
	negativeInfinity = math.Inf(-1)
)

var jsonPowersOfTen = [...]float64{1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11,
	1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19, 1e20, 1e21, 1e22}

type jsonParser struct {
	bytes   []byte   // the UTF-8 chunk (JSONUTF8Decode)
	units   []uint16 // the UTF-16 chunk (JSONDecode)
	isUTF16 bool
	source  any // the chunk, as FormatException's source
	end     int

	state        int
	states       []int
	partialState int
	numStart     int // start of a numeral still open at the end of the chunk

	str     []uint16 // the string being built (beginString's buffer)
	decoder utf8Decoder

	// _JsonListener
	stack     []jsonFrame
	container []any
	inside    bool // currentContainer != null
	value     any
}

func (p *jsonParser) char(i int) int {
	if p.isUTF16 {
		return int(p.units[i])
	}
	return int(p.bytes[i])
}

func (p *jsonParser) fail(position int, message ...string) {
	msg := "Unexpected character"
	if len(message) > 0 {
		msg = message[0]
	} else if position == p.end {
		msg = "Unexpected end of input"
	}
	panic(FormatException{Message: msg, Source: p.source, Offset: position, HasOffset: true})
}

// ---- _JsonListener

// jsonFrame is an entry of _JsonListener.stack.
type jsonFrame struct {
	container []any
	inside    bool
}

func (p *jsonParser) beginContainer() {
	p.stack = append(p.stack, jsonFrame{p.container, p.inside})
	p.container, p.inside = []any{}, true
}

func (p *jsonParser) popContainer() {
	p.value = p.container
	n := len(p.stack) - 1
	p.container, p.inside = p.stack[n].container, p.stack[n].inside
	p.stack = p.stack[:n]
}

func (p *jsonParser) propertyName() {
	p.container = append(p.container, p.value)
	p.value = nil
}

func (p *jsonParser) propertyValue() {
	p.container = append(p.container, p.value)
	p.value = nil
}

func (p *jsonParser) endObject() {
	p.popContainer()
	pairs := p.value.([]any)
	m := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		m[pairs[i].(string)] = pairs[i+1]
	}
	p.value = m
}

func (p *jsonParser) arrayElement() {
	p.container = append(p.container, p.value)
	p.value = nil
}

func (p *jsonParser) endArray() { p.popContainer() }

func (p *jsonParser) result() any { return p.value }

// ---- _ChunkedJsonParser

func (p *jsonParser) close() {
	if p.partialState != jsonNoPartial {
		switch p.partialState & jsonMaskPartial {
		case jsonPartialNumeral:
			p.finishChunkNumber(p.partialState&^jsonMaskPartial, p.numStart, p.end)
		case jsonPartialString:
			p.fail(p.end, "Unterminated string")
		default: // Incomplete literal.
			p.fail(p.end)
		}
	}
	if p.state != jsonStateEnd {
		p.fail(p.end)
	}
}

// parsePartial continues the only partial value a single chunk starts
// with: the UTF-8 parser's optional byte order mark.
func (p *jsonParser) parsePartial(position int) int {
	if position == p.end {
		return position
	}
	partialState := p.partialState
	p.partialState = jsonNoPartial
	return p.parsePartialKeyword(position, partialState&^jsonMaskPartial)
}

func (p *jsonParser) parsePartialKeyword(position, partialState int) int {
	keywordType := partialState & jsonKwdTypeMask
	count := partialState >> jsonKwdCountShift
	keyword := [...]string{"null", "true", "false", "\xEF\xBB\xBF"}[keywordType>>jsonKwdTypeShift]
	for {
		if position == p.end {
			p.partialState = jsonPartialKeyword | keywordType | count<<jsonKwdCountShift
			return p.end
		}
		if p.char(position) != int(keyword[count]) {
			if count == 0 {
				return position
			}
			p.fail(position)
		}
		position++
		count++
		if count >= len(keyword) {
			break
		}
	}
	if keywordType == jsonKwdNull {
		p.value = nil
	} else if keywordType != jsonKwdBOM {
		p.value = keywordType == jsonKwdTrue
	}
	return position
}

func parseHexDigit(char int) int {
	digit := char ^ 0x30
	if digit <= 9 {
		return digit
	}
	letter := (char | 0x20) ^ 0x60
	if letter <= 6 && letter > 0 {
		return letter + 9
	}
	return -1
}

func (p *jsonParser) parse(position int) {
	length := p.end
	if p.partialState != jsonNoPartial {
		position = p.parsePartial(position)
		if position == length {
			return
		}
	}
	state := p.state
outer:
	for position < length {
		char := 0
		for {
			char = p.char(position)
			if p.isUTF16 && char > 0xFF {
				break
			}
			if (int(jsonCharacterAttributes[char])-' ')&jsonCharWhitespace == 0 {
				break
			}
			position++
			if position >= length {
				break outer
			}
		}
		switch char {
		case '"':
			if state&jsonAllowStringMask != 0 {
				p.fail(position)
			}
			state |= jsonValueReadBits
			position = p.parseString(position + 1)
		case '[':
			if state&jsonAllowValueMask != 0 {
				p.fail(position)
			}
			p.beginContainer()
			p.states = append(p.states, state)
			state = jsonStateArrayEmpty
			position++
		case '{':
			if state&jsonAllowValueMask != 0 {
				p.fail(position)
			}
			p.beginContainer()
			p.states = append(p.states, state)
			state = jsonStateObjectEmpty
			position++
		case 'n':
			if state&jsonAllowValueMask != 0 {
				p.fail(position)
			}
			state |= jsonValueReadBits
			position = p.parseKeyword(position, "null", jsonKwdNull)
		case 'f':
			if state&jsonAllowValueMask != 0 {
				p.fail(position)
			}
			state |= jsonValueReadBits
			position = p.parseKeyword(position, "false", jsonKwdFalse)
		case 't':
			if state&jsonAllowValueMask != 0 {
				p.fail(position)
			}
			state |= jsonValueReadBits
			position = p.parseKeyword(position, "true", jsonKwdTrue)
		case ':':
			if state != jsonStateObjectKey {
				p.fail(position)
			}
			p.propertyName()
			state = jsonStateObjectColon
			position++
		case ',':
			if state == jsonStateObjectValue {
				p.propertyValue()
				state = jsonStateObjectComma
				position++
			} else if state == jsonStateArrayValue {
				p.arrayElement()
				state = jsonStateArrayComma
				position++
			} else {
				p.fail(position)
			}
		case ']':
			if state == jsonStateArrayEmpty {
				p.endArray()
			} else if state == jsonStateArrayValue {
				p.arrayElement()
				p.endArray()
			} else {
				p.fail(position)
			}
			state = p.restoreState() | jsonValueReadBits
			position++
		case '}':
			if state == jsonStateObjectEmpty {
				p.endObject()
			} else if state == jsonStateObjectValue {
				p.propertyValue()
				p.endObject()
			} else {
				p.fail(position)
			}
			state = p.restoreState() | jsonValueReadBits
			position++
		default:
			if state&jsonAllowValueMask != 0 {
				p.fail(position)
			}
			state |= jsonValueReadBits
			position = p.parseNumber(char, position)
		}
	}
	p.state = state
}

func (p *jsonParser) restoreState() int {
	n := len(p.states) - 1
	s := p.states[n]
	p.states = p.states[:n]
	return s
}

// parseKeyword is parseTrue/parseFalse/parseNull, with parseKeywordPrefix
// when the chunk ends inside the keyword.
func (p *jsonParser) parseKeyword(position int, keyword string, typ int) int {
	if p.end < position+len(keyword) {
		start := position
		count := 1
		for position++; position < p.end; position++ {
			if p.char(position) != int(keyword[count]) {
				p.fail(start)
			}
			count++
		}
		p.partialState = jsonPartialKeyword | typ | count<<jsonKwdCountShift
		return p.end
	}
	for i := 1; i < len(keyword); i++ {
		if p.char(position+i) != int(keyword[i]) {
			p.fail(position)
		}
	}
	if typ == jsonKwdNull {
		p.value = nil
	} else {
		p.value = typ == jsonKwdTrue
	}
	return position + len(keyword)
}

func (p *jsonParser) isSimpleStringEnd(char int) bool {
	return (int(jsonCharacterAttributes[char])-' ')&jsonCharSimpleStringEnd != 0
}

// getString is the parser's getString for a literal without escapes.
func (p *jsonParser) getString(start, end, bits int) string {
	if p.isUTF16 {
		return unitsToString(p.units[start:end])
	}
	if bits <= 0x7f {
		return string(p.bytes[start:end])
	}
	p.beginString()
	if start < end {
		p.addSliceToString(start, end)
	}
	return p.endString()
}

func (p *jsonParser) beginString() {
	p.decoder = utf8Decoder{} // decoder.reset()
	p.str = p.str[:0]
}

func (p *jsonParser) addSliceToString(start, end int) {
	if p.isUTF16 {
		p.str = append(p.str, p.units[start:end]...)
		return
	}
	var errState, errIndex int
	p.str, errState, errIndex = p.decoder.decode(p.bytes, start, end, p.str)
	if errState != 0 {
		panic(FormatException{Message: utf8ErrorDescription(errState), Source: p.bytes, Offset: errIndex, HasOffset: true})
	}
}

func (p *jsonParser) endString() string {
	if !p.isUTF16 {
		p.decoder.flush()
	}
	return unitsToString(p.str)
}

func (p *jsonParser) parseString(position int) int {
	start := position
	end := p.end
	bits := 0
	char := 0
	if position < end {
		for {
			char = p.char(position)
			position++
			bits |= char
			if !(p.isUTF16 && char > 0xFF) && p.isSimpleStringEnd(char) {
				break
			}
			if position >= end {
				break
			}
		}
		if char == '"' {
			sliceEnd := position - 1
			p.value = p.getString(start, sliceEnd, bits)
			return sliceEnd + 1
		}
		if char == '\\' {
			sliceEnd := position - 1
			p.beginString()
			if start < sliceEnd {
				p.addSliceToString(start, sliceEnd)
			}
			return p.parseStringToBuffer(sliceEnd)
		}
		if char < ' ' {
			p.fail(position-1, "Control character in string")
		}
	}
	p.beginString()
	if start < end {
		p.addSliceToString(start, end)
	}
	return p.chunkString()
}

func (p *jsonParser) chunkString() int {
	p.partialState = jsonPartialString
	return p.end
}

func (p *jsonParser) parseStringToBuffer(position int) int {
	end := p.end
	start := position
	for {
		if position == end {
			if position > start {
				p.addSliceToString(start, position)
			}
			return p.chunkString()
		}
		char := 0
		for {
			char = p.char(position)
			position++
			if !(p.isUTF16 && char > 0xFF) && p.isSimpleStringEnd(char) {
				break
			}
			if position >= end {
				break
			}
		}
		if char < ' ' {
			p.fail(position - 1) // Control character in string.
		}
		if char == '"' {
			quotePosition := position - 1
			if quotePosition > start {
				p.addSliceToString(start, quotePosition)
			}
			p.value = p.endString()
			return position
		}
		if char != '\\' {
			continue
		}
		// Handle escape.
		if position-1 > start {
			p.addSliceToString(start, position-1)
		}
		if position == end {
			return p.chunkString()
		}
		position = p.parseStringEscape(position)
		if position == end {
			return position
		}
		start = position
	}
}

func (p *jsonParser) parseStringEscape(position int) int {
	char := p.char(position)
	position++
	length := p.end
	switch char {
	case 'b':
		char = 0x08
	case 'f':
		char = 0x0c
	case 'n':
		char = 0x0a
	case 'r':
		char = 0x0d
	case 't':
		char = 0x09
	case '/', '\\', '"':
	case 'u':
		hexStart := position - 1
		value := 0
		for i := 0; i < 4; i++ {
			if position == length {
				return p.chunkString() // a partial \u escape: unterminated at close
			}
			char = p.char(position)
			position++
			digit := char ^ 0x30
			value *= 16
			if digit <= 9 {
				value += digit
			} else {
				digit = (char | 0x20) - 'a'
				if digit < 0 || digit > 5 {
					p.fail(hexStart, "Invalid unicode escape")
				}
				value += digit + 10
			}
		}
		char = value
	default:
		if char < ' ' {
			p.fail(position, "Control character in string")
		}
		p.fail(position, "Unrecognized string escape")
	}
	p.addCharUnit(char)
	if position == length {
		return p.chunkString()
	}
	return position
}

// addCharUnit is addCharToString for a UTF-16 code unit (StringBuffer
// .writeCharCode of an escape's value, which may be a surrogate).
func (p *jsonParser) addCharUnit(unit int) {
	if !p.isUTF16 {
		p.decoder.flush()
	}
	p.str = append(p.str, uint16(unit))
}

func (p *jsonParser) beginChunkNumber(state, start int) int {
	p.numStart = start
	p.partialState = jsonPartialNumeral | state
	return p.end
}

func (p *jsonParser) numberText(start, end int) string {
	if p.isUTF16 {
		return unitsToString(p.units[start:end])
	}
	return string(p.bytes[start:end])
}

func (p *jsonParser) finishChunkNumber(state, start, end int) {
	if state == jsonNumZero {
		p.value = 0
		return
	}
	switch state {
	case jsonNumDigit:
		p.value = numParse(p.numberText(start, end))
	case jsonNumDotDigit, jsonNumEDigit:
		p.value = doubleParse(p.numberText(start, end))
	default:
		p.fail(p.end, "Unterminated number literal")
	}
}

// numParse is num.parse for a valid JSON integer literal: an int, or a
// double when it doesn't fit in 64 bits.
func numParse(s string) any {
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(v)
	}
	return doubleParse(s)
}

// doubleParse is double.parse for a valid JSON number literal (correctly
// rounded; out-of-range values are ±Infinity or ±0.0).
func doubleParse(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func (p *jsonParser) parseNumber(char, position int) int {
	// Also called on any unexpected character.
	// Format:
	//  '-'?('0'|[1-9][0-9]*)('.'[0-9]+)?([eE][+-]?[0-9]+)?
	start := position
	length := p.end
	// Collects an int value while parsing (negated, to represent -2^63).
	intValue := 0
	doubleValue := 0.0
	sign := 1
	isDouble := false
	if char == '-' {
		sign = -1
		position++
		if position == length {
			return p.beginChunkNumber(jsonNumSign, start)
		}
		char = p.char(position)
	}
	digit := char ^ '0'
	if digit > 9 {
		if sign < 0 {
			p.fail(position, "Missing expected digit")
		} else {
			// If it doesn't even start out as a numeral.
			p.fail(position)
		}
	}
	if digit == 0 {
		position++
		if position == length {
			return p.beginChunkNumber(jsonNumZero, start)
		}
		char = p.char(position)
		digit = char ^ '0'
		// If starting with zero, next character must not be digit.
		if digit <= 9 {
			p.fail(position)
		}
	} else {
		digitCount := 0
		for {
			if digitCount >= 18 {
				// Check for overflow.
				highDigit := digit >> 3
				if sign < 0 {
					highDigit &= digit
				}
				if digitCount == 19 || intValue-highDigit < -922337203685477580 {
					isDouble = true
					// Not trusted to be exact: reparsed with double.parse.
					doubleValue = 9223372036854775808.0
				}
			}
			intValue = 10*intValue - digit
			digitCount++
			position++
			if position == length {
				return p.beginChunkNumber(jsonNumDigit, start)
			}
			char = p.char(position)
			digit = char ^ '0'
			if digit > 9 {
				break
			}
		}
	}
	if char == '.' {
		if !isDouble {
			isDouble = true
			if intValue == 0 {
				doubleValue = 0.0
			} else {
				doubleValue = -float64(intValue)
			}
		}
		intValue = 0
		position++
		if position == length {
			return p.beginChunkNumber(jsonNumDot, start)
		}
		char = p.char(position)
		digit = char ^ '0'
		if digit > 9 {
			p.fail(position)
		}
		for {
			doubleValue = float64(10.0*doubleValue) + float64(digit)
			intValue--
			position++
			if position == length {
				return p.beginChunkNumber(jsonNumDotDigit, start)
			}
			char = p.char(position)
			digit = char ^ '0'
			if digit > 9 {
				break
			}
		}
	}
	if char|0x20 == 'e' {
		if !isDouble {
			isDouble = true
			if intValue == 0 {
				doubleValue = 0.0
			} else {
				doubleValue = -float64(intValue)
			}
			intValue = 0
		}
		position++
		if position == length {
			return p.beginChunkNumber(jsonNumE, start)
		}
		char = p.char(position)
		expSign := 1
		exponent := 0
		if (char+1)|2 == 0x2e { // + or -
			expSign = 0x2C - char // -1 for MINUS, +1 for PLUS
			position++
			if position == length {
				return p.beginChunkNumber(jsonNumESign, start)
			}
			char = p.char(position)
		}
		digit = char ^ '0'
		if digit > 9 {
			p.fail(position, "Missing expected digit")
		}
		exponentOverflow := false
		for {
			exponent = 10*exponent + digit
			if exponent > 400 {
				exponentOverflow = true
			}
			position++
			if position == length {
				return p.beginChunkNumber(jsonNumEDigit, start)
			}
			char = p.char(position)
			digit = char ^ '0'
			if digit > 9 {
				break
			}
		}
		if exponentOverflow {
			if doubleValue == 0.0 || expSign < 0 {
				if sign < 0 {
					p.value = negativeZero
				} else {
					p.value = 0.0
				}
			} else if sign < 0 {
				p.value = negativeInfinity
			} else {
				p.value = positiveInfinity
			}
			return position
		}
		intValue += expSign * exponent
	}
	if !isDouble {
		bitFlag := -(sign + 1) >> 1 // 0 if sign == -1, -1 if sign == 1
		// Negate if bitFlag is -1 by doing ~intValue + 1
		p.value = (intValue ^ bitFlag) - bitFlag
		return position
	}
	// Double values at or above 2^53 may have lost precision.
	const maxExactDouble = 9007199254740992.0
	if doubleValue < maxExactDouble {
		exponent := intValue
		signedMantissa := float64(doubleValue * float64(sign))
		if exponent >= -22 {
			if exponent < 0 {
				p.value = signedMantissa / jsonPowersOfTen[-exponent]
				return position
			}
			if exponent == 0 {
				p.value = signedMantissa
				return position
			}
			if exponent <= 22 {
				p.value = float64(signedMantissa * jsonPowersOfTen[exponent])
				return position
			}
		}
	}
	// Outside the exact range: the system double parsing.
	p.value = doubleParse(p.numberText(start, position))
	return position
}
