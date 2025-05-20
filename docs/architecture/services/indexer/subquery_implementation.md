# SubQuery Implementation Guide for Blackhole

This document provides detailed implementation guidance for integrating SubQuery as the primary indexing solution for the Blackhole platform.

## Overview

SubQuery is a leading indexing protocol specifically designed for Substrate and EVM-compatible chains. For Blackhole, it provides the ideal balance of performance, flexibility, and decentralization while supporting Root Network's unique architecture.

## Why SubQuery for Blackhole

### Technical Advantages

1. **Substrate Native Support**
   - Built specifically for Substrate chains
   - Optimized for Substrate event processing
   - Native support for custom pallets
   - Efficient state queries

2. **EVM Compatibility**
   - Full Ethereum Virtual Machine support
   - Smart contract event indexing
   - Web3 compatibility
   - Hybrid chain support

3. **Performance Optimization**
   - Parallel processing capabilities
   - Batch indexing support
   - Query result caching
   - Incremental indexing

4. **Developer Experience**
   - TypeScript/JavaScript support
   - Comprehensive CLI tools
   - Code generation utilities
   - Extensive documentation

### Architectural Fit

```
┌─────────────────────────────────────────────────────────────────┐
│                    Blackhole Architecture                       │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │    Root     │  │  SubQuery   │  │  Blackhole  │             │
│  │   Network   │◄─┤   Indexer   ├─►│    Nodes    │             │
│  └─────────────┘  └──────┬──────┘  └─────────────┘             │
│                          │                                      │
│                          ▼                                      │
│                    ┌───────────┐                                │
│                    │  GraphQL  │                                │
│                    │    API    │                                │
│                    └─────┬─────┘                                │
│                          │                                      │
│  ┌─────────────┐  ┌─────▼─────┐  ┌─────────────┐               │
│  │   Service   │  │   Query    │  │     End     │               │
│  │  Providers  │◄─┤  Service   ├─►│    Users    │               │
│  └─────────────┘  └───────────┘  └─────────────┘               │
└─────────────────────────────────────────────────────────────────┘
```

## Implementation Architecture

### Project Structure

```
blackhole-subquery/
├── project.yaml              # Project configuration
├── schema.graphql           # GraphQL schema definitions
├── src/
│   ├── index.ts            # Main entry point
│   ├── mappings/           # Event handlers
│   │   ├── tokenHandlers.ts
│   │   ├── licenseHandlers.ts
│   │   ├── royaltyHandlers.ts
│   │   └── marketHandlers.ts
│   ├── types/              # Generated types
│   └── utils/              # Helper functions
├── package.json
├── tsconfig.json
└── docker-compose.yml      # Local development setup
```

### Configuration Components

#### Project Configuration (project.yaml)

```yaml
specVersion: 1.0.0
name: blackhole-indexer
version: 1.0.0
runner:
  node:
    name: "@subql/node"
    version: ">=3.0.0"
  query:
    name: "@subql/query"
    version: ">=3.0.0"
description: Blackhole platform blockchain indexer
repository: https://github.com/blackhole/subquery

network:
  chainId: "0x..." # Root Network chain ID
  endpoint: 
    - wss://root-network.io/ws
    - https://root-network.io/rpc
  dictionary: https://api.subquery.network/sq/root-network

dataSources:
  - kind: substrate/Runtime
    startBlock: 1
    mapping:
      file: ./dist/index.js
      handlers:
        - handler: handleBlock
          kind: substrate/BlockHandler
        - handler: handleEvent
          kind: substrate/EventHandler
          filter:
            module: tokens
            method: TokenCreated
        - handler: handleCall
          kind: substrate/CallHandler
          filter:
            module: tokens
            method: transfer

  - kind: substrate/Evm
    startBlock: 1
    assets:
      Token:
        file: ./contracts/Token.json
      Marketplace:
        file: ./contracts/Marketplace.json
    mapping:
      file: ./dist/index.js
      handlers:
        - handler: handleTokenTransfer
          kind: substrate/EvmEventHandler
          filter:
            topics:
              - Transfer(address,address,uint256)
        - handler: handleLicenseCreated
          kind: substrate/EvmEventHandler
          filter:
            topics:
              - LicenseCreated(uint256,address,address)
```

