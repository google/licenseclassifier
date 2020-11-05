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
		if got := cleanupToken(test.input); got != test.output {
			t.Errorf("%q: got %q want %q", test.input, got, test.output)
		}
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output *document
	}{
		{
			name: "basic scenario",
			input: `The AWESOME Project LICENSE

Modifi-
cations prohibited

Copyright 1996-2002, 2006 by A. Developer

Introduction

The AWESOME Project`,
			output: &document{
				Tokens: []*token{
					{
						Text:  "the",
						Index: 0,
						Line:  1,
					},
					{
						Text:  "awesome",
						Index: 1,
						Line:  1,
					},
					{
						Text:  "project",
						Index: 2,
						Line:  1,
					},
					{
						Text:  "license",
						Index: 3,
						Line:  1,
					},
					{
						Text:  "modifications",
						Index: 4,
						Line:  3,
					},
					{
						Text:  "prohibited",
						Index: 5,
						Line:  4,
					},
					{
						Text:  "introduction",
						Index: 6,
						Line:  8,
					},
					{
						Text:  "the",
						Index: 7,
						Line:  10,
					},
					{
						Text:  "awesome",
						Index: 8,
						Line:  10,
					},
					{
						Text:  "project",
						Index: 9,
						Line:  10,
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := tokenize([]byte(test.input))
			if !cmp.Equal(d, test.output, cmpopts.IgnoreUnexported(document{})) {
				t.Errorf("%s failed: %s", test.name, cmp.Diff(d, test.output))
			}
		})
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
			d := tokenize([]byte(test.input))
			var b strings.Builder
			for _, tok := range d.Tokens {
				b.WriteString(tok.Text)
				b.WriteString(" ")
			}
			actual := strings.TrimSpace(b.String())
			if actual != test.output {
				t.Errorf("Tokenize(%q): got %q want %q", test.name, actual, test.output)
			}
		})
	}
}
