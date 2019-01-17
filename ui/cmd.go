package ui

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/eaburns/T/edit"
	"github.com/eaburns/T/re1"
	"github.com/eaburns/T/rope"
)

// execCmd handles 2-click text.
// c is non-nil
// s may be nil
func execCmd(c *Col, s *Sheet, text string) error {
	switch cmd, _ := splitCmd(text); cmd {
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
		if text == "" {
			return nil
		}
		if isDir, err := openDir(c, s, text); isDir {
			return err
		}
		go func() {
			if err := shellCmd(c.win, text); err != nil {
				c.win.OutputString(err.Error())
			}
		}()
		return nil
	}
	return nil
}

func shellCmd(w *Win, text string) error {
	// TODO: set 2-click shell command CWD to the sheet's directory.
	// If executed from outside of a sheet, then don't set it specifically.
	cmd := exec.Command("sh", "-c", text)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stderr.Close()
		return err
	}
	if err := cmd.Start(); err != nil {
		stderr.Close()
		stdout.Close()
		return err
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go pipeOutput(&wg, w, stdout)
	go pipeOutput(&wg, w, stderr)
	wg.Wait()
	return cmd.Wait()
}

func pipeOutput(wg *sync.WaitGroup, w *Win, pipe io.Reader) {
	defer wg.Done()
	var buf [4096]byte
	for {
		n, err := pipe.Read(buf[:])
		if n > 0 {
			w.OutputBytes(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

func lookText(c *Col, s *Sheet, text string) error {
	if text == "" {
		return nil
	}

	path, err := abs(s, text)
	if err != nil {
		setLook(c, s, text)
		return nil
	}

	if focusSheet(c.win, path) {
		return nil
	}
	if focusSheet(c.win, ensureTrailingSlash(path)) {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		setLook(c, s, text)
		return nil
	}
	defer f.Close()
	s = NewSheet(c.win, path)
	if err := get(s, f); err != nil {
		return err
	}
	c.Add(s)
	return nil
}

func focusSheet(w *Win, title string) bool {
	for _, c := range w.cols {
		for _, r := range c.rows {
			s, ok := r.(*Sheet)
			if !ok || s.Title() != title {
				continue
			}
			setWinFocus(w, c)
			setColFocus(c, s)
			return true
		}
	}
	return false
}

func setLook(c *Col, s *Sheet, text string) {
	// TODO: 3-clicking a non-file should highlight matches in the sheet.
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
