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

type tokenID int // type to ensure safety when manipulating token identifiers.

// token provides detailed information about a single textual token in the document.
type token struct {
	Text     string  // normalized text of the token
	Index    int     // the token's location in the tokenized document
	Line     int     // line position of this token in the source
	Previous string  // for the first token in a line, any previous text.
	ID       tokenID // identifier of the text in the dictionary
}

// document is the representation of the input text for downstream filtering and matching.
type document struct {
	Tokens []*token // ordered tokens of the document
}

type indexedToken struct {
	Index int     // the token's location in the tokenized document
	Line  int     // line position of this token in the source
	ID    tokenID // identifier of the text in the dictionary
}

type indexedDocument struct {
	Tokens []indexedToken  // ordered tokens of the document
	f      *frequencyTable // frequencies computed for this document
	dict   *dictionary     // The corpus dictionary for this document
}

// Corpus is a collection of documents with a shared dictionary. Matching occurs
// within all documents in the corpus.
// TODO: This type may not be public in the long-term. I need to write the A/B classifier
// facade and see how it best works out.
type Corpus struct {
	dict      *dictionary
	docs      map[string]*indexedDocument
	threshold float64
}

// NewCorpus creates an empty corpus.
func NewCorpus(threshold float64) *Corpus {
	corpus := &Corpus{
		dict:      newDictionary(),
		docs:      make(map[string]*indexedDocument),
		threshold: threshold,
	}
	return corpus
}

// AddContent incorporates the provided textual content into the corpus for matching.
func (c *Corpus) AddContent(name, content string) {
	doc := tokenize(content)
	c.addDocument(name, doc)
}

// addDocument takes a textual document and incorporates it into the corpus for matching.
func (c *Corpus) addDocument(name string, doc *document) {
	// For documents that are part of the corpus, we add them to the dictionary and
	// compute their associated search data eagerly so they are ready for matching against
	// candidates.
	id := c.generateIndexedDocument(doc, true)
	id.generateFrequencies()
	c.docs[name] = id
}

// generateIndexedDocument creates an indexedDocument from the supplied document. if addWords
// is true, the corpus dictionary is updated with new tokens encountered in the document.
func (c *Corpus) generateIndexedDocument(d *document, addWords bool) *indexedDocument {
	id := &indexedDocument{
		Tokens: make([]indexedToken, 0, len(d.Tokens)),
		dict:   c.dict,
	}

	for _, t := range d.Tokens {
		var tokID tokenID
		if addWords {
			tokID = id.dict.add(t.Text)
		} else {
			tokID = id.dict.getIndex(t.Text)
		}

		id.Tokens = append(id.Tokens, indexedToken{
			Index: t.Index,
			Line:  t.Line,
			ID:    tokID,
		})

	}
	id.generateFrequencies()
	return id
}

// createTargetIndexedDocument creates an indexed document without adding the
// words to the corpus dictionary. This should be used for matching targets, not
// populating the corpus.
func (c *Corpus) createTargetIndexedDocument(in string) *indexedDocument {
	doc := tokenize(in)
	return c.generateIndexedDocument(doc, false)
}

// dictionary is used to intern all the token words encountered in the text corpus.
// words and indices form an inverse mapping relationship. It is just a convenience type
// over a pair of correlated maps.
type dictionary struct {
	words   map[tokenID]string
	indices map[string]tokenID
}

func newDictionary() *dictionary {
	return &dictionary{
		words:   make(map[tokenID]string),
		indices: make(map[string]tokenID),
	}
}

// add inserts the provided word into the dictionary if it does not already exist.
func (d *dictionary) add(word string) tokenID {
	if idx := d.getIndex(word); idx != -1 {
		return idx
	}
	idx := tokenID(len(d.words))
	d.words[idx] = word
	d.indices[word] = idx
	return idx
}

var unknownWord = "UNKNOWN"
var unknownIndex = tokenID(-1)

// getIndex returns the index of the supplied word, or -1 if the word is not in the dictionary.
func (d *dictionary) getIndex(word string) tokenID {
	if idx, found := d.indices[word]; found {
		return idx
	}
	return unknownIndex
}

// getWord returns the word associated with the index.
func (d *dictionary) getWord(index tokenID) string {
	if word, found := d.words[index]; found {
		return word
	}
	return unknownWord
}
