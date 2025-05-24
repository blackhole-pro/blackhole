# Blackhole Plugin Marketplace

The official plugin marketplace for the Blackhole Framework, hosted on GitHub.

## Quick Start

### For Users

```bash
# Search for plugins
blackhole plugin search analytics

# Install a plugin
blackhole plugin install node

# Install specific version
blackhole plugin install node@1.0.0

# Update plugins
blackhole plugin update
```

### For Plugin Developers

```bash
# Create your plugin from template
gh repo create my-awesome-plugin --template blackhole-foundation/plugin-template

# Build and test locally
make build
make test

# Create release (auto-publishes to marketplace)
git tag v1.0.0
git push origin v1.0.0
```

## How It Works

The Blackhole plugin marketplace leverages GitHub's infrastructure:

1. **Plugin Hosting**: Each plugin is a GitHub repository
2. **Distribution**: Plugin binaries are GitHub Release assets  
3. **Marketplace Catalog**: JSON catalog hosted on GitHub Pages
4. **Discovery**: Web interface at marketplace.blackhole.io
5. **Installation**: CLI fetches from GitHub releases

## Repository Structure

```
blackhole-foundation/
├── plugin-marketplace/     # Marketplace catalog and website
├── plugin-template/        # Template for new plugins
├── node-plugin/           # Official node plugin
├── storage-plugin/        # Official storage plugin
└── identity-plugin/       # Official identity plugin
```

## Benefits

- **Free Hosting**: No infrastructure costs
- **Reliable**: GitHub's CDN and uptime
- **Familiar**: Standard GitHub workflow
- **Automated**: CI/CD handles publishing
- **Secure**: GPG signing and checksums

## Getting Started

See [GITHUB_MARKETPLACE.md](GITHUB_MARKETPLACE.md) for detailed implementation guide.

## Plugin Categories

- **Networking**: P2P, mesh, protocols
- **Storage**: Distributed storage, databases
- **Security**: Authentication, encryption
- **Analytics**: Metrics, monitoring, logs
- **Development**: Tools, debuggers, profilers
- **Integration**: Cloud providers, services

## Community

- Submit plugins via pull request
- Star your favorite plugins
- Report issues on plugin repos
- Join discussions in marketplace repo

## Links

- [Marketplace Website](https://marketplace.blackhole.io)
- [Plugin Development Guide](https://docs.blackhole.io/plugins)
- [Marketplace API](https://marketplace.blackhole.io/api/v1/)
- [Submit a Plugin](https://github.com/blackhole-foundation/plugin-marketplace)