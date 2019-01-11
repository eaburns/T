package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/eaburns/T/edit"
	"github.com/eaburns/T/rope"
	"github.com/eaburns/T/syntax"
)

type highlighter struct {
	syntax.Tokenizer
}

func (h *highlighter) Update(hi []syntax.Highlight, diffs edit.Diffs, txt rope.Rope) (res []syntax.Highlight) {
	t0 := time.Now()
	defer func() {
		dur := time.Since(t0)
		if dur > time.Second {
			fmt.Println(dur, len(res))
		}
	}()
	if len(diffs) == 0 {
		hi = update(h.Tokenizer, hi, nil, txt)
		return hi
	}
	for _, diff := range diffs {
		var tail []syntax.Highlight
		if len(hi) > 0 {
			i := sort.Search(len(hi), func(i int) bool {
				return hi[i].At[1] > diff.At[0]
			})
			hi, tail = hi[:i:i], hi[i:]
		}
		for i := range tail {
			tail[i].At = diff.Update(tail[i].At)
		}
		for len(tail) > 0 && tail[0].At[0] == tail[0].At[1] {
			tail = tail[1:]
		}
		hi = update(h.Tokenizer, hi, tail, txt)
	}
	return hi
}

func update(tok syntax.Tokenizer, hi, tail []syntax.Highlight, txt rope.Rope) []syntax.Highlight {
	var at int64
	if len(hi) > 0 {
		at = hi[len(hi)-1].At[1]
	}
	for {
		h, ok := tok.NextToken(rope.Slice(txt, at, txt.Len()))
		if !ok {
			return append(hi, tail...)
		}
		h.At[0] += at
		h.At[1] += at
		if len(tail) > 0 && tail[0] == h {
			return append(hi, tail...)
		}
		for len(tail) > 0 && tail[0].At[0] < h.At[1] {
			tail = tail[1:]
		}
		at = h.At[1]
		hi = append(hi, h)
	}
}
