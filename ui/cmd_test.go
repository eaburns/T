package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eaburns/T/rope"
)

func TestCmd_openDir(t *testing.T) {
	dir := tmpdir()
	defer os.RemoveAll(dir)
	touch(dir, "a")
	touch(dir, "b")
	touch(dir, "c")
	mkSubDir(dir, "1")
	mkSubDir(dir, "2")
	mkSubDir(dir, "3")

	const title = "/Users/testuser/some_non_directory_file.go"
	var (
		w = newTestWin()
		c = w.cols[0]
		s = NewSheet(w, title)
	)
	c.Add(s)

	if err := execCmd(c, s, dir); err != nil {
		t.Fatalf("execCmd(.., [%q], %q) failed with %v", title, dir, err)
	}

	var dirSheet *Sheet
	for _, row := range c.rows {
		if s, ok := row.(*Sheet); ok && s.Title() == ensureTrailingSlash(dir) {
			dirSheet = s
			break
		}
	}

	if dirSheet == nil {
		t.Fatalf("no new sheet")
	}

	const want = "1/\n2/\n3/\na\nb\nc\n"
	if s := dirSheet.body.text.String(); s != want {
		t.Errorf("body is %q, want %q\n", s, want)
	}
}

func TestCmd_addDir(t *testing.T) {
	dir := tmpdir()
	defer os.RemoveAll(dir)
	mkSubDir(dir, "sub")
	sub := filepath.Join(dir, "sub")
	touch(sub, "a")
	touch(sub, "b")
	touch(sub, "c")
	mkSubDir(sub, "1")
	mkSubDir(sub, "2")
	mkSubDir(sub, "3")

	tests := []struct {
		name  string
		title string
		body  string
		exec  string
		want  string
	}{
		{
			name:  "empty body",
			title: dir,
			body:  "",
			exec:  sub,
			want: `sub/
sub/1/
sub/2/
sub/3/
sub/a
sub/b
sub/c
`,
		},
		{
			name:  "append to body",
			title: dir,
			body: `α/
β/
ξ/
`,
			exec: sub,
			want: `α/
β/
ξ/
sub/
sub/1/
sub/2/
sub/3/
sub/a
sub/b
sub/c
`,
		},
		{
			name:  "insert into body",
			title: dir,
			body: `α/
β/
sub/
ξ/
`,
			exec: sub,
			want: `α/
β/
sub/
sub/1/
sub/2/
sub/3/
sub/a
sub/b
sub/c
ξ/
`,
		},
		{
			name:  "rel-path",
			title: dir,
			body: `α/
β/
sub/
ξ/
`,
			exec: "sub",
			want: `α/
β/
sub/
sub/1/
sub/2/
sub/3/
sub/a
sub/b
sub/c
ξ/
`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			var (
				w = newTestWin()
				c = w.cols[0]
				s = NewSheet(w, ensureTrailingSlash(test.title))
			)
			s.SetText(rope.New(test.body))
			c.Add(s)
			if err := execCmd(c, s, test.exec); err != nil {
				t.Fatalf("execCmd(.., [%q], %q) failed with %v", test.title, test.exec, err)
			}
			if str := s.body.text.String(); str != test.want {
				t.Errorf("body=%q, want %q", str, test.want)
			}
		})
	}
}

func TestLook_newSheetAbsolute(t *testing.T) {
	dir := tmpdir()
	defer os.RemoveAll(dir)
	const text = "Hello, World!"
	path := filepath.Join(dir, "a")
	write(path, text)

	var (
		w = newTestWin()
		c = w.cols[0]
	)

	if err := lookText(c, nil, path); err != nil {
		t.Fatalf("lookText failed: %v", err)
	}

	if len(c.rows) != 2 {
		t.Fatalf("%d rows, wanted 2", len(c.rows))
	}
	s, ok := c.rows[1].(*Sheet)
	if !ok {
		t.Fatalf("row[1] is type %T, wanted *Sheet", c.rows[1])
	}
	if s.Title() != path {
		t.Errorf("sheet title is %q, wanted %q", s.Title(), path)
	}
	if txt := s.body.text.String(); txt != text {
		t.Errorf("sheet body is %q, wanted %q", txt, text)
	}
}

func TestLook_newSheetRelative(t *testing.T) {
	dir := tmpdir()
	defer os.RemoveAll(dir)
	const text = "Hello, World!"
	path := filepath.Join(dir, "a")
	write(path, text)

	var (
		w  = newTestWin()
		c  = w.cols[0]
		s0 = NewSheet(w, ensureTrailingSlash(dir))
	)
	c.Add(s0)

	if err := lookText(c, s0, "a"); err != nil {
		t.Fatalf("lookText failed: %v", err)
	}

	if len(c.rows) != 3 {
		t.Fatalf("%d rows, wanted 3", len(c.rows))
	}
	s, ok := c.rows[2].(*Sheet)
	if !ok {
		t.Fatalf("row[2] is type %T, wanted *Sheet", c.rows[2])
	}
	if s.Title() != path {
		t.Errorf("sheet title is %q, wanted %q", s.Title(), path)
	}
	if txt := s.body.text.String(); txt != text {
		t.Errorf("sheet body is %q, wanted %q", txt, text)
	}
}

func newTestCol(w *Win) *Col {
	c := NewCol(w)
	w.cols = append(w.cols, c)
	return c
}

func TestLook_focusExistingSheet(t *testing.T) {
	dir := tmpdir()
	defer os.RemoveAll(dir)
	const text = "Hello, World!"
	path := filepath.Join(dir, "a")

	var (
		w = newTestWin()
		s = NewSheet(w, path)
	)
	// Put the to-be-focused sheet in a new, out-of-focus column.
	c := NewCol(w)
	w.cols = append(w.cols, c)
	c.Add(s)
	// Focus the 0th row, which is the column background.
	c.Row.Focus(false)
	c.Row = c.rows[0]
	c.Row.Focus(true)

	if w.Col == c {
		t.Fatalf("bad setup, c sholud be out-of-focus")
	}
	if c.Row == s {
		t.Fatalf("bad setup, s sholud be out-of-focus")
	}

	if err := lookText(c, nil, path); err != nil {
		t.Fatalf("lookText failed: %v", err)
	}

	if len(c.rows) != 2 {
		t.Fatalf("%d rows, wanted 2", len(c.rows))
	}
	s, ok := c.rows[1].(*Sheet)
	if !ok {
		t.Fatalf("row[1] is type %T, wanted *Sheet", c.rows[2])
	}
	if s.Title() != path {
		t.Errorf("sheet title is %q, wanted %q", s.Title(), path)
	}
	// Don't test the win body. It was never "gotten", so it's just empty.
	if w.Col != c {
		t.Errorf("col not focused, wanted it to be focused")
	}
	if c.Row != s {
		t.Errorf("sheet not focused, wanted it to be focused")
	}
}
