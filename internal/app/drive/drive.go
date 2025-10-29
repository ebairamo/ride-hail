package dal

import (
	"context"
	"log/slog"

	"ride-hail/config"
	"ride-hail/internal/adapters/http/handle"
	"ride-hail/internal/adapters/http/server"
	"ride-hail/internal/adapters/postgres"
	rabbitadapter "ride-hail/internal/adapters/rabbit"
	"ride-hail/internal/adapters/websocket"
	"ride-hail/internal/core/service"
	"ride-hail/pkg/logger"
	pg "ride-hail/pkg/potgres"
	rb "ride-hail/pkg/rabbit"
	"ride-hail/pkg/txm"
)

type DriverService struct {
	server    server.Server
	consumer  *rabbitadapter.DriverMatchingConsumer
	wsManager *websocket.DriverWebSocketManager
}

func New(ctx context.Context, cfg config.Config) (*DriverService, error) {
	log := logger.NewLogger(
		cfg.Mode, logger.LoggerOptions{
			Pretty: true,
			Level:  slog.LevelDebug,
		},
	)

	pgConn, err := pg.New(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}

	driverRepo := postgres.NewDriverRepository(pgConn.Pool)
	locationRepo := postgres.NewLocationRepository(pgConn.Pool)

	rbConn, err := rb.New(cfg.RabbitMQ)
	if err != nil {
		return nil, err
	}

	if err = rabbitadapter.InitRabbitTopology(rbConn); err != nil {
		return nil, err
	}

	txm := txm.NewTXManager(pgConn.Pool)

	dalService := service.NewDalService(log, txm, driverRepo, locationRepo)

	rabbitPub := rabbitadapter.NewPublisher(rbConn.Conn)

	wsManager := websocket.NewDriverWebSocketManager(log, rabbitPub)

	consumer := rabbitadapter.NewDriverMatchingConsumer(dalService, rabbitPub, log, wsManager)

	consumerChan := rb.NewConsumer(rbConn.Conn, "ride_topic", "driver_matching")
	consumerChan.SetHandler(rabbitHandler(consumer.HandleRideRequest))
	go func() {
		if err := consumerChan.StartConsuming(ctx); err != nil {
			// log.Error(ctx, "consumer_error", "failed to start consumer", "error", err)
			log.Slog.Error("consumer_error", "failed to start consumer", "error")
		}
	}()

	authHandle := handle.New(cfg, nil, log)
	dalHandle := handle.NewDalHandle(dalService, log)

	wsHandler := websocket.NewDriverWebSocketHandler(wsManager, log)

	serv, err := server.New(cfg, log, authHandle, nil, dalHandle, wsHandler)
	if err != nil {
		return nil, err
	}

	return &DriverService{
		server:    serv,
		consumer:  consumer,
		wsManager: wsManager,
	}, nil
}

func rabbitHandler(fn func(context.Context, []byte, string) error) rb.MessageHandler {
	return rb.MessageHandlerFunc(fn)
}

func (r *DriverService) Run() {
	r.server.Run()
}

func (r *DriverService) Stop(ctx context.Context) error {
	return r.server.Stop(ctx)
}
