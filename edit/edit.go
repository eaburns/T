// Package edit implements a subset of the Sam editing language.
// It does not implement undo/redo, or multi-buffer commands.
// However, these could be easily added on top of this implementation.
//
// The langage is described below using an informal, EBNF-like style.
// Items enclosed in brackets, [ ] ,are optional, and
// items in braces, { }, may be repeated 0 or more times.
//
// Some details may differ from the original.
// See https://9fans.github.io/plan9port/man/man1/sam.html
// for the original language description.
//
// Addresses
//
// The edit language is comprised of addresses and commands.
// Addresses identify a sequence of runes in the text.
// They are described by the grammar:
// 	addr = range.
//
// 	range = [ relative ] "," [ range ] | [ relative ] ";" [ range ] | relative.
// 		[a2],[a3] is the string from the start of the address a2 to the end of a3.
// 		If the a2 is absent, 0 is used. If the a3 is absent, $ is used.
//
// 		[a2];[a3] is like the previous,
// 		but with . set to the address a2 before evaluating a3.
// 		If the a2 is absent, 0 is used. If the a3 is absent, $ is used.
//
// 	relative = [ simple ] "+" [ relative ] | [ simple ] "-" [ relative ] | simple relative | simple.
// 		[a1]+[a2] is the address a2 evaluated from the end of a1.
// 		If the a1 is absent, . is used. If the a2 is absent, 1 is used.
//
// 		[a1]-[a2] is the address a2 evaluated in reverse from the start of a1.
// 		If the a1 is absent, . is used. If the a2 is absent, 1 is used.
//
// 		a1 a2 is the same as a1+a2; the + is inserted.
//
// 	simple = "$" | "." | "#" digits | digits | "/" regexp [  "/"  ].
// 		$ is the empty string at the end of the text.
// 		. is the current address of the editor, called dot.
// 		#n is the empty string after rune number n. If n is absent then 1 is used.
// 		n is the nth line in the text. 0 is the string before the first full line.
// 		/ regexp / is the first match of the regular expression going forward.
//
// 		A regexp is an re1 regular expression delimited by / or a newline.
// 		(See https://godoc.org/github.com/eaburns/T/re1)
// 		Regexp matches wrap at the end (or beginning) of the text.
// 		The resulting match may straddle the starting point.
//
// 	All operators are left-associative.
//
// Commands
//
// Commands print text or compute diffs on the text which later can be applied.
//
// In the following, the literal "/" can be any non-space rune
// that replaces "/" consistently throughout the production.
// Such a rune acts as a user-chosen delimiter.
//
// For any command that begin with an optional, leading address,
// if the address is elided, then dot is used instead.
//
// A command is one of the following:
// 	[ addr ] ( "a" | "c" | "i" ) "/"  [ text ] [ "/" ].
// 	[ addr ] ( "a" | "c" | "i" ) "\n" lines of text "\n.\n".
// 		Appends a string after the address (a),
// 		changes the string at the address (c), or
// 		inserts a string before the address (i).
//
// 		The text can be supplied in one of two forms:
//
// 		The first form begins with a non-space rune, the delimiter.
// 		The text consists of all runes until the end of input,
// 		a non-escaped newline, or a non-escaped delimiter.
// 		Pairs of runes beginning with \ are called escapes.
// 		They are interpreted specially:
// 			\n is a newline
// 			\t is a tab
// 			\ followed by any other rune is that rune.
// 			If the rune is the delimiter, it is non-delimiting.
// 	  		\ followed by no rune (end of input) is \.
//
// 		For example:
// 			#3 a/Hello, World!/
// 		appends the string "Hello, World!" after the 3rd rune.
//
// 		The second form begins with a newline and
// 		ends with a line containing only a period, ".".
// 		In this form, escapes are not interpreted; they are literal.
// 		This form is convenient for multi-line text.
//
// 		For example:
// 			#3 a
// 			Hello,
// 			World
// 			!
// 			.
// 		appends the string "Hello,\nWorld\n!" after the 3rd rune.
//
// 	[ addr ] "d".
// 		Deletes the string at the address.
//
// 	[ addr ] "t" addr.
// 		Copies the string from the first address to after the second.
//
// 	[ addr ] "m" addr.
// 		Moves the string from the first address to after the second.
//
// 	[ addr ] "p".
// 		Prints the string at the address.
//
// 	[ addr ] "s" [ digits ] "/" regexp "/" [ substitution ] "/"  [ "g" ].
// 		Substitutes matches of a regexp within the address.
// 		As above, the regexp uses the re1 syntax.
//
// 		A number N after s indicates to substitute the Nth match
// 		of the regular expression in the address range.
// 		If n == 0, the first match is substituted.
// 		If the substitution is followed by the letter g,
// 		all matches in the address range are substituted.
// 		If both a number N and the letter g are present,
// 		the Nth match and all following  in the address range
// 		are substituted.
//
// 		Substitution text replaces matches of the regular expression.
// 		The substitution text is interpreted the same way as
// 		the delimited form of the a, c, and i commands, except that
// 		an escaped digit (0-9) is not interpreted as the digit itself
// 		but is substituted with text matched by the regexp.
// 		The digits 1-9 correspond to text matched by capturing groups
// 		numbered from left-to-right by order of their open parenthesis.
// 		The 0 is the entire match.
//
// 	[ addr ] ( "x" | "y" ) "/" regexp "/" command.
// 		Executes a command for each match of the regular expression in the address.
//
// 		Dot is set either to each match of the regular expression (x)
// 		or to the strings before, between, and after the matches (y),
// 		and the command is executed.
// 		It is an error if the resulting edits are not in ascending order.
//
// 		Note, the command is only interpreted if there is a match,
// 		so a malformed command is not reported if there is no match.
//
// 	[ addr ] ( "g" | "v" ) "/" regexp "/" command.
// 		Conditionally executes a command if a regular expression
// 		matches in the address.
//
// 		If the regular expression matches (g) or doesn't match (v)
// 		in the addressed string, dot is set to the address
// 		and the command is executed.
// 		It is an error if the resulting edits are not in ascending order.
//
// 		Note, the command is only interpreted if the condition is satisfied,
// 		so a malformed command is not reported if not.
//
// 	[ addr ] ( "|" | "<" | ">" ) shell command.
// 		Pipes the addressed string to and/or from shell commands.
//
// 		The | form pipes the addressed string to standard input
// 		of a shell command and overwrites it with the
// 		the standard output of the command.
// 		The < and > forms are like the | form,
// 		but < only overwrites with the command's standard output,
// 		and > only pipes to the command's standard input.
// 		In all cases, the standard error of the command is printed.
// 		In the case of >, the standard output is also printed.
//
// 		The shell command is any text terminated by either
// 		the end of input or an un-escaped newline.
// 		The text is passed as the -c argument of
// 		the shell program from the SHELL environment variable.
// 		If SHELL is unset, /bin/sh is used.
//
// 	[ addr ] "{" { "\n" command } [ "\n" ] [ "}" ].
// 		Performs a sequence of commands.
//
// 		Before performing each command, dot is set to the address.
// 		Commands do not see modifications made by each other.
// 		Each sees the original text, before any changes are made.
// 		It is an error if the resulting edits are not in ascending order.
package edit

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/eaburns/T/re1"
	"github.com/eaburns/T/rope"
)

