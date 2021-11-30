
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

oc create secret generic git-secret \
    --from-literal=username=$MY_GITHUB_USER \
    --from-literal=password=$MY_GITHUB_TOKEN \
    --type=kubernetes.io/basic-auth

oc create secret generic registry-secret \
    --from-literal=username=$MY_QUAY_USER \
    --from-literal=password=$MY_QUAY_TOKEN \
    --type=kubernetes.io/basic-auth