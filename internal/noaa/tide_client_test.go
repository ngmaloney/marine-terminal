package noaa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ngmaloney/mariner-tui/internal/models"
)

func TestNewTideClient(t *testing.T) {
	client := NewTideClient()

	if client == nil {
		t.Fatal("NewTideClient() returned nil")
	}

	if client.baseURL != "https://api.tidesandcurrents.noaa.gov/api/prod/datagetter" {
		t.Errorf("baseURL = %s, unexpected value", client.baseURL)
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", client.httpClient.Timeout)
	}
}

func TestNOAATideClient_GetTidePredictions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		query := r.URL.Query()
		if query.Get("station") != "9447130" {
			t.Errorf("station param = %s, want 9447130", query.Get("station"))
		}
		if query.Get("product") != "predictions" {
			t.Error("product param should be 'predictions'")
		}
		if query.Get("datum") != "MLLW" {
			t.Error("datum param should be 'MLLW'")
		}
		if query.Get("interval") != "hilo" {
			t.Error("interval param should be 'hilo'")
		}

		w.Header().Set("Content-Type", "application/json")
		data, _ := os.ReadFile("../../testdata/noaa_tide_response.json")
		w.Write(data)
	}))
	defer server.Close()

	client := NewTideClient()
	client.baseURL = server.URL

	ctx := context.Background()
	startDate := time.Date(2025, 11, 27, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 11, 29, 0, 0, 0, 0, time.UTC)

	tideData, err := client.GetTidePredictions(ctx, "9447130", startDate, endDate)

	if err != nil {
		t.Fatalf("GetTidePredictions() error = %v", err)
	}

	if tideData == nil {
		t.Fatal("GetTidePredictions() returned nil")
	}

	if tideData.StationID != "9447130" {
		t.Errorf("StationID = %s, want 9447130", tideData.StationID)
	}

	if tideData.StationName != "Seattle" {
		t.Errorf("StationName = %s, want Seattle", tideData.StationName)
	}

	if len(tideData.Events) != 4 {
		t.Errorf("len(Events) = %d, want 4", len(tideData.Events))
	}

	// Verify first event (low tide)
	if len(tideData.Events) > 0 {
		event := tideData.Events[0]
		if event.Type != models.TideLow {
			t.Errorf("First event type = %v, want TideLow", event.Type)
		}
		if event.Height != 0.5 {
			t.Errorf("First event height = %v, want 0.5", event.Height)
		}
	}

	// Verify second event (high tide)
	if len(tideData.Events) > 1 {
		event := tideData.Events[1]
		if event.Type != models.TideHigh {
			t.Errorf("Second event type = %v, want TideHigh", event.Type)
		}
		if event.Height != 5.2 {
			t.Errorf("Second event height = %v, want 5.2", event.Height)
		}
	}
}

func TestNOAATideClient_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Station not found"))
	}))
	defer server.Close()

	client := NewTideClient()
	client.baseURL = server.URL

	ctx := context.Background()
	startDate := time.Now()
	endDate := startDate.Add(3 * 24 * time.Hour)

	_, err := client.GetTidePredictions(ctx, "invalid", startDate, endDate)

	if err == nil {
		t.Error("Expected error for invalid station, got nil")
	}
}
