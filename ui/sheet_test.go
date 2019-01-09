package ui

import (
	"testing"

	"github.com/eaburns/T/rope"
)

func TestTitle(t *testing.T) {
	tests := []struct {
		text  string
		title string
	}{
		{text: "", title: ""},
		{text: " ", title: ""},
		{text: " not title", title: ""},
		{text: "''", title: ""},
		{text: "'", title: ""},
		{text: "Hello |", title: "Hello"},
		{text: "/home/test/src/T |", title: "/home/test/src/T"},
		{text: "/home/test/src/T", title: "/home/test/src/T"},
		{text: `'Hello, World!'`, title: "Hello, World!"},
		{text: `'Hello, World!`, title: "Hello, World!"}, // no ' terminator
		{text: `'\\'`, title: `\`},
		{text: `'\''`, title: `'`},
	}

	for _, test := range tests {
		s := NewSheet(testWin, "")
		s.tag.text = rope.New(test.text)
		title := s.Title()
		if title != test.title {
			t.Errorf("got %q, want %q", title, test.title)
		}
	}
}

func TestSetTitle(t *testing.T) {
	tests := []struct {
		text  string
		title string
		want  string
	}{
		{text: "", title: "", want: ""},
		{text: "/User/T/src/github.com/T", title: "", want: ""},
		{text: "/User/T/src/github.com/T | Del", title: "", want: " | Del"},
		{text: ` | Del`, title: "", want: " | Del"},
		{text: `'' | Del`, title: "", want: " | Del"},
		{text: "", title: "", want: ""},
		{text: "``", title: "", want: ""},
		{text: "'Hello, World' | Del", title: "", want: " | Del"},
		{text: "'Hello, World'| Del", title: "", want: " | Del"},

		{text: " | Del", title: "/path/to/file", want: "/path/to/file | Del"},
		{text: " | Del", title: "Hello, World!", want: "'Hello, World!' | Del"},
		{text: "'xyz'| Del", title: "1 2 3", want: "'1 2 3' | Del"},
		{
			text:  "/path/to/file | Del",
			title: `'Hello, World!'`,
			want:  `'\'Hello, World!\'' | Del`,
		},
		{
			text:  "/path/to/file | Del",
			title: `'Hello,\World!'`,
			want:  `'\'Hello,\\World!\'' | Del`,
		},
	}

	for _, test := range tests {
		s := NewSheet(testWin, "")
		s.tag.text = rope.New(test.text)
		s.SetTitle(test.title)
		if got := s.tag.text.String(); got != test.want {
			t.Errorf("(%q).SetTitle(%q) = %q, want %q",
				test.text, test.title, got, test.want)
		}
	}
}
