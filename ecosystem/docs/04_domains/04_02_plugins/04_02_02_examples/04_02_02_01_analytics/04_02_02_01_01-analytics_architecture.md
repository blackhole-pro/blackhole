# Blackhole Analytics Architecture

*Version: 2.0*  
*Last Updated: May 23, 2025*  
*Status: Design Phase*

## Table of Contents

1. [Overview](#overview)
2. [Design Philosophy](#design-philosophy)
3. [Architecture Components](#architecture-components)
4. [OpenTelemetry Foundation](#opentelemetry-foundation)
5. [Analytics Service Design](#analytics-service-design)
6. [Data Collection Strategy](#data-collection-strategy)
7. [Privacy Architecture](#privacy-architecture)
8. [Storage and Processing](#storage-and-processing)
9. [Implementation Phases](#implementation-phases)
10. [Integration Patterns](#integration-patterns)
11. [Performance Considerations](#performance-considerations)
12. [Future Evolution](#future-evolution)

---

## Overview

Blackhole's analytics architecture provides data-driven insights for optimizing network performance, understanding system behavior, and supporting operational decisions while maintaining strict privacy guarantees. The system uses OpenTelemetry as the foundational layer for data collection and processing.

### Key Principles

- **Privacy-First**: Zero PII collection, differential privacy for sensitive metrics
- **Problem-Driven**: Implement analytics only to solve specific identified problems
- **Local-First**: Each node operates independently with optional network sharing
- **Performance-Conscious**: Minimal overhead on core P2P functionality
- **Extensible**: Foundation supports incremental feature addition

---

## Design Philosophy

### Problem-Driven Analytics

Instead of collecting comprehensive metrics "just in case," Blackhole analytics focuses on solving specific operational challenges:

```
Problem Identification → Targeted Analytics → Data-Driven Solution → Metric Retention/Removal
```

### Privacy by Design

All analytics respect user privacy through:
- **No Individual Tracking**: Only aggregate, anonymized data
- **Differential Privacy**: Mathematical privacy guarantees for sensitive metrics
- **Opt-in Network Sharing**: Users control what data leaves their node
- **Automatic Expiration**: Configurable data retention limits

### Operational Focus

Analytics serve three primary purposes:
1. **Performance Optimization**: Identify and resolve system bottlenecks
2. **Network Health**: Monitor and maintain P2P network stability
3. **Development Insights**: Support feature development and debugging

---

## Architecture Components

### System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Blackhole Node                           │
├─────────────────────────────────────────────────────────────┤
│  Services (Identity, Node, Social, etc.)                   │
│       │                                                     │
│       ▼ (OpenTelemetry SDK)                                │
├─────────────────────────────────────────────────────────────┤
│  Analytics Service (Subprocess)                            │
│  ┌─────────────────┬──────────────────┬─────────────────┐   │
│  │ Data Collector  │ Stream Processor │ Privacy Engine  │   │
│  └─────────────────┴──────────────────┴─────────────────┘   │
│       │                    │                    │           │
│       ▼                    ▼                    ▼           │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │            DuckDB Analytics Database                    │ │
│  └─────────────────────────────────────────────────────────┘ │
│       │                                                     │
│       ▼                                                     │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │         Dashboard & API Layer                           │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
       │ (Optional, Privacy-Preserving)
       ▼
┌─────────────────────────────────────────────────────────────┐
│              Federated Network Analytics                    │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

#### Analytics Service (Subprocess)
- **Data Collector**: Receives telemetry data from OpenTelemetry
- **Stream Processor**: Real-time analysis and aggregation
- **Privacy Engine**: Applies privacy-preserving transformations
- **Storage Manager**: Manages DuckDB analytics database
- **Query Engine**: Serves analytics queries to dashboard and APIs

#### OpenTelemetry Foundation
- **Instrumentation**: Automatic and manual metric collection
- **Data Pipeline**: Standardized data transport and formatting
- **Sampling**: Intelligent data sampling for performance
- **Export**: Unified export to analytics service

---

## OpenTelemetry Foundation

### Why OpenTelemetry for Analytics

OpenTelemetry provides the perfect foundation for Blackhole analytics:

1. **Unified Data Model**: Consistent metrics, traces, and logs
2. **Vendor Neutral**: No lock-in to specific analytics platforms
3. **Rich Instrumentation**: Automatic gRPC, HTTP, and database instrumentation
4. **Flexible Pipeline**: Configurable processing and export
5. **Industry Standard**: Future-proof technology choice

### Foundation Setup

```go
// internal/analytics/foundation.go
package analytics

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
)

type Foundation struct {
    meterProvider  *metric.MeterProvider
    traceProvider  *trace.TracerProvider
    resource       *resource.Resource
    exporter       metric.Exporter
}

func NewFoundation(ctx context.Context, config *Config) (*Foundation, error) {
    // Resource identification
    res := resource.NewWithAttributes(
        semconv.SchemaURL,
        semconv.ServiceNameKey.String("blackhole-analytics"),
        semconv.ServiceVersionKey.String(version.Get()),
        semconv.ServiceInstanceIDKey.String(nodeID),
    )
    
    // Metric exporter to analytics service
    exporter, err := otlpmetricgrpc.New(ctx,
        otlpmetricgrpc.WithEndpoint(config.AnalyticsEndpoint),
        otlpmetricgrpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create metric exporter: %w", err)
    }
    
    // Meter provider with sampling and filtering
    meterProvider := metric.NewMeterProvider(
        metric.WithResource(res),
        metric.WithReader(metric.NewPeriodicReader(
            exporter,
            metric.WithInterval(config.CollectionInterval),
        )),
        metric.WithView(privacyViews()...),
    )
    
    otel.SetMeterProvider(meterProvider)
    
    return &Foundation{
        meterProvider: meterProvider,
        resource:      res,
        exporter:      exporter,
    }, nil
}

// Privacy-preserving views that aggregate sensitive data
func privacyViews() []metric.View {
    return []metric.View{
        // Aggregate peer metrics by region, not individual peers
        metric.NewView(
            metric.Instrument{Name: "p2p.peer.connection"},
            metric.Stream{
                Aggregation: metric.AggregationSum{},
                AttributeFilter: attribute.NewSet(
                    attribute.String("region", ""),
                    // Remove peer_id attribute for privacy
                ),
            },
        ),
        // Bucket content sizes to prevent fingerprinting
        metric.NewView(
            metric.Instrument{Name: "content.size"},
            metric.Stream{
                Aggregation: metric.AggregationExplicitBucketHistogram{
                    Boundaries: []float64{1024, 10240, 102400, 1048576}, // KB buckets
                },
            },
        ),
    }
}
```

### Service Integration Pattern

```go
// Each service uses OpenTelemetry for analytics
package node

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
)

type NodeService struct {
    // Analytics instrumentation
    meter               metric.Meter
    connectionCounter   metric.Int64Counter
    discoveryLatency    metric.Float64Histogram
    bandwidthGauge      metric.Int64UpDownCounter
}

func NewNodeService() (*NodeService, error) {
    meter := otel.Meter("blackhole-node")
    
    connectionCounter, err := meter.Int64Counter(
        "p2p.connections.total",
        metric.WithDescription("Total P2P connections established"),
        metric.WithUnit("1"),
    )
    if err != nil {
        return nil, err
    }
    
    discoveryLatency, err := meter.Float64Histogram(
        "p2p.discovery.latency",
        metric.WithDescription("Peer discovery latency"),
        metric.WithUnit("ms"),
        metric.WithExplicitBucketBoundaries(10, 50, 100, 500, 1000, 5000),
    )
    if err != nil {
        return nil, err
    }
    
    return &NodeService{
        meter:             meter,
        connectionCounter: connectionCounter,
        discoveryLatency:  discoveryLatency,
    }, nil
}

// Analytics-aware method implementation
func (n *NodeService) ConnectToPeer(ctx context.Context, addr peer.AddrInfo) error {
    start := time.Now()
    
    // Core business logic
    err := n.p2pHost.Connect(ctx, addr)
    
    // Analytics collection (privacy-safe)
    if err == nil {
        n.connectionCounter.Add(ctx, 1, 
            metric.WithAttributes(
                attribute.String("peer_type", classifyPeer(addr)),
                attribute.String("connection_method", "manual"),
            ))
    }
    
    n.discoveryLatency.Record(ctx, 
        float64(time.Since(start).Milliseconds()),
        metric.WithAttributes(
            attribute.Bool("success", err == nil),
            attribute.String("peer_type", classifyPeer(addr)),
        ))
    
    return err
}

// Privacy-safe peer classification
func classifyPeer(addr peer.AddrInfo) string {
    // Return general categories, never individual identifiers
    if isBootstrapPeer(addr) {
        return "bootstrap"
    }
    if isLocalPeer(addr) {
        return "local"
    }
    return "network"
}
```

---

## Analytics Service Design

### Service Architecture

```go
// internal/services/analytics/service.go
package analytics

type AnalyticsService struct {
    // Core components
    collector       *DataCollector
    processor       *StreamProcessor
    privacyEngine   *PrivacyEngine
    storage         *AnalyticsStorage
    queryEngine     *QueryEngine
    
    // Configuration
    config          *Config
    logger          *zap.Logger
    
    // Lifecycle
    ctx             context.Context
    cancel          context.CancelFunc
    wg              sync.WaitGroup
}

type Config struct {
    // Collection settings
    CollectionInterval time.Duration `yaml:"collection_interval"`
    BatchSize          int           `yaml:"batch_size"`
    
    // Privacy settings
    DifferentialPrivacy bool          `yaml:"differential_privacy"`
    PrivacyBudget      float64       `yaml:"privacy_budget"`
    RetentionPeriod    time.Duration `yaml:"retention_period"`
    
    // Storage settings
    DatabasePath       string        `yaml:"database_path"`
    MaxDatabaseSize    int64         `yaml:"max_database_size"`
    
    // Network settings
    NetworkSharing     bool          `yaml:"network_sharing"`
    FederationEndpoint string        `yaml:"federation_endpoint"`
}

func NewAnalyticsService(config *Config) (*AnalyticsService, error) {
    ctx, cancel := context.WithCancel(context.Background())
    
    // Initialize storage
    storage, err := NewAnalyticsStorage(config.DatabasePath)
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to initialize storage: %w", err)
    }
    
    // Initialize privacy engine
    privacyEngine := NewPrivacyEngine(&PrivacyConfig{
        DifferentialPrivacy: config.DifferentialPrivacy,
        PrivacyBudget:      config.PrivacyBudget,
    })
    
    // Initialize stream processor
    processor := NewStreamProcessor(&StreamConfig{
        BatchSize: config.BatchSize,
        FlushInterval: config.CollectionInterval,
    })
    
    // Initialize data collector (OpenTelemetry receiver)
    collector, err := NewDataCollector(&CollectorConfig{
        GRPCEndpoint: ":4317",
        HTTPEndpoint: ":4318",
    })
    if err != nil {
        cancel()
        return nil, fmt.Errorf("failed to initialize collector: %w", err)
    }
    
    return &AnalyticsService{
        collector:     collector,
        processor:     processor,
        privacyEngine: privacyEngine,
        storage:      storage,
        queryEngine:  NewQueryEngine(storage),
        config:       config,
        logger:       zap.L().Named("analytics"),
        ctx:          ctx,
        cancel:       cancel,
    }, nil
}
```

### Data Collection Pipeline

```go
// Data collection and processing pipeline
func (s *AnalyticsService) Start() error {
    // Start OpenTelemetry data collector
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        s.collector.Start(s.ctx)
    }()
    
    // Start stream processing pipeline
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        s.runProcessingPipeline()
    }()
    
    // Start retention cleanup
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        s.runRetentionCleanup()
    }()
    
    return nil
}

func (s *AnalyticsService) runProcessingPipeline() {
    ticker := time.NewTicker(s.config.CollectionInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-s.ctx.Done():
            return
        case <-ticker.C:
            s.processBatch()
        case data := <-s.collector.DataChannel():
            s.processor.Add(data)
        }
    }
}

func (s *AnalyticsService) processBatch() {
    batch := s.processor.GetBatch()
    if len(batch) == 0 {
        return
    }
    
    // Apply privacy transformations
    privacyBatch := s.privacyEngine.ProcessBatch(batch)
    
    // Store in analytics database
    if err := s.storage.StoreBatch(privacyBatch); err != nil {
        s.logger.Error("Failed to store analytics batch", zap.Error(err))
    }
    
    // Generate real-time insights
    insights := s.generateInsights(privacyBatch)
    if len(insights) > 0 {
        s.logger.Info("Generated analytics insights", 
            zap.Int("insight_count", len(insights)))
    }
}
```

---

## Data Collection Strategy

### Metric Categories

#### Infrastructure Metrics (Always Safe)
```go
// Process and system-level metrics
var InfrastructureMetrics = []MetricDefinition{
    {
        Name:        "process.cpu.usage",
        Description: "CPU usage percentage per service",
        Type:        MetricTypeGauge,
        Unit:        "percent",
        Privacy:     PrivacyLevelPublic,
    },
    {
        Name:        "process.memory.usage",
        Description: "Memory usage in bytes per service",
        Type:        MetricTypeGauge,
        Unit:        "bytes",
        Privacy:     PrivacyLevelPublic,
    },
    {
        Name:        "grpc.request.duration",
        Description: "gRPC request processing time",
        Type:        MetricTypeHistogram,
        Unit:        "milliseconds",
        Privacy:     PrivacyLevelPublic,
    },
}
```

#### Network Metrics (Privacy-Sensitive)
```go
// P2P networking metrics with privacy controls
var NetworkMetrics = []MetricDefinition{
    {
        Name:        "p2p.peer.connections",
        Description: "Number of active peer connections by type",
        Type:        MetricTypeCounter,
        Unit:        "connections",
        Privacy:     PrivacyLevelSensitive,
        Aggregation: AggregationByType, // Aggregate by peer type, not individual peers
    },
    {
        Name:        "p2p.bandwidth.usage",
        Description: "Network bandwidth usage",
        Type:        MetricTypeCounter,
        Unit:        "bytes",
        Privacy:     PrivacyLevelSensitive,
        Bucketing:   BandwidthBuckets, // Use buckets to prevent fingerprinting
    },
    {
        Name:        "p2p.discovery.latency",
        Description: "Peer discovery performance",
        Type:        MetricTypeHistogram,
        Unit:        "milliseconds",
        Privacy:     PrivacyLevelInternal,
    },
}
```

#### Business Metrics (Highly Privacy-Sensitive)
```go
// Business logic metrics with strict privacy controls
var BusinessMetrics = []MetricDefinition{
    {
        Name:        "content.operations",
        Description: "Content upload/download operations by type",
        Type:        MetricTypeCounter,
        Unit:        "operations",
        Privacy:     PrivacyLevelHighlySensitive,
        Aggregation: AggregationByContentType, // Type only, no content identification
        NoiseLevel:  DifferentialPrivacyHigh,
    },
    {
        Name:        "identity.operations",
        Description: "DID operations by type",
        Type:        MetricTypeCounter,
        Unit:        "operations",
        Privacy:     PrivacyLevelHighlySensitive,
        Aggregation: AggregationByOperationType, // Operation type only, no DID values
        NoiseLevel:  DifferentialPrivacyHigh,
    },
}
```

### Collection Implementation

```go
// Privacy-aware metric collection
type MetricCollector struct {
    definitions map[string]MetricDefinition
    privacy     *PrivacyEngine
    aggregator  *MetricAggregator
}

func (c *MetricCollector) CollectMetric(name string, value float64, attributes []attribute.KeyValue) {
    def, exists := c.definitions[name]
    if !exists {
        return // Unknown metrics are dropped
    }
    
    // Apply privacy transformations based on sensitivity level
    switch def.Privacy {
    case PrivacyLevelPublic:
        // No transformation needed
        c.aggregator.Add(name, value, attributes)
        
    case PrivacyLevelSensitive:
        // Apply aggregation rules
        aggregatedAttrs := c.privacy.AggregateAttributes(attributes, def.Aggregation)
        c.aggregator.Add(name, value, aggregatedAttrs)
        
    case PrivacyLevelHighlySensitive:
        // Apply differential privacy
        noisyValue := c.privacy.AddNoise(value, def.NoiseLevel)
        aggregatedAttrs := c.privacy.AggregateAttributes(attributes, def.Aggregation)
        c.aggregator.Add(name, noisyValue, aggregatedAttrs)
        
    default:
        // Unknown privacy level - drop metric
        return
    }
}
```

---

## Privacy Architecture

### Privacy Guarantees

Blackhole analytics provides mathematical privacy guarantees through:

1. **No Individual Tracking**: Metrics never identify individual users or content
2. **Differential Privacy**: Formal privacy guarantees for sensitive aggregations
3. **Data Minimization**: Collect only what's needed for specific analytics goals
4. **Automatic Expiration**: All data has configurable retention limits

### Privacy Engine Implementation

```go
// internal/services/analytics/privacy.go
package analytics

import (
    "math"
    "math/rand"
)

type PrivacyEngine struct {
    config       *PrivacyConfig
    noiseSource  *rand.Rand
    budgetTracker *PrivacyBudgetTracker
}

type PrivacyConfig struct {
    DifferentialPrivacy bool    `yaml:"differential_privacy"`
    PrivacyBudget      float64 `yaml:"privacy_budget"`
    SensitivityFactor  float64 `yaml:"sensitivity_factor"`
}

type PrivacyLevel int

const (
    PrivacyLevelPublic PrivacyLevel = iota
    PrivacyLevelInternal
    PrivacyLevelSensitive
    PrivacyLevelHighlySensitive
)

// Add calibrated noise for differential privacy
func (pe *PrivacyEngine) AddNoise(value float64, level PrivacyLevel) float64 {
    if !pe.config.DifferentialPrivacy {
        return value
    }
    
    // Calculate noise scale based on privacy level
    epsilon := pe.budgetTracker.AllocateBudget(level)
    if epsilon <= 0 {
        return 0 // Privacy budget exhausted
    }
    
    // Laplace mechanism for differential privacy
    sensitivity := pe.config.SensitivityFactor
    scale := sensitivity / epsilon
    noise := pe.sampleLaplace(scale)
    
    noisyValue := value + noise
    
    // Ensure non-negative for count metrics
    if noisyValue < 0 {
        noisyValue = 0
    }
    
    return noisyValue
}

// Sample from Laplace distribution
func (pe *PrivacyEngine) sampleLaplace(scale float64) float64 {
    u := pe.noiseSource.Float64() - 0.5
    return -scale * math.Copysign(math.Log(1-2*math.Abs(u)), u)
}

// Aggregate attributes to prevent individual identification
func (pe *PrivacyEngine) AggregateAttributes(attrs []attribute.KeyValue, aggregation AggregationType) []attribute.KeyValue {
    switch aggregation {
    case AggregationByType:
        // Keep only type attributes, remove individual identifiers
        return filterByAllowedKeys(attrs, []string{"type", "category", "region"})
        
    case AggregationByRegion:
        // Geographic aggregation with minimum population
        region := getRegionAttribute(attrs)
        if getRegionPopulation(region) < 1000 {
            region = "other" // Hide small regions
        }
        return []attribute.KeyValue{attribute.String("region", region)}
        
    case AggregationByTimeWindow:
        // Temporal aggregation to prevent timing attacks
        timestamp := getTimestampAttribute(attrs)
        bucket := roundToTimeWindow(timestamp, time.Hour)
        return []attribute.KeyValue{attribute.String("time_bucket", bucket)}
        
    default:
        return []attribute.KeyValue{} // No attributes for unknown aggregation
    }
}

// Privacy budget tracking to prevent information leakage
type PrivacyBudgetTracker struct {
    totalBudget    float64
    usedBudget     float64
    levelBudgets   map[PrivacyLevel]float64
    mutex          sync.RWMutex
}

func (pbt *PrivacyBudgetTracker) AllocateBudget(level PrivacyLevel) float64 {
    pbt.mutex.Lock()
    defer pbt.mutex.Unlock()
    
    requiredBudget := pbt.levelBudgets[level]
    if pbt.usedBudget + requiredBudget > pbt.totalBudget {
        return 0 // Budget exhausted
    }
    
    pbt.usedBudget += requiredBudget
    return requiredBudget
}
```

---

## Storage and Processing

### Analytics Database Design

```sql
-- DuckDB schema for analytics storage
-- Optimized for analytical queries with time-series data

-- Metrics table with efficient time-series storage
CREATE TABLE metrics (
    id BIGINT PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    metric_name VARCHAR NOT NULL,
    metric_type VARCHAR NOT NULL, -- counter, gauge, histogram
    value DOUBLE NOT NULL,
    attributes JSON, -- Store attributes as JSON for flexibility
    node_id VARCHAR, -- For federated analytics (optional)
    privacy_level INTEGER, -- Track privacy level for audit
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient queries
CREATE INDEX idx_metrics_timestamp ON metrics(timestamp);
CREATE INDEX idx_metrics_name_timestamp ON metrics(metric_name, timestamp);
CREATE INDEX idx_metrics_node_timestamp ON metrics(node_id, timestamp) WHERE node_id IS NOT NULL;

-- Aggregated metrics for faster queries
CREATE TABLE metric_aggregates (
    id BIGINT PRIMARY KEY,
    time_bucket TIMESTAMP NOT NULL, -- Hourly/daily buckets
    metric_name VARCHAR NOT NULL,
    aggregation_type VARCHAR NOT NULL, -- sum, avg, max, min
    aggregated_value DOUBLE NOT NULL,
    sample_count INTEGER NOT NULL,
    attributes JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insights table for storing analytical insights
CREATE TABLE insights (
    id BIGINT PRIMARY KEY,
    insight_type VARCHAR NOT NULL,
    title VARCHAR NOT NULL,
    description TEXT,
    severity VARCHAR, -- info, warning, critical
    metrics JSON, -- Supporting metrics
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP -- Automatic cleanup
);

-- Configuration for retention policies
CREATE TABLE retention_policies (
    metric_pattern VARCHAR PRIMARY KEY,
    retention_days INTEGER NOT NULL,
    aggregation_strategy VARCHAR, -- Strategy for old data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default retention policies
INSERT INTO retention_policies VALUES
('infrastructure.*', 30, 'hourly_aggregation'),
('p2p.*', 14, 'daily_aggregation'),
('business.*', 7, 'daily_aggregation_with_noise');
```

### Storage Implementation

```go
// internal/services/analytics/storage.go
package analytics

import (
    "database/sql"
    "encoding/json"
    "time"
    
    _ "github.com/marcboeker/go-duckdb"
)

type AnalyticsStorage struct {
    db     *sql.DB
    config *StorageConfig
    logger *zap.Logger
}

type StorageConfig struct {
    DatabasePath    string        `yaml:"database_path"`
    MaxSize         int64         `yaml:"max_size"`
    RetentionPeriod time.Duration `yaml:"retention_period"`
}

type MetricRecord struct {
    Timestamp   time.Time              `json:"timestamp"`
    MetricName  string                 `json:"metric_name"`
    MetricType  string                 `json:"metric_type"`
    Value       float64                `json:"value"`
    Attributes  map[string]interface{} `json:"attributes"`
    PrivacyLevel int                   `json:"privacy_level"`
}

func NewAnalyticsStorage(path string) (*AnalyticsStorage, error) {
    db, err := sql.Open("duckdb", path)
    if err != nil {
        return nil, fmt.Errorf("failed to open analytics database: %w", err)
    }
    
    storage := &AnalyticsStorage{
        db:     db,
        logger: zap.L().Named("analytics-storage"),
    }
    
    if err := storage.initializeSchema(); err != nil {
        return nil, fmt.Errorf("failed to initialize schema: %w", err)
    }
    
    return storage, nil
}

func (s *AnalyticsStorage) StoreBatch(records []MetricRecord) error {
    tx, err := s.db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    stmt, err := tx.Prepare(`
        INSERT INTO metrics (timestamp, metric_name, metric_type, value, attributes, privacy_level)
        VALUES (?, ?, ?, ?, ?, ?)
    `)
    if err != nil {
        return fmt.Errorf("failed to prepare statement: %w", err)
    }
    defer stmt.Close()
    
    for _, record := range records {
        attributesJSON, err := json.Marshal(record.Attributes)
        if err != nil {
            s.logger.Warn("Failed to marshal attributes", zap.Error(err))
            continue
        }
        
        _, err = stmt.Exec(
            record.Timestamp,
            record.MetricName,
            record.MetricType,
            record.Value,
            string(attributesJSON),
            record.PrivacyLevel,
        )
        if err != nil {
            s.logger.Error("Failed to insert metric record", zap.Error(err))
            continue
        }
    }
    
    return tx.Commit()
}

// Efficient analytical queries using DuckDB's OLAP capabilities
func (s *AnalyticsStorage) QueryTimeSeriesAggregation(metric string, start, end time.Time, interval time.Duration) ([]TimeSeriesPoint, error) {
    query := `
        SELECT 
            time_bucket(INTERVAL ? MINUTE, timestamp) as bucket,
            avg(value) as avg_value,
            max(value) as max_value,
            min(value) as min_value,
            count(*) as sample_count
        FROM metrics 
        WHERE metric_name = ? 
          AND timestamp BETWEEN ? AND ?
        GROUP BY bucket
        ORDER BY bucket
    `
    
    rows, err := s.db.Query(query, 
        int(interval.Minutes()), 
        metric, 
        start, 
        end)
    if err != nil {
        return nil, fmt.Errorf("failed to execute time series query: %w", err)
    }
    defer rows.Close()
    
    var points []TimeSeriesPoint
    for rows.Next() {
        var point TimeSeriesPoint
        err := rows.Scan(
            &point.Timestamp,
            &point.AvgValue,
            &point.MaxValue,
            &point.MinValue,
            &point.SampleCount,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan time series row: %w", err)
        }
        points = append(points, point)
    }
    
    return points, nil
}
```

---

## Implementation Phases

### Phase 1: Foundation and MVP (Weeks 1-2)

**Goals**: Establish OpenTelemetry foundation and basic analytics collection

**Deliverables**:
```go
// OpenTelemetry foundation setup
- OpenTelemetry SDK integration in all services
- Basic metric instrumentation (process metrics, gRPC metrics)
- Analytics service subprocess with OTLP receiver
- DuckDB storage with basic schema
- Simple query engine for dashboard integration
```

**Metrics Collected**:
- Process CPU, memory usage per service
- gRPC request rates, latencies, error rates
- Basic P2P connection counts
- Service health status

**Success Criteria**:
- Zero-overhead OpenTelemetry instrumentation
- Analytics service receives and stores metrics
- Dashboard displays real-time service health
- No impact on core functionality performance

### Phase 2: P2P Intelligence (Weeks 3-4)

**Goals**: Add comprehensive P2P networking analytics

**Deliverables**:
```go
// P2P-specific analytics
- Peer discovery performance metrics
- Network bandwidth and latency tracking
- Connection stability analysis
- Geographic distribution insights (privacy-safe)
```

**Privacy Features**:
- Differential privacy for sensitive P2P metrics
- Geographic aggregation with minimum thresholds
- Peer type classification without individual identification

**Success Criteria**:
- Identify P2P network bottlenecks and optimization opportunities
- Privacy-compliant network health monitoring
- Actionable insights for network performance improvement

### Phase 3: Business Intelligence (Weeks 5-6)

**Goals**: Add business logic analytics with strict privacy controls

**Deliverables**:
```go
// Business metrics with privacy preservation
- Content operation analytics (type-based, not content-specific)
- Identity operation metrics (operation types, not DIDs)
- User interaction patterns (aggregated, anonymized)
- Network-wide health indicators
```

**Advanced Privacy**:
- Advanced differential privacy implementation
- k-anonymity guarantees for user behavior metrics
- Automatic data retention and cleanup

**Success Criteria**:
- Data-driven insights for feature development
- User behavior understanding without privacy violations
- Network-wide performance optimization recommendations

### Phase 4: Federated Analytics (Weeks 7-8)

**Goals**: Enable opt-in network-wide analytics sharing

**Deliverables**:
```go
// Federated network analytics
- Privacy-preserving metric aggregation across nodes
- Network-wide topology and health insights
- Federated performance optimization
- Global network intelligence
```

**Privacy Features**:
- Secure aggregation protocols
- Zero-knowledge proofs for sensitive metrics
- Opt-in participation with granular controls

**Success Criteria**:
- Network-wide performance insights
- Collaborative optimization across participating nodes
- Strong privacy guarantees for federated data

---

## Integration Patterns

### Service Integration

```go
// Pattern for adding analytics to existing services
package service

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
)

type AnalyticsIntegration struct {
    meter       metric.Meter
    // Standard metrics for all services
    requestCounter   metric.Int64Counter
    errorCounter     metric.Int64Counter
    latencyHistogram metric.Float64Histogram
    // Service-specific metrics
    customMetrics    map[string]metric.Instrument
}

func NewAnalyticsIntegration(serviceName string) *AnalyticsIntegration {
    meter := otel.Meter(fmt.Sprintf("blackhole-%s", serviceName))
    
    // Initialize standard metrics
    requestCounter, _ := meter.Int64Counter(
        fmt.Sprintf("%s.requests.total", serviceName),
        metric.WithDescription("Total requests processed"),
    )
    
    errorCounter, _ := meter.Int64Counter(
        fmt.Sprintf("%s.errors.total", serviceName),
        metric.WithDescription("Total errors encountered"),
    )
    
    latencyHistogram, _ := meter.Float64Histogram(
        fmt.Sprintf("%s.request.duration", serviceName),
        metric.WithDescription("Request processing duration"),
        metric.WithUnit("ms"),
    )
    
    return &AnalyticsIntegration{
        meter:            meter,
        requestCounter:   requestCounter,
        errorCounter:     errorCounter,
        latencyHistogram: latencyHistogram,
        customMetrics:    make(map[string]metric.Instrument),
    }
}

// Middleware for automatic gRPC analytics
func (ai *AnalyticsIntegration) GRPCInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        start := time.Now()
        
        // Process request
        resp, err := handler(ctx, req)
        
        // Record analytics
        ai.requestCounter.Add(ctx, 1,
            metric.WithAttributes(
                attribute.String("method", info.FullMethod),
                attribute.Bool("success", err == nil),
            ))
        
        if err != nil {
            ai.errorCounter.Add(ctx, 1,
                metric.WithAttributes(
                    attribute.String("method", info.FullMethod),
                    attribute.String("error_type", classifyError(err)),
                ))
        }
        
        ai.latencyHistogram.Record(ctx, float64(time.Since(start).Milliseconds()),
            metric.WithAttributes(
                attribute.String("method", info.FullMethod),
                attribute.Bool("success", err == nil),
            ))
        
        return resp, err
    }
}
```

### Dashboard Integration

```javascript
// Real-time analytics dashboard integration
class AnalyticsDashboard {
    constructor() {
        this.ws = new WebSocket('ws://localhost:8080/analytics/stream');
        this.charts = {};
        this.setupEventHandlers();
    }
    
    setupEventHandlers() {
        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.updateCharts(data);
        };
    }
    
    updateCharts(analyticsData) {
        // Update service health metrics
        if (analyticsData.type === 'service_health') {
            this.updateServiceHealthChart(analyticsData);
        }
        
        // Update P2P network metrics
        if (analyticsData.type === 'p2p_metrics') {
            this.updateP2PChart(analyticsData);
        }
        
        // Update performance metrics
        if (analyticsData.type === 'performance') {
            this.updatePerformanceChart(analyticsData);
        }
    }
    
    // Query historical analytics
    async queryMetrics(metric, timeRange) {
        const response = await fetch(`/api/analytics/query`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                metric: metric,
                start_time: timeRange.start,
                end_time: timeRange.end,
                aggregation: 'avg',
                interval: '5m'
            })
        });
        
        return await response.json();
    }
}
```

---

## Performance Considerations

### Overhead Minimization

The analytics system is designed for minimal impact on core functionality:

1. **Asynchronous Collection**: All metric collection is non-blocking
2. **Sampling**: Intelligent sampling reduces data volume for high-frequency events
3. **Batch Processing**: Metrics are processed in batches to reduce overhead
4. **Efficient Storage**: DuckDB provides fast analytical queries with minimal resource usage

### Resource Management

```go
// Resource-aware analytics configuration
type ResourceConfig struct {
    MaxCPUUsage    float64 `yaml:"max_cpu_usage"`    // Max 5% CPU for analytics
    MaxMemoryUsage int64   `yaml:"max_memory_usage"` // Max memory for buffers
    MaxDiskUsage   int64   `yaml:"max_disk_usage"`   // Max storage for analytics
}

// Resource monitor ensures analytics doesn't impact core functionality
type ResourceMonitor struct {
    config        *ResourceConfig
    cpuMonitor    *CPUMonitor
    memoryMonitor *MemoryMonitor
    diskMonitor   *DiskMonitor
}

func (rm *ResourceMonitor) ShouldThrottle() bool {
    return rm.cpuMonitor.CurrentUsage() > rm.config.MaxCPUUsage ||
           rm.memoryMonitor.CurrentUsage() > rm.config.MaxMemoryUsage ||
           rm.diskMonitor.CurrentUsage() > rm.config.MaxDiskUsage
}

// Adaptive sampling based on resource availability
func (ac *AnalyticsCollector) AdaptiveSampling() {
    if ac.resourceMonitor.ShouldThrottle() {
        // Reduce sampling rate to decrease resource usage
        ac.samplingRate *= 0.5
        ac.logger.Info("Reducing analytics sampling due to resource constraints",
            zap.Float64("new_rate", ac.samplingRate))
    } else if ac.samplingRate < ac.maxSamplingRate {
        // Increase sampling rate when resources are available
        ac.samplingRate = math.Min(ac.samplingRate*1.1, ac.maxSamplingRate)
    }
}
```

---

## Future Evolution

### Planned Enhancements

1. **Machine Learning Integration**: Anomaly detection and predictive analytics
2. **Advanced Visualization**: Interactive analytics dashboards with custom queries
3. **Federated Learning**: Privacy-preserving collaborative analytics across the network
4. **Real-time Alerting**: Sophisticated alerting based on analytical insights
5. **Performance Optimization**: Automated performance tuning based on analytics

### Integration with External Systems

```go
// Plugin architecture for external analytics integrations
type AnalyticsPlugin interface {
    Name() string
    Export(ctx context.Context, data []MetricRecord) error
    Configure(config map[string]interface{}) error
}

// Example: Prometheus exporter plugin
type PrometheusExporter struct {
    endpoint string
    client   *prometheus.Client
}

func (pe *PrometheusExporter) Export(ctx context.Context, data []MetricRecord) error {
    // Convert internal metrics to Prometheus format
    for _, record := range data {
        promMetric := convertToPrometheusMetric(record)
        if err := pe.client.Push(promMetric); err != nil {
            return err
        }
    }
    return nil
}
```

---

## Conclusion

Blackhole's analytics architecture provides a foundation for data-driven optimization while maintaining strict privacy guarantees. The OpenTelemetry-based approach ensures flexibility and industry-standard practices, while the privacy-first design protects user data and maintains the decentralized nature of the platform.

The phased implementation approach allows for incremental value delivery, starting with essential operational metrics and evolving toward sophisticated network intelligence. This ensures that analytics serve real operational needs rather than being built speculatively.

---

**Document Status**: DRAFT  
**Last Updated**: May 23, 2025  
**Author**: Blackhole Development Team  
**Review Required**: Architecture Review Board