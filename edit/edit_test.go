package edit

import (
	"regexp"
	"strings"
	"testing"

	"github.com/eaburns/T/rope"
)

func TestDiffs_Update(t *testing.T) {
	tests := []struct {
		name  string
		dot   Dot
		diffs Diffs
		want  Dot
	}{
		{
			name:  "no diff",
			dot:   Dot{5, 10},
			diffs: Diffs{},
			want:  Dot{5, 10},
		},
		{
			name:  "delete after dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{11, 15}, Text: nil}},
			want:  Dot{5, 10},
		},
		{
			name:  "add after dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{11, 11}, Text: rope.New("Hello")}},
			want:  Dot{5, 10},
		},
		{
			name:  "grow after dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{11, 12}, Text: rope.New("123")}},
			want:  Dot{5, 10},
		},
		{
			name:  "shrink after dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{11, 100}, Text: rope.New("123")}},
			want:  Dot{5, 10},
		},
		{
			name:  "delete before dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{0, 3}, Text: nil}},
			want:  Dot{2, 7},
		},
		{
			name:  "add before dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{0, 0}, Text: rope.New("xyz")}},
			want:  Dot{8, 13},
		},
		{
			name: "grow before dot",
			dot:  Dot{5, 10},
			// change from 1 → 3
			diffs: Diffs{{At: Dot{0, 1}, Text: rope.New("123")}},
			want:  Dot{7, 12},
		},
		{
			name: "shrink before dot",
			dot:  Dot{5, 10},
			// change from 3 → 1
			diffs: Diffs{{At: Dot{1, 4}, Text: rope.New("1")}},
			want:  Dot{3, 8},
		},
		{
			name:  "delete inside dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{6, 9}, Text: nil}},
			want:  Dot{5, 7},
		},
		{
			name:  "add inside dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{6, 6}, Text: rope.New("xyz")}},
			want:  Dot{5, 13},
		},
		{
			name: "grow inside dot",
			dot:  Dot{5, 10},
			// change from 1 → 3
			diffs: Diffs{{At: Dot{6, 7}, Text: rope.New("123")}},
			want:  Dot{5, 12},
		},
		{
			name: "shrink inside dot",
			dot:  Dot{5, 10},
			// change from 3 → 1
			diffs: Diffs{{At: Dot{6, 9}, Text: rope.New("1")}},
			want:  Dot{5, 8},
		},
		{
			name:  "delete exactly dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{5, 10}, Text: nil}},
			want:  Dot{5, 5},
		},
		{
			// This effectively deletes dot and inserts before it.
			name:  "grow exactly dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{5, 10}, Text: rope.New("1234567890")}},
			want:  Dot{15, 15},
		},
		{
			// This effectively deletes dot and inserts before it.
			name:  "shrink exactly dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{5, 10}, Text: rope.New("1")}},
			want:  Dot{6, 6},
		},
		{
			name:  "delete over all of dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{0, 15}, Text: nil}},
			want:  Dot{0, 0},
		},
		{
			// This effectively deletes dot and inserts before it.
			name:  "grow over all of dot",
			dot:   Dot{5, 6},
			diffs: Diffs{{At: Dot{4, 7}, Text: rope.New("1234567890")}},
			want:  Dot{14, 14},
		},
		{
			// This effectively deletes dot and inserts before it.
			name:  "shrink over all of dot",
			dot:   Dot{5, 6},
			diffs: Diffs{{At: Dot{4, 7}, Text: rope.New("1")}},
			want:  Dot{5, 5},
		},
		{
			name:  "delete over beginning of dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{1, 7}, Text: nil}},
			want:  Dot{1, 4},
		},
		{
			// This deletes the beginning of dot and inserts before it.
			name:  "grow over beginning of dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{4, 7}, Text: rope.New("1234567890")}},
			want:  Dot{14, 17},
		},
		{
			// This deletes the beginning of dot and inserts before it.
			name:  "shrink over beginning of dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{4, 7}, Text: rope.New("1")}},
			want:  Dot{5, 8},
		},
		{
			name:  "delete over end of dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{8, 20}, Text: nil}},
			want:  Dot{5, 8},
		},
		{
			// This deletes the end of dot and inserts after it.
			name:  "grow over end of dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{8, 12}, Text: rope.New("1234567890")}},
			want:  Dot{5, 8},
		},
		{
			// This deletes the end of dot and inserts after it.
			name:  "shrink over end of dot",
			dot:   Dot{5, 10},
			diffs: Diffs{{At: Dot{8, 12}, Text: rope.New("1")}},
			want:  Dot{5, 8},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.diffs.Update(test.dot); got != test.want {
				t.Errorf("%+v.Update(%v)=%v, want %v",
					test.diffs, test.dot, got, test.want)
			}
		})
	}
}

