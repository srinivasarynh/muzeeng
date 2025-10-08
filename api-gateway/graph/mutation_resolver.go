package graph

import (
	"api-gateway/graph/helpers"
	"api-gateway/graph/model"
	"context"
	"fmt"

	authpb "auth-service/pb"
	commentpb "comment-service/pb"
	followpb "follow-service/pb"
	likepb "like-service/pb"
	notificationpb "notification-service/pb"
	postpb "post-service/pb"
	userpb "user-service/pb"

	"github.com/google/uuid"
)

// Register is the resolver for the register field.
func (r *mutationResolver) register(ctx context.Context, input model.RegisterInput) (*model.AuthResponse, error) {

	resp, err := r.AuthClient.Register(ctx, &authpb.RegisterRequest{
		Username: input.Username,
		Email:    input.Email,
		Password: input.Password,
		Bio:      input.Bio,
	})
	if err != nil {
		return nil, fmt.Errorf("registration failed: %w", err)
	}

	message := helpers.StringPtr(resp.Message)
	return &model.AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User: &model.User{
			ID:             uuid.MustParse(resp.User.Id),
			Username:       resp.User.Username,
			Email:          resp.User.Email,
			Bio:            resp.User.Bio,
			CreatedAt:      resp.User.CreatedAt.String(),
			UpdatedAt:      resp.User.UpdatedAt.String(),
			FollowersCount: 0,
			FollowingCount: 0,
			PostsCount:     0,
		},
		ExpiresIn: int32(resp.ExpiresIn),
		Message:   message,
	}, nil
}

