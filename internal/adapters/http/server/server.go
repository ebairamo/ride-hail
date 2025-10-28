package server

import (
	"context"
	"net/http"
	"strconv"

	"ride-hail/config"
	"ride-hail/internal/adapters/http/handle"
	"ride-hail/internal/adapters/websocket"
	"ride-hail/internal/core/domain/action"
	"ride-hail/internal/core/domain/types"
	"ride-hail/pkg/logger"
)

type API struct {
	h    *handlers
	serv *http.Server
	cfg  config.Config
	log  *logger.Logger
	addr int
}

type handlers struct {
	auth handle.AuthHandle
	ride handle.RideHandler
	dal  handle.DalHandler
	ws   websocket.DriverWebSocketHandler
}

type Server interface {
	Run()
	Stop(ctx context.Context) error
}

func New(cfg config.Config, log *logger.Logger, auth handle.AuthHandle, ride handle.RideHandler, dal handle.DalHandler, ws *websocket.DriverWebSocketHandler) (*API, error) {
	h := &handlers{
		auth: auth,
		ride: ride,
		dal:  dal,
		ws:   *ws,
	}

	api := &API{
		h:   h,
		cfg: cfg,
		log: log,
	}
	mux := http.NewServeMux()
	if err := api.setupRoutes(mux); err != nil {
		return nil, err
	}

	api.initAddr()
	api.serv = &http.Server{
		Addr:    ":" + strconv.Itoa(api.addr),
		Handler: api.middleware(mux),
	}

	return api, nil
}

func (a *API) Run() {
	log := a.log.Func("api.Run")
	log.Info(context.Background(), action.StartApplication, "server starting", "addr", a.serv.Addr)

	if err := a.serv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error(context.Background(), action.StartApplication, "error in run server", "error", err)
		return
	}
	log.Info(context.Background(), action.StartApplication, "server started")
}

func (a *API) initAddr() {
	switch a.cfg.Mode {
	case types.ModeAdmin:
		a.addr = a.cfg.Services.AdminService
	case types.ModeDAL:
		a.addr = a.cfg.Services.DriverLocationService
	case types.ModeRide:
		a.addr = a.cfg.Services.RideService
	}
}

func (a *API) Stop(ctx context.Context) error {
	log := a.log.Func("api.Stop")
	log.Info(ctx, action.StopApplication, "shutting down server")
	return a.serv.Shutdown(ctx)
}
