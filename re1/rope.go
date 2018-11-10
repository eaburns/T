package re1

import (
	"github.com/eaburns/T/rope"
)

// FindInRope returns the left-most, longest match of a regulax expression
// between byte offsets s (inclusive) and e (exclusive) in a rope.
func (re *Regexp) FindInRope(ro rope.Rope, s, e int64) []int64 {
	sl := rope.Slice(ro, s, ro.Len())
	rr := rope.NewReader(sl)
	v := newVM(re, rr)
	v.c = prevRune(ro, s)
	v.at, v.lim = s, e
	return run(v)
}

func prevRune(ro rope.Rope, i int64) rune {
	sl := rope.Slice(ro, 0, i)
	rr := rope.NewReverseReader(sl)
	r, _, err := rr.ReadRune()
	if err != nil {
		return eof
	}
	return r
}

// FindReverseInRope returns the right-most, longest match
// of a reverse-compiled regulax expression
// between byte offsets s (inclusive) and e (exclusive) in a rope.
//
// The receiver is assumed to be compiled for a reverse match.
func (re *Regexp) FindReverseInRope(ro rope.Rope, s, e int64) []int64 {
	sl := rope.Slice(ro, 0, e)
	rr := rope.NewReverseReader(sl)
	v := newVM(re, rr)
	v.c = nextRune(ro, e)
	v.lim = e - s
	ms := run(v)
	for i := 0; i < len(ms); i += 2 {
		if ms[i] >= 0 {
			ms[i], ms[i+1] = e-ms[i+1], e-ms[i]
		}
	}
	return ms
}

func nextRune(ro rope.Rope, i int64) rune {
	sl := rope.Slice(ro, i, ro.Len())
	rr := rope.NewReader(sl)
	r, _, err := rr.ReadRune()
	if err != nil {
		return eof
	}
	return r
}
