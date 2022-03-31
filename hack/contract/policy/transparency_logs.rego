
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

#
# Ensure that we can see the data from the log entry
#
# If it's missing we can assume it isn't available or somehow
# rekor-cli didn't confirm its validity
#
# TODO:
# - How to pass back a more useful error message including the specific
#   reason for the failure. Not sure how to do that when using "every" like this
# - Could we convert the logic from "not every true" to "any not true" and
#   stop using every entirely? (Better write some tests before trying that.)
#
deny[{ "msg": msg }] {
  not all_transparency_log_entries_are_present
  msg := "One or more transparency log entries were unavailable!"
}

deny[{ "msg": msg }] {
  not all_transparency_log_entries_appear_sane
  msg := "One or more transparency log entries seems to be invalid!"
}

all_transparency_log_entries_are_present {
  every tr in data.k8s.TaskRun {
    url := tr.metadata.annotations["chains.tekton.dev/transparency"]
    rekor_host := split(url, "/")[2]
    log_index := split(url, "=")[1]

    # This should be true only if log entry is present
    data.rekor[rekor_host].logIndex[log_index]
  }
}

all_transparency_log_entries_appear_sane {
  #
  # TODO: This is mostly copy pasted from above. Refactor so we
  # aren't repeating the code
  #
  every tr in data.k8s.TaskRun {
    url := tr.metadata.annotations["chains.tekton.dev/transparency"]
    rekor_host := split(url, "/")[2]
    log_index := split(url, "=")[1]
    log_entry_data := data.rekor[rekor_host].logIndex[log_index]

    # The log index in the data matches what we expected
    # (Not sure if there's a better way to compare the int and the string, but this works)
    log_index == sprintf("%d",[log_entry_data.LogIndex])

    # It has a body at least
    log_entry_data.Body
  }
}
