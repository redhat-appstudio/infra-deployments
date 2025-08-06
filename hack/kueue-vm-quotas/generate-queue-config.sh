#!/bin/bash -e
# This script generates Kueue cluster queue configuration files for VM quotas
# by processing host configuration files and invoking the update-kueue-vm-quotas.py
# script with the appropriate input and output paths.
#
# Usage: generate-queue-config.sh [--verify-no-change]
#   --verify-no-change: Verify that no changes were made to the output files

declare -r ROOT="${BASH_SOURCE[0]%/*}"

usage() {
    echo "Usage: $0 [--verify-no-change]"
    echo "  --verify-no-change: Verify that no changes were made to the output files"
    exit 1
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
    
    # Generate queue configurations
    for input_file in "${!queue_configs[@]}"; do
        local output_file="${queue_configs[$input_file]}"
        echo "Generating queue config: $input_file -> $output_file"
        python3 "$cli" "$input_file" "$output_file"
    done

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
