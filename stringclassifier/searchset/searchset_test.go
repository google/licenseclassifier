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
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	shortPostmodernThesis = "In the works of Joyce, a predominant concept is the concept of semioticist"
	postmodernThesis      = `1. Joyce and neotextual modernist theory

In the works of Joyce, a predominant concept is the concept of semioticist
culture. The without/within distinction intrinsic to Finnegan's Wake emerges
again in Ulysses, although in a more neomodern sense.
`
)

var parent = map[string]map[string]present{
	"[0:14]":  {"[0:7]": present{}, "[3:10]": present{}, "[6:13]": present{}},
	"[0:7]":   {"[0:4]": present{}, "[2:6]": present{}},
	"[3:10]":  {"[4:8]": present{}, "[6:10]": present{}},
	"[6:13]":  {"[6:10]": present{}, "[8:12]": present{}},
	"[0:4]":   {"[0:3]": present{}, "[1:4]": present{}},
	"[2:6]":   {"[2:5]": present{}, "[3:6]": present{}},
	"[4:8]":   {"[4:7]": present{}, "[5:8]": present{}},
	"[6:10]":  {"[6:9]": present{}, "[7:10]": present{}},
	"[8:12]":  {"[8:11]": present{}, "[9:12]": present{}},
	"[10:14]": {"[10:13]": present{}, "[11:14]": present{}},
	"[0:3]":   {},
	"[1:4]":   {},
	"[2:5]":   {},
	"[3:6]":   {},
	"[4:7]":   {},
	"[5:8]":   {},
	"[6:9]":   {},
	"[7:10]":  {},
	"[8:11]":  {},
	"[9:12]":  {},
	"[10:13]": {},
	"[11:14]": {},
}

func TestSearchSet_LatticeConstruction(t *testing.T) {
	s := New(shortPostmodernThesis, DefaultGranularity)
	want := "[0:14] :: " +
		"[0:7] -> [3:10] -> [6:13] :: " +
		"[0:4] -> [2:6] -> [4:8] -> [6:10] -> [8:12] -> [10:14] :: " +
		"[0:3] -> [1:4] -> [2:5] -> [3:6] -> [4:7] -> [5:8] -> [6:9] -> [7:10] -> [8:11] -> [9:12] -> [10:13] -> [11:14]"
	if got := s.lattice.String(); got != want {
		t.Errorf("Lattice = got:\n%s\nwant:\n%s", got, want)
	}

	for n := s.lattice.root; n != nil; n = n.sibling {
		pnode := n.String()
		if _, ok := parent[pnode]; !ok {
			t.Errorf("LatticeNode(%q) = cannot find in parent list", pnode)
			continue
		}
		if got, want := len(n.children), len(parent[pnode]); got != want {
			t.Errorf("LatticeNode(%q) = %d number of children, want %d", pnode, got, want)
			continue
		}
		for child := range n.children {
			cnode := child.String()
			if _, ok := parent[pnode][cnode]; !ok {
				t.Errorf("LatticeNode(%q) = cannot find child node %q", pnode, cnode)
			}
		}
	}
}

func TestSearchSet_MarkChildrenVisited(t *testing.T) {
	s := New(shortPostmodernThesis, DefaultGranularity)
	visited := make(map[*node]present)
	s.lattice.root.markChildrenVisited(visited)
	for n := s.lattice.root.sibling; n != nil; n = n.sibling {
		r := n.String()
		// Not all nodes will be visited. Only those that are children
		// to the parent nodes.
		if _, ok := visited[n]; !ok && r != "[10:14]" && r != "[10:13]" && r != "[11:14]" {
			t.Errorf("MarkChildrenVisited = node [%d:%d] not visited", n.tokens.Start, n.tokens.End)
		}
	}
}

func TestSearchSet_Tokenize(t *testing.T) {
	tests := []struct {
		text string
		want tokens
	}{
		{
			text: "Tokenize",
			want: tokens{&token{Token: "Tokenize", Offset: 0}},
		},
		{
			text: "Hello world",
			want: tokens{
				&token{Token: "Hello", Offset: 0},
				&token{Token: "world", Offset: 6},
			},
		},
		{
			text: `Goodnight,
Irene
`,
			want: tokens{
				&token{Token: "Goodnight", Offset: 0},
				&token{Token: ",", Offset: 9},
				&token{Token: "Irene", Offset: 11},
			},
		},
		{
			text: "Copyright © 2017 Yoyodyne, Inc.",
			want: tokens{
				&token{Token: "Copyright", Offset: 0},
				&token{Token: "©", Offset: 10},
				&token{Token: "2017", Offset: 12},
				&token{Token: "Yoyodyne", Offset: 17},
				&token{Token: ",", Offset: 25},
				&token{Token: "Inc", Offset: 27},
				&token{Token: ".", Offset: 30},
			},
		},
	}

	for _, tt := range tests {
		if got := tokenize(tt.text); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Tokenize(%q) = %+v, want %+v", tt.text, got, tt.want)
		}
	}
}

