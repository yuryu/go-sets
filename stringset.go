// Package stringset implements a lightweight (finite) set-of-strings type
// based on Go's built-in map type.  A Set provides some convenience methods
// for common set operations.  A nil Set is ready for use as an empty set.  The
// basic set methods (Diff, Intersect, Union, IsSubset, Map, Filter, Partition)
// do not mutate their arguments.
//
// There are also mutating operations (Add, Discard, Update, Remove) that
// modify their receiver in-place.
//
// A Set can also be traversed and modified using the normal map operations.
// Being a map, a Set is not safe for concurrent access by multiple goroutines.
//
// Example:
//    one := stringset.New("one") // ⇒ {"one"}
//    none := one.Intersect(nil)  // ⇒ ø
//    nat := stringset.New("0", "1", "2", "3", "4")
//    some := nat.Union(one)
//     // ⇒ {"0", "1", "2", "3", "4", "one"}
//
//    nat.Remove("2", "4")
//    fmt.Println(nat)
//     // ⇒ {"0", "1", "3"}
//
//    one.Add("one", "perfect", "question")
//    fmt.Println(one)
//     // ⇒ {"one", "perfect", "question"}
//
package stringset

import (
	"sort"
	"strconv"
	"strings"
)

// A Set represents a set of string values.  A nil Set is a valid
// representation of an empty set.
type Set map[string]struct{}

// String implements the fmt.Stringer interface.  It renders s in standard set
// notation, e.g., ø for an empty set, {"a", "b", "c"} for a nonempty one.
func (s Set) String() string {
	if s.Empty() {
		return "ø"
	}
	keys := make([]string, len(s))
	for i, k := range s.Keys() {
		keys[i] = strconv.Quote(k)
	}
	return "{" + strings.Join(keys, ", ") + "}"
}

// New returns a new set containing exactly the specified elements.  Returns
// nil if no elements are specified.
func New(ss ...string) Set {
	if len(ss) == 0 {
		return nil
	}
	set := make(Set)
	for _, s := range ss {
		set[s] = struct{}{}
	}
	return set
}

// Len returns the number of elements in s.
func (s Set) Len() int { return len(s) }

// Keys returns a lexicographically sorted slice of the elements in s.
func (s Set) Keys() []string {
	var keys []string
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Contains reports whether s contains one or more of the given strings.
// It is equivalent in meaning to
//   !s.Intersect(stringset.New(strs...)).Empty()
// but does not construct an intermediate set.
func (s Set) Contains(strs ...string) bool {
	for _, key := range strs {
		if _, ok := s[key]; ok {
			return true
		}
	}
	return false
}

// IsSubset reports whether s1 is a subset of s2, s1 ⊆ s2.
func (s1 Set) IsSubset(s2 Set) bool {
	if s1.Empty() {
		return true
	} else if s2.Empty() {
		return false
	}
	for k := range s1 {
		if _, ok := s2[k]; !ok {
			return false
		}
	}
	return true
}

// Equals reports whether s1 is equal to s2, having exactly the same elements.
func (s1 Set) Equals(s2 Set) bool { return s1.IsSubset(s2) && s2.IsSubset(s1) }

// Empty reports whether s is empty.
func (s Set) Empty() bool { return len(s) == 0 }

// Union constructs the union s1 ∪ s2.
func (s1 Set) Union(s2 Set) Set {
	if s1.Empty() {
		return s2
	} else if s2.Empty() {
		return s1
	}
	set := make(Set)
	for k := range s1 {
		set[k] = struct{}{}
	}
	for k := range s2 {
		set[k] = struct{}{}
	}
	return set
}

// Intersect constructs the intersection s1 ∩ s2.
func (s1 Set) Intersect(s2 Set) Set {
	if s1.Empty() || s2.Empty() {
		return nil
	}
	var set Set
	for k := range s1 {
		if _, ok := s2[k]; ok {
			if set == nil {
				set = make(Set)
			}
			set[k] = struct{}{}
		}
	}
	return set
}

// Diff constructs the set difference s1 \ s2.
func (s1 Set) Diff(s2 Set) Set {
	if s1.Empty() || s2.Empty() {
		return s1
	}
	var set Set
	for k := range s1 {
		if _, ok := s2[k]; !ok {
			if set == nil {
				set = make(Set)
			}
			set[k] = struct{}{}
		}
	}
	return set
}

// Update adds the elements of s2 to *s1 in-place. If *s1 == nil a new set is
// allocated that is a copy of s2 and reports whether anything was added.
func (s1 *Set) Update(s2 Set) bool {
	in := len(*s1)
	if *s1 == nil {
		*s1 = make(Set)
	}
	for k := range s2 {
		(*s1)[k] = struct{}{}
	}
	return len(*s1) != in
}

// Add adds the specified strings to *s in-place and reports whether anything
// was added.  If *s == nil, a new set equivalent to New(ss...) is stored in *s.
func (s *Set) Add(ss ...string) bool {
	in := len(*s)
	if *s == nil {
		*s = make(Set)
	}
	for _, key := range ss {
		(*s)[key] = struct{}{}
	}
	return len(*s) != in
}

// Remove removes the elements of s2 from s1 in-place and reports whether
// anything was removed.
//
// Equivalent to s1 = s1.Diff(s2), but does not allocate a new set.
func (s1 Set) Remove(s2 Set) bool {
	in := s1.Len()
	if !s1.Empty() {
		for k := range s2 {
			delete(s1, k)
		}
	}
	return s1.Len() != in
}

// Discard removes the elements of ss from s in-place and reports whether
// anything was removed.
//
// Equivalent to s = s.Diff(New(ss...)), but does not allocate an intermediate
// set for ss.
func (s Set) Discard(ss ...string) bool {
	in := s.Len()
	if !s.Empty() {
		for _, key := range ss {
			delete(s, key)
		}
	}
	return s.Len() != in
}
