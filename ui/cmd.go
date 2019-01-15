package ui

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/eaburns/T/edit"
	"github.com/eaburns/T/re1"
	"github.com/eaburns/T/rope"
)

// c is non-nil
// s may be nil
func execCmd(c *Col, s *Sheet, exec string) error {
	switch cmd, _ := splitCmd(exec); cmd {
	case "Del":
		if s == nil {
			c.win.Del(c)
			return nil
		}
		for _, r := range c.rows {
			if getSheet(r) == s {
				c.Del(r)
			}
		}

	case "NewCol":
		c.win.Add()

	case "NewRow":
		c.Add(NewSheet(c.win, ""))

	case "Get":
		if s == nil {
			return nil
		}
		return s.Get()

	case "Put":
		if s == nil {
			return nil
		}
		return s.Put()

	default:
		if isDir, err := openDir(c, s, exec); isDir {
			return err
		}
	}
	return nil
}

func openDir(c *Col, s *Sheet, path string) (bool, error) {
	var err error
	if path, err = abs(s, path); err != nil {
		return false, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return false, nil
	}
	defer f.Close()
	if st, err := f.Stat(); err != nil || !st.IsDir() {
		return st.IsDir(), err
	}

	if s != nil {
		title := s.Title()
		if r, _ := utf8.DecodeLastRuneInString(title); r == os.PathSeparator {
			if rel, err := filepath.Rel(title, path); err == nil {
				return true, addDir(s, f, rel)
			}
		}
	}

	s = NewSheet(c.win, path)
	if err := get(s, f); err != nil {
		return true, err
	}
	c.Add(s)
	return true, nil
}

func addDir(s *Sheet, f *os.File, rel string) error {
	r, err := readFromDir(rel, f)
	if err != nil {
		return err
	}
	rel = ensureTrailingSlash(rel)

	at := s.body.text.Len()
	if addr, ok := findDir(s.body.text, rel); ok {
		at = addr[1]
	} else {
		r = rope.Append(rope.New(rel+"\n"), r)
	}
	s.body.Change(edit.Diffs{{At: [2]int64{at, at}, Text: r}})
	showAddr(s.body, at)
	return nil
}

func findDir(r rope.Rope, dir string) ([2]int64, bool) {
	match := re1.Escape(dir)
	match = strings.Replace(match, "/", `\/`, -1)
	addr, err := edit.Addr([2]int64{0, r.Len()}, "/^"+match+"$\\n", r)
	if err != nil {
		return [2]int64{}, false
	}
	return addr, true
}

func abs(s *Sheet, path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	if s != nil {
		return filepath.Join(filepath.Dir(s.Title()), path), nil
	}
	return filepath.Abs(path)
}

func splitCmd(exec string) (string, string) {
	exec = strings.TrimSpace(exec)
	i := strings.IndexFunc(exec, unicode.IsSpace)
	if i < 0 {
		return exec, ""
	}
	return exec[:i], strings.TrimSpace(exec[i:])
}
