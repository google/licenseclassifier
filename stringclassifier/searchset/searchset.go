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

// Package searchset generates hashes for all substrings of a text. Potential
// matches between two SearchSet objects can then be determined quickly.
// Generating the hashes can be expensive, so it's best to perform it once. If
// the text is part of a known corpus, then the SearchSet can be serialized and
// kept in an archive.
package searchset

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"hash/crc32"
	"io"
	"sort"
	"strings"
	"unicode"

	"github.com/google/licenseclassifier/stringclassifier/internal/sets"
)

// DefaultGranularity is the minimum size (in words) of the hash chunks.
const DefaultGranularity = 2

// SearchSet is a set of substrings that have hashes associated with them,
// making it fast to search for potential matches.
type SearchSet struct {
	// Tokens is a tokenized list of the original input string.
	Tokens tokens
	// Hashes is a map of checksums to a range of tokens.
	Hashes hash
	// Checksums is a list of checksums ordered from longest range to
	// shortest.
	Checksums []uint32
	// ChecksumRanges are the token ranges for the above checksums.
	ChecksumRanges TokenRanges
	lattice        lattice
}

// lattice is a data structure laid on top of the search set that organizes the
// nodes in a progressively finer-grained order.
type lattice struct {
	root *node
}

func (lat *lattice) String() string {
	n := lat.root
	var out []string
	for {
		var line []string
		for n != nil {
			line = append(line, n.String())
			n = n.sibling
			if n == nil || n.tokens.Start == 0 {
				break
			}
		}
		out = append(out, strings.Join(line, " -> "))
		if n == nil {
			break
		}
	}
	return strings.Join(out, " :: ")
}

type present struct{}

// node is a lattice node. It consists of a range of tokens along with the
// checksum for those tokens. A node's sibling is the next node in a BFS
// traversal of the lattice.
type node struct {
	tokens   *TokenRange
	checksum uint32
	sibling  *node
	children map[*node]present
}

// markChildrenVisited recursively marks the node's children as "visited".
func (n *node) markChildrenVisited(visited map[*node]present) {
	for child := range n.children {
		visited[child] = present{}
		child.markChildrenVisited(visited)
	}
}

func (n *node) String() string {
	return fmt.Sprintf("[%d:%d]", n.tokens.Start, n.tokens.End)
}

type hash map[uint32]TokenRanges

// add associates a token range, [start, end], to a checksum.
func (h hash) add(checksum uint32, start, end int) {
	ntr := &TokenRange{Start: start, End: end}
	if r, ok := h[checksum]; ok {
		for _, tr := range r {
			if tr.Start == ntr.Start && tr.End == ntr.End {
				// The token range already exists at this
				// checksum. No need to re-add it.
				return
			}
		}
	}
	h[checksum] = append(h[checksum], ntr)
}

// size returns the approximate size of the hash in bytes.
func (h hash) size() int {
	s := 0
	for _, v := range h {
		s += v.size() + 8 // 8 bytes for the key.
	}
	return s
}

// New creates a new SearchSet object. It generates a hash for each substring
// of "s".
func New(s string, granularity int) *SearchSet {
	toks := tokenize(s)

	// Start generating hash values for all substrings within the text. It
	// does this by creating a "window" over the token list that's half the
	// size of the list, then a third the size, etc., and moving each
	// window down the list.
	processed := sets.NewIntSet()
	var hashes []hash
	var checksums []uint32
	var tokenRanges TokenRanges
	for window, n := len(toks), 2; window > granularity; window, n = len(toks)/n, n+1 {
		if processed.Contains(window) {
			continue
		}
		processed.Insert(window)

		h := make(hash)
		hashes = append(hashes, h)

		cs, tr := toks.generateHashes(h, window)
		checksums = append(checksums, cs...)
		tokenRanges = append(tokenRanges, tr...)
	}

	combinedHash := make(hash)
	for _, h := range hashes {
		for checksum, ranges := range h {
			combinedHash[checksum] = ranges.combineUnique(combinedHash[checksum])
		}
	}

	sset := &SearchSet{
		Tokens:         toks,
		Hashes:         combinedHash,
		Checksums:      checksums,
		ChecksumRanges: tokenRanges,
	}
	sset.ConstructLattice()
	return sset
}

