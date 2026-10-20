#!/usr/bin/env bash

# Generates release notes from the commits since the previous release.
#
# Branching model:
#   v1     -> v1.x.y releases
#   master -> v2.x.y releases, including pre-releases such as v2.0.0-pre12
#
# The previous release is the highest tag of the same major version that is
# reachable from the release, so v1 and v2 releases never mix even though
# both lines are tagged in the same repository.

set -euo pipefail

version_pattern='^v([0-9]+)\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'

function usage() {
  echo "Usage: $0 [OPTIONS]"
  echo "Generate release notes from the commits since the previous release of the same major version."
  echo
  echo "Options:"
  echo "  -o, --output FILE  Write release notes to FILE (default: release_notes.txt)"
  echo "  -h, --help         Display this help message and exit"
  echo
  echo "If HEAD is tagged, the notes describe that release. Otherwise they describe"
  echo "the commits since the latest release on the current branch."
}

notes_file="release_notes.txt"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -o|--output)
      notes_file="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Error: Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Error: Not in a git repository" >&2
  exit 1
fi

# Prints the highest vMAJOR.* tag reachable from the given ref. Pre-releases
# sort before the release they precede, e.g. v2.0.0-pre11 < v2.0.0.
function latest_release_tag() {
  local major="$1" ref="$2"
  local tags

  tags=$(git -c versionsort.suffix=- tag --merged "$ref" --list "v${major}.*" --sort=-v:refname)
  echo "${tags%%$'\n'*}"
}

current_tag=$(git describe --exact-match --tags --match 'v[0-9]*' HEAD 2>/dev/null || true)

if [ -n "$current_tag" ]; then
  echo "Current HEAD is at tag: $current_tag"
  release_name="$current_tag"
  version="$current_tag"
  # The previous release is the latest one before this commit
  search_ref="HEAD^"
else
  echo "Current HEAD is not tagged (preparing notes for upcoming release)"
  release_name="upcoming release"
  version=$(git describe --tags --abbrev=0 --match 'v[0-9]*' HEAD 2>/dev/null || true)
  search_ref="HEAD"
fi

if [[ ! $version =~ $version_pattern ]]; then
  echo "Error: cannot determine the major version from '${version}'" >&2
  exit 1
fi
major="${BASH_REMATCH[1]}"

previous_tag=""
if git rev-parse --quiet --verify "$search_ref" >/dev/null; then
  previous_tag=$(latest_release_tag "$major" "$search_ref")
fi

if [ -n "$previous_tag" ]; then
  echo "Previous v${major} release: $previous_tag"
  range="$previous_tag..HEAD"
  since="$previous_tag"
else
  echo "No previous v${major} release found, using the whole history"
  range="HEAD"
  since="the first commit"
fi

commits=$(git log --no-merges --pretty=format:"* %s" "$range")

if [ -z "$commits" ]; then
  echo "No changes since $since" > "$notes_file"
  echo "Warning: No changes detected since $since" >&2
else
  commit_count=$(echo "$commits" | wc -l | tr -d ' ')

  {
    echo "Release Notes ($release_name):"
    echo ""
    echo "Changes since $since ($commit_count commits):"
    echo ""
    echo "$commits"
  } > "$notes_file"

  echo "Created release notes for $commit_count commits since $since"
fi

# Expose the path to the release notes file to later steps in GitHub Actions
if [ -n "${GITHUB_ENV:-}" ]; then
  echo "RELEASE_NOTES_PATH=$notes_file" >> "$GITHUB_ENV"
fi
