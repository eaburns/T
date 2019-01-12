package re1

import (
	"reflect"
	"strings"
	"testing"
)

func TestUnion(t *testing.T) {
	tests := []struct {
		name string
		res  []string
		text string
		want []int64
	}{
		{
			name: "one regexp mismatch",
			res:  []string{"abc"},
			text: "abx",
			want: nil,
		},
		{
			name: "one regexp match",
			res:  []string{"abc"},
			text: "abc",
			want: []int64{0, 3, 0},
		},
		{
			name: "one regexp submatch",
			res:  []string{"((a)(b)(c))"},
			text: "abc",
			want: []int64{0, 3, 0, 3, 0, 1, 1, 2, 2, 3, 0},
		},
		{
			name: "two regexp mismatch",
			res:  []string{"a+bc", "d+ef"},
			text: "xxxxx",
			want: nil,
		},
		{
			name: "two regexp first match",
			res:  []string{"a+bc", "d+ef"},
			text: "aaaaabc",
			want: []int64{0, 7, 0},
		},
		{
			name: "two regexp first match",
			res:  []string{"a+bc", "d+ef"},
			text: "aaaaabc",
			want: []int64{0, 7, 0},
		},
		{
			name: "two regexp first submatch",
			res:  []string{"(a+)bc", "(d+)ef"},
			text: "aaaaabc",
			want: []int64{0, 7, 0, 5, 0},
		},
		{
			name: "two regexp second match",
			res:  []string{"a+bc", "d+ef"},
			text: "dddddef",
			want: []int64{0, 7, 1},
		},
		{
			name: "two regexp second submatch",
			res:  []string{"(a+)bc", "(d+)ef"},
			text: "dddddef",
			want: []int64{0, 7, 0, 5, 1},
		},
		{
			name: "left fewer than right sub-expressions",
			res:  []string{"a(b+)c", "(d)(e)(f)"},
			text: "abbbbbc",
			// (d)(e)(f) has three sub-expressions,
			// but a(b+)c was the matching expr, and it only has one,
			// so only its one is used.
			want: []int64{0, 7, 1, 6, -1, -1, -1, -1, 0},
		},
		{
			name: "right fewer than right sub-expressions",
			res:  []string{"(a)(b)(c)", "d(e+)f"},
			text: "deeeeef",
			// (a)(b)(c) has three sub-expressions,
			// but d(e+)f was the matching expr, and it only has one,
			// so only its one is used.
			want: []int64{0, 7, 1, 6, -1, -1, -1, -1, 1},
		},
		{
			name: "char class left matches",
			res:  []string{"[a-z]+", "[0-9]+"},
			text: "abcdefg",
			want: []int64{0, 7, 0},
		},
		{
			name: "char class right matches",
			res:  []string{"[a-z]+", "[0-9]+"},
			text: "0123456",
			want: []int64{0, 7, 1},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			var res []*Regexp
			var inputs []string
			for i, r := range test.res {
				re, residue, err := New(r, Opts{ID: i})
				if residue != "" || err != nil {
					t.Fatalf("New(%q, %d)=_,%q,%v, want _, \"\", nil",
						r, i, residue, err)
				}
				res = append(res, re)
				inputs = append(inputs, r)
			}
			u := Union(res...)
			t.Logf("Union(%q)=%q\n", inputs, u.source)
			got := u.Find(strings.NewReader(test.text))
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Union(%q).Find(%q)=%v, want %v",
					inputs, test.text, got, test.want)
			}
		})
	}
}

func TestUnionNil(t *testing.T) {
	if x := Union(); x != nil {
		t.Errorf("Union()=%v, want nil", x)
	}
}
