package rope

import (
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestEmpty(t *testing.T) {
	e := Empty()
	if s := e.String(); s != "" {
		t.Errorf("Empty().String()=%q, want \"\"", s)
	}
	if l := e.Len(); l != 0 {
		t.Errorf("Empty().Len()=%d, want 0", l)
	}
}

func TestNew(t *testing.T) {
	tests := []string{
		"",
		"Hello, World",
		"Hello, 世界",
		strings.Repeat("Hello, 世界", smallSize*2/len("Hello, 世界")),
	}
	for _, test := range tests {
		r := New(test)
		if got := r.String(); got != test {
			t.Errorf("New(%q)=%q", test, got)
		}
		if r.Len() != int64(len(test)) {
			t.Errorf("New(%q).Len()=%d, want %d", test, r.Len(), len(test))
		}
	}
}

func TestReadFrom(t *testing.T) {
	tests := []string{
		"",
		"Hello, World",
		"Hello, 世界",
		strings.Repeat("Hello, 世界", smallSize*2/len("Hello, 世界")),
	}
	for _, test := range tests {
		r, err := ReadFrom(strings.NewReader(test))
		if err != nil {
			t.Errorf("ReadFrom(%q)=_,%v", test, err)
			continue
		}
		if got := r.String(); got != test {
			t.Errorf("ReadFrom(%q)=%q", test, got)
		}
		if r.Len() != int64(len(test)) {
			t.Errorf("New(%q).Len()=%d, want %d", test, r.Len(), len(test))
		}
	}
}

func TestAppendEmpty(t *testing.T) {
	r := Append(Empty(), New("x"))
	l, ok := r.(*leaf)
	if !ok || l.text != "x" {
		t.Errorf(`Append(Empty(), New("x"))=%v, wanted &leaf{"x"}`, r)
	}

	r = Append(New("x"), Empty())
	l, ok = r.(*leaf)
	if !ok || l.text != "x" {
		t.Errorf(`Append(New("x"), Empty())=%v, wanted &leaf{"x"}`, r)
	}
}

func TestAppendSmall(t *testing.T) {
	xs := strings.Repeat("x", smallSize-1)

	r := Append(New(xs), New("y"))
	l, ok := r.(*leaf)
	if !ok || l.text != xs+"y" {
		t.Errorf(`Append(New(%q), New("y"))=%#v, wanted &leaf{%q}`, xs, r, xs+"y")
	}

	r = Append(New("y"), New(xs))
	l, ok = r.(*leaf)
	if !ok || l.text != "y"+xs {
		t.Errorf(`Append(New("y"), New(%q))=%#v, wanted &leaf{%q}`, xs, r, "y"+xs)
	}
}

func TestAppendSmallNiece(t *testing.T) {
	// If the right child of the left node is a leaf,
	// and appending to it would keep the size
	// beneath smallSize, then just append to it.

	xs := strings.Repeat("x", smallSize-1)
	niece := &node{
		left:  &leaf{text: "w"},
		right: &leaf{text: xs},
		len:   smallSize,
	}

	expect := &node{
		left:  &leaf{text: "w"},
		right: &leaf{text: xs + "y"},
		len:   smallSize + 1,
	}
	if r := Append(niece, New("y")); !reflect.DeepEqual(r, expect) {
		t.Errorf("got %#v (%s), want %#v", r, r, expect)
	}
}

func TestSplit(t *testing.T) {
	for i := 0; i < len(deepText); i++ {
		want := [2]string{deepText[:i], deepText[i:]}
		r0, r1 := Split(deepRope, int64(i))
		got := [2]string{r0.String(), r1.String()}
		if got != want {
			t.Errorf("Split(%q, %d)=%q, want %q", deepRope, i, got, want)
		}
	}
}

func TestDelete(t *testing.T) {
	for i := 0; i < len(deepText); i++ {
		for n := 1; n < len(deepText)-i; n++ {
			want := deepText[:i] + deepText[i+n:]
			r := Delete(deepRope, int64(i), int64(n))
			if got := r.String(); got != want {
				t.Errorf("Delete(%q, %d, %d)=%q, want %q",
					deepRope, i, n, got, want)
			}
		}
	}
}

func TestInsert(t *testing.T) {
	big := strings.Repeat("x", smallSize*2)
	small := "x"
	for i := 0; i < len(deepText); i++ {
		for _, ins := range [...]string{big, small} {
			want := deepText[:i] + ins + deepText[i:]
			r := Insert(deepRope, int64(i), New(ins))
			if got := r.String(); got != want {
				t.Errorf("Insert(%q, %d, %q)=%q, want %q",
					deepRope, i, ins, got, want)
			}
		}
	}
}

func TestSlice(t *testing.T) {
	for i := 0; i < len(deepText); i++ {
		for j := i; j < len(deepText); j++ {
			want := deepText[i:j]
			r := Slice(deepRope, int64(i), int64(j))
			if got := r.String(); got != want {
				t.Errorf("Slice(%q, %d, %d)=%q, want %q",
					deepRope, i, j, got, want)
			}
		}
	}
}

func TestRead(t *testing.T) {
	tests := []struct {
		rope Rope
		text string
	}{
		{rope: New(""), text: ""},
		{rope: New("Hello, World"), text: "Hello, World"},
		{rope: New("Hello, 世界"), text: "Hello, 世界"},
		{rope: New(longText), text: longText},
		{rope: smallRope, text: smallText},
		{rope: deepRope, text: deepText},
	}
	for _, test := range tests {
		data, err := ioutil.ReadAll(NewReader(test.rope))
		if err != nil {
			t.Errorf("NewReader(%q).Read() error: %v",
				test.text, err)
			continue
		}
		if got := string(data); got != test.text {
			t.Errorf("NewReader(%q).Read()=%q", test.text, got)
		}
	}
}

func TestReadByte(t *testing.T) {
	tests := []struct {
		rope Rope
		text string
	}{
		{rope: New(""), text: ""},
		{rope: New("Hello, World"), text: "Hello, World"},
		{rope: New("Hello, 世界"), text: "Hello, 世界"},
		{rope: New(longText), text: longText},
		{rope: smallRope, text: smallText},
		{rope: deepRope, text: deepText},
	}
	for _, test := range tests {
		data, err := readAllByte(NewReader(test.rope))
		if err != nil {
			t.Errorf("NewReader(%q).ReadByte() error: %v",
				test.text, err)
			continue
		}
		if got := string(data); got != test.text {
			t.Errorf("NewReader(%q).ReadByte()=%q", test.text, got)
		}
	}
}

func TestReadRune(t *testing.T) {
	tests := []struct {
		rope Rope
		text string
	}{
		{rope: New(""), text: ""},
		{rope: New("Hello, World"), text: "Hello, World"},
		{rope: New("Hello, 世界"), text: "Hello, 世界"},
		{rope: New(longText), text: longText},
		{rope: smallRope, text: smallText},
		{rope: deepRope, text: deepText},
		{
			// Three bad UTF8 bytes give three replacement characters.
			rope: New("\x80\x80\x80abc"),
			text: repl + repl + repl + "abc",
		},
		{
			// The first two bytes of a 3-byte UTF8 character
			// give two replacement runes.
			rope: New("\xE2\x98abc"), // \xE2\x98\xBA == ☺
			text: repl + repl + "abc",
		},
		{
			// Interleaved good and bad bytes.
			rope: New("\x80α\x80β\x80ξ"),
			text: repl + "α" + repl + "β" + repl + "ξ",
		},
	}
	for _, test := range tests {
		got, err := readAllRune(NewReader(test.rope))
		if err != nil {
			t.Errorf("NewReader(%q).ReadRune() error: %v",
				test.text, err)
			continue
		}
		if got != test.text {
			t.Errorf("NewReader(%q).ReadRune()=%q", test.text, got)
		}
	}
}

func TestReadRuneThenRead(t *testing.T) {
	// Contains the first two bytes of a 3-byte UTF8 rune.
	rope := New("\xE2\x98abc")
	r := NewReader(rope)

	// Read the first byte of the bad rune.
	ru, w, err := r.ReadRune()
	if ru != utf8.RuneError || w != 1 || err != nil {
		t.Fatalf("r.ReadRune()=%x,%d,%v, want %x,1,nil", ru, w, err, repl)
	}

	// Read the next byte with Read
	var b [1]byte
	n, err := r.Read(b[:])
	if n != 1 || b[0] != 0x98 || err != nil {
		t.Fatalf("r.Read(1)=%x,%d,%v, want 0x98,1,nil", b[0], n, err)
	}

	// Now read the next rune.
	ru, w, err = r.ReadRune()
	if ru != 'a' || w != 1 || err != nil {
		t.Fatalf("r.ReadRune()=%c,%d,%v, want 'a',1,nil", ru, w, err)
	}
}

func TestReverseRead(t *testing.T) {
	tests := []struct {
		rope Rope
		text string
	}{
		{rope: New(""), text: ""},
		{rope: New("Hello, World"), text: reverseBytes("Hello, World")},
		{rope: New("Hello, 世界"), text: reverseBytes("Hello, 世界")},
		{rope: New(longText), text: reverseBytes(longText)},
		{rope: smallRope, text: reverseBytes(smallText)},
		{rope: deepRope, text: reverseBytes(deepText)},
	}
	for _, test := range tests {
		data, err := ioutil.ReadAll(NewReverseReader(test.rope))
		if err != nil {
			t.Errorf("NewReverseReader(%q).Read() error: %v", test.text, err)
			continue
		}
		if got := string(data); got != test.text {
			t.Errorf("NewReverseReader(%q).Read()=%q", test.text, got)
		}
	}
}

func TestReverseReadByte(t *testing.T) {
	tests := []struct {
		rope Rope
		text string
	}{
		{rope: New(""), text: ""},
		{rope: New("Hello, World"), text: reverseBytes("Hello, World")},
		{rope: New("Hello, 世界"), text: reverseBytes("Hello, 世界")},
		{rope: New(longText), text: reverseBytes(longText)},
		{rope: smallRope, text: reverseBytes(smallText)},
		{rope: deepRope, text: reverseBytes(deepText)},
	}
	for _, test := range tests {
		data, err := readAllByte(NewReverseReader(test.rope))
		if err != nil {
			t.Errorf("NewReverseReader(%q).ReadByte() error: %v",
				test.text, err)
			continue
		}
		if got := string(data); got != test.text {
			t.Errorf("NewReverseReader(%q).ReadByte()=%q", test.text, got)
		}
	}
}

func TestReverseReaderReadRune(t *testing.T) {
	tests := []struct {
		rope Rope
		text string
	}{
		{rope: New(""), text: ""},
		{rope: New("Hello, World"), text: reverseRunes("Hello, World")},
		{rope: New("Hello, 世界"), text: reverseRunes("Hello, 世界")},
		{rope: New(longText), text: reverseRunes(longText)},
		{rope: smallRope, text: reverseRunes(smallText)},
		{rope: deepRope, text: reverseRunes(deepText)},
		{
			// Three bad UTF8 bytes give three replacement characters.
			rope: New("\x80\x80\x80abc"),
			text: "cba" + repl + repl + repl,
		},
		{
			// The first two bytes of a 3-byte UTF8 character
			// give two replacement runes.
			rope: New("\xE2\x98abc"), // \xE2\x98\xBA == ☺
			text: "cba" + repl + repl,
		},
		{
			// Interleaved good and bad bytes.
			rope: New("\x80a\x80b\x80c"),
			text: "c" + repl + "b" + repl + "a" + repl,
		},
		{
			// Interleaved good and bad bytes.
			rope: New("\x80α\x80β\x80ξ"),
			text: "ξ" + repl + "β" + repl + "α" + repl,
		},
	}
	for _, test := range tests {
		got, err := readAllRune(NewReverseReader(test.rope))
		if err != nil {
			t.Errorf("NewReverseReader(%q).ReadRune() error: %v",
				test.text, err)
			continue
		}
		if got != test.text {
			t.Errorf("NewReverseReader(%q).ReadRune()=%q", test.text, got)
		}
	}
}

func TestReverseReadRuneThenRead(t *testing.T) {
	// The suffix is the first two bytes of a 3-byte UTF8 rune.
	rope := New("abc\xE2\x98")
	r := NewReverseReader(rope)

	// Read the last byte of the bad rune.
	ru, w, err := r.ReadRune()
	if ru != utf8.RuneError || w != 1 || err != nil {
		t.Fatalf("r.ReadRune()=%x,%d,%v, want %x,1,nil", ru, w, err, repl)
	}

	// Read the next byte with Read
	var b [1]byte
	n, err := r.Read(b[:])
	if n != 1 || b[0] != 0xE2 || err != nil {
		t.Fatalf("r.Read(1)=%x,%d,%v, want 0xE2,1,nil", b[0], n, err)
	}

	// Now read the next rune.
	ru, w, err = r.ReadRune()
	if ru != 'c' || w != 1 || err != nil {
		t.Fatalf("r.ReadRune()=%c,%d,%v, want 'c',1,nil", ru, w, err)
	}
}

func readAllByte(r io.ByteReader) ([]byte, error) {
	var data []byte
	for {
		b, err := r.ReadByte()
		if err == io.EOF {
			return data, nil
		}
		if err != nil {
			return nil, err
		}
		data = append(data, b)
	}
}

func readAllRune(r io.RuneReader) (string, error) {
	var data []rune
	for {
		switch ru, _, err := r.ReadRune(); {
		case err == io.EOF:
			return string(data), nil
		case err != nil:
			return "", err
		default:
			data = append(data, ru)
		}
	}
}

func reverseBytes(str string) string {
	bs := []byte(str)
	for i := 0; i < len(str)/2; i++ {
		bs[i], bs[len(bs)-1-i] = bs[len(bs)-1-i], bs[i]
	}
	return string(bs)
}

func reverseRunes(str string) string {
	rs := []rune(str)
	for i := 0; i < len(rs)/2; i++ {
		rs[i], rs[len(rs)-1-i] = rs[len(rs)-1-i], rs[i]
	}
	return string(rs)
}

const repl = "\xEF\xBF\xBD"

var (
	longText = strings.Repeat("Hello, 世界", smallSize*2/len("Hello, 世界"))

	smallRope, smallText = func() (Rope, string) {
		const text = "Hello, 世界"
		return &leaf{text: text}, text
	}()

	deepRope, deepText = func() (Rope, string) {
		r := &node{
			left: &leaf{},
			right: &node{
				left: &node{
					left: &node{
						left: &leaf{text: "H"},
						right: &node{
							left:  &leaf{text: "e"},
							right: &leaf{},
							len:   1,
						},
						len: 2,
					},
					right: &node{
						left:  &leaf{text: "l"},
						right: &leaf{text: "l"},
						len:   2,
					},
					len: 4,
				},
				right: &node{
					left: &node{
						left:  &leaf{text: "o"},
						right: &leaf{text: ", "},
						len:   3,
					},
					right: &node{
						left: &node{
							left:  &leaf{text: "World"},
							right: &leaf{text: "!"},
							len:   6,
						},
						right: &leaf{},
						len:   6,
					},
					len: 8,
				},
				len: 12,
			},
			len: 12,
		}
		return r, "Hello, World!"
	}()
)

// Some old test that tests who-knows-what.
func TestAppendSplit(t *testing.T) {
	tests := []string{
		"",
		"Hello, World",
		"Hello, 世界",
		strings.Repeat("Hello, 世界", smallSize*2/len("Hello, 世界")),
	}
	for _, test := range tests {
		for i := range test {
			r := Append(New(test[:i]), New(test[i:]))
			ok := true
			if got := r.String(); got != test {
				t.Errorf("Append(%q, %q)=%q", test[:i], test[i:], got)
				ok = false
			}
			if r.Len() != int64(len(test)) {
				t.Errorf("Append(%q, %q).Len()=%d, want %d",
					test[:i], test[i:], r.Len(), len(test))
				ok = false
			}
			if !ok {
				continue
			}

			for j := range test {
				left, right := Split(r, int64(j))
				gotLeft := left.String()
				gotRight := right.String()
				if gotLeft != test[:j] || gotRight != test[j:] {
					t.Errorf("Split(Append(%q, %q))=%q,%q",
						test[:i], test[i:], gotLeft, gotRight)
				}
				if left.Len() != int64(j) {
					t.Errorf("Split(Append(%q, %q)).left.Len()=%d, want %d",
						test[:i], test[i:], left.Len(), j)
				}
				if right.Len() != int64(len(test)-j) {
					t.Errorf("Split(Append(%q, %q)).right.Len()=%d, want %d",
						test[:i], test[i:], right.Len(), len(test)-j)
				}
			}
		}
	}
}
