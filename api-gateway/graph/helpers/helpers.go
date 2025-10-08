package helpers

import (
	"api-gateway/graph/model"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"

	commentpb "comment-service/pb"
	followpb "follow-service/pb"
	notificationpb "notification-service/pb"
	postpb "post-service/pb"
	userpb "user-service/pb"
)

// NotificationAdded is the resolver for the notificationAdded field.
func ParseUUIDPtr(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

func GetTokenFromContext(ctx context.Context) string {
	token, ok := ctx.Value("token").(string)
	if !ok || token == "" {
		return ""
	}
	return token
}

func AddTokenToContext(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", token)
}

func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
func Int32Ptr(i int32) *int32 {
	return &i
}

// --------------------
// Connection Builders
// --------------------

func buildPostConnection(posts []*postpb.Post, limit int) *model.PostConnection {
	hasNextPage := len(posts) > limit
	if hasNextPage {
		posts = posts[:limit]
	}

	edges := make([]*model.PostEdge, len(posts))
	totalCount := len(posts)

	for i, p := range posts {
		edges[i] = &model.PostEdge{
			Cursor: p.CreatedAt.AsTime().Format(time.RFC3339),
			Node:   protoPostToModel(p),
		}
	}

	var endCursor *string
	var startCursor *string
	if len(posts) > 0 {
		startStr := posts[0].CreatedAt.AsTime().Format(time.RFC3339)
		startCursor = &startStr

		lastStr := posts[len(posts)-1].CreatedAt.AsTime().Format(time.RFC3339)
		endCursor = &lastStr
	}

	return &model.PostConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			StartCursor:     startCursor,
			EndCursor:       endCursor,
			HasNextPage:     hasNextPage,
			HasPreviousPage: false,
		},
		TotalCount: int32(totalCount),
	}
}

func buildCommentConnection(comments []*commentpb.Comment, limit int) *model.CommentConnection {
	hasNextPage := len(comments) > limit
	if hasNextPage {
		comments = comments[:limit]
	}

	edges := make([]*model.CommentEdge, len(comments))
	totalCount := len(comments)

	for i, c := range comments {
		edges[i] = &model.CommentEdge{
			Cursor: c.CreatedAt.AsTime().Format(time.RFC3339),
			Node:   protoCommentToModel(c),
		}
	}

	var endCursor *string
	var startCursor *string
	if len(comments) > 0 {
		startStr := comments[0].CreatedAt.AsTime().Format(time.RFC3339)
		startCursor = &startStr

		lastStr := comments[len(comments)-1].CreatedAt.AsTime().Format(time.RFC3339)
		endCursor = &lastStr
	}

	return &model.CommentConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			StartCursor:     startCursor,
			EndCursor:       endCursor,
			HasNextPage:     hasNextPage,
			HasPreviousPage: false,
		},
		TotalCount: int32(totalCount),
	}
}

func buildFollowConnection(edgesData []*followpb.FollowEdge, limit int) *model.FollowConnection {
	hasNextPage := len(edgesData) > limit
	if hasNextPage {
		edgesData = edgesData[:limit]
	}

	edges := make([]*model.FollowEdge, len(edgesData))
	totalCount := len(edgesData)

	var endCursor *string
	var startCursor *string
	if len(edgesData) > 0 {
		startStr := edgesData[0].FollowedAt.AsTime().Format(time.RFC3339)
		startCursor = &startStr

		lastStr := edgesData[len(edgesData)-1].FollowedAt.AsTime().Format(time.RFC3339)
		endCursor = &lastStr
	}

	return &model.FollowConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			StartCursor:     startCursor,
			EndCursor:       endCursor,
			HasNextPage:     hasNextPage,
			HasPreviousPage: false,
		},
		TotalCount: int32(totalCount),
	}
}

