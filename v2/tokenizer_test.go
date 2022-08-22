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
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCleanupToken(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{{
		input:  "cleanup!",
		output: "cleanup",
	},
		{
			input:  "12345",
			output: "12345",
		},
		{
			input:  "r1@zx42-",
			output: "rzx",
		},
		{
			input:  "12345,",
			output: "12345",
		},
		{
			input:  "12345-6789",
			output: "12345-6789",
		},
		{
			input:  "1(a)",
			output: "1",
		},
		{
			input:  "1.2.3",
			output: "1.2.3",
		},
	}
	for _, test := range tests {
		if got := cleanupToken(0, test.input, true); got != test.output {
			t.Errorf("%q: got %q want %q", test.input, got, test.output)
		}
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output *indexedDocument
	}{
		{name: "hyphenization recovery",
			input: `basket-
ball`,
			output: &indexedDocument{
				Tokens: []indexedToken{
					{
						ID:   1,
						Line: 1,
					},
				},
				Norm: "basketball",
			},
		},
		{
			name: "basic scenario",
			input: `The AWESOME Project LICENSE

Modifi-
cations prohibited

Copyright 1996-2002, 2006 by A. Developer

Introduction

The AWESOME Project`,
			output: &indexedDocument{
				Tokens: []indexedToken{
					{
						ID:   1,
						Line: 1,
					},
					{
						ID:   2,
						Line: 1,
					},
					{
						ID:   3,
						Line: 1,
					},
					{
						ID:   4,
						Line: 1,
					},
					{
						ID:   5,
						Line: 3,
					},
					{
						ID:   6,
						Line: 4,
					},
					{
						ID:   7,
						Line: 8,
					},
					{
						ID:   1,
						Line: 10,
					},
					{
						ID:   2,
						Line: 10,
					},
					{
						ID:   3,
						Line: 10,
					},
				},
				Matches: Matches{&Match{Name: "Copyright", Confidence: 1.0, MatchType: "Copyright", StartLine: 6, EndLine: 6}},
				Norm:    "the awesome project license modifications prohibited introduction the awesome project",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d, err := tokenizeStream(bytes.NewReader([]byte(test.input)), true, newDictionary(), true)
			if err != nil {
				t.Errorf("%s failed: got unexpected error %v", test.name, err)
			}
			if diff := cmp.Diff(d, test.output, cmpopts.IgnoreUnexported(indexedDocument{})); diff != "" {
				t.Errorf("%s failed:\nDiff(+got,-want): %s", test.name, diff)
			}
		})
	}
}

type mockReader struct {
	t        *testing.T
	schedule []int
	cur      int
}

func (m *mockReader) Read(buf []byte) (int, error) {
	if m.cur > len(m.schedule) {
		m.t.Fatal("Unexpected read on mock")
	}

	if m.cur == len(m.schedule) {
		return 0, io.EOF
	}

	if len(buf) != m.schedule[m.cur] {
		m.t.Fatalf("step %d: got %d, want %d", m.cur, len(buf), m.schedule[m.cur])
	}
	m.cur++

	for i := range buf {
		buf[i] = 'a'
	}

	return len(buf), nil
}

func TestTokenizerBuffering(t *testing.T) {
	dict := newDictionary()
	mr := mockReader{
		t:        t,
		schedule: []int{1024, 1020, 1020},
	}
	d, err := tokenizeStream(&mr, true, dict, true)
	if err != nil {
		t.Errorf("Read returned unexpected error: %v", err)
	}

	// Do a basic test to make sure the data returned is sound
	if len(d.Tokens) != 1 {
		t.Errorf("Got %d tokens, expected 1", len(d.Tokens))
	}

	if len(d.Norm) != 3064 {
		t.Errorf("Got %d bytes, expected 3064", len(d.Norm))
	}
}

func TestTokenizer(t *testing.T) {
	// This test focuses primarily on the textual content extracted and does not look
	// at the other parts of the document.
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "Basic Tokens",
			input:  "Here are some words. ",
			output: "here are some words",
		},
		{
			name:   "skips bullet headers",
			input:  "* item the first\n· item the second",
			output: "item the first item the second",
		},
		{
			name:   "preserves version numbers but not header numbers",
			input:  "sample rules\n1. Python 2.7.8 is a version of the language.",
			output: "sample rules python 2.7.8 is a version of the language",
		},
		{
			name:   "preserves version numbers across line breaks",
			input:  "Python version\n2.7.8 is a version of the language.",
			output: "python version 2.7.8 is a version of the language",
		},
		{
			name:   "preserves punctuation",
			input:  "Bill, Larry, and Sergey agree precision is critical!",
			output: "bill larry and sergey agree precision is critical",
		},
		{
			name:   "ignores comment characters and bullet formatting",
			input:  "/* * item the first",
			output: "item the first",
		},
		{
			name:   "produces blank line as needed",
			input:  "/* *",
			output: "",
		},
		{
			name:   "clobbers header looking thing as appropriate",
			input:  " iv. this is a test",
			output: "this is a test",
		},
		{
			name:   "clobbers header looking thing as appropriate even in comment",
			input:  "/* 1.2.3. this is a test",
			output: "this is a test",
		},
		{
			name:   "preserve version number (not a header, but header-looking) not at beginning of sentence",
			input:  "This is version 1.1.",
			output: "this is version 1.1",
		},
		{
			name:   "copyright inside a comment",
			input:  " /* Copyright (c) 1998-2008 The OpenSSL Project. All rights reserved",
			output: "",
		},
		{
			name: "FTL copyright text",
			input: `The FreeType Project LICENSE

2006-Jan-27
2006-01-27

Copyright 1996-2002, 2006 by David Turner, Robert Wilhelm, and Werner Lemberg

Introduction

The FreeType Project`,
			output: "the freetype project license introduction the freetype project",
		},
		{
			name: "Separated text",
			input: `distribution and modifi‐
				       cation follow.`,
			output: "distribution and modification follow",
		},
		{
			name:   "preserve internal references, even on line break",
			input:  "(ii) should be preserved as (ii) is preserved",
			output: "ii should be preserved as ii is preserved",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dict := newDictionary()
			d, err := tokenizeStream(bytes.NewReader([]byte(test.input)), true, dict, true)
			if err != nil {
				t.Errorf("%s failed: got unexpected error %v", test.name, err)
			}
			var b strings.Builder
			for _, tok := range d.Tokens {
				b.WriteString(dict.getWord(tok.ID))
				b.WriteString(" ")
			}
			actual := strings.TrimSpace(b.String())
			if actual != test.output {
				t.Errorf("Tokenize(%q): got %q want %q", test.name, actual, test.output)
			}
		})
	}
}
