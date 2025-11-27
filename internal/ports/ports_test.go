package ports

import (
	"context"
	"testing"
)

func TestNewStaticPortClient(t *testing.T) {
	client := NewStaticPortClient()

	if client == nil {
		t.Fatal("NewStaticPortClient() returned nil")
	}

	if len(client.ports) == 0 {
		t.Error("StaticPortClient should have default ports")
	}
}

func TestStaticPortClient_SearchByLocation(t *testing.T) {
	client := NewStaticPortClient()
	ctx := context.Background()

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantErr   bool
	}{
		{"search by city - Seattle", "Seattle", 1, false},
		{"search by city - San Francisco", "san francisco", 1, false},
		{"search by state - CA", "CA", 2, false},
		{"search by state - WA", "wa", 1, false},
		{"search by name", "battery", 1, false},
		{"case insensitive", "SEATTLE", 1, false},
		{"no results", "NonexistentCity", 0, true},
		{"empty query", "", 0, true},
		{"whitespace query", "   ", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := client.SearchByLocation(ctx, tt.query)

			if (err != nil) != tt.wantErr {
				t.Errorf("SearchByLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) != tt.wantCount {
				t.Errorf("SearchByLocation() returned %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestStaticPortClient_GetPortByID(t *testing.T) {
	client := NewStaticPortClient()
	ctx := context.Background()

	tests := []struct {
		name      string
		stationID string
		wantName  string
		wantErr   bool
	}{
		{"valid - Seattle", "9447130", "Seattle", false},
		{"valid - San Francisco", "9414290", "San Francisco", false},
		{"valid - New York", "8518750", "The Battery", false},
		{"invalid station", "0000000", "", true},
		{"empty station", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := client.GetPortByID(ctx, tt.stationID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPortByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if port == nil {
					t.Fatal("GetPortByID() returned nil port")
				}
				if port.Name != tt.wantName {
					t.Errorf("GetPortByID() port name = %s, want %s", port.Name, tt.wantName)
				}
				if port.ID != tt.stationID {
					t.Errorf("GetPortByID() port ID = %s, want %s", port.ID, tt.stationID)
				}
			}
		})
	}
}

func TestStaticPortClient_PortData(t *testing.T) {
	client := NewStaticPortClient()
	ctx := context.Background()

	// Verify Seattle port has complete data
	port, err := client.GetPortByID(ctx, "9447130")
	if err != nil {
		t.Fatalf("GetPortByID() error = %v", err)
	}

	if port.City != "Seattle" {
		t.Errorf("City = %s, want Seattle", port.City)
	}
	if port.State != "WA" {
		t.Errorf("State = %s, want WA", port.State)
	}
	if port.Latitude == 0 || port.Longitude == 0 {
		t.Error("Latitude/Longitude should be set")
	}
	if port.TideStation == "" {
		t.Error("TideStation should be set")
	}
	if port.Type != "coastal" {
		t.Errorf("Type = %s, want coastal", port.Type)
	}
}

func TestGetDefaultPorts(t *testing.T) {
	ports := getDefaultPorts()

	if len(ports) < 5 {
		t.Errorf("getDefaultPorts() returned %d ports, want at least 5", len(ports))
	}

	// Verify all ports have required fields
	for i, port := range ports {
		if port.ID == "" {
			t.Errorf("Port %d missing ID", i)
		}
		if port.Name == "" {
			t.Errorf("Port %d missing Name", i)
		}
		if port.City == "" {
			t.Errorf("Port %d missing City", i)
		}
		if port.State == "" {
			t.Errorf("Port %d missing State", i)
		}
		if port.Latitude == 0 || port.Longitude == 0 {
			t.Errorf("Port %d missing coordinates", i)
		}
	}
}
