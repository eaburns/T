package ui

import (
	"strings"
)

// c is non-nil
// s may be nil
func execCmd(c *Col, s *Sheet, cmd string) {
	switch strings.TrimSpace(cmd) {
	case "DelCol":
		c.win.Del(c)
	case "AddCol":
		c.win.Add()
	case "Del":
		for _, r := range c.rows {
			if getSheet(r) == s {
				c.Del(r)
			}
		}
	case "Add":
		c.Add(NewSheet(c.win, ""))
	}
}
