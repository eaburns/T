package ui

import (
	"io/ioutil"
	"os"
	"path/filepath"
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

func TestSheetGet_TextFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "t")
	if err != nil {
		t.Fatalf("TempDir failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "file")

	const text = "Hello, World!"
	write(path, text)

	sh := NewSheet(testWin, path)
	if err := sh.Get(); err != nil {
		t.Fatalf("Get()=%v, want nil", err)
	}
	if s := sh.body.text.String(); s != text {
		t.Errorf("body text is %q, want %q", s, text)
	}
}

func TestSheetGet_Dir(t *testing.T) {
	dir, err := ioutil.TempDir("", "t")
	if err != nil {
		t.Fatalf("TempDir failed: %v\n", err)
	}
	defer os.RemoveAll(dir)

	touch(dir, "c")
	touch(dir, "b")
	touch(dir, "a")
	mkdir(dir, "3")
	mkdir(dir, "2")
	mkdir(dir, "1")

	sh := NewSheet(testWin, dir)
	if err := sh.Get(); err != nil {
		t.Fatalf("Get()=%v, want nil", err)
	}
	const text = "1/\n2/\n3/\na\nb\nc\n"
	if s := sh.body.text.String(); s != text {
		t.Errorf("body text is %q, want %q", s, text)
	}
}

func TestSheetPut(t *testing.T) {
	dir, err := ioutil.TempDir("", "t")
	if err != nil {
		t.Fatalf("TempDir failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "file")

	const text = "Hello, World!"
	sh := NewSheet(testWin, path)
	sh.SetText(rope.New(text))
	if err := sh.Put(); err != nil {
		t.Fatalf("Put()=%v, want nil", err)
	}
	if s := read(path); s != text {
		t.Errorf("read(%q)=%q, want %q", path, s, text)
	}
}

func read(path string) string {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(d)
}

func write(path, text string) {
	if err := ioutil.WriteFile(path, []byte(text), os.ModePerm); err != nil {
		panic(err)
	}
}

func touch(dir, file string) {
	f, err := os.Create(filepath.Join(dir, file))
	if err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
}

func mkdir(dir, subdir string) {
	if err := os.Mkdir(filepath.Join(dir, subdir), os.ModePerm); err != nil {
		panic(err)
	}
}
