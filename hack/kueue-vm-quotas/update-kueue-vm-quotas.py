#!/usr/bin/env python3
"""
Script to update Kueue ClusterQueue with VM quotas from multi-platform-controller host-config.yaml.

This script reads platform quotas from host-config.yaml and updates cluster-queue.yaml
with appropriate resource groups and quotas, respecting Kueue's constraints:
- Maximum 16 resource groups
- Maximum 16 resources per group
- Each resource group has a single flavor
- Each flavor is unique to one resource group

The script processes three types of platforms:
1. Dynamic platforms (from dynamic.*.max-instances keys)
2. Static platforms (from host.*.concurrency keys)
3. Local platforms (from local-platforms key, with fixed quota of 1000)

Usage:
    python3 update-kueue-vm-quotas.py host-config.yaml cluster-queue.yaml [--dry-run]

See README.md for detailed usage examples and workflow integration.
"""

import yaml
import sys
import argparse
from pathlib import Path
from typing import Dict, List, Any
from dataclasses import dataclass


@dataclass
class PlatformQuota:
    """Represents a platform and its quota."""
    name: str
    quota: int
    
    def __post_init__(self) -> None:
        if self.quota < 0:
            raise ValueError(f"Quota must be non-negative, got {self.quota}")


@dataclass
class ResourceSpec:
    """Represents a Kueue resource specification."""
    name: str
    nominal_quota: str


@dataclass
class ResourceGroup:
    """Represents a Kueue resource group."""
    covered_resources: List[str]
    resources: List[ResourceSpec]


def extract_dynamic_platform(key: str, value: str) -> PlatformQuota:
    """Extract platform quota from dynamic platform config entry."""
    # Extract platform name from key like "dynamic.linux-arm64.max-instances"
    platform_name = key.split('.')[1]
    quota = int(value)
    print(f"Dynamic platform: {platform_name} -> {quota}")
    return PlatformQuota(name=platform_name, quota=quota)


def extract_static_platform(key: str, value: str, data: Dict[str, Any]) -> PlatformQuota:
    """Extract platform quota from static platform config entry."""
    # Extract platform info from corresponding .platform key
    host_prefix = '.'.join(key.split('.')[:-1])  # Remove .concurrency
    platform_key = f"{host_prefix}.platform"

    platform_value = data[platform_key]  # e.g., "linux/s390x"
    platform_name = platform_value.replace('/', '-')  # Convert to "linux-s390x"
    quota = int(value)

    print(f"Static platform: {platform_name} -> {quota}")
    return PlatformQuota(name=platform_name, quota=quota)


def extract_local_platforms(value: str, quota: int = 1000) -> List[PlatformQuota]:
    platforms = []
    # Split by comma and clean up whitespace and empty strings
    platform_names = [name.strip() for name in values.split(',') if name.strip()]
    
    for platform_name in platform_names:
        # Convert platform names like "linux/amd64" to "linux-amd64"
        normalized_name = platform_name.replace('/', '-').replace('_', '-')
        print(f"Local platform: {normalized_name} -> {quota}")
        platforms.append(PlatformQuota(name=normalized_name, quota=quota))
    
    return platforms


def add_or_aggregate_platform(platform: PlatformQuota, platform_quotas: Dict[str, PlatformQuota]) -> None:
    """Add platform to quotas dictionary or aggregate if it already exists."""
    if platform.name in platform_quotas:
        existing = platform_quotas[platform.name]
        new_quota = existing.quota + platform.quota
        platform_quotas[platform.name] = PlatformQuota(platform.name, new_quota)
        print(f"Aggregated platform: {platform.name} -> {new_quota} (was {existing.quota})")
    else:
        platform_quotas[platform.name] = platform


