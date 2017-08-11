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
package searchset

import (
	"reflect"
	"testing"

	"github.com/google/licenseclassifier/stringclassifier/searchset/tokenizer"
)

const (
	shortPostmodernThesis = "In the works of Joyce, a predominant concept is the concept of semioticist"
	postmodernThesis      = `1. Joyce and neotextual modernist theory

In the works of Joyce, a predominant concept is the concept of semioticist
culture. The without/within distinction intrinsic to Finnegan's Wake emerges
again in Ulysses, although in a more neomodern sense.
`
)

func TestSearchSet_New(t *testing.T) {
	tests := []struct {
		description string
		text        string
		granularity int
		want        *SearchSet
	}{
		{
			description: "Empty string",
			text:        "",
			granularity: DefaultGranularity,
			want: &SearchSet{
				Tokens:         nil,
				Hashes:         make(tokenizer.Hash),
				Checksums:      nil,
				ChecksumRanges: nil,
			},
		},
		{
			description: "Small string",
			text:        "Hello world",
			granularity: 4,
			want: &SearchSet{
				Tokens: tokenizer.Tokens{
					{Text: "Hello", Offset: 0},
					{Text: "world", Offset: 6},
				},
				Hashes:         tokenizer.Hash{2346098258: tokenizer.TokenRanges{{Start: 0, End: 2}}},
				Checksums:      []uint32{2346098258},
				ChecksumRanges: tokenizer.TokenRanges{{Start: 0, End: 2}},
				nodes:          []*node{{2346098258, &tokenizer.TokenRange{Start: 0, End: 2}}},
			},
		},
	}

	for _, tt := range tests {
		if got := New(tt.text, tt.granularity); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("New(%q) = %+v, want %+v", tt.description, got, tt.want)
		}
	}
}

func TestSearchSet_NodeConstruction(t *testing.T) {
	s := New(shortPostmodernThesis, DefaultGranularity)
	want := []string{
		"[0:3]", "[1:4]", "[2:5]", "[3:6]", "[4:7]", "[5:8]", "[6:9]",
		"[7:10]", "[8:11]", "[9:12]", "[10:13]", "[11:14]",
	}

	if len(s.nodes) != len(want) {
		t.Errorf("Number of nodes %v, want %v", len(s.nodes), len(want))
		return
	}

	for i := 0; i < len(s.nodes); i++ {
		if got := s.nodes[i].String(); got != want[i] {
			t.Errorf("Nodes = got:\n%s\nwant:\n%s", got, want[i])
		}
	}
}

