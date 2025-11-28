package database

import "path/filepath"

// DBPath returns the path to the single shared database
func DBPath() string {
	return filepath.Join("data", "marine-terminal.db")
}
