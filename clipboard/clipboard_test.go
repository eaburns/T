package clipboard

import (
	"testing"

	"github.com/eaburns/T/rope"
)

func TestNewMem(t *testing.T) {
	const (
		text0 = "Hello, World!"
		text1 = "Hello, 世界"
	)
	c := NewMem()
	if got, err := c.Fetch(); err != nil || got.String() != "" {
		t.Errorf("Fetch()=%q, %v want %q,nil", got, err, "")
	}
	c.Store(rope.New(text0))
	if got, err := c.Fetch(); err != nil || got.String() != text0 {
		t.Errorf("Fetch()=%q, %v want %q,nil", got, err, text0)
	}
	c.Store(rope.New(text1))
	if got, err := c.Fetch(); err != nil || got.String() != text1 {
		t.Errorf("Fetch()=%q, %v want %q,nil", got, err, text1)
	}
}
