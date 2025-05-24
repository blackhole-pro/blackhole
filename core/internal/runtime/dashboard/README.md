# Blackhole Service Status Dashboard

A web-based dashboard for monitoring the status of all Blackhole services in real-time.

## Features

- **Real-time Service Monitoring**: Monitor the status of all 8 Blackhole services
- **Service Control**: Start, stop, and restart individual services with one-click buttons
- **Detailed Service Pages**: Click any service card to view comprehensive service details
- **Live Service Logs**: Real-time log streaming with filtering and download capabilities
- **Service Metrics**: CPU usage, memory consumption, uptime, and performance data
- **Quick Actions**: Health checks, configuration viewing, log exports, and more
- **Overall System Health**: See total running/stopped services and system uptime
- **Auto-refresh**: Configurable automatic status updates
- **Responsive Design**: Works on desktop and mobile devices
- **RESTful API**: JSON endpoints for integration with other tools
- **Instant Feedback**: Success/error notifications for all service actions

## Quick Start

### Option 1: Integrated with Daemon (Recommended)

1. **Start the daemon with integrated dashboard:**
   ```bash
   ./bin/blackhole daemon --foreground
   ```

2. **Open your browser:**
   Navigate to http://localhost:8080

3. **Monitor your services:**
   The dashboard will automatically start alongside the daemon and show real-time status of all services.

### Option 2: Standalone Dashboard

1. **Start the dashboard server separately:**
   ```bash
   ./bin/blackhole dashboard
   ```

2. **Open your browser:**
   Navigate to http://localhost:8080

3. **Monitor your services:**
   The dashboard will show the current status of all services with auto-refresh enabled.

## Command Line Options

### Daemon with Integrated Dashboard (Recommended)

```bash
./bin/blackhole daemon [flags]

Dashboard-related flags:
  --dashboard               Start web dashboard with daemon (default: true)
  --dashboard-host string   Host for web dashboard (default "localhost")
  --dashboard-port int      Port for web dashboard (default 8080)
  --foreground              Run in foreground (attached to terminal)
  --background              Run in background (detached from terminal, default)
```

### Standalone Dashboard

```bash
./bin/blackhole dashboard [flags]

Flags:
  -h, --help          help for dashboard
  -H, --host string   Host to bind dashboard server to (default "localhost")
  -p, --port int      Port to serve dashboard on (default "8080")
```

### Examples

```bash
# Start daemon with integrated dashboard (foreground)
./bin/blackhole daemon --foreground

# Start daemon with dashboard on custom port
./bin/blackhole daemon --foreground --dashboard-port 9090

# Start daemon without dashboard
./bin/blackhole daemon --foreground --dashboard=false

# Start daemon in background with dashboard
./bin/blackhole daemon --background --dashboard-port 8080

# Stop daemon (also stops dashboard automatically)
./bin/blackhole daemon stop

# Stop daemon using dedicated command
./bin/blackhole daemon-stop

# Start standalone dashboard
./bin/blackhole dashboard

# Start standalone dashboard on custom port
./bin/blackhole dashboard --port 9090

# Bind dashboard to all interfaces
./bin/blackhole dashboard --host 0.0.0.0 --port 8080
```

## Monitored Services

The dashboard monitors these Blackhole services:

- **Identity Service** - DID authentication and OAuth integration
- **Storage Service** - IPFS/Filecoin content storage with encryption
- **Node Service** - P2P networking and node operations
- **Ledger Service** - Blockchain integration with Root Network
- **Social Service** - ActivityPub federation
- **Indexer Service** - Content indexing with SubQuery
- **Analytics Service** - Usage analytics and metrics
- **Wallet Service** - Cryptocurrency wallet functionality

## API Endpoints

### Service Status
```
GET /api/status
```

Returns the current status of all services in JSON format:

```json
{
  "timestamp": "2025-05-22T10:30:00Z",
  "uptime": "2h 15m",
  "services": {
    "identity": {
      "status": "running",
      "port": 8001,
      "pid": 12345,
      "lastCheck": "2025-05-22T10:30:00Z"
    },
    "storage": {
      "status": "stopped",
      "port": null,
      "pid": null,
      "lastCheck": "2025-05-22T10:30:00Z"
    }
  }
}
```

### Health Check
```
GET /api/health
```

Returns the overall system health:

```json
{
  "status": "healthy",
  "timestamp": "2025-05-22T10:30:00Z",
  "uptime": "2h 15m"
}
```

### Service Actions
```
POST /api/services/{service}/{action}
```

Control individual services with the following actions:
- **start**: Start a stopped service
- **stop**: Stop a running service  
- **restart**: Restart a service (stop then start)

Example requests:
```bash
# Start the identity service
curl -X POST http://localhost:8080/api/services/identity/start

# Stop the storage service
curl -X POST http://localhost:8080/api/services/storage/stop

# Restart the node service
curl -X POST http://localhost:8080/api/services/node/restart
```

### Service Details
```
GET /api/services/{service}/details
```

Get comprehensive service information including metrics and configuration:

```json
{
  "serviceName": "identity",
  "socketPath": "sockets/identity.sock",
  "uptime": "2h 15m",
  "cpuUsage": "18.5%",
  "memoryUsage": "64.2 MB",
  "startCount": 3,
  "serviceType": "Standard",
  "autoRestart": true,
  "logLevel": "info",
  "configFile": "configs/blackhole.yaml"
}
```

