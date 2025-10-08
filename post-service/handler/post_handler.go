package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"post-service/events"
	"post-service/model"
	pb "post-service/pb"
	"post-service/publisher"
	"post-service/repository"
)

type PostHandler struct {
	pb.UnimplementedPostServiceServer
	repo      repository.PostRepository
	publisher *publisher.EventPublisher
}

func NewPostHandler(repo repository.PostRepository, pub *publisher.EventPublisher) *PostHandler {
	return &PostHandler{
		repo:      repo,
		publisher: pub,
	}
}

func (h *PostHandler) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.Post, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	now := time.Now()
	post := &models.Post{
		ID:            uuid.New(),
		UserID:        userID,
		Content:       req.Content,
		CreatedAt:     now,
		UpdatedAt:     now,
		LikesCount:    0,
		CommentsCount: 0,
	}

	event := events.PostCreatedEvent{
		PostID:    post.ID,
		UserID:    post.UserID,
		Content:   post.Content,
		CreatedAt: time.Now(),
	}

	if err := h.publisher.PublishPostCreated(event); err != nil {
		log.Printf("Failed to publish post created event: %v", err)
	}

	if err := h.repo.Create(ctx, post); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create post: %v", err))
	}

	return postToProto(post, nil), nil
}

func (h *PostHandler) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.Post, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	var requestingUserID *uuid.UUID
	if req.RequestingUserId != nil && *req.RequestingUserId != "" {
		id, err := uuid.Parse(*req.RequestingUserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid requesting_user_id format")
		}
		requestingUserID = &id
	}

	post, err := h.repo.GetByID(ctx, postID, requestingUserID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("post not found: %v", err))
	}

	return postWithLikeStatusToProto(post), nil
}

func (h *PostHandler) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.Post, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	existingPost, err := h.repo.GetByID(ctx, postID, nil)
	if err != nil {
		return nil, status.Error(codes.NotFound, "post not found")
	}

	if existingPost.Post.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "you can only update your own posts")
	}

	post := &models.Post{
		ID:            postID,
		UserID:        userID,
		Content:       req.Content,
		UpdatedAt:     time.Now(),
		CreatedAt:     existingPost.Post.CreatedAt,
		LikesCount:    existingPost.Post.LikesCount,
		CommentsCount: existingPost.Post.CommentsCount,
	}

	if err := h.repo.Update(ctx, post); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update post: %v", err))
	}

	return postToProto(post, nil), nil
}

func (h *PostHandler) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*pb.Response, error) {
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

	existingPost, err := h.repo.GetByID(ctx, postID, nil)
	if err != nil {
		return nil, status.Error(codes.NotFound, "post not found")
	}

	if existingPost.Post.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "you can only delete your own posts")
	}

	if err := h.repo.Delete(ctx, postID); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete post: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Post deleted successfully",
	}, nil
}

func (h *PostHandler) GetUserPosts(ctx context.Context, req *pb.GetUserPostsRequest) (*pb.PostConnection, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	var requestingUserID *uuid.UUID
	if req.RequestingUserId != nil && *req.RequestingUserId != "" {
		id, err := uuid.Parse(*req.RequestingUserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid requesting_user_id format")
		}
		requestingUserID = &id
	}

	first := req.First
	if first <= 0 {
		first = 10
	}
	if first > 100 {
		first = 100
	}

	connection, err := h.repo.GetUserPosts(ctx, userID, first, req.After, requestingUserID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get user posts: %v", err))
	}

	return connectionToProto(connection, requestingUserID), nil
}

func (h *PostHandler) IncrementCommentsCount(ctx context.Context, req *pb.IncrementCommentsCountRequest) (*pb.Response, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	if err := h.repo.IncrementCommentsCount(ctx, postID); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to increment comments count: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Comments count incremented successfully",
	}, nil
}

func (h *PostHandler) DecrementCommentsCount(ctx context.Context, req *pb.DecrementCommentsCountRequest) (*pb.Response, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	if err := h.repo.DecrementCommentsCount(ctx, postID); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to decrement comments count: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Comments count decremented successfully",
	}, nil
}

func (h *PostHandler) IncrementLikesCount(ctx context.Context, req *pb.IncrementLikesCountRequest) (*pb.Response, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	if err := h.repo.IncrementLikesCount(ctx, postID); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to increment likes count: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Likes count incremented successfully",
	}, nil
}

func (h *PostHandler) DecrementLikesCount(ctx context.Context, req *pb.DecrementLikesCountRequest) (*pb.Response, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "post_id is required")
	}

	postID, err := uuid.Parse(req.PostId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid post_id format")
	}

	if err := h.repo.DecrementLikesCount(ctx, postID); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to decrement likes count: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Likes count decremented successfully",
	}, nil
}

// Helper functions to convert between models and proto

func postToProto(post *models.Post, isLiked *bool) *pb.Post {
	return &pb.Post{
		Id:            post.ID.String(),
		UserId:        post.UserID.String(),
		Content:       post.Content,
		CreatedAt:     timestamppb.New(post.CreatedAt),
		UpdatedAt:     timestamppb.New(post.UpdatedAt),
		LikesCount:    post.LikesCount,
		CommentsCount: post.CommentsCount,
		IsLiked:       isLiked,
	}
}

func postWithLikeStatusToProto(post *models.PostWithLikeStatus) *pb.Post {
	return &pb.Post{
		Id:            post.Post.ID.String(),
		UserId:        post.Post.UserID.String(),
		Content:       post.Post.Content,
		CreatedAt:     timestamppb.New(post.Post.CreatedAt),
		UpdatedAt:     timestamppb.New(post.Post.UpdatedAt),
		LikesCount:    post.Post.LikesCount,
		CommentsCount: post.Post.CommentsCount,
		IsLiked:       post.IsLiked,
	}
}

func connectionToProto(conn *models.PostConnection, requestingUserID *uuid.UUID) *pb.PostConnection {
	edges := make([]*pb.PostEdge, len(conn.Edges))
	for i, edge := range conn.Edges {
		edges[i] = &pb.PostEdge{
			Cursor: edge.Cursor,
			Node:   postToProto(&edge.Node, nil),
		}
	}

	return &pb.PostConnection{
		Edges: edges,
		PageInfo: &pb.PageInfo{
			EndCursor:       conn.PageInfo.EndCursor,
			HasNextPage:     conn.PageInfo.HasNextPage,
			StartCursor:     conn.PageInfo.StartCursor,
			HasPreviousPage: conn.PageInfo.HasPreviousPage,
		},
		TotalCount: conn.TotalCount,
	}
}
