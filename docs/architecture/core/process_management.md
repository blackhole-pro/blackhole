# Process Management Architecture

## Overview

The Process Management system provides a robust framework for spawning, managing, and orchestrating service processes within the Blackhole platform. Each service runs as an independent OS process, communicating through gRPC, with the orchestrator binary managing their lifecycle, health monitoring, and resource allocation.

## Core Design Principles

Services run as separate processes, providing:
- **Process Isolation**: True OS-level isolation between services
- **Fault Tolerance**: Service crashes don't affect other services
- **Resource Control**: OS-level CPU, memory, and I/O limits
- **Independent Updates**: Restart individual services without downtime
- **Better Debugging**: Process-level profiling and monitoring

## Architecture Components

### Process Manager

The central orchestrator for all service processes:

```go
type ProcessManager struct {
    processes   map[string]*ServiceProcess
    supervisor  *ProcessSupervisor
    config      *ConfigManager
    monitor     *ResourceMonitor
    mu          sync.RWMutex
}

type ServiceProcess struct {
    Name          string
    Command       *exec.Cmd
    PID           int
    State         ProcessState
    Config        ServiceConfig
    StartTime     time.Time
    RestartCount  int
    UnixSocket    string
    TCPPort       int
    Resources     *ResourceMetrics
}
```

#### Core Responsibilities
- **Process Lifecycle**: Start, stop, restart service processes
- **Health Monitoring**: Monitor process health and performance
- **Resource Management**: Apply and enforce resource limits
- **Configuration Distribution**: Pass configuration to services
- **Process Discovery**: Track running processes and endpoints

### Process Spawning

Services are spawned as child processes:

```go
func (pm *ProcessManager) SpawnService(name string, config ServiceConfig) error {
    // Build command
    cmd := exec.Command(
        pm.binaryPath,
        "service",
        name,
        "--config", config.ConfigPath,
        "--socket", config.UnixSocket,
        "--port", fmt.Sprintf("%d", config.TCPPort),
    )
    
    // Set environment
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("BLACKHOLE_SERVICE=%s", name),
        fmt.Sprintf("BLACKHOLE_DATA_DIR=%s", config.DataDir),
        fmt.Sprintf("BLACKHOLE_LOG_LEVEL=%s", config.LogLevel),
        fmt.Sprintf("GOMEMLIMIT=%d", config.Resources.MemoryMB*1024*1024),
    )
    
    // Apply resource limits
    if err := pm.applyResourceLimits(cmd, config.Resources); err != nil {
        return fmt.Errorf("apply limits: %w", err)
    }
    
    // Start process
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("start process: %w", err)
    }
    
    // Create process record
    process := &ServiceProcess{
        Name:       name,
        Command:    cmd,
        PID:        cmd.Process.Pid,
        State:      ProcessStateStarting,
        Config:     config,
        StartTime:  time.Now(),
        UnixSocket: config.UnixSocket,
        TCPPort:    config.TCPPort,
    }
    
    // Register process
    pm.mu.Lock()
    pm.processes[name] = process
    pm.mu.Unlock()
    
    // Start monitoring
    go pm.supervisor.Monitor(process)
    
    // Wait for process to be ready
    return pm.waitForReady(process)
}
```

### Resource Isolation

Each service process has OS-level resource limits:

```go
type ResourceLimits struct {
    CPUPercent  float64  // CPU quota as percentage
    MemoryMB    int64    // Memory limit in MB
    IOWeight    int      // IO priority (100-1000)
    OpenFiles   int      // File descriptor limit
    Processes   int      // Process/thread limit
}

func (pm *ProcessManager) applyResourceLimits(cmd *exec.Cmd, limits ResourceLimits) error {
    // Create cgroup for process
    cgroupPath := pm.createProcessCgroup(cmd.Args[2])
    
    // CPU limits
    if limits.CPUPercent > 0 {
        cpuMax := fmt.Sprintf("%d 100000", int(limits.CPUPercent*1000))
        if err := os.WriteFile(
            filepath.Join(cgroupPath, "cpu.max"),
            []byte(cpuMax),
            0644,
        ); err != nil {
            return fmt.Errorf("set cpu limit: %w", err)
        }
    }
    
    // Memory limits
    if limits.MemoryMB > 0 {
        memoryBytes := limits.MemoryMB * 1024 * 1024
        if err := os.WriteFile(
            filepath.Join(cgroupPath, "memory.max"),
            []byte(fmt.Sprintf("%d", memoryBytes)),
            0644,
        ); err != nil {
            return fmt.Errorf("set memory limit: %w", err)
        }
    }
    
    // IO weight
    if limits.IOWeight > 0 {
        if err := os.WriteFile(
            filepath.Join(cgroupPath, "io.weight"),
            []byte(fmt.Sprintf("%d", limits.IOWeight)),
            0644,
        ); err != nil {
            return fmt.Errorf("set io weight: %w", err)
        }
    }
    
    // File descriptor limits (using rlimits)
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,  // Create new process group
        Rlimits: []syscall.Rlimit{
            {
                Type: syscall.RLIMIT_NOFILE,
                Cur:  uint64(limits.OpenFiles),
                Max:  uint64(limits.OpenFiles),
            },
            {
                Type: syscall.RLIMIT_NPROC,
                Cur:  uint64(limits.Processes),
                Max:  uint64(limits.Processes),
            },
        },
    }
    
    return nil
}
```

## Process Lifecycle

### Startup Sequence

Services start in dependency order:

```go
func (pm *ProcessManager) StartAll(ctx context.Context) error {
    // Service dependency order
    startOrder := []string{
        "identity",  // Base service, no dependencies
        "storage",   // Depends on identity
        "node",      // Depends on identity
        "ledger",    // Depends on identity
        "social",    // Depends on identity, storage
        "indexer",   // Depends on storage
        "analytics", // Depends on multiple services
        "telemetry", // Depends on all services
        "wallet",    // Depends on identity, ledger
    }
    
    // Start services in order
    for _, service := range startOrder {
        config := pm.config.GetServiceConfig(service)
        if !config.Enabled {
            log.Printf("Service %s is disabled, skipping", service)
            continue
        }
        
        log.Printf("Starting service: %s", service)
        
        if err := pm.SpawnService(service, config); err != nil {
            // Rollback on failure
            pm.stopStartedServices(startOrder, service)
            return fmt.Errorf("start %s: %w", service, err)
        }
        
        // Wait for health check
        if err := pm.waitForHealthy(service); err != nil {
            pm.stopStartedServices(startOrder, service)
            return fmt.Errorf("health check %s: %w", service, err)
        }
        
        log.Printf("Service %s started successfully", service)
    }
    
    return nil
}
```

### Health Monitoring

Continuous health checks for all processes:

```go
type ProcessHealth struct {
    Service      string
    PID          int
    Status       HealthStatus
    LastCheck    time.Time
    Uptime       time.Duration
    CPUUsage     float64
    MemoryUsage  int64
    RestartCount int
}

func (pm *ProcessManager) MonitorHealth(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            pm.checkAllProcesses()
        case <-ctx.Done():
            return
        }
    }
}

func (pm *ProcessManager) checkProcess(name string) (*ProcessHealth, error) {
    pm.mu.RLock()
    process, exists := pm.processes[name]
    pm.mu.RUnlock()
    
    if !exists {
        return nil, fmt.Errorf("process not found: %s", name)
    }
    
    health := &ProcessHealth{
        Service:      name,
        PID:          process.PID,
        RestartCount: process.RestartCount,
        Uptime:       time.Since(process.StartTime),
    }
    
    // Check if process is running
    if err := process.Command.Process.Signal(syscall.Signal(0)); err != nil {
        health.Status = HealthStatusDown
        return health, nil
    }
    
    // Get process metrics
    if metrics, err := pm.monitor.GetProcessMetrics(process.PID); err == nil {
        health.CPUUsage = metrics.CPUPercent
        health.MemoryUsage = metrics.MemoryRSS
    }
    
    // Check gRPC health endpoint
    conn, err := grpc.Dial("unix://"+process.UnixSocket,
        grpc.WithInsecure(),
        grpc.WithTimeout(3*time.Second),
    )
    if err != nil {
        health.Status = HealthStatusUnhealthy
        return health, nil
    }
    defer conn.Close()
    
    client := grpc_health_v1.NewHealthClient(conn)
    resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{
        Service: name,
    })
    
    if err != nil || resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
        health.Status = HealthStatusUnhealthy
    } else {
        health.Status = HealthStatusHealthy
    }
    
    health.LastCheck = time.Now()
    return health, nil
}
```

