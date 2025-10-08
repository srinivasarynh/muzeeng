package events

import (
	"time"

	"github.com/google/uuid"
)

const (
	PostCreated = "post.created"
)

// Event payloads
type PostCreatedEvent struct {
	PostID    uuid.UUID `json:"post_id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
