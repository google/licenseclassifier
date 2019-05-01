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
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/licenseclassifier/stringclassifier"
)

var (
	agpl30, agpl30Header, apache20, bsd3, gpl20, ccbync20 string
	classifier                                            *License
)

func TestMain(m *testing.M) {
	a30, err := ReadLicenseFile("AGPL-3.0.txt")
	if err != nil {
		log.Fatalf("error reading contents of AGPL-3.0.txt: %v", err)
	}
	a30h, err := ReadLicenseFile("AGPL-3.0.header.txt")
	if err != nil {
		log.Fatalf("error reading contents of AGPL-3.0.header.txt: %v", err)
	}
	a20, err := ReadLicenseFile("Apache-2.0.txt")
	if err != nil {
		log.Fatalf("error reading contents of Apache-2.0.txt: %v", err)
	}
	b3, err := ReadLicenseFile("BSD-3-Clause.txt")
	if err != nil {
		log.Fatalf("error reading contents of BSD-3-Clause.txt: %v", err)
	}
	g2, err := ReadLicenseFile("GPL-2.0.txt")
	if err != nil {
		log.Fatalf("error reading contents of GPL-2.0.txt: %v", err)
	}
	cc20, err := ReadLicenseFile("CC-BY-NC-2.0.txt")
	if err != nil {
		log.Fatalf("error reading contents of CC-BY-NC-2.0.txt: %v", err)
	}

	agpl30 = TrimExtraneousTrailingText(string(a30))
	agpl30Header = TrimExtraneousTrailingText(string(a30h))
	apache20 = TrimExtraneousTrailingText(string(a20))
	bsd3 = TrimExtraneousTrailingText(string(b3))
	gpl20 = TrimExtraneousTrailingText(string(g2))
	ccbync20 = TrimExtraneousTrailingText(string(cc20))

	classifier, err = New(DefaultConfidenceThreshold)
	if err != nil {
		log.Fatalf("cannot create license classifier: %v", err)
	}
	os.Exit(m.Run())
}

func TestClassifier_NearestMatch(t *testing.T) {
	tests := []struct {
		description    string
		filename       string
		extraText      string
		wantLicense    string
		wantConfidence float64
	}{
		{
			description:    "AGPL 3.0 license",
			filename:       "AGPL-3.0.txt",
			wantLicense:    "AGPL-3.0",
			wantConfidence: 1.0,
		},
		{
			description:    "Apache 2.0 license",
			filename:       "Apache-2.0.txt",
			wantLicense:    "Apache-2.0",
			wantConfidence: 1.0,
		},
		{
			description:    "GPL 2.0 license",
			filename:       "GPL-2.0.txt",
			wantLicense:    "GPL-2.0",
			wantConfidence: 1.0,
		},
		{
			description:    "BSD 3 Clause license with extra text",
			filename:       "BSD-3-Clause.txt",
			extraText:      "New BSD License\nCopyright © 1998 Yoyodyne, Inc.\n",
			wantLicense:    "BSD-3-Clause",
			wantConfidence: 0.94,
		},
	}

	classifier.Threshold = DefaultConfidenceThreshold
	for _, tt := range tests {
		content, err := ReadLicenseFile(tt.filename)
		if err != nil {
			t.Errorf("error reading contents of %q license: %v", tt.wantLicense, err)
			continue
		}

		m := classifier.NearestMatch(tt.extraText + TrimExtraneousTrailingText(string(content)))
		if got, want := m.Name, tt.wantLicense; got != want {
			t.Errorf("NearestMatch(%q) = %q, want %q", tt.description, got, want)
		}
		if got, want := m.Confidence, tt.wantConfidence; got < want {
			t.Errorf("NearestMatch(%q) = %v, want %v", tt.description, got, want)
		}
	}
}

