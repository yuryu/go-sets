// Program makeset generates source code for a set package.  The type of the
// elements of the set is determined by a JSON configuration stored either in a
// file (named by the -config flag) or read from standard input.
//
// Usage:
//   makeset -output $DIR -config config.txt
//
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// A Config describes the nature of the set to be constructed.
type Config struct {
	// The name of the resulting set package, e.g., "intset" (required).
	Package string `json:"package"`

	// The name of the type contained in the set, e.g., "int" (required).
	Type string `json:"type"`

	// The spelling of the zero value for the set type, e.g., "0" (required).
	Zero string `json:"zero"`

	// If set, a type definition is added to the package mapping Type to this
	// structure, e.g., "struct { ... }".
	Decl string `json:"decl,omitempty"`

	// If set, the body of a function with signature func(x, y Type) bool
	// reporting whether x is less than y.
	//
	// For example:
	//   if x[0] == y[0] {
	//     return x[1] < y[1]
	//   }
	//   return x[0] < y[0]
	Less string `json:"less,omitempty"`

	// If set, the body of a function with signature func(x Type) string that
	// converts x to a human-readable string.
	//
	// For example:
	//   return strconv.Itoa(x)
	ToString string `json:"toString,omitempty"`

	// If set, additional packages to import in the generated code.
	Imports []string `json:"imports,omitempty"`
}

func (c *Config) validate() error {
	if c.Package == "" {
		return errors.New("invalid: missing package name")
	} else if c.Type == "" {
		return errors.New("invalid: missing type name")
	} else if c.Zero == "" {
		return errors.New("invalid: missing zero value")
	}
	return nil
}

var (
	configPath = flag.String("config", "", `Path of configuration file ("" to read stdin)`)
	outDir     = flag.String("output", "", "Output directory path (required)")

	mainT       = template.Must(template.New("main").Parse(strings.TrimSpace(mainFile)))
	utilT       = template.Must(template.New("util").Parse(strings.TrimSpace(utilFile)))
	baseImports = []string{"reflect", "sort", "strings"}
)

