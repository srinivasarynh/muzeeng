package publisher

import (
	"comment-service/events"
	natsClient "comment-service/nats"
	"encoding/json"
	"log"
)

type EventPublisher struct {
	nats *natsClient.Client
}

func NewEventPublisher(nats *natsClient.Client) *EventPublisher {
	return &EventPublisher{nats: nats}
}

func (p *EventPublisher) PublishCommentAdded(event events.CommentAddedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if err := p.nats.Publish(events.CommentAdded, data); err != nil {
		return err
	}

	log.Printf("Published event: %s for comment %s", events.CommentAdded, event.CommentID)
	return nil
}
