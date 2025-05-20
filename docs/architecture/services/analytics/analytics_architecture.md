# Blackhole Analytics Architecture

This document provides a comprehensive overview of the Blackhole analytics system architecture, including design decisions, implementation strategies, and operational considerations.

## Overview

The Blackhole analytics system runs as a dedicated subprocess, providing distributed, privacy-preserving metrics collection and analysis that delivers insights into content usage, system performance, and network health. As an isolated subprocess, it maintains the decentralized nature of the platform while communicating with other services via gRPC.

## Core Design Principles

1. **Decentralized Analytics**: Each node independently collects and stores its own metrics
2. **Privacy-First**: All data collection follows strict privacy guidelines with user consent
3. **Resource Efficiency**: Minimal impact on node performance and storage
4. **Federated Architecture**: Optional aggregation of anonymized data across nodes
5. **Real-time Processing**: Stream processing for immediate insights
6. **Scalable Design**: Handles growth from individual nodes to network-wide analytics
7. **Embedded Database**: Self-contained analytics with no external dependencies
8. **Process Isolation**: Runs as independent subprocess with dedicated resources

## Subprocess Architecture

The Analytics Service runs as an isolated subprocess with dedicated resources for metrics collection and analysis:

```mermaid
graph TD
    subgraph Orchestrator
        Orch[Process Manager]
        SD[Service Discovery]
        Mon[Monitor]
    end
    
    subgraph Analytics Subprocess
        gRPC[gRPC Server :9006]
        Collector[Metrics Collector]
        EventProc[Event Processor]
        TSDB[Time-Series DB]
        Engine[Analytics Engine]
        AlertMgr[Alert Manager]
    end
    
    subgraph Services
        Identity[Identity Service]
        Storage[Storage Service]
        Social[Social Service]
        Ledger[Ledger Service]
    end
    
    Orch -->|spawn| Analytics Subprocess
    SD -->|register| gRPC
    Mon -->|health check| gRPC
    
    Services -->|metrics| Analytics Subprocess
    Analytics Subprocess -->|queries| Services
```

### Service Entry Point

```go
// cmd/blackhole/service/analytics/main.go
package main

import (
    "context"
    "flag"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/blackhole/internal/services/analytics"
    "github.com/blackhole/pkg/api/analytics/v1"
    "google.golang.org/grpc"
)

var (
    port       = flag.Int("port", 9006, "gRPC port")
    unixSocket = flag.String("unix-socket", "/tmp/blackhole-analytics.sock", "Unix socket path")
    config     = flag.String("config", "", "Configuration file path")
)

func main() {
    flag.Parse()
    
    // Initialize service
    cfg, err := analytics.LoadConfig(*config)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    service, err := analytics.New(cfg)
    if err != nil {
        log.Fatalf("Failed to create service: %v", err)
    }
    
    // Initialize embedded database
    if err := service.InitializeDatabase(context.Background()); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    
    // Create gRPC server
    grpcServer := grpc.NewServer(
        grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB
        grpc.MaxSendMsgSize(10 * 1024 * 1024),
    )
    
    // Register service
    analyticsv1.RegisterAnalyticsServiceServer(grpcServer, service)
    
    // Listen on Unix socket for local communication
    unixListener, err := net.Listen("unix", *unixSocket)
    if err != nil {
        log.Fatalf("Failed to listen on unix socket: %v", err)
    }
    defer os.Remove(*unixSocket)
    
    // Listen on TCP for remote communication
    tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
    if err != nil {
        log.Fatalf("Failed to listen on TCP: %v", err)
    }
    
    // Start background processors
    go service.StartEventProcessor(context.Background())
    go service.StartAggregator(context.Background())
    go service.StartAlertManager(context.Background())
    
    // Handle shutdown gracefully
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    go func() {
        <-sigChan
        log.Println("Shutting down analytics service...")
        service.Shutdown()
        grpcServer.GracefulStop()
        cancel()
    }()
    
    // Start serving
    go func() {
        log.Printf("Analytics service listening on Unix socket: %s", *unixSocket)
        if err := grpcServer.Serve(unixListener); err != nil {
            log.Fatalf("Failed to serve Unix socket: %v", err)
        }
    }()
    
    log.Printf("Analytics service listening on TCP port: %d", *port)
    if err := grpcServer.Serve(tcpListener); err != nil {
        log.Fatalf("Failed to serve TCP: %v", err)
    }
}
```

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Node-Level Analytics                      │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Metrics   │  │   Event     │  │  Time-Series│             │
│  │  Collector  │  │  Processor  │  │   Database  │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│         │                │                │                     │
│         └────────────────┴────────────────┘                     │
│                          │                                      │
│                    ┌─────┴─────┐                                │
│                    │Analytics  │                                │
│                    │  Engine   │                                │
│                    └─────┬─────┘                                │
│                          │                                      │
│    ┌──────────┬──────────┴──────────┬──────────┐               │
│    │          │                     │          │               │
│    ▼          ▼                     ▼          ▼               │
│  ┌─────┐  ┌─────┐             ┌─────┐  ┌─────┐               │
│  │Local│  │Real-│             │Batch│  │Alert│               │
│  │Store│  │time │             │Proc │  │Mgmt │               │
│  └─────┘  └─────┘             └─────┘  └─────┘               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Optional Federation
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Network-Level Analytics                      │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │Aggregation  │  │Cross-Node   │  │ Global      │             │
│  │  Service    │  │Analytics    │  │ Dashboard   │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. Metrics Collector

