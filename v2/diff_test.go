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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	gettysburg = `Four score and seven years ago our fathers brought forth
on this continent, a new nation, conceived in Liberty, and dedicated to the
proposition that all men are created equal.`
	modifiedGettysburg = `Four score and seven years ago our fathers brought forth
on this continent, a nation that was new and improved, conceived in Liberty, and
dedicated to the proposition that all men are created equal.`
	extra = `In the current state of affairs`

	declaration = `When in the Course of human events, it becomes necessary
for one people to dissolve the political bands which have connected them with
another, and to assume among the powers of the earth, the separate and equal
station to which the Laws of Nature and of Nature's God entitle them, a decent
respect to the opinions of mankind requires that they should declare the causes
which impel them to the separation.`

	loremipsum = `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nulla
varius enim mattis, rhoncus lectus id, aliquet sem. Phasellus eget ex in dolor
feugiat ultricies. Etiam interdum sit amet nisl in placerat.  Sed vitae enim
vulputate, tempus leo commodo, accumsan nulla.`
	lessModifiedLorem = `Lorem ipsum dolor sot amet, consectetur adipiscing elit. Nulla
varius enim mattis, rhoncus lectus id, aliquet. Phasellus eget ex in dolor
feugiat ultricies. Etiam interdum sit amet nisl in placerat.  Sed vitae enim
vulputate, tempus leo commodo, accumsan nulla.`
)

func TestTextLength(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []diffmatchpatch.Diff
		expected int
	}{
		{
			name:     "empty diff",
			diffs:    nil,
			expected: 0,
		},
		{
			name: "deletion diff",
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "deleted text",
				},
			},
			expected: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := textLength(test.diffs); got != test.expected {
				t.Errorf("got %d, want %d", got, test.expected)
			}
		})
	}
}

func TestWordLen(t *testing.T) {
	tests := []struct {
		in       string
		expected int
	}{
		{
			in:       "short string",
			expected: 2,
		},
		{
			in:       "",
			expected: 0,
		},
		{
			in:       "word",
			expected: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			if got := wordLen(test.in); got != test.expected {
				t.Errorf("got %d, want %d", got, test.expected)
			}
		})
	}
}

func TestDiffing(t *testing.T) {
	tests := []struct {
		name           string
		unknown, known string
		start, end     int
		diffs          []diffmatchpatch.Diff
	}{
		{
			name:    "identical",
			unknown: declaration,
			known:   declaration,
			start:   0,
			end:     1,
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: `when in the course of human events it becomes necessary for one people to dissolve the political bands which have connected them with another and to assume among the powers of the earth the separate and equal station to which the laws of nature and of natures god entitle them a decent respect to the opinions of mankind requires that they should declare the causes which impel them to the separation`,
				},
			},
		},
		{
			name:    "lorem",
			unknown: lessModifiedLorem,
			known:   loremipsum,
			start:   0,
			end:     6,
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "lorem ipsum dolor",
				},
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "UNKNOWN",
				},
				{
					Type: diffmatchpatch.DiffInsert,
					Text: "sit",
				},
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "amet consectetur adipiscing elit nulla varius enim mattis rhoncus lectus id aliquet",
				},
				{
					Type: diffmatchpatch.DiffInsert,
					Text: "sem",
				},
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "phasellus eget ex in dolor feugiat ultricies etiam interdum sit amet nisl in placerat sed vitae enim vulputate tempus leo commodo accumsan nulla",
				},
			},
		},
		{
			name:    "whole diff retained",
			unknown: modifiedGettysburg,
			known:   gettysburg,
			start:   0,
			end:     6,
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "four score and seven years ago our fathers brought forth on this continent a",
				},
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "nation that UNKNOWN",
				},
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "new",
				},
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "and UNKNOWN",
				},
				{
					Type: diffmatchpatch.DiffInsert,
					Text: "nation",
				},
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "conceived in liberty and dedicated to the proposition that all men are created equal",
				},
			},
		},
		{
			name:    "extra at beginning",
			unknown: extra + " " + gettysburg,
			known:   gettysburg,
			start:   1,
			end:     2,
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "in the UNKNOWN UNKNOWN UNKNOWN UNKNOWN",
				},
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "four score and seven years ago our fathers brought forth on this continent a new nation conceived in liberty and dedicated to the proposition that all men are created equal",
				},
			},
		},
		{
			name:    "extra at end",
			unknown: gettysburg + " " + extra,
			known:   gettysburg,
			start:   0,
			end:     1,
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "four score and seven years ago our fathers brought forth on this continent a new nation conceived in liberty and dedicated to the proposition that all men are created equal",
				},
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "in the UNKNOWN UNKNOWN UNKNOWN UNKNOWN",
				},
			},
		},
		{
			name:    "extra at both ends",
			unknown: extra + " " + gettysburg + " " + extra,
			known:   gettysburg,
			start:   1,
			end:     2,
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "in the UNKNOWN UNKNOWN UNKNOWN UNKNOWN",
				},
				{
					Type: diffmatchpatch.DiffEqual,
					Text: "four score and seven years ago our fathers brought forth on this continent a new nation conceived in liberty and dedicated to the proposition that all men are created equal",
				},
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "in the UNKNOWN UNKNOWN UNKNOWN UNKNOWN",
				},
			},
		},
		{
			name:    "completely different",
			unknown: "this",
			known:   "that",
			start:   1,
			end:     2,
			diffs: []diffmatchpatch.Diff{
				{
					Type: diffmatchpatch.DiffDelete,
					Text: "UNKNOWN",
				},
				{
					Type: diffmatchpatch.DiffInsert,
					Text: "that",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := NewClassifier(.8)
			c.AddContent("known", []byte(test.known))
			kd := c.docs["known"]
			ud := c.createTargetIndexedDocument([]byte(test.unknown))
			diffs := docDiff("known", ud, 0, ud.size(), kd, 0, kd.size())
			start, end := diffRange(kd.normalized(), diffs)
			if start != test.start {
				t.Errorf("start: got %d want %d", start, test.start)
			}
			if end != test.end {
				t.Errorf("end: got %d want %d", end, test.end)
			}
			if !cmp.Equal(diffs, test.diffs) {
				t.Errorf(cmp.Diff(diffs, test.diffs))
			}
		})
	}
}
