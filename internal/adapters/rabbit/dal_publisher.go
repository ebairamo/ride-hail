package rabbit

import (
	"encoding/json"
	"fmt"
	"time"
)

// PublishDriverLocation publishes driver location updates to location_fanout exchange
func (dp *Publisher) PublishDriverLocation(locationMsg interface{}) error {
	message, err := json.Marshal(locationMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal location message: %w", err)
	}

	// Fanout exchange doesn't use routing key
	return dp.Publish("location_fanout", "", message)
}

// PublishDriverStatus publishes driver status updates to driver_topic exchange
func (dp *Publisher) PublishDriverStatus(driverID string, status string, rideID string) error {
	statusMsg := map[string]interface{}{
		"driver_id": driverID,
		"status":    status,
		"ride_id":   rideID,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	message, err := json.Marshal(statusMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal driver status: %w", err)
	}

	routingKey := fmt.Sprintf("driver.status.%s", driverID)
	return dp.Publish("driver_topic", routingKey, message)
}
