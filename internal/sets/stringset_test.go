// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sets

import (
	"sort"
	"testing"
)

func checkSameStringSet(t *testing.T, set *StringSet, unique []string) {
	// Check that lengths are the same.
	want := len(unique)
	got := set.Len()

	if got != want {
		t.Errorf("NewStringSet(%v) want length %v, got %v", unique, want, got)
	}

	// Check that all strings are present in set.
	for _, s := range unique {
		want := true
		got := set.Contains(s)

		if got != want {
			t.Errorf("Contains(%v) want %v, got %v", s, want, got)
		}
	}

	// Check that all elements are present in strings.
	sort.Strings(unique)

	for i, got := range set.Sorted() {
		want := unique[i]

		if got != want {
			t.Errorf("Sorted(%d) want %v, got %v", i, want, got)
		}
	}
}

func TestNewStringSet(t *testing.T) {
	empty := NewStringSet()
	want := 0
	got := empty.Len()

	if got != want {
		t.Errorf("NewStringSet() want length %v, got %v", want, got)
	}

	unique := []string{"a", "b", "c"}
	set := NewStringSet(unique...)
	checkSameStringSet(t, set, unique)

	// Append an already-present element.
	nonUnique := append(unique, unique[0])
	set = NewStringSet(nonUnique...)

	// Non-unique unique should collapse to one.
	want = len(unique)
	got = set.Len()

	if got != want {
		t.Errorf("NewStringSet(%v) want length %v, got %v", nonUnique, want, got)
	}
}

func TestStringSet_Copy(t *testing.T) {
	// Check both copies represent the same set.
	base := []string{"a", "b", "c"}
	orig := NewStringSet(base...)
	cpy := orig.Copy()
	checkSameStringSet(t, orig, base)
	checkSameStringSet(t, cpy, base)

	// Check the two copies are independent.
	more := []string{"d"}
	orig.Insert(more...)
	more = append(base, more...)
	checkSameStringSet(t, orig, more)
	checkSameStringSet(t, cpy, base)
}

func TestStringSet_Insert(t *testing.T) {
	unique := []string{"a", "b", "c"}
	set := NewStringSet(unique...)

	// Insert existing element, which should basically be a no-op.
	set.Insert(unique[0])
	checkSameStringSet(t, set, unique)

	// Actually insert new unique elements.
	additional := []string{"d", "e"}
	longer := append(unique, additional...)
	set.Insert(additional...)
	checkSameStringSet(t, set, longer)
}

func TestStringSet_Delete(t *testing.T) {
	unique := []string{"a", "b", "c"}
	set := NewStringSet(unique...)

	// Delete non-existent element, which should basically be a no-op.
	set.Delete("z")
	checkSameStringSet(t, set, unique)

	// Actually delete existing elements.
	set.Delete(unique[1:]...)
	checkSameStringSet(t, set, unique[:1])
}

func TestStringSet_Intersect(t *testing.T) {
	input1 := []string{"a", "c", "d", "e", "f"}
	input2 := []string{"b", "c", "e"}

	// Check Intersect(nil) returns an empty set.
	setA := NewStringSet(input1...)
	got := setA.Intersect(nil)
	checkSameStringSet(t, got, []string{})
	// Check that the receiver is unchanged.
	checkSameStringSet(t, setA, input1)

	// Check Intersect returns the correct result.
	setB := NewStringSet(input2...)
	got = setA.Intersect(setB)
	want := []string{"c", "e"}
	checkSameStringSet(t, got, want)
	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setB, input2)

	// Reverse the inputs and verify Intersect produces the same results.
	setA = NewStringSet(input2...)
	setB = NewStringSet(input1...)
	got = setA.Intersect(setB)
	checkSameStringSet(t, got, want)
	// Check the sources are again unchanged.
	checkSameStringSet(t, setA, input2)
	checkSameStringSet(t, setB, input1)
}

