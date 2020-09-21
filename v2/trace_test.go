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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInitTrace(t *testing.T) {
	tests := []struct {
		name, licFlag, phaseFlag string
		expectedLics             map[string]bool
		expectedPhases           map[string]bool
	}{
		{
			name:           "empty flags",
			licFlag:        "",
			phaseFlag:      "",
			expectedLics:   map[string]bool{},
			expectedPhases: map[string]bool{},
		},
		{
			name:           "single entries",
			licFlag:        "one_license",
			phaseFlag:      "setup",
			expectedLics:   map[string]bool{"one_license": true},
			expectedPhases: map[string]bool{"setup": true},
		},
		{
			name:           "multiple entries",
			licFlag:        "one_license,two_license",
			phaseFlag:      "setup,teardown",
			expectedLics:   map[string]bool{"one_license": true, "two_license": true},
			expectedPhases: map[string]bool{"setup": true, "teardown": true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := &TraceConfiguration{
				TraceLicenses: test.licFlag,
				TracePhases:   test.phaseFlag,
			}
			tc.init()
			if !cmp.Equal(tc.traceLicenses, test.expectedLics) {
				t.Errorf("got %v want %v", traceLicenses, test.expectedLics)
			}
			if !cmp.Equal(tc.tracePhases, test.expectedPhases) {
				t.Errorf("got %v want %v", traceLicenses, test.expectedPhases)
			}
		})
	}
}

func TestPhaseWildcardMatching(t *testing.T) {
	tests := []struct {
		name   string
		phases string
		hits   []string
		misses []string
	}{
		{
			name:   "exact match",
			phases: "scoring",
			hits:   []string{"scoring"},
			misses: []string{"tokenize"},
		},
		{
			name:   "all match",
			phases: "*",
			hits:   []string{"scoring", "tokenize"},
			misses: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := &TraceConfiguration{
				TracePhases: test.phases,
			}
			tc.init()
			for _, h := range test.hits {
				if !tc.shouldTrace(h) {
					t.Errorf("unexpected miss on phase %s", h)
				}
			}

			for _, m := range test.misses {
				if tc.shouldTrace(m) {
					t.Errorf("unexpected hit on phase %s", m)
				}
			}
		})
	}
}

func TestLicenseWildcardMatching(t *testing.T) {
	tests := []struct {
		name     string
		licenses string
		hits     []string
		misses   []string
	}{
		{
			name:     "exact match",
			hits:     []string{"GPL-2.0"},
			misses:   []string{"Apache-2.0", "GPL-3.0"},
			licenses: "GPL-2.0",
		},
		{
			name:     "prefix match",
			hits:     []string{"GPL-2.0", "GPL-3.0"},
			misses:   []string{"Apache-2.0"},
			licenses: "GPL-*",
		},
		{
			name:     "all match",
			hits:     []string{"GPL-2.0", "GPL-3.0", "Apache-2.0"},
			misses:   nil,
			licenses: "*",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := &TraceConfiguration{
				TraceLicenses: test.licenses,
			}
			tc.init()
			for _, h := range test.hits {
				if !tc.isTraceLicense(h) {
					t.Errorf("unexpected miss on license %s", h)
				}
			}

			for _, m := range test.misses {
				if tc.isTraceLicense(m) {
					t.Errorf("unexpected hit on license %s", m)
				}
			}
		})
	}
}

// The TraceConfiguration is only explicitly initialized and propagated to a
// variety of helper structs. For convenience, we just make it work safely in
// the case the pointer is nil. This test ensures that behavior so users of the
// TraceConfiguration don't need to explicitly initialize it.
func TestNilSafety(t *testing.T) {
	var tc *TraceConfiguration
	tc.init()
	if tc.isTraceLicense("GPL-2.0") {
		t.Errorf("unexpected hit on license")
	}

	if tc.shouldTrace("scoring") {
		t.Errorf("unexpected hit on phase")
	}
}
