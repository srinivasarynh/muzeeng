package events

import (
	"time"

	"github.com/google/uuid"
)

const (
	CommentAdded = "post.comment.added"
)

type CommentAddedEvent struct {
	CommentID  uuid.UUID `json:"comment_id"`
	PostID     uuid.UUID `json:"post_id"`
	PostUserID uuid.UUID `json:"post_user_id"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}
