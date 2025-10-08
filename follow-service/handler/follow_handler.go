package handler

import (
	"context"
	"fmt"

	pb "follow-service/pb"
	"follow-service/repository"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FollowHandler struct {
	pb.UnimplementedFollowServiceServer
	repo repository.FollowRepository
}

func NewFollowHandler(repo repository.FollowRepository) *FollowHandler {
	return &FollowHandler{
		repo: repo,
	}
}

// FollowUser handles the FollowUser RPC
func (h *FollowHandler) FollowUser(ctx context.Context, req *pb.FollowUserRequest) (*pb.Response, error) {
	if req.FollowerId == "" {
		return nil, status.Error(codes.InvalidArgument, "follower_id is required")
	}
	if req.FollowingId == "" {
		return nil, status.Error(codes.InvalidArgument, "following_id is required")
	}

	followerID, err := uuid.Parse(req.FollowerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid follower_id format")
	}

	followingID, err := uuid.Parse(req.FollowingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid following_id format")
	}

	if followerID == followingID {
		return nil, status.Error(codes.InvalidArgument, "users cannot follow themselves")
	}

	if err := h.repo.FollowUser(ctx, followerID, followingID); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to follow user: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Successfully followed user",
	}, nil
}

// UnfollowUser handles the UnfollowUser RPC
func (h *FollowHandler) UnfollowUser(ctx context.Context, req *pb.UnfollowUserRequest) (*pb.Response, error) {
	if req.FollowerId == "" {
		return nil, status.Error(codes.InvalidArgument, "follower_id is required")
	}
	if req.FollowingId == "" {
		return nil, status.Error(codes.InvalidArgument, "following_id is required")
	}

	followerID, err := uuid.Parse(req.FollowerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid follower_id format")
	}

	followingID, err := uuid.Parse(req.FollowingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid following_id format")
	}

	if err := h.repo.UnfollowUser(ctx, followerID, followingID); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unfollow user: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Successfully unfollowed user",
	}, nil
}

// GetFollowers handles the GetFollowers RPC
func (h *FollowHandler) GetFollowers(ctx context.Context, req *pb.GetFollowersRequest) (*pb.FollowConnection, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	first := req.First
	if first <= 0 {
		first = 10
	}
	if first > 100 {
		first = 100
	}

	var after *string
	if req.After != nil {
		after = req.After
	}

	connection, err := h.repo.GetFollowers(ctx, userID, first, after)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get followers: %v", err))
	}

	pbEdges := make([]*pb.FollowEdge, len(connection.Edges))
	for i, edge := range connection.Edges {
		pbEdges[i] = &pb.FollowEdge{
			Cursor:     edge.Cursor,
			UserId:     edge.UserID.String(),
			FollowedAt: timestamppb.New(edge.FollowedAt),
		}
	}

	pbPageInfo := &pb.PageInfo{
		HasNextPage:     connection.PageInfo.HasNextPage,
		HasPreviousPage: connection.PageInfo.HasPreviousPage,
	}
	if connection.PageInfo.EndCursor != nil {
		pbPageInfo.EndCursor = connection.PageInfo.EndCursor
	}
	if connection.PageInfo.StartCursor != nil {
		pbPageInfo.StartCursor = connection.PageInfo.StartCursor
	}

	return &pb.FollowConnection{
		Edges:      pbEdges,
		PageInfo:   pbPageInfo,
		TotalCount: connection.TotalCount,
	}, nil
}