func TestAddr(t *testing.T) {
	tests := []test{
		{
			name: "empty address",
			str:  "Hello, 世界",
			cases: []testCase{
				{edit: "", err: "no address"},
			},
		},
		{
			name: "trailing runes",
			str:  "Hello, 世界",
			cases: []testCase{
				{edit: "1 xyz", err: "expected end-of-input"},
			},
		},
		{
			name: "end of file",
			str:  "Hello, 世界",
			cases: []testCase{
				{edit: "$", want: ""},
			},
		},
		{
			name: "forward rune",
			str:  "Hello, 世界",
			cases: []testCase{
				{edit: "#", want: ""},
				{edit: "#1,#2", want: "e"},
				{edit: "#7,#8", want: "世"},
				{edit: "#9", want: ""},
				{edit: "#10", err: "address out of range"},
			},
		},
		{
			name: "reverse rune",
			str:  "Hello, 世界",
			cases: []testCase{
				{edit: "#9-#0,#9", want: ""},
				{edit: "#9-#1,#9", want: "界"},
				{edit: "#9-#4,#9", want: ", 世界"},
				{edit: "#9-#9,#9", want: "Hello, 世界"},
				{edit: "#9-#10,#9", err: "address out of range"},
			},
		},
		{
			name: "bad rune",
			str:  "Hello, 世界",
			cases: []testCase{
				{edit: "#18446744073709551615", err: "value out of range"},
			},
		},
		{
			name: "forward line",
			str:  "Hello,\n世界\n123",
			cases: []testCase{
				{edit: "0", want: ""},
				{edit: "1", want: "Hello,\n"},
				{edit: "2", want: "世界\n"},
				{edit: "3", want: "123"},
				{edit: "4", err: "address out of range"},
				{edit: "100", err: "address out of range"},
				{edit: "#1+0", want: "ello,\n"},
				{edit: "1+1", want: "世界\n"},
				{edit: "1+0", want: ""},
				{edit: "/世界/+0", want: "\n"},
				{edit: "1+3", err: "address out of range"},
			},
		},
		{
			name: "reverse line",
			str:  "Hello,\n世界\n123",
			cases: []testCase{
				{edit: "$-0", want: "123"},
				{edit: "$-1", want: "世界\n"},
				{edit: "$-2", want: "Hello,\n"},
				{edit: "$-3", want: ""}, // line 0
				{edit: "$-4", err: "address out of range"},
				{edit: "$-100", err: "address out of range"},
			},
		},
		{
			name: "bad line",
			str:  "Hello, 世界",
			cases: []testCase{
				{edit: "18446744073709551615", err: "value out of range"},
			},
		},
		{
			name: "forward regexp",
			str:  `Hello, 世界`,
			cases: []testCase{
				{edit: "/世界", want: "世界"},
				{edit: "/世界/", want: "世界"},
				{edit: "/[a-z][a-z][a-z]", want: "ell"},
				{edit: "/X*", want: ""},
				{edit: "#2+/...", want: "llo"},
				{edit: "$+/世界", want: "世界"}, // wrap
				{edit: "/NoMatch", err: "no match"},
				{edit: "/(", err: "unclosed [(]"},
			},
		},
		{
			name: "reverse regexp",
			str:  `Hello, 世界`,
			cases: []testCase{
				{edit: "$-/..", want: "世界"},
				{edit: "#1-/..", want: "世界"}, // wrap
				{edit: "#1-/NoMatch", err: "no match"},
			},
		},
		{
			// This tests a bug in a previous implementation of addr
			// that used forward matching to implement reverse match.
			// It started from the beginning and matched forward,
			// returning the last match it hit.
			// However, it started from the end of the previous match
			// to compute the next match, and consequently fails this.
			name: "reverse regexp overlapping",
			str:  `12345`,
			cases: []testCase{
				{edit: "$-/[0-9][0-9][0-9]", want: "345"},
			},
		},
		{
			name: "reverse regexp anchor",
			str:  "XXX123\nXXX\n123XXX",
			cases: []testCase{
				{edit: "$-/^123$", err: "no match"},
				{edit: "$-/^XXX$", want: "XXX"},
			},
		},
		{
			name: "regexp escaped delimiter",
			str:  `Hello, 世界/`,
			cases: []testCase{
				{edit: `/世界\/`, want: "世界/"},
			},
		},
		{
			name: "regexp escaped escape",
			str:  `Hello, 世界\`,
			cases: []testCase{
				{edit: `/世界\\`, want: `世界\`},
			},
		},
		{
			name: "regexp un-escaped newline",
			str:  "Hello, 世界\n",
			cases: []testCase{
				{edit: `/世界\n`, want: "世界\n"},
			},
		},
		{
			name: "regexp escaped newline",
			str:  "Hello, 世界\nxyz",
			cases: []testCase{
				{edit: "/世界\\\nxyz", want: "世界\nxyz"},
			},
		},
		{
			name: "plus addr",
			str:  "line1\nline2\nline3",
			dot:  Dot{0, 1}, // l in line1
			cases: []testCase{
				{edit: ".", want: "l"}, // verify .

				{edit: "$+$", want: ""},
				{edit: ".+$", want: ""},
				{edit: "#1+$", want: ""},
				{edit: "1+$", want: ""},
				{edit: "/line/+$", want: ""},

				{edit: "$+.", want: "l"},
				{edit: ".+.", want: "l"},
				{edit: "#1+.", want: "l"},
				{edit: "1+.", want: "l"},
				{edit: "/line/+.", want: "l"},

				{edit: "$+#1", err: "out of range"},
				{edit: ".+#1", want: ""},
				{edit: "#1+#1", want: ""},
				{edit: "1+#1", want: ""},
				{edit: "/line/+#1", want: ""},

				{edit: "$+1", err: "out of range"},
				{edit: ".+1", want: "line2\n"},
				{edit: "#1+1", want: "line2\n"},
				{edit: "1+1", want: "line2\n"},
				{edit: "/line/+1", want: "line2\n"},

				{edit: "$+/line[0-9]/", want: "line1"},
				{edit: ".+/line[0-9]/", want: "line2"},
				{edit: "#1+/line[0-9]/", want: "line2"},
				{edit: "1+/line[0-9]/", want: "line2"},
				{edit: "/line/+/line[0-9]/", want: "line2"},

				{edit: "1+1+1", want: "line3"},
				{edit: "1+", want: "line2\n"},
				{edit: "$+", err: "address out of range"},
				{edit: ".+1", want: "line2\n"},
				{edit: "+1", want: "line2\n"},
				{edit: ".+/[a-z0-9]*/", want: "ine1"},
				{edit: "+/[a-z0-9]*/", want: "ine1"},

				{edit: "0+0", want: ""},
				{edit: "0+1", want: "line1\n"},

				// Whitespace
				{edit: "1 + 1", want: "line2\n"},
			},
		},
		{
			name: "insert plus",
			str:  "line1\nline2\nline3",
			dot:  Dot{0, 1}, // l in line1
			cases: []testCase{
				// Insert +.
				{edit: "1 /line.*/", want: "line2"},
				{edit: "1 2", want: "line3"},
			},
		},
		{
			name: "minus addr",
			str:  "line1\nline2\nline3",
			dot:  Dot{15, 16}, // e in line3
			cases: []testCase{
				{edit: ".", want: "e"}, // verify .

				{edit: "$-$", want: ""},
				{edit: ".-$", want: ""},
				{edit: "#1-$", want: ""},
				{edit: "1-$", want: ""},
				{edit: "/line/-$", want: ""},

				{edit: "$-.", want: "e"},
				{edit: ".-.", want: "e"},
				{edit: "#1-.", want: "e"},
				{edit: "1-.", want: "e"},
				{edit: "/line/-.", want: "e"},

				{edit: "$-#1", want: ""},
				{edit: ".-#1", want: ""},
				{edit: "#1-#1", want: ""},
				{edit: "1-#1", err: "address out of range"},
				{edit: "2-#1", want: ""},
				{edit: "/line2/-#1", want: ""},

				{edit: "$-1", want: "line2\n"},
				{edit: ".-1", want: "line2\n"},
				{edit: "#1-1", want: ""}, // line 0 == empty string at bof
				{edit: "#1-2", err: "address out of range"},
				{edit: "#6-1", want: "line1\n"},
				{edit: "1-1", want: ""}, // line 0 == empty string at bof
				{edit: "2-2", want: ""}, // line 0 == empty string at bof
				{edit: "3-3", want: ""}, // line 0 == empty string at bof
				{edit: "1-2", err: "address out of range"},
				{edit: "/line2/-1", want: "line1\n"},

				{edit: "$-/line[0-9]/", want: "line3"},
				{edit: ".-/line[0-9]/", want: "line2"},
				{edit: "#1-/line[0-9]/", want: "line3"},
				{edit: "1-/line[0-9]/", want: "line3"},
				{edit: "/line/-/line[0-9]/", want: "line3"},

				{edit: "3-1-1", want: "line1\n"},
				{edit: "2-", want: "line1\n"},
				{edit: "0-", want: ""},
				{edit: ".-1", want: "line2\n"},
				{edit: "-1", want: "line2\n"},
				{edit: ".-/[a-z0-9]*$/", want: "line2"},
				{edit: "-/[a-z0-9]*$/", want: "line2"},

				// Whitespace
				{edit: "3 - 1", want: "line2\n"},
			},
		},
		{
			name: "comma address",
			str:  "line1\nline2\nline3",
			dot:  Dot{0, 1}, // l in line1
			cases: []testCase{
				{edit: "$,$", want: ""},
				{edit: "$,$+1", err: "address out of range"},
				{edit: ".,.", want: "l"},
				{edit: ".,2", want: "line1\nline2\n"},
				{edit: "#1,#1", want: ""},
				{edit: "#1,#5", want: "ine1"},
				{edit: "0,0", want: ""},
				{edit: "1,1", want: "line1\n"},
				{edit: "1,2", want: "line1\nline2\n"},
				{edit: "1+1,3", want: "line2\nline3"},
				{edit: "3-1,3", want: "line2\nline3"},
				{edit: "1,2,3", want: "line1\nline2\nline3"},

				{edit: "0,2", want: "line1\nline2\n"},
				{edit: ",2", want: "line1\nline2\n"},
				{edit: "2,$", want: "line2\nline3"},
				{edit: "2,", want: "line2\nline3"},
				{edit: ",", want: "line1\nline2\nline3"},
				{edit: "3,1", err: "address out of order"},

				// , doesn't set dot for computing a2.
				// Were it to set ., . would be "line2\n"
				// for computing +1; we would get
				// "line2\nline3".
				{edit: "2,.+1", want: "line2\n"},
			},
		},
		{
			name: "semi-colon address",
			str:  "line1\nline2\nline3",
			dot:  Dot{0, 1}, // l in line1
			cases: []testCase{
				{edit: "$;$", want: ""},
				{edit: "$;$+1", err: "address out of range"},
				{edit: ".;.", want: "l"},
				{edit: ".;2", want: "line1\nline2\n"},
				{edit: "#1;#1", want: ""},
				{edit: "#1;#5", want: "ine1"},
				{edit: "0;0", want: ""},
				{edit: "1;1", want: "line1\n"},
				{edit: "1;2", want: "line1\nline2\n"},
				{edit: "1+1;3", want: "line2\nline3"},
				{edit: "3-1;3", want: "line2\nline3"},
				{edit: "1;2;3", want: "line1\nline2\nline3"},

				{edit: "0;2", want: "line1\nline2\n"},
				{edit: ";2", want: "line1\nline2\n"},
				{edit: "2;$", want: "line2\nline3"},
				{edit: "2;", want: "line2\nline3"},
				{edit: ";", want: "line1\nline2\nline3"},
				{edit: "3;1", err: "address out of order"},

				// ; sets dot for computing a2.
				// . is set to "line2\n" for computing +1,
				// and so we get "line2\nline3".
				{edit: "2;.+1", want: "line2\nline3"},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) { runAddrTest(t, test) })
	}
}

func TestEdit(t *testing.T) {
	tests := []test{
		{
			name: "errors",
			str:  "Hello, 世界",
			cases: []testCase{
				{edit: "100", err: "address out of range"},
				{edit: "a/hi/ N", err: "expected end-of-input"},
				{edit: "", err: "no command"},
				{edit: "N", err: "bad command N"},
			},
		},
		{
			name: "append",
			str:  "Hello",
			cases: []testCase{
				{edit: "0 a//", want: "Hello"},
				{edit: "0 a/Oh, /", want: "Oh, Hello"},
				{edit: "$ a/, World!", want: "Hello, World!"},
				{edit: "/H/ a//", want: "Hello"},
				{edit: "/H/ a/XYZ/", want: "HXYZello"},
			},
		},
		{
			name: "change",
			str:  "Hello",
			cases: []testCase{
				{edit: "0 c//", want: "Hello"},
				{edit: "0 c/Oh, /", want: "Oh, Hello"},
				{edit: "$ c/, World!", want: "Hello, World!"},
				{edit: "/H/ c/h/", want: "hello"},
			},
		},
		{
			name: "delete",
			str:  "Hello",
			cases: []testCase{
				{edit: "0 d", want: "Hello"},
				{edit: "#3 d", want: "Hello"},
				{edit: "/llo/d", want: "He"},
			},
		},
		{
			name: "insert",
			str:  "Hello",
			cases: []testCase{
				{edit: "0 i//", want: "Hello"},
				{edit: "0 i/Oh, /", want: "Oh, Hello"},
				{edit: "$ i/, World!", want: "Hello, World!"},
				{edit: "/H/ i/XYZ/", want: "XYZHello"},
				{edit: "/H/ i//", want: "Hello"},
			},
		},
		{
			name: "delimited text",
			str:  "Hello, 世界",
			cases: []testCase{
				{edit: "0 a", want: "Hello, 世界"},
				{edit: "0 a//", want: "Hello, 世界"},
				{edit: "0 a/", want: "Hello, 世界"},
				{edit: "0 aDD", want: "Hello, 世界"},
				{edit: "0 aD", want: "Hello, 世界"},
				{edit: "0 a/Hi! /", want: "Hi! Hello, 世界"},
				{edit: "0 a /Hi! /", want: "Hi! Hello, 世界"},
				{edit: "0 a\t/Hi! /", want: "Hi! Hello, 世界"},
				{edit: "/Hello,/ a / my/", want: "Hello, my 世界"},
				{edit: "/Hello,/ a / my", want: "Hello, my 世界"},
				{edit: "/Hello,/ a / my\n", want: "Hello, my 世界"},
				{edit: "/Hello,/ a / my\\\nok", want: "Hello, my\nok 世界"},
				{edit: "/Hello,/ a D myD", want: "Hello, my 世界"},
				{edit: "/Hello,/ a D my", want: "Hello, my 世界"},
				{edit: `/Hello,/ a /\\/`, want: `Hello,\ 世界`},
				{edit: `/Hello,/ a /\t/`, want: `Hello,	 世界`},
				{edit: `/Hello,/ a /\n/`, want: "Hello,\n 世界"},
				{edit: `/Hello,/ a /\/stuff/`, want: "Hello,/stuff 世界"},
				{edit: `/Hello,/ a /\`, want: `Hello,\ 世界`},
			},
		},
		{
			name: "multi-line text",
			str:  "Hello, 世界",
			cases: []testCase{
				{edit: "0 a\n", want: "Hello, 世界"},
				{edit: "0 a\n.", want: "Hello, 世界"},
				{edit: "0 a\n.\n", want: "Hello, 世界"},

				{edit: "0 a\nHi.\n", want: "Hi.\nHello, 世界"},
				{edit: "0 a\nHi.\n.", want: "Hi.Hello, 世界"},
				{edit: "0 a\nHi.\n.\n", want: "Hi.Hello, 世界"},

				{edit: "0 a\nline1\nline2\nline3", want: "line1\nline2\nline3Hello, 世界"},
				{edit: "0 a\nline1\nline2\nline3\n", want: "line1\nline2\nline3\nHello, 世界"},
				{edit: "0 a\nline1\nline2\nline3\n.", want: "line1\nline2\nline3Hello, 世界"},
				{edit: "0 a\nline1\nline2\nline3\n.\n", want: "line1\nline2\nline3Hello, 世界"},
			},
		},
		{
			name: "move",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: "1m", err: "expected address"},
				{edit: "1m4", err: "address out of range"},
				{edit: "1m2", want: "line2\nline1\nline3"},
				{edit: "2m0", want: "line2\nline1\nline3"},
				{edit: "#2,#3m1", want: "lie1\nnline2\nline3"},
				{edit: "1m#3", want: "line1\nline2\nline3"},
				{edit: "0m1", want: "line1\nline2\nline3"},
			},
		},
		{
			name: "print",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: "pxyz", err: "expected end-of-input"},
				{
					edit:  "0p",
					want:  "line1\nline2\nline3",
					print: "",
				},
				{
					edit:  "1p",
					want:  "line1\nline2\nline3",
					print: "line1\n",
				},
			},
		},
		{
			name: "copy",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: "1t", err: "expected address"},
				{edit: "1t4", err: "address out of range"},
				{edit: "1t2", want: "line1\nline2\nline1\nline3"},
				{edit: "0t1", want: "line1\nline2\nline3"},
			},
		},
		{
			name: "sub",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: ",s", err: "expected regular expression"},
				{edit: ",s/*", err: "unexpected *"},
				{edit: ",s/NO MATCH//g", err: "no match"},
				{edit: ",s/line/LINE/", want: "LINE1\nline2\nline3"},
				{edit: ",s/line/LINE/g", want: "LINE1\nLINE2\nLINE3"},
				{edit: ",s   /line/LINE/g", want: "LINE1\nLINE2\nLINE3"},
				{edit: "2,3s/line/LINE/g", want: "line1\nLINE2\nLINE3"},
				{edit: ",s/line//g", want: "1\n2\n3"},
				{edit: ",s/line/1/g", want: "11\n12\n13"},
				{edit: ",s/line/12345/g", want: "123451\n123452\n123453"},
				{edit: `1s/line/\0\0/g`, want: "lineline1\nline2\nline3"},
				{edit: `1s/line/\1\1/g`, want: "1\nline2\nline3"},
				{edit: `1s/(li)(ne)(X?)/\1\1/g`, want: "lili1\nline2\nline3"},
				{edit: `1s/(li)(ne)(X?)/\2\2/g`, want: "nene1\nline2\nline3"},
				{edit: `1s/(li)(ne)(X?)/\3\3/g`, want: "1\nline2\nline3"},
				{edit: `1s/(li)(ne)(X?)/\4\4/g`, want: "1\nline2\nline3"},
				{edit: `1s/(li)(ne)(X?)/\\/g`, want: "\\1\nline2\nline3"},
				{edit: `,s/(li)(ne)(X?)/\1/g`, want: "li1\nli2\nli3"},
				{edit: `,s//./g`, want: ".l.i.n.e.1.\n.l.i.n.e.2.\n.l.i.n.e.3."},

				{edit: `,s0/line/LINE/`, want: "LINE1\nline2\nline3"},
				{edit: `,s1/line/LINE/`, want: "LINE1\nline2\nline3"},
				{edit: `,s2/line/LINE/`, want: "line1\nLINE2\nline3"},
				{edit: `,s3/line/LINE/`, want: "line1\nline2\nLINE3"},
				{edit: `,s4/line/LINE/`, err: "no match"},
				{edit: `,s2/line/LINE/g`, want: "line1\nLINE2\nLINE3"},
				{edit: `,s  2  /line/LINE/g`, want: "line1\nLINE2\nLINE3"},
			},
		},
		{
			name: "cond match",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: ",g", err: "expected regular expression"},
				{edit: ",g/*", err: "unexpected *"},
				{edit: ",g/line/12d", err: "address out of range"},
				{edit: ",g/line/d junk", err: "expected end-of-input"},
				{edit: ",g/no match/d junk", want: "line1\nline2\nline3"},
				{edit: ",g/no match/,d", want: "line1\nline2\nline3"},
				{edit: ",g/line/.d", want: ""},
				{edit: "2g/line/.d", want: "line1\nline3"},
			},
		},
		{
			name: "cond mismatch",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: ",v", err: "expected regular expression"},
				{edit: ",v/*", err: "unexpected *"},
				{edit: ",v/no match/12d", err: "address out of range"},
				{edit: ",v/no match/d junk", err: "expected end-of-input"},
				{edit: ",v/line/d junk", want: "line1\nline2\nline3"},
				{edit: ",v/line/,d", want: "line1\nline2\nline3"},
				{edit: ",v/no match/.d", want: ""},
				{edit: "2v/no match/.d", want: "line1\nline3"},
			},
		},
		{
			name: "loop match",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: ",x", err: "expected regular expression"},
				{edit: ",x/*", err: "unexpected *"},

				// No match isn't an error for loops as it is for sub.
				{edit: ",x/NO MATCH/d", want: "line1\nline2\nline3"},

				// Error in the nested command.
				{edit: ",x/line/.+100d", err: "address out of range"},
				// Trailing newline is OK.
				{edit: ",x/line/d\n", want: "1\n2\n3"},
				// Trailing junk is not OK, but we only notice on a match.
				{edit: ",x/line/d junk", err: "expected end-of-input"},

				// Here there is no match, so we don't notice.
				{edit: ",x/NO MATCH/d junk", want: "line1\nline2\nline3"},

				{edit: ",x/line/,d", err: "out of order"},

				{edit: ",x/line/d", want: "1\n2\n3"},
				{edit: ",x/line/c/LINE", want: "LINE1\nLINE2\nLINE3"},
				{edit: ",x/line/c/1", want: "11\n12\n13"},
				{edit: ",x/line/c/12345", want: "123451\n123452\n123453"},
				{edit: ",x/./.d", want: "\n\n"},
				{edit: ",x/^.*$/x/[a-z]/c/x", want: "xxxx1\nxxxx2\nxxxx3"},
				{edit: ",x//c/.", want: ".l.i.n.e.1.\n.l.i.n.e.2.\n.l.i.n.e.3."},
				{edit: ",x/no match/d junk", want: "line1\nline2\nline3"},
			},
		},
		{
			name: "loop mismatch",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: ",y", err: "expected regular expression"},
				{edit: ",y/*", err: "unexpected *"},
				// No match isn't an error for loops as it is for sub.
				{edit: ",y/NO MATCH/d", want: ""},
				{edit: ",y/line/,d", err: "out of order"},
				// Error in the first nested command.
				{edit: ",y/line/.+100d", err: "address out of range"},
				// Error in the trailing nested command.
				{edit: ",y/line/.+1d", err: "address out of range"},
				{edit: ",y/line2/d", want: "line2"},
				{edit: ",y/line/d", want: "linelineline"},
				{edit: ",y/line/c/12345", want: "12345line12345line12345line12345"},

				// The final, out-of-the-for-loop edit  is out of order.
				{edit: ",y/^.*$/ .+/line/d", err: "out of order"},
			},
		},
		{
			name: "sequence",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: "1{", err: "unclosed {"},
				{edit: "1{p}", err: "expected end-of-input"},
				{edit: "1{ }", want: "line1\nline2\nline3"},
				{edit: "1{\n\n}", want: "line1\nline2\nline3"},
				{
					edit: `
							1,2{
								,d
								,d
							}
						`,
					err: "out of order",
				},
				{
					edit:  "1{p\n}",
					want:  "line1\nline2\nline3",
					print: "line1\n",
				},
				{
					edit:  "1{\n\np\n}",
					want:  "line1\nline2\nline3",
					print: "line1\n",
				},
				{
					edit: `
							1,2{
								p
								d
							}
						`,
					want:  "line3",
					print: "line1\nline2\n",
				},
				{
					edit: `
							,{
								1d
								2d
							}
						`,
					want: "line3",
				},
				{
					edit: `
						,{
							1a/Hello, World\n/
							2d
						}
					`,
					want: "line1\nHello, World\nline3",
				},
				{
					// Empty lines.
					edit: `
						,{

							1a/Hello, World\n/

							2d

						}
					`,
					want: "line1\nHello, World\nline3",
				},
			},
		},
		{
			name: "pipe to",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: ",>", err: "expected command"},
				{
					edit: ",> notfound",
					err:  "exit status.*",
				},
				{
					edit:  ",> cat",
					want:  "line1\nline2\nline3",
					print: "line1\nline2\nline3",
				},
				{
					edit:  "2> cat",
					want:  "line1\nline2\nline3",
					print: "line2\n",
				},
				{
					edit:  "0> cat",
					want:  "line1\nline2\nline3",
					print: "",
				},
			},
		},
		{
			name: "pipe from",
			str:  "line1\nline2\nline3",
			cases: []testCase{
				{edit: ",<", err: "expected command"},
				{
					edit: ",< notfound",
					err:  "exit status.*",
				},
				{
					edit: ",< echo line0",
					want: "line0\n",
				},
				{
					edit: "2< echo line0",
					want: "line1\nline0\nline3",
				},
				{
					edit:  "0< echo line0",
					want:  "line0\nline1\nline2\nline3",
					print: "",
				},
			},
		},
		{
			name: "pipe",
			str:  "line1\nline2\nline3\n",
			cases: []testCase{
				{edit: ",|", err: "expected command"},
				{
					edit: ",| notfound",
					err:  "exit status.*",
				},
				{
					edit: ",| sed 's/line/LINE/'",
					want: "LINE1\nLINE2\nLINE3\n",
				},
				{
					edit: "2| sed 's/line/LINE/'",
					want: "line1\nLINE2\nline3\n",
				},
				{
					edit: "0| sed 's/line/LINE/'",
					want: "line1\nline2\nline3\n",
				},
				{
					// This is an attempt to verify that the command
					// is indeed interpreted by -c of some shell
					// by assuming that all shells can interpret |.
					edit: "2| sed 's/line/LINE/' | sed 's/2/0/'",
					want: "line1\nLINE0\nline3\n",
				},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) { runEditTest(t, test) })
	}
}

