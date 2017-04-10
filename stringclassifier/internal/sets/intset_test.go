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

func checkSameIntSet(t *testing.T, set *IntSet, unique []int) {
	// Check that lengths are the same.
	want := len(unique)
	got := set.Len()

	if got != want {
		t.Errorf("NewIntSet(%v) want length %v, got %v", unique, want, got)
	}

	// Check that all ints are present in set.
	for _, s := range unique {
		want := true
		got := set.Contains(s)

		if got != want {
			t.Errorf("Contains(%v) want %v, got %v", s, want, got)
		}
	}

	// Check that all elements are present in ints.
	sort.Ints(unique)

	for i, got := range set.Sorted() {
		want := unique[i]
		if got != want {
			t.Errorf("Sorted(%d) want %v, got %v", i, want, got)
		}
	}
}

type enumTest int

const (
	et0 enumTest = iota
	et1
	et2
	et3
	et4
)

func TestNewIntSet(t *testing.T) {
	empty := NewIntSet()
	want := 0
	got := empty.Len()

	if got != want {
		t.Errorf("NewIntSet() want length %v, got %v", want, got)
	}

	unique := []int{0, 1, 2}
	set := NewIntSet(unique...)
	checkSameIntSet(t, set, unique)

	// Append an already-present element.
	nonUnique := append(unique, unique[0])
	set = NewIntSet(nonUnique...)

	// Non-unique unique should collapse to one.
	want = len(unique)
	got = set.Len()

	if got != want {
		t.Errorf("NewIntSet(%v) want length %v, got %v", nonUnique, want, got)
	}

	// Initialize with enum values cast to int.
	set = NewIntSet(int(et0), int(et1), int(et2))
	checkSameIntSet(t, set, unique)
}

func TestIntSet_Copy(t *testing.T) {
	// Check both copies represent the same set.
	base := []int{1, 2, 3}
	orig := NewIntSet(base...)
	cpy := orig.Copy()
	checkSameIntSet(t, orig, base)
	checkSameIntSet(t, cpy, base)

	// Check the two copies are independent.
	more := []int{4}
	orig.Insert(more...)
	more = append(base, more...)
	checkSameIntSet(t, orig, more)
	checkSameIntSet(t, cpy, base)
}

func TestIntSet_Insert(t *testing.T) {
	unique := []int{0, 1, 2}
	set := NewIntSet(unique...)

	// Insert existing element, which should basically be a no-op.
	set.Insert(unique[0])
	checkSameIntSet(t, set, unique)

	// Actually insert new unique elements (cast from enum values this time).
	additional := []int{int(et3), int(et4)}
	longer := append(unique, additional...)
	set.Insert(additional...)
	checkSameIntSet(t, set, longer)
}

func TestIntSet_Delete(t *testing.T) {
	unique := []int{0, 1, 2}
	set := NewIntSet(unique...)

	// Delete non-existent element, which should basically be a no-op.
	set.Delete(int(et4))
	checkSameIntSet(t, set, unique)

	// Actually delete existing elements.
	set.Delete(unique[1:]...)
	checkSameIntSet(t, set, unique[:1])
}

func TestIntSet_Intersect(t *testing.T) {
	input1 := []int{1, 3, 4, 5, 6}
	input2 := []int{2, 3, 5}

	// Check Intersect(nil) returns an empty set.
	setA := NewIntSet(input1...)
	got := setA.Intersect(nil)
	checkSameIntSet(t, got, []int{})
	// Check that the receiver is unchanged.
	checkSameIntSet(t, setA, input1)

	// Check Intersect returns the correct result.
	setB := NewIntSet(input2...)
	got = setA.Intersect(setB)
	want := []int{3, 5}
	checkSameIntSet(t, got, want)
	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setB, input2)

	// Reverse the inputs and verify Intersect produces the same results.
	setA = NewIntSet(input2...)
	setB = NewIntSet(input1...)
	got = setA.Intersect(setB)
	checkSameIntSet(t, got, want)
	// Check the sources are again unchanged.
	checkSameIntSet(t, setA, input2)
	checkSameIntSet(t, setB, input1)
}

func TestIntSet_Disjoint(t *testing.T) {
	input1 := []int{1, 3, 4, 5, 6}
	input2 := []int{2, 3, 5}
	input3 := []int{98, 99, 100}

	// Check that sets are always disjoint with the empty set or nil
	setA := NewIntSet(input1...)
	emptySet := NewIntSet()

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
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, emptySet, []int{})

	// Check two non-empty, non-nil disjoint sets.
	setC := NewIntSet(input3...)

	if disjoint := setA.Disjoint(setC); !disjoint {
		t.Errorf("Disjoint(%s, %s) want %v, got %v", setA, setC, true, disjoint)
	}

	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setC, input3)

	// Check that two intersecting sets are not Disjoint.
	setB := NewIntSet(input2...)

	if disjoint := setA.Disjoint(setB); disjoint {
		t.Errorf("Disjoint(%s, %s) want %v, got %v", setA, setB, false, disjoint)
	}

	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setB, input2)
}

