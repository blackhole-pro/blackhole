# Data Persistence Architecture

## Overview

The Data Persistence architecture provides reliable, performant, and scalable storage solutions for the Blackhole node. It implements multiple storage backends, ensures data durability, manages data lifecycle, and optimizes for both read and write operations while maintaining data integrity and supporting disaster recovery.

## Architecture Overview

### Core Components

#### Persistence Manager
- **Storage Orchestrator**: Coordinates multiple storage backends
- **Transaction Manager**: Handles ACID transactions
- **Cache Layer**: In-memory caching for performance
- **Replication Engine**: Data redundancy management
- **Recovery System**: Disaster recovery and backups

#### Storage Backends
- **Key-Value Store**: Fast lookups and simple data
- **Document Store**: Complex hierarchical data
- **Time-Series DB**: Metrics and temporal data
- **Graph Database**: Relationship-heavy data
- **Object Storage**: Large binary objects

#### Data Models
- **Structured Data**: Schema-enforced data
- **Semi-Structured**: Flexible schemas
- **Unstructured**: Binary blobs
- **Streaming Data**: Continuous data flows
- **Immutable Logs**: Append-only data

## Storage Backends

### Key-Value Store

#### RocksDB Implementation
```typescript
interface KeyValueStore {
  get(key: string): Promise<Buffer | null>;
  put(key: string, value: Buffer): Promise<void>;
  delete(key: string): Promise<void>;
  batch(operations: BatchOperation[]): Promise<void>;
  iterator(options?: IteratorOptions): AsyncIterator<KeyValue>;
}

class RocksDBStore implements KeyValueStore {
  private db: RocksDB;
  private writeOptions: WriteOptions;
  private readOptions: ReadOptions;
  
  constructor(options: RocksDBOptions) {
    this.db = new RocksDB(options.path);
    
    // Configure for performance
    this.writeOptions = {
      sync: false,  // Async writes for performance
      disableWAL: false,  // Keep WAL for durability
    };
    
    this.readOptions = {
      fillCache: true,  // Use block cache
      verify_checksums: true  // Data integrity
    };
    
    this.configureDB(options);
  }
  
  private configureDB(options: RocksDBOptions): void {
    // Column families for logical separation
    this.db.createColumnFamily('metadata');
    this.db.createColumnFamily('data');
    this.db.createColumnFamily('indexes');
    
    // Compression settings
    this.db.setOptions({
      compression: CompressionType.LZ4,
      bottommost_compression: CompressionType.ZSTD,
      compression_opts: {
        level: 3,
        parallel_threads: 4
      }
    });
    
    // Performance tuning
    this.db.setOptions({
      max_background_compactions: 4,
      max_background_flushes: 2,
      bytes_per_sync: 1048576,  // 1MB
      compaction_readahead_size: 2097152  // 2MB
    });
  }
  
  async get(key: string): Promise<Buffer | null> {
    try {
      return await this.db.get(key, this.readOptions);
    } catch (error) {
      if (error.notFound) {
        return null;
      }
      throw error;
    }
  }
  
  async put(key: string, value: Buffer): Promise<void> {
    await this.db.put(key, value, this.writeOptions);
  }
  
  async batch(operations: BatchOperation[]): Promise<void> {
    const batch = this.db.batch();
    
    for (const op of operations) {
      switch (op.type) {
        case 'put':
          batch.put(op.key, op.value);
          break;
        case 'delete':
          batch.delete(op.key);
          break;
      }
    }
    
    await batch.write(this.writeOptions);
  }
  
  async *iterator(options?: IteratorOptions): AsyncIterator<KeyValue> {
    const iter = this.db.iterator({
      gte: options?.gte,
      lte: options?.lte,
      reverse: options?.reverse,
      limit: options?.limit
    });
    
    for await (const [key, value] of iter) {
      yield { key, value };
    }
  }
}
```

### Document Store

#### Document Database
```typescript
interface DocumentStore {
  create(collection: string, document: Document): Promise<string>;
  read(collection: string, id: string): Promise<Document | null>;
  update(collection: string, id: string, update: Partial<Document>): Promise<void>;
  delete(collection: string, id: string): Promise<void>;
  query(collection: string, query: Query): Promise<Document[]>;
  createIndex(collection: string, index: Index): Promise<void>;
}

class EmbeddedDocumentDB implements DocumentStore {
  private collections: Map<string, Collection> = new Map();
  private indexes: Map<string, IndexManager> = new Map();
  
  async create(
    collection: string,
    document: Document
  ): Promise<string> {
    const coll = this.getCollection(collection);
    const id = document._id || this.generateId();
    
    // Add metadata
    const doc = {
      ...document,
      _id: id,
      _created: Date.now(),
      _updated: Date.now(),
      _version: 1
    };
    
    // Store document
    await coll.insert(doc);
    
    // Update indexes
    await this.updateIndexes(collection, doc);
    
    return id;
  }
  
  async query(
    collection: string,
    query: Query
  ): Promise<Document[]> {
    const coll = this.getCollection(collection);
    
    // Check for index usage
    const queryPlan = this.optimizeQuery(collection, query);
    
    if (queryPlan.useIndex) {
      return this.queryWithIndex(collection, queryPlan);
    }
    
    // Full collection scan
    return this.scanCollection(coll, query);
  }
  
  private optimizeQuery(
    collection: string,
    query: Query
  ): QueryPlan {
    const indexes = this.indexes.get(collection);
    if (!indexes) {
      return { useIndex: false };
    }
    
    // Find best index for query
    const queryKeys = Object.keys(query.filter);
    const availableIndexes = indexes.getIndexes();
    
    for (const index of availableIndexes) {
      if (this.indexMatchesQuery(index, queryKeys)) {
        return {
          useIndex: true,
          indexName: index.name,
          indexKeys: index.keys
        };
      }
    }
    
    return { useIndex: false };
  }
  
  async createIndex(
    collection: string,
    index: Index
  ): Promise<void> {
    let indexes = this.indexes.get(collection);
    
    if (!indexes) {
      indexes = new IndexManager();
      this.indexes.set(collection, indexes);
    }
    
    // Create index structure
    const indexImpl = new BTreeIndex(index);
    
    // Build index from existing documents
    const coll = this.getCollection(collection);
    for await (const doc of coll.scan()) {
      await indexImpl.insert(doc);
    }
    
    indexes.addIndex(index.name, indexImpl);
  }
}

// B-Tree index implementation
class BTreeIndex {
  private tree: BTree<string, string[]>;
  private index: Index;
  
  constructor(index: Index) {
    this.index = index;
    this.tree = new BTree({
      order: 100,  // B-tree order
      unique: index.unique || false
    });
  }
  
  async insert(document: Document): Promise<void> {
    const key = this.extractKey(document);
    const docId = document._id;
    
    const existing = this.tree.get(key) || [];
    existing.push(docId);
    
    this.tree.set(key, existing);
  }
  
  async find(value: any): Promise<string[]> {
    const key = this.normalizeKey(value);
    return this.tree.get(key) || [];
  }
  
  async range(min: any, max: any): Promise<string[]> {
    const results: string[] = [];
    
    for (const [key, docIds] of this.tree.entries()) {
      if (key >= min && key <= max) {
        results.push(...docIds);
      }
    }
    
    return results;
  }
  
  private extractKey(document: Document): string {
    const values = this.index.keys.map(key => document[key]);
    return this.normalizeKey(values);
  }
  
  private normalizeKey(value: any): string {
    if (Array.isArray(value)) {
      return value.map(v => String(v)).join(':');
    }
    return String(value);
  }
}
```

