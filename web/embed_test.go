package web

import (
	"io/fs"
	"testing"
)

func TestDistIsEmbeddedEvenBeforeTheFirstBuild(t *testing.T) {
	t.Parallel()
	// The placeholder is tracked and copied on every build, so it is always there.
	if _, err := fs.Stat(Dist(), ".gitkeep"); err != nil {
		t.Fatalf("Stat(.gitkeep) error = %v, want the embedded folder to exist", err)
	}
}
