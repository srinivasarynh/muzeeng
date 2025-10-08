package publisher

import (
	"encoding/json"
	"log"
	"post-service/events"
	natsClient "post-service/nats"
)

type EventPublisher struct {
	nats *natsClient.Client
}

func NewEventPublisher(nats *natsClient.Client) *EventPublisher {
	return &EventPublisher{nats: nats}
}

func (p *EventPublisher) PublishPostCreated(event events.PostCreatedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if err := p.nats.Publish(events.PostCreated, data); err != nil {
		return err
	}

	log.Printf("Published event: %s for post %s", events.PostCreated, event.PostID)
	return nil
}
