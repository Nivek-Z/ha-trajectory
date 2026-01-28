package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type TrackRepository struct {
	db *gorm.DB
}

func NewTrackRepository(db *gorm.DB) *TrackRepository {
	return &TrackRepository{db: db}
}

func (r *TrackRepository) CreatePoint(ctx context.Context, deviceID string, lat, lon float64, ts time.Time) error {
	ewkt := fmt.Sprintf("SRID=4326;POINT(%f %f)", lon, lat)

	return r.db.WithContext(ctx).Exec(
		"INSERT INTO track_points (device_id, latitude, longitude, geom, created_at) VALUES (?, ?, ?, ST_GeomFromEWKT(?), ?)",
		deviceID, lat, lon, ewkt, ts,
	).Error
}

func (r *TrackRepository) GetPathGeoJSON(ctx context.Context, deviceID string, start, end time.Time) (*string, error) {
	var geojson sql.NullString

	err := r.db.WithContext(ctx).Raw(
		`SELECT ST_AsGeoJSON(ST_MakeLine(geom ORDER BY created_at))
		 FROM track_points
		 WHERE device_id = ? AND created_at >= ? AND created_at < ?`,
		deviceID, start, end,
	).Scan(&geojson).Error
	if err != nil {
		return nil, err
	}

	if !geojson.Valid {
		return nil, nil
	}

	return &geojson.String, nil
}

func (r *TrackRepository) GetPathGeoJSONAll(ctx context.Context, deviceID string) (*string, error) {
	var geojson sql.NullString

	err := r.db.WithContext(ctx).Raw(
		`SELECT ST_AsGeoJSON(ST_MakeLine(geom ORDER BY created_at))
		 FROM track_points
		 WHERE device_id = ?`,
		deviceID,
	).Scan(&geojson).Error
	if err != nil {
		return nil, err
	}

	if !geojson.Valid {
		return nil, nil
	}

	return &geojson.String, nil
}
