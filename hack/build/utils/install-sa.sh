 
read -r -d '' SA <<'SA'
apiVersion: v1
kind: ServiceAccount
metadata:
  name: build-service-sa
secrets:
  - name: quay-registry-secret
SA
 
echo "$SA" | oc apply -f -

