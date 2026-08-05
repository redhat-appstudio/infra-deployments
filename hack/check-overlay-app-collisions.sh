#!/usr/bin/env bash
set -e -o pipefail

# check-overlay-app-collisions.sh
#
# Detects ArgoCD ApplicationSets that would template colliding generated
# Application names when deployed to the same cluster / ArgoCD control-plane
# namespace.
#
# Background: argo-cd-apps overlays generate child Applications from an
# ApplicationSet's `spec.template.metadata.name`, e.g. `kueue-{{nameNormalized}}`.
# Kustomize's `nameSuffix` only renames the ApplicationSet object itself
# (e.g. "kueue" -> "kueue-ring-0"); it does NOT rewrite the templated string
# inside spec.template.metadata.name, since that's opaque spec data to
# Kustomize, not a tracked object reference. So if two *different*
# ApplicationSets are deployed to the same cluster and both template the same
# generated Application name (e.g. both produce "kueue-{{nameNormalized}}"),
# ArgoCD's ApplicationSet controller will fight over ownership of the
# resulting Application object -- one of them loses, and that Application
# gets stuck/flapping.
#
# This matters most during the argo-cd-apps/overlays/development ->
# argo-cd-apps/overlays/rd-dev ring-based migration (see
# docs/ring-deployments/): a component can temporarily exist in both trees
# while being migrated, and both overlays are deployed to the same cluster
# during e2e (see the dual `apply_and_wait_for_root_application` calls in
# hack/preview.sh). This script fails loudly at PR/CI time instead of
# letting the collision surface later as a flaky ArgoCD ownership conflict
# during e2e or in production.
#
# Usage: hack/check-overlay-app-collisions.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD="${SCRIPT_DIR}/kustomize-build-with-retry.sh"

# Field separator used for the intermediate "<template>|<appset-name>|<overlay>"
# records below. A literal pipe is used (rather than "\t") because yq's string
# literals don't expand backslash escapes -- "\t" comes out as the two
# characters '\' and 't', not an actual tab, which silently breaks any
# downstream `cut`/`read -d $'\t'` splitting. '|' can't appear in a
# Kubernetes object name or an ArgoCD ApplicationSet Go-template string, so
# it's a safe delimiter here.
SEP='|'

# Add new pairings here as new ring overlays are wired into shared bootstrap
# targets (see hack/preview.sh and argo-cd-apps/overlays/development-operator
# for existing examples of two trees being co-deployed).
#
# NOTE: intentionally not named "GROUPS" -- that's a bash builtin array
# variable (the current user's group IDs); assigning to it silently fails,
# and reading it back returns GIDs instead of our overlay paths.
CO_DEPLOYED_OVERLAY_GROUPS=(
    "argo-cd-apps/overlays/development argo-cd-apps/overlays/rd-dev"
    "argo-cd-apps/overlays/development-operator"
)

overall_status=0

for group in "${CO_DEPLOYED_OVERLAY_GROUPS[@]}"; do
    # shellcheck disable=SC2206
    overlays=(${group})
    echo "== Checking co-deployed group: ${overlays[*]} =="

    # Creates an entry in the temp file for each ApplicationSet in the group: "<generated-app-name-template>|<applicationset-name>|<overlay>"
    entries_file=$(mktemp)
    for overlay in "${overlays[@]}"; do
        manifest=$("${BUILD}" "${ROOT}/${overlay}")
        while IFS="${SEP}" read -r as_name tmpl_name; do
            [ -z "${tmpl_name}" ] && continue
            printf '%s%s%s%s%s\n' "${tmpl_name}" "${SEP}" "${as_name}" "${SEP}" "${overlay}" >>"${entries_file}"
        done < <(echo "${manifest}" | yq e --no-doc "select(.kind == \"ApplicationSet\") | (.metadata.name // \"<unnamed>\") + \"${SEP}\" + (.spec.template.metadata.name // \"<unnamed>\")" -)
    done
    
    # Finds duplicate generated Application name templates in the temp file
    dup_templates=$(cut -d"${SEP}" -f1 "${entries_file}" | sort | uniq -d)

    if [ -z "${dup_templates}" ]; then
        echo "  OK: no colliding generated Application names"
    else
        overall_status=1
        while IFS= read -r dup; do
            [ -z "${dup}" ] && continue
            echo ""
            echo "COLLISION: generated Application name '${dup}' is templated by multiple ApplicationSets:"
            grep -F -- "${dup}${SEP}" "${entries_file}" | while IFS="${SEP}" read -r _ as_name overlay; do
                echo "  - ApplicationSet '${as_name}' in ${overlay}"
            done
        done <<<"${dup_templates}"
    fi

    rm -f "${entries_file}"
done

echo ""
if [ "${overall_status}" -ne 0 ]; then
    echo "One or more ApplicationSets would collide when co-deployed to the same cluster."
    echo "Fix one of the following before merging:"
    echo "  - Remove/disable the superseded ApplicationSet from its overlay (see the"
    echo "    delete-applications.yaml / delete-legacy-konflux-member-appsets.yaml"
    echo "    patterns used by argo-cd-apps/overlays/development(-operator))."
    echo "  - Rename the colliding Application via spec.template.metadata.name so the"
    echo "    two ApplicationSets no longer template the same generated Application name."
    exit 1
fi

echo "No overlay Application-name collisions detected."
