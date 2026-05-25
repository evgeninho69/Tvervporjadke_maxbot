package handlers

import (
	"activeTver/internal/delivery/http/v1/dto"
	"activeTver/internal/delivery/http/v1/responses"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"activeTver/internal/pkg/raycast"
	ucports "activeTver/internal/usecase/ports"
	"net/http"
	"strconv"
	"sync"
)

type Map struct {
	ClaimsService ucports.ClaimsUseCase
	MapService    ucports.MapUseCase
}

func (h Map) Setup() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /claims", h.GetClaimsByArea)
	mux.HandleFunc("GET /check-point", h.IsPointInPolygon)
	mux.HandleFunc("GET /districts", h.GetDistricts)

	return mux
}

const opMapGetClaimsByArea = "delivery.http.handlers.Map.GetClaimsByArea"

func (h Map) GetClaimsByArea(w http.ResponseWriter, r *http.Request) {
	correlationID := common.GetCorrelationID(r.Context())

	query := r.URL.Query()
	lat1 := query.Get("lat1")
	long1 := query.Get("long1")
	lat2 := query.Get("lat2")
	long2 := query.Get("long2")

	if lat1 == "" || long1 == "" || lat2 == "" || long2 == "" {
		logs.Warn(
			"empty query parameters",
			correlationID,
			opMapGetClaimsByArea,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)
		return
	}

	lat1float, err := strconv.ParseFloat(lat1, 64)
	if err != nil {
		logs.Warn(
			err.Error(),
			correlationID,
			opMapGetClaimsByArea,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)
		return
	}

	long1float, err := strconv.ParseFloat(long1, 64)
	if err != nil {
		logs.Warn(
			err.Error(),
			correlationID,
			opMapGetClaimsByArea,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)
		return
	}

	lat2float, err := strconv.ParseFloat(lat2, 64)
	if err != nil {
		logs.Warn(
			err.Error(),
			correlationID,
			opMapGetClaimsByArea,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)
		return
	}

	long2float, err := strconv.ParseFloat(long2, 64)
	if err != nil {
		logs.Warn(
			err.Error(),
			correlationID,
			opMapGetClaimsByArea,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)
		return
	}

	claims, err := h.ClaimsService.GetByArea(r.Context(), lat1float, long1float, lat2float, long2float)
	if err != nil {
		responses.InternalServerError(r.Context(), w)
		return
	}

	resp := make(map[string]any)

	claimsForResponse := make([]dto.ClaimsDataForMapResponse, 0, len(claims))

	for _, claim := range claims {
		claimsForResponse = append(claimsForResponse, dto.ClaimsDataForMapResponse{
			ID:        claim.ID,
			Latitude:  claim.Latitude,
			Longitude: claim.Longitude,
		})
	}

	resp["claims"] = claimsForResponse

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

const opMapPointInPolygon = "delivery.http.handlers.Map.IsPointInPolygon"

func (h Map) IsPointInPolygon(w http.ResponseWriter, r *http.Request) {
	latString := r.URL.Query().Get("lat")
	if latString == "" {
		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			"empty query parameter: lat",
		)

		return
	}

	longString := r.URL.Query().Get("long")
	if longString == "" {
		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			"empty query parameter: long",
		)
		return
	}

	lat, err := strconv.ParseFloat(latString, 64)
	if err != nil || !(-90 <= lat || lat <= 90) {
		logs.Warn(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opMapPointInPolygon,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			"incorrect query parameter: lat",
		)
		return
	}

	long, err := strconv.ParseFloat(longString, 64)
	if err != nil || !(-180 <= long || long <= 180) {
		logs.Warn(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opMapPointInPolygon,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			"incorrect query parameter: long",
		)
		return
	}

	resp := dto.PointInPolygonResponse{
		Inside: h.MapService.IsPointInPolygon(long, lat),
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

var preparedDistricts dto.DistrictsResponse

var onceGetDistricts sync.Once

func (h Map) raycastPolygonToDTOPolygon(rc raycast.Polygon) []dto.PointResponse {
	polygon := make([]dto.PointResponse, 0, len(rc))

	for _, point := range rc {
		polygon = append(polygon, dto.PointResponse{
			Longitude: point.X,
			Latitude:  point.Y,
		})
	}

	return polygon
}

func (h Map) GetDistricts(w http.ResponseWriter, r *http.Request) {
	onceGetDistricts.Do(func() {
		districts := h.MapService.GetDistricts()

		resp := dto.DistrictsResponse{
			Districts: make([]dto.DistrictResponse, 0, len(districts)),
		}

		for key, value := range districts {

			resp.Districts = append(resp.Districts, dto.DistrictResponse{
				Title:   key,
				Polygon: h.raycastPolygonToDTOPolygon(value),
			})
		}

		preparedDistricts = resp
	})

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		preparedDistricts,
	)
}
