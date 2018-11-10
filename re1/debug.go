package re1

import (
	"fmt"
	"strconv"
	"strings"
)

// Debug enables debuging output for the vm.
var Debug = false

func debug(f string, args ...interface{}) {
	if Debug {
		fmt.Printf(f, args...)
	}
}

// DebugString returns a string of the regexp program for debugging.
func (re *Regexp) DebugString() string {
	var s strings.Builder
	s.WriteString(re.source)
	s.WriteRune('\n')
	for pc, instr := range re.prog {
		if pc > 0 {
			s.WriteRune('\n')
		}
		fmt.Fprintf(&s, "%4d:\t", pc)
		s.WriteString(instr.DebugString(re, pc))
	}
	return s.String()
}

func (instr instr) DebugString(re *Regexp, pc int) string {
	switch instr.op {
	case any:
		return "any"
	case nclass, class:
		s := "class"
		if instr.op == nclass {
			s = "nclass"
		}
		for _, c := range re.class[instr.arg] {
			s += " " + string([]rune{c[0]})
			if c[0] < c[1] {
				s += "-" + string([]rune{c[1]})
			}
		}
		return s
	case match:
		return "match"
	case jmp:
		return fmt.Sprintf("jmp %d", pc+instr.arg)
	case fork:
		return fmt.Sprintf("fork %d %d", pc+1, pc+instr.arg)
	case rfork:
		return fmt.Sprintf("rfork %d %d", pc+instr.arg, pc+1)
	case save:
		return fmt.Sprintf("save %d", instr.arg)
	case bol:
		return "bol"
	case eol:
		return "eol"
	default:
		return strconv.QuoteRune(rune(instr.op))
	}
}
