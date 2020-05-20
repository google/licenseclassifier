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
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

type scenario struct {
	expected []string
	data     string
}

// TODO(wcn): refactor some of this into a helper constructor for the classifier.
func LoadLicenses(c *Corpus, dir string, t *testing.T) error {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !strings.HasSuffix(path, "txt") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("encountered error walking licenses directory: %v", err)
	}

	for _, f := range files {
		_, name := path.Split(f)
		name = strings.Replace(name, ".txt", "", 1)
		b, err := ioutil.ReadFile(f)
		if err != nil {
			t.Fatalf("encountered error reading license file %v: %v", f, err)
		}

		c.AddContent(name, string(b))
	}

	return nil
}

func TestScenarios(t *testing.T) {
	c := NewCorpus(.8)
	licenseDir := "./licenses"
	LoadLicenses(c, licenseDir, t)

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

	if err != nil {
		t.Fatalf("encountered error walking scenarios directory: %v", err)
	}

	for _, f := range files {
		s := readScenario(f)

		m := c.Match(s.data)
		if len(m) != len(s.expected) {
			t.Errorf("Match(%q) number matches: %v, want %v: %v", f, len(m), len(s.expected), spew.Sdump(m))
			continue
		}

		for i := 0; i < len(m); i++ {
			w := s.expected[i]
			if got, want := m[i].Name, w; got != want {
				t.Errorf("Match(%q) = %q, want %q", f, got, want)
			}
		}
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
	s.data = lines[1]
	return &s
}
