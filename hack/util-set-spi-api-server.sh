#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
PATCH_FILE="$ROOT/components/spi/oauth-service-deployment-patch.json"
API_SERVER=$1

TMP_FILE=$(mktemp)

cat $PATCH_FILE | jq '
    map(
        if (.op == "add" and .path == "/spec/template/spec/containers/0/env/-") then
            {
                "op": .op,
                "path": .path,
                "value": [
                    .value[] | if .name == "API_SERVER" then
                        {"name": "API_SERVER", "value": "'"$API_SERVER"'"}
                    else
                        .
                    end
                ]
            }
        else
            .
        end
    )' > "$TMP_FILE"

mv "$TMP_FILE" "$PATCH_FILE"
