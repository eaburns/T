package rope

import "testing"

func TestIndexFunc(t *testing.T) {
	tests := []struct {
		str string
		r   rune
		at  int64
	}{
		{"", 'x', -1},
		{"abc", 'a', 0},
		{"abc", 'b', 1},
		{"abc", 'c', 2},
		{"abc", 'd', -1},
		{"☺x", 'x', int64(len("☺"))},
		{"abcabc", 'b', 1},
	}
	for _, test := range tests {
		ro := New(test.str)
		got := IndexFunc(ro, func(r rune) bool { return r == test.r })
		if got != test.at {
			t.Errorf("IndexFunc(%q, =%q)=%d, want %d",
				test.str, test.r, got, test.at)
		}
	}
}

func TestLastIndexFunc(t *testing.T) {
	tests := []struct {
		str string
		r   rune
		at  int64
	}{
		{"", 'x', -1},
		{"abc", 'a', 0},
		{"abc", 'b', 1},
		{"abc", 'c', 2},
		{"abc", 'd', -1},
		{"☺x", 'x', int64(len("☺"))},
		{"abcabc", 'b', 4},
	}
	for _, test := range tests {
		ro := New(test.str)
		got := LastIndexFunc(ro, func(r rune) bool { return r == test.r })
		if got != test.at {
			t.Errorf("LastIndexFunc(%q, =%q)=%d, want %d",
				test.str, test.r, got, test.at)
		}
	}
}
