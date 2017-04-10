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
package pq

import "testing"

func TestQueue(t *testing.T) {
	pq := NewQueue(func(x, y interface{}) bool {
		return x.(string) < y.(string)
	}, nil)
	if n := pq.Len(); n != 0 {
		t.Fatalf("pq.Len() = %d want 0", n)
	}
	pq.Push("Go")
	pq.Push("C++")
	pq.Push("Java")
	for i, want := range []string{"C++", "Go", "Java"} {
		wantLen := 3 - i
		if n := pq.Len(); n != wantLen {
			t.Fatalf("pq.Len() = %d want %d", n, wantLen)
		}
		if s := pq.Min().(string); s != want {
			t.Fatalf("pq.Min() = %q want %q", s, want)
		}
		if n := pq.Len(); n != wantLen {
			t.Fatalf("pq.Len() = %d want %d", n, wantLen)
		}
		if s := pq.Pop().(string); s != want {
			t.Fatalf("pq.Pop() = %q want %q", s, want)
		}
		if n := pq.Len(); n != wantLen-1 {
			t.Fatalf("pq.Len() = %d want %d", n, wantLen-1)
		}
	}
	if n := pq.Len(); n != 0 {
		t.Fatalf("pq.Len() = %d want 0", n)
	}
}

// Test that reprioritizing an item works.
func TestFix(t *testing.T) {
	type item struct {
		value int
		index int
	}
	pq := NewQueue(func(x, y interface{}) bool {
		return x.(*item).value < y.(*item).value
	}, func(x interface{}, index int) {
		x.(*item).index = index
	})
	if n := pq.Len(); n != 0 {
		t.Fatalf("pq.Len() = %d want 0", n)
	}
	i1 := &item{value: 1}
	i2 := &item{value: 2}
	i3 := &item{value: 3}
	pq.Push(i3)
	pq.Push(i1)
	pq.Push(i2)
	if n := pq.Len(); n != 3 {
		t.Fatalf("pq.Len() = %d want 3", n)
	}
	for i, it := range []*item{i1, i2, i3} {
		if i == 0 && it.index != 0 {
			t.Errorf("item %+v want index 0", it)
		}
		if it.value != i+1 {
			t.Errorf("item %+v want value %d", it, i+1)
		}
	}
	i1.value = 4
	pq.Fix(i1.index)
	if n := pq.Len(); n != 3 {
		t.Fatalf("pq.Len() = %d want 3", n)
	}
	for i, it := range []*item{i2, i3, i1} {
		if i == 0 && it.index != 0 {
			t.Errorf("item %+v want index 0", it)
		}
		if it.value != i+2 {
			t.Errorf("item %+v want value %d", it, i+2)
		}
	}
	for i, want := range []int{2, 3, 4} {
		wantLen := 3 - i
		if n := pq.Len(); n != wantLen {
			t.Fatalf("pq.Len() = %d want %d", n, wantLen)
		}
		if it := pq.Min().(*item); it.value != want {
			t.Fatalf("pq.Min() = %+v want value %d", it, want)
		}
		if n := pq.Len(); n != wantLen {
			t.Fatalf("pq.Len() = %d want %d", n, wantLen)
		}
		if it := pq.Pop().(*item); it.value != want {
			t.Fatalf("pq.Pop() = %+v want value %d", it, want)
		}
		if n := pq.Len(); n != wantLen-1 {
			t.Fatalf("pq.Len() = %d want %d", n, wantLen-1)
		}
	}
	if n := pq.Len(); n != 0 {
		t.Fatalf("pq.Len() = %d want 0", n)
	}
}

func TestRemove(t *testing.T) {
	type item struct {
		value int
		index int
	}
	pq := NewQueue(func(x, y interface{}) bool {
		return x.(*item).value < y.(*item).value
	}, func(x interface{}, index int) {
		x.(*item).index = index
	})
	if n := pq.Len(); n != 0 {
		t.Fatalf("pq.Len() = %d want 0", n)
	}
	i1 := &item{value: 1}
	i2 := &item{value: 2}
	i3 := &item{value: 3}
	pq.Push(i3)
	pq.Push(i1)
	pq.Push(i2)
	if n := pq.Len(); n != 3 {
		t.Fatalf("pq.Len() = %d want 3", n)
	}
	for i, it := range []*item{i1, i2, i3} {
		if i == 0 && it.index != 0 {
			t.Errorf("item %+v want index 0", it)
		}
		if it.value != i+1 {
			t.Errorf("item %+v want value %d", it, i+1)
		}
	}
	pq.Remove(i3.index)
	if n := pq.Len(); n != 2 {
		t.Fatalf("pq.Len() = %d want 2", n)
	}
	for i, it := range []*item{i1, i2} {
		if i == 0 && it.index != 0 {
			t.Errorf("item %+v want index 0", it)
		}
		if it.value != i+1 {
			t.Errorf("item %+v want value %d", it, i+2)
		}
	}
	for i, want := range []int{1, 2} {
		wantLen := 2 - i
		if n := pq.Len(); n != wantLen {
			t.Fatalf("pq.Len() = %d want %d", n, wantLen)
		}
		if it := pq.Min().(*item); it.value != want {
			t.Fatalf("pq.Min() = %+v want value %d", it, want)
		}
		if n := pq.Len(); n != wantLen {
			t.Fatalf("pq.Len() = %d want %d", n, wantLen)
		}
		if it := pq.Pop().(*item); it.value != want {
			t.Fatalf("pq.Pop() = %+v want value %d", it, want)
		}
		if n := pq.Len(); n != wantLen-1 {
			t.Fatalf("pq.Len() = %d want %d", n, wantLen-1)
		}
	}
	if n := pq.Len(); n != 0 {
		t.Fatalf("pq.Len() = %d want 0", n)
	}
}
