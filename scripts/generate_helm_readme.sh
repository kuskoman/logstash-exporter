#!/bin/bash

set -euo pipefail

generator_dir="helm-generator"
generator_script="$PWD/$generator_dir/bin/index.js"

if [ ! -d "$generator_dir" ]; then
  echo "Directory $generator_dir does not exist. Please run install_helm_readme_generator.sh first."
  exit 1
fi

$generator_script \
  -s "./chart/schema.json" \
  -v "./chart/values.yaml" \
  -r "./chart/README.md"