### Time-Series Database

#### Time-Series Storage
```typescript
interface TimeSeriesDB {
  write(metric: Metric): Promise<void>;
  query(query: TimeSeriesQuery): Promise<DataPoint[]>;
  aggregate(query: AggregateQuery): Promise<AggregateResult>;
  downsample(policy: DownsamplePolicy): Promise<void>;
}

class EmbeddedTimeSeriesDB implements TimeSeriesDB {
  private shards: Map<string, TimeShard> = new Map();
  private retentionPolicies: Map<string, RetentionPolicy> = new Map();
  
  async write(metric: Metric): Promise<void> {
    const shardKey = this.getShardKey(metric.timestamp);
    let shard = this.shards.get(shardKey);
    
    if (!shard) {
      shard = new TimeShard(shardKey);
      this.shards.set(shardKey, shard);
    }
    
    await shard.write(metric);
    
    // Check retention
    await this.enforceRetention();
  }
  
  async query(query: TimeSeriesQuery): Promise<DataPoint[]> {
    const shardKeys = this.getShardKeys(query.start, query.end);
    const results: DataPoint[] = [];
    
    // Query relevant shards
    for (const shardKey of shardKeys) {
      const shard = this.shards.get(shardKey);
      if (shard) {
        const shardResults = await shard.query(query);
        results.push(...shardResults);
      }
    }
    
    // Sort by timestamp
    results.sort((a, b) => a.timestamp - b.timestamp);
    
    return results;
  }
  
  async aggregate(query: AggregateQuery): Promise<AggregateResult> {
    const points = await this.query({
      metric: query.metric,
      start: query.start,
      end: query.end,
      tags: query.tags
    });
    
    return this.performAggregation(points, query);
  }
  
  private performAggregation(
    points: DataPoint[],
    query: AggregateQuery
  ): AggregateResult {
    const buckets = this.createTimeBuckets(
      query.start,
      query.end,
      query.interval
    );
    
    // Group points into buckets
    const bucketedData = new Map<number, DataPoint[]>();
    
    for (const point of points) {
      const bucketTime = this.getBucketTime(
        point.timestamp,
        query.interval
      );
      
      const bucket = bucketedData.get(bucketTime) || [];
      bucket.push(point);
      bucketedData.set(bucketTime, bucket);
    }
    
    // Aggregate each bucket
    const results: AggregatePoint[] = [];
    
    for (const [bucketTime, bucketPoints] of bucketedData) {
      const aggregated = this.aggregateBucket(
        bucketPoints,
        query.function
      );
      
      results.push({
        timestamp: bucketTime,
        value: aggregated
      });
    }
    
    return { points: results };
  }
  
  private aggregateBucket(
    points: DataPoint[],
    func: AggregateFunction
  ): number {
    const values = points.map(p => p.value);
    
    switch (func) {
      case 'avg':
        return values.reduce((a, b) => a + b, 0) / values.length;
      case 'sum':
        return values.reduce((a, b) => a + b, 0);
      case 'min':
        return Math.min(...values);
      case 'max':
        return Math.max(...values);
      case 'count':
        return values.length;
      case 'p95':
        return this.percentile(values, 95);
      case 'p99':
        return this.percentile(values, 99);
      default:
        throw new Error(`Unknown aggregate function: ${func}`);
    }
  }
}

// Time-based sharding
class TimeShard {
  private data: Map<string, SeriesData> = new Map();
  private index: TimeIndex;
  
  constructor(private shardKey: string) {
    this.index = new TimeIndex();
  }
  
  async write(metric: Metric): Promise<void> {
    const seriesKey = this.getSeriesKey(metric);
    let series = this.data.get(seriesKey);
    
    if (!series) {
      series = new SeriesData();
      this.data.set(seriesKey, series);
    }
    
    // Append data point
    series.append({
      timestamp: metric.timestamp,
      value: metric.value
    });
    
    // Update index
    this.index.index(metric);
  }
  
  async query(query: TimeSeriesQuery): Promise<DataPoint[]> {
    // Use index to find matching series
    const seriesKeys = this.index.findSeries(query);
    const results: DataPoint[] = [];
    
    for (const seriesKey of seriesKeys) {
      const series = this.data.get(seriesKey);
      if (series) {
        const points = series.query(query.start, query.end);
        results.push(...points);
      }
    }
    
    return results;
  }
  
  private getSeriesKey(metric: Metric): string {
    const tags = Object.entries(metric.tags)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([k, v]) => `${k}:${v}`)
      .join(',');
    
    return `${metric.name}{${tags}}`;
  }
}
```

### Graph Database

