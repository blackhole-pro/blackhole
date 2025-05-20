# Zero-Knowledge Circuit Specifications

This document specifies the zero-knowledge circuits used throughout the Blackhole platform for privacy-preserving authentication and verification.

## Overview

The Blackhole platform utilizes zero-knowledge proofs to enable users to prove statements about their identity and credentials without revealing the underlying data. This specification defines the standard circuits, their parameters, and implementation details.

## Circuit Architecture

### Base Circuit Framework

```go
type Circuit interface {
    // Define circuit constraints
    Define(cs constraint.System) error
    
    // Generate witness for proof
    GenerateWitness(inputs PrivateInputs) (Witness, error)
    
    // Verify proof
    Verify(proof Proof, publicInputs PublicInputs) (bool, error)
}

type CircuitConfig struct {
    Name        string
    Version     string
    CurveID     ecc.ID
    InputWires  int
    OutputWires int
    Constraints int
    GateTypes   []GateType
}
```

## Standard Circuits

### 1. Age Verification Circuit

Proves that a user's age is above a certain threshold without revealing the actual birthdate.

```go
type AgeVerificationCircuit struct {
    // Private inputs
    BirthYear  frontend.Variable `gnark:",secret"`
    BirthMonth frontend.Variable `gnark:",secret"`
    BirthDay   frontend.Variable `gnark:",secret"`
    
    // Public inputs
    CurrentYear  frontend.Variable `gnark:",public"`
    CurrentMonth frontend.Variable `gnark:",public"`
    CurrentDay   frontend.Variable `gnark:",public"`
    MinimumAge   frontend.Variable `gnark:",public"`
    
    // Output
    IsOldEnough frontend.Variable `gnark:",public"`
}

func (c *AgeVerificationCircuit) Define(cs constraint.System) error {
    // Calculate age in days
    birthDateInDays := cs.Mul(c.BirthYear, 365)
    birthDateInDays = cs.Add(birthDateInDays, cs.Mul(c.BirthMonth, 30))
    birthDateInDays = cs.Add(birthDateInDays, c.BirthDay)
    
    currentDateInDays := cs.Mul(c.CurrentYear, 365)
    currentDateInDays = cs.Add(currentDateInDays, cs.Mul(c.CurrentMonth, 30))
    currentDateInDays = cs.Add(currentDateInDays, c.CurrentDay)
    
    // Calculate age difference
    ageInDays := cs.Sub(currentDateInDays, birthDateInDays)
    minimumAgeInDays := cs.Mul(c.MinimumAge, 365)
    
    // Compare age
    c.IsOldEnough = cs.IsGreaterOrEqual(ageInDays, minimumAgeInDays)
    
    return nil
}
```

**Parameters:**
- Curve: BLS12-381
- Constraints: ~1,000
- Proof size: 192 bytes
- Verification time: ~5ms

### 2. Credential Ownership Circuit

Proves ownership of a credential without revealing the credential details.

```go
type CredentialOwnershipCircuit struct {
    // Private inputs
    CredentialHash  frontend.Variable `gnark:",secret"`
    IssuerSignature frontend.Variable `gnark:",secret"`
    HolderPrivKey   frontend.Variable `gnark:",secret"`
    
    // Public inputs
    IssuerPubKey    frontend.Variable `gnark:",public"`
    HolderPubKey    frontend.Variable `gnark:",public"`
    CredentialType  frontend.Variable `gnark:",public"`
    
    // Output
    ValidOwnership  frontend.Variable `gnark:",public"`
}

func (c *CredentialOwnershipCircuit) Define(cs constraint.System) error {
    // Verify issuer signature
    issuerValid := eddsa.Verify(cs, c.IssuerSignature, c.CredentialHash, c.IssuerPubKey)
    
    // Verify holder owns the private key
    derivedPubKey := eddsa.PublicKeyFromPrivate(cs, c.HolderPrivKey)
    holderValid := cs.IsEqual(derivedPubKey, c.HolderPubKey)
    
    // Both must be valid
    c.ValidOwnership = cs.And(issuerValid, holderValid)
    
    return nil
}
```

**Parameters:**
- Curve: BLS12-381
- Constraints: ~2,000
- Proof size: 192 bytes
- Verification time: ~8ms

### 3. KYC Verification Circuit

Proves KYC compliance without revealing personal information.

