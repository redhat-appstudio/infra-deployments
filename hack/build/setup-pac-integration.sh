#!/usr/bin/env bash

PAC_NAMESPACE='openshift-pipelines'
PAC_SECRET_NAME='pipelines-as-code-secret'

setup-pac-app() (
        # Inspired by implementation by Will Haley at:
        #   http://willhaley.com/blog/generate-jwt-with-bash/

        # Shared content to use as template
        header_template='{
        "typ": "JWT",
        "kid": "0001",
        "iss": "https://stackoverflow.com/questions/46657001/how-do-you-create-an-rs256-jwt-assertion-with-bash-shell-scripting"
        }'

        now=$(date +%s)
        build_header() {
                jq -c \
                        --arg iat_str "$now" \
                        --arg alg "${1:-HS256}" \
                '
                ($iat_str | tonumber) as $iat
                | .alg = $alg
                | .iat = $iat
                | .exp = ($iat + 10)
                ' <<<"$header_template" | tr -d '\n'
        }

        b64enc() { openssl enc -base64 -A | tr '+/' '-_' | tr -d '='; }
        json() { jq -c . | LC_CTYPE=C tr -d '\n'; }
        hs_sign() { openssl dgst -binary -sha"${1}" -hmac "$2"; }
        rs_sign() { openssl dgst -binary -sha"${1}" -sign <(printf '%s\n' "$2"); }

        sign() {
                local algo payload header sig secret=$3
                algo=${1:-RS256}; algo=${algo^^}
                header=$(build_header "$algo") || return
                payload=${2:-$test_payload}
                signed_content="$(json <<<"$header" | b64enc).$(json <<<"$payload" | b64enc)"
                case $algo in
                        HS*) sig=$(printf %s "$signed_content" | hs_sign "${algo#HS}" "$secret" | b64enc) ;;
                        RS*) sig=$(printf %s "$signed_content" | rs_sign "${algo#RS}" "$secret" | b64enc) ;;
                        *) echo "Unknown algorithm" >&2; return 1 ;;
                esac
                printf '%s.%s\n' "${signed_content}" "${sig}"
        }
        payload="{ \"iss\": $PAC_GITHUB_APP_ID, \"iat\": ${now}, \"exp\": $((now+10)) }"

        webhook_secret=$(openssl rand -hex 20)

        token=$(sign rs256 "$payload" "$(echo "$PAC_GITHUB_APP_PRIVATE_KEY" | base64 -d)")

        local retry=0
        while ! oc get -n $PAC_NAMESPACE route pipelines-as-code-controller >/dev/null 2>&1 ; do
                if [ "$retry" -eq "20" ]; then
                        echo "[ERROR] Failed to wait for Pac route to be available" >&2
                        exit 1
                fi
                echo "Waiting for Pac route to be available" >&2
                sleep 5
                retry=$((retry+1))
        done
        pac_host=$(oc get -n $PAC_NAMESPACE route pipelines-as-code-controller -o go-template="{{ .spec.host }}")
        curl \
        -X PATCH \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Authorization: Bearer $token" \
        https://api.github.com/app/hook/config \
        -d "{\"content_type\":\"json\",\"insecure_ssl\":\"1\",\"secret\":\"$webhook_secret\",\"url\":\"https://$pac_host\"}" &>/dev/null

        echo "$webhook_secret"
)

if [ -n "${PAC_GITHUB_APP_ID}" ] && [ -n "${PAC_GITHUB_APP_PRIVATE_KEY}" ]; then
        # if using the existing QE sprayproxy, we suppose the setup between sprayproxy and github App is already done
        if [ -n "${PAC_GITHUB_APP_WEBHOOK_SECRET}" ]; then
                echo "Setup Pac with existing QE sprayproxy and github App"
                WEBHOOK_SECRET="$PAC_GITHUB_APP_WEBHOOK_SECRET"
        else
        # if not, we setup pac with github App directly, this step will update the webhook secret and webhook url in the github App
                echo "Setup Pac with github App"
                WEBHOOK_SECRET=$(setup-pac-app)
        fi
        GITHUB_APP_PRIVATE_KEY=$(echo "$PAC_GITHUB_APP_PRIVATE_KEY" | base64 -d)
        GITHUB_APP_DATA="--from-literal github-private-key='$GITHUB_APP_PRIVATE_KEY' --from-literal github-application-id='${PAC_GITHUB_APP_ID}' --from-literal webhook.secret='$WEBHOOK_SECRET'"                
fi

if [ -n "${PAC_GITHUB_TOKEN}" ]; then
        GITHUB_WEBHOOK_DATA="--from-literal github.token='${PAC_GITHUB_TOKEN}'"
else 
        if [ -n "${MY_GITHUB_TOKEN}" ]; then
                GITHUB_WEBHOOK_DATA="--from-literal github.token='${MY_GITHUB_TOKEN}'"
        fi
fi

if [ -n "${PAC_GITLAB_TOKEN}" ]; then
        GITLAB_WEBHOOK_DATA="--from-literal gitlab.token='${PAC_GITLAB_TOKEN}'"
fi

oc create namespace -o yaml --dry-run=client ${PAC_NAMESPACE} | oc apply -f-
oc create namespace -o yaml --dry-run=client build-service | oc apply -f-

eval "oc -n '$PAC_NAMESPACE' create secret generic '$PAC_SECRET_NAME' $GITHUB_APP_DATA $GITHUB_WEBHOOK_DATA $GITLAB_WEBHOOK_DATA -o yaml --dry-run=client" | oc apply -f-
eval "oc -n build-service create secret generic '$PAC_SECRET_NAME' $GITHUB_APP_DATA $GITHUB_WEBHOOK_DATA $GITLAB_WEBHOOK_DATA -o yaml --dry-run=client" | oc apply -f-
echo "Configured ${PAC_SECRET_NAME} secret in ${PAC_NAMESPACE} namespace"
