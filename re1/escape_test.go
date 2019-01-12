package re1

import (
	"reflect"
	"strings"
	"testing"
)

func TestEscape(t *testing.T) {
	const meta = `|*+?.^$()[]\`
	str := meta + "abc" + meta + "abc"
	re, residual, err := New(Escape(str), Opts{})
	if err != nil || residual != "" {
		t.Fatalf("New(%q)=_,%q,%v, want _,\"\",nil", str, residual, err)
	}
	want := []int64{0, int64(len(str)), 0 /* id */}
	if got := re.Find(strings.NewReader(str)); !reflect.DeepEqual(got, want) {
		t.Errorf("got=%v, want=%v\n", got, want)
	}
}
