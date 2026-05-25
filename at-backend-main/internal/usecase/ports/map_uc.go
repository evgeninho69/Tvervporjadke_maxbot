package ucports

import "activeTver/internal/pkg/raycast"

type MapUseCase interface {
	IsPointInPolygon(long, lat float64) bool
	GetDistricts() map[string]raycast.Polygon
}