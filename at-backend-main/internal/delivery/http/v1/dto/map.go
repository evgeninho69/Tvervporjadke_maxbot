package dto

type PointInPolygonResponse struct {
	Inside bool `json:"inside"`
}

type DistrictsResponse struct {
	Districts []DistrictResponse `json:"districts"`
}

type DistrictResponse struct {
	Title   string    `json:"title"`
	Polygon []PointResponse `json:"polygon"`
}

type PointResponse struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}
