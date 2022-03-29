
package hacbs.contract.main
import data.hacbs.contract.k8s_sanity
import data.hacbs.contract.transparency_urls
import data.hacbs.contract.transparency_log_attestations

deny[msg] {
  count(k8s_sanity.deny[msg]) > 0
}

deny[msg] {
  count(transparency_urls.deny[msg]) > 0
}

deny[msg] {
  count(transparency_log_attestations.deny[msg]) > 0
}
