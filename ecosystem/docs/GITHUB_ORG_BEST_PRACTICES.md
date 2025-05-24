# GitHub Organization Structure Best Practices

## GitHub Organization Limitations

**Important**: GitHub does **NOT** support sub-organizations. It's a flat structure.

You cannot have:
```
blackhole-pro/
└── blackhole-io/  ❌ Not possible
    └── blackhole
```

## What You CAN Do

### 1. **Single Organization with Project Prefixes** (Common)
```
blackhole-pro/
├── blackhole                    # Main project
├── blackhole-plugins            # Plugins repo
├── blackhole-marketplace        # Marketplace
├── blackhole-docs              # Documentation
├── other-project               # Your other projects
└── another-project
```

**Pros:**
- Everything under one org
- Easier permissions management
- Single billing/settings

**Cons:**
- Can get cluttered with many projects
- Less clear branding for Blackhole

### 2. **Separate Organizations** (Recommended for Large Projects)
```
blackhole-pro/                      # Your company/personal projects
├── project-a
└── project-b

blackhole-io/                   # Dedicated to Blackhole
├── blackhole                   # Core framework
├── plugins                     # Official plugins
├── marketplace                 # Marketplace
└── community                   # Community resources
```

**Pros:**
- Clear separation and branding
- Dedicated namespace
- Can have different teams/permissions
- Professional appearance

**Cons:**
- Need to manage multiple orgs
- Separate billing (if using paid features)

## Industry Examples

### Projects with Dedicated Orgs:
- **Kubernetes**: `kubernetes` + `kubernetes-sigs`
- **Docker**: `docker` + `moby`
- **Node.js**: `nodejs` + `npm`
- **Rust**: `rust-lang` + `rust-lang-nursery`
- **Vue.js**: `vuejs` + `vitejs`

### Companies with Multiple Orgs:
- **Microsoft**: `microsoft`, `dotnet`, `Azure`, `PowerShell`
- **Google**: `google`, `golang`, `kubernetes`, `grpc`
- **Facebook**: `facebook`, `reactjs`, `facebookresearch`

## Best Practice: Transfer vs Rename

### Standard Practice is TRANSFER:

1. **Create New Organization First**
   ```
   https://github.com/organizations/new
   → Name: blackhole-io
   ```

2. **Transfer Repository**
   ```
   blackhole-pro/blackhole → Settings → Danger Zone → Transfer ownership
   → New owner: blackhole-io
   ```

3. **What Happens:**
   - GitHub automatically redirects old URLs
   - Git remotes continue to work
   - Issues, PRs, stars, watches preserved
   - Contributors maintain access
   - No data loss

### Why Transfer is Better than Fork:
- ✅ Preserves all history, issues, PRs
- ✅ Maintains star count and watchers  
- ✅ Automatic redirects from old URLs
- ✅ SEO preserved
- ✅ No duplicate repositories

## Recommended Approach for Blackhole

### Option A: Keep Everything in Handcraft (Simplest)
```
blackhole-pro/
├── blackhole
├── blackhole-plugins
├── blackhole-marketplace
└── blackhole-examples
```

Use if:
- You want simplicity
- Handcraft is your main brand
- You have other projects there

### Option B: Create Dedicated Org (Professional)
```
1. Create blackhole-io organization
2. Transfer blackhole repository
3. Keep blackhole-pro for other projects
```

Use if:
- Blackhole is becoming a major project
- You want professional/enterprise appearance
- You plan community governance
- You want clear separation

### Option C: Gradual Migration
```
Phase 1: Keep in blackhole-pro
Phase 2: When project matures, create blackhole-io
Phase 3: Transfer when ready
```

## How to Decide

Ask yourself:
1. **Is Blackhole the main focus?** → Dedicated org
2. **Will you have many Blackhole-related repos?** → Dedicated org
3. **Want to build a community/foundation?** → Dedicated org
4. **Just starting out?** → Stay in blackhole-pro for now
5. **Have many other projects?** → Keep separate

## Migration Timeline Example

### Week 1: Preparation
- Decide on organization name
- Check domain availability
- Plan repository structure

### Week 2: Setup
- Create new organization
- Set up teams and permissions
- Configure organization settings

### Week 3: Transfer
- Transfer main repository
- Update documentation
- Set up redirects

### Week 4: Communication
- Announce to community
- Update all external links
- Monitor for issues

## Technical Considerations

### What Changes with Transfer:
```bash
# Old URL (redirects automatically)
https://github.com/blackhole-pro/blackhole

# New URL
https://github.com/blackhole-io/blackhole

# Git remote (auto-redirected but should update)
git remote set-url origin https://github.com/blackhole-io/blackhole
```

### What Stays the Same:
- All issues and PRs
- All commits and history
- All stars and watches
- All forks and their connection
- All GitHub Pages settings

## Cost Analysis

### GitHub Free:
- Unlimited public repos
- 2,000 Actions minutes/month
- Basic features

### GitHub Team ($4/user/month):
- Advanced permissions
- Protected branches
- Code owners
- Draft PRs

### When You Need Paid:
- Private repos with >3 collaborators
- Advanced security features
- More Actions minutes
- SAML SSO

## My Recommendation

For Blackhole, I recommend **Option B: Create Dedicated Organization** because:

1. **Professional Image**: Shows project maturity
2. **Clear Namespace**: `blackhole-io/blackhole` is cleaner
3. **Future Growth**: Room for plugins, tools, community
4. **Easy Transfer**: GitHub makes it seamless
5. **No Downside**: Redirects preserve everything

The process is:
1. Create `blackhole-io` organization
2. Transfer repository (takes 30 seconds)
3. Update remote in your local clone
4. Done!

This is standard practice for projects that are growing beyond personal/company projects into community projects.