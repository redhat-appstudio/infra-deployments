
# deprecated util for installing secrets for git repo

must_exist () {
    ENAME=$1
    if [ -z "${!ENAME}" ]
    then
      echo Missing env var named "$ENAME"
      exit -1 
    fi
}  
must_exist MY_GITHUB_USER
must_exist MY_GITHUB_TOKEN
must_exist MY_QUAY_USER
must_exist MY_QUAY_TOKEN  

NS=$(oc project --short)
echo "$NS for Secrets"
echo "Creating git-repo-secret and quay-registry-secret for auth workspaces"

read -r -d '' GITSECRET <<'GITSECRET'
kind: Secret
apiVersion: v1
metadata:
  name: git-repo-secret
type: Opaque
stringData:
  .gitconfig: |
    [credential "https://github.com"]
      helper = store
  .git-credentials: |
    https://<user>:<pass>@github.com
GITSECRET
 
PATCH="$(printf '.stringData.".git-credentials"="https://%q:%q@github.com"' $MY_GITHUB_USER $MY_GITHUB_TOKEN)" 
echo "$GITSECRET" | yq e $PATCH -  | oc apply -f -
 
oc create secret -n $NS  docker-registry quay-registry-secret \
  --docker-server="https://quay.io" \
  --docker-username=$MY_QUAY_USER \
  --docker-password=$MY_QUAY_TOKEN 