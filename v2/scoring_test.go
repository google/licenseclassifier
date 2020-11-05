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
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestLevenshteinDiff(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []diffmatchpatch.Diff
		expected int
	}{
		{
			name: "identical text",
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "equivalent text",
				},
			},
			expected: 0,
		},
		{
			name: "changed text",
			// Adjacent inverse changes get scored with the maximum of the 2 change scores
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "removed words",
				},
				{
					Type: diffmatchpatch.DiffInsert,
					Text: "inserted text here",
				},
			},
			expected: 3,
		},
		{
			name: "inserted text",
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "identical words",
				},
				{
					Type: diffmatchpatch.DiffInsert,
					Text: "inserted",
				},
			},
			expected: 1,
		},
		{
			name: "deleted text",
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "many extraneous deleted words",
				},
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "before the equivalent text",
				},
			},
			expected: 4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := diffLevenshteinWord(test.diffs); got != test.expected {
				t.Errorf("got %d wanted %d", got, test.expected)
			}
		})
	}
}

func TestScoreDiffs(t *testing.T) {
	tests := []struct {
		name     string
		license  string
		diffs    []diffmatchpatch.Diff
		expected int
	}{
		{
			name:     "identical text",
			diffs:    nil,
			expected: 0,
		},
		{
			name: "acceptable change",
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "license",
				},
				{
					Type: diffmatchpatch.DiffInsert,
					Text: "as needed",
				},
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "when necessary",
				},
			},
			expected: 2,
		},
		{
			name: "version change",
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "version",
				},
				{
					Type: diffmatchpatch.DiffInsert,
					Text: "2",
				},
			},
			expected: versionChange,
		},
		{
			name: "license name change by deletion",
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "gnu",
				},
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "lesser",
				},
			},
			expected: lesserGPLChange,
		},
		{
			name: "license name change by insertion",
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "gnu",
				},
				{
					Type: diffmatchpatch.DiffInsert,
					Text: "lesser",
				},
			},
			expected: lesserGPLChange,
		},
		{
			name:    "license name change by name insertion",
			license: "ImageMagick",
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "license",
				},
				{
					Type: diffmatchpatch.DiffInsert,
					Text: "imagemagick",
				},
			},
			expected: introducedPhraseChange,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := scoreDiffs(test.license, test.diffs); got != test.expected {
				t.Errorf("got %d, want %d", got, test.expected)
			}
		})
	}
}

func TestConfidencePercentage(t *testing.T) {
	tests := []struct {
		name           string
		klen, distance int
		expected       float64
	}{
		{
			name:     "empty text",
			klen:     0,
			distance: 0,
			expected: 1.0,
		},
		{
			name:     "99% match",
			klen:     100,
			distance: 1,
			expected: 0.99,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := confidencePercentage(test.klen, test.distance); got != test.expected {
				t.Errorf("got %v want %v", got, test.expected)
			}
		})
	}
}

func TestScore(t *testing.T) {
	tests := []struct {
		name                       string
		known, unknown             string
		expectedConf               float64
		expectedStart, expectedEnd int
	}{
		{
			name:          "identical text",
			known:         "here is some sample text",
			unknown:       "here is some sample text",
			expectedConf:  1.00,
			expectedStart: 0,
			expectedEnd:   0,
		},
		{
			name:          "close match with matching sizes",
			known:         "here is some sample text",
			unknown:       "here is different sample text",
			expectedConf:  .8,
			expectedStart: 0,
			expectedEnd:   0,
		},
		{
			name:          "close match with different sizes",
			known:         "here is some sample text",
			unknown:       "padding before here is different sample text",
			expectedConf:  .8,
			expectedStart: 2,
			expectedEnd:   0,
		},
		{
			name:          "no match due to unacceptable diff",
			known:         "here is some sample text for version 2 of the license",
			unknown:       "padding before here is different sample text for version 3 of the licenses",
			expectedConf:  0.0,
			expectedStart: 0,
			expectedEnd:   0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var trace strings.Builder
			c := NewClassifier(.8)
			c.SetTraceConfiguration(&TraceConfiguration{
				TraceLicenses: "*",
				TracePhases:   "*",
				Tracer: func(f string, args ...interface{}) {
					trace.WriteString(fmt.Sprintf(f, args...))
				},
			})
			c.AddContent("known", []byte(test.known))
			kd := c.docs["known"]
			ud := c.createTargetIndexedDocument([]byte(test.unknown))
			conf, so, eo := c.score(test.name, ud, kd, 0, ud.size())

			success := true
			if conf != test.expectedConf {
				t.Errorf("conf: got %v want %v", conf, test.expectedConf)
				success = false
			}
			if so != test.expectedStart {
				t.Errorf("start offset: got %v want %v", so, test.expectedStart)
				success = false
			}
			if eo != test.expectedEnd {
				t.Errorf("end offset: got %v want %v", so, test.expectedEnd)
				success = false
			}

			if !success {
				t.Errorf("Trace:\n%s", trace.String())
			}
		})
	}
}