#### Graph Storage
```typescript
interface GraphDB {
  addNode(node: Node): Promise<string>;
  addEdge(edge: Edge): Promise<string>;
  getNode(id: string): Promise<Node | null>;
  getEdges(nodeId: string, direction?: Direction): Promise<Edge[]>;
  traverse(start: string, query: TraversalQuery): Promise<Path[]>;
  shortestPath(start: string, end: string): Promise<Path | null>;
}

class EmbeddedGraphDB implements GraphDB {
  private nodes: Map<string, Node> = new Map();
  private edges: Map<string, Edge> = new Map();
  private adjacencyList: Map<string, Set<string>> = new Map();
  private reverseAdjacencyList: Map<string, Set<string>> = new Map();
  
  async addNode(node: Node): Promise<string> {
    const id = node.id || this.generateId();
    
    const fullNode = {
      ...node,
      id,
      created: Date.now(),
      updated: Date.now()
    };
    
    this.nodes.set(id, fullNode);
    
    // Initialize adjacency lists
    this.adjacencyList.set(id, new Set());
    this.reverseAdjacencyList.set(id, new Set());
    
    return id;
  }
  
  async addEdge(edge: Edge): Promise<string> {
    const id = edge.id || this.generateId();
    
    // Verify nodes exist
    if (!this.nodes.has(edge.from) || !this.nodes.has(edge.to)) {
      throw new Error('Source or target node does not exist');
    }
    
    const fullEdge = {
      ...edge,
      id,
      created: Date.now()
    };
    
    this.edges.set(id, fullEdge);
    
    // Update adjacency lists
    this.adjacencyList.get(edge.from)!.add(edge.to);
    this.reverseAdjacencyList.get(edge.to)!.add(edge.from);
    
    return id;
  }
  
  async traverse(
    start: string,
    query: TraversalQuery
  ): Promise<Path[]> {
    const paths: Path[] = [];
    const visited = new Set<string>();
    const stack: TraversalState[] = [{
      nodeId: start,
      path: [start],
      depth: 0
    }];
    
    while (stack.length > 0) {
      const state = stack.pop()!;
      
      if (state.depth >= query.maxDepth) {
        continue;
      }
      
      // Get neighbors based on direction
      const neighbors = this.getNeighbors(
        state.nodeId,
        query.direction
      );
      
      for (const neighbor of neighbors) {
        if (visited.has(neighbor)) {
          continue;
        }
        
        const edge = this.findEdge(state.nodeId, neighbor);
        
        // Apply edge filter
        if (query.edgeFilter && !query.edgeFilter(edge)) {
          continue;
        }
        
        const node = this.nodes.get(neighbor)!;
        
        // Apply node filter
        if (query.nodeFilter && !query.nodeFilter(node)) {
          continue;
        }
        
        const newPath = [...state.path, neighbor];
        
        // Check if this is a valid end state
        if (this.isValidEndState(node, query)) {
          paths.push({
            nodes: newPath,
            edges: this.getPathEdges(newPath)
          });
        }
        
        // Continue traversal
        if (state.depth + 1 < query.maxDepth) {
          stack.push({
            nodeId: neighbor,
            path: newPath,
            depth: state.depth + 1
          });
        }
        
        visited.add(neighbor);
      }
    }
    
    return paths;
  }
  
  async shortestPath(
    start: string,
    end: string
  ): Promise<Path | null> {
    // Dijkstra's algorithm
    const distances = new Map<string, number>();
    const previous = new Map<string, string | null>();
    const queue = new PriorityQueue<string>();
    
    // Initialize
    for (const nodeId of this.nodes.keys()) {
      distances.set(nodeId, Infinity);
      previous.set(nodeId, null);
    }
    
    distances.set(start, 0);
    queue.enqueue(start, 0);
    
    while (!queue.isEmpty()) {
      const current = queue.dequeue()!;
      
      if (current === end) {
        // Reconstruct path
        return this.reconstructPath(previous, start, end);
      }
      
      const currentDistance = distances.get(current)!;
      const neighbors = this.adjacencyList.get(current)!;
      
      for (const neighbor of neighbors) {
        const edge = this.findEdge(current, neighbor);
        const weight = edge.weight || 1;
        const distance = currentDistance + weight;
        
        if (distance < distances.get(neighbor)!) {
          distances.set(neighbor, distance);
          previous.set(neighbor, current);
          queue.enqueue(neighbor, distance);
        }
      }
    }
    
    return null; // No path found
  }
  
  private getNeighbors(
    nodeId: string,
    direction?: Direction
  ): Set<string> {
    switch (direction) {
      case 'out':
        return this.adjacencyList.get(nodeId) || new Set();
      case 'in':
        return this.reverseAdjacencyList.get(nodeId) || new Set();
      case 'both':
      default:
        const out = this.adjacencyList.get(nodeId) || new Set();
        const in = this.reverseAdjacencyList.get(nodeId) || new Set();
        return new Set([...out, ...in]);
    }
  }
}
```

### Object Storage

#### Binary Object Store
```typescript
interface ObjectStore {
  put(key: string, data: Buffer, metadata?: Metadata): Promise<void>;
  get(key: string): Promise<ObjectData | null>;
  delete(key: string): Promise<void>;
  list(prefix?: string): Promise<ObjectInfo[]>;
  createMultipartUpload(key: string): Promise<string>;
  uploadPart(uploadId: string, partNumber: number, data: Buffer): Promise<string>;
  completeMultipartUpload(uploadId: string, parts: Part[]): Promise<void>;
}

class FileSystemObjectStore implements ObjectStore {
  private basePath: string;
  private metadataStore: MetadataStore;
  private uploads: Map<string, MultipartUpload> = new Map();
  
  constructor(config: ObjectStoreConfig) {
    this.basePath = config.basePath;
    this.metadataStore = new MetadataStore(config.metadataPath);
  }
  
  async put(
    key: string,
    data: Buffer,
    metadata?: Metadata
  ): Promise<void> {
    const objectPath = this.getObjectPath(key);
    
    // Ensure directory exists
    await fs.mkdir(path.dirname(objectPath), { recursive: true });
    
    // Write data
    await fs.writeFile(objectPath, data);
    
    // Store metadata
    if (metadata) {
      await this.metadataStore.set(key, metadata);
    }
    
    // Update size tracking
    await this.updateSizeTracking(key, data.length);
  }
  
  async get(key: string): Promise<ObjectData | null> {
    const objectPath = this.getObjectPath(key);
    
    try {
      const data = await fs.readFile(objectPath);
      const metadata = await this.metadataStore.get(key);
      
      return {
        data,
        metadata,
        size: data.length,
        lastModified: (await fs.stat(objectPath)).mtime
      };
    } catch (error) {
      if (error.code === 'ENOENT') {
        return null;
      }
      throw error;
    }
  }
  
  async createMultipartUpload(key: string): Promise<string> {
    const uploadId = this.generateUploadId();
    
    const upload: MultipartUpload = {
      id: uploadId,
      key,
      parts: new Map(),
      created: Date.now()
    };
    
    this.uploads.set(uploadId, upload);
    
    // Create temporary directory for parts
    const uploadDir = this.getUploadPath(uploadId);
    await fs.mkdir(uploadDir, { recursive: true });
    
    return uploadId;
  }
  
  async uploadPart(
    uploadId: string,
    partNumber: number,
    data: Buffer
  ): Promise<string> {
    const upload = this.uploads.get(uploadId);
    
    if (!upload) {
      throw new Error('Upload not found');
    }
    
    // Write part to temporary location
    const partPath = this.getPartPath(uploadId, partNumber);
    await fs.writeFile(partPath, data);
    
    // Calculate ETag
    const etag = this.calculateETag(data);
    
    // Record part
    upload.parts.set(partNumber, {
      partNumber,
      etag,
      size: data.length
    });
    
    return etag;
  }
  
  async completeMultipartUpload(
    uploadId: string,
    parts: Part[]
  ): Promise<void> {
    const upload = this.uploads.get(uploadId);
    
    if (!upload) {
      throw new Error('Upload not found');
    }
    
    // Verify all parts
    for (const part of parts) {
      const uploadedPart = upload.parts.get(part.partNumber);
      
      if (!uploadedPart || uploadedPart.etag !== part.etag) {
        throw new Error(`Invalid part ${part.partNumber}`);
      }
    }
    
    // Combine parts
    const objectPath = this.getObjectPath(upload.key);
    await fs.mkdir(path.dirname(objectPath), { recursive: true });
    
    const writeStream = fs.createWriteStream(objectPath);
    
    for (const part of parts.sort((a, b) => a.partNumber - b.partNumber)) {
      const partPath = this.getPartPath(uploadId, part.partNumber);
      const partData = await fs.readFile(partPath);
      writeStream.write(partData);
    }
    
    writeStream.end();
    
    // Cleanup temporary files
    await this.cleanupUpload(uploadId);
    
    // Remove from active uploads
    this.uploads.delete(uploadId);
  }
  
  private calculateETag(data: Buffer): string {
    const hash = crypto.createHash('md5');
    hash.update(data);
    return hash.digest('hex');
  }
}
```

