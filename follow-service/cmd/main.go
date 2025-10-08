package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"follow-service/config"
	"follow-service/db"
	"follow-service/handler"
	"follow-service/interceptor"
	pb "follow-service/pb"
	"follow-service/repository"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("failed to load Follow .env")
	}
	// Load database configuration
	dbCfg, err := config.LoadDatabaseConfig("")
	if err != nil {
		log.Fatalf("Failed to load Follow database config: %v", err)
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
		log.Fatalf("Failed to connect to Follow database: %v", err)
	}
	defer dbConn.Close()

	log.Println("Successfully connected to database")

	// Load other service-level configs
	grpcPort := getEnv("GRPC_PORT", "50055")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")

	// Initialize repository and handler
	followRepo := repository.NewFollowRepository(dbConn.DB)
	followHandler := handler.NewFollowHandler(followRepo)

	// Initialize auth interceptor (allowing public routes)
	authInterceptor := interceptor.NewAuthInterceptor(jwtSecret, []string{
		"/follow.FollowService/GetFollowers",
		"/follow.FollowService/GetFollowing",
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

		log.Println("Follow service Shutting down gracefully...")
		grpcServer.GracefulStop()

		ctx, cancel := context.WithTimeout(context.Background(), dbCfg.MaxLifetime)
		defer cancel()

		if err := dbConn.HealthCheck(ctx); err == nil {
			_ = dbConn.Close()
			log.Println("Follow Database connection closed")
		}

		log.Println("Server stopped")
		os.Exit(0)
	}()

	// Register the gRPC service
	pb.RegisterFollowServiceServer(grpcServer, followHandler)

	// Enable reflection for debugging tools like grpcurl
	reflection.Register(grpcServer)

	// Start listening for connections
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	log.Printf("Follow Service gRPC server listening on port %s", grpcPort)

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
