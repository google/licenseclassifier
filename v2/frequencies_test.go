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

import "testing"

func TestTokenSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		sim  float64
	}{
		{
			name: "identical match",
			a:    "this text is the same in both scenarios",
			b:    "this text is the same in both scenarios",
			sim:  1.0,
		},
		{
			name: "no match",
			a:    "this text is the same in both scenarios",
			b:    "completely different stuff here",
			sim:  0.0,
		},
		{
			name: "half match",
			a:    "this text is one sample sentence",
			b:    "that text is some different sample",
			sim:  0.5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := NewClassifier(.8) // This value doesn't affect the test.
			c.AddContent("b", []byte(test.b))
			a := c.createTargetIndexedDocument([]byte(test.a))
			if actual := a.tokenSimilarity(c.docs["b"]); actual != test.sim {
				t.Errorf("got %v want %v", actual, test.sim)
			}
		})
	}
}