## Transaction Management

### ACID Transactions

#### Transaction Manager
```typescript
interface Transaction {
  id: string;
  begin(): Promise<void>;
  commit(): Promise<void>;
  rollback(): Promise<void>;
  get(key: string): Promise<any>;
  put(key: string, value: any): Promise<void>;
  delete(key: string): Promise<void>;
}

class TransactionManager {
  private activeTransactions: Map<string, TransactionImpl> = new Map();
  private lockManager: LockManager;
  private wal: WriteAheadLog;
  
  async beginTransaction(
    options?: TransactionOptions
  ): Promise<Transaction> {
    const txId = this.generateTransactionId();
    
    const tx = new TransactionImpl(txId, {
      isolationLevel: options?.isolationLevel || IsolationLevel.READ_COMMITTED,
      readOnly: options?.readOnly || false,
      timeout: options?.timeout || 30000
    });
    
    // Initialize transaction
    await tx.initialize(this.lockManager, this.wal);
    
    this.activeTransactions.set(txId, tx);
    
    return tx;
  }
  
  async recover(): Promise<void> {
    // Recover from WAL
    const uncommittedTransactions = await this.wal.getUncommitted();
    
    for (const txData of uncommittedTransactions) {
      if (txData.state === 'prepared') {
        // Transaction was prepared but not committed
        await this.rollbackTransaction(txData);
      }
    }
  }
}

class TransactionImpl implements Transaction {
  private state: TransactionState = 'active';
  private readSet: Map<string, any> = new Map();
  private writeSet: Map<string, any> = new Map();
  private deleteSet: Set<string> = new Set();
  private locks: Set<Lock> = new Set();
  
  constructor(
    public id: string,
    private options: TransactionOptions
  ) {}
  
  async initialize(
    lockManager: LockManager,
    wal: WriteAheadLog
  ): Promise<void> {
    this.lockManager = lockManager;
    this.wal = wal;
    
    // Log transaction begin
    await this.wal.log({
      type: 'begin',
      transactionId: this.id,
      timestamp: Date.now()
    });
  }
  
  async get(key: string): Promise<any> {
    this.checkState();
    
    // Check write set first
    if (this.writeSet.has(key)) {
      return this.writeSet.get(key);
    }
    
    // Check delete set
    if (this.deleteSet.has(key)) {
      return null;
    }
    
    // Check read set
    if (this.readSet.has(key)) {
      return this.readSet.get(key);
    }
    
    // Acquire read lock
    const lock = await this.lockManager.acquireReadLock(key, this.id);
    this.locks.add(lock);
    
    // Read from storage
    const value = await this.storage.get(key);
    
    // Add to read set
    this.readSet.set(key, value);
    
    return value;
  }
  
  async put(key: string, value: any): Promise<void> {
    this.checkState();
    
    if (this.options.readOnly) {
      throw new Error('Cannot write in read-only transaction');
    }
    
    // Acquire write lock
    const lock = await this.lockManager.acquireWriteLock(key, this.id);
    this.locks.add(lock);
    
    // Add to write set
    this.writeSet.set(key, value);
    
    // Remove from delete set if present
    this.deleteSet.delete(key);
  }
  
  async commit(): Promise<void> {
    this.checkState();
    
    try {
      // Change state to committing
      this.state = 'committing';
      
      // Validate transaction
      await this.validate();
      
      // Prepare phase (2PC)
      await this.prepare();
      
      // Commit phase
      await this.doCommit();
      
      // Release locks
      await this.releaseLocks();
      
      this.state = 'committed';
    } catch (error) {
      await this.rollback();
      throw error;
    }
  }
  
  private async validate(): Promise<void> {
    // Serializable isolation check
    if (this.options.isolationLevel === IsolationLevel.SERIALIZABLE) {
      // Check for conflicts with concurrent transactions
      for (const [key, value] of this.readSet) {
        const currentValue = await this.storage.get(key);
        
        if (currentValue !== value) {
          throw new Error('Serialization conflict detected');
        }
      }
    }
  }
  
  private async prepare(): Promise<void> {
    // Log prepare
    await this.wal.log({
      type: 'prepare',
      transactionId: this.id,
      writeSet: Array.from(this.writeSet.entries()),
      deleteSet: Array.from(this.deleteSet),
      timestamp: Date.now()
    });
    
    this.state = 'prepared';
  }
  
  private async doCommit(): Promise<void> {
    // Apply write set
    for (const [key, value] of this.writeSet) {
      await this.storage.put(key, value);
    }
    
    // Apply delete set
    for (const key of this.deleteSet) {
      await this.storage.delete(key);
    }
    
    // Log commit
    await this.wal.log({
      type: 'commit',
      transactionId: this.id,
      timestamp: Date.now()
    });
  }
  
  async rollback(): Promise<void> {
    if (this.state === 'committed') {
      throw new Error('Cannot rollback committed transaction');
    }
    
    this.state = 'aborting';
    
    // Log rollback
    await this.wal.log({
      type: 'rollback',
      transactionId: this.id,
      timestamp: Date.now()
    });
    
    // Release locks
    await this.releaseLocks();
    
    // Clear sets
    this.readSet.clear();
    this.writeSet.clear();
    this.deleteSet.clear();
    
    this.state = 'aborted';
  }
  
  private async releaseLocks(): Promise<void> {
    for (const lock of this.locks) {
      await this.lockManager.releaseLock(lock);
    }
    this.locks.clear();
  }
}
```