// A Diffs is a sequence of Diffs.
type Diffs []Diff

// A Diff describes a single change to a contiguous span of bytes.
type Diff struct {
	// At is the byte address of the span changed.
	At [2]int64
	// Text is the text to which the span changed.
	// Text may be nil if the addressed string was deleted.
	Text rope.Rope
}

// NoCommandError is returned when there was no command to execute.
type NoCommandError struct {
	// At contains the evaluation of the address preceding the missing command.
	// An empty address is dot, so an empty edit results in a NoCommandError.
	// At is set to the value of dot.
	At [2]int64
}

// Error returns the error string "no command".
func (e NoCommandError) Error() string {
	return "no command"
}

// TextLen is the length of Text; 0 if Text is nil.
func (d Diff) TextLen() int64 {
	if d.Text == nil {
		return 0
	}
	return d.Text.Len()
}

// Update returns an updated dot accounting for the changes of the diffs.
func (ds Diffs) Update(dot [2]int64) [2]int64 {
	for _, d := range ds {
		dot = d.Update(dot)
	}
	return dot
}

// Update returns an updated dot accounting for the change of the diff.
// For example, if the diff added or deleted text before the dot,
// then dot is increased or decreased accordingly.
func (d Diff) Update(dot [2]int64) [2]int64 {
	switch delta := d.TextLen() - (d.At[1] - d.At[0]); {
	case d.At[0] >= dot[1]:
		// after dot
		break

	case d.At[1] <= dot[0]:
		// before dot
		dot[0] += delta
		dot[1] += delta

	case dot[0] <= d.At[0] && d.At[1] < dot[1]:
		// inside dot
		dot[1] += delta

	case d.At[0] <= dot[0] && dot[1] < d.At[1]:
		// over dot
		dot[0] = d.At[0]
		dot[1] = d.At[0]

	case d.At[1] < dot[1]:
		// a prefix of dot
		dot[0] = d.At[0] + d.TextLen()
		dot[1] = dot[0] + (dot[1] - d.At[1])

	default:
		// a suffix of dot
		dot[1] = d.At[0]
	}
	return dot
}

