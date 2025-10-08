package interceptor

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ContextKey type for context keys
type ContextKey string

const (
	UserIDKey ContextKey = "user_id"
)

// AuthInterceptor provides gRPC interceptor for JWT authentication
type AuthInterceptor struct {
	jwtSecret     string
	publicMethods map[string]bool
}

// NewAuthInterceptor creates a new auth interceptor with public methods
func NewAuthInterceptor(jwtSecret string, publicMethods []string) *AuthInterceptor {
	methodMap := make(map[string]bool)
	for _, method := range publicMethods {
		methodMap[method] = true
	}

	return &AuthInterceptor{
		jwtSecret:     jwtSecret,
		publicMethods: methodMap,
	}
}

// AddPublicMethod adds a method that doesn't require authentication
func (interceptor *AuthInterceptor) AddPublicMethod(method string) {
	interceptor.publicMethods[method] = true
}

// AddPublicMethods adds multiple methods that don't require authentication
func (interceptor *AuthInterceptor) AddPublicMethods(methods []string) {
	for _, method := range methods {
		interceptor.publicMethods[method] = true
	}
}

// Unary returns a server interceptor function to authenticate and authorize unary RPC
func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if interceptor.publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		userID, err := interceptor.authorize(ctx)
		if err != nil {
			return nil, err
		}

		ctx = context.WithValue(ctx, UserIDKey, userID)

		return handler(ctx, req)
	}
}

// Stream returns a server interceptor function to authenticate and authorize stream RPC
func (interceptor *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if interceptor.publicMethods[info.FullMethod] {
			return handler(srv, stream)
		}

		userID, err := interceptor.authorize(stream.Context())
		if err != nil {
			return err
		}

		ctx := context.WithValue(stream.Context(), UserIDKey, userID)
		wrappedStream := &wrappedStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// authorize verifies the JWT token and returns the user ID
func (interceptor *AuthInterceptor) authorize(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "authorization token is not provided")
	}

	token := values[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format")
	}
	token = strings.TrimPrefix(token, "Bearer ")

	claims, err := interceptor.verifyToken(token)
	if err != nil {
		return "", status.Error(codes.Unauthenticated, fmt.Sprintf("invalid token: %v", err))
	}

	return claims.UserID, nil
}

// verifyToken verifies the JWT token and extracts claims
func (interceptor *AuthInterceptor) verifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(interceptor.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// wrappedStream wraps grpc.ServerStream with a custom context
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return "", fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}
