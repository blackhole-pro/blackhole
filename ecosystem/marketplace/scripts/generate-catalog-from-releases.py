#!/usr/bin/env python3
"""
Generate marketplace catalog from GitHub releases
This can be run periodically to update the catalog
"""
import json
import subprocess
from datetime import datetime

def get_plugin_releases():
    """Get all plugin releases from GitHub"""
    cmd = ['gh', 'release', 'list', '--json', 'tagName,name,body,publishedAt,assets', '--limit', '100']
    result = subprocess.run(cmd, capture_output=True, text=True)
    
    if result.returncode != 0:
        raise Exception(f"Failed to get releases: {result.stderr}")
    
    releases = json.loads(result.stdout)
    
    # Filter for plugin releases
    plugin_releases = []
    for release in releases:
        if release['tagName'].startswith('plugin-'):
            plugin_releases.append(release)
    
    return plugin_releases

def parse_plugin_release(release):
    """Parse a plugin release into catalog format"""
    tag = release['tagName']
    # Extract plugin ID and version from tag (e.g., plugin-node-v1.0.0)
    parts = tag.split('-')
    if len(parts) < 3:
        return None
    
    plugin_id = parts[1]
    version = parts[2].lstrip('v')
    
    # Find plugin assets
    assets = release.get('assets', [])
    downloads = {}
    
    for asset in assets:
        name = asset['name']
        if name.endswith('.plugin'):
            # Extract platform from filename
            for platform in ['darwin-amd64', 'darwin-arm64', 'linux-amd64', 'linux-arm64']:
                if platform in name:
                    downloads[platform] = {
                        "url": asset['url'],
                        "checksumUrl": asset['url'].replace('.plugin', '.plugin.sha256')
                    }
    
    if not downloads:
        return None
    
    # Parse release body for metadata (if structured)
    metadata = {
        "description": release.get('body', '').split('\n')[0] if release.get('body') else '',
        "publishedAt": release.get('publishedAt', '')
    }
    
    return {
        "id": plugin_id,
        "name": plugin_id.replace('-', ' ').title(),
        "version": version,
        "official": True,
        "downloads": downloads,
        "metadata": metadata
    }

def generate_catalog():
    """Generate the complete catalog"""
    releases = get_plugin_releases()
    plugins = []
    
    for release in releases:
        plugin = parse_plugin_release(release)
        if plugin:
            plugins.append(plugin)
    
    catalog = {
        "version": "1.0",
        "updated": datetime.utcnow().isoformat() + 'Z',
        "baseUrl": "https://github.com/blackhole-pro/blackhole/releases/download",
        "plugins": plugins
    }
    
    return catalog

if __name__ == "__main__":
    try:
        catalog = generate_catalog()
        print(json.dumps(catalog, indent=2))
    except Exception as e:
        print(f"Error: {e}")
        exit(1)