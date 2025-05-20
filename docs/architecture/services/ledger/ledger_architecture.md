# Blackhole Ledger Architecture

## Introduction

The Blackhole Ledger Service runs as a dedicated subprocess, providing the financial backbone for the Blackhole distributed content sharing platform, leveraging Root Network's Semi-Fungible Token (SFT) capabilities to enable secure content ownership, royalty distribution, and monetization. As a subprocess, it maintains isolation from other services while communicating via gRPC. This document outlines the architecture, components, and integration points of the ledger system.

## System Overview

The Blackhole Ledger system sits at the intersection of content creation, distribution, and monetization, providing a transparent and decentralized way to track content ownership, usage rights, and revenue flows. By utilizing Root Network's SFT capabilities, we can represent content as tokenized assets with built-in programmable logic for royalties, revenue sharing, and licensing terms.

## Subprocess Architecture

The Ledger Service runs as an isolated subprocess handling all blockchain interactions:

```mermaid
graph TD
    subgraph Orchestrator
        Orch[Process Manager]
        SD[Service Discovery]
        Mon[Monitor]
    end
    
    subgraph Ledger Subprocess
        gRPC[gRPC Server :9004]
        TokenSys[Tokenization System]
        RevDist[Revenue Distribution]
        RightsMgmt[Rights Management]
        TxMon[Transaction Monitor]
    end
    
    subgraph Root Network
        BC[Blockchain]
        SC[Smart Contracts]
    end
    
    subgraph Storage Subprocess
        StgRPC[gRPC Server :9003]
        ContentStore[Content Store]
    end
    
    Orch -->|spawn| Ledger Subprocess
    SD -->|register| gRPC
    Mon -->|health check| gRPC
    
    Ledger Subprocess -->|WebSocket/RPC| Root Network
    Ledger Subprocess -->|gRPC :9003| Storage Subprocess
```

### Service Entry Point

```go
// cmd/blackhole/service/ledger/main.go
package main

import (
    "context"
    "flag"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/blackhole/internal/services/ledger"
    "github.com/blackhole/pkg/api/ledger/v1"
    "google.golang.org/grpc"
)

var (
    port       = flag.Int("port", 9004, "gRPC port")
    unixSocket = flag.String("unix-socket", "/tmp/blackhole-ledger.sock", "Unix socket path")
    config     = flag.String("config", "", "Configuration file path")
)

func main() {
    flag.Parse()
    
    // Initialize service
    cfg, err := ledger.LoadConfig(*config)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    service, err := ledger.New(cfg)
    if err != nil {
        log.Fatalf("Failed to create service: %v", err)
    }
    
    // Initialize blockchain connection
    if err := service.ConnectToRootNetwork(context.Background()); err != nil {
        log.Fatalf("Failed to connect to Root Network: %v", err)
    }
    
    // Create gRPC server
    grpcServer := grpc.NewServer(
        grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB
        grpc.MaxSendMsgSize(10 * 1024 * 1024),
    )
    
    // Register service
    ledgerv1.RegisterLedgerServiceServer(grpcServer, service)
    
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
    
    // Handle shutdown gracefully
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    go func() {
        <-sigChan
        log.Println("Shutting down ledger service...")
        service.DisconnectFromRootNetwork()
        grpcServer.GracefulStop()
        cancel()
    }()
    
    // Start serving
    go func() {
        log.Printf("Ledger service listening on Unix socket: %s", *unixSocket)
        if err := grpcServer.Serve(unixListener); err != nil {
            log.Fatalf("Failed to serve Unix socket: %v", err)
        }
    }()
    
    log.Printf("Ledger service listening on TCP port: %d", *port)
    if err := grpcServer.Serve(tcpListener); err != nil {
        log.Fatalf("Failed to serve TCP: %v", err)
    }
}
```

### Key Components

1. **Content Tokenization System**: Converts content into SFTs on the Root Network
2. **Revenue Distribution Engine**: Manages royalty payments and revenue splits
3. **Rights Management System**: Enforces licensing and usage rights 
4. **Token Wallet Integration**: Manages user token balances and transactions
5. **Transaction Monitoring System**: Tracks on-chain activity for analytics
6. **Marketplace Integration**: Enables token trading and content monetization

## Architectural Layers

The ledger system follows our platform's three-layer architecture:

### 1. Infrastructure Layer (Blackhole Nodes)

The P2P node infrastructure contains the core ledger services responsible for blockchain interactions, transaction management, and token operations.

**Responsibilities:**
- Direct interaction with Root Network blockchain
- Transaction building, signing, and submission
- Smart contract deployment and management
- Token metadata storage and synchronization
- Transaction validation and verification
- Block monitoring and event handling

### 2. Service Provider Layer (Client SDK)

The service provider tools provide simplified interfaces for applications to interact with the ledger system without requiring deep blockchain knowledge.