### Optimistic Concurrency Control

#### MVCC Implementation
```typescript
class MVCCStore {
  private versions: Map<string, VersionedValue[]> = new Map();
  private transactionTimestamps: Map<string, number> = new Map();
  private currentTimestamp = 0;
  
  beginTransaction(txId: string): number {
    const timestamp = this.currentTimestamp++;
    this.transactionTimestamps.set(txId, timestamp);
    return timestamp;
  }
  
  async get(
    key: string,
    txId: string
  ): Promise<any> {
    const timestamp = this.transactionTimestamps.get(txId)!;
    const versions = this.versions.get(key) || [];
    
    // Find the latest version visible to this transaction
    for (let i = versions.length - 1; i >= 0; i--) {
      const version = versions[i];
      
      if (version.timestamp <= timestamp && !version.deleted) {
        return version.value;
      }
    }
    
    return null;
  }
  
  async put(
    key: string,
    value: any,
    txId: string
  ): Promise<void> {
    const timestamp = this.transactionTimestamps.get(txId)!;
    
    let versions = this.versions.get(key);
    if (!versions) {
      versions = [];
      this.versions.set(key, versions);
    }
    
    // Check for write conflicts
    const latestVersion = versions[versions.length - 1];
    if (latestVersion && latestVersion.timestamp > timestamp) {
      throw new Error('Write conflict detected');
    }
    
    // Add new version
    versions.push({
      value,
      timestamp,
      deleted: false
    });
  }
  
  async delete(
    key: string,
    txId: string
  ): Promise<void> {
    const timestamp = this.transactionTimestamps.get(txId)!;
    
    let versions = this.versions.get(key);
    if (!versions) {
      return; // Already deleted
    }
    
    // Add deletion marker
    versions.push({
      value: null,
      timestamp,
      deleted: true
    });
  }
  
  async vacuum(): Promise<void> {
    // Clean up old versions
    const minTimestamp = Math.min(
      ...Array.from(this.transactionTimestamps.values())
    );
    
    for (const [key, versions] of this.versions) {
      // Keep only versions visible to active transactions
      const filteredVersions = versions.filter(
        v => v.timestamp >= minTimestamp
      );
      
      if (filteredVersions.length === 0) {
        this.versions.delete(key);
      } else {
        this.versions.set(key, filteredVersions);
      }
    }
  }
}
```

## Replication

### Primary-Secondary Replication

#### Replication Manager
```typescript
class ReplicationManager {
  private role: ReplicationRole;
  private primary: PrimaryNode | null = null;
  private secondaries: Map<string, SecondaryNode> = new Map();
  private replicationLog: ReplicationLog;
  
  async configurePrimary(): Promise<void> {
    this.role = ReplicationRole.PRIMARY;
    this.primary = new PrimaryNode();
    
    // Start accepting writes
    await this.primary.start();
    
    // Start replication log
    this.replicationLog = new ReplicationLog();
    await this.replicationLog.initialize();
  }
  
  async configureSecondary(primaryUrl: string): Promise<void> {
    this.role = ReplicationRole.SECONDARY;
    
    // Connect to primary
    const connection = await this.connectToPrimary(primaryUrl);
    
    // Start replication
    await this.startReplication(connection);
  }
  
  async write(operation: WriteOperation): Promise<void> {
    if (this.role !== ReplicationRole.PRIMARY) {
      throw new Error('Cannot write to secondary');
    }
    
    // Log operation
    const logEntry = await this.replicationLog.append(operation);
    
    // Apply locally
    await this.applyOperation(operation);
    
    // Replicate to secondaries
    await this.replicateToSecondaries(logEntry);
  }
  
  private async replicateToSecondaries(
    entry: LogEntry
  ): Promise<void> {
    const promises = Array.from(this.secondaries.values()).map(
      secondary => secondary.replicate(entry)
    );
    
    // Wait for majority
    const results = await Promise.allSettled(promises);
    const successful = results.filter(r => r.status === 'fulfilled').length;
    
    if (successful < Math.floor(this.secondaries.size / 2) + 1) {
      throw new Error('Failed to replicate to majority');
    }
  }
  
  async handleReplicationStream(
    stream: ReplicationStream
  ): Promise<void> {
    for await (const entry of stream) {
      try {
        // Validate entry
        if (!this.validateLogEntry(entry)) {
          throw new Error('Invalid log entry');
        }
        
        // Apply operation
        await this.applyOperation(entry.operation);
        
        // Acknowledge
        await stream.acknowledge(entry.id);
      } catch (error) {
        this.logger.error('Replication error:', error);
        
        // Request resync
        await this.requestResync(entry.id);
      }
    }
  }
  
  async failover(): Promise<void> {
    if (this.role !== ReplicationRole.SECONDARY) {
      throw new Error('Only secondary can failover');
    }
    
    // Promote to primary
    this.role = ReplicationRole.PRIMARY;
    this.primary = new PrimaryNode();
    
    // Get latest log position
    const latestPosition = await this.getLatestLogPosition();
    
    // Start accepting writes from this position
    await this.primary.startFromPosition(latestPosition);
    
    // Notify other nodes
    await this.broadcastNewPrimary();
  }
}
```

### Multi-Master Replication

#### Conflict Resolution
```typescript
class MultiMasterReplication {
  private nodeId: string;
  private vectorClock: VectorClock;
  private conflictResolver: ConflictResolver;
  
  async write(
    key: string,
    value: any
  ): Promise<void> {
    // Update vector clock
    this.vectorClock.increment(this.nodeId);
    
    const version = {
      value,
      vectorClock: this.vectorClock.clone(),
      nodeId: this.nodeId,
      timestamp: Date.now()
    };
    
    // Write locally
    await this.storage.put(key, version);
    
    // Replicate to other nodes
    await this.replicate(key, version);
  }
  
  async handleReplication(
    key: string,
    remoteVersion: Version
  ): Promise<void> {
    const localVersion = await this.storage.get(key);
    
    if (!localVersion) {
      // No local version, accept remote
      await this.storage.put(key, remoteVersion);
      return;
    }
    
    // Compare vector clocks
    const comparison = this.vectorClock.compare(
      localVersion.vectorClock,
      remoteVersion.vectorClock
    );
    
    switch (comparison) {
      case ClockComparison.BEFORE:
        // Remote is newer
        await this.storage.put(key, remoteVersion);
        break;
        
      case ClockComparison.AFTER:
        // Local is newer, ignore remote
        break;
        
      case ClockComparison.CONCURRENT:
        // Conflict detected
        const resolved = await this.conflictResolver.resolve(
          localVersion,
          remoteVersion
        );
        await this.storage.put(key, resolved);
        break;
    }
    
    // Update vector clock
    this.vectorClock.merge(remoteVersion.vectorClock);
  }
}

class ConflictResolver {
  async resolve(
    local: Version,
    remote: Version
  ): Promise<Version> {
    // Last-write-wins by default
    if (local.timestamp > remote.timestamp) {
      return local;
    } else {
      return remote;
    }
  }
}

// Application-specific resolver
class CustomConflictResolver extends ConflictResolver {
  async resolve(
    local: Version,
    remote: Version
  ): Promise<Version> {
    // Custom resolution logic
    if (this.isCounter(local.value)) {
      // Merge counters
      return {
        value: local.value + remote.value,
        vectorClock: local.vectorClock.merge(remote.vectorClock),
        nodeId: this.nodeId,
        timestamp: Date.now()
      };
    }
    
    if (this.isSet(local.value)) {
      // Merge sets
      return {
        value: new Set([...local.value, ...remote.value]),
        vectorClock: local.vectorClock.merge(remote.vectorClock),
        nodeId: this.nodeId,
        timestamp: Date.now()
      };
    }
    
    // Fall back to last-write-wins
    return super.resolve(local, remote);
  }
}
```

