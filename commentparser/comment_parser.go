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

// Package commentparser does a basic parse over a source file and returns all
// of the comments from the code. This is useful for when you want to analyze
// text written in comments (like copyright notices) but not in the code
// itself.
package commentparser

import (
	"bytes"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/google/licenseclassifier/commentparser/language"
)

const (
	eofInString            = "commentparser: Line %d > EOF in string"
	eofInSingleLineComment = "commentparser: Line %d > EOF in single line comment"
	eofInMultilineComment  = "commentparser: Line %d > EOF in multiline comment"
)

// Parse parses the input data and returns the comments.
func Parse(contents []byte, lang language.Language) Comments {
	if len(contents) == 0 {
		return nil
	}

	c := string(contents)
	if !strings.HasSuffix(c, "\n") {
		// Force a terminating newline if one isn't present.
		c += "\n"
	}
	i := &input{
		s:      c,
		lang:   lang,
		offset: 0,
		pos:    position{line: 1, lineRune: []int{0}},
	}
	i.lex()
	return i.comments
}

// Comment is either a single line or multiline comment in a source code file.
// A single line comment has StartLine equal to EndLine. The lines are 1-based.
type Comment struct {
	StartLine int
	EndLine   int
	Text      string
}

// Comments allows us to treat a slice of comments as a unit.
type Comments []*Comment

// ChunkIterator returns a read-only channel and generates the comments in a
// goroutine, then closes the channel.
func (c Comments) ChunkIterator() <-chan Comments {
	ch := make(chan Comments)
	go func() {
		defer close(ch)

		if len(c) == 0 {
			return
		}

		prevChunk := c[0]
		for index := 0; index < len(c); index++ {
			var chunk Comments
			for ; index < len(c); index++ {
				if c[index].StartLine > prevChunk.StartLine+1 {
					break
				}
				if c[index].StartLine == prevChunk.StartLine+2 {
					if c[index].StartLine != c[index].EndLine || prevChunk.StartLine != prevChunk.EndLine {
						break
					}
				}
				chunk = append(chunk, c[index])
				prevChunk = c[index]
			}
			if len(chunk) == 0 {
				break
			}

			ch <- chunk
			if index >= len(c) {
				break
			}

			prevChunk = c[index]
			index--
		}
	}()
	return ch
}

// StartLine is the line number (1-based) the first part of the comment block
// starts on.
func (c Comments) StartLine() int {
	if len(c) == 0 {
		return 0
	}
	return c[0].StartLine
}

// String creates a string out of the text of the comments. Comment begin and
// end markers are removed.
func (c Comments) String() string {
	var s []string
	for _, cmt := range c {
		s = append(s, cmt.Text)
	}
	return strings.Join(s, "\n")
}

// position records the location of a lexeme.
type position struct {
	line     int   // Line number of input: 1-based
	lineRune []int // Rune offset from beginning of line: 0-based
}

// input holds the current state of the lexer.
type input struct {
	s        string            // Entire input.
	lang     language.Language // Source code language.
	offset   int               // Offset into input.
	pos      position          // Current position in the input.
	comments Comments          // Comments in the source file.
}

// lex is called to obtain the comments.
func (i *input) lex() {
	for {
		c, ok := i.peekRune()
		if !ok {
			break
		}

		switch c {
		case '"', '\'', '`': // String
			// Ignore strings because they could contain comment
			// start or end sequences which we need to ignore.
			if i.lang == language.HTML {
				// Quotes in HTML-like files aren't meaningful,
				// because it's basically plain text
				break
			}

			ok, hasEscape := i.lang.QuoteCharacter(c)
			if !ok {
				break
			}

			var content bytes.Buffer
			isDocString := false
			quote := string(c)
			if i.lang == language.Python {
				if c == '\'' && i.match("'''") {
					quote = "'''"
					// Assume module-level docstrings start at the
					// beginning of a line.  Function docstrings not
					// supported.
					if i.pos.lineRune[len(i.pos.lineRune)-1] == 3 {
						isDocString = true
					}
				} else if c == '"' && i.match(`"""`) {
					quote = `"""`
					if i.pos.lineRune[len(i.pos.lineRune)-1] == 3 {
						isDocString = true
					}
				} else {
					i.readRune() // Eat quote.
				}
			} else {
				i.readRune() // Eat quote.
			}

			startLine := i.pos.line
			for {
				c, ok = i.peekRune()
				if !ok {
					log.Printf(eofInString, startLine)
					return
				}
				if hasEscape && c == '\\' {
					i.readRune() // Eat escape.
				} else if i.match(quote) {
					break
				} else if (i.lang == language.JavaScript || i.lang == language.Perl) && c == '\n' {
					// JavaScript and Perl allow you to
					// specify regexes without quotes, but
					// which contain quotes. So treat the
					// newline as terminating the string.
					break
				}
				c := i.readRune()
				if isDocString {
					content.WriteRune(c)
				}
				if i.eof() {
					log.Printf(eofInString, startLine)
					return
				}
			}
			if isDocString {
				i.comments = append(i.comments, &Comment{
					StartLine: startLine,
					EndLine:   i.pos.line,
					Text:      content.String(),
				})
			}
		default:
			startLine := i.pos.line
			var comment bytes.Buffer
			if ok, start, end := i.multiLineComment(); ok { // Multiline comment
				nesting := 0
				startLine := i.pos.line
				for {
					if i.eof() {
						log.Printf(eofInMultilineComment, startLine)
						return
					}
					c := i.readRune()
					comment.WriteRune(c)
					if i.lang.NestedComments() && i.match(start) {
						// Allows nested comments.
						comment.WriteString(start)
						nesting++
					}
					if i.match(end) {
						if nesting > 0 {
							comment.WriteString(end)
							nesting--
						} else {
							break
						}
					}
				}
				i.comments = append(i.comments, &Comment{
					StartLine: startLine,
					EndLine:   i.pos.line,
					Text:      comment.String(),
				})
			} else if i.singleLineComment() { // Single line comment
				for {
					if i.eof() {
						log.Printf(eofInSingleLineComment, i.pos.line)
						return
					}
					c = i.readRune()
					if c == '\n' {
						i.unreadRune(c)
						break
					}
					comment.WriteRune(c)
				}
				i.comments = append(i.comments, &Comment{
					StartLine: startLine,
					EndLine:   i.pos.line,
					Text:      comment.String(),
				})
			}
		}

		i.readRune() // Ignore non-comments.
	}
}

