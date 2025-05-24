# Monitoring and Diagnostics Architecture

## Overview

The Monitoring and Diagnostics system provides comprehensive visibility into the Blackhole orchestrator and service subprocesses. It enables real-time monitoring of process health, resource usage, RPC performance, and system behavior through metrics collection, distributed tracing, centralized logging, and subprocess health checks.

## Architecture Overview

### Core Components

#### Process Metrics Collector
- **Process Metrics**: CPU, memory, FD usage per subprocess
- **System Metrics**: Host-level resource monitoring
- **RPC Metrics**: Request latency and throughput
- **Aggregation Engine**: Cross-process statistics
- **Alert Manager**: Process health alerts

#### Distributed Trace Collector
- **Cross-Process Tracing**: RPC call tracing
- **Context Propagation**: Trace ID across processes
- **Process Boundaries**: Subprocess span collection
- **Performance Analysis**: Bottleneck identification
- **Query Engine**: Trace exploration

#### Log Aggregator
- **Process Logs**: Per-subprocess log collection
- **Structured Logging**: JSON format with process metadata
- **Central Storage**: Unified log repository
- **Retention Policy**: Time-based cleanup
- **Query Interface**: Cross-process log search

#### Subprocess Health Monitor
- **Process Liveness**: PID tracking and monitoring
- **gRPC Health**: Service health endpoints
- **Resource Limits**: Cgroup/rlimit monitoring
- **Restart Tracking**: Crash detection and recovery
- **Status Dashboard**: Process health visualization

## Metrics System

### Process Metric Types

#### Counter Metrics
```go
type Counter struct {
    Name    string
    Labels  map[string]string
    Value   int64
    mu      sync.Mutex
}

func (c *Counter) Increment(amount int64) {
    c.mu.Lock()
    c.Value += amount
    c.mu.Unlock()
}
  reset(): void;
  get(): number;
}

// Usage example
const requestCounter = new Counter({
  name: 'http_requests_total',
  labels: {
    method: 'GET',
    endpoint: '/api/users',
    status: '200'
  }
});

requestCounter.increment();
```

#### Gauge Metrics
```typescript
interface Gauge {
  name: string;
  labels: Record<string, string>;
  value: number;
  
  set(value: number): void;
  increment(amount?: number): void;
  decrement(amount?: number): void;
  get(): number;
}

// Usage example
const connectionGauge = new Gauge({
  name: 'active_connections',
  labels: {
    protocol: 'websocket',
    server: 'node-1'
  }
});

connectionGauge.set(42);
```

#### Histogram Metrics
```typescript
interface Histogram {
  name: string;
  labels: Record<string, string>;
  buckets: number[];
  
  observe(value: number): void;
  percentile(p: number): number;
  mean(): number;
  sum(): number;
  count(): number;
}

// Usage example
const latencyHistogram = new Histogram({
  name: 'request_duration_seconds',
  labels: {
    service: 'api',
    operation: 'get_user'
  },
  buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5]
});

latencyHistogram.observe(0.023);
```

#### Summary Metrics
```typescript
interface Summary {
  name: string;
  labels: Record<string, string>;
  objectives: Map<number, number>; // quantile -> error
  maxAge: number;
  
  observe(value: number): void;
  quantile(q: number): number;
  count(): number;
  sum(): number;
}

// Usage example
const responseSummary = new Summary({
  name: 'response_size_bytes',
  labels: {
    content_type: 'application/json'
  },
  objectives: new Map([
    [0.5, 0.05],  // 50th percentile with 5% error
    [0.9, 0.01],  // 90th percentile with 1% error
    [0.99, 0.001] // 99th percentile with 0.1% error
  ]),
  maxAge: 600 // 10 minutes
});
```

### Metric Collection

#### Push vs Pull
```typescript
// Pull-based metrics (Prometheus style)
class MetricsEndpoint {
  async handleMetricsRequest(): Promise<string> {
    const metrics = await this.collector.collect();
    return this.formatter.format(metrics);
  }
}

// Push-based metrics (Graphite style)
class MetricsPusher {
  private interval: NodeJS.Timer;
  
  start(intervalMs: number = 10000): void {
    this.interval = setInterval(async () => {
      const metrics = await this.collector.collect();
      await this.backend.push(metrics);
    }, intervalMs);
  }
}
```

