package stringset

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

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

	if s := New("something"); s.Empty() {
		t.Errorf("Nonempty Set is reported empty: %v", s)
	}
}

func TestClone(t *testing.T) {
	a := New(strings.Fields("an apple in a basket is worth two in the weasels")...)
	b := New("a", "an", "the")
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
	s := New("e", "a", "a", "c", "a", "b", "d", "c", "a", "e", "d", "e", "c", "a")
	if got, want := s.Len(), 5; got != want {
		t.Errorf("s.Len(): got %d, want %d [%v]", got, want, s)
	}

	// Keys should come out sorted.
	wantKeys := []string{"a", "b", "c", "d", "e"}
	if got := s.Elements(); !reflect.DeepEqual(got, wantKeys) {
		t.Errorf("s.Elements(): got %+q, want %+q", got, wantKeys)
	}
}

func TestMembership(t *testing.T) {
	const count = 5
	words := []string{
		"alpha", "bravo", "charlie", "delta", "echo", "foxtrot",
		"golf", "hotel", "india", "juliet", "kilo", "lima",
	}
	s := New(words[:count]...)
	for i, w := range words {
		if got, want := s.ContainsAny(w), i < count; got != want {
			t.Errorf("s.ContainsAny(%q): got %v, want %v", w, got, want)
		}
	}

	// Test non-mutating selection.
	re := regexp.MustCompile("^e")
	if got, ok := s.Choose(re.MatchString); !ok {
		t.Errorf(`Choose(%q): missing element`, re)
	} else {
		t.Logf(`Found %q for regexp %q`, got, re)
	}
	if got, ok := s.Choose(func(string) bool { return false }); ok {
		t.Errorf(`Choose(impossible): got %q, want ""`, got)
	}
	if got, ok := New().Choose(nil); ok {
		t.Errorf(`Choose(nil): got %q, want ""`, got)
	}

	// Test mutating selection.
	if got, ok := s.Pop(func(s string) bool { return strings.HasPrefix(s, "c") }); !ok {
		t.Error(`Pop("c*"): missing element`)
	} else {
		t.Logf(`Found %q for prefix "c"`, got)
	}
	// A popped item is removed from the set.
	if len(s) != count-1 {
		t.Errorf("Length after pop: got %d, want %d", len(s), count-1)
	}
	// Pop of a nonexistent key returns not-found.
	if got, ok := s.Pop(func(string) bool { return false }); ok {
		t.Errorf(`Pop(impossible): got %q, want ""`, got)
	}
	// Pop from an empty set returns not-found.
	if got, ok := New().Pop(nil); ok {
		t.Errorf(`Pop(nil) on ø: got %q, want ""`, got)
	}
}

func TestContainsAny(t *testing.T) {
	set := New("2", "3", "5", "7", "11", "13")
	tests := []struct {
		keys []string
		want bool
	}{
		{nil, false},
		{[]string{}, false},
		{[]string{"1", "4"}, false},
		{[]string{""}, false},
		{[]string{"7"}, true},
		{[]string{"8", "3", "1", "9"}, true},
		{[]string{"q", "r", "13", "s"}, true},
	}
	t.Logf("Test set: %v", set)
	for _, test := range tests {
		got := set.ContainsAny(test.keys...)
		if got != test.want {
			t.Errorf("ContainsAny(%+q): got %v, want %v", test.keys, got, test.want)
		}
	}
}

func TestContainsAll(t *testing.T) {
	set := New("a", "e", "i", "y")
	tests := []struct {
		keys []string
		want bool
	}{
		{nil, true},
		{[]string{}, true},
		{[]string{"a", "e", "i"}, true},
		{[]string{"a", "e", "i", "o"}, false},
		{[]string{"b"}, false},
		{[]string{"a", "a", "a"}, true},
	}
	t.Logf("Test set: %v", set)
	for _, test := range tests {
		got := set.Contains(test.keys...)
		if got != test.want {
			t.Errorf("Contains(%+q): got %v, want %v", test.keys, got, test.want)
		}
	}
}

