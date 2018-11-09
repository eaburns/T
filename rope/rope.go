// Package rope implements Ropes, a copy-on-write, string-like data structure
// that is optimized for efficient modification of large sequences of bytes
// at the cost of more expensive random-access.
package rope

import (
	"io"
	"strings"
	"unicode/utf8"
)

// Rope is a copy-on-write string, optimized for concatenation and splitting.
type Rope interface {
	Len() int64
	String() string
	WriteTo(io.Writer) (int64, error)
}

type node struct {
	left, right Rope
	len         int64
}

func (n *node) Len() int64 { return n.len }

func (n *node) String() string {
	var s strings.Builder
	n.WriteTo(&s)
	return s.String()
}

func (n *node) WriteTo(w io.Writer) (int64, error) {
	nleft, err := n.left.WriteTo(w)
	if err != nil {
		return nleft, err
	}
	nright, err := n.right.WriteTo(w)
	return nleft + nright, err
}

type leaf struct {
	text string
}

func (l *leaf) Len() int64     { return int64(len(l.text)) }
func (l *leaf) String() string { return l.text }

func (l *leaf) WriteTo(w io.Writer) (int64, error) {
	num, err := io.WriteString(w, l.text)
	num64 := int64(num)
	return num64, err
}

// Empty returns an empty Rope.
func Empty() Rope { return New("") }

// New returns a new Rope of the given string.
func New(text string) Rope { return &leaf{text: text} }

// ReadFrom returns a new Rope containing
// all of the bytes read from a reader until io.EOF.
// On error, the returned Rope contains any bytes
// read from the Reader before the error.
// If no bytes were read, the Rope is empty.
func ReadFrom(r io.Reader) (Rope, error) {
	buf := make([]byte, 32*1024)
	rope := Empty()
	for {
		n, err := r.Read(buf)
		rope = Append(rope, New(string(buf[:n])))
		switch {
		case err == io.EOF:
			return rope, nil
		case err != nil:
			return rope, err
		}
	}
}

const smallSize = 32

// Append returns the concatenation of l and then r.
func Append(l, r Rope) Rope {
	switch {
	case l.Len() == 0:
		return r
	case r.Len() == 0:
		return l
	case l.Len()+r.Len() <= smallSize:
		return &leaf{text: l.String() + r.String()}
	}
	if l, ok := l.(*node); ok && l.right.Len()+r.Len() <= smallSize {
		return &node{
			left:  l.left,
			right: &leaf{text: l.right.String() + r.String()},
			len:   l.Len() + r.Len(),
		}
	}
	return &node{left: l, right: r, len: l.Len() + r.Len()}
}

// Split returns two new Ropes, the first contains the first i bytes,
// and the second contains the remaining.
// Split panics if i < 0 || i >= r.Len().
func Split(r Rope, i int64) (left, right Rope) {
	return split(r, i)
}

// Delete deletes n bytes from r beginning at index start.
// Delete panics if start < 0, n < 0, or start+n > r.Len().
func Delete(r Rope, start, n int64) Rope {
	r, l := split(r, start)
	_, l = split(l, start+n-r.Len())
	return Append(r, l)
}

// Insert inserts ins into r at index i.
// Insert panics if i < 0 or i > r.Len().
func Insert(r Rope, i int64, ins Rope) Rope {
	r, l := split(r, i)
	return Append(Append(r, ins), l)
}

// Slice returns a new Rope containing the bytes
// between start (inclusive) and end (exclusive).
// Slice panics if start < 0, end < start, or end > r.Len().
func Slice(r Rope, start, end int64) Rope {
	_, r = split(r, start)
	r, _ = split(r, end-start)
	return r
}

func split(rope Rope, i int64) (left, right Rope) {
	if i < 0 || i > rope.Len() {
		panic("index out of bounds")
	}
	switch rope := rope.(type) {
	case *leaf:
		return New(rope.text[:i]), New(rope.text[i:])
	case *node:
		switch {
		case i <= rope.left.Len():
			l, r := split(rope.left, i)
			return l, Append(r, rope.right)
		default:
			l, r := split(rope.right, i-rope.left.Len())
			return Append(rope.left, l), r
		}
	default:
		panic("impossible")
	}
}

// Reader implements io.Reader, io.ByteReader, and io.RuneReader,
// reading from the contents of a Rope.
type Reader struct {
	iter
	buf [utf8.UTFMax]byte
	n   int
}

// NewReader returns a new *Reader
// that reads the contents of the Rope.
func NewReader(rope Rope) *Reader {
	return &Reader{iter: iter{todo: []Rope{rope}}}
}

// Read reads into p and returns the number of bytes read.
// If there is nothing left to read, Read returns 0 and io.EOF.
// Read does not return errors other than io.EOF.
func (r *Reader) Read(p []byte) (int, error) {
	if r.n > 0 {
		n := copy(p, r.buf[:r.n])
		copy(r.buf[:], r.buf[n:])
		r.n -= n
		return n, nil
	}
	return r.read(p)
}

