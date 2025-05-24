# Resource Monitoring and Metrics Design

*Date: January 19, 2025*

## Overview

The resource monitoring system provides real-time visibility into resource usage patterns, enabling the adaptive resource manager to make informed allocation decisions. This document outlines the monitoring architecture, metrics collection, and alerting mechanisms.

## Architecture

### 1. Core Components

```go
type ResourceMonitor struct {
    // Metrics collection
    collectors    map[string]MetricCollector
    aggregator    *MetricsAggregator
    
    // Time-series storage
    metricsStore  *TimeSeriesDB
    
    // Real-time processing
    stream        *MetricsStream
    analyzer      *ResourceAnalyzer
    
    // Alerting
    alertManager  *AlertManager
    
    // Dashboard
    dashboard     *ResourceDashboard
}
```

### 2. Metric Types

```go
// Resource metric definition
type ResourceMetric struct {
    Service    string
    Type       MetricType
    Value      float64
    Timestamp  time.Time
    Labels     map[string]string
}

type MetricType string

const (
    CPUUsage     MetricType = "cpu_usage_percent"
    MemoryUsage  MetricType = "memory_usage_bytes"
    DiskIO       MetricType = "disk_io_bytes"
    NetworkIO    MetricType = "network_io_bytes"
    GoroutineCount MetricType = "goroutine_count"
    RequestRate  MetricType = "request_rate"
    ErrorRate    MetricType = "error_rate"
    Latency      MetricType = "latency_ms"
)
```

## Metrics Collection

### 1. Service-Level Metrics

```go
type ServiceCollector struct {
    service     string
    resources   *AdaptiveResourceManager
    interval    time.Duration
}

func (c *ServiceCollector) Collect(ctx context.Context) {
    ticker := time.NewTicker(c.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            metrics := c.gatherMetrics()
            c.publish(metrics)
        case <-ctx.Done():
            return
        }
    }
}

func (c *ServiceCollector) gatherMetrics() []ResourceMetric {
    metrics := []ResourceMetric{}
    
    // CPU metrics
    cpu := c.getCPUUsage()
    metrics = append(metrics, ResourceMetric{
        Service:   c.service,
        Type:      CPUUsage,
        Value:     cpu,
        Timestamp: time.Now(),
        Labels: map[string]string{
            "tier": c.getTier(),
            "node": c.getNodeID(),
        },
    })
    
    // Memory metrics
    memory := c.getMemoryUsage()
    metrics = append(metrics, ResourceMetric{
        Service:   c.service,
        Type:      MemoryUsage,
        Value:     float64(memory),
        Timestamp: time.Now(),
    })
    
    // Add other metrics...
    
    return metrics
}
```

### 2. System-Level Metrics

```go
type SystemCollector struct {
    interval time.Duration
}

func (c *SystemCollector) Collect(ctx context.Context) {
    ticker := time.NewTicker(c.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            systemMetrics := c.gatherSystemMetrics()
            c.publish(systemMetrics)
        case <-ctx.Done():
            return
        }
    }
}

func (c *SystemCollector) gatherSystemMetrics() []SystemMetric {
    return []SystemMetric{
        {Type: "total_cpu", Value: c.getTotalCPU()},
        {Type: "total_memory", Value: c.getTotalMemory()},
        {Type: "disk_usage", Value: c.getDiskUsage()},
        {Type: "network_bandwidth", Value: c.getNetworkBandwidth()},
    }
}
```

## Real-Time Analysis

### 1. Streaming Processor

```go
type MetricsStream struct {
    input       chan ResourceMetric
    processors  []MetricProcessor
    output      chan ProcessedMetric
}

type MetricProcessor interface {
    Process(metric ResourceMetric) (*ProcessedMetric, error)
}

// Rolling average processor
type RollingAverageProcessor struct {
    window    time.Duration
    buffer    *CircularBuffer
}

func (p *RollingAverageProcessor) Process(metric ResourceMetric) (*ProcessedMetric, error) {
    p.buffer.Add(metric)
    
    average := p.calculateAverage()
    trend := p.calculateTrend()
    
    return &ProcessedMetric{
        Original: metric,
        Stats: ResourceStatistics{
            average: average,
            trend:   trend,
        },
    }, nil
}
```

### 2. Anomaly Detection

