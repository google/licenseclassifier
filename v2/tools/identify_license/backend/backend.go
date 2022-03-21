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

// Package backend contains the necessary functions to classify a license.
package backend

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	//gc "google3/devtools/compliance/common/licenseclassifier/classifier"

	classifier "github.com/google/licenseclassifier/v2"
	"github.com/google/licenseclassifier/v2/assets"
	"github.com/google/licenseclassifier/v2/tools/identify_license/results"
)

// ClassifierInterface is the interface each backend must implement.
type ClassifierInterface interface {
	Close()
	SetTraceConfiguration(tc *classifier.TraceConfiguration)
	ClassifyLicenses(numTasks int, filenames []string, headers bool) []error
	ClassifyLicensesWithContext(ctx context.Context, numTasks int, filenames []string, headers bool) []error
	GetResults() results.LicenseTypes
}

// ClassifierBackend is an object that handles classifying a license.
type ClassifierBackend struct {
	results    results.LicenseTypes
	mu         sync.Mutex
	classifier *classifier.Classifier
}

// New creates a new backend working on the local filesystem.
func New() (*ClassifierBackend, error) {
	_, err := assets.ReadLicenseDir()
	if err != nil {
		return nil, err
	}
	lc, err := assets.DefaultClassifier()
	if err != nil {
		return nil, err
	}
	return &ClassifierBackend{classifier: lc}, nil
}

// Close does nothing here since there's nothing to close.
func (b *ClassifierBackend) Close() {
}

// SetTraceConfiguration injects the supplied trace configuration
func (b *ClassifierBackend) SetTraceConfiguration(tc *classifier.TraceConfiguration) {
	//b.classifier.SetTraceConfiguration((*gc.TraceConfiguration)(tc))
}

// ClassifyLicenses runs the license classifier over the given file.
func (b *ClassifierBackend) ClassifyLicenses(numTasks int, filenames []string, headers bool) (errors []error) {
	// Create a pool from which tasks can later be started. We use a pool because the OS limits
	// the number of files that can be open at any one time.
	task := make(chan bool, numTasks)
	for i := 0; i < numTasks; i++ {
		task <- true
	}

	errs := make(chan error, len(filenames))

	var wg sync.WaitGroup
	analyze := func(filename string) {
		defer func() {
			wg.Done()
			task <- true
		}()
		if err := b.classifyLicense(filename, headers); err != nil {
			errs <- err
		}
	}

	for _, filename := range filenames {
		wg.Add(1)
		<-task
		go analyze(filename)
	}
	go func() {
		wg.Wait()
		close(task)
		close(errs)
	}()

	for err := range errs {
		errors = append(errors, err)
	}
	return errors
}

// ClassifyLicensesWithContext runs the license classifier over the given file; ensure that it will respect the timeout in the provided context.
func (b *ClassifierBackend) ClassifyLicensesWithContext(ctx context.Context, numTasks int, filenames []string, headers bool) (errors []error) {
	done := make(chan bool)
	go func() {
		errors = b.ClassifyLicenses(numTasks, filenames, headers)
		done <- true
	}()
	select {
	case <-ctx.Done():
		err := ctx.Err()
		errors = append(errors, err)
		return errors
	case <-done:
		return errors
	}
}

// classifyLicense is called by a Go-function to perform the actual
// classification of a license.
func (b *ClassifierBackend) classifyLicense(filename string, headers bool) error {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("unable to read %q: %v", filename, err)
	}

	matchLoop := func(contents []byte) {
		for _, m := range b.classifier.Match(contents).Matches {
			// If not looking for headers, skip them
			if !headers && m.MatchType == "Header" {
				continue
			}

			b.mu.Lock()
			b.results = append(b.results, &results.LicenseType{
				Filename:   filename,
				MatchType:  m.MatchType,
				Name:       m.Name,
				Variant:    m.Variant,
				Confidence: m.Confidence,
				StartLine:  m.StartLine,
				EndLine:    m.EndLine,
			})
			b.mu.Unlock()
		}
	}

	log.Printf("Classifying license(s): %s", filename)
	start := time.Now()
	matchLoop(contents)
	log.Printf("Finished Classifying License %q: %v", filename, time.Since(start))
	return nil
}

// GetResults returns the results of the classifications.
func (b *ClassifierBackend) GetResults() results.LicenseTypes {
	return b.results
}
