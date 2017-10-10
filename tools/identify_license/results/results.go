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

// LicenseType is the assumed type of the unknown license.
type LicenseType struct {
	Filename   string
	Name       string
	Confidence float64
	Offset     int
	Extent     int
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
	return lt[i].Filename < lt[j].Filename
}
