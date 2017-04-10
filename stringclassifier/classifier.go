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

// Package stringclassifier finds the nearest match between a string and a set
// of known values. It uses the Levenshtein Distance algorithm to determine
// this. A confidence percentage is returned, which indicates how confident the
// algorithm is that the match is correct. The higher the percentage, the
// greater the confidence that the match is correct.
//
// Example Usage:
//
//   type Text struct {
//     Name string
//     Text string
//   }
//
//   func NewClassifier(knownTexts []Text) (*stringclassifier.Classifier, error) {
//     sc := stringclassifier.New(stringclassifier.FlattenWhitespace)
//     for _, known := range knownTexts {
//       if err := sc.AddValue(known.Name, known.Text); err != nil {
//         return nil, err
//       }
//     }
//     return sc, nil
//   }
//
//   func IdentifyTexts(sc *stringclassifier.Classifier, unknownTexts []*Text) {
//     for _, unknown := range unknownTexts {
//       m := sc.NearestMatch(unknown.Text)
//       log.Printf("The nearest match to %q is %q (confidence: %v)",
//         unknown.Name, m.Name, m.Confidence)
//     }
//   }
package stringclassifier

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/google/licenseclassifier/stringclassifier/internal/pq"
	"github.com/google/licenseclassifier/stringclassifier/internal/sets"
	"github.com/google/licenseclassifier/stringclassifier/searchset"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// The diff/match/patch algorithm.
var dmp = diffmatchpatch.New()

const defaultMinDiffRatio float64 = 0.75

// A Classifier matches a string to a set of known values.
type Classifier struct {
	muValues    sync.RWMutex
	values      map[string]*knownValue
	normalizers []NormalizeFunc

	// MinDiffRatio defines the minimum ratio of the length difference
	// allowed to consider a known value a possible match. This is used as
	// a performance optimization to eliminate values that are unlikely to
	// be a match.
	//
	// For example, a value of 0.75 means that the shorter string must be
	// at least 75% the length of the longer string to consider it a
	// possible match.
	//
	// Setting this to 1.0 will require that strings are identical length.
	// Setting this to 0 will consider all known values as possible
	// matches.
	MinDiffRatio float64
}

// NormalizeFunc is a function that is used to normalize a string prior to comparison.
type NormalizeFunc func(string) string

// New creates a new Classifier with the provided NormalizeFuncs. Each
// NormalizeFunc is applied in order to a string before comparison.
func New(funcs ...NormalizeFunc) *Classifier {
	return &Classifier{
		values:       make(map[string]*knownValue),
		normalizers:  append([]NormalizeFunc(nil), funcs...),
		MinDiffRatio: defaultMinDiffRatio,
	}
}

// knownValue identifies a value in the corpus to match against.
type knownValue struct {
	key             string
	normalizedValue string
	set             *searchset.SearchSet
}

// AddValue adds a known value to be matched against. If a value already exists
// for key, an error is returned.
func (c *Classifier) AddValue(key, value string) error {
	c.muValues.Lock()
	defer c.muValues.Unlock()
	if _, ok := c.values[key]; ok {
		return fmt.Errorf("value already registered with key %q", key)
	}
	c.values[key] = &knownValue{key: key, normalizedValue: c.normalize(value)}
	return nil
}

// AddPrecomputedValue adds a known value to be matched against. The value has
// already been normalized and the SearchSet object deserialized, so no
// processing is necessary.
func (c *Classifier) AddPrecomputedValue(key, value string, set *searchset.SearchSet) error {
	c.muValues.Lock()
	defer c.muValues.Unlock()
	if _, ok := c.values[key]; ok {
		return fmt.Errorf("value already registered with key %q", key)
	}
	set.ConstructLattice()
	c.values[key] = &knownValue{
		key:             key,
		normalizedValue: value,
		set:             set,
	}
	return nil
}

// normalize a string by applying each of the registered NormalizeFuncs.
func (c *Classifier) normalize(s string) string {
	for _, fn := range c.normalizers {
		s = fn(s)
	}
	return s
}

// Match identifies the result of matching a string against a knownValue.
type Match struct {
	Name       string  // Name of knownValue that was matched
	Confidence float64 // Confidence percentage
	Offset     int     // The offset into the unknown string the match was made
	Extent     int     // The length from the offset into the unknown string
}

// Matches is a list of Match-es. This is here mainly so that the list can be
// sorted.
type Matches []*Match