// MatchRange is the range within the text that is a good potential match.
type MatchRange struct {
	// Offsets into the source text.
	SrcStart, SrcEnd int
	// Offsets into the target text.
	TargetStart, TargetEnd int
}

// matchNodes goes through the source (known) search set and finds those nodes
// which match nodes in the target (unknown) search set.
func matchNodes(src, target *SearchSet, curNode *node, srcRanges map[*TokenRange]TokenRanges, visited map[*node]present) TokenRanges {
	var tr TokenRanges
	for curNode != nil {
		if _, ok := visited[curNode]; !ok {
			visited[curNode] = present{}
			if sv, ok := src.Hashes[curNode.checksum]; ok {
				for _, val := range target.Hashes[curNode.checksum] {
					newval := &TokenRange{Start: val.Start, End: val.End}
					tr = append(tr, newval)
					srcRanges[newval] = sv
				}
				curNode.markChildrenVisited(visited)
			}
		}
		curNode = curNode.sibling
	}
	return tr
}

// FindPotentialMatches returns the offset(s) into "target"'s text that are
// best potential matches for text in "src".
func FindPotentialMatches(src, target *SearchSet) []MatchRange {
	visited := make(map[*node]present)
	visited[target.lattice.root] = present{}

	srcRanges := make(map[*TokenRange]TokenRanges)
	tarRanges := matchNodes(src, target, target.lattice.root, srcRanges, visited)

	if len(tarRanges) == 0 {
		return nil
	}

	// Coalesce the token ranges. The end result will be contiguous ranges
	// over the "target" search set that match substrings in the "src"
	// search set.
	sort.Sort(tarRanges) // Sort by start index.
	results := coalesceTokenRanges(tarRanges, srcRanges)

	// Now go through the final results and try to determine which of the
	// coalesced ranges should be considered a potential match.
	var mr []MatchRange
	var prevRange *TokenRange
	for _, result := range results {
		// The range of the match in the target text.
		targetStart := target.Tokens[result.Start]
		targetEnd := target.Tokens[result.End-1]

		// The length of the matching texts.
		length := targetEnd.Offset + len(targetEnd.Token) - targetStart.Offset

		// The start of the range of the match in the source text.
		srcRange := srcRanges[result][0]
		if prevRange != nil {
			// If there are multiple ranges that match. Go through
			// the source matches until we find one that's beyond
			// the previous range.
			for _, v := range srcRanges[result] {
				if prevRange.End <= v.Start {
					srcRange = v
					break
				}
			}
		}

		mr = append(mr, MatchRange{
			SrcStart:    src.Tokens[srcRange.Start].Offset,
			SrcEnd:      src.Tokens[srcRange.Start].Offset + length,
			TargetStart: targetStart.Offset,
			TargetEnd:   targetStart.Offset + length,
		})
		prevRange = srcRange
	}

	// Go through the results once more to concatenate ranges.
	matches := []MatchRange{mr[0]}
	for prev, curr := 0, 1; curr < len(mr); prev, curr = curr, curr+1 {
		last := len(matches) - 1
		if mr[prev].SrcStart <= mr[curr].SrcStart && mr[prev].SrcEnd >= mr[curr].SrcStart {
			// p: |---|
			// c:  |+++|
			// r: |****|
			matches[last].SrcEnd = mr[curr].SrcEnd
			matches[last].TargetEnd = mr[curr].TargetEnd
		} else if mr[prev].SrcEnd <= mr[curr].SrcStart {
			// p: |---|
			// c:       |+++|
			// r: |*********|
			matches[last].SrcEnd = mr[curr].SrcEnd
			matches[last].TargetEnd = mr[curr].TargetEnd
		} else if mr[prev].SrcStart > mr[curr].SrcStart {
			matches = append(matches, mr[curr])
		} else {
			matches[last].SrcEnd = mr[curr].SrcEnd
			matches[last].TargetEnd = mr[curr].TargetEnd
		}
	}

	if len(matches) > 1 {
		// Remove matches that aren't near the start of the source text.
		lastToken := src.Tokens[len(src.Tokens)-1]
		slen := lastToken.Offset + len(lastToken.Token)
		for curr := len(matches) - 1; curr >= 0; curr-- {
			if slen != 0 && float64(matches[curr].SrcStart)/float64(slen) > 0.20 {
				matches = append(matches[:curr], matches[curr+1:]...)
			}
		}
	}

	return matches
}

