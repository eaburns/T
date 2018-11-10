// Package re1 implements a very simple regular expression language.
// The language is inspired by plan9 regular expressions
// (https://9fans.github.io/plan9port/man/man7/regexp.html),
// rsc's regexp blog posts (https://swtch.com/~rsc/regexp),
// and nominally by the much more sohpistocated RE2 library
// (https://github.com/google/re2).
//
// The grammar is:
// 	regexp = alternate.
// 	alternate = concat [ "|" alternate ].
// 	concat = repeat [ concat ].
// 	repeat = term { "*" | "+" | "?" }.
// 	term = "." | "^" | "$" | "(" regexp ")" | charclass | literal.
// 	charclass = "[" [ "^" ] ( classlit [ "-" classlit ] ) { classlit [ "-" classlit ] } "]".
// 	A literal is any non-meta rune or a rune preceded by \.
// 	A classlit is any non-"]", non-"-" rune or a rune preceded by \.
//
// The meta characters are:
// 	| alternation
// 	* zero or more, greedy
// 	+ one or more, greedy
// 	? zero or one
// 	. any non-newline rune
// 	^ beginning of file or line
// 	$ end of file or line
// 	() capturing group
// 	[] character class (^ negates, - is a range)
// 	\n newline
// 	\t tab
// 	\ otherwise is the literal of the following rune
// 	  or is \ itself if there is no following rune.
package re1

