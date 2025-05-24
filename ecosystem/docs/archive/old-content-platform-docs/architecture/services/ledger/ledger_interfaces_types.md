# Ledger Services: Interfaces and Types

This document outlines the key interfaces and types that form the foundation of the Blackhole ledger services. These definitions provide a standardized way to interact with blockchain providers while maintaining flexibility for different implementations.

## Core Provider Interfaces

### Blockchain Provider Interface

The core provider interface that all blockchain implementations must satisfy:

```typescript
// @blackhole/shared/types/ledger/providers/provider.ts

export interface ConnectionStatus {
  connected: boolean;
  networkId: string;
  latestBlock: number;
  syncStatus: SyncStatus;
}

export interface TransactionParams {
  from: string;
  to?: string;
  value?: BigNumber;
  data?: string;
  gasLimit?: BigNumber;
  gasPrice?: BigNumber;
  nonce?: number;
}

export interface BlockchainProvider {
  /**
   * Get provider name
   */
  readonly name: string;
  
  /**
   * Connect to the blockchain network
   */
  connect(): Promise<ConnectionStatus>;
  
  /**
   * Disconnect from the blockchain network
   */
  disconnect(): Promise<void>;
  
  /**
   * Get current connection status
   */
  getStatus(): ConnectionStatus;
  
  /**
   * Create a new transaction
   */
  createTransaction(params: TransactionParams): Transaction;
  
  /**
   * Sign a transaction with the provided signer
   */
  signTransaction(tx: Transaction, signer: Signer): Promise<SignedTransaction>;
  
  /**
   * Submit a signed transaction to the network
   */
  submitTransaction(tx: SignedTransaction): Promise<TransactionResult>;
  
  /**
   * Get transaction details by ID
   */
  getTransaction(txId: string): Promise<Transaction>;
  
  /**
   * Get transaction receipt by ID
   */
  getTransactionReceipt(txId: string): Promise<TransactionReceipt>;
  
  /**
   * Estimate gas for a transaction
   */
  estimateGas(params: TransactionParams): Promise<BigNumber>;
  
  /**
   * Get current gas price
   */
  getGasPrice(): Promise<BigNumber>;
  
  /**
   * Get balance for an address
   */
  getBalance(address: string, tokenAddress?: string): Promise<BigNumber>;
  
  /**
   * Subscribe to blockchain events
   */
  subscribeToEvents(filters: EventFilter, callback: EventCallback): Subscription;
  
  /**
   * Unsubscribe from events
   */
  unsubscribeFromEvents(subscription: Subscription): Promise<void>;
}
```

### Token Provider Interface

Interface for token operations across different providers:

```typescript
// @blackhole/shared/types/ledger/providers/token-provider.ts

export interface TokenCreationParams {
  creator: string;
  name: string;
  symbol: string;
  initialSupply?: BigNumber;
  decimals?: number;
  metadata?: TokenMetadata;
  contentCid?: string;
}

export interface TokenProvider {
  /**
   * Create a new token
   */
  createToken(params: TokenCreationParams): Promise<TokenCreationResult>;
  
  /**
   * Get token details
   */
  getToken(tokenId: string): Promise<Token>;
  
  /**
   * Transfer token ownership
   */
  transferToken(params: TokenTransferParams): Promise<TransferResult>;
  
  /**
   * Get token balance for an address
   */
  getTokenBalance(tokenId: string, address: string): Promise<BigNumber>;
  
  /**
   * Get tokens owned by an address
   */
  getTokensForAddress(address: string): Promise<Token[]>;
  
  /**
   * Update token metadata
   */
  updateTokenMetadata(tokenId: string, metadata: TokenMetadata): Promise<TokenUpdateResult>;
}
```

### Semi-Fungible Token Provider Interface

Extension for SFT-specific functionality:

