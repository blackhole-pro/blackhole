# Operations Guide

## Overview

This guide provides comprehensive operational procedures for managing Blackhole platform deployments. It covers day-to-day operations, troubleshooting, maintenance, and emergency response procedures.

## Day-to-Day Operations

### Service Health Monitoring

Monitor service health using the built-in health check system:

```bash
# Check all services
blackhole status

# Check specific service
blackhole status identity

# Watch service health in real-time
blackhole status --watch

# Get detailed health metrics
blackhole status --detailed
```

Health check implementation:

```go
type HealthChecker struct {
    services map[string]ServiceClient
    metrics  *prometheus.Registry
}

func (h *HealthChecker) CheckAll() []HealthResult {
    results := make([]HealthResult, 0)
    
    for name, client := range h.services {
        result := h.checkService(name, client)
        results = append(results, result)
        
        // Update metrics
        h.updateMetrics(result)
    }
    
    return results
}

func (h *HealthChecker) checkService(name string, client ServiceClient) HealthResult {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    resp, err := client.Health(ctx, &pb.HealthRequest{})
    if err != nil {
        return HealthResult{
            Service: name,
            Status:  "unhealthy",
            Error:   err.Error(),
        }
    }
    
    return HealthResult{
        Service:  name,
        Status:   resp.Status,
        Uptime:   resp.Uptime,
        Metrics:  resp.Metrics,
    }
}
```

### Log Management

Centralized logging for all services:

```bash
# View orchestrator logs
journalctl -u blackhole -f

# View specific service logs
blackhole logs identity --tail 100

# Search logs by pattern
blackhole logs --grep "error" --since 1h

# Export logs for analysis
blackhole logs --export --format json > logs.json
```

Log aggregation setup:

```yaml
logging:
  level: info
  format: json
  outputs:
    - type: file
      path: /var/log/blackhole/blackhole.log
      rotate:
        max_size: 100MB
        max_age: 7d
        max_backups: 10
    
    - type: syslog
      address: syslog-server:514
      facility: local0
      
    - type: elasticsearch
      addresses:
        - http://elasticsearch:9200
      index: blackhole-logs
```

### Metrics Collection

Prometheus metrics endpoint:

```go
func (s *Service) SetupMetrics() {
    // Request metrics
    requestDuration := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "grpc_request_duration_seconds",
            Help:    "Duration of gRPC requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"service", "method", "status"},
    )
    
    // Process metrics
    processGauge := prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "process_resource_usage",
            Help: "Process resource usage",
        },
        []string{"service", "resource"},
    )
    
    prometheus.MustRegister(requestDuration, processGauge)
    
    // Start metrics server
    http.Handle("/metrics", promhttp.Handler())
    go http.ListenAndServe(":9090", nil)
}
```

## Service Management

### Starting Services

Start services with proper order:

```bash
# Start all services
blackhole start

# Start specific services
blackhole start identity storage

# Start with custom config
blackhole start --config /etc/blackhole/custom.yaml

# Dry run to validate config
blackhole start --dry-run
```

Service startup sequence:

```go
func (o *Orchestrator) StartServices(ctx context.Context) error {
    // Start in dependency order
    order := []string{"identity", "storage", "ledger", "node", "indexer"}
    
    for _, service := range order {
        if err := o.startService(ctx, service); err != nil {
            // Rollback on failure
            o.stopStartedServices()
            return fmt.Errorf("start %s: %w", service, err)
        }
        
        // Wait for health check
        if err := o.waitForHealthy(ctx, service, 30*time.Second); err != nil {
            return fmt.Errorf("health check %s: %w", service, err)
        }
    }
    
    return nil
}
```

### Stopping Services

Graceful shutdown procedures:

```bash
# Stop all services gracefully
blackhole stop

# Stop specific service
blackhole stop storage

# Force stop with timeout
blackhole stop --force --timeout 30s

# Emergency stop (kill processes)
blackhole stop --emergency
```

Graceful shutdown implementation:

```go
func (s *Service) Shutdown(ctx context.Context) error {
    log.Printf("Starting graceful shutdown of %s", s.Name)
    
    // Stop accepting new requests
    s.grpcServer.GracefulStop()
    
    // Wait for ongoing requests
    done := make(chan struct{})
    go func() {
        s.waitForRequests()
        close(done)
    }()
    
    select {
    case <-done:
        log.Printf("Graceful shutdown completed")
    case <-ctx.Done():
        log.Printf("Forcing shutdown after timeout")
        s.grpcServer.Stop()
    }
    
    // Cleanup resources
    return s.cleanup()
}
```

