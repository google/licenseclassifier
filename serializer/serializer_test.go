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
package serializer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/google/licenseclassifier"
	"github.com/google/licenseclassifier/stringclassifier/searchset"
)

var (
	apache20Header, mit []byte
	normApache, normMIT string
)

func TestMain(m *testing.M) {
	var err error
	apache20Header, err = licenseclassifier.ReadLicenseFile("Apache-2.0.header.txt")
	if err != nil {
		log.Fatalf("error reading contents of Apache-2.0.header.txt: %v", err)
	}
	normApache = normalize(string(apache20Header))

	mit, err = licenseclassifier.ReadLicenseFile("MIT.txt")
	if err != nil {
		log.Fatalf("error reading contents of MIT.txt: %v", err)
	}
	normMIT = normalize(string(mit))

	os.Exit(m.Run())
}

type entry struct {
	name     string
	size     int64
	contents string
}

func TestSerializer_ArchiveLicense(t *testing.T) {
	tests := []struct {
		description string
		licenses    []string
		want        []entry
	}{
		{
			description: "Archiving Apache 2.0 header",
			licenses:    []string{"Apache-2.0.header.txt"},
			want: []entry{
				{
					name:     "Apache-2.0.header.txt",
					size:     int64(len(normApache)),
					contents: normApache,
				},
			},
		},
		{
			description: "Archiving Apache 2.0 header + MIT",
			licenses:    []string{"Apache-2.0.header.txt", "MIT.txt"},
			want: []entry{
				{
					name:     "Apache-2.0.header.txt",
					size:     int64(len(normApache)),
					contents: normApache,
				},
				{
					name:     "MIT.txt",
					size:     int64(len(normMIT)),
					contents: normMIT,
				},
			},
		},
	}

	for _, tt := range tests {
		var writer bytes.Buffer
		if err := ArchiveLicenses(tt.licenses, &writer); err != nil {
			t.Errorf("ArchiveLicenses(%q): cannot archive license: %v", tt.description, err)
			continue
		}

		reader := bytes.NewReader(writer.Bytes())
		gr, err := gzip.NewReader(reader)
		if err != nil {
			t.Errorf("ArchiveLicenses(%q): cannot create gzip reader: %v", tt.description, err)
			continue
		}

		tr := tar.NewReader(gr)
		for i := 0; ; i++ {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("ArchiveLicenses(%q): cannot read header: %v", tt.description, err)
				break
			}

			if i >= len(tt.want)+1 {
				t.Errorf("ArchiveLicenses(%q): too many files in tar, %d want %d", tt.description, i, len(tt.want))
				break
			}

			if hdr.Name != tt.want[i].name {
				t.Errorf("ArchiveLicenses(%q) = %+v, want %+v", tt.description, hdr.Name, tt.want[i].name)
			}
			if hdr.Size != tt.want[i].size {
				t.Errorf("ArchiveLicenses(%q) = %v, want %v", tt.description, hdr.Size, tt.want[i].size)
			}

			var b bytes.Buffer
			if _, err = io.Copy(&b, tr); err != nil {
				t.Errorf("ArchiveLicenses(%q): cannot read contents: %v", tt.description, err)
				break
			}

			if got, want := b.String(), tt.want[i].contents; got != want {
				t.Errorf("ArchiveLicenses(%q) = got\n%s\nwant:\n%s", tt.description, got, want)
			}

			hdr, err = tr.Next()
			if err != nil {
				t.Errorf("ArchiveLicenses(%q): no hash file found in archive: %v", tt.description, err)
				break
			}

			if hdr.Name != strings.TrimSuffix(tt.want[i].name, "txt")+"hash" {
				t.Errorf("ArchiveLicenses(%q) = %+v, want %+v", tt.description, hdr.Name, strings.TrimSuffix(tt.want[i].name, "txt")+"hash")
			}

			b.Reset()
			if _, err = io.Copy(&b, tr); err != nil {
				t.Errorf("ArchiveLicenses(%q): cannot read contents: %v", tt.description, err)
				break
			}

			var got searchset.SearchSet
			if err := searchset.Deserialize(&b, &got); err != nil {
				t.Errorf("ArchiveLicenses(%q): cannot deserialize search set: %v", tt.description, err)
				break
			}

			want := searchset.New(tt.want[i].contents, searchset.DefaultGranularity)
			if err := compareSearchSets(want, &got); err != nil {
				t.Errorf("ArchiveLicenses(%q): search sets not equal: %v", tt.description, err)
				break
			}
		}
	}
}

type sortUInt32 []uint32

func (s sortUInt32) Len() int           { return len(s) }
func (s sortUInt32) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s sortUInt32) Less(i, j int) bool { return s[i] < s[j] }

func compareSearchSets(x, y *searchset.SearchSet) error {
	// Check to see that the tokens are equal.
	if len(x.Tokens) != len(y.Tokens) {
		return fmt.Errorf("Lengths differ = %d vs %d", len(x.Tokens), len(y.Tokens))
	}
	for i := 0; i < len(x.Tokens); i++ {
		if x.Tokens[i].Text != y.Tokens[i].Text {
			return fmt.Errorf("Token values at %d differ = %q vs %q", i, x.Tokens[i].Text, y.Tokens[i].Text)
		}
		if x.Tokens[i].Offset != y.Tokens[i].Offset {
			return fmt.Errorf("Token offsets at %d differ = %d vs %d", i, x.Tokens[i].Offset, y.Tokens[i].Offset)
		}
	}

	// Now check that the hash maps are equal.
	var xKeys []uint32
	for k := range x.Hashes {
		xKeys = append(xKeys, k)
	}
	var yKeys []uint32
	for k := range y.Hashes {
		yKeys = append(yKeys, k)
	}

	if len(xKeys) != len(yKeys) {
		return fmt.Errorf("Lengths of hashes differ = %d vs %d", len(xKeys), len(yKeys))
	}

	sort.Sort(sortUInt32(xKeys))
	sort.Sort(sortUInt32(yKeys))

	for i := 0; i < len(xKeys); i++ {
		if xKeys[i] != yKeys[i] {
			return fmt.Errorf("Hash keys differ = %d vs %d", xKeys[i], yKeys[i])
		}
		if !reflect.DeepEqual(x.Hashes[xKeys[i]], y.Hashes[yKeys[i]]) {
			return fmt.Errorf("Hash values differ = %v vs %v", x.Hashes[xKeys[i]], y.Hashes[yKeys[i]])
		}
	}

	return nil
}

func normalize(s string) string {
	for _, n := range licenseclassifier.Normalizers {
		s = n(s)
	}
	return s
}
