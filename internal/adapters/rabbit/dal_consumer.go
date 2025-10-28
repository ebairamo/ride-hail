package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ride-hail/internal/core/domain/models"
	"ride-hail/internal/core/domain/types"
	"ride-hail/internal/core/ports"
	"ride-hail/pkg/logger"
)

// DriverMatchingConsumer handles ride request messages for driver matching
type DriverMatchingConsumer struct {
	dalService   ports.DalService
	publisher    *Publisher
	log          *logger.Logger
	wsManager    DriverWebSocketManager
	offerTimeout time.Duration
}

type RideRequestMessage struct {
	RideID         string `json:"ride_id"`
	RideNumber     string `json:"ride_number"`
	PickupLocation struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Address string  `json:"address"`
	} `json:"pickup_location"`
	DestinationLocation struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Address string  `json:"address"`
	} `json:"destination_location"`
	RideType       string  `json:"ride_type"`
	EstimatedFare  float64 `json:"estimated_fare"`
	MaxDistanceKm  float64 `json:"max_distance_km"`
	TimeoutSeconds int     `json:"timeout_seconds"`
	CorrelationID  string  `json:"correlation_id"`
}

func NewDriverMatchingConsumer(dalService ports.DalService, publisher *Publisher, log *logger.Logger, wsManager DriverWebSocketManager) *DriverMatchingConsumer {
	return &DriverMatchingConsumer{
		dalService:   dalService,
		publisher:    publisher,
		log:          log,
		wsManager:    wsManager,
		offerTimeout: 30 * time.Second,
	}
}

func (c *DriverMatchingConsumer) HandleRideRequest(ctx context.Context, message []byte, routingKey string) error {
	log := c.log.Func("DriverMatchingConsumer.HandleRideRequest")

	var rideReq RideRequestMessage
	if err := json.Unmarshal(message, &rideReq); err != nil {
		log.Error(ctx, "handle_ride_request", "failed to unmarshal message", "error", err)
		return fmt.Errorf("failed to unmarshal ride request: %w", err)
	}

	log.Info(ctx, "handle_ride_request", "processing ride request", "ride_id", rideReq.RideID, "ride_type", rideReq.RideType)

	// Find nearby available drivers within 5km radius
	radiusMeters := int(rideReq.MaxDistanceKm * 1000)
	drivers, err := c.dalService.FindNearbyDrivers(ctx, rideReq.PickupLocation.Lat, rideReq.PickupLocation.Lng,
		rideReq.RideType, radiusMeters, 10)
	if err != nil {
		log.Error(ctx, "handle_ride_request", "failed to find nearby drivers", "error", err)
		return fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	if len(drivers) == 0 {
		log.Warn(ctx, "handle_ride_request", "no drivers found", "ride_id", rideReq.RideID)
		return nil // No drivers available, but not an error
	}

	log.Info(ctx, "handle_ride_request", fmt.Sprintf("found %d nearby drivers", len(drivers)), "count", len(drivers))

	// Create ride offer for each driver
	offerID := fmt.Sprintf("offer_%s_%d", rideReq.RideID, time.Now().Unix())
	offers := make([]DriverOffer, 0, len(drivers))

	for _, driver := range drivers {
		offer := DriverOffer{
			OfferID:             offerID,
			RideID:              rideReq.RideID,
			RideNumber:          rideReq.RideNumber,
			DriverID:            driver.ID,
			PickupLocation:      rideReq.PickupLocation,
			DestinationLocation: rideReq.DestinationLocation,
			RideType:            rideReq.RideType,
			EstimatedFare:       rideReq.EstimatedFare,
			DriverEarnings:      rideReq.EstimatedFare * 0.8, // 80% to driver
			ExpiresAt:           time.Now().Add(c.offerTimeout),
			CorrelationID:       rideReq.CorrelationID,
		}

		// Try to send offer via WebSocket
		if c.wsManager != nil && c.wsManager.IsDriverConnected(driver.ID) {
			err := c.wsManager.SendRideOffer(ctx, driver.ID, offer)
			if err != nil {
				log.Warn(ctx, "handle_ride_request", "failed to send offer via WebSocket", "error", err, "driver_id", driver.ID)
				continue
			}
			offers = append(offers, offer)
			log.Info(ctx, "handle_ride_request", "ride offer sent", "driver_id", driver.ID, "ride_id", rideReq.RideID)
		}
	}

	// Track offers for timeout handling
	if len(offers) > 0 {
		c.wsManager.TrackOffers(rideReq.RideID, offers)

		// Set up timeout handler
		go func() {
			time.Sleep(c.offerTimeout)
			c.wsManager.HandleOfferTimeout(rideReq.RideID)
		}()
	}

	log.Info(ctx, "handle_ride_request", fmt.Sprintf("sent %d ride offers", len(offers)), "count", len(offers))
	return nil
}

func (c *DriverMatchingConsumer) HandleRideStatusUpdate(ctx context.Context, message []byte, routingKey string) error {
	log := c.log.Func("DriverMatchingConsumer.HandleRideStatusUpdate")

	var statusUpdate models.Ride
	if err := json.Unmarshal(message, &statusUpdate); err != nil {
		log.Error(ctx, "handle_ride_status_update", "failed to unmarshal message", "error", err)
		return fmt.Errorf("failed to unmarshal ride status: %w", err)
	}

	log.Info(ctx, "handle_ride_status_update", "ride status update received", "ride_id", statusUpdate.ID, "status", statusUpdate.Status)

	switch statusUpdate.Status {
	case types.RideStatusCOMPLETED:
		// Update driver status to AVAILABLE after completion
		if statusUpdate.DriverID != "" {
			err := c.dalService.ChangeDriverStatus(ctx, statusUpdate.DriverID, types.DriverStatusAvailable, types.DriverStatusBusy)
			if err != nil {
				log.Warn(ctx, "handle_ride_status_update", "failed to change driver status to AVAILABLE", "error", err, "driver_id", statusUpdate.DriverID)
			}
		}

	case types.RideStatusCANCELLED:
		// Make driver available again if ride was cancelled
		if statusUpdate.DriverID != "" {
			err := c.dalService.ChangeDriverStatus(ctx, statusUpdate.DriverID, types.DriverStatusAvailable, types.DriverStatusBusy)
			if err != nil {
				log.Warn(ctx, "handle_ride_status_update", "failed to change driver status to AVAILABLE", "error", err, "driver_id", statusUpdate.DriverID)
			}
		}
	}

	return nil
}

type DriverOffer struct {
	OfferID        string
	RideID         string
	RideNumber     string
	DriverID       string
	PickupLocation struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Address string  `json:"address"`
	}
	DestinationLocation struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Address string  `json:"address"`
	}
	RideType       string
	EstimatedFare  float64
	DriverEarnings float64
	ExpiresAt      time.Time
	CorrelationID  string
}
