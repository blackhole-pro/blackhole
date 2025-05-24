#!/bin/bash
# Script to migrate from handcraft to blackhole-io

set -e

echo "🚀 Starting organization name migration..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Old and new org names
OLD_ORG="handcraft"
NEW_ORG="blackhole-io"

echo -e "${YELLOW}Migrating from ${OLD_ORG} to ${NEW_ORG}${NC}"

# Find and update all Go files
echo "📝 Updating Go imports..."
find . -type f -name "*.go" -print0 | xargs -0 sed -i.bak "s/github.com\/${OLD_ORG}/github.com\/${NEW_ORG}/g"

# Update go.mod
echo "📦 Updating go.mod..."
if [ -f "go.mod" ]; then
    sed -i.bak "s/github.com\/${OLD_ORG}/github.com\/${NEW_ORG}/g" go.mod
fi

# Update Markdown files
echo "📄 Updating documentation..."
find . -type f -name "*.md" -print0 | xargs -0 sed -i.bak "s/github.com\/${OLD_ORG}/github.com\/${NEW_ORG}/g"

# Update YAML files
echo "⚙️  Updating YAML configurations..."
find . -type f -name "*.yml" -o -name "*.yaml" -print0 | xargs -0 sed -i.bak "s/${OLD_ORG}/${NEW_ORG}/g"

# Update JSON files
echo "📋 Updating JSON files..."
find . -type f -name "*.json" -print0 | xargs -0 sed -i.bak "s/${OLD_ORG}/${NEW_ORG}/g"

# Update Makefiles
echo "🔧 Updating Makefiles..."
find . -type f -name "Makefile" -o -name "*.mk" -print0 | xargs -0 sed -i.bak "s/${OLD_ORG}/${NEW_ORG}/g"

# Update Dockerfiles
echo "🐳 Updating Dockerfiles..."
find . -type f -name "Dockerfile*" -print0 | xargs -0 sed -i.bak "s/${OLD_ORG}/${NEW_ORG}/g"

# Update shell scripts
echo "📜 Updating shell scripts..."
find . -type f -name "*.sh" -print0 | xargs -0 sed -i.bak "s/${OLD_ORG}/${NEW_ORG}/g"

# Clean up backup files
echo "🧹 Cleaning up backup files..."
find . -name "*.bak" -type f -delete

# Update git remote
echo "🔗 Updating git remote..."
if git remote get-url origin | grep -q "${OLD_ORG}"; then
    OLD_URL=$(git remote get-url origin)
    NEW_URL=$(echo $OLD_URL | sed "s/${OLD_ORG}/${NEW_ORG}/g")
    echo -e "${YELLOW}Current remote: ${OLD_URL}${NC}"
    echo -e "${GREEN}New remote: ${NEW_URL}${NC}"
    
    read -p "Update git remote? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        git remote set-url origin "$NEW_URL"
        echo -e "${GREEN}✓ Git remote updated${NC}"
    fi
fi

# Run go mod tidy
echo "🔄 Running go mod tidy..."
go mod tidy

# Final check
echo -e "\n${YELLOW}Checking for remaining references...${NC}"
if grep -r "${OLD_ORG}" . --exclude-dir=.git --exclude-dir=vendor --exclude="*.bak" --exclude="migrate-org-name.sh"; then
    echo -e "${RED}⚠️  Found remaining references to ${OLD_ORG}${NC}"
    echo "Please review and update manually."
else
    echo -e "${GREEN}✅ No remaining references found!${NC}"
fi

echo -e "\n${GREEN}✨ Migration complete!${NC}"
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Review the changes: git diff"
echo "2. Commit the changes: git add -A && git commit -m 'Migrate from ${OLD_ORG} to ${NEW_ORG}'"
echo "3. Create the new organization on GitHub: https://github.com/organizations/new"
echo "4. Transfer or push the repository to the new organization"
echo "5. Update any CI/CD secrets and webhooks"