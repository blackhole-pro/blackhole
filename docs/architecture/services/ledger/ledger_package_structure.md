# Ledger Services Package Structure

This document outlines the package structure for the Blackhole ledger services, following the standardized project architecture with a provider-based approach that enables multi-chain support.

## Overview

The ledger services are organized across three main packages with a provider-based architecture:

1. **Node Services** (`@blackhole/node/services/ledger`): P2P infrastructure for blockchain interaction
2. **Client SDK** (`@blackhole/client-sdk/services/ledger`): Service provider tools for ledger functionality
3. **Shared Types** (`@blackhole/shared/types/ledger`): Common types and interfaces for ledger services

## Provider-Based Architecture

The ledger services implement a provider pattern that:

1. **Abstracts Chain-Specific Logic**: Isolates blockchain-specific code behind common interfaces
2. **Enables Multi-Chain Support**: Allows addition of new blockchain providers with minimal changes
3. **Provides Consistent APIs**: Maintains uniform interfaces regardless of underlying blockchain
4. **Supports Provider Switching**: Facilitates migration between different blockchain providers
5. **Maintains Separation of Concerns**: Cleanly divides core functionality from provider implementations

## Detailed Package Structure

### Node Services

```
@blackhole/node/services/ledger/             # P2P ledger infrastructure
├── core/                                    # Core ledger functionality
│   ├── connection.ts                        # Provider connection management
│   ├── transaction.ts                       # Common transaction interfaces
│   ├── events.ts                            # Generalized event handling
│   └── sync.ts                              # Abstract chain synchronization
│
├── providers/                               # Blockchain provider implementations
│   ├── provider.ts                          # Provider interface definition
│   ├── registry.ts                          # Provider registry and factory
│   │
│   ├── root-network/                        # Root Network provider implementation
│   │   ├── client.ts                        # Root Network client
│   │   ├── transaction.ts                   # Transaction implementation
│   │   ├── events.ts                        # Event handling implementation
│   │   ├── blocks.ts                        # Block processing 
│   │   ├── sft.ts                           # SFT implementation
│   │   └── contracts/                       # Root Network contracts
│   │       ├── token.ts                     # Token contract interface
│   │       ├── marketplace.ts               # Marketplace contract interface
│   │       └── abi/                         # Contract ABIs
│   │
│   └── future-providers/                    # Placeholder for future providers
│       └── another-chain/                   # Another blockchain implementation
│
├── tokenization/                            # Content tokenization system
│   ├── factory.ts                           # Provider-agnostic token creation
│   ├── metadata.ts                          # Token metadata management
│   ├── classes.ts                           # Content class definitions
│   └── verification.ts                      # Token verification service
│
├── contracts/                               # Smart contract interactions
│   ├── registry.ts                          # Contract registry
│   ├── deployment.ts                        # Provider-agnostic deployment
│   └── interaction.ts                       # Contract method calls
│
├── royalties/                               # Royalty management system
│   ├── distribution.ts                      # Revenue distribution logic
│   ├── calculation.ts                       # Royalty calculation
│   ├── settlement.ts                        # Payment settlement
│   └── verification.ts                      # Royalty compliance checking
│
├── rights/                                  # Rights management
│   ├── licensing.ts                         # License creation and verification
│   ├── enforcement.ts                       # Rights enforcement
│   ├── verification.ts                      # License verification
│   └── revocation.ts                        # License revocation
│
├── marketplace/                             # Marketplace functionality
│   ├── listings.ts                          # Content listing management
│   ├── sales.ts                             # Sales processing
│   ├── auctions.ts                          # Auction mechanisms
│   └── subscriptions.ts                     # Subscription management
│
└── api/                                     # API layer
    ├── routes.ts                            # API route definitions
    ├── controllers.ts                       # Request handlers
    ├── validation.ts                        # Request validation
    └── middleware.ts                        # API middleware
```

### Client SDK