**Responsibilities:**
- Transaction request building and validation
- Wallet management interfaces
- Revenue stream visualization and analytics
- Rights management user interfaces 
- Content monetization options
- Marketplace integration tools

### 3. End User Layer (Applications)

The end-user applications provide intuitive interfaces for interacting with tokenized content, managing rights, and handling payments.

**Responsibilities:**
- User-friendly wallet interfaces
- Content purchasing flows
- Creator royalty dashboards
- Licensing management tools
- Transaction history visualization
- Marketplace browsing and trading

## Root Network SFT Integration

The Blackhole platform leverages Root Network's Semi-Fungible Token (SFT) standard as its foundational tokenization model. SFTs provide a powerful middle ground between fungible tokens (like cryptocurrencies) and non-fungible tokens (NFTs), making them ideal for content representation.

### Why SFTs for Content?

1. **Balanced Uniqueness**: SFTs allow content to be both unique (like the original master version) and partially fungible (like licensed copies)
2. **Efficient Batch Operations**: Multiple content pieces can be managed in batches while maintaining individual properties
3. **Flexible Rights Assignment**: Different rights levels can be represented within the same token class
4. **On-Chain Royalty Enforcement**: Royalty logic is encoded directly in the token smart contract
5. **Scalable Content Management**: More efficient than using pure NFTs for large content libraries

### SFT Implementation on Root Network

Root Network provides native support for SFTs with these key features:

1. **Customizable Token Classes**: Define different token classes for various content types (videos, music, documents)
2. **Token Properties**: Attach content metadata, licensing terms, and royalty parameters directly to tokens
3. **Conditional Transfers**: Enforce rights and licensing conditions during token transfers
4. **Batch Operations**: Efficiently process multiple tokens in single transactions
5. **Royalty Automation**: Automatic fee distribution based on predefined splits

## Content Tokenization Model

The content tokenization process converts digital content into on-chain assets with the following components:

### Token Structure

Each tokenized content item contains these elements:

1. **Content Metadata Hash**: Points to content metadata stored on IPFS
2. **Content CID**: Content Identifier in IPFS/Filecoin
3. **Creator Identity**: DID of the content creator
4. **Token Class**: Defines the content type and default properties
5. **Token Properties**: Custom attributes specific to the content item
6. **Royalty Configuration**: Revenue split parameters and recipient addresses
7. **License Terms**: On-chain encoding of usage rights and restrictions

### Content Classes

The platform defines several token classes to represent different content types:

1. **Master Content**: Original, unmodified content owned by creators
2. **Licensed Content**: Content with specific usage rights
3. **Derivative Content**: Modified content with royalties to original creators
4. **Collection Content**: Bundled content with composite rights
5. **Time-Limited Content**: Content with expiring access rights

## Revenue Sharing and Royalty System

The Blackhole platform includes a sophisticated royalty and revenue sharing system built on Root Network's programmable tokens.

### Royalty Types

1. **Creator Royalties**: Payments to original content creators
2. **Platform Fees**: Optional fees for service providers
3. **Collaboration Splits**: Revenue sharing among multiple contributors
4. **Derivative Work Royalties**: Payments for derivative content
5. **Referral Commissions**: Rewards for content promotion

### Revenue Distribution Mechanisms

1. **On-Chain Splitting**: Automatic division of payments at transaction time
2. **Streaming Payments**: Continuous micro-payments for ongoing content consumption
3. **Batch Settlements**: Periodic settlement of accumulated royalties
4. **Cross-Chain Payouts**: Bridging to external chains for settlement

### Royalty Enforcement

1. **Smart Contract Verification**: Validation of royalty parameters before transactions
2. **On-Chain Disputes**: Resolution mechanism for royalty disagreements
3. **Transparent Tracking**: Public verification of royalty flows
4. **Programmable Thresholds**: Minimum accumulation before settlement

## Rights Management System

The rights management system governs how content can be used, distributed, and monetized, with all terms encoded in SFT properties.

### License Types

1. **View-Only License**: Basic consumption rights
2. **Distribution License**: Rights to share with others
3. **Commercial License**: Rights for business use
4. **Derivative License**: Rights to create modified versions
5. **Time-Limited License**: Temporary access rights
6. **Geographic License**: Region-restricted permissions

### Rights Enforcement

1. **Token-Gated Access**: Content access controlled by token ownership
2. **License Verification**: Validation before content usage
3. **Revocation Mechanism**: Ability to withdraw licenses when terms are violated
4. **License Upgrading**: Pathways to enhance rights through additional payments
5. **License Transfer Restrictions**: Controls on reselling or transferring rights

## Marketplace Integration

The ledger system provides interfaces for content marketplace functionality:

