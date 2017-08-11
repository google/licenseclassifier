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

// Package language contains methods and information about the different
// programming languages the comment parser supports.
package language

import (
	"path/filepath"
	"strings"
)

// Language is the progamming language we're grabbing the comments from.
type Language int

// Languages we can retrieve comments from.
const (
	Unknown Language = iota
	AppleScript
	Assembly
	Batch
	C
	Clif
	Clojure
	CMake
	CSharp
	Dart
	Fortran
	GLSLF // OpenGL Shading Language
	Go
	Haskell
	HTML
	Java
	JavaScript
	Flex
	Lisp
	Matlab
	MySQL
	NinjaBuild
	ObjectiveC
	Perl
	Python
	R
	Ruby
	Rust
	Shader
	Shell
	SQL
	Swift
	SWIG
	TypeScript
	Yacc
	Yaml
)

// style is the comment styles that a language uses.
type style int

// Comment styles.
const (
	unknown     style = iota
	applescript       // -- ... and (* ... *)
	batch             // @REM
	bcpl              // // ... and /* ... */
	cmake             // # ... and #[[ ... ]]
	fortran           // ! ...
	haskell           // -- ... and {- ... -}
	html              // <!-- ... -->
	lisp              // ;; ...
	matlab            // % ...
	mysql             // # ... and /* ... */
	ruby              // # ... and =begin ... =end
	shell             // # ... and %{ ... %}
	sql               // -- ... and /* ... */
)

// ClassifyLanguage determines what language the source code was written in. It
// does this by looking at the file's extension.
func ClassifyLanguage(filename string) Language {
	ext := strings.ToLower(filepath.Ext(filename))
	if len(ext) == 0 || ext[0] != '.' {
		return Unknown
	}

	switch ext[1:] { // Skip the '.'.
	case "applescript":
		return AppleScript
	case "bat":
		return Batch
	case "c", "cc", "cpp", "c++", "h", "hh", "hpp":
		return C
	case "clif":
		return Clif
	case "cmake":
		return CMake
	case "cs":
		return CSharp
	case "dart":
		return Dart
	case "f", "f90", "f95":
		return Fortran
	case "glslf":
		return GLSLF
	case "go":
		return Go
	case "hs":
		return Haskell
	case "html", "htm", "md", "ng", "sgml":
		return HTML
	case "java":
		return Java
	case "js":
		return JavaScript
	case "l":
		return Flex
	case "lisp", "el", "clj":
		return Lisp
	case "m", "mm":
		return ObjectiveC
	case "gn":
		return NinjaBuild
	case "pl", "pm":
		return Perl
	case "py", "pi":
		return Python
	case "r":
		return R
	case "rb":
		return Ruby
	case "rs":
		return Rust
	case "s":
		return Assembly
	case "sh":
		return Shell
	case "shader":
		return Shader
	case "sql":
		return SQL
	case "swift":
		return Swift
	case "swig":
		return SWIG
	case "ts", "tsx":
		return TypeScript
	case "y":
		return Yacc
	case "yaml":
		return Yaml
	}
	return Unknown
}

// commentStyle returns the language's comment style.
func (lang Language) commentStyle() style {
	switch lang {
	case Assembly, C, CSharp, Dart, GLSLF, Go, Java, JavaScript, Flex, ObjectiveC, Rust, Shader, Swift, SWIG, TypeScript, Yacc:
		return bcpl
	case Batch:
		return batch
	case CMake:
		return cmake
	case Fortran:
		return fortran
	case Haskell:
		return haskell
	case HTML:
		return html
	case Clojure, Lisp:
		return lisp
	case Ruby:
		return ruby
	case Clif, NinjaBuild, Perl, Python, R, Shell, Yaml:
		return shell
	case Matlab:
		return matlab
	case MySQL:
		return mysql
	case SQL:
		return sql
	}
	return unknown
}

// SingleLineCommentStart returns the starting string of a single line comment
// for the given language. There is no equivalent "End" method, because it's
// the end of line.
func (lang Language) SingleLineCommentStart() string {
	switch lang.commentStyle() {
	case applescript, haskell, sql:
		return "--"
	case batch:
		return "@REM"
	case bcpl:
		return "//"
	case fortran:
		return "!"
	case lisp:
		return ";"
	case matlab:
		return "%"
	case shell, ruby, cmake, mysql:
		return "#"
	}
	return ""
}

// MultilineCommentStart returns the starting string of a multiline comment for
// the given language.
func (lang Language) MultilineCommentStart() string {
	switch lang.commentStyle() {
	case applescript:
		return "(*"
	case bcpl, mysql:
		if lang != Rust {
			return "/*"
		}
	case cmake:
		return "#[["
	case haskell:
		return "{-"
	case html:
		return "<!--"
	case matlab:
		return "%{"
	case ruby:
		return "=begin"
	}
	return ""
}

// MultilineCommentEnd returns the ending string of a multiline comment for the
// given langauge.
func (lang Language) MultilineCommentEnd() string {
	switch lang.commentStyle() {
	case applescript:
		return "*)"
	case bcpl, mysql:
		if lang != Rust {
			return "*/"
		}
	case cmake:
		return "]]"
	case haskell:
		return "-}"
	case html:
		return "-->"
	case matlab:
		return "%}"
	case ruby:
		return "=end"
	}
	return ""
}

// QuoteCharacter returns 'true' if the character is considered the beginning
// of a string in the given language. The second return value is true if the
// string allows for escaping.
func (lang Language) QuoteCharacter(quote rune) (ok bool, escape bool) {
	switch quote {
	case '"', '\'':
		return true, true
	case '`':
		if lang == Go {
			return true, false
		}
	}
	return false, false
}

// NestedComments returns true if the language allows for nested multiline comments.
func (lang Language) NestedComments() bool {
	return lang == Swift
}
