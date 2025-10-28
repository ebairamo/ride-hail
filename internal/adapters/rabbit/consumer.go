package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ride-hail/internal/core/domain/models"
	"ride-hail/internal/core/ports"
)

// DALConsumer handles all message consumption for Driver & Location Service
type DALConsumer struct {
	dalService    ports.DalService
	ridePublisher *Publisher
}

func NewDALConsumer(dalService ports.DalService, publisher *Publisher) *DALConsumer {
	return &DALConsumer{
		dalService:    dalService,
		ridePublisher: publisher,
	}
}

// HandleRideRequest processes incoming ride requests for driver matching
func (dc *DALConsumer) HandleRideRequest(ctx context.Context, message []byte, routingKey string) error {
	var rideReq models.CreateRideRequest
	if err := json.Unmarshal(message, &rideReq); err != nil {
		return fmt.Errorf("failed to unmarshal ride request: %w", err)
	}

	// Find nearby available drivers
	availableDrivers, err := dc.dalService.ListAvailableDriversNear(ctx)
	if err != nil {
		return fmt.Errorf("failed to list available drivers: %w", err)
	}

	// Filter drivers by proximity and vehicle type
	nearbyDrivers := dc.filterNearbyDrivers(availableDrivers, rideReq.RideType)

	// Send ride offers to nearby drivers via WebSocket
	// This would be handled by your WebSocket service
	dc.sendRideOffersToDrivers(ctx, rideReq, nearbyDrivers)

	// Log the matching attempt
	fmt.Printf("Processed ride request %s, found %d nearby drivers\n",
		rideReq.PassengerID, len(nearbyDrivers))

	return nil
}

// HandleRideStatusUpdate processes ride status updates
func (dc *DALConsumer) HandleRideStatusUpdate(ctx context.Context, message []byte, routingKey string) error {
	var statusUpdate models.Ride
	if err := json.Unmarshal(message, &statusUpdate); err != nil {
		return fmt.Errorf("failed to unmarshal ride status: %w", err)
	}

	// Update driver status based on ride status
	switch statusUpdate.Status {
	case "COMPLETED":
		// Update driver earnings and make them available again
		// This would involve updating driver stats and status
		fmt.Printf("Ride %s completed, updating driver %s\n",
			statusUpdate.ID, statusUpdate.DriverID)
	case "CANCELLED":
		// Make driver available again if they were matched
		fmt.Printf("Ride %s cancelled, making driver %s available\n",
			statusUpdate.ID, statusUpdate.DriverID)
	}

	return nil
}

// HandleDriverResponse processes driver responses to ride offers (from WebSocket)
func (dc *DALConsumer) HandleDriverResponse(ctx context.Context, driverID string, rideID string, accepted bool) error {
	// Publish driver response to driver_topic exchange
	responseMsg := map[string]interface{}{
		"ride_id":   rideID,
		"driver_id": driverID,
		"accepted":  accepted,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	message, err := json.Marshal(responseMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal driver response: %w", err)
	}

	// Publish to driver_topic exchange with routing key driver.response.{ride_id}
	routingKey := fmt.Sprintf("driver.response.%s", rideID)
	err = dc.ridePublisher.Publish("driver_topic", routingKey, message)
	if err != nil {
		return fmt.Errorf("failed to publish driver response: %w", err)
	}

	// Update driver status if accepted
	if accepted {
		// This would update driver status to BUSY
		fmt.Printf("Driver %s accepted ride %s\n", driverID, rideID)
	}

	return nil
}

// Helper methods
func (dc *DALConsumer) filterNearbyDrivers(drivers []models.Driver, rideType string) []models.Driver {
	// Implement proximity filtering logic based on coordinates
	// This is a simplified version - in production you'd use PostGIS
	var nearbyDrivers []models.Driver

	for _, driver := range drivers {
		// Check if driver's vehicle type matches
		if driver.VehicleType != nil && *driver.VehicleType == rideType {
			// In real implementation, calculate distance using Haversine formula
			// and check if within max_distance_km
			nearbyDrivers = append(nearbyDrivers, driver)
		}
	}

	return nearbyDrivers
}

func (dc *DALConsumer) sendRideOffersToDrivers(ctx context.Context, rideReq models.CreateRideRequest, drivers []models.Driver) {
	// This would integrate with your WebSocket service to send offers to drivers
	// For now, just log the action
	for _, driver := range drivers {
		fmt.Printf("Sending ride offer for ride %s to driver %s\n",
			rideReq.PassengerID, driver.ID)
	}
}
