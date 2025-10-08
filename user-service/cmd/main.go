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

	"user-service/config"
	"user-service/db"
	"user-service/handler"
	"user-service/interceptor"
	pb "user-service/pb"
	"user-service/repository"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Failed to load User .env")
	}

	// Load database configuration using config package
	dbCfg, err := config.LoadDatabaseConfig("")
	if err != nil {
		log.Fatalf("Failed to load User database config: %v", err)
	}

	// Connect to database using database package
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
		log.Fatalf("Failed to connect to User database: %v", err)
	}
	defer dbConn.Close()

	log.Println("Successfully connected to User database")

	// Load other configuration (non-database)
	grpcPort := getEnv("GRPC_PORT", "50052")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")

	// Initialize repository and handler
	userRepo := repository.NewUserRepository(dbConn.DB)
	userHandler := handler.NewUserHandler(userRepo)

	// Setup auth interceptor
	authInterceptor := interceptor.NewAuthInterceptor(jwtSecret, []string{
		"/user.UserService/GetProfile",
		"/user.UserService/GetUsersByIds",
	})

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.Unary()),
	)

	// Register service
	pb.RegisterUserServiceServer(grpcServer, userHandler)

	// Enable reflection for debugging tools
	reflection.Register(grpcServer)

	// Start listening on the specified port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("User Service gRPC server listening on port %s", grpcPort)

	// Graceful shutdown handling
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("User service Shutting down gracefully...")
		grpcServer.GracefulStop()

		ctx, cancel := context.WithTimeout(context.Background(), dbCfg.MaxLifetime)
		defer cancel()

		if err := dbConn.HealthCheck(ctx); err == nil {
			_ = dbConn.Close()
			log.Println("User Database connection closed")
		}

		log.Println("Server stopped")
		os.Exit(0)
	}()

	// Serve gRPC
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// helper to read environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
