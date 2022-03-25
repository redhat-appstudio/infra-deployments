#!/bin/bash

set -e

trap 'last_command=$current_command; current_command=$BASH_COMMAND' DEBUG
trap 'echo "\"${last_command}\" command failed with exit code $?."' EXIT

export QUICKLABKEY=/tmp/quicklab.key

if [ -z "$1" ]; then
    echo "usage: setup-nfs-quicklab.sh <upi host name>"
    exit 1
fi

REMOTE=$1
QOCP="https://$(echo ${REMOTE} |sed s/upi-0/api/):6443"

scp -o StrictHostKeyChecking=no -i $QUICKLABKEY quicklab@$REMOTE:/home/quicklab/oc4/auth/kubeconfig /tmp/kubeconfig
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

# default to 1 PV
[[ -z $2 ]] && PVN="1" || PVN=$2
PORT=2049

wget -q https://gitlab.cee.redhat.com/cee_ops/quicklab/raw/master/docs/quicklab.key -O $QUICKLABKEY
chmod 600 $QUICKLABKEY

SSH="ssh -o StrictHostKeyChecking=no -i $QUICKLABKEY quicklab@$REMOTE"

echo -e Adding $PVN nfs folders on $REMOTE using $QUICKLABKEY

#ENABLE NFS PORTS ON DEFAULT FIREWALL
export DZONE=$($SSH -C 'sudo firewall-cmd --get-default-zone')
export ip=$($SSH -C "sudo ifconfig"|grep inet|head -1|xargs|cut -d ' ' -f2)
echo -e default firewall zone: $DZONE
echo -e upi-0 IP: $ip
$SSH -C sudo firewall-cmd --zone=$DZONE --add-port=$PORT/tcp --permanent
$SSH -C sudo firewall-cmd --reload

if $SSH -C "sudo firewall-cmd --info-zone=$DZONE"|grep $PORT > /dev/null; then
  echo "port 2049/tcp successfully added to firewall";
else 
  echo "Cannot add port 2049/tcp exiting";
  exit 1;
fi

#CREATE DIRS AND ADD IT TO /etc/exports
#$SSH -C 'sudo mkdir -p /opt/nfs/pv000{1..'${PVN}'};'
DIRS=$(echo 'sudo mkdir -p /opt/nfs; INDEX=1; COUNT=0; NAMES=(); while [ $COUNT -lt '$PVN' ]; do   if ! sudo mkdir /opt/nfs/pv$(printf "%04i" ${INDEX}) > /dev/null 2>&1; then     INDEX=$(( INDEX + 1 ));   else  NAMES+=(/opt/nfs/pv$(printf "%04i" ${INDEX})); INDEX=$(( INDEX + 1 )); COUNT=$(( COUNT + 1 ));   fi; done; echo ${NAMES[*]}'|$SSH)
$SSH -C 'sudo chmod -R 0777 /opt/nfs/*'
#echo 'cp /etc/exports ./exports; for i in {1..'$PVN'}; do  echo "/opt/nfs/pv000$i *(no_root_squash,rw,sync)" >> ./exports; done; sudo cp ./exports /etc/exports; rm ./exports'|$SSH
echo 'cp /etc/exports ./exports; for i in '${DIRS}'; do  echo "$i *(no_root_squash,rw,sync)" >> ./exports; done; sudo cp ./exports /etc/exports; rm ./exports'|$SSH

#START AND ENABLE RPCBIND AND NFS SERVICES
$SSH -C "sudo systemctl restart rpcbind && sudo systemctl restart nfs"

set +e
oc create namespace openshift-nfs-storage
oc label namespace openshift-nfs-storage "openshift.io/cluster-monitoring=true" --overwrite=true
oc project openshift-nfs-storage
NAMESPACE=`oc project -q`
oc apply -f templates/rbac.yaml
oc adm policy add-scc-to-user hostmount-anyuid system:serviceaccount:$NAMESPACE:nfs-client-provisioner
echo -e "$(eval "echo -e \"$(<templates/deployment.yaml)\"")"|oc create -f -
oc -n $NAMESPACE wait --for=condition=ready pod --all
oc apply -f templates/storageClass.yaml
set -e

unset QUICKLABKEY
unset REMOTE
unset PVN
unset DZONE
unset ip
unset SSH
unset PORT