#### Metric Aggregation
```typescript
class MetricAggregator {
  aggregate(metrics: Metric[], windowSize: number): AggregatedMetrics {
    return {
      min: this.calculateMin(metrics),
      max: this.calculateMax(metrics),
      mean: this.calculateMean(metrics),
      p50: this.calculatePercentile(metrics, 50),
      p95: this.calculatePercentile(metrics, 95),
      p99: this.calculatePercentile(metrics, 99),
      rate: this.calculateRate(metrics, windowSize)
    };
  }
  
  private calculateRate(metrics: Metric[], windowSize: number): number {
    const firstValue = metrics[0].value;
    const lastValue = metrics[metrics.length - 1].value;
    return (lastValue - firstValue) / windowSize;
  }
}
```

### Metric Storage

#### Time-Series Database
```typescript
interface TimeSeriesDB {
  write(metric: Metric): Promise<void>;
  query(query: MetricQuery): Promise<Metric[]>;
  aggregate(query: AggregateQuery): Promise<AggregatedMetrics>;
  downsample(retention: RetentionPolicy): Promise<void>;
}

class InfluxDBBackend implements TimeSeriesDB {
  async write(metric: Metric): Promise<void> {
    const point = new Point(metric.name)
      .timestamp(metric.timestamp)
      .floatValue(metric.value);
    
    for (const [key, value] of Object.entries(metric.labels)) {
      point.tag(key, value);
    }
    
    await this.client.writePoint(point);
  }
  
  async query(query: MetricQuery): Promise<Metric[]> {
    const flux = `
      from(bucket: "${this.bucket}")
        |> range(start: ${query.start}, stop: ${query.end})
        |> filter(fn: (r) => r._measurement == "${query.metric}")
        ${this.buildLabelFilter(query.labels)}
    `;
    
    return this.client.query(flux);
  }
}
```

## Distributed Tracing

### Trace Model

#### Span Structure
```typescript
interface Span {
  traceId: string;
  spanId: string;
  parentSpanId?: string;
  operationName: string;
  serviceName: string;
  startTime: number;
  endTime: number;
  tags: Record<string, any>;
  logs: LogEntry[];
  status: SpanStatus;
  
  // Relationships
  childSpans: Span[];
  followsFrom?: Span[];
}

interface SpanStatus {
  code: StatusCode;
  message?: string;
}

enum StatusCode {
  OK = 0,
  CANCELLED = 1,
  UNKNOWN = 2,
  INVALID_ARGUMENT = 3,
  DEADLINE_EXCEEDED = 4,
  NOT_FOUND = 5,
  // ... other codes
}
```

#### Trace Context
```typescript
class TraceContext {
  private static HEADER_NAME = 'x-trace-context';
  
  static inject(context: SpanContext, carrier: Headers): void {
    const header = `${context.traceId}-${context.spanId}-${context.flags}`;
    carrier.set(TraceContext.HEADER_NAME, header);
  }
  
  static extract(carrier: Headers): SpanContext | null {
    const header = carrier.get(TraceContext.HEADER_NAME);
    if (!header) return null;
    
    const [traceId, spanId, flags] = header.split('-');
    return {
      traceId,
      spanId,
      flags: parseInt(flags),
      baggage: new Map()
    };
  }
}
```

### Trace Collection

#### Trace Sampling
```typescript
interface Sampler {
  shouldSample(
    traceId: string,
    operation: string,
    parentContext?: SpanContext
  ): SamplingDecision;
}

class AdaptiveSampler implements Sampler {
  private targetRate = 100; // samples per second
  private currentRate = 0;
  
  shouldSample(
    traceId: string,
    operation: string,
    parentContext?: SpanContext
  ): SamplingDecision {
    // Always sample if parent was sampled
    if (parentContext?.sampled) {
      return { sampled: true, probability: 1.0 };
    }
    
    // Priority sampling for errors
    if (operation.includes('error')) {
      return { sampled: true, probability: 1.0 };
    }
    
    // Adaptive sampling based on rate
    const probability = Math.min(1.0, this.targetRate / this.currentRate);
    const sampled = Math.random() < probability;
    
    return { sampled, probability };
  }
}
```

