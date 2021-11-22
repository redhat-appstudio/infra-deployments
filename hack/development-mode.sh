
#!/bin/bash
ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/.. 

REPO=$(git config --get remote.origin.url)
 
#set the local cluster to point to the current git repo and update the path to development
$ROOT/hack/util-update-app-of-apps.sh $REPO development
# reset the default repos in the development directory to be the current git repo
# this needs to be pushed to your fork to be seen by argocd
$ROOT/hack/util-set-development-repos.sh $REPO development