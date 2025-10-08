package nats

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type Client struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

type Config struct {
	URL           string
	MaxReconnects int
	ReconnectWait time.Duration
	ClusterID     string
	ClientID      string
}

func NewClient(config Config) (*Client, error) {
	opts := []nats.Option{
		nats.Name(config.ClientID),
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Printf("NATS disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("NATS connection closed")
		}),
	}

	nc, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context for durable subscriptions
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	log.Printf("Connected to NATS at %s", nc.ConnectedUrl())

	return &Client{
		conn: nc,
		js:   js,
	}, nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) Publish(subject string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = c.conn.Publish(subject, payload)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func (c *Client) PublishAsync(subject string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = c.js.PublishAsync(subject, payload)
	if err != nil {
		return fmt.Errorf("failed to publish async event: %w", err)
	}

	return nil
}

func (c *Client) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	log.Printf("Subscribed to subject: %s", subject)
	return sub, nil
}

func (c *Client) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.conn.QueueSubscribe(subject, queue, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to queue subscribe to %s: %w", subject, err)
	}

	log.Printf("Queue subscribed to subject: %s (queue: %s)", subject, queue)
	return sub, nil
}

// JetStream durable subscription for guaranteed delivery
func (c *Client) SubscribeDurable(subject, durableName, queueGroup string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.js.QueueSubscribe(
		subject,
		queueGroup,
		handler,
		nats.Durable(durableName),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.MaxDeliver(3),
		nats.AckWait(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create durable subscription to %s: %w", subject, err)
	}

	log.Printf("Durable subscription created: %s (durable: %s, queue: %s)", subject, durableName, queueGroup)
	return sub, nil
}

func (c *Client) CreateStream(streamName string, subjects []string) error {
	_, err := c.js.AddStream(&nats.StreamConfig{
		Name:      streamName,
		Subjects:  subjects,
		Storage:   nats.FileStorage,
		MaxAge:    24 * time.Hour * 7,
		Retention: nats.WorkQueuePolicy,
	})

	if err != nil {
		return fmt.Errorf("failed to create stream %s: %w", streamName, err)
	}

	log.Printf("Stream created: %s", streamName)
	return nil
}

func DecodeEvent(msg *nats.Msg, v interface{}) error {
	return json.Unmarshal(msg.Data, v)
}