#### Trace Instrumentation
```typescript
// Automatic instrumentation
class Tracer {
  async trace<T>(
    operation: string,
    fn: () => Promise<T>,
    options?: TraceOptions
  ): Promise<T> {
    const span = this.startSpan(operation, options);
    
    try {
      const result = await fn();
      span.setStatus({ code: StatusCode.OK });
      return result;
    } catch (error) {
      span.setStatus({
        code: StatusCode.UNKNOWN,
        message: error.message
      });
      span.log({
        level: 'error',
        message: error.message,
        stack: error.stack
      });
      throw error;
    } finally {
      span.end();
    }
  }
  
  // Manual instrumentation
  startSpan(operation: string, options?: TraceOptions): Span {
    const parent = options?.parent || this.activeSpan();
    const span = new Span({
      traceId: parent?.traceId || this.generateTraceId(),
      spanId: this.generateSpanId(),
      parentSpanId: parent?.spanId,
      operationName: operation,
      serviceName: this.serviceName,
      startTime: Date.now()
    });
    
    this.activeSpans.push(span);
    return span;
  }
}
```

### Trace Analysis

#### Trace Visualization
```typescript
class TraceVisualizer {
  generateWaterfall(trace: Trace): WaterfallData {
    const spans = this.flattenSpans(trace.rootSpan);
    const minTime = Math.min(...spans.map(s => s.startTime));
    const maxTime = Math.max(...spans.map(s => s.endTime));
    
    return {
      duration: maxTime - minTime,
      spans: spans.map(span => ({
        id: span.spanId,
        name: span.operationName,
        service: span.serviceName,
        start: span.startTime - minTime,
        duration: span.endTime - span.startTime,
        depth: this.calculateDepth(span),
        status: span.status.code
      }))
    };
  }
  
  findCriticalPath(trace: Trace): Span[] {
    const path: Span[] = [];
    let current = trace.rootSpan;
    
    while (current.childSpans.length > 0) {
      // Find child with maximum end time
      const criticalChild = current.childSpans.reduce((a, b) => 
        a.endTime > b.endTime ? a : b
      );
      
      path.push(criticalChild);
      current = criticalChild;
    }
    
    return path;
  }
}
```

## Logging System

### Log Structure

#### Structured Logging
```typescript
interface LogEntry {
  timestamp: number;
  level: LogLevel;
  message: string;
  context: LogContext;
  fields: Record<string, any>;
}

interface LogContext {
  service: string;
  hostname: string;
  pid: number;
  traceId?: string;
  spanId?: string;
  userId?: string;
  requestId?: string;
}

enum LogLevel {
  TRACE = 0,
  DEBUG = 1,
  INFO = 2,
  WARN = 3,
  ERROR = 4,
  FATAL = 5
}

class StructuredLogger {
  log(level: LogLevel, message: string, fields?: Record<string, any>): void {
    const entry: LogEntry = {
      timestamp: Date.now(),
      level,
      message,
      context: this.getCurrentContext(),
      fields: fields || {}
    };
    
    this.transport.write(entry);
  }
  
  private getCurrentContext(): LogContext {
    return {
      service: this.serviceName,
      hostname: os.hostname(),
      pid: process.pid,
      traceId: this.tracer.activeSpan()?.traceId,
      spanId: this.tracer.activeSpan()?.spanId,
      userId: this.userContext?.userId,
      requestId: this.requestContext?.requestId
    };
  }
}
```

### Log Aggregation

#### Log Pipeline
```typescript
class LogPipeline {
  private processors: LogProcessor[] = [];
  private transports: LogTransport[] = [];
  
  addProcessor(processor: LogProcessor): void {
    this.processors.push(processor);
  }
  
  addTransport(transport: LogTransport): void {
    this.transports.push(transport);
  }
  
  async process(entry: LogEntry): Promise<void> {
    let processed = entry;
    
    // Apply processors in order
    for (const processor of this.processors) {
      processed = await processor.process(processed);
    }
    
    // Send to all transports
    await Promise.all(
      this.transports.map(t => t.write(processed))
    );
  }
}

// Example processors
class TimestampNormalizer implements LogProcessor {
  async process(entry: LogEntry): Promise<LogEntry> {
    return {
      ...entry,
      timestamp: this.normalizeTimestamp(entry.timestamp)
    };
  }
}

class SensitiveDataRedactor implements LogProcessor {
  private patterns = [
    /password["\s]*[:=]["\s]*["']?[^"',}\s]+/gi,
    /api[_-]key["\s]*[:=]["\s]*["']?[^"',}\s]+/gi,
    /\b\d{16}\b/g // Credit card numbers
  ];
  
  async process(entry: LogEntry): Promise<LogEntry> {
    const message = this.redactSensitiveData(entry.message);
    const fields = this.redactFields(entry.fields);
    
    return { ...entry, message, fields };
  }
}
```

