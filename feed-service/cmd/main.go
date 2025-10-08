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
	"google.golang.org/grpc/reflection"

	"feed-service/config"
	"feed-service/db"
	"feed-service/handler"
	"feed-service/interceptor"
	pb "feed-service/pb"
	"feed-service/repository"
)

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		log.Println("failed to load Comment .env file")
	}

	// Load database configuration
	dbCfg, err := config.LoadDatabaseConfig("")
	if err != nil {
		log.Fatalf("Failed to load Feed database config: %v", err)
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
		log.Fatalf("Failed to connect to Feed database: %v", err)
	}
	defer dbConn.Close()
	log.Println("Feed Database connected successfully")

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

	// Load service port
	grpcPort := getEnv("PORT", "50054")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")

	// Initialize repository and handler
	feedRepo := repository.NewFeedRepository(dbConn.DB, redisClient)
	feedHandler := handler.NewFeedHandler(feedRepo)

	// Initialize auth interceptor (allowing public routes)
	authInterceptor := interceptor.NewAuthInterceptor(jwtSecret, []string{})

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.Unary()),
	)

	// Register the FeedService
	pb.RegisterFeedServiceServer(grpcServer, feedHandler)

	// Enable reflection (for grpcurl/testing)
	reflection.Register(grpcServer)

	// Start listening
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	// Background cleanup job
	go startBackgroundJobs(feedRepo)

	// Start gRPC server in a goroutine
	go func() {
		log.Printf("Feed Service gRPC server listening on port %s", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Feed Service...")
	grpcServer.GracefulStop()
	redisClient.Close()
	dbConn.Close()
	log.Println("Feed Service stopped cleanly")
}

// --- Background Jobs ---

func startBackgroundJobs(feedRepo repository.FeedRepository) {
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer cleanupTicker.Stop()

	for range cleanupTicker.C {
		ctx := context.Background()
		olderThan := time.Now().AddDate(0, 0, -30) // keep last 30 days

		log.Println("🧺 Starting feed cleanup job...")
		if err := feedRepo.CleanupOldFeedItems(ctx, olderThan); err != nil {
			log.Printf("Feed cleanup failed: %v", err)
		} else {
			log.Println("Feed cleanup completed successfully")
		}
	}
}

// --- Interceptors ---

func loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)

	status := "✅"
	if err != nil {
		status = "❌"
	}

	log.Printf("[%s] %s - %v (took %v)", status, info.FullMethod, err, duration)
	return resp, err
}

// --- Utility helpers ---

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var intVal int
		fmt.Sscanf(val, "%d", &intVal)
		return intVal
	}
	return defaultVal
}