### Process Supervision

Automatic restart with exponential backoff:

```go
type ProcessSupervisor struct {
    manager    *ProcessManager
    policies   map[string]RestartPolicy
    monitor    *ProcessMonitor
}

type RestartPolicy struct {
    MaxRestarts   int
    InitialDelay  time.Duration
    MaxDelay      time.Duration
    BackoffFactor float64
    RestartWindow time.Duration
}

func (s *ProcessSupervisor) Monitor(process *ServiceProcess) {
    policy := s.policies[process.Name]
    
    for {
        // Wait for process to exit
        err := process.Command.Wait()
        
        // Check if shutdown was requested
        if process.State == ProcessStateStopping {
            log.Printf("Process %s stopped normally", process.Name)
            return
        }
        
        // Log crash
        exitCode := process.Command.ProcessState.ExitCode()
        log.Printf("Process %s crashed with exit code %d: %v", 
            process.Name, exitCode, err)
        
        // Check restart policy
        if process.RestartCount >= policy.MaxRestarts {
            log.Printf("Process %s exceeded max restarts (%d)", 
                process.Name, policy.MaxRestarts)
            process.State = ProcessStateFailed
            return
        }
        
        // Calculate backoff
        delay := s.calculateBackoff(process.RestartCount, policy)
        log.Printf("Restarting %s in %v (attempt %d/%d)", 
            process.Name, delay, process.RestartCount+1, policy.MaxRestarts)
        
        // Wait before restart
        time.Sleep(delay)
        
        // Restart process
        process.RestartCount++
        if err := s.manager.SpawnService(process.Name, process.Config); err != nil {
            log.Printf("Failed to restart %s: %v", process.Name, err)
            process.State = ProcessStateFailed
            return
        }
    }
}

func (s *ProcessSupervisor) calculateBackoff(attempts int, policy RestartPolicy) time.Duration {
    delay := policy.InitialDelay
    for i := 0; i < attempts; i++ {
        delay = time.Duration(float64(delay) * policy.BackoffFactor)
        if delay > policy.MaxDelay {
            delay = policy.MaxDelay
            break
        }
    }
    return delay
}
```

### Graceful Shutdown

Coordinated shutdown in reverse dependency order:

```go
func (pm *ProcessManager) Shutdown(ctx context.Context) error {
    // Reverse dependency order
    shutdownOrder := []string{
        "wallet",
        "telemetry",
        "analytics",
        "indexer",
        "social",
        "ledger",
        "node",
        "storage",
        "identity",
    }
    
    log.Println("Beginning graceful shutdown...")
    
    // Stop services in reverse order
    for _, service := range shutdownOrder {
        pm.mu.RLock()
        process, exists := pm.processes[service]
        pm.mu.RUnlock()
        
        if !exists {
            continue
        }
        
        log.Printf("Stopping service: %s", service)
        
        // Mark as stopping
        process.State = ProcessStateStopping
        
        // Send graceful shutdown signal
        if err := pm.gracefulStop(process); err != nil {
            log.Printf("Graceful stop failed for %s: %v", service, err)
            // Force kill after timeout
            process.Command.Process.Kill()
        }
        
        // Wait for process to exit
        process.Command.Wait()
        
        // Cleanup resources
        pm.cleanup(service)
        
        log.Printf("Service %s stopped", service)
    }
    
    return nil
}

func (pm *ProcessManager) gracefulStop(process *ServiceProcess) error {
    // Send SIGTERM
    if err := process.Command.Process.Signal(syscall.SIGTERM); err != nil {
        return err
    }
    
    // Wait for graceful shutdown with timeout
    done := make(chan error, 1)
    go func() {
        done <- process.Command.Wait()
    }()
    
    select {
    case err := <-done:
        return err
    case <-time.After(30 * time.Second):
        return fmt.Errorf("shutdown timeout")
    }
}
```

