#!/usr/bin/env bash

# Copyright 2025 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

set -o errexit

# Optional path to a terraform-docs config, via the first arg or the
# TF_DOCS_CONFIG env var. When unset, the --config flag is not passed at all.
config="${1:-${TF_DOCS_CONFIG:-}}"

config_args=()
if [[ -n "${config}" ]]; then
  # Resolve to an absolute path so it works regardless of the module dir we run in.
  config_dir="$(dirname "${config}")"
  config_base="$(basename "${config}")"
  config_dir="$(cd "${config_dir}" && pwd)"
  config="${config_dir}/${config_base}"
  if [[ ! -f "${config}" ]]; then
    echo "Error: terraform-docs config '${config}' not found." >&2
    exit 1
  fi
  config_args=(--config="${config}")
fi

# Find all directories containing .tf files
directories=$(find . -name '*.tf' -not -path "./.*" -exec dirname {} \;)

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
        "${config_args[@]}" \
        "${d}"
done
