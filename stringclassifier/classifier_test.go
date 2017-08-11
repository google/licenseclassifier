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
package stringclassifier

import (
	"reflect"
	"sort"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	gettysburg = `Four score and seven years ago our fathers brought forth
on this continent, a new nation, conceived in Liberty, and dedicated to the
proposition that all men are created equal.`
	modifiedGettysburg = `Four score and seven years ago our fathers brought forth
on this continent, a nation that was new and improved, conceived in Liberty, and
dedicated to the proposition that all men are created equal.`

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
	modifiedLorem = `Lorem ipsum dolor amet, consectetur adipiscing elit. Nulla
varius enim mattis, lectus id, aliquet rhoncus  sem. Phasellus eget ex in dolor
feugiat ultricies. Etiam interdum sit amet sit  nisl in placerat.  Sed vitae enim
vulputate, tempus leo commodo, accumsan nulla.`
	lessModifiedLorem = `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nulla
varius enim mattis, rhoncus lectus id, aliquet. Phasellus eget ex in dolor
feugiat ultricies. Etiam interdum sit amet nisl in placerat.  Sed vitae enim
vulputate, tempus leo commodo, accumsan nulla.`

	postmodernThesisCollapse = `1. Expressions of collapse

If one examines postcultural Marxism, one is faced with a choice: either
reject capitalist submodern theory or conclude that the purpose of the reader
is significant form. Bataille uses the term ‘capitalist construction’ to denote
not, in fact, discourse, but prediscourse.

Therefore, in Stardust, Gaiman analyses postcultural Marxism; in
The Books of Magic, although, he denies capitalist submodern theory. If
capitalist construction holds, we have to choose between capitalist submodern
theory and Baudrillardist simulacra.

However, conceptualist socialism implies that narrativity may be used to
oppress the proletariat, given that sexuality is distinct from art. The subject
is interpolated into a capitalist construction that includes language as a
paradox.
`
	postmodernThesisNarratives = `1. Narratives of failure

The main theme of the works of Joyce is the defining characteristic, and some
would say the economy, of neocultural class. But Bataille promotes the use of
socialist realism to deconstruct sexual identity.

The subject is interpolated into a Baudrillardist simulation that includes
consciousness as a whole. Thus, the primary theme of Pickett's[1] model of
socialist realism is the role of the reader as artist.

The subject is contextualised into a postcapitalist discourse that includes
language as a paradox. It could be said that if Baudrillardist simulation
holds, the works of Gibson are postmodern. The characteristic theme of the
works of Gibson is the common ground between society and narrativity. However,
Sartre uses the term 'postcapitalist discourse' to denote not, in fact,
narrative, but postnarrative.
`
	postmodernThesisFatalFlaw = `1. Contexts of fatal flaw

"Narrativity is part of the dialectic of culture," says Marx; however,
according to Hamburger[1] , it is not so much narrativity that is part of the
dialectic of culture, but rather the stasis, and hence the defining
characteristic, of narrativity. Bataille promotes the use of Batailleist
'powerful communication' to modify society.

If one examines the presemioticist paradigm of reality, one is faced with a
choice: either reject Batailleist 'powerful communication' or conclude that
concensus must come from the masses. Therefore, Baudrillard uses the term 'the
presemioticist paradigm of reality' to denote the difference between class and
society. The subject is interpolated into a subtextual capitalist theory that
includes consciousness as a whole.

However, Pickett[2] implies that we have to choose between neotextual feminism
and dialectic appropriation. Debord suggests the use of subtextual capitalist
theory to deconstruct the status quo.
`
	nullifiable = `[[ , _ , _ , _
? _ : _
? _ : _
? _ : _
]}
`
)

func TestClassify_NearestMatch(t *testing.T) {
	c := New(DefaultConfidenceThreshold, FlattenWhitespace)
	c.AddValue("gettysburg", gettysburg)
	c.AddValue("declaration", declaration)
	c.AddValue("loremipsum", loremipsum)

	tests := []struct {
		description string
		input       string  // input string to match
		name        string  // name of expected nearest match
		minConf     float64 // the lowest confidence accepted for the match
		maxConf     float64 // the highest confidence we expect for this match
	}{
		{
			description: "Full Declaration",
			input:       declaration,
			name:        "declaration",
			minConf:     1.0,
			maxConf:     1.0,
		},
		{
			description: "Modified Lorem",
			input:       modifiedLorem,
			name:        "loremipsum",
			minConf:     0.90,
			maxConf:     0.91,
		},
		{
			description: "Modified Gettysburg",
			input:       modifiedGettysburg,
			name:        "gettysburg",
			minConf:     0.86,
			maxConf:     0.87,
		},
	}

	for _, tt := range tests {
		m := c.NearestMatch(tt.input)

		if got, want := m.Name, tt.name; got != want {
			t.Errorf("NearestMatch(%q) = %q, want %q", tt.description, got, want)
		}
		if got, want := m.Confidence, tt.minConf; got < want {
			t.Errorf("NearestMatch(%q) returned confidence %v, want minimum of %v", tt.description, got, want)
		}
		if got, want := m.Confidence, tt.maxConf; got > want {
			t.Errorf("NearestMatch(%q) = %v, want maxiumum of %v", tt.description, got, want)
		}
	}
}