### Schema Design

#### GraphQL Schema (schema.graphql)

```graphql
# Account entity representing users/addresses
type Account @entity {
  id: ID! # Address or DID
  address: String! @index
  did: String @index
  createdAt: Date!
  updatedAt: Date!
  
  # Relationships
  createdContent: [Content!] @derivedFrom(field: "creator")
  ownedTokens: [Token!] @derivedFrom(field: "owner")
  grantedLicenses: [License!] @derivedFrom(field: "licensor")
  receivedLicenses: [License!] @derivedFrom(field: "licensee")
  royaltyPayments: [RoyaltyPayment!] @derivedFrom(field: "recipient")
  
  # Aggregated data
  totalContentCreated: Int!
  totalTokensOwned: Int!
  totalRoyaltiesEarned: BigInt!
  reputation: Int!
}

# Content entity representing uploaded content
type Content @entity {
  id: ID! # Content CID
  cid: String! @index
  creator: Account!
  createdAt: Date!
  updatedAt: Date!
  
  # Metadata
  title: String
  description: String
  contentType: String! @index
  size: BigInt!
  mimeType: String
  
  # Storage information
  storageProvider: String!
  replicationFactor: Int!
  isAvailable: Boolean!
  
  # Relationships
  token: Token
  licenses: [License!] @derivedFrom(field: "content")
  analytics: ContentAnalytics
  
  # Social stats
  views: Int!
  likes: Int!
  shares: Int!
}

# Token entity representing SFTs
type Token @entity {
  id: ID! # Chain ID + Token ID
  tokenId: String! @index
  contractAddress: String! @index
  type: TokenType!
  
  # Ownership
  creator: Account!
  owner: Account!
  
  # Content reference
  content: Content!
  contentCid: String! @index
  
  # Token properties
  totalSupply: BigInt!
  decimals: Int
  symbol: String
  name: String!
  
  # Metadata
  metadata: String # JSON string
  properties: String # JSON string
  
  # Royalty configuration
  royaltyBasisPoints: Int!
  royaltyRecipients: [RoyaltyRecipient!] @derivedFrom(field: "token")
  
  # Market data
  currentPrice: BigInt
  lastSalePrice: BigInt
  lastSaleDate: Date
  
  # Relationships
  transfers: [Transfer!] @derivedFrom(field: "token")
  licenses: [License!] @derivedFrom(field: "token")
  listings: [MarketListing!] @derivedFrom(field: "token")
  
  # Status
  status: TokenStatus!
  createdAt: Date!
  updatedAt: Date!
}

# License entity representing usage rights
type License @entity {
  id: ID! # License ID
  token: Token!
  content: Content!
  
  # Parties
  licensor: Account!
  licensee: Account!
  
  # License details
  type: LicenseType!
  terms: String # JSON string
  price: BigInt
  
  # Time constraints
  issuedAt: Date!
  expiresAt: Date
  duration: Int # In seconds
  
  # Geographic constraints
  territories: [String!]
  isWorldwide: Boolean!
  
  # Status
  status: LicenseStatus!
  isRevocable: Boolean!
  isTransferable: Boolean!
  revocationReason: String
  
  # Transaction reference
  transactionHash: String! @index
}

# Royalty payment entity
type RoyaltyPayment @entity {
  id: ID! # Payment ID
  token: Token!
  sale: MarketSale!
  
  # Payment details
  recipient: Account!
  amount: BigInt!
  percentage: Float!
  
  # Transaction info
  transactionHash: String! @index
  blockNumber: Int!
  timestamp: Date!
  
  # Status
  status: PaymentStatus!
}

# Market listing entity
type MarketListing @entity {
  id: ID! # Listing ID
  token: Token!
  seller: Account!
  
  # Listing details
  price: BigInt!
  currency: String!
  quantity: BigInt!
  remainingQuantity: BigInt!
  
  # Time constraints
  createdAt: Date!
  expiresAt: Date
  
  # Status
  status: ListingStatus!
  isAuction: Boolean!
  
  # Auction specific
  reservePrice: BigInt
  highestBid: BigInt
  highestBidder: Account
  bidCount: Int
  
  # Relationships
  sales: [MarketSale!] @derivedFrom(field: "listing")
  bids: [AuctionBid!] @derivedFrom(field: "listing")
}

# Market sale entity
type MarketSale @entity {
  id: ID! # Sale ID
  listing: MarketListing!
  token: Token!
  
  # Parties
  seller: Account!
  buyer: Account!
  
  # Sale details
  price: BigInt!
  quantity: BigInt!
  totalAmount: BigInt!
  
  # Royalty distribution
  royaltyAmount: BigInt!
  sellerAmount: BigInt!
  
  # Transaction info
  transactionHash: String! @index
  blockNumber: Int!
  timestamp: Date!
  
  # Relationships
  royaltyPayments: [RoyaltyPayment!] @derivedFrom(field: "sale")
}

# Transfer entity
type Transfer @entity {
  id: ID! # Transfer ID
  token: Token!
  
  # Parties
  from: Account!
  to: Account!
  
  # Transfer details
  amount: BigInt!
  transferType: TransferType!
  
  # Transaction info
  transactionHash: String! @index
  blockNumber: Int!
  timestamp: Date!
}

# Analytics entity for content
type ContentAnalytics @entity {
  id: ID! # Same as Content ID
  content: Content!
  
  # View metrics
  totalViews: Int!
  uniqueViewers: Int!
  avgViewDuration: Int!
  
  # Engagement metrics
  totalLikes: Int!
  totalShares: Int!
  totalComments: Int!
  engagementRate: Float!
  
  # Revenue metrics
  totalRevenue: BigInt!
  royaltiesGenerated: BigInt!
  licensesIssued: Int!
  
  # Time series data (JSON)
  dailyViews: String
  monthlyRevenue: String
  
  lastUpdated: Date!
}

# Enums
enum TokenType {
  FUNGIBLE
  NON_FUNGIBLE
  SEMI_FUNGIBLE
}

enum TokenStatus {
  ACTIVE
  PAUSED
  BURNED
  LOCKED
}

enum LicenseType {
  VIEW_ONLY
  DISTRIBUTION
  COMMERCIAL
  DERIVATIVE
  EXCLUSIVE
}

enum LicenseStatus {
  ACTIVE
  EXPIRED
  REVOKED
  TRANSFERRED
}

enum ListingStatus {
  ACTIVE
  SOLD
  CANCELLED
  EXPIRED
}

enum PaymentStatus {
  PENDING
  COMPLETED
  FAILED
}

enum TransferType {
  MINT
  TRANSFER
  BURN
  SALE
}
```

