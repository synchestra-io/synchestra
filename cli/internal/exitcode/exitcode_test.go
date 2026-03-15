package exitcode_test

import (
	"errors"
	"testing"

	"github.com/synchesta-io/synchestra/cli/internal/exitcode"
)

func TestError_Error(t *testing.T) {
	e := exitcode.New(3, "repo %q not found", "foo")
	if e.Error() != `repo "foo" not found` {
		t.Errorf("Error() = %q", e.Error())
	}
	if e.Code != 3 {
		t.Errorf("Code = %d, want 3", e.Code)
	}
}

func TestError_Unwrap(t *testing.T) {
	e := exitcode.New(1, "conflict")
	var target *exitcode.Error
	if !errors.As(e, &target) {
		t.Error("errors.As should find *exitcode.Error")
	}
	if target.Code != 1 {
		t.Errorf("Code = %d, want 1", target.Code)
	}
}
