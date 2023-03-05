#!/bin/bash

function is_head_tagged() {
  git describe --exact-match --tags HEAD 2>/dev/null 1>&2
}

if is_head_tagged; then
  current_tag=$(git describe --abbrev=0 --tags HEAD)
  previous_tag_hash=$(git rev-list --tags --skip=1 --max-count=1)
else
  previous_tag_hash=$(git rev-list --tags --max-count=1)
fi

previous_tag=$(git describe --abbrev=0 --tags $previous_tag_hash)


range="$previous_tag..HEAD"

commits_since_previous_tag=$(git log --no-merges --pretty=format:"* %s" $range)

notes_file="release_notes.txt"

if [ -z "$commits_since_previous_tag" ]; then
  echo "No changes from previous release" > "$notes_file"
else
  echo "Release Notes ($current_tag):" > "$notes_file"
  echo "" >> "$notes_file"
  echo "Since the last tag ($previous_tag), the following changes have been made:" >> "$notes_file"
  echo "" >> "$notes_file"
  echo "$commits_since_previous_tag" >> "$notes_file"
fi