func TestStringSet_Disjoint(t *testing.T) {
	input1 := []string{"a", "c", "d", "e", "f"}
	input2 := []string{"b", "c", "e"}
	input3 := []string{"x", "y", "z"}

	// Check that sets are always disjoint with the empty set or nil
	setA := NewStringSet(input1...)
	emptySet := NewStringSet()

	if disjoint := setA.Disjoint(nil); !disjoint {
		t.Errorf("Disjoint(%s, %v) want %v, got %v", setA, nil, true, disjoint)
	}

	if disjoint := setA.Disjoint(emptySet); !disjoint {
		t.Errorf("Disjoint(%s, %s) want %v, got %v", setA, emptySet, true, disjoint)
	}

	if disjoint := emptySet.Disjoint(setA); !disjoint {
		t.Errorf("Disjoint(%s, %s) want %v, got %v", emptySet, setA, true, disjoint)
	}

	if disjoint := emptySet.Disjoint(emptySet); !disjoint {
		t.Errorf("Disjoint(%s, %s) want %v, got %v", emptySet, emptySet, true, disjoint)
	}

	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, emptySet, []string{})

	// Check two non-empty, non-nil disjoint sets.
	setC := NewStringSet(input3...)

	if disjoint := setA.Disjoint(setC); !disjoint {
		t.Errorf("Disjoint(%s, %s) want %v, got %v", setA, setC, true, disjoint)
	}

	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setC, input3)

	// Check that two intersecting sets are not Disjoint.
	setB := NewStringSet(input2...)

	if disjoint := setA.Disjoint(setB); disjoint {
		t.Errorf("Disjoint(%s, %s) want %v, got %v", setA, setB, false, disjoint)
	}

	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setB, input2)
}

func TestStringSet_Difference(t *testing.T) {
	input1 := []string{"a", "c", "d", "e", "f"}
	input2 := []string{"b", "c", "e"}
	input3 := []string{"x", "y", "z"}

	// Check Difference(nil) returns a copy of the receiver.
	setA := NewStringSet(input1...)
	got := setA.Difference(nil)
	checkSameStringSet(t, got, input1)
	// Check that the receiver is unchanged.
	checkSameStringSet(t, setA, input1)

	// Check A - A returns the empty set.
	got = setA.Difference(setA)

	if !got.Empty() {
		t.Errorf("Difference(%s, %s).Empty() want %v, got %v",
			setA, setA, true, false)
	}

	checkSameStringSet(t, got, []string{})
	// Check that the receiver is unchanged.
	checkSameStringSet(t, setA, input1)

	// Check A - C simply returns elements in A if A and C are disjoint.
	setC := NewStringSet(input3...)
	got = setA.Difference(setC)
	checkSameStringSet(t, got, input1)
	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setC, input3)

	// Check A - B returns elements in A not in B.
	setB := NewStringSet(input2...)
	got = setA.Difference(setB)
	want := []string{"a", "d", "f"}
	checkSameStringSet(t, got, want)

	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setB, input2)

	// Check B - A returns elements in B not in A.
	got = setB.Difference(setA)
	want = []string{"b"}
	checkSameStringSet(t, got, want)
	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setB, input2)
}

func TestStringSet_Unique(t *testing.T) {
	input1 := []string{"a", "c", "d", "e", "f"}
	input2 := []string{"b", "c", "e"}
	input3 := []string{"x", "y", "z"}

	// Check Unique(nil) returns a copy of the receiver.
	setA := NewStringSet(input1...)
	got := setA.Unique(nil)
	checkSameStringSet(t, got, input1)
	// Check that the receiver is unchanged.
	checkSameStringSet(t, setA, input1)

	// Check Unique returns only elements in A and B not in both A and B.
	setB := NewStringSet(input2...)
	got = setA.Unique(setB)
	want := []string{"a", "b", "d", "f"}
	checkSameStringSet(t, got, want)
	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setB, input2)

	// Check Unique of two disjoint sets is the Union of those sets.
	setC := NewStringSet(input3...)
	got = setA.Unique(setC)
	union := setA.Union(setC)

	if equal := union.Equal(got); !equal {
		t.Errorf("Union of disjoint Equal(%s, %s) want %v, got %v",
			union, got, true, equal)
	}

	// Check Unique is the Union of A - B and B - A.
	aNotInB := setA.Difference(setB)
	bNotInA := setB.Difference(setA)
	union = aNotInB.Union(bNotInA)
	want = []string{"a", "b", "d", "f"}
	checkSameStringSet(t, union, want)
	got = setA.Unique(setB)

	if equal := union.Equal(got); !equal {
		t.Errorf("Union of differences Equal(%s, %s) want %v, got %v",
			union, got, true, equal)
	}

	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setB, input2)
}

