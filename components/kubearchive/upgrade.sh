#!/bin/bash -e

echo "Upgrading KubeArchive from $1 to $2..."

curl -Lo components/kubearchive/development/kubearchive.yaml https://github.com/kubearchive/kubearchive/releases/download/$2/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-ocp-p01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-osp-p01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-prd-rh02/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-prd-rh03/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/kflux-rhel-p01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/stone-prd-rh01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/stone-prod-p01/kubearchive.yaml
cp components/kubearchive/development/kubearchive.yaml components/kubearchive/production/stone-prod-p02/kubearchive.yaml
sed -i "s/$1/$2/g" components/kubearchive/production/**/kustomization.yaml components/kubearchive/development/kustomization.yaml
