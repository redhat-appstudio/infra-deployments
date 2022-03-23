
package hacbs.contract.transparency_urls

import future.keywords.every

#
# Sanity check the configuration of tekton-chains in the cluster
#
# Actually this is a weak test since the configuration might have changed
# since the pipeline ran.
#
# Todo:
# - This doesn't fail if the key is entirely absent, which is bad.
# - Maybe check that rest the chains configuration is what we require.
#
deny[{ "msg": msg }] {
  data.k8s.ConfigMap["chains-config"].data["transparency.enabled"] != "true"

  msg := "Chains configuration has transparency disabled"
}

#
# Sanity check that each taskrun has been marked as signed by chains
#
# Also not a strong test, but if it says "failed" then we should be failing.
#
deny[{ "msg": msg }] {
  some tr
  signed_annotation := data.k8s.TaskRun[tr].metadata.annotations["chains.tekton.dev/signed"]
  signed_annotation != "true"

  msg := sprintf("Taskrun %s has signed status of '%s'", [tr, signed_annotation])
}

#
# Ensure that each taskrun has some expected annotations
#
# Todo:
# - The message is not very good here but I don't know how to fix that.
# - Should be able to provide a list of annotations to validate the presence of.
# - How can we do a general purpose "not present or present and != expected_value"?
#
deny[{ "msg": msg }] {
  not all_annotations_present

  msg := "Not all taskruns have 'chains.tekton.dev/transparency' and 'chains.tekton.dev/signed' annotations set"
}

all_annotations_present {
  every tr in data.k8s.TaskRun {
    tr.metadata.annotations["chains.tekton.dev/transparency"]
    tr.metadata.annotations["chains.tekton.dev/signed"]
  }
}

#
# Ensure that each taskrun's transparency url is for the same rekor server
#
# It would be weird if they were different, so this is unlikely to fail,
# but there's no harm in verifying it.
#
deny[{ "msg": msg }] {
  not all_urls_match

  msg := "Not all taskruns have the same transparency log server!"
}

all_urls_match {
  url_base_set := { url_base |
    url := data.k8s.TaskRun[_].metadata.annotations["chains.tekton.dev/transparency"]
    # Strip off the url param which is likely to be "logIndex=1234"
    url_base := split(url, "?")[0]
  }

  # If the set size is one then they all match
  count(url_base_set) == 1
}
