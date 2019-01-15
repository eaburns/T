package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eaburns/T/rope"
)

func newTestCol(w *Win) *Col { return NewCol(w) }

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
		c = newTestCol(w)
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
				c = newTestCol(w)
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
