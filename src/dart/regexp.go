package dart

// dart:core's RegExp and Match with Dart's (ECMAScript's) matching rules on
// UTF-16 code units, and Pattern.allMatches for every Pattern.
//
// A RegExp's source is parsed by Go's regexp/syntax (after rewriting the
// ECMAScript escapes RE2 spells differently: \uXXXX, \u{X}, \s, \S) and run
// by a backtracking matcher (a port of Go's regexp backtracker) over the
// input's UTF-16 code units, or its code points with `unicode: true`. So,
// as in Dart: leftmost-first alternation, `.` and classes match one code
// unit in non-Unicode mode (a surrogate pair is two characters), `.` does
// not match \n, \r, U+2028 or U+2029 unless dotAll, multiline ^/$ treat
// those four as line terminators, \s is ECMAScript's white space, \b/\w are
// ASCII, offsets are UTF-16 indices, and allMatches iterates as Dart does
// (an empty match right after a match is found; after an empty match the
// search resumes one character later). Not supported (RE2 syntax): look-
// around, backreferences; a source using them panics with a
// FormatException. Case-insensitive matching folds with Unicode simple case
// folding (ECMAScript non-Unicode mode upper-cases, which differs for a few
// characters such as U+017F); a non-BMP literal or class in a non-Unicode
// pattern matches nothing (ECMAScript sees its two surrogates).

