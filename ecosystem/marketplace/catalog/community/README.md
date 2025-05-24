# Community Plugins

This directory contains metadata for community-contributed plugins.

## How to Submit Your Plugin

1. **Fork this repository**

2. **Create your plugin metadata file** in this directory:
   ```
   ecosystem/marketplace/catalog/community/your-plugin-name.json
   ```

3. **Use this template**:
   ```json
   {
     "id": "your-plugin-id",
     "name": "Your Plugin Name",
     "version": "1.0.0",
     "description": "What your plugin does",
     "category": "networking|storage|security|analytics|other",
     "author": {
       "name": "Your Name",
       "email": "your-email@example.com",
       "github": "your-github-username"
     },
     "repository": "https://github.com/your-username/your-plugin",
     "homepage": "https://your-plugin-website.com",
     "documentation": "https://github.com/your-username/your-plugin/blob/main/README.md",
     "license": "MIT|Apache-2.0|GPL-3.0|other",
     "keywords": ["tag1", "tag2", "tag3"],
     "requirements": {
       "blackhole": ">=1.0.0"
     },
     "downloads": {
       "darwin-amd64": {
         "url": "https://github.com/your-username/your-plugin/releases/download/v1.0.0/plugin-darwin-amd64.tar.gz",
         "sha256": "your-checksum-here"
       },
       "darwin-arm64": {
         "url": "https://github.com/your-username/your-plugin/releases/download/v1.0.0/plugin-darwin-arm64.tar.gz",
         "sha256": "your-checksum-here"
       },
       "linux-amd64": {
         "url": "https://github.com/your-username/your-plugin/releases/download/v1.0.0/plugin-linux-amd64.tar.gz",
         "sha256": "your-checksum-here"
       },
       "linux-arm64": {
         "url": "https://github.com/your-username/your-plugin/releases/download/v1.0.0/plugin-linux-arm64.tar.gz",
         "sha256": "your-checksum-here"
       }
     }
   }
   ```

4. **Submit a Pull Request**
   - Title: `Add [your-plugin-name] to marketplace`
   - Description: Brief description of what your plugin does
   - Include link to your plugin repository

## Requirements

- Plugin must follow the [Blackhole Plugin Packaging Standard](../../PACKAGING.md)
- Plugin must be open source with an OSI-approved license
- Plugin must not contain malicious code
- Plugin should have documentation and usage examples
- Plugin binaries should be signed (recommended)

## Review Process

1. Automated validation checks your metadata file
2. Community review for functionality and security
3. Merge and automatic marketplace update

## Best Practices

- Use semantic versioning (e.g., 1.0.0)
- Provide binaries for all major platforms
- Include SHA256 checksums for all downloads
- Keep your plugin updated with latest Blackhole versions
- Respond to user issues promptly

## Getting Help

- Join our [Discord community](https://discord.gg/blackhole)
- Check the [Plugin Development Guide](https://docs.blackhole.io/plugins)
- See [example plugins](https://github.com/blackhole-foundation/blackhole/tree/main/core/pkg/plugins)