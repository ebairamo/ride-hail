package service

import (
	"context"
	"ride-hail/internal/core/domain/action"
	"ride-hail/internal/core/domain/models"
	"ride-hail/internal/core/ports"
	"ride-hail/pkg/logger"
	"time"
)

type AdminService struct {
	log  *logger.Logger
	repo ports.AdminRepository
}

func NewAdminService(log *logger.Logger, repo ports.AdminRepository) *AdminService {
	return &AdminService{
		log:  log,
		repo: repo,
	}
}

func (s *AdminService) GetSystemOverview(ctx context.Context) (models.SystemOverview, error) {
	log := s.log.Func("GetSystemOverview")

	metrics, err := s.repo.GetSystemMetrics(ctx)
	if err != nil {
		log.Error(ctx, action.GetSystemOverview, "failed to get system metrics", "error", err)
		return models.SystemOverview{}, err
	}

	distribution, err := s.repo.GetDriverDistribution(ctx)
	if err != nil {
		log.Error(ctx, action.GetSystemOverview, "failed to get driver distribution", "error", err)
		return models.SystemOverview{}, err
	}

	hotspots, err := s.repo.GetHotspots(ctx, 5)
	if err != nil {
		log.Error(ctx, action.GetSystemOverview, "failed to get hotspot", "error", err)
		return models.SystemOverview{}, err
	}

	overview := models.SystemOverview{
		Timestamp:          time.Now(),
		Metrics:            metrics,
		DriverDistribution: distribution,
		Hotspots:           hotspots,
	}
	log.Info(ctx, action.GetSystemOverview, "system overview retrieved successfully")
	return overview, nil
}

func (s *AdminService) GetActiveRides(ctx context.Context, params models.AdminPaginationParams) (models.ActiveRidesResponse, error) {
	log := s.log.Func("GetActiveRides")
	if params.Page < 1 || params.PageSize < 1 {
		log.Error(ctx, action.GetActiveRides, "Page or Page size is invalid")
		return models.ActiveRidesResponse{}, nil
	}
	if params.PageSize > 100 {
		params.PageSize = 100
		log.Info(ctx, action.GetActiveRides, "Page size can not be bigger than 100, now Page size is 100")
	}

	activeRides, numberOfRides, err := s.repo.GetActiveRides(ctx, params.Page, params.PageSize)
	if err != nil {
		log.Error(ctx, action.GetActiveRides, "failed to get active rides", "error", err)
		return models.ActiveRidesResponse{}, err
	}
	response := models.ActiveRidesResponse{
		Rides:      activeRides,
		TotalCount: numberOfRides,
		Page:       params.Page,
		PageSize:   params.PageSize,
	}
	log.Info(ctx, action.GetActiveRides, "active rides retrieved successfully",
		"page", params.Page,
		"page_size", params.PageSize,
		"total_count", numberOfRides)
	return response, nil
}
