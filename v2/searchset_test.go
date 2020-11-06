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
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
)

// hundredLicenseText is a baseline for debugging precise ordering issues to demonstrate that the token-run assembly process can identify maximally fragmented entries successfully.
var hundredLicenseText = `
1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60 61 62 63 64 65 66 67 68 69 70 71 72 73 74 75 76 77 78 79 80 81 82 83 84 85 86 87 88 89 90 91 92 93 94 95 96 97 98 99 100`

// prefixMissingText exercises the error margin detection case at the beginning of a body of text.
var prefixMissingText = `
21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60 61 62 63 64 65 66 67 68 69 70 71 72 73 74 75 76 77 78 79 80 81 82 83 84 85 86 87 88 89 90 91 92 93 94 95 96 97 98 99 100`

// suffixMissingText exercises the error margin detection case at the beginning of a body of text.
var suffixMissingText = `
1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60 61 62 63 64 65 66 67 68 69 70 71 72 73 74 75 76 77 78 79 80`

// fragmentedText is worst-case fragmentation that requires maximum reassembly with full error tolerance.
var fragmentedText = `
1 2 3 4 X 6 7 8 9 X 11 12 13 14 X 16 17 18 19 X 21 22 23 24 X 26 27 28 29 X 31 32 33 34 X 36 37 38 39 X 41 42 43 44 X 46 47 48 49 X 51 52 53 54 X 56 57 58 59 X 61 62 63 64 X 66 67 68 69 X 71 72 73 74 X 76 77 78 79 X 81 82 83 84 X 86 87 88 89 X 91 92 93 94 X 96 97 98 99 X`

// bigChunkText has a gap of maximal length to ensure that reassembly works.
var bigChunkText = `
1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 X X X X X X X X X X X X X X X X X X X X 70 71 72 73 74 75 76 77 78 79 80 81 82 83 84 85 86 87 88 89 90 91 92 93 94 95 96 97 98 99 100`

// The 50 license text leverages repeated ordered tokens to exercise reassembly options of repeated text in a worst-case situation.
var fiftyLicenseText = `
1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50`

var repeatedWithBreakagesText = `
1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 X X X X X X X X X X X X X X X X X 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 X X X`

func TestSearchSet_New(t *testing.T) {
	tests := []struct {
		description string
		text        string
		q           int
		want        *searchSet
	}{
		{
			description: "Empty string",
			text:        "",
			q:           4,
			want: &searchSet{
				Tokens:         []indexedToken{},
				Hashes:         make(hash),
				Checksums:      nil,
				ChecksumRanges: nil,
			},
		},
		{
			description: "Small string",
			text:        "Hello world",
			q:           4,
			want: &searchSet{
				Tokens: []indexedToken{
					{Index: 0, Line: 1, ID: 1},
					{Index: 1, Line: 1, ID: 2},
				},
				Hashes:         hash{1957950203: tokenRanges{&tokenRange{Start: 0, End: 2}}},
				Checksums:      []uint32{1957950203},
				ChecksumRanges: tokenRanges{&tokenRange{Start: 0, End: 2}},
				nodes:          []*node{{1957950203, &tokenRange{Start: 0, End: 2}}},
				q:              2,
			},
		},
	}

	for _, tt := range tests {
		var trace strings.Builder
		c := NewClassifier(.8) // This value doesn't affect the test.
		c.SetTraceConfiguration(&TraceConfiguration{
			TraceLicenses: "*",
			TracePhases:   "*",
			Tracer: func(f string, args ...interface{}) {
				trace.WriteString(fmt.Sprintf(f, args...))
			},
		})
		c.AddContent("text", []byte(tt.text))
		if got := newSearchSet(c.docs["text"], tt.q); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("New(%q) = %+v, want %+v", tt.description, spew.Sdump(got), spew.Sdump(tt.want))
			t.Errorf("Trace:\n%s", trace.String())
		}
	}
}
func TestFindPotentialMatches(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		target       string
		confidence   float64
		expectedHits int
	}{
		{
			name:         "maximally fragmented",
			src:          hundredLicenseText,
			target:       fragmentedText,
			confidence:   .8,
			expectedHits: 1,
		},
		{
			name:         "prefix missing",
			src:          hundredLicenseText,
			target:       prefixMissingText,
			confidence:   .8,
			expectedHits: 1,
		},
		{
			name:         "suffix missing",
			src:          hundredLicenseText,
			target:       suffixMissingText,
			confidence:   .8,
			expectedHits: 1,
		},
		{
			name:         "maximum-length error",
			src:          hundredLicenseText,
			target:       bigChunkText,
			confidence:   .8,
			expectedHits: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var trace strings.Builder
			c := NewClassifier(test.confidence)
			c.SetTraceConfiguration(&TraceConfiguration{
				TraceLicenses: "*",
				TracePhases:   "*",
				Tracer: func(f string, args ...interface{}) {
					trace.WriteString(fmt.Sprintf(f, args...))
				},
			})
			c.AddContent("source", []byte(test.src))

			doc := c.createTargetIndexedDocument([]byte(test.target))
			doc.generateSearchSet(c.q)
			hits := c.findPotentialMatches(c.docs["source"].s, doc.s, test.confidence)
			if actual := len(hits); actual != test.expectedHits {
				t.Errorf("got %d hits, wanted %d", actual, test.expectedHits)
				t.Errorf("Trace:\n%s", trace.String())
			}
		})
	}
}

