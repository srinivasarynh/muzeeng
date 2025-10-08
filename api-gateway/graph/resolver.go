package graph

// This file will not be regenerated automatically.
// It serves as dependency injection for your app; add any dependencies you require here.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"api-gateway/graph/helpers"
	"api-gateway/graph/model"
	authpb "auth-service/pb"
	commentpb "comment-service/pb"
	feedpb "feed-service/pb"
	followpb "follow-service/pb"
	likepb "like-service/pb"
	notificationpb "notification-service/pb"
	postpb "post-service/pb"
	userpb "user-service/pb"
)

// Resolver contains all gRPC clients and dependencies for GraphQL resolvers
type Resolver struct {
	AuthClient         authpb.AuthServiceClient
	UserClient         userpb.UserServiceClient
	PostClient         postpb.PostServiceClient
	CommentClient      commentpb.CommentServiceClient
	LikeClient         likepb.LikeServiceClient
	FollowClient       followpb.FollowServiceClient
	NotificationClient notificationpb.NotificationServiceClient
	FeedClient         feedpb.FeedServiceClient
	NatsConn           *nats.Conn
}

// NewResolver initializes gRPC clients and NATS connection
func NewResolver(ctx context.Context) (*Resolver, error) {
	dial := func(addr string) (*grpc.ClientConn, error) {
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
		}
		return conn, nil
	}

	authConn, err := dial("auth-service:50051")
	if err != nil {
		return nil, err
	}
	userConn, err := dial("user-service:50052")
	if err != nil {
		return nil, err
	}
	postConn, err := dial("post-service:50053")
	if err != nil {
		return nil, err
	}
	commentConn, err := dial("comment-service:50055")
	if err != nil {
		return nil, err
	}
	likeConn, err := dial("like-service:50057")
	if err != nil {
		return nil, err
	}
	followConn, err := dial("follow-service:50060")
	if err != nil {
		return nil, err
	}
	notifConn, err := dial("notification-service:50058")
	if err != nil {
		return nil, err
	}
	feedConn, err := dial("feed-service:50054")
	if err != nil {
		return nil, err
	}

	nc, err := nats.Connect("nats://nats:4222")
	if err != nil {
		log.Printf("⚠️ Warning: Failed to connect to NATS: %v", err)
	}

	return &Resolver{
		AuthClient:         authpb.NewAuthServiceClient(authConn),
		UserClient:         userpb.NewUserServiceClient(userConn),
		PostClient:         postpb.NewPostServiceClient(postConn),
		CommentClient:      commentpb.NewCommentServiceClient(commentConn),
		LikeClient:         likepb.NewLikeServiceClient(likeConn),
		FollowClient:       followpb.NewFollowServiceClient(followConn),
		NotificationClient: notificationpb.NewNotificationServiceClient(notifConn),
		FeedClient:         feedpb.NewFeedServiceClient(feedConn),
		NatsConn:           nc,
	}, nil
}

// Adds the JWT token from context to gRPC metadata
func (r *Resolver) getAuthContext(ctx context.Context) context.Context {
	token, ok := ctx.Value("token").(string)
	if !ok || token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", token)
}

// GetFeed implements cursor-based pagination for feed posts (uses FeedService)
func (r *Resolver) getFeed(ctx context.Context, first *int32, after *string) (*model.PostConnection, error) {
	limit := 10
	if first != nil && *first > 0 {
		limit = int(*first)
	}

	afterTime, err := helpers.ParseCursor(after)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination cursor: %w", err)
	}

	req := &feedpb.GetFeedRequest{
		First: int32(limit + 1),
	}
	if afterTime != nil {
		afterStr := afterTime.Format(time.RFC3339)
		req.After = &afterStr
	}

	resp, err := r.FeedClient.GetFeed(r.getAuthContext(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed from FeedService: %w", err)
	}

	edges := make([]*model.PostEdge, len(resp.Edges))
	for i, e := range resp.Edges {
		edges[i] = &model.PostEdge{
			Cursor: e.Node.CreatedAt.String(),
			Node: &model.Post{
				ID:         uuid.MustParse(e.Node.Id),
				UserID:     uuid.MustParse(e.Node.UserId),
				Content:    e.Node.Content,
				CreatedAt:  e.Node.CreatedAt.String(),
				LikesCount: int32(e.Node.LikesCount),
			},
		}
	}

	var endCursor *string
	hasNextPage := false
	if len(edges) > 0 {
		last := edges[len(edges)-1].Cursor
		endCursor = &last
		hasNextPage = len(edges) > limit
	}

	return &model.PostConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			EndCursor:   endCursor,
			HasNextPage: hasNextPage,
		},
	}, nil
}