func TestClassifier_MultipleMatch(t *testing.T) {
	tests := []struct {
		description string
		text        string
		want        stringclassifier.Matches
	}{
		{
			description: "Two licenses",
			text:        "Copyright (c) 2016 Yoyodyne, Inc.\n" + apache20 + strings.Repeat("-", 80) + "\n" + bsd3,
			want: stringclassifier.Matches{
				{
					Name:       "Apache-2.0",
					Confidence: 1.0,
				},
				{
					Name:       "BSD-3-Clause",
					Confidence: 1.0,
				},
			},
		},
		{
			description: "Two licenses: partial match",
			text: "Copyright (c) 2016 Yoyodyne, Inc.\n" +
				string(apache20[:len(apache20)/2-1]) + string(apache20[len(apache20)/2+7:]) + strings.Repeat("-", 80) + "\n" +
				string(bsd3[:len(bsd3)/2]) + "intervening stuff" + string(bsd3[len(bsd3)/2:]),
			want: stringclassifier.Matches{
				{
					Name:       "Apache-2.0",
					Confidence: 0.99,
				},
				{
					Name:       "BSD-3-Clause",
					Confidence: 0.98,
				},
			},
		},
		{
			description: "Two licenses: one forbidden the other okay",
			text:        "Copyright (c) 2016 Yoyodyne, Inc.\n" + apache20 + strings.Repeat("-", 80) + "\n" + ccbync20,
			want: stringclassifier.Matches{
				{
					Name:       "Apache-2.0",
					Confidence: 0.99,
				},
				{
					Name:       "CC-BY-NC-2.0",
					Confidence: 1.0,
				},
			},
		},
		{
			description: "Two licenses without any space between them.",
			text:        apache20 + "." + bsd3,
			want: stringclassifier.Matches{
				{
					Name:       "Apache-2.0",
					Confidence: 1.0,
				},
				{
					Name:       "BSD-3-Clause",
					Confidence: 1.0,
				},
			},
		},
	}

	classifier.Threshold = 0.95
	defer func() {
		classifier.Threshold = DefaultConfidenceThreshold
	}()
	for _, tt := range tests {
		m := classifier.MultipleMatch(tt.text, false)
		if len(m) != len(tt.want) {
			t.Fatalf("MultipleMatch(%q) number matches: %v, want %v", tt.description, len(m), len(tt.want))
			continue
		}

		for i := 0; i < len(m); i++ {
			w := tt.want[i]
			if got, want := m[i].Name, w.Name; got != want {
				t.Errorf("MultipleMatch(%q) = %q, want %q", tt.description, got, want)
			}
			if got, want := m[i].Confidence, w.Confidence; got < want {
				t.Errorf("MultipleMatch(%q) = %v, want %v", tt.description, got, want)
			}
		}
	}
}

func TestClassifier_MultipleMatch_Headers(t *testing.T) {
	tests := []struct {
		description string
		text        string
		want        stringclassifier.Matches
	}{
		{
			description: "AGPL-3.0 header",
			text:        "Copyright (c) 2016 Yoyodyne, Inc.\n" + agpl30Header,
			want: stringclassifier.Matches{
				{
					Name:       "AGPL-3.0",
					Confidence: 1.0,
					Offset:     0,
				},
			},
		},
		{
			description: "Modified LGPL-2.1 header",
			text: `Common Widget code.

Copyright (C) 2013-2015 Yoyodyne, Inc.

This library is free software; you can redistribute it and/or
modify it under the terms of the GNU Lesser General Public
License as published by the Free Software Foundation; either
version 2.1 of the License, or (at your option) any later version (but not!).

This library is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public
License along with this library; if not, write to the Free Software
Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301  USA
`,
			want: stringclassifier.Matches{
				{
					Name:       "LGPL-2.1",
					Confidence: 0.97,
					Offset:     197,
				},
			},
		},
	}

	classifier.Threshold = 0.90
	defer func() {
		classifier.Threshold = DefaultConfidenceThreshold
	}()
	for _, tt := range tests {
		m := classifier.MultipleMatch(tt.text, true)
		if len(m) != len(tt.want) {
			t.Errorf("MultipleMatch(%q) number matches: %v, want %v", tt.description, len(m), len(tt.want))
			continue
		}

		for i := 0; i < len(m); i++ {
			w := tt.want[i]
			if got, want := m[i].Name, w.Name; got != want {
				t.Errorf("MultipleMatch(%q) = %q, want %q", tt.description, got, want)
			}
			if got, want := m[i].Confidence, w.Confidence; got < want {
				t.Errorf("MultipleMatch(%q) = %v, want %v", tt.description, got, want)
			}
		}
	}
}

