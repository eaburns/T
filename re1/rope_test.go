package re1

import (
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/eaburns/T/rope"
)

func TestFindInRope(t *testing.T) {
	tests := []ropeTest{
		{
			re: "",
			cases: []ropeTestCase{
				{str: "", s: 0, e: 0, want: []string{""}},
				{str: "z", s: 0, e: 0, want: []string{""}},
				{str: "z", s: 1, e: 1, want: []string{""}},
				{str: "z", s: 0, e: 1, want: []string{""}},
			},
		},
		{
			re: "a",
			cases: []ropeTestCase{
				{str: "", want: nil},
				{str: "z", s: 0, e: 1, want: nil},
				{str: "a", s: 0, e: 0, want: nil},
				{str: "a", s: 1, e: 1, want: nil},
				{str: "a", s: 0, e: 1, want: []string{"a"}},
			},
		},
		{
			re: "abc",
			cases: []ropeTestCase{
				{str: "", want: nil},
				{str: "z", s: 0, e: 1, want: nil},
				{str: "abc", s: 0, e: 0, want: nil},
				{str: "abc", s: 0, e: 1, want: nil},
				{str: "abc", s: 0, e: 2, want: nil},
				{str: "abc", s: 0, e: 3, want: []string{"abc"}},
				{str: "abc", s: 1, e: 1, want: nil},
				{str: "abc", s: 1, e: 2, want: nil},
				{str: "abc", s: 1, e: 3, want: nil},
				{str: "abc", s: 2, e: 2, want: nil},
				{str: "abc", s: 2, e: 3, want: nil},
				{str: "abc", s: 3, e: 3, want: nil},
			},
		},
		{
			re: "^abc",
			cases: []ropeTestCase{
				{str: "abc", s: 0, e: 3, want: []string{"abc"}},
				{str: "\nabc", s: 1, e: 4, want: []string{"abc"}},
				{str: "xabc", s: 1, e: 4, want: nil},
				{str: "xabc\nabc", s: 0, e: 8, want: []string{"abc"}},
			},
		},
		{
			re: "abc$",
			cases: []ropeTestCase{
				{str: "abc", s: 0, e: 3, want: []string{"abc"}},
				{str: "abc\n", s: 0, e: 3, want: []string{"abc"}},
				{str: "abcx", s: 0, e: 3, want: nil},
				{str: "abcxabc\n", s: 0, e: 8, want: []string{"abc"}},
				{str: "abcxabc", s: 0, e: 7, want: []string{"abc"}},
			},
		},
		{
			re: "^abc$",
			cases: []ropeTestCase{
				{str: "abc", s: 0, e: 3, want: []string{"abc"}},
				{str: "abc\n", s: 0, e: 3, want: []string{"abc"}},
				{str: "\nabc", s: 1, e: 4, want: []string{"abc"}},
				{str: "xabc", s: 1, e: 4, want: nil},
				{str: "abcx", s: 0, e: 3, want: nil},
				{str: "xabc\n", s: 1, e: 4, want: nil},
				{str: "\nabcx", s: 1, e: 4, want: nil},
			},
		},
		{
			re: "a*",
			cases: []ropeTestCase{
				{str: "aaa", s: 0, e: 0, want: []string{""}},
				{str: "aaa", s: 0, e: 1, want: []string{"a"}},
				{str: "aaa", s: 0, e: 2, want: []string{"aa"}},
				{str: "aaa", s: 0, e: 3, want: []string{"aaa"}},
				{str: "aaa", s: 1, e: 3, want: []string{"aa"}},
				{str: "aaa", s: 2, e: 3, want: []string{"a"}},
				{str: "aaa", s: 3, e: 3, want: []string{""}},
				{str: "aaa", s: 1, e: 1, want: []string{""}},
				{str: "aaa", s: 2, e: 2, want: []string{""}},
			},
		},
	}
	for _, test := range append(ropeFromFindTests(findTests), tests...) {
		runRopeTest(t, test, Opts{})
	}
}

