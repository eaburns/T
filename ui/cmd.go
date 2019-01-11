package ui

import (
	"fmt"
	"strings"
	"unicode"
)

// c is non-nil
// s may be nil
func execCmd(c *Col, s *Sheet, exec string) {
	switch cmd, _ := splitCmd(exec); cmd {
	case "Del":
		if s == nil {
			c.win.Del(c)
			return
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
			return
		}
		if err := s.Get(); err != nil {
			// TODO: report command errors to an errors window.
			fmt.Println(err.Error())
		}

	case "Put":
		if s == nil {
			return
		}
		if err := s.Put(); err != nil {
			// TODO: report command errors to an errors window.
			fmt.Println(err.Error())
		}
	}
}

func splitCmd(exec string) (string, string) {
	exec = strings.TrimSpace(exec)
	i := strings.IndexFunc(exec, unicode.IsSpace)
	if i < 0 {
		return exec, ""
	}
	return exec[:i], strings.TrimSpace(exec[i:])
}
