package openapi

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// initialisms are rendered upper-case when they form a whole word, so generated
// identifiers read the way a Go author would write them (UserID, not UserId).
var initialisms = map[string]string{
	"id":   "ID",
	"url":  "URL",
	"http": "HTTP",
	"api":  "API",
	"uri":  "URI",
	"uuid": "UUID",
	"json": "JSON",
}

// goKeywords are reserved words that must never be emitted as bare identifiers.
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// Exported converts an arbitrary spec name into a stable, exported Go identifier
// in PascalCase, preserving known initialisms.
func Exported(s string) string {
	words := splitWords(s)
	var b strings.Builder
	for _, w := range words {
		b.WriteString(pascalWord(w))
	}
	out := b.String()
	if out == "" {
		return "Generated"
	}
	if r, _ := utf8.DecodeRuneInString(out); !unicode.IsLetter(r) {
		out = "X" + out
	}
	return out
}

// fieldName produces an exported field identifier (same rules as Exported).
func fieldName(s string) string { return Exported(s) }

// paramGoName produces an exported identifier for a request-struct field bound
// from a parameter. Exported names are inherently keyword-safe (keywords are all
// lower-case); the guard stays as a defensive backstop.
func paramGoName(s string) string {
	n := Exported(s)
	if goKeywords[n] {
		n += "Param"
	}
	return n
}

// FileStem converts a name into a lower_snake_case file stem (without extension).
func FileStem(s string) string { return snake(s) }

// snake converts a name into a lower_snake_case file stem.
func snake(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "_")
}

// splitWords breaks an identifier on separators and camelCase / acronym
// boundaries, e.g. "getUserByID" -> [get User By ID], "user_id" -> [user id].
func splitWords(s string) []string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = nil
		}
	}
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			flush()
		case unicode.IsUpper(r):
			if i > 0 {
				prev := runes[i-1]
				if unicode.IsLower(prev) || unicode.IsDigit(prev) {
					flush()
				} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
					flush()
				}
			}
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return words
}

func pascalWord(w string) string {
	if up, ok := initialisms[strings.ToLower(w)]; ok {
		return up
	}
	r := []rune(strings.ToLower(w))
	if len(r) == 0 {
		return ""
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// refName extracts the schema name from a local $ref like
// "#/components/schemas/User".
func refName(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}
