package handler

import (
	"context"
	"log"
	"time"

	"comment-service/events"
	"comment-service/model"
	pb "comment-service/pb"
	"comment-service/publisher"
	"comment-service/repository"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CommentHandler struct {
	pb.UnimplementedCommentServiceServer
	repo      repository.CommentRepository
	publisher *publisher.EventPublisher
}

func NewCommentHandler(repo repository.CommentRepository, pub *publisher.EventPublisher) *CommentHandler {
	return &CommentHandler{
		repo:      repo,
		publisher: pub,
	}
}

// CreateComment handles the creation of a new comment
func (h *CommentHandler) CreateComment(ctx context.Context, req *pb.CreateCommentRequest) (*pb.Comment, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}
	if len(req.Content) > 2000 {
		return nil, status.Error(codes.InvalidArgument, "content must be less than 2000 characters")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	now := time.Now()
	comment := &models.Comment{
		ID:        uuid.New(),
		PostID:    postID,
		UserID:    userID,
		Content:   req.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	event := events.CommentAddedEvent{
		CommentID:  comment.ID,
		PostID:     comment.PostID,
		PostUserID: comment.UserID,
		Content:    comment.Content,
		CreatedAt:  time.Now(),
	}

	if err := h.publisher.PublishCommentAdded(event); err != nil {
		log.Printf("Failed to publish post created event: %v", err)
	}

	if err := h.repo.Create(ctx, comment); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create comment: %v", err)
	}

	return commentToProto(comment), nil
}

// GetPostComments retrieves comments for a specific post with pagination
func (h *CommentHandler) GetPostComments(ctx context.Context, req *pb.GetPostCommentsRequest) (*pb.CommentConnection, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	first := req.First
	if first <= 0 {
		first = 10
	}

	connection, err := h.repo.GetPostComments(ctx, postID, first, req.After)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get comments: %v", err)
	}

	return commentConnectionToProto(connection), nil
}

// UpdateComment updates an existing comment
func (h *CommentHandler) UpdateComment(ctx context.Context, req *pb.UpdateCommentRequest) (*pb.Comment, error) {
	if req.CommentId == "" {
		return nil, status.Error(codes.InvalidArgument, "comment_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}
	if len(req.Content) > 2000 {
		return nil, status.Error(codes.InvalidArgument, "content must be less than 2000 characters")
	}

	commentID, err := uuid.Parse(req.CommentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid comment_id format")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	exists, err := h.repo.CheckOwnership(ctx, commentID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify ownership: %v", err)
	}
	if !exists {
		return nil, status.Error(codes.PermissionDenied, "you don't have permission to update this comment")
	}

	comment, err := h.repo.GetByID(ctx, commentID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "comment not found")
	}

	comment.Content = req.Content
	comment.UpdatedAt = time.Now()

	if err := h.repo.Update(ctx, comment); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update comment: %v", err)
	}

	return commentToProto(comment), nil
}

// DeleteComment removes a comment
func (h *CommentHandler) DeleteComment(ctx context.Context, req *pb.DeleteCommentRequest) (*pb.Response, error) {
	if req.CommentId == "" {
		return nil, status.Error(codes.InvalidArgument, "comment_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	commentID, err := uuid.Parse(req.CommentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid comment_id format")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	exists, err := h.repo.CheckOwnership(ctx, commentID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify ownership: %v", err)
	}
	if !exists {
		return nil, status.Error(codes.PermissionDenied, "you don't have permission to delete this comment")
	}

	if err := h.repo.Delete(ctx, commentID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete comment: %v", err)
	}

	return &pb.Response{
		Success: true,
		Message: "comment deleted successfully",
	}, nil
}

// Helper functions for proto conversion

func commentToProto(c *models.Comment) *pb.Comment {
	return &pb.Comment{
		Id:        c.ID.String(),
		PostId:    c.PostID.String(),
		UserId:    c.UserID.String(),
		Content:   c.Content,
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
}

func commentConnectionToProto(conn *models.CommentConnection) *pb.CommentConnection {
	edges := make([]*pb.CommentEdge, len(conn.Edges))
	for i, edge := range conn.Edges {
		edges[i] = &pb.CommentEdge{
			Cursor: edge.Cursor,
			Node:   commentToProto(&edge.Node),
		}
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     conn.PageInfo.HasNextPage,
		HasPreviousPage: conn.PageInfo.HasPreviousPage,
	}

	if conn.PageInfo.StartCursor != nil {
		pageInfo.StartCursor = conn.PageInfo.StartCursor
	}
	if conn.PageInfo.EndCursor != nil {
		pageInfo.EndCursor = conn.PageInfo.EndCursor
	}

	return &pb.CommentConnection{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: conn.TotalCount,
	}
}
