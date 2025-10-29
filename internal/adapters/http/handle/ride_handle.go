package handle

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"ride-hail/internal/adapters/http/handle/dto"
	"ride-hail/internal/core/domain/action"
	"ride-hail/internal/core/domain/models"
	"ride-hail/internal/core/domain/types"
	"ride-hail/internal/core/ports"
	"ride-hail/pkg/logger"
	"ride-hail/pkg/wsm"
	"strings"
	"time"
)

type RideHandle struct {
	svc ports.RideService
	wsm wsm.HandlerWS
	log *logger.Logger
}

func NewRideHandle(svc ports.RideService, wsm wsm.HandlerWS, log *logger.Logger) *RideHandle {
	return &RideHandle{
		svc: svc,
		wsm: wsm,
		log: log,
	}
}

type RideHandler interface {
	CreateNewRide(w http.ResponseWriter, r *http.Request)
	CancelRide(w http.ResponseWriter, r *http.Request)
}

func (h *RideHandle) CreateNewRide(w http.ResponseWriter, r *http.Request) {
	log := h.log.Func("RideHandle.CreateNewRide")
	ctx := r.Context()

	log.Debug(ctx, action.CreateRide, "request to create a Ride has been launched")

	if logger.GetRole(ctx) != types.RoleCustomer {
		log.Error(ctx, action.CreateRide, "invalid role", "role", logger.GetRole(ctx))
		http.Error(w, msgForbidden, http.StatusForbidden)
		return
	}
	var rideDto models.CreateRideRequest

	if err := json.NewDecoder(r.Body).Decode(&rideDto); err != nil {
		log.Error(ctx, "decode error", "msg", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if ok, err := dto.ValidateRideDTO(rideDto); !ok {
		log.Warn(ctx, action.CreateRide, "invalid request")
		http.Error(w, err, http.StatusBadRequest)
		return
	}

	if resp, err := h.svc.CreateNewRide(ctx, rideDto); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	} else {
		log.Debug(ctx, action.CreateRide, "the request to create a trip was successfully completed")
		writeJSON(w, http.StatusCreated, resp)
		return
	}
}

func (h *RideHandle) CancelRide(w http.ResponseWriter, r *http.Request) {
	log := h.log.Func("RideHandle.CancelRide")
	ctx := r.Context()

	log.Debug(ctx, action.CloseRide, "a request to close the ride has been launched")
	if logger.GetRole(ctx) != types.RoleCustomer {
		log.Error(ctx, action.CloseRide, "invalid role")
		http.Error(w, msgForbidden, http.StatusForbidden)
		return
	}

	closeReq := models.CloseRideRequest{
		RideID: getRideID(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&closeReq); err != nil {
		log.Error(ctx, action.CloseRide, "error decoding body", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if resp, err := h.svc.CloseRide(ctx, closeReq); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	} else {
		log.Debug(ctx, action.CloseRide, "the request to cancel the ride has been completed")
		writeJSON(w, http.StatusOK, resp)
		return
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *RideHandle) WSPassenger(w http.ResponseWriter, r *http.Request) {
	log := h.log.Func("RideHandle.WSPassenger")
	ctx := r.Context()

	log.Debug(ctx, action.WSPassenger, "connection request received from passenger")
	if logger.GetRole(ctx) != types.RoleCustomer {
		log.Error(ctx, action.WSPassenger, "invalid role")
		http.Error(w, msgForbidden, http.StatusForbidden)
		return
	}

	var passengerID string
	if _, err := fmt.Sscanf(r.URL.Path, "/ws/passengers/%s", &passengerID); err != nil {
		log.Error(ctx, action.WSPassenger, "invalid passenger ID", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if passengerID != logger.GetUserID(ctx) {
		log.Error(ctx, action.WSPassenger, "invalid passenger ID")
		http.Error(w, "invalid passenger ID", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(ctx, action.WSPassenger, "error upgrading connection", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.wsm.AddConn(passengerID, conn)
	defer conn.Close()

	log.Debug(ctx, action.WSPassenger, "connection established")
	if err = conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Error(ctx, action.WSPassenger, "error while setting read deadline", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	conn.SetPongHandler(func(appData string) error {
		log.Debug(ctx, action.WSPassenger, "deadline update")
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	var auth dto.Auth
	messageType, data, err := conn.ReadMessage()
	if err != nil {
		log.Error(ctx, action.WSPassenger, "error reading message", "error", err)
		return
	}

	if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
		log.Error(ctx, action.WSPassenger, "invalid message type")
		return
	}

	if err = json.Unmarshal(data, &auth); err != nil {
		log.Error(ctx, action.WSPassenger, "error decoding auth message", "error", err)
		h.wsm.Send(passengerID, []byte("incorrect auth message"))
		return
	}

	if err = auth.Validate(); err != nil {
		log.Error(ctx, action.WSPassenger, "invalid auth", "error", err)
		if err := h.wsm.Send(passengerID, []byte("invalid auth token")); err != nil {
			log.Error(ctx, action.WSPassenger, "error sending auth", "error", err)
			return
		}
		return
	}

	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Debug(ctx, action.WSPassenger, "pink")
				if err = conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Error(ctx, action.WSPassenger, "error while writing ping message", "error", err)
					conn.Close()
					return
				}
			case <-done:
				log.Debug(ctx, action.WSPassenger, "stopped goroutine in ticker ping-pong")
				return
			}
		}
	}()
}

func getRideID(r *http.Request) string {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	return parts[2]
}
