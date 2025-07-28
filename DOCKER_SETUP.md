# Traedor Docker Setup

This guide will help you run the complete Traedor trading system using Docker Compose.

## 🚀 Quick Start

### Prerequisites
- Docker Engine 20.10+ 
- Docker Compose 2.0+
- At least 4GB of available RAM
- 2GB of free disk space

### One-Command Setup
```bash
make setup
```

This will build all images, start all services, and perform health checks.

## 📋 Services Overview

The Docker Compose setup includes:

| Service   | Port | Description                    |
|-----------|------|--------------------------------|
| Frontend  | 3000 | React/Next.js web interface   |
| Backend   | 8080 | Go API server                  |
| Postgres  | 5432 | PostgreSQL database            |
| Redis     | 6379 | Cache (optional, for future)  |

## 🛠️ Available Commands

### Basic Operations
```bash
# Start all services
make up

# Stop all services  
make down

# Restart all services
make restart

# View logs
make logs

# Follow logs in real-time
make logs-f

# Check service health
make health
```

### Development
```bash
# Start only database (for local Go development)
make db-only

# Build only backend
make backend

# Build only frontend
make frontend

# Run local development
make dev-backend  # Starts DB, run Go locally
make dev-frontend # Runs frontend in dev mode
```

### Maintenance
```bash
# Clean up all Docker resources
make clean

# Access database shell
make db-shell

# Run migrations manually
make db-migrate
```

## 🔧 Configuration

### Environment Variables
Copy and customize the environment file:
```bash
cp .env.example .env
# Edit .env with your preferences
```

### Data Directory
Place your trading data files in the `./data` directory:
```
data/
├── MESH23_FUT_CME.txt        # Main price data
├── MESH23_FUT_CME-1Min.txt   # 1-minute study data
└── other_symbols/            # Additional data files
```

### Custom Configuration
Modify `config.docker.yaml` for:
- Database connection strings
- API settings
- Trading parameters
- Strategy configurations

## 🌐 Access Points

Once running, access the application at:

- **Web Interface**: http://localhost:3000
- **API Documentation**: http://localhost:8080/api/v1/health
- **Database**: localhost:5432 (user: traedor, password: traedor_password)

## 📊 Using the System

### 1. Web Interface
Navigate to http://localhost:3000 to:
- View backtest run history
- Create new backtests
- Manage trading signals
- Analyze performance metrics

### 2. API Access
Use the REST API directly:
```bash
# Health check
curl http://localhost:8080/api/v1/health

# List runs
curl http://localhost:8080/api/v1/runs

# Create a new backtest
curl -X POST http://localhost:8080/api/v1/runs \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "symbol": "/MES",
      "timeframe": "1m",
      "start_time": "2023-01-01T00:00:00Z",
      "end_time": "2023-12-31T23:59:59Z",
      "datafeeds": [{
        "type": "SC",
        "symbol": "/MES", 
        "data_path": "./data/MESH23_FUT_CME.txt",
        "interval": "0ns"
      }],
      "broker": {
        "type": "Futures",
        "starting_balance": 10000,
        "trailing_stop_amount": 10,
        "fee_per_side": 2.5,
        "open_slippage": 0.25,
        "symbol": {
          "name": "/MES",
          "margin": 1200,
          "point_price": 5
        }
      },
      "strategies": [{
        "type": "SC",
        "symbol": "/MES",
        "params": {
          "data_path": "./data/MESH23_FUT_CME-1Min.txt",
          "values": ["12B"]
        }
      }],
      "signals": []
    }
  }'
```

## 🐛 Troubleshooting

### Common Issues

#### Services Won't Start
```bash
# Check Docker status
docker --version
docker-compose --version

# Check logs for errors
make logs

# Restart everything
make clean && make setup
```

#### Database Connection Issues
```bash
# Check PostgreSQL status
docker-compose exec postgres pg_isready -U traedor

# View database logs
docker-compose logs postgres

# Reset database
docker-compose down -v
docker-compose up -d postgres
```

#### Frontend Build Errors
```bash
# Rebuild frontend
make frontend

# Check Node.js version in container
docker-compose exec frontend node --version
```

#### API Not Responding
```bash
# Check backend logs
docker-compose logs backend

# Verify Go build
docker-compose exec backend ./traedor --help

# Test database connectivity
docker-compose exec backend nc -zv postgres 5432
```

### Performance Issues

#### Low Memory
- Increase Docker memory limit to 4GB+
- Monitor with `docker stats`

#### Slow Database
- Ensure data directory has sufficient space
- Check PostgreSQL logs for slow queries

#### Network Issues
- Verify no port conflicts (3000, 8080, 5432)
- Check firewall settings

## 🔒 Security Considerations

### Production Deployment
For production use, consider:

1. **Change Default Passwords**
   ```bash
   # Update .env file with secure passwords
   POSTGRES_PASSWORD=your_secure_password
   ```

2. **Use TLS/SSL**
   - Enable HTTPS for frontend
   - Use SSL for database connections

3. **Network Security**
   - Don't expose database port publicly
   - Use a reverse proxy (nginx/traefik)

4. **Resource Limits**
   ```yaml
   # Add to docker-compose.yml
   deploy:
     resources:
       limits:
         memory: 512M
         cpus: '0.5'
   ```

## 📈 Monitoring

### Health Checks
All services include health checks:
```bash
# Check all services
make health

# Individual service status
docker-compose ps
```

### Logs
```bash
# All logs
make logs

# Specific service
docker-compose logs backend
docker-compose logs frontend
docker-compose logs postgres
```

### Resource Usage
```bash
# Monitor resource usage
docker stats

# Disk usage
docker system df
```

## 🔄 Updates and Maintenance

### Updating the Application
```bash
# Pull latest changes
git pull

# Rebuild and restart
make clean && make setup
```

### Database Backups
```bash
# Backup database
docker-compose exec postgres pg_dump -U traedor traedor > backup.sql

# Restore database
docker-compose exec -T postgres psql -U traedor traedor < backup.sql
```

### Log Rotation
```bash
# Clear logs (be careful!)
docker-compose down
docker system prune -f
make up
```

## 📞 Support

If you encounter issues:

1. Check this troubleshooting guide
2. Review service logs: `make logs`
3. Verify configuration files
4. Ensure data files are in correct location
5. Check Docker and system resources

For development questions, refer to the main project documentation and API endpoints.