## Configuration Management

### Service Configuration

Each service has its own configuration:

```go
type ServiceConfig struct {
    Name       string
    Enabled    bool
    DataDir    string
    LogLevel   string
    
    // Process endpoints
    UnixSocket string
    TCPPort    int
    
    // Resource limits
    Resources ResourceLimits
    
    // Health check settings
    HealthCheck HealthCheckConfig
    
    // Restart policy
    Restart RestartPolicy
    
    // Service-specific config
    Config map[string]interface{}
}

type HealthCheckConfig struct {
    Interval time.Duration
    Timeout  time.Duration
    Retries  int
}
```

Example configuration:
```yaml
services:
  identity:
    enabled: true
    data_dir: /var/lib/blackhole/identity
    log_level: info
    unix_socket: /var/run/blackhole/identity.sock
    tcp_port: 50001
    
    resources:
      cpu_percent: 200      # 2 CPU cores
      memory_mb: 1024       # 1GB memory
      io_weight: 500        # Medium I/O priority
      open_files: 10000     # Max file descriptors
      processes: 100        # Max threads/processes
      
    health_check:
      interval: 5s
      timeout: 3s
      retries: 3
      
    restart:
      max_restarts: 5
      initial_delay: 1s
      max_delay: 5m
      backoff_factor: 2.0
      restart_window: 1h
      
    config:
      database_url: postgres://localhost/identity
      jwt_secret: ${JWT_SECRET}
      cache_size: 256MB
```

## Process Monitoring

### Resource Metrics

Track resource usage per process:

```go
type ProcessMetrics struct {
    PID          int
    CPUPercent   float64
    MemoryRSS    int64
    MemoryVMS    int64
    Threads      int
    OpenFiles    int
    Connections  int
    IOReadBytes  int64
    IOWriteBytes int64
    Uptime       time.Duration
}

func (m *ProcessMonitor) GetProcessMetrics(pid int) (*ProcessMetrics, error) {
    // Use procfs or platform-specific APIs
    proc, err := process.NewProcess(int32(pid))
    if err != nil {
        return nil, err
    }
    
    metrics := &ProcessMetrics{PID: pid}
    
    // CPU usage
    if cpu, err := proc.CPUPercent(); err == nil {
        metrics.CPUPercent = cpu
    }
    
    // Memory info
    if memInfo, err := proc.MemoryInfo(); err == nil {
        metrics.MemoryRSS = int64(memInfo.RSS)
        metrics.MemoryVMS = int64(memInfo.VMS)
    }
    
    // Thread count
    if threads, err := proc.NumThreads(); err == nil {
        metrics.Threads = int(threads)
    }
    
    // Open files
    if files, err := proc.OpenFiles(); err == nil {
        metrics.OpenFiles = len(files)
    }
    
    // Network connections
    if conns, err := proc.Connections(); err == nil {
        metrics.Connections = len(conns)
    }
    
    // IO stats
    if ioStats, err := proc.IOCounters(); err == nil {
        metrics.IOReadBytes = int64(ioStats.ReadBytes)
        metrics.IOWriteBytes = int64(ioStats.WriteBytes)
    }
    
    // Process uptime
    if createTime, err := proc.CreateTime(); err == nil {
        metrics.Uptime = time.Since(time.Unix(createTime/1000, 0))
    }
    
    return metrics, nil
}
```

### Log Aggregation

Centralized logging from all processes:

