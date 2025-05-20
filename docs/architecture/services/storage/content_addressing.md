# Blackhole Content Pipeline Architecture

This document outlines the internal content pipeline architecture that streamlines data flow within the single binary design, incorporating high-parity Reed-Solomon encoding for efficient persistence and replication.

## Overview

The Content Pipeline is a high-performance, internal processing system that replaces the previous multi-hop content flow with an efficient, single-process pipeline. By leveraging shared memory, goroutines, and Reed-Solomon encoding, it reduces latency from 100ms+ to ~10ms while providing superior data durability.

## Architecture

### Core Pipeline Structure

```go
type ContentPipeline struct {
    stages       []ProcessingStage
    cache        *InternalCache
    workers      *WorkerPool
    rsEncoder    *ReedSolomonEncoder
    metrics      *MetricsCollector
}

type ProcessingStage interface {
    Name() string
    Process(ctx context.Context, data *ContentData) error
    CanRunParallel() bool
}
```

### Pipeline Stages

The content pipeline consists of the following stages:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Content Pipeline                                │
├──────────┬──────────┬──────────┬──────────┬──────────┬──────────┬──────┤
│ Validate │ Encrypt  │ RS       │ Chunk    │ Store    │ Index    │Meta  │
│          │          │ Encode   │ Optimize │          │          │data  │
└──────────┴──────────┴──────────┴──────────┴──────────┴──────────┴──────┘
     ▲          ▲          ▲          ▲          ▲          ▲         ▲
     │          │          │          │          │          │         │
  Serial    Parallel   Parallel   Parallel    Serial    Parallel  Serial
```

1. **Validation Stage**: Content type detection, size limits, security checks
2. **Encryption Stage**: DID-based encryption, key derivation
3. **Reed-Solomon Encoding Stage**: High-parity encoding for durability
4. **Chunking Stage**: Optimal chunk size selection, streaming support
5. **Storage Stage**: IPFS/Filecoin persistence
6. **Indexing Stage**: Search index updates, metadata extraction
7. **Metadata Stage**: Final metadata updates, notifications

## Reed-Solomon Integration

### High-Parity Configuration

```go
type ReedSolomonConfig struct {
    DataShards   int // k parameter (e.g., 10)
    ParityShards int // n-k parameter (e.g., 20)
    TotalShards  int // n parameter (e.g., 30)
    ChunkSize    int // Optimal chunk size for content type
}

// Example configurations by content type
var RSConfigurations = map[string]ReedSolomonConfig{
    "video": {
        DataShards:   10,
        ParityShards: 20,
        TotalShards:  30,
        ChunkSize:    4 * 1024 * 1024, // 4MB chunks
    },
    "document": {
        DataShards:   5,
        ParityShards: 7,
        TotalShards:  12,
        ChunkSize:    256 * 1024, // 256KB chunks
    },
    "default": {
        DataShards:   8,
        ParityShards: 16,
        TotalShards:  24,
        ChunkSize:    1024 * 1024, // 1MB chunks
    },
}
```

### Reed-Solomon Processing Stage

```go
type ReedSolomonStage struct {
    encoder    *reedsolomon.Encoder
    config     ReedSolomonConfig
    chunkPool  *sync.Pool
}

func (rs *ReedSolomonStage) Process(ctx context.Context, data *ContentData) error {
    // 1. Select configuration based on content type
    config := rs.selectConfig(data.ContentType)
    
    // 2. Chunk the content
    chunks := rs.chunkContent(data.Content, config.ChunkSize)
    
    // 3. Encode each chunk with Reed-Solomon
    var encodedChunks []EncodedChunk
    for _, chunk := range chunks {
        encoded, err := rs.encodeChunk(chunk, config)
        if err != nil {
            return err
        }
        encodedChunks = append(encodedChunks, encoded)
    }
    
    // 4. Update content data with encoded fragments
    data.EncodedFragments = encodedChunks
    data.EncodingParams = config
    
    return nil
}

