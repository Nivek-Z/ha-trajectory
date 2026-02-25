package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ha-trajectory/internal/models"
)

type TrackHandler struct {
	repo TrackRepository
}

type TrackRepository interface {
	CreatePoint(ctx context.Context, deviceID string, lat, lon float64, ts time.Time) error
	GetPathGeoJSON(ctx context.Context, deviceID string, start, end time.Time) (*string, error)
	GetPathGeoJSONAll(ctx context.Context, deviceID string) (*string, error)
	GetPoints(ctx context.Context, deviceID string, start, end time.Time) ([]models.TrackPointView, error)
	GetPointsAll(ctx context.Context, deviceID string) ([]models.TrackPointView, error)
}

// TrackRequest matches Home Assistant push format.
type TrackRequest struct {
	DeviceID  string    `json:"device_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   json.RawMessage        `json:"geometry"`
}

func NewTrackHandler(repo TrackRepository) *TrackHandler {
	return &TrackHandler{repo: repo}
}

func (h *TrackHandler) HandleTrack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req TrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.DeviceID == "" || req.Timestamp.IsZero() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "device_id and timestamp are required"})
		return
	}

	if err := h.repo.CreatePoint(r.Context(), req.DeviceID, req.Latitude, req.Longitude, req.Timestamp); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *TrackHandler) HandlePath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	deviceID := r.URL.Query().Get("device_id")
	dateStr := r.URL.Query().Get("date")
	daysStr := r.URL.Query().Get("days")
	allStr := r.URL.Query().Get("all")
	if deviceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "device_id is required"})
		return
	}

	var (
		geojson *string
		err     error
		points  []models.TrackPointView
	)

	if allStr == "1" || allStr == "true" {
		geojson, err = h.repo.GetPathGeoJSONAll(r.Context(), deviceID)
		if err == nil {
			points, err = h.repo.GetPointsAll(r.Context(), deviceID)
		}
	} else if daysStr != "" {
		days, convErr := parsePositiveInt(daysStr)
		if convErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "days must be a positive integer"})
			return
		}
		end := time.Now().UTC()
		start := end.AddDate(0, 0, -days)
		geojson, err = h.repo.GetPathGeoJSON(r.Context(), deviceID, start, end)
		if err == nil {
			points, err = h.repo.GetPoints(r.Context(), deviceID, start, end)
		}
	} else if dateStr != "" {
		date, parseErr := time.Parse("2006-01-02", dateStr)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "date format must be YYYY-MM-DD"})
			return
		}
		start := date.UTC()
		end := start.AddDate(0, 0, 1)
		geojson, err = h.repo.GetPathGeoJSON(r.Context(), deviceID, start, end)
		if err == nil {
			points, err = h.repo.GetPoints(r.Context(), deviceID, start, end)
		}
	} else {
		now := time.Now().UTC()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 0, 1)
		geojson, err = h.repo.GetPathGeoJSON(r.Context(), deviceID, start, end)
		if err == nil {
			points, err = h.repo.GetPoints(r.Context(), deviceID, start, end)
		}
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}

	fc, buildErr := buildFeatureCollection(deviceID, geojson, points)
	if buildErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "build geojson failed"})
		return
	}

	writeJSON(w, http.StatusOK, fc)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func parsePositiveInt(input string) (int, error) {
	value, err := strconv.Atoi(input)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid")
	}
	return value, nil
}

func buildFeatureCollection(deviceID string, lineGeoJSON *string, points []models.TrackPointView) (featureCollection, error) {
	features := make([]feature, 0, len(points)+1)

	if lineGeoJSON != nil {
		features = append(features, feature{
			Type: "Feature",
			Properties: map[string]interface{}{
				"kind":      "path",
				"device_id": deviceID,
			},
			Geometry: json.RawMessage(*lineGeoJSON),
		})
	}

	for _, point := range points {
		geom, err := marshalGeometry("Point", []interface{}{point.Longitude, point.Latitude})
		if err != nil {
			return featureCollection{}, err
		}
		features = append(features, feature{
			Type: "Feature",
			Properties: map[string]interface{}{
				"kind":      "point",
				"device_id": deviceID,
				"time":      point.CreatedAt.UTC().Format(time.RFC3339),
			},
			Geometry: geom,
		})
	}

	return featureCollection{
		Type:     "FeatureCollection",
		Features: features,
	}, nil
}

func marshalGeometry(geomType string, coordinates interface{}) (json.RawMessage, error) {
	payload := map[string]interface{}{
		"type":        geomType,
		"coordinates": coordinates,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}