Responsible for gathering raw metrics from various node services:

- Content access patterns
- Storage operations metrics
- Network performance data
- System resource utilization
- API request metrics
- P2P protocol statistics

#### 2. Event Processor

Processes raw events into structured analytics data:

- Event normalization and validation
- Privacy filtering and anonymization
- Event correlation and enrichment
- Stream processing for real-time metrics
- Event routing to appropriate handlers

#### 3. Embedded Time-Series Database

Local storage for analytics data with the following characteristics:

- Lightweight embedded database (e.g., DuckDB, BadgerDB, or custom TSDB)
- Optimized for time-series data storage and retrieval
- Configurable retention policies
- Automatic data compaction and downsampling
- Resource-constrained operation mode

#### 4. Analytics Engine

Core processing engine for analytics operations:

- Statistical analysis and aggregation
- Pattern detection and anomaly identification
- Trend analysis and forecasting
- Custom metric calculations
- Privacy-preserving computations

#### 5. Local Storage Manager

Manages the embedded database and local analytics data:

- Data retention policy enforcement
- Storage space monitoring and management
- Data compaction and archival
- Backup and recovery operations
- Performance optimization

#### 6. Real-time Processing

Handles streaming analytics for immediate insights:

- Live metric dashboards
- Real-time alerting
- Immediate anomaly detection
- Stream processing pipelines
- Low-latency event handling

#### 7. Batch Processing

Performs scheduled analytics computations:

- Historical data analysis
- Complex aggregations
- Report generation
- Data export preparation
- Long-term trend analysis

#### 8. Alert Management

Monitors metrics and generates alerts:

- Configurable alert thresholds
- Multi-channel notifications
- Alert correlation and deduplication
- Escalation policies
- Alert history tracking

## Embedded Database Design

### Database Selection Criteria

1. **Lightweight footprint**: Minimal memory and CPU usage
2. **Time-series optimization**: Efficient storage and retrieval of time-stamped data
3. **Embedded capability**: Runs within the node process
4. **Reliability**: ACID compliance and crash recovery
5. **Performance**: Fast writes and efficient range queries
6. **Maintenance-free**: Minimal operational overhead

### Recommended Database Options

1. **Primary Choice: DuckDB**
   - Columnar storage ideal for analytics
   - SQL interface for complex queries
   - Excellent compression ratios
   - ACID compliant
   - Zero-dependency embedded database

2. **Alternative: BadgerDB**
   - Key-value store with LSM tree
   - High write throughput
   - Native Go implementation
   - Built-in compression
   - Simple API