1. **Direct Sales**: Creator-to-consumer content sales
2. **Subscription Models**: Recurring access to content libraries
3. **Auctions**: Competitive bidding for unique content
4. **Secondary Markets**: Trading of previously purchased content
5. **Bundle Sales**: Grouped content offerings with package pricing
6. **Pre-Sales and Crowdfunding**: Advance purchase of upcoming content

## Security Considerations

The ledger system implements several security measures:

1. **Multisignature Operations**: Critical operations require multiple approvals
2. **Transaction Simulation**: Pre-execution validation to prevent errors
3. **Rate Limiting**: Protection against transaction spam
4. **Economic Security**: Fee mechanisms to prevent abuse
5. **Audit Trails**: Complete history of all token operations
6. **Emergency Controls**: Circuit breakers for critical vulnerabilities

## Integration Points

The ledger system integrates with other Blackhole components:

1. **Identity System**: Uses DIDs for transaction authorization and creator verification
2. **Storage System**: Links tokens to content stored in IPFS/Filecoin
3. **Analytics System**: Provides transaction data for content performance metrics
4. **Social System**: Enables token-based actions in social interactions
5. **Indexer**: Supplies token data for content discovery and search

## Implementation Timeline

The ledger system will be implemented in phases:

### Phase 1: Core Infrastructure (6-8 weeks)
- Root Network integration and basic transaction handling
- Core token model implementation
- Wallet integration with identity system
- Basic transaction monitoring

### Phase 2: Tokenization and Rights (4-6 weeks)
- Content tokenization pipeline
- SFT class definitions and properties
- Basic licensing model implementation
- Rights verification system

### Phase 3: Revenue and Royalties (6-8 weeks)
- Royalty configuration and management
- Revenue distribution mechanisms
- Payment splitting implementation
- Settlement processes

### Phase 4: Marketplace Foundation (4-6 weeks)
- Direct sale functionality
- Secondary market support
- Subscription model implementation
- Integration with client applications

## Process Resource Management

The Ledger Service subprocess has dedicated resources tuned for blockchain operations:

### Resource Configuration

```go
// Ledger service resource limits
type LedgerServiceConfig struct {
    ProcessLimits ProcessResourceLimits {
        CPUQuota    "150%"         // 1.5 CPU cores
        MemoryLimit "1GB"          // 1GB memory limit
        IOWeight    100            // Standard IO priority
        Nice        0              // Standard scheduling priority
    }
    
    // Connection pooling for blockchain
    ConnectionPool struct {
        MaxConnections     10
        IdleTimeout        30 * time.Second
        KeepAlive          10 * time.Second
    }
}

// Monitor blockchain connection health
func (l *LedgerService) MonitorConnectionHealth(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            health := l.checkRootNetworkHealth()
            if !health.IsHealthy {
                log.Errorf("Root Network connection unhealthy: %v", health.Error)
                l.reconnect()
            }
            
        case <-ctx.Done():
            return
        }
    }
}
```

### Resource Isolation Benefits

1. **Transaction Processing**: Dedicated CPU for transaction signing and verification
2. **Memory Management**: Isolated memory prevents blockchain operations from affecting other services
3. **Connection Pooling**: Efficient management of blockchain connections
4. **Crash Recovery**: Ledger service failures don't impact content or identity services
5. **Monitoring**: Process-level metrics for blockchain performance analysis

## Configuration

```yaml
ledger_service:
  # Service configuration
  service:
    name: "ledger"
    port: 9004
    unix_socket: "/tmp/blackhole-ledger.sock"
    log_level: "info"
    
  # Process management
  process:
    cpu_limit: "150%"          # 1.5 CPU cores
    memory_limit: "1GB"        # 1GB memory limit
    restart_policy: "always"
    restart_delay: "5s"
    health_check_interval: "30s"
    
  # Root Network configuration
  blockchain:
    network: "root-network"
    endpoint: "wss://root.network"
    chain_id: 1
    block_time: 6s
    
  # Transaction configuration
  transactions:
    max_gas_price: 100
    gas_limit: 1000000
    confirmation_blocks: 6
    retry_count: 3
    
  # Connection management
  connections:
    max_connections: 10
    idle_timeout: 30s
    keep_alive: 10s
```

## Benefits of Subprocess Architecture

1. **Blockchain Isolation**: Blockchain operations run independently from other services
2. **Resource Control**: CPU and memory limits prevent blockchain sync from affecting system
3. **Fault Tolerance**: Ledger crashes don't bring down the entire platform
4. **Security Boundaries**: Process isolation adds an extra security layer for financial operations
5. **Performance**: Dedicated resources ensure consistent transaction processing
6. **Monitoring**: Process-level metrics for detailed blockchain performance analysis
7. **Upgrade Safety**: Can update ledger service without affecting content delivery
8. **Clean APIs**: gRPC interfaces enforce clear boundaries for financial operations