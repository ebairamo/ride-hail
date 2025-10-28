package rabbit

import (
	"context"
	"fmt"
	"sync"

	"ride-hail/pkg/rabbit"
)

type ConsumerManager struct {
	consumers []*rabbit.Consumer
	wg        sync.WaitGroup
}

func NewConsumerManager() *ConsumerManager {
	return &ConsumerManager{
		consumers: make([]*rabbit.Consumer, 0),
	}
}

// StartDALConsumers starts all required consumers for DAL service
func (cm *ConsumerManager) StartDALConsumers(ctx context.Context, conn *rabbit.Rabbit, dalConsumer *DALConsumer) error {
	// Consumer for ride requests (driver matching)
	rideRequestConsumer := rabbit.NewConsumer(conn.Conn, "ride_topic", "driver_matching")
	rideRequestConsumer.SetHandler(rabbit.MessageHandlerFunc(dalConsumer.HandleRideRequest))

	// Consumer for ride status updates
	rideStatusConsumer := rabbit.NewConsumer(conn.Conn, "ride_topic", "ride_status")
	rideStatusConsumer.SetHandler(rabbit.MessageHandlerFunc(dalConsumer.HandleRideStatusUpdate))

	// Start consumers
	consumers := []struct {
		consumer *rabbit.Consumer
		name     string
	}{
		{rideRequestConsumer, "ride_request_consumer"},
		{rideStatusConsumer, "ride_status_consumer"},
	}

	for _, c := range consumers {
		cm.wg.Add(1)
		go func(consumer *rabbit.Consumer, name string) {
			defer cm.wg.Done()

			fmt.Printf("Starting %s...\n", name)
			if err := consumer.StartConsuming(ctx); err != nil {
				fmt.Printf("Error starting %s: %v\n", name, err)
			}
		}(c.consumer, c.name)

		cm.consumers = append(cm.consumers, c.consumer)
	}

	return nil
}

// StopAll stops all consumers gracefully
func (cm *ConsumerManager) StopAll() {
	cm.wg.Wait()
	fmt.Println("All consumers stopped")
}
