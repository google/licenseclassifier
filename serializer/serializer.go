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

// Package serializer normalizes the license text and calculates the hash
// values for all substrings in the license. It then outputs the normalized
// text and hashes to disk in a compressed archive.
package serializer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/google/licenseclassifier"
	"github.com/google/licenseclassifier/stringclassifier/searchset"
)

// ArchiveLicenses takes all of the known license texts, normalizes them, then
// calculates the hash values of all substrings. The resulting normalized text
// and hashed substring values are then serialized into an archive file.
func ArchiveLicenses(licenses []string, w io.Writer) error {
	gw := gzip.NewWriter(w)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	for _, license := range licenses {
		// All license files have a ".txt" extension.
		ext := filepath.Ext(license)
		if ext != ".txt" {
			continue
		}

		contents, err := licenseclassifier.ReadLicenseFile(license)
		if err != nil {
			return err
		}

		str := licenseclassifier.TrimExtraneousTrailingText(string(contents))
		for _, n := range licenseclassifier.Normalizers {
			str = n(str)
		}

		baseName := strings.TrimSuffix(license, ext)

		// Serialize the normalized license text.
		log.Printf("Serializing %q", baseName)
		hdr := &tar.Header{
			Name: license,
			Mode: 0644,
			Size: int64(len(str)),
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(str)); err != nil {
			return err
		}

		// Calculate the substrings' checksums
		set := searchset.New(str, searchset.DefaultGranularity)

		var s bytes.Buffer
		if err := set.Serialize(&s); err != nil {
			return err
		}

		// Serialize the checksums.
		hdr = &tar.Header{
			Name: baseName + ".hash",
			Mode: 0644,
			Size: int64(s.Len()),
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(s.Bytes()); err != nil {
			return err
		}
	}

	return tw.Close()
}