func (m Matches) Len() int      { return len(m) }
func (m Matches) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m Matches) Less(i, j int) bool {
	if math.Abs(m[j].Confidence-m[i].Confidence) < math.SmallestNonzeroFloat64 {
		if m[i].Name == m[j].Name {
			if m[i].Offset > m[j].Offset {
				return false
			}
			if m[i].Offset == m[j].Offset {
				return m[i].Extent > m[j].Extent
			}
			return true
		}
		return m[i].Name < m[j].Name
	}
	return m[i].Confidence > m[j].Confidence
}

// Names returns an unsorted slice of the names of the matched licenses.
func (m Matches) Names() []string {
	var names []string
	for _, n := range m {
		names = append(names, n.Name)
	}
	return names
}

// uniquify goes through the matches and removes any that are contained within
// one with a higher confidence. This assumes that Matches is sorted.
func (m Matches) uniquify() Matches {
	type matchedRange struct {
		offset, extent int
	}

	var matched []matchedRange
	var matches Matches
OUTER:
	for _, match := range m {
		for _, mr := range matched {
			if match.Offset >= mr.offset && match.Offset <= mr.offset+mr.extent {
				continue OUTER
			}
		}
		matched = append(matched, matchedRange{match.Offset, match.Extent})
		matches = append(matches, match)
	}

	return matches
}

// NearestMatch returns the name of the known value that most closely matches
// the unknown string and a confidence percentage is returned indicating how
// confident the classifier is in the result. A percentage of "1.0" indicates
// an exact match, while a percentage of "0.0" indicates a complete mismatch.
//
// If the string is equidistant from multiple known values, it is undefined
// which will be returned.
func (c *Classifier) NearestMatch(s string) *Match {
	pq := c.nearestMatch(s)
	if pq.Len() == 0 {
		return &Match{}
	}
	return pq.Pop().(*Match)
}

// MultipleMatch tries to determine which known strings are found within an
// unknown string. This differs from "NearestMatch" in that it looks only at
// those areas within the unknown string that are likely to match. A list of
// potential matches are returned. It's up to the caller to determine which
// ones are acceptable.
func (c *Classifier) MultipleMatch(s string) Matches {
	pq := c.multipleMatch(s)

	// A map to remove duplicate entries.
	m := make(map[Match]bool)

	var matches Matches
	for pq.Len() != 0 {
		v := pq.Pop().(*Match)
		if _, ok := m[*v]; !ok {
			m[*v] = true
			matches = append(matches, v)
		}
	}

	sort.Sort(matches)
	return matches.uniquify()
}

// possibleMatch identifies a known value and it's diffRatio to a given string.
type possibleMatch struct {
	value     *knownValue
	diffRatio float64
}

// likelyMatches is a slice of possibleMatches that can be sorted by their
// diffRatio to a given string, such that the most likely matches (based on
// length) are at the beginning.
type likelyMatches []possibleMatch

func (m likelyMatches) Len() int           { return len(m) }
func (m likelyMatches) Less(i, j int) bool { return m[i].diffRatio > m[j].diffRatio }
func (m likelyMatches) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }

// nearestMatch returns a Queue of values that the unknown string may be. The
// values are compared via their Levenshtein Distance and ranked with the
// nearest match at the beginning.
func (c *Classifier) nearestMatch(unknown string) *pq.Queue {
	var mu sync.Mutex // Protect the priority queue.
	pq := pq.NewQueue(func(x, y interface{}) bool {
		return x.(*Match).Confidence > y.(*Match).Confidence
	}, nil)

	unknown = c.normalize(unknown)
	if len(unknown) == 0 {
		return pq
	}

	c.muValues.RLock()
	var likely likelyMatches
	for _, v := range c.values {
		dr := diffRatio(unknown, v.normalizedValue)
		if dr < c.MinDiffRatio {
			continue
		}
		if unknown == v.normalizedValue {
			// We found an exact match.
			pq.Push(&Match{Name: v.key, Confidence: 1.0, Offset: 0, Extent: len(unknown)})
			c.muValues.RUnlock()
			return pq
		}
		likely = append(likely, possibleMatch{value: v, diffRatio: dr})
	}
	c.muValues.RUnlock()
	sort.Sort(likely)

	var wg sync.WaitGroup
	classifyString := func(name, unknown, known string) {
		defer wg.Done()

		diffs := dmp.DiffMain(unknown, known, true)
		distance := dmp.DiffLevenshtein(diffs)
		confidence := confidencePercentage(len(unknown), len(known), distance)
		if confidence > 0.0 {
			mu.Lock()
			pq.Push(&Match{Name: name, Confidence: confidence, Offset: 0, Extent: len(unknown)})
			mu.Unlock()
		}
	}

	wg.Add(len(likely))
	for _, known := range likely {
		go classifyString(known.value.key, unknown, known.value.normalizedValue)
	}
	wg.Wait()
	return pq
}