func TestStringSet_Equal(t *testing.T) {
	input1 := []string{"a", "c", "d", "e", "f"}
	input2 := []string{"b", "c", "e"}
	input3 := []string{"a", "c", "d", "e", "g"}

	// Check Equal(nil) returns false.
	setA := NewStringSet(input1...)

	if equal := setA.Equal(nil); equal {
		t.Errorf("Equal(%s, %v) want %v, got %v", setA, nil, false, true)
	}

	// Check that the receiver is unchanged.
	checkSameStringSet(t, setA, input1)

	// Check Equal returns true for a set and itself.
	if equal := setA.Equal(setA); !equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, setA, true, false)
	}

	// Check that the receiver is unchanged.
	checkSameStringSet(t, setA, input1)

	// Check Equal returns false for sets of non-equal length.
	setB := NewStringSet(input2...)

	if equal := setA.Equal(setB); equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, setB, false, true)
	}

	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setB, input2)

	// Check Equal returns false for equal-length sets with different elements.
	setC := NewStringSet(input3...)

	if equal := setA.Equal(setC); equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, setC, false, true)
	}

	if equal := setC.Equal(setA); equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setC, setA, false, true)
	}

	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setC, input3)

	// Check Equal returns true for a set with itself.
	if equal := setA.Equal(setA); !equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, setA, true, false)
	}

	// Also check the source is unchanged.
	checkSameStringSet(t, setA, input1)

	// Check Equal returns true for two separate equal sets.
	anotherA := NewStringSet(input1...)

	if equal := setA.Equal(anotherA); !equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, anotherA, true, false)
	}

	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, anotherA, input1)

	// Check for equality comparing to nil struct.
	var nilSet *StringSet
	if equal := nilSet.Equal(setA); equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", nilSet, setA, false, true)
	}
	if equal := setA.Equal(nilSet); equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, nilSet, false, true)
	}
	if equal := nilSet.Equal(nilSet); !equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", nilSet, nilSet, true, false)
	}

	// Edge case: consider the empty set to be different than the nil set.
	emptySet := NewStringSet()
	if equal := nilSet.Equal(emptySet); equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", nilSet, emptySet, false, true)
	}
	if equal := emptySet.Equal(nilSet); equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", emptySet, nilSet, false, true)
	}
	if equal := emptySet.Equal(emptySet); !equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", emptySet, emptySet, true, false)
	}
}

func TestStringSet_Union(t *testing.T) {
	input1 := []string{"a", "c", "d", "e", "f"}
	input2 := []string{"b", "c", "e"}

	// Check Union(nil) returns a copy of the receiver.
	setA := NewStringSet(input1...)
	got := setA.Union(nil)
	checkSameStringSet(t, got, input1)
	// Check that the receiver is unchanged.
	checkSameStringSet(t, setA, input1)

	// Check Union returns the correct result.
	setB := NewStringSet(input2...)
	got = setA.Union(setB)
	want := []string{"a", "b", "c", "d", "e", "f"}
	checkSameStringSet(t, got, want)
	// Also check the sources are unchanged.
	checkSameStringSet(t, setA, input1)
	checkSameStringSet(t, setB, input2)

	// Reverse the inputs and verify Union produces the same results.
	setA = NewStringSet(input2...)
	setB = NewStringSet(input1...)
	got = setA.Union(setB)
	checkSameStringSet(t, got, want)
	// Check the sources are again unchanged.
	checkSameStringSet(t, setA, input2)
	checkSameStringSet(t, setB, input1)
}
