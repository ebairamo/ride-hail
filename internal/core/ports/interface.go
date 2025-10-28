package ports

import (
	"context"
	"time"

	"ride-hail/internal/core/domain/models"
)

type AuthServices interface {
	Run() error
}

// auth ports

type AuthService interface {
	CreateNewUser(ctx context.Context, user models.User) error
	Login(ctx context.Context, user models.User) (string, error)
}

type UserRepository interface {
	CreateNewUser(ctx context.Context, user models.User) error
	GetGyUserEmail(ctx context.Context, email string) (models.User, error)
}

// ride ports
type RideService interface {
	CreateNewRide(ctx context.Context, r models.CreateRideRequest) (models.CreateRideResponse, error)
	CloseRide(ctx context.Context, req models.CloseRideRequest) (models.CloseRideResponse, error)
}

type RidePublisher interface {
	Publish(exName, queue string, message []byte) error
}

type RideRepository interface {
	CreateNewRide(ctx context.Context, ride models.Ride) (string, error)
	GetRide(ctx context.Context, id string) (models.Ride, error)
	GenerateRideNumber(ctx context.Context) (int, error)
}

type CoordinatesRepository interface {
	CreateNewCoordinate(ctx context.Context, c models.Coordinate) (string, error)
	GetCoordinate(ctx context.Context, id string) (models.Coordinate, error)
}

// dal ports
type DriverRepository interface {
	CreateDriver(ctx context.Context, driver models.Driver) (string, error)
	GetDriverByID(ctx context.Context, id string) (*models.Driver, error)
	UpdateDriver(ctx context.Context, driver models.Driver) error
	DeleteDriver(ctx context.Context, id string) error
	ListDriversByStatus(ctx context.Context, status string, limit, offset int) ([]models.Driver, error)
	UpdateDriverStatus(ctx context.Context, driverID string, newStatus string) error
}

type LocationRepository interface {
	SaveLocation(ctx context.Context, location models.LocationHistory) (string, error)
	GetLastLocationByDriver(ctx context.Context, driverID string) (*models.LocationHistory, error)
	GetLocationHistoryByDriver(ctx context.Context, driverID string, limit int) ([]models.LocationHistory, error)
	DeleteLocationHistory(ctx context.Context, driverID string, before time.Time) error
}

type DalService interface {
	RegisterDriver(ctx context.Context, driver models.Driver) (string, error)
	GetDriverProfile(ctx context.Context, driverID string) (*models.Driver, error)
	UpdateDriverProfile(ctx context.Context, driver models.Driver) error
	DeleteDriver(ctx context.Context, driverID string) error

	ChangeDriverStatus(ctx context.Context, driverID string, newStatus string, expectedStatus string) error
	ListAvailableDriversNear(ctx context.Context) ([]models.Driver, error)
	RecordDriverLocation(ctx context.Context, location models.LocationHistory) (string, error)

	GetDriverLastLocation(ctx context.Context, driverID string) (*models.LocationHistory, error)
	GetDriverLocationHistory(ctx context.Context, driverID string, limit int) ([]models.LocationHistory, error)

	ClearOldLocations(ctx context.Context, driverID string, before time.Time) error

	StartDriverSession(ctx context.Context, driverID string) (string, error)
	EndDriverSession(ctx context.Context, sessionID string) error
}