### Service Logs
```
GET /api/services/{service}/logs
```

Retrieve recent log entries for a specific service:

```json
{
  "logs": [
    {
      "timestamp": "2025-05-22 10:25:00",
      "level": "info",
      "message": "identity service initialized successfully"
    },
    {
      "timestamp": "2025-05-22 10:27:00",
      "level": "warn",
      "message": "High memory usage detected"
    }
  ]
}
```

### Health Check
```
POST /api/services/{service}/health
```

Perform a comprehensive health check on a service:

```json
{
  "service": "identity",
  "status": "healthy",
  "checks": {
    "database": "connected",
    "memory": "normal",
    "disk_space": "sufficient",
    "network": "accessible"
  },
  "timestamp": "2025-05-22T10:30:00Z"
}
```

Example action response:
```json
{
  "success": true,
  "message": "Successfully performed start on identity service"
}
```

Error response:
```json
{
  "success": false,
  "error": "Service is already running"
}
```

## Status Indicators

Services can have the following status values:

- **🟢 Running** - Service is active and healthy
- **🔴 Stopped** - Service is not running
- **🟡 Error** - Service process exists but has issues
- **⚪ Unknown** - Status cannot be determined

## Features in Detail

### Auto-Refresh
- Toggle auto-refresh with the "Enable Auto-Refresh" button
- Updates every 5 seconds when enabled
- Manual refresh available at any time

### Responsive Design
- Mobile-friendly interface
- Grid layout adapts to screen size
- Touch-friendly controls

### Service Health Detection
The dashboard detects service health by:
1. Checking PID files in the `sockets/` directory
2. Verifying processes are actually running
3. Confirming Unix socket files exist
4. Validating service availability

### Service Control Actions
- **Smart Button States**: Start button disabled when service is running, stop/restart buttons disabled when stopped
- **Loading Indicators**: Buttons show spinning animation during action execution
- **Instant Feedback**: Success/error notifications appear after each action
- **Auto-refresh**: Status automatically updates after service actions complete

### Service Detail Pages
- **Click to Navigate**: Click any service card to open a detailed service page
- **Comprehensive Metrics**: View CPU usage, memory consumption, uptime, and start count
- **Live Log Streaming**: Real-time log entries with level filtering (Error, Warn, Info, Debug)
- **Log Management**: Clear logs, download logs, toggle auto-scroll
- **Quick Actions**: Health checks, configuration viewing, debug restarts
- **Enhanced Controls**: Larger action buttons with advanced service management
- **Configuration Info**: Service type, auto-restart settings, log levels, config files

### Automatic Cleanup
When the daemon is stopped, the system automatically:
- Stops the dashboard server gracefully
- Removes uptime tracking files
- Cleans up stale service PID files
- Ensures no orphaned processes remain

Both `daemon stop` and `daemon-stop` commands perform complete cleanup.

## Keyboard Shortcuts

- **Ctrl/Cmd + R** - Manual refresh (overrides browser default)

## File Structure

```
web/dashboard/
├── index.html          # Main dashboard page
├── styles.css          # Responsive CSS styling
├── dashboard.js        # JavaScript for real-time updates
└── README.md           # This documentation
```

## Integration

The dashboard can be integrated with other monitoring tools via the JSON API endpoints:

```bash
# Get service status via curl
curl http://localhost:8080/api/status

# Monitor specific service
curl http://localhost:8080/api/status | jq '.services.identity'

# Check if any services are down
curl http://localhost:8080/api/status | jq '.services | to_entries[] | select(.value.status != "running")'
```

## Development

To modify the dashboard:

1. **HTML**: Edit `web/dashboard/index.html` for structure changes
2. **CSS**: Modify `web/dashboard/styles.css` for styling
3. **JavaScript**: Update `web/dashboard/dashboard.js` for functionality
4. **Backend**: Modify `cmd/blackhole/commands/dashboard.go` for API changes

The dashboard uses vanilla HTML/CSS/JavaScript for simplicity and no external dependencies.

## Troubleshooting

### Dashboard won't start
- Ensure port is not in use: `lsof -i :8080`
- Check if you have permission to bind to the port
- Try a different port: `--port 9090`

### Services show as "Unknown"
- Verify services are started: `./bin/blackhole status`
- Check if `sockets/` directory exists
- Ensure PID files are being created correctly

### Auto-refresh not working
- Check browser console for JavaScript errors
- Verify API endpoint is accessible: `curl http://localhost:8080/api/status`
- Disable browser extensions that might block requests

### Styling issues
- Clear browser cache
- Check if CSS file is loading: view source and click CSS link
- Verify `web/dashboard/styles.css` exists and is readable

## Security Considerations

- Dashboard serves on localhost by default for security
- No authentication is implemented (intended for local development)
- Consider using a reverse proxy with authentication for production
- API endpoints return service information that could be sensitive

## Future Enhancements

Planned features for future releases:
- WebSocket support for real-time updates
- Service log viewing
- Performance metrics and charts
- Authentication and access control
- Historical data and trends
- Service restart capabilities
- Configuration management interface