import (
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Match is dart:core's Match (a RegExpMatch for a RegExp's matches, a
// StringMatch for a String pattern's): Start and End are UTF-16 indices
// into Input.
type Match struct {
	Input   string
	Pattern Pattern
	Start   int
	End     int
	groups  []int // UTF-16 index pairs, -1 when a group did not match; nil: group 0 only
	units   []uint16
}

// GroupCount is the number of capture groups in the pattern.
func (m Match) GroupCount() int {
	if len(m.groups) == 0 {
		return 0
	}
	return len(m.groups)/2 - 1
}

// Group is group(index) (match[index]): nil when the group did not
// participate in the match. Group 0 is the whole match.
func (m Match) Group(index int) *string {
	if index < 0 || index > m.GroupCount() {
		panic(NewRangeErrorRange(index, 0, m.GroupCount(), "", ""))
	}
	start, end := m.Start, m.End
	if index > 0 {
		start, end = m.groups[2*index], m.groups[2*index+1]
		if start < 0 {
			return nil
		}
	}
	units := m.units
	if units == nil {
		units = utf16Units(m.Input)
	}
	s := StringFromCharCodes(units[start:end])
	return &s
}

// RegExpOptions are RegExp's named parameters (CaseSensitive nil: true).
type RegExpOptions struct {
	MultiLine     bool
	CaseSensitive *bool
	Unicode       bool
	DotAll        bool
}

// RegExp is dart:core's RegExp. Build it with NewRegExp; it is a Pattern.
type RegExp struct {
	source          string
	isMultiLine     bool
	isCaseSensitive bool
	isUnicode       bool
	isDotAll        bool
	prog            *syntax.Prog
	numCap          int
}

// NewRegExp is RegExp(source, {multiLine, caseSensitive, unicode,
// dotAll}). An invalid (or unsupported, see above) source panics with a
// FormatException, as Dart's constructor throws.
func NewRegExp(source string, opts ...RegExpOptions) *RegExp {
	var o RegExpOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	re := &RegExp{source: source, isMultiLine: o.MultiLine, isCaseSensitive: o.CaseSensitive == nil || *o.CaseSensitive,
		isUnicode: o.Unicode, isDotAll: o.DotAll}
	translated, err := translateECMAScript(source)
	if err == nil {
		flags := syntax.Perl
		if re.isMultiLine {
			flags &^= syntax.OneLine
		}
		if !re.isCaseSensitive {
			flags |= syntax.FoldCase
		}
		if re.isDotAll {
			flags |= syntax.DotNL
		}
		var parsed *syntax.Regexp
		if parsed, err = syntax.Parse(translated, flags); err == nil {
			rewriteForECMAScript(parsed, re.isUnicode)
			re.numCap = 2 * (1 + parsed.MaxCap())
			re.prog, err = syntax.Compile(parsed.Simplify())
		}
	}
	if err != nil {
		panic(NewFormatException("Invalid regular expression: " + err.Error()))
	}
	return re
}

// Pattern is the source (Dart's `pattern` getter).
func (re *RegExp) Pattern() string      { return re.source }
func (re *RegExp) IsMultiLine() bool     { return re.isMultiLine }
func (re *RegExp) IsCaseSensitive() bool { return re.isCaseSensitive }
func (re *RegExp) IsUnicode() bool       { return re.isUnicode }
func (re *RegExp) IsDotAll() bool        { return re.isDotAll }

// String is RegExp's toString: RegExp/<source>/<flags>.
func (re *RegExp) String() string {
	flags := ""
	if re.isMultiLine {
		flags += "m"
	}
	if !re.isCaseSensitive {
		flags += "i"
	}
	if re.isUnicode {
		flags += "u"
	}
	if re.isDotAll {
		flags += "s"
	}
	return "RegExp/" + re.source + "/" + flags
}

// MatchString is Pattern's test (hasMatch).
func (re *RegExp) MatchString(s string) bool { return re.FirstMatch(s) != nil }

// HasMatch is hasMatch(input).
func (re *RegExp) HasMatch(input string) bool { return re.FirstMatch(input) != nil }

// FirstMatch is firstMatch(input) (nil: null).
func (re *RegExp) FirstMatch(input string) *Match {
	in := re.newInput(input)
	return re.matchFrom(in, 0, false)
}

// StringMatch is stringMatch(input): the first match's text, nil if none.
func (re *RegExp) StringMatch(input string) *string {
	if m := re.FirstMatch(input); m != nil {
		return m.Group(0)
	}
	return nil
}

// MatchAsPrefix is matchAsPrefix(string, [start = 0]).
func (re *RegExp) MatchAsPrefix(s string, start ...int) *Match {
	in := re.newInput(s)
	st := 0
	if len(start) > 0 {
		st = start[0]
	}
	if st < 0 || st > len(in.units) {
		panic(NewRangeErrorRange(st, 0, len(in.units), "start", ""))
	}
	return re.matchFrom(in, in.charIndex(st), true)
}

// AllMatches is allMatches(input, [start = 0]).
func (re *RegExp) AllMatches(input string, start ...int) []Match {
	in := re.newInput(input)
	st := 0
	if len(start) > 0 {
		st = start[0]
	}
	if st < 0 || st > len(in.units) {
		panic(NewRangeErrorRange(st, 0, len(in.units), "start", ""))
	}
	var out []Match
	next := st
	for next <= len(in.units) {
		m := re.matchFrom(in, in.charIndex(next), false)
		if m == nil {
			break
		}
		out = append(out, *m)
		next = m.End
		if m.Start == m.End {
			// Empty match: resume one character later (a surrogate pair is
			// one character in Unicode mode).
			if re.isUnicode && next+1 < len(in.units) && isLead(in.units[next]) && isTrail(in.units[next+1]) {
				next++
			}
			next++
		}
	}
	return out
}

func isLead(u uint16) bool  { return u >= 0xD800 && u < 0xDC00 }
func isTrail(u uint16) bool { return u >= 0xDC00 && u < 0xE000 }

// regexpInput is the input as the matcher's characters (code units, or
// code points in Unicode mode) with their UTF-16 offsets.
type regexpInput struct {
	s       string
	units   []uint16
	chars   []rune
	offsets []int // offsets[i]: UTF-16 index of chars[i]; offsets[len(chars)] = len(units)
}

func (re *RegExp) newInput(s string) *regexpInput {
	in := &regexpInput{s: s, units: utf16Units(s)}
	for i := 0; i < len(in.units); i++ {
		in.offsets = append(in.offsets, i)
		u := in.units[i]
		if re.isUnicode && isLead(u) && i+1 < len(in.units) && isTrail(in.units[i+1]) {
			in.chars = append(in.chars, utf16.DecodeRune(rune(u), rune(in.units[i+1])))
			i++
			continue
		}
		in.chars = append(in.chars, rune(u))
	}
	in.offsets = append(in.offsets, len(in.units))
	return in
}

// charIndex is the index of the character containing UTF-16 index i (in
// Unicode mode a search starting inside a surrogate pair starts at the
// pair, as Dart's does).
func (in *regexpInput) charIndex(i int) int {
	for c := len(in.offsets) - 1; c >= 0; c-- {
		if in.offsets[c] <= i {
			return c
		}
	}
	return 0
}

func isLineTerminator(r rune) bool { return r == '\n' || r == '\r' || r == 0x2028 || r == 0x2029 }

func isWordChar(r rune) bool {
	return r == '_' || ('0' <= r && r <= '9') || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
}

func (in *regexpInput) context(pos int) syntax.EmptyOp {
	r1, r2 := rune(-1), rune(-1)
	if pos > 0 {
		r1 = in.chars[pos-1]
	}
	if pos < len(in.chars) {
		r2 = in.chars[pos]
	}
	var op syntax.EmptyOp
	if r1 < 0 {
		op |= syntax.EmptyBeginText | syntax.EmptyBeginLine
	} else if isLineTerminator(r1) {
		op |= syntax.EmptyBeginLine
	}
	if r2 < 0 {
		op |= syntax.EmptyEndText | syntax.EmptyEndLine
	} else if isLineTerminator(r2) {
		op |= syntax.EmptyEndLine
	}
	if isWordChar(r1) != isWordChar(r2) {
		op |= syntax.EmptyWordBoundary
	} else {
		op |= syntax.EmptyNoWordBoundary
	}
	return op
}

type btJob struct {
	pc  uint32
	arg bool
	pos int
}

// matchFrom finds the leftmost-first match starting at character pos or
// later (only at pos when anchored).
func (re *RegExp) matchFrom(in *regexpInput, pos int, anchored bool) *Match {
	n := len(in.chars)
	visited := make([]uint32, (len(re.prog.Inst)*(n+1)+31)/32)
	caps := make([]int, re.numCap)
	matchcap := make([]int, re.numCap)
	for start := pos; start <= n; start++ {
		for i := range caps {
			caps[i] = -1
		}
		caps[0] = start
		if re.tryBacktrack(in, visited, caps, matchcap, uint32(re.prog.Start), start) {
			m := &Match{Input: in.s, Pattern: re, Start: in.offsets[matchcap[0]], End: in.offsets[matchcap[1]], units: in.units}
			m.groups = make([]int, re.numCap)
			for i, c := range matchcap {
				if c < 0 {
					m.groups[i] = -1
				} else {
					m.groups[i] = in.offsets[c]
				}
			}
			return m
		}
		if anchored {
			break
		}
	}
	return nil
}

// tryBacktrack is Go's regexp backtracker (backtrack.go) for
// leftmost-first matching, over the input's characters.
func (re *RegExp) tryBacktrack(in *regexpInput, visited []uint32, caps, matchcap []int, pc0 uint32, pos0 int) bool {
	n := len(in.chars)
	jobs := []btJob{{pc: pc0, pos: pos0}}
	shouldVisit := func(pc uint32, pos int) bool {
		k := uint(int(pc)*(n+1) + pos)
		if visited[k/32]&(1<<(k&31)) != 0 {
			return false
		}
		visited[k/32] |= 1 << (k & 31)
		return true
	}
	for len(jobs) > 0 {
		j := jobs[len(jobs)-1]
		jobs = jobs[:len(jobs)-1]
		pc, pos, arg := j.pc, j.pos, j.arg
		skipCheck := true
		for {
			if !skipCheck && !shouldVisit(pc, pos) {
				break
			}
			skipCheck = false
			inst := &re.prog.Inst[pc]
			advance := false
			switch inst.Op {
			case syntax.InstFail:
			case syntax.InstAlt, syntax.InstAltMatch:
				if arg {
					arg = false
					pc = inst.Arg
				} else {
					jobs = append(jobs, btJob{pc: pc, pos: pos, arg: true})
					pc = inst.Out
				}
				advance = true
			case syntax.InstRune, syntax.InstRune1, syntax.InstRuneAny, syntax.InstRuneAnyNotNL:
				if pos < n && inst.MatchRune(in.chars[pos]) {
					pos++
					pc = inst.Out
					advance = true
				}
			case syntax.InstCapture:
				if arg {
					caps[inst.Arg] = pos
				} else {
					if int(inst.Arg) < len(caps) {
						jobs = append(jobs, btJob{pc: pc, pos: caps[inst.Arg], arg: true})
						caps[inst.Arg] = pos
					}
					pc = inst.Out
					advance = true
				}
			case syntax.InstEmptyWidth:
				if syntax.EmptyOp(inst.Arg)&^in.context(pos) == 0 {
					pc = inst.Out
					advance = true
				}
			case syntax.InstNop:
				pc = inst.Out
				advance = true
			case syntax.InstMatch:
				caps[1] = pos
				copy(matchcap, caps)
				return true
			}
			if !advance {
				break
			}
		}
	}
	return false
}

// translateECMAScript rewrites the ECMAScript escapes RE2 spells
// differently: \uXXXX and \u{X...} become \x{...}; \s and \S become
// ECMAScript's white space class (\S inside a class is not supported).
func translateECMAScript(source string) (string, error) {
	const ws = `\t\n\x{0B}\f\r \x{A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}\x{FEFF}`
	var b strings.Builder
	inClass := false
	for i := 0; i < len(source); i++ {
		c := source[i]
		switch {
		case isWTF8Surrogate(source, i):
			// A lone surrogate (WTF-8): RE2 parses UTF-8, so escape it.
			u := 0xD000 | int(source[i+1]&0x3F)<<6 | int(source[i+2]&0x3F)
			b.WriteString(`\x{` + strconv.FormatInt(int64(u), 16) + `}`)
			i += 2
		case c == '\\' && i+1 < len(source):
			next := source[i+1]
			switch {
			case next == 'u' && i+2 < len(source) && source[i+2] == '{':
				end := strings.IndexByte(source[i:], '}')
				if end < 0 {
					return "", &syntax.Error{Code: syntax.ErrInvalidEscape, Expr: source[i:]}
				}
				b.WriteString(`\x{` + source[i+3:i+end] + `}`)
				i += end
			case next == 'u' && i+5 < len(source) && isHex4(source[i+2:i+6]):
				b.WriteString(`\x{` + source[i+2:i+6] + `}`)
				i += 5
			case next == 's':
				if inClass {
					b.WriteString(ws)
				} else {
					b.WriteString("[" + ws + "]")
				}
				i++
			case next == 'S':
				if inClass {
					return "", &syntax.Error{Code: syntax.ErrInvalidCharRange, Expr: `\S`}
				}
				b.WriteString("[^" + ws + "]")
				i++
			default:
				b.WriteByte(c)
				b.WriteByte(next)
				i++
			}
		case c == '[' && !inClass && strings.HasPrefix(source[i:], "[]"):
			// ECMAScript: [] matches nothing; RE2 reads a leading ] as a
			// literal.
			b.WriteString(`[^\x{0}-\x{10FFFF}]`)
			i++
		case c == '[' && !inClass && strings.HasPrefix(source[i:], "[^]"):
			// ECMAScript: [^] matches any character.
			b.WriteString(`[\x{0}-\x{10FFFF}]`)
			i += 2
		case c == '[' && !inClass:
			inClass = true
			b.WriteByte(c)
			if i+1 < len(source) && source[i+1] == '^' {
				b.WriteByte('^')
				i++
			}
		case c == ']' && inClass:
			inClass = false
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), nil
}

func isHex4(s string) bool {
	if len(s) != 4 {
		return false
	}
	_, err := strconv.ParseUint(s, 16, 16)
	return err == nil
}

// rewriteForECMAScript makes `.` exclude ECMAScript's line terminators
// and, in non-Unicode mode, splits non-BMP literals into surrogates.
func rewriteForECMAScript(re *syntax.Regexp, unicode bool) {
	switch re.Op {
	case syntax.OpAnyCharNotNL:
		re.Op = syntax.OpCharClass
		re.Rune = []rune{0, 0x09, 0x0B, 0x0C, 0x0E, 0x2027, 0x202A, 0x10FFFF}
	case syntax.OpLiteral:
		if !unicode {
			var out []rune
			for _, r := range re.Rune {
				if r > 0xFFFF {
					r1, r2 := utf16.EncodeRune(r)
					out = append(out, r1, r2)
				} else {
					out = append(out, r)
				}
			}
			re.Rune = out
		}
	}
	for _, sub := range re.Sub {
		rewriteForECMAScript(sub, unicode)
	}
}

// AllMatches is Pattern.allMatches(string, [start = 0]): a String
// pattern's occurrences (an empty pattern matches at every index), a
// RegExp's matches (see above), or a *regexp.Regexp's (Go's iteration,
// with UTF-16 offsets).
func AllMatches(p Pattern, s string, start ...int) []Match {
	st := 0
	if len(start) > 0 {
		st = start[0]
	}
	switch p := p.(type) {
	case *RegExp:
		return p.AllMatches(s, st)
	case StringPattern:
		units := utf16Units(s)
		pat := utf16Units(string(p))
		if st < 0 || st > len(units) {
			panic(NewRangeErrorRange(st, 0, len(units), "start", ""))
		}
		var out []Match
		index := st
		for index+len(pat) <= len(units) {
			found := -1
			for i := index; i+len(pat) <= len(units); i++ {
				if equalUnits(units[i:i+len(pat)], pat) {
					found = i
					break
				}
			}
			if found < 0 {
				break
			}
			end := found + len(pat)
			out = append(out, Match{Input: s, Pattern: p, Start: found, End: end, units: units})
			if end == index {
				end++ // Empty match, don't start at the same location again.
			}
			index = end
		}
		return out
	case *regexp.Regexp:
		var out []Match
		units := utf16Units(s)
		for _, loc := range p.FindAllStringIndex(s, -1) {
			a, b := StringLength(s[:loc[0]]), StringLength(s[:loc[1]])
			if a >= st {
				out = append(out, Match{Input: s, Pattern: p, Start: a, End: b, units: units})
			}
		}
		return out
	}
	panic(ArgumentError{Message: "unsupported Pattern", Value: p, HasValue: true})
}

func equalUnits(a, b []uint16) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
