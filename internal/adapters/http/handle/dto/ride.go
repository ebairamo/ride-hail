package dto

import (
	"fmt"
	"regexp"
	"strings"

	"ride-hail/internal/core/domain/models"
)

type RideRules struct {
	AllowRideTypes []string
}

var DefaultRideRules = RideRules{
	AllowRideTypes: []string{"ECONOMY", "PREMIUM", "XL"},
}

func isValidUUID(u string) bool {
	re := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	return re.MatchString(u)
}

func ValidateRideDTO(dto models.CreateRideRequest) (bool, string) {
	var reasons []string

	// Проверка PassengerID
	if dto.PassengerID == "" || !isValidUUID(dto.PassengerID) {
		reasons = append(reasons, "invalid_passenger_id")
	}

	// Проверка координат
	if dto.PickupLatitude < -90 || dto.PickupLatitude > 90 {
		reasons = append(reasons, "invalid_pickup_latitude")
	}
	if dto.PickupLongitude < -180 || dto.PickupLongitude > 180 {
		reasons = append(reasons, "invalid_pickup_longitude")
	}
	if dto.DestinationLatitude < -90 || dto.DestinationLatitude > 90 {
		reasons = append(reasons, "invalid_destination_latitude")
	}
	if dto.DestinationLongitude < -180 || dto.DestinationLongitude > 180 {
		reasons = append(reasons, "invalid_destination_longitude")
	}

	if strings.TrimSpace(dto.PickupAddress) == "" {
		reasons = append(reasons, "empty_pickup_address")
	}
	if strings.TrimSpace(dto.DestinationAddress) == "" {
		reasons = append(reasons, "empty_destination_address")
	}

	validType := false
	for _, allowed := range DefaultRideRules.AllowRideTypes {
		if strings.EqualFold(dto.RideType, allowed) {
			validType = true
			break
		}
	}
	if !validType {
		reasons = append(reasons, fmt.Sprintf("invalid_ride_type: %s", dto.RideType))
	}

	return len(reasons) == 0, strings.Join(reasons, ", ")
}