### Mapping Handlers

#### Token Event Handlers

```typescript
// src/mappings/tokenHandlers.ts

import { SubstrateEvent, SubstrateExtrinsic } from "@subql/types";
import { Account, Token, Transfer, Content } from "../types";

export async function handleTokenCreated(event: SubstrateEvent): Promise<void> {
  const { data: [creator, tokenId, contentCid, metadata] } = event.event;
  
  // Create or get creator account
  let account = await Account.get(creator.toString());
  if (!account) {
    account = Account.create({
      id: creator.toString(),
      address: creator.toString(),
      createdAt: event.block.timestamp,
      updatedAt: event.block.timestamp,
      totalContentCreated: 0,
      totalTokensOwned: 0,
      totalRoyaltiesEarned: BigInt(0),
      reputation: 0
    });
  }
  
  // Create content record if not exists
  let content = await Content.get(contentCid.toString());
  if (!content) {
    content = Content.create({
      id: contentCid.toString(),
      cid: contentCid.toString(),
      creatorId: creator.toString(),
      createdAt: event.block.timestamp,
      updatedAt: event.block.timestamp,
      storageProvider: "IPFS",
      replicationFactor: 3,
      isAvailable: true,
      views: 0,
      likes: 0,
      shares: 0
    });
    await content.save();
  }
  
  // Create token record
  const token = Token.create({
    id: `${event.block.chainId}-${tokenId.toString()}`,
    tokenId: tokenId.toString(),
    contractAddress: event.address,
    type: "SEMI_FUNGIBLE",
    creatorId: creator.toString(),
    ownerId: creator.toString(),
    contentId: contentCid.toString(),
    contentCid: contentCid.toString(),
    totalSupply: BigInt(1),
    royaltyBasisPoints: 1000, // 10% default
    status: "ACTIVE",
    createdAt: event.block.timestamp,
    updatedAt: event.block.timestamp
  });
  
  // Update account stats
  account.totalContentCreated += 1;
  account.totalTokensOwned += 1;
  account.updatedAt = event.block.timestamp;
  
  // Save entities
  await Promise.all([
    account.save(),
    token.save()
  ]);
  
  // Create initial transfer record (mint)
  const transfer = Transfer.create({
    id: `${event.block.chainId}-${event.idx}`,
    tokenId: token.id,
    fromId: "0x0", // Mint from zero address
    toId: creator.toString(),
    amount: BigInt(1),
    transferType: "MINT",
    transactionHash: event.extrinsic.hash.toString(),
    blockNumber: event.block.block.header.number.toNumber(),
    timestamp: event.block.timestamp
  });
  
  await transfer.save();
}

export async function handleTokenTransfer(event: SubstrateEvent): Promise<void> {
  const { data: [from, to, tokenId, amount] } = event.event;
  
  // Get token
  const token = await Token.get(`${event.block.chainId}-${tokenId.toString()}`);
  if (!token) {
    logger.warn(`Token not found: ${tokenId}`);
    return;
  }
  
  // Update token owner if full transfer
  if (amount.toString() === token.totalSupply.toString()) {
    token.ownerId = to.toString();
    token.updatedAt = event.block.timestamp;
    await token.save();
  }
  
  // Create/update accounts
  let fromAccount = await Account.get(from.toString());
  let toAccount = await Account.get(to.toString());
  
  if (!toAccount) {
    toAccount = Account.create({
      id: to.toString(),
      address: to.toString(),
      createdAt: event.block.timestamp,
      updatedAt: event.block.timestamp,
      totalContentCreated: 0,
      totalTokensOwned: 0,
      totalRoyaltiesEarned: BigInt(0),
      reputation: 0
    });
  }
  
  // Update account stats
  if (fromAccount) {
    fromAccount.totalTokensOwned -= 1;
    fromAccount.updatedAt = event.block.timestamp;
    await fromAccount.save();
  }
  
  toAccount.totalTokensOwned += 1;
  toAccount.updatedAt = event.block.timestamp;
  await toAccount.save();
  
  // Create transfer record
  const transfer = Transfer.create({
    id: `${event.block.chainId}-${event.idx}`,
    tokenId: token.id,
    fromId: from.toString(),
    toId: to.toString(),
    amount: BigInt(amount.toString()),
    transferType: "TRANSFER",
    transactionHash: event.extrinsic.hash.toString(),
    blockNumber: event.block.block.header.number.toNumber(),
    timestamp: event.block.timestamp
  });
  
  await transfer.save();
}
```

