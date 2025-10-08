# **Project Documentation**

## **Overview**

This project implements a **distributed social media platform** architecture using **Golang gRPC microservices** and a **GraphQL API Gateway**.  
It supports **queries, mutations, and subscriptions**, with **JWT authentication** and **role-based access control**.

## **Architecture**

### **High-Level Design**

               ┌────────────────────────┐  
               │     Client (Web/Mobile)│  
               └────────────┬───────────┘  
                            │ GraphQL over HTTP/WebSocket  
                            ▼  
                  ┌───────────────────────┐  
                  │  GraphQL API Gateway  │  
                  │ (gRPC Resolver Layer) │  
                  └──────────┬────────────┘  
         ┌────────────────┼────────────────┐  
         ▼                ▼                ▼  
 ┌────────────┐       ┌────────────┐        ┌────────────┐  
 │ auth-svc   │       │ post-svc   │        │ user-svc   │  
 └────────────┘       └────────────┘        └────────────┘  
         ▼                   ▼                    ▼  
 ┌────────────┐       ┌────────────┐        ┌────────────┐  
 │ follow-svc │       │ like-svc   │        │ comment-svc│  
 └────────────┘       └────────────┘        └────────────┘  
         ▼                   ▼                    ▼  
 ┌────────────┐       ┌────────────┐  
 │ feed-svc   │       │ notification-svc │  
 └────────────┘       └───── ──────┘

Each service is independent, communicates over **gRPC**, and exposes well-defined protobuf contracts.

# **Golang gRPC Microservices with GraphQL API Gateway**

### **Overview**

This project is a **Go-based GraphQL API Gateway** that integrates **8 gRPC microservices** into a unified, secure, and real-time API layer.  
It acts as the **entry point** for clients, handling authentication, authorization, data federation, and subscriptions using **WebSockets**.

Acts as the single entry point for all clients.

Uses **GraphQL** to aggregate and orchestrate data from multiple gRPC services.

Includes:

* JWT authentication (`@auth`)  
* Role-based access (`@hasRole`)  
* Subscriptions via WebSocket (e.g., new posts, comments, notifications)  
* Pagination and connection-based data retrieval

## **Architecture**

### **System Components**

| Service | Description |
| ----- | ----- |
| **Auth-service** | Handles user registration, login, JWT generation, and token refresh. |
| **User-service** | Manages user profiles, bios, and profile updates. |
| **Post-service** | Handles post creation, update, deletion, and retrieval. |
| **Comment-service** | Manages comments on posts, including CRUD operations. |
| **Like-service** | Handles like/unlike actions and retrieves like-related info. |
| **Follow-service** | Manages following/unfollowing and follower relationships. |
| **Notification-service** | Sends and tracks notifications for likes, follows, and comments. |
| **Feed-service** | Generates personalized user feeds from posts and following data. |

All services communicate over **gRPC** with **Protocol Buffers (protobuf)** as the schema definition format.

**Technology Stack**

| Layer | Technology |
| ----- | ----- |
| **Language** | Go (Golang) |
| **GraphQL Layer** | gqlgen |
| **Transport** | gRPC |
| **Authentication** | JWT (access & refresh tokens) |
| **Subscriptions** | WebSocket (GraphQL Subscriptions) |
| **Schema Management** | `schema.graphql` |
| **Containerization** | Docker / Docker Compose |
| **Persistence** | PostgreSQL / Redis (depending on service) |
|  |  |

**Project Structure**

/root/  
  ├── graphql-api-gateway/  
  ├── auth-service/  
  ├── user-service/  
  ├── post-service/  
  ├── comment-service/  
  ├── like-service/  
  ├── follow-service/  
  ├── feed-service/  
  ├── notification-service/  
  ├── ...

## 

## 

## **Authentication & Authorization**

* **JWT Authentication**  
  All protected queries and mutations are annotated with `@auth`.  
  The gateway validates JWT tokens before executing resolvers.  
