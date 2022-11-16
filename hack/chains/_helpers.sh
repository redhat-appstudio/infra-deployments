#
#
# Misc utilties for bash scripted demos
#
# Typical usage:
#   source $(dirname $0)/_helpers.sh
#


#------------------------------------------------

#
# Useful directories
#
ROOT=$(git rev-parse --show-toplevel)
HACK_CHAINS=$ROOT/hack/chains
SCRIPTDIR=$HACK_CHAINS


#------------------------------------------------

#
# Cosign knows how to fetch the key from the secret in the cluster.
# Requires that you're authenticated with an account that can access
# the signing-secret, i.e. kubeadmin but not developer.
#
K8S_SIG_KEY=k8s://tekton-chains/signing-secrets

#
# This works if you have the public key locally, perhaps because you
# just created it. See also create-signing-secret.sh.
#
LOCAL_SIG_KEY=$ROOT/cosign.pub

# Presumably real public keys can be published somewhere in future
#PUB_SIG_KEY=?

# For now use this by default
[[ -z $SIG_KEY ]] && SIG_KEY=$K8S_SIG_KEY


#------------------------------------------------

#
# Support running demos quietly or without pauses
#
QUIET=
FAST=
for arg in "$@"; do
  #
  #
  if [[ $arg == "--quiet" ]]; then
    QUIET=1
    FAST=1
  fi

  if [[ $arg == "--fast" ]]; then
    FAST=1
  fi
done

#
# Create a pretty heading
#
title() {
  [[ -n $QUIET ]] && return

  echo
  echo "🔗 ---- $* ----"
}

#
# Quiet aware echo
#
say() {
  [[ -n $QUIET ]] && return

  echo $*
}

#
# Pause and wait for user to hit enter
#
pause() {
  [[ -n $QUIET ]] || [[ -n $FAST ]] && return

  echo
  local MSG="$*"
  [[ -z "$MSG" ]] && MSG="Hit enter to continue..."
  read -p "$MSG"
}

#
# Show a command then run it after user hits enter
# (Could use set -x instead I guess.)
#
show-then-run() {
  [[ -z $QUIET ]] && [[ -z $FAST ]] && read -p "\$ $*"
  $*
}


#------------------------------------------------

#
# Pretty print json as yaml
#
yq-pretty() {
  yq -P -C e ${1:-} -
}

#
# Fetch json with curl
#
curl-json() {
  curl -s -H "Accept: application/json" $@
}

#
# Trim the type prefix from a k8s name
#
trim-name() {
  echo "$1" | sed 's#.*/##'
}