// Apply returns the result of applying the diffs
// and a new sequence of diffs that will undo the changes
// if applied to the returned rope.
func (ds Diffs) Apply(ro rope.Rope) (rope.Rope, Diffs) {
	undo := make(Diffs, len(ds))
	for i, d := range ds {
		ro, undo[len(ds)-i-1] = d.Apply(ro)
	}
	return ro, undo
}

// Apply returns the result of applying the diff
// and a new Diff that will undo the change
// if applied to the returned rope.
func (d Diff) Apply(ro rope.Rope) (rope.Rope, Diff) {
	var deleted rope.Rope
	if d.At[0] < d.At[1] {
		deleted = rope.Slice(ro, d.At[0], d.At[1])
		ro = rope.Delete(ro, d.At[0], d.At[1]-d.At[0])
	}
	if d.TextLen() > 0 {
		ro = rope.Insert(ro, d.At[0], d.Text)
	}
	return ro, Diff{At: [2]int64{d.At[0], d.At[0] + d.TextLen()}, Text: deleted}
}

// Addr computes an address using the given value for dot.
func Addr(dot [2]int64, t string, ro rope.Rope) ([2]int64, error) {
	var err error
	switch dot, t, err = addr(&dot, ro, t); {
	case err != nil:
		return [2]int64{}, err
	case strings.TrimSpace(t) != "":
		return [2]int64{}, errors.New("expected end-of-input")
	case dot[0] < 0:
		return [2]int64{}, errors.New("no address")
	default:
		return dot, nil
	}
}

// Edit computes an edit on the rope using the given value for dot.
func Edit(dot [2]int64, t string, print io.Writer, ro rope.Rope) (Diffs, error) {
	switch ds, t, err := edit(dot, t, print, ro); {
	case err != nil:
		return nil, err
	case strings.TrimSpace(t) != "":
		return nil, errors.New("expected end-of-input")
	default:
		return ds, nil
	}
}

func edit(dot [2]int64, t string, print io.Writer, ro rope.Rope) (Diffs, string, error) {
	a, t, err := addr(&dot, ro, t)
	switch {
	case err != nil:
		return nil, "", err
	case a[0] < 0:
		a = dot
	}
	switch r, t := next(trimSpaceLeft(t)); r {
	default:
		return nil, "", errors.New("bad command " + string([]rune{r}))
	case eof, '\n':
		return nil, "", NoCommandError{At: a}
	case 'a', 'c', 'd', 'i':
		return change(a, t, r, ro)
	case 'm':
		return move(dot, a, t, ro)
	case 'p':
		_, err := rope.Slice(ro, a[0], a[1]).WriteTo(print)
		return nil, t, err
	case 't':
		return copy(dot, a, t, ro)
	case 's':
		return sub(a, t, ro)
	case 'g', 'v':
		return cond(a, t, r, print, ro)
	case 'x', 'y':
		return loop(a, t, r, print, ro)
	case '{':
		return seq(a, t, print, ro)
	case '<', '>', '|':
		return pipe(a, t, r, print, ro)
	}
}

func change(a [2]int64, t string, op rune, ro rope.Rope) (Diffs, string, error) {
	switch op {
	case 'a':
		a[0] = a[1]
	case 'i':
		a[1] = a[0]
	}
	var text string
	if op != 'd' {
		text, t = parseText(t)
	}
	return Diffs{{At: a, Text: rope.New(text)}}, t, nil
}

