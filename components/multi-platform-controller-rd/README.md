# multi-platform-controller-rd

Ring-deployment copy of `multi-platform-controller` used while onboarding
to the Konflux-hosted ArgoCD instances.

The original `components/multi-platform-controller/` overlay remains in place
until later migration steps remove it.

| Ring | Clusters |
|------|----------|
| ring-0 | development (base only until rd-dev is wired) |
| ring-1 | `stone-stg-rh01`, `stone-stage-p01`, `lightwell-dev` |
| ring-2 | `kflux-fedora-01`, `kflux-ocp-p01`, `kflux-osp-p01`, `kflux-prd-rh03`, `kflux-rhel-p01`, `stone-prod-p01`, `kflux-lw-p01` |
| ring-3 | `stone-prd-rh01`, `stone-prod-p02` |
| ring-4 | `kflux-prd-rh02` |