3. **Specialized Option: Victoria Metrics TSDB**
   - Purpose-built for metrics
   - Excellent compression
   - Fast ingestion rates
   - Minimal resource usage
   - PromQL compatibility

### Storage Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Embedded TSDB Storage                     │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Hot Tier    │  │  Warm Tier   │  │  Cold Tier   │      │
│  │  (1-7 days)  │  │  (7-30 days) │  │  (30+ days)  │      │
│  │              │  │              │  │              │      │
│  │  Raw Data    │  │  Downsampled │  │  Aggregated  │      │
│  │  Full Detail │  │  5min Avg    │  │  Hourly Avg  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                             │
│  ┌──────────────────────────────────────────────────┐      │
│  │            Retention Manager                     │      │
│  │                                                  │      │
│  │  - Automatic data aging                          │      │
│  │  - Compression and compaction                    │      │
│  │  - Space monitoring                              │      │
│  │  - Emergency cleanup                             │      │
│  └──────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### Data Retention Policies

1. **Hot Tier (Recent Data)**
   - Duration: 1-7 days
   - Resolution: Full detail (1-second precision)
   - Storage: Uncompressed or lightly compressed
   - Purpose: Real-time dashboards and immediate analysis

2. **Warm Tier (Recent History)**
   - Duration: 7-30 days
   - Resolution: 5-minute averages
   - Storage: Compressed
   - Purpose: Short-term trending and analysis

3. **Cold Tier (Historical Data)**
   - Duration: 30+ days (configurable)
   - Resolution: Hourly or daily aggregates
   - Storage: Highly compressed
   - Purpose: Long-term trends and historical comparison

4. **Archive Tier (Optional)**
   - Duration: Permanent
   - Resolution: Daily/weekly summaries
   - Storage: External storage or IPFS
   - Purpose: Compliance and long-term analysis

## Resource Management

### Performance Optimization

1. **Resource Limits**
   - Configurable memory cap for analytics (e.g., 10% of node memory)
   - CPU throttling during high load
   - I/O rate limiting for database operations
   - Separate thread pool for analytics

2. **Graceful Degradation**
   - Sampling under high load
   - Temporary pause of non-critical metrics
   - Prioritization of essential metrics
   - Fallback to basic metrics only

3. **Storage Management**
   - Automatic cleanup when disk usage exceeds threshold
   - Progressive data aging based on available space
   - Emergency purge of old data
   - Compression ratios monitoring

### Operational Modes

1. **Full Analytics Mode**
   - All metrics collected
   - Real-time processing enabled
   - Complete historical data retention
   - Advanced analytics features

2. **Balanced Mode (Default)**
   - Core metrics collection
   - Moderate retention periods
   - Balanced resource usage
   - Standard analytics features

3. **Minimal Mode**
   - Essential metrics only
   - Short retention periods
   - Minimal resource usage
   - Basic analytics features

4. **Disabled Mode**
   - No metrics collection
   - Database shutdown
   - Zero analytics overhead
   - Emergency failsafe

## Privacy Architecture

### Privacy Layers

1. **Collection Layer**
   - Opt-in/opt-out mechanisms
   - Granular consent management
   - Data minimization
   - Purpose limitation

2. **Processing Layer**
   - Client-side anonymization
   - Differential privacy algorithms
   - K-anonymity enforcement
   - Sensitive data filtering

3. **Storage Layer**
   - Encryption at rest
   - Access control
   - Audit logging
   - Secure deletion

4. **Federation Layer**
   - Anonymous aggregation only
   - No raw data sharing
   - Cryptographic proofs
   - Zero-knowledge protocols

### Privacy Controls

```
┌─────────────────────────────────────────────────────────────┐
│                    Privacy Control Flow                     │
│                                                             │
│  Raw Event → Privacy Filter → Anonymizer → Storage          │
│                    │               │                        │
│                    ▼               ▼                        │
│              Consent Check    Add Noise                     │
│                    │               │                        │
│                    ▼               ▼                        │
│              Drop if Denied   Differential                  │
│                              Privacy                        │
└─────────────────────────────────────────────────────────────┘
```

