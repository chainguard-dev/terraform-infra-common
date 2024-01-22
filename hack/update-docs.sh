#!/bin/bash
set -e

# Update terraform docs
for d in `find . -name '*.tf' -exec dirname {} \; | sort | uniq`; do
    echo "###############################################"
    echo "# Generating $d"
    terraform-docs markdown table \
        --output-file README.md \
        --output-mode inject \
        $d
done