func (rs *ReedSolomonStage) encodeChunk(chunk []byte, config ReedSolomonConfig) (EncodedChunk, error) {
    // Create encoder for this configuration
    encoder, err := reedsolomon.New(config.DataShards, config.ParityShards)
    if err != nil {
        return EncodedChunk{}, err
    }
    
    // Split chunk into data shards
    dataShards, err := encoder.Split(chunk)
    if err != nil {
        return EncodedChunk{}, err
    }
    
    // Generate parity shards
    err = encoder.Encode(dataShards)
    if err != nil {
        return EncodedChunk{}, err
    }
    
    return EncodedChunk{
        DataShards:   dataShards[:config.DataShards],
        ParityShards: dataShards[config.DataShards:],
        ChunkIndex:   chunk.Index,
        OriginalSize: len(chunk),
    }, nil
}
```

## Storage Strategy

### Fragment Distribution

```go
type FragmentDistributor struct {
    ipfsClients []IPFSClient
    nodeSelector *NodeSelector
}

func (fd *FragmentDistributor) DistributeFragments(fragments []Fragment) error {
    // Select nodes for each fragment (no replication needed)
    distribution := fd.nodeSelector.SelectNodes(fragments)
    
    // Store each fragment on a single node
    for i, fragment := range fragments {
        node := distribution[i]
        cid, err := node.Store(fragment)
        if err != nil {
            return err
        }
        fragment.CID = cid
        fragment.NodeID = node.ID
    }
    
    return nil
}
```

### Streaming Support

```go
type StreamingRetriever struct {
    rsDecoder *ReedSolomonDecoder
    cache     *ChunkCache
}

func (sr *StreamingRetriever) Stream(contentID string, writer io.Writer, start, end int64) error {
    metadata := sr.getMetadata(contentID)
    
    // Calculate chunk range for streaming
    startChunk := start / metadata.ChunkSize
    endChunk := end / metadata.ChunkSize
    
    // Stream chunks progressively
    for i := startChunk; i <= endChunk; i++ {
        chunk, err := sr.retrieveChunk(contentID, i)
        if err != nil {
            return err
        }
        
        // Write relevant portion of chunk
        offset := int64(0)
        if i == startChunk {
            offset = start % metadata.ChunkSize
        }
        
        length := metadata.ChunkSize - offset
        if i == endChunk {
            length = (end % metadata.ChunkSize) - offset + 1
        }
        
        _, err = writer.Write(chunk[offset : offset+length])
        if err != nil {
            return err
        }
    }
    
    return nil
}

func (sr *StreamingRetriever) retrieveChunk(contentID string, chunkIndex int64) ([]byte, error) {
    // Check cache first
    if cached := sr.cache.Get(contentID, chunkIndex); cached != nil {
        return cached, nil
    }
    
    // Retrieve minimum required fragments
    metadata := sr.getMetadata(contentID)
    fragments := sr.gatherFragments(contentID, chunkIndex, metadata.RSParams.DataShards)
    
    // Decode chunk
    chunk, err := sr.rsDecoder.DecodeChunk(fragments, metadata.RSParams)
    if err != nil {
        return nil, err
    }
    
    // Cache for future use
    sr.cache.Put(contentID, chunkIndex, chunk)
    
    return chunk, nil
}
```

## Performance Optimizations

### Worker Pool Pattern

```go
type WorkerPool struct {
    workers   int
    taskQueue chan Task
    wg        sync.WaitGroup
}

func (wp *WorkerPool) Execute(stages []ProcessingStage, content Content) error {
    ctx := context.Background()
    
    // Sequential stages run in order
    for _, stage := range stages {
        if !stage.CanRunParallel() {
            if err := stage.Process(ctx, content); err != nil {
                return err
            }
            continue
        }
        
        // Parallel stages use worker pool
        task := Task{
            Stage:   stage,
            Content: content,
            Done:    make(chan error, 1),
        }
        
        wp.taskQueue <- task
        
        // Wait for parallel stage to complete
        if err := <-task.Done; err != nil {
            return err
        }
    }
    
    return nil
}
```

### Internal Cache Design

```go
type InternalCache struct {
    chunks    *lru.Cache
    metadata  *lru.Cache
    fragments *lru.Cache
    mu        sync.RWMutex
}

