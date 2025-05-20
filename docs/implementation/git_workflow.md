# Git Workflow for Blackhole Project

This document outlines the Git workflow and best practices for the Blackhole project, ensuring consistent and efficient collaboration across the team.

## 1. Repository Initialization and Structure

The Blackhole repository follows a structured branching model based on the GitFlow workflow, adapted for our subprocess architecture.

### Main Branches

- **`main`**: The production-ready branch containing stable releases
- **`develop`**: The integration branch for ongoing development
- **`release/*`**: Feature-complete releases undergoing final testing
- **`hotfix/*`**: Emergency fixes for production issues

### Service-Specific Feature Branches

Given Blackhole's subprocess architecture, we organize feature branches by service:

```
feature/identity/did-resolution
feature/storage/ipfs-integration
feature/node/peer-discovery
feature/ledger/transaction-validation
feature/analytics/privacy-preserving-metrics
```

## 2. Branching Strategy (GitFlow)

We use a modified GitFlow workflow optimized for our subprocess architecture:

```
        hotfix/1.0.1  ------>|
                             v
main     ---------------o----o-------------------o---->
           ^                                     ^
           |                                     |
release    |    release/1.0  o---->o----------->|
           ^                 ^                   
           |                 |                   
develop    o---o---o---o---o-o---o---o---o---o---o---->
                 ^       ^     ^         ^
                 |       |     |         |
feature    -->o--o-->    -->o--o-->  -->o--o-->
```

### Workflow Rules

1. **Feature Development**:
   - Create feature branches from `develop`
   - Prefix with service name: `feature/service-name/feature-description`
   - Merge back to `develop` via pull request

2. **Release Preparation**:
   - Branch `release/x.y.z` from `develop` when ready
   - Only bug fixes and documentation on release branches
   - Merge to both `main` and `develop` when complete

3. **Hotfix Process**:
   - Branch `hotfix/x.y.z` from `main` for urgent production fixes
   - Merge to both `main` and `develop` when complete

4. **Main Branch Protection**:
   - Direct commits to `main` and `develop` are prohibited
   - All changes must go through pull requests with reviews

## 3. Commit Message Conventions

We use conventional commits to maintain a clean and informative git history.

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation changes
- **style**: Formatting changes (whitespace, formatting, etc.)
- **refactor**: Code changes that neither fix bugs nor add features
- **perf**: Performance improvements
- **test**: Test-related changes
- **chore**: Maintenance tasks, build changes, etc.

### Scopes

For Blackhole, scopes align with our service architecture:

- **core**: Core orchestration layer
- **mesh**: Service mesh components
- **identity**: Identity service
- **storage**: Storage service
- **node**: Node service
- **ledger**: Ledger service
- **indexer**: Indexer service
- **social**: Social service
- **analytics**: Analytics service
- **telemetry**: Telemetry service
- **wallet**: Wallet service
- **build**: Build system
- **docs**: Documentation
- **config**: Configuration

### Examples

```
feat(identity): implement DID resolution endpoint

Added HTTP and gRPC handlers for DID resolution according to the W3C spec.
This enables clients to resolve DIDs through our identity service.

Resolves: #123
```

```
fix(storage): correct IPFS connection retry logic

Fixed an issue where IPFS connection would fail permanently after timeout
instead of implementing the exponential backoff strategy.

Fixes: #456
```

## 4. Pull Request Workflow

Pull requests are the primary method for integrating changes into the codebase.

### PR Creation Guidelines

1. **Naming**: Use prefix + brief description:
   - `[FEATURE] Add DID resolution endpoint`
   - `[FIX] Correct IPFS connection retry logic`
   - `[DOCS] Update storage service architecture`
   - `[REFACTOR] Improve process manager performance`

2. **Description Template**:
   ```markdown
   ## Description
   Brief description of changes and motivation

   ## Type of change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update

   ## Service Impact
   - Primary service: identity
   - Related services: storage, node

   ## Testing
   - [ ] Unit tests added/updated
   - [ ] Integration tests added/updated
   - [ ] Manual testing completed

   ## Checklist
   - [ ] My code follows the style guidelines
   - [ ] I have performed a self-review
   - [ ] I have commented my code appropriately
   - [ ] My changes generate no new warnings
   - [ ] I have updated the documentation

   Closes #789
   ```

3. **Size Guidelines**:
   - Keep PRs focused on a single logical change
   - Target under 500 lines of changes where possible
   - Split large features into smaller, sequential PRs

### Review Process

