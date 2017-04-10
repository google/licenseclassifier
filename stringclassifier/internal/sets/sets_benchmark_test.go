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
	"math/rand"
	"testing"
)

const numInts = 2048

var (
	// Random int lists:
	intsA = ints()
	intsB = ints()
)

// Int Sets:

func BenchmarkIntSets_NewIntSet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewIntSet(intsA...)
	}
}

func BenchmarkIntSets_Copy(b *testing.B) {
	s := NewIntSet(intsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Copy()
	}
}

func BenchmarkIntSets_Insert(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewIntSet()
		s.Insert(intsA...)
		s.Insert(intsB...)
	}
}

func BenchmarkIntSets_Delete(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewIntSet(intsA...)
		s.Delete(intsB...)
	}
}

func BenchmarkIntSets_Intersect(b *testing.B) {
	s := NewIntSet(intsA...)
	t := NewIntSet(intsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Intersect(t)
	}
}

func BenchmarkIntSets_Disjoint(b *testing.B) {
	s := NewIntSet(intsA...)
	t := NewIntSet(intsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Disjoint(t)
	}
}

func BenchmarkIntSets_Difference(b *testing.B) {
	s := NewIntSet(intsA...)
	t := NewIntSet(intsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Difference(t)
	}
}

func BenchmarkIntSets_Unique(b *testing.B) {
	s := NewIntSet(intsA...)
	t := NewIntSet(intsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Unique(t)
	}
}

func BenchmarkIntSets_Equal(b *testing.B) {
	s := NewIntSet(intsA...)
	t := NewIntSet(intsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Equal(t)
	}
}

func BenchmarkIntSets_Union(b *testing.B) {
	s := NewIntSet(intsA...)
	t := NewIntSet(intsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Union(t)
	}
}

func BenchmarkIntSets_Contains(b *testing.B) {
	s := NewIntSet(intsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, w := range intsB {
			s.Contains(w)
		}
	}
}

func BenchmarkIntSets_Len(b *testing.B) {
	s := NewIntSet(intsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Len()
	}
}

func BenchmarkIntSets_Empty(b *testing.B) {
	s := NewIntSet(intsA...)
	t := NewIntSet()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Empty()
		t.Empty()
	}
}

func BenchmarkIntSets_Elements(b *testing.B) {
	s := NewIntSet(intsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Elements()
	}
}

func BenchmarkIntSets_Sorted(b *testing.B) {
	s := NewIntSet(intsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Sorted()
	}
}

func BenchmarkIntSets_Int(b *testing.B) {
	s := NewIntSet(intsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.String()
	}
}

func int64s() []int64 {
	var r []int64
	for i := 0; i < numInts; i++ {
		r = append(r, rand.Int63())
	}
	return r
}

func ints() []int {
	var r []int
	for i := 0; i < numInts; i++ {
		r = append(r, rand.Int())
	}
	return r
}
