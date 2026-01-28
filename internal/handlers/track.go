package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type TrackHandler struct {
	repo TrackRepository
}

type TrackRepository interface {
	CreatePoint(ctx context.Context, deviceID string, lat, lon float64, ts time.Time) error
	GetPathGeoJSON(ctx context.Context, deviceID string, start, end time.Time) (*string, error)
	GetPathGeoJSONAll(ctx context.Context, deviceID string) (*string, error)
}

// TrackRequest matches Home Assistant push format.
type TrackRequest struct {
	DeviceID  string    `json:"device_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

type pathResponse struct {
	DeviceID string  `json:"device_id"`
	Date     string  `json:"date,omitempty"`
	Start    string  `json:"start,omitempty"`
	End      string  `json:"end,omitempty"`
	GeoJSON  *string `json:"geojson"`
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
		resp    pathResponse
	)

	resp.DeviceID = deviceID

	if allStr == "1" || allStr == "true" {
		geojson, err = h.repo.GetPathGeoJSONAll(r.Context(), deviceID)
	} else if daysStr != "" {
		days, convErr := parsePositiveInt(daysStr)
		if convErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "days must be a positive integer"})
			return
		}
		end := time.Now().UTC()
		start := end.AddDate(0, 0, -days)
		resp.Start = start.Format(time.RFC3339)
		resp.End = end.Format(time.RFC3339)
		geojson, err = h.repo.GetPathGeoJSON(r.Context(), deviceID, start, end)
	} else if dateStr != "" {
		date, parseErr := time.Parse("2006-01-02", dateStr)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "date format must be YYYY-MM-DD"})
			return
		}
		start := date.UTC()
		end := start.AddDate(0, 0, 1)
		resp.Date = dateStr
		geojson, err = h.repo.GetPathGeoJSON(r.Context(), deviceID, start, end)
	} else {
		now := time.Now().UTC()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 0, 1)
		resp.Date = start.Format("2006-01-02")
		geojson, err = h.repo.GetPathGeoJSON(r.Context(), deviceID, start, end)
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}

	resp.GeoJSON = geojson
	writeJSON(w, http.StatusOK, resp)
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
