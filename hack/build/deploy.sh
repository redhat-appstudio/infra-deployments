#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

#   deploy the image passed as first  parameter

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
            initialDelaySeconds: 10
            periodSeconds: 5
            httpGet:
              path: /
              port: 8080  
DEPLOYMENT

cat > $SCRIPTDIR/tmp.service.yaml <<SERVICE
apiVersion: v1
kind: Service
metadata:
  name: container-service 
spec:
  selector:
    app: REPLACE_ME_SVC_APP_SELECTOR
  ports:
  - port: 8080
    targetPort: 8080 
SERVICE

cat > $SCRIPTDIR/tmp.route.yaml <<ROUTE
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: REPLACE_ME_RT_NAME
spec:
  port:
    targetPort: 8080
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

echo running image $IMAGE at $ROUTE  

APP=$ROUTE-app
 
yq -M e ".metadata.name=\"$ROUTE-deployment\"" $SCRIPTDIR/tmp.deployment.yaml  | \
 yq -M e ".spec.selector.matchLabels.app=\"$APP\"" - |  \
 yq -M e ".spec.template.metadata.labels.app=\"$APP\"" - |  \
 yq -M e ".spec.template.spec.containers[0].image=\"$CONTAINER\"" -  | \
 oc apply -f -
 
yq -M e ".metadata.name=\"$ROUTE-service\"" $SCRIPTDIR/tmp.service.yaml  | \
 yq -M e ".spec.selector.app=\"$APP\"" -  |  \
 oc apply -f - 
 
yq -M e ".metadata.name=\"$ROUTE\"" $SCRIPTDIR/tmp.route.yaml  | \
 yq -M e ".spec.to.name=\"$ROUTE-service\"" - |  \
 oc apply -f -

rm -rf $SCRIPTDIR/tmp.* 
RT=$( oc get route $ROUTE  -o yaml | yq e '.spec.host' -)
echo "Find your app at https://$RT"