func TestSearchSet_GenerateHashes(t *testing.T) {
	tests := []struct {
		text       string
		sizeFactor int
		wantHash   []uint32
		wantRanges TokenRanges
	}{
		{
			text:       "",
			sizeFactor: 1,
			wantHash:   []uint32{0},
			wantRanges: TokenRanges{{Start: 0, End: 0}},
		},
		{
			text:       "Hashes",
			sizeFactor: 1,
			wantHash:   []uint32{408116689},
			wantRanges: TokenRanges{{Start: 0, End: 1}},
		},
		{
			text:       "hello world",
			sizeFactor: 1,
			wantHash:   []uint32{222957957},
			wantRanges: TokenRanges{{Start: 0, End: 2}},
		},
		{
			text:       "Copyright © 2017 Yoyodyne, Inc.",
			sizeFactor: 3,
			wantHash:   []uint32{2473816729, 966085113, 3025678301, 3199087486, 850352802, 1274745089},
			wantRanges: TokenRanges{
				{Start: 0, End: 2},
				{Start: 1, End: 3},
				{Start: 2, End: 4},
				{Start: 3, End: 5},
				{Start: 4, End: 6},
				{Start: 5, End: 7},
			},
		},
	}

	for _, tt := range tests {
		hash := make(hash)
		toks := tokenize(tt.text)
		h, tr := toks.generateHashes(hash, len(toks)/tt.sizeFactor)
		if !reflect.DeepEqual(h, tt.wantHash) {
			t.Errorf("GenerateHashes(hash) = %v, want %v", h, tt.wantHash)
		}
		if !reflect.DeepEqual(tr, tt.wantRanges) {
			t.Errorf("GenerateHashes(ranges) = %v, want %v", tr, tt.wantRanges)
		}
	}
}

func TestSearchSet_CoalesceTokenRanges(t *testing.T) {
	tests := []struct {
		description string
		tr          TokenRanges
		sr          func(TokenRanges) map[*TokenRange]TokenRanges
		want        TokenRanges
	}{
		{
			description: "Non-overlapping Ranges",
			tr: TokenRanges{
				{Start: 0, End: 27},
				{Start: 37, End: 927},
			},
			sr: func(TokenRanges) map[*TokenRange]TokenRanges {
				return make(map[*TokenRange]TokenRanges)
			},
			want: TokenRanges{
				{Start: 0, End: 27},
				{Start: 37, End: 927},
			},
		},
		{
			description: "Identical Ranges",
			tr: TokenRanges{
				{Start: 0, End: 37},
				{Start: 0, End: 37},
			},
			sr: func(TokenRanges) map[*TokenRange]TokenRanges {
				return make(map[*TokenRange]TokenRanges)
			},
			want: TokenRanges{{Start: 0, End: 37}},
		},
		{
			description: "Sequential Ranges",
			tr: TokenRanges{
				{Start: 0, End: 37},
				{Start: 37, End: 927},
			},
			sr: func(tr TokenRanges) map[*TokenRange]TokenRanges {
				sr := make(map[*TokenRange]TokenRanges)
				sr[tr[0]] = TokenRanges{{Start: 0, End: 37}}
				sr[tr[1]] = TokenRanges{{Start: 37, End: 927}}
				return sr
			},
			want: TokenRanges{{Start: 0, End: 927}},
		},
		{
			description: "Non-Sequential Ranges",
			tr: TokenRanges{
				{Start: 0, End: 37},
				{Start: 37, End: 927},
			},
			sr: func(tr TokenRanges) map[*TokenRange]TokenRanges {
				sr := make(map[*TokenRange]TokenRanges)
				sr[tr[0]] = TokenRanges{{Start: 0, End: 37}}
				sr[tr[1]] = TokenRanges{{Start: 127, End: 927}}
				return sr
			},
			want: TokenRanges{
				{Start: 0, End: 37},
				{Start: 37, End: 927},
			},
		},
		{
			description: "Overlapping Ranges - Same Start",
			tr: TokenRanges{
				{Start: 0, End: 37},
				{Start: 0, End: 927},
			},
			sr: func(TokenRanges) map[*TokenRange]TokenRanges {
				return make(map[*TokenRange]TokenRanges)
			},
			want: TokenRanges{{Start: 0, End: 927}},
		},
		{
			description: "Overlapping Ranges - Different Start",
			tr: TokenRanges{
				{Start: 0, End: 37},
				{Start: 27, End: 927},
			},
			sr: func(TokenRanges) map[*TokenRange]TokenRanges {
				return make(map[*TokenRange]TokenRanges)
			},
			want: TokenRanges{{Start: 0, End: 927}},
		},
		{
			description: "Overlapping Ranges - Same End",
			tr: TokenRanges{
				{Start: 0, End: 37},
				{Start: 27, End: 37},
			},
			sr: func(TokenRanges) map[*TokenRange]TokenRanges {
				return make(map[*TokenRange]TokenRanges)
			},
			want: TokenRanges{{Start: 0, End: 37}},
		},
		{
			description: "Completely Overlapping Ranges",
			tr: TokenRanges{
				{Start: 0, End: 42},
				{Start: 27, End: 37},
			},
			sr: func(TokenRanges) map[*TokenRange]TokenRanges {
				return make(map[*TokenRange]TokenRanges)
			},
			want: TokenRanges{{Start: 0, End: 42}},
		},
	}

	for _, tt := range tests {
		got := coalesceTokenRanges(tt.tr, tt.sr(tt.tr))
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("CoalesceTokenRanges(%q) = %+v, want %+v", tt.description, got, tt.want)
		}
	}
}