// coalesceTokenRanges coalesces token ranges into contiguous regions.
func coalesceTokenRanges(ranges TokenRanges, srcRanges map[*TokenRange]TokenRanges) TokenRanges {
	results := TokenRanges{ranges[0]}
	for prev, i := results[0], 1; i < len(ranges); prev, i = ranges[i], i+1 {
		last := len(results) - 1
		if results[last].End <= ranges[i].Start {
			if results[last].End == ranges[i].Start {
				//   i: |---|
				// i+1:     |+++|
				pranges := srcRanges[prev]
				cranges := srcRanges[ranges[i]]

				// There may be situations where we have text that looks like:
				//
				//     yyyy XXXX yyyy
				//
				// where the first "yyyy" is identical to the last "yyyy", but
				// is part of a separate text. If we allow "yyyy XXXX" to match
				// as consecutive, then we will miss-classify the "XXXX yyyy" text.
				// Therefore, if the text isn't consecutive in the source, then
				// we don't make it consecutive here.
				var consecutive bool
			OUTER:
				for _, pr := range pranges {
					for _, cr := range cranges {
						if pr.End == cr.Start {
							consecutive = true
							break OUTER
						}
					}
				}

				if consecutive {
					results[last].End = ranges[i].End
				} else {
					results = append(results, ranges[i])
				}
			} else {
				//   i: |---|
				// i+1:       |+++|
				results = append(results, ranges[i])
			}
		} else if results[last].Start < ranges[i].Start {
			if results[last].End > ranges[i].Start && results[last].End < ranges[i].End {
				//   i: |---|
				// i+1:    |+++|
				results[last].End = ranges[i].End
			}
		} else if results[last].Start == ranges[i].Start {
			if results[last].End < ranges[i].End {
				//   i: |---|
				// i+1: |++++|
				results[last].End = ranges[i].End
			}
		}
	}
	return results
}

// Serialize emits the SearchSet out so that it can be recreated at a later
// time.
func (s *SearchSet) Serialize(w io.Writer) error {
	e := gob.NewEncoder(w)
	return e.Encode(s)
}

// Size returns the approximate memory usage of the SearchSet in bytes.
func (s *SearchSet) Size() int {
	return s.Tokens.size() + s.Hashes.size()
}

// ConstructLattice goes through the search set and links the nodes into a
// lattice. This allows us to efficiently query the search set to find matching
// nodes. We can also avoid visiting nodes multiple times.
func (s *SearchSet) ConstructLattice() {
	if len(s.Checksums) == 0 {
		return
	}
	var root *node
	var prev *node
	for i := 0; i < len(s.Checksums); i++ {
		n := &node{
			tokens:   s.ChecksumRanges[i],
			checksum: s.Checksums[i],
			children: make(map[*node]present),
		}
		if prev != nil {
			prev.sibling = n
		} else {
			root = n
		}
		prev = n
	}

	// For each node find its children node. Those are defined as nodes
	// whose token range falls within the parent's token range.
	prevRow, curNode := root, root.sibling
OUTER:
	for {
		rowBegin := curNode
		var prevNode *node
		for parentNode := prevRow; parentNode != nil; parentNode = parentNode.sibling {
			// Check the previous node to see if it's a child of
			// this parent as well.
			if prevNode != nil && parentNode.tokens.Start <= prevNode.tokens.Start && parentNode.tokens.End >= prevNode.tokens.End {
				parentNode.children[prevNode] = present{}
			}

			for curNode != nil {
				if parentNode.tokens.Start > curNode.tokens.Start || parentNode.tokens.End < curNode.tokens.End {
					break
				}
				parentNode.children[curNode] = present{}

				prevNode = curNode
				curNode = curNode.sibling
				if curNode != nil && curNode.tokens.Start == 0 {
					break
				}
			}

			if parentNode.sibling != nil && parentNode.sibling.tokens.Start == 0 {
				for curNode != nil && curNode.tokens.Start != 0 {
					curNode = curNode.sibling
				}
				prevNode = nil
			}

			if curNode == nil {
				break OUTER
			}
		}

		prevRow = rowBegin
	}
	s.lattice = lattice{root}
}