```typescript
// @blackhole/shared/types/ledger/providers/sft-provider.ts

export interface SFTCreationParams extends TokenCreationParams {
  classId: string;
  properties: Record<string, any>;
  maxSupply?: BigNumber;
  batchSize?: number;
}

export interface SFTProvider extends TokenProvider {
  /**
   * Create a new SFT collection
   */
  createSFTCollection(params: SFTCollectionParams): Promise<SFTCollectionResult>;
  
  /**
   * Create a new SFT within a collection
   */
  createSFT(params: SFTCreationParams): Promise<SFTCreationResult>;
  
  /**
   * Mint additional tokens of an existing SFT
   */
  mintSFT(sftId: string, amount: BigNumber, recipient: string): Promise<MintResult>;
  
  /**
   * Burn SFT tokens
   */
  burnSFT(sftId: string, amount: BigNumber): Promise<BurnResult>;
  
  /**
   * Get SFT collection details
   */
  getSFTCollection(collectionId: string): Promise<SFTCollection>;
  
  /**
   * Get properties for an SFT
   */
  getSFTProperties(sftId: string): Promise<Record<string, any>>;
  
  /**
   * Update SFT properties
   */
  updateSFTProperties(sftId: string, properties: Record<string, any>): Promise<SFTUpdateResult>;
}
```

### Rights Management Provider Interface

Interface for token-based rights management:

```typescript
// @blackhole/shared/types/ledger/providers/rights-provider.ts

export interface LicenseParams {
  tokenId: string;
  licenseType: LicenseType;
  terms: LicenseTerms;
  licensor: string;
  licensee: string;
  duration?: number; // In seconds, 0 for perpetual
  territorial?: string[]; // ISO country codes, empty for worldwide
  revocable: boolean;
}

export interface RightsProvider {
  /**
   * Create a new license
   */
  createLicense(params: LicenseParams): Promise<LicenseResult>;
  
  /**
   * Verify if a license is valid
   */
  verifyLicense(licenseId: string): Promise<LicenseVerificationResult>;
  
  /**
   * Get license details
   */
  getLicense(licenseId: string): Promise<License>;
  
  /**
   * Get all licenses for a token
   */
  getLicensesForToken(tokenId: string): Promise<License[]>;
  
  /**
   * Get all licenses granted to an address
   */
  getLicensesForAddress(address: string): Promise<License[]>;
  
  /**
   * Revoke a license (if revocable)
   */
  revokeLicense(licenseId: string, reason: string): Promise<RevocationResult>;
  
  /**
   * Transfer a license to a new licensee (if transferable)
   */
  transferLicense(licenseId: string, newLicensee: string): Promise<TransferResult>;
}
```

### Royalty Provider Interface

Interface for royalty and revenue distribution:

```typescript
// @blackhole/shared/types/ledger/providers/royalty-provider.ts

export interface RoyaltyParams {
  tokenId: string;
  recipients: RoyaltyRecipient[];
  basisPoints: number; // Total royalty in basis points (100 = 1%)
}

export interface RoyaltyRecipient {
  address: string;
  share: number; // Share in basis points (total must equal basisPoints)
}

export interface RoyaltyProvider {
  /**
   * Set royalty configuration for a token
   */
  setRoyalties(params: RoyaltyParams): Promise<RoyaltyResult>;
  
  /**
   * Get royalty configuration for a token
   */
  getRoyalties(tokenId: string): Promise<RoyaltyInfo>;
  
  /**
   * Calculate royalty amount for a sale
   */
  calculateRoyalty(tokenId: string, saleAmount: BigNumber): Promise<RoyaltyCalculation>;
  
  /**
   * Pay royalties for a sale
   */
  payRoyalties(tokenId: string, saleAmount: BigNumber): Promise<PaymentResult>;
  
  /**
   * Get royalty payment history for a token
   */
  getRoyaltyHistory(tokenId: string): Promise<RoyaltyPayment[]>;
  
  /**
   * Get earnings for a royalty recipient
   */
  getEarnings(address: string): Promise<EarningsInfo>;
}
```

### Marketplace Provider Interface

Interface for marketplace functionality:

```typescript
// @blackhole/shared/types/ledger/providers/marketplace-provider.ts

export interface ListingParams {
  tokenId: string;
  seller: string;
  price: BigNumber;
  quantity: BigNumber;
  expirationTime?: number; // Unix timestamp
  paymentToken?: string; // Address of ERC20 token, null for native currency
}

export interface AuctionParams extends ListingParams {
  reservePrice: BigNumber;
  minBidIncrement: BigNumber;
  duration: number; // In seconds
}

export interface MarketplaceProvider {
  /**
   * Create a new fixed-price listing
   */
  createListing(params: ListingParams): Promise<ListingResult>;
  
  /**
   * Cancel an existing listing
   */
  cancelListing(listingId: string): Promise<CancelResult>;
  
  /**
   * Buy from a fixed-price listing
   */
  buyItem(listingId: string, buyer: string, quantity: BigNumber): Promise<PurchaseResult>;
  
  /**
   * Create a new auction
   */
  createAuction(params: AuctionParams): Promise<AuctionResult>;
  
  /**
   * Place a bid on an auction
   */
  placeBid(auctionId: string, bidder: string, amount: BigNumber): Promise<BidResult>;
  
  /**
   * Finalize an auction after it ends
   */
  finalizeAuction(auctionId: string): Promise<FinalizationResult>;
  
  /**
   * Get listing details
   */
  getListing(listingId: string): Promise<Listing>;
  
  /**
   * Get auction details
   */
  getAuction(auctionId: string): Promise<Auction>;
  
  /**
   * Get all listings for a token
   */
  getListingsForToken(tokenId: string): Promise<Listing[]>;
  
  /**
   * Get all listings by a seller
   */
  getListingsBySeller(seller: string): Promise<Listing[]>;
  
  /**
   * Get all active auctions
   */
  getActiveAuctions(): Promise<Auction[]>;
  
  /**
   * Create a new subscription plan
   */
  createSubscriptionPlan(params: SubscriptionPlanParams): Promise<SubscriptionPlanResult>;
  
  /**
   * Subscribe to a plan
   */
  subscribe(planId: string, subscriber: string): Promise<SubscriptionResult>;
  
  /**
   * Cancel a subscription
   */
  cancelSubscription(subscriptionId: string): Promise<CancellationResult>;
}
```

## Core Data Types

### Token Types

```typescript
// @blackhole/shared/types/ledger/token/token.ts

export enum TokenType {
  FUNGIBLE = 'fungible',
  NON_FUNGIBLE = 'non-fungible',
  SEMI_FUNGIBLE = 'semi-fungible'
}

export enum TokenStatus {
  ACTIVE = 'active',
  PAUSED = 'paused',
  REVOKED = 'revoked',
  EXPIRED = 'expired'
}

export interface TokenMetadata {
  name: string;
  description?: string;
  image?: string;
  contentCid?: string;
  contentType?: string;
  properties?: Record<string, any>;
  externalUrl?: string;
  createdAt: number; // Unix timestamp
}

export interface Token {
  id: string;
  type: TokenType;
  contractAddress: string;
  tokenId: string;
  creator: string;
  owner: string;
  metadata: TokenMetadata;
  status: TokenStatus;
  totalSupply: BigNumber;
  createdAt: number; // Unix timestamp
  updatedAt: number; // Unix timestamp
  providerSpecific?: Record<string, any>;
}

export interface TokenCreationResult {
  token: Token;
  transactionId: string;
}

export interface TokenTransferParams {
  tokenId: string;
  from: string;
  to: string;
  amount: BigNumber;
}

export interface TransferResult {
  success: boolean;
  transactionId: string;
  from: string;
  to: string;
  amount: BigNumber;
}
```

### Semi-Fungible Token Types