### Log Storage

#### Log Indexing
```typescript
class LogIndexer {
  async index(entry: LogEntry): Promise<void> {
    const document = {
      timestamp: entry.timestamp,
      level: entry.level,
      message: entry.message,
      service: entry.context.service,
      hostname: entry.context.hostname,
      traceId: entry.context.traceId,
      ...entry.fields
    };
    
    await this.elasticsearch.index({
      index: `logs-${this.getDateIndex(entry.timestamp)}`,
      body: document
    });
  }
  
  async search(query: LogQuery): Promise<LogEntry[]> {
    const esQuery = {
      query: {
        bool: {
          must: [
            { range: { timestamp: { gte: query.start, lte: query.end } } },
            { match: { message: query.text } }
          ],
          filter: this.buildFilters(query.filters)
        }
      },
      sort: [{ timestamp: 'desc' }],
      size: query.limit || 100
    };
    
    const result = await this.elasticsearch.search({
      index: 'logs-*',
      body: esQuery
    });
    
    return result.hits.hits.map(hit => hit._source);
  }
}
```

## Health Checks

### Health Check Types

#### Liveness Checks
```typescript
interface LivenessCheck {
  name: string;
  check(): Promise<HealthStatus>;
}

class BasicLivenessCheck implements LivenessCheck {
  name = 'basic_liveness';
  
  async check(): Promise<HealthStatus> {
    // Simple check that the process is running
    return {
      status: 'healthy',
      timestamp: Date.now()
    };
  }
}

class MemoryLivenessCheck implements LivenessCheck {
  name = 'memory_liveness';
  
  async check(): Promise<HealthStatus> {
    const usage = process.memoryUsage();
    const limit = 2 * 1024 * 1024 * 1024; // 2GB
    
    if (usage.rss > limit) {
      return {
        status: 'unhealthy',
        message: 'Memory usage exceeds limit',
        details: { usage, limit }
      };
    }
    
    return { status: 'healthy' };
  }
}
```

#### Readiness Checks
```typescript
interface ReadinessCheck {
  name: string;
  check(): Promise<HealthStatus>;
}

class DatabaseReadinessCheck implements ReadinessCheck {
  name = 'database_readiness';
  
  async check(): Promise<HealthStatus> {
    try {
      await this.db.ping();
      return { status: 'healthy' };
    } catch (error) {
      return {
        status: 'unhealthy',
        message: 'Database connection failed',
        error: error.message
      };
    }
  }
}

class DependencyReadinessCheck implements ReadinessCheck {
  name = 'dependency_readiness';
  
  async check(): Promise<HealthStatus> {
    const results = await Promise.all(
      this.dependencies.map(dep => dep.healthCheck())
    );
    
    const unhealthy = results.filter(r => r.status !== 'healthy');
    
    if (unhealthy.length > 0) {
      return {
        status: 'unhealthy',
        message: 'Dependencies unhealthy',
        dependencies: unhealthy
      };
    }
    
    return { status: 'healthy' };
  }
}
```

### Health Check Aggregation

