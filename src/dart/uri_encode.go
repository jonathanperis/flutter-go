package dart

import "strings"

// isEncodeFullChar is the Dart SDK's _encodeFullMask: the characters
// Uri.encodeFull leaves alone (ECMAScript's encodeURI set).
func isEncodeFullChar(c byte) bool {
	return isUnreserved(c) || strings.IndexByte("!#$&'()*+,/:;=?@", c) >= 0
}

// URIEncodeFull is Uri.encodeFull: the UTF-8 bytes of s (a lone surrogate
// is U+FFFD, as utf8.encode does) with every byte outside the
// encodeFull set written as %XX (upper-case hex).
func URIEncodeFull(s string) string {
	const hexDigits = "0123456789ABCDEF"
	var b strings.Builder
	for _, c := range UTF8Encode(s) {
		if c < 128 && isEncodeFullChar(c) {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0x0f])
		}
	}
	return b.String()
}

// URIPath is `Uri(path: path).path`: the path normalized as the Uri
// constructor does without a scheme or authority (_makePath: characters
// outside the path set percent-encoded, valid escapes kept and
// upper-cased, backslashes as slashes; then dot segments removed, leading
// ".." kept, and a leading segment that looks like a scheme has its ':'
// escaped).
func URIPath(path string) string {
	p := normalizePathChars(string(UTF8Encode(path)))
	if strings.HasPrefix(p, "/") {
		return removeDotSegments(p)
	}
	// _makePath's _normalizePath, then the constructor's
	// _normalizeRelativePath (both with allowScheme false).
	p = escapedRelativePath(p)
	return escapedRelativePath(p)
}

func escapedRelativePath(p string) string {
	if !mayContainDotSegments(p) {
		return escapeScheme(p)
	}
	p = normalizeRelativePath(p)
	first, rest, found := strings.Cut(p, "/")
	first = escapeScheme(first)
	if found {
		return first + "/" + rest
	}
	return first
}

// escapeScheme is _Uri._escapeScheme: a path whose start reads as a scheme
// ("a:b") gets that colon escaped (%3A).
func escapeScheme(path string) string {
	if len(path) >= 2 && isSchemeStart(path[0]) {
		for i := 1; i < len(path); i++ {
			c := path[i]
			if c == ':' {
				return path[:i] + "%3A" + path[i+1:]
			}
			if c > 127 || !(isSchemeStart(c) || c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.') {
				break
			}
		}
	}
	return path
}
