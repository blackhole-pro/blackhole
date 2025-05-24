#!/usr/bin/env python3
"""
Convert plugin.yaml manifest to marketplace catalog JSON format
"""
import sys
import json
import yaml
from datetime import datetime

def convert_manifest(manifest_path):
    """Convert plugin.yaml to marketplace JSON format"""
    with open(manifest_path, 'r') as f:
        manifest = yaml.safe_load(f)
    
    # Extract metadata
    metadata = manifest.get('metadata', {})
    spec = manifest.get('spec', {})
    
    # Build marketplace entry
    marketplace_entry = {
        "id": metadata.get('name'),
        "name": spec.get('displayName', metadata.get('name')),
        "version": spec.get('version'),
        "description": spec.get('description', ''),
        "category": spec.get('category', 'uncategorized'),
        "author": spec.get('author', {}),
        "license": spec.get('license', 'MIT'),
        "source": {
            "type": "release",
            "url": f"https://github.com/blackhole-pro/blackhole/releases/tag/plugin-{metadata.get('name')}-v{spec.get('version')}"
        },
        "repository": spec.get('repository', 'https://github.com/blackhole-pro/blackhole'),
        "documentation": spec.get('documentation', ''),
        "keywords": spec.get('keywords', []),
        "requirements": spec.get('requirements', {}),
        "capabilities": spec.get('capabilities', []),
        "resources": spec.get('resources', {}),
        "stats": {
            "updated": datetime.utcnow().isoformat() + 'Z'
        }
    }
    
    return marketplace_entry

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: convert-plugin-manifest.py <plugin.yaml>", file=sys.stderr)
        sys.exit(1)
    
    try:
        result = convert_manifest(sys.argv[1])
        print(json.dumps(result, indent=2))
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)