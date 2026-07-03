package cache

import (
	"context"
	"testing"
)

func TestCheckerNilClientReturnsError(t *testing.T) {
	t.Parallel()

	var checker Checker
	if err := checker.Check(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