```go
type AnomalyDetector struct {
    zscore    float64
    baseline  map[string]*ServiceBaseline
}

func (d *AnomalyDetector) Detect(metric ResourceMetric) bool {
    baseline := d.baseline[metric.Service]
    if baseline == nil {
        return false
    }
    
    deviation := math.Abs(metric.Value - baseline.Mean)
    zscore := deviation / baseline.StdDev
    
    return zscore > d.zscore
}

type ServiceBaseline struct {
    Service   string
    Type      MetricType
    Mean      float64
    StdDev    float64
    UpdatedAt time.Time
}
```

## Time-Series Storage

### 1. Storage Design

```go
type TimeSeriesDB struct {
    storage   Storage
    retention RetentionPolicy
    compactor Compactor
}

type RetentionPolicy struct {
    Raw        time.Duration  // 1 hour
    Minute     time.Duration  // 24 hours
    Hour       time.Duration  // 7 days
    Day        time.Duration  // 30 days
}

func (db *TimeSeriesDB) Store(metric ResourceMetric) error {
    // Store raw data point
    err := db.storage.WritePoint(metric)
    if err != nil {
        return err
    }
    
    // Update aggregates
    return db.updateAggregates(metric)
}

func (db *TimeSeriesDB) Query(query TimeSeriesQuery) ([]DataPoint, error) {
    // Select appropriate resolution
    resolution := db.selectResolution(query.TimeRange)
    
    // Fetch data points
    return db.storage.Query(query, resolution)
}
```

### 2. Data Compaction

```go
type Compactor struct {
    interval time.Duration
}

func (c *Compactor) Compact(ctx context.Context) {
    ticker := time.NewTicker(c.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            c.compactOldData()
        case <-ctx.Done():
            return
        }
    }
}

func (c *Compactor) compactOldData() error {
    // Compact minute data to hourly
    minuteData := c.getMinuteData(time.Now().Add(-2 * time.Hour))
    hourlyAgg := c.aggregateToHourly(minuteData)
    
    if err := c.storeHourlyData(hourlyAgg); err != nil {
        return err
    }
    
    // Delete old minute data
    return c.deleteMinuteData(minuteData)
}
```

## Alerting System

### 1. Alert Rules

```go
type AlertRule struct {
    Name        string
    Service     string
    Metric      MetricType
    Condition   AlertCondition
    Threshold   float64
    Duration    time.Duration
    Severity    AlertSeverity
    Actions     []AlertAction
}

type AlertCondition string

const (
    Above   AlertCondition = "above"
    Below   AlertCondition = "below"
    Outside AlertCondition = "outside"
    Change  AlertCondition = "change"
)

// Example alert rules
var defaultAlertRules = []AlertRule{
    {
        Name:      "high_cpu_usage",
        Service:   "identity",
        Metric:    CPUUsage,
        Condition: Above,
        Threshold: 80,
        Duration:  5 * time.Minute,
        Severity:  Warning,
        Actions:   []AlertAction{LogAlert, NotifyOps},
    },
    {
        Name:      "memory_exhaustion",
        Service:   "storage",
        Metric:    MemoryUsage,
        Condition: Above,
        Threshold: 0.9, // 90% of allocated
        Duration:  1 * time.Minute,
        Severity:  Critical,
        Actions:   []AlertAction{LogAlert, NotifyOps, TriggerScaling},
    },
}
```

### 2. Alert Manager

```go
type AlertManager struct {
    rules    []AlertRule
    state    map[string]*AlertState
    actions  map[AlertAction]AlertHandler
}

func (m *AlertManager) Evaluate(metric ResourceMetric) {
    for _, rule := range m.rules {
        if rule.Service != metric.Service || rule.Metric != metric.Type {
            continue
        }
        
        triggered := m.checkCondition(metric, rule)
        m.updateState(rule, triggered)
        
        if m.shouldFire(rule) {
            m.fireAlert(rule, metric)
        }
    }
}

func (m *AlertManager) fireAlert(rule AlertRule, metric ResourceMetric) {
    alert := Alert{
        Rule:      rule,
        Metric:    metric,
        Timestamp: time.Now(),
        State:     Firing,
    }
    
    for _, action := range rule.Actions {
        handler := m.actions[action]
        handler.Handle(alert)
    }
}
```

## Resource Dashboard

### 1. Real-Time View

