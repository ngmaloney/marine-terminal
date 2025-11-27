package noaa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewWeatherClient(t *testing.T) {
	client := NewWeatherClient()

	if client == nil {
		t.Fatal("NewWeatherClient() returned nil")
	}

	if client.baseURL != "https://api.weather.gov" {
		t.Errorf("baseURL = %s, want https://api.weather.gov", client.baseURL)
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", client.httpClient.Timeout)
	}

	if client.userAgent == "" {
		t.Error("userAgent should not be empty")
	}
}

func TestNOAAWeatherClient_GetGridPoint(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify User-Agent header
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent header not set")
		}

		// Verify Accept header
		if r.Header.Get("Accept") != "application/json" {
			t.Error("Accept header should be application/json")
		}

		// Return mock point response
		pointResp := pointResponse{
			Properties: struct {
				GridID string `json:"gridId"`
				GridX  int    `json:"gridX"`
				GridY  int    `json:"gridY"`
			}{
				GridID: "SEW",
				GridX:  124,
				GridY:  67,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pointResp)
	}))
	defer server.Close()

	client := NewWeatherClient()
	client.baseURL = server.URL

	ctx := context.Background()
	gridPoint, err := client.getGridPoint(ctx, 47.6062, -122.3321)

	if err != nil {
		t.Fatalf("getGridPoint() error = %v", err)
	}

	if gridPoint.GridID != "SEW" {
		t.Errorf("GridID = %s, want SEW", gridPoint.GridID)
	}

	if gridPoint.GridX != 124 {
		t.Errorf("GridX = %d, want 124", gridPoint.GridX)
	}

	if gridPoint.GridY != 67 {
		t.Errorf("GridY = %d, want 67", gridPoint.GridY)
	}
}

func TestNOAAWeatherClient_GetMarineConditions(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// First call: points endpoint
		if callCount == 0 {
			data, _ := os.ReadFile("../../testdata/noaa_point_response.json")
			w.Write(data)
			callCount++
			return
		}

		// Second call: forecast endpoint
		data, _ := os.ReadFile("../../testdata/noaa_forecast_response.json")
		w.Write(data)
	}))
	defer server.Close()

	client := NewWeatherClient()
	client.baseURL = server.URL

	ctx := context.Background()
	conditions, err := client.GetMarineConditions(ctx, 47.6062, -122.3321)

	if err != nil {
		t.Fatalf("GetMarineConditions() error = %v", err)
	}

	if conditions == nil {
		t.Fatal("GetMarineConditions() returned nil")
	}

	if conditions.Conditions != "Partly Cloudy" {
		t.Errorf("Conditions = %s, want Partly Cloudy", conditions.Conditions)
	}

	if conditions.Temperature != 58 {
		t.Errorf("Temperature = %v, want 58", conditions.Temperature)
	}
}

func TestNOAAWeatherClient_GetMarineForecast(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if callCount == 0 {
			data, _ := os.ReadFile("../../testdata/noaa_point_response.json")
			w.Write(data)
			callCount++
			return
		}

		data, _ := os.ReadFile("../../testdata/noaa_forecast_response.json")
		w.Write(data)
	}))
	defer server.Close()

	client := NewWeatherClient()
	client.baseURL = server.URL

	ctx := context.Background()
	forecast, err := client.GetMarineForecast(ctx, 47.6062, -122.3321)

	if err != nil {
		t.Fatalf("GetMarineForecast() error = %v", err)
	}

	if forecast == nil {
		t.Fatal("GetMarineForecast() returned nil")
	}

	if len(forecast.Periods) != 2 {
		t.Errorf("len(Periods) = %d, want 2", len(forecast.Periods))
	}

	if len(forecast.Periods) > 0 {
		period := forecast.Periods[0]
		if period.PeriodName != "This Afternoon" {
			t.Errorf("Period name = %s, want 'This Afternoon'", period.PeriodName)
		}

		if period.Conditions != "Partly Cloudy" {
			t.Errorf("Conditions = %s, want 'Partly Cloudy'", period.Conditions)
		}
	}
}

func TestNOAAWeatherClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"404 not found", http.StatusNotFound, true},
		{"500 server error", http.StatusInternalServerError, true},
		{"503 unavailable", http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("error"))
			}))
			defer server.Close()

			client := NewWeatherClient()
			client.baseURL = server.URL

			ctx := context.Background()
			_, err := client.getGridPoint(ctx, 47.6062, -122.3321)

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