## Federation Architecture

### Federation Levels

1. **No Federation (Default)**
   - Completely local analytics
   - No data sharing
   - Full data sovereignty
   - Maximum privacy

2. **Anonymous Aggregation**
   - Share aggregated metrics only
   - No individual data points
   - Statistical summaries
   - Network-wide insights

3. **Selective Sharing**
   - User-controlled data sharing
   - Specific metrics only
   - Partner node exchanges
   - Research participation

### Federation Protocol

```
┌─────────────────────────────────────────────────────────────┐
│                    Federation Protocol                      │
│                                                             │
│  Node A                                          Node B     │
│    │                                               │        │
│    ├──── Aggregation Request ─────────────────────►│        │
│    │                                               │        │
│    │◄─── Capability Negotiation ──────────────────┤        │
│    │                                               │        │
│    ├──── Privacy Parameters ──────────────────────►│        │
│    │                                               │        │
│    │◄─── Aggregated Data ─────────────────────────┤        │
│    │                                               │        │
│    ├──── Verification ────────────────────────────►│        │
│    │                                               │        │
└─────────────────────────────────────────────────────────────┘
```

## Query Architecture

### Query Types

1. **Real-time Queries**
   - Live metric values
   - Current system state
   - Active alerts
   - Recent events

2. **Historical Queries**
   - Time-range aggregations
   - Trend analysis
   - Comparative analysis
   - Pattern detection

3. **Analytical Queries**
   - Complex aggregations
   - Statistical analysis
   - Predictive models
   - Correlation analysis

### Query Optimization

1. **Indexing Strategy**
   - Time-based primary index
   - Secondary indices for common dimensions
   - Bloom filters for existence checks
   - Materialized views for common queries

2. **Caching Layer**
   - Result caching for expensive queries
   - Partial result caching
   - Time-based cache invalidation
   - LRU eviction policy

3. **Query Planning**
   - Cost-based optimization
   - Parallel query execution
   - Push-down predicates
   - Adaptive query plans

## Integration Points

### Service Integration

1. **Storage Service**
   - Content access metrics
   - Storage operation latencies
   - Replication statistics
   - Provider performance

2. **Network Service**
   - P2P connection metrics
   - Bandwidth utilization
   - Peer discovery statistics
   - Protocol performance

3. **Identity Service**
   - Authentication metrics
   - DID operation statistics
   - Credential verification times
   - Access pattern analysis

4. **Ledger Service**
   - Transaction metrics
   - Smart contract interactions
   - Gas consumption patterns
   - Economic indicators

### API Integration

