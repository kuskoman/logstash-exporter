#!/bin/bash
set -eo pipefail

# Print usage information
function usage() {
  echo "Usage: $0 [OPTIONS]"
  echo "Generate release notes based on git commits since the previous tag on the current branch."
  echo
  echo "Options:"
  echo "  -o, --output FILE  Write release notes to FILE (default: release_notes.txt)"
  echo "  -h, --help         Display this help message and exit"
  echo
  echo "The script will find the most recent tag on the current branch and list"
  echo "all non-merge commits made since that tag."
}

# Parse command line arguments
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

# Ensure we're in a git repository
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Error: Not in a git repository" >&2
  exit 1
fi

# Check if HEAD is tagged
function is_head_tagged() {
  git describe --exact-match --tags HEAD 2>/dev/null 1>&2
}

# Determine current tag if we're on a tag
if is_head_tagged; then
  current_tag=$(git describe --exact-match --tags HEAD)
  echo "Current HEAD is at tag: $current_tag"
else
  current_tag="upcoming release"
  echo "Current HEAD is not tagged (preparing notes for upcoming release)"
fi

# Find the most recent tag on the current branch
# This gets all tags reachable from the current HEAD
previous_tag=$(git describe --abbrev=0 --tags --first-parent 2>/dev/null)

# If we're on a tag now, we need to find the previous tag
if is_head_tagged; then
  # Get all tags on the current branch, sorted by commit date
  all_tags_on_branch=$(git tag --sort=-committerdate --merged HEAD)
  
  # The first tag is the current one, so we need to get the second tag
  previous_tag=$(echo "$all_tags_on_branch" | grep -v "$current_tag" | head -n 1)
  
  # If no previous tag found, get the earliest commit as reference
  if [ -z "$previous_tag" ]; then
    echo "No previous tag found, using the first commit as reference"
    previous_tag=$(git rev-list --max-parents=0 HEAD)
  fi
fi

echo "Previous tag found: $previous_tag"

# Set the range for commit log
range="$previous_tag..HEAD"

# Get commits between previous tag and HEAD
commits_since_previous_tag=$(git log --no-merges --pretty=format:"* %s" "$range")

# Generate release notes
if [ -z "$commits_since_previous_tag" ]; then
  echo "No changes since previous tag ($previous_tag)" > "$notes_file"
  echo "Warning: No changes detected since previous tag" >&2
else
  # Count the commits for a summary
  commit_count=$(echo "$commits_since_previous_tag" | wc -l | tr -d ' ')
  
  # Start release notes with header
  if is_head_tagged; then
    echo "# Release Notes for $current_tag" > "$notes_file"
  else
    echo "# Release Notes for upcoming release" > "$notes_file"
  fi
  
  {
    echo ""
    echo "## Changes since $previous_tag ($commit_count commits)"
    echo ""
    echo "$commits_since_previous_tag"
    echo ""
    echo "---"
    echo "Generated on $(date '+%Y-%m-%d %H:%M:%S')"
  } >> "$notes_file"
  
  echo "Successfully created release notes for $commit_count commits since $previous_tag"
fi

# Output the path to the release notes file for potential use in a CI pipeline
echo "RELEASE_NOTES_PATH=$notes_file" >> "${GITHUB_ENV:-/dev/null}"
