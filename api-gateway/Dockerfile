# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy all services for replace paths
COPY ./auth-service ./auth-service
COPY ./user-service ./user-service
COPY ./post-service ./post-service
COPY ./comment-service ./comment-service
COPY ./like-service ./like-service
COPY ./follow-service ./follow-service
COPY ./feed-service ./feed-service
COPY ./notification-service ./notification-service

# Copy API Gateway dependencies
COPY ./api-gateway/go.mod ./api-gateway/go.sum ./api-gateway/

WORKDIR /app/api-gateway
RUN go mod download

# Copy API Gateway source code
COPY ./api-gateway/ ./

# Build
RUN go build -o server .

# Runtime stage
FROM alpine:3.18
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/api-gateway/server .
CMD ["./server"]