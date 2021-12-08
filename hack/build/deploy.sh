#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

#   deploy the image passed as first  parameter

DEPLOY=$2 
if [ -z "$DEPLOY" ]
then 
echo "Using Built In Deployment YAML"
cat > $SCRIPTDIR/tmp.deployment.yaml <<DEPLOYMENT
apiVersion: apps/v1
kind: Deployment
metadata:
  name: REPLACE_ME_DEPLOY_NAME
spec:
  replicas: 1
  selector:
    matchLabels:
      app: REPLACE_ME_APP_LABEL
  template:
    metadata:
      labels:
        app: REPLACE_ME_APP_LABEL
    spec: 
      containers:
        - name: container-image
          image: REPLACE_ME_CONTAINER_IMAGENAME
          resources:
            limits:
              cpu: "200m"
              memory: "512Mi"
            requests:
              cpu: "100m"
              memory: "512Mi"
          imagePullPolicy: Always
          ports:
            - containerPort: 8080
          readinessProbe:
            tcpSocket:
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 10
          livenessProbe:
            initialDelaySeconds: 120
            periodSeconds: 5
            httpGet:
              path: /
              port: 8080  
DEPLOYMENT
else
echo "Using Deploy $DEPLOY from build results - This typically comes from the devfile."
cp $DEPLOY $SCRIPTDIR/tmp.deployment.yaml 
fi

#find port in deployment for patching
PORT=$(yq e '.spec.template.spec.containers[].ports[].containerPort' $SCRIPTDIR/tmp.deployment.yaml)
if [ -z "$PORT" ]
then
  PORT=8080
fi 

cat > $SCRIPTDIR/tmp.service.yaml <<SERVICE
apiVersion: v1
kind: Service
metadata:
  name: container-service 
spec:
  selector:
    app: REPLACE_ME_SVC_APP_SELECTOR
  ports:
  - port: REPLACE_ME_TARGET_PORT
    targetPort: REPLACE_ME_TARGET_PORT 
SERVICE

cat > $SCRIPTDIR/tmp.route.yaml <<ROUTE
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: REPLACE_ME_RT_NAME
spec:
  port:
    targetPort: REPLACE_ME_TARGET_PORT
  tls:
    insecureEdgeTerminationPolicy: Redirect
    termination: edge
  to:
    kind: Service
    name: container-service
    weight: 100
ROUTE

CONTAINER=$1 
if [ -z "$CONTAINER" ]
then
      echo Missing CONTAINER Image URL to Run
      exit -1 
fi
IMAGE=$(basename $CONTAINER)
# route is just the base image name
ROUTE=$(echo $IMAGE | cut -d ':' -f 1)

echo "Running image: $IMAGE"  
echo 
APP=$ROUTE-app
 
yq -M e ".metadata.name=\"$ROUTE-deployment\"" $SCRIPTDIR/tmp.deployment.yaml  | \
 yq -M e ".spec.selector.matchLabels.app=\"$APP\"" - |  \
 yq -M e ".spec.template.metadata.labels.app=\"$APP\"" - |  \
 yq -M e ".spec.template.spec.containers[0].image=\"$CONTAINER\"" -  | \
 tee $SCRIPTDIR/dbg.deployment.yaml | 
 oc apply -f -
 
yq -M e ".metadata.name=\"$ROUTE-service\"" $SCRIPTDIR/tmp.service.yaml  | \
 yq -M e ".spec.selector.app=\"$APP\"" -  |  \
 yq -M e ".spec.ports[0].port=$PORT" -  |  \
 yq -M e ".spec.ports[0].targetPort=$PORT" -  |  \
 tee $SCRIPTDIR/dbg.service.yaml | 
 oc apply -f - 
 
yq -M e ".metadata.name=\"$ROUTE\"" $SCRIPTDIR/tmp.route.yaml  | \
 yq -M e ".spec.to.name=\"$ROUTE-service\"" - |  \
 yq -M e ".spec.port.targetPort=$PORT" -  |  \
 tee $SCRIPTDIR/dbg.route.yaml | 
 oc apply -f -

rm -rf $SCRIPTDIR/tmp.* 
rm -rf $SCRIPTDIR/dbg.* 
RT=$( oc get route $ROUTE  -o yaml | yq e '.spec.host' -)
echo "Find your app at https://$RT"
echo 