### Service Restart

Restart services with minimal downtime:

```bash
# Restart all services
blackhole restart

# Restart specific service
blackhole restart identity

# Rolling restart (one at a time)
blackhole restart --rolling

# Restart with new configuration
blackhole restart --reload-config
```

### Service Scaling

Scale services based on load:

```bash
# Scale service instances
blackhole scale storage --replicas 3

# Auto-scale based on metrics
blackhole scale --auto --min 1 --max 5

# Scale based on resource usage
blackhole scale --cpu-threshold 80 --mem-threshold 90
```

## Resource Management

### Memory Management

Monitor and manage memory usage:

```bash
# Check memory usage
blackhole resources memory

# Set memory limits
blackhole resources set-limit storage --memory 4G

# Clear caches
blackhole resources clear-cache --service storage

# Trigger garbage collection
blackhole resources gc --service identity
```

Memory monitoring:

```go
func (m *ResourceMonitor) MonitorMemory() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        for name, proc := range m.processes {
            stats := getProcessMemoryStats(proc.PID)
            
            // Check against limits
            if stats.RSS > proc.MemoryLimit*0.9 {
                log.Printf("Warning: %s approaching memory limit: %d/%d MB",
                    name, stats.RSS/1024/1024, proc.MemoryLimit/1024/1024)
                
                // Trigger memory pressure handling
                m.handleMemoryPressure(name, stats)
            }
            
            // Update metrics
            m.metrics.MemoryUsage.WithLabelValues(name).Set(float64(stats.RSS))
        }
    }
}
```

### CPU Management

Control CPU usage:

```bash
# Check CPU usage
blackhole resources cpu

# Set CPU limits
blackhole resources set-limit ledger --cpu 2.0

# Adjust process priority
blackhole resources priority storage --nice 10

# CPU profiling
blackhole debug cpu-profile --service identity --duration 30s
```

### Disk Management

Monitor and manage disk usage:

```bash
# Check disk usage
blackhole resources disk

# Set I/O limits
blackhole resources set-limit storage --iops 1000

# Clean up old data
blackhole maintenance cleanup --older-than 30d

# Compact databases
blackhole maintenance compact --service indexer
```

## Backup and Recovery

### Scheduled Backups

Configure automatic backups:

```yaml
backup:
  schedule:
    full: "0 2 * * 0"     # Weekly full backup
    incremental: "0 2 * * *" # Daily incremental
    
  retention:
    full: 4               # Keep 4 full backups
    incremental: 7        # Keep 7 incremental backups
    
  storage:
    type: s3
    bucket: blackhole-backups
    region: us-east-1
    prefix: production/
```

Backup implementation:

```go
func (b *BackupManager) PerformBackup(backupType string) error {
    backup := &Backup{
        ID:        uuid.New().String(),
        Type:      backupType,
        Timestamp: time.Now(),
        Services:  make(map[string]ServiceBackup),
    }
    
    // Backup each service
    for _, service := range b.services {
        serviceBackup, err := b.backupService(service)
        if err != nil {
            return fmt.Errorf("backup %s: %w", service.Name, err)
        }
        
        backup.Services[service.Name] = serviceBackup
    }
    
    // Upload to storage
    if err := b.storage.Upload(backup); err != nil {
        return fmt.Errorf("upload backup: %w", err)
    }
    
    // Update backup catalog
    return b.catalog.Add(backup)
}
```

### Manual Backups

Perform manual backups:

```bash
# Full backup
blackhole backup create --full

# Service-specific backup
blackhole backup create --service storage

# Backup to specific location
blackhole backup create --destination /backup/manual/

# Verify backup integrity
blackhole backup verify <backup-id>
```

### Restore Procedures

Restore from backup:

```bash
# List available backups
blackhole backup list

# Restore specific backup
blackhole backup restore <backup-id>

# Restore specific service
blackhole backup restore <backup-id> --service identity

# Test restore (dry run)
blackhole backup restore <backup-id> --dry-run

# Point-in-time recovery
blackhole backup restore --timestamp "2023-01-01 12:00:00"
```

Restore implementation:

