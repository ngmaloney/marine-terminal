package geocoding

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProvisionZipcodeDatabase(t *testing.T) {
	// Create a temporary directory for the database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "zipcodes.db")

	// Ensure the database doesn't exist initially
	if _, err := os.Stat(dbPath); err == nil {
		t.Fatalf("Database file %s already exists before provisioning", dbPath)
	}

	// Provision the database
	err := ProvisionZipcodeDatabase(dbPath)
	if err != nil {
		t.Fatalf("ProvisionZipcodeDatabase failed: %v", err)
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("Database file %s was not created", dbPath)
	}

	// Verify that the zipcodes table exists and is populated
	needs, err := NeedsProvisioning(dbPath)
	if err != nil {
		t.Fatalf("NeedsProvisioning failed after provisioning: %v", err)
	}
	if needs {
		t.Fatal("Database still needs provisioning after successful run")
	}

	// Optional: Add a simple query to ensure data is present
	db, err := getZipcodeDB(dbPath) // This function now exists in zipcode.go for simplicity.
	if err != nil {
		t.Fatalf("Failed to get zipcode DB: %v", err)
	}
	defer db.Close()

	loc, err := lookupZipcodeInDB(db, "90210") // Assuming 90210 is in the uszips.csv
	if err != nil {
		t.Errorf("lookupZipcodeInDB failed: %v", err)
	}
	if loc == nil {
		t.Error("Expected a location for zipcode 90210, got nil")
	}
}
