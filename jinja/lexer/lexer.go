package lexer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

const tabSize = 4

type lexer struct {
	reader *bufio.Reader

	// scanner state information
	currentRune  rune
	nextRune     rune
	runePosition Position

	// Token state tracking
	tokenPosition Position

	// lexer state tracking
	inBlock bool
}

func LexFile(file io.Reader) ([]*Token, error) {
	lexer := &lexer{
		reader:       bufio.NewReader(file),
		runePosition: Position{0, 1},
	}

	// Read the first character into the nextRune buffer
	if err := lexer.readRune(); err != nil {
		return nil, err
	}

	tokens := make([]*Token, 0)

	for {
		t, err := lexer.NextToken()
		if err != nil {
			fmt.Println(t)
			return nil, err
		}

		tokens = append(tokens, t)

		if t.Type == EOFToken {
			break
		}
	}

	return tokens, nil
}

func (l *lexer) newToken(t TokenType) *Token {
	return l.newTokenWithValue(t, "")
}

func (l *lexer) newTokenWithValue(t TokenType, value string) *Token {
	return &Token{
		Type:  t,
		Value: value,
		Start: l.tokenPosition,
		End:   l.runePosition,
	}
}

func (l *lexer) readRune() error {
	r, _, err := l.reader.ReadRune()
	if err == io.EOF {
		r = 0
	} else if err != nil {
		return err
	}

	// If the last current rune was a new line then
	if l.currentRune == '\n' {
		// the next rune is in column 1 of the next row
		l.runePosition.Column = 1
		l.runePosition.Row++
	} else if l.currentRune == '\t' {
		l.runePosition.Column += tabSize
	} else {
		l.runePosition.Column++
	}

	l.currentRune = l.nextRune
	l.nextRune = r

	return nil
}

// Consume all runes until the next non-whitespace character
func (l *lexer) consumeWhitespace() error {
	for unicode.IsSpace(l.currentRune) {
		if err := l.readRune(); err != nil {
			return nil
		}
	}

	// Update the Token start position to ignore any whitespace
	l.tokenPosition = l.runePosition

	return nil
}

func (l *lexer) NextToken() (*Token, error) {
	// Read the next rune
	if err := l.readRune(); err != nil {
		return nil, err
	}

	// Copy the position of the first rune for this Token
	l.tokenPosition = l.runePosition

	// Have we reached the end of the file?
	if l.currentRune == 0 {
		return l.newToken(EOFToken), nil
	}

	// Are we in a JINJA code block or still in text mode?
	if l.inBlock {
		return l.nextBlockToken()
	} else {
		return l.nextTextModeToken()
	}
}

// Get the next block of text from the file or the
// block opening Token
func (l *lexer) nextTextModeToken() (*Token, error) {
	if l.currentRune == '{' &&
		l.nextRune == '{' {
		// Swap to being in a block
		l.inBlock = true

		// Consume the second opening brace
		if err := l.readRune(); err != nil {
			return nil, err
		}

		// Return a Block Open Token
		return l.newToken(TemplateBlockOpen), nil
	}

	if l.currentRune == '{' &&
		l.nextRune == '%' {
		// Swap to being in a block
		l.inBlock = true

		// Consume the second opening brace
		if err := l.readRune(); err != nil {
			return nil, err
		}

		// Detect a trim prefix command `{%-`
		if l.nextRune == '-' {
			if err := l.readRune(); err != nil {
				return nil, err
			}
			return l.newToken(ExpressionBlockOpenTrim), nil
		}

		// Return a Block Open Token
		return l.newToken(ExpressionBlockOpen), nil
	}

	// Comment block support
	if l.currentRune == '{' &&
		l.nextRune == '#' {

		// Consume the # and the move onto the following character
		// This is so we don't parse {#} as {##}
		if err := l.readRune(); err != nil {
			return nil, err
		}
		if err := l.readRune(); err != nil {
			return nil, err
		}

		for !(l.currentRune == '#' && l.nextRune == '}') {
			if err := l.readRune(); err != nil {
				return nil, err
			}
		}

		// Consume the closing '#}'
		if err := l.readRune(); err != nil {
			return nil, err
		}
		if err := l.readRune(); err != nil {
			return nil, err
		}
	}

	if l.currentRune == 0 {
		return l.newToken(EOFToken), nil
	}

	// Read the rest of the string block
	var buf strings.Builder
	for l.nextRune != '{' && l.nextRune != 0 {
		buf.WriteRune(l.currentRune)

		if err := l.readRune(); err != nil {
			return nil, err
		}
	}
	buf.WriteRune(l.currentRune) // and the final non exit character

	return l.newTokenWithValue(TextToken, buf.String()), nil
}

