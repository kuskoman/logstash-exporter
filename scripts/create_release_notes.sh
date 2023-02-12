#!/bin/bash

# Get the latest tag
last_tag=$(git describe --abbrev=0 --tags)

# Get the commit messages since the last tag
commits=$(git log $last_tag..HEAD --pretty=format:"%s")

notes_file="release_notes.txt"

if [ -z "$commits" ]; then
  echo "No changes from previous release" >> "$notes_file"
else
  # Write the release notes to a file
  echo "Release Notes:" > "$notes_file"
  echo "" >> "$notes_file"
  echo "Since the last tag ($last_tag), the following changes have been made:" >> "$notes_file"
  echo "" >> "$notes_file"
  echo "$commits" >> "$notes_file"
fi