func TestSearchSet_CoalesceTokenRanges(t *testing.T) {
	tests := []struct {
		description string
		mr          MatchRanges
		want        MatchRanges
	}{
		{
			description: "Non-overlapping Ranges",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 27, TargetStart: 0, TargetEnd: 27},
				{SrcStart: 37, SrcEnd: 927, TargetStart: 37, TargetEnd: 927},
			},
			want: MatchRanges{
				{SrcStart: 0, SrcEnd: 27, TargetStart: 0, TargetEnd: 27},
				{SrcStart: 37, SrcEnd: 927, TargetStart: 37, TargetEnd: 927},
			},
		},
		{
			description: "Identical Ranges",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 37, TargetStart: 0, TargetEnd: 37},
				{SrcStart: 0, SrcEnd: 37, TargetStart: 0, TargetEnd: 37},
			},
			want: MatchRanges{{SrcStart: 0, SrcEnd: 37, TargetStart: 0, TargetEnd: 37}},
		},
		{
			description: "Sequential Ranges",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 37, TargetStart: 0, TargetEnd: 37},
				{SrcStart: 37, SrcEnd: 927, TargetStart: 37, TargetEnd: 927},
			},
			want: MatchRanges{{SrcStart: 0, SrcEnd: 927, TargetStart: 0, TargetEnd: 927}},
		},
		{
			description: "Overlapping Ranges - Same Start",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 37, TargetStart: 0, TargetEnd: 37},
				{SrcStart: 0, SrcEnd: 927, TargetStart: 0, TargetEnd: 927},
			},
			want: MatchRanges{{SrcStart: 0, SrcEnd: 927, TargetStart: 0, TargetEnd: 927}},
		},
		{
			description: "Overlapping Ranges - Different Start",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 37, TargetStart: 0, TargetEnd: 37},
				{SrcStart: 27, SrcEnd: 927, TargetStart: 27, TargetEnd: 927},
			},
			want: MatchRanges{{SrcStart: 0, SrcEnd: 927, TargetStart: 0, TargetEnd: 927}},
		},
		{
			description: "Overlapping Ranges - Same End",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 37, TargetStart: 0, TargetEnd: 37},
				{SrcStart: 27, SrcEnd: 37, TargetStart: 27, TargetEnd: 37},
			},
			want: MatchRanges{{SrcStart: 0, SrcEnd: 37, TargetStart: 0, TargetEnd: 37}},
		},
		{
			description: "Completely Overlapping Ranges",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 42, TargetStart: 0, TargetEnd: 42},
				{SrcStart: 27, SrcEnd: 37, TargetStart: 27, TargetEnd: 37},
			},
			want: MatchRanges{{SrcStart: 0, SrcEnd: 42, TargetStart: 0, TargetEnd: 42}},
		},
	}

	for _, tt := range tests {
		got := coalesceMatchRanges(tt.mr)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("CoalesceTokenRanges(%q) = %+v, want %+v", tt.description, got, tt.want)
		}
	}
}

func TestSearchSet_FindPotentialMatches(t *testing.T) {
	known := New(postmodernThesis, DefaultGranularity)

	size := len(postmodernThesis)
	modified := "hello world "
	modified += postmodernThesis[:size/3] + " hello world "
	modified += postmodernThesis[size/3 : 2*size/3-4]
	modified += postmodernThesis[2*size/3+7:]
	unknown := New(modified, DefaultGranularity)

	want := []MatchRanges{{
		{SrcStart: 0, SrcEnd: 15, TargetStart: 2, TargetEnd: 17},
		{SrcStart: 16, SrcEnd: 28, TargetStart: 21, TargetEnd: 33},
		{SrcStart: 31, SrcEnd: 46, TargetStart: 34, TargetEnd: 49},
	}}

	got := FindPotentialMatches(known, unknown)
	if len(got) != len(want) {
		t.Errorf("Number of matches %v, want %v", len(got), len(want))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Offsets = %+v, want %+v", got, want)
	}

	known = New(`again in Ulysses, although in a more neomodern sense.
culture. The without/within distinction intrinsic to Finnegan's Wake emerges
`, DefaultGranularity)

	want = []MatchRanges{
		{
			{SrcStart: 11, SrcEnd: 18, TargetStart: 26, TargetEnd: 33},
			{SrcStart: 21, SrcEnd: 25, TargetStart: 34, TargetEnd: 38},
		},
		{{SrcStart: 0, SrcEnd: 11, TargetStart: 38, TargetEnd: 49}},
	}

	got = FindPotentialMatches(known, unknown)
	if len(got) != len(want) {
		t.Errorf("Number of matches %v, want %v", len(got), len(want))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Offsets = %+v, want %+v", got, want)
	}
}

func TestSearchSet_GetMatchedRanges(t *testing.T) {
	const (
		source = "a b c d e f g c d e h i j"
		target = "a b c _ _ c d e _ f g h _ c d  e _ h i j"
	)

	src := New(source, DefaultGranularity)
	tar := New(target, DefaultGranularity)

	want := []MatchRanges{
		{
			{SrcStart: 0, SrcEnd: 3, TargetStart: 0, TargetEnd: 3},
			{SrcStart: 2, SrcEnd: 5, TargetStart: 5, TargetEnd: 8},
			{SrcStart: 7, SrcEnd: 10, TargetStart: 13, TargetEnd: 16},
			{SrcStart: 10, SrcEnd: 13, TargetStart: 17, TargetEnd: 20},
		},
	}

	got := getMatchedRanges(src, tar)
	if len(got) != len(want) {
		t.Errorf("Number of matches %v, want %v", len(got), len(want))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Match Ranges = %+v, want %+v", got, want)
	}
}

