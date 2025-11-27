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

func TestNewAlertClient(t *testing.T) {
	client := NewAlertClient()

	if client == nil {
		t.Fatal("NewAlertClient() returned nil")
	}

	if client.baseURL != "https://api.weather.gov" {
		t.Errorf("baseURL = %s, want https://api.weather.gov", client.baseURL)
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", client.httpClient.Timeout)
	}
}

func TestNOAAAlertClient_GetActiveAlerts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent header not set")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("Accept header should be application/json")
		}

		w.Header().Set("Content-Type", "application/json")
		data, _ := os.ReadFile("../../testdata/noaa_alert_response.json")
		w.Write(data)
	}))
	defer server.Close()

	client := NewAlertClient()
	client.baseURL = server.URL

	ctx := context.Background()
	alertData, err := client.GetActiveAlerts(ctx, 47.6062, -122.3321)

	if err != nil {
		t.Fatalf("GetActiveAlerts() error = %v", err)
	}

	if alertData == nil {
		t.Fatal("GetActiveAlerts() returned nil")
	}

	if len(alertData.Alerts) != 1 {
		t.Errorf("len(Alerts) = %d, want 1", len(alertData.Alerts))
	}

	if len(alertData.Alerts) > 0 {
		alert := alertData.Alerts[0]

		if alert.Event != "Small Craft Advisory" {
			t.Errorf("Event = %s, want 'Small Craft Advisory'", alert.Event)
		}

		if alert.Severity != models.SeverityModerate {
			t.Errorf("Severity = %v, want SeverityModerate", alert.Severity)
		}

		if alert.Urgency != "Expected" {
			t.Errorf("Urgency = %s, want 'Expected'", alert.Urgency)
		}

		if len(alert.Areas) == 0 {
			t.Error("Areas should not be empty")
		}

		if !alert.IsMarine() {
			t.Error("Small Craft Advisory should be identified as marine alert")
		}
	}
}

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  models.AlertSeverity
	}{
		{"Extreme", models.SeverityExtreme},
		{"Severe", models.SeveritySevere},
		{"Moderate", models.SeverityModerate},
		{"Minor", models.SeverityMinor},
		{"Unknown", models.SeverityUnknown},
		{"Invalid", models.SeverityUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapSeverity(tt.input)
			if got != tt.want {
				t.Errorf("mapSeverity(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNOAAAlertClient_NoAlerts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"features":[]}`))
	}))
	defer server.Close()

	client := NewAlertClient()
	client.baseURL = server.URL

	ctx := context.Background()
	alertData, err := client.GetActiveAlerts(ctx, 47.6062, -122.3321)

	if err != nil {
		t.Fatalf("GetActiveAlerts() error = %v", err)
	}

	if len(alertData.Alerts) != 0 {
		t.Errorf("len(Alerts) = %d, want 0", len(alertData.Alerts))
	}
}