#### Health Monitor
```typescript
class HealthMonitor {
  private livenessChecks: LivenessCheck[] = [];
  private readinessChecks: ReadinessCheck[] = [];
  private startupChecks: StartupCheck[] = [];
  
  async checkLiveness(): Promise<OverallHealth> {
    const results = await Promise.all(
      this.livenessChecks.map(check => 
        this.executeCheck(check, 'liveness')
      )
    );
    
    return this.aggregateResults(results);
  }
  
  async checkReadiness(): Promise<OverallHealth> {
    const results = await Promise.all(
      this.readinessChecks.map(check => 
        this.executeCheck(check, 'readiness')
      )
    );
    
    return this.aggregateResults(results);
  }
  
  private async executeCheck(
    check: HealthCheck,
    type: string
  ): Promise<CheckResult> {
    const start = Date.now();
    
    try {
      const status = await Promise.race([
        check.check(),
        this.timeout(check.name)
      ]);
      
      return {
        name: check.name,
        type,
        status,
        duration: Date.now() - start
      };
    } catch (error) {
      return {
        name: check.name,
        type,
        status: {
          status: 'unhealthy',
          error: error.message
        },
        duration: Date.now() - start
      };
    }
  }
  
  private aggregateResults(results: CheckResult[]): OverallHealth {
    const unhealthy = results.filter(r => r.status.status !== 'healthy');
    
    return {
      status: unhealthy.length === 0 ? 'healthy' : 'unhealthy',
      checks: results,
      timestamp: Date.now()
    };
  }
}
```

## Alerting System

### Alert Rules

#### Rule Definition
```typescript
interface AlertRule {
  name: string;
  query: MetricQuery;
  condition: AlertCondition;
  duration: number;
  severity: AlertSeverity;
  labels: Record<string, string>;
  annotations: Record<string, string>;
}

interface AlertCondition {
  operator: ComparisonOperator;
  threshold: number;
  aggregation?: AggregationFunction;
}

enum ComparisonOperator {
  GREATER_THAN = '>',
  LESS_THAN = '<',
  EQUAL = '==',
  NOT_EQUAL = '!=',
  GREATER_THAN_OR_EQUAL = '>=',
  LESS_THAN_OR_EQUAL = '<='
}

enum AlertSeverity {
  INFO = 'info',
  WARNING = 'warning',
  ERROR = 'error',
  CRITICAL = 'critical'
}
```

#### Alert Evaluation
```typescript
class AlertEvaluator {
  async evaluate(rule: AlertRule): Promise<AlertResult> {
    // Query metrics
    const metrics = await this.metricsDB.query(rule.query);
    
    // Apply aggregation if specified
    const value = rule.condition.aggregation
      ? this.aggregate(metrics, rule.condition.aggregation)
      : metrics[metrics.length - 1].value;
    
    // Check condition
    const triggered = this.checkCondition(
      value,
      rule.condition.operator,
      rule.condition.threshold
    );
    
    // Check duration requirement
    if (triggered) {
      const durationMet = await this.checkDuration(rule);
      
      if (durationMet) {
        return {
          triggered: true,
          rule,
          value,
          timestamp: Date.now()
        };
      }
    }
    
    return { triggered: false, rule };
  }
  
  private async checkDuration(rule: AlertRule): Promise<boolean> {
    const history = await this.getAlertHistory(rule.name);
    const firstTrigger = history.find(h => h.triggered);
    
    if (!firstTrigger) return false;
    
    return Date.now() - firstTrigger.timestamp >= rule.duration;
  }
}
```

### Alert Notification

#### Notification Channels
```typescript
interface NotificationChannel {
  send(alert: Alert): Promise<void>;
}

class EmailChannel implements NotificationChannel {
  async send(alert: Alert): Promise<void> {
    const email = {
      to: this.config.recipients,
      subject: `[${alert.severity}] ${alert.name}`,
      body: this.formatAlert(alert),
      priority: this.mapSeverityToPriority(alert.severity)
    };
    
    await this.emailService.send(email);
  }
}

class SlackChannel implements NotificationChannel {
  async send(alert: Alert): Promise<void> {
    const message = {
      channel: this.config.channel,
      text: alert.summary,
      attachments: [{
        color: this.getSeverityColor(alert.severity),
        fields: this.formatFields(alert),
        ts: alert.timestamp
      }]
    };
    
    await this.slackClient.postMessage(message);
  }
}

class PagerDutyChannel implements NotificationChannel {
  async send(alert: Alert): Promise<void> {
    if (alert.severity < AlertSeverity.ERROR) {
      return; // Only page for high severity
    }
    
    const incident = {
      routing_key: this.config.routingKey,
      event_action: 'trigger',
      dedup_key: alert.fingerprint,
      payload: {
        summary: alert.summary,
        severity: alert.severity,
        source: alert.source,
        custom_details: alert.details
      }
    };
    
    await this.pagerduty.sendEvent(incident);
  }
}
```

## Dashboards

### Dashboard Components