func TestSearchSet_TargetMatchedRanges(t *testing.T) {
	const (
		source = "a b c d e f g c d e h i j"
		target = "a b c d e _ _ c d e _ f g h _ c d  e _ h i j"
	)

	src := New(source, DefaultGranularity)
	tar := New(target, DefaultGranularity)

	want := MatchRanges{
		{SrcStart: 0, SrcEnd: 3, TargetStart: 0, TargetEnd: 3},
		{SrcStart: 1, SrcEnd: 4, TargetStart: 1, TargetEnd: 4},
		{SrcStart: 2, SrcEnd: 5, TargetStart: 2, TargetEnd: 5},
		{SrcStart: 7, SrcEnd: 10, TargetStart: 2, TargetEnd: 5},
		{SrcStart: 2, SrcEnd: 5, TargetStart: 7, TargetEnd: 10},
		{SrcStart: 7, SrcEnd: 10, TargetStart: 7, TargetEnd: 10},
		{SrcStart: 2, SrcEnd: 5, TargetStart: 15, TargetEnd: 18},
		{SrcStart: 7, SrcEnd: 10, TargetStart: 15, TargetEnd: 18},
		{SrcStart: 10, SrcEnd: 13, TargetStart: 19, TargetEnd: 22},
	}

	got := targetMatchedRanges(src, tar)
	if len(got) != len(want) {
		t.Errorf("Number of matches %v, want %v", len(got), len(want))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Match Range = %+v, want %+v", got, want)
	}
}

