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
//	$ identifylicense <LICENSE_OR_DIRECTORY>  <LICENSE_OR_DIRECTORY> ...
//	LICENSE2: MIT (confidence: 0.987)
//	LICENSE1: BSD-2-Clause (confidence: 0.833)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	//"google3/file/base/go/contrib/walk/walk"
	//"google3/file/base/go/file"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	classifier "github.com/google/licenseclassifier/v2"
	"github.com/google/licenseclassifier/v2/tools/identify_license/backend"
	"github.com/google/licenseclassifier/v2/tools/identify_license/results"
)

var (
	headers       = flag.Bool("headers", false, "match license headers")
	jsonFname     = flag.String("json", "", "filename to write JSON output to.")
	includeText   = flag.Bool("include_text", false, "include the license text in the JSON output")
	numTasks      = flag.Int("tasks", 1000, "the number of license scanning tasks running concurrently")
	timeout       = flag.Duration("timeout", 24*time.Hour, "timeout before giving up on classifying a file.")
	tracePhases   = flag.String("trace_phases", "", "comma-separated list of phases of the license classifier to trace")
	traceLicenses = flag.String("trace_licenses", "", "comma-separated list of licenses for the license classifier to trace")
	ignorePaths   = flag.String("ignore_paths_re", "", "comma-separated list of regular expressions that match file paths to ignore")
)

// expandFiles recursively returns a list of files stored in a list of
// directories. If an input is not a directory, it is added to the output list.
func expandFiles(ctx context.Context, paths []string) ([]string, error) {
	var finalPaths []string

	ip, err := parseIgnorePaths()
	if err != nil {
		return nil, fmt.Errorf("could not parse ignore paths: %v", err)
	}

	handleFile := func(path string) {
		if shouldIgnore(ip, path) {
			return
		}
		finalPaths = append(finalPaths, path)
	}

	for _, p := range paths {
		p, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}

		err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if shouldIgnore(ip, info.Name()) {
					return fs.SkipDir
				}
				return nil // walk the directory
			}
			handleFile(path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return finalPaths, nil
}

func shouldIgnore(ignorePaths []*regexp.Regexp, path string) bool {
	for _, r := range ignorePaths {
		if exactRegexMatch(r, path) {
			return true
		}
	}
	return false
}

func exactRegexMatch(r *regexp.Regexp, s string) bool {
	m := r.FindStringIndex(s)
	if m == nil {
		return false
	}
	return (m[0] == 0) && (m[1] == len(s))
}

func parseIgnorePaths() (out []*regexp.Regexp, err error) {
	for _, p := range strings.Split(*ignorePaths, ",") {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// outputJSON writes the output formatted as JSON to a file.
func outputJSON(filename *string, res results.LicenseTypes, includeText bool) error {
	d, err := results.NewJSONResult(res, includeText)
	if err != nil {
		return err
	}
	fc, err := json.MarshalIndent(d, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(*filename, fc, 0644)
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

	be, err := backend.New()
	if err != nil {
		log.Fatalf("cannot create license classifier: %v", err)
	}

	paths, err := expandFiles(context.Background(), flag.Args())
	defer be.Close()
	be.SetTraceConfiguration(
		&classifier.TraceConfiguration{
			TracePhases:   *tracePhases,
			TraceLicenses: *traceLicenses,
		})

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if errs := be.ClassifyLicensesWithContext(ctx, *numTasks, paths, *headers); errs != nil {
		be.Close()
		for _, err := range errs {
			log.Printf("classify license failed: %v", err)
		}
		log.Fatal("cannot classify licenses")
	}

	results := be.GetResults()
	if len(results) == 0 {
		log.Fatal("Couldn't classify license(s)")
	}

	sort.Sort(results)
	for _, r := range results {
		name := r.Name
		if r.MatchType != "License" && r.MatchType != "Header" {
			name = fmt.Sprintf("%s:%s", r.MatchType, r.Name)
		}
		fmt.Printf("%s %s (variant: %v, confidence: %v, start: %v, end: %v)\n",
			r.Filename, name, r.Variant, r.Confidence, r.StartLine, r.EndLine)
	}
	if len(*jsonFname) > 0 {
		err = outputJSON(jsonFname, results, *includeText)
		if err != nil {
			log.Fatalf("Couldn't write JSON output to file %s: %v", *jsonFname, err)
		}
	}
}
