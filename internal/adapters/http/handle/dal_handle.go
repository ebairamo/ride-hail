package handle

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"ride-hail/internal/adapters/http/handle/dto"
	"ride-hail/internal/core/domain/action"
	"ride-hail/internal/core/domain/types"
	"ride-hail/internal/core/ports"
	"ride-hail/pkg/logger"
)

type DalHandle struct {
	svc ports.DalService
	log *logger.Logger
}

func NewDalHandle(svc ports.DalService, log *logger.Logger) *DalHandle {
	return &DalHandle{
		svc: svc,
		log: log,
	}
}

type DalHandler interface {
	DriverGoesOnline(w http.ResponseWriter, r *http.Request)
	DriverGoesOffline(w http.ResponseWriter, r *http.Request)
	UpdateDriverLocation(w http.ResponseWriter, r *http.Request)
	StartRide(w http.ResponseWriter, r *http.Request)
	CompleteRide(w http.ResponseWriter, r *http.Request)
}

func (h *DalHandle) DriverGoesOnline(w http.ResponseWriter, r *http.Request) {
	log := h.log.Func("DalHandle.DriverGoesOnline")
	ctx := r.Context()

	driverID := extractDriverID(r)
	if driverID == "" {
		log.Error(ctx, action.StartDriverSession, "invalid driver_id")
		http.Error(w, "invalid driver_id", http.StatusBadRequest)
		return
	}

	log.Debug(ctx, action.StartDriverSession, "driver going online", "driver_id", driverID)

	var req dto.DriverGoOnlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, action.StartDriverSession, "error parsing request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := dto.ValidateDriverGoOnlineRequest(req); err != nil {
		log.Error(ctx, action.StartDriverSession, "validation error", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Start driver session
	sessionID, err := h.svc.StartDriverSession(ctx, driverID)
	if err != nil {
		log.Error(ctx, action.StartDriverSession, "failed to start session", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record initial location
	_, err = h.svc.RecordDriverLocation(ctx, driverID, req.Latitude, req.Longitude, nil, nil, nil)
	if err != nil {
		log.Warn(ctx, action.StartDriverSession, "failed to record initial location", "error", err)
		// Don't fail the request, just log the warning
	}

	response := dto.DriverGoOnlineResponse{
		Status:    "AVAILABLE",
		SessionID: sessionID,
		Message:   "You are now online and ready to accept rides",
	}

	log.Info(ctx, action.StartDriverSession, "driver is now online", "driver_id", driverID, "session_id", sessionID)
	writeJSON(w, http.StatusOK, response)
}

func (h *DalHandle) DriverGoesOffline(w http.ResponseWriter, r *http.Request) {
	log := h.log.Func("DalHandle.DriverGoesOffline")
	ctx := r.Context()

	driverID := extractDriverID(r)
	if driverID == "" {
		log.Error(ctx, action.EndDriverSession, "invalid driver_id")
		http.Error(w, "invalid driver_id", http.StatusBadRequest)
		return
	}

	log.Debug(ctx, action.EndDriverSession, "driver going offline", "driver_id", driverID)

	// End driver session
	session, err := h.svc.EndDriverSession(ctx, driverID)
	if err != nil {
		log.Error(ctx, action.EndDriverSession, "failed to end session", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate session duration
	var durationHours float64
	if session.EndedAt != nil {
		durationHours = session.EndedAt.Sub(session.StartedAt).Hours()
	}

	response := dto.DriverGoOfflineResponse{
		Status:    "OFFLINE",
		SessionID: session.ID,
		SessionSummary: struct {
			DurationHours  float64 `json:"duration_hours"`
			RidesCompleted int     `json:"rides_completed"`
			Earnings       float64 `json:"earnings"`
		}{
			DurationHours:  durationHours,
			RidesCompleted: session.TotalRides,
			Earnings:       session.TotalEarnings,
		},
		Message: "You are now offline",
	}

	log.Info(ctx, action.EndDriverSession, "driver is now offline", "driver_id", driverID, "session_id", session.ID)
	writeJSON(w, http.StatusOK, response)
}

func (h *DalHandle) UpdateDriverLocation(w http.ResponseWriter, r *http.Request) {
	log := h.log.Func("DalHandle.UpdateDriverLocation")
	ctx := r.Context()

	driverID := extractDriverID(r)
	if driverID == "" {
		log.Error(ctx, action.UpdateDriverLocation, "invalid driver_id")
		http.Error(w, "invalid driver_id", http.StatusBadRequest)
		return
	}

	log.Debug(ctx, action.UpdateDriverLocation, "updating driver location", "driver_id", driverID)

	var req dto.UpdateDriverLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, action.UpdateDriverLocation, "error parsing request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := dto.ValidateUpdateDriverLocationRequest(req); err != nil {
		log.Error(ctx, action.UpdateDriverLocation, "validation error", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Record driver location
	coordinateID, err := h.svc.RecordDriverLocation(ctx, driverID, req.Latitude, req.Longitude,
		req.AccuracyMeters, req.SpeedKmh, req.HeadingDegrees)
	if err != nil {
		log.Error(ctx, action.UpdateDriverLocation, "failed to update location", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := dto.UpdateDriverLocationResponse{
		CoordinateID: coordinateID,
		UpdatedAt:    time.Now().Format(time.RFC3339),
	}

	log.Info(ctx, action.UpdateDriverLocation, "driver location updated", "driver_id", driverID, "coordinate_id", coordinateID)
	writeJSON(w, http.StatusOK, response)
}

func (h *DalHandle) StartRide(w http.ResponseWriter, r *http.Request) {
	log := h.log.Func("DalHandle.StartRide")
	ctx := r.Context()

	driverID := extractDriverID(r)
	if driverID == "" {
		log.Error(ctx, action.StartRide, "invalid driver_id")
		http.Error(w, "invalid driver_id", http.StatusBadRequest)
		return
	}

	log.Debug(ctx, action.StartRide, "starting ride", "driver_id", driverID)

	var req dto.StartRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, action.StartRide, "error parsing request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := dto.ValidateStartRideRequest(req); err != nil {
		log.Error(ctx, action.StartRide, "validation error", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update driver status to BUSY
	err := h.svc.ChangeDriverStatus(ctx, driverID, types.DriverStatusBusy, types.DriverStatusAvailable)
	if err != nil {
		log.Error(ctx, action.StartRide, "failed to change driver status", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record driver location
	_, err = h.svc.RecordDriverLocation(ctx, driverID, req.DriverLocation.Latitude,
		req.DriverLocation.Longitude, nil, nil, nil)
	if err != nil {
		log.Warn(ctx, action.StartRide, "failed to record location", "error", err)
		// Don't fail the request
	}

	response := dto.StartRideResponse{
		RideID:    req.RideID,
		Status:    "BUSY",
		StartedAt: time.Now().Format(time.RFC3339),
		Message:   "Ride started successfully",
	}

	log.Info(ctx, action.StartRide, "ride started", "driver_id", driverID, "ride_id", req.RideID)
	writeJSON(w, http.StatusOK, response)
}

func (h *DalHandle) CompleteRide(w http.ResponseWriter, r *http.Request) {
	log := h.log.Func("DalHandle.CompleteRide")
	ctx := r.Context()

	driverID := extractDriverID(r)
	if driverID == "" {
		log.Error(ctx, action.CompleteRide, "invalid driver_id")
		http.Error(w, "invalid driver_id", http.StatusBadRequest)
		return
	}

	log.Debug(ctx, action.CompleteRide, "completing ride", "driver_id", driverID)

	var req dto.CompleteRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, action.CompleteRide, "error parsing request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := dto.ValidateCompleteRideRequest(req); err != nil {
		log.Error(ctx, action.CompleteRide, "validation error", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Record final location
	_, err := h.svc.RecordDriverLocation(ctx, driverID, req.FinalLocation.Latitude,
		req.FinalLocation.Longitude, nil, nil, nil)
	if err != nil {
		log.Warn(ctx, action.CompleteRide, "failed to record final location", "error", err)
	}

	// Update driver status back to AVAILABLE
	err = h.svc.ChangeDriverStatus(ctx, driverID, types.DriverStatusAvailable, types.DriverStatusBusy)
	if err != nil {
		log.Error(ctx, action.CompleteRide, "failed to change driver status", "error", err)
		// Don't fail the request, driver is completing a ride
	}

	// Calculate driver earnings (80% of estimated fare)
	// For now, we'll use a placeholder. In a real implementation,
	// we'd fetch the ride details and calculate the actual earnings.
	driverEarnings := req.ActualDistanceKm * 100 // Placeholder calculation

	response := dto.CompleteRideResponse{
		RideID:         req.RideID,
		Status:         "AVAILABLE",
		CompletedAt:    time.Now().Format(time.RFC3339),
		DriverEarnings: driverEarnings,
		Message:        "Ride completed successfully",
	}

	log.Info(ctx, action.CompleteRide, "ride completed", "driver_id", driverID, "ride_id", req.RideID)
	writeJSON(w, http.StatusOK, response)
}

func extractDriverID(r *http.Request) string {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[2]
}

func extractRideID(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "rides" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
