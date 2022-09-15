#!/bin/bash

SCRIPT_DESC=$1
EXTRA_PARAMS=$2
EXTRA_HELP=$3

user_help () {
  echo "${SCRIPT_DESC}"
  echo "options:"
  echo "-h,  --help                   Show this help info"
  echo "-kk, --kcp-kubeconfig         Kubeconfig pointing to the kcp instance"
  echo "-ck, --cluster-kubeconfig     Kubeconfig pointing to the OpenShift cluster that will be used as a sync target"
  echo "-rw, --root-workspace         Fully-qualified name of the kcp workspace that should be used as root (default is 'root')"
  if [[ -n ${EXTRA_PARAMS} ]]
  then
    ${EXTRA_HELP}
  fi
  exit 0
}

parse_flags() {
  while test $# -gt 0; do
    case "$1" in
      -h|--help)
        user_help
        ;;
      -kk|--kcp-kubeconfig)
        shift
        export KCP_KUBECONFIG=$1
        shift
        ;;
      -ck|--cluster-kubeconfig)
        shift
        export CLUSTER_KUBECONFIG=$1
        shift
        ;;
      -rw|--root-workspace)
        shift
        export ROOT_WORKSPACE=$1
        shift
        ;;
      *)
        if [[ -n ${EXTRA_PARAMS} ]]
        then
          ${EXTRA_PARAMS} $1 $2
          shift
          shift
        else
          echo "ERROR: '$1' is not a recognized flag!" >&2
          user_help >&2
          exit 1
        fi
       ;;
    esac
  done

  if [[ -z ${KCP_KUBECONFIG} ]] || [[ -z ${CLUSTER_KUBECONFIG} ]]
  then
    echo "ERROR: Both parameters --kcp-kubeconfig and --cluster-kubeconfig are mandatory" >&2
    exit 1
  fi
  export ROOT_WORKSPACE=${ROOT_WORKSPACE:-"root"}

  # Check config files and version compatibility
  if kubectl version -o yaml --kubeconfig ${CLUSTER_KUBECONFIG} | yq '.serverVersion.gitVersion' | grep -q kcp; then
    echo CLUSTER_KUBECONFIG=${CLUSTER_KUBECONFIG} points to KCP not to cluster.
    exit 1
  fi
  KCP_SERVER_VERSION=$(kubectl version -o yaml --kubeconfig ${KCP_KUBECONFIG} | yq '.serverVersion.gitVersion')
  if ! echo "$KCP_SERVER_VERSION" | grep -q 'kcp\|v0.0.0-master'; then
    echo KCP_KUBECONFIG=${KCP_KUBECONFIG} does not point to KCP cluster.
    exit 1
  fi
  KCP_SERVER=$(echo $KCP_SERVER_VERSION | sed 's/.*kcp-v\(.*\)\..*/\1/')
  KCP_CLIENT=$(kubectl kcp --version | sed 's/.*kcp-v\(.*\)\..*/\1/')
  if echo "$KCP_SERVER_VERSION" | grep -q 'v0.0.0-master'; then
    echo "KCP server is self compiled, cannot check kubectl kcp plugin compatibility"
  elif [ "$KCP_SERVER" != "$KCP_CLIENT" ]; then
    echo "KCP server version($KCP_SERVER) does not match kcp plugin version($KCP_CLIENT)"
    exit 1
  fi
}
