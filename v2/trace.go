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
	"flag"
	"fmt"
	"strings"
)

// This file contains routines for a simple trace execution mechanism.
//
// The constant map lookups do incur some overhead and could be optimized. One possible approach
// would be to sample the values at the time Match() is called and then store the results in a cached
// format. This would have to be done in a threadsafe manner.
var traceLicensesFlag = flag.String("trace_licenses", "", "comma-separated list of licenses for tracing")
var tracePhasesFlag = flag.String("trace_phases", "", "comma-separated list of licenses for tracing")

func initTrace() {
	// Sample the command line flags and set the tracing variables
	traceLicenses = make(map[string]bool)
	tracePhases = make(map[string]bool)

	if len(*traceLicensesFlag) > 0 {
		for _, lic := range strings.Split(*traceLicensesFlag, ",") {
			traceLicenses[lic] = true
		}
	}

	if len(*tracePhasesFlag) > 0 {
		for _, phase := range strings.Split(*tracePhasesFlag, ",") {
			tracePhases[phase] = true
		}
	}
}

var traceLicenses map[string]bool
var tracePhases map[string]bool

func shouldTrace(phase string) bool {
	return tracePhases[phase]
}

func isTraceLicense(lic string) bool {
	return traceLicenses[lic]
}

func traceSearchset(lic string) bool {
	return traceLicenses[lic] && shouldTrace("searchset")
}

func traceTokenize(lic string) bool {
	return traceLicenses[lic] && shouldTrace("tokenize")
}

func traceScoring(lic string) bool {
	return traceLicenses[lic] && shouldTrace("score")
}

func traceFrequency(lic string) bool {
	return traceLicenses[lic] && shouldTrace("frequency")
}

type traceFunc func(string, ...interface{}) (int, error)

// Trace holds the function that should be called to emit data. This can be overridden as desired,
// defaulting to output on stdout.
var Trace traceFunc = fmt.Printf
