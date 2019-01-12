package re1

// Union returns a single regular expression that matches the union of a set of regular expressions.
// The Union of no regexps is nil.
//
// The last element of the slice returned by a call to Find will be the ID (Opts.ID) of the component expression that matched.
//
// The capture groups are numbered with respect to their corresponding numbers for the matched component regexp.
// For example, Union("(a)bc", "(d)ef") will return a match for "a" as capture group 1 if component expression "(a)bc" matches.
// However it will return a match for "d" as capture group 1 if component expression "(d)ef" matches.
func Union(res ...*Regexp) *Regexp {
	switch len(res) {
	case 0:
		return nil
	case 1:
		return res[0]
	}

	left := new(Regexp)
	*left = *res[0]
	left.prog = append([]instr{}, left.prog...)
	left.class = append([][][2]rune{}, left.class...)
	// We use non-capturing group syntax here
	// even though it's not actually supported by re1.
	// But source is only used for debugging,
	// and using a capturing group would be incorrectly,
	// because the capture numbers would be wrong.
	left.source = "(?:" + left.source + ")"
	for _, right := range res[1:] {
		left = union2(left, right)
		left.source += "|(?:" + right.source + ")"
	}
	return left
}

// union2 is like catProg, but
// 	it doesn't re-number capture groups,
// 	it doesn't reverse, and
// 	it sets the capture count to the max, not the sum.
func union2(left, right *Regexp) *Regexp {
	prog := make([]instr, 0, 2+len(left.prog)+len(right.prog))
	prog = append(prog, instr{op: fork, arg: len(left.prog) + 2})
	prog = append(prog, left.prog...)
	left.prog = append(prog, instr{op: jmp, arg: len(right.prog) + 1})
	for _, instr := range right.prog {
		if instr.op == class || instr.op == nclass {
			instr.arg += len(left.class)
		}
		left.prog = append(left.prog, instr)
	}
	left.class = append(left.class, right.class...)
	if right.ncap > left.ncap {
		left.ncap = right.ncap
	}
	return left
}
