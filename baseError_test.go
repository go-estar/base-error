package baseError

import (
	"testing"
)

func TestNewBaseError(t *testing.T) {
	err := New("1", "test%d%s", WithStack())
	e1 := NewClone(err, WithArgs(1, "e1"))
	t.Log(e1)
}