#### Metric Visualization
```typescript
interface DashboardPanel {
  id: string;
  title: string;
  type: PanelType;
  query: MetricQuery;
  visualization: VisualizationConfig;
  position: GridPosition;
}

enum PanelType {
  GRAPH = 'graph',
  STAT = 'stat',
  TABLE = 'table',
  HEATMAP = 'heatmap',
  GAUGE = 'gauge'
}

interface VisualizationConfig {
  type: PanelType;
  options: Record<string, any>;
  thresholds?: Threshold[];
  unit?: string;
  decimals?: number;
}

class DashboardRenderer {
  async renderPanel(panel: DashboardPanel): Promise<RenderedPanel> {
    // Query data
    const data = await this.queryData(panel.query);
    
    // Apply visualization
    const visualized = this.visualizers[panel.type].render(
      data,
      panel.visualization
    );
    
    return {
      id: panel.id,
      title: panel.title,
      content: visualized,
      lastUpdate: Date.now()
    };
  }
}
```

### Real-Time Updates

#### WebSocket Streaming
```typescript
class DashboardWebSocket {
  private connections = new Map<string, WebSocket>();
  private subscriptions = new Map<string, Set<string>>();
  
  handleConnection(ws: WebSocket, dashboardId: string): void {
    this.connections.set(dashboardId, ws);
    
    ws.on('message', (data) => {
      const message = JSON.parse(data);
      
      switch (message.type) {
        case 'subscribe':
          this.subscribe(dashboardId, message.panels);
          break;
        case 'unsubscribe':
          this.unsubscribe(dashboardId, message.panels);
          break;
      }
    });
    
    ws.on('close', () => {
      this.connections.delete(dashboardId);
      this.subscriptions.delete(dashboardId);
    });
  }
  
  private subscribe(dashboardId: string, panelIds: string[]): void {
    const subs = this.subscriptions.get(dashboardId) || new Set();
    panelIds.forEach(id => subs.add(id));
    this.subscriptions.set(dashboardId, subs);
    
    // Start streaming updates
    this.startStreaming(dashboardId, panelIds);
  }
  
  private async startStreaming(
    dashboardId: string,
    panelIds: string[]
  ): Promise<void> {
    const ws = this.connections.get(dashboardId);
    if (!ws) return;
    
    // Create metric stream
    const stream = this.metricsDB.stream(
      panelIds.map(id => this.getPanelQuery(id))
    );
    
    stream.on('data', (update) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          type: 'update',
          panelId: update.panelId,
          data: update.data
        }));
      }
    });
  }
}
```

## Performance Profiling

### CPU Profiling

#### Profile Collection
```typescript
class CPUProfiler {
  private v8Profiler = require('v8-profiler-next');
  private profiles = new Map<string, CPUProfile>();
  
  startProfiling(name: string, options?: ProfileOptions): void {
    this.v8Profiler.setSamplingInterval(options?.samplingInterval || 10);
    this.v8Profiler.startProfiling(name, options?.recSamples);
  }
  
  stopProfiling(name: string): CPUProfile {
    const profile = this.v8Profiler.stopProfiling(name);
    const result = this.processCPUProfile(profile);
    profile.delete();
    
    this.profiles.set(name, result);
    return result;
  }
  
  private processCPUProfile(profile: any): CPUProfile {
    const processed = {
      title: profile.title,
      duration: profile.endTime - profile.startTime,
      samples: profile.samples,
      timestamps: profile.timestamps,
      hotFunctions: this.findHotFunctions(profile),
      flameGraph: this.generateFlameGraph(profile)
    };
    
    return processed;
  }
  
  private findHotFunctions(profile: any): HotFunction[] {
    const functionMap = new Map<string, number>();
    
    // Count samples per function
    profile.samples.forEach((sample: any) => {
      const stack = this.getStackForSample(profile, sample);
      stack.forEach((frame: any) => {
        const key = `${frame.functionName}:${frame.url}:${frame.lineNumber}`;
        functionMap.set(key, (functionMap.get(key) || 0) + 1);
      });
    });
    
    // Sort by sample count
    return Array.from(functionMap.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 20)
      .map(([key, count]) => {
        const [functionName, url, lineNumber] = key.split(':');
        return {
          functionName,
          url,
          lineNumber: parseInt(lineNumber),
          sampleCount: count,
          percentage: (count / profile.samples.length) * 100
        };
      });
  }
}
```

