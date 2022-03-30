
package hacbs.contract.transparency_log_attestations

import future.keywords.every

#
# Just experimenting here with decoding attestations
# from the rekor transparency logs.
#
# (It might be more convenient to extract them in the
# fetch-data script for easy access, but here we'll try
# doing that in pure rego.)
#
#
deny[{ "msg": msg }] {
  attestation := all_attestations[_]
  # An example, not sure if we would do this check for real:
  attestation.data._type != "https://in-toto.io/Statement/v0.1"

  msg := sprintf(
    "Unexpected attestation type '%s' in log index %s on %s",
    [attestation.data._type, attestation.log_index, attestation.rekor_host])
}

all_attestations = attestations {
  attestations := [attestation | rekor_hosts := data.rekor[rekor_host]
                                 log_entry := rekor_hosts.logIndex[log_index]
                                 attestation := {
                                   "rekor_host" : rekor_host,
                                   "log_index"  : log_index,
                                   "data": decode_attestation(log_entry.Attestation)
                                 }]
}

# Extract an attestation from the logIndex
decode_attestation(encoded_attestation) = result {
  result := json.unmarshal(base64.decode(encoded_attestation))
}

deny[{ "msg": msg }] {

  attestation := all_attestations[_]

  some step_index
  step = attestation.data.predicate.buildConfig.steps[step_index]
  registry := concat("/", array.slice(split(step.environment.image, "/"), 0, 2))
  not registry_is_allowed(registry)

  msg := sprintf("Step %d has disallowed registry '%s'", [step_index, registry])
}

registry_is_allowed(registry) {
  allowed_registries[_] = registry
}

# Hypothetical list of allowed registries
# for task images used to run task steps
#
allowed_registries = [
  "quay.io/redhat-appstudio",
  "registry.redhat.io/openshift-pipelines"
]
