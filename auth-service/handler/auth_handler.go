package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"auth-service/model"
	pb "auth-service/pb"
	"auth-service/pkg/jwt"
	"auth-service/repository"
)

type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	repo          repository.AuthRepository
	jwtManager    *jwt.Manager
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewAuthHandler(repo repository.AuthRepository, jwtManager *jwt.Manager, accessExpiry, refreshExpiry time.Duration) *AuthHandler {
	return &AuthHandler{
		repo:          repo,
		jwtManager:    jwtManager,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username, email, and password are required")
	}

	existingUser, _ := h.repo.GetUserByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
	}

	existingUser, _ = h.repo.GetUserByUsername(ctx, req.Username)
	if existingUser != nil {
		return nil, status.Error(codes.AlreadyExists, "username already taken")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	now := time.Now()
	user := &models.User{
		ID:             uuid.New(),
		Username:       req.Username,
		Email:          req.Email,
		PasswordHash:   string(hashedPassword),
		CreatedAt:      now,
		UpdatedAt:      now,
		FollowersCount: 0,
		FollowingCount: 0,
		PostsCount:     0,
	}

	if req.Bio != nil {
		user.Bio = req.Bio
	}

	if err := h.repo.CreateUser(ctx, user); err != nil {
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	userRole := &models.UserRole{
		ID:        uuid.New(),
		UserID:    user.ID,
		Role:      models.RoleUser,
		CreatedAt: now,
	}
	if err := h.repo.CreateUserRole(ctx, userRole); err != nil {
		return nil, status.Error(codes.Internal, "failed to create user role")
	}

	roles := []string{string(models.RoleUser)}
	accessToken, err := h.jwtManager.Generate(user.ID.String(), roles, h.accessExpiry)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID.String(), h.refreshExpiry)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate refresh token")
	}

	refreshTokenModel := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: now.Add(h.refreshExpiry),
		CreatedAt: now,
		IsRevoked: false,
	}
	if err := h.repo.CreateRefreshToken(ctx, refreshTokenModel); err != nil {
		return nil, status.Error(codes.Internal, "failed to store refresh token")
	}

	return &pb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         convertUserToProto(user),
		ExpiresIn:    int32(h.accessExpiry.Seconds()),
		Message:      "Registration successful",
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	roles, err := h.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user roles")
	}

	rolesStr := make([]string, len(roles))
	for i, role := range roles {
		rolesStr[i] = string(role)
	}

	accessToken, err := h.jwtManager.Generate(user.ID.String(), rolesStr, h.accessExpiry)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID.String(), h.refreshExpiry)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate refresh token")
	}

	now := time.Now()
	refreshTokenModel := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: now.Add(h.refreshExpiry),
		CreatedAt: now,
		IsRevoked: false,
	}
	if err := h.repo.CreateRefreshToken(ctx, refreshTokenModel); err != nil {
		return nil, status.Error(codes.Internal, "failed to store refresh token")
	}

	return &pb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         convertUserToProto(user),
		ExpiresIn:    int32(h.accessExpiry.Seconds()),
		Message:      "Login successful",
	}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.AuthResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	claims, err := h.jwtManager.Verify(req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	_, err = h.repo.GetRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "refresh token not found or expired")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid user ID")
	}

	user, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	roles, err := h.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user roles")
	}

	rolesStr := make([]string, len(roles))
	for i, role := range roles {
		rolesStr[i] = string(role)
	}

	accessToken, err := h.jwtManager.Generate(user.ID.String(), rolesStr, h.accessExpiry)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	newRefreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID.String(), h.refreshExpiry)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate refresh token")
	}

	if err := h.repo.RevokeRefreshToken(ctx, req.RefreshToken); err != nil {
		return nil, status.Error(codes.Internal, "failed to revoke old refresh token")
	}

	now := time.Now()
	newRefreshTokenModel := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     newRefreshToken,
		ExpiresAt: now.Add(h.refreshExpiry),
		CreatedAt: now,
		IsRevoked: false,
	}
	if err := h.repo.CreateRefreshToken(ctx, newRefreshTokenModel); err != nil {
		return nil, status.Error(codes.Internal, "failed to store refresh token")
	}

	return &pb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		User:         convertUserToProto(user),
		ExpiresIn:    int32(h.accessExpiry.Seconds()),
		Message:      "Token refreshed successfully",
	}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.Response, error) {
	if req.UserId == "" || req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and access_token are required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	claims, err := h.jwtManager.Verify(req.AccessToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid access token")
	}

	blacklisted, err := h.repo.IsTokenBlacklisted(ctx, req.AccessToken)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check token blacklist")
	}

	if !blacklisted {
		if claims.ExpiresAt != nil {
			expiresAt := claims.ExpiresAt.Time
			if err := h.repo.AddTokenToBlacklist(ctx, req.AccessToken, expiresAt); err != nil {
				return nil, status.Error(codes.Internal, "failed to blacklist token")
			}
		} else {
			// fallback if ExpiresAt is missing (optional)
			if err := h.repo.AddTokenToBlacklist(ctx, req.AccessToken, time.Now().Add(time.Hour)); err != nil {
				return nil, status.Error(codes.Internal, "failed to blacklist token")
			}
		}
	}

	if err := h.repo.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
		return nil, status.Error(codes.Internal, "failed to revoke refresh tokens")
	}

	return &pb.Response{
		Success: true,
		Message: "Logout successful",
	}, nil
}

func (h *AuthHandler) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.Response, error) {
	if req.UserId == "" || req.CurrentPassword == "" || req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, current_password, and new_password are required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	user, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "current password is incorrect")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash new password")
	}

	if err := h.repo.UpdateUserPassword(ctx, userID, string(hashedPassword)); err != nil {
		return nil, status.Error(codes.Internal, "failed to update password")
	}

	if err := h.repo.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
		return nil, status.Error(codes.Internal, "failed to revoke refresh tokens")
	}

	return &pb.Response{
		Success: true,
		Message: "Password changed successfully",
	}, nil
}

func (h *AuthHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: "token is required",
		}, nil
	}

	blacklisted, err := h.repo.IsTokenBlacklisted(ctx, req.Token)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check token blacklist")
	}

	if blacklisted {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: "token has been revoked",
		}, nil
	}

	claims, err := h.jwtManager.Verify(req.Token)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: fmt.Sprintf("invalid token: %v", err),
		}, nil
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: "invalid user ID in token",
		}, nil
	}

	_, err = h.repo.GetUserByID(ctx, userID)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: "user not found",
		}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:   true,
		UserId:  claims.UserID,
		Roles:   claims.Roles,
		Message: "token is valid",
	}, nil
}

// Helper function to convert models.User to pb.User
func convertUserToProto(user *models.User) *pb.User {
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

	return pbUser
}
