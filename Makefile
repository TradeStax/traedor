.PHONY: help build up down restart logs clean test

# Default target
help:
	@echo "Available commands:"
	@echo "  build     - Build all Docker images"
	@echo "  up        - Start all services"
	@echo "  down      - Stop all services"
	@echo "  restart   - Restart all services"
	@echo "  logs      - Show logs for all services"
	@echo "  logs-f    - Follow logs for all services"
	@echo "  clean     - Clean up Docker resources"
	@echo "  test      - Run tests"
	@echo "  backend   - Build only backend"
	@echo "  frontend  - Build only frontend"
	@echo "  db-only   - Start only database"

# Build all images
build:
	docker-compose build

# Start all services
up:
	docker-compose up -d

# Stop all services
down:
	docker-compose down

# Restart all services
restart: down up

# Show logs
logs:
	docker-compose logs

# Follow logs
logs-f:
	docker-compose logs -f

# Clean up
clean:
	docker-compose down -v --rmi all --remove-orphans
	docker system prune -f

# Run tests (when implemented)
test:
	docker-compose exec backend go test ./...

# Build only backend
backend:
	docker-compose build backend

# Build only frontend
frontend:
	docker-compose build frontend

# Start only database (useful for development)
db-only:
	docker-compose up -d postgres

# Database management
db-migrate:
	docker-compose exec postgres psql -U traedor -d traedor -f /docker-entrypoint-initdb.d/001_initial_schema.sql

db-shell:
	docker-compose exec postgres psql -U traedor -d traedor

# Development helpers
dev-backend:
	docker-compose up -d postgres
	@echo "Database started. You can now run 'go run . --api' locally"

dev-frontend:
	cd frontend && npm run dev

# Development with hot reload
dev:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml up

dev-build:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml build

dev-down:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml down

# Health checks
health:
	@echo "Checking service health..."
	@curl -f http://localhost:8080/api/v1/health > /dev/null 2>&1 && echo "✅ Backend healthy" || echo "❌ Backend unhealthy"
	@curl -f http://localhost:3000 > /dev/null 2>&1 && echo "✅ Frontend healthy" || echo "❌ Frontend unhealthy"

# Quick setup
setup: build up
	@echo "Waiting for services to start..."
	@sleep 10
	@make health
	@echo ""
	@echo "🚀 Traedor is ready!"
	@echo "Frontend: http://localhost:3000"
	@echo "Backend API: http://localhost:8080/api/v1"
	@echo "Database: localhost:5432"