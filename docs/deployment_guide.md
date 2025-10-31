# Deployment Guide - Reservas Calculator

## Application Architecture Overview

**Deployment Philosophy:** Simple, focused utility application rather than complex microservices architecture. The reserves calculator is a specialized financial calculation tool best deployed as a standalone application or command-line utility.

## Application Types

### 1. Command-Line Utility (Recommended)
**Purpose:** Batch processing and scheduled calculations
**Use Case:** End-of-day/monthly reserve calculations, regulatory reporting
**Deployment:** Single binary, no external dependencies
**Advantages:**
- Simple deployment and maintenance
- No security surface area
- Easy automation with cron/systemd
- Reliable for scheduled regulatory requirements

### 2. TUI Application (Optional)
**Purpose:** Interactive reserve calculations and data exploration
**Use Case:** Actuarial analysis, manual calculations
**Deployment:** Terminal-based interface
**Advantages:**
- Interactive data exploration
- Real-time calculation feedback
- No external dependencies
- Suitable for actuarial teams

### 3. HTTP API (Limited Scope)
**Purpose:** Integration with existing insurance systems
**Use Case:** Real-time reserve inquiries from policy systems
**Deployment:** Simple HTTP server, not complex microservices
**Advantages:**
- Standard integration protocols
- Stateless and scalable
- Simple monitoring and health checks
- Limited scope (reserves calculation only)

## Production Environment Requirements

### System Prerequisites

#### Operating Systems
- **Windows 10+** - Primary target (Chilean banking environments)
- **Linux (Ubuntu 20.04+, CentOS 8+)** - Server deployments
- **macOS 12+** - Development and testing

#### Runtime Dependencies
- **Go 1.21+** - For building from source
- **SQLite 3.35+** - Database engine (embedded)
- **Minimum RAM:** 4GB (8GB+ recommended for large batch processing)
- **Storage:** 10GB+ (mortality tables + database)
- **CPU:** 4+ cores (parallel processing benefit)

## Build & Deployment Process

### 1. Build Process

#### Development Build
```bash
# Clone repository
git clone <repository_url>
cd reservas

# Create build directory
mkdir -p bin

# Build for current platform
go build -o bin/calculator cmd/calculator/main.go

# Run with configuration
./bin/calculator --config config/config.json
```

#### Production Builds
```bash
# Build for Windows (amd64)
GOOS=windows GOARCH=amd64 go build -o bin/calculator-windows.exe cmd/calculator/main.go

# Build for Linux (amd64)
GOOS=linux GOARCH=amd64 go build -o bin/calculator-linux cmd/calculator/main.go

# Build for macOS (amd64) - for testing
GOOS=darwin GOARCH=amd64 go build -o bin/calculator-darwin cmd/calculator/main.go

# Build for macOS (arm64) - Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o bin/calculator-darwin-arm64 cmd/calculator/main.go
```

### 2. Configuration Management

#### Production Configuration Structure
```json
{
    "database": {
        "path": "/opt/reservas/data/reservas.db",
        "backup_enabled": true,
        "backup_path": "/opt/reservas/backups/",
        "backup_interval": "24h"
    },
    "calculation": {
        "max_workers": 8,
        "timeout_seconds": 300,
        "precision": 8,
        "batch_size": 1000
    },
    "cmf": {
        "vtd_provider_url": "https://www.cmf.cl/oficios",
        "update_interval": "24h",
        "fallback_rates": "/opt/reservas/data/default_vtd.json"
    },
    "logging": {
        "level": "info",
        "audit_file": "/opt/reservas/logs/audit.log",
        "max_file_size": "100MB",
        "max_files": 10
    }
}
```

#### Environment Variables
```bash
# Production environment variables
export RESERVAS_ENV=production
export RESERVAS_DB_PATH=/opt/reservas/data/reservas.db
export RESERVAS_LOG_LEVEL=info
export RESERVAS_MAX_WORKERS=8
export RESERVAS_CMFT_UPDATE_INTERVAL=24h
```

### 3. Directory Structure