// singleLineComment returns 'true' if we've run across a single line comment
// in the given language.
func (i *input) singleLineComment() bool {
	if i.match(i.lang.SingleLineCommentStart()) {
		return true
	}

	if i.lang == language.SQL {
		return i.match(language.MySQL.SingleLineCommentStart())
	} else if i.lang == language.ObjectiveC {
		return i.match(language.Matlab.SingleLineCommentStart())
	}

	return false
}

// multiLineComment returns 'true' if we've run across a multiline comment in
// the given language.
func (i *input) multiLineComment() (bool, string, string) {
	if s := i.lang.MultilineCommentStart(); i.match(s) {
		return true, s, i.lang.MultilineCommentEnd()
	}

	if i.lang == language.SQL {
		if s := language.MySQL.MultilineCommentStart(); i.match(s) {
			return true, s, language.MySQL.MultilineCommentEnd()
		}
	} else if i.lang == language.ObjectiveC {
		if s := language.Matlab.MultilineCommentStart(); i.match(s) {
			return true, s, language.Matlab.MultilineCommentEnd()
		}
	}

	return false, "", ""
}

// match returns 'true' if the next tokens in the stream match the given
// string.
func (i *input) match(s string) bool {
	if s == "" {
		return false
	}
	saved := s
	var read []rune
	for len(s) > 0 && !i.eof() {
		r, size := utf8.DecodeRuneInString(s)
		if c, ok := i.peekRune(); ok && c == r {
			read = append(read, c)
		} else {
			// No match. Push the tokens we read back onto the stack.
			for idx := len(read) - 1; idx >= 0; idx-- {
				i.unreadRune(read[idx])
			}
			return false
		}
		s = s[size:]
		i.readRune() // Eat token.
	}
	return string(read) == saved
}

// eof reports whether the input has reached the end of the file.
func (i *input) eof() bool {
	return len(i.s) <= i.offset
}

// peekRune returns the next rune in the input without consuming it.
func (i *input) peekRune() (rune, bool) {
	if i.eof() {
		return rune(0), false
	}
	r, _ := utf8.DecodeRuneInString(i.s[i.offset:])
	return r, true
}

// readRune consumes and returns the next rune in the input.
func (i *input) readRune() rune {
	r, size := utf8.DecodeRuneInString(i.s[i.offset:])
	if r == '\n' {
		i.pos.line++
		i.pos.lineRune = append(i.pos.lineRune, 0)
	} else {
		i.pos.lineRune[len(i.pos.lineRune)-1]++
	}
	i.offset += size
	return r
}

// unreadRune winds the lexer's state back to before the rune was read.
func (i *input) unreadRune(c rune) {
	p := make([]byte, utf8.UTFMax)
	size := utf8.EncodeRune(p, c)
	i.offset -= size
	if c == '\n' {
		i.pos.line--
		if len(i.pos.lineRune) > 1 {
			i.pos.lineRune = i.pos.lineRune[:len(i.pos.lineRune)-1]
		} else {
			i.pos.lineRune[len(i.pos.lineRune)-1] = 0
		}
	} else {
		i.pos.lineRune[len(i.pos.lineRune)-1]--
	}
}
