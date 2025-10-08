package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"api-gateway/graph"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/v2/ast"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Initialize resolver with gRPC clients and NATS
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resolver, err := graph.NewResolver(ctx)
	if err != nil {
		log.Fatalf("failed to initialize resolver: %v", err)
	}

	// --- Create GraphQL server ---
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// Add transports
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	// WebSocket transport for subscriptions
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	})

	// Query cache
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	// Add extensions
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100)})

	// --- HTTP handlers ---
	http.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	http.Handle("/query", srv)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("🚀 Server running at http://localhost:%s/", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
