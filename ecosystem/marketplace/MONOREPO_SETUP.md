# Managing Plugin Marketplace from Blackhole Monorepo

## Overview

Instead of creating separate repositories, we can manage the entire plugin marketplace from within the main Blackhole repository using the `ecosystem/marketplace/` directory.

## Repository Structure

```
blackhole/
├── core/
│   └── pkg/
│       └── plugins/          # Plugin development SDK
│           ├── node/         # Official node plugin source
│           ├── storage/      # Official storage plugin source
│           └── identity/     # Official identity plugin source
├── ecosystem/
│   └── marketplace/
│       ├── .github/
│       │   └── workflows/
│       │       ├── publish-plugins.yml    # Build & publish official plugins
│       │       └── update-catalog.yml     # Update marketplace catalog
│       ├── catalog/
│       │   ├── official/                  # Official plugin metadata
│       │   │   ├── node.json
│       │   │   ├── storage.json
│       │   │   └── identity.json
│       │   └── community/                 # Community plugin metadata
│       │       └── README.md
│       ├── website/                       # GitHub Pages source
│       │   ├── index.html                 # Marketplace homepage
│       │   ├── api/
│       │   │   └── v1/
│       │   │       └── catalog.json       # Generated catalog
│       │   └── assets/
│       ├── scripts/
│       │   ├── build-catalog.sh           # Generate catalog from metadata
│       │   ├── publish-plugin.sh          # Publish plugin to releases
│       │   └── validate-plugin.sh         # Validate plugin package
│       └── README.md
```

## GitHub Actions Workflow

### 1. Publishing Official Plugins

Create `.github/workflows/publish-plugins.yml`:

```yaml
name: Publish Official Plugins
on:
  push:
    tags:
      - 'plugins/*/v*'  # Trigger on tags like plugins/node/v1.0.0

jobs:
  publish:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - plugin: node
            path: core/pkg/plugins/node
          - plugin: storage
            path: core/pkg/plugins/storage
          - plugin: identity
            path: core/pkg/plugins/identity
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Extract version
        id: version
        run: |
          # Extract version from tag (plugins/node/v1.0.0 -> 1.0.0)
          VERSION=${GITHUB_REF#refs/tags/plugins/${{ matrix.plugin }}/v}
          echo "version=$VERSION" >> $GITHUB_OUTPUT
      
      - name: Build plugin
        working-directory: ${{ matrix.path }}
        run: |
          make clean
          make build-all  # Build for all platforms
          make package    # Create .plugin archives
      
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          name: "${{ matrix.plugin }} Plugin v${{ steps.version.outputs.version }}"
          tag_name: "plugins/${{ matrix.plugin }}/v${{ steps.version.outputs.version }}"
          files: |
            ${{ matrix.path }}/dist/*.plugin
            ${{ matrix.path }}/dist/*.plugin.sha256
            ${{ matrix.path }}/dist/*.plugin.sig
          body_path: ${{ matrix.path }}/CHANGELOG.md
      
      - name: Update catalog metadata
        run: |
          ./ecosystem/marketplace/scripts/update-plugin-metadata.sh \
            --plugin ${{ matrix.plugin }} \
            --version ${{ steps.version.outputs.version }} \
            --path ${{ matrix.path }}
      
      - name: Commit catalog updates
        uses: EndBug/add-and-commit@v9
        with:
          add: 'ecosystem/marketplace/catalog/'
          message: 'Update ${{ matrix.plugin }} plugin to v${{ steps.version.outputs.version }}'
```

### 2. Building Marketplace Catalog

Create `ecosystem/marketplace/scripts/build-catalog.sh`:

```bash
#!/bin/bash
set -e

CATALOG_DIR="ecosystem/marketplace/catalog"
OUTPUT_FILE="ecosystem/marketplace/website/api/v1/catalog.json"

echo "Building marketplace catalog..."

# Start catalog JSON
cat > "$OUTPUT_FILE" << EOF
{
  "version": "1.0",
  "updated": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "plugins": [
EOF

# Process official plugins
first=true
for plugin_file in "$CATALOG_DIR"/official/*.json; do
    if [ -f "$plugin_file" ]; then
        if [ "$first" = false ]; then
            echo "," >> "$OUTPUT_FILE"
        fi
        first=false
        
        # Merge plugin metadata with release info
        plugin_name=$(basename "$plugin_file" .json)
        
        # Get latest release info from GitHub API
        release_info=$(gh api "repos/$GITHUB_REPOSITORY/releases" \
            --jq ".[] | select(.tag_name | startswith(\"plugins/$plugin_name/\")) | {tag_name, published_at, assets: .assets[] | {name, browser_download_url, size}}" \
            | head -1)
        
        # Merge metadata with release info
        jq -s '.[0] + {release: .[1]}' "$plugin_file" <(echo "$release_info") >> "$OUTPUT_FILE"
    fi
done

# Process community plugins
for plugin_file in "$CATALOG_DIR"/community/*.json; do
    if [ -f "$plugin_file" ]; then
        echo "," >> "$OUTPUT_FILE"
        cat "$plugin_file" >> "$OUTPUT_FILE"
    fi
done

# Close catalog JSON
cat >> "$OUTPUT_FILE" << EOF

  ]
}
EOF

echo "Catalog built successfully!"
```