// ReadByte returns the next byte.
// If there are no more bytes to read, ReadByte returns 0 and io.EOF.
// ReadByte does not return errors other than io.EOF.
func (r *Reader) ReadByte() (byte, error) {
	var b [1]byte
	_, err := r.Read(b[:])
	return b[0], err
}

// ReadRune returns the next rune and it's byte-width.
// If the next bytes are not valid UTF8,
// ReadRune returns utf8.RuneError, 1.
// If there are no more bytes to read, ReadRune returns 0 and io.EOF.
// ReadRune does not return errors other than io.EOF.
func (r *Reader) ReadRune() (rune, int, error) {
	for invalidUTF8(r.buf[:r.n]) && r.n < len(r.buf) {
		n, err := r.read(r.buf[r.n:])
		r.n += n
		if err == io.EOF {
			break
		}
	}
	if r.n == 0 {
		return 0, 0, io.EOF
	}
	ru, w := utf8.DecodeRune(r.buf[:])
	r.n = copy(r.buf[:], r.buf[w:r.n])
	return ru, w, nil
}

func (r *Reader) read(p []byte) (int, error) {
	for len(r.text) == 0 {
		if !next(&r.iter, false) {
			return 0, io.EOF
		}
	}
	n := copy(p, r.text)
	r.text = r.text[n:]
	return n, nil
}

// ReverseReader implements
// io.Reader, io.ByteReader, and io.RuneReader,
// reading from the contents of a Rope in reverse.
type ReverseReader struct {
	iter
	buf [utf8.UTFMax]byte
	n   int
}

// NewReverseReader returns a new *ReverseReader
// that reads the contents of the Rope in reverse.
func NewReverseReader(rope Rope) *ReverseReader {
	return &ReverseReader{iter: iter{todo: []Rope{rope}}}
}

// Read reads into p and returns the number of bytes read.
// If there is nothing left to read, Read returns 0 and io.EOF.
// Read does not return errors other than io.EOF.
func (r *ReverseReader) Read(p []byte) (int, error) {
	if r.n > 0 {
		n := ypocByte(p, r.buf[:r.n])
		r.n -= n
		return n, nil
	}
	return r.read(p)
}

// ReadByte returns the next byte.
// If there are no more bytes to read, ReadByte returns 0 and io.EOF.
// ReadByte does not return errors other than io.EOF.
func (r *ReverseReader) ReadByte() (byte, error) {
	var b [1]byte
	_, err := r.Read(b[:])
	return b[0], err
}

// ReadRune returns the next rune and it's byte-width.
// If the next bytes are not valid UTF8,
// ReadRune returns utf8.RuneError, 1.
// If there are no more bytes to read, ReadRune returns 0 and io.EOF.
// ReadRune does not return errors other than io.EOF.
func (r *ReverseReader) ReadRune() (rune, int, error) {
	for invalidUTF8(r.buf[:r.n]) && r.n < len(r.buf) {
		r.buf[1], r.buf[2], r.buf[3] = r.buf[0], r.buf[1], r.buf[2]
		if _, err := r.read(r.buf[:1]); err == io.EOF {
			break
		}
		r.n++
	}
	if r.n == 0 {
		return 0, 0, io.EOF
	}
	ru, w := utf8.DecodeLastRune(r.buf[:r.n])
	r.n -= w
	return ru, w, nil
}

func invalidUTF8(p []byte) bool {
	r, _ := utf8.DecodeRune(p)
	return r == utf8.RuneError
}

func (r *ReverseReader) read(p []byte) (int, error) {
	for len(r.text) == 0 {
		if !next(&r.iter, true) {
			return 0, io.EOF
		}
	}
	n := ypocStr(p, r.text)
	r.text = r.text[:len(r.text)-n]
	return n, nil
}

func ypocStr(dst []byte, src string) int {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	for i := 0; i < n; i++ {
		dst[i] = src[len(src)-1-i]
	}
	return n
}

func ypocByte(dst []byte, src []byte) int {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	for i := 0; i < n; i++ {
		dst[i] = src[len(src)-1-i]
	}
	return n
}

type iter struct {
	todo []Rope
	text string
}

func next(it *iter, rev bool) bool {
	n := len(it.todo)
	if n == 0 {
		return false
	}
	rope := it.todo[n-1]
	it.todo = it.todo[:n-1]
	for {
		switch n := rope.(type) {
		case *leaf:
			it.text = n.text
			return true
		case *node:
			if rev {
				it.todo = append(it.todo, n.left)
				rope = n.right
			} else {
				it.todo = append(it.todo, n.right)
				rope = n.left
			}
		default:
			panic("impossible")
		}
	}
}
