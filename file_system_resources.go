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

package licenseclassifier

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	// LicenseDirectory is the directory where the prototype licenses are kept.
	LicenseDirectory = "src/github.com/google/licenseclassifier/licenses"
	// LicenseArchive is the name of the archive containing preprocessed
	// license texts.
	LicenseArchive = "licenses.db"
	// ForbiddenLicenseArchive is the name of the archive containing preprocessed
	// forbidden license texts only.
	ForbiddenLicenseArchive = "forbidden_licenses.db"
)

// ReadLicenseFile locates and reads the license file.
func ReadLicenseFile(filename string) ([]byte, error) {
	for _, path := range filepath.SplitList(os.Getenv("GOPATH")) {
		archive := filepath.Join(path, LicenseDirectory, filename)
		if _, err := os.Stat(archive); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		return ioutil.ReadFile(archive)
	}
	return nil, nil
}

// ReadLicenseDir reads directory containing the license files.
func ReadLicenseDir() ([]os.FileInfo, error) {
	for _, path := range filepath.SplitList(os.Getenv("GOPATH")) {
		dir := filepath.Join(path, LicenseDirectory)
		filename := filepath.Join(dir, LicenseArchive)
		if _, err := os.Stat(filename); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		return ioutil.ReadDir(dir)
	}
	return nil, nil
}