def parse_host_config(host_config_path: str) -> Dict[str, PlatformQuota]:
    """Parse host-config.yaml and extract platform quotas."""
    with open(host_config_path, 'r') as f:
        config = yaml.safe_load(f)
    
    platform_quotas: Dict[str, PlatformQuota] = {}
    data = config.get('data', {})
    
    # Sort keys for deterministic processing order
    for key, value in sorted(data.items()):
        platform = None
        
        # Process dynamic platforms
        if key.startswith('dynamic.') and key.endswith('.max-instances'):
            platform = extract_dynamic_platform(key, value)

        # Process static platforms
        elif key.startswith('host.') and key.endswith('.concurrency'):
            platform_key = f"{'.'.join(key.split('.')[:-1])}.platform"
            if platform_key in data:
                platform = extract_static_platform(key, value, data)

        # Process local platforms
        elif key == 'local-platforms':
            local_platforms = extract_local_platforms(value)
            for local_platform in local_platforms:
                add_or_aggregate_platform(local_platform, platform_quotas)

        # Handle single platform
        if platform:
            add_or_aggregate_platform(platform, platform_quotas)
    
    return platform_quotas


def distribute_platforms(platforms: List[PlatformQuota], max_per_group: int = 16) -> List[List[PlatformQuota]]:
    """Distribute platforms across groups respecting size constraints."""
    groups: List[List[PlatformQuota]] = []
    current_group: List[PlatformQuota] = []
    
    for platform in sorted(platforms, key=lambda p: p.name):
        if len(current_group) >= max_per_group:
            groups.append(current_group)
            current_group = []
        current_group.append(platform)
    
    if current_group:
        groups.append(current_group)
    
    return groups


def create_platform_resource_group(platforms: List[PlatformQuota]) -> ResourceGroup:
    """Create a resource group for a list of platforms."""
    # Only include platform names in covered resources (no tekton.dev/pipelineruns)
    covered_resources = [p.name for p in sorted(platforms, key=lambda p: p.name)]
    
    # Create individual platform resources
    resources = [
        ResourceSpec(name=p.name, nominal_quota=str(p.quota))
        for p in sorted(platforms, key=lambda p: p.name)
    ]
    
    return ResourceGroup(covered_resources, resources)


def resource_group_to_dict(resource_group: ResourceGroup, flavor_name: str) -> Dict[str, Any]:
    """Convert ResourceGroup to dictionary with flavor."""
    return {
        'coveredResources': resource_group.covered_resources,
        'flavors': [{
            'name': flavor_name,
            'resources': [
                {'name': res.name, 'nominalQuota': res.nominal_quota}
                for res in resource_group.resources
            ]
        }]
    }


def create_resource_flavor(name: str) -> Dict[str, Any]:
    """Create a ResourceFlavor object."""
    return {
        'apiVersion': 'kueue.x-k8s.io/v1beta1',
        'kind': 'ResourceFlavor',
        'metadata': {'name': name},
        'spec': {}
    }


def find_document_by_kind(documents: List[Dict[str, Any]], kind: str) -> Dict[str, Any]:
    """Find document by kind, raising ValueError if not found."""
    for doc in documents:
        if doc and doc.get('kind') == kind:
            return doc
    raise ValueError(f"{kind} document not found")


def get_existing_flavor_names(documents: List[Dict[str, Any]]) -> List[str]:
    """Extract existing ResourceFlavor names from documents."""
    return [
        doc.get('metadata', {}).get('name')
        for doc in documents
        if doc and doc.get('kind') == 'ResourceFlavor' and doc.get('metadata', {}).get('name')
    ]


