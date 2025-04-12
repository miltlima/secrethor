#!/bin/bash
set -euo pipefail

git fetch --tags

latest_tag=$(git tag --sort=-v:refname | grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+$' | head -n 1 || echo "v0.0.0")
clean_tag="${latest_tag#v}"

commits=$(git log "$latest_tag"..HEAD --pretty=format:"%s%n%b")

increment="patch"
if echo "$commits" | grep -qE "^feat(\(.+\))?: "; then
  increment="minor"
fi
if echo "$commits" | grep -qE "BREAKING CHANGE:"; then
  increment="major"
fi

IFS='.' read -r major minor patch <<< "$clean_tag"

case "$increment" in
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

next_version="v$major.$minor.$patch"

echo "🔖 Próxima versão: $next_version (detected as $increment)"
echo "RELEASE_VERSION=$next_version" >> "$GITHUB_ENV"
echo "next_version=$next_version" >> "$GITHUB_OUTPUT"