1. **Required Approvals**: Minimum 1 approval, 2 for core components
2. **Code Owners**: Automatic assignment based on service
3. **CI Checks**: All checks must pass before merging
4. **Review SLA**: Reviews completed within 24 hours

### Merging Strategy

1. **Squash and Merge** for feature branches to maintain clean history
2. **Merge Commit** for release and hotfix branches to preserve history

## 5. Code Review Practices

Effective code reviews ensure quality and knowledge sharing.

### Reviewer Guidelines

1. **Focus Areas**:
   - Code correctness
   - Service isolation boundaries
   - RPC communication patterns
   - Security implications
   - Resource management
   - Testing coverage
   - Documentation completeness

2. **Review Checklist**:
   - Does the code follow our design principles?
   - Are subprocess boundaries properly maintained?
   - Is error handling comprehensive?
   - Are there sufficient tests?
   - Is the code optimized for our use case?
   - Are there any security implications?
   - Is the documentation updated?

3. **Constructive Feedback**:
   - Be specific and actionable
   - Reference patterns and documentation
   - Suggest alternatives
   - Distinguish between required changes and suggestions

### Author Responsibilities

1. **PR Preparation**:
   - Self-review before submission
   - Run full test suite locally
   - Document design decisions
   - Explain complex or non-obvious code

2. **Responding to Feedback**:
   - Address all comments
   - Explain changes made
   - Request re-review when ready

## 6. Versioning Strategy

We use Semantic Versioning (SemVer) for the Blackhole project.

### Version Format

```
MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
```

- **MAJOR**: Incompatible API changes
- **MINOR**: Backward-compatible new functionality
- **PATCH**: Backward-compatible bug fixes
- **PRERELEASE**: Alpha, beta, rc designations (e.g., 1.0.0-beta.1)
- **BUILD**: Build metadata (e.g., 1.0.0+20230501)

### Version Bumping Guidelines

1. **MAJOR Version**: Incremented for:
   - Changes to service RPC interfaces
   - Breaking changes to client APIs
   - Significant architectural changes

2. **MINOR Version**: Incremented for:
   - New services or features
   - Non-breaking API additions
   - Substantial improvements to existing functionality

3. **PATCH Version**: Incremented for:
   - Bug fixes
   - Small optimizations
   - Documentation improvements

### Example Version Progression

```
1.0.0-alpha.1 -> 1.0.0-alpha.2 -> 1.0.0-beta.1 -> 1.0.0-rc.1 -> 1.0.0 -> 1.0.1 -> 1.1.0 -> 2.0.0
```

## 7. Release Process

The release process ensures stable and predictable software delivery.

### Release Preparation

1. **Create Release Branch**:
   ```bash
   git checkout develop
   git pull
   git checkout -b release/1.0.0
   ```

2. **Version Bump**:
   - Update version in appropriate files:
     - `cmd/blackhole/version.go`
     - `configs/version.yaml`
     - Documentation references

3. **Release Testing**:
   - Run comprehensive test suite
   - Deploy to staging environment
   - Perform integration testing
   - Validate service interoperability

4. **Release Notes**:
   - Compile changelog from commits
   - Document upgrade steps
   - Note breaking changes
   - Highlight new features

### Release Finalization

1. **Final Pull Request**:
   - Create PR from `release/1.0.0` to `main`
   - Include finalized release notes
   - Require approvals from senior team members

2. **Merge to Main**:
   ```bash
   # After PR approval
   git checkout main
   git pull
   git merge --no-ff release/1.0.0
   git tag -a v1.0.0 -m "Release 1.0.0"
   git push origin main --tags
   ```

3. **Merge Back to Develop**:
   ```bash
   git checkout develop
   git pull
   git merge --no-ff release/1.0.0
   git push origin develop
   ```

4. **Cleanup**:
   ```bash
   git branch -d release/1.0.0
   ```

### Release Artifacts

1. **Binary Releases**:
   - Cross-platform binaries (Linux, macOS, Windows)
   - Docker images
   - Checksums and signatures

2. **Documentation**:
   - Updated user guides
   - API documentation
   - Release notes

3. **Announcements**:
   - GitHub releases
   - Project website
   - Community channels

## 8. Handling Hotfixes

Hotfixes address critical issues in production releases.

### Hotfix Workflow

1. **Create Hotfix Branch**:
   ```bash
   git checkout main
   git pull
   git checkout -b hotfix/1.0.1
   ```

2. **Implement Fix**:
   - Keep changes minimal and focused
   - Add specific tests for the issue
   - Update version numbers

3. **Review and Testing**:
   - Thorough code review
   - Comprehensive testing
   - Verification that the fix addresses the issue

