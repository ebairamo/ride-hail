package ports

import (
	"context"
	"ride-hail/internal/core/domain/models"
)

type AdminRepository interface {
	GetActiveRides(ctx context.Context, page int, pageSize int) ([]models.ActiveRide, int, error)
	GetSystemMetrics(ctx context.Context) (models.Metrics, error)
	GetDriverDistribution(ctx context.Context) (map[string]int, error)
	GetHotspots(ctx context.Context, limit int) ([]models.Hotspot, error)
}

type AdminService interface {
	GetSystemOverview(ctx context.Context) (models.SystemOverview, error)
	GetActiveRides(ctx context.Context, pagination models.AdminPaginationParams) (models.ActiveRidesResponse, error)
}
