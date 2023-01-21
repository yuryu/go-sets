package stringset_test

import (
	"reflect"
	"testing"

	"bitbucket.org/creachadair/stringset"
)

// testValues contains an ordered sequence of ten set keys used for testing.
// The order of the keys must reflect the expected order of key listings.
var testValues = [10]string{
	"eight",
	"five",
	"four",
	"nine",
	"one",
	"seven",
	"six",
	"ten",
	"three",
	"two",
}

func testKeys(ixs ...int) (keys []string) {
	for _, i := range ixs {
		keys = append(keys, testValues[i])
	}
	return
}

func testSet(ixs ...int) stringset.Set {
	return stringset.New(testKeys(ixs...)...)
}

func keyPos(key string) int {
	for i, v := range testValues {
		if v == key {
			return i
		}
	}
	return -1
}

func TestEmptiness(t *testing.T) {
	var s stringset.Set
	if !s.Empty() {
		t.Errorf("nil Set is not reported empty: %v", s)
	}

	s = stringset.New()
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
	a := stringset.New(testValues[:]...)
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

	var s stringset.Set
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

	// Test non-mutating selection.
	if got, ok := s.Choose(func(s string) bool {
		return s == testValues[0]
	}); !ok {
		t.Error("Choose(0): missing element")
	} else {
		t.Logf("Found %v for element 0", got)
	}
	if got, ok := s.Choose(func(string) bool { return false }); ok {
		t.Errorf(`Choose(impossible): got %v, want ""`, got)
	}
	if got, ok := stringset.New().Choose(nil); ok {
		t.Errorf(`Choose(nil): got %v, want ""`, got)
	}

	// Test mutating selection.
	if got, ok := s.Pop(func(s string) bool {
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
	if got, ok := s.Pop(func(string) bool { return false }); ok {
		t.Errorf(`Pop(impossible): got %v, want ""`, got)
	}
	// Pop from an empty set returns not-found.
	if got, ok := stringset.New().Pop(nil); ok {
		t.Errorf(`Pop(nil) on empty: got %v, want ""`, got)
	}
}

func TestContainsAny(t *testing.T) {
	set := stringset.New(testValues[2:]...)
	tests := []struct {
		keys []string
		want bool
	}{
		{nil, false},
		{[]string{}, false},
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
	set := stringset.New(testValues[2:]...)
	tests := []struct {
		keys []string
		want bool
	}{
		{nil, true},
		{[]string{}, true},
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
	var empty stringset.Set
	key := testSet(0, 2, 6, 7, 9)
	for _, test := range [][]string{
		{}, testKeys(2, 6), testKeys(0, 7, 9),
	} {
		probe := stringset.New(test...)
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
		probe, key stringset.Set
	}{
		{testSet(0), stringset.New()},
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
	nat := stringset.New(testValues[:]...)
	odd := testSet(1, 3, 4, 5, 8)
	tests := []struct {
		left, right stringset.Set
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
	if got := stringset.New().Union(vowels).Elements(); !reflect.DeepEqual(got, vkeys) {
		t.Errorf("ø ∪ Vowels: got %+v, want %+v", got, vkeys)
	}

	if got, want := vowels.Union(consonants).Elements(), testValues[:]; !reflect.DeepEqual(got, want) {
		t.Errorf("Vowels ∪ Consonants: got %+v, want %+v", got, want)
	}
}

func TestIntersect(t *testing.T) {
	empty := stringset.New()
	nat := stringset.New(testValues[:]...)
	odd := testSet(1, 3, 5, 7, 9)
	prime := testSet(2, 3, 5, 7)

	tests := []struct {
		left, right stringset.Set
		want        []string
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
	empty := stringset.New()
	nat := stringset.New(testValues[:]...)
	odd := testSet(1, 3, 5, 7, 9)
	prime := testSet(2, 3, 5, 7)

	tests := []struct {
		left, right stringset.Set
		want        []string
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
	empty := stringset.New()

	tests := []struct {
		left, right stringset.Set
		want        []string
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
		before, update stringset.Set
		want           []string
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
		before       stringset.Set
		update, want []string
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
		before, update stringset.Set
		want           []string
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
		before       stringset.Set
		update, want []string
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

func TestMap(t *testing.T) {
	in := stringset.New(testValues[:]...)
	got := make([]string, len(testValues))
	out := in.Map(func(s string) string {
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
	in := stringset.New(testValues[:]...)
	saw := make(map[string]int)
	in.Each(func(name string) {
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
	in := stringset.New(testValues[:]...)
	want := testSet(0, 2, 4, 6, 8)
	if got := in.Select(func(s string) bool {
		pos := keyPos(s)
		return pos >= 0 && pos%2 == 0
	}); !got.Equals(want) {
		t.Errorf("%v.Select(evens): got %v, want %v", in, got, want)
	}
	if got := stringset.New().Select(func(string) bool { return true }); !got.Empty() {
		t.Errorf("%v.Select(true): got %v, want empty", stringset.New(), got)
	}
	if got := in.Select(func(string) bool { return false }); !got.Empty() {
		t.Errorf("%v.Select(false): got %v, want empty", in, got)
	}
}

func TestPartition(t *testing.T) {
	in := stringset.New(testValues[:]...)
	tests := []struct {
		in, left, right stringset.Set
		f               func(string) bool
		desc            string
	}{
		{testSet(0, 1), testSet(0, 1), nil,
			func(string) bool { return true },
			"all true",
		},
		{testSet(0, 1), nil, testSet(0, 1),
			func(string) bool { return false },
			"all false",
		},
		{in,
			testSet(0, 1, 2, 3, 4),
			testSet(5, 6, 7, 8, 9),
			func(s string) bool { return keyPos(s) < 5 },
			"pos(s) < 5",
		},
		{in,
			testSet(1, 3, 5, 7, 9), // odd
			testSet(0, 2, 4, 6, 8), // even
			func(s string) bool { return keyPos(s)%2 == 1 },
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

func TestIndex(t *testing.T) {
	tests := []struct {
		needle string
		keys   []string
		want   int
	}{
		{testValues[0], nil, -1},
		{testValues[1], []string{}, -1},
		{testValues[2], testKeys(0, 1), -1},
		{testValues[0], testKeys(0, 1), 0},
		{testValues[1], testKeys(0, 1), 1},
		{testValues[2], testKeys(0, 2, 1, 2), 1},
		{testValues[9], testKeys(0, 2, 1, 9, 6), 3},
		{testValues[4], testKeys(0, 2, 4, 9, 4), 2},
	}
	for _, test := range tests {
		got := stringset.Index(test.needle, test.keys)
		if got != test.want {
			t.Errorf("Index(%+v, %+v): got %d, want %d", test.needle, test.keys, got, test.want)
		}
	}
}

type keyer []string

func (k keyer) Keys() []string {
	p := make([]string, len(k))
	copy(p, k)
	return p
}

type uniq int

func TestFromValues(t *testing.T) {
	tests := []struct {
		input interface{}
		want  []string
	}{
		{nil, nil},
		{map[float64]string{}, nil},
		{map[int]string{1: testValues[1], 2: testValues[2], 3: testValues[2]}, testKeys(1, 2)},
		{map[string]string{"foo": testValues[4], "baz": testValues[4]}, testKeys(4)},
		{map[int]uniq{1: uniq(2), 3: uniq(4), 5: uniq(6)}, nil},
		{map[*int]string{nil: testValues[0]}, testKeys(0)},
	}
	for _, test := range tests {
		got := stringset.FromValues(test.input)
		want := stringset.New(test.want...)
		if !got.Equals(want) {
			t.Errorf("MapValues %v: got %v, want %v", test.input, got, want)
		}
	}
}

func TestFromKeys(t *testing.T) {
	tests := []struct {
		input interface{}
		want  stringset.Set
	}{
		{3.5, nil},                  // unkeyable type
		{map[uniq]uniq{1: 1}, nil},  // unkeyable type
		{nil, nil},                  // empty
		{[]string{}, nil},           // empty
		{map[string]float64{}, nil}, // empty
		{testValues[0], testSet(0)},
		{testKeys(0, 1, 0, 0), testSet(0, 1)},
		{map[string]int{testValues[0]: 1, testValues[1]: 2}, testSet(0, 1)},
		{keyer(testValues[:3]), testSet(0, 1, 2)},
		{testSet(4, 7, 8), testSet(4, 7, 8)},
		{map[string]struct{}{testValues[2]: {}, testValues[7]: {}}, testSet(2, 7)},
	}
	for _, test := range tests {
		got := stringset.FromKeys(test.input)
		if !got.Equals(test.want) {
			t.Errorf("FromKeys %v: got %v, want %v", test.input, got, test.want)
		}
	}
}

func TestContainsFunc(t *testing.T) {
	tests := []struct {
		input  interface{}
		needle string
		want   bool
	}{
		{[]string(nil), testValues[0], false},
		{[]string{}, testValues[0], false},
		{testKeys(0), testValues[0], true},
		{testKeys(1), testValues[0], false},
		{testKeys(0, 1, 9, 2), testValues[0], true},

		{map[string]int(nil), testValues[2], false},
		{map[string]int{}, testValues[2], false},
		{map[string]int{testValues[2]: 1}, testValues[2], true},
		{map[string]int{testValues[3]: 3}, testValues[2], false},
		{map[string]float32{testValues[2]: 1, testValues[4]: 2}, testValues[2], true},
		{map[string]float32{testValues[5]: 0, testValues[6]: 1, testValues[7]: 2, testValues[8]: 3}, testValues[2], false},

		{stringset.Set(nil), testValues[3], false},
		{stringset.New(), testValues[3], false},
		{stringset.New(testValues[3]), testValues[3], true},
		{stringset.New(testValues[5]), testValues[3], false},
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
		got := stringset.Contains(test.input, test.needle)
		if got != test.want {
			t.Errorf("Contains(%+v, %v): got %v, want %v", test.input, test.needle, got, test.want)
		}
	}
}

func TestFromIndexed(t *testing.T) {
	tests := []struct {
		input []int
		want  stringset.Set
	}{
		{nil, nil},
		{[]int{}, nil},
		{[]int{0}, testSet(0)},
		{[]int{1, 8, 2, 9}, testSet(1, 2, 8, 9)},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, stringset.New(testValues[:]...)},
	}
	for _, test := range tests {
		got := stringset.FromIndexed(len(test.input), func(i int) string {
			return testValues[test.input[i]]
		})
		if !got.Equals(test.want) {
			t.Errorf("FromIndexed(%d, <...>): got %v, want %v", len(test.input), got, test.want)
		}
	}
}
