package baseError

import (
	"testing"
)

func TestNewBaseError(t *testing.T) {
	err := NewCode("1", "test%d%s", WithStack())
	e1 := Clone(err, WithMsgArgs(1, "e1"))
	t.Log(e1)
}