```
┌─────────────────────────────────────────────────────────────┐
│                     Analytics API                           │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   Metrics   │  │   Query     │  │   Export    │         │
│  │   Ingestion │  │   Engine    │  │   Service   │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   Real-time │  │   Alert     │  │   Dashboard │         │
│  │   Stream    │  │   API       │  │   API       │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

## Monitoring and Alerting

### Alert Categories

1. **System Alerts**
   - Node health issues
   - Resource exhaustion
   - Service failures
   - Performance degradation

2. **Content Alerts**
   - Availability issues
   - Access anomalies
   - Replication failures
   - Content violations

3. **Security Alerts**
   - Suspicious access patterns
   - Authentication failures
   - DDoS indicators
   - Anomalous behavior

4. **Business Alerts**
   - Usage thresholds
   - Economic indicators
   - Growth metrics
   - SLA violations

### Alert Processing

```
┌─────────────────────────────────────────────────────────────┐
│                    Alert Processing Flow                    │
│                                                             │
│  Metric → Threshold → Alert → Dedup → Route → Notify        │
│              Check     Gen                                  │
│                │         │        │       │        │        │
│                ▼         ▼        ▼       ▼        ▼        │
│           Conditions  Create   Correlate  Channel  User     │
│                      Alert    w/ Others  Select            │
└─────────────────────────────────────────────────────────────┘
```

## Disaster Recovery

### Backup Strategy

1. **Continuous Backup**
   - Incremental backups
   - Point-in-time recovery
   - Off-site replication
   - Encrypted archives

2. **Recovery Procedures**
   - Database restoration
   - Configuration recovery
   - State reconciliation
   - Service restart

3. **Data Export**
   - Regular data exports
   - Multiple format support
   - Compression options
   - Integrity verification

### Failover Mechanisms

1. **Local Failover**
   - Automatic database recovery
   - Service restart procedures
   - Data integrity checks
   - Graceful degradation

2. **Node Migration**
   - Export analytics data
   - Transfer to new node
   - Import and verification
   - Service continuity

## Performance Benchmarks

### Expected Performance

1. **Ingestion Rates**
   - 10,000+ metrics/second per node
   - Sub-millisecond latency
   - Batch optimization
   - Backpressure handling

2. **Query Performance**
   - Real-time queries: <100ms
   - Historical queries: <1s
   - Complex analytics: <10s
   - Dashboard refresh: <500ms

3. **Storage Efficiency**
   - 10:1 compression ratio
   - <1GB/day for typical node
   - Automatic cleanup
   - Predictable growth

## Implementation Roadmap

### Phase 1: Foundation (4 weeks)
- Embedded database selection and integration
- Basic metrics collection
- Local storage management
- Simple query interface

### Phase 2: Analytics Engine (4 weeks)
- Real-time processing pipeline
- Batch analytics implementation
- Alert system development
- Dashboard API

### Phase 3: Privacy & Federation (3 weeks)
- Privacy controls implementation
- Anonymous aggregation
- Federation protocol
- Consent management

### Phase 4: Advanced Features (3 weeks)
- Complex analytics queries
- Predictive analytics
- Advanced visualizations
- Performance optimization

### Phase 5: Operations (2 weeks)
- Monitoring and alerting
- Backup and recovery
- Documentation
- Testing and validation

## Security Considerations

1. **Data Protection**
   - Encryption at rest
   - Access control lists
   - Audit trails
   - Secure deletion

2. **Query Security**
   - Query validation
   - Resource limits
   - Injection prevention
   - Rate limiting

3. **Federation Security**
   - Mutual authentication
   - Encrypted transport
   - Data validation
   - Trust management

## Future Enhancements

1. **Machine Learning Integration**
   - Anomaly detection models
   - Predictive analytics
   - Pattern recognition
   - Automated insights

2. **Advanced Visualization**
   - Real-time dashboards
   - Interactive reports
   - Mobile applications
   - VR/AR interfaces

3. **Cross-Network Analytics**
   - Multi-network support
   - Comparative analysis
   - Industry benchmarks
   - Ecosystem insights

## Process Resource Management

The Analytics Service subprocess has dedicated resources optimized for metrics processing:

### Resource Configuration

```go
// Analytics service resource limits
type AnalyticsServiceConfig struct {
    ProcessLimits ProcessResourceLimits {
        CPUQuota    "150%"         // 1.5 CPU cores
        MemoryLimit "1GB"          // 1GB memory limit
        IOWeight    100            // Standard IO priority
        Nice        5              // Slightly lower priority
    }
    
    // Embedded database configuration
    DatabaseConfig struct {
        Type        string        // "duckdb" or "badgerdb"
        MaxSize     int64         // 10GB max database size
        CacheSize   int           // 256MB cache
        Compression bool          // Enable compression
    }
}

