# Ad Tracking System

A high-performance, scalable ad tracking system built with Go, PostgreSQL, and Redis. Features real-time analytics, comprehensive monitoring, and production-ready architecture.

## 🚀 Features

- **High Performance**: Handles concurrent requests with async processing
- **Real-time Analytics**: Live click tracking and performance metrics
- **Data Integrity**: No data loss with retry mechanisms and queue processing
- **Scalability**: Designed for high-traffic scenarios
- **Monitoring**: Prometheus metrics and Grafana dashboards
- **Production Ready**: Docker containerization and comprehensive logging

## 📋 API Endpoints

### GET /api/v1/ads
Returns a list of active ads with metadata.

**Response:**
```json
{
  "ads": [
    {
      "id": 1,
      "image_url": "https://example.com/ad1.jpg",
      "target_url": "https://example.com/product1",
      "title": "Amazing Product 1",
      "active": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### POST /api/v1/ads/click
Records a click event for an ad.

**Request:**
```json
{
  "ad_id": 1,
  "timestamp": 1704067200,
  "video_playback_time": 30
}
```

**Response:**
```json
{
  "status": "recorded"
}
```

### GET /api/v1/ads/analytics
Returns analytics data for ads.

**Query Parameters:**
- `ad_id` (optional): Specific ad ID
- `timeframe` (optional): `1h`, `24h`, `7d` (default: `24h`)

**Response:**
```json
{
  "analytics": [
    {
      "ad_id": 1,
      "click_count": 150,
      "last_hour": 12,
      "last_day": 150
    }
  ]
}
```

## 🛠️ Quick Start

### Prerequisites
- Docker and Docker Compose
- Go 1.21+ (for local development)

### 1. Clone and Setup
```bash
git clone <repository-url>
cd ad-tracking-system
make setup
```

### 2. Start All Services
```bash
make start
```

### 3. Test the API
```bash
# Get ads
curl http://localhost:8080/api/v1/ads

# Record a click
curl -X POST http://localhost:8080/api/v1/ads/click \
  -H "Content-Type: application/json" \
  -d '{"ad_id": 1, "video_playback_time": 30}'

# Get analytics
curl http://localhost:8080/api/v1/ads/analytics
```

## 🔧 Development

### Local Development
```bash
# Run in development mode
make dev

# Run with auto-reload
go run . --mode=debug
```

### Database Operations
```bash
# Run migrations
make db-migrate

# Seed test data
make db-seed
```

### Testing
```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run load tests
make load-test
```

## 📊 Monitoring

Access the monitoring stack:

- **Application**: http://localhost:8080
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)
- **pgAdmin**: http://localhost:5050 (admin@example.com/admin)

### Key Metrics
- `ad_clicks_received_total`: Total clicks received
- `ad_clicks_processed_total`: Total clicks processed
- `http_request_duration_seconds`: Request latency
- `click_queue_size`: Queue size for async processing

## 🏗️ Architecture

### Components
- **API Server**: Gin-based HTTP server
- **Click Queue**: Async processing with batching
- **PostgreSQL**: Primary data store
- **Redis**: Caching and session storage
- **Prometheus**: Metrics collection
- **Grafana**: Visualization and alerting

### Data Flow
1. Client sends click event to API
2. API validates and responds immediately
3. Click event queued for async processing
4. Batch processor writes to database
5. Analytics queries aggregated data

## 🚀 Production Deployment

### Docker
```bash
# Build image
make docker-build

# Run container
make docker-run
```

### Environment Variables
```bash
# Database
DATABASE_URL=postgresql://user:pass@host:5432/db

# Server
PORT=8080
GIN_MODE=release
LOG_LEVEL=info

# Optional
REDIS_URL=redis://localhost:6379
```

### Kubernetes
```bash
# Deploy to Kubernetes
kubectl apply -f deployments/k8s/
```

## 🔒 Security

- Input validation on all endpoints
- Rate limiting (configurable)
- CORS protection
- SQL injection prevention with GORM
- Environment-based configuration

## 📝 Configuration

### Database Connection
The system supports both local PostgreSQL and external services:

```bash
# Local PostgreSQL
DATABASE_URL=postgresql://ad_user:ad_password@localhost:5432/ad_tracking_db?sslmode=disable

# External service (like Neon)
DATABASE_URL=postgresql://user:pass@host/db?sslmode=require
```

### Performance Tuning
- Connection pooling: 100 max connections
- Batch processing: 100 events per batch
- Queue size: 10,000 events buffer
- Retry logic: 3 attempts with backoff

## 🛠️ Available Commands

```bash
make help           # Show all available commands
make setup          # Setup local development environment
make start          # Quick start (setup + compose-up)
make stop           # Stop all services
make compose-up     # Start services with docker-compose
make compose-down   # Stop services
make compose-logs   # View logs
make test           # Run tests
make build          # Build application
make clean          # Clean up resources
```

## 🐛 Troubleshooting

### Common Issues

1. **Database Connection Issues**
   ```bash
   # Check database status
   docker-compose ps postgres
   
   # View database logs
   docker-compose logs postgres
   ```

2. **Application Not Starting**
   ```bash
   # Check application logs
   docker-compose logs app
   
   # Verify environment variables
   docker-compose exec app env
   ```

3. **High Memory Usage**
   ```bash
   # Check queue size
   curl http://localhost:8080/metrics | grep queue_size
   
   # Monitor system resources
   docker stats
   ```

### Performance Optimization
- Increase queue buffer size for high traffic
- Adjust batch processing parameters
- Add database indexes for analytics queries
- Configure connection pooling

## 📚 Documentation

- [API Documentation](docs/api.md)
- [Architecture Guide](docs/architecture.md)
- [Deployment Guide](docs/deployment.md)
- [Troubleshooting](docs/troubleshooting.md)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