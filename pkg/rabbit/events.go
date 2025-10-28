package rabbit

const (
	RideRequestEvent    = "ride.request.{ride_type}"
	RideStatusEvent     = "ride.status.{status}"
	DriverResponseEvent = "driver.response.{ride_id}"
	DriverStatusEvent   = "driver.status.{driver_id}"
)
