#!/bin/bash
set -e

echo "🔹 Building Docker images..."
docker-compose build

echo "🔹 Starting services..."
docker-compose up -d

echo "⏳ Waiting for services to be healthy..."
# Wait for all services to be running
sleep 10

# Check if key services are up
echo "📊 Checking service status..."
docker-compose ps

echo ""
echo "🎉 Services started!"
echo "🌐 GraphQL Playground: http://localhost:8080/"
echo ""
echo "💡 Tips:"
echo "  - View logs: docker-compose logs -f"
echo "  - View specific service: docker-compose logs -f api-gateway"
echo "  - Stop all: docker-compose down"