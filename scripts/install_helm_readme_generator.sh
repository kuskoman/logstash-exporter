#!/bin/bash

set -euo pipefail

destination_dir="helm-generator"

if [ -d "$destination_dir" ]; then
  echo "Directory $destination_dir already exists. Skipping installation."
  echo "If you want to reinstall, please remove the directory first."
  exit 0
fi

git clone https://github.com/bitnami-labs/readme-generator-for-helm "$destination_dir"
cd "$destination_dir"
npm install
npm run-script test
