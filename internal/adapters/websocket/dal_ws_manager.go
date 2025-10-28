package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"ride-hail/internal/adapters/rabbit"
	"ride-hail/pkg/logger"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, validate origin properly
	},
}

type DriverWebSocketManager struct {
	connections       map[string]*DriverConnection
	offers            map[string][]rabbit.DriverOffer // ride_id -> offers
	mu                sync.RWMutex
	log               *logger.Logger
	publisher         *rabbit.Publisher
	lastLocationTime  map[string]time.Time // driver_id -> last update time
	locationRateLimit time.Duration
}

type DriverConnection struct {
	conn          *websocket.Conn
	driverID      string
	authenticated bool
	authTimeout   time.Time
	send          chan []byte
	lastPing      time.Time
}

type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	RideID    string      `json:"ride_id,omitempty"`
	OfferID   string      `json:"offer_id,omitempty"`
	Accepted  bool        `json:"accepted,omitempty"`
	Latitude  float64     `json:"latitude,omitempty"`
	Longitude float64     `json:"longitude,omitempty"`
	Token     string      `json:"token,omitempty"`
}

type RideOfferMessage struct {
	Type           string `json:"type"`
	OfferID        string `json:"offer_id"`
	RideID         string `json:"ride_id"`
	RideNumber     string `json:"ride_number"`
	PickupLocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Address   string  `json:"address"`
	} `json:"pickup_location"`
	DestinationLocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Address   string  `json:"address"`
	} `json:"destination_location"`
	EstimatedFare      float64 `json:"estimated_fare"`
	DriverEarnings     float64 `json:"driver_earnings"`
	DistanceToPickupKm float64 `json:"distance_to_pickup_km"`
	EstimatedDuration  int     `json:"estimated_duration_minutes"`
	ExpiresAt          string  `json:"expires_at"`
}

func NewDriverWebSocketManager(log *logger.Logger, publisher *rabbit.Publisher) *DriverWebSocketManager {
	return &DriverWebSocketManager{
		connections:       make(map[string]*DriverConnection),
		offers:            make(map[string][]rabbit.DriverOffer),
		log:               log,
		publisher:         publisher,
		lastLocationTime:  make(map[string]time.Time),
		locationRateLimit: 3 * time.Second, // Max 1 update per 3 seconds
	}
}

func (m *DriverWebSocketManager) HandleDriverConnection(w http.ResponseWriter, r *http.Request, driverID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.log.Func("HandleDriverConnection").Error(r.Context(), "websocket_upgrade_error", "failed to upgrade connection", "error", err)
		return
	}

	driverConn := &DriverConnection{
		conn:          conn,
		driverID:      driverID,
		authenticated: false,
		authTimeout:   time.Now().Add(5 * time.Second),
		send:          make(chan []byte, 256),
		lastPing:      time.Now(),
	}

	// Register connection
	m.mu.Lock()
	m.connections[driverID] = driverConn
	m.mu.Unlock()

	// Cleanup on disconnect
	defer func() {
		m.mu.Lock()
		delete(m.connections, driverID)
		m.mu.Unlock()
		conn.Close()
	}()

	// Start goroutines
	go m.writePump(driverConn)
	m.readPump(r.Context(), driverConn)
}

func (m *DriverWebSocketManager) readPump(ctx context.Context, conn *DriverConnection) {
	defer conn.conn.Close()

	conn.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.conn.SetPongHandler(func(string) error {
		conn.lastPing = time.Now()
		conn.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg WSMessage
		err := conn.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				m.log.Func("readPump").Error(ctx, "websocket_read_error", "unexpected close", "error", err)
			}
			break
		}

		if !conn.authenticated && msg.Type != "auth" {
			m.log.Func("readPump").Warn(ctx, "unauthenticated_request", "ignoring unauthenticated message")
			break
		}

		if time.Now().After(conn.authTimeout) && !conn.authenticated {
			m.log.Func("readPump").Warn(ctx, "auth_timeout", "authentication timeout")
			break
		}

		m.handleMessage(ctx, conn, msg)
	}
}

func (m *DriverWebSocketManager) handleMessage(ctx context.Context, conn *DriverConnection, msg WSMessage) {
	log := m.log.Func("handleMessage")

	switch msg.Type {
	case "auth":
		m.handleAuth(ctx, conn, msg)
	case "ride_response":
		m.handleRideResponse(ctx, conn, msg)
	case "location_update":
		m.handleLocationUpdate(ctx, conn, msg)
	default:
		log.Warn(ctx, "unknown_message_type", "unknown message type", "type", msg.Type)
	}
}

func (m *DriverWebSocketManager) handleAuth(ctx context.Context, conn *DriverConnection, msg WSMessage) {
	// In production, validate JWT token here
	conn.authenticated = true
	conn.authTimeout = time.Time{} // Clear timeout

	m.log.Func("handleAuth").Info(ctx, "driver_authenticated", "driver authenticated", "driver_id", conn.driverID)

	// Send confirmation
	response := WSMessage{Type: "auth_success"}
	conn.send <- m.marshalMessage(response)
}

