package trip_test

import (
	"errors"
	"math"
	"testing"

	"github.com/a-pelikh/tripgo/internal/domain/trip"
)

func TestNewCoordinates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		wantField string
	}{
		{name: "valid", latitude: 59.9398, longitude: 30.3146},
		{name: "boundary", latitude: -90, longitude: 180},
		{name: "latitude below range", latitude: -90.1, wantField: "latitude"},
		{name: "latitude above range", latitude: 90.1, wantField: "latitude"},
		{name: "latitude NaN", latitude: math.NaN(), wantField: "latitude"},
		{name: "longitude below range", longitude: -180.1, wantField: "longitude"},
		{name: "longitude above range", longitude: 180.1, wantField: "longitude"},
		{name: "longitude infinity", longitude: math.Inf(1), wantField: "longitude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			coordinates, err := trip.NewCoordinates(tt.latitude, tt.longitude)
			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("NewCoordinates() error = %v", err)
				}
				if coordinates.Latitude() != tt.latitude || coordinates.Longitude() != tt.longitude {
					t.Fatalf("NewCoordinates() = (%v, %v), want (%v, %v)", coordinates.Latitude(), coordinates.Longitude(), tt.latitude, tt.longitude)
				}

				return
			}

			var validationErr trip.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("NewCoordinates() error = %v, want ValidationError", err)
			}
			if validationErr.Field != tt.wantField {
				t.Fatalf("ValidationError.Field = %q, want %q", validationErr.Field, tt.wantField)
			}
		})
	}
}
