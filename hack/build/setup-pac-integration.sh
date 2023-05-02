#!/usr/bin/env bash

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

        if ! oc get -n openshift-pipelines secret pipelines-as-code-secret &>/dev/null; then
                token=$(sign rs256 "$payload" "$(echo "$PAC_GITHUB_APP_PRIVATE_KEY" | base64 -d)")
                webhook_url=$(oc whoami --show-console | sed 's/console-openshift-console/pipelines-as-code-controller-openshift-pipelines/')
                curl \
                -X PATCH \
                -H "Accept: application/vnd.github.v3+json" \
                -H "Authorization: Bearer $token" \
                https://api.github.com/app/hook/config \
                -d "{\"content_type\":\"json\",\"insecure_ssl\":\"1\",\"secret\":\"$webhook_secret\",\"url\":\"$webhook_url\"}" &>/dev/null
        fi

        echo $webhook_secret
)

if [ -n "${PAC_GITHUB_APP_ID}" ] && [ -n "${PAC_GITHUB_APP_PRIVATE_KEY}" ]; then
        WEBHOOK_SECRET=$(setup-pac-app)
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

PAC_NAMESPACE='openshift-pipelines'
PAC_SECRET_NAME='pipelines-as-code-secret'

oc create namespace -o yaml --dry-run=client ${PAC_NAMESPACE} | oc apply -f-
oc create namespace -o yaml --dry-run=client build-service | oc apply -f-

eval "oc -n '$PAC_NAMESPACE' create secret generic '$PAC_SECRET_NAME' $GITHUB_APP_DATA $GITHUB_WEBHOOK_DATA $GITLAB_WEBHOOK_DATA -o yaml --dry-run=client" | oc apply -f-
eval "oc -n build-service create secret generic '$PAC_SECRET_NAME' $GITHUB_APP_DATA $GITHUB_WEBHOOK_DATA $GITLAB_WEBHOOK_DATA -o yaml --dry-run=client" | oc apply -f-
echo "Configured ${PAC_SECRET_NAME} secret in ${PAC_NAMESPACE} namespace"
