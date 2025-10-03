#!/bin/bash -e
# This script generates Kueue cluster queue configuration files for VM quotas
# by processing host configuration files and invoking the update-kueue-vm-quotas.py
# script with the appropriate input and output paths.
#
# The script can generate host-config.yaml files on-the-fly using helm template
# if they don't exist, using the corresponding host-values.yaml file.
# Generated files are automatically cleaned up after processing.
# For backward compatibility, existing host-config.yaml files are used if present.
#
# Usage: generate-queue-config.sh [--verify-no-change]
#   --verify-no-change: Verify that no changes were made to the output files

declare -r ROOT="${BASH_SOURCE[0]%/*}"

usage() {
    echo "Usage: $0 [--verify-no-change]"
    echo "  --verify-no-change: Verify that no changes were made to the output files"
    exit 1
}

generate_host_config() {
    local input_file="$1"
    local input_dir="${input_file%/*}"
    local host_values_file="$input_dir/host-values.yaml"
    

    # Check if host-values.yaml exists for helm template generation
    if [[ ! -f "$host_values_file" ]]; then
        echo "ERROR: Neither $input_file nor $host_values_file exists"
        return 1
    fi
    
    # Determine the relative path to the base chart
    # Since we know the full path, we can calculate relative path directly
    # Extract the part after components/multi-platform-controller/
    local subpath="${input_dir#*components/multi-platform-controller/}"
    
    # Count directory levels to determine how many "../" we need
    # If subpath is empty, we're directly in multi-platform-controller (depth=0)
    # Otherwise, count slashes + 1 for the number of directory levels
    local depth
    if [[ -z "$subpath" ]]; then
        depth=0
    else
        depth=$(echo "$subpath" | tr -cd '/' | wc -c)
        depth=$((depth + 1))  # Add 1 because we're in at least one subdirectory
    fi
    
    
    # Build relative path to base
    local relative_base="base/host-config-chart"
    for ((i=0; i<depth; i++)); do
        relative_base="../$relative_base"
    done
    
    echo "Generating host-config.yaml using helm template: $input_file"
    echo "  Base path: $relative_base"
    echo "  Values file: $host_values_file"
    
    # Generate host-config.yaml using helm template
    (
        cd "$input_dir"
        helm template host-config "$relative_base" \
            --namespace multi-platform-controller \
            -f "$(basename "$host_values_file")" > "$(basename "$input_file")"
    )
    
    if [[ $? -ne 0 ]]; then
        echo "ERROR: Failed to generate $input_file using helm template"
        return 1
    fi
    
    echo "Successfully generated: $input_file"
    return 0  # Return 0 to indicate file was successfully generated
}

main() {
    local verify_no_change=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --verify-no-change)
                verify_no_change=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                echo "Unknown argument: $1"
                usage
                ;;
        esac
    done

    local cli="hack/kueue-vm-quotas/update-kueue-vm-quotas.py"
    
    # Define input-output file pairs
    local -A queue_configs=(
        ["components/multi-platform-controller/staging/host-config.yaml"]="components/kueue/staging/stone-stg-rh01/queue-config/cluster-queue.yaml"

        ["components/multi-platform-controller/staging-downstream/host-config.yaml"]="components/kueue/staging/stone-stage-p01/queue-config/cluster-queue.yaml"

        ["components/multi-platform-controller/production/kflux-prd-rh02/host-config.yaml"]="components/kueue/production/kflux-prd-rh02/queue-config/cluster-queue.yaml"

        ["components/multi-platform-controller/production/kflux-prd-rh03/host-config.yaml"]="components/kueue/production/kflux-prd-rh03/queue-config/cluster-queue.yaml"

        ["components/multi-platform-controller/production-downstream/kflux-ocp-p01/host-config.yaml"]="components/kueue/production/kflux-ocp-p01/queue-config/cluster-queue.yaml"

        ["components/multi-platform-controller/production-downstream/kflux-rhel-p01/host-config.yaml"]="components/kueue/production/kflux-rhel-p01/queue-config/cluster-queue.yaml"

        ["components/multi-platform-controller/production-downstream/kflux-osp-p01/host-config.yaml"]="components/kueue/production/kflux-osp-p01/queue-config/cluster-queue.yaml"

        ["components/multi-platform-controller/production-downstream/stone-prod-p01/host-config.yaml"]="components/kueue/production/stone-prod-p01/queue-config/cluster-queue.yaml"

        ["components/multi-platform-controller/production-downstream/stone-prod-p02/host-config.yaml"]="components/kueue/production/stone-prod-p02/queue-config/cluster-queue.yaml"

        ["components/multi-platform-controller/production/stone-prd-rh01/host-config.yaml"]="components/kueue/production/stone-prd-rh01/queue-config/cluster-queue.yaml"
    )
    
    # Track generated files for cleanup
    local generated_files=()
    
    # Generate queue configurations
    for input_file in "${!queue_configs[@]}"; do
        local output_file="${queue_configs[$input_file]}"
        echo "Generating queue config: $input_file -> $output_file"
        
        # Generate host-config.yaml if needed (using helm template)
        if generate_host_config "$input_file"; then
            # File was generated, add to cleanup list
            generated_files+=("$input_file")
        fi
        
        # Check if generation/preparation was successful
        if [[ ! -f "$input_file" ]]; then
            echo "ERROR: Failed to prepare host config for $input_file"
            exit 1
        fi
        
        python3 "$cli" "$input_file" "$output_file"
    done
    
    # Clean up generated files
    if [[ ${#generated_files[@]} -gt 0 ]]; then
        echo ""
        echo "Cleaning up generated host-config.yaml files..."
        for generated_file in "${generated_files[@]}"; do
            if [[ -f "$generated_file" ]]; then
                rm -f "$generated_file"
                echo "Removed generated file: $generated_file"
            fi
        done
    fi

    # Verify no changes if flag is set
    if [[ "$verify_no_change" != "true" ]]; then
        return 0
    fi

    echo "Verifying no changes were made to cluster-queue.yaml files..."
    
    local changes_detected=false
    for input_file in "${!queue_configs[@]}"; do
        local output_file="${queue_configs[$input_file]}"
        if ! git diff --exit-code --quiet "$output_file" 2>/dev/null; then
            echo "ERROR: Changes detected in $output_file"
            git diff "$output_file"
            changes_detected=true
        fi
    done
    
    if [[ "$changes_detected" == "true" ]]; then
        echo "ERROR: Verification failed - changes were detected in output files"
        exit 1
    else
        echo "SUCCESS: No changes detected in cluster-queue.yaml files"
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    (
        cd "$ROOT/../.."
        main "$@"
    )
fi
