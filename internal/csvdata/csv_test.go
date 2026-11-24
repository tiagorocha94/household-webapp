package csvdata

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteCSVReplacesFileWithoutLeftovers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	for _, v := range []string{"1", "2"} {
		if err := Write(path, [][]string{{"a"}, {v}}); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	got, err := Read(path, []string{"a"})
	if err != nil || !reflect.DeepEqual(got, [][]string{{"2"}}) {
		t.Fatalf("Read() = %v, %v, want [[2]]", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("directory = %v, %v, want only out.csv", entries, err)
	}
}