type result struct {
	key    string // key of expected nearest match
	offset int    // offset of match in unknown string

	// The confidence values are retrieved by simply running the classifier
	// and noting the output. A value greater than the "max" is fine and
	// the tests can be adjusted to account for it. A value less than "min"
	// should be carefully scrutinzed before adjusting the tests.
	minConf float64 // the lowest confidence accepted for the match
	maxConf float64 // the highest confidence we expect for this match
}

func TestClassify_MultipleMatch(t *testing.T) {
	c := New(DefaultConfidenceThreshold, FlattenWhitespace)
	c.AddValue("gettysburg", gettysburg)
	c.AddValue("declaration", declaration)
	c.AddValue("declaration-close", declaration[:len(declaration)/2-1]+"_"+declaration[len(declaration)/2:])
	c.AddValue("loremipsum", loremipsum)

	tests := []struct {
		description string
		input       string // input string to match
		want        []result
	}{
		{
			description: "Exact text match",
			input:       postmodernThesisNarratives + declaration + postmodernThesisCollapse,
			want: []result{
				{
					key:     "declaration",
					offset:  842,
					minConf: 1.0,
					maxConf: 1.0,
				},
			},
		},
		{
			description: "Partial text match",
			input:       postmodernThesisNarratives + modifiedLorem + postmodernThesisCollapse,
			want: []result{
				{
					key:     "loremipsum",
					offset:  842,
					minConf: 0.90,
					maxConf: 0.91,
				},
			},
		},
		{
			description: "Two partial matches",
			input:       postmodernThesisNarratives + modifiedLorem + postmodernThesisCollapse + modifiedGettysburg + postmodernThesisFatalFlaw,
			want: []result{
				{
					key:     "loremipsum",
					offset:  842,
					minConf: 0.90,
					maxConf: 0.91,
				},
				{
					key:     "gettysburg",
					offset:  1900,
					minConf: 0.86,
					maxConf: 0.87,
				},
			},
		},
		{
			description: "Partial matches of similar text",
			input:       postmodernThesisNarratives + modifiedLorem + postmodernThesisCollapse + lessModifiedLorem + postmodernThesisFatalFlaw,
			want: []result{
				{
					key:     "loremipsum",
					offset:  1900,
					minConf: 0.98,
					maxConf: 0.99,
				},
				{
					key:     "loremipsum",
					offset:  842,
					minConf: 0.90,
					maxConf: 0.91,
				},
			},
		},
		{
			description: "Nullifiable text",
			input:       nullifiable,
			want:        nil,
		},
		{
			description: "No match",
			input:       postmodernThesisNarratives + postmodernThesisCollapse,
			want:        nil,
		},
	}

	for _, tt := range tests {
		matches := c.MultipleMatch(tt.input)
		if len(matches) != len(tt.want) {
			t.Errorf("MultipleMatch(%q) not enough matches = %v, want %v", tt.description, len(matches), len(tt.want))
		}

		for i := 0; i < len(matches); i++ {
			m := matches[i]
			w := tt.want[i]
			if got, want := m.Name, w.key; got != want {
				t.Errorf("MultipleMatch(%q) = %q, want %q", tt.description, got, want)
			}
			if got, want := m.Confidence, w.minConf; got < want {
				t.Errorf("MultipleMatch(%q) %q = %v, want minimum of %v", tt.description, w.key, got, want)
			}
			if got, want := m.Confidence, w.maxConf; got > want {
				t.Errorf("MultipleMatch(%q) %q = %v, want maximum of %v", tt.description, w.key, got, want)
			}
			if got, want := m.Offset, w.offset; got != want {
				t.Errorf("MultipleMatch(%q) %q = %v, want offset of %v", tt.description, w.key, got, want)
			}
		}
	}
}

