# Blackhole DID System Architecture

This document provides a comprehensive overview of the Decentralized Identifier (DID) system within the Blackhole platform, including the DID architecture, registry functionality, and resolution mechanisms.

## Introduction

The Blackhole DID system implements self-sovereign identity through decentralized identifiers, leveraging IPFS for document storage and integrating registry functionality directly into the P2P node network. This design provides a lightweight, flexible identity foundation that supports cross-method authentication without requiring a separate blockchain registry.

## System Overview

The DID system combines several key capabilities:

1. **DID Creation and Management**: Generation and control of decentralized identifiers
2. **Document Storage**: IPFS-based storage for DID documents
3. **Registry Services**: Discovery and resolution integrated into P2P nodes
4. **Universal Resolution**: Support for multiple DID methods
5. **Verification Framework**: Cryptographic authentication and validation

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────┐
│                   Service Provider Layer                 │
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Wallet     │  │   Identity   │  │   Platform   │  │
│  │  Interface   │  │   Services   │  │   Services   │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│                                                         │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    P2P Node Layer                        │
│                                                         │
│  ┌────────────────────────────────────────────────────┐ │
│  │              DID Registry Service                  │ │
│  │                                                    │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │ │
│  │  │   DID    │  │Registry  │  │Universal │       │ │
│  │  │Operations│  │  Index   │  │ Resolver │       │ │
│  │  └──────────┘  └──────────┘  └──────────┘       │ │
│  │                                                    │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │ │
│  │  │Document  │  │Discovery │  │  Trust   │       │ │
│  │  │ Manager  │  │  Engine  │  │Framework │       │ │
│  │  └──────────┘  └──────────┘  └──────────┘       │ │
│  │                                                    │ │
│  └────────────────────────────────────────────────────┘ │
│                                                         │
│  ┌─────────────────┐              ┌─────────────────┐  │
│  │  Storage Service│              │ Network Service │  │
│  │     (IPFS)      │              │                 │  │
│  └─────────────────┘              └─────────────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### DID Method Specification (`did:blackhole`)

- **Format**: `did:blackhole:<identifier>`
- **Identifier**: Cryptographically generated or derived from public key
- **Storage**: DID documents stored on IPFS with CID-based addressing
- **Resolution**: Direct IPFS retrieval through P2P nodes

### Registry Integration

The DID registry is not a separate service but an integrated capability of P2P nodes:

- **Index Management**: Efficient indexes for DID resolution and discovery
- **Consensus Integration**: Registry updates coordinated via P2P consensus
- **Data Distribution**: Registry data replicated across nodes
- **API Layer**: Clean interfaces for registry operations

## Key Workflows

### 1. DID Creation and Registration

```
┌───────────┐         ┌───────────┐         ┌───────────┐
│   User    │         │  Service  │         │ P2P Node  │
│           │         │ Provider  │         │           │
└─────┬─────┘         └─────┬─────┘         └─────┬─────┘
      │                     │                     │
      │  Create DID Request │                     │
      │─────────────────────►                     │
      │                     │                     │
      │                     │  Generate DID       │
      │                     │─────────────────────►
      │                     │                     │
      │                     │  Create Document    │
      │                     │  Store on IPFS      │
      │                     │                     │
      │                     │  Update Registry    │
      │                     │  Index              │
      │                     │                     │
      │                     │  Return DID & CID   │
      │                     │◄─────────────────────
      │                     │                     │
      │  DID Created        │                     │
      │◄─────────────────────                     │
      │                     │                     │
```

### 2. DID Resolution

```
┌───────────┐         ┌───────────┐         ┌───────────┐
│ Requester │         │  Service  │         │ P2P Node  │
│           │         │ Provider  │         │           │
└─────┬─────┘         └─────┬─────┘         └─────┬─────┘
      │                     │                     │
      │  Resolve DID        │                     │
      │─────────────────────►                     │
      │                     │                     │
      │                     │  Query Registry     │
      │                     │─────────────────────►
      │                     │                     │
      │                     │  Fetch from IPFS   │
      │                     │                     │
      │                     │  Verify Document    │
      │                     │                     │
      │                     │  Return Document    │
      │                     │◄─────────────────────
      │                     │                     │
      │  DID Document       │                     │
      │◄─────────────────────                     │
      │                     │                     │
```

### 3. DID Discovery

