package stale_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aldous/jevium/internal/stale"
)

func TestIsUsesErrorsIsNotStrings(t *testing.T) {
	t.Parallel()
	if stale.Is(errors.New("page changed since the decision. Observe again")) {
		t.Fatal("string match must not count")
	}
	if !stale.Is(stale.Error{Msg: "target moved"}) {
		t.Fatal("typed stale missed")
	}
	if !stale.Is(fmt.Errorf("wrap: %w", stale.Error{Msg: "inner"})) {
		t.Fatal("wrapped typed stale missed")
	}
}