func TestIsSubset(t *testing.T) {
	var empty Set
	key := New("some", "of", "what", "a", "fool", "thinks", "often", "remains")
	for _, test := range [][]string{
		{}, {"of", "a"}, {"some", "what", "fool"},
	} {
		probe := New(test...)
		if !probe.IsSubset(key) {
			t.Errorf("IsSubset %+q ⊂ %+q is false", probe, key)
		}
		if !empty.IsSubset(probe) { // ø is a subset of everything, including itself.
			t.Errorf("IsSubset ø ⊂ %+q is false", probe)
		}
	}
}

func TestNotSubset(t *testing.T) {
	tests := []struct {
		probe, key Set
	}{
		{New("a"), New()},
		{New("a"), New("b")},
		{New("a", "b"), New("b")},
		{New("a", "c", "b"), New("a", "c", "d")},
	}
	for _, test := range tests {
		if test.probe.IsSubset(test.key) {
			t.Errorf("IsSubset %v ⊂ %v is true", test.probe, test.key)
		}
	}
}

func TestEquality(t *testing.T) {
	nat := New("1", "2", "3", "4", "5")
	odd := New("1", "3", "5")
	tests := []struct {
		left, right Set
		eq          bool
	}{
		{nil, nil, true},
		{nat, nat, true},           // Equality with the same value
		{New("a"), New("a"), true}, // Equality with Different values
		{New("a"), nil, false},
		{nat, odd, false},
		{nil, New("a"), false},
		{New("a"), New("b"), false},

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

		{New("a", "b", "c").Diff(New("b", "m", "x")), New("c").Union(New("a")), true},
	}
	for _, test := range tests {
		if got := test.left.Equals(test.right); got != test.eq {
			t.Errorf("%v.Equals(%v): got %v, want %v", test.left, test.right, got, test.eq)
		}
	}
}

func TestUnion(t *testing.T) {
	vowels := New("e", "o", "i", "a", "u")
	vkeys := []string{"a", "e", "i", "o", "u"}

	consonants := New("h", "f", "b", "d", "g", "c")

	if got := vowels.Union(nil).Elements(); !reflect.DeepEqual(got, vkeys) {
		t.Errorf("Vowels ∪ ø: got %+q, want %+q", got, vkeys)
	}
	if got := New().Union(vowels).Elements(); !reflect.DeepEqual(got, vkeys) {
		t.Errorf("ø ∪ Vowels: got %+q, want %+q", got, vkeys)
	}

	want := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "o", "u"}
	if got := vowels.Union(consonants).Elements(); !reflect.DeepEqual(got, want) {
		t.Errorf("Vowels ∪ Consonants: got %+q, want %+q", got, want)
	}
}

func TestIntersect(t *testing.T) {
	empty := New()
	nat := New("0", "1", "2", "3", "4", "5", "6", "7")
	odd := New("1", "a", "3", "5", "7", "p", "q")
	prime := New("2", "m", "3", "d", "x", "5", "7", "!")

	tests := []struct {
		left, right Set
		want        []string
	}{
		{empty, empty, nil},
		{empty, nat, nil},
		{nat, empty, nil},
		{nat, nat, []string{"0", "1", "2", "3", "4", "5", "6", "7"}},
		{nat, odd, []string{"1", "3", "5", "7"}},
		{odd, nat, []string{"1", "3", "5", "7"}},
		{odd, prime, []string{"3", "5", "7"}},
		{prime, nat, []string{"2", "3", "5", "7"}},
	}
	for _, test := range tests {
		got := test.left.Intersect(test.right).Elements()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%v ∩ %v: got %+q, want %+q", test.left, test.right, got, test.want)
		} else if want, ok := len(test.want) != 0, test.left.Intersects(test.right); ok != want {
			t.Errorf("%v.Intersects(%v): got %v, want %v", test.left, test.right, ok, want)
		}
	}
}

