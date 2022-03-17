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

// Package results contains the result type returned by the classifier backend.
// Placing the type into a separate module allows us to swap out backends and
// still use the same datatype.
package results

import (
	"bufio"
	"fmt"
	"os"
	"sort"
)

// LicenseType is the assumed type of the unknown license.
type LicenseType struct {
	Filename   string
	Name       string
	MatchType  string
	Variant    string
	Confidence float64
	StartLine  int
	EndLine    int
}

// LicenseTypes is a list of LicenseType objects.
type LicenseTypes []*LicenseType

func (lt LicenseTypes) Len() int      { return len(lt) }
func (lt LicenseTypes) Swap(i, j int) { lt[i], lt[j] = lt[j], lt[i] }
func (lt LicenseTypes) Less(i, j int) bool {
	if lt[i].Confidence > lt[j].Confidence {
		return true
	}
	if lt[i].Confidence < lt[j].Confidence {
		return false
	}
	if lt[i].Filename < lt[j].Filename {
		return true
	}
	if lt[i].Filename > lt[j].Filename {
		return false
	}
	return lt[i].EndLine < lt[j].EndLine
}

// Classification is the license classification for a segment of a file.
type Classification struct {
	Name       string
	Confidence float64
	StartLine  int
	EndLine    int
	Text       string `json:",omitempty"`
}

// Classifications contains all license classifications for a file
type Classifications []*Classification

// FileClassifications contains the license classifications for a particular file.
type FileClassifications struct {
	Filepath        string
	Classifications Classifications
}

// JSONResult is the format for the jr JSON file
type JSONResult []*FileClassifications

func (jr JSONResult) Len() int           { return len(jr) }
func (jr JSONResult) Swap(i, j int)      { jr[i], jr[j] = jr[j], jr[i] }
func (jr JSONResult) Less(i, j int) bool { return jr[i].Filepath < jr[j].Filepath }

// readFileLines will read a specified range of lines of a file
func readFileLines(filename string, startLine, endLine int) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lines := ""
	i := 0
	for scanner.Scan() {
		i++ // lines are 1-indexed
		if i < startLine {
			continue
		} else if i > endLine {
			break
		}
		lines += scanner.Text() + "\n"
	}
	if i < endLine {
		return "", fmt.Errorf(
			"line %d was the last line read from file %s, but endLine was set to %d", i, filename, endLine)
	}
	return lines, nil
}

// NewJSONResult creates a new JSONResult object from a LicenseTypes object.
func NewJSONResult(licenses LicenseTypes, includeText bool) (JSONResult, error) {
	fMap := map[string]*FileClassifications{}
	for _, l := range licenses {
		currF, ok := fMap[l.Filename]
		if !ok {
			currF = &FileClassifications{Filepath: l.Filename}
			fMap[l.Filename] = currF
		}
		c := &Classification{
			Name:       l.Name,
			Confidence: l.Confidence,
			StartLine:  l.StartLine,
			EndLine:    l.EndLine,
		}
		if includeText {
			text, err := readFileLines(l.Filename, l.StartLine, l.EndLine)
			if err != nil {
				return nil, err
			}
			c.Text = text
		}
		currF.Classifications = append(currF.Classifications, c)
	}

	jr := JSONResult{}
	for _, fc := range fMap {
		jr = append(jr, fc)
	}
	sort.Sort(jr)
	return jr, nil
}