func TestClassifier_CopyrightHolder(t *testing.T) {
	tests := []struct {
		copyright string
		want      string
	}{
		{
			copyright: "Copyright 2008 Yoyodyne Inc. All Rights Reserved.",
			want:      "Yoyodyne Inc.",
		},
		{
			copyright: "Copyright 2010-2016 Yoyodyne, Inc.",
			want:      "Yoyodyne, Inc.",
		},
		{
			copyright: "Copyright 2010, 2011, 2012 Yoyodyne, Inc., All rights reserved.",
			want:      "Yoyodyne, Inc.",
		},
		{
			copyright: "Copyright (c) 2015 Yoyodyne, Inc. All rights reserved.",
			want:      "Yoyodyne, Inc.",
		},
		{
			copyright: "Copyright © 1998 by Yoyodyne, Inc., San Narciso, CA, US.",
			want:      "Yoyodyne, Inc., San Narciso, CA, US",
		},
		{
			copyright: "Copyright (c) 2015 The Algonquin Round Table. All rights reserved.",
			want:      "The Algonquin Round Table",
		},
		{
			copyright: "Copyright 2016, The Android Open Source Project",
			want:      "The Android Open Source Project",
		},
		{
			copyright: `---------------------------------------------------------
foo.c:
Copyright 2016, The Android Open Source Project
`,
			want: "The Android Open Source Project",
		},
	}

	for _, tt := range tests {
		got := CopyrightHolder(tt.copyright)
		if got != tt.want {
			t.Errorf("CopyrightHolder(%q) = %q, want %q", tt.copyright, got, tt.want)
		}
	}
}

func TestClassifier_WithinConfidenceThreshold(t *testing.T) {
	tests := []struct {
		description string
		text        string
		confDef     bool
		conf99      bool
		conf93      bool
		conf5       bool
	}{
		{
			description: "Apache 2.0",
			text:        apache20,
			confDef:     true,
			conf99:      true,
			conf93:      true,
			conf5:       true,
		},
		{
			description: "GPL 2.0",
			text:        gpl20,
			confDef:     true,
			conf99:      true,
			conf93:      true,
			conf5:       true,
		},
		{
			description: "BSD 3 Clause license with extra text",
			text:        "New BSD License\nCopyright © 1998 Yoyodyne, Inc.\n" + bsd3,
			confDef:     true,
			conf99:      true,
			conf93:      true,
			conf5:       true,
		},
		{
			description: "Very low confidence",
			text:        strings.Repeat("Random text is random, but not a license\n", 40),
			confDef:     false,
			conf99:      false,
			conf93:      false,
			conf5:       true,
		},
	}

	defer func() {
		classifier.Threshold = DefaultConfidenceThreshold
	}()
	for _, tt := range tests {
		classifier.Threshold = DefaultConfidenceThreshold
		m := classifier.NearestMatch(tt.text)
		if got := classifier.WithinConfidenceThreshold(m.Confidence); got != tt.confDef {
			t.Errorf("WithinConfidenceThreshold(%q) = %v, want %v", tt.description, got, tt.confDef)
		}

		classifier.Threshold = 0.99
		m = classifier.NearestMatch(tt.text)
		if got := classifier.WithinConfidenceThreshold(m.Confidence); got != tt.conf99 {
			t.Errorf("WithinConfidenceThreshold(%q) = %v, want %v", tt.description, got, tt.conf99)
		}

		classifier.Threshold = 0.93
		m = classifier.NearestMatch(tt.text)
		if got := classifier.WithinConfidenceThreshold(m.Confidence); got != tt.conf93 {
			t.Errorf("WithinConfidenceThreshold(%q) = %v, want %v", tt.description, got, tt.conf93)
		}

		classifier.Threshold = 0.05
		m = classifier.NearestMatch(tt.text)
		if got := classifier.WithinConfidenceThreshold(m.Confidence); got != tt.conf5 {
			t.Errorf("WithinConfidenceThreshold(%q) = %v, want %v", tt.description, got, tt.conf5)
		}
	}
}

