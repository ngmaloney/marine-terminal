package ports

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ngmaloney/marine-terminal/internal/database"
	"github.com/ngmaloney/marine-terminal/internal/models"
	_ "modernc.org/sqlite"
)

// Repository handles persistence for user-configured ports
type Repository struct{}

// NewRepository creates a new port repository
func NewRepository() *Repository {
	return &Repository{}
}

// SavePort saves a user port configuration to the database
func (r *Repository) SavePort(port *models.Port) error {
	// Ensure schema exists (safe to call multiple times)
	if err := database.EnsureUserSchema(database.DBPath()); err != nil {
		return err
	}

	db, err := sql.Open("sqlite", database.DBPath())
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	query := `
		INSERT INTO user_ports (name, state, city, zipcode, marine_zone_id, tide_station_id, latitude, longitude, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			state = excluded.state,
			city = excluded.city,
			zipcode = excluded.zipcode,
			marine_zone_id = excluded.marine_zone_id,
			tide_station_id = excluded.tide_station_id,
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			created_at = excluded.created_at
	`

	if port.CreatedAt.IsZero() {
		port.CreatedAt = time.Now()
	}

	res, err := db.Exec(query,
		port.Name,
		port.State,
		port.City,
		port.Zipcode,
		port.MarineZoneID,
		port.TideStationID,
		port.Latitude,
		port.Longitude,
		port.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("saving port: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id: %w", err)
	}
	port.ID = id

	return nil
}

// ListPorts retrieves all saved user ports
func (r *Repository) ListPorts() ([]models.Port, error) {
	if err := database.EnsureUserSchema(database.DBPath()); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", database.DBPath())
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, name, state, city, zipcode, marine_zone_id, tide_station_id, latitude, longitude, created_at FROM user_ports ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("querying ports: %w", err)
	}
	defer rows.Close()

	var ports []models.Port
	for rows.Next() {
		var p models.Port
		var state, city, zipcode sql.NullString // Handle potential nulls

		if err := rows.Scan(&p.ID, &p.Name, &state, &city, &zipcode, &p.MarineZoneID, &p.TideStationID, &p.Latitude, &p.Longitude, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning port: %w", err)
		}
		p.State = state.String
		p.City = city.String
		p.Zipcode = zipcode.String
		p.StationID = p.TideStationID
		ports = append(ports, p)
	}

	return ports, nil
}

// DeletePort removes a port by name
func (r *Repository) DeletePort(name string) error {
	if err := database.EnsureUserSchema(database.DBPath()); err != nil {
		return err
	}

	db, err := sql.Open("sqlite", database.DBPath())
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM user_ports WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("deleting port: %w", err)
	}

	return nil
}