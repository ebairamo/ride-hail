package ride

import (
	"context"
	"log/slog"
	"ride-hail/internal/adapters/http/handle"
	"ride-hail/internal/adapters/http/server"
	"ride-hail/internal/adapters/postgres"
	"ride-hail/internal/adapters/rabbit"
	"ride-hail/internal/core/service"
	"ride-hail/pkg/logger"
	rb "ride-hail/pkg/rabbit"
	"ride-hail/pkg/txm"
	"ride-hail/pkg/wsm"

	"ride-hail/config"
	pg "ride-hail/pkg/potgres"
)

type RideService struct {
	server server.Server
}

func New(ctx context.Context, cfg config.Config) (*RideService, error) {
	log := logger.NewLogger(
		cfg.Mode, logger.LoggerOptions{
			Pretty: true,
			Level:  slog.LevelDebug,
		},
	)
	pg, err := pg.New(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}

	uRepo := postgres.NewRepo(pg.Pool)
	cRepo := postgres.NewCordRepository(pg.Pool)
	rRepo := postgres.NewRideRepository(pg.Pool)

	rb, err := rb.New(cfg.RabbitMQ)
	if err != nil {
		return nil, err
	}

	if err = rabbit.InitRabbitTopology(rb); err != nil {
		return nil, err
	}

	rPub := rabbit.NewRidePublisher(rb.Conn)

	tmx := txm.NewTXManager(pg.Pool)

	wsM := wsm.NewWSManager()

	authServ := service.NewAuthService(cfg, uRepo, log)
	rideServ := service.NewRideService(log, tmx, rRepo, cRepo, rPub)

	authHandle := handle.New(cfg, authServ, log)
	rideHandle := handle.NewRideHandle(rideServ, log)

	serv, err := server.New(cfg, log, authHandle, rideHandle)
	if err != nil {
		return nil, err
	}

	return &RideService{
		server: serv,
	}, nil
}

func (r *RideService) Run() {
	r.server.Run()
}

func (r *RideService) Stop(ctx context.Context) error {
	return r.server.Stop(ctx)
}