```go
type ProcessLogAggregator struct {
    logs   map[string]*LogStream
    output LogOutput
}

type LogStream struct {
    Service string
    PID     int
    Stdout  io.ReadCloser
    Stderr  io.ReadCloser
}

func (a *ProcessLogAggregator) AttachProcess(name string, cmd *exec.Cmd) error {
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return err
    }
    
    stderr, err := cmd.StderrPipe()
    if err != nil {
        return err
    }
    
    stream := &LogStream{
        Service: name,
        PID:     cmd.Process.Pid,
        Stdout:  stdout,
        Stderr:  stderr,
    }
    
    a.logs[name] = stream
    
    // Start streaming logs
    go a.streamLogs(stream.Stdout, name, "stdout")
    go a.streamLogs(stream.Stderr, name, "stderr")
    
    return nil
}

func (a *ProcessLogAggregator) streamLogs(reader io.Reader, service, stream string) {
    scanner := bufio.NewScanner(reader)
    
    for scanner.Scan() {
        log := LogEntry{
            Timestamp: time.Now(),
            Service:   service,
            Stream:    stream,
            Message:   scanner.Text(),
        }
        
        a.output.Write(log)
    }
}
```

### Metrics Export

Export process metrics for monitoring:

```go
func (pm *ProcessManager) RegisterMetrics() {
    // Process count by state
    prometheus.MustRegister(prometheus.NewGaugeFunc(
        prometheus.GaugeOpts{
            Namespace: "blackhole",
            Name:      "processes_total",
            Help:      "Total number of service processes by state",
        },
        func() float64 {
            pm.mu.RLock()
            defer pm.mu.RUnlock()
            return float64(len(pm.processes))
        },
    ))
    
    // CPU usage per process
    prometheus.MustRegister(prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "blackhole",
            Name:      "process_cpu_percent",
            Help:      "CPU usage percentage by service",
        },
        []string{"service"},
    ))
    
    // Memory usage per process
    prometheus.MustRegister(prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "blackhole",
            Name:      "process_memory_bytes",
            Help:      "Memory usage in bytes by service",
        },
        []string{"service"},
    ))
    
    // Process restarts
    prometheus.MustRegister(prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "blackhole",
            Name:      "process_restarts_total",
            Help:      "Total number of process restarts",
        },
        []string{"service"},
    ))
    
    // Process uptime
    prometheus.MustRegister(prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "blackhole",
            Name:      "process_uptime_seconds",
            Help:      "Process uptime in seconds",
        },
        []string{"service"},
    ))
}
```

## Command-Line Management

Process management via CLI:

```bash
# Start all services
blackhole start --all

# Start specific services
blackhole start --services=identity,storage,ledger

# Stop services
blackhole stop --services=storage

# Restart a service
blackhole restart identity

# Check process status
blackhole status

# View process logs
blackhole logs identity --follow

# Monitor resource usage
blackhole top

# Health check
blackhole health
```

## Best Practices

### Process Implementation

1. **Graceful Shutdown**: Handle SIGTERM properly
2. **Health Endpoints**: Implement gRPC health service
3. **Resource Awareness**: Work within resource limits
4. **Structured Logging**: Use consistent log format
5. **Configuration**: Support environment variables
6. **Security**: Run with minimal privileges

### Process Management

1. **Restart Policies**: Configure appropriate restart behavior
2. **Resource Limits**: Set realistic limits per service
3. **Health Checks**: Implement meaningful health indicators
4. **Monitoring**: Export comprehensive metrics
5. **Logging**: Centralize log collection
6. **Backup**: Regular configuration backups

### Operations

1. **Deployment**: Use configuration management
2. **Updates**: Rolling updates with health checks
3. **Monitoring**: Set up alerts for process failures
4. **Debugging**: Enable debug logging when needed
5. **Documentation**: Keep runbooks updated

## Conclusion

The process management architecture provides:

- **True Isolation**: OS-level process separation
- **Fault Tolerance**: Process crashes don't cascade
- **Resource Control**: OS-enforced resource limits
- **Operational Flexibility**: Individual process management
- **Production Readiness**: Comprehensive monitoring and control

This design ensures that Blackhole nodes are both robust and operationally simple, with clear boundaries between services and efficient process management.