import (
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

// Regexp is a compiled regular expression.
type Regexp struct {
	prog   []instr
	ncap   int
	class  [][][2]rune
	source string
}

// Opts are compile-time options. The zero value is default.
type Opts struct {
	// Reverse compiles the expression for reverse matching.
	// This swaps the order of concatenations, and it swaps ^ and $.
	Reverse bool
	// Delimiter specifies a rune that delimits parsing if unescaped.
	Delimiter rune
}

// New compiles a regular expression.
// The expression is terminated by the end of the string,
// an un-escaped newline,
// or an un-escaped delimiter (if set in opts).
func New(t string, opts Opts) (*Regexp, string, error) {
	src := t
	switch re, t, err := alternate(t, 0, opts); {
	case err != nil:
		return nil, "", err
	case re == nil:
		re = &Regexp{}
		fallthrough
	default:
		re = groupProg(re)
		re.prog = append(re.prog, instr{op: match})
		re.source = src
		return re, t, nil
	}
}

const (
	match = -iota
	any
	bol
	eol
	class  // arg is class index
	nclass // arg is class index
	jmp    // arg is jump offset
	fork   // arg is low-priority fork offset (high-priority is 1)
	rfork  // arg is high-priority fork offset (low-priority is 1)
	save   // arg is save-to index
)

type instr struct {
	op  int
	arg int
}

func alternate(t0 string, depth int, opts Opts) (*Regexp, string, error) {
	switch left, t, err := concat(t0, depth, opts); {
	case err != nil:
		return nil, "", err
	case peek(t) != '|':
		return left, t, nil
	case left == nil:
		return nil, "", errors.New("unexpected |")
	default:
		_, t = next(t) // eat |
		var right *Regexp
		if right, t, err = alternate(t, depth, opts); err != nil {
			return nil, "", err
		}
		return altProg(left, right), t, nil
	}
}

func concat(t string, depth int, opts Opts) (*Regexp, string, error) {
	left, t, err := repeat(t, depth, opts)
	if left == nil || err != nil {
		return left, t, err
	}
	var right *Regexp
	switch right, t, err = concat(t, depth, opts); {
	case err != nil:
		return nil, "", err
	case right != nil:
		left = catProg(left, right, opts.Reverse)
		fallthrough
	default:
		return left, t, err
	}
}

func repeat(t string, depth int, opts Opts) (*Regexp, string, error) {
	left, t, err := term(t, depth, opts)
	if left == nil || err != nil {
		return left, t, err
	}
	for strings.ContainsRune("*+?", peek(t)) {
		var r rune
		r, t = next(t)
		left = repProg(left, r)
	}
	return left, t, nil
}

func term(t0 string, depth int, opts Opts) (*Regexp, string, error) {
	switch r, t := next(t0); r {
	case eof, '\n', opts.Delimiter:
		return nil, t, nil
	case '\\':
		r, t = esc(t)
		fallthrough
	default:
		return opProg(int(r)), t, nil
	case '.':
		return opProg(any), t, nil
	case '^':
		if opts.Reverse {
			return opProg(eol), t, nil
		}
		return opProg(bol), t, nil
	case '$':
		if opts.Reverse {
			return opProg(bol), t, nil
		}
		return opProg(eol), t, nil
	case '(':
		return group(t, depth, opts)
	case '[':
		return charclass(t)
	case '|':
		return nil, t0, nil
	case ')':
		if depth == 0 {
			return nil, t, errors.New("unopened )")
		}
		return nil, t0, nil
	case '*', '+', '?':
		return nil, "", errors.New("unexpected " + string([]rune{r}))
	}
}

func group(t string, depth int, opts Opts) (*Regexp, string, error) {
	left, t, err := alternate(t, depth+1, opts)
	switch r, t := next(t); {
	case err != nil:
		return nil, "", err
	case r != ')':
		return nil, "", errors.New("unclosed (")
	case left == nil:
		left = &Regexp{}
		fallthrough
	default:
		return groupProg(left), t, nil
	}
}

func charclass(t string) (*Regexp, string, error) {
	op := class
	if peek(t) == '^' {
		_, t = next(t) // eat ^
		op = nclass
	}
	var r, p rune
	var cl [][2]rune
	for len(t) > 0 {
		switch r, t = next(t); r {
		case ']':
			if p != 0 {
				cl = append(cl, [2]rune{p, p})
			}
			if len(cl) == 0 {
				return nil, "", errors.New("empty charclass")
			}
			return charClassProg(op, cl), t, nil
		case '-':
			if p == 0 || peek(t) == ']' || peek(t) == '-' {
				return nil, "", errors.New("bad range")
			}
			r, t = next(t)
			if r == '\\' {
				r, t = esc(t)
			}
			if p >= r {
				return nil, "", errors.New("bad range")
			}
			cl = append(cl, [2]rune{p, r})
			p = 0
		case '\\':
			r, t = esc(t)
			fallthrough
		default:
			if p != 0 {
				cl = append(cl, [2]rune{p, p})
			}
			p = r
		}
	}
	return nil, "", errors.New("unclosed [")
}

func altProg(left, right *Regexp) *Regexp {
	prog := make([]instr, 0, 2+len(left.prog)+len(right.prog))
	prog = append(prog, instr{op: fork, arg: len(left.prog) + 2})
	prog = append(prog, left.prog...)
	left.prog = append(prog, instr{op: jmp, arg: len(right.prog) + 1})
	return catProg(left, right, false)
}

func catProg(left, right *Regexp, rev bool) *Regexp {
	for i := range right.prog {
		instr := &right.prog[i]
		if instr.op == save {
			instr.arg += left.ncap * 2
		}
	}
	if rev {
		left, right = right, left
	}
	for _, instr := range right.prog {
		if instr.op == class || instr.op == nclass {
			instr.arg += len(left.class)
		}
		left.prog = append(left.prog, instr)
	}
	left.ncap += right.ncap
	left.class = append(left.class, right.class...)
	return left
}

func repProg(left *Regexp, op rune) *Regexp {
	prog := make([]instr, 0, len(left.prog)+2)
	switch op {
	case '+':
		prog = append(left.prog, instr{op: rfork, arg: -len(left.prog)})
	case '*':
		prog = []instr{instr{op: fork, arg: len(left.prog) + 2}}
		prog = append(prog, left.prog...)
		prog = append(prog, instr{op: rfork, arg: -len(prog) + 1})
	case '?':
		prog = []instr{instr{op: fork, arg: len(left.prog) + 1}}
		prog = append(prog, left.prog...)
	}
	left.prog = prog
	return left
}

func groupProg(left *Regexp) *Regexp {
	prog := make([]instr, 0, len(left.prog)+2)
	prog = append(prog, instr{op: save, arg: 0})
	for _, instr := range left.prog {
		if instr.op == save {
			instr.arg += 2
		}
		prog = append(prog, instr)
	}
	left.prog = append(prog, instr{op: save, arg: 1})
	left.ncap++
	return left
}

func charClassProg(op int, cl [][2]rune) *Regexp {
	return &Regexp{prog: []instr{{op: op}}, class: [][][2]rune{cl}}
}

func opProg(op int) *Regexp { return &Regexp{prog: []instr{{op: op}}} }

const eof = -1

func next(t string) (rune, string) {
	if len(t) == 0 {
		return eof, ""
	}
	r, w := utf8.DecodeRuneInString(t)
	return r, t[w:]
}

func peek(t string) rune {
	r, _ := next(t)
	return r
}

func esc(t string) (rune, string) {
	var r rune
	switch r, t = next(t); r {
	case eof:
		r = '\\'
	case 'n':
		r = '\n'
	case 't':
		r = '\t'
	}
	return r, t
}

// Find returns the left-most, longest match and sub-expression matches.
func (re *Regexp) Find(rr io.RuneReader) []int64 {
	debug("prog:\n%s\n", re.DebugString())
	return run(newVM(re, rr))
}

type vm struct {
	re        *Regexp
	rr        io.RuneReader
	at, lim   int64
	c, n      rune    // cur and next rune.
	seen      []int64 // at for which each pc was last add()ed.
	cur, next []thread
	free      [][]int64
	match     []int64
}

type thread struct {
	pc  int
	mem []int64
}

func newVM(re *Regexp, rr io.RuneReader) *vm {
	v := &vm{re: re, rr: rr, lim: -1, c: eof, n: eof}
	v.seen = make([]int64, len(re.prog))
	for i := range v.seen {
		v.seen[i] = -1
	}
	read(v)
	return v
}

func newMem(v *vm, init []int64) (m []int64) {
	if n := len(v.free); n > 0 {
		m, v.free = v.free[n-1], v.free[:n-1]
	} else {
		m = make([]int64, 2*v.re.ncap)
	}
	if init != nil {
		copy(m, init)
		return m
	}
	for i := range m {
		m[i] = -1
	}
	return m
}

func run(v *vm) []int64 {
	debug("%2d, c=%q (%[2]x), n=%q (%[3]x)\n", v.at, v.c, v.n)
	for {
		if v.match == nil {
			add(v, 0, newMem(v, nil))
		}
		if v.lim >= 0 && v.at >= v.lim {
			// Check this after add() to allow empty regexps to match empty.
			return v.match
		}
		read(v)
		v.cur, v.next = v.next, v.cur[:0]
		debug("%2d, c=%c (%[2]x), n=%c (%[3]x)\n", v.at, v.c, v.n)
		for _, t := range v.cur {
			step(v, t.pc, t.mem)
		}
		if v.c == eof || (v.match != nil && len(v.next) == 0) {
			return v.match
		}
	}
}

func read(v *vm) {
	if v.n != eof {
		v.at += int64(utf8.RuneLen(v.n))
	}
	v.c = v.n
	var err error
	if v.n, _, err = v.rr.ReadRune(); err != nil {
		v.n = eof
	}
}

func step(v *vm, pc int, mem []int64) {
	debug("	step %d %v\n", pc, mem)
	debug("		%s\n", v.re.prog[pc].DebugString(v.re, pc))
	if !accepts(v, v.re.prog[pc]) {
		v.free = append(v.free, mem)
		return
	}
	add(v, pc+1, mem)
}

func accepts(v *vm, instr instr) bool {
	switch instr.op {
	case any:
		return v.c != '\n' && v.c != eof
	case class, nclass:
		cl := v.re.class[instr.arg]
		return classAccepts(v.c, cl, instr.op == nclass)
	default:
		return int(v.c) == instr.op
	}
}

func classAccepts(r rune, class [][2]rune, neg bool) bool {
	if r == eof {
		return false
	}
	for _, c := range class {
		if c[0] <= r && r <= c[1] {
			return !neg
		}
	}
	return neg
}

func add(v *vm, pc int, mem []int64) {
	if v.seen[pc] == v.at {
		v.free = append(v.free, mem)
		return
	}
	v.seen[pc] = v.at
	_add(v, pc, mem)
}

func _add(v *vm, pc int, mem []int64) {
	debug("		%s\n", v.re.prog[pc].DebugString(v.re, pc))
	switch instr := v.re.prog[pc]; instr.op {
	default:
		v.next = append(v.next, thread{pc: pc, mem: mem})
	case jmp:
		add(v, pc+instr.arg, mem)
	case fork:
		clone := newMem(v, mem)
		add(v, pc+1, mem)
		add(v, pc+instr.arg, clone)
	case rfork:
		clone := newMem(v, mem)
		add(v, pc+instr.arg, mem)
		add(v, pc+1, clone)
	case save:
		mem[instr.arg] = v.at
		add(v, pc+1, mem)
	case bol:
		if v.c != eof && v.c != '\n' {
			v.free = append(v.free, mem)
			return
		}
		add(v, pc+1, mem)
	case eol:
		if v.n != eof && v.n != '\n' {
			v.free = append(v.free, mem)
			return
		}
		add(v, pc+1, mem)
	case match:
		debug("			%v\n", mem)
		setMatch(v, mem)
	}
}

func setMatch(v *vm, mem []int64) {
	switch {
	case v.match == nil:
		v.match = mem
	case mem[0] <= v.match[0] && mem[1] > v.match[1]:
		v.free = append(v.free, v.match)
		v.match = mem
	default:
		v.free = append(v.free, mem)
	}
}