```go
func (r *RestoreManager) RestoreBackup(backupID string) error {
    // Download backup
    backup, err := r.storage.Download(backupID)
    if err != nil {
        return fmt.Errorf("download backup: %w", err)
    }
    
    // Verify integrity
    if err := r.verifyBackup(backup); err != nil {
        return fmt.Errorf("verify backup: %w", err)
    }
    
    // Stop services
    if err := r.orchestrator.StopAll(); err != nil {
        return fmt.Errorf("stop services: %w", err)
    }
    
    // Restore each service
    for name, serviceBackup := range backup.Services {
        if err := r.restoreService(name, serviceBackup); err != nil {
            return fmt.Errorf("restore %s: %w", name, err)
        }
    }
    
    // Start services
    return r.orchestrator.StartAll()
}
```

## Monitoring and Alerting

### Alert Configuration

Configure monitoring alerts:

```yaml
alerts:
  - name: high_memory_usage
    condition: "process_memory_bytes > 0.9 * limit"
    severity: warning
    actions:
      - log
      - email
      
  - name: service_down
    condition: "up == 0"
    severity: critical
    actions:
      - log
      - pagerduty
      - restart_service
      
  - name: high_error_rate
    condition: "rate(errors[5m]) > 0.1"
    severity: warning
    actions:
      - log
      - slack
```

### Alert Handlers

Implement alert actions:

```go
type AlertHandler interface {
    Handle(alert Alert) error
}

type EmailHandler struct {
    smtp SMTPConfig
}

func (h *EmailHandler) Handle(alert Alert) error {
    message := fmt.Sprintf(`
        Alert: %s
        Severity: %s
        Service: %s
        Description: %s
        Time: %s
        
        Actions taken: %s
    `, alert.Name, alert.Severity, alert.Service, 
       alert.Description, alert.Time, alert.Actions)
    
    return h.smtp.Send(alert.Recipients, "Blackhole Alert", message)
}

type RestartHandler struct {
    orchestrator *Orchestrator
}

func (h *RestartHandler) Handle(alert Alert) error {
    if alert.Service == "" {
        return fmt.Errorf("no service specified for restart")
    }
    
    log.Printf("Restarting service %s due to alert %s", 
        alert.Service, alert.Name)
    
    return h.orchestrator.RestartService(alert.Service)
}
```

## Troubleshooting

### Common Issues

#### Service Won't Start

```bash
# Check logs
blackhole logs <service> --tail 100

# Verify configuration
blackhole config validate

# Check port availability
netstat -tlnp | grep 5000

# Check file permissions
ls -la /var/run/blackhole/

# Test in isolation
blackhole service --name identity --debug
```

#### High Memory Usage

```bash
# Check memory details
blackhole debug memory --service storage

# Dump heap profile
blackhole debug heap-dump --service storage

# Analyze memory leaks
go tool pprof storage-heap.prof

# Force garbage collection
blackhole resources gc --service storage --force
```

#### Performance Issues

```bash
# CPU profiling
blackhole debug cpu-profile --service ledger --duration 60s

# Trace requests
blackhole debug trace --service identity --duration 30s

# Check goroutine count
blackhole debug goroutines --service node

# Analyze slow queries
blackhole debug slow-queries --service indexer
```

### Debug Mode

Run services in debug mode:

```bash
# Enable debug logging
blackhole start --debug

# Enable verbose gRPC logging
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info

# Enable pprof endpoints
blackhole start --enable-pprof

# Trace specific requests
blackhole debug trace --filter "method=Authenticate"
```

### Emergency Procedures

#### Service Failure

```bash
# 1. Check service status
blackhole status <service>

# 2. Check recent logs
blackhole logs <service> --since 5m

# 3. Attempt restart
blackhole restart <service>

# 4. If restart fails, check resources
blackhole resources check --service <service>

# 5. Force kill if necessary
blackhole stop <service> --force

# 6. Clean up and restart
blackhole cleanup <service>
blackhole start <service>
```

#### Data Corruption

```bash
# 1. Stop affected service
blackhole stop <service>

# 2. Verify data integrity
blackhole verify --service <service>

# 3. If corruption confirmed, restore from backup
blackhole backup restore --service <service> --latest

# 4. If no backup, attempt repair
blackhole repair --service <service>

# 5. Start service
blackhole start <service>
```

#### Complete System Failure

```bash
# 1. Check system resources
df -h
free -m
top

# 2. Check for core dumps
find /var/crash -name "core.*"

# 3. Emergency stop all services
blackhole stop --emergency

# 4. Clear all data and restart
blackhole reset --confirm

# 5. Restore from backup
blackhole backup restore --latest
```

## Maintenance

### Regular Maintenance

Schedule regular maintenance tasks:

```yaml
maintenance:
  schedule:
    cleanup:
      interval: 1d
      tasks:
        - clean_temp_files
        - rotate_logs
        - compact_database
        
    optimize:
      interval: 1w
      tasks:
        - defragment_storage
        - rebuild_indexes
        - analyze_tables
        
    update:
      interval: 1m
      tasks:
        - check_updates
        - update_geoip
        - refresh_certificates
```

### Database Maintenance

```bash
# Compact database
blackhole maintenance compact

# Rebuild indexes
blackhole maintenance rebuild-indexes

# Analyze query performance
blackhole maintenance analyze-queries

# Vacuum database
blackhole maintenance vacuum
```

### Certificate Management

```bash
# Check certificate expiration
blackhole certs check

# Renew certificates
blackhole certs renew

# Rotate certificates
blackhole certs rotate

# Verify certificate chain
blackhole certs verify
```

## Performance Tuning

### System Tuning

Optimize system settings:

```bash
#!/bin/bash
# System tuning script

# Network settings
sysctl -w net.core.somaxconn=65535
sysctl -w net.ipv4.tcp_max_syn_backlog=65535
sysctl -w net.ipv4.ip_local_port_range="1024 65535"

# File descriptors
ulimit -n 65536

# Memory settings
echo 'vm.swappiness = 10' >> /etc/sysctl.conf
echo 'vm.dirty_ratio = 15' >> /etc/sysctl.conf

# Apply settings
sysctl -p
```

### Service Tuning

Optimize service configurations:

```yaml
performance:
  identity:
    connection_pool: 100
    cache_size: 1024MB
    query_timeout: 5s
    
  storage:
    buffer_size: 64MB
    compression: true
    async_writes: true
    
  ledger:
    batch_size: 1000
    sync_interval: 100ms
    max_pending: 10000
```

## Security Operations

### Security Monitoring

Monitor security events:

```bash
# Check authentication failures
blackhole security auth-failures --since 1h

# Monitor suspicious activity
blackhole security monitor --real-time

# Audit access logs
blackhole security audit --service identity

# Check firewall rules
blackhole security firewall list
```

### Incident Response

Security incident procedures:

```go
type SecurityIncident struct {
    ID          string
    Type        string
    Severity    string
    Service     string
    Description string
    Timestamp   time.Time
    Actions     []string
}

func (s *SecurityMonitor) HandleIncident(incident SecurityIncident) error {
    // Log incident
    s.logger.Error("Security incident detected",
        "id", incident.ID,
        "type", incident.Type,
        "severity", incident.Severity,
        "service", incident.Service,
    )
    
    // Execute response actions
    for _, action := range incident.Actions {
        if err := s.executeAction(action, incident); err != nil {
            s.logger.Error("Failed to execute action",
                "action", action,
                "error", err,
            )
        }
    }
    
    // Notify security team
    return s.notifySecurityTeam(incident)
}
```

## Automation

### Automated Tasks

```yaml
automation:
  tasks:
    - name: daily_backup
      schedule: "0 2 * * *"
      command: "blackhole backup create --incremental"
      
    - name: health_check
      schedule: "*/5 * * * *"
      command: "blackhole health check --all"
      
    - name: log_cleanup
      schedule: "0 0 * * *"
      command: "blackhole logs cleanup --older-than 7d"
```

### Runbooks

Automated runbooks for common procedures:

```go
type Runbook struct {
    Name        string
    Description string
    Triggers    []Trigger
    Steps       []Step
    Rollback    []Step
}

func (r *RunbookEngine) Execute(runbook Runbook) error {
    log.Printf("Executing runbook: %s", runbook.Name)
    
    // Execute steps
    for i, step := range runbook.Steps {
        log.Printf("Step %d: %s", i+1, step.Description)
        
        if err := r.executeStep(step); err != nil {
            log.Printf("Step failed: %v", err)
            
            // Execute rollback
            return r.rollback(runbook.Rollback[:i])
        }
    }
    
    log.Printf("Runbook completed successfully")
    return nil
}
```

## Best Practices

1. **Monitor Continuously**: Never run without monitoring
2. **Backup Regularly**: Test restores frequently
3. **Document Everything**: Keep runbooks updated
4. **Automate Repetitive Tasks**: Reduce human error
5. **Practice Incident Response**: Regular drills
6. **Rotate Credentials**: Regular security updates
7. **Capacity Planning**: Plan for growth
8. **Performance Baselines**: Know normal behavior
9. **Change Management**: Document all changes
10. **Team Training**: Keep skills current

## Conclusion

This operations guide provides comprehensive procedures for managing the Blackhole platform. Regular review and updates ensure smooth operations and quick incident response.