#### Production Directory Layout
```
/opt/reservas/
├── bin/
│   ├── calculator-linux         # Production binary
│   └── calculator-linux.exe    # Windows binary
├── data/
│   ├── reservas.db            # Production database
│   ├── normativo/
│   │   └── articles-20210_tablas_mort_hist.xlsx
│   └── backups/               # Database backups
├── config/
│   ├── config.json            # Production configuration
│   └── config.dev.json        # Development configuration
├── logs/
│   ├── audit.log              # Audit trail for CMF
│   ├── calculator.log          # Application logs
│   └── errors.log             # Error logs
├── scripts/
│   ├── install.sh             # Installation script
│   ├── backup.sh              # Backup script
│   └── update.sh              # Update script
└── systemd/
    └── reservas.service       # System service definition
```

### 4. Installation Scripts

#### Linux Installation (install.sh)
```bash
#!/bin/bash
set -e

# Installation directories
INSTALL_DIR="/opt/reservas"
SERVICE_USER="reservas"

# Create service user
sudo useradd -r -s /bin/false $SERVICE_USER

# Create directories
sudo mkdir -p $INSTALL_DIR/{bin,data,config,logs}
sudo chown -R $SERVICE_USER:$SERVICE_USER $INSTALL_DIR

# Copy binary and configuration
sudo cp bin/calculator-linux $INSTALL_DIR/bin/
sudo cp config/config.json $INSTALL_DIR/config/
sudo chmod +x $INSTALL_DIR/bin/calculator-linux

# Set permissions
sudo chown -R $SERVICE_USER:$SERVICE_USER $INSTALL_DIR
sudo chmod 750 $INSTALL_DIR
sudo chmod 640 $INSTALL_DIR/config/config.json

# Copy mortality tables
sudo cp -r data/normativo $INSTALL_DIR/data/
sudo chown -R $SERVICE_USER:$SERVICE_USER $INSTALL_DIR/data

# Install systemd service
sudo cp systemd/reservas.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable reservas

echo "Installation completed. Start service with: sudo systemctl start reservas"
```

#### Windows Installation (install.ps1)
```powershell
# Windows installation script
$InstallDir = "C:\Program Files\ReservasCalculator"
$ServiceName = "ReservasCalculator"

# Create installation directory
New-Item -ItemType Directory -Path $InstallDir -Force

# Copy binary and configuration
Copy-Item "bin\calculator-windows.exe" "$InstallDir\"
Copy-Item "config\config.json" "$InstallDir\"

# Create data directory
New-Item -ItemType Directory -Path "$InstallDir\data" -Force
Copy-Item "data\normativo" "$InstallDir\data\" -Recurse

# Create Windows service
New-Service -Name $ServiceName -BinaryPathName "$InstallDir\calculator-windows.exe" -DisplayName "Reservas Calculator" -StartupType Automatic

Write-Host "Installation completed. Start service with: Start-Service $ServiceName"
```

## Service Management

### 1. Systemd Service (Linux)

#### reservas.service
```ini
[Unit]
Description=Reservas Calculator Service
After=network.target

[Service]
Type=simple
User=reservas
Group=reservas
WorkingDirectory=/opt/reservas
ExecStart=/opt/reservas/bin/calculator-linux --config /opt/reservas/config/config.json
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/opt/reservas/data /opt/reservas/logs

[Install]
WantedBy=multi-user.target
```

#### Service Commands
```bash
# Start service
sudo systemctl start reservas

# Stop service
sudo systemctl stop reservas

# Check status
sudo systemctl status reservas

# View logs
sudo journalctl -u reservas -f

# Enable on boot
sudo systemctl enable reservas
```

### 2. Windows Service

#### Service Management Commands
```powershell
# Install service
New-Service -Name "ReservasCalculator" -BinaryPathName "C:\Program Files\ReservasCalculator\calculator-windows.exe" -DisplayName "Reservas Calculator" -StartupType Automatic

# Start service
Start-Service -Name "ReservasCalculator"

# Stop service
Stop-Service -Name "ReservasCalculator"

# Check status
Get-Service -Name "ReservasCalculator"

# Remove service
Remove-Service -Name "ReservasCalculator"
```

## Monitoring & Maintenance

### 1. Health Checks

#### API Health Endpoint (if applicable)
```go
// Health check endpoint
func HealthCheck(c *gin.Context) {
    health := map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now(),
        "version": "1.0.0",
        "database": checkDatabaseHealth(),
        "last_vtd_update": getLastVTDUpdate(),
    }
    c.JSON(200, health)
}
```

