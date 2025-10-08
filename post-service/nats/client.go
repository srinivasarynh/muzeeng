package nats

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type Config struct {
	URL           string
	MaxReconnects int
	ReconnectWait time.Duration
	ClientID      string
}

type Client struct {
	conn *nats.Conn
}

func NewClient(cfg Config) (*Client, error) {
	opts := []nats.Option{
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Printf("NATS disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
		}),
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn}, nil
}

func (c *Client) Publish(subject string, data []byte) error {
	return c.conn.Publish(subject, data)
}

func (c *Client) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return c.conn.Subscribe(subject, handler)
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
