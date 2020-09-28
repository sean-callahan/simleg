package simleg

import (
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

func Lex(input string) {
	l := lex(input)

	for i := l.nextItem(); i.typ != itemEOF; {
		fmt.Println(i)
		i = l.nextItem()
	}

}

const eof = -1

// The following code is based on Rob Pike's talk on Lexical Scanning in Go.
// https://talks.golang.org/2011/lex.slide

type itemType int

const (
	itemEOF itemType = iota
	itemError
	itemName
	itemInteger
	itemColon  // :
	itemComma  // ,
	itemLbrack // [
	itemRbrack // ]
)

type item struct {
	typ  itemType
	text string
}

type lexer struct {
	mu    sync.RWMutex
	input string
	start int
	pos   int
	width int
	state stateFn
	items chan item
}

// lex creates a new scanner for the input string.
func lex(input string) *lexer {
	l := &lexer{
		input: input,
		state: lexInput,
		items: make(chan item, 2), // Two items sufficient.
	}
	return l
}

// run lexes the input by executing state functions until
// the state is nil.
func (l *lexer) run() {
	for state := lexInput; state != nil; {
		state = state(l)
	}
	close(l.items) // No more tokens will be delivered.
}

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			l.state = l.state(l)
		}
	}
}

type stateFn func(*lexer) stateFn

func lexInput(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			l.emit(itemEOF)
			return nil
		case unicode.IsSpace(r):
			l.ignore()
		case r == '\n':
			l.ignore()
		case r == ';':
			l.ignoreLine()
		case r == ':':
			l.emit(itemColon)
		case r == ',':
			l.emit(itemComma)
		case r == '[':
			l.emit(itemLbrack)
		case r == ']':
			l.emit(itemRbrack)
		case r == '/':
			if nr := l.next(); nr == '/' {
				l.ignoreLine()
				break
			}
			return l.errorf("unexpected '%c'", r)
		case r == '#':
			if nr := l.next(); unicode.IsDigit(nr) {
				l.backup() // digit
				l.backup() // #
				return lexInteger
			}
			return l.errorf("missing digit")
		case unicode.IsLetter(r):
			l.backup()
			return lexName
		case unicode.IsDigit(r):
			l.backup()
			return lexInteger
		default:
			return l.errorf("unexpected '%c'", r)
		}
	}
}

func lexName(l *lexer) stateFn {
	l.acceptRange(unicode.IsLetter, unicode.IsDigit)
	l.accept(".") // might be a B.?
	l.acceptRange(unicode.IsLetter, unicode.IsDigit)
	l.emit(itemName)
	return lexInput
}

func lexInteger(l *lexer) stateFn {
	l.accept("#") // optional hash
	l.acceptRange(unicode.IsDigit)
	l.emit(itemInteger)
	return lexInput
}

// error returns an error token and terminates the scan
// by passing back a nil pointer that will be the next
// state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

func (l *lexer) emit(t itemType) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

// next returns the next rune in the input.
func (l *lexer) next() (r rune) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.start = l.pos
}

func (l *lexer) ignoreLine() {
	var r rune
	for r = l.next(); !(r == '\n' || r == eof); {
		r = l.next()
	}
	if r == eof {
		l.backup()
	}
	l.start = l.pos
}

// backup steps back one rune.
// Can be called only once per call of next.
func (l *lexer) backup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pos -= l.width
}

// peek returns but does not consume
// the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// accept consumes the next rune
// if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func (l *lexer) acceptRange(fns ...func(r rune) bool) {
	for {
	next:
		r := l.next()
		for _, fn := range fns {
			if fn(r) {
				goto next
			}
		}
		break
	}
	l.backup()
}
