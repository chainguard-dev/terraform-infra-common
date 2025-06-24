#!/usr/bin/env python3
"""
Static analysis of Terraform HCL files to check for missing labels
"""
import argparse
import glob
import json
import os
import sys
from itertools import chain
from pathlib import Path
from pprint import pformat

import hcl2


def load_label_supporting_resources_from_schema(schema_path):
    """
    Load provider schema and extract resources that support labels.
    Equivalent to: jq '.provider_schemas."registry.terraform.io/hashicorp/google".resource_schemas
                      | to_entries[]
                      | select(.value.block.attributes.labels != null or .value.block.attributes.resource_labels != null)
                      | .key'
    """
    with open(schema_path, 'r') as f:
        schema_data = json.load(f)

    # Navigate to the Google provider resource schemas
    provider_schemas = schema_data.get("provider_schemas", {})
    google_provider = None

    # Look for Google provider (try different possible keys)
    possible_keys = [
        "registry.terraform.io/hashicorp/google",
        "hashicorp/google",
        "google"
    ]

    for key in possible_keys:
        if key in provider_schemas:
            google_provider = provider_schemas[key]
            break

    if not google_provider:
        print("❌ Google provider not found in schema. Available providers:")
        for key in provider_schemas.keys():
            print(f"  • {key}")
        return

    resource_schemas = google_provider.get("resource_schemas", {})

    for resource_type, resource_schema in resource_schemas.items():
        # Check if resource has labels or resource_labels attributes
        block = resource_schema.get("block", {})
        attributes = block.get("attributes", {})

        if "labels" in attributes or "resource_labels" in attributes:
            yield resource_type



def parse_terraform_file(file_path, label_supporting_resources):
    """Parse a single Terraform file and return violations"""
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    # Skip empty files
    if not content.strip():
        return

    parsed = hcl2.loads(content)

    # Check resources
    resource_list = parsed.get('resource', [])

    for resources in resource_list:
        for resource_type, resource_instances in resources.items():
            if resource_type in label_supporting_resources:
                for resource_name, resource_config in resource_instances.items():
                    # Check for labels or resource_labels
                    has_labels = (
                        'labels' in resource_config and resource_config['labels'] or
                        'resource_labels' in resource_config and resource_config['resource_labels']
                    )

                    if not has_labels:
                        yield {
                            "file": str(file_path),
                            "resource_type": resource_type,
                            "resource_name": resource_name,
                            "address": f"{resource_type}.{resource_name}",
                            "line_number": None  # HCL2 parser doesn't provide line numbers
                        }


def find_terraform_files(directory="."):
    """Find all .tf files recursively"""
    # Filter out common exclusions
    exclusions = {'.terraform', 'terraform.tfstate', 'terraform.tfstate.backup'}

    tf_files = chain.from_iterable(
        glob.glob(os.path.join(directory, pattern), recursive=True)
        for pattern in ["**/*.tf", "**/*.tf.json"]
    )
    for file_path in tf_files:
        if not any(exclusion in file_path for exclusion in exclusions):
            yield file_path


def generate_label_suggestions(violations):
    """Generate suggestions for adding labels"""
    suggestions = []

    for violation in violations[:10]:  # Show first 10 as examples
        resource_type = violation["resource_type"]
        resource_name = violation["resource_name"]

        # Different resources might use different label attributes
        label_attr = "resource_labels" if resource_type in {
            "google_cloudfunctions_function",
            "google_cloudfunctions2_function"
        } else "labels"

        suggestion = f"""
# In {violation['file']}
# Add to resource "{resource_type}" "{resource_name}":

  {label_attr} = {{
    environment = var.environment
    project     = var.project_name
    team        = var.team
    managed_by  = "terraform"
  }}
"""
        suggestions.append(suggestion)

    return suggestions

def analyze_terraform_files(label_supporting_resources, directory="."):
    """Main analysis function"""

    print(f"🔍 Analyzing Terraform files in {os.path.abspath(directory)}...")
    label_supporting_resources = frozenset(label_supporting_resources)
    print(f"📋 Checking {len(label_supporting_resources)} resource types for labels")

    tf_files = list(find_terraform_files(directory))

    if not tf_files:
        print("❌ No Terraform files found")
        return False

    print(f"📁 Found {len(tf_files)} Terraform files")

    all_violations = []
    files_with_violations = 0

    for tf_file in tf_files:
        violations = list(parse_terraform_file(tf_file, label_supporting_resources))
        if violations:
            all_violations.extend(violations)
            files_with_violations += 1

    # Summary by resource type
    violations_by_type = {}
    violations_by_file = {}

    for violation in all_violations:
        resource_type = violation["resource_type"]
        file_path = violation["file"]

        if resource_type not in violations_by_type:
            violations_by_type[resource_type] = []
        violations_by_type[resource_type].append(violation)

        if file_path not in violations_by_file:
            violations_by_file[file_path] = []
        violations_by_file[file_path].append(violation)

    # Print results
    if all_violations:
        print(f"\n❌ Found {len(all_violations)} resources missing labels in {files_with_violations} files")

        print(f"\n📊 Violations by resource type:")
        for resource_type, violations in sorted(violations_by_type.items()):
            print(f"  • {resource_type}: {len(violations)} resources")

        print(f"\n📁 Violations by file:")
        for file_path, violations in sorted(violations_by_file.items()):
            print(f"\n  {file_path} ({len(violations)} violations):")
            for violation in violations:
                print(f"    - {violation['address']}")

        print(f"\n💡 Example label configurations:")
        suggestions = generate_label_suggestions(all_violations)
        for suggestion in suggestions[:3]:  # Show first 3
            print(suggestion)

        if len(suggestions) > 3:
            print(f"... and {len(suggestions) - 3} more suggestions")

        return False
    else:
        print(f"\n✅ All {len(label_supporting_resources)} label-supporting resource types have labels!")
        return True


def list_supported_resources(label_supporting_resources):
    """List all resources that should have labels"""
    print("📋 Resources that should have labels:")
    for resource_type in sorted(label_supporting_resources):
        print(f"  • {resource_type}")



def main():
    parser = argparse.ArgumentParser(description="Check Terraform HCL files for missing labels")
    parser.add_argument("--schema", "-s", help="Path to provider-schema.json file")
    parser.add_argument("--directory", "-d", default=".", help="Directory to scan (default: current)")
    parser.add_argument("--list-resources", action="store_true", help="List supported resource types")

    args = parser.parse_args()

    # Load label-supporting resources from schema or use defaults
    if not args.schema:
        print("❌ No schema provided")
        sys.exit(2)

    print(f"📄 Loading resource schema from {args.schema}")
    label_supporting_resources = load_label_supporting_resources_from_schema(args.schema)

    if args.list_resources:
        list_supported_resources(label_supporting_resources)
        sys.exit(0)

    success = analyze_terraform_files(label_supporting_resources, args.directory)
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