// GetUserPosts implements cursor-based pagination for a user's posts
func (r *Resolver) getUserPosts(ctx context.Context, userID uuid.UUID, first *int32, after *string) (*model.PostConnection, error) {
	limit := 10
	if first != nil && *first > 0 {
		limit = int(*first)
	}

	afterTime, err := helpers.ParseCursor(after)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	req := &postpb.GetUserPostsRequest{
		UserId: userID.String(),
		First:  int32(limit + 1),
	}
	if afterTime != nil {
		afterStr := afterTime.Format(time.RFC3339)
		req.After = &afterStr
	}

	resp, err := r.PostClient.GetUserPosts(r.getAuthContext(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user posts: %w", err)
	}

	edges := make([]*model.PostEdge, len(resp.Edges))
	for i, e := range resp.Edges {
		edges[i] = &model.PostEdge{
			Cursor: e.Node.CreatedAt.String(),
			Node: &model.Post{
				ID:         uuid.MustParse(e.Node.Id),
				UserID:     uuid.MustParse(e.Node.UserId),
				Content:    e.Node.Content,
				CreatedAt:  e.Node.CreatedAt.String(),
				LikesCount: int32(e.Node.LikesCount),
			},
		}
	}

	var endCursor *string
	hasNextPage := false
	if len(edges) > 0 {
		last := edges[len(edges)-1].Cursor
		endCursor = &last
		hasNextPage = len(edges) > limit
	}

	return &model.PostConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			EndCursor:   endCursor,
			HasNextPage: hasNextPage,
		},
	}, nil
}

// GetPostComments implements cursor-based pagination for comments
func (r *Resolver) getPostComments(ctx context.Context, postID uuid.UUID, first *int32, after *string) (*model.CommentConnection, error) {
	limit := 10
	if first != nil && *first > 0 {
		limit = int(*first)
	}

	afterTime, err := helpers.ParseCursor(after)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	req := &commentpb.GetPostCommentsRequest{
		PostId: postID.String(),
		First:  int32(limit + 1),
	}
	if afterTime != nil {
		afterStr := afterTime.Format(time.RFC3339)
		req.After = &afterStr
	}

	resp, err := r.CommentClient.GetPostComments(r.getAuthContext(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch comments: %w", err)
	}

	edges := make([]*model.CommentEdge, len(resp.Edges))
	for i, e := range resp.Edges {
		edges[i] = &model.CommentEdge{
			Cursor: e.Node.CreatedAt.String(),
			Node: &model.Comment{
				ID:        uuid.MustParse(e.Node.Id),
				PostID:    uuid.MustParse(e.Node.PostId),
				UserID:    uuid.MustParse(e.Node.UserId),
				Content:   e.Node.Content,
				CreatedAt: e.Node.CreatedAt.String(),
			},
		}
	}

	var endCursor *string
	hasNextPage := false
	if len(edges) > 0 {
		last := edges[len(edges)-1].Cursor
		endCursor = &last
		hasNextPage = len(edges) > limit
	}

	return &model.CommentConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			EndCursor:   endCursor,
			HasNextPage: hasNextPage,
		},
	}, nil
}

