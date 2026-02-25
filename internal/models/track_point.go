package models

import "time"

// TrackPoint stores a single device location.
type TrackPoint struct {
	ID        uint      `gorm:"primaryKey"`
	DeviceID  string    `gorm:"index:idx_device_time,priority:1;not null"`
	Latitude  float64   `gorm:"not null"`
	Longitude float64   `gorm:"not null"`
	Geom      string    `gorm:"type:geometry(Point,4326);not null"`
	CreatedAt time.Time `gorm:"index:idx_device_time,priority:2;not null"`
}

func (TrackPoint) TableName() string {
	return "track_points"
}

// TrackPointView is a lightweight projection for queries.
type TrackPointView struct {
	Latitude  float64
	Longitude float64
	CreatedAt time.Time
}
