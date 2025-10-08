package handler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"feed-service/model"
	pb "feed-service/pb"
	"feed-service/repository"
)

type FeedHandler struct {
	pb.UnimplementedFeedServiceServer
	feedRepo repository.FeedRepository
}

func NewFeedHandler(feedRepo repository.FeedRepository) *FeedHandler {
	return &FeedHandler{
		feedRepo: feedRepo,
	}
}

// GetFeed retrieves the personalized feed for a user
func (h *FeedHandler) GetFeed(ctx context.Context, req *pb.GetFeedRequest) (*pb.PostConnection, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	limit := req.First
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var after *string
	if req.After != nil {
		after = req.After
	}

	feedConnection, err := h.feedRepo.GetFeed(ctx, userID, int(limit), after)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get feed: %v", err)
	}

	postIDs := make([]uuid.UUID, len(feedConnection.Edges))
	for i, edge := range feedConnection.Edges {
		postIDs[i] = edge.Node.ID
	}

	likeStatus, err := h.feedRepo.GetPostsWithLikeStatus(ctx, userID, postIDs)
	if err != nil {
		fmt.Printf("Failed to get like status: %v\n", err)
		likeStatus = make(map[uuid.UUID]bool)
	}

	return h.toProtoPostConnection(feedConnection, likeStatus), nil
}

// Helper function to convert models.PostConnection to protobuf PostConnection
func (h *FeedHandler) toProtoPostConnection(conn *models.PostConnection, likeStatus map[uuid.UUID]bool) *pb.PostConnection {
	edges := make([]*pb.PostEdge, len(conn.Edges))

	for i, edge := range conn.Edges {
		isLiked := likeStatus[edge.Node.ID]

		edges[i] = &pb.PostEdge{
			Cursor: edge.Cursor,
			Node: &pb.Post{
				Id:            edge.Node.ID.String(),
				UserId:        edge.Node.UserID.String(),
				Content:       edge.Node.Content,
				CreatedAt:     timestamppb.New(edge.Node.CreatedAt),
				UpdatedAt:     timestamppb.New(edge.Node.UpdatedAt),
				LikesCount:    edge.Node.LikesCount,
				CommentsCount: edge.Node.CommentsCount,
				IsLiked:       &isLiked,
			},
		}
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     conn.PageInfo.HasNextPage,
		HasPreviousPage: conn.PageInfo.HasPreviousPage,
	}

	if conn.PageInfo.EndCursor != nil {
		pageInfo.EndCursor = conn.PageInfo.EndCursor
	}
	if conn.PageInfo.StartCursor != nil {
		pageInfo.StartCursor = conn.PageInfo.StartCursor
	}

	return &pb.PostConnection{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: conn.TotalCount,
	}
}
