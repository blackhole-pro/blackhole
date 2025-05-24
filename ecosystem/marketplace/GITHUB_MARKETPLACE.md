# Blackhole Plugin Marketplace on GitHub

## Overview

This document outlines how to implement the Blackhole plugin marketplace using GitHub's free infrastructure, providing professional plugin distribution without additional hosting costs.

## Architecture

### 1. Repository Structure

We'll use a dedicated repository (`blackhole-foundation/plugin-marketplace`) with this structure:

```
blackhole-plugin-marketplace/
├── .github/
│   └── workflows/
│       ├── validate-plugin.yml      # Validate plugin submissions
│       ├── publish-release.yml      # Auto-publish plugin releases
│       └── update-marketplace.yml   # Update marketplace catalog
├── plugins/                         # Plugin source/metadata
│   ├── official/                    # Official plugins
│   │   ├── node/
│   │   │   ├── manifest.json
│   │   │   ├── README.md
│   │   │   └── releases.json
│   │   ├── storage/
│   │   └── identity/
│   └── community/                   # Community plugins
│       └── awesome-plugin/
├── marketplace/                     # GitHub Pages site
│   ├── index.html                  # Marketplace website
│   ├── catalog.json                # Complete plugin catalog
│   ├── api/                        # JSON API endpoints
│   │   └── v1/
│   │       ├── plugins.json
│   │       └── search.json
│   └── assets/                     # Web assets
└── tools/                          # Marketplace tools
    ├── validate.sh
    ├── publish.sh
    └── update-catalog.sh
```

### 2. Plugin Distribution Methods

#### A. GitHub Releases (Recommended)
Each plugin has its own repository with binary releases:

```yaml
# Plugin repository: blackhole-foundation/node-plugin
# Release URL: https://github.com/blackhole-foundation/node-plugin/releases/download/v1.0.0/node-v1.0.0.plugin
```

**Advantages:**
- 2GB file size limit per release asset
- Automatic CDN distribution
- Download statistics
- Release notes and changelogs

#### B. GitHub Packages (Alternative)
For container-based plugins:

```dockerfile
# Publish as OCI artifact
FROM scratch
COPY plugin-binary /
COPY plugin.yaml /
LABEL org.opencontainers.image.source=https://github.com/blackhole-foundation/node-plugin
```

### 3. Marketplace Catalog Format

```json
{
  "version": "1.0",
  "updated": "2024-01-25T00:00:00Z",
  "plugins": [
    {
      "id": "node",
      "name": "Node Plugin",
      "description": "P2P networking and distributed communication",
      "category": "networking",
      "official": true,
      "versions": [
        {
          "version": "1.0.0",
          "date": "2024-01-25",
          "compatibility": {
            "blackhole": ">=1.0.0",
            "mesh": ">=1.0.0"
          },
          "downloads": {
            "darwin-amd64": {
              "url": "https://github.com/blackhole-foundation/node-plugin/releases/download/v1.0.0/node-darwin-amd64-v1.0.0.plugin",
              "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
              "size": 15728640
            },
            "darwin-arm64": {
              "url": "https://github.com/blackhole-foundation/node-plugin/releases/download/v1.0.0/node-darwin-arm64-v1.0.0.plugin",
              "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
              "size": 15728640
            },
            "linux-amd64": {
              "url": "https://github.com/blackhole-foundation/node-plugin/releases/download/v1.0.0/node-linux-amd64-v1.0.0.plugin",
              "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
              "size": 15728640
            }
          }
        }
      ],
      "repository": "https://github.com/blackhole-foundation/node-plugin",
      "homepage": "https://docs.blackhole.io/plugins/node",
      "license": "MIT",
      "author": {
        "name": "Blackhole Foundation",
        "url": "https://github.com/blackhole-foundation"
      },
      "tags": ["p2p", "networking", "distributed"],
      "stats": {
        "downloads": 10234,
        "stars": 156,
        "lastUpdated": "2024-01-25T00:00:00Z"
      }
    }
  ]
}
```

### 4. Marketplace Website (GitHub Pages)

Host at: `https://marketplace.blackhole.io` (or `https://blackhole-foundation.github.io/plugin-marketplace`)

**Features:**
- Plugin search and filtering
- Installation instructions
- Version history
- Download statistics
- Security verification status

### 5. CLI Integration

```bash
# List available plugins
blackhole plugin search

# Get plugin info
blackhole plugin info node

# Install specific version
blackhole plugin install node@1.0.0

# Install from custom marketplace
blackhole plugin install node --marketplace=https://custom-marketplace.com
```