func TestDiff(t *testing.T) {
	empty := New()
	nat := New("0", "1", "2", "3", "4", "5", "6", "7")
	odd := New("1", "a", "3", "5", "7", "p", "q")
	prime := New("2", "m", "3", "d", "x", "5", "7", "!")

	tests := []struct {
		left, right Set
		want        []string
	}{
		{empty, empty, nil},
		{empty, nat, nil},
		{nat, empty, []string{"0", "1", "2", "3", "4", "5", "6", "7"}},
		{nat, nat, nil},
		{nat, odd, []string{"0", "2", "4", "6"}},
		{odd, nat, []string{"a", "p", "q"}},
		{odd, prime, []string{"1", "a", "p", "q"}},
		{prime, nat, []string{"!", "d", "m", "x"}},
	}
	for _, test := range tests {
		got := test.left.Diff(test.right).Elements()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%v \\ %v: got %+q, want %+q", test.left, test.right, got, test.want)
		}
	}
}

func TestSymDiff(t *testing.T) {
	a := New("a", "b", "c", "d", "e")
	b := New("a", "e", "i", "o", "u")
	c := New("d", "e", "f", "i")
	empty := New()

	tests := []struct {
		left, right Set
		want        []string
	}{
		{empty, empty, nil},
		{empty, a, a.Elements()},
		{b, empty, b.Elements()},
		{a, a, nil},
		{a, b, []string{"b", "c", "d", "i", "o", "u"}},
		{b, a, []string{"b", "c", "d", "i", "o", "u"}},
		{a, c, []string{"a", "b", "c", "f", "i"}},
		{c, a, []string{"a", "b", "c", "f", "i"}},
		{c, b, []string{"a", "d", "f", "o", "u"}},
	}
	for _, test := range tests {
		got := test.left.SymDiff(test.right).Elements()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%v ∆ %v: got %+q, want %+q", test.left, test.right, got, test.want)
		}
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		before, update Set
		want           []string
		changed        bool
	}{
		{nil, nil, nil, false},
		{nil, New("a"), []string{"a"}, true},
		{New("pdq"), nil, []string{"pdq"}, false},
		{New("a", "b"), New("c", "c", "b"), []string{"a", "b", "c"}, true},
	}
	for _, test := range tests {
		ok := test.before.Update(test.update)
		if got := test.before.Elements(); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Update %v: got %+q, want %+q", test.before, got, test.want)
		}
		if ok != test.changed {
			t.Errorf("Update %v reported change=%v, want %v", test.before, ok, test.changed)
		}
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		before       Set
		update, want []string
		changed      bool
	}{
		{nil, nil, nil, false},
		{nil, []string{"a"}, []string{"a"}, true},
		{New("pdq"), nil, []string{"pdq"}, false},
		{New("a", "b"), []string{"c", "c", "b"}, []string{"a", "b", "c"}, true},
	}
	for _, test := range tests {
		ok := test.before.Add(test.update...)
		if got := test.before.Elements(); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Add %v: got %+q, want %+q", test.before, got, test.want)
		}
		if ok != test.changed {
			t.Errorf("Add %v reported change=%v, want %v", test.before, ok, test.changed)
		}
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		before, update Set
		want           []string
		changed        bool
	}{
		{nil, nil, nil, false},
		{nil, New("a"), nil, false},
		{New("pdq"), nil, []string{"pdq"}, false},
		{New("a", "b"), New("c", "c", "b"), []string{"a"}, true},
		{New("a", "b", "c"), New("d", "e"), []string{"a", "b", "c"}, false},
	}
	for _, test := range tests {
		ok := test.before.Remove(test.update)
		if got := test.before.Elements(); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Remove %v: got %+q, want %+q", test.before, got, test.want)
		}
		if ok != test.changed {
			t.Errorf("Remove %v reported change=%v, want %v", test.before, ok, test.changed)
		}
	}
}

