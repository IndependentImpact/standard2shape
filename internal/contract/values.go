package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	VersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
	DigestPattern  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

func IsIRI(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs()
}

func IsVersion(value string) bool {
	return VersionPattern.MatchString(value)
}

func IsDigest(value string) bool {
	return DigestPattern.MatchString(value)
}

func IsNormalizedPath(value string) bool {
	return value != "" && !strings.Contains(value, "\\") && !strings.HasPrefix(value, "/") &&
		path.Clean(value) == value && value != "." && value != ".." && !strings.HasPrefix(value, "../")
}

// IsTimestamp accepts RFC 3339 instants in UTC only, so equal instants have
// equal serialized bytes.
func IsTimestamp(value string) bool {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return err == nil && strings.HasSuffix(value, "Z") && parsed.UTC().Equal(parsed)
}

// IsBlank mirrors the schemas' `"pattern": "\\S"` under the ECMA-262 regex
// semantics JSON Schema specifies: a string is blank when every rune is in
// ECMA's WhiteSpace or LineTerminator classes (which include NBSP and BOM but
// not NEL, so unicode.IsSpace alone would diverge in both directions).
func IsBlank(value string) bool {
	return strings.TrimFunc(value, isECMAWhitespace) == ""
}

func isECMAWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', '\u00a0', '\ufeff', '\u2028', '\u2029':
		return true
	}
	return unicode.Is(unicode.Zs, r)
}

func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
