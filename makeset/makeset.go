// Program makeset generates source code for a set package.  The type of the
// elements of the set is determined by a JSON configuration stored either in a
// file (named by the -config flag) or read from standard input.
//
// Usage:
//   go run makeset.go -output $DIR -config config.json
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
	// A human-readable description of the set this config defines.
	// This is ignored by the code generator, but may serve as documentation.
	Desc string `json:"desc,omitempty"`

	// The name of the resulting set package, e.g., "intset" (required).
	Package string `json:"package"`

	// The name of the type contained in the set, e.g., "int" (required).
	Type string `json:"type"`

	// The spelling of the zero value for the set type, e.g., "0" (required).
	Zero string `json:"zero"`

	// If set, a type definition is added to the package mapping Type to this
	// structure, e.g., "struct { ... }". You may prefix Decl with "=" to
	// generate a type alias (this requires Go ≥ 1.9).
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

	// If set, additional packages to import in the test.
	TestImports []string `json:"testImports,omitempty"`

	// If true, include transformations, e.g., Map, Partition, Each.
	Transforms bool `json:"transforms,omitempty"`

	// A list of exactly ten ordered test values used for the construction of
	// unit tests. If omitted, unit tests are not generated.
	TestValues []interface{} `json:"testValues,omitempty"`
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
	testT       = template.Must(template.New("test").Parse(strings.TrimSpace(testFile)))
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
	if len(conf.TestValues) > 0 && len(conf.TestValues) != 10 {
		log.Fatalf("Wrong number of test values (%d); exactly 10 are required", len(conf.TestValues))
	}
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("Unable to create output directory: %v", err)
	}

	mainPath := filepath.Join(*outDir, conf.Package+".go")
	if err := generate(mainT, conf, mainPath); err != nil {
		log.Fatal(err)
	}
	if len(conf.TestValues) != 0 {
		testPath := filepath.Join(*outDir, conf.Package+"_test.go")
		if err := generate(testT, conf, testPath); err != nil {
			log.Fatal(err)
		}
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
	set := make(Set, len(elts))
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

// IsSubset reports whether s is a subset of s2, s ⊆ s2.
func (s Set) IsSubset(s2 Set) bool {
	if s.Empty() {
		return true
	} else if len(s) > len(s2) {
		return false
	}
	for k := range s {
		if _, ok := s2[k]; !ok {
			return false
		}
	}
	return true
}

// Equals reports whether s is equal to s2, having exactly the same elements.
func (s Set) Equals(s2 Set) bool { return len(s) == len(s2) && s.IsSubset(s2) }

// Empty reports whether s is empty.
func (s Set) Empty() bool { return len(s) == 0 }

// Intersects reports whether the intersection s ∩ s2 is non-empty, without
// explicitly constructing the intersection.
func (s Set) Intersects(s2 Set) bool {
	a, b := s, s2
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

// Union constructs the union s ∪ s2.
func (s Set) Union(s2 Set) Set {
	if s.Empty() {
		return s2
	} else if s2.Empty() {
		return s
	}
	set := make(Set)
	for k := range s {
		set[k] = struct{}{}
	}
	for k := range s2 {
		set[k] = struct{}{}
	}
	return set
}

// Intersect constructs the intersection s ∩ s2.
func (s Set) Intersect(s2 Set) Set {
	if s.Empty() || s2.Empty() {
		return nil
	}
	set := make(Set)
	for k := range s {
		if _, ok := s2[k]; ok {
			set[k] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

// Diff constructs the set difference s \ s2.
func (s Set) Diff(s2 Set) Set {
	if s.Empty() || s2.Empty() {
		return s
	}
	set := make(Set)
	for k := range s {
		if _, ok := s2[k]; !ok {
			set[k] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

// SymDiff constructs the symmetric difference s ∆ s2.
// It is equivalent in meaning to (s ∪ s2) \ (s ∩ s2).
func (s Set) SymDiff(s2 Set) Set {
	return s.Union(s2).Diff(s.Intersect(s2))
}

// Update adds the elements of s2 to *s in-place, and reports whether anything
// was added.
// If *s == nil and s2 ≠ ø, a new set is allocated that is a copy of s2.
func (s *Set) Update(s2 Set) bool {
	in := len(*s)
	if *s == nil && len(s2) > 0 {
		*s = make(Set)
	}
	for k := range s2 {
		(*s)[k] = struct{}{}
	}
	return len(*s) != in
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

// Remove removes the elements of s2 from s in-place and reports whether
// anything was removed.
//
// Equivalent to s = s.Diff(s2), but does not allocate a new set.
func (s Set) Remove(s2 Set) bool {
	in := s.Len()
	if !s.Empty() {
		for k := range s2 {
			delete(s, k)
		}
	}
	return s.Len() != in
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

// FromKeys returns a Set of {{.Type}}s from v, which must either be a {{.Type}},
// a []{{.Type}}, a map[{{.Type}}]T, or a Keyer. It returns nil if v's type does
// not have one of these forms.
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

{{if .Transforms}}
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
	if f == nil {
		for k := range s {
			return k, true
		}
	}
	for k := range s {
		if f(k) {
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
}
{{end}}{{/* transforms */}}
`

// testFile contains the unit tests.
const testFile = `
package {{.Package}}

import (
	"reflect"
	"testing"

{{range .TestImports}}{{printf "%q" .}}
{{end}}
)

// testValues contains an ordered sequence of ten set keys used for testing.
// The order of the keys must reflect the expected order of key listings.
var testValues = [10]{{.Type}}{
{{range .TestValues}}   {{.}},
{{end}}
}

func testKeys(ixs ...int) (keys []{{.Type}}) {
	for _, i := range ixs {
		keys = append(keys, testValues[i])
	}
	return
}

func testSet(ixs ...int) Set { return New(testKeys(ixs...)...) }

func keyPos(key {{.Type}}) int {
	for i, v := range testValues {
		if v == key {
			return i
		}
	}
	return -1
}

func TestEmptiness(t *testing.T) {
	var s Set
	if !s.Empty() {
		t.Errorf("nil Set is not reported empty: %v", s)
	}

	s = New()
	if !s.Empty() {
		t.Errorf("Empty Set is not reported empty: %v", s)
	}
	if s == nil {
		t.Error("New() unexpectedly returned nil")
	}

	if s := testSet(0); s.Empty() {
		t.Errorf("Nonempty Set is reported empty: %v", s)
	}
}

func TestClone(t *testing.T) {
	a := New(testValues[:]...)
	b := testSet(1, 8, 5)
	c := a.Clone()
	c.Remove(b)
	if c.Equals(a) {
		t.Errorf("Unexpected equality: %v == %v", a, c)
	} else {
		t.Logf("%v.Clone().Remove(%v) == %v", a, b, c)
	}
	c.Update(b)
	if !c.Equals(a) {
		t.Errorf("Unexpected inequality: %v != %v", a, c)
	}

	var s Set
	if got := s.Clone(); got != nil {
		t.Errorf("Clone of nil set: got %v, want nil", got)
	}
}

func TestUniqueness(t *testing.T) {
	// Sets should not contain duplicates.  Obviously this is impossible with
	// the map implementation, but other representations are viable.
	s := testSet(0, 5, 1, 2, 1, 3, 8, 4, 9, 4, 4, 6, 7, 2, 0, 0, 1, 4, 8, 4, 9)
	if got, want := s.Len(), len(testValues); got != want {
		t.Errorf("s.Len(): got %d, want %d [%v]", got, want, s)
	}

	// Keys should come out sorted.
	if got := s.Elements(); !reflect.DeepEqual(got, testValues[:]) {
		t.Errorf("s.Elements():\n got %+v,\nwant %+v", got, testValues)
	}
}

func TestMembership(t *testing.T) {
	s := testSet(0, 1, 2, 3, 4)
	for i, v := range testValues {
		if got, want := s.ContainsAny(v), i < 5; got != want {
			t.Errorf("s.ContainsAny(%v): got %v, want %v", v, got, want)
		}
	}

{{if .Transforms}}
	// Test non-mutating selection.
	if got, ok := s.Choose(func(s {{.Type}}) bool {
		return s == testValues[0]
	}); !ok {
		t.Error("Choose(0): missing element")
	} else {
		t.Logf("Found %v for element 0", got)
	}
	if got, ok := s.Choose(func({{.Type}}) bool { return false }); ok {
		t.Errorf(` + "`" + `Choose(impossible): got %v, want {{.Zero}}` + "`" + `, got)
	}
	if got, ok := New().Choose(nil); ok {
		t.Errorf(` + "`" + `Choose(nil): got %v, want {{.Zero}}` + "`" + `, got)
	}

	// Test mutating selection.
	if got, ok := s.Pop(func(s {{.Type}}) bool {
		return s == testValues[1]
	}); !ok {
		t.Error("Pop(1): missing element")
	} else {
		t.Logf("Found %v for element 1", got)
	}
	// A popped item is removed from the set.
	if len(s) != 4 {
		t.Errorf("Length after pop: got %d, want %d", len(s), 4)
	}
	// Pop of a nonexistent key returns not-found.
	if got, ok := s.Pop(func({{.Type}}) bool { return false }); ok {
		t.Errorf(` + "`" + `Pop(impossible): got %v, want {{.Zero}}` + "`" + `, got)
	}
	// Pop from an empty set returns not-found.
	if got, ok := New().Pop(nil); ok {
		t.Errorf(` + "`" + `Pop(nil) on empty: got %v, want {{.Zero}}` + "`" + `, got)
	}{{end}}
}

func TestContainsAny(t *testing.T) {
	set := New(testValues[2:]...)
	tests := []struct {
		keys []{{.Type}}
		want bool
	}{
		{nil, false},
		{[]{{.Type}}{}, false},
		{testKeys(0), false},
		{testKeys(1), false},
		{testKeys(0, 1), false},
		{testKeys(7), true},
		{testKeys(8, 3, 4, 9), true},
		{testKeys(0, 7, 1, 0), true},
	}
	t.Logf("Test set: %v", set)
	for _, test := range tests {
		got := set.ContainsAny(test.keys...)
		if got != test.want {
			t.Errorf("ContainsAny(%+v): got %v, want %v", test.keys, got, test.want)
		}
	}
}

func TestContainsAll(t *testing.T) {
	//set := New("a", "e", "i", "y")
	set := New(testValues[2:]...)
	tests := []struct {
		keys []{{.Type}}
		want bool
	}{
		{nil, true},
		{[]{{.Type}}{}, true},
		{testKeys(2, 4, 6), true},
		{testKeys(1, 3, 5, 7), false},
		{testKeys(0), false},
		{testKeys(5, 5, 5), true},
	}
	t.Logf("Test set: %v", set)
	for _, test := range tests {
		got := set.Contains(test.keys...)
		if got != test.want {
			t.Errorf("Contains(%+v): got %v, want %v", test.keys, got, test.want)
		}
	}
}

func TestIsSubset(t *testing.T) {
	var empty Set
	key := testSet(0, 2, 6, 7, 9)
	for _, test := range [][]{{.Type}}{
		{}, testKeys(2, 6), testKeys(0, 7, 9),
	} {
		probe := New(test...)
		if !probe.IsSubset(key) {
			t.Errorf("IsSubset %+v ⊂ %+v is false", probe, key)
		}
		if !empty.IsSubset(probe) { // ø is a subset of everything, including itself.
			t.Errorf("IsSubset ø ⊂ %+v is false", probe)
		}
	}
}

func TestNotSubset(t *testing.T) {
	tests := []struct {
		probe, key Set
	}{
		{testSet(0), New()},
		{testSet(0), testSet(1)},
		{testSet(0, 1), testSet(1)},
		{testSet(0, 2, 1), testSet(0, 2, 3)},
	}
	for _, test := range tests {
		if test.probe.IsSubset(test.key) {
			t.Errorf("IsSubset %+v ⊂ %+v is true", test.probe, test.key)
		}
	}
}

func TestEquality(t *testing.T) {
	nat := New(testValues[:]...)
	odd := testSet(1, 3, 4, 5, 8)
	tests := []struct {
		left, right Set
		eq          bool
	}{
		{nil, nil, true},
		{nat, nat, true},               // Equality with the same value
		{testSet(0), testSet(0), true}, // Equality with Different values
		{testSet(0), nil, false},
		{nat, odd, false},
		{nil, testSet(0), false},
		{testSet(0), testSet(1), false},

		// Various set operations...
		{nat.Intersect(odd), odd, true},
		{odd, nat.Intersect(odd), true},
		{odd.Intersect(nat), odd, true},
		{odd, odd.Intersect(nat), true},
		{nat.Intersect(nat), nat, true},
		{nat, nat.Intersect(nat), true},
		{nat.Union(odd), nat, true},
		{nat, nat.Union(odd), true},
		{odd.Diff(nat), odd, false},
		{odd, odd.Diff(nat), false},
		{odd.Diff(nat), nil, true},
		{nil, odd.Diff(nat), true},

		{testSet(0, 1, 2).Diff(testSet(2, 5, 6)), testSet(1).Union(testSet(0)), true},
	}
	for _, test := range tests {
		if got := test.left.Equals(test.right); got != test.eq {
			t.Errorf("%v.Equals(%v): got %v, want %v", test.left, test.right, got, test.eq)
		}
	}
}

func TestUnion(t *testing.T) {
	vkeys := testKeys(0, 4)
	vowels := testSet(4, 0)
	consonants := testSet(1, 2, 3, 5, 6, 7, 8, 9)

	if got := vowels.Union(nil).Elements(); !reflect.DeepEqual(got, vkeys) {
		t.Errorf("Vowels ∪ ø: got %+v, want %+v", got, vkeys)
	}
	if got := New().Union(vowels).Elements(); !reflect.DeepEqual(got, vkeys) {
		t.Errorf("ø ∪ Vowels: got %+v, want %+v", got, vkeys)
	}

	if got, want := vowels.Union(consonants).Elements(), testValues[:]; !reflect.DeepEqual(got, want) {
		t.Errorf("Vowels ∪ Consonants: got %+v, want %+v", got, want)
	}
}

func TestIntersect(t *testing.T) {
	empty := New()
	nat := New(testValues[:]...)
	odd := testSet(1, 3, 5, 7, 9)
	prime := testSet(2, 3, 5, 7)

	tests := []struct {
		left, right Set
		want        []{{.Type}}
	}{
		{empty, empty, nil},
		{empty, nat, nil},
		{nat, empty, nil},
		{nat, nat, testValues[:]},
		{nat, odd, testKeys(1, 3, 5, 7, 9)},
		{odd, nat, testKeys(1, 3, 5, 7, 9)},
		{odd, prime, testKeys(3, 5, 7)},
		{prime, nat, testKeys(2, 3, 5, 7)},
	}
	for _, test := range tests {
		got := test.left.Intersect(test.right).Elements()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%v ∩ %v: got %+v, want %+v", test.left, test.right, got, test.want)
		} else if want, ok := len(test.want) != 0, test.left.Intersects(test.right); ok != want {
			t.Errorf("%+v.Intersects(%+v): got %v, want %v", test.left, test.right, ok, want)
		}
	}
}

func TestDiff(t *testing.T) {
	empty := New()
	nat := New(testValues[:]...)
	odd := testSet(1, 3, 5, 7, 9)
	prime := testSet(2, 3, 5, 7)

	tests := []struct {
		left, right Set
		want        []{{.Type}}
	}{
		{empty, empty, nil},
		{empty, nat, nil},
		{nat, empty, testValues[:]},
		{nat, nat, nil},
		{nat, odd, testKeys(0, 2, 4, 6, 8)},
		{odd, nat, nil},
		{odd, prime, testKeys(1, 9)},
		{prime, nat, nil},
	}
	for _, test := range tests {
		got := test.left.Diff(test.right).Elements()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%v \\ %v: got %+q, want %+q", test.left, test.right, got, test.want)
		}
	}
}

func TestSymDiff(t *testing.T) {
	a := testSet(0, 1, 2, 3, 4)
	b := testSet(0, 4, 5, 6, 7)
	c := testSet(3, 4, 8, 9)
	empty := New()

	tests := []struct {
		left, right Set
		want        []{{.Type}}
	}{
		{empty, empty, nil},
		{empty, a, a.Elements()},
		{b, empty, b.Elements()},
		{a, a, nil},
		{a, b, testKeys(1, 2, 3, 5, 6, 7)},
		{b, a, testKeys(1, 2, 3, 5, 6, 7)},
		{a, c, testKeys(0, 1, 2, 8, 9)},
		{c, a, testKeys(0, 1, 2, 8, 9)},
		{c, b, testKeys(0, 3, 5, 6, 7, 8, 9)},
	}
	for _, test := range tests {
		got := test.left.SymDiff(test.right).Elements()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%v ∆ %v: got %+v, want %+v", test.left, test.right, got, test.want)
		}
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		before, update Set
		want           []{{.Type}}
		changed        bool
	}{
		{nil, nil, nil, false},
		{nil, testSet(0), testKeys(0), true},
		{testSet(1), nil, testKeys(1), false},
		{testSet(2, 3), testSet(4, 4, 3), testKeys(2, 3, 4), true},
	}
	for _, test := range tests {
		ok := test.before.Update(test.update)
		if got := test.before.Elements(); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Update %v: got %+v, want %+q", test.before, got, test.want)
		}
		if ok != test.changed {
			t.Errorf("Update %v reported change=%v, want %v", test.before, ok, test.changed)
		}
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		before       Set
		update, want []{{.Type}}
		changed      bool
	}{
		{nil, nil, nil, false},
		{nil, testKeys(0), testKeys(0), true},
		{testSet(1), nil, testKeys(1), false},
		{testSet(0, 1), testKeys(2, 2, 1), testKeys(0, 1, 2), true},
	}
	for _, test := range tests {
		ok := test.before.Add(test.update...)
		if got := test.before.Elements(); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Add %v: got %+v, want %+v", test.before, got, test.want)
		}
		if ok != test.changed {
			t.Errorf("Add %v reported change=%v, want %v", test.before, ok, test.changed)
		}
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		before, update Set
		want           []{{.Type}}
		changed        bool
	}{
		{nil, nil, nil, false},
		{nil, testSet(0), nil, false},
		{testSet(5), nil, testKeys(5), false},
		{testSet(3, 9), testSet(5, 1, 9), testKeys(3), true},
		{testSet(0, 1, 2), testSet(4, 6), testKeys(0, 1, 2), false},
	}
	for _, test := range tests {
		ok := test.before.Remove(test.update)
		if got := test.before.Elements(); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Remove %v: got %+v, want %+v", test.before, got, test.want)
		}
		if ok != test.changed {
			t.Errorf("Remove %v reported change=%v, want %v", test.before, ok, test.changed)
		}
	}
}

func TestDiscard(t *testing.T) {
	tests := []struct {
		before       Set
		update, want []{{.Type}}
		changed      bool
	}{
		{nil, nil, nil, false},
		{nil, testKeys(0), nil, false},
		{testSet(1), nil, testKeys(1), false},
		{testSet(0, 1), testKeys(2, 2, 1), testKeys(0), true},
		{testSet(0, 1, 2), testKeys(3, 4), testKeys(0, 1, 2), false},
	}
	for _, test := range tests {
		ok := test.before.Discard(test.update...)
		if got := test.before.Elements(); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Discard %v: got %+v, want %+v", test.before, got, test.want)
		}
		if ok != test.changed {
			t.Errorf("Discard %v reported change=%v, want %v", test.before, ok, test.changed)
		}
	}
}

{{if .Transforms}}
func TestMap(t *testing.T) {
	in := New(testValues[:]...)
	got := make([]{{.Type}}, len(testValues))
	out := in.Map(func(s {{.Type}}) {{.Type}} {
		if p := keyPos(s); p < 0 {
			t.Errorf("Unknown input key %v", s)
		} else {
			got[p] = s
		}
		return s
	})
	if !reflect.DeepEqual(got, testValues[:]) {
		t.Errorf("Incomplete mapping:\n got %+v\nwant %+v", got, testValues)
	}
	if !out.Equals(in) {
		t.Errorf("Incorrect mapping:\n got %v\nwant %v", out, in)
	}
}

func TestEach(t *testing.T) {
	in := New(testValues[:]...)
	saw := make(map[{{.Type}}]int)
	in.Each(func(name {{.Type}}) {
		saw[name]++
	})
	for want := range in {
		if saw[want] != 1 {
			t.Errorf("Saw [%v] %d times, wanted 1", want, saw[want])
		}
	}
	for got, n := range saw {
		if _, ok := in[got]; !ok {
			t.Errorf("Saw [%v] %d times, wanted 0", got, n)
		}
	}
}

func TestSelection(t *testing.T) {
	in := New(testValues[:]...)
	want := testSet(0, 2, 4, 6, 8)
	if got := in.Select(func(s {{.Type}}) bool {
		pos := keyPos(s)
		return pos >= 0 && pos%2 == 0
	}); !got.Equals(want) {
		t.Errorf("%v.Select(evens): got %v, want %v", in, got, want)
	}
	if got := New().Select(func({{.Type}}) bool { return true }); !got.Empty() {
		t.Errorf("%v.Select(true): got %v, want empty", New(), got)
	}
	if got := in.Select(func({{.Type}}) bool { return false }); !got.Empty() {
		t.Errorf("%v.Select(false): got %v, want empty", in, got)
	}
}

func TestPartition(t *testing.T) {
	in := New(testValues[:]...)
	tests := []struct {
		in, left, right Set
		f               func({{.Type}}) bool
		desc            string
	}{
		{testSet(0, 1), testSet(0, 1), nil,
			func({{.Type}}) bool { return true },
			"all true",
		},
		{testSet(0, 1), nil, testSet(0, 1),
			func({{.Type}}) bool { return false },
			"all false",
		},
		{in,
			testSet(0, 1, 2, 3, 4),
			testSet(5, 6, 7, 8, 9),
			func(s {{.Type}}) bool { return keyPos(s) < 5 },
			"pos(s) < 5",
		},
		{in,
			testSet(1, 3, 5, 7, 9), // odd
			testSet(0, 2, 4, 6, 8), // even
			func(s {{.Type}}) bool { return keyPos(s)%2 == 1 },
			"odd/even",
		},
	}
	for _, test := range tests {
		gotLeft, gotRight := test.in.Partition(test.f)
		if !gotLeft.Equals(test.left) {
			t.Errorf("Partition %s left: got %v, want %v", test.desc, gotLeft, test.left)
		}
		if !gotRight.Equals(test.right) {
			t.Errorf("Partition %s right: got %v, want %v", test.desc, gotRight, test.right)
		}
		t.Logf("Partition %v %s\n\t left: %v\n\tright: %v", test.in, test.desc, gotLeft, gotRight)
	}
}
{{end}}

func TestIndex(t *testing.T) {
	tests := []struct {
		needle {{.Type}}
		keys   []{{.Type}}
		want   int
	}{
		{testValues[0], nil, -1},
		{testValues[1], []{{.Type}}{}, -1},
		{testValues[2], testKeys(0, 1), -1},
		{testValues[0], testKeys(0, 1), 0},
		{testValues[1], testKeys(0, 1), 1},
		{testValues[2], testKeys(0, 2, 1, 2), 1},
		{testValues[9], testKeys(0, 2, 1, 9, 6), 3},
		{testValues[4], testKeys(0, 2, 4, 9, 4), 2},
	}
	for _, test := range tests {
		got := Index(test.needle, test.keys)
		if got != test.want {
			t.Errorf("Index(%+v, %+v): got %d, want %d", test.needle, test.keys, got, test.want)
		}
	}
}

type keyer []{{.Type}}

func (k keyer) Keys() []{{.Type}} {
	p := make([]{{.Type}}, len(k))
	copy(p, k)
	return p
}

type uniq int

func TestFromValues(t *testing.T) {
	tests := []struct {
		input interface{}
		want  []{{.Type}}
	}{
		{nil, nil},
		{map[float64]{{.Type}}{}, nil},
		{map[int]{{.Type}}{1: testValues[1], 2: testValues[2], 3: testValues[2]}, testKeys(1, 2)},
		{map[string]{{.Type}}{"foo": testValues[4], "baz": testValues[4]}, testKeys(4)},
		{map[int]uniq{1: uniq(2), 3: uniq(4), 5: uniq(6)}, nil},
		{map[*int]{{.Type}}{nil: testValues[0]}, testKeys(0)},
	}
	for _, test := range tests {
		got := FromValues(test.input)
		want := New(test.want...)
		if !got.Equals(want) {
			t.Errorf("MapValues %v: got %v, want %v", test.input, got, want)
		}
	}
}

func TestFromKeys(t *testing.T) {
	tests := []struct {
		input interface{}
		want  Set
	}{
		{3.5, nil},                  // unkeyable type
		{map[uniq]uniq{1: 1}, nil},  // unkeyable type
		{nil, nil},                  // empty
		{[]string{}, nil},           // empty
		{map[{{.Type}}]float64{}, nil}, // empty
		{testValues[0], testSet(0)},
		{testKeys(0, 1, 0, 0), testSet(0, 1)},
		{map[{{.Type}}]int{testValues[0]: 1, testValues[1]: 2}, testSet(0, 1)},
		{keyer(testValues[:3]), testSet(0, 1, 2)},
		{testSet(4, 7, 8), testSet(4, 7, 8)},
		{map[{{.Type}}]struct{}{testValues[2]: {}, testValues[7]: {}}, testSet(2, 7)},
	}
	for _, test := range tests {
		got := FromKeys(test.input)
		if !got.Equals(test.want) {
			t.Errorf("FromKeys %v: got %v, want %v", test.input, got, test.want)
		}
	}
}

func TestContainsFunc(t *testing.T) {
	tests := []struct {
		input  interface{}
		needle {{.Type}}
		want   bool
	}{
		{[]{{.Type}}(nil), testValues[0], false},
		{[]{{.Type}}{}, testValues[0], false},
		{testKeys(0), testValues[0], true},
		{testKeys(1), testValues[0], false},
		{testKeys(0, 1, 9, 2), testValues[0], true},

		{map[{{.Type}}]int(nil), testValues[2], false},
		{map[{{.Type}}]int{}, testValues[2], false},
		{map[{{.Type}}]int{testValues[2]: 1}, testValues[2], true},
		{map[{{.Type}}]int{testValues[3]: 3}, testValues[2], false},
		{map[{{.Type}}]float32{testValues[2]: 1, testValues[4]: 2}, testValues[2], true},
		{map[{{.Type}}]float32{testValues[5]: 0, testValues[6]: 1, testValues[7]: 2, testValues[8]: 3}, testValues[2], false},

		{Set(nil), testValues[3], false},
		{New(), testValues[3], false},
		{New(testValues[3]), testValues[3], true},
		{New(testValues[5]), testValues[3], false},
		{testSet(0, 1), testValues[3], false},
		{testSet(0, 3, 1), testValues[3], true},

		{keyer(nil), testValues[9], false},
		{keyer{}, testValues[9], false},
		{keyer{testValues[9]}, testValues[9], true},
		{keyer{testValues[0]}, testValues[9], false},
		{keyer(testKeys(0, 6, 9)), testValues[9], true},
		{keyer(testKeys(0, 6, 7)), testValues[9], false},
	}
	for _, test := range tests {
		got := Contains(test.input, test.needle)
		if got != test.want {
			t.Errorf("Contains(%+v, %v): got %v, want %v", test.input, test.needle, got, test.want)
		}
	}
}
`