```typescript
// @blackhole/shared/types/ledger/token/sft.ts

export interface SFTClass {
  id: string;
  name: string;
  symbol: string;
  creator: string;
  maxSupply?: BigNumber;
  defaultProperties: Record<string, any>;
  createdAt: number; // Unix timestamp
}

export interface SFTCollectionParams {
  name: string;
  symbol: string;
  creator: string;
  defaultProperties?: Record<string, any>;
  maxSupply?: BigNumber;
}

export interface SFTCollectionResult {
  collection: SFTClass;
  transactionId: string;
}

export interface SFT extends Token {
  classId: string;
  class: SFTClass;
  properties: Record<string, any>;
  edition: number;
  maxSupply?: BigNumber;
}

export interface SFTCreationResult extends TokenCreationResult {
  token: SFT;
}

export interface MintResult {
  success: boolean;
  transactionId: string;
  amount: BigNumber;
  recipient: string;
}

export interface BurnResult {
  success: boolean;
  transactionId: string;
  amount: BigNumber;
}
```

### License and Rights Types

```typescript
// @blackhole/shared/types/ledger/rights/license.ts

export enum LicenseType {
  VIEW = 'view',
  DISTRIBUTION = 'distribution',
  COMMERCIAL = 'commercial',
  DERIVATIVE = 'derivative',
  TIME_LIMITED = 'time-limited',
  GEOGRAPHIC = 'geographic'
}

export interface LicenseTerms {
  commercialUse: boolean;
  derivativeWorks: boolean;
  attribution: boolean;
  sharing: boolean;
  timeLimit?: number; // In seconds, 0 for perpetual
  geographicLimit?: string[]; // ISO country codes
  transferable: boolean;
  revocable: boolean;
  additionalTerms?: Record<string, any>;
}

export interface License {
  id: string;
  tokenId: string;
  licenseType: LicenseType;
  terms: LicenseTerms;
  licensor: string;
  licensee: string;
  issuedAt: number; // Unix timestamp
  expiresAt?: number; // Unix timestamp, undefined for perpetual
  status: 'active' | 'expired' | 'revoked';
  transactionId: string;
}

export interface LicenseResult {
  license: License;
  transactionId: string;
}

export interface LicenseVerificationResult {
  valid: boolean;
  license: License;
  validations: {
    active: boolean;
    notExpired: boolean;
    notRevoked: boolean;
    territoryValid: boolean;
  };
  errors?: string[];
}

export interface RevocationResult {
  success: boolean;
  transactionId: string;
}
```

### Royalty Types

```typescript
// @blackhole/shared/types/ledger/royalties/distribution.ts

export interface RoyaltyInfo {
  tokenId: string;
  basisPoints: number; // Total royalty in basis points (100 = 1%)
  recipients: RoyaltyRecipient[];
}

export interface RoyaltyCalculation {
  tokenId: string;
  saleAmount: BigNumber;
  totalRoyalty: BigNumber;
  distributions: {
    recipient: string;
    amount: BigNumber;
  }[];
}

export interface RoyaltyPayment {
  id: string;
  tokenId: string;
  saleId: string;
  saleAmount: BigNumber;
  totalRoyalty: BigNumber;
  distributions: {
    recipient: string;
    amount: BigNumber;
    transactionId: string;
  }[];
  timestamp: number; // Unix timestamp
}

export interface PaymentResult {
  success: boolean;
  payment: RoyaltyPayment;
}

export interface EarningsInfo {
  address: string;
  totalEarnings: BigNumber;
  pendingEarnings: BigNumber;
  paymentHistory: {
    tokenId: string;
    amount: BigNumber;
    timestamp: number;
    transactionId: string;
  }[];
}
```

### Marketplace Types

