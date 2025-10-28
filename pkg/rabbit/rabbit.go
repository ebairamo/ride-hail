package rabbit

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Rabbit struct {
	Conn *amqp.Connection
	Cfg  Config
}

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
}

func (c Config) GetRabbitDsn() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%d/",
		c.User,
		c.Password,
		c.Host,
		c.Port,
	)
}

func New(cfg Config) (*Rabbit, error) {
	conn, err := amqp.Dial(cfg.GetRabbitDsn())
	if err != nil {
		return nil, err
	}
	return &Rabbit{Conn: conn, Cfg: cfg}, nil
}

func (r *Rabbit) Close() {
	if r.Conn != nil {
		_ = r.Conn.Close()
	}
}

type QueueConfig struct {
	Name       string
	RoutingKey string
}

func (r *Rabbit) SetupExchangesAndQueues(exchangeName, exchangeType string, queues []QueueConfig) error {
	ch, err := r.Conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err = r.ensureExchange(ch, exchangeName, exchangeType); err != nil {
		return err
	}

	for _, qCfg := range queues {
		q, err := r.ensureQueue(ch, qCfg.Name)
		if err != nil {
			return err
		}

		if err = ch.QueueBind(
			q.Name,
			qCfg.RoutingKey,
			exchangeName,
			false,
			nil,
		); err != nil {
			return err
		}
	}

	return nil
}

func (r *Rabbit) ensureExchange(ch *amqp.Channel, name, kind string) error {
	return ch.ExchangeDeclare(name, kind, true, false, false, false, nil)
}

func (r *Rabbit) ensureQueue(ch *amqp.Channel, name string) (amqp.Queue, error) {
	return ch.QueueDeclare(name, true, false, false, false, nil)
}
