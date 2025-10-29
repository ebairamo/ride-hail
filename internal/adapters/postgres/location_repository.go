package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ride-hail/internal/core/domain/models"
	"ride-hail/pkg/executor"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LocationRepository struct {
	pool *pgxpool.Pool
}

func NewLocationRepository(pool *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{
		pool: pool,
	}
}

func (repo *LocationRepository) SaveLocation(ctx context.Context, location models.LocationHistory) (string, error) {
	ex := executor.GetExecutor(ctx, repo.pool)

	query := `INSERT INTO location_history (id, coordinate_id, driver_id, latitude, 
	longitude, accuracy_meters, speed_kmh, heading_degrees, recorded_at, ride_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	RETURNING id;`

	var id string
	err := ex.QueryRow(
		ctx, query,
		location.ID,
		location.CoordinateID,
		location.DriverID,
		location.Latitude,
		location.Longitude,
		location.AccuracyMeters,
		location.SpeedKmh,
		location.HeadingDegrees,
		location.RecordedAt,
		location.RideID,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to save location: %w", err)
	}

	return id, nil
}

func (repo *LocationRepository) GetLastLocationByDriver(ctx context.Context, driverID string) (*models.LocationHistory, error) {
	ex := executor.GetExecutor(ctx, repo.pool)

	query := `SELECT id, coordinate_id, driver_id, latitude, 
	longitude, accuracy_meters, speed_kmh, heading_degrees, recorded_at, ride_id
	FROM location_history
	WHERE driver_id = $1
	ORDER BY recorded_at DESC
	LIMIT 1;`

	var location models.LocationHistory
	err := ex.QueryRow(ctx, query, driverID).Scan(
		&location.ID,
		&location.CoordinateID,
		&location.DriverID,
		&location.Latitude,
		&location.Longitude,
		&location.AccuracyMeters,
		&location.SpeedKmh,
		&location.HeadingDegrees,
		&location.RecordedAt,
		&location.RideID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last location for driver %s: %w", driverID, err)
	}

	return &location, nil
}

func (repo *LocationRepository) GetLocationHistoryByDriver(ctx context.Context, driverID string, limit int) ([]models.LocationHistory, error) {
	ex := executor.GetExecutor(ctx, repo.pool)

	query := `SELECT id, coordinate_id, driver_id, latitude, 
	longitude, accuracy_meters, speed_kmh, heading_degrees, recorded_at, ride_id
	FROM location_history
	WHERE driver_id = $1
	ORDER BY recorded_at DESC
	LIMIT $2;`

	rows, err := ex.Query(ctx, query, driverID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get location history for driver %s: %w", driverID, err)
	}
	defer rows.Close()

	var locations []models.LocationHistory
	for rows.Next() {
		var location models.LocationHistory
		err := rows.Scan(
			&location.ID,
			&location.CoordinateID,
			&location.DriverID,
			&location.Latitude,
			&location.Longitude,
			&location.AccuracyMeters,
			&location.SpeedKmh,
			&location.HeadingDegrees,
			&location.RecordedAt,
			&location.RideID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan location history: %w", err)
		}
		locations = append(locations, location)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating location history: %w", err)
	}

	return locations, nil
}

func (repo *LocationRepository) DeleteLocationHistory(ctx context.Context, driverID string, before time.Time) error {
	ex := executor.GetExecutor(ctx, repo.pool)

	query := `DELETE FROM location_history WHERE driver_id = $1 AND recorded_at < $2;`

	result, err := ex.Exec(ctx, query, driverID, before)
	if err != nil {
		return fmt.Errorf("failed to delete location history: %w", err)
	}

	_ = result.RowsAffected()

	return nil
}

// UpdateDriverCurrentLocation updates the current location for a driver
func (repo *LocationRepository) UpdateDriverCurrentLocation(ctx context.Context, driverID string, coordinateID string, latitude, longitude float64, accuracyMeters, speedKmh, headingDegrees *float64) (string, error) {
	ex := executor.GetExecutor(ctx, repo.pool)

	// First, set all previous current locations to false
	updatePrevQuery := `UPDATE coordinates SET is_current = false WHERE entity_id = $1 AND entity_type = 'driver';`
	_, err := ex.Exec(ctx, updatePrevQuery, driverID)
	if err != nil {
		return "", fmt.Errorf("failed to update previous locations: %w", err)
	}

	// If coordinate_id is provided, update existing coordinate
	if coordinateID != "" {
		updateQuery := `UPDATE coordinates 
			SET latitude = $1, longitude = $2, distance_km = $3, duration_minutes = $4,
				is_current = true, updated_at = now()
			WHERE id = $5 AND entity_id = $6 AND entity_type = 'driver'
			RETURNING id;`
		var id string
		err = ex.QueryRow(ctx, updateQuery, latitude, longitude, accuracyMeters, speedKmh, coordinateID, driverID).Scan(&id)
		if err != nil {
			return "", fmt.Errorf("failed to update coordinate: %w", err)
		}
		return id, nil
	}

	// Otherwise, create new coordinate
	insertQuery := `INSERT INTO coordinates (id, entity_id, entity_type, latitude, longitude, distance_km, is_current)
		VALUES (gen_random_uuid(), $1, 'driver', $2, $3, $4, true)
		RETURNING id;`
	var id string
	err = ex.QueryRow(ctx, insertQuery, driverID, latitude, longitude, accuracyMeters).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to create coordinate: %w", err)
	}
	return id, nil
}

// ArchiveOldCoordinates moves old coordinates to location_history
func (repo *LocationRepository) ArchiveOldCoordinates(ctx context.Context, before time.Time) error {
	ex := executor.GetExecutor(ctx, repo.pool)

	query := `INSERT INTO location_history (id, coordinate_id, driver_id, latitude, longitude, recorded_at)
	SELECT gen_random_uuid(), c.id, c.entity_id, c.latitude, c.longitude, c.created_at
	FROM coordinates c
	WHERE c.entity_type = 'driver' 
		AND c.is_current = false
		AND c.updated_at < $1
		AND c.entity_id IN (SELECT id FROM drivers)
	RETURNING id;`

	result, err := ex.Exec(ctx, query, before)
	if err != nil {
		return fmt.Errorf("failed to archive old coordinates: %w", err)
	}

	_ = result.RowsAffected()

	// Delete the archived coordinates
	deleteQuery := `DELETE FROM coordinates 
		WHERE entity_type = 'driver' 
			AND is_current = false
			AND updated_at < $1;`
	_, err = ex.Exec(ctx, deleteQuery, before)
	if err != nil {
		return fmt.Errorf("failed to delete archived coordinates: %w", err)
	}

	return nil
}