func TestClassify_DiffRatio(t *testing.T) {
	tests := []struct {
		x, y string
		want float64
	}{
		{"", "", 1.0},
		{"a", "b", 1.0},
		{"", "abc", 0},
		{"ab", "c", 0.5},
		{"a", "bc", 0.5},
		{"a", "bcde", 0.25},
	}

	for _, tt := range tests {
		if got, want := diffRatio(tt.x, tt.y), tt.want; got != want {
			t.Errorf("diffRatio(%q, %q) = %f, want %f", tt.x, tt.y, got, want)
		}
	}
}

func TestClassify_Matches(t *testing.T) {
	tests := []struct {
		description string
		matches     Matches
		want        Matches
	}{
		{
			description: "Different names, same confidences, same offset",
			matches: Matches{
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     0,
				},
				&Match{
					Name:       "a",
					Confidence: 0.42,
					Offset:     0,
				},
			},
			want: Matches{
				&Match{
					Name:       "a",
					Confidence: 0.42,
					Offset:     0,
				},
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     0,
				},
			},
		},
		{
			description: "Same names, different confidences, same offset",
			matches: Matches{
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     0,
				},
				&Match{
					Name:       "b",
					Confidence: 0.90,
					Offset:     0,
				},
			},
			want: Matches{
				&Match{
					Name:       "b",
					Confidence: 0.90,
					Offset:     0,
				},
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     0,
				},
			},
		},
		{
			description: "Same names, same confidences, different offsets",
			matches: Matches{
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     42,
				},
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     0,
				},
			},
			want: Matches{
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     0,
				},
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     42,
				},
			},
		},

		{
			description: "Different names, different confidences, same offset",
			matches: Matches{
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     0,
				},
				&Match{
					Name:       "a",
					Confidence: 0.90,
					Offset:     0,
				},
			},
			want: Matches{
				&Match{
					Name:       "a",
					Confidence: 0.90,
					Offset:     0,
				},
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     0,
				},
			},
		},
		{
			description: "Different names, same confidences, different offset",
			matches: Matches{
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     37,
				},
				&Match{
					Name:       "a",
					Confidence: 0.42,
					Offset:     0,
				},
			},
			want: Matches{
				&Match{
					Name:       "a",
					Confidence: 0.42,
					Offset:     0,
				},
				&Match{
					Name:       "b",
					Confidence: 0.42,
					Offset:     37,
				},
			},
		},
		{
			description: "Different names, different confidences, different offset",
			matches: Matches{
				&Match{
					Name:       "a",
					Confidence: 0.42,
					Offset:     0,
				},
				&Match{
					Name:       "b",
					Confidence: 0.90,
					Offset:     37,
				},
			},
			want: Matches{
				&Match{
					Name:       "b",
					Confidence: 0.90,
					Offset:     37,
				},
				&Match{
					Name:       "a",
					Confidence: 0.42,
					Offset:     0,
				},
			},
		},
	}

	for _, tt := range tests {
		sort.Sort(tt.matches)
		if !reflect.DeepEqual(tt.matches, tt.want) {
			for _, x := range tt.matches {
				t.Errorf("got: %v", x)
			}
			for _, x := range tt.want {
				t.Errorf("want: %v", x)
			}
			t.Errorf("MatchesSort(%q) = %v, want %v", tt.description, tt.matches, tt.want)
		}
	}
}

func TestClassify_DiffRangeEnd(t *testing.T) {
	dmp := diffmatchpatch.New()
	tests := []struct {
		description string
		unknown     string
		known       string
		end         int
	}{
		{
			description: "identical",
			unknown:     declaration,
			known:       declaration,
			end:         1,
		},
		{
			description: "lorem",
			unknown:     lessModifiedLorem,
			known:       loremipsum,
			end:         3,
		},
		{
			description: "gettysburg",
			unknown:     modifiedGettysburg,
			known:       gettysburg,
			end:         19,
		},
	}

	for _, tt := range tests {
		diffs := dmp.DiffMain(tt.unknown, tt.known, true)
		if e := diffRangeEnd(tt.known, diffs); e != tt.end {
			t.Errorf("DiffRangeEnd(%q) = end %v, want %v", tt.description, e, tt.end)
		}
	}
}

func BenchmarkClassifier(b *testing.B) {
	c := New(DefaultConfidenceThreshold, FlattenWhitespace)
	c.AddValue("gettysburg", gettysburg)
	c.AddValue("declaration", declaration)
	c.AddValue("loremipsum", loremipsum)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.NearestMatch(modifiedLorem)
	}
}