```typescript
// @blackhole/shared/types/ledger/marketplace/listing.ts

export enum ListingStatus {
  ACTIVE = 'active',
  SOLD = 'sold',
  CANCELLED = 'cancelled',
  EXPIRED = 'expired'
}

export interface Listing {
  id: string;
  tokenId: string;
  seller: string;
  price: BigNumber;
  originalQuantity: BigNumber;
  remainingQuantity: BigNumber;
  paymentToken?: string; // Address of ERC20 token, null for native currency
  createdAt: number; // Unix timestamp
  expiresAt?: number; // Unix timestamp
  status: ListingStatus;
}

export interface ListingResult {
  listing: Listing;
  transactionId: string;
}

export interface CancelResult {
  success: boolean;
  transactionId: string;
}

export interface PurchaseResult {
  success: boolean;
  transactionId: string;
  listing: Listing;
  buyer: string;
  quantity: BigNumber;
  totalPaid: BigNumber;
}

export interface Auction extends Listing {
  reservePrice: BigNumber;
  minBidIncrement: BigNumber;
  endTime: number; // Unix timestamp
  highestBid?: BigNumber;
  highestBidder?: string;
  bids: {
    bidder: string;
    amount: BigNumber;
    timestamp: number;
  }[];
}

export interface AuctionResult {
  auction: Auction;
  transactionId: string;
}

export interface BidResult {
  success: boolean;
  transactionId: string;
  auction: Auction;
  bid: {
    bidder: string;
    amount: BigNumber;
    timestamp: number;
  };
}

export interface FinalizationResult {
  success: boolean;
  transactionId: string;
  auction: Auction;
  winner: string;
  finalPrice: BigNumber;
}
```

## Root Network Provider Implementation

The Root Network provider implements these interfaces with SFT-specific functionality:

```typescript
// @blackhole/shared/types/ledger/providers/root-network/sft.ts

export interface RootNetworkSFTConfig {
  contractAddress: string;
  collectionFactoryAddress: string;
  marketplaceAddress: string;
  rightsManagerAddress: string;
}

export interface RootNetworkSFT extends SFT {
  // Root Network specific fields
  rootTokenId: string;
  collectionId: string;
  transferable: boolean;
  supportsRoyalties: boolean;
}

export interface RootNetworkCollection extends SFTClass {
  // Root Network specific fields
  collectionAddress: string;
  collectionId: string;
  baseURI: string;
  creator: string;
}
```

## Client Interfaces

High-level client interfaces for the SDK layer:

```typescript
// @blackhole/shared/types/ledger/client.ts

export interface LedgerClient {
  /**
   * Get the current provider
   */
  getProvider(): BlockchainProvider;
  
  /**
   * Switch to a different provider
   */
  switchProvider(providerName: string): Promise<void>;
  
  /**
   * Get wallet interface
   */
  getWallet(): WalletClient;
  
  /**
   * Get token interface
   */
  getTokenClient(): TokenClient;
  
  /**
   * Get marketplace interface
   */
  getMarketplaceClient(): MarketplaceClient;
  
  /**
   * Get rights management interface
   */
  getRightsClient(): RightsClient;
  
  /**
   * Get royalty management interface
   */
  getRoyaltyClient(): RoyaltyClient;
}

export interface TokenClient {
  /**
   * Create a new token
   */
  createToken(params: ClientTokenParams): Promise<Token>;
  
  /**
   * Get token details
   */
  getToken(tokenId: string): Promise<Token>;
  
  /**
   * Create a new SFT class
   */
  createSFTClass(params: ClientSFTClassParams): Promise<SFTClass>;
  
  /**
   * Create a new SFT in a class
   */
  createSFTInstance(params: ClientSFTParams): Promise<SFT>;
  
  /**
   * Transfer token
   */
  transferToken(params: ClientTransferParams): Promise<TransferResult>;
}

export interface RightsClient {
  /**
   * Create a new license
   */
  createLicense(params: ClientLicenseParams): Promise<License>;
  
  /**
   * Verify license
   */
  verifyLicense(licenseId: string): Promise<LicenseVerificationResult>;
  
  /**
   * Get licenses for a token
   */
  getLicensesForToken(tokenId: string): Promise<License[]>;
  
  /**
   * Get user's licenses
   */
  getMyLicenses(): Promise<License[]>;
}

export interface RoyaltyClient {
  /**
   * Set royalties for a token
   */
  setRoyalties(params: ClientRoyaltyParams): Promise<RoyaltyInfo>;
  
  /**
   * Get earnings
   */
  getMyEarnings(): Promise<EarningsInfo>;
  
  /**
   * Get royalty history for a token
   */
  getRoyaltyHistory(tokenId: string): Promise<RoyaltyPayment[]>;
}

export interface MarketplaceClient {
  /**
   * List token for sale
   */
  listForSale(params: ClientListingParams): Promise<Listing>;
  
  /**
   * Buy token
   */
  buy(listingId: string, quantity: number): Promise<PurchaseResult>;
  
  /**
   * Create auction
   */
  createAuction(params: ClientAuctionParams): Promise<Auction>;
  
  /**
   * Place bid
   */
  placeBid(auctionId: string, amount: number): Promise<BidResult>;
  
  /**
   * Search marketplace
   */
  searchListings(params: SearchParams): Promise<Listing[]>;
}

export interface WalletClient {
  /**
   * Get current account
   */
  getAccount(): Promise<string>;
  
  /**
   * Get balance
   */
  getBalance(): Promise<BigNumber>;
  
  /**
   * Get tokens owned by current account
   */
  getMyTokens(): Promise<Token[]>;
  
  /**
   * Sign message
   */
  signMessage(message: string): Promise<string>;
  
  /**
   * Sign transaction
   */
  signTransaction(tx: Transaction): Promise<SignedTransaction>;
}
```