func parseText(t0 string) (text, t string) {
	t = strings.TrimLeftFunc(t0, func(r rune) bool {
		return unicode.IsSpace(r) && r != '\n'
	})
	switch {
	case len(t) == 0:
		return "", ""
	case t[0] == '\n':
		return parseLines(t[1:])
	default:
		delim, w := utf8.DecodeRuneInString(t)
		return parseDelimited(t[w:], nil, delim)
	}
}

func parseLines(t string) (string, string) {
	var i int
	nl := true
	var s strings.Builder
	for {
		var r rune
		switch r, t = next(t); {
		case r == eof:
			if nl && i > 0 {
				s.WriteRune('\n')
			}
			return s.String(), ""
		case nl && r == '.' && len(t) == 0:
			return s.String(), t
		case nl && r == '.' && t[0] == '\n':
			return s.String(), t[1:]
		default:
			if nl && i > 0 {
				s.WriteRune('\n')
			}
			i++
			if r == '\n' {
				nl = true
			} else {
				s.WriteRune(r)
				nl = false
			}
		}
	}
}

func parseDelimited(t string, sub func(int) string, delim rune) (string, string) {
	var s strings.Builder
	for {
		var r rune
		switch r, t = next(t); {
		case r == eof || r == '\n' || r == delim:
			return s.String(), t
		case r == '\\' && sub != nil:
			if r, t1 := next(t); '0' <= r && r <= '9' {
				t = t1
				s.WriteString(sub(int(r - '0')))
				continue
			}
			fallthrough
		case r == '\\':
			r, t = next(t)
			s.WriteRune(esc(r))
		default:
			s.WriteRune(r)
		}
	}
}

func esc(r rune) rune {
	switch r {
	case eof:
		return '\\'
	case 'n', '\n':
		return '\n'
	case 't':
		return '\t'
	default:
		return r
	}
}

func move(dot, a [2]int64, t string, ro rope.Rope) (Diffs, string, error) {
	b, t, err := addr(&dot, ro, t)
	switch {
	case err != nil:
		return nil, "", err
	case b[0] < 0:
		return nil, "", errors.New("expected address")
	case a[0] == a[1]:
		// Moving nothing is a no-op,
		return nil, t, nil
	case a[0] <= b[1] && b[1] < a[1]:
		// Moving to a destination inside the moved text is a no-op,
		return nil, t, nil
	case a[1] < b[1]:
		// Moving text from before the dest, slide left by the delta
		b[1] -= a[1] - a[0]
	}
	ds := Diffs{
		{At: a, Text: nil},
		{At: [2]int64{b[1], b[1]}, Text: rope.Slice(ro, a[0], a[1])},
	}
	return ds, t, nil
}

func copy(dot, a [2]int64, t string, ro rope.Rope) (Diffs, string, error) {
	b, t, err := addr(&dot, ro, t)
	switch {
	case err != nil:
		return nil, "", err
	case a[0] == a[1]:
		// Copying nothing is a no-op,
		return nil, t, nil
	case b[0] < 0:
		return nil, "", errors.New("expected address")
	}
	return Diffs{{At: [2]int64{b[1], b[1]}, Text: rope.Slice(ro, a[0], a[1])}}, t, nil
}

func sub(a [2]int64, t string, ro rope.Rope) (Diffs, string, error) {
	n, t, _ := number(trimSpaceLeft(t))
	delim, _ := next(trimSpaceLeft(t))
	re, t, err := parseRegexp(t)
	if err != nil {
		return nil, "", err
	}
	tmpl := t
	_, t = parseDelimited(t, nil, delim)
	var global bool
	if r, t1 := next(trimSpaceLeft(t)); r == 'g' {
		t = t1
		global = true
	}

	var ds Diffs
	var adj int64
	var ms []int64
	sub := func(n int) string {
		i := n * 2
		if i >= len(ms) || ms[i] < 0 {
			return ""
		}
		return rope.Slice(ro, ms[i], ms[i+1]).String()
	}
	for a[0] <= a[1] {
		if ms = re.FindInRope(ro, a[0], a[1]); ms == nil {
			break
		}
		if len(ms) > 0 {
			ms = ms[:len(ms)-1] // trim regexp ID
		}
		if ms[1] == ms[0] {
			a[0]++
		} else {
			a[0] = ms[1]
		}
		n--
		if n > 0 {
			continue
		}
		s, _ := parseDelimited(tmpl, sub, delim)
		ds = append(ds, Diff{
			At:   [2]int64{ms[0] - adj, ms[1] - adj},
			Text: rope.New(s),
		})
		if !global {
			break
		}
		adj += ms[1] - ms[0] - int64(len(s))
	}
	if len(ds) == 0 {
		return nil, "", errors.New("no match")
	}
	return ds, t, nil
}

