#!/bin/bash

SCRIPT_DESC=$1
EXTRA_PARAMS=$2
EXTRA_HELP=$3

user_help () {
  echo "${SCRIPT_DESC}"
  echo "options:"
  echo "-h,  --help                   Show this help info"
  echo "-kk, --kcp-kubeconfig         Kubeconfig pointing to the kcp instance."
  echo "                              Don't use in any of the preview modes - use only the preview.env file."
  echo "-ck, --cluster-kubeconfig     Kubeconfig pointing to the OpenShift cluster that will be used as a sync target."
  echo "                              Don't use in any of the preview modes - use only the preview.env file."
  echo "-rw, --root-workspace         Fully-qualified name of the kcp workspace that should be used as root (default is 'root')."
  echo "                              Don't use in the preview-cps mode - it uses the home workspace of the kcp user automatically."
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

  if echo ${MODE} | grep -q preview
  then
    if [[ -n ${KCP_KUBECONFIG_FLAG}${CLUSTER_KUBECONFIG_FLAG} ]]
    then
      echo "ERROR: You cannot use the parameter --kcp-kubeconfig nor --cluster-kubeconfig in preview mode - use only the './hack/preview.env' file" >&2
      exit 1
    elif [[ -f ${ROOT}/hack/preview.env ]]
    then
      echo "Loading environment variables from ${ROOT}/hack/preview.env"
      source ${ROOT}/hack/preview.env
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
}