// multipleMatch returns a Queue of values that might be within the unknown
// string. The values are compared via their Levenshtein Distance and ranked
// with the nearest match at the beginning.
func (c *Classifier) multipleMatch(unknown string) *pq.Queue {
	var mu sync.Mutex // Protect the priority queue.
	queue := pq.NewQueue(func(x, y interface{}) bool {
		return x.(*Match).Confidence > y.(*Match).Confidence
	}, nil)

	normUnknown := c.normalize(unknown)
	if normUnknown == "" {
		return queue
	}
	set := searchset.New(normUnknown, searchset.DefaultGranularity)

	const threshold = 1 << 21 // 1MB
	setSize := set.Size()
	if setSize > threshold {
		// The string is simply too big to perform a multi-match. Do a
		// simple match to see if we can identify something.
		return c.nearestMatch(normUnknown)
	}

	var wg sync.WaitGroup
	classifyString := func(unknown, known *searchset.SearchSet, normUnknown, normKnown, name string) {
		defer wg.Done()

		if known == nil {
			known = searchset.New(normKnown, searchset.DefaultGranularity)
			c.muValues.Lock()
			c.values[name].set = known
			c.muValues.Unlock()
		}

		var indices []int
		for _, m := range searchset.FindPotentialMatches(known, unknown) {
			indices = append(indices, m.TargetStart)
		}

		q := pq.NewQueue(func(x, y interface{}) bool {
			return x.(*Match).Confidence > y.(*Match).Confidence
		}, nil)

		c.levenshteinDistances(normUnknown, normKnown, name, indices, q)
		if q.Len() != 0 {
			mu.Lock()
			for q.Len() > 0 {
				queue.Push(q.Pop().(*Match))
			}
			mu.Unlock()
		}
	}

	c.muValues.RLock()
	var kvals []*knownValue
	for _, known := range c.values {
		kvals = append(kvals, known)
	}
	c.muValues.RUnlock()

	wg.Add(len(kvals))
	for _, known := range kvals {
		go classifyString(set, known.set, normUnknown, known.normalizedValue, known.key)
	}
	wg.Wait()
	return queue
}

// levenshteinDistances searches through the "unknown" text for instances of
// the "known" text. The matching may not be 100%, so the Levenshtein Distance
// is calculated for each potential match. A list of candidate offsets are the
// offsets which have the best chance of matching.
func (c *Classifier) levenshteinDistances(unknown, known, name string, offsets []int, mpq *pq.Queue) {
	if len(known) == 0 || len(unknown) == 0 {
		log.Printf("Zero-sized texts in Levenshtein Distance algorithm: known==%d, unknown==%d", len(known), len(unknown))
		return
	}

	var muMPQ sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(offsets))
	for i := 0; i < len(offsets); i++ {
		go func(offset int) {
			defer wg.Done()

			// First, we calculate the range that's most likely to
			// match from the offset.
			end := endOfUnknownSubstring(unknown, known, offset)
			extent := end - offset

			if float64(extent)/float64(len(known)) < c.MinDiffRatio {
				// The size of the unknown text is too small
				// relative to the known text.
				return
			}

			// Calculate the differences between the potentially
			// matching known text and the unknown text.
			diffs := dmp.DiffMain(unknown[offset:end], known, false)
			end = diffRangeEnd(known, diffs)

			// Now execute the Levenshtein Distance algorithm to
			// see how much it does match.
			distance := dmp.DiffLevenshtein(diffs[:end])
			confidence := confidencePercentage(unknownTextLength(unknown, diffs), len(known), distance)
			if confidence > 0.0 {
				muMPQ.Lock()
				mpq.Push(&Match{Name: name, Confidence: confidence, Offset: offset, Extent: extent})
				muMPQ.Unlock()
			}
		}(offsets[i])
	}
	wg.Wait()
}

