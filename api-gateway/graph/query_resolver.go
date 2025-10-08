package graph

import (
	"api-gateway/graph/helpers"
	"api-gateway/graph/model"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	likepb "like-service/pb"
	postpb "post-service/pb"
	userpb "user-service/pb"
)

// HealthCheck is the resolver for the healthCheck field.
func (r *queryResolver) healthCheck(ctx context.Context) (*model.HealthCheckResponse, error) {
	return &model.HealthCheckResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
		Services: []*model.ServiceStatus{
			{Name: "AuthService", Status: "healthy", Latency: helpers.Int32Ptr(10)},
			{Name: "UserService", Status: "healthy", Latency: helpers.Int32Ptr(15)},
			{Name: "PostService", Status: "healthy", Latency: helpers.Int32Ptr(12)},
		},
	}, nil
}

// Me is the resolver for the me field.
func (r *queryResolver) me(ctx context.Context) (*model.User, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.UserClient.GetMe(ctx, &userpb.GetMeRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return &model.User{
		ID:             uuid.MustParse(resp.Id),
		Username:       resp.Username,
		Email:          resp.Email,
		Bio:            resp.Bio,
		CreatedAt:      resp.CreatedAt.String(),
		UpdatedAt:      resp.UpdatedAt.String(),
		FollowersCount: int32(resp.FollowersCount),
		FollowingCount: int32(resp.FollowingCount),
		PostsCount:     int32(resp.PostsCount),
	}, nil
}

// GetProfile is the resolver for the getProfile field.
func (r *queryResolver) getProfile(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	resp, err := r.UserClient.GetProfile(ctx, &userpb.GetProfileRequest{
		UserId: userID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return &model.User{
		ID:             uuid.MustParse(resp.Id),
		Username:       resp.Username,
		Email:          resp.Email,
		Bio:            resp.Bio,
		CreatedAt:      resp.CreatedAt.String(),
		UpdatedAt:      resp.UpdatedAt.String(),
		FollowersCount: int32(resp.FollowersCount),
		FollowingCount: int32(resp.FollowingCount),
		PostsCount:     int32(resp.PostsCount),
	}, nil
}

// GetPost is the resolver for the getPost field.
func (r *queryResolver) getPost(ctx context.Context, postID uuid.UUID) (*model.Post, error) {
	resp, err := r.PostClient.GetPost(ctx, &postpb.GetPostRequest{
		PostId: postID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	return &model.Post{
		ID:            uuid.MustParse(resp.Id),
		UserID:        uuid.MustParse(resp.UserId),
		Content:       resp.Content,
		CreatedAt:     resp.CreatedAt.String(),
		UpdatedAt:     resp.UpdatedAt.String(),
		LikesCount:    int32(resp.LikesCount),
		CommentsCount: int32(resp.CommentsCount),
	}, nil
}

func (r *queryResolver) getPostLikes(ctx context.Context, postID uuid.UUID) (*model.LikeInfo, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.LikeClient.GetPostLikes(ctx, &likepb.GetPostLikesRequest{
		PostId: postID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get likes: %w", err)
	}

	recentLikers := make([]*model.User, len(resp.GetRecentLikerIds()))
	for i, userID := range resp.GetRecentLikerIds() {
		userResp, err := r.UserClient.GetProfile(ctx, &userpb.GetProfileRequest{
			UserId: userID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user %s: %w", userID, err)
		}
		recentLikers[i] = helpers.ProtoUserToModel(userResp)
	}

	return &model.LikeInfo{
		Count:        resp.Count,
		RecentLikers: recentLikers,
	}, nil
}