// Deserialize reads a file with a serialized SearchSet in it and reconstructs it.
func Deserialize(r io.Reader, s *SearchSet) error {
	d := gob.NewDecoder(r)
	if err := d.Decode(&s); err != nil {
		return err
	}
	s.ConstructLattice()
	return nil
}

// TokenRange indicates the range of tokens that map to a particular checksum.
type TokenRange struct {
	Start int
	End   int
}

// TokenRanges is a list of TokenRange objects. The chance that two different
// strings map to the same checksum is very small, but unfortunately isn't
// zero, so we use this instead of making the assumption that they will all be
// unique.
type TokenRanges []*TokenRange

func (t TokenRanges) Len() int           { return len(t) }
func (t TokenRanges) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t TokenRanges) Less(i, j int) bool { return t[i].Start < t[j].Start }

// combineUnique returns the combination of both token ranges with no duplicates.
func (t TokenRanges) combineUnique(other TokenRanges) TokenRanges {
	if len(other) == 0 {
		return t
	}
	if len(t) == 0 {
		return other
	}

	cu := append(t, other...)
	sort.Sort(cu)

	if len(cu) == 0 {
		return nil
	}

	res := TokenRanges{cu[0]}
	for prev, i := cu[0], 1; i < len(cu); i++ {
		if prev.Start != cu[i].Start || prev.End != cu[i].End {
			res = append(res, cu[i])
			prev = cu[i]
		}
	}
	return res
}

// size returns the approximate size of the token ranges in bytes.
func (t TokenRanges) size() int {
	return len(t) * 16
}

// token is a non-whitespace sequence (i.e., word or punctuation) in the
// original string. This is not meant for use outside of this package.
type token struct {
	Token  string
	Offset int
}

type tokens []*token

// generateHashes generates hashes for "size" length substrings. The
// "stringifyTokens" call takes a long time to run, so not all substrings have
// hashes.
func (t tokens) generateHashes(h hash, size int) ([]uint32, TokenRanges) {
	var css []uint32
	var tr TokenRanges
	for offset := 0; offset+size <= len(t); offset += size / 2 {
		var b bytes.Buffer
		t.stringifyTokens(&b, offset, size)
		cs := crc32.ChecksumIEEE(b.Bytes())
		css = append(css, cs)
		tr = append(tr, &TokenRange{offset, offset + size})
		h.add(cs, offset, offset+size)
		if size <= 1 {
			break
		}
	}

	return css, tr
}

// stringifyTokens serializes a sublist of tokens into a bytes buffer.
func (t tokens) stringifyTokens(b *bytes.Buffer, offset, size int) {
	for j := offset; j < offset+size; j++ {
		if j != offset {
			b.WriteRune(' ')
		}
		b.WriteString(t[j].Token)
	}
}

// size returnes the number of token objects.
func (t tokens) size() int {
	s := 0
	for _, e := range t {
		s += len(e.Token) + 8 // 8 bytes for the Offset.
	}
	return s
}

// newToken creates a new token object with an invalid (negative) offset, which
// will be set before the token's used.
func newToken() *token {
	return &token{Offset: -1}
}

// tokenize converts a string into a stream of tokens.
func tokenize(s string) (toks tokens) {
	xRunes := []rune(s)
	tok := newToken()
	for i, offset := 0, 0; offset < len(xRunes); i, offset = i+1, offset+1 {
		r := xRunes[offset]
		switch {
		case unicode.IsSpace(r):
			if tok.Offset >= 0 {
				toks = append(toks, tok)
				tok = newToken()
			}
		case unicode.IsPunct(r):
			if tok.Offset >= 0 {
				toks = append(toks, tok)
				tok = newToken()
			}
			toks = append(toks, &token{
				Token:  string(r),
				Offset: offset,
			})
		default:
			if tok.Offset == -1 {
				tok.Offset = offset
			}
			tok.Token += string(r)
		}
	}
	if tok.Offset != -1 {
		// Add any remaining token that wasn't yet included in the list.
		toks = append(toks, tok)
	}
	return toks
}