```
@blackhole/client-sdk/services/ledger/       # Service provider ledger tools
├── client.ts                                # Main ledger client (provider-agnostic)
├── transaction.ts                           # Transaction building
├── wallet.ts                                # Wallet operations
│
├── providers/                               # Provider-specific client adapters
│   ├── provider.ts                          # Provider interface
│   ├── factory.ts                           # Provider factory
│   │
│   ├── root-network/                        # Root Network client adapter
│   │   ├── client.ts                        # Root Network client adapter
│   │   ├── wallet.ts                        # Root Network wallet adapter
│   │   └── transaction.ts                   # Transaction formatting
│   │
│   └── future-providers/                    # Placeholder for future providers
│       └── another-chain/                   # Another blockchain client adapter
│
├── tokenization/                            # Token operations
│   ├── client.ts                            # Provider-agnostic token client
│   ├── creation.ts                          # Token creation workflow
│   ├── metadata.ts                          # Metadata management
│   └── classes.ts                           # Content classes
│
├── royalties/                               # Royalty client functionality
│   ├── client.ts                            # Royalty client
│   ├── configuration.ts                     # Royalty setup
│   ├── tracking.ts                          # Revenue tracking
│   └── distribution.ts                      # Distribution management
│
├── rights/                                  # Rights management client
│   ├── client.ts                            # Rights client
│   ├── licensing.ts                         # License operations
│   ├── verification.ts                      # Rights verification
│   └── templates.ts                         # License templates
│
├── marketplace/                             # Marketplace client
│   ├── client.ts                            # Marketplace client
│   ├── listings.ts                          # Content listing
│   ├── purchases.ts                         # Purchase workflow
│   └── subscriptions.ts                     # Subscription management
│
└── analytics/                               # Ledger analytics
    ├── transactions.ts                      # Transaction analysis
    ├── revenue.ts                           # Revenue tracking
    ├── usage.ts                             # Content usage metrics
    └── reporting.ts                         # Financial reporting
```

### Shared Types

```
@blackhole/shared/types/ledger/              # Shared ledger types
├── transaction.ts                           # Common transaction types
├── block.ts                                 # Block type definitions
├── chain.ts                                 # Chain type definitions
│
├── providers/                               # Provider-related types
│   ├── provider.ts                          # Provider interface types
│   ├── root-network/                        # Root Network specific types
│   │   ├── transaction.ts                   # Root Network transaction types
│   │   ├── block.ts                         # Root Network block types
│   │   └── sft.ts                           # SFT type definitions
│   │
│   └── future-providers/                    # Types for future providers
│       └── another-chain/                   # Another blockchain types
│
├── token/                                   # Provider-agnostic token types
│   ├── token.ts                             # Core token definitions
│   ├── metadata.ts                          # Metadata type definitions
│   ├── class.ts                             # Token class definitions
│   └── events.ts                            # Token event types
│
├── royalties/                               # Royalty types
│   ├── distribution.ts                      # Distribution model types
│   ├── payment.ts                           # Payment type definitions
│   ├── split.ts                             # Revenue split types
│   └── schedule.ts                          # Payment schedule types
│
├── rights/                                  # Rights management types
│   ├── license.ts                           # License type definitions
│   ├── terms.ts                             # License terms types
│   ├── usage.ts                             # Usage rights types
│   └── verification.ts                      # Verification types
│
├── marketplace/                             # Marketplace types
│   ├── listing.ts                           # Listing type definitions
│   ├── offer.ts                             # Offer type definitions
│   ├── sale.ts                              # Sale type definitions
│   └── subscription.ts                      # Subscription type definitions
│
└── events/                                  # Event types
    ├── token.ts                             # Token event types
    ├── marketplace.ts                       # Marketplace event types
    ├── rights.ts                            # Rights event types
    └── royalty.ts                           # Royalty event types
```

## Provider Implementation Pattern

The provider-based architecture follows these implementation patterns:

### Provider Interface

All blockchain providers implement a common interface:

```typescript
// In @blackhole/shared/types/ledger/providers/provider.ts
export interface BlockchainProvider {
  // Core functionality
  connect(): Promise<ConnectionStatus>;
  disconnect(): Promise<void>;
  getStatus(): ConnectionStatus;
  
  // Transaction handling
  createTransaction(params: TransactionParams): Transaction;
  signTransaction(tx: Transaction, signer: Signer): Promise<SignedTransaction>;
  submitTransaction(tx: SignedTransaction): Promise<TransactionResult>;
  getTransaction(txId: string): Promise<Transaction>;
  
  // Token operations
  createToken(params: TokenCreationParams): Promise<TokenCreationResult>;
  getToken(tokenId: string): Promise<Token>;
  transferToken(params: TokenTransferParams): Promise<TransferResult>;
  
  // Contract operations
  callContract(params: ContractCallParams): Promise<ContractCallResult>;
  deployContract(params: ContractDeployParams): Promise<ContractDeployResult>;
  
  // Event handling
  subscribeToEvents(eventType: EventType, callback: EventCallback): Subscription;
  unsubscribeFromEvents(subscription: Subscription): Promise<void>;
}
```

