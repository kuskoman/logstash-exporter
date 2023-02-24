#!/bin/bash

last_tag=$(git describe --abbrev=0 --tags)

commits_since_last_tag=$(git log $last_tag..HEAD --no-merges --pretty=format:"%s")

notes_file="release_notes.txt"

if [ -z "$commits_since_last_tag" ]; then
  echo "No changes from previous release" >> "$notes_file"
else
  echo "Release Notes:" > "$notes_file"
  echo "" >> "$notes_file"
  echo "Since the last tag ($last_tag), the following changes have been made:" >> "$notes_file"
  echo "" >> "$notes_file"
  echo "$commits_since_last_tag" >> "$notes_file"
fi
