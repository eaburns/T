package re1

import (
	"io"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParse(t *testing.T) {
	tests := []struct {
		re      string
		residue string
		err     string
		delim   rune
	}{
		{
			re: "",
		},
		{
			re: "ab/c",
		},
		{
			re:      "ab/c",
			delim:   '/',
			residue: "c",
		},
		{
			re:    "ab\\/c",
			delim: '/',
		},
		{
			re:      "ab\nc", // literal newline
			residue: "c",
		},
		{
			re: `ab\nc`, // \n newline
		},
		{
			re: "ab\\\nc", // escaped newline
		},
		{
			re:  "?",
			err: "unexpected ?",
		},
		{
			re:  "*",
			err: "unexpected *",
		},
		{
			re:  "+",
			err: "unexpected +",
		},
		{
			re:  "|xyz",
			err: "unexpected |",
		},
		{
			re:  "abc||",
			err: "unexpected |",
		},
		{
			re:  "(",
			err: "unclosed (",
		},
		{
			re:  "(()",
			err: "unclosed (",
		},
		{
			re:  ")",
			err: "unopened )",
		},
		{
			re:  "())",
			err: "unopened )",
		},
		{
			re:  "[",
			err: "unclosed [",
		},
		{
			re:  "[]",
			err: "empty charclass",
		},
		{
			re: "[[]", // OK
		},
		{
			re:  "[^]",
			err: "empty charclass",
		},
		{
			re:  "[-z]",
			err: "bad range",
		},
		{
			re:  "[a-]",
			err: "bad range",
		},
		{
			re:  "[^-z]",
			err: "bad range",
		},
		{
			re:  "[a-z-Z]",
			err: "bad range",
		},
		{
			re:  "[z-a]",
			err: "bad range",
		},
		{
			re: `[\-a]`, // OK
		},
		{
			re: `[[-\]]`, // OK
		},
		{
			re: `[\]]`, // OK
		},
	}
	for _, test := range tests {
		var opts Opts
		if test.delim != 0 {
			opts.Delimiter = test.delim
		}
		_, residue, err := New(test.re, opts)
		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		if residue != test.residue || errStr != test.err {
			t.Errorf("New(%q, %#v)=_,%q,%v, want _,%q,%v",
				test.re, opts, residue, errStr, test.residue, test.err)
		}
	}
}

// These tests are pretty incomplete, because we rely on the RE2 suite instead.
var findTests = []findTest{
	{
		re: "",
		cases: []findTestCase{
			{str: "", want: []string{""}},
			{str: "z", want: []string{""}},
		},
	},
	{
		re: `\`,
		cases: []findTestCase{
			{str: ``, want: nil},
			{str: ``, want: nil},
			{str: `\`, want: []string{`\`}},
		},
	},
	{
		re: `\\`,
		cases: []findTestCase{
			{str: `\`, want: []string{`\`}},
		},
	},
	{
		re: `\n`,
		cases: []findTestCase{
			{str: "\n", want: []string{"\n"}},
		},
	},
	{
		re: `\t`,
		cases: []findTestCase{
			{str: `	`, want: []string{"	"}},
		},
	},
	{
		re: `\*`,
		cases: []findTestCase{
			{str: "***", want: []string{"*"}},
		},
	},
	{
		re: `[*]`,
		cases: []findTestCase{
			{str: "***", want: []string{"*"}},
		},
	},
}

func TestFind(t *testing.T) {
	for _, test := range findTests {
		runTest(t, test, Opts{})
	}
}

var findReverseTests = []findTest{
	{
		re: "",
		cases: []findTestCase{
			{str: "", want: []string{""}},
			{str: "z", want: []string{""}},
		},
	},
	{
		re: "a",
		cases: []findTestCase{
			{str: "", want: nil},
			{str: "z", want: nil},
			{str: "a", want: []string{"a"}},
			{str: "aa", want: []string{"a"}},
			{str: "xaa", want: []string{"a"}},
			{str: "aax", want: []string{"a"}},
			{str: "xaax", want: []string{"a"}},
			{str: "axaax", want: []string{"a"}},
			{str: "xaaxa", want: []string{"a"}},
		},
	},
	{
		re: "aa",
		cases: []findTestCase{
			{str: "", want: nil},
			{str: "z", want: nil},
			{str: "a", want: nil},
			{str: "aa", want: []string{"aa"}},
			{str: "xaa", want: []string{"aa"}},
			{str: "aax", want: []string{"aa"}},
			{str: "xaax", want: []string{"aa"}},
			{str: "axaax", want: []string{"aa"}},
			{str: "xaaxa", want: []string{"aa"}},
		},
	},
	{
		re: "a*",
		cases: []findTestCase{
			{str: "", want: []string{""}},
			{str: "z", want: []string{""}},
			{str: "a", want: []string{"a"}},
			{str: "aa", want: []string{"aa"}},
			{str: "xaa", want: []string{"aa"}},
			{str: "aax", want: []string{""}},
			{str: "xaax", want: []string{""}},
			{str: "axaax", want: []string{""}},
			{str: "xaaxa", want: []string{"a"}},
		},
	},
	{
		re: "abc",
		cases: []findTestCase{
			{str: "", want: nil},
			{str: "z", want: nil},
			{str: "abc", want: []string{"abc"}},
			{str: "cba", want: nil},
			{str: "abcxyz", want: []string{"abc"}},
		},
	},
	{
		re: "(abc)*",
		cases: []findTestCase{
			{str: "", want: []string{"", ""}},
			{str: "z", want: []string{"", ""}},
			{str: "abc", want: []string{"abc", "abc"}},
			{str: "abcabc", want: []string{"abcabc", "abc"}},
			{str: "cba", want: []string{"", ""}},
			{str: "abcxyz", want: []string{"", ""}},
			{str: "abcabc123abcxyz", want: []string{"", ""}},
			{str: "abcabcabcxyz", want: []string{"", ""}},
		},
	},
	{
		re: "(abc)+",
		cases: []findTestCase{
			{str: "", want: nil},
			{str: "z", want: nil},
			{str: "abc", want: []string{"abc", "abc"}},
			{str: "cba", want: nil},
			{str: "abcxyz", want: []string{"abc", "abc"}},
			{str: "abcabc123abcxyz", want: []string{"abc", "abc"}},
			{str: "abcabcabcxyz", want: []string{"abcabcabc", "abc"}},
		},
	},
	{
		re: "^abc",
		cases: []findTestCase{
			{str: "", want: nil},
			{str: "abc", want: []string{"abc"}},
			{str: "xabc", want: nil},
			{str: "x\nabc", want: []string{"abc"}},
		},
	},
	{
		re: "abc$",
		cases: []findTestCase{
			{str: "", want: nil},
			{str: "abc", want: []string{"abc"}},
			{str: "abcx", want: nil},
			{str: "abc\nx", want: []string{"abc"}},
		},
	},
	{
		re: "^abc$",
		cases: []findTestCase{
			{str: "", want: nil},
			{str: "xyz", want: nil},
			{str: "abc", want: []string{"abc"}},
			{str: "123\nabc", want: []string{"abc"}},
			{str: "abc\n123", want: []string{"abc"}},
			{str: "123\nabc\n123", want: []string{"abc"}},
			{str: "123abc", want: nil},
			{str: "abc123", want: nil},
			{str: "123abc", want: nil},
			{str: "abc123", want: nil},
			{str: "123abc123", want: nil},
		},
	},
	{
		re: "^$",
		cases: []findTestCase{
			{str: "", want: []string{""}},
			{str: "\n", want: []string{""}},
			{str: "\nxyz", want: []string{""}},
			{str: "xyz\n", want: []string{""}},
			{str: "xyz\n123", want: nil},
		},
	},
	{
		re: "(foo)(bar)",
		cases: []findTestCase{
			{str: "foobar", want: []string{"foobar", "foo", "bar"}},
		},
	},
}

func TestReverseFind(t *testing.T) {

	for _, test := range findReverseTests {
		runTest(t, test, Opts{Reverse: true})
	}
}

type findTest struct {
	re    string
	cases []findTestCase
}

type findTestCase struct {
	str  string
	want []string
}

func runTest(t *testing.T, test findTest, opts Opts) {
	t.Helper()
	re, residue, err := New(test.re, opts)
	if err != nil || residue != "" {
		t.Errorf("New(%q)=_,%q,%v", test.re, residue, err)
		return
	}
	for _, c := range test.cases {
		if opts.Reverse {
			runTestCaseReverse(t, re, c)
		} else {
			runTestCaseForward(t, re, c)
		}
	}
}

func runTestCaseForward(t *testing.T, re *Regexp, c findTestCase) {
	var got []string
	ms := re.Find(strings.NewReader(c.str))
	for i := 0; i < len(ms); i += 2 {
		if ms[i] < 0 || ms[i+1] < 0 {
			got = append(got, "")
			continue
		}
		s, e := int(ms[i]), int(ms[i+1])
		got = append(got, c.str[s:e])
	}
	if !reflect.DeepEqual(got, c.want) {
		t.Errorf("New(%q).Find(%q)=%#v, want %#v", re.source, c.str, got, c.want)
	}
}

func runTestCaseReverse(t *testing.T, re *Regexp, c findTestCase) {
	rr, nrunes := reverse(c.str)
	var got []string
	ms := re.Find(rr)
	for i := 0; i < len(ms); i += 2 {
		if ms[i] < 0 || ms[i+1] < 0 {
			got = append(got, "")
			continue
		}
		s := nrunes - int(ms[i+1])
		e := nrunes - int(ms[i])
		got = append(got, c.str[s:e])
	}
	if !reflect.DeepEqual(got, c.want) {
		t.Errorf("New(%q).Find(%q)=%#v, want %#v", re.source, c.str, got, c.want)
	}
}

func reverse(str string) (io.RuneReader, int) {
	n := utf8.RuneCountInString(str)
	return &revReader{str: str, i: n}, n
}

type revReader struct {
	str string
	i   int
}

func (rr *revReader) ReadRune() (rune, int, error) {
	if rr.i == 0 {
		return 0, 0, io.EOF
	}
	r, w := utf8.DecodeLastRuneInString(rr.str[:rr.i])
	rr.i -= w
	return r, w, nil
}
