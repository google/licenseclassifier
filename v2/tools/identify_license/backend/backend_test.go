// Copyright 2026 Google LLC
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

package backend

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/licenseclassifier/v2/assets"
)

func writeLicense(t *testing.T) string {
	t.Helper()
	b, err := assets.ReadLicenseFile("License/MIT/pristine.txt")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "LICENSE")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestClassifyLicensesRepeated calls ClassifyLicenses many times on one file.
// Closing the task channel used to race with the last worker returning its
// slot, which panicked with "send on closed channel". Run with -race for the
// most reliable reproduction.
func TestClassifyLicensesRepeated(t *testing.T) {
	be, err := New()
	if err != nil {
		t.Fatal(err)
	}
	path := writeLicense(t)

	for i := 0; i < 200; i++ {
		if errs := be.ClassifyLicenses(10, []string{path}, false); errs != nil {
			t.Fatalf("ClassifyLicenses() = %v", errs)
		}
	}

	found := false
	for _, r := range be.GetResults() {
		if r.Name == "MIT" {
			found = true
		}
	}
	if !found {
		t.Errorf("GetResults() did not contain MIT")
	}
}

// TestClassifyLicensesWithContextDone runs a classification with a done
// context, then a normal one on the same backend. The done run must not start
// any work or leave any running, so only the second run's single result shows
// up. This used to fail because the done run kept classifying in the
// background and its results leaked into the second run.
func TestClassifyLicensesWithContextDone(t *testing.T) {
	be, err := New()
	if err != nil {
		t.Fatal(err)
	}
	path := writeLicense(t)
	paths := make([]string, 50)
	for i := range paths {
		paths[i] = path
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	errs := be.ClassifyLicensesWithContext(ctx, 4, paths, false)
	if len(errs) != 1 || errs[0] != context.Canceled {
		t.Fatalf("ClassifyLicensesWithContext() = %v, want [%v]", errs, context.Canceled)
	}

	if errs := be.ClassifyLicenses(4, []string{path}, false); errs != nil {
		t.Fatalf("ClassifyLicenses() = %v", errs)
	}
	if got := be.GetResults(); len(got) != 1 {
		t.Errorf("GetResults() has %d results, want 1", len(got))
	}
}
