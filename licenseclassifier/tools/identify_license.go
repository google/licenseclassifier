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

// The identify_license program tries to identify the license type of an
// unknown license. The file containing the license text is specified on the
// command line. Multiple license files can be analyzed with a single command.
// The type of the license is returned along with the confidence level of the
// match. The confidence level is between 0.0 and 1.0, with 1.0 indicating an
// exact match and 0.0 indicating a complete mismatch. The results are sorted
// by confidence level.
//
//   $ identifylicense LICENSE1 LICENSE2
//   LICENSE2: MIT (confidence: 0.987)
//   LICENSE1: BSD-2-Clause (confidence: 0.833)
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/licenseclassifier/licenseclassifier"
	"github.com/google/licenseclassifier/licenseclassifier/internal/commentparser"
	"github.com/google/licenseclassifier/licenseclassifier/internal/commentparser/language"
)

var (
	forbiddenOnly = flag.Bool("forbidden", false, "identify using forbidden licenses archive")
	threshold     = flag.Float64("threshold", licenseclassifier.DefaultConfidenceThreshold, "confidence threshold")
	headers       = flag.Bool("headers", false, "match license headers")
)

// licenseType is the assumed type of the unknown license.
type licenseType struct {
	filename   string
	name       string
	confidence float64
	offset     int
	extent     int
}

type licenseTypes []*licenseType

func (lt licenseTypes) Len() int      { return len(lt) }
func (lt licenseTypes) Swap(i, j int) { lt[i], lt[j] = lt[j], lt[i] }
func (lt licenseTypes) Less(i, j int) bool {
	if lt[i].confidence > lt[j].confidence {
		return true
	}
	if lt[i].confidence < lt[j].confidence {
		return false
	}
	return lt[i].filename < lt[j].filename
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s <licensefile> ...

Identify an unknown license.

Options:
`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	var lc *licenseclassifier.License
	var err error
	if *forbiddenOnly {
		lc, err = licenseclassifier.NewWithForbiddenLicenses(*threshold)
	} else {
		lc, err = licenseclassifier.New(*threshold)
	}
	if err != nil {
		log.Fatalf("cannot create license classifier: %v", err)
	}

	var mu sync.Mutex
	var matches licenseTypes

	// Create a pool from which tasks can later be started. We use a pool because the OS limits
	// the number of files that can be open at one time.
	const numTasks = 1000
	task := make(chan bool, numTasks)
	for i := 0; i < numTasks; i++ {
		task <- true
	}

	var wg sync.WaitGroup
	classifyLicense := func(filename string) {
		defer func() {
			wg.Done()
			task <- true
		}()

		contents, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatalf("cannot read %q: %v", filename, err)
			return
		}

		start := time.Now()
		if lang := language.ClassifyLanguage(filename); lang == language.Unknown {
			log.Printf("Classifying license(s): %s", filename)
			for _, m := range lc.MultipleMatch(string(contents), *headers) {
				mu.Lock()
				matches = append(matches, &licenseType{
					filename:   filename,
					name:       m.Name,
					confidence: m.Confidence,
					offset:     m.Offset,
					extent:     m.Extent,
				})
				mu.Unlock()
			}
		} else {
			comments := commentparser.Parse(contents, lang)
			for ch := range comments.ChunkIterator() {
				for _, m := range lc.MultipleMatch(ch.String(), *headers) {
					mu.Lock()
					matches = append(matches, &licenseType{
						filename:   filename,
						name:       m.Name,
						confidence: m.Confidence,
						offset:     m.Offset,
						extent:     m.Extent,
					})
					mu.Unlock()
				}
			}
		}

		log.Printf("Finished Classifying License %q: %v", filename, time.Since(start))
	}

	for _, unknown := range flag.Args() {
		wg.Add(1)
		<-task
		go classifyLicense(unknown)
	}
	wg.Wait()

	if len(matches) == 0 {
		log.Fatalf("Couldn't classify license(s)")
	}

	sort.Sort(matches)
	for _, r := range matches {
		fmt.Printf("%s: %s (confidence: %v, offset: %v, extent: %v)\n",
			r.filename, r.name, r.confidence, r.offset, r.extent)
	}
}