* **Role-Based Access Control (RBAC)**  
  Certain operations use `@hasRole(roles: [USER,ADMIN])` for role-specific access.  
* **Token Handling**  
  * `register`, `login`, `refreshToken` — public  
  * `logout`, `updateProfile`, etc. — require valid JWT

## **GraphQL Schema**

Defined in `schema.graphql`.  
Key types include:

* **Query** — Data retrieval (posts, users, feeds, comments, etc.)  
* **Mutation** — State-changing actions (like, follow, post creation)  
* **Subscription** — Real-time updates for new posts, comments, and notifications.

Example Subscription Flow:  
subscription {  
  notificationAdded {  
    id  
    message  
    type  
  }  
}

Triggered whenever a new notification is published via gRPC → Pub/Sub → Gateway → Client.

## **gRPC Integration**

Each resolver delegates requests to the corresponding microservice via a gRPC client.  
Example (pseudo):

func (r \*mutationResolver) CreatePost(ctx context.Context, input CreatePostInput) (\*Post, error) {  
    userId := auth.GetUserIDFromContext(ctx)  
    resp, err := r.grpcClients.PostService.CreatePost(ctx, \&postpb.CreatePostRequest{  
        UserId:  userId,  
        Content: input.Content,  
    })  
    if err \!= nil {  
        return nil, err  
    }  
    return mapPostFromProto(resp.Post), nil  
}

Each gRPC client is initialized on startup and injected into the GraphQL resolvers.

## **Subscriptions**

* Powered by **WebSockets** with **gqlgen subscriptions**.  
* Events are pushed from microservices via a **Pub/Sub layer** (Redis, NATS, or Kafka).  
* Supported subscription events:  
  * `notificationAdded`  
  * `postAdded`  
  * `commentAdded`

## **Health Check**

**Query:**

{  
  healthCheck {  
    status  
    timestamp  
    services {  
      name  
      status  
      latency  
    }  
  }  
}

**Example Queries**

mutation {  
  createPost(input: { content: "My first post\!" }) {  
    id  
    content  
    createdAt  
  }  
}

## **Development Setup**

### **Prerequisites**

* Go 1.22+  
* Docker & Docker Compose  
* Protobuf compiler (`protoc`)  
* gqlgen (`go install github.com/99designs/gqlgen@latest`)

# **Project Setup & Commands**

## **1\. Prerequisites**

Before starting, ensure you have installed:

* **Go** (\>= 1.21)  
* **Docker & Docker Compose**  
* **Protobuf Compiler (`protoc`)**  
* **Golang gRPC & GraphQL Tools**  
* **NATS server**  
* **Optional: Postman / GraphQL Playground** for testing GraphQL APIs

## **Setup Steps**

### Step 1: Clone the repository
git clone https://github.com/srinivasarynh/muzeeng
cd muzeeng

### Step 2: Install Go dependencies inside each service
go mod tidy

### Step 3: Generate gRPC code from proto files inside each service
protoc --go_out=pb --go-grpc_out=pb proto/*.proto

### Step 4: Generate GraphQL types & resolvers
cd api-gateway

gqlgen generate

### Step 5: Build Docker images and Running the Project using BASH script file at root folder
./run-all.sh   

### Step 6: Kill Docker Container and Stop the Project
docker-compose down -v

### **Option 2: Run services locally (for development)**

\# Start auth service  
cd auth-service  
go run [main.go](http://main.go)

\# Start post service  
cd post-service  
go run main.go

\# Start GraphQL gateway  
cd api-gateway  
go run server.go

**Common Development Commands**

| Command | Description |
| ----- | ----- |
| `go mod tidy` | Update dependencies |
| `protoc --go_out` | Generate gRPC Go code |
| `gqlgen generate` | Generate GraphQL types/resolvers |
| `docker-compose up --build` | Build & start all services |
| `docker-compose down` | Stop all services |
| `go run main.go` | Run a single service locally |
|  |  |

