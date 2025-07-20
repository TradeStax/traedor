#!/bin/bash

# Traedor Docker Setup Script
set -e

echo "🚀 Setting up Traedor Trading System"
echo "===================================="

# Check prerequisites
echo "📋 Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose not found. Please install Docker Compose first."
    exit 1
fi

echo "✅ Docker and Docker Compose found"

# Check Docker is running
if ! docker info &> /dev/null; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

echo "✅ Docker is running"

# Create environment file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating environment file from template..."
    cp .env.example .env
    echo "✅ Created .env file. You can customize it if needed."
fi

# Create data directory
echo "📁 Setting up data directory..."
mkdir -p data
echo "✅ Data directory created at ./data"

# Build and start services
echo "🔨 Building Docker images..."
docker-compose build

echo "🚀 Starting services..."
docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
sleep 15

# Health checks
echo "🏥 Performing health checks..."

# Check backend
if curl -f http://localhost:8080/api/v1/health > /dev/null 2>&1; then
    echo "✅ Backend is healthy"
else
    echo "⚠️  Backend health check failed, but it might still be starting..."
fi

# Check frontend
if curl -f http://localhost:3000 > /dev/null 2>&1; then
    echo "✅ Frontend is healthy"
else
    echo "⚠️  Frontend health check failed, but it might still be starting..."
fi

# Check database
if docker-compose exec -T postgres pg_isready -U traedor > /dev/null 2>&1; then
    echo "✅ Database is ready"
else
    echo "❌ Database is not ready"
fi

echo ""
echo "🎉 Traedor setup complete!"
echo "========================="
echo ""
echo "📍 Access points:"
echo "   Frontend:  http://localhost:3000"
echo "   API:       http://localhost:8080/api/v1"
echo "   Database:  localhost:5432 (user: traedor, password: traedor_password)"
echo ""
echo "📚 Quick commands:"
echo "   make logs      - View all logs"
echo "   make down      - Stop all services"
echo "   make restart   - Restart all services"
echo "   make health    - Check service health"
echo "   make clean     - Clean up everything"
echo ""
echo "📖 For more information, see DOCKER_SETUP.md"
echo ""

# Show running containers
echo "🐳 Running containers:"
docker-compose ps

echo ""
echo "💡 Tip: Place your trading data files in the ./data directory"
echo "💡 Tip: Customize configuration in config.docker.yaml"