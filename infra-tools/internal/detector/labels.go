package detector

import (
	"iter"
	"sort"
)

// LabelSet holds the generated labels split by category so callers can decide
// which ones to use.
type LabelSet struct {
	Environments []string // e.g. ["environment/development", "environment/production"]
	Clusters     []string // e.g. ["cluster/kflux-ocp-p01"]
}

// All returns an iterator over every label in the set (environments + clusters),
// yielded in sorted order.
func (ls LabelSet) All() iter.Seq[string] {
	return func(yield func(string) bool) {
		all := make([]string, 0, len(ls.Environments)+len(ls.Clusters))
		all = append(all, ls.Environments...)
		all = append(all, ls.Clusters...)
		sort.Strings(all)
		for _, l := range all {
			if !yield(l) {
				return
			}
		}
	}
}

// Labels builds the LabelSet from the detection result.
func (r *Result) Labels() LabelSet {
	var ls LabelSet
	for env := range r.AffectedEnvironments {
		ls.Environments = append(ls.Environments, "environment/"+string(env))
	}
	for cluster := range r.AffectedClusters {
		ls.Clusters = append(ls.Clusters, "cluster/"+cluster)
	}
	sort.Strings(ls.Environments)
	sort.Strings(ls.Clusters)
	return ls
}
