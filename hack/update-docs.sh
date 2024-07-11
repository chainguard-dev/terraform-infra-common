#!/usr/bin/env bash

set -o errexit

# Find all directories containing .tf files
directories=$(find . -name '*.tf' -exec dirname {} \;)

# Check if the find command found any directories
if [[ -z "${directories}" ]]; then
  echo "No .tf files found."
  exit 0
fi

sorted=$(echo "${directories}" | sort)
sorted=$(echo "${sorted}" | uniq)

# Update terraform docs
for d in ${sorted}; do
    echo "###############################################"
    echo "# Generating ${d}"
    terraform-docs markdown table \
        --output-file README.md \
        --output-mode inject \
        --lockfile=false \
        "${d}"
done
