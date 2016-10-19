package dummy

import (
	"fmt"
	"testing"
)

func TestGetDesc(t *testing.T) {
	exp := "test string"
	i := &Info{Desc: exp}
	res := i.GetDesc()
	if res != exp {
		t.Error(fmt.Errorf("The strings are different: %s <> %s", res, exp))
	}
}
