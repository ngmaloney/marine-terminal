package models

import (
	"testing"
)

func TestWindData_HasGustLogic(t *testing.T) {
	tests := []struct {
		name     string
		wind     WindData
		wantGust bool
	}{
		{
			name: "with gusts",
			wind: WindData{
				Direction: "W",
				SpeedMin:  15,
				SpeedMax:  20,
				GustSpeed: 30,
				HasGust:   true,
			},
			wantGust: true,
		},
		{
			name: "no gusts",
			wind: WindData{
				Direction: "E",
				SpeedMin:  10,
				SpeedMax:  15,
				GustSpeed: 0,
				HasGust:   false,
			},
			wantGust: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wind.HasGust != tt.wantGust {
				t.Errorf("WindData.HasGust = %v, want %v", tt.wind.HasGust, tt.wantGust)
			}
			if tt.wantGust && tt.wind.GustSpeed == 0 {
				t.Errorf("WindData has gust but GustSpeed is 0")
			}
		})
	}
}

func TestWaveComponent_Structure(t *testing.T) {
	// Test that WaveComponent can represent the NOAA format:
	// "S 5 ft at 8 seconds"
	wave := WaveComponent{
		Direction: "S",
		Height:    5.0,
		Period:    8,
	}

	if wave.Direction != "S" {
		t.Errorf("WaveComponent.Direction = %v, want 'S'", wave.Direction)
	}
	if wave.Height != 5.0 {
		t.Errorf("WaveComponent.Height = %v, want 5.0", wave.Height)
	}
	if wave.Period != 8 {
		t.Errorf("WaveComponent.Period = %v, want 8", wave.Period)
	}
}

func TestSeaState_MultipleComponents(t *testing.T) {
	// Test that SeaState can represent:
	// "Seas 5 to 7 ft. Wave Detail: S 5 ft at 8 seconds and W 4 ft at 5 seconds"
	seas := SeaState{
		HeightMin: 5.0,
		HeightMax: 7.0,
		Components: []WaveComponent{
			{Direction: "S", Height: 5.0, Period: 8},
			{Direction: "W", Height: 4.0, Period: 5},
		},
	}

	if seas.HeightMin != 5.0 || seas.HeightMax != 7.0 {
		t.Errorf("SeaState height range = %v to %v, want 5.0 to 7.0",
			seas.HeightMin, seas.HeightMax)
	}

	if len(seas.Components) != 2 {
		t.Fatalf("SeaState.Components length = %d, want 2", len(seas.Components))
	}

	// Verify first component (S 5 ft at 8 seconds)
	if c := seas.Components[0]; c.Direction != "S" || c.Height != 5.0 || c.Period != 8 {
		t.Errorf("First component = %+v, want {S 5.0 8}", c)
	}

	// Verify second component (W 4 ft at 5 seconds)
	if c := seas.Components[1]; c.Direction != "W" || c.Height != 4.0 || c.Period != 5 {
		t.Errorf("Second component = %+v, want {W 4.0 5}", c)
	}
}
