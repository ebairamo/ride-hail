package service

import (
	"context"
	"fmt"
	"time"

	"ride-hail/internal/core/domain/models"
	"ride-hail/internal/core/domain/types"
	"ride-hail/internal/core/ports"
	"ride-hail/pkg/logger"
	"ride-hail/pkg/txm"
)

type DalService struct {
	log  *logger.Logger
	repo DalRepository
	txm  txm.Manager
}

type DalRepository struct {
	driver   ports.DriverRepository
	location ports.LocationRepository
}

func NewDalService(log *logger.Logger, txm txm.Manager, driverRepo ports.DriverRepository, locationRepo ports.LocationRepository) *DalService {
	return &DalService{
		log: log,
		txm: txm,
		repo: DalRepository{
			driver:   driverRepo,
			location: locationRepo,
		},
	}
}

func (svc *DalService) RegisterDriver(ctx context.Context, driver models.Driver) (string, error) {
	log := svc.log.Func("RegisterDriver")

	if driver.VehicleType == nil {
		log.Warn(ctx, "register_driver", "vehicle type is required")
		return "", fmt.Errorf("vehicle type is required")
	}

	if driver.Status == "" {
		driver.Status = types.DriverStatusOffline
	}

	if driver.Rating == 0 {
		driver.Rating = 5.0
	}

	id, err := svc.repo.driver.CreateDriver(ctx, driver)
	if err != nil {
		log.Error(ctx, "register_driver", "failed to create driver", "error", err)
		return "", fmt.Errorf("failed to register driver: %w", err)
	}

	log.Info(ctx, "register_driver", "driver registered successfully", "driver_id", id)
	return id, nil
}

func (svc *DalService) GetDriverProfile(ctx context.Context, driverID string) (*models.Driver, error) {
	log := svc.log.Func("GetDriverProfile")

	driver, err := svc.repo.driver.GetDriverByID(ctx, driverID)
	if err != nil {
		log.Error(ctx, "get_driver_profile", "failed to get driver", "error", err, "driver_id", driverID)
		return nil, fmt.Errorf("failed to get driver profile: %w", err)
	}

	log.Info(ctx, "get_driver_profile", "driver profile retrieved", "driver_id", driverID)
	return driver, nil
}

func (svc *DalService) UpdateDriverProfile(ctx context.Context, driver models.Driver) error {
	log := svc.log.Func("UpdateDriverProfile")

	err := svc.repo.driver.UpdateDriver(ctx, driver)
	if err != nil {
		log.Error(ctx, "update_driver_profile", "failed to update driver", "error", err, "driver_id", driver.ID)
		return fmt.Errorf("failed to update driver profile: %w", err)
	}

	log.Info(ctx, "update_driver_profile", "driver profile updated", "driver_id", driver.ID)
	return nil
}

func (svc *DalService) DeleteDriver(ctx context.Context, driverID string) error {
	log := svc.log.Func("DeleteDriver")

	err := svc.repo.driver.DeleteDriver(ctx, driverID)
	if err != nil {
		log.Error(ctx, "delete_driver", "failed to delete driver", "error", err, "driver_id", driverID)
		return fmt.Errorf("failed to delete driver: %w", err)
	}

	log.Info(ctx, "delete_driver", "driver deleted", "driver_id", driverID)
	return nil
}

func (svc *DalService) ChangeDriverStatus(ctx context.Context, driverID string, newStatus string, expectedStatus string) error {
	log := svc.log.Func("ChangeDriverStatus")

	err := svc.repo.driver.UpdateDriverWithStatusCondition(ctx, driverID, newStatus, expectedStatus)
	if err != nil {
		log.Error(ctx, "change_driver_status", "failed to change driver status", "error", err, "driver_id", driverID, "new_status", newStatus)
		return fmt.Errorf("failed to change driver status: %w", err)
	}

	log.Info(ctx, "change_driver_status", "driver status changed", "driver_id", driverID, "old_status", expectedStatus, "new_status", newStatus)
	return nil
}

