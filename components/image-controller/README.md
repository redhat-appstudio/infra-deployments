---
title: Image Controller
---

Deployment of [image-controller](https://github.com/redhat-appstudio/image-controller)

## Image Controller secrets

List of secrets:

| Name | Source | Description |
| -- | -- | -- |
| quaytoken | appsre vault | Secret containing 'organization' and 'quaytoken' with permissions to create repositories |

Rotation rule: Secrets must be rotated within 7 days after someone with access leaves the organization. Secrets older than one year should be rotated.

### Instructions for rotation of quaytoken

Prerequisite:
- User must be owner of quay.io organization [redhat-user-workloads](https://quay.io/organization/redhat-user-workloads) for production or [redhat-user-workloads-stage](https://quay.io/organization/redhat-user-workloads-stage) for stage instance.

Process for production instance:
1. Reset Client Secret on [Application Oauth page](https://quay.io/organization/redhat-user-workloads/application/VMLM8D3FUBUGMBMY173Z?tab=oauth)
2. Generate new Token on [Application generate token page](https://quay.io/organization/redhat-user-workloads/application/VMLM8D3FUBUGMBMY173Z?tab=gen-token), with permissions:
  - Administer Organization 
  - Administer Repositories
  - Create Repositories
3. Put token from step 2. to app-sre vault to `stonesoup/production/build/image-controller`

Process for stage instance:
1. Reset Client Secret on [Application Oauth page](https://quay.io/organization/redhat-user-workloads-stage/application/259WVA0L323BVTQCQZ9B?tab=oauth)
2. Generate new Token on [Application generate token page](https://quay.io/organization/redhat-user-workloads-stage/application/259WVA0L323BVTQCQZ9B?tab=gen-token), with permissions:
  - Administer Organization 
  - Administer Repositories
  - Create Repositories
3. Put token from step 2. to app-sre vault to `stonesoup/staging/build/image-controller`