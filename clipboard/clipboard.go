// Package clipboard provides access to a copy/paste clipboard.
// If available it uses the system clipboard,
// but if unavailable it falls back to a simple, memory buffer.
//
// It is a wrapper on top of github.com/atotto/clipboard.
package clipboard

import (
	"sync"

	"github.com/atotto/clipboard"
	"github.com/eaburns/T/rope"
)

// A Clipboard provides means to store and fetch text.
// Implementations should support concurrent access.
type Clipboard interface {
	// Store stores the text ot the clipboard.
	Store(rope.Rope) error
	// Fetch returns the text from the clipboard.
	Fetch() (rope.Rope, error)
}

// New returns a new clipboard.
//
// If the system clipboard is available,
// then the returned Clipboard uses the system clipboard.
//
// If the system clipboard is unavailable,
// then a empty, memory-based clipboard is returned.
func New() Clipboard {
	if clipboard.Unsupported {
		return NewMem()
	}
	return sysClipboard{}
}

// NewMem returns a new, empty, memory-based clipboard.
// This method is useful for tests.
func NewMem() Clipboard {
	return &memClipboard{text: rope.Empty()}
}

type sysClipboard struct{}

func (sysClipboard) Store(r rope.Rope) error {
	return clipboard.WriteAll(r.String())
}

func (sysClipboard) Fetch() (rope.Rope, error) {
	str, err := clipboard.ReadAll()
	if err != nil {
		return nil, err
	}
	return rope.New(str), nil
}

type memClipboard struct {
	text rope.Rope
	mu   sync.Mutex
}

func (m *memClipboard) Store(r rope.Rope) error {
	m.mu.Lock()
	m.text = r
	m.mu.Unlock()
	return nil
}

func (m *memClipboard) Fetch() (rope.Rope, error) {
	m.mu.Lock()
	r := m.text
	m.mu.Unlock()
	return r, nil
}
