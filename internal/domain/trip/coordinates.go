package trip

import "math"

type Coordinates struct {
	latitude  float64
	longitude float64
}

func NewCoordinates(latitude, longitude float64) (Coordinates, error) {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 {
		return Coordinates{}, invalid("latitude", "must be between -90 and 90")
	}

	if math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 {
		return Coordinates{}, invalid("longitude", "must be between -180 and 180")
	}

	return Coordinates{
		latitude:  latitude,
		longitude: longitude,
	}, nil
}

func (c Coordinates) Latitude() float64 {
	return c.latitude
}

func (c Coordinates) Longitude() float64 {
	return c.longitude
}