// GetFollowing handles the GetFollowing RPC
func (h *FollowHandler) GetFollowing(ctx context.Context, req *pb.GetFollowingRequest) (*pb.FollowConnection, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	first := req.First
	if first <= 0 {
		first = 10
	}
	if first > 100 {
		first = 100
	}

	var after *string
	if req.After != nil {
		after = req.After
	}

	connection, err := h.repo.GetFollowing(ctx, userID, first, after)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get following: %v", err))
	}

	pbEdges := make([]*pb.FollowEdge, len(connection.Edges))
	for i, edge := range connection.Edges {
		pbEdges[i] = &pb.FollowEdge{
			Cursor:     edge.Cursor,
			UserId:     edge.UserID.String(),
			FollowedAt: timestamppb.New(edge.FollowedAt),
		}
	}

	pbPageInfo := &pb.PageInfo{
		HasNextPage:     connection.PageInfo.HasNextPage,
		HasPreviousPage: connection.PageInfo.HasPreviousPage,
	}
	if connection.PageInfo.EndCursor != nil {
		pbPageInfo.EndCursor = connection.PageInfo.EndCursor
	}
	if connection.PageInfo.StartCursor != nil {
		pbPageInfo.StartCursor = connection.PageInfo.StartCursor
	}

	return &pb.FollowConnection{
		Edges:      pbEdges,
		PageInfo:   pbPageInfo,
		TotalCount: connection.TotalCount,
	}, nil
}

// IsFollowing handles the IsFollowing RPC
func (h *FollowHandler) IsFollowing(ctx context.Context, req *pb.IsFollowingRequest) (*pb.IsFollowingResponse, error) {
	if req.FollowerId == "" {
		return nil, status.Error(codes.InvalidArgument, "follower_id is required")
	}
	if req.FollowingId == "" {
		return nil, status.Error(codes.InvalidArgument, "following_id is required")
	}

	followerID, err := uuid.Parse(req.FollowerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid follower_id format")
	}

	followingID, err := uuid.Parse(req.FollowingId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid following_id format")
	}

	isFollowing, err := h.repo.IsFollowing(ctx, followerID, followingID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to check following status: %v", err))
	}

	return &pb.IsFollowingResponse{
		IsFollowing: isFollowing,
	}, nil
}

// GetFollowStatus handles the GetFollowStatus RPC
func (h *FollowHandler) GetFollowStatus(ctx context.Context, req *pb.GetFollowStatusRequest) (*pb.GetFollowStatusResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if len(req.TargetUserIds) == 0 {
		return &pb.GetFollowStatusResponse{Statuses: []*pb.FollowStatus{}}, nil
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	targetUserIDs := make([]uuid.UUID, len(req.TargetUserIds))
	for i, idStr := range req.TargetUserIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid target_user_id at index %d", i))
		}
		targetUserIDs[i] = id
	}

	statuses, err := h.repo.GetFollowStatus(ctx, userID, targetUserIDs)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get follow status: %v", err))
	}

	pbStatuses := make([]*pb.FollowStatus, len(statuses))
	for i, s := range statuses {
		pbStatuses[i] = &pb.FollowStatus{
			UserId:      s.UserID.String(),
			IsFollowing: s.IsFollowing,
		}
	}

	return &pb.GetFollowStatusResponse{
		Statuses: pbStatuses,
	}, nil
}

// GetFollowersCounts handles the GetFollowersCounts RPC
func (h *FollowHandler) GetFollowersCounts(ctx context.Context, req *pb.GetFollowersCountsRequest) (*pb.GetFollowersCountsResponse, error) {
	if len(req.UserIds) == 0 {
		return &pb.GetFollowersCountsResponse{Counts: []*pb.UserFollowCounts{}}, nil
	}

	userIDs := make([]uuid.UUID, len(req.UserIds))
	for i, idStr := range req.UserIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid user_id at index %d", i))
		}
		userIDs[i] = id
	}

	counts, err := h.repo.GetFollowersCounts(ctx, userIDs)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get follower counts: %v", err))
	}

	pbCounts := make([]*pb.UserFollowCounts, len(counts))
	for i, c := range counts {
		pbCounts[i] = &pb.UserFollowCounts{
			UserId:         c.UserID.String(),
			FollowersCount: c.FollowersCount,
			FollowingCount: c.FollowingCount,
		}
	}

	return &pb.GetFollowersCountsResponse{
		Counts: pbCounts,
	}, nil
}