type test struct {
	name  string
	str   string
	dot   Dot
	cases []testCase
}

type testCase struct {
	edit  string
	want  string
	print string // regex
	err   string // regex
}

func runAddrTest(t *testing.T, test test) {
	t.Helper()
	ro := rope.New(test.str)
	for _, c := range test.cases {
		switch got, err := Addr(test.dot, c.edit, ro); {
		case c.err == "" && err == nil:
			str := rope.Slice(ro, got[0], got[1]).String()
			if str != c.want {
				t.Errorf("(%q).Addr(%q)=%q, want %q",
					test.str, c.edit, str, c.want)
			}

		case c.err == "" && err != nil:
			t.Errorf("(%q).Addr(%q)=_,%v, want nil", test.str, c.edit, err)

		case c.err != "" && err == nil:
			t.Errorf("(%q).Addr(%q)=_,nil, want matching %q",
				test.str, c.edit, c.err)

		default: // c.err != " && err != nil:
			if !match(c.err, err.Error()) {
				t.Errorf("(%q).Addr(%q)=_,%q, want matching %q",
					test.str, c.edit, err.Error(), c.err)
			}
		}
	}
}

func runEditTest(t *testing.T, test test) {
	t.Helper()
	ro := rope.New(test.str)
	for _, c := range test.cases {
		var print strings.Builder
		switch diff, err := Edit(test.dot, c.edit, &print, ro); {
		case c.err == "" && err == nil:
			text, undo := diff.Apply(ro)
			if text.String() != c.want {
				t.Errorf("(%q).Edit(%q) buf=%q want buf=%q",
					test.str, c.edit, text.String(), c.want)
			}
			if !match(c.print, print.String()) {
				t.Errorf("(%q).Edit(%q) print=%q want matching %q",
					test.str, c.edit, print.String(), c.print)
			}
			if undone, _ := undo.Apply(text); undone.String() != test.str {
				t.Errorf("(%q).Edit(%q) undo=%q, want %q",
					test.str, c.edit, undone.String(), test.str)
			}

		case c.err == "" && err != nil:
			t.Errorf("(%q).Edit(%q)=_,%v, want nil", test.str, c.edit, err)

		case c.err != "" && err == nil:
			t.Errorf("(%q).Edit(%q)=_,nil, want matching %q",
				test.str, c.edit, c.err)

		default: // c.err != " && err != nil:
			if !match(c.err, err.Error()) {
				t.Errorf("(%q).Edit(%q)=_,%q, want matching %q",
					test.str, c.edit, err.Error(), c.err)
			}
			if !match(c.print, print.String()) {
				t.Errorf("(%q).Edit(%q) print=%q, want matthing %q",
					test.str, c.edit, print.String(), c.print)
			}
		}
	}
}

func match(re, str string) bool {
	return regexp.MustCompile(re).MatchString(str)
}
