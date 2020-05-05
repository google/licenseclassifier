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
			*traceLicensesFlag = test.licFlag
			*tracePhasesFlag = test.phaseFlag
			initTrace()
			if !cmp.Equal(traceLicenses, test.expectedLics) {
				t.Errorf("got %v want %v", traceLicenses, test.expectedLics)
			}
			if !cmp.Equal(tracePhases, test.expectedPhases) {
				t.Errorf("got %v want %v", traceLicenses, test.expectedPhases)
			}
		})
	}
}
