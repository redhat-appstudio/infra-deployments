#!/bin/bash
#
# A wrapper for kubectl patch so we can easily inspect and modify
# tekton chains configuration.
#
# For reference:
#   https://github.com/tektoncd/chains/blob/main/docs/config.md
#   https://github.com/tektoncd/chains/blob/main/pkg/config/config.go#L99
#
# See also ./gitops-sync.sh. You need to disable gitops automatic
# self healing before you can modify the configuration.
#

CHAINS_CONFIG="configmap/chains-config -n tekton-chains"

case "$1" in

  get )
    # Show the current config
    kubectl get $CHAINS_CONFIG -o yaml | yq e .data -

    ;;

  default )
    # This matches the current gitops config
    # Should work for the kaniko demo
    $0 '{
      "artifacts.taskrun.format": "in-toto",
      "artifacts.taskrun.storage": "oci",
      "artifacts.oci.storage": "oci",
      "transparency.enabled": "true"
    }' $2

    ;;

  dual-storage )
    # Same as the default but also store as tekton
    $0 '{
      "artifacts.taskrun.format": "in-toto",
      "artifacts.taskrun.storage": "tekton,oci",
      "artifacts.oci.storage": "tekton,oci",
      "transparency.enabled": "true"
    }' $2

    ;;

  simple )
    # Should work for the simple demo
    $0 '{
      "artifacts.taskrun.format": "tekton",
      "artifacts.taskrun.storage": "tekton",
      "artifacts.oci.storage": "tekton",
      "transparency.enabled": "false"
      }' $2

    ;;

  quay )
    # Currently quay.io doesn't support oci storage for
    # attestations so fall back to tekton storage
    #
    # See https://issues.redhat.com/browse/PROJQUAY-3386
    # Once that's done then quay can use the default config
    # and we can delete this.
    #
    $0 '{
      "artifacts.taskrun.format": "in-toto",
      "artifacts.taskrun.storage": "tekton",
      "artifacts.oci.storage": "oci",
      "transparency.enabled": "true"
      }' $2

    ;;

  rekor-on )
    # (Needs to be a string not a boolean FYI)
    $0 'transparency.enabled: "true"' $2

    ;;

  rekor-off )
    $0 'transparency.enabled: "false"' $2

    ;;

  rekor-local )
    REKOR_SERVER=$(kubectl get ingress -n rekor-server -o yaml|yq e '.items[.spec].spec.rules[.host].host')
    $0 "transparency.url: \"$REKOR_SERVER\"" $2

    ;;

  rekor-default )
    # if setting rekor-default we should ensure we don't have a transparency.url value
    $0 remove-key 'transparency.url' $2

    ;;

  remove-key )
    PATCH=$(echo "[{"op": "remove", "path": "/data/$2"}]" | yq -o=json --indent=0 e - )
    if [[ $3 == '--dry-run' ]]; then
      echo "$PATCH" | yq e -P 'sort_keys(..)' -
    else
      set -x
      kubectl patch $CHAINS_CONFIG --type=json --patch $PATCH 
      $0 restart-controller
      $0 get
    fi

    ;;

  restart-controller )
    # Restart the controller to make sure the new config takes effect
    kubectl delete pod -n tekton-chains -l app=tekton-chains-controller
    
    ;;

  * )
    # Avoid clearing all config if no param is given
    [[ -z $1 ]] && $0 get && exit

    # Use yq to convert the input to a single line of json
    # so we can use yaml or json for the input
    PATCH=$( echo "$1" | yq -o=json --indent=0 e - )

    if [[ $2 == "--dry-run" ]]; then
      # Just show the desired config
      echo "$PATCH" | yq e -P 'sort_keys(..)' -
    else
      # Apply the patch
      set -x
      kubectl patch $CHAINS_CONFIG --patch "{\"data\":$PATCH} --type=merge"

      # Restart the controller to ensure new config takes effect
      $0 restart-controller

      # Show the config as a confirmation
      $0 get
    fi

    ;;

esac
