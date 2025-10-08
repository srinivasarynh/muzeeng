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

	"auth-service/config"
	"auth-service/db"
	"auth-service/handler"
	pb "auth-service/pb"
	"auth-service/pkg/jwt"
	"auth-service/repository"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No Auth .env file found")
	}

	// Load Database Config
	dbConfig, err := config.LoadDatabaseConfig("AUTH_")
	if err != nil {
		log.Fatalf("Failed to load database config: %v", err)
	}

	db, err := database.NewConnection(database.Config{
		Host:         dbConfig.Host,
		Port:         dbConfig.Port,
		User:         dbConfig.User,
		Password:     dbConfig.Password,
		DBName:       dbConfig.DBName,
		SSLMode:      dbConfig.SSLMode,
		MaxOpenConns: dbConfig.MaxOpenConns,
		MaxIdleConns: dbConfig.MaxIdleConns,
		MaxLifetime:  dbConfig.MaxLifetime,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Auth-database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to Auth-database")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.HealthCheck(ctx); err != nil {
		log.Fatalf("Auth Database health check failed: %v", err)
	}
	log.Println("Auth Database health check passed")

	// JWT Setup
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	jwtManager := jwt.NewManager(jwtSecret)

	// Token expiration configs
	accessExpiry := getEnvAsDuration("ACCESS_TOKEN_EXPIRY", 15*time.Minute)
	refreshExpiry := getEnvAsDuration("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour)

	// Repository & Handler
	authRepo := repository.NewAuthRepository(db.DB)
	authHandler := handler.NewAuthHandler(authRepo, jwtManager, accessExpiry, refreshExpiry)

	// Start gRPC Server
	port := getEnv("GRPC_PORT", "50051")
	address := fmt.Sprintf(":%s", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	server := grpc.NewServer()
	pb.RegisterAuthServiceServer(server, authHandler)

	// Enable server reflection for debugging
	reflection.Register(server)

	// Graceful shutdown handling
	go func() {
		log.Printf("Auth Service running on port %s", port)
		if err := server.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gRPC Auth server...")
	server.GracefulStop()
	log.Println("Auth Server stopped cleanly")
}

// Helper functions
func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	dur, err := time.ParseDuration(val)
	if err != nil {
		log.Printf("Invalid duration for %s, using default %v", key, defaultVal)
		return defaultVal
	}
	return dur
}
