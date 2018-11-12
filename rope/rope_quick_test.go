package rope

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

var quickConfig *quick.Config

func TestMain(m *testing.M) {
	seed := time.Now().Unix()
	if s, err := strconv.ParseInt(os.Getenv("QUICK_TEST_SEED"), 10, 64); err == nil {
		seed = s
	}
	fmt.Println("seed", seed)
	quickConfig = &quick.Config{
		MaxCount: 1000,
		Rand:     rand.New(rand.NewSource(seed)),
	}
	os.Exit(m.Run())
}

func TestQuickAppend(t *testing.T) {
	err := quick.CheckEqual(
		func(ss [][]byte) string {
			var accum strings.Builder
			for _, s := range ss {
				accum.Write(s)
			}
			return accum.String()
		},
		func(ss [][]byte) string {
			accum := Empty()
			for _, s := range ss {
				accum = Append(accum, New(string(s)))
			}
			return accum.String()
		},
		quickConfig)
	if err != nil {
		t.Error(err)
	}
}

func TestQuickSplit(t *testing.T) {
	err := quick.CheckEqual(
		func(ss [][]byte, i int) [2]string {
			var accum strings.Builder
			for _, s := range ss {
				accum.Write(s)
			}
			str := accum.String()
			i = randLen(i, len(str))
			return [2]string{str[:i], str[i:]}
		},
		func(ss [][]byte, i int) [2]string {
			accum := Empty()
			for _, s := range ss {
				accum = Append(accum, New(string(s)))
			}
			i = randLen(i, int(accum.Len()))
			l, r := Split(accum, int64(i))
			return [2]string{l.String(), r.String()}
		},
		quickConfig)
	if err != nil {
		t.Error(err)
	}
}

func TestQuickDelete(t *testing.T) {
	err := quick.CheckEqual(
		func(ss [][]byte, start, n int) string {
			var accum strings.Builder
			for _, s := range ss {
				accum.Write(s)
			}
			str := accum.String()
			start = randLen(start, len(str))
			n = randLen(n, len(str)-start)
			return str[:start] + str[start+n:]
		},
		func(ss [][]byte, start, n int) string {
			accum := Empty()
			for _, s := range ss {
				accum = Append(accum, New(string(s)))
			}
			start = randLen(start, int(accum.Len()))
			n = randLen(n, int(accum.Len())-start)
			return Delete(accum, int64(start), int64(n)).String()
		},
		quickConfig)
	if err != nil {
		t.Error(err)
	}
}

func TestQuickInsert(t *testing.T) {
	err := quick.CheckEqual(
		func(ss [][]byte, ins []byte, i int) string {
			var accum strings.Builder
			for _, s := range ss {
				accum.Write(s)
			}
			str := accum.String()
			i = randLen(i, len(str))
			return str[:i] + string(ins) + str[i:]
		},
		func(ss [][]byte, ins []byte, i int) string {
			accum := Empty()
			for _, s := range ss {
				accum = Append(accum, New(string(s)))
			}
			i = randLen(i, int(accum.Len()))
			return Insert(accum, int64(i), New(string(ins))).String()
		},
		quickConfig)
	if err != nil {
		t.Error(err)
	}
}

func TestQuickSlice(t *testing.T) {
	err := quick.CheckEqual(
		func(ss [][]byte, start, n int) string {
			var accum strings.Builder
			for _, s := range ss {
				accum.Write(s)
			}
			str := accum.String()
			start = randLen(start, len(str))
			n = randLen(n, len(str)-start)
			return str[start : start+n]
		},
		func(ss [][]byte, start, n int) string {
			accum := Empty()
			for _, s := range ss {
				accum = Append(accum, New(string(s)))
			}
			start = randLen(start, int(accum.Len()))
			n = randLen(n, int(accum.Len())-start)
			return Slice(accum, int64(start), int64(start+n)).String()
		},
		quickConfig)
	if err != nil {
		t.Error(err)
	}
}

func TestQuickReadRune(t *testing.T) {
	err := quick.CheckEqual(
		func(ss []string, start, n int) string {
			var accum strings.Builder
			for _, s := range ss {
				accum.WriteString(s)
			}
			return accum.String()
		},
		func(ss []string, start, n int) string {
			accum := Empty()
			for _, s := range ss {
				accum = Append(accum, New(s))
			}
			str, err := readAllRune(NewReader(accum))
			if err != nil {
				panic(err.Error())
			}
			return str
		},
		quickConfig)
	if err != nil {
		t.Error(err)
	}
}

func TestQuickReverseReadRune(t *testing.T) {
	err := quick.CheckEqual(
		func(ss []string, start, n int) string {
			var accum strings.Builder
			for _, s := range ss {
				accum.WriteString(s)
			}
			return reverseRunes(accum.String())
		},
		func(ss []string, start, n int) string {
			accum := Empty()
			for _, s := range ss {
				accum = Append(accum, New(s))
			}
			str, err := readAllRune(NewReverseReader(accum))
			if err != nil {
				panic(err.Error())
			}
			return str
		},
		quickConfig)
	if err != nil {
		t.Error(err)
	}
}

func randLen(i, max int) int {
	if max == 0 {
		return 0
	}
	if i < 0 {
		i = -i
	}
	return i % max
}
