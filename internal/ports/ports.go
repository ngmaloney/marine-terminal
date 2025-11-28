package ports

import (
	"context"

	"github.com/ngmaloney/mariner-tui/internal/models"
)

// Client defines the interface for searching ports
type Client interface {
	SearchByLocation(ctx context.Context, query string) ([]models.Port, error)
	GetPortByID(ctx context.Context, stationID string) (*models.Port, error)
}