#### License Event Handlers

```typescript
// src/mappings/licenseHandlers.ts

import { SubstrateEvent } from "@subql/types";
import { License, Token, Account } from "../types";

export async function handleLicenseCreated(event: SubstrateEvent): Promise<void> {
  const { data: [licenseId, tokenId, licensor, licensee, licenseType, terms, price] } = event.event;
  
  // Get token
  const token = await Token.get(`${event.block.chainId}-${tokenId.toString()}`);
  if (!token) {
    logger.warn(`Token not found for license: ${tokenId}`);
    return;
  }
  
  // Parse license terms
  const termsData = JSON.parse(terms.toString());
  
  // Create license record
  const license = License.create({
    id: licenseId.toString(),
    tokenId: token.id,
    contentId: token.contentId,
    licensorId: licensor.toString(),
    licenseeId: licensee.toString(),
    type: mapLicenseType(licenseType),
    terms: terms.toString(),
    price: BigInt(price.toString()),
    issuedAt: event.block.timestamp,
    expiresAt: termsData.expiresAt ? new Date(termsData.expiresAt) : null,
    duration: termsData.duration || null,
    territories: termsData.territories || [],
    isWorldwide: !termsData.territories || termsData.territories.length === 0,
    status: "ACTIVE",
    isRevocable: termsData.revocable || false,
    isTransferable: termsData.transferable || false,
    transactionHash: event.extrinsic.hash.toString()
  });
  
  await license.save();
}

export async function handleLicenseRevoked(event: SubstrateEvent): Promise<void> {
  const { data: [licenseId, reason] } = event.event;
  
  const license = await License.get(licenseId.toString());
  if (!license) {
    logger.warn(`License not found: ${licenseId}`);
    return;
  }
  
  license.status = "REVOKED";
  license.revocationReason = reason.toString();
  
  await license.save();
}

function mapLicenseType(type: any): string {
  const typeMap: Record<string, string> = {
    "0": "VIEW_ONLY",
    "1": "DISTRIBUTION",
    "2": "COMMERCIAL",
    "3": "DERIVATIVE",
    "4": "EXCLUSIVE"
  };
  
  return typeMap[type.toString()] || "VIEW_ONLY";
}
```

