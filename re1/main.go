// +build ignore

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/eaburns/T/re1"
	"github.com/eaburns/T/rope"
)

var (
	r = flag.Bool("r", false, "reverse")
	s = flag.Int("s", 0, "start")
	e = flag.Int("e", -1, "end")
)

func main() {
	flag.Parse()
	re1.Debug = true
	re, rest, err := re1.New(flag.Args()[0], re1.Opts{Reverse: *r})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if rest != "" {
		fmt.Println("rest:", rest)
	}
	if re == nil {
		fmt.Println("no regexp")
		os.Exit(1)
	}

	str := flag.Args()[1]
	if *r {
		ro := rope.New(str)
		if *e < 0 {
			*e = int(ro.Len())
		}
		m := re.FindReverseInRope(ro, int64(*s), int64(*e))
		fmt.Println(m)
		for i := 0; i < len(m); i += 2 {
			s, e := int(m[i]), int(m[i+1])
			if s < 0 || e < 0 {
				fmt.Println("\"\"")
				continue
			}
			fmt.Println(str[s:e])
		}
	} else {
		m := re.Find(strings.NewReader(str))
		fmt.Println(m)
		for i := 0; i < len(m); i += 2 {
			s, e := int(m[i]), int(m[i+1])
			if s < 0 || e < 0 {
				fmt.Println("\"\"")
				continue
			}
			fmt.Println(str[s:e])
		}
	}
}

func reverse(str string) (io.RuneReader, int) {
	n := utf8.RuneCountInString(str)
	return &revReader{str: str, i: n}, n
}

type revReader struct {
	str string
	i   int
}

func (rr *revReader) ReadRune() (rune, int, error) {
	if rr.i == 0 {
		return 0, 0, io.EOF
	}
	r, w := utf8.DecodeLastRuneInString(rr.str[:rr.i])
	rr.i -= w
	return r, w, nil
}