4. **Merge to Main**:
   ```bash
   git checkout main
   git pull
   git merge --no-ff hotfix/1.0.1
   git tag -a v1.0.1 -m "Hotfix 1.0.1"
   git push origin main --tags
   ```

5. **Merge to Develop**:
   ```bash
   git checkout develop
   git pull
   git merge --no-ff hotfix/1.0.1
   git push origin develop
   ```

6. **Cleanup**:
   ```bash
   git branch -d hotfix/1.0.1
   ```

### Emergency Release Process

For critical issues requiring immediate attention:

1. **Expedited Review**: Shorter but mandatory review cycle
2. **Emergency Testing**: Focused test suite execution
3. **Staged Rollout**: Phased deployment to detect issues
4. **Post-mortem**: Document root cause and prevention

## 9. Git Hooks for Quality Control

Git hooks automate quality checks and ensure consistency.

### Pre-commit Hooks

1. **Linting**:
   - Go: `golangci-lint`
   - JavaScript/TypeScript: `eslint`
   - YAML: `yamllint`

2. **Formatting**:
   - Go: `gofmt` or `goimports`
   - JavaScript/TypeScript: `prettier`
   - YAML/JSON: `prettier`

3. **License Headers**:
   - Verify copyright notices
   - Check license headers

4. **Commit Message Validation**:
   - Ensure conventional commit format
   - Check for scope validity
   - Verify references to issues

### Setup Instructions

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install
```

### Sample `.pre-commit-config.yaml`

```yaml
repos:
-   repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.4.0
    hooks:
    -   id: trailing-whitespace
    -   id: end-of-file-fixer
    -   id: check-yaml
    -   id: check-added-large-files

-   repo: https://github.com/golangci/golangci-lint
    rev: v1.51.2
    hooks:
    -   id: golangci-lint

-   repo: https://github.com/commitizen-tools/commitizen
    rev: v2.42.1
    hooks:
    -   id: commitizen
        stages: [commit-msg]
```

## 10. CI/CD Integration Considerations

Our CI/CD pipeline is tightly integrated with our Git workflow.

### CI Pipeline Components

1. **Build Verification**:
   - Compile all services
   - Verify module dependencies
   - Check for build warnings

2. **Test Execution**:
   - Unit tests
   - Integration tests
   - End-to-end tests

3. **Code Quality**:
   - Static analysis
   - Test coverage reporting
   - Duplicate code detection

4. **Security Scanning**:
   - Dependency vulnerability checking
   - SAST (Static Application Security Testing)
   - License compliance

### CD Pipeline Components

1. **Automated Deployments**:
   - Development environment: Every merge to `develop`
   - Staging environment: Every merge to `release/*`
   - Production environment: Every merge to `main` (with approval)

2. **Deployment Verification**:
   - Smoke tests
   - Health checks
   - Performance validation

3. **Rollback Capability**:
   - Automated rollback for failed deployments
   - Version pinning
   - Deployment history

### GitHub Actions Example

```yaml
name: Blackhole CI

on:
  push:
    branches: [main, develop, 'release/*', 'hotfix/*']
  pull_request:
    branches: [main, develop, 'release/*']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.22'
      - name: Build
        run: make build
      - name: Test
        run: make test
      - name: Lint
        run: make lint

  service-tests:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [identity, storage, node, ledger, indexer, social, analytics, telemetry, wallet]
    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.22'
      - name: Test ${{ matrix.service }} service
        run: cd internal/services/${{ matrix.service }} && go test -v ./...

  deploy-dev:
    if: github.ref == 'refs/heads/develop'
    needs: [build, service-tests]
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to development
        run: |
          echo "Deploying to development environment"
          # Deployment steps

  deploy-staging:
    if: startsWith(github.ref, 'refs/heads/release/')
    needs: [build, service-tests]
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to staging
        run: |
          echo "Deploying to staging environment"
          # Deployment steps

  deploy-production:
    if: github.ref == 'refs/heads/main'
    needs: [build, service-tests]
    runs-on: ubuntu-latest
    environment: production
    steps:
      - name: Deploy to production
        run: |
          echo "Deploying to production environment"
          # Deployment steps with approval
```

## Conclusion

Following this Git workflow ensures consistent, high-quality development across the Blackhole project. By adhering to these practices, we maintain a clean repository history, enable efficient collaboration, and streamline our release process.

The process is designed specifically for our subprocess architecture, with special consideration for service boundaries, RPC interfaces, and independent versioning needs of different components.

## References

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Flow](https://guides.github.com/introduction/flow/)
- [Git Best Practices](https://sethrobertson.github.io/GitBestPractices/)