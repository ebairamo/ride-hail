package postgres

import (
	"context"
	"fmt"
	"ride-hail/internal/core/domain/models"
	"ride-hail/pkg/executor"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminRepository struct {
	pool *pgxpool.Pool
}

func NewAdminRepository(pool *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{
		pool: pool,
	}
}

func (repo *AdminRepository) GetDriverDistribution(ctx context.Context) (map[string]int, error) {
	ex := executor.GetExecutor(ctx, repo.pool)

	query := `SELECT vehicle_type, COUNT(*) as count 
	FROM drivers 
	WHERE status IN ('AVAILABLE', 'BUSY', 'EN_ROUTE') 
	GROUP BY vehicle_type; `

	rows, err := ex.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get [название данных]: %w", err)
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var vehicleType string
		var count int
		err := rows.Scan(&vehicleType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan driver distribution row: %w", err)
		}
		result[vehicleType] = count
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating driver distribution rows: %w", err)
	}
	return result, nil

}

func (repo *AdminRepository) GetSystemMetrics(ctx context.Context) (models.Metrics, error) {
	ex := executor.GetExecutor(ctx, repo.pool)

	metrics := models.Metrics{}
	err := ex.QueryRow(ctx, "SELECT COUNT(*) FROM rides WHERE status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')").Scan(&metrics.ActiveRides)
	if err != nil {
		return models.Metrics{}, fmt.Errorf("failed to get active rides count: %w", err)
	}

	err = ex.QueryRow(ctx, "SELECT COUNT(*) FROM drivers WHERE status = 'AVAILABLE'").Scan(&metrics.AvailableDrivers)
	if err != nil {
		return models.Metrics{}, fmt.Errorf("failed to get available drivers count: %w", err)
	}

	err = ex.QueryRow(ctx, "SELECT COUNT(*) FROM drivers WHERE status IN ('BUSY', 'EN_ROUTE')").Scan(&metrics.BusyDrivers)
	if err != nil {
		return models.Metrics{}, fmt.Errorf("failed to get busy drivers count: %w", err)
	}

	err = ex.QueryRow(ctx, "SELECT COUNT(*) FROM rides WHERE DATE(created_at) = CURRENT_DATE").Scan(&metrics.TotalRidesToday)
	if err != nil {
		return models.Metrics{}, fmt.Errorf("failed to get total rides today count: %w", err)
	}

	err = ex.QueryRow(ctx, "SELECT COALESCE(SUM(final_fare), 0) FROM rides WHERE DATE(completed_at) = CURRENT_DATE AND status = 'COMPLETED'").Scan(&metrics.TotalRevenueToday)
	if err != nil {
		return models.Metrics{}, fmt.Errorf("failed to get total revenue today: %w", err)
	}

	err = ex.QueryRow(ctx, "SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (started_at - matched_at)) / 60), 0) FROM rides WHERE started_at IS NOT NULL AND matched_at IS NOT NULL AND DATE(created_at) = CURRENT_DATE").Scan(&metrics.AverageWaitTimeMinutes)
	if err != nil {
		return models.Metrics{}, fmt.Errorf("failed to get average wait time: %w", err)
	}

	err = ex.QueryRow(ctx, "SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) / 60), 0) FROM rides WHERE completed_at IS NOT NULL AND started_at IS NOT NULL AND DATE(created_at) = CURRENT_DATE").Scan(&metrics.AverageRideDurationMinutes)
	if err != nil {
		return models.Metrics{}, fmt.Errorf("failed to get average ride duration: %w", err)
	}

	err = ex.QueryRow(ctx, "SELECT COALESCE((SELECT COUNT(*) FROM rides WHERE status = 'CANCELLED' AND DATE(created_at) = CURRENT_DATE) / NULLIF((SELECT COUNT(*) FROM rides WHERE DATE(created_at) = CURRENT_DATE), 0)::float, 0)").Scan(&metrics.CancellationRate)
	if err != nil {
		return models.Metrics{}, fmt.Errorf("failed to get cancellation rate: %w", err)
	}

	return metrics, nil
}

func (repo *AdminRepository) GetHotspots(ctx context.Context, limit int) ([]models.Hotspot, error) {
	ex := executor.GetExecutor(ctx, repo.pool)

	query := `
	SELECT 
		c.address as location,
		COUNT(CASE WHEN r.status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS') THEN 1 END) as active_rides,
		COUNT(CASE WHEN d.status = 'AVAILABLE' THEN 1 END) as waiting_drivers
	FROM coordinates c
	LEFT JOIN rides r ON c.id = r.pickup_coordinate_id
	LEFT JOIN drivers d ON c.entity_id = d.id AND c.entity_type = 'driver'
	GROUP BY c.address
	HAVING 
		COUNT(CASE WHEN r.status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS') THEN 1 END) > 0
		OR COUNT(CASE WHEN d.status = 'AVAILABLE' THEN 1 END) > 0
	ORDER BY (
		COUNT(CASE WHEN r.status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS') THEN 1 END) + 
		COUNT(CASE WHEN d.status = 'AVAILABLE' THEN 1 END)
	) DESC
	LIMIT $1
	`

	rows, err := ex.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get hotspots: %w", err)
	}
	defer rows.Close()

	var hotspots []models.Hotspot
	for rows.Next() {
		var hotspot models.Hotspot
		err := rows.Scan(
			&hotspot.Location,
			&hotspot.ActiveRides,
			&hotspot.WaitingDrivers,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hotspot row: %w", err)
		}
		hotspots = append(hotspots, hotspot)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating hotspots: %w", err)
	}

	return hotspots, nil
}