func cond(a [2]int64, t string, op rune, print io.Writer, ro rope.Rope) (Diffs, string, error) {
	re, t, err := parseRegexp(t)
	if err != nil {
		return nil, "", err
	}
	cmd, t := splitNewline(t)
	ms := re.FindInRope(ro, a[0], a[1])
	if op == 'g' && ms == nil || op == 'v' && ms != nil {
		return nil, t, nil
	}
	ds, err := Edit(a, cmd, print, ro)
	return ds, t, err
}

func loop(a [2]int64, t string, op rune, print io.Writer, ro rope.Rope) (Diffs, string, error) {
	re, t, err := parseRegexp(t)
	if err != nil {
		return nil, "", err
	}
	cmd, t := splitNewline(t)
	var diffs Diffs
	prev := a[0]
	at := int64(-1)
	var adj int64
	for a[0] <= a[1] {
		ms := re.FindInRope(ro, a[0], a[1])
		if ms == nil {
			break
		}
		if ms[1] == ms[0] {
			a[0]++
		} else {
			a[0] = ms[1]
		}
		dot := [2]int64{ms[0], ms[1]}
		if op == 'y' {
			dot = [2]int64{prev, ms[0]}
			prev = ms[1]
		}
		var ds Diffs
		if ds, err = Edit(dot, cmd, print, ro); err != nil {
			return nil, "", err
		}
		if at, adj, diffs, err = appendAdjusted(at, adj, diffs, ds); err != nil {
			return nil, "", err
		}
	}
	if op == 'y' {
		ds, err := Edit([2]int64{prev, a[1]}, cmd, print, ro)
		if err != nil {
			return nil, "", err
		}
		if _, _, diffs, err = appendAdjusted(at, adj, diffs, ds); err != nil {
			return nil, "", err
		}
	}
	return diffs, t, nil
}

func seq(a [2]int64, t string, print io.Writer, ro rope.Rope) (Diffs, string, error) {
	var diffs Diffs
	at := int64(-1)
	var adj int64
	for {
		t = trimSpaceLeft(t)
		if r, t1 := next(t); r == '}' {
			return diffs, t1, nil
		} else if r == eof {
			return nil, "", errors.New("unclosed {")
		}

		var cmd string
		cmd, t = splitNewline(t)
		ds, err := Edit(a, cmd, print, ro)
		if err != nil {
			return nil, "", err
		}
		if at, adj, diffs, err = appendAdjusted(at, adj, diffs, ds); err != nil {
			return nil, "", err
		}
	}
}

func appendAdjusted(at, adj int64, diffs, ds Diffs) (int64, int64, Diffs, error) {
	var atNext int64
	for i := range ds {
		d := &ds[i]
		if d.At[0] < at {
			return 0, 0, nil, errors.New("out of order")
		}
		// TODO: if a single diff was a delete+add instead of delete|add,
		// then we could just update at here.
		atNext = d.At[1]

		d.At[0] += adj
		d.At[1] += adj
	}
	at = atNext

	for _, d := range ds {
		adj += d.Text.Len() - (d.At[1] - d.At[0])
	}
	diffs = append(diffs, ds...)
	return at, adj, diffs, nil
}

func pipe(a [2]int64, t string, op rune, print io.Writer, ro rope.Rope) (Diffs, string, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	arg, t := splitNewline(t)
	if arg = strings.TrimSpace(arg); arg == "" {
		return nil, "", errors.New("expected command")
	}
	cmd := exec.Command(shell, "-c", arg)
	stdin, stdout, err := openPipes(cmd, print, op)
	if err != nil {
		return nil, "", err
	}
	if err := cmd.Start(); err != nil {
		closeIfNonNil(stdin)
		closeIfNonNil(stdout)
		return nil, "", err
	}

	var ds Diffs
	var wg sync.WaitGroup
	if op == '<' || op == '|' {
		wg.Add(1)
		go func() {
			txt, _ := rope.ReadFrom(stdout)
			ds = Diffs{{At: a, Text: txt}}
			wg.Done()
		}()
	}
	if op == '>' || op == '|' {
		wg.Add(1)
		go func() {
			rope.Slice(ro, a[0], a[1]).WriteTo(stdin)
			stdin.Close()
			wg.Done()
		}()
	}

	wg.Wait()
	err = cmd.Wait()
	return ds, t, err
}

