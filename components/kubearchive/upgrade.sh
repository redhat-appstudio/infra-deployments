#!/bin/bash -e

# Usage: upgrade.sh <old-version> <new-version> [<new-migration-version>]
# Example: upgrade.sh v1.21.3 v1.21.4
# Example: upgrade.sh v1.21.4 v1.22.0 14

OLD_VERSION=$1
NEW_VERSION=$2
NEW_MIGRATION_VERSION=$3

echo "Upgrading KubeArchive from $OLD_VERSION to $NEW_VERSION..."

curl -Lo components/kubearchive/development/kubearchive.yaml https://github.com/kubearchive/kubearchive/releases/download/$NEW_VERSION/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-fedora-01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-ocp-p01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-osp-p01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-prd-rh02/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-prd-rh03/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-rhel-p01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/stone-prd-rh01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/stone-prod-p01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/stone-prod-p02/kubearchive.yaml
sed -i "s/$OLD_VERSION/$NEW_VERSION/g" components/kubearchive/production/**/kustomization.yaml components/kubearchive/development/kustomization.yaml

if [ -n "$NEW_MIGRATION_VERSION" ]; then
  DEV_KUSTOMIZATION=components/kubearchive/development/kustomization.yaml
  OLD_MIGRATION_VERSION=$(grep -m1 'MIGRATION_VERSION=' "$DEV_KUSTOMIZATION" | sed 's/.*MIGRATION_VERSION=//')

  echo "Updating schema migration from v$OLD_MIGRATION_VERSION to v$NEW_MIGRATION_VERSION..."

  sed -i "s/MIGRATION_VERSION=$OLD_MIGRATION_VERSION/MIGRATION_VERSION=$NEW_MIGRATION_VERSION/g" "$DEV_KUSTOMIZATION"
  sed -i "s/kubearchive-schema-migration-v$OLD_MIGRATION_VERSION/kubearchive-schema-migration-v$NEW_MIGRATION_VERSION/g" "$DEV_KUSTOMIZATION"
fi