### Query Examples

#### Content Discovery Queries

```graphql
# Search for content by type
query SearchContent($contentType: String!, $limit: Int) {
  contents(
    first: $limit
    orderBy: CREATED_AT_DESC
    filter: {
      contentType: {
        equalTo: $contentType
      }
      isAvailable: {
        equalTo: true
      }
    }
  ) {
    nodes {
      id
      title
      description
      creator {
        id
        did
        reputation
      }
      token {
        id
        currentPrice
        royaltyBasisPoints
      }
      analytics {
        totalViews
        engagementRate
      }
    }
  }
}

# Get trending content
query TrendingContent($timeframe: String!, $limit: Int) {
  contents(
    first: $limit
    orderBy: VIEWS_DESC
    filter: {
      updatedAt: {
        greaterThan: $timeframe
      }
    }
  ) {
    nodes {
      id
      title
      views
      likes
      shares
      creator {
        id
        did
      }
    }
  }
}
```

#### Token Market Queries

```graphql
# Get active listings
query ActiveListings($tokenType: TokenType, $maxPrice: BigInt) {
  marketListings(
    filter: {
      status: {
        equalTo: ACTIVE
      }
      price: {
        lessThanOrEqualTo: $maxPrice
      }
      token: {
        type: {
          equalTo: $tokenType
        }
      }
    }
    orderBy: CREATED_AT_DESC
  ) {
    nodes {
      id
      price
      quantity
      seller {
        id
        reputation
      }
      token {
        id
        name
        content {
          title
          contentType
        }
      }
    }
  }
}

# Get token price history
query TokenPriceHistory($tokenId: ID!) {
  marketSales(
    filter: {
      token: {
        id: {
          equalTo: $tokenId
        }
      }
    }
    orderBy: TIMESTAMP_DESC
    first: 50
  ) {
    nodes {
      price
      timestamp
      quantity
    }
  }
}
```

#### Account Analytics Queries

```graphql
# Get creator analytics
query CreatorAnalytics($creatorId: ID!) {
  account(id: $creatorId) {
    id
    totalContentCreated
    totalRoyaltiesEarned
    reputation
    createdContent(orderBy: VIEWS_DESC, first: 10) {
      nodes {
        id
        title
        views
        totalRevenue
      }
    }
    royaltyPayments(orderBy: TIMESTAMP_DESC, first: 10) {
      nodes {
        amount
        timestamp
        token {
          name
        }
      }
    }
  }
}
```

