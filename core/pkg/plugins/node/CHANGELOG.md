# Changelog

All notable changes to the Node Plugin will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial plugin architecture implementation
- gRPC service definition
- Mesh network integration
- Plugin manifest specification

## [1.0.0] - 2024-01-15

### Added
- P2P networking with libp2p
- Peer discovery via mDNS and DHT
- Topic-based publish/subscribe messaging
- Direct peer messaging
- NAT traversal support
- Connection management
- Health monitoring
- Graceful shutdown

### Security
- Peer authentication via peer IDs
- Encrypted transport support (TLS, Noise)
- Message signing and verification

### Performance
- Connection pooling
- Message batching
- Efficient routing algorithms
- Resource limits enforcement

## [0.9.0] - 2024-01-01

### Added
- Beta release with core P2P functionality
- Basic peer management
- Simple messaging protocol

### Changed
- Migrated from service to plugin architecture
- Updated to use mesh network for communication

### Fixed
- Memory leaks in peer connection handling
- Race conditions in message delivery

## [0.1.0] - 2023-12-01

### Added
- Initial proof of concept
- Basic P2P connectivity
- Simple peer discovery

[Unreleased]: https://github.com/blackhole/core/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/blackhole/core/compare/v0.9.0...v1.0.0
[0.9.0]: https://github.com/blackhole/core/compare/v0.1.0...v0.9.0
[0.1.0]: https://github.com/blackhole/core/releases/tag/v0.1.0