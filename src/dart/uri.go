package dart

import (
	"fmt"
	"net/url"
	"strings"
)

// URI is the part of dart:core's Uri that Flutter's stack-frame parser
// reads: the scheme and the path (with Dart's normalization).
type URI struct {
	// Scheme is lower-cased; "" when there is none.
	Scheme string
	// Path is percent-encoded, with dot segments removed.
	Path string
	// HasAuthority is true when the URI has a "//" authority part.
	HasAuthority bool
}

// ParseURI is the subset of Uri.parse that StackFrame uses: scheme,
// authority (skipped), path, query and fragment (dropped). Path
// normalization follows the Dart SDK's _makePath and _normalizePath: invalid
// characters are percent-encoded (valid escapes are kept, upper-cased), a
// backslash is a slash, file: paths get a leading slash, and dot segments
// are removed (leading ".." kept in a relative path without a scheme).
// Unlike Dart it never throws: input Dart rejects (a bad port, say) parses
// leniently.
func ParseURI(s string) URI {
	var u URI
	rest := s
	if i := strings.IndexAny(s, ":/?#"); i > 0 && s[i] == ':' && isSchemeStart(s[0]) && validScheme(s[:i]) {
		u.Scheme = strings.ToLower(s[:i])
		rest = s[i+1:]
	}
	if strings.HasPrefix(rest, "//") {
		u.HasAuthority = true
		rest = rest[2:]
		end := strings.IndexAny(rest, "/?#")
		if end < 0 {
			end = len(rest)
		}
		rest = rest[end:]
	}
	if end := strings.IndexAny(rest, "?#"); end >= 0 {
		rest = rest[:end]
	}
	path := normalizePathChars(rest)
	if path == "" {
		if u.Scheme == "file" {
			path = "/"
		}
	} else if (u.Scheme == "file" || u.HasAuthority) && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if u.Scheme == "" && !u.HasAuthority && !strings.HasPrefix(path, "/") {
		u.Path = normalizeRelativePath(path)
	} else {
		u.Path = removeDotSegments(path)
	}
	return u
}

// PathSegments is Uri.pathSegments: the path without its leading slash,
// split at slashes, each segment percent-decoded.
func (u URI) PathSegments() []string {
	p := strings.TrimPrefix(u.Path, "/")
	if p == "" {
		return []string{}
	}
	segments := strings.Split(p, "/")
	for i, s := range segments {
		if d, err := url.PathUnescape(s); err == nil {
			segments[i] = d
		}
	}
	return segments
}

func isSchemeStart(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func validScheme(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(isSchemeStart(c) || c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.') {
			return false
		}
	}
	return true
}

func isHex(c byte) bool { return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' }

func isUnreserved(c byte) bool {
	return isSchemeStart(c) || c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_' || c == '~'
}

// isPathChar is Dart's _pathCharOrSlashMask: unreserved, sub-delims, ':',
// '@' and '/'.
func isPathChar(c byte) bool {
	return isUnreserved(c) || strings.IndexByte("!$&'()*+,;=:@/", c) >= 0
}

func normalizePathChars(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]):
			v := unhex(s[i+1])<<4 | unhex(s[i+2])
			if isUnreserved(v) {
				b.WriteByte(v)
			} else {
				fmt.Fprintf(&b, "%%%02X", v)
			}
			i += 2
		case c == '\\':
			b.WriteByte('/')
		case isPathChar(c):
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c) // UTF-8 bytes, as Dart encodes
		}
	}
	return b.String()
}

func unhex(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	}
	return c - 'A' + 10
}

func mayContainDotSegments(path string) bool {
	return strings.HasPrefix(path, ".") || strings.Contains(path, "/.")
}

// removeDotSegments is _Uri._removeDotSegments (RFC 3986, leading ".."
// dropped).
func removeDotSegments(path string) string {
	if !mayContainDotSegments(path) {
		return path
	}
	var output []string
	appendSlash := false
	for _, segment := range strings.Split(path, "/") {
		appendSlash = false
		if segment == ".." {
			if len(output) > 0 {
				output = output[:len(output)-1]
				if len(output) == 0 {
					output = append(output, "")
				}
			}
			appendSlash = true
		} else if segment == "." {
			appendSlash = true
		} else {
			output = append(output, segment)
		}
	}
	if appendSlash {
		output = append(output, "")
	}
	return strings.Join(output, "/")
}

// normalizeRelativePath is _Uri._normalizeRelativePath (leading ".." kept).
func normalizeRelativePath(path string) string {
	if !mayContainDotSegments(path) {
		return path
	}
	var output []string
	appendSlash := false
	for _, segment := range strings.Split(path, "/") {
		if segment == ".." {
			appendSlash = true
			if len(output) > 0 && output[len(output)-1] != ".." {
				output = output[:len(output)-1]
			} else {
				output = append(output, "..")
			}
		} else if segment == "." {
			appendSlash = true
		} else {
			appendSlash = false
			if segment == "" && len(output) == 0 {
				segment = "./"
			}
			output = append(output, segment)
		}
	}
	if len(output) == 0 {
		return "./"
	}
	if appendSlash {
		output = append(output, "")
	}
	return strings.Join(output, "/")
}
