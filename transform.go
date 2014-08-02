package stringset

// Map returns the Set that results from applying f to each element of set.
func (set Set) Map(f func(string) string) Set {
	var out Set
	for k := range set {
		out.Add(f(k))
	}
	return out
}

// Filter returns the subset of set for which f returns true.
func (set Set) Filter(f func(string) bool) Set {
	var out Set
	for k := range set {
		if f(k) {
			out.Add(k)
		}
	}
	return out
}

// Partition returns two disjoint sets, yes containing the subset of set for
// which f returns true and no containing the subset for which f returns false.
func (set Set) Partition(f func(string) bool) (yes, no Set) {
	for k := range set {
		if f(k) {
			yes.Add(k)
		} else {
			no.Add(k)
		}
	}
	return
}

// Select returns an element of set for which f returns true, if one exists.
// The second result reports whether such an element was found.  If f == nil,
// selects an arbitrary element of set.
func (set Set) Select(f func(string) bool) (string, bool) {
	for k := range set {
		if f == nil || f(k) {
			return k, true
		}
	}
	return "", false
}
