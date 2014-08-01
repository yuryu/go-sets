// Package stringset implements a lightweight set-of-strings type based on Go's
// built-in map type.  A Set provides some convenience methods for common set
// operations.  A nil Set is ready for use as an empty set.  The basic set
// methods (Diff, Intersect, Union, IsSubset) do not mutate their arguments.
//
// There are also mutating operations (Add, Discard, Update, Remove) that
// modify their receiver in-place.
//
// Example:
//    one := stringset.New("one")
//    none := one.Intersect(nil)
//    nat := stringset.New("0", "1", "2", "3", "4")
//    some := nat.Union(one)
//    fmt.Println(some)
//     => {"0", "1", "2", "3", "4", "one"}
//
//    nat.Remove("2", "4")
//    fmt.Println(nat)
//     => {"0", "1", "3"}
//
//    one.Add("one", "perfect", "question")
//    fmt.Println(one)
//     => {"one", "perfect", "question"}
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

// String implements the fmt.Stringer interface.
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

// Len returns the number of elements in the set.
func (s Set) Len() int { return len(s) }

// Keys returns a lexicographically sorted slice of the elements in the set.
func (s Set) Keys() []string {
	var keys []string
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Contains reports whether the set contains the given string.
func (s Set) Contains(str string) bool {
	_, ok := s[str]
	return ok
}

// IsSubset reports whether s1 is a subset of s2.
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

// Empty reports whether the set is empty.
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
// allocated that is a copy of s2.  Reports whether anything was added.
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

// Add adds the specified strings to *set in-place.  If *set == nil, allocates
// a new set equivalent to New(ss...).  Reports whether anything was added.
func (set *Set) Add(ss ...string) bool {
	in := len(*set)
	if *set == nil {
		*set = make(Set)
	}
	for _, s := range ss {
		(*set)[s] = struct{}{}
	}
	return len(*set) != in
}

// Remove removes the elements of s2 from s1 in-place.  Reports whether
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

// Discard removes the elements of ss from set in-place.  Reports whether
// anything was removed.
//
// Equivalent to set = set.Diff(New(ss...)), but does not allocate a set for ss.
func (set Set) Discard(ss ...string) bool {
	in := set.Len()
	if !set.Empty() {
		for _, s := range ss {
			delete(set, s)
		}
	}
	return set.Len() != in
}