// Monitor analytics health and resources
func (a *AnalyticsService) MonitorHealth(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            stats := a.getProcessStats()
            if stats.MemoryMB > a.config.MemoryWarning {
                log.Warnf("Analytics memory usage high: %d MB", stats.MemoryMB)
                a.triggerGC()
            }
            
            // Monitor database size
            dbSize := a.database.GetSize()
            if dbSize > a.config.DatabaseConfig.MaxSize * 0.9 {
                a.performDataAging()
            }
            
        case <-ctx.Done():
            return
        }
    }
}
```

### Resource Isolation Benefits

1. **Process Isolation**: Analytics operations don't affect service performance
2. **Memory Control**: Time-series data managed independently
3. **CPU Management**: Aggregations don't impact system responsiveness
4. **I/O Isolation**: Database operations isolated from service I/O
5. **Crash Recovery**: Analytics failures don't bring down services

## gRPC Integration

The Analytics Service communicates with other services for metrics collection:

```go
type AnalyticsService struct {
    // Core components
    collector    *MetricsCollector
    processor    *EventProcessor
    database     *TimeSeriesDB
    alertManager *AlertManager
    
    // Metrics ingestion
    metricsQueue chan *Metric
    eventQueue   chan *Event
}

// Collect metrics from other services
func (a *AnalyticsService) CollectMetrics(stream analyticsv1.Analytics_CollectMetricsServer) error {
    for {
        metric, err := stream.Recv()
        if err == io.EOF {
            return stream.SendAndClose(&analyticsv1.CollectResponse{
                Status: "success",
            })
        }
        if err != nil {
            return err
        }
        
        // Apply privacy filters
        filtered := a.applyPrivacyFilter(metric)
        
        // Queue for processing
        select {
        case a.metricsQueue <- filtered:
        case <-stream.Context().Done():
            return stream.Context().Err()
        }
    }
}

// Query analytics data
func (a *AnalyticsService) Query(ctx context.Context, req *analyticsv1.QueryRequest) (*analyticsv1.QueryResponse, error) {
    // Validate query
    if err := a.validateQuery(req); err != nil {
        return nil, err
    }
    
    // Execute query against time-series database
    results, err := a.database.Query(ctx, req.Query, req.TimeRange)
    if err != nil {
        return nil, err
    }
    
    return &analyticsv1.QueryResponse{
        Results: results,
        Metadata: &analyticsv1.QueryMetadata{
            ExecutionTime: time.Since(start).Milliseconds(),
            RowsReturned:  int64(len(results)),
        },
    }, nil
}
```

## Service Configuration

```yaml
analytics_service:
  # Service configuration
  service:
    name: "analytics"
    port: 9006
    unix_socket: "/tmp/blackhole-analytics.sock"
    log_level: "info"
    
  # Process management
  process:
    cpu_limit: "150%"          # 1.5 CPU cores
    memory_limit: "1GB"        # 1GB memory limit
    restart_policy: "on_failure"
    restart_delay: "10s"
    health_check_interval: "30s"
    
  # Database configuration
  database:
    type: "duckdb"             # Embedded columnar database
    path: "/var/lib/blackhole/analytics.db"
    max_size: "10GB"
    cache_size: "256MB"
    compression: true
    
  # Retention policies
  retention:
    hot_tier: "7d"             # Full resolution
    warm_tier: "30d"           # 5-minute aggregates
    cold_tier: "365d"          # Hourly aggregates
    
  # Privacy settings
  privacy:
    anonymization: true
    differential_privacy: true
    k_anonymity: 5
    noise_level: 0.1
    
  # Collection settings
  collection:
    batch_size: 1000
    flush_interval: "10s"
    max_queue_size: 10000
    
  # Alert configuration
  alerts:
    check_interval: "1m"
    evaluation_window: "5m"
    max_alerts: 1000
```

## Subprocess Benefits

1. **Resource Isolation**: Analytics workloads don't impact services
2. **Independent Scaling**: Can allocate more resources to analytics
3. **Fault Tolerance**: Crashes don't affect core platform functionality
4. **Security**: Process-level boundaries for sensitive metrics
5. **Flexibility**: Can use different databases per deployment
6. **Monitoring**: Process-specific health metrics

---

This analytics architecture provides Blackhole with a robust, privacy-preserving, and scalable analytics system that maintains the platform's decentralized principles while delivering valuable insights. Running as an isolated subprocess ensures that intensive analytics operations never impact service performance, while the embedded database design keeps the system self-contained and operationally simple.