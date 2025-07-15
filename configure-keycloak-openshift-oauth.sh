#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Configuring Keycloak as OpenShift Identity Provider ===${NC}"

# Get cluster info
echo -e "${YELLOW}Getting cluster information...${NC}"
CLUSTER_DOMAIN=$(kubectl get ingresses.config.openshift.io/cluster -o jsonpath='{.spec.domain}')
API_SERVER=$(kubectl get infrastructure/cluster -o jsonpath='{.status.apiServerURL}')

echo "Cluster domain: $CLUSTER_DOMAIN"
echo "API server: $API_SERVER"

# Get Keycloak URL
echo -e "${YELLOW}Getting Keycloak information...${NC}"
KEYCLOAK_URL=https://$(kubectl get route keycloak -n dev-sso -o jsonpath='{.spec.host}')
KEYCLOAK_REALM_URL="$KEYCLOAK_URL/auth/realms/redhat-external"

echo "Keycloak URL: $KEYCLOAK_URL"
echo "Keycloak realm URL: $KEYCLOAK_REALM_URL"

# Get admin credentials
echo -e "${YELLOW}Getting Keycloak admin credentials...${NC}"
ADMIN_USERNAME=$(kubectl get secret credential-keycloak -n dev-sso -o jsonpath='{.data.ADMIN_USERNAME}' | base64 -d)
ADMIN_PASSWORD=$(kubectl get secret credential-keycloak -n dev-sso -o jsonpath='{.data.ADMIN_PASSWORD}' | base64 -d)

echo "Admin username: $ADMIN_USERNAME"
echo "Admin password: [HIDDEN]"

# OAuth redirect URI
OAUTH_REDIRECT_URI="https://oauth-openshift.apps.$CLUSTER_DOMAIN/oauth2callback/keycloak"
echo "OAuth redirect URI: $OAUTH_REDIRECT_URI"

echo -e "${GREEN}=== Manual Steps Required ===${NC}"
echo ""
echo "1. Access Keycloak Admin Console:"
echo "   URL: $KEYCLOAK_URL/auth/admin"
echo "   Username: $ADMIN_USERNAME"
echo "   Password: $ADMIN_PASSWORD"
echo ""
echo "2. Navigate to 'redhat-external' realm"
echo ""
echo "3. Create OpenShift OAuth client:"
echo "   - Go to Clients → Create Client"
echo "   - Client ID: openshift-oauth"
echo "   - Client Protocol: openid-connect"
echo "   - Valid Redirect URIs: $OAUTH_REDIRECT_URI"
echo "   - Web Origins: https://oauth-openshift.apps.$CLUSTER_DOMAIN"
echo "   - Access Type: confidential"
echo "   - Standard Flow Enabled: ON"
echo "   - Direct Access Grants Enabled: OFF"
echo ""
echo "4. Get the client secret from Credentials tab"
echo ""
echo "5. Apply the OAuth configuration:"
echo ""

# Generate the OAuth configuration
cat > keycloak-oauth-config.yaml << EOF
apiVersion: v1
kind: Secret
metadata:
  name: keycloak-secret
  namespace: openshift-config
type: Opaque
stringData:
  clientSecret: "YOUR_CLIENT_SECRET_FROM_KEYCLOAK"
---
apiVersion: config.openshift.io/v1
kind: OAuth
metadata:
  name: cluster
spec:
  identityProviders:
  - name: keycloak
    type: OpenID
    openID:
      clientID: openshift-oauth
      clientSecret:
        name: keycloak-secret
      issuer: $KEYCLOAK_REALM_URL
      claims:
        preferredUsername:
        - preferred_username
        name:
        - name
        email:
        - email
        groups:
        - groups
    mappingMethod: claim
EOF

echo "   Replace YOUR_CLIENT_SECRET_FROM_KEYCLOAK in keycloak-oauth-config.yaml"
echo "   Then run: kubectl apply -f keycloak-oauth-config.yaml"
echo ""
echo -e "${GREEN}Configuration file 'keycloak-oauth-config.yaml' has been generated.${NC}"
echo ""
echo -e "${YELLOW}After applying the configuration:${NC}"
echo "- OAuth pods will restart automatically"
echo "- You can login to OpenShift using Keycloak credentials"
echo "- Access the OpenShift console and you should see 'keycloak' as a login option"
echo ""
echo -e "${GREEN}=== Additional Configuration ===${NC}"
echo ""
echo "To add users to Keycloak:"
echo "1. Go to Keycloak Admin Console → Users → Add user"
echo "2. Set username, email, and enable the user"
echo "3. Go to Credentials tab → Set password"
echo "4. The user can now login to OpenShift using Keycloak"
echo ""
echo "To configure user groups/roles:"
echo "1. Create groups in Keycloak (Groups → New)"
echo "2. Add users to groups"
echo "3. Create group mappers for the openshift-oauth client"
echo "4. Users will inherit OpenShift permissions based on group membership" 