func (ic *InternalCache) GetChunk(contentID string, chunkIndex int) []byte {
    ic.mu.RLock()
    defer ic.mu.RUnlock()
    
    key := fmt.Sprintf("%s:%d", contentID, chunkIndex)
    if val, ok := ic.chunks.Get(key); ok {
        return val.([]byte)
    }
    return nil
}

func (ic *InternalCache) PutChunk(contentID string, chunkIndex int, data []byte) {
    ic.mu.Lock()
    defer ic.mu.Unlock()
    
    key := fmt.Sprintf("%s:%d", contentID, chunkIndex)
    ic.chunks.Add(key, data)
}
```

## Migration Strategy

### Phase 1: Pipeline Infrastructure

```go
// Implement core pipeline with existing services
func createInitialPipeline() *ContentPipeline {
    return &ContentPipeline{
        stages: []ProcessingStage{
            &ValidationStage{},
            &EncryptionStage{identityService: ids},
            &StorageStage{storageService: storage},
            &MetadataStage{},
        },
        cache:   NewInternalCache(),
        workers: NewWorkerPool(runtime.NumCPU()),
    }
}
```

### Phase 2: Add Reed-Solomon

```go
// Add Reed-Solomon encoding stage
func addReedSolomonStage(pipeline *ContentPipeline) {
    rsStage := &ReedSolomonStage{
        encoder: NewReedSolomonEncoder(),
        config:  RSConfigurations["default"],
    }
    
    // Insert after encryption, before storage
    pipeline.InsertStage(rsStage, 2)
}
```

### Phase 3: Optimize Storage

```go
// Replace replication with high-parity RS
func optimizeStorage(pipeline *ContentPipeline) {
    // Update storage stage to handle fragments
    pipeline.stages[3] = &FragmentStorageStage{
        distributor: NewFragmentDistributor(),
        selector:    NewGeographicNodeSelector(),
    }
}
```

## Monitoring and Metrics

```go
type PipelineMetrics struct {
    StageLatencies   map[string]time.Duration
    ThroughputBytes  int64
    RSEncodingTime   time.Duration
    CacheHitRate     float64
    FragmentHealth   map[string]float64
}

func (p *ContentPipeline) CollectMetrics() PipelineMetrics {
    return PipelineMetrics{
        StageLatencies:  p.getStageLatencies(),
        ThroughputBytes: p.getThroughput(),
        RSEncodingTime:  p.rsEncoder.GetAvgEncodingTime(),
        CacheHitRate:    p.cache.GetHitRate(),
        FragmentHealth:  p.getFragmentHealth(),
    }
}
```

## Configuration

```yaml
content_pipeline:
  stages:
    - validation:
        max_size: 5GB
        allowed_types: ["video/*", "image/*", "text/*", "application/*"]
    - encryption:
        algorithm: "AES-256-GCM"
        key_derivation: "did-based"
    - reed_solomon:
        default:
          data_shards: 10
          parity_shards: 20
        video:
          data_shards: 10
          parity_shards: 20
          chunk_size: 4MB
        document:
          data_shards: 5
          parity_shards: 7
          chunk_size: 256KB
    - storage:
      fragment_distribution:
        strategy: "geographic"
        min_distance_km: 1000
    - indexing:
        parallel_workers: 4
  cache:
    chunk_cache_size: 1GB
    metadata_cache_size: 100MB
    ttl: 3600
  workers:
    pool_size: 8
    queue_size: 1000
```

## Benefits

1. **Performance**: ~10ms latency vs previous 100ms+
2. **Efficiency**: 180% storage overhead vs 300% with replication
3. **Availability**: Survives loss of 66% of nodes (20 of 30)
4. **Scalability**: Better load distribution across nodes
5. **Streaming**: Efficient chunk-based streaming support
6. **Cost**: Reduced storage costs with high durability

## Future Enhancements

1. **Adaptive Encoding**: Adjust RS parameters based on content access patterns
2. **Predictive Caching**: ML-based cache warming for popular content
3. **Cross-Region Optimization**: Intelligent fragment placement for global access
4. **Progressive Encoding**: Start with low redundancy, increase over time for cold storage

---

This content pipeline architecture provides the foundation for efficient, reliable content processing within Blackhole's single binary design, achieving both operational simplicity and architectural sophistication.