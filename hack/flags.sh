#!/bin/bash

SCRIPT_DESC=$1
EXTRA_PARAMS=$2
EXTRA_HELP=$3

user_help () {
  echo "${SCRIPT_DESC}"
  echo "options:"
  echo "-h,  --help                   Show this help info"
  echo "-kk, --kcp-kubeconfig         Kubeconfig pointing to the kcp instance."
  echo "                              Don't use in the preview mode - use only the preview.env file."
  echo "-ck, --cluster-kubeconfig     Kubeconfig pointing to the OpenShift cluster that will be used as a sync target."
  echo "                              Don't use in the preview mode - use only the preview.env file."
  echo "-rw, --root-workspace         Fully-qualified name of the kcp workspace that should be used as root (default is 'root')."
  echo "                              Don't use in the preview mode - use only the preview.env file."
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
        KCP_KUBECONFIG_FLAG=$1
        shift
        ;;
      -ck|--cluster-kubeconfig)
        shift
        CLUSTER_KUBECONFIG_FLAG=$1
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

  if [ "${MODE}" == "preview" ]
  then
    if [[ -n ${KCP_KUBECONFIG_FLAG}${CLUSTER_KUBECONFIG_FLAG} ]]
    then
      echo "ERROR: You cannot use the parameter --kcp-kubeconfig nor --cluster-kubeconfig in preview mode - use only the './hack/preview.env' file" >&2
      exit 1
    elif [[ -f ${ROOT}/hack/preview.env ]]
    then
      echo "Loading environment variables from ${ROOT}/hack/preview.env"
      source ${ROOT}/hack/preview.env
      $ROOT/hack/util-validate-preview-env.sh
    else
      echo "ERROR: No ${ROOT}/hack/preview.env was found" >&2
      exit 1
    fi
  elif [[ -z ${KCP_KUBECONFIG_FLAG} ]] || [[ -z ${CLUSTER_KUBECONFIG_FLAG} ]]
  then
    echo "ERROR: Both parameters --kcp-kubeconfig and --cluster-kubeconfig are mandatory" >&2
    exit 1
  else
    export KCP_KUBECONFIG=${KCP_KUBECONFIG_FLAG}
    export CLUSTER_KUBECONFIG=${CLUSTER_KUBECONFIG_FLAG}
  fi
  export ROOT_WORKSPACE=${ROOT_WORKSPACE:-"root"}

  # Convert home ROOT_WORKSPACE to full path
  if [ "$ROOT_WORKSPACE" == "~" ]; then
    ROOT_WORKSPACE=$(KUBECONFIG=${KCP_KUBECONFIG} kubectl ws '~' --short)
  fi

  # Check config files and version compatibility
  if kubectl version -o yaml --kubeconfig ${CLUSTER_KUBECONFIG} | yq '.serverVersion.gitVersion' | grep -q kcp; then
    echo CLUSTER_KUBECONFIG=${CLUSTER_KUBECONFIG} points to KCP not to cluster.
    exit 1
  fi
  KCP_SERVER_VERSION=$(kubectl version -o yaml --kubeconfig ${KCP_KUBECONFIG} 2>/dev/null | yq '.serverVersion.gitVersion')
  if ! echo "$KCP_SERVER_VERSION" | grep -q 'kcp\|v0.0.0-master'; then
    echo KCP_KUBECONFIG=${KCP_KUBECONFIG} does not point to KCP cluster.
    exit 1
  fi
  KUBECTL_CLIENT_VERSION=$(kubectl version --client -o yaml)
  if [ $(echo "$KUBECTL_CLIENT_VERSION" | yq '.clientVersion.minor') -lt 24 ]; then
    echo 'kubectl 1.24.x or newer needs to be used'
    exit 1
  fi
  KCP_SERVER=$(echo $KCP_SERVER_VERSION | sed 's/.*kcp-v\(.*\)\..*/\1/')
  KCP_CLIENT=$(kubectl kcp --version | sed 's/.*kcp-v\(.*\)\..*/\1/')
  if echo "$KCP_SERVER_VERSION" | grep -q 'v0.0.0-master'; then
    export KCP_VERSION=${KCP_VERSION:-"main"}
    echo "KCP server is self compiled, cannot check kubectl kcp plugin compatibility"
  elif [ "$KCP_SERVER" != "$KCP_CLIENT" ]; then
    echo "KCP server version($KCP_SERVER) does not match kcp plugin version($KCP_CLIENT)"
    exit 1
  fi
  export KCP_VERSION=${KCP_VERSION:-"$(echo ${KCP_SERVER_VERSION} | sed 's/.*kcp-\([^-]*\).*/\1/')"}
}
