# Testing Dashboard Integration with Daemon

This document shows how the dashboard integration works with the Blackhole daemon.

## Available Commands

### 1. Daemon with Integrated Dashboard (Default)
```bash
# Start daemon with dashboard (foreground mode for testing)
./bin/blackhole daemon --foreground

# Expected output:
# 🌐 Starting web dashboard on http://localhost:8080
# 📊 Dashboard available at http://localhost:8080
# Blackhole daemon running. Press Ctrl+C to stop.
```

### 2. Daemon with Custom Dashboard Port
```bash
# Start daemon with dashboard on port 9090
./bin/blackhole daemon --foreground --dashboard-port 9090

# Dashboard will be available at http://localhost:9090
```

### 3. Daemon without Dashboard
```bash
# Start daemon without dashboard
./bin/blackhole daemon --foreground --dashboard=false

# No dashboard server will start
```

### 4. Background Daemon with Dashboard
```bash
# Start daemon in background with dashboard
./bin/blackhole daemon --background

# Check if running
./bin/blackhole daemon --status

# Stop background daemon (cleans up dashboard and all files)
./bin/blackhole daemon stop

# Alternative stop command
./bin/blackhole daemon-stop

# Force stop if needed
./bin/blackhole daemon-stop --force
```

## What the Integration Provides

1. **Single Command**: Users only need to run `blackhole daemon` to get both the service orchestrator and monitoring dashboard
2. **Automatic Startup**: Dashboard starts automatically when daemon starts
3. **Graceful Shutdown**: Dashboard stops when daemon stops
4. **Complete Cleanup**: Both `daemon stop` and `daemon-stop` commands clean up all files
5. **Configurable**: Users can customize dashboard host/port or disable it entirely
6. **Uptime Tracking**: Daemon creates uptime file for dashboard to show system uptime
7. **Process Management**: Dashboard server runs as a goroutine within the daemon process
8. **Stale File Cleanup**: Stop commands remove uptime files and stale service PID files

## Dashboard Features When Integrated

- Real-time service status monitoring
- Service health detection via PID files and Unix sockets
- Overview statistics (running/stopped services, uptime)
- RESTful API endpoints (/api/status, /api/health)
- Auto-refresh functionality
- Responsive web interface

## Testing the Integration

1. Start daemon: `./bin/blackhole daemon --foreground`
2. Open browser to: http://localhost:8080
3. Verify dashboard shows all 8 services as "stopped" (since no services are running yet)
4. Check API directly: `curl http://localhost:8080/api/status`
5. Stop daemon with Ctrl+C
6. Verify dashboard is no longer accessible

This integration makes Blackhole much more user-friendly by providing monitoring out-of-the-box with zero additional setup required.