#!/usr/bin/env bash

set -euo pipefail

function ask_confirmation() {
  local prompt="$1"
  local yn

  while true; do
    read -pr "$prompt [y/n]: " yn
    case $yn in
      [Yy]* ) return 0;;
      [Nn]* ) return 1;;
      * ) echo "Please answer y or n.";;
    esac
  done
}

git_status=$(git status --porcelain)
if [[ -n "$git_status" ]]; then
  echo "You have unsaved changes:"
  echo "$git_status"
  ask_confirmation "You have unsaved changes. Do you want to proceed?" || exit
fi

latest_tag=$(git describe --tags "$(git rev-list --tags --max-count=1)")
echo "Latest tag on the given branch: $latest_tag"
read -pr "Enter the version to release: " release_version

if [[ $release_version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  chart_version=${release_version:1}
else
  echo "Warning: Version does not match the format vX.Y.Z."
  ask_confirmation "Do you want to continue?" || exit
  chart_version=$release_version
fi

yq eval ".version = \"$chart_version\" | .appVersion = \"$chart_version\"" chart/Chart.yaml -i
sed -i "/^## @param image.tag\s*$/,/^\s*tag:\s*\"[^\"]*\"\s*$/s/\(^\s*tag:\s*\).*\$/\1\"$release_version\"/" chart/values.yaml
./scripts/generate_helm_readme.sh

git diff
ask_confirmation "Do you want to proceed with the above changes?" || exit

git add .
git commit -m "Release $release_version"
git tag "$release_version"