func TestDiscard(t *testing.T) {
	tests := []struct {
		before       Set
		update, want []string
		changed      bool
	}{
		{nil, nil, nil, false},
		{nil, []string{"a"}, nil, false},
		{New("pdq"), nil, []string{"pdq"}, false},
		{New("a", "b"), []string{"c", "c", "b"}, []string{"a"}, true},
		{New("a", "b", "c"), []string{"d", "e"}, []string{"a", "b", "c"}, false},
	}
	for _, test := range tests {
		ok := test.before.Discard(test.update...)
		if got := test.before.Elements(); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Discard %v: got %+q, want %+q", test.before, got, test.want)
		}
		if ok != test.changed {
			t.Errorf("Discard %v reported change=%v, want %v", test.before, ok, test.changed)
		}
	}
}

func TestMap(t *testing.T) {
	in := New("", "w", "x", "y")
	out := in.Map(func(s string) string {
		return "-" + s + "-"
	})
	for key := range out {
		want := strings.Trim(key, "-")
		if !strings.HasPrefix(key, "-") || !strings.HasPrefix(key, "-") {
			t.Errorf("Mapped key has the wrong form: %q", key)
		}
		if !in.ContainsAny(want) {
			t.Errorf("Mapped key %q is missing its antecedent %q", key, want)
		}
	}
}

func TestEach(t *testing.T) {
	in := New("alice", "basil", "clara", "desmond", "ernie")
	saw := make(map[string]int)
	in.Each(func(name string) {
		saw[name]++
	})
	for want := range in {
		if saw[want] != 1 {
			t.Errorf("Saw %q %d times, wanted 1", want, saw[want])
		}
	}
	for got, n := range saw {
		if _, ok := in[got]; !ok {
			t.Errorf("Saw %q %d times, wanted 0", got, n)
		}
	}
}

func TestSelection(t *testing.T) {
	in := New("ant", "bee", "cat", "dog", "aardvark", "apatasaurus", "emu")
	want := New("bee", "cat", "dog", "emu")
	if got := in.Select(func(s string) bool {
		return !strings.HasPrefix(s, "a")
	}); !got.Equals(want) {
		t.Errorf(`%v.Select("a*"): got %v, want %v`, in, got, want)
	}
	if got := New().Select(func(string) bool { return true }); !got.Empty() {
		t.Errorf("%v.Select(true): got %v, want empty", New(), got)
	}
	if got := in.Select(func(string) bool { return false }); !got.Empty() {
		t.Errorf("%v.Select(false): got %v, want empty", in, got)
	}
}

