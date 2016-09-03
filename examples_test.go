package stringset

import (
	"fmt"
	"path/filepath"
	"regexp"
)

func ExampleSet_Intersect() {
	fmt.Println(New("one").Intersect(nil))
	// Output: ø
}

func ExampleSet_Union() {
	fmt.Println(New("0", "1", "2").Union(New("x")))
	// Output: {"0", "1", "2", "x"}
}

func ExampleSet_Discard() {
	nat := New("0", "1", "2", "3", "4")
	ok := nat.Discard("2", "4", "6")
	fmt.Println(ok, nat)
	// Output: true {"0", "1", "3"}
}

func ExampleSet_Add() {
	one := New("one")
	one.Add("one", "perfect", "question")
	fmt.Println(one)
	// Output: {"one", "perfect", "question"}
}

func ExampleSet_Select() {
	re := regexp.MustCompile(`[a-z]\d+`)
	s := New("a", "b15", "c9", "q").Select(re.MatchString)
	fmt.Println(s.Elements())
	// Output: [b15 c9]
}

func ExampleSet_Choose() {
	s := New("a", "ab", "abc", "abcd")
	long, ok := s.Choose(func(c string) bool { return len(c) > 3 })
	fmt.Println(long, ok)
	// Output: abcd true
}

func ExampleSet_Contains() {
	s := New("a", "b", "c", "d", "e")
	ae := s.Contains("a", "e")       // all present
	bdx := s.Contains("b", "d", "x") // x missing
	fmt.Println(ae, bdx)
	// Output: true false
}

func ExampleSet_ContainsAny() {
	s := New("a", "b", "c")
	fm := s.ContainsAny("f", "m")       // all missing
	bdx := s.ContainsAny("b", "d", "x") // b present
	fmt.Println(fm, bdx)
	// Output: false true
}

func ExampleSet_Diff() {
	a := New("a", "b", "c")
	v := New("a", "e", "i")
	fmt.Println(a.Diff(v))
	// Output: {"b", "c"}
}

func ExampleSet_Each() {
	sum := 0
	New("one", "two", "three").Each(func(s string) {
		sum += len(s)
	})
	fmt.Println(sum)
	// Output: 11
}

func ExampleSet_Pop() {
	s := New("a", "bc", "def", "ghij")
	p, ok := s.Pop(func(s string) bool {
		return len(s) == 2
	})
	fmt.Println(p, ok, s)
	// Output: bc true {"a", "def", "ghij"}
}

func ExampleSet_Partition() {
	s := New("aba", "d", "qpc", "ff")
	a, b := s.Partition(func(s string) bool {
		return s[0] == s[len(s)-1]
	})
	fmt.Println(a, b)
	// Output: {"aba", "d", "ff"} {"qpc"}
}

func ExampleSet_SymDiff() {
	s := New("a", "b", "c")
	t := New("a", "c", "t")
	fmt.Println(s.SymDiff(t))
	// Output: {"b", "t"}
}

func ExampleFromKeys() {
	s := FromKeys(map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	})
	fmt.Println(s)
	// Output: {"one", "three", "two"}
}

func ExampleFromValues() {
	s := FromValues(map[int]string{
		1: "red",
		2: "green",
		3: "red",
		4: "blue",
		5: "green",
	})
	fmt.Println(s)
	// Output: {"blue", "green", "red"}
}

func ExampleSet_Map() {
	names := New("stdio.h", "main.cc", "lib.go", "BUILD", "fixup.py")
	ext := names.Map(filepath.Ext)
	fmt.Println(ext)
	fmt.Println("Legacy:", ext.Contains(".h", ".cc"))
	// Output:
	// {"", ".cc", ".go", ".h", ".py"}
	// Legacy: true
}