func TestIntSet_Difference(t *testing.T) {
	input1 := []int{1, 3, 4, 5, 6}
	input2 := []int{2, 3, 5}
	input3 := []int{98, 99, 100}

	// Check Difference(nil) returns a copy of the receiver.
	setA := NewIntSet(input1...)
	got := setA.Difference(nil)
	checkSameIntSet(t, got, input1)
	// Check that the receiver is unchanged.
	checkSameIntSet(t, setA, input1)

	// Check A - A returns the empty set.
	got = setA.Difference(setA)

	if !got.Empty() {
		t.Errorf("Difference(%s, %s).Empty() want %v, got %v",
			setA, setA, true, false)
	}

	checkSameIntSet(t, got, []int{})
	// Check that the receiver is unchanged.
	checkSameIntSet(t, setA, input1)

	// Check A - C simply returns elements in A if A and C are disjoint.
	setC := NewIntSet(input3...)
	got = setA.Difference(setC)
	checkSameIntSet(t, got, input1)
	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setC, input3)

	// Check A - B returns elements in A not in B.
	setB := NewIntSet(input2...)
	got = setA.Difference(setB)
	want := []int{1, 4, 6}
	checkSameIntSet(t, got, want)

	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setB, input2)

	// Check B - A returns elements in B not in A.
	got = setB.Difference(setA)
	want = []int{2}
	checkSameIntSet(t, got, want)
	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setB, input2)
}

func TestIntSet_Unique(t *testing.T) {
	input1 := []int{1, 3, 4, 5, 6}
	input2 := []int{2, 3, 5}
	input3 := []int{98, 99, 100}

	// Check Unique(nil) returns a copy of the receiver.
	setA := NewIntSet(input1...)
	got := setA.Unique(nil)
	checkSameIntSet(t, got, input1)
	// Check that the receiver is unchanged.
	checkSameIntSet(t, setA, input1)

	// Check Unique returns only elements in A and B not in both A and B.
	setB := NewIntSet(input2...)
	got = setA.Unique(setB)
	want := []int{1, 2, 4, 6}
	checkSameIntSet(t, got, want)
	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setB, input2)

	// Check Unique of two disjoint sets is the Union of those sets.
	setC := NewIntSet(input3...)
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
	want = []int{1, 2, 4, 6}
	checkSameIntSet(t, union, want)
	got = setA.Unique(setB)

	if equal := union.Equal(got); !equal {
		t.Errorf("Union of differences Equal(%s, %s) want %v, got %v",
			union, got, true, equal)
	}

	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setB, input2)
}

func TestIntSet_Equal(t *testing.T) {
	input1 := []int{1, 3, 4, 5, 6}
	input2 := []int{2, 3, 5}
	input3 := []int{1, 3, 4, 5, 7}

	// Check Equal(nil) returns false.
	setA := NewIntSet(input1...)

	if equal := setA.Equal(nil); equal {
		t.Errorf("Equal(%s, %v) want %v, got %v", setA, nil, false, true)
	}

	// Check that the receiver is unchanged.
	checkSameIntSet(t, setA, input1)

	// Check Equal returns true for a set and itself.
	if equal := setA.Equal(setA); !equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, setA, true, false)
	}

	// Check that the receiver is unchanged.
	checkSameIntSet(t, setA, input1)

	// Check Equal returns false for sets of non-equal length.
	setB := NewIntSet(input2...)

	if equal := setA.Equal(setB); equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, setB, false, true)
	}

	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setB, input2)

	// Check Equal returns false for equal-length sets with different elements.
	setC := NewIntSet(input3...)

	if equal := setA.Equal(setC); equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, setC, false, true)
	}

	if equal := setC.Equal(setA); equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setC, setA, false, true)
	}

	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setC, input3)

	// Check Equal returns true for a set with itself.
	if equal := setA.Equal(setA); !equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, setA, true, false)
	}

	// Also check the source is unchanged.
	checkSameIntSet(t, setA, input1)

	// Check Equal returns true for two separate equal sets.
	anotherA := NewIntSet(input1...)

	if equal := setA.Equal(anotherA); !equal {
		t.Errorf("Equal(%s, %s) want %v, got %v", setA, anotherA, true, false)
	}

	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, anotherA, input1)

	// Check for equality comparing to nil struct.
	var nilSet *IntSet
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
	emptySet := NewIntSet()
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

func TestIntSet_Union(t *testing.T) {
	input1 := []int{1, 3, 4, 5, 6}
	input2 := []int{2, 3, 5}

	// Check Union(nil) returns a copy of the receiver.
	setA := NewIntSet(input1...)
	got := setA.Union(nil)
	checkSameIntSet(t, got, input1)
	// Check that the receiver is unchanged.
	checkSameIntSet(t, setA, input1)

	// Check Union returns the correct result.
	setB := NewIntSet(input2...)
	got = setA.Union(setB)
	want := []int{1, 2, 3, 4, 5, 6}
	checkSameIntSet(t, got, want)
	// Also check the sources are unchanged.
	checkSameIntSet(t, setA, input1)
	checkSameIntSet(t, setB, input2)

	// Reverse the inputs and verify Union produces the same results.
	setA = NewIntSet(input2...)
	setB = NewIntSet(input1...)
	got = setA.Union(setB)
	checkSameIntSet(t, got, want)
	// Check the sources are again unchanged.
	checkSameIntSet(t, setA, input2)
	checkSameIntSet(t, setB, input1)
}
