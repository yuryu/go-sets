// Package stringset implements a lightweight (finite) set-of-strings type
// based on Go's built-in map type.  A Set provides some convenience methods
// for common set operations.  A nil Set is ready for use as an empty set.  The
// basic set methods (Diff, Intersect, Union, IsSubset, Map, Choose, Partition)
// do not mutate their arguments.
//
// There are also mutating operations (Add, Discard, Pop, Remove, Update) that
// modify their receiver in-place.
//
// A Set can also be traversed and modified using the normal map operations.
// Being a map, a Set is not safe for concurrent access by multiple goroutines
// unless all the concurrent accesses are reads.
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
	"reflect"
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

// Clone returns a new Set distinct from s, containing the same keys.
func (s Set) Clone() Set {
	var c Set
	c.Update(s)
	return c
}

// ContainsAny reports whether s contains one or more of the given strings.
// It is equivalent in meaning to
//   s.Intersects(stringset.New(strs...))
// but does not construct an intermediate set.
func (s Set) ContainsAny(strs ...string) bool {
	for _, key := range strs {
		if _, ok := s[key]; ok {
			return true
		}
	}
	return false
}

// ContainsAll reports whether s contains all the given strings.
// It is equivalent in meaning to
//   New(strs...).IsSubset(s)
// but does not construct an intermediate set.
func (s Set) ContainsAll(strs ...string) bool {
	for _, str := range strs {
		if _, ok := s[str]; !ok {
			return false
		}
	}
	return true
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

// Intersects reports whether the intersection s1 ∩ s2 is non-empty, without
// explicitly constructing the intersection.
func (s1 Set) Intersects(s2 Set) bool {
	a, b := s1, s2
	if len(b) < len(a) {
		a, b = b, a // Iterate over the smaller set
	}
	for k := range a {
		if _, ok := b[k]; ok {
			return true
		}
	}
	return false
}

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

// SymDiff constructs the symmetric difference s1 ∆ s2.
// It is equivalent in meaning to (s1 ∪ s2) \ (s1 ∩ s2).
func (s1 Set) SymDiff(s2 Set) Set {
	return s1.Union(s2).Diff(s1.Intersect(s2))
}

// Update adds the elements of s2 to *s1 in-place, and reports whether anything
// was added.
// If *s1 == nil and s2 ≠ ø, a new set is allocated that is a copy of s2.
func (s1 *Set) Update(s2 Set) bool {
	in := len(*s1)
	if *s1 == nil && len(s2) > 0 {
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
// Equivalent to s.Remove(New(ss...)), but does not allocate an intermediate
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

// Index returns the first offset of needle in ss, if it occurs; otherwise -1.
func Index(needle string, ss ...string) int {
	for i, s := range ss {
		if s == needle {
			return i
		}
	}
	return -1
}

// A Keyer implements a Keys method that returns the keys of a collection such
// as a map or a Set.
type Keyer interface {
	// Keys returns the keys of the receiver, which may be nil.
	Keys() []string
}

// Keys returns a slice of string keys from v, which must either be a Keyer or
// have type string, []string or map[string]T. It will panic if the type of v
// does not have one of these forms. If v is a map value, its keys will be
// returned in lexicographic order as defined by sort.Strings.
func Keys(v interface{}) []string {
	switch t := v.(type) {
	case Keyer:
		return t.Keys()
	case []string:
		return t
	case string:
		return []string{t}
	}
	var keys []string
	for _, key := range reflect.ValueOf(v).MapKeys() {
		keys = append(keys, key.Interface().(string))
	}
	sort.Strings(keys)
	return keys
}
