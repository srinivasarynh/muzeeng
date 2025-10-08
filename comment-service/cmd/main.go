package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"comment-service/config"
	"comment-service/db"
	"comment-service/handler"
	"comment-service/interceptor"
	natsClient "comment-service/nats"
	pb "comment-service/pb"
	"comment-service/publisher"
	"comment-service/repository"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("failed to load Comment .env file")
	}
	// Load database configuration
	dbCfg, err := config.LoadDatabaseConfig("")
	if err != nil {
		log.Fatalf("Failed to load Comment database config: %v", err)
	}

	// Connect to the database
	dbConn, err := database.NewConnection(database.Config{
		Host:         dbCfg.Host,
		Port:         dbCfg.Port,
		User:         dbCfg.User,
		Password:     dbCfg.Password,
		DBName:       dbCfg.DBName,
		SSLMode:      dbCfg.SSLMode,
		MaxOpenConns: dbCfg.MaxOpenConns,
		MaxIdleConns: dbCfg.MaxIdleConns,
		MaxLifetime:  dbCfg.MaxLifetime,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Comment database: %v", err)
	}
	defer dbConn.Close()

	log.Println("Successfully connected to Comment database")

	// Load other service-level configs
	grpcPort := getEnv("GRPC_PORT", "50056")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")
	natsURL := getEnv("NATS_URL", "nats://nats:4222")
	natsClientID := getEnv("NATS_CLIENT_ID", "comment-service")

	// Initialize NATS client
	natsCfg := natsClient.Config{
		URL:           natsURL,
		MaxReconnects: 10,
		ReconnectWait: 2 * time.Second,
		ClientID:      natsClientID,
	}

	nats, err := natsClient.NewClient(natsCfg)
	if err != nil {
		log.Fatalf("Failed to initialize NATS client: %v", err)
	}
	defer nats.Close()
	log.Println("NATS client initialized successfully")

	// Initialize event publisher
	eventPublisher := publisher.NewEventPublisher(nats)

	// Initialize repository and handler
	commentRepo := repository.NewCommentRepository(dbConn.DB)
	commentHandler := handler.NewCommentHandler(commentRepo, eventPublisher)

	// Initialize auth interceptor (allowing public routes)
	authInterceptor := interceptor.NewAuthInterceptor(jwtSecret, []string{
		"/comment.CommentService/GetPostComments",
		"/comment.CommentService/GetComment",
	})

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.Unary()),
		grpc.StreamInterceptor(authInterceptor.Stream()),
	)

	// Graceful shutdown handling
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Comment server Shutting down gracefully...")
		grpcServer.GracefulStop()

		ctx, cancel := context.WithTimeout(context.Background(), dbCfg.MaxLifetime)
		defer cancel()

		if err := dbConn.HealthCheck(ctx); err == nil {
			_ = dbConn.Close()
			log.Println("Comment Database connection closed")
		}

		log.Println("Server stopped")
		os.Exit(0)
	}()

	// Register the gRPC service
	pb.RegisterCommentServiceServer(grpcServer, commentHandler)

	// Enable reflection for debugging tools like grpcurl
	reflection.Register(grpcServer)

	// Start listening for connections
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	log.Printf("Comment Service gRPC server listening on port %s", grpcPort)

	// Serve requests
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// small helper for optional env vars
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
