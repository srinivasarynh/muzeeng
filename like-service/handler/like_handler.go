package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "like-service/pb"
	"like-service/repository"
)

type LikeHandler struct {
	pb.UnimplementedLikeServiceServer
	likeRepo repository.LikeRepository
}

func NewLikeHandler(likeRepo repository.LikeRepository) *LikeHandler {
	return &LikeHandler{
		likeRepo: likeRepo,
	}
}

// LikePost handles the request to like a post
func (h *LikeHandler) LikePost(ctx context.Context, req *pb.LikePostRequest) (*pb.Response, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	err = h.likeRepo.CreateLike(ctx, postID, userID)
	if err != nil {
		if err.Error() == "like already exists" {
			return &pb.Response{
				Success: true,
				Message: "Post already liked",
			}, nil
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to like post: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Post liked successfully",
	}, nil
}

// UnlikePost handles the request to unlike a post
func (h *LikeHandler) UnlikePost(ctx context.Context, req *pb.UnlikePostRequest) (*pb.Response, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	err = h.likeRepo.DeleteLike(ctx, postID, userID)
	if err != nil {
		if errors.Is(err, errors.New("like not found")) {
			return &pb.Response{
				Success: true,
				Message: "Like already removed",
			}, nil
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unlike post: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Post unliked successfully",
	}, nil
}

// GetPostLikes retrieves like information for a post
func (h *LikeHandler) GetPostLikes(ctx context.Context, req *pb.GetPostLikesRequest) (*pb.LikeInfo, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	count, err := h.likeRepo.GetLikeCountByPost(ctx, postID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get like count: %v", err))
	}

	limit := req.RecentLikersLimit
	if limit <= 0 {
		limit = 5
	}

	recentLikers, err := h.likeRepo.GetRecentLikersByPost(ctx, postID, limit)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get recent likers: %v", err))
	}

	recentLikerIDs := make([]string, len(recentLikers))
	for i, id := range recentLikers {
		recentLikerIDs[i] = id.String()
	}

	response := &pb.LikeInfo{
		Count:          count,
		RecentLikerIds: recentLikerIDs,
	}

	if req.RequestingUserId != nil && *req.RequestingUserId != "" {
		userID, err := uuid.Parse(*req.RequestingUserId)
		if err == nil {
			isLiked, err := h.likeRepo.IsPostLikedByUser(ctx, postID, userID)
			if err == nil {
				response.IsLikedByCurrentUser = &isLiked
			}
		}
	}

	return response, nil
}

// IsPostLikedByUser checks if a user has liked a specific post
func (h *LikeHandler) IsPostLikedByUser(ctx context.Context, req *pb.IsPostLikedByUserRequest) (*pb.IsPostLikedByUserResponse, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	isLiked, err := h.likeRepo.IsPostLikedByUser(ctx, postID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to check like status: %v", err))
	}

	return &pb.IsPostLikedByUserResponse{
		IsLiked: isLiked,
	}, nil
}

// GetPostLikesByUsers retrieves like status for multiple posts by a specific user
func (h *LikeHandler) GetPostLikesByUsers(ctx context.Context, req *pb.GetPostLikesByUsersRequest) (*pb.GetPostLikesByUsersResponse, error) {
	if len(req.PostIds) == 0 {
		return &pb.GetPostLikesByUsersResponse{
			Likes: []*pb.PostLikeStatus{},
		}, nil
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	postIDs := make([]uuid.UUID, len(req.PostIds))
	for i, postIDStr := range req.PostIds {
		postID, err := uuid.Parse(postIDStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid post_id format at index %d", i))
		}
		postIDs[i] = postID
	}

	likeStatuses, err := h.likeRepo.GetPostLikesByUsers(ctx, postIDs, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get post likes: %v", err))
	}

	pbLikeStatuses := make([]*pb.PostLikeStatus, len(likeStatuses))
	for i, status := range likeStatuses {
		pbLikeStatuses[i] = &pb.PostLikeStatus{
			PostId:  status.PostID.String(),
			IsLiked: status.IsLiked,
		}
	}

	return &pb.GetPostLikesByUsersResponse{
		Likes: pbLikeStatuses,
	}, nil
}
