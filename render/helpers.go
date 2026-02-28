package render

import (
	"regexp"
	"strings"
)

// preBlockRe matches <pre>...</pre> blocks (case-insensitive, dotall).
var preBlockRe = regexp.MustCompile(`(?is)<pre[\s\S]*?</pre>`)

// protectPreBlocks stashes <pre> blocks with placeholders while f runs,
// then restores them. This prevents filters from mangling code blocks.
func protectPreBlocks(s string, f func(string) string) string {
	var stash []string
	s = preBlockRe.ReplaceAllStringFunc(s, func(m string) string {
		stash = append(stash, m)
		return placeholder("PRE", len(stash)-1)
	})
	s = f(s)
	for i, m := range stash {
		s = strings.ReplaceAll(s, placeholder("PRE", i), m)
	}
	return s
}

func placeholder(tag string, i int) string {
	return "[[[" + tag + "_BLOCK_" + itoa(i) + "]]]"
}

func itoa(i int) string { return strconvItoa(i) }

// tiny local int->string (keeps imports tidy)
func strconvItoa(i int) string {
	var digits [20]byte
	pos := len(digits)
	n := i
	if n == 0 {
		return "0"
	}
	for n > 0 {
		pos--
		digits[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[pos:])
}

func htmlEscapeAttr(s string) string {
	s = strings.ReplaceAll(s, `"`, `&quot;`)
	s = strings.ReplaceAll(s, `&`, `&amp;`)
	s = strings.ReplaceAll(s, `<`, `&lt;`)
	s = strings.ReplaceAll(s, `>`, `&gt;`)
	return s
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
