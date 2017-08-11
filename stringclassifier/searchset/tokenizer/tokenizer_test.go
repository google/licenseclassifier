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
package tokenizer

import (
	"reflect"
	"testing"
)

func TestTokenizer_Tokenize(t *testing.T) {
	tests := []struct {
		text string
		want Tokens
	}{
		{
			text: "Tokenize",
			want: Tokens{&token{Text: "Tokenize", Offset: 0}},
		},
		{
			text: "Hello world",
			want: Tokens{
				&token{Text: "Hello", Offset: 0},
				&token{Text: "world", Offset: 6},
			},
		},
		{
			text: `Goodnight,
Irene
`,
			want: Tokens{
				&token{Text: "Goodnight", Offset: 0},
				&token{Text: ",", Offset: 9},
				&token{Text: "Irene", Offset: 11},
			},
		},
		{
			text: "Copyright © 2017 Yoyodyne, Inc.",
			want: Tokens{
				&token{Text: "Copyright", Offset: 0},
				&token{Text: "©", Offset: 10},
				&token{Text: "2017", Offset: 13},
				&token{Text: "Yoyodyne", Offset: 18},
				&token{Text: ",", Offset: 26},
				&token{Text: "Inc", Offset: 28},
				&token{Text: ".", Offset: 31},
			},
		},
	}

	for _, tt := range tests {
		if got := Tokenize(tt.text); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Tokenize(%q) = %+v, want %+v", tt.text, got, tt.want)
		}
	}
}

func TestTokenizer_GenerateHashes(t *testing.T) {
	tests := []struct {
		text       string
		sizeFactor int
		wantHash   []uint32
		wantRanges TokenRanges
	}{
		{
			text:       "",
			sizeFactor: 1,
			wantHash:   nil,
			wantRanges: nil,
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
		hash := make(Hash)
		toks := Tokenize(tt.text)
		h, tr := toks.GenerateHashes(hash, len(toks)/tt.sizeFactor)
		if !reflect.DeepEqual(h, tt.wantHash) {
			t.Errorf("GenerateHashes(hash) = %v, want %v", h, tt.wantHash)
		}
		if !reflect.DeepEqual(tr, tt.wantRanges) {
			t.Errorf("GenerateHashes(ranges) = %v, want %v", tr, tt.wantRanges)
		}
	}
}