func TestRemoveIgnorableText(t *testing.T) {
	const want = `Lorem ipsum dolor sit amet, pellentesque wisi tortor duis, amet adipiscing bibendum elit aliquam
leo. Mattis commodo sed accumsan at in.
`

	tests := []struct {
		original string
		want     string
	}{
		{"MIT License\n", "\n"},
		{"The MIT License\n", "\n"},
		{"The MIT License (MIT)\n", "\n"},
		{"BSD License\n", "\n"},
		{"New BSD License\n", "\n"},
		{"COPYRIGHT AND PERMISSION NOTICE\n", "\n"},
		{"Copyright (c) 2016, Yoyodyne, Inc.\n", "\n"},
		{"All rights reserved.\n", "\n"},
		{"Some rights reserved.\n", "\n"},
		{"@license\n", "\n"},

		// Now with wanted texts.
		{
			original: `The MIT License

Copyright (c) 2016, Yoyodyne, Inc.
All rights reserved.
` + want,
			want: strings.ToLower(want),
		},
	}

	for _, tt := range tests {
		if got := removeIgnorableTexts(strings.ToLower(tt.original)); got != tt.want {
			t.Errorf("Mismatch(%q) =>\n%s\nwant:\n%s", tt.original, got, tt.want)
		}
	}
}

func TestRemoveShebangLine(t *testing.T) {
	tests := []struct {
		original string
		want     string
	}{
		{
			original: "",
			want:     "",
		},
		{
			original: "#!/usr/bin/env python -C",
			want:     "#!/usr/bin/env python -C",
		},
		{
			original: `#!/usr/bin/env python -C
# First line of license text.
# Second line of license text.
`,
			want: `# First line of license text.
# Second line of license text.
`,
		},
		{
			original: `# First line of license text.
# Second line of license text.
`,
			want: `# First line of license text.
# Second line of license text.
`,
		},
	}

	for _, tt := range tests {
		got := removeShebangLine(tt.original)
		if got != tt.want {
			t.Errorf("RemoveShebangLine(%q) =>\n%s\nwant:\n%s", tt.original, got, tt.want)
		}
	}
}

