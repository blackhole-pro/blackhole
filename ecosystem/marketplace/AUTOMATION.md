# Marketplace Automation

The Blackhole plugin marketplace can be automated in several ways:

## Current Manual Process

1. Developer creates plugin in `core/pkg/plugins/<plugin-name>/`
2. Developer creates metadata in `ecosystem/marketplace/catalog/official/<plugin>.json`
3. Push to main branch triggers marketplace deployment

## Automated Approaches

### 1. Release-Based Automation (Recommended)

When you create a GitHub release with tag `plugin-<name>-v<version>`:
- GitHub Action automatically updates the marketplace catalog
- Extracts metadata from release assets
- Rebuilds and deploys marketplace

**To publish a plugin:**
```bash
# Build and package plugin
cd core/pkg/plugins/node
make package

# Create release with plugin files
gh release create plugin-node-v1.0.0 \
  --title "Node Plugin v1.0.0" \
  --notes "P2P networking plugin for Blackhole" \
  dist/*.plugin \
  dist/*.sha256 \
  plugin.yaml
```

### 2. Dynamic Catalog Generation

Use `generate-catalog-from-releases.py` to build catalog from GitHub releases:
```bash
# Run periodically (e.g., in GitHub Action)
python3 ecosystem/marketplace/scripts/generate-catalog-from-releases.py \
  > ecosystem/marketplace/website/api/v1/catalog.json
```

### 3. Pull Request Automation

Developers submit PRs with plugin metadata:
- CI validates plugin manifest
- Merge triggers marketplace update
- No manual catalog editing needed

### 4. Plugin Registry API (Future)

Build a proper plugin registry service:
- REST API for plugin submission
- Automated validation and testing
- Version management
- Download statistics

## Removing Manual Steps

To fully automate, combine approaches:

1. **For Official Plugins**: Use release-based automation
2. **For Community Plugins**: Accept PRs to `catalog/community/`
3. **For Testing**: Use dynamic generation from releases

## Quick Start

Remove the sample node plugin (since it's not published):
```bash
rm ecosystem/marketplace/catalog/official/node.json
cd ecosystem/marketplace && bash scripts/build-catalog.sh
git add -A && git commit -m "Remove unpublished plugin from marketplace"
git push origin main
```

Then use the automated workflows for future plugins.