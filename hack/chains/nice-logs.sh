#!/bin/bash

ANSI_RED="[27, 91, 49, 59, 51, 49, 109]"
ANSI_PURPLE="[27, 91, 49, 59, 51, 53, 109]"
ANSI_RESET="[27, 91, 48, 109]"

SET_COLOR="
if .level == \"error\" then
  $ANSI_RED
elif .level == \"warn\" then
  $ANSI_PURPLE
else
  $ANSI_RESET
end | implode"

oc -n tekton-chains logs -f deployment/tekton-chains-controller |
  jq -R -r --unbuffered ". | fromjson? | \"\($SET_COLOR)\(.ts) \(.level) \(.msg)\""
