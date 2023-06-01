#!/bin/bash

set -e

trap 'last_command=$current_command; current_command=$BASH_COMMAND' DEBUG
trap 'echo "\"${last_command}\" command completed with exit code $?."' EXIT

ROOT="$(realpath -mq ${BASH_SOURCE[0]}/../../..)"

export QUICKCLUSTERKEY=${2:-~/.ssh/id_rsa}
export NAMESPACE=openshift-nfs-storage
PORT=2049

if [ -z "$1" ]; then
    echo "usage: setup-nfs-quickcluster.sh <upi host name>"
    exit 1
fi

export REMOTE=$1
QOCP="https://$(echo ${REMOTE} |sed s/upi-0/api/):6443"

scp -o StrictHostKeyChecking=no -i $QUICKCLUSTERKEY quickcluster@$REMOTE:/home/quickcluster/oc4/auth/kubeconfig /tmp/kubeconfig
export KUBECONFIG=/tmp/kubeconfig

if ! oc whoami > /dev/null 2>&1;
then
  echo "Please login to your openshift cluster: ${QOCP}";
  if ! oc login --server=${QOCP};
    then
      exit 1;
  fi
else
  OCP=$(oc whoami --show-server=true)
  if [ "${OCP}" != "${QOCP}" ];
  then
    echo "It seems you are logged in to the wrong OCP cluster (${OCP}). Please login to ${QOCP} and retry the script";
    exit 1;
  fi;
fi

PORT=2049

SSH="ssh -o StrictHostKeyChecking=no -i $QUICKCLUSTERKEY quickcluster@$REMOTE"

echo -e Adding nfs folder on $REMOTE using $QUICKCLUSTERKEY

#ENABLE NFS PORTS ON DEFAULT FIREWALL
export DZONE=$($SSH -C 'sudo firewall-cmd --get-default-zone')
export ip=$($SSH -C 'hostname -i')
echo -e default firewall zone: $DZONE
echo -e upi-0 IP: $ip
$SSH -C sudo firewall-cmd --zone=$DZONE --add-port=$PORT/tcp --permanent
$SSH -C sudo firewall-cmd --reload

if $SSH -C "sudo firewall-cmd --info-zone=$DZONE"|grep $PORT > /dev/null; then
  echo "port $PORT/tcp successfully added to firewall";
else 
  echo "Cannot add port $PORT/tcp exiting";
  exit 1;
fi

#CREATE DIRS AND ADD IT TO /etc/exports
$SSH -C 'sudo mkdir -p /opt/nfs ; sudo mkdir -p /opt/nfs/pv0001 > /dev/null 2>&1;'
$SSH -C 'sudo chmod -R 0777 /opt/nfs/*'
$SSH -C 'cp /etc/exports ./exports; echo "/opt/nfs/pv0001 *(no_root_squash,rw,sync)" >> ./exports ; sudo cp ./exports /etc/exports ; rm ./exports'

#START AND ENABLE RPCBIND AND NFS SERVICES
$SSH -C "sudo systemctl restart rpcbind && sudo systemctl restart nfs"

set +e
oc create namespace $NAMESPACE
oc label namespace $NAMESPACE "openshift.io/cluster-monitoring=true" --overwrite=true
oc project $NAMESPACE

oc apply -f $ROOT/hack/quickcluster/templates/rbac.yaml
oc adm policy add-scc-to-user hostmount-anyuid system:serviceaccount:$NAMESPACE:nfs-client-provisioner
export REMOTE
envsubst < $ROOT/hack/quickcluster/templates/deployment.yaml | oc apply -f -
oc -n $NAMESPACE wait --for=condition=ready pod --all
oc apply -f $ROOT/hack/quickcluster/templates/storageClass.yaml

unset NAMESPACE
unset QUICKCLUSTERKEY
unset REMOTE
unset DZONE
unset ip
unset SSH
unset PORT

exit 0
