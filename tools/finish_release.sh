#!/usr/bin/env bash

set -e

VERSION=$1

REPO=siliconflow/siliconcloud-cli

if [[ "$VERSION" == *"-beta" ]]; then
  echo "beta. skip."
else
  RELEASE_ID=$(curl -fsSL -X GET -H "Content-Type: application/json" \
    -H "Authorization: token $GITHUB_TOKEN" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://api.github.com/repos/$REPO/releases/tags/v"$VERSION" | jq '.["id"]')

  echo "release id: $RELEASE_ID"

  # shellcheck disable=SC2125
  DATA={\"draft\":false,\"prerelease\":false,\"make_latest\":true}

  curl -fsSL \
    -X PATCH \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: token $GITHUB_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://api.github.com/repos/$REPO/releases/"$RELEASE_ID" \
    -d "$DATA"
fi