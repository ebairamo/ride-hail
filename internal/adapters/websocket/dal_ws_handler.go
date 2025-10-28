package websocket

import (
	"net/http"
	"strings"

	"ride-hail/pkg/logger"
)

// DriverWebSocketHandler handles WebSocket connections for drivers
type DriverWebSocketHandler struct {
	manager *DriverWebSocketManager
	log     *logger.Logger
}

func NewDriverWebSocketHandler(manager *DriverWebSocketManager, log *logger.Logger) *DriverWebSocketHandler {
	return &DriverWebSocketHandler{
		manager: manager,
		log:     log,
	}
}

func (h *DriverWebSocketHandler) HandleDriverWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract driver_id from path /ws/drivers/{driver_id}
	driverID := extractDriverIDFromPath(r.URL.Path)
	if driverID == "" {
		h.log.Func("HandleDriverWebSocket").Error(r.Context(), "invalid_driver_id", "invalid driver ID in path")
		http.Error(w, "invalid driver_id", http.StatusBadRequest)
		return
	}

	h.log.Func("HandleDriverWebSocket").Info(r.Context(), "ws_connection_attempt", "driver attempting WebSocket connection", "driver_id", driverID)

	// Handle WebSocket connection
	h.manager.HandleDriverConnection(w, r, driverID)
}

func extractDriverIDFromPath(path string) string {
	// Path format: /ws/drivers/{driver_id}
	parts := strings.Split(path, "/")
	if len(parts) >= 4 && parts[1] == "ws" && parts[2] == "drivers" {
		return parts[3]
	}
	return ""
}