// Get the next Token out of the code block we're in
func (l *lexer) nextBlockToken() (*Token, error) {
	if err := l.consumeWhitespace(); err != nil {
		return nil, err
	}

	switch {
	// Check if we're exiting block
	case l.currentRune == '}' && l.nextRune == '}':
		// consume the next Token too
		if err := l.readRune(); err != nil {
			return nil, err
		}

		// Mark us as leaving a code block
		l.inBlock = false

		return l.newToken(TemplateBlockClose), nil

	case l.currentRune == '%' && l.nextRune == '}':
		// consume the next Token too
		if err := l.readRune(); err != nil {
			return nil, err
		}

		// Mark us as leaving a code block
		l.inBlock = false

		return l.newToken(ExpressionBlockClose), nil

	case l.currentRune == '-' && l.nextRune == '%':
		// consume the next Token too
		if err := l.readRune(); err != nil {
			return nil, err
		}

		if l.nextRune != '}' {
			return l.newTokenWithValue(ErrorToken, fmt.Sprintf("Expected } got %s", string(l.nextRune))), errors.New("unexpected char")
		}

		// Finally consume the '}'
		if err := l.readRune(); err != nil {
			return nil, err
		}

		// Mark us as leaving a code block
		l.inBlock = false

		return l.newToken(ExpressionBlockCloseTrim), nil

	case l.currentRune == '(':
		return l.newToken(LeftParenthesesToken), nil

	case l.currentRune == ')':
		return l.newToken(RightParenthesesToken), nil

	case l.currentRune == '[':
		return l.newToken(LeftBracketToken), nil

	case l.currentRune == ']':
		return l.newToken(RightBracketToken), nil

	case l.currentRune == '{':
		return l.newToken(LeftBraceToken), nil

	case l.currentRune == '}':
		return l.newToken(RightBraceToken), nil

	case l.currentRune == '=' && l.nextRune == '=':
		if err := l.readRune(); err != nil {
			return nil, err
		}
		return l.newToken(IsEqualsToken), nil

	case l.currentRune == '=':
		return l.newToken(EqualsToken), nil

	case l.currentRune == '!' && l.nextRune == '=':
		if err := l.readRune(); err != nil {
			return nil, err
		}
		return l.newToken(NotEqualsToken), nil

	case l.currentRune == '<' && l.nextRune == '=':
		if err := l.readRune(); err != nil {
			return nil, err
		}
		return l.newToken(LessThanEqualsToken), nil

	case l.currentRune == '<':
		return l.newToken(LessThanToken), nil

	case l.currentRune == '>' && l.nextRune == '=':
		if err := l.readRune(); err != nil {
			return nil, err
		}
		return l.newToken(GreaterThanEqualsToken), nil

	case l.currentRune == '>':
		return l.newToken(GreaterThanToken), nil

	case l.currentRune == ':':
		return l.newToken(ColonToken), nil

	case l.currentRune == ',':
		return l.newToken(CommaToken), nil

	case l.currentRune == '.':
		return l.newToken(PeriodToken), nil

	case l.currentRune == '-':
		return l.newToken(MinusToken), nil

	case l.currentRune == '+':
		return l.newToken(PlusToken), nil

	case l.currentRune == '*':
		return l.newToken(MultiplyToken), nil

	case l.currentRune == '/':
		return l.newToken(DivideToken), nil

	case l.currentRune == '|':
		return l.newToken(PipeToken), nil

	case l.currentRune == '~':
		return l.newToken(TildeToken), nil

	case l.currentRune == '\'' && l.nextRune == '\'':
		return l.readMultilineStringToken()

	case l.currentRune == '"' || l.currentRune == '\'':
		return l.readStringToken(l.currentRune)

	case unicode.IsNumber(l.currentRune):
		return l.readNumberToken()

	case isIdentRune(l.currentRune):
		return l.readIdentifierToken()

	default:
		// Read the rest of the string block
		var buf strings.Builder
		for l.nextRune != '}' && l.nextRune != 0 {
			buf.WriteRune(l.currentRune)

			if err := l.readRune(); err != nil {
				return nil, err
			}
		}
		buf.WriteRune(l.currentRune)

		return l.newTokenWithValue(ErrorToken, buf.String()), errors.New("lexer error encountered")
	}
}