func TestFindReverseInRope(t *testing.T) {
	tests := []ropeTest{
		{
			// abcX would match, but X isn't in [s,e).
			re: "abcX",
			cases: []ropeTestCase{
				{str: "abcX", s: 0, e: 3, want: nil},
			},
		},
		{
			re: "abc$",
			cases: []ropeTestCase{
				{str: "abc", s: 0, e: 3, want: []string{"abc"}},
				{str: "abc\n", s: 0, e: 3, want: []string{"abc"}},
				{str: "abcx", s: 0, e: 3, want: nil},
			},
		},
		{
			re: "^abc",
			cases: []ropeTestCase{
				{str: "abc", s: 0, e: 3, want: []string{"abc"}},
				{str: "\nabc", s: 1, e: 4, want: []string{"abc"}},
				{str: "xabc", s: 1, e: 4, want: nil},
			},
		},
		{
			re: "^abc$",
			cases: []ropeTestCase{
				{str: "abc", s: 0, e: 3, want: []string{"abc"}},
				{str: "\nabc", s: 1, e: 4, want: []string{"abc"}},
				{str: "\nabc", s: 0, e: 4, want: []string{"abc"}},
				{str: "abc\n", s: 0, e: 3, want: []string{"abc"}},
				{str: "abc\n", s: 0, e: 4, want: []string{"abc"}},
				{str: "\nabc\n", s: 1, e: 4, want: []string{"abc"}},
				{str: "\nabc\n", s: 0, e: 4, want: []string{"abc"}},
				{str: "\nabc\n", s: 1, e: 5, want: []string{"abc"}},
				{str: "\nabc\n", s: 0, e: 5, want: []string{"abc"}},
				{str: "xabc\n", s: 1, e: 4, want: nil},
				{str: "xabc\n", s: 0, e: 4, want: nil},
				{str: "xabc\n", s: 0, e: 5, want: nil},
				{str: "xabc\n", s: 1, e: 5, want: nil},
				{str: "\nabcx", s: 1, e: 4, want: nil},
				{str: "\nabcx", s: 0, e: 4, want: nil},
				{str: "\nabcx", s: 0, e: 5, want: nil},
				{str: "\nabcx", s: 1, e: 5, want: nil},
			},
		},
		{
			re: "^$",
			cases: []ropeTestCase{
				{str: "", s: 0, e: 0, want: []string{""}},
				{str: "\n", s: 0, e: 1, want: []string{""}},
				{str: "abc\n\nxyz", s: 4, e: 8, want: []string{""}},
				{str: "abc\n\nxyz", s: 0, e: 5, want: []string{""}},
			},
		},
		{
			re: "(foo)(bar)",
			cases: []ropeTestCase{
				{str: "foobarbaz", s: 0, e: 7, want: []string{"foobar", "foo", "bar"}},
			},
		},
		{
			re: "a+",
			cases: []ropeTestCase{
				{str: "aaa", s: 0, e: 0, want: nil},
				{str: "aaa", s: 0, e: 1, want: []string{"a"}},
				{str: "aaa", s: 0, e: 2, want: []string{"aa"}},
				{str: "aaa", s: 0, e: 3, want: []string{"aaa"}},
				{str: "aaa", s: 1, e: 3, want: []string{"aa"}},
				{str: "aaa", s: 2, e: 3, want: []string{"a"}},
				{str: "aaa", s: 3, e: 3, want: nil},
				{str: "aaaxyz", s: 0, e: 6, want: []string{"aaa"}},
				{str: "aaaxyz", s: 1, e: 6, want: []string{"aa"}},
				{str: "aaaxyz", s: 2, e: 6, want: []string{"a"}},
				{str: "aaaxyz", s: 3, e: 6, want: nil},
			},
		},
	}
	for _, test := range append(ropeFromFindTests(findReverseTests), tests...) {
		runRopeTest(t, test, Opts{Reverse: true})
	}

}

type ropeTest struct {
	re    string
	cases []ropeTestCase
}

type ropeTestCase struct {
	str  string
	s, e int64
	want []string
}

func ropeFromFindTests(ftests []findTest) []ropeTest {
	rtests := make([]ropeTest, len(ftests))
	for i, ftest := range ftests {
		rtests[i] = ropeTest{re: ftest.re}
		for _, fc := range ftest.cases {
			e := int64(utf8.RuneCountInString(fc.str))
			rc := ropeTestCase{str: fc.str, s: 0, e: e, want: fc.want}
			rtests[i].cases = append(rtests[i].cases, rc)
		}
	}
	return rtests
}

func runRopeTest(t *testing.T, test ropeTest, opts Opts) {
	t.Helper()
	re, residue, err := New(test.re, opts)
	if err != nil || residue != "" {
		t.Errorf("New(%q)=_,%q,%v", test.re, residue, err)
		return
	}
	for _, c := range test.cases {
		var got []string
		var ms []int64
		if opts.Reverse {
			ms = re.FindReverseInRope(rope.New(c.str), c.s, c.e)
		} else {
			ms = re.FindInRope(rope.New(c.str), c.s, c.e)
		}
		for i := 0; i < len(ms); i += 2 {
			if ms[i] < 0 || ms[i+1] < 0 {
				got = append(got, "")
				continue
			}
			s, e := int(ms[i]), int(ms[i+1])
			got = append(got, c.str[s:e])
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("New(%q).Find(%q, %d, %d)=%#v, want %#v",
				re.source, c.str, c.s, c.e, got, c.want)
		}
	}
}