func openPipes(cmd *exec.Cmd, print io.Writer, op rune) (io.WriteCloser, io.ReadCloser, error) {
	cmd.Stderr = print
	if op == '>' {
		cmd.Stdout = print
	}
	var err error
	var stdin io.WriteCloser
	if op == '>' || op == '|' {
		if stdin, err = cmd.StdinPipe(); err != nil {
			return nil, nil, err
		}
	}
	var stdout io.ReadCloser
	if op == '<' || op == '|' {
		if stdout, err = cmd.StdoutPipe(); err != nil {
			closeIfNonNil(stdin)
			return nil, nil, err
		}
	}
	return stdin, stdout, nil
}

func closeIfNonNil(c io.Closer) {
	if c != nil {
		c.Close()
	}
}

func splitNewline(str string) (string, string) {
	i := strings.IndexRune(str, '\n')
	if i < 0 {
		return str, ""
	}
	return str[:i+1], str[i+1:]
}

func parseRegexp(t string) (*re1.Regexp, string, error) {
	delim, t := next(trimSpaceLeft(t))
	if delim == eof {
		return nil, "", errors.New("expected regular expression")
	}
	re, t, err := re1.New(t, re1.Opts{Delimiter: delim})
	return re, t, err
}

func addr(dot *[2]int64, ro rope.Rope, t string) ([2]int64, string, error) {
	left, t, err := addr1(*dot, 0, false, ro, t)
	if err != nil {
		return [2]int64{}, "", err
	}
	if left, t, err = addr2(*dot, left, ro, t); err != nil {
		return [2]int64{}, "", err
	}
	return addr3(dot, left, ro, t)
}

func addr3(dot *[2]int64, left [2]int64, ro rope.Rope, t0 string) ([2]int64, string, error) {
	r, t := next(trimSpaceLeft(t0))
	switch {
	case r == eof:
		return left, "", nil
	case r != ',' && r != ';':
		return left, t0, nil
	case left[0] < 0:
		left = [2]int64{}
	}
	if r == ';' {
		*dot = left
	}
	switch right, t, err := addr(dot, ro, t); {
	case err != nil:
		return [2]int64{}, "", err
	case right[0] < 0:
		right = [2]int64{ro.Len(), ro.Len()}
		fallthrough
	default:
		if left[0] > right[1] {
			return [2]int64{}, t, errors.New("address out of order")
		}
		return addr3(dot, [2]int64{left[0], right[1]}, ro, t)
	}
}

func addr2(dot, left [2]int64, ro rope.Rope, t0 string) ([2]int64, string, error) {
	r, t := next(trimSpaceLeft(t0))
	switch {
	case r == eof:
		return left, "", nil
	case strings.ContainsRune(addr1First, r):
		t, r = t0, '+' // Insert +
	case r != '+' && r != '-':
		return left, t0, nil
	}
	if left[0] < 0 {
		left = dot
	}
	at := left[1]
	if r == '-' {
		at = left[0]
	}
	switch right, t, err := addr1(dot, at, r == '-', ro, t); {
	case err != nil:
		return [2]int64{}, "", err
	case right[0] < 0:
		if right, _, err = addr1(dot, at, r == '-', ro, "1"); err != nil {
			return [2]int64{}, t, err
		}
		fallthrough
	default:
		return addr2(dot, right, ro, t)
	}
}

const addr1First = ".'#0123456789/$"

func addr1(dot [2]int64, at int64, rev bool, ro rope.Rope, t0 string) ([2]int64, string, error) {
	t0 = trimSpaceLeft(t0)
	switch r, t := next(t0); r {
	case eof:
		return [2]int64{-1, -1}, "", nil
	default:
		return [2]int64{-1, -1}, t0, nil
	case '.':
		return dot, t, nil
	case '#':
		return runeAddr(ro, at, rev, t)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return lineAddr(ro, at, rev, t0)
	case '/':
		return regexpAddr(ro, at, rev, t)
	case '$':
		return [2]int64{ro.Len(), ro.Len()}, t, nil
	}
}

