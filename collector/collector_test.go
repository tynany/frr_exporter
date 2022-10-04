package collector

import (
	"os"
	"path/filepath"
	"testing"
)

func readTestFixture(t *testing.T, filename string) []byte {
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("cannot read test fixture: %v", err)
	}
	return data
}