```go
type ResourceDashboard struct {
    cache    *DashboardCache
    streamer *MetricsStreamer
}

type DashboardData struct {
    Services         map[string]ServiceStatus
    ResourceUsage    ResourceUsageSummary
    AlertsSummary    AlertsSummary
    HistoricalTrends HistoricalData
}

func (d *ResourceDashboard) GetDashboard() *DashboardData {
    return &DashboardData{
        Services:         d.getServiceStatuses(),
        ResourceUsage:    d.getResourceUsage(),
        AlertsSummary:    d.getAlertsSummary(),
        HistoricalTrends: d.getHistoricalTrends(),
    }
}

type ServiceStatus struct {
    Name            string
    Tier            ServiceTier
    CurrentUsage    ResourceUsage
    AllocatedQuota  ResourceAmount
    GuaranteedQuota ResourceAmount
    BurstUsage      ResourceAmount
    Health          HealthStatus
}
```

### 2. Historical Analysis

```go
type HistoricalAnalyzer struct {
    store TimeSeriesDB
}

func (a *HistoricalAnalyzer) AnalyzeTrends(service string, window time.Duration) *TrendAnalysis {
    endTime := time.Now()
    startTime := endTime.Add(-window)
    
    query := TimeSeriesQuery{
        Service:   service,
        Metrics:   []MetricType{CPUUsage, MemoryUsage},
        StartTime: startTime,
        EndTime:   endTime,
    }
    
    data, _ := a.store.Query(query)
    
    return &TrendAnalysis{
        Service:        service,
        Window:         window,
        CPUTrend:       a.calculateTrend(data, CPUUsage),
        MemoryTrend:    a.calculateTrend(data, MemoryUsage),
        PeakUsage:      a.findPeaks(data),
        Recommendations: a.generateRecommendations(data),
    }
}
```

## Integration Points

### 1. Service Integration

```go
// Service-side integration
type MonitoredService struct {
    BaseService
    monitor *ResourceMonitor
}

func (s *MonitoredService) ExecuteOperation(ctx context.Context, op Operation) error {
    // Record operation start
    s.monitor.RecordOperationStart(op.ID)
    
    // Execute operation
    err := s.performOperation(ctx, op)
    
    // Record operation end
    s.monitor.RecordOperationEnd(op.ID, err)
    
    return err
}
```

### 2. Resource Manager Integration

```go
// Adaptive resource manager uses monitoring data
func (m *AdaptiveResourceManager) adjustQuotas() {
    for service, quota := range m.adaptiveQuotas {
        // Get historical metrics
        metrics := m.monitor.GetServiceMetrics(service, 24*time.Hour)
        
        // Calculate new guarantee
        newGuarantee := m.calculateGuarantee(service, metrics)
        
        // Update quota with dampening
        adjustedGuarantee := m.dampener.Dampen(
            quota.currentGuarantee,
            newGuarantee,
        )
        
        quota.currentGuarantee = adjustedGuarantee
        quota.lastAdjusted = time.Now()
    }
}
```

## Performance Considerations

### 1. Metric Collection Overhead
- Use sampling for high-frequency metrics
- Batch metric writes to reduce I/O
- Implement back-pressure for overload protection

### 2. Storage Optimization
- Compress historical data
- Use appropriate data retention policies
- Implement efficient time-series compression

### 3. Query Performance
- Pre-aggregate common queries
- Use materialized views for dashboards
- Implement query result caching

## Configuration

```yaml
monitoring:
  # Collection intervals
  intervals:
    service_metrics: 1s
    system_metrics: 5s
    resource_usage: 10s
  
  # Storage retention
  retention:
    raw: 1h
    minute: 24h
    hour: 7d
    day: 30d
  
  # Alerting
  alerting:
    enabled: true
    evaluation_interval: 30s
    notification_channels:
      - type: log
        severity: [warning, critical]
      - type: webhook
        url: "https://ops.example.com/alerts"
        severity: [critical]
  
  # Dashboard
  dashboard:
    refresh_interval: 5s
    history_window: 24h
    cache_ttl: 1s
```

## Benefits

1. **Real-Time Visibility**: Immediate insight into resource usage
2. **Proactive Management**: Alerts before resource exhaustion
3. **Historical Analysis**: Trends and patterns for optimization
4. **Adaptive Tuning**: Data-driven quota adjustments
5. **Operational Excellence**: Better platform reliability

## Conclusion

The resource monitoring system provides comprehensive visibility into the Blackhole platform's resource usage, enabling intelligent allocation decisions and maintaining optimal performance across all services.