func (svc *DalService) ListAvailableDriversNear(ctx context.Context, latitude, longitude float64, vehicleType string, radiusMeters int, limit int) ([]models.Driver, error) {
	log := svc.log.Func("ListAvailableDriversNear")

	if latitude < -90 || latitude > 90 {
		return nil, fmt.Errorf("invalid latitude: must be between -90 and 90")
	}
	if longitude < -180 || longitude > 180 {
		return nil, fmt.Errorf("invalid longitude: must be between -180 and 180")
	}

	drivers, err := svc.repo.driver.FindNearbyDrivers(ctx, latitude, longitude, vehicleType, radiusMeters, limit)
	if err != nil {
		log.Error(ctx, "list_available_drivers_near", "failed to find nearby drivers", "error", err)
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	log.Info(ctx, "list_available_drivers_near", fmt.Sprintf("found %d nearby drivers", len(drivers)), "count", len(drivers))
	return drivers, nil
}

func (svc *DalService) RecordDriverLocation(ctx context.Context, driverID string, latitude, longitude float64, accuracyMeters, speedKmh, headingDegrees *float64) (string, error) {
	log := svc.log.Func("RecordDriverLocation")

	if latitude < -90 || latitude > 90 {
		return "", fmt.Errorf("invalid latitude: must be between -90 and 90")
	}
	if longitude < -180 || longitude > 180 {
		return "", fmt.Errorf("invalid longitude: must be between -180 and 180")
	}

	if headingDegrees != nil {
		if *headingDegrees < 0 || *headingDegrees > 360 {
			return "", fmt.Errorf("invalid heading_degrees: must be between 0 and 360")
		}
	}

	coordinateID, err := svc.repo.location.UpdateDriverCurrentLocation(ctx, driverID, "", latitude, longitude, accuracyMeters, speedKmh, headingDegrees)
	if err != nil {
		log.Error(ctx, "record_driver_location", "failed to record location", "error", err, "driver_id", driverID)
		return "", fmt.Errorf("failed to record driver location: %w", err)
	}

	log.Info(ctx, "record_driver_location", "location recorded", "driver_id", driverID, "coordinate_id", coordinateID)
	return coordinateID, nil
}

func (svc *DalService) GetDriverLastLocation(ctx context.Context, driverID string) (*models.LocationHistory, error) {
	log := svc.log.Func("GetDriverLastLocation")

	location, err := svc.repo.location.GetLastLocationByDriver(ctx, driverID)
	if err != nil {
		log.Error(ctx, "get_driver_last_location", "failed to get last location", "error", err, "driver_id", driverID)
		return nil, fmt.Errorf("failed to get driver last location: %w", err)
	}

	if location == nil {
		log.Warn(ctx, "get_driver_last_location", "no location history found", "driver_id", driverID)
		return nil, nil
	}

	log.Info(ctx, "get_driver_last_location", "last location retrieved", "driver_id", driverID)
	return location, nil
}

func (svc *DalService) GetDriverLocationHistory(ctx context.Context, driverID string, limit int) ([]models.LocationHistory, error) {
	log := svc.log.Func("GetDriverLocationHistory")

	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	locations, err := svc.repo.location.GetLocationHistoryByDriver(ctx, driverID, limit)
	if err != nil {
		log.Error(ctx, "get_driver_location_history", "failed to get location history", "error", err, "driver_id", driverID)
		return nil, fmt.Errorf("failed to get driver location history: %w", err)
	}

	log.Info(ctx, "get_driver_location_history", fmt.Sprintf("retrieved %d location records", len(locations)), "driver_id", driverID, "count", len(locations))
	return locations, nil
}

func (svc *DalService) ClearOldLocations(ctx context.Context, driverID string, before time.Time) error {
	log := svc.log.Func("ClearOldLocations")

	err := svc.repo.location.DeleteLocationHistory(ctx, driverID, before)
	if err != nil {
		log.Error(ctx, "clear_old_locations", "failed to clear old locations", "error", err, "driver_id", driverID)
		return fmt.Errorf("failed to clear old locations: %w", err)
	}

	log.Info(ctx, "clear_old_locations", "old locations cleared", "driver_id", driverID)
	return nil
}

func (svc *DalService) StartDriverSession(ctx context.Context, driverID string) (string, error) {
	log := svc.log.Func("StartDriverSession")

	driver, err := svc.repo.driver.GetDriverByID(ctx, driverID)
	if err != nil {
		log.Error(ctx, "start_driver_session", "driver not found", "error", err, "driver_id", driverID)
		return "", fmt.Errorf("driver not found: %w", err)
	}

	sessionID, err := svc.repo.driver.GetActiveSessionID(ctx, driverID)
	if err != nil {
		log.Error(ctx, "start_driver_session", "failed to check active session", "error", err, "driver_id", driverID)
		return "", fmt.Errorf("failed to check active session: %w", err)
	}
	if sessionID != "" {
		log.Warn(ctx, "start_driver_session", "driver already has active session", "driver_id", driverID, "session_id", sessionID)
		return sessionID, nil
	}

	sessionID, err = svc.repo.driver.CreateSession(ctx, driverID)
	if err != nil {
		log.Error(ctx, "start_driver_session", "failed to create session", "error", err, "driver_id", driverID)
		return "", fmt.Errorf("failed to start driver session: %w", err)
	}

	if driver.Status != types.DriverStatusAvailable {
		err = svc.repo.driver.UpdateDriverStatus(ctx, driverID, types.DriverStatusAvailable)
		if err != nil {
			log.Warn(ctx, "start_driver_session", "failed to update driver status", "error", err, "driver_id", driverID)
			// здесь return не нужен
		}
	}

	log.Info(ctx, "start_driver_session", "driver session started", "driver_id", driverID, "session_id", sessionID)
	return sessionID, nil
}

func (svc *DalService) EndDriverSession(ctx context.Context, driverID string) (*models.DriverSession, error) {
	log := svc.log.Func("EndDriverSession")

	session, err := svc.repo.driver.EndSession(ctx, driverID)
	if err != nil {
		log.Error(ctx, "end_driver_session", "failed to end session", "error", err, "driver_id", driverID)
		return nil, fmt.Errorf("failed to end driver session: %w", err)
	}

	err = svc.repo.driver.UpdateDriverStatus(ctx, driverID, types.DriverStatusOffline)
	if err != nil {
		log.Warn(ctx, "end_driver_session", "failed to update driver status", "error", err, "driver_id", driverID)
		// здесь return не нужен
	}

	log.Info(ctx, "end_driver_session", "driver session ended", "driver_id", driverID, "session_id", session.ID)
	return session, nil
}

func (svc *DalService) FindNearbyDrivers(ctx context.Context, latitude, longitude float64, vehicleType string, radiusMeters int, limit int) ([]models.Driver, error) {
	return svc.ListAvailableDriversNear(ctx, latitude, longitude, vehicleType, radiusMeters, limit)
}