### 3. Plugin Metadata Format

Create `ecosystem/marketplace/catalog/official/node.json`:

```json
{
  "id": "node",
  "name": "Node Plugin",
  "description": "P2P networking and distributed communication plugin",
  "category": "networking",
  "official": true,
  "repository": {
    "type": "monorepo",
    "url": "https://github.com/blackhole-foundation/blackhole",
    "directory": "core/pkg/plugins/node"
  },
  "author": {
    "name": "Blackhole Foundation",
    "url": "https://github.com/blackhole-foundation"
  },
  "license": "MIT",
  "keywords": ["p2p", "networking", "distributed"],
  "requirements": {
    "blackhole": ">=1.0.0",
    "mesh": ">=1.0.0"
  },
  "platforms": [
    "darwin-amd64",
    "darwin-arm64", 
    "linux-amd64",
    "linux-arm64"
  ]
}
```

## GitHub Pages Deployment

### 1. Enable GitHub Pages

In repository settings:
- Source: Deploy from a branch
- Branch: `main`
- Folder: `/ecosystem/marketplace/website`

### 2. Automatic Deployment

Create `.github/workflows/deploy-marketplace.yml`:

```yaml
name: Deploy Marketplace
on:
  push:
    branches: [main]
    paths:
      - 'ecosystem/marketplace/**'
  workflow_run:
    workflows: ["Publish Official Plugins"]
    types: [completed]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build catalog
        run: |
          cd ecosystem/marketplace
          ./scripts/build-catalog.sh
      
      - name: Generate plugin stats
        run: |
          cd ecosystem/marketplace
          ./scripts/fetch-stats.sh > website/api/v1/stats.json
      
      - name: Build search index
        run: |
          cd ecosystem/marketplace
          ./scripts/build-search-index.sh > website/api/v1/search.json
      
      - name: Commit updates
        uses: EndBug/add-and-commit@v9
        with:
          add: 'ecosystem/marketplace/website/api/'
          message: 'Update marketplace catalog and stats'
```

## CLI Integration

Update the Blackhole CLI to use the monorepo marketplace:

```go
// pkg/cli/plugin/marketplace.go
const (
    // Default marketplace URL (GitHub Pages)
    DefaultMarketplaceURL = "https://blackhole-foundation.github.io/blackhole/marketplace/api/v1/catalog.json"
    
    // Release URL pattern for monorepo plugins
    ReleaseURLPattern = "https://github.com/blackhole-foundation/blackhole/releases/download/plugins/%s/v%s/%s"
)

func (m *Marketplace) InstallPlugin(name, version string) error {
    // Fetch catalog
    catalog, err := m.fetchCatalog()
    if err != nil {
        return err
    }
    
    // Find plugin
    plugin := catalog.FindPlugin(name)
    if plugin == nil {
        return fmt.Errorf("plugin %s not found", name)
    }
    
    // Build download URL for monorepo plugin
    platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
    filename := fmt.Sprintf("%s-%s-v%s.plugin", name, platform, version)
    downloadURL := fmt.Sprintf(ReleaseURLPattern, name, version, filename)
    
    // Download and install
    return m.downloadAndInstall(downloadURL, plugin.Checksum)
}
```

## Benefits of Monorepo Approach

1. **Simplified Management**
   - All code in one place
   - Unified CI/CD pipeline
   - Consistent versioning

2. **Easier Development**
   - No need to manage multiple repos
   - Shared dependencies
   - Integrated testing

3. **Better Integration**
   - Plugin SDK and plugins in same repo
   - Easier to keep in sync
   - Unified documentation

4. **Cost Effective**
   - Single repository
   - Shared GitHub Actions minutes
   - One set of secrets/configs

## Release Strategy

### Tagging Convention

```bash
# Tag the main framework
git tag v1.0.0

# Tag individual plugins
git tag plugins/node/v1.0.0
git tag plugins/storage/v1.0.0
git tag plugins/identity/v1.0.0
```

### Release Commands

```bash
# Release node plugin
cd core/pkg/plugins/node
make release
git tag plugins/node/v1.0.0
git push origin plugins/node/v1.0.0

# GitHub Actions automatically:
# 1. Builds plugin for all platforms
# 2. Creates GitHub release
# 3. Updates marketplace catalog
# 4. Deploys to GitHub Pages
```

## Community Plugins

Community plugins can be added by:

1. Fork the repository
2. Add metadata to `ecosystem/marketplace/catalog/community/`
3. Submit PR with plugin information
4. Plugin is validated and added to catalog
5. Plugin binaries remain in developer's repository

## Custom Domain Setup

To use `marketplace.blackhole.io`:

1. Add CNAME file: `ecosystem/marketplace/website/CNAME`
2. Configure DNS:
   ```
   marketplace.blackhole.io CNAME blackhole-foundation.github.io
   ```
3. Enable HTTPS in GitHub Pages settings

## Conclusion

This monorepo approach allows you to manage the entire plugin ecosystem from within the main Blackhole repository while still leveraging GitHub's free infrastructure for distribution and hosting.