func TestPartition(t *testing.T) {
	in := New("a", "be", "cat", "dirt", "ennui", "faiths", "garbage", "horseman")
	tests := []struct {
		in, left, right Set
		f               func(string) bool
		desc            string
	}{
		{New("a", "b"), New("a", "b"), nil,
			func(string) bool { return true },
			"all true",
		},
		{New("a", "b"), nil, New("a", "b"),
			func(string) bool { return false },
			"all false",
		},
		{in,
			New("a", "be", "cat", "dirt", "ennui"),
			New("faiths", "garbage", "horseman"),
			func(s string) bool { return len(s) < 6 },
			"len(s) < 6",
		},
		{in,
			New("a", "cat", "ennui", "garbage"),     // odd
			New("be", "dirt", "faiths", "horseman"), // even
			func(s string) bool { return len(s)%2 == 1 },
			"odd/even",
		},
		{New(":x", ":y", "a", "z", ":m", "p"),
			New(":m", ":x", ":y"),
			New("a", "p", "z"),
			func(s string) bool { return strings.HasPrefix(s, ":") },
			"keywords",
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

func TestCount(t *testing.T) {
	in := New("three", "great", "apes", "ate", "moldy", "bananas", "in", "kansas")
	t.Logf("Input set: %s", in)
	if got, want := in.Count(func(s string) bool { return s[0] == 'a' }), 2; got != want {
		t.Errorf("Count(s[0]=='a'): got %d, want %d", got, want)
	}
	if got, want := in.Count(func(s string) bool { return len(s) < 5 }), 3; got != want {
		t.Errorf("Count(len(s) < 5): got %d, want %d", got, want)
	}
}

func TestIndex(t *testing.T) {
	tests := []struct {
		needle string
		keys   []string
		want   int
	}{
		{"", nil, -1},
		{"a", nil, -1},
		{"c", []string{"a", "b"}, -1},
		{"a", []string{"a", "b"}, 0},
		{"b", []string{"a", "b"}, 1},
		{"c", []string{"a", "c", "b", "c"}, 1},
		{"q", []string{"a", "c", "b", "q", ""}, 3},
		{"", []string{"a", "c", "", "q", ""}, 2},
	}
	for _, test := range tests {
		got := Index(test.needle, test.keys)
		if got != test.want {
			t.Errorf("Index(%q, %q): got %d, want %d", test.needle, test.keys, got, test.want)
		}
	}
}

type keyer []string

func (k keyer) Keys() []string {
	p := make([]string, len(k))
	copy(p, k)
	sort.Strings(p)
	return p
}

func TestFromValues(t *testing.T) {
	tests := []struct {
		input interface{}
		want  []string
	}{
		{nil, nil},
		{map[float64]string{}, nil},
		{map[int]string{1: "one", 2: "two", 3: "two"}, []string{"one", "two"}},
		{map[string]string{"foo": "bar", "baz": "bar"}, []string{"bar"}},
		{map[int]int{1: 2, 3: 4, 5: 6}, nil},
		{map[*int]string{nil: "blah"}, []string{"blah"}},
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
		{map[int]int{1: 1}, nil},    // unkeyable type
		{nil, nil},                  // empty
		{[]string{}, nil},           // empty
		{map[string]float64{}, nil}, // empty
		{"foo", New("foo")},
		{[]string{"foo", "bar", "foo", "foo"}, New("foo", "bar")},
		{map[string]int{"one": 1, "two": 2}, New("one", "two")},
		{keyer{"alpha", "charlie", "echo"}, New("alpha", "charlie", "echo")},
		{New("p", "d", "q"), New("p", "d", "q")},
		{map[string]struct{}{"fizz": {}, "buzz": {}}, New("fizz", "buzz")},
	}
	for _, test := range tests {
		got := FromKeys(test.input)
		if !got.Equals(test.want) {
			t.Errorf("MapKeys %v: got %v, want %v", test.input, got, test.want)
		}
	}
}

func TestContainsFunc(t *testing.T) {
	tests := []struct {
		input  interface{}
		needle string
		want   bool
	}{
		{[]string(nil), "x", false},
		{[]string{}, "x", false},
		{[]string{"x"}, "x", true},
		{[]string{"y"}, "x", false},
		{[]string{"a", "b", "x", "c"}, "x", true},

		{map[string]int(nil), "q", false},
		{map[string]int{}, "q", false},
		{map[string]int{"q": 1}, "q", true},
		{map[string]int{"v": 3}, "q", false},
		{map[string]float32{"q": 1, "r": 2}, "q", true},
		{map[string]float32{"s": 0, "t": 1, "f": 2, "u": 3}, "q", false},

		{Set(nil), "m", false},
		{New(), "m", false},
		{New("m"), "m", true},
		{New("p"), "m", false},
		{New("a", "b"), "m", false},
		{New("a", "m", "b"), "m", true},

		{keyer(nil), "*", false},
		{keyer{}, "*", false},
		{keyer{"*"}, "*", true},
		{keyer{"?"}, "*", false},
		{keyer{"a", "s", "*"}, "*", true},
		{keyer{"a", "s", "k"}, "*", false},
	}
	for _, test := range tests {
		got := Contains(test.input, test.needle)
		if got != test.want {
			t.Errorf("Contains(%v, %q): got %v, want %v", test.input, test.needle, got, test.want)
		}
	}
}