### Memory Profiling

#### Heap Snapshot
```typescript
class MemoryProfiler {
  private v8Profiler = require('v8-profiler-next');
  
  async takeHeapSnapshot(name: string): Promise<HeapSnapshot> {
    const snapshot = this.v8Profiler.takeSnapshot(name);
    const processed = await this.processHeapSnapshot(snapshot);
    snapshot.delete();
    
    return processed;
  }
  
  private async processHeapSnapshot(snapshot: any): Promise<HeapSnapshot> {
    return new Promise((resolve) => {
      const chunks: string[] = [];
      
      snapshot.serialize((chunk: string, done: boolean) => {
        chunks.push(chunk);
        
        if (done) {
          const data = JSON.parse(chunks.join(''));
          resolve(this.analyzeHeapSnapshot(data));
        }
      });
    });
  }
  
  private analyzeHeapSnapshot(data: any): HeapSnapshot {
    return {
      totalSize: data.snapshot.meta.node_fields
        .reduce((sum: number, node: any) => sum + node.size, 0),
      nodeCount: data.nodes.length,
      statistics: this.calculateStatistics(data),
      dominators: this.calculateDominators(data),
      retainers: this.findRetainers(data),
      leaks: this.detectLeaks(data)
    };
  }
  
  compareSnapshots(
    before: HeapSnapshot,
    after: HeapSnapshot
  ): HeapComparison {
    return {
      sizeDiff: after.totalSize - before.totalSize,
      nodeCountDiff: after.nodeCount - before.nodeCount,
      newObjects: this.findNewObjects(before, after),
      deletedObjects: this.findDeletedObjects(before, after),
      grownObjects: this.findGrownObjects(before, after)
    };
  }
}
```

## Debug Tools

### Debug Endpoints

#### Runtime Inspection
```typescript
class DebugEndpoints {
  registerEndpoints(app: Express): void {
    // Thread dump
    app.get('/debug/threads', async (req, res) => {
      const threads = await this.getThreadDump();
      res.json(threads);
    });
    
    // Memory dump
    app.get('/debug/memory', async (req, res) => {
      const memory = process.memoryUsage();
      const detailed = await this.getDetailedMemoryInfo();
      res.json({ ...memory, detailed });
    });
    
    // Config dump
    app.get('/debug/config', (req, res) => {
      const config = this.configManager.getAll();
      const sanitized = this.sanitizeConfig(config);
      res.json(sanitized);
    });
    
    // Goroutine dump (for Go services)
    app.get('/debug/goroutines', async (req, res) => {
      const goroutines = await this.getGoroutineDump();
      res.json(goroutines);
    });
    
    // CPU profile
    app.post('/debug/profile/cpu', async (req, res) => {
      const duration = req.body.duration || 30000;
      const profile = await this.collectCPUProfile(duration);
      res.json(profile);
    });
    
    // Heap profile
    app.get('/debug/profile/heap', async (req, res) => {
      const profile = await this.collectHeapProfile();
      res.json(profile);
    });
  }
  
  private sanitizeConfig(config: any): any {
    const sensitive = ['password', 'secret', 'key', 'token'];
    
    return Object.entries(config).reduce((acc, [key, value]) => {
      if (sensitive.some(s => key.toLowerCase().includes(s))) {
        acc[key] = '***REDACTED***';
      } else if (typeof value === 'object' && value !== null) {
        acc[key] = this.sanitizeConfig(value);
      } else {
        acc[key] = value;
      }
      return acc;
    }, {} as any);
  }
}
```

### Debug Logging

