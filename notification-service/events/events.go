package events

import (
	"time"

	"github.com/google/uuid"
)

// Event subjects (topics)
const (
	SubjectPostCommented = "post.commented"
	SubjectPostCreated   = "post.created"
)

// PostCommentedEvent is published when a user comments on a post
type PostCommentedEvent struct {
	PostID      uuid.UUID `json:"post_id"`
	PostOwner   uuid.UUID `json:"post_owner"`
	CommentID   uuid.UUID `json:"comment_id"`
	CommentedBy uuid.UUID `json:"commented_by"`
	Timestamp   time.Time `json:"timestamp"`
}

// PostCreatedEvent is published when a user creates a post
type PostCreatedEvent struct {
	PostID    uuid.UUID `json:"post_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
