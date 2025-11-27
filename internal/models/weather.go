package models

import "time"

// WindData represents wind conditions in marine format
type WindData struct {
	Direction     string  // e.g., "W", "NW", "Variable"
	SpeedMin      float64 // knots
	SpeedMax      float64 // knots
	GustSpeed     float64 // knots (0 if no gusts)
	HasGust       bool
	RawText       string // Original NOAA format
}

// WaveComponent represents a single wave/swell component
type WaveComponent struct {
	Direction string  // e.g., "S", "W", "NW"
	Height    float64 // feet
	Period    int     // seconds
}

// SeaState represents overall sea conditions
type SeaState struct {
	HeightMin float64         // feet
	HeightMax float64         // feet
	Components []WaveComponent // Detailed wave breakdown
	RawText   string          // Original NOAA format
}

// MarineConditions represents current marine weather conditions
type MarineConditions struct {
	Location     string
	Temperature  float64 // Fahrenheit
	Conditions   string  // e.g., "Sunny", "Cloudy", "Rainy"
	Wind         WindData
	Seas         SeaState
	Visibility   float64 // nautical miles
	Pressure     float64 // millibars or inHg
	UpdatedAt    time.Time
}

// MarineForecast represents a forecast period for marine conditions
type MarineForecast struct {
	Date          time.Time
	DayOfWeek     string
	PeriodName    string // e.g., "Tonight", "Friday", "Friday Night"
	Conditions    string
	Wind          WindData
	Seas          SeaState
	Temperature   float64 // Fahrenheit (if applicable)
	RawText       string  // Full forecast text from NOAA
}

// ThreeDayForecast contains marine forecasts for the next 3 days
type ThreeDayForecast struct {
	Periods   []MarineForecast
	UpdatedAt time.Time
}