#### Conditional Logging
```typescript
class DebugLogger {
  private debugFlags = new Map<string, boolean>();
  
  setDebugFlag(flag: string, enabled: boolean): void {
    this.debugFlags.set(flag, enabled);
  }
  
  debug(flag: string, message: string, data?: any): void {
    if (!this.debugFlags.get(flag)) return;
    
    this.logger.debug({
      flag,
      message,
      data,
      timestamp: Date.now(),
      stack: this.captureStack()
    });
  }
  
  trace(operation: string, fn: Function): Function {
    return (...args: any[]) => {
      const start = Date.now();
      const id = this.generateId();
      
      this.debug('trace', `Start ${operation}`, { id, args });
      
      try {
        const result = fn(...args);
        
        if (result instanceof Promise) {
          return result
            .then(value => {
              this.debug('trace', `Complete ${operation}`, {
                id,
                duration: Date.now() - start,
                result: value
              });
              return value;
            })
            .catch(error => {
              this.debug('trace', `Error ${operation}`, {
                id,
                duration: Date.now() - start,
                error: error.message
              });
              throw error;
            });
        }
        
        this.debug('trace', `Complete ${operation}`, {
          id,
          duration: Date.now() - start,
          result
        });
        
        return result;
      } catch (error) {
        this.debug('trace', `Error ${operation}`, {
          id,
          duration: Date.now() - start,
          error: error.message
        });
        throw error;
      }
    };
  }
}
```

## Integration

### Monitoring Stack

#### Prometheus Integration
```typescript
class PrometheusExporter {
  private registry = new Registry();
  private metrics = new Map<string, Metric>();
  
  constructor() {
    this.setupDefaultMetrics();
  }
  
  private setupDefaultMetrics(): void {
    // Process metrics
    collectDefaultMetrics({ register: this.registry });
    
    // Custom metrics
    this.registerMetric('http_request_duration_seconds', 
      new Histogram({
        name: 'http_request_duration_seconds',
        help: 'Duration of HTTP requests in seconds',
        labelNames: ['method', 'route', 'status'],
        buckets: [0.001, 0.005, 0.015, 0.05, 0.1, 0.5, 1, 5]
      })
    );
  }
  
  async exportMetrics(): Promise<string> {
    return this.registry.metrics();
  }
  
  middleware(): RequestHandler {
    return (req, res, next) => {
      const timer = this.metrics
        .get('http_request_duration_seconds')
        .startTimer();
      
      res.on('finish', () => {
        timer({
          method: req.method,
          route: req.route?.path || 'unknown',
          status: res.statusCode
        });
      });
      
      next();
    };
  }
}
```

#### Grafana Integration
```typescript
class GrafanaDashboard {
  async createDashboard(config: DashboardConfig): Promise<Dashboard> {
    const dashboard = {
      uid: config.uid,
      title: config.title,
      tags: config.tags,
      timezone: 'browser',
      panels: config.panels.map(this.createPanel),
      time: {
        from: 'now-6h',
        to: 'now'
      },
      refresh: '10s'
    };
    
    const response = await this.grafanaAPI.post('/dashboards/db', {
      dashboard,
      overwrite: true
    });
    
    return response.data;
  }
  
  private createPanel(config: PanelConfig): Panel {
    return {
      id: config.id,
      gridPos: config.position,
      title: config.title,
      type: config.type,
      datasource: config.datasource,
      targets: config.queries.map(q => ({
        expr: q.expression,
        refId: q.refId,
        interval: q.interval
      })),
      options: config.options,
      fieldConfig: {
        defaults: {
          thresholds: config.thresholds,
          unit: config.unit,
          decimals: config.decimals
        }
      }
    };
  }
}
```

## Best Practices

### Monitoring Strategy
- **Golden Signals**: Latency, traffic, errors, saturation
- **Service Level Objectives**: Define SLOs
- **Alert Fatigue**: Meaningful alerts only
- **Gradual Rollout**: Monitor during deployment
- **Capacity Planning**: Predict growth

### Instrumentation
- **Automatic**: Use auto-instrumentation
- **Manual**: Critical paths only
- **Sampling**: Reduce overhead
- **Context Propagation**: Maintain trace context
- **Standard Labels**: Consistent tagging

### Log Management
- **Structured Logging**: JSON format
- **Log Levels**: Appropriate verbosity
- **Retention Policy**: Storage management
- **Privacy**: Redact sensitive data
- **Correlation**: Link logs to traces

### Performance
- **Low Overhead**: Minimal impact
- **Async Collection**: Non-blocking
- **Batch Export**: Reduce network
- **Local Aggregation**: Edge computing
- **Adaptive Sampling**: Dynamic rates

### Debugging
- **Production Safety**: Safe debug tools
- **Feature Flags**: Control debug features
- **Audit Trail**: Log debug access
- **Time Limits**: Prevent resource exhaustion
- **Cleanup**: Remove debug artifacts