## Event Types

```typescript
// @blackhole/shared/types/ledger/events/token.ts

export enum TokenEventType {
  CREATED = 'token.created',
  TRANSFERRED = 'token.transferred',
  METADATA_UPDATED = 'token.metadata.updated',
  BURNED = 'token.burned'
}

export interface TokenEvent {
  type: TokenEventType;
  tokenId: string;
  transactionId: string;
  timestamp: number;
  data: any; // Event-specific data
}

export interface TokenCreatedEvent extends TokenEvent {
  type: TokenEventType.CREATED;
  data: {
    creator: string;
    initialOwner: string;
    initialSupply: BigNumber;
    metadata: TokenMetadata;
  };
}

export interface TokenTransferredEvent extends TokenEvent {
  type: TokenEventType.TRANSFERRED;
  data: {
    from: string;
    to: string;
    amount: BigNumber;
  };
}
```

## Implementation Examples

### Creating a Token with Root Network Provider

```typescript
// Example implementation in @blackhole/node/services/ledger/providers/root-network/client.ts

export class RootNetworkProvider implements SFTProvider {
  // ... other methods
  
  async createSFT(params: SFTCreationParams): Promise<SFTCreationResult> {
    try {
      // 1. Validate parameters
      this.validateSFTParams(params);
      
      // 2. Connect to Root Network
      const collection = await this.getCollection(params.classId);
      if (!collection) {
        throw new Error(`Collection ${params.classId} not found`);
      }
      
      // 3. Prepare transaction
      const contract = new this.web3.eth.Contract(
        this.abis.sftCollection,
        collection.collectionAddress
      );
      
      const data = contract.methods.mint(
        params.creator,
        params.initialSupply || 1,
        JSON.stringify(params.properties),
        params.contentCid || ''
      ).encodeABI();
      
      const tx = await this.createTransaction({
        from: params.creator,
        to: collection.collectionAddress,
        data
      });
      
      // 4. Sign and submit transaction
      const signedTx = await this.signTransaction(tx, this.getSigner(params.creator));
      const result = await this.submitTransaction(signedTx);
      
      // 5. Extract token ID from transaction receipt
      const receipt = await this.getTransactionReceipt(result.transactionId);
      const mintEvent = receipt.logs
        .find(log => log.address.toLowerCase() === collection.collectionAddress.toLowerCase())
        .topics;
      
      const rootTokenId = this.web3.utils.hexToNumber(mintEvent[1]);
      
      // 6. Construct SFT object
      const sft: RootNetworkSFT = {
        id: `${collection.collectionId}:${rootTokenId}`,
        type: TokenType.SEMI_FUNGIBLE,
        contractAddress: collection.collectionAddress,
        tokenId: rootTokenId.toString(),
        rootTokenId: rootTokenId.toString(),
        collectionId: collection.collectionId,
        creator: params.creator,
        owner: params.creator,
        metadata: {
          name: params.name,
          description: params.metadata?.description || '',
          image: params.metadata?.image || '',
          contentCid: params.contentCid || '',
          contentType: params.metadata?.contentType || '',
          properties: params.properties,
          createdAt: Math.floor(Date.now() / 1000)
        },
        status: TokenStatus.ACTIVE,
        totalSupply: params.initialSupply || new BigNumber(1),
        createdAt: Math.floor(Date.now() / 1000),
        updatedAt: Math.floor(Date.now() / 1000),
        classId: params.classId,
        class: collection,
        properties: params.properties,
        edition: 1,
        transferable: true,
        supportsRoyalties: true
      };
      
      // 7. Return creation result
      return {
        token: sft,
        transactionId: result.transactionId
      };
    } catch (error) {
      console.error('Error creating SFT:', error);
      throw new Error(`Failed to create SFT: ${error.message}`);
    }
  }
}
```

