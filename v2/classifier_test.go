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
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/davecgh/go-spew/spew"
)

type scenario struct {
	expected []string
	data     []byte
}

var defaultThreshold = .8
var baseLicenses = "./licenses"

func classifier() (*Classifier, error) {
	c := NewClassifier(defaultThreshold)
	return c, c.LoadLicenses(baseLicenses)
}

func getScenarioFilenames() ([]string, error) {
	scenarios := "./scenarios"
	var files []string
	err := filepath.Walk(scenarios, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "md") || info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})

	return files, err
}

func TestMatchScenarios(t *testing.T) {
	c, err := classifier()
	if err != nil {
		t.Fatalf("couldn't instantiate standard test classifier: %v", err)
	}

	files, err := getScenarioFilenames()
	if err != nil {
		t.Fatalf("encountered error walking scenarios directory: %v", err)
	}

	for _, f := range files {
		s := readScenario(f)

		m := c.Match(s.data)
		checkMatches(t, m, f, s.expected)
	}
}

func readScenario(path string) *scenario {
	var s scenario
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Couldn't read scenario %s: %v", path, err)
	}

	// A scenario consists of any number of comment lines, which are ignored, then a line of the form
	// EXPECTED: A,B,C
	//
	// or EXPECTED:<EOL>
	// where A,B,C is a comma-separated list of expected licenses.
	lines := strings.SplitN(string(b), "EXPECTED:", 2)
	// The first part of lines is description, which we ignore. We then split on a linefeed to get the
	// list of licenses and the rest of the data content.
	lines = strings.SplitN(lines[1], "\n", 2)
	if lines[0] != "" {
		s.expected = strings.Split(lines[0], ",")
	} else {
		s.expected = []string{}
	}
	s.data = []byte(lines[1])
	return &s
}

func TestContainsAndOverlaps(t *testing.T) {
	tests := []struct {
		name     string
		a, b     *Match
		contains bool
		overlaps bool
	}{
		{
			name: "no intersection",
			a: &Match{
				StartLine: 1,
				EndLine:   3,
			},
			b: &Match{
				StartLine: 4,
				EndLine:   5,
			},
			contains: false,
			overlaps: false,
		},
		{
			name: "overlap at end",
			a: &Match{
				StartLine: 4,
				EndLine:   10,
			},
			b: &Match{
				StartLine: 1,
				EndLine:   5,
			},
			contains: false,
			overlaps: true,
		},
		{
			name: "overlap at end",
			a: &Match{
				StartLine: 1,
				EndLine:   10,
			},
			b: &Match{
				StartLine: 4,
				EndLine:   12,
			},
			contains: false,
			overlaps: true,
		},
		{
			name: "contains",
			a: &Match{
				StartLine: 1,
				EndLine:   10,
			},
			b: &Match{
				StartLine: 4,
				EndLine:   7,
			},
			contains: true,
			overlaps: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := contains(test.a, test.b); got != test.contains {
				t.Errorf("contains: got %v want %v", got, test.contains)
			}
			if got := overlaps(test.a, test.b); got != test.overlaps {
				t.Errorf("overlaps: got %v want %v", got, test.overlaps)
			}
		})
	}
}

func TestLicName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			// The filename for a license
			name:     "GPL-2.0.txt",
			expected: "GPL-2.0",
		},
		{
			// The filename for a header reference to a license
			name:     "GPL-2.0.header.txt",
			expected: "GPL-2.0",
		},
		{
			// The filename for a variant header reference to a license
			name:     "GPL-2.0.header_a.txt",
			expected: "GPL-2.0",
		},
		{
			// The filename for a variant license body
			name:     "Apache-2.0_no_toc.txt",
			expected: "Apache-2.0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

		})
	}
}

func TestTrimExtraneousText(t *testing.T) {
	in := `Sample text
END OF TERMS AND CONDITIONS
Extra text is here`

	expected := "Sample text\n"
	if got := trimExtraneousTrailingText(in); got != expected {
		t.Errorf("trimExtraneousTrailingText: got %q want %q", got, expected)
	}
}

func TestMatchFrom(t *testing.T) {
	tr := iotest.TimeoutReader(strings.NewReader("some data"))
	c, err := classifier()
	if err != nil {
		t.Fatalf("couldn't instantiate standard Google classifier: %v", err)
	}

	_, err = c.MatchFrom(tr)
	if !errors.Is(err, iotest.ErrTimeout) {
		t.Errorf("got %v want %v", err, iotest.ErrTimeout)
	}

	files, err := getScenarioFilenames()

	if err != nil {
		t.Fatalf("encountered error walking scenarios directory: %v", err)
	}

	for _, f := range files {
		s := readScenario(f)
		r := bytes.NewReader(s.data)
		m, err := c.MatchFrom(r)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		checkMatches(t, m, f, s.expected)
	}
}

// checkMatches diffs the resulting matches against the expected content and
// sets test results.
func checkMatches(t *testing.T, m Matches, f string, e []string) {
	found := make(map[string]bool)
	// Uniquify the licenses found
	for _, l := range m {
		found[l.Name] = true
	}

	var names []string
	for l := range found {
		names = append(names, l)
	}
	sort.Strings(names)

	if len(names) != len(e) {
		t.Errorf("Match(%q) number matches: %v, want %v: %v", f, len(names), len(e), spew.Sdump(m))
		return
	}

	for i := 0; i < len(names); i++ {
		w := strings.TrimSpace(e[i])
		if got, want := names[i], w; got != want {
			t.Errorf("Match(%q) = %q, want %q", f, got, want)
		}
	}
}