// unknownTextLength returns the length of the unknown text based on the diff
// range.
func unknownTextLength(unknown string, diffs []diffmatchpatch.Diff) int {
	last := len(diffs) - 1
	for ; last >= 0; last-- {
		if diffs[last].Type == diffmatchpatch.DiffEqual {
			break
		}
	}
	ulen := 0
	for i := 0; i < last+1; i++ {
		switch diffs[i].Type {
		case diffmatchpatch.DiffEqual, diffmatchpatch.DiffDelete:
			ulen += len(diffs[i].Text)
		}
	}
	return ulen
}

// endOfUnknownSubstring returns the end point of the substring within the
// unknown text that is likely to match the known value.
func endOfUnknownSubstring(unknown, known string, offset int) (end int) {
	klen := len(known)
	if len(unknown[offset:]) > klen {
		// If the unknown text is longer than the known text, we want
		// to make sure we don't skip any text within the unknown bit
		// that could match. So go backwards through the known text and
		// find a common suffix between the known and unknown.
		for i := 1; i < klen; i++ {
			if strings.HasPrefix(unknown[offset+klen:], known[klen-i-1:]) {
				end = klen + i + 1 + offset
				break
			}
		}
	}
	if end == 0 {
		// We couldn't find a common suffix or the known text was
		// longer than the unknown text.  Therefore, we just stop
		// either at the end of the unknown text or after the length of
		// the known text from the offset.
		end = min(len(unknown[offset:]), klen) + offset
	}
	return end
}

// diffRangeEnd returns the end index for the "Diff" objects that constructs
// (or nearly constructs) the "known" value.
func diffRangeEnd(known string, diffs []diffmatchpatch.Diff) (end int) {
	var seen string
	for end = 0; end < len(diffs); end++ {
		if seen == known {
			// Once we've constructed the "known" value, then we've
			// reached the point in the diff list where more
			// "Diff"s would just make the Levenshtein Distance
			// less valid. There shouldn't be further "DiffEqual"
			// nodes, because there's nothing further to match in
			// the "known" text.
			break
		}
		switch diffs[end].Type {
		case diffmatchpatch.DiffEqual, diffmatchpatch.DiffInsert:
			seen += diffs[end].Text
		}
	}
	return end
}

// findStartingIndices returns the indices into the "unknown" text which give
// the best bet on where to start calculating the Levenshtein Distance.
func findStartingIndices(known, unknown string) []int {
	klen, ulen := len(known), len(unknown)
	if klen > ulen {
		return []int{0}
	}

	// There may be several similar matching texts within the unknown text.
	// Gather them all and let the Levenshtein Distance calculation decide
	// which is the best match.
	indices := sets.NewIntSet()
	for klen > 4 {
		for start := 0; start < len(unknown); {
			i := strings.Index(unknown[start:], known[:klen])
			if i == -1 {
				break
			}
			if !indices.Contains(start + i) {
				indices.Insert(start + i)
			}
			start += i + klen
		}
		klen /= 2
	}
	return indices.Sorted()
}

// confidencePercentage calculates how confident we are in the result of the
// match. A percentage of "1.0" means an identical match. A confidence of "0.0"
// means a complete mismatch.
func confidencePercentage(ulen, klen, distance int) float64 {
	if ulen == 0 && klen == 0 {
		return 1.0
	}
	if ulen == 0 || klen == 0 || (distance > ulen && distance > klen) {
		return 0.0
	}
	return 1.0 - float64(distance)/float64(max(ulen, klen))
}

// diffRatio calculates the ratio of the length of s1 and s2, returned as a
// percentage of the length of the longer string. E.g., diffLength("abcd", "e")
// would return 0.25 because "e" is 25% of the size of "abcd". Comparing
// strings of equal length will return 1.
func diffRatio(s1, s2 string) float64 {
	x, y := len(s1), len(s2)
	if x == 0 && y == 0 {
		// Both strings are zero length
		return 1.0
	}
	if x < y {
		return float64(x) / float64(y)
	}
	return float64(y) / float64(x)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// wsRegexp is a regexp used to identify blocks of whitespace.
var wsRegexp = regexp.MustCompile(`\s+`)

// FlattenWhitespace will flatten contiguous blocks of whitespace down to a single space.
var FlattenWhitespace NormalizeFunc = func(s string) string {
	return wsRegexp.ReplaceAllString(s, " ")
}
