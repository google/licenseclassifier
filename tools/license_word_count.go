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

// The license_word_count program counts the frequency of words as they appear
// in the known licenses. This information is useful if we want to be more
// selective about which files we run through the license classifier.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/google/licenseclassifier"
	"github.com/google/licenseclassifier/internal/sets"
	"github.com/google/stringclassifier/searchset"
)

var (
	top     = flag.Int("common", 10, "top n words in every license")
	verbose = flag.Bool("verbose", false, "verbose output")
)

type word struct {
	word  string
	count int
}

type words []word

func (w words) Len() int           { return len(w) }
func (w words) Swap(i, j int)      { w[i], w[j] = w[j], w[i] }
func (w words) Less(i, j int) bool { return w[i].count > w[j].count }

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s [OPTIONS]

Count word frequency in known licenses.

Options:
`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	licenses, err := licenseclassifier.ReadLicenseDir()
	if err != nil {
		log.Fatalf("error: cannot read licenses directory: %v", err)
	}

	wordMap := make(map[string]int)
	licenseMap := make(map[string]map[string]int)
	all := sets.NewStringSet()
	for _, license := range licenses {
		// All license files have a ".txt" extension.
		ext := filepath.Ext(license.Name())
		if ext != ".txt" {
			continue
		}

		contents, err := licenseclassifier.ReadLicenseFile(license.Name())
		if err != nil {
			log.Fatalf("error: cannot read license %q: %v", license.Name(), err)
		}

		str := licenseclassifier.TrimExtraneousTrailingText(string(contents))
		for _, n := range licenseclassifier.Normalizers {
			str = n(str)
		}

		// Generate the search set.
		set := searchset.New(str, searchset.DefaultGranularity)

		base := strings.TrimSuffix(license.Name(), ext)
		all.Insert(base)

		// Count the number of words used by the license.
		for _, tok := range set.Tokens {
			if common.Contains(tok.Token) || unicode.IsPunct([]rune(tok.Token)[0]) ||
				unicode.IsDigit([]rune(tok.Token)[0]) || len(tok.Token) < 3 {
				continue
			}

			wordMap[tok.Token]++
			if licenseMap[tok.Token] == nil {
				licenseMap[tok.Token] = make(map[string]int)
			}
			licenseMap[tok.Token][base]++
		}
	}

	var ws words
	for w, c := range wordMap {
		ws = append(ws, word{w, c})
	}
	sort.Sort(ws)

	lics := sets.NewStringSet()
	var com []string
	for i := 0; i < *top; i++ {
		com = append(com, ws[i].word)
		for k := range licenseMap[ws[i].word] {
			lics.Insert(k)
		}

		if lics.Len() == len(licenses) {
			break
		}
	}

	fmt.Printf("Words Common to [%d/%d] Licenses:\n    %s\n", lics.Len(), all.Len(), strings.Join(com, ", "))
	fmt.Printf("Missing Licenses: %v\n", all.Difference(lics))

	for _, w := range ws {
		if *verbose {
			fmt.Println(strings.Repeat("=", 80))
		}
		fmt.Printf("%v: %v\n", w.word, w.count)
		if *verbose {
			var keys []string
			for k := range licenseMap[w.word] {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			fmt.Printf("    [%d/%d] %s\n", len(keys), len(licenses), strings.Join(keys, ", "))
		}
	}
}

var common = sets.NewStringSet(
	"the", "and", "this", "you", "any", "that", "is", "word", "for", "as",
	"not", "with", "use", "from", "are", "all", "must", "your", "without",
	"its", "may", "which", "will", "copy", "such", "under",
)