## Data Recovery

### Backup System

#### Backup Manager
```typescript
class BackupManager {
  private backupSchedule: BackupSchedule;
  private backupStorage: BackupStorage;
  private snapshotter: Snapshotter;
  
  async performBackup(
    type: BackupType = BackupType.INCREMENTAL
  ): Promise<BackupResult> {
    const backupId = this.generateBackupId();
    const startTime = Date.now();
    
    try {
      switch (type) {
        case BackupType.FULL:
          return await this.performFullBackup(backupId);
          
        case BackupType.INCREMENTAL:
          return await this.performIncrementalBackup(backupId);
          
        case BackupType.DIFFERENTIAL:
          return await this.performDifferentialBackup(backupId);
      }
    } catch (error) {
      this.logger.error(`Backup failed: ${error}`);
      throw error;
    }
  }
  
  private async performFullBackup(
    backupId: string
  ): Promise<BackupResult> {
    // Create snapshot
    const snapshot = await this.snapshotter.createSnapshot();
    
    // Compress snapshot
    const compressed = await this.compress(snapshot);
    
    // Encrypt if configured
    const encrypted = this.config.encryption
      ? await this.encrypt(compressed)
      : compressed;
    
    // Upload to backup storage
    await this.backupStorage.upload(backupId, encrypted);
    
    // Create manifest
    const manifest: BackupManifest = {
      id: backupId,
      type: BackupType.FULL,
      timestamp: Date.now(),
      size: encrypted.length,
      checksum: this.calculateChecksum(encrypted),
      encryption: this.config.encryption ? {
        algorithm: this.config.encryption.algorithm,
        keyId: this.config.encryption.keyId
      } : null
    };
    
    await this.backupStorage.saveManifest(manifest);
    
    return {
      backupId,
      type: BackupType.FULL,
      size: encrypted.length,
      duration: Date.now() - startTime
    };
  }
  
  private async performIncrementalBackup(
    backupId: string
  ): Promise<BackupResult> {
    // Get last backup point
    const lastBackup = await this.getLastBackup();
    
    if (!lastBackup) {
      // No previous backup, perform full backup
      return this.performFullBackup(backupId);
    }
    
    // Get changes since last backup
    const changes = await this.getChangesSince(lastBackup.timestamp);
    
    if (changes.length === 0) {
      return {
        backupId,
        type: BackupType.INCREMENTAL,
        size: 0,
        duration: Date.now() - startTime,
        skipped: true
      };
    }
    
    // Create incremental backup
    const incrementalData = await this.createIncrementalData(changes);
    const compressed = await this.compress(incrementalData);
    const encrypted = this.config.encryption
      ? await this.encrypt(compressed)
      : compressed;
    
    // Upload
    await this.backupStorage.upload(backupId, encrypted);
    
    // Create manifest
    const manifest: BackupManifest = {
      id: backupId,
      type: BackupType.INCREMENTAL,
      parentId: lastBackup.id,
      timestamp: Date.now(),
      size: encrypted.length,
      checksum: this.calculateChecksum(encrypted)
    };
    
    await this.backupStorage.saveManifest(manifest);
    
    return {
      backupId,
      type: BackupType.INCREMENTAL,
      size: encrypted.length,
      duration: Date.now() - startTime
    };
  }
}
```

### Recovery Process

#### Recovery Manager
```typescript
class RecoveryManager {
  private backupStorage: BackupStorage;
  private validator: DataValidator;
  
  async recover(
    targetTime?: number
  ): Promise<RecoveryResult> {
    // Find appropriate backup
    const backup = await this.findBackupForTime(targetTime);
    
    if (!backup) {
      throw new Error('No suitable backup found');
    }
    
    // Prepare for recovery
    await this.prepareRecovery();
    
    try {
      // Restore from backup
      await this.restoreFromBackup(backup);
      
      // Apply incremental changes if needed
      if (backup.type === BackupType.INCREMENTAL) {
        await this.applyIncrementalChanges(backup, targetTime);
      }
      
      // Validate restored data
      await this.validateRestoredData();
      
      // Rebuild indexes
      await this.rebuildIndexes();
      
      return {
        success: true,
        backup,
        recoveredTo: targetTime || backup.timestamp,
        duration: Date.now() - startTime
      };
    } catch (error) {
      await this.rollbackRecovery();
      throw error;
    }
  }
  
  private async restoreFromBackup(
    backup: BackupManifest
  ): Promise<void> {
    // Download backup
    const data = await this.backupStorage.download(backup.id);
    
    // Verify checksum
    const checksum = this.calculateChecksum(data);
    if (checksum !== backup.checksum) {
      throw new Error('Backup checksum mismatch');
    }
    
    // Decrypt if needed
    const decrypted = backup.encryption
      ? await this.decrypt(data, backup.encryption)
      : data;
    
    // Decompress
    const decompressed = await this.decompress(decrypted);
    
    // Restore data
    await this.restoreData(decompressed);
  }
  
  private async applyIncrementalChanges(
    backup: BackupManifest,
    targetTime?: number
  ): Promise<void> {
    // Get all incremental backups after this one
    const incrementals = await this.getIncrementalBackupsAfter(backup);
    
    for (const incremental of incrementals) {
      if (targetTime && incremental.timestamp > targetTime) {
        break;
      }
      
      await this.applyIncremental(incremental);
    }
  }
  
  private async validateRestoredData(): Promise<void> {
    // Run data integrity checks
    const validation = await this.validator.validate();
    
    if (!validation.valid) {
      throw new Error(`Data validation failed: ${validation.errors.join(', ')}`);
    }
  }
  
  private async rebuildIndexes(): Promise<void> {
    // Rebuild all indexes
    const indexes = await this.getIndexDefinitions();
    
    for (const index of indexes) {
      await this.rebuildIndex(index);
    }
  }
}
```

