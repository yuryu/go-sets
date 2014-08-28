package stringset

// Map returns the Set that results from applying f to each element of s.
func (s Set) Map(f func(string) string) Set {
	var out Set
	for k := range s {
		out.Add(f(k))
	}
	return out
}

// Filter returns the subset of s for which f returns true.
func (s Set) Filter(f func(string) bool) Set {
	var out Set
	for k := range s {
		if f(k) {
			out.Add(k)
		}
	}
	return out
}

// Partition returns two disjoint sets, yes containing the subset of s for
// which f returns true and no containing the subset for which f returns false.
func (s Set) Partition(f func(string) bool) (yes, no Set) {
	for k := range s {
		if f(k) {
			yes.Add(k)
		} else {
			no.Add(k)
		}
	}
	return
}

// Select returns an element of s for which f returns true, if one exists.  The
// second result reports whether such an element was found.  If f == nil,
// selects an arbitrary element of s.
func (s Set) Select(f func(string) bool) (string, bool) {
	for k := range s {
		if f == nil || f(k) {
			return k, true
		}
	}
	return "", false
}
