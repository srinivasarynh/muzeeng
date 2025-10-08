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

	"post-service/config"
	"post-service/db"
	"post-service/handler"
	"post-service/interceptor"
	natsClient "post-service/nats"
	pb "post-service/pb"
	"post-service/publisher"

	"post-service/repository"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("failed to load Post .env")
	}
	// Load database configuration
	dbCfg, err := config.LoadDatabaseConfig("")
	if err != nil {
		log.Fatalf("Failed to load Post database config: %v", err)
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
		log.Fatalf("Failed to connect to Post database: %v", err)
	}
	defer dbConn.Close()

	log.Println("Successfully connected to Post database")

	// Load other service-level configs
	grpcPort := getEnv("GRPC_PORT", "50053")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")
	natsURL := getEnv("NATS_URL", "nats://nats:4222")
	natsClientID := getEnv("NATS_CLIENT_ID", "post-service")

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
	postRepo := repository.NewPostRepository(dbConn.DB)
	postHandler := handler.NewPostHandler(postRepo, eventPublisher)

	// Initialize auth interceptor (allowing public routes)
	authInterceptor := interceptor.NewAuthInterceptor(jwtSecret, []string{
		"/post.PostService/GetPost",
		"/post.PostService/GetUserPosts",
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

		log.Println("Post service Shutting down gracefully...")
		grpcServer.GracefulStop()

		ctx, cancel := context.WithTimeout(context.Background(), dbCfg.MaxLifetime)
		defer cancel()

		if err := dbConn.HealthCheck(ctx); err == nil {
			_ = dbConn.Close()
			log.Println("Post Database connection closed")
		}

		log.Println("Server stopped")
		os.Exit(0)
	}()

	// Register the gRPC service
	pb.RegisterPostServiceServer(grpcServer, postHandler)

	// Enable reflection for debugging tools like grpcurl
	reflection.Register(grpcServer)

	// Start listening for connections
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	log.Printf("Post Service gRPC server listening on port %s", grpcPort)

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