**Implementation in CLI:**
```go
// Fetch catalog
catalog := fetchCatalog("https://marketplace.blackhole.io/api/v1/catalog.json")

// Find plugin
plugin := catalog.FindPlugin("node", "1.0.0")

// Download with checksum verification
downloadURL := plugin.GetDownloadURL(runtime.GOOS, runtime.GOARCH)
if err := downloadAndVerify(downloadURL, plugin.Checksum); err != nil {
    return err
}
```

### 6. Plugin Submission Process

#### For Official Plugins
1. Create plugin in `blackhole-foundation/plugin-name` repository
2. Use standardized CI/CD pipeline
3. Automatic marketplace updates on release

#### For Community Plugins
1. Fork marketplace repository
2. Add plugin metadata to `plugins/community/`
3. Submit pull request
4. Automated validation checks
5. Manual security review
6. Merge and auto-publish

### 7. GitHub Actions Workflows

#### A. Plugin Validation
```yaml
name: Validate Plugin
on:
  pull_request:
    paths:
      - 'plugins/**'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Validate Manifest
        run: |
          for manifest in $(find plugins -name manifest.json); do
            ./tools/validate.sh "$manifest"
          done
      
      - name: Security Scan
        run: |
          # Check for security issues
          ./tools/security-scan.sh
      
      - name: License Check
        run: |
          # Verify license compatibility
          ./tools/license-check.sh
```

#### B. Marketplace Update
```yaml
name: Update Marketplace
on:
  push:
    branches: [main]
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Fetch Plugin Stats
        run: |
          # Get download counts, stars, etc.
          ./tools/fetch-stats.sh
      
      - name: Build Catalog
        run: |
          ./tools/update-catalog.sh > marketplace/api/v1/catalog.json
      
      - name: Generate Search Index
        run: |
          ./tools/build-search-index.sh
      
      - name: Deploy to GitHub Pages
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./marketplace
```

### 8. Security Features

#### Plugin Signing
```bash
# Sign plugin with GPG
gpg --detach-sign --armor node-v1.0.0.plugin

# Publish signature alongside plugin
# node-v1.0.0.plugin
# node-v1.0.0.plugin.asc
```

#### Verification in CLI
```go
// Verify plugin signature
if err := verifySignature(pluginPath, signaturePath, trustedKeys); err != nil {
    return fmt.Errorf("plugin signature verification failed: %w", err)
}
```

### 9. Marketplace API Endpoints

Served via GitHub Pages:

- `GET /api/v1/catalog.json` - Complete plugin catalog
- `GET /api/v1/plugins/{id}.json` - Plugin details
- `GET /api/v1/search.json` - Search index
- `GET /api/v1/stats.json` - Download statistics

### 10. Cost Analysis

**GitHub Free Tier:**
- Unlimited public repositories
- 500MB GitHub Pages site
- 2GB release assets
- 2,000 GitHub Actions minutes/month

**Estimated Costs:** $0 for most use cases

**When to Consider Alternatives:**
- >100GB/month bandwidth (use CDN)
- >2GB plugin sizes (use external storage)
- Private plugins (GitHub Enterprise)

## Implementation Roadmap

### Phase 1: Basic Marketplace (Week 1)
- [ ] Create marketplace repository
- [ ] Set up GitHub Pages
- [ ] Create basic catalog format
- [ ] Implement CLI catalog fetching

### Phase 2: Plugin Publishing (Week 2)
- [ ] Create plugin template repository
- [ ] Set up automated releases
- [ ] Implement checksum generation
- [ ] Add CLI installation from marketplace

### Phase 3: Web Interface (Week 3)
- [ ] Design marketplace website
- [ ] Implement search/filter
- [ ] Add installation guides
- [ ] Create plugin badges

### Phase 4: Security & Analytics (Week 4)
- [ ] Implement plugin signing
- [ ] Add security scanning
- [ ] Create analytics dashboard
- [ ] Set up monitoring

## Example Plugin Publishing

```bash
# 1. Create plugin repository
gh repo create blackhole-foundation/my-plugin --public

# 2. Add plugin code and manifest
cd my-plugin
cp /path/to/plugin/* .

# 3. Create and push tag
git tag v1.0.0
git push origin v1.0.0

# 4. GitHub Action automatically:
#    - Builds plugin for all platforms
#    - Creates GitHub release
#    - Uploads plugin binaries
#    - Updates marketplace catalog

# 5. Users can now install:
blackhole plugin install my-plugin
```

## Conclusion

GitHub provides an excellent, cost-effective platform for hosting the Blackhole plugin marketplace. This approach offers:

- **Zero hosting costs** for most scenarios
- **Professional infrastructure** with CDN, APIs, and analytics
- **Community-friendly** with familiar PR workflow
- **Scalable** to thousands of plugins
- **Secure** with signing and verification

The implementation leverages GitHub's strengths while providing a seamless experience for both plugin developers and users.