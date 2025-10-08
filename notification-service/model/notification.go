package models

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationTypeComment NotificationType = "COMMENT"
	NotificationTypePost    NotificationType = "POST"
)

type Notification struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	UserID    uuid.UUID        `json:"user_id" db:"user_id"`
	Type      NotificationType `json:"type" db:"type"`
	Message   string           `json:"message" db:"message"`
	ActorID   *uuid.UUID       `json:"actor_id,omitempty" db:"actor_id"`
	RelatedID *uuid.UUID       `json:"related_id,omitempty" db:"related_id"`
	IsRead    bool             `json:"is_read" db:"is_read"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
}

type NotificationEdge struct {
	Cursor string       `json:"cursor"`
	Node   Notification `json:"node"`
}

type PageInfo struct {
	EndCursor       *string `json:"end_cursor,omitempty"`
	HasNextPage     bool    `json:"has_next_page"`
	StartCursor     *string `json:"start_cursor,omitempty"`
	HasPreviousPage bool    `json:"has_previous_page"`
}

type NotificationConnection struct {
	Edges       []NotificationEdge `json:"edges"`
	PageInfo    PageInfo           `json:"page_info"`
	TotalCount  int32              `json:"total_count"`
	UnreadCount int32              `json:"unread_count"`
}