func TestSearchSet_UntangleSourceRanges(t *testing.T) {
	tests := []struct {
		description string
		mr          MatchRanges
		want        MatchRanges
	}{
		{
			description: "Single Match - In Order",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 3, TargetStart: 10, TargetEnd: 13},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 14, TargetEnd: 19},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 15, TargetEnd: 20},
			},
			want: MatchRanges{
				{SrcStart: 0, SrcEnd: 3, TargetStart: 10, TargetEnd: 13},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 14, TargetEnd: 19},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 15, TargetEnd: 20},
			},
		},
		{
			description: "Single Match - Out of Order",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 3, TargetStart: 10, TargetEnd: 13},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 14, TargetEnd: 19},
				{SrcStart: 15, SrcEnd: 20, TargetStart: 14, TargetEnd: 19},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 15, TargetEnd: 20},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 24, TargetEnd: 19},
				{SrcStart: 15, SrcEnd: 20, TargetStart: 24, TargetEnd: 19},
				{SrcStart: 23, SrcEnd: 29, TargetStart: 30, TargetEnd: 37},
			},
			want: MatchRanges{
				{SrcStart: 0, SrcEnd: 3, TargetStart: 10, TargetEnd: 13},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 14, TargetEnd: 19},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 15, TargetEnd: 20},
				{SrcStart: 15, SrcEnd: 20, TargetStart: 24, TargetEnd: 19},
				{SrcStart: 23, SrcEnd: 29, TargetStart: 30, TargetEnd: 37},
			},
		},
		{
			description: "Multiple Match - In Order",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 3, TargetStart: 10, TargetEnd: 13},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 14, TargetEnd: 19},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 15, TargetEnd: 20},
				{SrcStart: 0, SrcEnd: 3, TargetStart: 110, TargetEnd: 113},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 114, TargetEnd: 119},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 115, TargetEnd: 120},
			},
			want: MatchRanges{
				{SrcStart: 0, SrcEnd: 3, TargetStart: 10, TargetEnd: 13},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 14, TargetEnd: 19},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 15, TargetEnd: 20},
				{SrcStart: 0, SrcEnd: 3, TargetStart: 110, TargetEnd: 113},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 114, TargetEnd: 119},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 115, TargetEnd: 120},
			},
		},
		{
			description: "Multiple Match - Out of Order",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 3, TargetStart: 10, TargetEnd: 13},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 14, TargetEnd: 19},
				{SrcStart: 15, SrcEnd: 20, TargetStart: 14, TargetEnd: 19},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 15, TargetEnd: 20},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 24, TargetEnd: 19},
				{SrcStart: 15, SrcEnd: 20, TargetStart: 24, TargetEnd: 19},
				{SrcStart: 23, SrcEnd: 29, TargetStart: 30, TargetEnd: 37},
				{SrcStart: 0, SrcEnd: 3, TargetStart: 110, TargetEnd: 113},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 114, TargetEnd: 119},
				{SrcStart: 15, SrcEnd: 20, TargetStart: 114, TargetEnd: 119},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 115, TargetEnd: 120},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 124, TargetEnd: 119},
				{SrcStart: 15, SrcEnd: 20, TargetStart: 124, TargetEnd: 119},
				{SrcStart: 23, SrcEnd: 29, TargetStart: 130, TargetEnd: 137},
			},
			want: MatchRanges{
				{SrcStart: 0, SrcEnd: 3, TargetStart: 10, TargetEnd: 13},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 14, TargetEnd: 19},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 15, TargetEnd: 20},
				{SrcStart: 15, SrcEnd: 20, TargetStart: 24, TargetEnd: 19},
				{SrcStart: 23, SrcEnd: 29, TargetStart: 30, TargetEnd: 37},
				{SrcStart: 0, SrcEnd: 3, TargetStart: 110, TargetEnd: 113},
				{SrcStart: 5, SrcEnd: 10, TargetStart: 114, TargetEnd: 119},
				{SrcStart: 6, SrcEnd: 11, TargetStart: 115, TargetEnd: 120},
				{SrcStart: 15, SrcEnd: 20, TargetStart: 124, TargetEnd: 119},
				{SrcStart: 23, SrcEnd: 29, TargetStart: 130, TargetEnd: 137},
			},
		},
	}

	for _, tt := range tests {
		got := untangleSourceRanges(tt.mr)
		if len(got) != len(tt.want) {
			t.Errorf("Number of matches %v, want %v", len(got), len(tt.want))
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("UntangleSourceRanges(%q) = %v, want %v", tt.description, got, tt.want)
		}
	}
}

