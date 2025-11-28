package geocoding

import (
	"testing"
)

func TestIsZipcode(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"12345", true},
		{"12345-6789", true},
		{"02139", true},
		{"90210", true},
		{"1234", false},
		{"123456", false},
		{"abcde", false},
		{"12a45", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isZipcode(tt.input); got != tt.expected {
				t.Errorf("isZipcode(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNewGeocoder(t *testing.T) {
	g := NewGeocoder()
	if g == nil {
		t.Error("NewGeocoder() returned nil")
	}
}