// readFile locates and reads the data file.
func readFile(filename, dir string) ([]byte, error) {
	for _, path := range filepath.SplitList(os.Getenv("GOPATH")) {
		archive := filepath.Join(path, dir, filename)
		if _, err := os.Stat(archive); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		return ioutil.ReadFile(archive)
	}
	return nil, nil
}

func TestSearchSet_FindPotentialMatches(t *testing.T) {
	postmodernThesisGOB, err := readFile("postmodern_thesis.gob", "src/github.com/google/licenseclassifier/stringclassifier/searchset/testdata")
	if err != nil {
		t.Fatalf("Cannot read postmodern thesis GOB file: %v", err)
	}

	r := bytes.NewReader(postmodernThesisGOB)
	var known SearchSet
	if err := Deserialize(r, &known); err != nil {
		t.Fatalf("Deserialization: cannot deserialize set: %v", err)
	}

	w := "[0:46] :: " +
		"[0:23] -> [11:34] -> [22:45] :: " +
		"[0:15] -> [7:22] -> [14:29] -> [21:36] -> [28:43] :: " +
		"[0:11] -> [5:16] -> [10:21] -> [15:26] -> [20:31] -> [25:36] -> [30:41] -> [35:46] :: " +
		"[0:9] -> [4:13] -> [8:17] -> [12:21] -> [16:25] -> [20:29] -> [24:33] -> [28:37] -> [32:41] -> [36:45] :: " +
		"[0:7] -> [3:10] -> [6:13] -> [9:16] -> [12:19] -> [15:22] -> [18:25] -> [21:28] -> [24:31] -> [27:34] -> [30:37] -> [33:40] -> [36:43] -> [39:46] :: " +
		"[0:6] -> [3:9] -> [6:12] -> [9:15] -> [12:18] -> [15:21] -> [18:24] -> [21:27] -> [24:30] -> [27:33] -> [30:36] -> [33:39] -> [36:42] -> [39:45] :: " +
		"[0:5] -> [2:7] -> [4:9] -> [6:11] -> [8:13] -> [10:15] -> [12:17] -> [14:19] -> [16:21] -> [18:23] -> [20:25] -> [22:27] -> [24:29] -> [26:31] -> [28:33] -> [30:35] -> [32:37] -> [34:39] -> [36:41] -> [38:43] -> [40:45] :: " +
		"[0:4] -> [2:6] -> [4:8] -> [6:10] -> [8:12] -> [10:14] -> [12:16] -> [14:18] -> [16:20] -> [18:22] -> [20:24] -> [22:26] -> [24:28] -> [26:30] -> [28:32] -> [30:34] -> [32:36] -> [34:38] -> [36:40] -> [38:42] -> [40:44] -> [42:46] :: " +
		"[0:3] -> [1:4] -> [2:5] -> [3:6] -> [4:7] -> [5:8] -> [6:9] -> [7:10] -> [8:11] -> [9:12] -> [10:13] -> [11:14] -> [12:15] -> [13:16] -> [14:17] -> [15:18] -> [16:19] -> [17:20] -> [18:21] -> [19:22] -> [20:23] -> [21:24] -> [22:25] -> [23:26] -> [24:27] -> [25:28] -> [26:29] -> [27:30] -> [28:31] -> [29:32] -> [30:33] -> [31:34] -> [32:35] -> [33:36] -> [34:37] -> [35:38] -> [36:39] -> [37:40] -> [38:41] -> [39:42] -> [40:43] -> [41:44] -> [42:45] -> [43:46]"

	if got := known.lattice.String(); got != w {
		t.Errorf("LatticeMismatch(known) = %s\nwant:\n%s", got, w)
	}

	size := len(postmodernThesis)
	modified := "hello world "
	modified += postmodernThesis[:size/3] + " hello world "
	modified += postmodernThesis[size/3 : 2*size/3-4]
	modified += postmodernThesis[2*size/3+7:]
	unknown := New(modified, DefaultGranularity)

	want := []MatchRange{{SrcStart: 0, SrcEnd: 247, TargetStart: 12, TargetEnd: 261}}
	got := FindPotentialMatches(&known, unknown)
	if len(got) != len(want) {
		t.Errorf("Number of matches %v, want %v", len(got), len(want))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Offsets = %+v, want %+v", got, want)
	}

	known = *New(`again in Ulysses, although in a more neomodern sense.
culture. The without/within distinction intrinsic to Finnegan's Wake emerges
`, DefaultGranularity)

	want = []MatchRange{{SrcStart: 0, SrcEnd: 53, TargetStart: 208, TargetEnd: 261}}
	got = FindPotentialMatches(&known, unknown)
	if len(got) != len(want) {
		t.Errorf("Number of matches %v, want %v", len(got), len(want))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Offsets = %+v, want %+v", got, want)
	}
}

