package stringset

import (
	"reflect"
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
	if s != nil {
		t.Errorf("New() returned non-nil: %v", s)
	}

	s = New("something")
	if s.Empty() {
		t.Errorf("Nonempty Set is reported empty: %v", s)
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
	if got := s.Keys(); !reflect.DeepEqual(got, wantKeys) {
		t.Errorf("s.Keys(): got %+q, want %+q", got, wantKeys)
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
		if got, want := s.Contains(w), i < count; got != want {
			t.Errorf("s.Contains(%q): got %v, want %v", w, got, want)
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

func TestUnion(t *testing.T) {
	vowels := New("e", "o", "i", "a", "u")
	vkeys := []string{"a", "e", "i", "o", "u"}

	consonants := New("h", "f", "b", "d", "g", "c")

	if got := vowels.Union(nil).Keys(); !reflect.DeepEqual(got, vkeys) {
		t.Errorf("Vowels ∪ ø: got %+q, want %+q", got, vkeys)
	}
	if got := New().Union(vowels).Keys(); !reflect.DeepEqual(got, vkeys) {
		t.Errorf("ø ∪ Vowels: got %+q, want %+q", got, vkeys)
	}

	want := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "o", "u"}
	if got := vowels.Union(consonants).Keys(); !reflect.DeepEqual(got, want) {
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
		got := test.left.Intersect(test.right).Keys()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%v ∩ %v: got %+q, want %+q", test.left, test.right, got, test.want)
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
		got := test.left.Diff(test.right).Keys()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%v \\ %v: got %+q, want %+q", test.left, test.right, got, test.want)
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
		if got := test.before.Keys(); !reflect.DeepEqual(got, test.want) {
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
		if got := test.before.Keys(); !reflect.DeepEqual(got, test.want) {
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
		if got := test.before.Keys(); !reflect.DeepEqual(got, test.want) {
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
		if got := test.before.Keys(); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Discard %v: got %+q, want %+q", test.before, got, test.want)
		}
		if ok != test.changed {
			t.Errorf("Discard %v reported change=%v, want %v", test.before, ok, test.changed)
		}
	}
}
