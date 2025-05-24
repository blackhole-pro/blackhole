#!/bin/bash
# Transfer repository to blackhole-fn organization using GitHub CLI

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}🚀 Transferring to blackhole-pro organization${NC}"

# Check if gh is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}❌ GitHub CLI (gh) is not installed${NC}"
    echo "Install it with: brew install gh"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo -e "${RED}❌ Not authenticated with GitHub CLI${NC}"
    echo "Run: gh auth login"
    exit 1
fi

# Current repo info
CURRENT_OWNER="handcraft"
REPO_NAME="blackhole"
NEW_ORG="blackhole-pro"

echo -e "\n${YELLOW}📋 Pre-transfer checklist:${NC}"
echo "1. Have you created the blackhole-pro organization?"
echo "   → https://github.com/organizations/new"
echo "2. Are you an owner of both organizations?"
echo "3. Have you committed all local changes?"
echo ""
read -p "Ready to proceed? (y/n) " -n 1 -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# Step 1: Create the organization (if needed)
echo -e "\n${YELLOW}Step 1: Organization Setup${NC}"
echo "Make sure blackhole-pro organization exists."
echo "Create it at: https://github.com/organizations/new"
read -p "Press Enter when the organization is created..."

# Step 2: Transfer the repository
echo -e "\n${YELLOW}Step 2: Transferring repository${NC}"
echo "Transferring ${CURRENT_OWNER}/${REPO_NAME} to ${NEW_ORG}/${REPO_NAME}..."

# Note: gh doesn't have a direct transfer command, so we'll use the API
gh api -X POST \
  repos/${CURRENT_OWNER}/${REPO_NAME}/transfer \
  -f new_owner="${NEW_ORG}" \
  -F team_ids='[]' \
  2>/dev/null || {
    echo -e "${RED}❌ Transfer failed${NC}"
    echo "Common reasons:"
    echo "- You're not an owner of both organizations"
    echo "- The target organization already has a repo with this name"
    echo "- Billing or plan restrictions"
    echo ""
    echo "Try transferring manually:"
    echo "https://github.com/${CURRENT_OWNER}/${REPO_NAME}/settings"
    echo "→ Danger Zone → Transfer ownership"
    exit 1
}

echo -e "${GREEN}✅ Repository transferred successfully!${NC}"

# Step 3: Update local git remote
echo -e "\n${YELLOW}Step 3: Updating local git remote${NC}"
OLD_REMOTE="https://github.com/${CURRENT_OWNER}/${REPO_NAME}"
NEW_REMOTE="https://github.com/${NEW_ORG}/${REPO_NAME}"

if git remote get-url origin &> /dev/null; then
    CURRENT_REMOTE=$(git remote get-url origin)
    echo "Current remote: ${CURRENT_REMOTE}"
    
    if [[ $CURRENT_REMOTE == *"${CURRENT_OWNER}/${REPO_NAME}"* ]]; then
        git remote set-url origin "${NEW_REMOTE}"
        echo -e "${GREEN}✅ Updated git remote to: ${NEW_REMOTE}${NC}"
    else
        echo -e "${YELLOW}⚠️  Remote doesn't match expected pattern${NC}"
        echo "Please update manually with:"
        echo "git remote set-url origin ${NEW_REMOTE}"
    fi
fi

# Step 4: Update repository references
echo -e "\n${YELLOW}Step 4: Updating repository references${NC}"
echo "This will update all references from ${CURRENT_OWNER} to ${NEW_ORG}"
read -p "Update all code references? (y/n) " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Update Go imports
    find . -type f -name "*.go" -exec sed -i.bak "s/github.com\/${CURRENT_OWNER}/github.com\/${NEW_ORG}/g" {} +
    find . -type f -name "*.go" -exec sed -i.bak "s/github.com\/handcraftdev/github.com\/${NEW_ORG}/g" {} +
    
    # Update go.mod
    sed -i.bak "s/github.com\/${CURRENT_OWNER}/github.com\/${NEW_ORG}/g" go.mod 2>/dev/null || true
    sed -i.bak "s/github.com\/handcraftdev/github.com\/${NEW_ORG}/g" go.mod 2>/dev/null || true
    
    # Update documentation
    find . -type f -name "*.md" -exec sed -i.bak "s/github.com\/${CURRENT_OWNER}/github.com\/${NEW_ORG}/g" {} +
    find . -type f -name "*.md" -exec sed -i.bak "s/github.com\/handcraftdev/github.com\/${NEW_ORG}/g" {} +
    
    # Update YAML/JSON files
    find . -type f \( -name "*.yml" -o -name "*.yaml" -o -name "*.json" \) -exec sed -i.bak "s/${CURRENT_OWNER}/${NEW_ORG}/g" {} +
    find . -type f \( -name "*.yml" -o -name "*.yaml" -o -name "*.json" \) -exec sed -i.bak "s/handcraftdev/${NEW_ORG}/g" {} +
    
    # Clean up backup files
    find . -name "*.bak" -type f -delete
    
    # Run go mod tidy
    echo "Running go mod tidy..."
    go mod tidy
    
    echo -e "${GREEN}✅ Updated all references${NC}"
fi

# Step 5: Next steps
echo -e "\n${GREEN}✨ Transfer complete!${NC}"
echo -e "\n${YELLOW}Next steps:${NC}"
echo "1. Verify the transfer: https://github.com/${NEW_ORG}/${REPO_NAME}"
echo "2. Update any CI/CD secrets in the new organization"
echo "3. Update GitHub Pages settings if used"
echo "4. Update any webhooks or integrations"
echo "5. Inform team members about the new location"
echo ""
echo "The old URL will automatically redirect to the new location."
echo ""
echo -e "${YELLOW}To commit the reference updates:${NC}"
echo "git add -A"
echo "git commit -m 'Update references to blackhole-pro organization'"
echo "git push origin main"