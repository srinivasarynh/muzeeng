package handler

import (
	"context"
	"fmt"
	"time"

	models "notification-service/model"
	pb "notification-service/pb"
	"notification-service/repository"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NotificationHandler struct {
	pb.UnimplementedNotificationServiceServer
	repo repository.NotificationRepository
}

func NewNotificationHandler(repo repository.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{
		repo: repo,
	}
}

func (h *NotificationHandler) GetNotifications(ctx context.Context, req *pb.GetNotificationsRequest) (*pb.NotificationConnection, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	first := req.First
	if first <= 0 {
		first = 10
	}
	if first > 100 {
		first = 100
	}

	connection, err := h.repo.GetByUserID(ctx, userID, int(first), req.After)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get notifications: %v", err))
	}

	return modelConnectionToProto(connection), nil
}

func (h *NotificationHandler) MarkRead(ctx context.Context, req *pb.MarkReadRequest) (*pb.Response, error) {
	notificationID, err := uuid.Parse(req.NotificationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid notification_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	err = h.repo.MarkAsRead(ctx, notificationID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mark notification as read: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Notification marked as read",
	}, nil
}

func (h *NotificationHandler) MarkAllRead(ctx context.Context, req *pb.MarkAllReadRequest) (*pb.Response, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	err = h.repo.MarkAllAsRead(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mark all notifications as read: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "All notifications marked as read",
	}, nil
}

func (h *NotificationHandler) CreateNotification(ctx context.Context, req *pb.CreateNotificationRequest) (*pb.Notification, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if req.Type == pb.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "notification type must be specified")
	}

	var actorID *uuid.UUID
	if req.ActorId != nil && *req.ActorId != "" {
		parsed, err := uuid.Parse(*req.ActorId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid actor_id")
		}
		actorID = &parsed
	}

	var relatedID *uuid.UUID
	if req.RelatedId != nil && *req.RelatedId != "" {
		parsed, err := uuid.Parse(*req.RelatedId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid related_id")
		}
		relatedID = &parsed
	}

	notification := &models.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      protoTypeToModel(req.Type),
		Message:   req.Message,
		ActorID:   actorID,
		RelatedID: relatedID,
		IsRead:    false,
		CreatedAt: time.Now().UTC(),
	}

	err = h.repo.Create(ctx, notification)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create notification: %v", err))
	}

	return modelNotificationToProto(notification), nil
}

func (h *NotificationHandler) DeleteNotification(ctx context.Context, req *pb.DeleteNotificationRequest) (*pb.Response, error) {
	notificationID, err := uuid.Parse(req.NotificationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid notification_id")
	}

	err = h.repo.Delete(ctx, notificationID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete notification: %v", err))
	}

	return &pb.Response{
		Success: true,
		Message: "Notification deleted successfully",
	}, nil
}

// Helper functions for model to proto conversion
func modelNotificationToProto(n *models.Notification) *pb.Notification {
	notification := &pb.Notification{
		Id:        n.ID.String(),
		UserId:    n.UserID.String(),
		Type:      modelTypeToProto(n.Type),
		Message:   n.Message,
		IsRead:    n.IsRead,
		CreatedAt: timestamppb.New(n.CreatedAt),
	}

	if n.ActorID != nil {
		actorID := n.ActorID.String()
		notification.ActorId = &actorID
	}

	if n.RelatedID != nil {
		relatedID := n.RelatedID.String()
		notification.RelatedId = &relatedID
	}

	return notification
}

func modelConnectionToProto(c *models.NotificationConnection) *pb.NotificationConnection {
	edges := make([]*pb.NotificationEdge, len(c.Edges))
	for i, edge := range c.Edges {
		edges[i] = &pb.NotificationEdge{
			Cursor: edge.Cursor,
			Node:   modelNotificationToProto(&edge.Node),
		}
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     c.PageInfo.HasNextPage,
		HasPreviousPage: c.PageInfo.HasPreviousPage,
	}

	if c.PageInfo.EndCursor != nil {
		pageInfo.EndCursor = c.PageInfo.EndCursor
	}

	if c.PageInfo.StartCursor != nil {
		pageInfo.StartCursor = c.PageInfo.StartCursor
	}

	return &pb.NotificationConnection{
		Edges:       edges,
		PageInfo:    pageInfo,
		TotalCount:  c.TotalCount,
		UnreadCount: c.UnreadCount,
	}
}

func modelTypeToProto(t models.NotificationType) pb.NotificationType {
	switch t {
	case models.NotificationTypePost:
		return pb.NotificationType_POST
	case models.NotificationTypeComment:
		return pb.NotificationType_COMMENT
	default:
		return pb.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED
	}
}

func protoTypeToModel(t pb.NotificationType) models.NotificationType {
	switch t {
	case pb.NotificationType_POST:
		return models.NotificationTypePost
	case pb.NotificationType_COMMENT:
		return models.NotificationTypeComment
	default:
		return ""
	}
}
