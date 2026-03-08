package uidc

import (
	"testing"
)

func TestNewID64(t *testing.T) {
	t.Parallel()

	a := NewID64()
	b := NewID64()

	if a == b {
		t.Errorf("Two UID should be different")
	}
}

func TestNewID128(t *testing.T) {
	t.Parallel()

	a := NewID128()
	b := NewID128()

	if a == b {
		t.Errorf("Two UID should be different")
	}
}