### Point-in-Time Recovery

#### PITR Implementation
```typescript
class PointInTimeRecovery {
  private wal: WriteAheadLog;
  private checkpointer: Checkpointer;
  
  async recoverToTime(targetTime: number): Promise<void> {
    // Find latest checkpoint before target time
    const checkpoint = await this.findCheckpointBefore(targetTime);
    
    if (!checkpoint) {
      throw new Error('No checkpoint found before target time');
    }
    
    // Restore from checkpoint
    await this.restoreFromCheckpoint(checkpoint);
    
    // Replay WAL up to target time
    await this.replayWAL(checkpoint.timestamp, targetTime);
    
    // Verify consistency
    await this.verifyConsistency();
  }
  
  private async replayWAL(
    fromTime: number,
    toTime: number
  ): Promise<void> {
    const entries = await this.wal.getEntries(fromTime, toTime);
    
    for (const entry of entries) {
      if (entry.timestamp > toTime) {
        break;
      }
      
      try {
        await this.applyWALEntry(entry);
      } catch (error) {
        this.logger.error(`Failed to apply WAL entry: ${error}`);
        
        if (this.config.strictMode) {
          throw error;
        }
      }
    }
  }
  
  private async applyWALEntry(entry: WALEntry): Promise<void> {
    switch (entry.type) {
      case 'write':
        await this.storage.put(entry.key, entry.value);
        break;
        
      case 'delete':
        await this.storage.delete(entry.key);
        break;
        
      case 'batch':
        await this.storage.batch(entry.operations);
        break;
    }
  }
  
  async createCheckpoint(): Promise<void> {
    const checkpointId = this.generateCheckpointId();
    
    // Pause writes momentarily
    await this.pauseWrites();
    
    try {
      // Create consistent snapshot
      const snapshot = await this.createSnapshot();
      
      // Save checkpoint
      await this.checkpointer.save(checkpointId, snapshot);
      
      // Update checkpoint index
      await this.updateCheckpointIndex(checkpointId);
      
      // Trim old WAL entries
      await this.trimWAL(snapshot.timestamp);
    } finally {
      // Resume writes
      await this.resumeWrites();
    }
  }
}
```

## Performance Optimization

### Caching Layer

#### Multi-Level Cache
```typescript
class CacheManager {
  private l1Cache: LRUCache; // Memory cache
  private l2Cache: RedisCache; // Redis cache
  private l3Cache: DiskCache; // Disk cache
  
  async get(key: string): Promise<any> {
    // Check L1 cache
    let value = this.l1Cache.get(key);
    if (value !== undefined) {
      this.metrics.recordHit('l1');
      return value;
    }
    
    // Check L2 cache
    value = await this.l2Cache.get(key);
    if (value !== undefined) {
      this.metrics.recordHit('l2');
      // Promote to L1
      this.l1Cache.set(key, value);
      return value;
    }
    
    // Check L3 cache
    value = await this.l3Cache.get(key);
    if (value !== undefined) {
      this.metrics.recordHit('l3');
      // Promote to L1 and L2
      await this.l2Cache.set(key, value);
      this.l1Cache.set(key, value);
      return value;
    }
    
    // Cache miss
    this.metrics.recordMiss();
    
    // Fetch from storage
    value = await this.storage.get(key);
    
    if (value !== null) {
      // Populate all cache levels
      await this.populateCaches(key, value);
    }
    
    return value;
  }
  
  private async populateCaches(
    key: string,
    value: any
  ): Promise<void> {
    // Determine cache levels based on access pattern
    const accessPattern = await this.analyzer.getAccessPattern(key);
    
    if (accessPattern.frequency > 0.8) {
      // Hot data - all levels
      this.l1Cache.set(key, value);
      await this.l2Cache.set(key, value);
      await this.l3Cache.set(key, value);
    } else if (accessPattern.frequency > 0.3) {
      // Warm data - L2 and L3
      await this.l2Cache.set(key, value);
      await this.l3Cache.set(key, value);
    } else {
      // Cold data - L3 only
      await this.l3Cache.set(key, value);
    }
  }
  
  async evict(key: string): Promise<void> {
    // Remove from all cache levels
    this.l1Cache.delete(key);
    await this.l2Cache.delete(key);
    await this.l3Cache.delete(key);
  }
}
```

### Write Optimization

#### Write Buffer
```typescript
class WriteBuffer {
  private buffer: Map<string, BufferEntry> = new Map();
  private flushInterval: number = 1000; // 1 second
  private maxBufferSize: number = 10000;
  private flushTimer: NodeJS.Timer;
  
  async write(key: string, value: any): Promise<void> {
    // Add to buffer
    this.buffer.set(key, {
      value,
      timestamp: Date.now(),
      dirty: true
    });
    
    // Check buffer size
    if (this.buffer.size >= this.maxBufferSize) {
      await this.flush();
    }
    
    // Start flush timer if not running
    if (!this.flushTimer) {
      this.startFlushTimer();
    }
  }
  
  private startFlushTimer(): void {
    this.flushTimer = setInterval(async () => {
      await this.flush();
    }, this.flushInterval);
  }
  
  async flush(): Promise<void> {
    if (this.buffer.size === 0) {
      return;
    }
    
    // Group writes by column family
    const batches = this.groupByColumnFamily();
    
    // Execute batches in parallel
    const promises = Array.from(batches.entries()).map(
      ([cf, entries]) => this.flushBatch(cf, entries)
    );
    
    await Promise.all(promises);
    
    // Clear buffer
    this.buffer.clear();
    
    // Stop timer if buffer is empty
    if (this.flushTimer && this.buffer.size === 0) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }
  }
  
  private groupByColumnFamily(): Map<string, BufferEntry[]> {
    const groups = new Map<string, BufferEntry[]>();
    
    for (const [key, entry] of this.buffer) {
      const cf = this.getColumnFamily(key);
      const group = groups.get(cf) || [];
      group.push({ key, ...entry });
      groups.set(cf, group);
    }
    
    return groups;
  }
  
  private async flushBatch(
    columnFamily: string,
    entries: BufferEntry[]
  ): Promise<void> {
    const batch = this.storage.batch();
    
    for (const entry of entries) {
      batch.put(entry.key, entry.value, { cf: columnFamily });
    }
    
    await batch.write();
  }
}
```