func runeAddr(ro rope.Rope, at int64, rev bool, t string) ([2]int64, string, error) {
	nrunes, t, err := number(t)
	if err != nil {
		return [2]int64{}, "", err
	}
	var r io.RuneReader
	if rev {
		sl := rope.Slice(ro, 0, at)
		r = rope.NewReverseReader(sl)
	} else {
		sl := rope.Slice(ro, at, ro.Len())
		r = rope.NewReader(sl)
	}
	var nbytes int64
	for nrunes > 0 {
		_, w, err := r.ReadRune()
		if err != nil {
			return [2]int64{}, "", errors.New("address out of range")
		}
		nbytes += int64(w)
		nrunes--
	}
	if rev {
		return [2]int64{at - nbytes, at - nbytes}, t, nil
	}
	return [2]int64{at + nbytes, at + nbytes}, t, nil
}

func lineAddr(ro rope.Rope, at int64, rev bool, t string) ([2]int64, string, error) {
	n, t, err := number(t)
	if err != nil {
		return [2]int64{}, "", err
	}
	var dot [2]int64
	if rev {
		r := rope.NewReverseReader(rope.Slice(ro, 0, at))
		dot, err = lineReverse(r, at, n)
	} else {
		// Check the previous rune.
		rr := rope.NewReverseReader(rope.Slice(ro, 0, at))
		if r, _, err := rr.ReadRune(); err != nil || r == '\n' {
			n--
		}
		r := rope.NewReader(rope.Slice(ro, at, ro.Len()))
		dot, err = lineForward(r, at, n)
	}
	return dot, t, err
}

func lineForward(in *rope.Reader, at int64, nlines int) ([2]int64, error) {
	dot := [2]int64{at, at}
	for nlines >= 0 {
		b, err := in.ReadByte()
		switch {
		case err != nil && nlines == 0:
			b = '\n'
		case err != nil:
			return [2]int64{}, errors.New("address out of range")
		default:
			at++
		}
		if b == '\n' {
			nlines--
			dot[0], dot[1] = dot[1], at
		}
	}
	return dot, nil
}

func lineReverse(in *rope.ReverseReader, at int64, nlines int) ([2]int64, error) {
	dot := [2]int64{at, at}
	for {
		b, err := in.ReadByte()
		switch {
		case err != nil && nlines == 0:
			b = '\n'
		case err != nil:
			if nlines == 1 {
				return [2]int64{}, nil
			}
			return [2]int64{}, errors.New("address out of range")
		}
		if b == '\n' {
			dot[0], dot[1] = at, dot[0]
			if nlines--; nlines < 0 {
				return dot, nil
			}
		}
		at--
	}
}

func regexpAddr(ro rope.Rope, at int64, rev bool, t string) ([2]int64, string, error) {
	re, t, err := re1.New(t, re1.Opts{Delimiter: '/', Reverse: rev})
	if err != nil {
		return [2]int64{}, "", err
	}
	var ms []int64
	if rev {
		ms = re.FindReverseInRope(ro, 0, at)
		if ms == nil {
			ms = re.FindReverseInRope(ro, 0, ro.Len())
		}
	} else {
		ms = re.FindInRope(ro, at, ro.Len())
		if ms == nil {
			ms = re.FindInRope(ro, 0, ro.Len())
		}
	}
	if len(ms) == 0 {
		return [2]int64{}, t, errors.New("no match")
	}
	return [2]int64{ms[0], ms[1]}, t, err
}

const eof = -1

func next(t string) (rune, string) {
	if len(t) == 0 {
		return eof, ""
	}
	r, w := utf8.DecodeRuneInString(t)
	return r, t[w:]
}

func trimSpaceLeft(t string) string {
	return strings.TrimLeftFunc(t, unicode.IsSpace)
}

func number(t string) (int, string, error) {
	var i int
	for {
		r, w := utf8.DecodeRuneInString(t[i:])
		if r < '0' || '9' < r {
			break
		}
		i += w
	}
	if i == 0 {
		return 1, t, nil // defaults to 1
	}
	n, err := strconv.Atoi(t[:i])
	if err != nil {
		return 0, t, err
	}
	return n, t[i:], nil
}