### Client SDK Usage Example

```typescript
// Example usage in a service provider application

// Initialize the ledger client with Root Network provider
const ledgerClient = new LedgerClient({
  defaultProvider: 'root-network',
  providers: {
    'root-network': {
      rpcUrl: 'https://rpc.rootnet.io',
      networkId: 1,
      apiKey: process.env.ROOT_NETWORK_API_KEY
    }
  }
});

// Get token client
const tokenClient = ledgerClient.getTokenClient();

// Create an SFT collection
const collection = await tokenClient.createSFTClass({
  name: 'My Content Collection',
  symbol: 'MCC',
  defaultProperties: {
    contentType: 'video',
    license: 'standard'
  }
});

// Create an SFT for a video
const videoToken = await tokenClient.createSFTInstance({
  classId: collection.id,
  name: 'My Awesome Video',
  contentCid: 'QmAbC123...',
  properties: {
    duration: 120,
    resolution: '1080p',
    format: 'mp4'
  }
});

// Set royalties for the token
const royaltyClient = ledgerClient.getRoyaltyClient();
await royaltyClient.setRoyalties({
  tokenId: videoToken.id,
  basisPoints: 1000, // 10%
  recipients: [
    { address: 'creator-address', share: 800 }, // 8%
    { address: 'collaborator-address', share: 200 } // 2%
  ]
});

// List the token on the marketplace
const marketplaceClient = ledgerClient.getMarketplaceClient();
const listing = await marketplaceClient.listForSale({
  tokenId: videoToken.id,
  price: 50, // In platform currency
  quantity: 1
});

console.log(`Token created and listed: ${listing.id}`);
```

## Type Extension Pattern

The architecture allows for provider-specific type extensions:

```typescript
// Base token interface in @blackhole/shared/types/ledger/token/token.ts
export interface Token {
  id: string;
  // ... common fields
  providerSpecific?: Record<string, any>;
}

// Root Network extension in @blackhole/shared/types/ledger/providers/root-network/token.ts
export interface RootNetworkToken extends Token {
  rootTokenId: string;
  // ... Root Network specific fields
}

// Type guard for checking provider-specific token
export function isRootNetworkToken(token: Token): token is RootNetworkToken {
  return (
    token.providerSpecific !== undefined &&
    'rootTokenId' in token.providerSpecific
  );
}

// Usage example
function processToken(token: Token) {
  // Common token processing
  console.log(`Processing token ${token.id}`);
  
  // Provider-specific handling
  if (isRootNetworkToken(token)) {
    console.log(`Root Network token ID: ${token.rootTokenId}`);
    // Root Network specific operations
  }
}
```