```
┌───────────┐         ┌───────────┐         ┌───────────┐
│ Requester │         │  Service  │         │ P2P Node  │
│           │         │ Provider  │         │           │
└─────┬─────┘         └─────┬─────┘         └─────┬─────┘
      │                     │                     │
      │  Discovery Query    │                     │
      │─────────────────────►                     │
      │                     │                     │
      │                     │  Search Registry    │
      │                     │─────────────────────►
      │                     │                     │
      │                     │  Apply Filters      │
      │                     │  Check Privacy      │
      │                     │                     │
      │                     │  Return Results     │
      │                     │◄─────────────────────
      │                     │                     │
      │  Matching DIDs      │                     │
      │◄─────────────────────                     │
      │                     │                     │
```

## DID Document Format

Standard W3C-compliant DID documents with Blackhole extensions:

```json
{
  "@context": ["https://www.w3.org/ns/did/v1"],
  "id": "did:blackhole:123456789abcdef",
  "verificationMethod": [
    {
      "id": "#key-1",
      "type": "Ed25519VerificationKey2020",
      "controller": "did:blackhole:123456789abcdef",
      "publicKeyMultibase": "zH3C2AVvLMv6gmMNam3uVAjZpfkcJCwDwnZn6z3wXmqPV"
    }
  ],
  "authentication": ["#key-1"],
  "service": [
    {
      "id": "#storage",
      "type": "IPFSStorageService",
      "serviceEndpoint": "ipfs://bafybeig6xv5nwphfmvcnektpnojts33jqcuam7bmye2pb54adnrtccjlsu"
    }
  ]
}
```

## Registry Data Model

### Registry Entry

```json
{
  "did": "did:blackhole:123456789abcdef",
  "currentCid": "bafybeig6xv5nwphfmvcnektpnojts33jqcuam7bmye2pb54adnrtccjlsu",
  "created": "2023-05-10T08:30:15Z",
  "updated": "2023-08-15T14:30:45Z",
  "controller": "did:blackhole:123456789abcdef",
  "status": "active",
  "version": 3,
  "discoverySettings": {
    "public": true,
    "searchable": true,
    "indexedProperties": ["name", "serviceEndpoint.type"]
  }
}
```

## Discovery and Resolution

### Universal Resolver

The system includes a universal resolver that handles multiple DID methods:

- **Method Detection**: Automatic identification of DID method
- **Routing**: Appropriate handler selection for each method
- **External Methods**: Support for `did:key`, `did:web`, `did:ethr`, etc.
- **Cross-Method**: Resolution of referenced DIDs across methods

### Discovery Capabilities

The registry provides rich discovery features:

- **Property Search**: Find DIDs by document properties
- **Service Discovery**: Locate DIDs offering specific services
- **Controller Search**: Find DIDs controlled by specific entities
- **Privacy Controls**: User-defined discovery permissions

### Discovery Query Example

```json
{
  "queryType": "discovery",
  "filters": [
    {
      "field": "service.type",
      "operator": "equals",
      "value": "StorageService"
    }
  ],
  "limit": 25,
  "includeMetadata": true
}
```

## Security and Privacy

### Permission Model

- **Control Permissions**: Update, deactivate, delegate control
- **Discovery Permissions**: Public, limited, or private discovery
- **Resolution Permissions**: Standard or selective resolution

### Privacy Settings

```json
{
  "did": "did:blackhole:123456789abcdef",
  "privacySettings": {
    "discoverable": "public",
    "indexableProperties": ["service.type"],
    "authorizedResolvers": [],
    "historyVisibility": "controllers"
  }
}
```

## Implementation Structure

### Node Implementation

```
@blackhole/node/
├── services/
│   └── identity/
│       ├── did/                  # DID operations
│       │   ├── creation.ts       # DID creation
│       │   ├── resolution.ts     # Resolution logic
│       │   └── verification.ts   # Verification
│       │
│       └── registry/             # Registry service
│           ├── storage.ts        # Registry storage
│           ├── index.ts          # Indexing service
│           ├── discovery.ts      # Discovery engine
│           └── sync.ts           # Synchronization
```

### Client SDK

```
@blackhole/client-sdk/
├── services/
│   └── identity/
│       ├── did-client.ts         # DID client interface
│       └── registry-client.ts    # Registry operations
```

## Performance Specifications

- **Resolution**: <500ms for cached DIDs, <2s for uncached
- **Creation**: <3s for complete DID registration
- **Discovery**: <1s for indexed queries
- **Throughput**: >1000 resolutions/second per node

## Standards Compliance

- W3C DID Core Specification v1.0
- W3C DID Resolution Specification
- DIF Universal Resolver Integration
- IPFS Content Addressing Standards

## Conclusion

The Blackhole DID system provides a comprehensive identity foundation that combines decentralized identifiers with integrated registry services. By leveraging the P2P node network and IPFS storage, the system delivers high performance, resilience, and privacy while maintaining standards compliance and interoperability.

---

*This document combines the DID architecture and registry functionality into a unified system description.*