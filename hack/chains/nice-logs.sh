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

# Some log messages are not wrapped in a json object. Convert as needed:
#   Input: 2022/04/27 17:28:36 Registering 3 clients
#   Output: {"ts": "2022-04-27T17:28:36.000Z", "level": "n/a", "msg": "Registering 3 clients"}
MOCK_JSON_REC='{ts: "\(.[:10] | gsub("/"; "-"))T\(.[11:19]).000Z", level: "n/a", msg: .[20:]}'

oc -n tekton-chains logs -f deployment/tekton-chains-controller |
  jq -R -r --unbuffered ". | fromjson? // ${MOCK_JSON_REC} | \"\($SET_COLOR)\(.ts) \(.level) \(.msg)\""
