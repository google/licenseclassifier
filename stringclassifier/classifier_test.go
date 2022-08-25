// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Select test data comes from
// The Project Gutenberg eBook of The humour of Ireland, by D. J., (David James), (1866-1917) O'Donoghue

package stringclassifier

import (
	"reflect"
	"regexp"
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
	gettysburgExtraWord = `Four score and seven years ago our fathers brought forth
on this continent, a new nation, conceived in Liberty, and dedicated to the
proposition that all men are created equal.Foobar`

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
	humourOfIreland = `As a rule, Irish poets have not extracted a pessimistic
philosophy from liquor; they are “elevated,” not depressed, and do not deem
it essential to the production of a poem that its author should be a cynic or
an evil prophet. One of the best attributes of Irish poetry is its constant
expression of the natural emotions. Previous to the close of the
seventeenth[xvi] century, it is said, drunkenness was not suggested by the
poets as common in Ireland—the popularity of Bacchanalian songs since that
date seems to prove that the vice soon became a virtue. Maginn is the
noisiest of modern revellers, and easily roars the others down.
`
	fellowInTheGoatSkin = `There was a poor widow living down there near the Iron
Forge when the country was all covered with forests, and you might walk on
the tops of trees from Carnew to the Lady’s Island, and she had one boy. She
was very poor, as I said before, and was not able to buy clothes for her son.
So when she was going out she fixed him snug and combustible in the ash-pit,
and piled the warm ashes about him. The boy knew no better, and was as happy
as the day was long; and he was happier still when a neighbour[10] gave his
mother a kid to keep him company when herself was abroad. The kid and the lad
played like two may-boys; and when she was old enough to give milk, wasn’t it
a godsend to the little family? You won’t prevent the boy from growing up
into a young man, but not a screed of clothes had he then no more than when
he was a gorsoon.
`
	oldCrowYoungCrow = `There was an old crow teaching a young crow one day, and
he said to him, “Now, my son,” says he, “listen to the advice I’m going to
give you. If you see a person coming near you and stooping, mind yourself,
and be on your keeping; he’s stooping for a stone to throw at you.”

“But tell me,” says the young crow, “what should I do if he had a stone
already down in his pocket?”

“Musha, go ’long out of that,” says the old crow, “you’ve learned enough; the
devil another learning I’m able to give you.”
`
	nullifiable = `[[ , _ , _ , _
? _ : _
? _ : _
? _ : _
]
}
`
	nonWords = regexp.MustCompile("[[:punct:]]+")
)

// removeNonWords removes non-words from the string, replacing them with empty
// string. (This is meant to exercise tokenization problems.)
func removeNonWords(s string) string {
	return nonWords.ReplaceAllString(s, "")
}

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

	cNormalize := New(DefaultConfidenceThreshold, FlattenWhitespace, removeNonWords)
	cNormalize.AddValue("gettysburg", gettysburg)

	tests := []struct {
		description string
		c           *Classifier
		input       string // input string to match
		want        []result
	}{
		{
			description: "Exact text match",
			c:           c,
			input:       fellowInTheGoatSkin + declaration + humourOfIreland,
			want: []result{
				{
					key:     "declaration",
					offset:  845,
					minConf: 1.0,
					maxConf: 1.0,
				},
			},
		},
		{
			description: "Partial text match",
			c:           c,
			input:       fellowInTheGoatSkin + modifiedLorem + humourOfIreland,
			want: []result{
				{
					key:     "loremipsum",
					offset:  845,
					minConf: 0.90,
					maxConf: 0.91,
				},
			},
		},
		{
			description: "Two partial matches",
			c:           c,
			input:       fellowInTheGoatSkin + modifiedLorem + humourOfIreland + modifiedGettysburg + oldCrowYoungCrow,
			want: []result{
				{
					key:     "loremipsum",
					offset:  845,
					minConf: 0.90,
					maxConf: 0.91,
				},
				{
					key:     "gettysburg",
					offset:  1750,
					minConf: 0.86,
					maxConf: 0.87,
				},
			},
		},
		{
			description: "Partial matches of similar text",
			c:           c,
			input:       fellowInTheGoatSkin + modifiedLorem + humourOfIreland + lessModifiedLorem + oldCrowYoungCrow,
			want: []result{
				{
					key:     "loremipsum",
					offset:  1750,
					minConf: 0.98,
					maxConf: 0.99,
				},
				{
					key:     "loremipsum",
					offset:  845,
					minConf: 0.90,
					maxConf: 0.91,
				},
			},
		},
		{
			description: "Nullifiable text",
			c:           c,
			input:       nullifiable,
			want:        nil,
		},
		{
			description: "No match",
			c:           c,
			input:       fellowInTheGoatSkin + humourOfIreland,
			want:        nil,
		},
		{
			description: "Exact text match, with extra word and non-word normalizer",
			c:           cNormalize,
			input:       fellowInTheGoatSkin + gettysburgExtraWord + humourOfIreland,
			want: []result{
				{
					key:     "gettysburg",
					offset:  825,
					minConf: 1.0,
					maxConf: 1.0,
				},
			},
		},
	}

	for _, tt := range tests {
		matches := tt.c.MultipleMatch(tt.input)
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