func TestRemoveNonWords(t *testing.T) {
	tests := []struct {
		original string
		want     string
	}{
		{
			original: `# # Hello
## World
`,
			want: ` Hello World `,
		},
		{
			original: ` * This text has a bulleted list:
 * * item 1
 * * item 2`,
			want: ` This text has a bulleted list item 1 item 2`,
		},
		{
			original: `

 * This text has a bulleted list:
 * * item 1
 * * item 2`,
			want: ` This text has a bulleted list item 1 item 2`,
		},
		{
			original: `// This text has a bulleted list:
// 1. item 1
// 2. item 2`,
			want: ` This text has a bulleted list 1 item 1 2 item 2`,
		},
		{
			original: `// «Copyright (c) 1998 Yoyodyne, Inc.»
// This text has a bulleted list:
// 1. item 1
// 2. item 2
`,
			want: ` «Copyright c 1998 Yoyodyne Inc » This text has a bulleted list 1 item 1 2 item 2 `,
		},
		{
			original: `*
 * This is the first line we want.
 * This is the second line we want.
 * This is the third line we want.
 * This is the last line we want.
`,
			want: ` This is the first line we want This is the second line we want This is the third line we want This is the last line we want `,
		},
		{
			original: `===---------------------------------------------===
***
* This is the first line we want.
* This is the second line we want.
* This is the third line we want.
* This is the last line we want.
***
===---------------------------------------------===
`,
			want: ` This is the first line we want This is the second line we want This is the third line we want This is the last line we want `,
		},
		{
			original: strings.Repeat("-", 80),
			want:     " ",
		},
		{
			original: strings.Repeat("=", 80),
			want:     " ",
		},
		{
			original: "/*\n",
			want:     " ",
		},
		{
			original: "/*\n * precursor text\n */\n",
			want:     " precursor text ",
		},
		// Test for b/63540492.
		{
			original: " */\n",
			want:     " ",
		},
		{
			original: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		if got := stringclassifier.FlattenWhitespace(RemoveNonWords(tt.original)); got != tt.want {
			t.Errorf("Mismatch(%q) => %v, want %v", tt.original, got, tt.want)
		}
	}
}

func TestNormalizePunctuation(t *testing.T) {
	tests := []struct {
		original string
		want     string
	}{
		// Hyphens and dashes.
		{"—", "-"},
		{"-", "-"},
		{"‒", "-"},
		{"–", "-"},
		{"—", "-"},

		// Quotes.
		{"'", "'"},
		{`"`, "'"},
		{"‘", "'"},
		{"’", "'"},
		{"“", "'"},
		{"”", "'"},
		{" ” ", " ' "},

		// Backtick.
		{"`", "'"},

		// Copyright mark.
		{"©", "(c)"},

		// Hyphen-separated words.
		{"general- purpose, non- compliant", "general-purpose, non-compliant"},

		// Section.
		{"§", "(s)"},
		{"¤", "(s)"},
	}

	for _, tt := range tests {
		if got := NormalizePunctuation(tt.original); got != tt.want {
			t.Errorf("Mismatch => %v, want %v", got, tt.want)
		}
	}
}

func TestNormalizeEquivalentWords(t *testing.T) {
	tests := []struct {
		original string
		want     string
	}{
		{"acknowledgment", "Acknowledgement"},
		{"ANalogue", "Analog"},
		{"AnAlyse", "Analyze"},
		{"ArtefacT", "Artifact"},
		{"authorisation", "Authorization"},
		{"AuthoriSed", "Authorized"},
		{"CalIbre", "Caliber"},
		{"CanCelled", "Canceled"},
		{"CapitaliSations", "Capitalizations"},
		{"CatalogUe", "Catalog"},
		{"CategoriSe", "Categorize"},
		{"CentRE", "Center"},
		{"EmphasiSed", "Emphasized"},
		{"FavoUr", "Favor"},
		{"FavoUrite", "Favorite"},
		{"FulfiL", "Fulfill"},
		{"FulfiLment", "Fulfillment"},
		{"InitialiSe", "Initialize"},
		{"JudGMent", "Judgement"},
		{"LabelLing", "Labeling"},
		{"LaboUr", "Labor"},
		{"LicenCe", "License"},
		{"MaximiSe", "Maximize"},
		{"ModelLed", "Modeled"},
		{"ModeLling", "Modeling"},
		{"OffenCe", "Offense"},
		{"OptimiSe", "Optimize"},
		{"OrganiSation", "Organization"},
		{"OrganiSe", "Organize"},
		{"PractiSe", "Practice"},
		{"ProgramME", "Program"},
		{"RealiSe", "Realize"},
		{"RecogniSe", "Recognize"},
		{"SignalLing", "Signaling"},
		{"sub-license", "Sublicense"},
		{"sub license", "Sublicense"},
		{"UtiliSation", "Utilization"},
		{"WhilST", "While"},
		{"WilfuL", "Wilfull"},
		{"Non-coMMercial", "Noncommercial"},
		{"Per Cent", "Percent"},
	}

	for _, tt := range tests {
		if got := NormalizeEquivalentWords(tt.original); got != tt.want {
			t.Errorf("Mismatch => %v, want %v", got, tt.want)
		}
	}
}

func TestTrimExtraneousTrailingText(t *testing.T) {
	tests := []struct {
		original string
		want     string
	}{
		{
			original: `12. IN NO EVENT UNLESS REQUIRED BY APPLICABLE LAW OR AGREED TO IN WRITING WILL
    ANY COPYRIGHT HOLDER, OR ANY OTHER PARTY WHO MAY MODIFY AND/OR REDISTRIBUTE
    THE PROGRAM AS PERMITTED ABOVE, BE LIABLE TO YOU FOR DAMAGES, INCLUDING ANY
    GENERAL, SPECIAL, INCIDENTAL OR CONSEQUENTIAL DAMAGES ARISING OUT OF THE
    USE OR INABILITY TO USE THE PROGRAM (INCLUDING BUT NOT LIMITED TO LOSS OF
    DATA OR DATA BEING RENDERED INACCURATE OR LOSSES SUSTAINED BY YOU OR THIRD
    PARTIES OR A FAILURE OF THE PROGRAM TO OPERATE WITH ANY OTHER PROGRAMS),
    EVEN IF SUCH HOLDER OR OTHER PARTY HAS BEEN ADVISED OF THE POSSIBILITY OF
    SUCH DAMAGES.

        END OF TERMS AND CONDITIONS

    How to Apply These Terms to Your New Programs

    If you develop a new program, and you want it to be of the greatest
    possible use to the public, the best way to achieve this is to make it free
    software which everyone can redistribute and change under these terms.
`,
			want: `12. IN NO EVENT UNLESS REQUIRED BY APPLICABLE LAW OR AGREED TO IN WRITING WILL
    ANY COPYRIGHT HOLDER, OR ANY OTHER PARTY WHO MAY MODIFY AND/OR REDISTRIBUTE
    THE PROGRAM AS PERMITTED ABOVE, BE LIABLE TO YOU FOR DAMAGES, INCLUDING ANY
    GENERAL, SPECIAL, INCIDENTAL OR CONSEQUENTIAL DAMAGES ARISING OUT OF THE
    USE OR INABILITY TO USE THE PROGRAM (INCLUDING BUT NOT LIMITED TO LOSS OF
    DATA OR DATA BEING RENDERED INACCURATE OR LOSSES SUSTAINED BY YOU OR THIRD
    PARTIES OR A FAILURE OF THE PROGRAM TO OPERATE WITH ANY OTHER PROGRAMS),
    EVEN IF SUCH HOLDER OR OTHER PARTY HAS BEEN ADVISED OF THE POSSIBILITY OF
    SUCH DAMAGES.

        END OF TERMS AND CONDITIONS`,
		},
	}

	for _, tt := range tests {
		if got := TrimExtraneousTrailingText(tt.original); got != tt.want {
			t.Errorf("Mismatch => %q, want %q", got, tt.want)
		}
	}
}

func TestCommonLicenseWords(t *testing.T) {
	files, err := ReadLicenseDir()
	if err != nil {
		t.Fatalf("error: cannot read licenses directory: %v", err)
	}
	if files == nil {
		t.Fatal("error: cannot get licenses from license directory")
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".txt" {
			continue
		}
		text, err := ReadLicenseFile(file.Name())
		if err != nil {
			t.Fatalf("error reading contents of %q: %v", file.Name(), err)
		}

		if got := classifier.hasCommonLicenseWords(string(text)); !got {
			t.Errorf("Mismatch(%q) => false, want true", file.Name())
		}
	}

	text := strings.Repeat("Þetta er ekki leyfi.\n", 80)
	if got := classifier.hasCommonLicenseWords(text); got {
		t.Error("Mismatch => true, want false")
	}
}

func TestLicenseMatchQuality(t *testing.T) {
	files, err := ReadLicenseDir()
	if err != nil {
		t.Fatalf("error: cannot read licenses directory: %v", err)
	}

	classifier.Threshold = 1.0
	defer func() {
		classifier.Threshold = DefaultConfidenceThreshold
	}()
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".txt" {
			continue
		}
		name := strings.TrimSuffix(file.Name(), ".txt")

		contents, err := ReadLicenseFile(file.Name())
		if err != nil {
			t.Fatalf("error reading contents of %q: %v", file.Name(), err)
		}

		m := classifier.NearestMatch(TrimExtraneousTrailingText(string(contents)))
		if m == nil {
			t.Errorf("Couldn't match %q", name)
			continue
		}

		if !classifier.WithinConfidenceThreshold(m.Confidence) {
			t.Errorf("ConfidenceMatch(%q) => %v, want %v", name, m.Confidence, 0.99)
		}
		want := strings.TrimSuffix(name, ".header")
		if want != m.Name {
			t.Errorf("LicenseMatch(%q) => %v, want %v", name, m.Name, want)
		}
	}
}

func BenchmarkClassifier(b *testing.B) {
	contents := apache20[:len(apache20)/2] + "hello" + apache20[len(apache20)/2:]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier, err := New(DefaultConfidenceThreshold)
		if err != nil {
			b.Errorf("Cannot create classifier: %v", err)
			continue
		}
		classifier.NearestMatch(contents)
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		desc        string
		options     []OptionFunc
		wantArchive string
		wantErr     bool
	}{
		{
			desc:        "no options, use default",
			options:     []OptionFunc{},
			wantArchive: LicenseArchive,
		},
		{
			desc:        "specify ForbiddenLicenseArchive",
			options:     []OptionFunc{Archive(ForbiddenLicenseArchive)},
			wantArchive: ForbiddenLicenseArchive,
		},
		{
			desc:        "file doesn't exist results in error",
			options:     []OptionFunc{Archive("doesnotexist")},
			wantArchive: "doesnotexist",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			c, err := New(0.5, tt.options...)
			if tt.wantErr != (err != nil) {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && c.archive != tt.wantArchive {
				t.Errorf("got archive %v, want %v", c.archive, tt.wantArchive)
			}
		})
	}

}