## Deployment Options

### Self-Hosted Deployment

```yaml
# docker-compose.yml for self-hosted deployment
version: '3'

services:
  postgres:
    image: postgres:14-alpine
    ports:
      - 5432:5432
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: postgres
    volumes:
      - postgres-data:/var/lib/postgresql/data

  subquery-node:
    image: onfinality/subql-node:latest
    depends_on:
      - postgres
    restart: always
    environment:
      DB_USER: postgres
      DB_PASS: postgres
      DB_DATABASE: postgres
      DB_HOST: postgres
      DB_PORT: 5432
    volumes:
      - ./:/app
    command:
      - -f=/app
      - --db-schema=app
      - --unsafe
    ports:
      - 3000:3000

  graphql-engine:
    image: onfinality/subql-query:latest
    depends_on:
      - postgres
      - subquery-node
    restart: always
    environment:
      DB_USER: postgres
      DB_PASS: postgres
      DB_DATABASE: postgres
      DB_HOST: postgres
      DB_PORT: 5432
    command:
      - --name=app
      - --playground
    ports:
      - 3001:3000

  redis:
    image: redis:7-alpine
    ports:
      - 6379:6379
    volumes:
      - redis-data:/data

volumes:
  postgres-data:
  redis-data:
```

### SubQuery Managed Service

```yaml
# subquery-project.yaml for managed service deployment
specVersion: 1.0.0
name: blackhole-indexer
version: 1.0.0
description: Blackhole platform blockchain indexer
repository: https://github.com/blackhole/subquery

network:
  chainId: "0x..." # Root Network chain ID
  endpoint: wss://root-network.io/ws

deployment:
  genesisHash: "0x..." # Root Network genesis hash
  indexer:
    api: https://api.subquery.network/sq/blackhole/indexer
    gateway: https://gateway.subquery.network/query/blackhole
  
monitoring:
  enabled: true
  alerts:
    - type: INDEXING_DELAY
      threshold: 300 # 5 minutes
    - type: ERROR_RATE
      threshold: 0.05 # 5% error rate
```

### Kubernetes Deployment

```yaml
# kubernetes deployment configuration
apiVersion: apps/v1
kind: Deployment
metadata:
  name: subquery-indexer
  namespace: blackhole
spec:
  replicas: 3
  selector:
    matchLabels:
      app: subquery-indexer
  template:
    metadata:
      labels:
        app: subquery-indexer
    spec:
      containers:
      - name: subquery-node
        image: onfinality/subql-node:latest
        env:
        - name: DB_HOST
          value: postgres-service
        - name: DB_PORT
          value: "5432"
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: username
        - name: DB_PASS
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
        livenessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: subquery-service
  namespace: blackhole
spec:
  selector:
    app: subquery-indexer
  ports:
  - protocol: TCP
    port: 3000
    targetPort: 3000
  type: LoadBalancer
```

## Monitoring and Operations

### Health Monitoring

```typescript
// src/utils/monitoring.ts

export interface IndexerHealth {
  chainHeight: number;
  indexedHeight: number;
  indexingDelay: number;
  errorRate: number;
  queueSize: number;
}

export async function checkIndexerHealth(): Promise<IndexerHealth> {
  const chainHeight = await api.query.system.number();
  const indexedHeight = await getLastIndexedBlock();
  
  return {
    chainHeight: chainHeight.toNumber(),
    indexedHeight,
    indexingDelay: chainHeight.toNumber() - indexedHeight,
    errorRate: await calculateErrorRate(),
    queueSize: await getQueueSize()
  };
}

export async function monitorPerformance() {
  setInterval(async () => {
    const health = await checkIndexerHealth();
    
    // Log metrics
    logger.info('Indexer health:', health);
    
    // Alert if delay is too high
    if (health.indexingDelay > 100) {
      logger.warn('Indexing delay is high:', health.indexingDelay);
      await sendAlert('High indexing delay', health);
    }
    
    // Alert if error rate is high
    if (health.errorRate > 0.05) {
      logger.error('High error rate:', health.errorRate);
      await sendAlert('High error rate', health);
    }
  }, 60000); // Check every minute
}
```

