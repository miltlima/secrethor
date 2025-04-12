#!/bin/bash
set -e

LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
if [ "$LAST_TAG" = "v0.0.0" ]; then
    echo "No tags found. Defaulting to v0.0.0."
fi

echo "Última tag: $LAST_TAG"

COMMITS=$(git log "$LAST_TAG"..HEAD --pretty=format:%s)

MAJOR=false
MINOR=false

while IFS= read -r COMMIT; do
  if echo "$COMMIT" | grep -q "BREAKING CHANGE"; then
    MAJOR=true
    break
  elif echo "$COMMIT" | grep -q "^feat"; then
    MINOR=true
  fi
done <<< "$COMMITS"

VERSION=${LAST_TAG#v}
IFS='.' read -r MAJOR_V MINOR_V PATCH_V <<< "$VERSION"

if $MAJOR; then
  ((MAJOR_V++))
  MINOR_V=0
  PATCH_V=0
elif $MINOR; then
  ((MINOR_V++))
  PATCH_V=0
else
  ((PATCH_V++))
fi

NEW_VERSION="v${MAJOR_V}.${MINOR_V}.${PATCH_V}"
echo "Nova versão: $NEW_VERSION"

echo "VERSION=$NEW_VERSION" >> "$GITHUB_ENV"
echo "version=$NEW_VERSION" >> "$GITHUB_OUTPUT"