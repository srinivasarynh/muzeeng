package handler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	models "user-service/model"
	pb "user-service/pb"
	"user-service/repository"
)

type UserHandler struct {
	pb.UnimplementedUserServiceServer
	repo repository.UserRepository
}

func NewUserHandler(repo repository.UserRepository) *UserHandler {
	return &UserHandler{
		repo: repo,
	}
}

func (h *UserHandler) GetMe(ctx context.Context, req *pb.GetMeRequest) (*pb.User, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	user, err := h.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	pbUser := &pb.User{
		Id:             user.ID.String(),
		Username:       user.Username,
		Email:          user.Email,
		CreatedAt:      timestamppb.New(user.CreatedAt),
		UpdatedAt:      timestamppb.New(user.UpdatedAt),
		FollowersCount: user.FollowersCount,
		FollowingCount: user.FollowingCount,
		PostsCount:     user.PostsCount,
	}

	if user.Bio != nil {
		pbUser.Bio = user.Bio
	}

	if req.RequestingUserId != nil && *req.RequestingUserId != "" && *req.RequestingUserId != req.UserId {
		requestingUserID, err := uuid.Parse(*req.RequestingUserId)
		if err == nil {
			isFollowing, err := h.repo.CheckFollowStatus(ctx, userID, requestingUserID)
			if err == nil {
				pbUser.IsFollowing = &isFollowing
			}
		}
	}

	return pbUser, nil
}

func (h *UserHandler) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.User, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	user, err := h.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	pbUser := &pb.User{
		Id:             user.ID.String(),
		Username:       user.Username,
		Email:          user.Email,
		CreatedAt:      timestamppb.New(user.CreatedAt),
		UpdatedAt:      timestamppb.New(user.UpdatedAt),
		FollowersCount: user.FollowersCount,
		FollowingCount: user.FollowingCount,
		PostsCount:     user.PostsCount,
	}

	if user.Bio != nil {
		pbUser.Bio = user.Bio
	}

	if req.RequestingUserId != nil && *req.RequestingUserId != "" && *req.RequestingUserId != req.UserId {
		requestingUserID, err := uuid.Parse(*req.RequestingUserId)
		if err == nil {
			isFollowing, err := h.repo.CheckFollowStatus(ctx, userID, requestingUserID)
			if err == nil {
				pbUser.IsFollowing = &isFollowing
			}
		}
	}

	return pbUser, nil
}

func (h *UserHandler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.User, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	if req.Username == nil && req.Email == nil && req.Bio == nil {
		return nil, status.Error(codes.InvalidArgument, "at least one field must be provided")
	}

	if req.Username != nil {
		if len(*req.Username) < 3 || len(*req.Username) > 30 {
			return nil, status.Error(codes.InvalidArgument, "username must be between 3 and 30 characters")
		}

		existingUser, err := h.repo.GetByUsername(ctx, *req.Username)
		if err == nil && existingUser.ID != userID {
			return nil, status.Error(codes.AlreadyExists, "username already taken")
		}
	}

	if req.Email != nil {
		existingUser, err := h.repo.GetByEmail(ctx, *req.Email)
		if err == nil && existingUser.ID != userID {
			return nil, status.Error(codes.AlreadyExists, "email already taken")
		}
	}

	updateInput := &models.UpdateUserInput{
		Username: req.Username,
		Email:    req.Email,
		Bio:      req.Bio,
	}

	user, err := h.repo.Update(ctx, userID, updateInput)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update profile: %v", err))
	}

	pbUser := &pb.User{
		Id:             user.ID.String(),
		Username:       user.Username,
		Email:          user.Email,
		CreatedAt:      timestamppb.New(user.CreatedAt),
		UpdatedAt:      timestamppb.New(user.UpdatedAt),
		FollowersCount: user.FollowersCount,
		FollowingCount: user.FollowingCount,
		PostsCount:     user.PostsCount,
	}

	if user.Bio != nil {
		pbUser.Bio = user.Bio
	}

	return pbUser, nil
}

func (h *UserHandler) GetUsersByIds(ctx context.Context, req *pb.GetUsersByIdsRequest) (*pb.GetUsersByIdsResponse, error) {
	if len(req.UserIds) == 0 {
		return &pb.GetUsersByIdsResponse{Users: []*pb.User{}}, nil
	}

	userIDs := make([]uuid.UUID, 0, len(req.UserIds))
	for _, idStr := range req.UserIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid user_id format: %s", idStr))
		}
		userIDs = append(userIDs, id)
	}

	users, err := h.repo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get users")
	}

	pbUsers := make([]*pb.User, 0, len(users))
	for _, user := range users {
		pbUser := &pb.User{
			Id:             user.ID.String(),
			Username:       user.Username,
			Email:          user.Email,
			CreatedAt:      timestamppb.New(user.CreatedAt),
			UpdatedAt:      timestamppb.New(user.UpdatedAt),
			FollowersCount: user.FollowersCount,
			FollowingCount: user.FollowingCount,
			PostsCount:     user.PostsCount,
		}

		if user.Bio != nil {
			pbUser.Bio = user.Bio
		}

		if req.RequestingUserId != nil && *req.RequestingUserId != "" {
			requestingUserID, err := uuid.Parse(*req.RequestingUserId)
			if err == nil && requestingUserID != user.ID {
				isFollowing, err := h.repo.CheckFollowStatus(ctx, user.ID, requestingUserID)
				if err == nil {
					pbUser.IsFollowing = &isFollowing
				}
			}
		}

		pbUsers = append(pbUsers, pbUser)
	}

	return &pb.GetUsersByIdsResponse{Users: pbUsers}, nil
}

func (h *UserHandler) IncrementPostsCount(ctx context.Context, req *pb.IncrementPostsCountRequest) (*pb.Response, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	err = h.repo.IncrementPostsCount(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to increment posts count")
	}

	return &pb.Response{
		Success: true,
		Message: "posts count incremented successfully",
	}, nil
}

func (h *UserHandler) DecrementPostsCount(ctx context.Context, req *pb.DecrementPostsCountRequest) (*pb.Response, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	err = h.repo.DecrementPostsCount(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to decrement posts count")
	}

	return &pb.Response{
		Success: true,
		Message: "posts count decremented successfully",
	}, nil
}
