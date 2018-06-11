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
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/licenseclassifier"
	"github.com/google/licenseclassifier/tools/identify_license/backend"
)

var (
	headers       = flag.Bool("headers", false, "match license headers")
	forbiddenOnly = flag.Bool("forbidden", false, "identify using forbidden licenses archive")
	threshold     = flag.Float64("threshold", licenseclassifier.DefaultConfidenceThreshold, "confidence threshold")
	timeout       = flag.Duration("timeout", 24*time.Hour, "timeout before giving up on classifying a file.")
)

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

	be, err := backend.New(*threshold, *forbiddenOnly)
	if err != nil {
		be.Close()
		log.Fatalf("cannot create license classifier: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if errs := be.ClassifyLicensesWithContext(ctx, flag.Args(), *headers); errs != nil {
		be.Close()
		for _, err := range errs {
			log.Printf("classify license failed: %v", err)
		}
		log.Fatal("cannot classify licenses")
	}

	results := be.GetResults()
	if len(results) == 0 {
		be.Close()
		log.Fatal("Couldn't classify license(s)")
	}

	sort.Sort(results)
	for _, r := range results {
		fmt.Printf("%s: %s (confidence: %v, offset: %v, extent: %v)\n",
			r.Filename, r.Name, r.Confidence, r.Offset, r.Extent)
	}
	be.Close()
}