def preserve_base_resource_group(existing_groups: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """Find and preserve the base resource group containing tekton.dev/pipelineruns."""
    for group in existing_groups:
        if "tekton.dev/pipelineruns" in group.get('coveredResources', []):
            print("Preserved base resource group with basic resources")
            return [group]
    return []


def validate_constraints(resource_groups: List[Dict[str, Any]]) -> None:
    """Validate that all Kueue constraints are met."""
    if len(resource_groups) > 16:
        raise ValueError(f"Generated {len(resource_groups)} resource groups, maximum is 16")
    
    for i, group in enumerate(resource_groups):
        covered_count = len(group.get('coveredResources', []))
        if covered_count > 16:
            raise ValueError(f"Group {i+1} has {covered_count} covered resources, maximum is 16")


def print_summary(resource_groups: List[Dict[str, Any]], platform_count: int) -> None:
    """Print summary of the update operation."""
    base_groups = sum(1 for group in resource_groups if 'cpu' in group.get('coveredResources', []))
    platform_groups = len(resource_groups) - base_groups
    
    print(f"\nTotal resource groups: {len(resource_groups)}")
    print(f"Base resource groups: {base_groups}, Platform resource groups: {platform_groups}")
    print(f"Total platforms processed: {platform_count}")
    
    print(f"\nConstraint validation:")
    print(f"- Resource groups: {len(resource_groups)}/16 ✓")
    for i, group in enumerate(resource_groups):
        covered_count = len(group.get('coveredResources', []))
        print(f"- Group {i+1} covered resources: {covered_count}/16 ✓")


def process_cluster_queue_update(cluster_queue_path: str, platform_quotas: Dict[str, PlatformQuota]) -> None:
    """Process the cluster queue update with platform quotas."""
    # Read documents
    with open(cluster_queue_path, 'r') as f:
        documents = list(yaml.safe_load_all(f))
    
    # Find main documents
    cluster_queue_doc = find_document_by_kind(documents, 'ClusterQueue')
    existing_flavors = get_existing_flavor_names(documents)
    
    # Preserve base resource group
    existing_groups = cluster_queue_doc.get('spec', {}).get('resourceGroups', [])
    new_resource_groups = preserve_base_resource_group(existing_groups)
    
    # Create platform resource groups
    platform_list = list(platform_quotas.values())
    platform_groups = distribute_platforms(platform_list)
    
    for group_index, platforms in enumerate(platform_groups, start=1):
        flavor_name = f'platform-group-{group_index}'
        
        # Create ResourceFlavor if needed
        if flavor_name not in existing_flavors:
            documents.append(create_resource_flavor(flavor_name))
            print(f"Created ResourceFlavor: {flavor_name}")
        
        # Create resource group
        resource_group = create_platform_resource_group(platforms)
        group_dict = resource_group_to_dict(resource_group, flavor_name)
        new_resource_groups.append(group_dict)
        
        platform_names = [p.name for p in platforms]
        print(f"Created resource group {group_index} with {len(platforms)} platforms: {', '.join(platform_names)}")
    
    # Validate and update
    validate_constraints(new_resource_groups)
    cluster_queue_doc['spec']['resourceGroups'] = new_resource_groups
    
    # Sort documents for consistent ordering
    documents.sort(key=lambda doc: (
        doc.get('kind', ''),
        doc.get('metadata', {}).get('name', '') if doc.get('metadata') else ''
    ))
    
    # Write back with sorted keys for deterministic output
    with open(cluster_queue_path, 'w') as f:
        yaml.dump_all(documents, f, default_flow_style=False, sort_keys=True)
    
    print(f"\nUpdated {cluster_queue_path}")
    print_summary(new_resource_groups, len(platform_quotas))


def validate_file_paths(host_config_path: str, cluster_queue_path: str) -> None:
    """Validate that required files exist."""
    if not Path(host_config_path).exists():
        raise FileNotFoundError(f"Host config file not found: {host_config_path}")
    
    if not Path(cluster_queue_path).exists():
        raise FileNotFoundError(f"Cluster queue file not found: {cluster_queue_path}")


def main() -> None:
    """Main function to parse arguments and orchestrate the update."""
    parser = argparse.ArgumentParser(
        description='Update Kueue ClusterQueue with VM quotas from host-config.yaml'
    )
    parser.add_argument('host_config', help='Path to host-config.yaml file')
    parser.add_argument('cluster_queue', help='Path to cluster-queue.yaml file to update')
    parser.add_argument('--dry-run', action='store_true', help='Show what would be updated without making changes')
    
    args = parser.parse_args()
    
    try:
        validate_file_paths(args.host_config, args.cluster_queue)
        
        print(f"Reading host config from: {args.host_config}")
        print(f"Updating cluster queue: {args.cluster_queue}")
        print()
        
        # Parse platform quotas
        platform_quotas = parse_host_config(args.host_config)
        
        print(f"\nFound {len(platform_quotas)} platforms:")
        for name, quota in sorted(platform_quotas.items()):
            print(f"  {name}: {quota.quota}")
        
        if args.dry_run:
            print("\nDry run mode - no changes will be made")
            return
        
        # Update cluster queue
        print(f"\nUpdating cluster queue...")
        process_cluster_queue_update(args.cluster_queue, platform_quotas)
        print("Done!")
        
    except (FileNotFoundError, yaml.YAMLError, ValueError, IOError) as e:
        print(f"ERROR: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main() 
