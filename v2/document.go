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

// Package classifier provides the implementation of the v2 license classifier.
package classifier

// token provides detailed information about a single textual token in the document.
type token struct {
	Text     string // normalized text of the token
	Index    int    // the token's location in the tokenized document
	Line     int    // line position of this token in the source
	Previous string // for the first token in a line, any previous text.
	ID       int    // identifier of the text in the dictionary
}

// document is the representation of the input text for downstream filtering and matching.
type document struct {
	Tokens []*token
}

type indexedToken struct {
	Index int // the token's location in the tokenized document
	Line  int // line position of this token in the source
	ID    int // identifier of the text in the dictionary
}

type indexedDocument struct {
	Tokens []indexedToken
}