func TestSearchSet_SplitRanges(t *testing.T) {
	tests := []struct {
		description string
		mr          MatchRanges
		want        []MatchRanges
	}{
		{
			description: "Single Match Range",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
				{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
			},
			want: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
			},
		},
		{
			description: "Two Match Ranges",
			mr: MatchRanges{
				{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
				{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				{SrcStart: 3, SrcEnd: 10, TargetStart: 108, TargetEnd: 115},
				{SrcStart: 23, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
			},
			want: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
				{
					{SrcStart: 3, SrcEnd: 10, TargetStart: 108, TargetEnd: 115},
					{SrcStart: 23, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
			},
		},
	}

	for _, tt := range tests {
		got := splitRanges(tt.mr)
		if len(got) != len(tt.want) {
			t.Errorf("Number of matches %v, want %v", len(got), len(tt.want))
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("SplitRanges(%q) = %v, want %v", tt.description, got, tt.want)
		}
	}
}

func TestSearchSet_MergeConsecutiveRanges(t *testing.T) {
	tests := []struct {
		description string
		mr          []MatchRanges
		want        []MatchRanges
	}{
		{
			description: "No Overlap",
			mr: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
				{
					{SrcStart: 3, SrcEnd: 10, TargetStart: 108, TargetEnd: 115},
					{SrcStart: 23, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
			},
			want: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
				{
					{SrcStart: 3, SrcEnd: 10, TargetStart: 108, TargetEnd: 115},
					{SrcStart: 23, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
			},
		},
		{
			description: "Consecutive Ranges No Overlap",
			mr: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
				{
					{SrcStart: 3, SrcEnd: 10, TargetStart: 35, TargetEnd: 41},
					{SrcStart: 23, SrcEnd: 30, TargetStart: 125, TargetEnd: 135},
				},
			},
			want: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
				{
					{SrcStart: 3, SrcEnd: 10, TargetStart: 35, TargetEnd: 41},
					{SrcStart: 23, SrcEnd: 30, TargetStart: 125, TargetEnd: 135},
				},
			},
		},
		{
			description: "Consecutive Ranges with First Element Overlap",
			mr: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
				{
					{SrcStart: 3, SrcEnd: 10, TargetStart: 34, TargetEnd: 41},
					{SrcStart: 33, SrcEnd: 40, TargetStart: 35, TargetEnd: 42},
				},
			},
			want: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 36, TargetStart: 25, TargetEnd: 41},
					{SrcStart: 33, SrcEnd: 40, TargetStart: 35, TargetEnd: 42},
				},
			},
		},
		{
			description: "Consecutive Ranges with Overlap",
			mr: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
				},
				{
					{SrcStart: 3, SrcEnd: 10, TargetStart: 34, TargetEnd: 41},
					{SrcStart: 33, SrcEnd: 40, TargetStart: 45, TargetEnd: 52},
				},
			},
			want: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 36, TargetStart: 25, TargetEnd: 41},
					{SrcStart: 33, SrcEnd: 40, TargetStart: 45, TargetEnd: 52},
				},
			},
		},
		{
			description: "Consecutive Ranges with Previous Deep Overlap",
			mr: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
					{SrcStart: 120, SrcEnd: 130, TargetStart: 37, TargetEnd: 47},
					{SrcStart: 122, SrcEnd: 132, TargetStart: 39, TargetEnd: 49},
				},
				{
					{SrcStart: 3, SrcEnd: 10, TargetStart: 34, TargetEnd: 41},
					{SrcStart: 33, SrcEnd: 40, TargetStart: 45, TargetEnd: 52},
				},
			},
			want: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 36, TargetStart: 25, TargetEnd: 41},
					{SrcStart: 33, SrcEnd: 40, TargetStart: 45, TargetEnd: 52},
				},
			},
		},
		{
			description: "Consecutive Ranges with Deep Overlap",
			mr: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
					{SrcStart: 24, SrcEnd: 34, TargetStart: 29, TargetEnd: 39},
					{SrcStart: 120, SrcEnd: 130, TargetStart: 37, TargetEnd: 47},
					{SrcStart: 122, SrcEnd: 132, TargetStart: 39, TargetEnd: 49},
				},
				{
					{SrcStart: 3, SrcEnd: 10, TargetStart: 26, TargetEnd: 33},
					{SrcStart: 5, SrcEnd: 12, TargetStart: 28, TargetEnd: 35},
					{SrcStart: 25, SrcEnd: 35, TargetStart: 31, TargetEnd: 41},
				},
			},
			want: []MatchRanges{
				{
					{SrcStart: 0, SrcEnd: 10, TargetStart: 5, TargetEnd: 15},
					{SrcStart: 20, SrcEnd: 30, TargetStart: 25, TargetEnd: 35},
					{SrcStart: 24, SrcEnd: 34, TargetStart: 29, TargetEnd: 39},
					{SrcStart: 25, SrcEnd: 35, TargetStart: 31, TargetEnd: 41},
				},
			},
		},
	}

	for _, tt := range tests {
		got := mergeConsecutiveRanges(tt.mr)
		if len(got) != len(tt.want) {
			t.Errorf("Number of matches %v, want %v", len(got), len(tt.want))
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("SplitRanges(%q) = %v, want %v", tt.description, got, tt.want)
		}
	}
}
