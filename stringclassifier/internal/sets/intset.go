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
	"fmt"
	"sort"
	"strings"
)

// IntSet stores a set of unique int elements.
type IntSet struct {
	set map[int]present
}

// NewIntSet creates an IntSet containing the supplied initial int elements.
func NewIntSet(elements ...int) *IntSet {
	s := &IntSet{}
	s.set = make(map[int]present)
	s.Insert(elements...)
	return s
}

// Copy returns a newly allocated copy of the supplied IntSet.
func (s *IntSet) Copy() *IntSet {
	c := NewIntSet()
	if s != nil {
		for e := range s.set {
			c.set[e] = present{}
		}
	}
	return c
}

// Insert zero or more int elements into the IntSet. As expected for a Set,
// elements already present in the IntSet are simply ignored.
func (s *IntSet) Insert(elements ...int) {
	for _, e := range elements {
		s.set[e] = present{}
	}
}

// Delete zero or more int elements from the IntSet. Any elements not present
// in the IntSet are simply ignored.
func (s *IntSet) Delete(elements ...int) {
	for _, e := range elements {
		delete(s.set, e)
	}
}

// Intersect returns a new IntSet containing the intersection of the receiver
// and argument IntSets. Returns an empty set if the argument is nil.
func (s *IntSet) Intersect(other *IntSet) *IntSet {
	if other == nil {
		return NewIntSet()
	}

	// Point a and b to the maps, setting a to the smaller of the two.
	a, b := s.set, other.set
	if len(b) < len(a) {
		a, b = b, a
	}

	// Perform the intersection.
	intersect := NewIntSet()
	for e := range a {
		if _, ok := b[e]; ok {
			intersect.set[e] = present{}
		}
	}
	return intersect
}

// Disjoint returns true if the intersection of the receiver and the argument
// IntSets is the empty set. Returns true if the argument is nil or either
// IntSet is the empty set.
func (s *IntSet) Disjoint(other *IntSet) bool {
	if other == nil || len(other.set) == 0 || len(s.set) == 0 {
		return true
	}

	// Point a and b to the maps, setting a to the smaller of the two.
	a, b := s.set, other.set
	if len(b) < len(a) {
		a, b = b, a
	}

	// Check for non-empty intersection.
	for e := range a {
		if _, ok := b[e]; ok {
			return false // Early-exit because intersecting.
		}
	}
	return true
}

// Difference returns a new IntSet containing the elements in the receiver that
// are not present in the argument IntSet. Returns a copy of the receiver if
// the argument is nil.
func (s *IntSet) Difference(other *IntSet) *IntSet {
	if other == nil {
		return s.Copy()
	}

	// Insert only the elements in the receiver that are not present in the
	// argument IntSet.
	diff := NewIntSet()
	for e := range s.set {
		if _, ok := other.set[e]; !ok {
			diff.set[e] = present{}
		}
	}
	return diff
}

// Unique returns a new IntSet containing the elements in the receiver that are
// not present in the argument IntSet *and* the elements in the argument IntSet
// that are not in the receiver. Returns a copy of the receiver if the argument
// is nil.
func (s *IntSet) Unique(other *IntSet) *IntSet {
	if other == nil {
		return s.Copy()
	}

	sNotInOther := s.Difference(other)
	otherNotInS := other.Difference(s)

	// Duplicate Union implementation here to avoid extra Copy, since both
	// sNotInOther and otherNotInS are already copies.
	unique := sNotInOther
	for e := range otherNotInS.set {
		unique.set[e] = present{}
	}
	return unique
}

// Equal returns true if the receiver and the argument IntSet contain exactly
// the same elements. Returns false if the argument is nil.
func (s *IntSet) Equal(other *IntSet) bool {
	if s == nil || other == nil {
		return s == nil && other == nil
	}

	// Two sets of different length cannot have the exact same unique
	// elements.
	if len(s.set) != len(other.set) {
		return false
	}

	// Only one loop is needed. If the two sets are known to be of equal
	// length, then the two sets are equal only if exactly all of the
	// elements in the first set are found in the second.
	for e := range s.set {
		if _, ok := other.set[e]; !ok {
			return false
		}
	}

	return true
}

// Union returns a new IntSet containing the union of the receiver and argument
// IntSets. Returns a copy of the receiver if the argument is nil.
func (s *IntSet) Union(other *IntSet) *IntSet {
	union := s.Copy()
	if other != nil {
		for e := range other.set {
			union.set[e] = present{}
		}
	}
	return union
}

// Contains returns true if element is in the IntSet.
func (s *IntSet) Contains(element int) bool {
	_, in := s.set[element]
	return in
}

// Len returns the number of unique elements in the IntSet.
func (s *IntSet) Len() int {
	return len(s.set)
}

// Empty returns true if the receiver is the empty set.
func (s *IntSet) Empty() bool {
	return len(s.set) == 0
}

// Elements returns a []int of the elements in the IntSet, in no particular (or
// consistent) order.
func (s *IntSet) Elements() []int {
	elements := []int{} // Return at least an empty slice rather than nil.
	for e := range s.set {
		elements = append(elements, e)
	}
	return elements
}

// Sorted returns a sorted []int of the elements in the IntSet.
func (s *IntSet) Sorted() []int {
	elements := s.Elements()
	sort.Ints(elements)
	return elements
}

// String formats the IntSet elements as sorted ints, representing them in
// "array initializer" syntax.
func (s *IntSet) String() string {
	elements := s.Sorted()
	var quoted []string
	for _, e := range elements {
		quoted = append(quoted, fmt.Sprintf("\"%d\"", e))
	}
	return fmt.Sprintf("{%s}", strings.Join(quoted, ", "))
}
