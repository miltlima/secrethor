#!/bin/bash
# script to calculate the next version based on commit messages
# It uses semantic versioning rules to determine if the next version should be a major, minor, or patch release
# It also handles the case where the latest tag is not in the format vX.Y.Z
# It assumes that the script is run in a git repository and that the latest tag is already fetched
# It also assumes that the script is run in a GitHub Actions workflow

# Calculates the next semantic version based on Conventional Commits.
#
# Commit type → Version bump mapping:
# - feat:                => MINOR
# - fix:                 => PATCH
# - chore:, refactor:, docs:, style:, test:, perf:, ci:, build:, revert:
#                        => PATCH
# - BREAKING CHANGE:     => MAJOR (even if commit is feat or fix)

set -euo pipefail

git fetch --tags

latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
clean_tag="${latest_tag#v}"

commits=$(git log "$latest_tag"..HEAD --pretty=format:"%s%n%b" || true )

if [ -z "$commits" ]; then
  echo "🚫 No new commits since $latest_tag. Skipping release."
  echo "SKIP_RELEASE=true" >> "$GITHUB_ENV"
  exit 0
fi

bump="patch"
if echo "$commits" | grep -qE "^feat(\(.+\))?: "; then
  bump="minor"
fi
if echo "$commits" | grep -qE "BREAKING CHANGE:"; then
  bump="major"
fi

IFS='.' read -r major minor patch <<< "$clean_tag"

case "$bump" in
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  patch)
    patch=$((patch + 1))
    ;;
esac

while git ls-remote --tags origin | grep -q "refs/tags/v$major.$minor.$patch"; do
  echo "⚠️ Tag v$major.$minor.$patch already exists. Incrementing..."
  patch=$((patch + 1))
done

next_version="$major.$minor.$patch"

echo "🔖 Próxima versão: v$next_version (detected as $bump)"
echo "RELEASE_VERSION=$next_version" >> "$GITHUB_ENV"
echo "next_version=$next_version" >> "$GITHUB_OUTPUT"