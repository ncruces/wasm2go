package sjlj

import "testing"

func TestNew(t *testing.T) {
	mod := New()
	got := mod.Xtest()
	if got != 42 {
		t.Errorf("got %d, want 42", got)
	}
	t.Log()
}
