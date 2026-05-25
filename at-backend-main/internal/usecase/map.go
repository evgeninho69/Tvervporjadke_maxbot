package usecase

import "activeTver/internal/pkg/raycast"

type MapService struct {
	Districts map[string]raycast.Polygon
}

func (ms MapService) IsPointInPolygon(long, lat float64) bool {
	point := raycast.Point{
		X: long,
		Y: lat,
	}

	for _, district := range ms.Districts {
		if raycast.IsPointInPolygon(point, district) {
			return true
		}
	}

	return false
}

func (ms MapService) GetDistricts() map[string]raycast.Polygon {
	return ms.Districts
}