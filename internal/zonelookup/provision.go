package zonelookup

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jonas-p/go-shp"
	_ "modernc.org/sqlite"
)

const (
	// NOAA Marine Zones Shapefile URL (updated quarterly)
	marineZonesURL = "https://www.weather.gov/source/gis/Shapefiles/WSOM/mz18mr25.zip"
	downloadDir    = "data"
	shapefileBase  = "mz18mr25"
)

// ProvisionDatabase checks if the marine_zones table exists and creates it if not
func ProvisionDatabase(dbPath string) error {
	// Check if marine_zones table already exists
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='marine_zones'").Scan(&count)
	if err != nil {
		return fmt.Errorf("checking for marine_zones table: %w", err)
	}
	if count > 0 {
		return nil // Table already exists
	}

	log.Println("Marine zones table not found, provisioning...")

	// Create data directory if it doesn't exist
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	// Download shapefile
	zipPath := filepath.Join(dataDir, shapefileBase+".zip")
	log.Printf("Downloading NOAA marine zones from %s...", marineZonesURL)
	if err := downloadFile(zipPath, marineZonesURL); err != nil {
		return fmt.Errorf("downloading shapefile: %w", err)
	}
	defer os.Remove(zipPath) // Clean up zip file after extraction

	// Extract shapefile
	log.Println("Extracting shapefile...")
	if err := unzipFile(zipPath, dataDir); err != nil {
		return fmt.Errorf("extracting shapefile: %w", err)
	}

	// Build database
	shapefilePath := filepath.Join(dataDir, shapefileBase+".shp")
	log.Println("Building marine zones database...")
	if err := buildDatabase(shapefilePath, dbPath); err != nil {
		return fmt.Errorf("building database: %w", err)
	}

	// Clean up shapefile files (keep only the database)
	cleanupShapefiles(dataDir, shapefileBase)

	log.Printf("Successfully provisioned marine zones database at %s", dbPath)
	return nil
}

// downloadFile downloads a file from a URL to a local path
func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// unzipFile extracts a zip file to a destination directory
func unzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Construct the full path
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip vulnerability
		if !filepath.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// buildDatabase creates the marine_zones table in the SQLite database from the shapefile
func buildDatabase(shapefilePath, dbPath string) error {
	// Open the shapefile
	shape, err := shp.Open(shapefilePath)
	if err != nil {
		return fmt.Errorf("opening shapefile: %w", err)
	}
	defer shape.Close()

	// Open SQLite database (don't remove - may contain other tables)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(`
		CREATE TABLE marine_zones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			zone_code TEXT NOT NULL,
			zone_name TEXT,
			geometry TEXT NOT NULL,
			bbox_min_lat REAL NOT NULL,
			bbox_max_lat REAL NOT NULL,
			bbox_min_lon REAL NOT NULL,
			bbox_max_lon REAL NOT NULL,
			center_lat REAL NOT NULL,
			center_lon REAL NOT NULL
		);

		CREATE INDEX idx_zones_bbox ON marine_zones(
			bbox_min_lat, bbox_max_lat, bbox_min_lon, bbox_max_lon
		);

		CREATE INDEX idx_zones_code ON marine_zones(zone_code);
		CREATE INDEX idx_zones_center ON marine_zones(center_lat, center_lon);
	`)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	// Process each zone
	count := 0
	for shape.Next() {
		n, p := shape.Shape()

		// Get attributes from DBF
		zoneCode := shape.ReadAttribute(n, 0)      // Field 0: ID (zone code)
		zoneName := shape.ReadAttribute(n, 3)      // Field 3: NAME (zone name)
		centerLonStr := shape.ReadAttribute(n, 4)  // Field 4: LON
		centerLatStr := shape.ReadAttribute(n, 5)  // Field 5: LAT

		// Parse center coordinates
		var centerLon, centerLat float64
		fmt.Sscanf(centerLonStr, "%f", &centerLon)
		fmt.Sscanf(centerLatStr, "%f", &centerLat)

		// Convert geometry to our format
		polygon, ok := p.(*shp.Polygon)
		if !ok {
			continue
		}

		// Calculate bounding box
		bbox := polygon.BBox()

		// For multi-part polygons, use only the LARGEST part (likely the outer boundary)
		coords := make([][]float64, 0)

		// Find the largest part
		largestPartIdx := 0
		largestPartSize := 0

		for partIdx := 0; partIdx < len(polygon.Parts); partIdx++ {
			startIdx := int(polygon.Parts[partIdx])
			endIdx := len(polygon.Points)
			if partIdx+1 < len(polygon.Parts) {
				endIdx = int(polygon.Parts[partIdx+1])
			}
			partSize := endIdx - startIdx
			if partSize > largestPartSize {
				largestPartSize = partSize
				largestPartIdx = partIdx
			}
		}

		// Extract the largest part
		startIdx := int(polygon.Parts[largestPartIdx])
		endIdx := len(polygon.Points)
		if largestPartIdx+1 < len(polygon.Parts) {
			endIdx = int(polygon.Parts[largestPartIdx+1])
		}

		for i := startIdx; i < endIdx; i++ {
			point := polygon.Points[i]
			coords = append(coords, []float64{point.X, point.Y})
		}

		geometryJSON, err := json.Marshal(coords)
		if err != nil {
			log.Printf("Error marshaling geometry for %s: %v", zoneCode, err)
			continue
		}

		// Insert into database
		_, err = db.Exec(`
			INSERT INTO marine_zones (
				zone_code, zone_name, geometry,
				bbox_min_lat, bbox_max_lat, bbox_min_lon, bbox_max_lon,
				center_lat, center_lon
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, zoneCode, zoneName, string(geometryJSON),
			bbox.MinY, bbox.MaxY, bbox.MinX, bbox.MaxX,
			centerLat, centerLon)

		if err != nil {
			log.Printf("Error inserting zone %s: %v", zoneCode, err)
			continue
		}

		count++
		if count%100 == 0 {
			log.Printf("Processed %d zones...", count)
		}
	}

	log.Printf("Successfully created database with %d marine zones", count)
	return nil
}

// cleanupShapefiles removes the extracted shapefile components
func cleanupShapefiles(dir, base string) {
	// Shapefile consists of multiple files with different extensions
	extensions := []string{".shp", ".shx", ".dbf", ".prj", ".cpg", ".shp.xml"}
	for _, ext := range extensions {
		path := filepath.Join(dir, base+ext)
		os.Remove(path) // Ignore errors
	}
}
