#!/bin/bash

# Can be executed on registry.redhat.io/openshift4/ose-cli:v4.11

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

if [[ -z ${1} ]]
then
  echo "Creates apiexport files in overlays for component"
  echo "     generate-apiexport-overlays.sh path-to-component-dir"
  exit 1
fi

COMPONENT_DIR=$1
BASE_DIR=$COMPONENT_DIR/base

if [ ! -d $BASE_DIR ]; then
  echo "component dir must contain 'base' dir"
fi

TEMP=$(mktemp -d)
kubectl kustomize $BASE_DIR | csplit -sf $TEMP/resource - /^---/ '{*}'

APIEXPORT_FILES=$(grep -lr 'kind: APIExport' $TEMP)
if [ -n "$APIEXPORT_FILES" ]; then
  cat $APIEXPORT_FILES > $COMPONENT_DIR/overlays/dev/apiexport.yaml
fi
rm -rf $TEMP

resolve_identity_hashes() {
  IDENTITY_HASHES=$(awk '$1 == "identityHash:" {print $2}' $1 | uniq)
  for IDENTITY_HASH_PLACEHOLDER in $IDENTITY_HASHES; do
    HASHFILE=$ROOT/identityhashes/$2/$IDENTITY_HASH_PLACEHOLDER
    if [ -f $HASHFILE ]; then
      SED_SUBST="${SED_SUBST}s/\b$IDENTITY_HASH_PLACEHOLDER\b/$(cat $HASHFILE)/g;"
    fi
  done
  if [ -n $SED_SUBST ]; then
    sed -e "$SED_SUBST" $1 > $3
  fi
}

resolve_identity_hashes $COMPONENT_DIR/overlays/dev/apiexport.yaml kcp-stable $COMPONENT_DIR/overlays/kcp-stable/apiexport.yaml
resolve_identity_hashes $COMPONENT_DIR/overlays/dev/apiexport.yaml kcp-unstable $COMPONENT_DIR/overlays/kcp-unstable/apiexport.yaml
