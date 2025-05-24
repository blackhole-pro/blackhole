# Organization Name Transition Plan

## Current State
- Current: `handcraft`
- Repository: `github.com/blackhole-pro/*`

## Naming Options Analysis

### 1. `blackhole-protocol` 
**Pros:**
- Very descriptive and clear
- Good for SEO
- Indicates it's a protocol/framework

**Cons:**
- Long (17 characters)
- URLs become lengthy: `github.com/blackhole-protocol/blackhole`
- Package names: `@blackhole-protocol/sdk`

### 2. `blackhole-io`
**Pros:**
- Short and memorable (12 characters)
- Common pattern for tech projects
- Clean URLs: `github.com/blackhole-io/blackhole`
- Good for packages: `@blackhole-io/sdk`

**Cons:**
- Need to secure blackhole.io domain
- Very common pattern

### 3. `blackhole-foundation`
**Pros:**
- Establishes authority and permanence
- Good for open source governance
- Examples: Apache Foundation, Linux Foundation
- Professional image

**Cons:**
- Longest option (20 characters)
- May imply non-profit status

### 4. `blackhole-dev`
**Pros:**
- Short and developer-focused
- Easy to type
- Clear tech focus

**Cons:**
- Less unique
- Might seem less official

### 5. `blkhl` (abbreviated)
**Pros:**
- Very short (5 characters)
- Unique identifier
- Easy to type

**Cons:**
- Not immediately recognizable
- Loses brand recognition
- Hard to pronounce

### 6. `blackhole` (if available)
**Pros:**
- Perfect brand match
- Shortest relevant option
- Clean and simple

**Cons:**
- Likely already taken
- Very competitive namespace

## Recommendation: `blackhole-io`

**Why this choice:**
1. **Balanced Length**: Not too long, not too short
2. **Professional**: Common pattern for serious projects
3. **Memorable**: Easy to remember and share
4. **Package Friendly**: `@blackhole-io/plugin-sdk`
5. **Domain Available**: Can use blackhole.io for website

## Transition Plan

### Phase 1: Preparation (Week 1)
1. **Create New Organization**
   ```bash
   # Create on GitHub
   https://github.com/organizations/new
   # Name: blackhole-io
   ```

2. **Secure Domains**
   - blackhole.io (primary)
   - blackhole.dev (backup)
   - blackholeprotocol.com (defensive)

3. **Update Documentation**
   - Find all references to `handcraft`
   - Prepare replacement scripts

### Phase 2: Migration (Week 2)

#### A. Repository Migration
```bash
# Transfer main repository
# GitHub: Settings → Transfer ownership → blackhole-io

# Or create new and push
git remote add blackhole-io https://github.com/blackhole-io/blackhole
git push blackhole-io --all
git push blackhole-io --tags
```

#### B. Update All References
```bash
# Script to update imports and URLs
find . -type f -name "*.go" -o -name "*.md" -o -name "*.json" -o -name "*.yaml" | \
  xargs sed -i 's/github.com\/handcraft/github.com\/blackhole-io/g'

find . -type f -name "*.go" -o -name "*.md" -o -name "*.json" -o -name "*.yaml" | \
  xargs sed -i 's/@handcraft/@blackhole-io/g'
```

#### C. Update Configuration Files
1. **go.mod**
   ```go
   module github.com/blackhole-io/blackhole
   ```

2. **package.json** (if any)
   ```json
   {
     "name": "@blackhole-io/core",
     "repository": "github.com/blackhole-io/blackhole"
   }
   ```

3. **Docker images**
   ```dockerfile
   FROM blackholeio/blackhole:latest
   ```

### Phase 3: Communication (Week 3)

1. **Announcement Blog Post**
   - Explain the change
   - New organization name
   - No breaking changes for users

2. **Update Documentation**
   - README.md
   - Contributing guidelines
   - Website

3. **Redirect Setup**
   - GitHub automatically redirects old URLs
   - Set up domain redirects

### Phase 4: Cleanup (Week 4)

1. **Archive Old Organization**
   - Keep for historical reference
   - Add deprecation notice

2. **Update External References**
   - Stack Overflow tags
   - Reddit communities
   - Discord server

## Impact Analysis

### What Changes:
- Git remote URLs
- Package names
- Docker image names
- Documentation links
- CI/CD configurations

### What Stays the Same:
- Project name: "Blackhole"
- Functionality
- APIs
- License

## Quick Migration Commands

```bash
# 1. Clone with new remote
git clone https://github.com/blackhole-pro/blackhole
cd blackhole
git remote set-url origin https://github.com/blackhole-io/blackhole

# 2. Update all imports (macOS)
find . -type f \( -name "*.go" -o -name "*.md" \) -exec \
  sed -i '' 's/github.com\/handcraft/github.com\/blackhole-io/g' {} +

# 3. Update all imports (Linux)
find . -type f \( -name "*.go" -o -name "*.md" \) -exec \
  sed -i 's/github.com\/handcraft/github.com\/blackhole-io/g' {} +

# 4. Update go.mod
go mod edit -module github.com/blackhole-io/blackhole

# 5. Update and tidy
go mod tidy

# 6. Verify
grep -r "handcraft" . --exclude-dir=.git
```

## Alternative: Keep Both Names

Consider keeping `handcraft` as the parent organization and `blackhole-io` as the project-specific org:

```
handcraft/
├── blackhole/          # Move to blackhole-io/blackhole
├── other-project/      # Keep under handcraft
└── ...
```

This allows you to:
- Maintain handcraft for other projects
- Have dedicated org for Blackhole
- Cleaner separation of concerns