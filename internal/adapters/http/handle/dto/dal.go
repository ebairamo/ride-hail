package dto

import (
	"errors"
	"fmt"

	"ride-hail/internal/core/domain/models"
	"ride-hail/internal/core/domain/types"
)

func ValidateDriver(d models.Driver) error {
	if d.ID == "" {
		return errors.New("driver ID (uuid string) is required")
	}
	if d.LicenseNumber == "" {
		return errors.New("license_number is required")
	}
	if len(d.LicenseNumber) > 50 {
		return errors.New("license_number too long (max 50)")
	}
	if d.Rating < 1.0 || d.Rating > 5.0 {
		return fmt.Errorf("rating must be between 1.0 and 5.0, got %v", d.Rating)
	}
	if d.TotalRides < 0 {
		return fmt.Errorf("total_rides must be >= 0, got %d", d.TotalRides)
	}
	if d.TotalEarnings < 0 {
		return fmt.Errorf("total_earnings must be >= 0, got %v", d.TotalEarnings)
	}
	validStatuses := []string{types.DriverStatusAvailable, types.DriverStatusBusy, types.DriverStatusEnRoute, types.DriverStatusOffline}
	validStatus := false
	for _, status := range validStatuses {
		if d.Status == status {
			validStatus = true
			break
		}
	}
	if !validStatus {
		return fmt.Errorf("invalid status: %q", d.Status)
	}
	return nil
}

func ValidateDriverSession(s *models.DriverSession) error {
	if s.ID == "" {
		return errors.New("driver_session id is required")
	}
	if s.DriverID == "" {
		return errors.New("driver_id is required")
	}
	if s.TotalRides < 0 {
		return fmt.Errorf("total_rides must be >= 0, got %d", s.TotalRides)
	}
	if s.TotalEarnings < 0 {
		return fmt.Errorf("total_earnings must be >= 0, got %v", s.TotalEarnings)
	}
	return nil
}

func ValidateLocationHistory(lh *models.LocationHistory) error {
	if lh.ID == "" {
		return errors.New("location_history id is required")
	}
	if lh.Latitude < -90 || lh.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90, got %v", lh.Latitude)
	}
	if lh.Longitude < -180 || lh.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180, got %v", lh.Longitude)
	}
	if lh.HeadingDegrees != nil {
		if *lh.HeadingDegrees < 0 || *lh.HeadingDegrees > 360 {
			return fmt.Errorf("heading_degrees must be between 0 and 360, got %v", *lh.HeadingDegrees)
		}
	}
	return nil
}

// Request DTOs
type DriverGoOnlineRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type UpdateDriverLocationRequest struct {
	Latitude       float64  `json:"latitude"`
	Longitude      float64  `json:"longitude"`
	AccuracyMeters *float64 `json:"accuracy_meters,omitempty"`
	SpeedKmh       *float64 `json:"speed_kmh,omitempty"`
	HeadingDegrees *float64 `json:"heading_degrees,omitempty"`
}

type StartRideRequest struct {
	RideID         string `json:"ride_id"`
	DriverLocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"driver_location"`
}

type CompleteRideRequest struct {
	RideID        string `json:"ride_id"`
	FinalLocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"final_location"`
	ActualDistanceKm      float64 `json:"actual_distance_km"`
	ActualDurationMinutes int     `json:"actual_duration_minutes"`
}

// Response DTOs
type DriverGoOnlineResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type DriverGoOfflineResponse struct {
	Status         string `json:"status"`
	SessionID      string `json:"session_id"`
	SessionSummary struct {
		DurationHours  float64 `json:"duration_hours"`
		RidesCompleted int     `json:"rides_completed"`
		Earnings       float64 `json:"earnings"`
	} `json:"session_summary"`
	Message string `json:"message"`
}

type UpdateDriverLocationResponse struct {
	CoordinateID string `json:"coordinate_id"`
	UpdatedAt    string `json:"updated_at"`
}

type StartRideResponse struct {
	RideID    string `json:"ride_id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
	Message   string `json:"message"`
}

type CompleteRideResponse struct {
	RideID         string  `json:"ride_id"`
	Status         string  `json:"status"`
	CompletedAt    string  `json:"completed_at"`
	DriverEarnings float64 `json:"driver_earnings"`
	Message        string  `json:"message"`
}

func ValidateDriverGoOnlineRequest(req DriverGoOnlineRequest) error {
	if req.Latitude < -90 || req.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90, got %v", req.Latitude)
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180, got %v", req.Longitude)
	}
	return nil
}

func ValidateUpdateDriverLocationRequest(req UpdateDriverLocationRequest) error {
	if req.Latitude < -90 || req.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90, got %v", req.Latitude)
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180, got %v", req.Longitude)
	}
	if req.HeadingDegrees != nil {
		if *req.HeadingDegrees < 0 || *req.HeadingDegrees > 360 {
			return fmt.Errorf("heading_degrees must be between 0 and 360, got %v", *req.HeadingDegrees)
		}
	}
	return nil
}

func ValidateStartRideRequest(req StartRideRequest) error {
	if req.RideID == "" {
		return errors.New("ride_id is required")
	}
	if req.DriverLocation.Latitude < -90 || req.DriverLocation.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90, got %v", req.DriverLocation.Latitude)
	}
	if req.DriverLocation.Longitude < -180 || req.DriverLocation.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180, got %v", req.DriverLocation.Longitude)
	}
	return nil
}

func ValidateCompleteRideRequest(req CompleteRideRequest) error {
	if req.RideID == "" {
		return errors.New("ride_id is required")
	}
	if req.FinalLocation.Latitude < -90 || req.FinalLocation.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90, got %v", req.FinalLocation.Latitude)
	}
	if req.FinalLocation.Longitude < -180 || req.FinalLocation.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180, got %v", req.FinalLocation.Longitude)
	}
	if req.ActualDistanceKm < 0 {
		return fmt.Errorf("actual_distance_km must be >= 0, got %v", req.ActualDistanceKm)
	}
	if req.ActualDurationMinutes < 0 {
		return fmt.Errorf("actual_duration_minutes must be >= 0, got %d", req.ActualDurationMinutes)
	}
	return nil
}
