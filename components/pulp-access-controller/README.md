# Pulp Access Controller

The `pulp-access-controller` is a CRD/Controller which creates namespace secrets that provide access to Pulp over TLS.

It is similar in concept to `image-controller`. Image Controller provisions quay repos and creates secrets for user pipelines to use. Pulp Access Controller provisions pulp tenants and provides secrets for user pipelines to access those.