```go
type KYCVerificationCircuit struct {
    // Private inputs
    NameHash        frontend.Variable `gnark:",secret"`
    AddressHash     frontend.Variable `gnark:",secret"`
    IDNumberHash    frontend.Variable `gnark:",secret"`
    KYCProviderSig  frontend.Variable `gnark:",secret"`
    
    // Public inputs
    KYCProviderKey  frontend.Variable `gnark:",public"`
    RequiredLevel   frontend.Variable `gnark:",public"`
    ExpirationTime  frontend.Variable `gnark:",public"`
    CurrentTime     frontend.Variable `gnark:",public"`
    
    // Output
    IsCompliant     frontend.Variable `gnark:",public"`
}

func (c *KYCVerificationCircuit) Define(cs constraint.System) error {
    // Combine personal data hashes
    personalDataHash := poseidon.Hash(cs, c.NameHash, c.AddressHash, c.IDNumberHash)
    
    // Verify KYC provider signature
    signatureValid := eddsa.Verify(cs, c.KYCProviderSig, personalDataHash, c.KYCProviderKey)
    
    // Check if not expired
    notExpired := cs.IsLess(c.CurrentTime, c.ExpirationTime)
    
    // Both conditions must be true
    c.IsCompliant = cs.And(signatureValid, notExpired)
    
    return nil
}
```

**Parameters:**
- Curve: BN254
- Constraints: ~3,000
- Proof size: 256 bytes
- Verification time: ~10ms

### 4. Balance Proof Circuit

Proves account balance meets a threshold without revealing the exact amount.

```go
type BalanceProofCircuit struct {
    // Private inputs
    Balance         frontend.Variable `gnark:",secret"`
    AccountNonce    frontend.Variable `gnark:",secret"`
    
    // Public inputs
    MinimumBalance  frontend.Variable `gnark:",public"`
    AccountHash     frontend.Variable `gnark:",public"`
    Timestamp       frontend.Variable `gnark:",public"`
    
    // Output
    SufficientFunds frontend.Variable `gnark:",public"`
}

func (c *BalanceProofCircuit) Define(cs constraint.System) error {
    // Verify account hash matches
    computedHash := poseidon.Hash(cs, c.Balance, c.AccountNonce)
    hashValid := cs.IsEqual(computedHash, c.AccountHash)
    
    // Check balance threshold
    balanceSufficient := cs.IsGreaterOrEqual(c.Balance, c.MinimumBalance)
    
    // Both must be valid
    c.SufficientFunds = cs.And(hashValid, balanceSufficient)
    
    return nil
}
```

**Parameters:**
- Curve: BLS12-377
- Constraints: ~500
- Proof size: 128 bytes
- Verification time: ~3ms

### 5. Group Membership Circuit

Proves membership in a group without revealing which member.

```go
type GroupMembershipCircuit struct {
    // Private inputs
    MemberID        frontend.Variable `gnark:",secret"`
    MemberSecret    frontend.Variable `gnark:",secret"`
    MerkleProof     []frontend.Variable `gnark:",secret"`
    
    // Public inputs
    GroupRoot       frontend.Variable `gnark:",public"`
    GroupID         frontend.Variable `gnark:",public"`
    
    // Output
    IsMember        frontend.Variable `gnark:",public"`
}

func (c *GroupMembershipCircuit) Define(cs constraint.System) error {
    // Compute member commitment
    memberCommitment := poseidon.Hash(cs, c.MemberID, c.MemberSecret)
    
    // Verify Merkle proof
    root := memberCommitment
    for i := 0; i < len(c.MerkleProof); i++ {
        root = poseidon.Hash(cs, root, c.MerkleProof[i])
    }
    
    // Check if computed root matches group root
    c.IsMember = cs.IsEqual(root, c.GroupRoot)
    
    return nil
}
```

**Parameters:**
- Curve: BLS12-381
- Constraints: ~5,000 (for 32-level tree)
- Proof size: 192 bytes
- Verification time: ~15ms

### 6. DID Ownership Circuit

Proves ownership of a DID without revealing the DID itself.

```go
type DIDOwnershipCircuit struct {
    // Private inputs
    DID             frontend.Variable `gnark:",secret"`
    PrivateKey      frontend.Variable `gnark:",secret"`
    Nonce           frontend.Variable `gnark:",secret"`
    
    // Public inputs
    DIDCommitment   frontend.Variable `gnark:",public"`
    Challenge       frontend.Variable `gnark:",public"`
    PublicKey       frontend.Variable `gnark:",public"`
    
    // Output
    ValidOwnership  frontend.Variable `gnark:",public"`
}

func (c *DIDOwnershipCircuit) Define(cs constraint.System) error {
    // Verify DID commitment
    computedCommitment := poseidon.Hash(cs, c.DID, c.Nonce)
    commitmentValid := cs.IsEqual(computedCommitment, c.DIDCommitment)
    
    // Verify private key corresponds to public key
    derivedPubKey := eddsa.PublicKeyFromPrivate(cs, c.PrivateKey)
    keyValid := cs.IsEqual(derivedPubKey, c.PublicKey)
    
    // Sign challenge to prove ownership
    signature := eddsa.Sign(cs, c.PrivateKey, c.Challenge)
    signatureValid := eddsa.Verify(cs, signature, c.Challenge, c.PublicKey)
    
    // All checks must pass
    c.ValidOwnership = cs.And(cs.And(commitmentValid, keyValid), signatureValid)
    
    return nil
}
```

