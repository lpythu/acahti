package order

import "sort"

// Names returns nodes in topological order (parents before children).
// deps maps a node to the names it depends on. Missing parents are ignored.
// Ready nodes are emitted by name so the order is stable and generic.
func Names(nodes []string, deps map[string][]string) []string {
	present := map[string]bool{}
	for _, n := range nodes {
		present[n] = true
	}
	indeg := map[string]int{}
	children := map[string][]string{}
	for _, n := range nodes {
		indeg[n] = 0
	}
	for _, n := range nodes {
		seen := map[string]bool{}
		for _, p := range deps[n] {
			if !present[p] || p == n || seen[p] {
				continue
			}
			seen[p] = true
			indeg[n]++
			children[p] = append(children[p], n)
		}
	}
	ready := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if indeg[n] == 0 {
			ready = append(ready, n)
		}
	}
	sort.Strings(ready)
	out := make([]string, 0, len(nodes))
	for len(ready) > 0 {
		n := ready[0]
		ready = ready[1:]
		out = append(out, n)
		next := append([]string{}, children[n]...)
		sort.Strings(next)
		for _, c := range next {
			indeg[c]--
			if indeg[c] == 0 {
				ready = append(ready, c)
			}
		}
		sort.Strings(ready)
	}
	if len(out) == len(nodes) {
		return out
	}
	left := make([]string, 0, len(nodes)-len(out))
	seen := map[string]bool{}
	for _, n := range out {
		seen[n] = true
	}
	for _, n := range nodes {
		if !seen[n] {
			left = append(left, n)
		}
	}
	sort.Strings(left)
	return append(out, left...)
}
