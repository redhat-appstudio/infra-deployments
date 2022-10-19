#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

TEMPDIR=$(mktemp -d)
REQUIRED=$TEMPDIR/required
ALL=$TEMPDIR/optional
PREVIEW=$TEMPDIR/preview
cut -d '=' -f 1 $ROOT/hack/preview-template.env | sed '/^# Optional$/q' | grep "^[^#]" | sort > $REQUIRED
cut -d '=' -f 1 $ROOT/hack/preview-template.env | grep "^[^#]" | sort > $ALL
cut -d '=' -f 1 $ROOT/hack/preview.env | grep "^[^#]" | sort > $PREVIEW

echo Validating required environments
MISSING_REQUIRED=$(comm -23 $REQUIRED $PREVIEW)
SAVEIFS=$IFS
IFS=$(echo -en "\n\b")
for ENV in $MISSING_REQUIRED; do
  ENV_NAME=$(echo $ENV | cut -f2 -d' ')
  if ! printenv $ENV_NAME >/dev/null; then
    echo Error: $ENV is missing in preview.env or exported manually
    FAIL=1
  else
    echo $ENV_NAME is set but not in preview.env
  fi
done
IFS=$SAVEIFS
if [ "$FAIL" == "1" ]; then
  rm -rf $TEMPDIR
  exit 1
fi

echo Validating that all environment variables from preview.env are defined in preview-template.env
ADDITIONAL=$(comm -13 $ALL $PREVIEW)
SAVEIFS=$IFS
IFS=$(echo -en "\n\b")
for ENV in $ADDITIONAL; do
  ENV_NAME=$(echo $ENV | cut -f2 -d' ')
  if ! printenv $ENV_NAME >/dev/null; then
    echo Warning: $ENV from preview.env is not known
  fi
done

echo These optional variables are not set: $(comm -13 $PREVIEW $ALL | cut -f2 -d' ')

rm -rf $TEMPDIR