// readConfig loads a configuration from the specified path or stdin, and
// reports whether it is valid.
func readConfig(path string) (*Config, error) {
	var data []byte
	var err error
	if path == "" {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	// Deduplicate the import list, including all those specified by the
	// configuration as well as those needed by the static code.
	imps := make(map[string]bool)
	for _, pkg := range baseImports {
		imps[pkg] = true
	}
	for _, pkg := range c.Imports {
		imps[pkg] = true
	}
	if c.ToString == "" {
		imps["fmt"] = true // for fmt.Sprint
	}
	c.Imports = make([]string, 0, len(imps))
	for pkg := range imps {
		c.Imports = append(c.Imports, pkg)
	}
	sort.Strings(c.Imports)
	return &c, c.validate()
}

// generate renders source text from t using the values in c, formats the
// output as Go source, and writes the result to path.
func generate(t *template.Template, c *Config, path string) error {
	var buf bytes.Buffer
	if err := t.Execute(&buf, c); err != nil {
		return fmt.Errorf("generating source for %q: %v", path, err)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting source for %q: %v", path, err)
	}
	return ioutil.WriteFile(path, src, 0644)
}

func main() {
	flag.Parse()
	if *outDir == "" {
		log.Fatal("You must specify a non-empty -output directory")
	}
	conf, err := readConfig(*configPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("Unable to create output directory: %v", err)
	}

	mainPath := filepath.Join(*outDir, conf.Package+".go")
	if err := generate(mainT, conf, mainPath); err != nil {
		log.Fatal(err)
	}
	utilPath := filepath.Join(*outDir, "transform.go")
	if err := generate(utilT, conf, utilPath); err != nil {
		log.Fatal(err)
	}
}

// mainFile contains the main source for the package, including doc comments.
const mainFile = `
// Package {{.Package}} implements a lightweight (finite) set-of-{{.Type}} type
// based on Go's built-in map.  A Set provides some convenience methods for
// common set operations.
//
// A nil Set is ready for use as an empty set.  The basic set methods (Diff,
// Intersect, Union, IsSubset, Map, Choose, Partition) do not mutate their
// arguments.  There are also mutating operations (Add, Discard, Pop, Remove,
// Update) that modify their receiver in-place.
//
// A Set can also be traversed and modified using the normal map operations.
// Being a map, a Set is not safe for concurrent access by multiple goroutines
// unless all the concurrent accesses are reads.
package {{.Package}}

import (
{{range .Imports}}{{printf "%q" .}}
{{end}}
)

{{if .Decl}}
// {{.Type}} is the type of the elements of the set.
type {{.Type}} {{.Decl}}{{end}}

{{if .Less}}
// isLess reports whether x is less than y in standard order.
func isLess(x, y {{.Type}}) bool {
	{{.Less}}
}{{end}}

{{if .ToString}}func toString(x {{.Type}}) string {
    {{.ToString}}
}{{end}}

// A Set represents a set of {{.Type}} values.  A nil Set is a valid
// representation of an empty set.
type Set map[{{.Type}}]struct{}

// byElement satisfies sort.Interface to order values of type {{.Type}}.
type byElement []{{.Type}}
func(e byElement) Len() int { return len(e) }
func (e byElement) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e byElement) Less(i, j int) bool {
	{{if .Less}}return isLess(e[i], e[j]){{else}}return e[i] < e[j]{{end}}
}

// String implements the fmt.Stringer interface.  It renders s in standard set
// notation, e.g., ø for an empty set, {a, b, c} for a nonempty one.
func (s Set) String() string {
	if s.Empty() {
		return "ø"
	}
	elts := make([]string, len(s))
	for i, elt := range s.Elements() {
		elts[i] = {{if .ToString}}toString{{else}}fmt.Sprint{{end}}(elt)
	}
	return "{" + strings.Join(elts, ", ") + "}"
}

// New returns a new set containing exactly the specified elements.  
// Returns a non-nil empty Set if no elements are specified.
func New(elts ...{{.Type}}) Set {
	set := make(Set)
	for _, elt := range elts {
		set[elt] = struct{}{}
	}
	return set
}

// NewSize returns a new empty set pre-sized to hold at least n elements.
// This is equivalent to make(Set, n) and will panic if n < 0.
func NewSize(n int) Set { return make(Set, n) }

// Len returns the number of elements in s.
func (s Set) Len() int { return len(s) }

// Elements returns an ordered slice of the elements in s.
func (s Set) Elements() []{{.Type}} {
	elts := s.Unordered()
	sort.Sort(byElement(elts))
	return elts
}

// Unordered returns an unordered slice of the elements in s.
func (s Set) Unordered() []{{.Type}} {
	if len(s) == 0 {
		return nil
	}
	elts := make([]{{.Type}}, 0, len(s))
	for elt := range s {
		elts = append(elts, elt)
	}
	return elts
}

// Clone returns a new Set distinct from s, containing the same elements.
func (s Set) Clone() Set {
	var c Set
	c.Update(s)
	return c
}

// ContainsAny reports whether s contains one or more of the given elements.
// It is equivalent in meaning to
//   s.Intersects({{.Package}}.New(elts...))
// but does not construct an intermediate set.
func (s Set) ContainsAny(elts ...{{.Type}}) bool {
	for _, key := range elts {
		if _, ok := s[key]; ok {
			return true
		}
	}
	return false
}

// Contains reports whether s contains (all) the given elements.
// It is equivalent in meaning to
//   New(elts...).IsSubset(s)
// but does not construct an intermediate set.
func (s Set) Contains(elts ...{{.Type}}) bool {
	for _, elt := range elts {
		if _, ok := s[elt]; !ok {
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

// Add adds the specified elements to *s in-place and reports whether anything
// was added.  If *s == nil, a new set equivalent to New(ss...) is stored in *s.
func (s *Set) Add(ss ...{{.Type}}) bool {
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

// Discard removes the elements of elts from s in-place and reports whether
// anything was removed.
//
// Equivalent to s.Remove(New(elts...)), but does not allocate an intermediate
// set for ss.
func (s Set) Discard(elts ...{{.Type}}) bool {
	in := s.Len()
	if !s.Empty() {
		for _, elt := range elts {
			delete(s, elt)
		}
	}
	return s.Len() != in
}

// Index returns the first offset of needle in elts, if it occurs; otherwise -1.
func Index(needle {{.Type}}, elts []{{.Type}}) int {
	for i, elt := range elts {
		if elt == needle {
			return i
		}
	}
	return -1
}

// Contains reports whether v contains s, for v having type Set, []{{.Type}},
// map[{{.Type}}]T, or Keyer. It returns false if v's type does not have one of
// these forms.
func Contains(v interface{}, s {{.Type}}) bool {
	switch t := v.(type) {
	case []{{.Type}}:
		return Index(s, t) >= 0
	case Set:
		return t.Contains(s)
	case Keyer:
		return Index(s, t.Keys()) >= 0
	}
	if m := reflect.ValueOf(v); m.IsValid() && m.Kind() == reflect.Map && m.Type().Key() == refType {
		return m.MapIndex(reflect.ValueOf(s)).IsValid()
	}
	return false
}

// A Keyer implements a Keys method that returns the keys of a collection such
// as a map or a Set.
type Keyer interface {
	// Keys returns the keys of the receiver, which may be nil.
	Keys() []{{.Type}}
}

var refType = reflect.TypeOf((*{{.Type}})(nil)).Elem()

// FromKeys returns a Set of type {{.Type}}s from v, which must either be
// a {{.Type}}, a []{{.Type}}, a map[{{.Type}}]T, or a Keyer. It returns nil
// if v's type does not have one of these forms.
func FromKeys(v interface{}) Set {
    var result Set
	switch t := v.(type) {
	case {{.Type}}:
		return New(t)
	case []{{.Type}}:
		for _, key := range t {
			result.Add(key)
		}
		return result
	case map[{{.Type}}]struct{}: // includes Set
		for key := range t {
			result.Add(key)
		}
		return result
	case Keyer:
		for _, key := range t.Keys() {
			result.Add(key)
		}
		return result
	case nil:
		return nil
	}
	m := reflect.ValueOf(v)
	if m.Kind() != reflect.Map || m.Type().Key() != refType {
		return nil
	}
	for _, key := range m.MapKeys() {
		result.Add(key.Interface().({{.Type}}))
	}
	return result
}

// FromValues returns a Set of the values from v, which has type map[T]{{.Type}}.
// Returns the empty set if v does not have a type of this form.
func FromValues(v interface{}) Set {
	if t := reflect.TypeOf(v); t == nil || t.Kind() != reflect.Map || t.Elem() != refType {
		return nil
	}
	var set Set
	m := reflect.ValueOf(v)
	for _, key := range m.MapKeys() {
		set.Add(m.MapIndex(key).Interface().({{.Type}}))
	}
	return set
}
`

// utilFile contains the set transformation and selection methods.
const utilFile = `
package {{.Package}}

// Map returns the Set that results from applying f to each element of s.
func (s Set) Map(f func({{.Type}}) {{.Type}}) Set {
	var out Set
	for k := range s {
		out.Add(f(k))
	}
	return out
}

// Each applies f to each element of s.
func (s Set) Each(f func({{.Type}})) {
	for k := range s {
		f(k)
	}
}

// Select returns the subset of s for which f returns true.
func (s Set) Select(f func({{.Type}}) bool) Set {
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
func (s Set) Partition(f func({{.Type}}) bool) (yes, no Set) {
	for k := range s {
		if f(k) {
			yes.Add(k)
		} else {
			no.Add(k)
		}
	}
	return
}

// Choose returns an element of s for which f returns true, if one exists.  The
// second result reports whether such an element was found.
// If f == nil, chooses an arbitrary element of s.
func (s Set) Choose(f func({{.Type}}) bool) ({{.Type}}, bool) {
	for k := range s {
		if f == nil || f(k) {
			return k, true
		}
	}
	return {{.Zero}}, false
}

// Pop removes and returns an element of s for which f returns true, if one
// exists (essentially Choose + Discard).  The second result reports whether
// such an element was found.  If f == nil, pops an arbitrary element of s.
func (s Set) Pop(f func({{.Type}}) bool) ({{.Type}}, bool) {
	if v, ok := s.Choose(f); ok {
		delete(s, v)
		return v, true
	}
	return {{.Zero}}, false
}

// Count returns the number of elements of s for which f returns true.
func (s Set) Count(f func({{.Type}}) bool) (n int) {
	for k := range s {
		if f(k) {
			n++
		}
	}
	return
}`