### Read Optimization

#### Read-Ahead Cache
```typescript
class ReadAheadCache {
  private predictions: PredictionEngine;
  private prefetchQueue: AsyncQueue<string>;
  private cache: Map<string, any> = new Map();
  
  async get(key: string): Promise<any> {
    // Check cache first
    if (this.cache.has(key)) {
      return this.cache.get(key);
    }
    
    // Fetch from storage
    const value = await this.storage.get(key);
    
    // Predict next reads
    const predictions = await this.predictions.predict(key);
    
    // Queue prefetch
    for (const predictedKey of predictions) {
      this.prefetchQueue.push(predictedKey);
    }
    
    return value;
  }
  
  private async prefetchWorker(): Promise<void> {
    while (true) {
      const key = await this.prefetchQueue.pop();
      
      // Skip if already cached
      if (this.cache.has(key)) {
        continue;
      }
      
      try {
        const value = await this.storage.get(key);
        this.cache.set(key, value);
        
        // Limit cache size
        if (this.cache.size > this.maxCacheSize) {
          this.evictOldest();
        }
      } catch (error) {
        this.logger.warn(`Prefetch failed for ${key}:`, error);
      }
    }
  }
  
  private evictOldest(): void {
    // Find least recently used entry
    let oldestKey: string | null = null;
    let oldestTime = Infinity;
    
    for (const [key, entry] of this.cache) {
      if (entry.lastAccess < oldestTime) {
        oldestTime = entry.lastAccess;
        oldestKey = key;
      }
    }
    
    if (oldestKey) {
      this.cache.delete(oldestKey);
    }
  }
}
```

## Monitoring and Maintenance

### Storage Metrics

#### Metrics Collector
```typescript
class StorageMetrics {
  private counters = {
    reads: new Counter('storage_reads_total'),
    writes: new Counter('storage_writes_total'),
    deletes: new Counter('storage_deletes_total'),
    errors: new Counter('storage_errors_total')
  };
  
  private histograms = {
    readLatency: new Histogram('storage_read_latency_seconds'),
    writeLatency: new Histogram('storage_write_latency_seconds'),
    batchSize: new Histogram('storage_batch_size')
  };
  
  private gauges = {
    diskUsage: new Gauge('storage_disk_usage_bytes'),
    cacheHitRate: new Gauge('storage_cache_hit_ratio'),
    activeTransactions: new Gauge('storage_active_transactions')
  };
  
  recordRead(duration: number, success: boolean): void {
    this.counters.reads.increment({ success });
    this.histograms.readLatency.observe(duration / 1000);
    
    if (!success) {
      this.counters.errors.increment({ operation: 'read' });
    }
  }
  
  recordWrite(duration: number, success: boolean): void {
    this.counters.writes.increment({ success });
    this.histograms.writeLatency.observe(duration / 1000);
    
    if (!success) {
      this.counters.errors.increment({ operation: 'write' });
    }
  }
  
  async collectSystemMetrics(): Promise<void> {
    // Disk usage
    const stats = await this.getStorageStats();
    this.gauges.diskUsage.set(stats.used);
    
    // Cache hit rate
    const cacheStats = await this.getCacheStats();
    const hitRate = cacheStats.hits / (cacheStats.hits + cacheStats.misses);
    this.gauges.cacheHitRate.set(hitRate);
    
    // Active transactions
    const txCount = await this.getActiveTransactionCount();
    this.gauges.activeTransactions.set(txCount);
  }
}
```

### Maintenance Tasks

#### Compaction Manager
```typescript
class CompactionManager {
  private schedule: CompactionSchedule;
  private compactor: Compactor;
  
  async runCompaction(
    level: CompactionLevel = CompactionLevel.MINOR
  ): Promise<CompactionResult> {
    const startTime = Date.now();
    
    try {
      switch (level) {
        case CompactionLevel.MINOR:
          return await this.runMinorCompaction();
          
        case CompactionLevel.MAJOR:
          return await this.runMajorCompaction();
          
        case CompactionLevel.FULL:
          return await this.runFullCompaction();
      }
    } catch (error) {
      this.logger.error('Compaction failed:', error);
      throw error;
    }
  }
  
  private async runMinorCompaction(): Promise<CompactionResult> {
    // Compact recent SSTables
    const tables = await this.getRecentSSTables();
    const compacted = await this.compactor.compact(tables);
    
    return {
      level: CompactionLevel.MINOR,
      inputSize: this.calculateSize(tables),
      outputSize: this.calculateSize([compacted]),
      duration: Date.now() - startTime,
      tablesCompacted: tables.length
    };
  }
  
  private async runMajorCompaction(): Promise<CompactionResult> {
    // Compact overlapping SSTables
    const levels = await this.getOverlappingLevels();
    const results: CompactionResult[] = [];
    
    for (const level of levels) {
      const result = await this.compactLevel(level);
      results.push(result);
    }
    
    return this.mergeResults(results);
  }
  
  async scheduleCompaction(): Promise<void> {
    // Schedule based on write rate
    const writeRate = await this.getWriteRate();
    
    if (writeRate > this.config.highWriteThreshold) {
      // Aggressive compaction
      this.schedule.setInterval(15 * 60 * 1000); // 15 minutes
    } else if (writeRate > this.config.mediumWriteThreshold) {
      // Normal compaction
      this.schedule.setInterval(60 * 60 * 1000); // 1 hour
    } else {
      // Relaxed compaction
      this.schedule.setInterval(4 * 60 * 60 * 1000); // 4 hours
    }
  }
}
```

## Best Practices

### Schema Design
- **Denormalization**: Optimize for reads
- **Composite Keys**: Efficient range queries
- **Time-Based Partitioning**: Archive old data
- **Index Selection**: Balance read/write performance
- **Data Types**: Choose appropriate types

### Performance Tuning
- **Buffer Sizing**: Optimize memory usage
- **Batch Operations**: Reduce round trips
- **Compression**: Save storage space
- **Caching Strategy**: Multi-level caching
- **Query Optimization**: Use indexes effectively

### Reliability
- **Replication**: Multiple copies
- **Backup Strategy**: Regular backups
- **Recovery Testing**: Verify procedures
- **Monitoring**: Track health metrics
- **Alerting**: Proactive notifications

### Maintenance
- **Compaction**: Regular cleanup
- **Vacuum**: Remove dead data
- **Index Rebuilding**: Maintain performance
- **Statistics Update**: Query optimization
- **Log Rotation**: Manage disk space

### Security
- **Encryption at Rest**: Protect data
- **Access Control**: Limit permissions
- **Audit Logging**: Track changes
- **Key Management**: Secure key storage
- **Data Masking**: Hide sensitive data