# multi-platform-controller-rd

Ring-deployment copy of `multi-platform-controller` used while onboarding
staging to the Konflux-hosted ArgoCD instances.

This tree currently contains Ring 0 (development) and Ring 1 (staging) only.
The original `components/multi-platform-controller/` overlay remains in place
until later migration steps remove it.

| Ring | Clusters |
|------|----------|
| ring-0 | development (base only until rd-dev is wired) |
| ring-1 | `stone-stg-rh01`, `stone-stage-p01`, `lightwell-dev` |