#### Command-Line Health Check Script
```bash
#!/bin/bash
# health_check.sh
./bin/calculator-linux --health-check
if [ $? -eq 0 ]; then
    echo "Service is healthy"
    exit 0
else
    echo "Service is unhealthy"
    exit 1
fi
```

### 2. Backup Strategy

#### Automated Backup Script (backup.sh)
```bash
#!/bin/bash
BACKUP_DIR="/opt/reservas/backups"
DB_PATH="/opt/reservas/data/reservas.db"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p $BACKUP_DIR

# Create database backup
sqlite3 $DB_PATH ".backup $BACKUP_DIR/reservas_$DATE.db"

# Compress backup
gzip $BACKUP_DIR/reservas_$DATE.db

# Remove backups older than 30 days
find $BACKUP_DIR -name "*.gz" -mtime +30 -delete

echo "Backup completed: reservas_$DATE.db.gz"
```

#### Cron Job for Automated Backups
```bash
# Add to crontab: crontab -e
0 2 * * * /opt/reservas/scripts/backup.sh >> /opt/reservas/logs/backup.log 2>&1
```

### 3. Log Management

#### Log Rotation Configuration
```bash
# /etc/logrotate.d/reservas
/opt/reservas/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 640 reservas reservas
    postrotate
        systemctl reload reservas
    endscript
}
```

## Performance Tuning

### 1. Database Optimization

#### SQLite Performance Settings
```sql
-- Performance optimizations
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = 10000;
PRAGMA temp_store = MEMORY;
PRAGMA mmap_size = 268435456; -- 256MB
```

#### Database Indexes
```sql
-- Essential indexes for performance
CREATE INDEX idx_poliza_estado ON poliza(estado);
CREATE INDEX idx_reserva_fecha ON reserva_calculada(fecha_calculo);
CREATE INDEX idx_mortalidad_nombre_edad ON tabla_mortalidad(nombre_estandar, edad);
CREATE INDEX idx_vtd_fecha ON vtd_mensual(fecha_publicacion);
```

### 2. Parallel Processing Tuning

#### Worker Pool Configuration
```go
// Optimize worker count based on system
workerCount := runtime.NumCPU()
if maxWorkers > 0 && maxWorkers < workerCount {
    workerCount = maxWorkers
}

// Tune batch size based on available memory
batchSize := 1000
if maxMemoryMB > 0 {
    batchSize = maxMemoryMB / 10 // Estimate 10MB per 1000 policies
}
```

## Security Considerations

### 1. File System Security

#### Directory Permissions
```bash
# Secure file permissions
sudo chmod 750 /opt/reservas
sudo chmod 640 /opt/reservas/config/config.json
sudo chmod 750 /opt/reservas/logs
sudo chown -R reservas:reservas /opt/reservas
```

### 2. Audit Trail Security

#### Immutable Logging
```go
// Write-only audit log directory
os.Chmod("/opt/reservas/logs/audit.log", 0400) // Read-only after creation
```

## Troubleshooting

### Common Issues

#### 1. Database Lock Errors
**Symptoms:** "database is locked" errors
**Solutions:**
- Ensure WAL journal mode is enabled
- Check for long-running transactions
- Increase timeout settings

#### 2. Performance Issues
**Symptoms:** Slow batch processing
**Solutions:**
- Increase worker count
- Optimize database indexes
- Check memory usage and swap

#### 3. Permission Errors
**Symptoms:** File access denied errors
**Solutions:**
- Verify user permissions
- Check directory ownership
- Ensure service runs as correct user

### Debug Mode

#### Debug Configuration
```json
{
    "logging": {
        "level": "debug",
        "enable_profiling": true,
        "profile_port": 6060
    }
}
```

## Deployment Recommendations

### For Production Use
1. **Command-Line Utility** with systemd service (Linux) or Windows service
2. **Scheduled Calculations** using cron/Task Scheduler for regulatory requirements
3. **TUI Application** for actuarial analysis teams (optional)
4. **Simple HTTP API** only if system integration is required

### For Development
1. **Local binary execution** with configuration file
2. **SQLite database** for development data
3. **Debug logging** for troubleshooting
4. **Mortality table loader** from Excel files

This simplified deployment approach focuses on the calculator as a specialized utility rather than a complex web application, aligning with its purpose as a regulatory financial calculation tool.