package graph

import (
	"api-gateway/graph/helpers"
	"api-gateway/graph/model"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

func (r *subscriptionResolver) notificationAdded(ctx context.Context) (<-chan *model.Notification, error) {
	token := helpers.GetTokenFromContext(ctx)
	if token == "" {
		return nil, fmt.Errorf("authentication required")
	}

	userID := "user-id-from-token"

	ch := make(chan *model.Notification, 1)
	subject := fmt.Sprintf("notifications.%s", userID)

	sub, err := r.NatsConn.Subscribe(subject, func(msg *nats.Msg) {
		var notif struct {
			ID        string `json:"id"`
			UserID    string `json:"user_id"`
			Type      string `json:"type"`
			Message   string `json:"message"`
			ActorID   string `json:"actor_id"`
			RelatedID string `json:"related_id"`
			IsRead    bool   `json:"is_read"`
			CreatedAt string `json:"created_at"`
		}

		if err := json.Unmarshal(msg.Data, &notif); err != nil {
			return
		}

		select {
		case ch <- &model.Notification{
			ID:        uuid.MustParse(notif.ID),
			UserID:    uuid.MustParse(notif.UserID),
			Type:      model.NotificationType(notif.Type),
			Message:   notif.Message,
			ActorID:   helpers.ParseUUIDPtr(notif.ActorID),
			RelatedID: helpers.ParseUUIDPtr(notif.RelatedID),
			IsRead:    notif.IsRead,
			CreatedAt: notif.CreatedAt,
		}:
		case <-ctx.Done():
			return
		}
	})

	if err != nil {
		close(ch)
		return nil, fmt.Errorf("failed to subscribe to notifications: %w", err)
	}

	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		close(ch)
	}()

	return ch, nil
}

// PostAdded is the resolver for the postAdded field.
func (r *subscriptionResolver) postAdded(ctx context.Context, userID uuid.UUID) (<-chan *model.Post, error) {
	token := helpers.GetTokenFromContext(ctx)
	if token == "" {
		return nil, fmt.Errorf("authentication required")
	}

	ch := make(chan *model.Post, 1)

	subject := fmt.Sprintf("posts.%s", userID)

	sub, err := r.NatsConn.Subscribe(subject, func(msg *nats.Msg) {
		var post struct {
			ID            string `json:"id"`
			UserID        string `json:"user_id"`
			Content       string `json:"content"`
			CreatedAt     string `json:"created_at"`
			UpdatedAt     string `json:"updated_at"`
			LikesCount    int32  `json:"likes_count"`
			CommentsCount int32  `json:"comments_count"`
		}

		if err := json.Unmarshal(msg.Data, &post); err != nil {
			return
		}

		select {
		case ch <- &model.Post{
			ID:            uuid.MustParse(post.ID),
			UserID:        uuid.MustParse(post.UserID),
			Content:       post.Content,
			CreatedAt:     post.CreatedAt,
			UpdatedAt:     post.UpdatedAt,
			LikesCount:    int32(post.LikesCount),
			CommentsCount: int32(post.CommentsCount),
		}:
		case <-ctx.Done():
			return
		}
	})

	if err != nil {
		close(ch)
		return nil, fmt.Errorf("failed to subscribe to posts: %w", err)
	}

	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		close(ch)
	}()

	return ch, nil
}

// CommentAdded is the resolver for the commentAdded field.
func (r *subscriptionResolver) commentAdded(ctx context.Context, postID uuid.UUID) (<-chan *model.Comment, error) {
	ch := make(chan *model.Comment, 1)

	subject := fmt.Sprintf("comments.%s", postID)

	sub, err := r.NatsConn.Subscribe(subject, func(msg *nats.Msg) {
		var comment struct {
			ID        string `json:"id"`
			PostID    string `json:"post_id"`
			UserID    string `json:"user_id"`
			Content   string `json:"content"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		}

		if err := json.Unmarshal(msg.Data, &comment); err != nil {
			return
		}

		select {
		case ch <- &model.Comment{
			ID:        uuid.MustParse(comment.ID),
			PostID:    uuid.MustParse(comment.PostID),
			UserID:    uuid.MustParse(comment.UserID),
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
			UpdatedAt: comment.UpdatedAt,
		}:
		case <-ctx.Done():
			return
		}
	})

	if err != nil {
		close(ch)
		return nil, fmt.Errorf("failed to subscribe to comments: %w", err)
	}

	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		close(ch)
	}()

	return ch, nil
}
