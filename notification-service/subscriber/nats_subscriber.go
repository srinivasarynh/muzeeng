package subscriber

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"notification-service/events"
	"notification-service/model"
	natsClient "notification-service/nats"
	"notification-service/repository"
)

type NotificationSubscriber struct {
	natsClient *natsClient.Client
	repo       repository.NotificationRepository
	ctx        context.Context
}

func NewNotificationSubscriber(
	natsClient *natsClient.Client,
	repo repository.NotificationRepository,
	ctx context.Context,
) *NotificationSubscriber {
	return &NotificationSubscriber{
		natsClient: natsClient,
		repo:       repo,
		ctx:        ctx,
	}
}

func (s *NotificationSubscriber) Start() error {
	subjects := []string{
		events.SubjectPostCreated,
		events.SubjectPostCommented,
	}

	err := s.natsClient.CreateStream("NOTIFICATIONS", subjects)
	if err != nil {
		log.Printf("Stream might already exist or error creating: %v", err)
	}

	if err := s.subscribeToPostCreated(); err != nil {
		return err
	}

	if err := s.subscribeToPostCommented(); err != nil {
		return err
	}

	log.Println("Notification subscriber started successfully")
	return nil
}

func (s *NotificationSubscriber) subscribeToPostCreated() error {
	handler := func(msg *nats.Msg) {
		var event events.PostCreatedEvent
		if err := natsClient.DecodeEvent(msg, &event); err != nil {
			log.Printf("Error decoding post created event: %v", err)
			msg.Nak()
			return
		}

		notification := &models.Notification{
			ID:        uuid.New(),
			UserID:    event.AuthorID,
			Type:      models.NotificationTypePost,
			Message:   "created a new post",
			ActorID:   &event.AuthorID,
			RelatedID: &event.PostID,
			IsRead:    false,
			CreatedAt: event.Timestamp,
		}

		if err := s.repo.Create(s.ctx, notification); err != nil {
			log.Printf("Error creating post notification: %v", err)
			msg.Nak()
			return
		}

		log.Printf("Created post notification for user %s", event.AuthorID)
		msg.Ack()
	}

	_, err := s.natsClient.SubscribeDurable(
		events.SubjectPostCreated,
		"notification-service-posts",
		"notification-workers",
		handler,
	)

	return err
}

func (s *NotificationSubscriber) subscribeToPostCommented() error {
	handler := func(msg *nats.Msg) {
		var event events.PostCommentedEvent
		if err := natsClient.DecodeEvent(msg, &event); err != nil {
			log.Printf("Error decoding post commented event: %v", err)
			msg.Nak()
			return
		}

		if event.PostOwner == event.CommentedBy {
			msg.Ack()
			return
		}

		notification := &models.Notification{
			ID:        uuid.New(),
			UserID:    event.PostOwner,
			Type:      models.NotificationTypeComment,
			Message:   "commented on your post",
			ActorID:   &event.CommentedBy,
			RelatedID: &event.PostID,
			IsRead:    false,
			CreatedAt: event.Timestamp,
		}

		if err := s.repo.Create(s.ctx, notification); err != nil {
			log.Printf("Error creating comment notification: %v", err)
			msg.Nak()
			return
		}

		log.Printf("Created comment notification for user %s", event.PostOwner)
		msg.Ack()
	}

	_, err := s.natsClient.SubscribeDurable(
		events.SubjectPostCommented,
		"notification-service-comments",
		"notification-workers",
		handler,
	)

	return err
}

func (s *NotificationSubscriber) Stop() error {
	if s.natsClient != nil {
		s.natsClient.Close()
	}
	return nil
}
