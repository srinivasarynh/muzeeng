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
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	"notification-service/config"
	"notification-service/db"
	"notification-service/handler"
	natsClient "notification-service/nats"
	pb "notification-service/pb"
	"notification-service/repository"
	"notification-service/subscriber"
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Println("failed to load Notification .env")
	}

	// Load database configuration
	dbCfg, err := config.LoadDatabaseConfig("NOTIFICATION_")
	if err != nil {
		log.Fatalf("Failed to load Notification database config: %v", err)
	}

	// Create database connection
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
		log.Fatalf("Failed to connect to Notification database: %v", err)
	}
	defer dbConn.Close()
	log.Println("Notification Database connected successfully")

	// Load other configurations
	grpcPort := getEnv("GRPC_PORT", "50058")
	natsURL := getEnv("NATS_URL", "nats://nats:4222")
	natsClientID := getEnv("NATS_CLIENT_ID", "notification-service")

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

	// Load Redis configuration
	redisURL := getEnv("REDIS_URL", "redis:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvAsInt("REDIS_DB", 0)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: redisPassword,
		DB:       redisDB,
		PoolSize: 10,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Feed Redis: %v", err)
	}
	log.Println("Feed Redis connected successfully")

	// Initialize repository
	repo := repository.NewNotificationRepository(dbConn.DB, redisClient)

	// Initialize gRPC handler
	grpcHandler := handler.NewNotificationHandler(repo)

	// Initialize NATS subscriber
	sub := subscriber.NewNotificationSubscriber(nats, repo, ctx)
	if err := sub.Start(); err != nil {
		log.Fatalf("Failed to start NATS subscriber: %v", err)
	}

	// Start gRPC server in a separate goroutine
	go func() {
		if err := startGRPCServer(grpcPort, grpcHandler); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	log.Printf("Notification Service started")
	log.Printf("gRPC listening on port %s", grpcPort)
	log.Printf("NATS subscriber active")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Notification Shutting down Notification Service...")
	sub.Stop()
	nats.Close()
	dbConn.Close()
	log.Println("Notification Service stopped cleanly")
}

// startGRPCServer starts the notification gRPC server
func startGRPCServer(port string, handler *handler.NotificationHandler) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024), // 10MB
		grpc.MaxSendMsgSize(10*1024*1024), // 10MB
	)

	pb.RegisterNotificationServiceServer(grpcServer, handler)

	log.Printf("gRPC server starting on port %s", port)
	return grpcServer.Serve(lis)
}

// helper for non-database environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var intVal int
		fmt.Sscanf(val, "%d", &intVal)
		return intVal
	}
	return defaultVal
}