**Parameters:**
- Curve: BLS12-381
- Constraints: ~2,500
- Proof size: 192 bytes
- Verification time: ~8ms

## Circuit Optimization

### Performance Optimizations

1. **Constraint Reduction**
   - Use efficient hash functions (Poseidon)
   - Minimize multiplication gates
   - Batch operations where possible

2. **Proof Size Optimization**
   - Use recursive proofs for complex circuits
   - Implement proof compression
   - Share common components

3. **Parallelization**
   - Parallel witness generation
   - Multi-threaded proving
   - GPU acceleration for large circuits

### Security Considerations

1. **Trusted Setup**
   - Use universal trusted setup where possible
   - Transparent setup ceremonies
   - Multi-party computation for circuit-specific setup

2. **Side-Channel Protection**
   - Constant-time operations
   - Memory access patterns
   - Timing attack resistance

3. **Quantum Resistance**
   - Post-quantum hash functions
   - Lattice-based alternatives
   - Hybrid schemes

## Integration Guidelines

### 1. Circuit Selection

```go
type CircuitSelector struct {
    circuits map[string]Circuit
    configs  map[string]CircuitConfig
}

func (s *CircuitSelector) SelectCircuit(proofType ZKProofType) (Circuit, error) {
    switch proofType {
    case ProofOfAge:
        return s.circuits["age_verification"], nil
    case ProofOfKYC:
        return s.circuits["kyc_verification"], nil
    case ProofOfBalance:
        return s.circuits["balance_proof"], nil
    default:
        return nil, ErrUnsupportedProofType
    }
}
```

### 2. Proof Generation

```go
func GenerateProof(
    circuit Circuit,
    privateInputs PrivateInputs,
    publicInputs PublicInputs,
) (*Proof, error) {
    // Generate witness
    witness, err := circuit.GenerateWitness(privateInputs)
    if err != nil {
        return nil, err
    }
    
    // Create proof
    proof, err := prover.Prove(circuit, witness, publicInputs)
    if err != nil {
        return nil, err
    }
    
    return proof, nil
}
```

### 3. Proof Verification

```go
func VerifyProof(
    circuit Circuit,
    proof *Proof,
    publicInputs PublicInputs,
) (bool, error) {
    // Verify proof
    valid, err := verifier.Verify(circuit, proof, publicInputs)
    if err != nil {
        return false, err
    }
    
    return valid, nil
}
```

## Testing and Validation

### Unit Tests

```go
func TestAgeVerificationCircuit(t *testing.T) {
    circuit := &AgeVerificationCircuit{}
    
    // Test case: 25 years old, minimum age 18
    privateInputs := PrivateInputs{
        "BirthYear":  1998,
        "BirthMonth": 6,
        "BirthDay":   15,
    }
    
    publicInputs := PublicInputs{
        "CurrentYear":  2023,
        "CurrentMonth": 10,
        "CurrentDay":   1,
        "MinimumAge":   18,
    }
    
    proof, err := GenerateProof(circuit, privateInputs, publicInputs)
    assert.NoError(t, err)
    
    valid, err := VerifyProof(circuit, proof, publicInputs)
    assert.NoError(t, err)
    assert.True(t, valid)
}
```

### Benchmarks

```go
func BenchmarkAgeVerificationProof(b *testing.B) {
    circuit := &AgeVerificationCircuit{}
    privateInputs := generateRandomInputs()
    publicInputs := generatePublicInputs()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = GenerateProof(circuit, privateInputs, publicInputs)
    }
}
```

## Deployment Configuration

```yaml
circuits:
  age_verification:
    version: "1.0.0"
    curve: "BLS12-381"
    max_constraints: 1500
    proving_key_size: "10MB"
    verification_key_size: "1KB"
    
  credential_ownership:
    version: "1.0.0"
    curve: "BLS12-381"
    max_constraints: 3000
    proving_key_size: "20MB"
    verification_key_size: "2KB"
    
  kyc_verification:
    version: "1.0.0"
    curve: "BN254"
    max_constraints: 5000
    proving_key_size: "30MB"
    verification_key_size: "3KB"
```

## Future Enhancements

1. **Recursive Proofs**: Combine multiple proofs into one
2. **Universal Circuits**: Programmable circuits for custom logic
3. **Batch Verification**: Verify multiple proofs efficiently
4. **Cross-Chain Proofs**: Interoperability with other blockchains
5. **Privacy-Preserving ML**: Machine learning on encrypted data