func TestSearchSet_CombineUnique(t *testing.T) {
	tests := []struct {
		description string
		this        TokenRanges
		that        TokenRanges
		want        func(this, that TokenRanges) TokenRanges
	}{
		{
			description: "Nil Range",
			this:        TokenRanges{{Start: 0, End: 37}},
			that:        nil,
			want: func(this, that TokenRanges) TokenRanges {
				return this
			},
		},
		{
			description: "Empty Range",
			this:        TokenRanges{},
			that:        TokenRanges{{Start: 0, End: 37}},
			want: func(this, that TokenRanges) TokenRanges {
				return that
			},
		},
		{
			description: "Disjoint Ranges",
			this:        TokenRanges{{Start: 0, End: 37}},
			that:        TokenRanges{{Start: 42, End: 927}},
			want: func(this, that TokenRanges) TokenRanges {
				return append(this, that...)
			},
		},
		{
			description: "Equal Ranges",
			this:        TokenRanges{{Start: 0, End: 37}},
			that:        TokenRanges{{Start: 0, End: 37}},
			want: func(this, _ TokenRanges) TokenRanges {
				return this
			},
		},
		{
			description: "Overlapping Ranges",
			this: TokenRanges{
				{Start: 27, End: 42},
				{Start: 0, End: 37},
				{Start: 3, End: 13},
				{Start: 27, End: 42},
			},
			that: TokenRanges{
				{Start: 927, End: 1024},
				{Start: 27, End: 42},
			},
			want: func(_, _ TokenRanges) TokenRanges {
				return TokenRanges{
					{Start: 0, End: 37},
					{Start: 3, End: 13},
					{Start: 27, End: 42},
					{Start: 927, End: 1024},
				}
			},
		},
	}

	for _, tt := range tests {
		got, want := tt.this.combineUnique(tt.that), tt.want(tt.this, tt.that)
		if len(got) != len(want) {
			t.Errorf("CombineUnique(%q) = length %v, want length %v", tt.description, len(got), len(want))
			continue
		}
		for i := 0; i < len(got); i++ {
			if !reflect.DeepEqual(got[i], want[i]) {
				t.Errorf("CombineUnique(%q) = position %d, %+v, want %+v", tt.description, i, got[i], want[i])
			}
		}
	}
}