### Query Performance Monitoring

```graphql
# Monitor query performance
query IndexerMetrics {
  _metadata {
    chainHeight
    indexerHeight
    indexerHealthy
    lastProcessedHeight
    lastProcessedTimestamp
    specName
    specVersion
    targetHeight
  }
}
```

## Best Practices

### Schema Design

1. **Use Proper Indices**
   - Add `@index` to frequently queried fields
   - Use composite indices for complex queries
   - Avoid over-indexing

2. **Optimize Relationships**
   - Use `@derivedFrom` for reverse lookups
   - Minimize nested queries
   - Consider denormalization for performance

3. **Handle Large Datasets**
   - Implement pagination
   - Use time-based partitioning
   - Archive old data

### Mapping Implementation

1. **Error Handling**
   ```typescript
   export async function handleEvent(event: SubstrateEvent): Promise<void> {
     try {
       // Process event
     } catch (error) {
       logger.error(`Error processing event: ${error.message}`);
       // Store error for monitoring
       await storeError(event, error);
     }
   }
   ```

2. **Batch Operations**
   ```typescript
   export async function handleBatchTransfer(transfers: Transfer[]): Promise<void> {
     const entities = transfers.map(transfer => createTransferEntity(transfer));
     await store.bulkCreate('Transfer', entities);
   }
   ```

3. **Data Validation**
   ```typescript
   export function validateTokenData(data: any): boolean {
     if (!data.tokenId || !data.creator) {
       logger.warn('Invalid token data');
       return false;
     }
     return true;
   }
   ```

### Performance Optimization

1. **Query Optimization**
   - Use specific field selections
   - Implement proper pagination
   - Cache frequently accessed data

2. **Indexing Optimization**
   - Process events in batches
   - Use parallel processing where possible
   - Implement incremental indexing

3. **Storage Optimization**
   - Regular data pruning
   - Archive historical data
   - Optimize database configurations

## Migration Strategy

### From Existing System

1. **Data Export**
   - Export existing indexed data
   - Transform to SubQuery schema
   - Validate data integrity

2. **Parallel Running**
   - Run both systems temporarily
   - Compare outputs
   - Verify consistency

3. **Gradual Migration**
   - Migrate read queries first
   - Test thoroughly
   - Switch write operations
   - Decommission old system

### Schema Evolution

1. **Backward Compatibility**
   - Add new fields as optional
   - Deprecate fields gradually
   - Maintain version history

2. **Migration Scripts**
   ```typescript
   export async function migrateSchema(fromVersion: string, toVersion: string) {
     // Handle schema migrations
     switch (`${fromVersion}-${toVersion}`) {
       case '1.0.0-1.1.0':
         await addNewFields();
         break;
       case '1.1.0-2.0.0':
         await restructureData();
         break;
     }
   }
   ```

## Security Considerations

1. **API Security**
   - Implement rate limiting
   - Use API keys for access
   - Monitor for abuse

2. **Data Privacy**
   - Filter sensitive information
   - Implement access controls
   - Follow GDPR requirements

3. **Infrastructure Security**
   - Use secure connections
   - Encrypt data at rest
   - Regular security audits

## Conclusion

SubQuery provides the ideal indexing solution for Blackhole's needs, offering native Substrate support, EVM compatibility, and a proven architecture for blockchain data indexing. This implementation guide provides the foundation for building a robust, scalable indexing system that can grow with the platform.

The modular design allows for easy extension and modification as new requirements emerge, while the performance optimizations ensure the system can handle high-volume data processing efficiently.

---

By following this implementation guide, the Blackhole platform can leverage SubQuery's powerful indexing capabilities to provide fast, reliable access to blockchain data while maintaining the flexibility to evolve with the platform's needs.