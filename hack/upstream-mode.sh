
#!/bin/bash
ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/.. 

# switch to known upstream and revert the development overlay to be the upstream repo
# this prevents accidental pull requests referencing forked repos used in dev
# note, could use $(git config --get remote.upstream.url) but hardcoded for now
# not everyone may have an upstream set
REPO=https://github.com/redhat-appstudio/infra-deployments.git

#set the local cluster to point back to the upstream  
$ROOT/hack/util-update-app-of-apps.sh $REPO staging
#reset the default content in the development directory to be the upstream
$ROOT/hack/util-set-development-repos.sh $REPO development
 