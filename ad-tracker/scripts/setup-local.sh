#!/bin/bash

# Local Development Setup Script for Ad Tracking System

set -e

echo "🚀 Setting up Ad Tracking System for local development..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if Docker Compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null 2>&1; then
    print_error "Docker Compose is not available. Please install Docker Compose and try again."
    exit 1
fi

# Create necessary directories
print_status "Creating project directories..."
mkdir -p scripts monitoring/grafana/provisioning/dashboards monitoring/grafana/provisioning/datasources

# Copy environment file
if [ ! -f .env ]; then
    print_status "Creating .env file from template..."
    cp .env.example .env
    print_warning "Please review and update the .env file with your configuration."
else
    print_status ".env file already exists."
fi

# Create monitoring directory structure
print_status "Setting up monitoring configuration..."

# Create Prometheus config if it doesn't exist
if [ ! -f monitoring/prometheus.yml ]; then
    cat > monitoring/prometheus.yml << EOF
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'ad-tracking-system'
    static_configs:
      - targets: ['app:8080']
    metrics_path: /metrics
    scrape_interval: 5s

  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
EOF
fi

# Create Grafana datasource config
if [ ! -f monitoring/grafana/provisioning/datasources/prometheus.yml ]; then
    cat > monitoring/grafana/provisioning/datasources/prometheus.yml << EOF
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
EOF
fi

# Start the services
print_status "Starting services with Docker Compose..."
docker-compose up -d postgres redis

# Wait for PostgreSQL to be ready
print_status "Waiting for PostgreSQL to be ready..."
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if docker-compose exec -T postgres pg_isready -U ad_user -d ad_tracking_db > /dev/null 2>&1; then
        print_status "PostgreSQL is ready!"
        break
    fi
    attempt=$((attempt + 1))
    echo -n "."
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    print_error "PostgreSQL failed to start within expected time."
    exit 1
fi

# Build and start the application
print_status "Building and starting the application..."
docker-compose up -d --build

# Wait for application to be ready
print_status "Waiting for application to be ready..."
sleep 10

# Check if services are running
print_status "Checking service status..."
if docker-compose ps | grep -q "Up"; then
    print_status "Services are running successfully!"
else
    print_error "Some services failed to start. Check the logs with 'docker-compose logs'"
    exit 1
fi

# Display service URLs
echo
echo "🎉 Setup complete! Services are running at:"
echo "┌─────────────────────────────────────────────────────┐"
echo "│ Service         │ URL                               │"
echo "├─────────────────────────────────────────────────────┤"
echo "│ Ad Tracking API │ http://localhost:8080             │"
echo "│ Health Check    │ http://localhost:8080/health      │"
echo "│ Metrics         │ http://localhost:8080/metrics     │"
echo "│ PostgreSQL      │ localhost:5432                    │"
echo "│ Redis           │ localhost:6379                    │"
echo "│ Prometheus      │ http://localhost:9090             │"
echo "│ Grafana         │ http://localhost:3000             │"
echo "│ pgAdmin         │ http://localhost:5050             │"
echo "└─────────────────────────────────────────────────────┘"
echo
echo "Default credentials:"
echo "- Grafana: admin/admin"
echo "- pgAdmin: admin@example.com/admin"
echo "- PostgreSQL: ad_user/ad_password"
echo
echo "📋 Quick commands:"
echo "- View logs: docker-compose logs -f"
echo "- Stop services: docker-compose down"
echo "- Restart services: docker-compose restart"
echo "- View API docs: curl http://localhost:8080/health"
echo
echo "🧪 Test the API:"
echo "- Get ads: curl http://localhost:8080/api/v1/ads"
echo "- Post click: curl -X POST http://localhost:8080/api/v1/ads/click -H 'Content-Type: application/json' -d '{\"ad_id\": 1}'"
echo "- Get analytics: curl http://localhost:8080/api/v1/ads/analytics"