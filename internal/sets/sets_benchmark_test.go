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
package sets

import (
	"strings"
	"testing"
)

const (
	postmodernThesisCollapse = `1. Expressions of collapse

If one examines postcultural Marxism, one is faced with a choice: either
reject capitalist submodern theory or conclude that the purpose of the reader
is significant form. Bataille uses the term ‘capitalist construction’ to denote
not, in fact, discourse, but prediscourse.

Therefore, in Stardust, Gaiman analyses postcultural Marxism; in
The Books of Magic, although, he denies capitalist submodern theory. If
capitalist construction holds, we have to choose between capitalist submodern
theory and Baudrillardist simulacra.

However, conceptualist socialism implies that narrativity may be used to
oppress the proletariat, given that sexuality is distinct from art. The subject
is interpolated into a capitalist construction that includes language as a
paradox.
`
	postmodernThesisNarratives = `1. Narratives of failure

The main theme of the works of Joyce is the defining characteristic, and some
would say the economy, of neocultural class. But Bataille promotes the use of
socialist realism to deconstruct sexual identity.

The subject is interpolated into a Baudrillardist simulation that includes
consciousness as a whole. Thus, the primary theme of Pickett's[1] model of
socialist realism is the role of the reader as artist.

The subject is contextualised into a postcapitalist discourse that includes
language as a paradox. It could be said that if Baudrillardist simulation
holds, the works of Gibson are postmodern. The characteristic theme of the
works of Gibson is the common ground between society and narrativity. However,
Sartre uses the term 'postcapitalist discourse' to denote not, in fact,
narrative, but postnarrative.
`
)

var (
	// Word lists:
	stringsA = strings.Fields(postmodernThesisCollapse)
	stringsB = strings.Fields(postmodernThesisNarratives)
)

func BenchmarkStringSets_NewStringSet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewStringSet(stringsA...)
	}
}

func BenchmarkStringSets_Copy(b *testing.B) {
	s := NewStringSet(stringsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Copy()
	}
}

func BenchmarkStringSets_Insert(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewStringSet()
		s.Insert(stringsA...)
		s.Insert(stringsB...)
	}
}

func BenchmarkStringSets_Delete(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewStringSet(stringsA...)
		s.Delete(stringsB...)
	}
}

func BenchmarkStringSets_Intersect(b *testing.B) {
	s := NewStringSet(stringsA...)
	t := NewStringSet(stringsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Intersect(t)
	}
}

func BenchmarkStringSets_Disjoint(b *testing.B) {
	s := NewStringSet(stringsA...)
	t := NewStringSet(stringsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Disjoint(t)
	}
}

func BenchmarkStringSets_Difference(b *testing.B) {
	s := NewStringSet(stringsA...)
	t := NewStringSet(stringsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Difference(t)
	}
}

func BenchmarkStringSets_Unique(b *testing.B) {
	s := NewStringSet(stringsA...)
	t := NewStringSet(stringsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Unique(t)
	}
}

func BenchmarkStringSets_Equal(b *testing.B) {
	s := NewStringSet(stringsA...)
	t := NewStringSet(stringsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Equal(t)
	}
}

func BenchmarkStringSets_Union(b *testing.B) {
	s := NewStringSet(stringsA...)
	t := NewStringSet(stringsB...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Union(t)
	}
}

func BenchmarkStringSets_Contains(b *testing.B) {
	s := NewStringSet(stringsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, w := range stringsB {
			s.Contains(w)
		}
	}
}

func BenchmarkStringSets_Len(b *testing.B) {
	s := NewStringSet(stringsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Len()
	}
}

func BenchmarkStringSets_Empty(b *testing.B) {
	s := NewStringSet(stringsA...)
	t := NewStringSet()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Empty()
		t.Empty()
	}
}

func BenchmarkStringSets_Elements(b *testing.B) {
	s := NewStringSet(stringsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Elements()
	}
}

func BenchmarkStringSets_Sorted(b *testing.B) {
	s := NewStringSet(stringsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Sorted()
	}
}

func BenchmarkStringSets_String(b *testing.B) {
	s := NewStringSet(stringsA...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.String()
	}
}
