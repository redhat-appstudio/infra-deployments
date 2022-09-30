#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

if [[ -z ${2} ]]
then
  echo "Resolve identityhashes placeholders "
  echo "     resolve-identityhashes.sh <kcp-stable|kcp-unstable> file-to-resolve"
  exit 1
fi


KCP_NAME=$1
FILE=$2

IDENTITY_HASHES=$(awk '$1 == "identityHash:" {print $2}' $FILE | uniq)
for IDENTITY_HASH_PLACEHOLDER in $IDENTITY_HASHES; do
  HASHFILE=$ROOT/identityhashes/$KCP_NAME/$IDENTITY_HASH_PLACEHOLDER
  if [ -f $HASHFILE ]; then
    SED_SUBST="${SED_SUBST}s/\b$IDENTITY_HASH_PLACEHOLDER\b/$(cat $HASHFILE)/g;"
  fi
done
if [ -n $SED_SUBST ]; then
  sed -e "$SED_SUBST" $FILE
fi
