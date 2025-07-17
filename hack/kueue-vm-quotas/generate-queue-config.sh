#!/bin/bash -e
# This script generates Kueue cluster queue configuration files for VM quotas
# by processing host configuration files and invoking the update-kueue-vm-quotas.py
# script with the appropriate input and output paths.

declare -r ROOT="${BASH_SOURCE[0]%/*}"

main() {
    local cli="hack/kueue-vm-quotas/update-kueue-vm-quotas.py"
    # public staging
    python3 \
        "$cli" \
        components/multi-platform-controller/staging/host-config.yaml \
        components/kueue/staging/stone-stg-rh01/queue-config/cluster-queue.yaml

    # private staging
    python3 \
        "$cli" \
        components/multi-platform-controller/staging-downstream/host-config.yaml \
        components/kueue/staging/stone-stage-p01/queue-config/cluster-queue.yaml
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    (
        cd "$ROOT/../.."
        main "$@"
    )
fi
