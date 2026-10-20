#!/usr/bin/env bash

# Creates a release commit and tag on the current branch.
#
# Branching model:
#   v1     -> v1.x.y releases
#   master -> v2.x.y releases, including pre-releases such as v2.0.0-pre12
#
# The script bumps the chart version, regenerates the chart README, commits
# the chart changes and creates the tag. It does not push anything.

set -euo pipefail

version_pattern='^v([0-9]+)\.([0-9]+)\.([0-9]+)(-([0-9A-Za-z.-]+))?$'

function ask_confirmation() {
  local prompt="$1"
  local yn

  while true; do
    read -r -p "$prompt [y/n]: " yn
    case $yn in
      [Yy]* ) return 0;;
      [Nn]* ) return 1;;
      * ) echo "Please answer y or n.";;
    esac
  done
}

# Prints the major version released from the given branch
function major_for_branch() {
  case "$1" in
    v1) echo 1;;
    master) echo 2;;
    *) return 1;;
  esac
}

# Prints the highest vMAJOR.* tag reachable from HEAD. Pre-releases sort
# before the release they precede, e.g. v2.0.0-pre11 < v2.0.0.
function latest_release_tag() {
  local major="$1"
  local tags

  tags=$(git -c versionsort.suffix=- tag --merged HEAD --list "v${major}.*" --sort=-v:refname)
  echo "${tags%%$'\n'*}"
}

# Compares two pre-release identifiers such as pre9 and pre11, numerically
# when they share a prefix. Returns success if $1 > $2.
function prerelease_gt() {
  local a="$1" b="$2"
  local number_pattern='^([^0-9]*)([0-9]+)$'

  if [[ $a =~ $number_pattern ]]; then
    local a_prefix="${BASH_REMATCH[1]}" a_number="${BASH_REMATCH[2]}"
    if [[ $b =~ $number_pattern ]] && [[ ${BASH_REMATCH[1]} == "$a_prefix" ]]; then
      [ "$((10#$a_number))" -gt "$((10#${BASH_REMATCH[2]}))" ]
      return
    fi
  fi

  [[ $a > $b ]]
}

# Returns success if version $1 is greater than version $2
function version_gt() {
  [[ $1 =~ $version_pattern ]] || return 1
  local a_major="${BASH_REMATCH[1]}" a_minor="${BASH_REMATCH[2]}" a_patch="${BASH_REMATCH[3]}" a_pre="${BASH_REMATCH[5]}"
  [[ $2 =~ $version_pattern ]] || return 0
  local b_major="${BASH_REMATCH[1]}" b_minor="${BASH_REMATCH[2]}" b_patch="${BASH_REMATCH[3]}" b_pre="${BASH_REMATCH[5]}"

  if [ "$a_major" -ne "$b_major" ]; then [ "$a_major" -gt "$b_major" ]; return; fi
  if [ "$a_minor" -ne "$b_minor" ]; then [ "$a_minor" -gt "$b_minor" ]; return; fi
  if [ "$a_patch" -ne "$b_patch" ]; then [ "$a_patch" -gt "$b_patch" ]; return; fi

  # A release is greater than its pre-releases
  if [ -z "$a_pre" ]; then [ -n "$b_pre" ]; return; fi
  if [ -z "$b_pre" ]; then return 1; fi
  prerelease_gt "$a_pre" "$b_pre"
}

if ! command -v yq >/dev/null 2>&1; then
  echo "yq is required to update chart/Chart.yaml: https://github.com/mikefarah/yq"
  exit 1
fi

if [ ! -d helm-generator ]; then
  echo "The Helm README generator is not installed. Run ./scripts/install_helm_readme_generator.sh first."
  exit 1
fi

git_status=$(git status --porcelain)
if [[ -n "$git_status" ]]; then
  echo "You have unsaved changes:"
  echo "$git_status"
  ask_confirmation "You have unsaved changes. Do you want to proceed?" || exit
fi

if ! branch=$(git symbolic-ref --quiet --short HEAD); then
  echo "HEAD is detached. Check out the v1 or master branch to release."
  exit 1
fi

if major=$(major_for_branch "$branch"); then
  echo "Releasing v${major}.x.y from the $branch branch"
else
  echo "Releases are made from the v1 branch (v1.x.y) or the master branch (v2.x.y), but you are on $branch."
  ask_confirmation "Do you want to continue anyway?" || exit
  major=""
fi

if ! git fetch --tags --quiet; then
  echo "Warning: could not fetch tags, the latest tag may be out of date."
fi

if [ -n "$major" ]; then
  latest_tag=$(latest_release_tag "$major")
  echo "Latest v${major} tag on $branch: ${latest_tag:-none}"
else
  latest_tag=""
fi

read -r -p "Enter the version to release: " release_version

if [[ ! $release_version =~ $version_pattern ]]; then
  echo "Version $release_version does not match the format vX.Y.Z or vX.Y.Z-suffix, e.g. v1.9.2 or v2.0.0-pre12."
  exit 1
fi

if [ -n "$major" ] && [ "${BASH_REMATCH[1]}" != "$major" ]; then
  echo "The $branch branch releases v${major}.x.y versions, not $release_version."
  exit 1
fi

if git rev-parse --quiet --verify "refs/tags/$release_version" >/dev/null; then
  echo "Tag $release_version already exists."
  exit 1
fi

if [ -n "$latest_tag" ] && ! version_gt "$release_version" "$latest_tag"; then
  echo "Version $release_version is not greater than the latest tag $latest_tag."
  ask_confirmation "Do you want to continue anyway?" || exit
fi

chart_version=${release_version#v}

yq eval ".version = \"$chart_version\" | .appVersion = \"$chart_version\"" chart/Chart.yaml -i
./scripts/generate_helm_readme.sh

git diff
ask_confirmation "Do you want to proceed with the above changes?" || exit

git add chart
git commit -m "Release $release_version"
git tag "$release_version"

echo
echo "Created the release commit and tag $release_version. To publish the release, run:"
echo "  git push origin $branch $release_version"
