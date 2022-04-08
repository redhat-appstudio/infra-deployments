#!/bin/bash

if ! tkn results &>/dev/null; then
   echo Command 'tkn results' is not installed
   echo https://github.com/tektoncd/results/blob/main/tools/tkn-results/README.md
fi

URL=$(oc whoami --show-console | sed 's|https://console-openshift-console|api-tekton-pipelines|')
openssl s_client -showcerts -connect $URL:443 </dev/null 2>/dev/null | sed -n -e '/-.BEGIN/,/-.END/ p' > ~/.config/tkn/cert.pem

cat > ~/.config/tkn/results.yaml <<EOF
address: $URL:443
ssl:
    roots_file_path: $HOME/.config/tkn/cert.pem
EOF
echo Configuration written to ~/.config/tkn/results.yaml
echo
echo Try it: tkn results list $(oc config view --minify -o 'jsonpath={..namespace}')