### Provider Factory

Providers are instantiated through a factory pattern:

```typescript
// In @blackhole/node/services/ledger/providers/registry.ts
export class ProviderRegistry {
  private providers: Map<string, Provider>;
  
  registerProvider(name: string, provider: Provider): void {
    this.providers.set(name, provider);
  }
  
  getProvider(name: string): Provider {
    const provider = this.providers.get(name);
    if (!provider) {
      throw new Error(`Provider ${name} not registered`);
    }
    return provider;
  }
  
  createProvider(name: string, config: ProviderConfig): Provider {
    const Provider = this.getProvider(name);
    return new Provider(config);
  }
}
```

### Root Network Provider Implementation

The Root Network provider implements the provider interface with SFT-specific functionality:

```typescript
// In @blackhole/node/services/ledger/providers/root-network/client.ts
export class RootNetworkProvider implements BlockchainProvider {
  constructor(private config: RootNetworkConfig) {}
  
  // Implement provider interface methods
  
  // Root Network specific methods
  createSFT(params: SFTCreationParams): Promise<SFTCreationResult> {
    // Implementation for Root Network SFT creation
  }
  
  getSFTCollection(collectionId: string): Promise<SFTCollection> {
    // Implementation for Root Network SFT collection retrieval
  }
  
  // Additional Root Network functionality
}
```

## Provider Configuration

The system supports runtime provider configuration:

```typescript
// In @blackhole/node/services/ledger/config.ts
export interface LedgerConfig {
  defaultProvider: string;
  providers: {
    [providerName: string]: ProviderConfig;
  };
}

// Configuration example
const ledgerConfig: LedgerConfig = {
  defaultProvider: 'root-network',
  providers: {
    'root-network': {
      rpcUrl: 'https://rpc.rootnet.io',
      networkId: 1,
      apiKey: process.env.ROOT_NETWORK_API_KEY,
      contractAddresses: {
        tokenFactory: '0x...',
        marketplace: '0x...',
        rights: '0x...'
      }
    },
    // Additional providers can be configured here
  }
};
```

## Adding New Providers

To add a new blockchain provider, developers would:

1. **Implement Provider Interface**: Create a new provider class implementing the `BlockchainProvider` interface
2. **Add Provider Types**: Define provider-specific types in `@blackhole/shared/types/ledger/providers/new-provider/`
3. **Register Provider**: Add the provider to the registry
4. **Add Configuration**: Configure the provider in the ledger configuration

## Cross-Chain Functionality

The architecture supports several cross-chain scenarios:

1. **Multi-Provider Deployments**: Running nodes with different providers in the same network
2. **Token Bridging**: Moving tokens between different blockchain networks (requires additional bridge components)
3. **Provider Migration**: Supporting migration of tokens from one provider to another
4. **Hybrid Operations**: Using different providers for different operations (e.g., one for marketplace, another for rights management)

## Implementation Considerations

### Provider Abstraction Boundaries

The architecture carefully balances abstraction with provider-specific optimizations:

1. **Core Domain Logic**: Completely provider-agnostic
2. **Data Models**: Provider-agnostic with provider-specific extensions
3. **Contract Interactions**: Provider-specific implementations behind common interfaces
4. **Event Processing**: Provider-specific event handling with standardized output formats

### Provider-Specific Optimizations

Each provider can implement optimizations specific to its blockchain:

1. **Root Network**: Optimized for SFT operations and batch transactions
2. **Future EVM Providers**: Could leverage EVM-specific optimizations
3. **Future Substrate Providers**: Could utilize Substrate-specific features

### Migration Considerations

The architecture supports future provider migrations:

1. **Data Mapping**: Clear mappings between generic types and provider-specific types
2. **State Synchronization**: Mechanisms to synchronize state between providers during migration
3. **Dual-Mode Operation**: Support for operating with multiple providers during transition periods