func BuildNotificationConnection(notifications []*notificationpb.Notification, unreadCount int32, limit int) *model.NotificationConnection {
	hasNextPage := len(notifications) > limit
	if hasNextPage {
		notifications = notifications[:limit]
	}

	edges := make([]*model.NotificationEdge, len(notifications))
	totalCount := len(notifications)

	for i, n := range notifications {
		edges[i] = &model.NotificationEdge{
			Cursor: n.CreatedAt.AsTime().Format(time.RFC3339),
			Node:   protoNotificationToModel(n),
		}
	}

	var endCursor *string
	var startCursor *string
	if len(notifications) > 0 {
		startStr := notifications[0].CreatedAt.AsTime().Format(time.RFC3339)
		startCursor = &startStr

		lastStr := notifications[len(notifications)-1].CreatedAt.AsTime().Format(time.RFC3339)
		endCursor = &lastStr
	}

	return &model.NotificationConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			StartCursor:     startCursor,
			EndCursor:       endCursor,
			HasNextPage:     hasNextPage,
			HasPreviousPage: false,
		},
		TotalCount:  int32(totalCount),
		UnreadCount: unreadCount,
	}
}

// Converts gRPC post response to GraphQL model
func protoPostToModel(p *postpb.Post) *model.Post {
	if p == nil {
		return nil
	}

	id, _ := uuid.Parse(p.Id)
	userID, _ := uuid.Parse(p.UserId)

	return &model.Post{
		ID:            id,
		UserID:        userID,
		Content:       p.Content,
		CreatedAt:     p.CreatedAt.String(),
		UpdatedAt:     p.UpdatedAt.String(),
		LikesCount:    p.LikesCount,
		CommentsCount: p.CommentsCount,
		IsLiked:       p.IsLiked,
	}
}

// Converts gRPC comment response to GraphQL model
func protoCommentToModel(c *commentpb.Comment) *model.Comment {
	if c == nil {
		return nil
	}

	id, _ := uuid.Parse(c.Id)
	postID, _ := uuid.Parse(c.PostId)
	userID, _ := uuid.Parse(c.UserId)

	return &model.Comment{
		ID:        id,
		PostID:    postID,
		UserID:    userID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt.String(),
		UpdatedAt: c.UpdatedAt.String(),
	}
}

func protoNotificationToModel(n *notificationpb.Notification) *model.Notification {
	if n == nil {
		return nil
	}

	id, _ := uuid.Parse(n.Id)
	userID, _ := uuid.Parse(n.UserId)

	var actorID *uuid.UUID
	switch v := any(n.ActorId).(type) {
	case string:
		if v != "" {
			aid, _ := uuid.Parse(v)
			actorID = &aid
		}
	case *string:
		if v != nil && *v != "" {
			aid, _ := uuid.Parse(*v)
			actorID = &aid
		}
	}

	var relatedID *uuid.UUID
	switch v := any(n.RelatedId).(type) {
	case string:
		if v != "" {
			rid, _ := uuid.Parse(v)
			relatedID = &rid
		}
	case *string:
		if v != nil && *v != "" {
			rid, _ := uuid.Parse(*v)
			relatedID = &rid
		}
	}

	notifType := model.NotificationType(n.Type)

	return &model.Notification{
		ID:        id,
		UserID:    userID,
		Type:      notifType,
		Message:   n.Message,
		ActorID:   actorID,
		RelatedID: relatedID,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt.String(),
	}
}

// --------------------
// Cursor-Based Pagination Resolvers
// --------------------

func ParseCursor(after *string) (*time.Time, error) {
	if after == nil || *after == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *after)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	return &t, nil
}

// Converts gRPC user response to GraphQL model
func ProtoUserToModel(u *userpb.User) *model.User {
	if u == nil {
		return nil
	}

	id, _ := uuid.Parse(u.Id)

	var isFollowing *bool
	if u.IsFollowing != nil {
		isFollowing = u.IsFollowing // already *bool in proto
	}

	return &model.User{
		ID:             id,
		Username:       u.Username,
		Email:          u.Email,
		Bio:            u.Bio,
		CreatedAt:      u.CreatedAt.AsTime().Format(time.RFC3339),
		UpdatedAt:      u.UpdatedAt.AsTime().Format(time.RFC3339),
		FollowersCount: int32(u.FollowersCount),
		FollowingCount: int32(u.FollowingCount),
		PostsCount:     int32(u.PostsCount),
		IsFollowing:    isFollowing,
	}
}