func (r *Resolver) getFollowers(ctx context.Context, userID uuid.UUID, first *int32, after *string) (*model.FollowConnection, error) {
	limit := 10
	if first != nil && *first > 0 {
		limit = int(*first)
	}

	afterTime, err := helpers.ParseCursor(after)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	req := &followpb.GetFollowersRequest{
		UserId: userID.String(),
		First:  int32(limit + 1),
	}
	if afterTime != nil {
		afterStr := afterTime.Format(time.RFC3339)
		req.After = &afterStr
	}

	resp, err := r.FollowClient.GetFollowers(r.getAuthContext(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch followers: %w", err)
	}

	edges := make([]*model.FollowEdge, len(resp.Edges))
	for i, e := range resp.Edges {
		cursor := e.FollowedAt.AsTime().Format(time.RFC3339)

		edges[i] = &model.FollowEdge{
			Cursor:     cursor,
			FollowedAt: cursor,
			Node: &model.User{
				ID:        uuid.MustParse(e.UserId),
				CreatedAt: cursor,
			},
		}
	}

	var startCursor, endCursor *string
	if len(edges) > 0 {
		startStr := edges[0].Cursor
		startCursor = &startStr
		endStr := edges[len(edges)-1].Cursor
		endCursor = &endStr
	}

	hasNextPage := len(edges) > limit
	if hasNextPage {
		edges = edges[:limit]
	}

	return &model.FollowConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			StartCursor:     startCursor,
			EndCursor:       endCursor,
			HasNextPage:     hasNextPage,
			HasPreviousPage: false,
		},
		TotalCount: int32(len(edges)),
	}, nil
}

func (r *Resolver) getFollowing(ctx context.Context, userID uuid.UUID, first *int32, after *string) (*model.FollowConnection, error) {
	limit := 10
	if first != nil && *first > 0 {
		limit = int(*first)
	}

	afterTime, err := helpers.ParseCursor(after)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	req := &followpb.GetFollowingRequest{
		UserId: userID.String(),
		First:  int32(limit + 1),
	}
	if afterTime != nil {
		afterStr := afterTime.Format(time.RFC3339)
		req.After = &afterStr
	}

	resp, err := r.FollowClient.GetFollowing(r.getAuthContext(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch following: %w", err)
	}

	edges := make([]*model.FollowEdge, len(resp.Edges))
	for i, e := range resp.Edges {
		cursor := e.FollowedAt.AsTime().Format(time.RFC3339)

		edges[i] = &model.FollowEdge{
			Cursor:     cursor,
			FollowedAt: cursor,
			Node: &model.User{
				ID:        uuid.MustParse(e.UserId),
				CreatedAt: e.FollowedAt.AsTime().Format(time.RFC3339),
			},
		}
	}

	var startCursor, endCursor *string
	if len(edges) > 0 {
		startStr := edges[0].Cursor
		startCursor = &startStr
		endStr := edges[len(edges)-1].Cursor
		endCursor = &endStr
	}

	hasNextPage := len(edges) > limit
	if hasNextPage {
		edges = edges[:limit]
	}

	return &model.FollowConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			StartCursor:     startCursor,
			EndCursor:       endCursor,
			HasNextPage:     hasNextPage,
			HasPreviousPage: false,
		},
		TotalCount: int32(len(edges)),
	}, nil
}

func (r *Resolver) getNotifications(ctx context.Context, first *int32, after *string) (*model.NotificationConnection, error) {
	limit := 10
	if first != nil && *first > 0 {
		limit = int(*first)
	}

	afterTime, err := helpers.ParseCursor(after)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	req := &notificationpb.GetNotificationsRequest{
		First: int32(limit + 1), // fetch one extra to detect next page
	}
	if afterTime != nil {
		afterStr := afterTime.Format(time.RFC3339)
		req.After = &afterStr
	}

	resp, err := r.NotificationClient.GetNotifications(r.getAuthContext(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notifications: %w", err)
	}

	notifications := make([]*notificationpb.Notification, len(resp.Edges))
	for i, edge := range resp.Edges {
		notifications[i] = edge.Node
	}

	return helpers.BuildNotificationConnection(notifications, resp.UnreadCount, limit), nil
}

// Publishes a notification message to NATS
func (r *Resolver) publishNotification(userID, notifType, message string) {
	if r.NatsConn == nil {
		log.Printf("⚠️ NATS connection not initialized; skipping notification")
		return
	}

	data, _ := json.Marshal(map[string]string{
		"user_id": userID,
		"type":    notifType,
		"message": message,
	})

	if err := r.NatsConn.Publish("notifications", data); err != nil {
		log.Printf("⚠️ Failed to publish NATS notification: %v", err)
	}
}
