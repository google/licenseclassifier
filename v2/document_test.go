// Copyright 2020 Google Inc.
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

package classifier

import (
	"fmt"
	"testing"
)

func TestDictionary(t *testing.T) {
	d := newDictionary()
	if len(d.words) > 0 {
		t.Errorf("new dictionary should not have words populated")
	}
	if len(d.indices) > 0 {
		t.Errorf("new dictionary should not have indices populated")
	}

	// Add a word to the dictionary
	d.add("hello")
	// verify internal contents
	if got := len(d.words); got != 1 {
		t.Errorf("dictionary has %d words, expected 1", got)
	}
	if got := len(d.indices); got != 1 {
		t.Errorf("dictionary has %d indices, expected 1", got)
	}
	if got := d.getIndex("hello"); got != 1 {
		t.Errorf("dictionary index: got %d, want 1", got)
	}
	if got := d.getWord(1); got != "hello" {
		t.Errorf("dictionary word: got %q, want %q", got, "hello")
	}

	// Adding the same word to the dictionary doesn't change the dictionary
	d.add("hello")
	// verify internal contents
	if got := len(d.words); got != 1 {
		t.Errorf("dictionary has %d words, expected 1", got)
	}
	if got := len(d.indices); got != 1 {
		t.Errorf("dictionary has %d indices, expected 1", got)
	}
	if got := d.getIndex("hello"); got != 1 {
		t.Errorf("dictionary index: got %d, want 1", got)
	}
	if got := d.getWord(1); got != "hello" {
		t.Errorf("dictionary word: got %q, want %q", got, "hello")
	}

	// Fetching an unknown index returns the special value
	if got := d.getWord(2); got != unknownWord {
		t.Errorf("dictionary word: got %q, want %q", got, unknownWord)
	}

	// Fetching an unknown word returns the special value
	if got := d.getIndex("unknown"); got != unknownIndex {
		t.Errorf("dictionary word: got %d, want %d", got, unknownIndex)
	}
}

func TestComputeQ(t *testing.T) {
	tests := []struct {
		threshold float64
		expected  int
	}{
		{
			threshold: .9,
			expected:  9,
		},
		{
			threshold: .8,
			expected:  4,
		},
		{
			threshold: .67,
			expected:  2,
		},
		{
			threshold: .5,
			expected:  1,
		},
		{
			threshold: 0.0,
			expected:  1,
		},
		{
			threshold: 1.0,
			expected:  10,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("threshold test %d", i), func(t *testing.T) {
			if actual := computeQ(test.threshold); actual != test.expected {
				t.Errorf("got %v want %v", actual, test.expected)
			}
		})
	}
}