func TestFuseRanges(t *testing.T) {
	// This test verifies that target ordering doesn't affect how ranges
	// get fused. The data that is input for fuseRanges is not always
	// ordered deterministically since it's the product of a map iteration.
	// This was a bug that occurred during development.
	tests := []struct {
		name string
		in   matchRanges
		out  matchRanges
		conf float64
		size int
	}{
		{
			name: "in target order",
			conf: .8,
			size: 100,
			in: matchRanges{
				&matchRange{
					SrcStart:      50,
					SrcEnd:        93,
					TargetStart:   0,
					TargetEnd:     43,
					TokensClaimed: 43,
				},
				&matchRange{
					SrcStart:      0,
					SrcEnd:        43,
					TargetStart:   0,
					TargetEnd:     43,
					TokensClaimed: 43,
				},
				&matchRange{
					SrcStart:      10,
					SrcEnd:        47,
					TargetStart:   60,
					TargetEnd:     97,
					TokensClaimed: 37,
				},
				&matchRange{
					SrcStart:      60,
					SrcEnd:        97,
					TargetStart:   60,
					TargetEnd:     97,
					TokensClaimed: 37,
				},
			},
			out: matchRanges{
				&matchRange{
					SrcStart:      0,
					SrcEnd:        97,
					TargetStart:   0,
					TargetEnd:     97,
					TokensClaimed: 80,
				},
			},
		},
		{
			name: "not in-target order",
			conf: .8,
			size: 100,
			in: matchRanges{
				&matchRange{
					SrcStart:      0,
					SrcEnd:        43,
					TargetStart:   0,
					TargetEnd:     43,
					TokensClaimed: 43,
				},
				&matchRange{
					SrcStart:      50,
					SrcEnd:        93,
					TargetStart:   0,
					TargetEnd:     43,
					TokensClaimed: 43,
				},
				&matchRange{
					SrcStart:      60,
					SrcEnd:        97,
					TargetStart:   60,
					TargetEnd:     97,
					TokensClaimed: 37,
				},
				&matchRange{
					SrcStart:      10,
					SrcEnd:        47,
					TargetStart:   60,
					TargetEnd:     97,
					TokensClaimed: 37,
				},
			},
			out: matchRanges{
				&matchRange{
					SrcStart:      0,
					SrcEnd:        97,
					TargetStart:   0,
					TargetEnd:     97,
					TokensClaimed: 80,
				},
			},
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
			runs := c.detectRuns(test.name, test.in, 100, 100, test.conf, 4)
			actual := c.fuseRanges(test.name, test.in, test.conf, test.size, runs, 100)
			if !cmp.Equal(actual, test.out) {
				t.Errorf("%v: %v", test.name, cmp.Diff(actual, test.out))
				t.Errorf("Trace:\n%s", trace.String())
			}
		})
	}
}

func TestDetectRuns(t *testing.T) {
	tests := []struct {
		name                          string
		matched                       matchRanges
		targetLength, subsetLength, q int
		threshold                     float64
		expected                      []matchRange
	}{
		{
			// For an exact match on 100 accurate tokens, the first q-gram
			// is the only possible location hit we can return.
			name:         "precise matching on perfect runs",
			threshold:    1.0,
			targetLength: 100,
			subsetLength: 100,
			q:            4,
			matched: matchRanges{
				&matchRange{TargetStart: 0, TargetEnd: 100},
			},
			expected: []matchRange{
				{SrcStart: 0, SrcEnd: 4},
			},
		},
		{
			// For an 80% match on 100 accurate tokens, the first 20 token
			// positions represent possible matches.
			name:         "approximate matching on perfect runs",
			threshold:    0.8,
			targetLength: 100,
			subsetLength: 100,
			q:            4,
			matched: matchRanges{
				&matchRange{TargetStart: 0, TargetEnd: 100},
			},
			expected: []matchRange{
				{SrcStart: 0, SrcEnd: 24},
			},
		},
		{
			name:         "multiple runs in a single target",
			threshold:    0.8,
			targetLength: 100,
			subsetLength: 10,
			q:            4,
			matched: matchRanges{
				&matchRange{TargetStart: 0, TargetEnd: 10},
				&matchRange{TargetStart: 20, TargetEnd: 25},
				&matchRange{TargetStart: 50, TargetEnd: 60},
				&matchRange{TargetStart: 70, TargetEnd: 77},
			},
			expected: []matchRange{
				// Runs end on 4-gram boundaries
				{SrcStart: 0, SrcEnd: 6},
				// The run starts early because of error tolerance
				{SrcStart: 48, SrcEnd: 56},
			},
		},
		{
			name:         "bridge broken runs in a single target",
			threshold:    0.8,
			targetLength: 100,
			subsetLength: 10,
			q:            4,
			matched: matchRanges{
				&matchRange{TargetStart: 20, TargetEnd: 25},
				&matchRange{TargetStart: 26, TargetEnd: 30},
				&matchRange{TargetStart: 60, TargetEnd: 67},
				&matchRange{TargetStart: 68, TargetEnd: 72},
			},
			expected: []matchRange{
				// Runs end on 4-gram boundaries and start early because
				// of error tolerance.
				{SrcStart: 19, SrcEnd: 25},
				{SrcStart: 59, SrcEnd: 67},
			},
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
			if got := c.detectRuns(test.name, test.matched, test.targetLength, test.subsetLength, test.threshold, test.q); !cmp.Equal(got, test.expected) {
				t.Errorf(cmp.Diff(got, test.expected))
				t.Errorf("Trace:\n%s", trace.String())
			}

		})
	}
}