// Login is the resolver for the login field.
func (r *mutationResolver) login(ctx context.Context, input model.LoginInput) (*model.AuthResponse, error) {
	resp, err := r.AuthClient.Login(ctx, &authpb.LoginRequest{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	message := helpers.StringPtr(resp.Message)
	return &model.AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User: &model.User{
			ID:             uuid.MustParse(resp.User.Id),
			Username:       resp.User.Username,
			Email:          resp.User.Email,
			Bio:            resp.User.Bio,
			CreatedAt:      resp.User.CreatedAt.String(),
			UpdatedAt:      resp.User.UpdatedAt.String(),
			FollowersCount: int32(resp.User.FollowersCount),
			FollowingCount: int32(resp.User.FollowingCount),
			PostsCount:     int32(resp.User.PostsCount),
		},
		ExpiresIn: int32(resp.ExpiresIn),
		Message:   message,
	}, nil
}

// RefreshToken is the resolver for the refreshToken field.
func (r *mutationResolver) refreshToken(ctx context.Context, refreshToken string) (*model.AuthResponse, error) {
	resp, err := r.AuthClient.RefreshToken(ctx, &authpb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	message := helpers.StringPtr(resp.Message)
	return &model.AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User: &model.User{
			ID:             uuid.MustParse(resp.User.Id),
			Username:       resp.User.Username,
			Email:          resp.User.Email,
			Bio:            resp.User.Bio,
			CreatedAt:      resp.User.CreatedAt.String(),
			UpdatedAt:      resp.User.UpdatedAt.String(),
			FollowersCount: int32(resp.User.FollowersCount),
			FollowingCount: int32(resp.User.FollowingCount),
			PostsCount:     int32(resp.User.PostsCount),
		},
		ExpiresIn: int32(resp.ExpiresIn),
		Message:   message,
	}, nil
}

// Logout is the resolver for the logout field.
func (r *mutationResolver) logout(ctx context.Context) (*model.Response, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.AuthClient.Logout(ctx, &authpb.LogoutRequest{})
	if err != nil {
		return nil, fmt.Errorf("logout failed: %w", err)
	}

	return &model.Response{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// UpdateProfile is the resolver for the updateProfile field.
func (r *mutationResolver) updateProfile(ctx context.Context, input model.UpdateProfileInput) (*model.User, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.UserClient.UpdateProfile(ctx, &userpb.UpdateProfileRequest{
		Username: input.Username,
		Email:    input.Email,
		Bio:      input.Bio,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
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

// ChangePassword is the resolver for the changePassword field.
func (r *mutationResolver) changePassword(ctx context.Context, input model.ChangePasswordInput) (*model.Response, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.AuthClient.ChangePassword(ctx, &authpb.ChangePasswordRequest{
		CurrentPassword: input.CurrentPassword,
		NewPassword:     input.NewPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to change password: %w", err)
	}

	return &model.Response{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// CreatePost is the resolver for the createPost field.
func (r *mutationResolver) createPost(ctx context.Context, input model.CreatePostInput) (*model.Post, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.PostClient.CreatePost(ctx, &postpb.CreatePostRequest{
		Content: input.Content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
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

// UpdatePost is the resolver for the updatePost field.
func (r *mutationResolver) updatePost(ctx context.Context, postID uuid.UUID, content string) (*model.Post, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.PostClient.UpdatePost(ctx, &postpb.UpdatePostRequest{
		PostId:  postID.String(),
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update post: %w", err)
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

// DeletePost is the resolver for the deletePost field.
func (r *mutationResolver) deletePost(ctx context.Context, postID uuid.UUID) (*model.Response, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.PostClient.DeletePost(ctx, &postpb.DeletePostRequest{
		PostId: postID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete post: %w", err)
	}

	return &model.Response{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// CreateComment is the resolver for the createComment field.
func (r *mutationResolver) createComment(ctx context.Context, input model.CreateCommentInput) (*model.Comment, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.CommentClient.CreateComment(ctx, &commentpb.CreateCommentRequest{
		PostId:  input.PostID.String(),
		Content: input.Content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	return &model.Comment{
		ID:        uuid.MustParse(resp.Id),
		PostID:    uuid.MustParse(resp.PostId),
		UserID:    uuid.MustParse(resp.UserId),
		Content:   resp.Content,
		CreatedAt: resp.CreatedAt.String(),
		UpdatedAt: resp.UpdatedAt.String(),
	}, nil
}

// UpdateComment is the resolver for the updateComment field.
func (r *mutationResolver) updateComment(ctx context.Context, commentID uuid.UUID, content string) (*model.Comment, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.CommentClient.UpdateComment(ctx, &commentpb.UpdateCommentRequest{
		CommentId: commentID.String(),
		Content:   content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	return &model.Comment{
		ID:        uuid.MustParse(resp.Id),
		PostID:    uuid.MustParse(resp.PostId),
		UserID:    uuid.MustParse(resp.UserId),
		Content:   resp.Content,
		CreatedAt: resp.CreatedAt.String(),
		UpdatedAt: resp.UpdatedAt.String(),
	}, nil
}

// DeleteComment is the resolver for the deleteComment field.
func (r *mutationResolver) deleteComment(ctx context.Context, commentID uuid.UUID) (*model.Response, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.CommentClient.DeleteComment(ctx, &commentpb.DeleteCommentRequest{
		CommentId: commentID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete comment: %w", err)
	}

	return &model.Response{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// LikePost is the resolver for the likePost field.
func (r *mutationResolver) likePost(ctx context.Context, postID uuid.UUID) (*model.Response, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.LikeClient.LikePost(ctx, &likepb.LikePostRequest{
		PostId: postID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to like post: %w", err)
	}

	return &model.Response{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// UnlikePost is the resolver for the unlikePost field.
func (r *mutationResolver) unlikePost(ctx context.Context, postID uuid.UUID) (*model.Response, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.LikeClient.UnlikePost(ctx, &likepb.UnlikePostRequest{
		PostId: postID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to unlike post: %w", err)
	}

	return &model.Response{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// FollowUser is the resolver for the followUser field.
func (r *mutationResolver) followUser(ctx context.Context, userID uuid.UUID) (*model.Response, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.FollowClient.FollowUser(ctx, &followpb.FollowUserRequest{
		FollowingId: userID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to follow user: %w", err)
	}

	return &model.Response{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// UnfollowUser is the resolver for the unfollowUser field.
func (r *mutationResolver) unfollowUser(ctx context.Context, userID uuid.UUID) (*model.Response, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.FollowClient.UnfollowUser(ctx, &followpb.UnfollowUserRequest{
		FollowingId: userID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to unfollow user: %w", err)
	}

	return &model.Response{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// MarkNotificationRead is the resolver for the markNotificationRead field.
func (r *mutationResolver) markNotificationRead(ctx context.Context, notificationID uuid.UUID) (*model.Response, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.NotificationClient.MarkRead(ctx, &notificationpb.MarkReadRequest{
		NotificationId: notificationID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return &model.Response{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// MarkAllNotificationsRead is the resolver for the markAllNotificationsRead field.
func (r *mutationResolver) markAllNotificationsRead(ctx context.Context) (*model.Response, error) {
	token := helpers.GetTokenFromContext(ctx)
	ctx = helpers.AddTokenToContext(ctx, token)

	resp, err := r.NotificationClient.MarkAllRead(ctx, &notificationpb.MarkAllReadRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	return &model.Response{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}
