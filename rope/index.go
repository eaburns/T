package rope

// IndexFunc returns the byte index
// of the first rune in the rope
// for which a function returns true.
// If the function never returns true,
// IndexFunc returns -1.
func IndexFunc(ro Rope, f func(r rune) bool) int64 {
	var i int64
	rr := NewReader(ro)
	for {
		r, w, err := rr.ReadRune()
		if err != nil {
			return -1
		}
		if f(r) {
			return i
		}
		i += int64(w)
	}
}

// LastIndexFunc returns the byte index
// of the last rune in the rope
// for which a function returns true.
// If the function never returns true,
// IndexFunc returns -1.
//
// LastIndexFunc traverses the rope from end to beginning.
func LastIndexFunc(ro Rope, f func(r rune) bool) int64 {
	i := ro.Len()
	rr := NewReverseReader(ro)
	for {
		r, w, err := rr.ReadRune()
		if err != nil {
			return -1
		}
		i -= int64(w)
		if f(r) {
			return i
		}
	}
}
