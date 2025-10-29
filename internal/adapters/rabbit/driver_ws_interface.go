package rabbit

import (
	"context"
)

// DriverWebSocketManager interface to avoid circular dependencies
type DriverWebSocketManager interface {
	IsDriverConnected(driverID string) bool
	SendRideOffer(ctx context.Context, driverID string, offer DriverOffer) error
	TrackOffers(rideID string, offers []DriverOffer)
	HandleOfferTimeout(rideID string)
}
