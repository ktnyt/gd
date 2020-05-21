package utils

import "strings"

// FlatFileSplit splits the string with the flatfile convention.
func FlatFileSplit(s string) []string {
	s = strings.TrimSuffix(s, ".")
	return strings.Split(s, "; ")
}

// AddPrefix adds the given prefix after each newline.
func AddPrefix(s, prefix string) string {
	return strings.Replace(s, "\n", "\n"+prefix, -1)
}