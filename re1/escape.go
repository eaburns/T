package re1

import "strings"

// Escape returns the argument with any meta-characters escaped.
func Escape(t string) string {
	var s strings.Builder
	for _, r := range t {
		if strings.ContainsRune(`|*+?.^$()[]\`, r) {
			s.WriteRune('\\')
		}
		s.WriteRune(r)
	}
	return s.String()
}