func (repo *AdminRepository) GetActiveRides(ctx context.Context, pagination models.AdminPaginationParams) (models.ActiveRidesResponse, error) {
	activeRides, totalCount, err := repo.getActiveRidesWithPagination(ctx, pagination.Page, pagination.PageSize)
	if err != nil {
		return models.ActiveRidesResponse{}, err
	}

	response := models.ActiveRidesResponse{
		Rides:      activeRides,
		TotalCount: totalCount,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
	}

	return response, nil
}

// Вспомогательный метод для получения активных поездок с пагинацией
func (repo *AdminRepository) getActiveRidesWithPagination(ctx context.Context, page int, pageSize int) ([]models.ActiveRide, int, error) {
	ex := executor.GetExecutor(ctx, repo.pool)

	// Запрос для подсчета общего количества активных поездок
	var totalCount int
	countQuery := `SELECT COUNT(*) 
	FROM rides 
	WHERE status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')`

	err := ex.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total active rides count: %w", err)
	}

	// Вычисляем смещение для пагинации
	offset := (page - 1) * pageSize

	// Основной запрос для получения списка активных поездок
	query := `
	SELECT 
		r.id AS ride_id,
		r.ride_number,
		r.status,
		r.passenger_id,
		r.driver_id,
		pickup.address AS pickup_address,
		dest.address AS destination_address,
		r.started_at,
		-- Расчет предполагаемого времени завершения: начало + длительность (из coordinate.duration_minutes)
		CASE 
			WHEN r.started_at IS NOT NULL AND dest.duration_minutes IS NOT NULL 
			THEN r.started_at + (dest.duration_minutes * INTERVAL '1 minute')
			ELSE NOW() + INTERVAL '15 minutes' -- Значение по умолчанию, если нет точных данных
		END AS estimated_completion,
		-- Текущее местоположение водителя из последней записи в location_history
		COALESCE(lh.latitude, pickup.latitude) AS current_latitude,
		COALESCE(lh.longitude, pickup.longitude) AS current_longitude,
		-- Расчет пройденного расстояния (упрощенно: процент от общего расстояния)
		CASE
			WHEN r.started_at IS NOT NULL AND dest.distance_km IS NOT NULL
			THEN dest.distance_km * 
				 EXTRACT(EPOCH FROM (NOW() - r.started_at)) / 
				 GREATEST(EXTRACT(EPOCH FROM (NOW() - r.started_at)) + COALESCE(dest.duration_minutes * 60, 0), 1)
			ELSE 0
		END AS distance_completed_km,
		-- Оставшееся расстояние
		CASE
			WHEN dest.distance_km IS NOT NULL
			THEN dest.distance_km - (
				CASE
					WHEN r.started_at IS NOT NULL
					THEN dest.distance_km * 
						 EXTRACT(EPOCH FROM (NOW() - r.started_at)) / 
						 GREATEST(EXTRACT(EPOCH FROM (NOW() - r.started_at)) + COALESCE(dest.duration_minutes * 60, 0), 1)
					ELSE 0
				END
			)
			ELSE 0
		END AS distance_remaining_km
	FROM 
		rides r
	-- Присоединяем координаты точки отправления
	JOIN 
		coordinates pickup ON r.pickup_coordinate_id = pickup.id
	-- Присоединяем координаты точки назначения
	JOIN 
		coordinates dest ON r.destination_coordinate_id = dest.id
	-- Левое соединение с последним местоположением водителя
	LEFT JOIN (
		SELECT 
			driver_id,
			latitude,
			longitude,
			recorded_at
		FROM 
			location_history lh1
		WHERE 
			recorded_at = (
				SELECT MAX(recorded_at) 
				FROM location_history lh2 
				WHERE lh2.driver_id = lh1.driver_id
			)
	) lh ON r.driver_id = lh.driver_id
	WHERE 
		r.status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
	ORDER BY 
		r.created_at DESC
	LIMIT $1
	OFFSET $2
	`

	rows, err := ex.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get active rides: %w", err)
	}
	defer rows.Close()

	var activeRides []models.ActiveRide
	for rows.Next() {
		var ride models.ActiveRide
		var currentLatitude, currentLongitude float64

		err := rows.Scan(
			&ride.RideID,
			&ride.RideNumber,
			&ride.Status,
			&ride.PassengerID,
			&ride.DriverID,
			&ride.PickupAddress,
			&ride.DestinationAddress,
			&ride.StartedAt,
			&ride.EstimatedCompletion,
			&currentLatitude,
			&currentLongitude,
			&ride.DistanceCompletedKm,
			&ride.DistanceRemainingKm,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan active ride row: %w", err)
		}

		// Заполняем местоположение водителя
		ride.CurrentDriverLocation = models.Location{
			Latitude:  currentLatitude,
			Longitude: currentLongitude,
		}

		activeRides = append(activeRides, ride)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating active rides: %w", err)
	}

	return activeRides, totalCount, nil
}
