package database

import (
	"path/filepath"
	"testing"
)

func TestDBPath(t *testing.T) {
	expected := filepath.Join("data", "marine-terminal.db")
	if got := DBPath(); got != expected {
		t.Errorf("DBPath() = %v, want %v", got, expected)
	}
}
