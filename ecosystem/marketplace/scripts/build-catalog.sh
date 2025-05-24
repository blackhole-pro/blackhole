#!/bin/bash
set -e

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MARKETPLACE_DIR="$(dirname "$SCRIPT_DIR")"
CATALOG_DIR="$MARKETPLACE_DIR/catalog"
WEBSITE_DIR="$MARKETPLACE_DIR/website"
OUTPUT_FILE="$WEBSITE_DIR/api/v1/catalog.json"

echo "Building marketplace catalog..."

# Ensure output directory exists
mkdir -p "$WEBSITE_DIR/api/v1"

# Start catalog JSON
cat > "$OUTPUT_FILE" << 'EOF'
{
  "version": "1.0",
  "updated": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "baseUrl": "https://github.com/blackhole-foundation/blackhole/releases/download",
  "plugins": [
EOF

# Replace the date placeholder
sed -i.bak "s/\$(date -u +\"%Y-%m-%dT%H:%M:%SZ\")/$(date -u +"%Y-%m-%dT%H:%M:%SZ")/g" "$OUTPUT_FILE" && rm "$OUTPUT_FILE.bak"

# Process official plugins
first=true
for metadata_file in "$CATALOG_DIR"/official/*.json; do
    if [ -f "$metadata_file" ]; then
        if [ "$first" = false ]; then
            echo "," >> "$OUTPUT_FILE"
        fi
        first=false
        
        plugin_name=$(basename "$metadata_file" .json)
        echo "Processing official plugin: $plugin_name"
        
        # Read metadata
        metadata=$(cat "$metadata_file")
        plugin_id=$(echo "$metadata" | jq -r '.id // empty')
        version=$(echo "$metadata" | jq -r '.version // empty')
        
        if [ -z "$plugin_id" ] || [ -z "$version" ]; then
            echo "Warning: Missing id or version in $metadata_file"
            continue
        fi
        
        # Build download URLs for each platform
        cat >> "$OUTPUT_FILE" << EOF
    {
      "id": "$plugin_id",
      "name": "$(echo "$metadata" | jq -r '.name // .id')",
      "description": "$(echo "$metadata" | jq -r '.description // ""')",
      "version": "$version",
      "official": true,
      "source": $(echo "$metadata" | jq '.source // {}'),
      "downloads": {
        "darwin-amd64": {
          "url": "https://github.com/blackhole-foundation/blackhole/releases/download/plugins/$plugin_id/v$version/$plugin_id-darwin-amd64-v$version.plugin",
          "checksumUrl": "https://github.com/blackhole-foundation/blackhole/releases/download/plugins/$plugin_id/v$version/$plugin_id-darwin-amd64-v$version.plugin.sha256"
        },
        "darwin-arm64": {
          "url": "https://github.com/blackhole-foundation/blackhole/releases/download/plugins/$plugin_id/v$version/$plugin_id-darwin-arm64-v$version.plugin",
          "checksumUrl": "https://github.com/blackhole-foundation/blackhole/releases/download/plugins/$plugin_id/v$version/$plugin_id-darwin-arm64-v$version.plugin.sha256"
        },
        "linux-amd64": {
          "url": "https://github.com/blackhole-foundation/blackhole/releases/download/plugins/$plugin_id/v$version/$plugin_id-linux-amd64-v$version.plugin",
          "checksumUrl": "https://github.com/blackhole-foundation/blackhole/releases/download/plugins/$plugin_id/v$version/$plugin_id-linux-amd64-v$version.plugin.sha256"
        },
        "linux-arm64": {
          "url": "https://github.com/blackhole-foundation/blackhole/releases/download/plugins/$plugin_id/v$version/$plugin_id-linux-arm64-v$version.plugin",
          "checksumUrl": "https://github.com/blackhole-foundation/blackhole/releases/download/plugins/$plugin_id/v$version/$plugin_id-linux-arm64-v$version.plugin.sha256"
        }
      },
      "metadata": $(echo "$metadata" | jq 'del(.id, .version, .source)')
    }
EOF
    fi
done

# Process community plugins
for metadata_file in "$CATALOG_DIR"/community/*.json; do
    if [ -f "$metadata_file" ] && [ "$metadata_file" != "$CATALOG_DIR/community/README.md" ]; then
        echo "," >> "$OUTPUT_FILE"
        echo "Processing community plugin: $(basename "$metadata_file" .json)"
        
        # Community plugins have external URLs
        cat "$metadata_file" | jq '. + {official: false}' >> "$OUTPUT_FILE"
    fi
done

# Close catalog JSON
cat >> "$OUTPUT_FILE" << 'EOF'

  ]
}
EOF

echo "Catalog built successfully at: $OUTPUT_FILE"

# Validate JSON
if command -v jq &> /dev/null; then
    if jq . "$OUTPUT_FILE" > /dev/null 2>&1; then
        echo "✓ Catalog JSON is valid"
    else
        echo "✗ Catalog JSON is invalid!"
        exit 1
    fi
else
    echo "Warning: jq not installed, skipping JSON validation"
fi