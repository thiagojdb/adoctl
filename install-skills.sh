#!/bin/bash
#
# Install script for adoctl Claude Code skills
#
# This script installs the adoctl skill files to the Claude Code skills directory
# at ~/.claude/skills/adoctl/
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_NAME="adoctl"
SKILLS_DIR="${HOME}/.claude/skills"
TARGET_DIR="${SKILLS_DIR}/${SKILL_NAME}"

echo "Installing adoctl Claude Code skills..."
echo ""

# Check if running from the correct directory
if [ ! -f "${SCRIPT_DIR}/SKILL.md" ] && [ ! -f "${SCRIPT_DIR}/docs/skill/SKILL.md" ]; then
    # Try to find skill file in common locations
    if [ -f "${SCRIPT_DIR}/cmd/adoctl/main.go" ]; then
        # Running from repo root, check if skill exists in docs/skill directory
        if [ -f "${SCRIPT_DIR}/docs/skill/SKILL.md" ]; then
            SOURCE_DIR="${SCRIPT_DIR}/docs/skill"
        else
            echo "Error: SKILL.md not found in expected locations"
            echo "Please run this script from the adoctl repository root"
            exit 1
        fi
    else
        echo "Error: Cannot find adoctl repository structure"
        echo "Please run this script from the adoctl repository root"
        exit 1
    fi
else
    # Determine source directory
    if [ -f "${SCRIPT_DIR}/SKILL.md" ]; then
        SOURCE_DIR="${SCRIPT_DIR}"
    else
        SOURCE_DIR="${SCRIPT_DIR}/docs/skill"
    fi
fi

# Create skills directory if it doesn't exist
mkdir -p "${SKILLS_DIR}"

# Check if skill already exists
if [ -d "${TARGET_DIR}" ]; then
    echo "Skill '${SKILL_NAME}' already exists at:"
    echo "  ${TARGET_DIR}"
    echo ""
    read -p "Do you want to overwrite it? (y/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Installation cancelled."
        exit 0
    fi
    rm -rf "${TARGET_DIR}"
fi

# Copy skill files
mkdir -p "${TARGET_DIR}"

if [ -f "${SOURCE_DIR}/SKILL.md" ]; then
    cp "${SOURCE_DIR}/SKILL.md" "${TARGET_DIR}/"
    echo "✓ Installed SKILL.md"
fi

# Check for additional skill files
for file in skill.yaml skill.json; do
    if [ -f "${SOURCE_DIR}/${file}" ]; then
        cp "${SOURCE_DIR}/${file}" "${TARGET_DIR}/"
        echo "✓ Installed ${file}"
    fi
done

echo ""
echo "Successfully installed adoctl skills to:"
echo "  ${TARGET_DIR}"
echo ""
echo "The skill is now available in Claude Code. You can use it by asking:"
echo "  'How do I use adoctl?'"
echo "  'Show me adoctl commands'"
echo "  'Help me create a PR with adoctl'"
echo ""

# Option to create symlink for development
read -p "Create symlink for development (updates automatically)? (y/N) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "${TARGET_DIR}"
    ln -s "${SOURCE_DIR}" "${TARGET_DIR}"
    echo "✓ Created symlink for development"
    echo "  Changes to the skill files will be reflected immediately"
fi

echo ""
echo "Done!"