func (l *lexer) readStringToken(exitRune rune) (*Token, error) {
	// Read all the characters in the string
	var buf strings.Builder
	for l.nextRune != exitRune && l.nextRune != 0 {
		if err := l.readRune(); err != nil {
			return nil, err
		}

		buf.WriteRune(l.currentRune)
	}

	// Consume the closing rune
	if err := l.readRune(); err != nil {
		return nil, err
	}

	return l.newTokenWithValue(StringToken, buf.String()), nil
}

// This is a string which starts with ''' and ends with '''
func (l *lexer) readMultilineStringToken() (*Token, error) {
	// Consume the opening quote
	if err := l.readRune(); err != nil {
		return nil, err
	}

	// Check if we've actually been given `blah '' blah` rather than `blah ''' blah`
	if l.nextRune != '\'' {
		return l.newTokenWithValue(StringToken, ""), nil
	}

	// Consume the third '
	if err := l.readRune(); err != nil {
		return nil, err
	}

	// Read all the characters in the string
	var buf strings.Builder
	exitCount := 0

	for {
		for l.currentRune != '\'' && l.currentRune != 0 {
			if exitCount > 0 {
				exitCount = 0
				buf.WriteString(strings.Repeat("'", exitCount))
			}

			buf.WriteRune(l.currentRune)

			if err := l.readRune(); err != nil {
				return nil, err
			}
		}

		if l.currentRune == '\'' || l.currentRune == 0 {
			exitCount++

			if exitCount == 3 {
				break
			}

			if err := l.readRune(); err != nil {
				return nil, err
			}
		}
	}

	return l.newTokenWithValue(StringToken, buf.String()), nil
}

func (l *lexer) readNumberToken() (*Token, error) {
	var buf strings.Builder

	hasDecimalPoint := false
	for unicode.IsNumber(l.nextRune) || (l.nextRune == '.' && !hasDecimalPoint) && l.currentRune != 0 {
		buf.WriteRune(l.currentRune)

		if err := l.readRune(); err != nil {
			return nil, err
		}

		if l.currentRune == '.' {
			hasDecimalPoint = true
		}
	}

	// Write the last character of the ident in
	buf.WriteRune(l.currentRune)

	return l.newTokenWithValue(NumberToken, buf.String()), nil
}

func (l *lexer) readIdentifierToken() (*Token, error) {
	var buf strings.Builder
	for (isIdentRune(l.nextRune) || unicode.IsNumber(l.nextRune)) && l.currentRune != 0 {
		buf.WriteRune(l.currentRune)

		if err := l.readRune(); err != nil {
			return nil, err
		}
	}

	// Write the last character of the ident in
	buf.WriteRune(l.currentRune)
	value := buf.String()

	// Keyword replacement
	switch strings.ToLower(value) {
	case "true":
		return l.newToken(TrueToken), nil
	case "false":
		return l.newToken(FalseToken), nil
	case "null":
		return l.newToken(NullToken), nil

	default:
		return l.newTokenWithValue(IdentToken, value), nil
	}
}

func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}
