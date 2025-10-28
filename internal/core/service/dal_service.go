package service

import (
	"context"
	"time"

	"ride-hail/internal/core/domain/models"
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
	return "", nil
}

func (svc *DalService) GetDriverProfile(ctx context.Context, driverID string) (*models.Driver, error) {
	return &models.Driver{}, nil
}

func (svc *DalService) UpdateDriverProfile(ctx context.Context, driver models.Driver) error {
	return nil
}

func (svc *DalService) DeleteDriver(ctx context.Context, driverID string) error {
	return nil
}

func (svc *DalService) ChangeDriverStatus(ctx context.Context, driverID string, newStatus string, expectedStatus string) error {
	return nil
}

func (svc *DalService) ListAvailableDriversNear(ctx context.Context) ([]models.Driver, error) {
	return []models.Driver{}, nil
}

func (svc *DalService) RecordDriverLocation(ctx context.Context, location models.LocationHistory) (string, error) {
	return "", nil
}

func (svc *DalService) GetDriverLastLocation(ctx context.Context, driverID string) (*models.LocationHistory, error) {
	return &models.LocationHistory{}, nil
}

func (svc *DalService) GetDriverLocationHistory(ctx context.Context, driverID string, limit int) ([]models.LocationHistory, error) {
	return []models.LocationHistory{}, nil
}

func (svc *DalService) ClearOldLocations(ctx context.Context, driverID string, before time.Time) error {
	return nil
}

func (svc *DalService) StartDriverSession(ctx context.Context, driverID string) (string, error) {
	return "", nil
}

func (svc *DalService) EndDriverSession(ctx context.Context, sessionID string) error {
	return nil
}