func (m *DriverWebSocketManager) handleRideResponse(ctx context.Context, conn *DriverConnection, msg WSMessage) {
	log := m.log.Func("handleRideResponse")

	log.Info(ctx, "ride_response_received", "driver responded to ride offer",
		"driver_id", conn.driverID, "ride_id", msg.RideID, "accepted", msg.Accepted)

	// Publish driver response to RabbitMQ
	responseMsg := map[string]interface{}{
		"ride_id":   msg.RideID,
		"driver_id": conn.driverID,
		"accepted":  msg.Accepted,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	routingKey := fmt.Sprintf("driver.response.%s", msg.RideID)
	message, _ := json.Marshal(responseMsg)
	if m.publisher != nil {
		m.publisher.Publish("driver_topic", routingKey, message)
	}

	// Update driver status if accepted
	if msg.Accepted {
		// Driver accepts ride - update status to BUSY
		log.Info(ctx, "driver_accepted_ride", "driver accepted ride", "driver_id", conn.driverID, "ride_id", msg.RideID)

		// Send ride details
		// In production, fetch ride details from database
		response := WSMessage{
			Type: "ride_details",
			Data: map[string]interface{}{
				"ride_id": msg.RideID,
				"message": "Ride accepted, proceed to pickup location",
			},
		}
		conn.send <- m.marshalMessage(response)
	}

	// Remove offer from tracking
	m.mu.Lock()
	delete(m.offers, msg.RideID)
	m.mu.Unlock()
}

func (m *DriverWebSocketManager) handleLocationUpdate(ctx context.Context, conn *DriverConnection, msg WSMessage) {
	log := m.log.Func("handleLocationUpdate")

	// Check rate limiting
	if !m.checkRateLimit(conn.driverID) {
		log.Warn(ctx, "rate_limit_exceeded", "location update rate limit exceeded", "driver_id", conn.driverID)
		return
	}

	// Update last location time
	m.mu.Lock()
	m.lastLocationTime[conn.driverID] = time.Now()
	m.mu.Unlock()

	// Publish location update to RabbitMQ fanout exchange
	locationMsg := map[string]interface{}{
		"driver_id": conn.driverID,
		"location": map[string]float64{
			"lat": msg.Latitude,
			"lng": msg.Longitude,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	message, _ := json.Marshal(locationMsg)
	if m.publisher != nil {
		m.publisher.Publish("location_fanout", "", message)
	}

	log.Debug(ctx, "location_update_published", "location update published", "driver_id", conn.driverID)
}

func (m *DriverWebSocketManager) checkRateLimit(driverID string) bool {
	m.mu.RLock()
	lastUpdate, exists := m.lastLocationTime[driverID]
	m.mu.RUnlock()

	if !exists {
		return true
	}

	return time.Since(lastUpdate) >= m.locationRateLimit
}

func (m *DriverWebSocketManager) writePump(conn *DriverConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-conn.send:
			conn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Send queued messages
			n := len(conn.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-conn.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			// Send ping
			conn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (m *DriverWebSocketManager) IsDriverConnected(driverID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[driverID] != nil && m.connections[driverID].authenticated
}

func (m *DriverWebSocketManager) SendRideOffer(ctx context.Context, driverID string, offer rabbit.DriverOffer) error {
	m.mu.RLock()
	conn, exists := m.connections[driverID]
	m.mu.RUnlock()

	if !exists || !conn.authenticated {
		return fmt.Errorf("driver %s not connected", driverID)
	}

	msg := RideOfferMessage{
		Type:           "ride_offer",
		OfferID:        offer.OfferID,
		RideID:         offer.RideID,
		RideNumber:     offer.RideNumber,
		EstimatedFare:  offer.EstimatedFare,
		DriverEarnings: offer.DriverEarnings,
		ExpiresAt:      offer.ExpiresAt.Format(time.RFC3339),
	}

	msg.PickupLocation.Latitude = offer.PickupLocation.Lat
	msg.PickupLocation.Longitude = offer.PickupLocation.Lng
	msg.DestinationLocation.Latitude = offer.DestinationLocation.Lat
	msg.DestinationLocation.Longitude = offer.DestinationLocation.Lng

	messageBytes, _ := json.Marshal(msg)
	conn.send <- messageBytes

	return nil
}

func (m *DriverWebSocketManager) TrackOffers(rideID string, offers []rabbit.DriverOffer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.offers[rideID] = offers
}

func (m *DriverWebSocketManager) HandleOfferTimeout(rideID string) {
	m.mu.Lock()
	offers, exists := m.offers[rideID]
	m.mu.Unlock()

	if !exists {
		return
	}

	m.log.Func("HandleOfferTimeout").Info(context.Background(), "offer_timeout", "handling offer timeout", "ride_id", rideID)

	// Notify drivers that offer expired
	for _, offer := range offers {
		m.mu.RLock()
		conn := m.connections[offer.DriverID]
		m.mu.RUnlock()

		if conn != nil && conn.authenticated {
			response := WSMessage{
				Type: "offer_expired",
				Data: map[string]interface{}{
					"offer_id": offer.OfferID,
					"ride_id":  rideID,
				},
			}
			conn.send <- m.marshalMessage(response)
		}
	}

	// Remove from tracking
	m.mu.Lock()
	delete(m.offers, rideID)
	m.mu.Unlock()
}

func (m *DriverWebSocketManager) marshalMessage(msg interface{}) []byte {
	data, _ := json.Marshal(msg)
	return data
}
