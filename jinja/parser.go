package jinja

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"ddbt/fs"
	"ddbt/jinja/ast"
	"ddbt/jinja/lexer"
)

type parser struct {
	tokens         []*lexer.Token
	nextTokenIndex int

	trimFollowingWhitespace bool
	previousTextBlock       *ast.TextBlock
}

func Parse(file *fs.File) (ast.AST, error) {
	var reader io.Reader

	if file.PrereadFileContents != "" {
		reader = strings.NewReader(file.PrereadFileContents)
	} else {
		f, err := os.Open(file.Path)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		reader = f
	}

	// TODO: Change tokens into a channel and lex the file async
	tokens, err := lexer.LexFile(file.Path, reader)
	if err != nil {
		return nil, err
	}

	p := &parser{
		tokens:         tokens,
		nextTokenIndex: 0,
	}

	body := ast.NewBody(p.peek())
	for {
		node, err := p.parse()
		if err != nil {
			return nil, err
		}

		body.Append(node)

		if _, ok := node.(*ast.EndOfFile); ok {
			break
		}
	}

	return body, nil
}

// Returns the next token
func (p *parser) next() *lexer.Token {
	t := p.tokens[p.nextTokenIndex]

	if t.Type != lexer.EOFToken {
		p.nextTokenIndex++
	}

	return t
}

// Peeks at the next token
func (p *parser) peek() *lexer.Token {
	return p.tokens[p.nextTokenIndex]
}

func (p *parser) peekIs(tokenType lexer.TokenType) bool {
	return p.tokens[p.nextTokenIndex].Type == tokenType ||
		p.tokens[p.nextTokenIndex].Type == lexer.EOFToken // Safety valve just in case
}

// Creates a parse error with the location information
func (p *parser) errorAt(atToken *lexer.Token, error string) error {
	return errors.New(
		fmt.Sprintf("%s at %s:%d:%d", error, atToken.Start.File, atToken.Start.Row, atToken.Start.Column),
	)
}

// Creates a parse error with what we expected and what we got
func (p *parser) expectedError(expected lexer.TokenType, got *lexer.Token) error {
	return p.errorAt(got, fmt.Sprintf("expected %s got %s", expected, got.Type))
}

func (p *parser) notImplemented() (ast.AST, error) {
	return nil, p.errorAt(p.tokens[p.nextTokenIndex-1], "not implemented")
}

func (p *parser) consumeIfPossible(tokenType lexer.TokenType) {
	if p.peekIs(tokenType) {
		p.next()
	}
}

func (p *parser) expectedAndConsumeValue(tokenType lexer.TokenType) (*lexer.Token, error) {
	t := p.next()
	if t.Type != tokenType {
		return nil, p.expectedError(tokenType, t)
	}

	return t, nil
}

func (p *parser) expectedAndConsumeIdentifier(keyword string) error {
	ident, err := p.expectedAndConsumeValue(lexer.IdentToken)
	if err != nil {
		return err
	}

	if ident.Value != keyword {
		return p.errorAt(ident, fmt.Sprintf("expected `%s` got `%s`", keyword, ident.Value))
	}

	return nil
}

func (p *parser) parse() (ast.AST, error) {
	switch p.peek().Type {
	case lexer.EOFToken:
		return ast.NewEndOfFile(p.next()), nil

	case lexer.TextToken:
		textToken := ast.NewTextBlock(p.next())

		if p.trimFollowingWhitespace {
			result := textToken.TrimPrefixWhitespace()

			if result == "" {
				// Ignore this text block because it's empty!
				return p.parse()
			}

			p.trimFollowingWhitespace = false // If here we have non-whitespace characters
		}

		p.previousTextBlock = textToken

		return textToken, nil

	case lexer.ExpressionBlockOpen, lexer.ExpressionBlockOpenTrim:
		p.trimFollowingWhitespace = false
		return p.parseExpressionBlock()

	case lexer.TemplateBlockOpen:
		// Reset our trim
		p.trimFollowingWhitespace = false
		p.previousTextBlock = nil

		return p.parseTemplateBlock()

	default:
		t := p.next()
		return nil, p.errorAt(t, "unexpected token: "+t.DisplayString())
	}
}

func (p *parser) parseExpressionBlock() (ast.AST, error) {
	if err := p.parseExpressionBlockOpen(); err != nil {
		return nil, err
	}

	t, err := p.expectedAndConsumeValue(lexer.IdentToken)
	if err != nil {
		return nil, err
	}

	// Assuming that atom expression blocks are always end markers, return early
	// i.e. {% endmacro %}  or {% endif %} or {% endfor %} or {% else %}
	if p.peekIs(lexer.ExpressionBlockClose) || p.peekIs(lexer.ExpressionBlockCloseTrim) {
		if err := p.parseExpressionBlockClose(); err != nil {
			return nil, err
		}

		return ast.NewAtomExpressionBlock(t), nil
	}

	switch t.Value {
	case "macro":
		return p.parseMacroDefinition()

	case "set":
		return p.parseSetCall()

	case "for":
		return p.parseForLoop()

	case "if":
		return p.parseIfStatement(false)

	case "elif":
		return p.parseIfStatement(true)

	case "call":
		return p.parseCallBlock()

	case "do":
		toRun, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		if err := p.parseExpressionBlockClose(); err != nil {
			return nil, err
		}

		return ast.NewDoBlock(t, toRun), nil

	case "materialization":
		// These are unsupported for now
		return p.parseUnsupportedBlockType(t)

	default:
		return nil, p.errorAt(t, "Expected `macro`, `set`, `for`, `if`, `call` got "+t.Value)
	}
}

func (p *parser) parseMacroDefinition() (ast.AST, error) {
	// Parse the macro header
	macroName, err := p.expectedAndConsumeValue(lexer.IdentToken)
	if err != nil {
		return nil, err
	}

	macro := ast.NewMacro(macroName)

	_, err = p.expectedAndConsumeValue(lexer.LeftParenthesesToken)
	if err != nil {
		return nil, err
	}

	// Loop over the arguments
	for !p.peekIs(lexer.RightParenthesesToken) {
		paramName, err := p.expectedAndConsumeValue(lexer.IdentToken)
		if err != nil {
			return nil, err
		}

		var defaultValue *lexer.Token

		// Do we have a default value?
		if p.peekIs(lexer.EqualsToken) {
			_ = p.next()            // consume the =
			defaultValue = p.next() // get the default

			if defaultValue.Type != lexer.StringToken &&
				defaultValue.Type != lexer.NumberToken &&
				defaultValue.Type != lexer.TrueToken && defaultValue.Type != lexer.FalseToken &&
				defaultValue.Type != lexer.NoneToken {
				return nil, p.errorAt(
					defaultValue,
					fmt.Sprintf("Expected string, number, boolean or `None` - got: %s", defaultValue.Type),
				)
			}
		}

		// Add the parameter
		if err := macro.AddParameter(paramName.Value, defaultValue); err != nil {
			return nil, p.errorAt(
				paramName,
				fmt.Sprintf("%s", err),
			)
		}

		// If no comma break out of the loop now
		if !p.peekIs(lexer.CommaToken) {
			break
		}

		_ = p.next() // consume the comma if it's there
	}

	_, err = p.expectedAndConsumeValue(lexer.RightParenthesesToken)
	if err != nil {
		return nil, err
	}

	err = p.parseExpressionBlockClose()
	if err != nil {
		return nil, err
	}

	if err := p.parseBodyUntilAtom("endmacro", macro); err != nil {
		return nil, err
	}

	return macro, nil
}

func (p *parser) parseTemplateBlock() (ast.AST, error) {
	if _, err := p.expectedAndConsumeValue(lexer.TemplateBlockOpen); err != nil {
		return nil, err
	}

	var node ast.AST
	var err error

	node, err = p.parseStatement()
	if err != nil {
		return nil, err
	}

	if variable, ok := node.(*ast.Variable); ok {
		variable.SetIsTemplateblock()
	}

	_, err = p.expectedAndConsumeValue(lexer.TemplateBlockClose)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (p *parser) parseFunctionCall(ident *lexer.Token) (*ast.FunctionCall, error) {
	funcCall := ast.NewFunctionCall(ident, ident.Value)

	if err := p.parseArgumentList(funcCall); err != nil {
		return nil, err
	}

	return funcCall, nil
}

func (p *parser) parseArgumentList(node ast.ArgumentHoldingAST) error {
	_, err := p.expectedAndConsumeValue(lexer.LeftParenthesesToken)
	if err != nil {
		return err
	}

	// Build the parameter list
	for !p.peekIs(lexer.RightParenthesesToken) {
		namedArg := ""

		firstToken := p.peek()

		statement, err := p.parseStatement()
		if err != nil {
			return err
		}

		// Was the statement actually a named parameter?
		if p.peekIs(lexer.EqualsToken) {
			if v, ok := statement.(*ast.Variable); ok && v.IsSimpleIdent(firstToken.Value) {
				namedArg = firstToken.Value

				_ = p.next() // consume the "="

				statement, err = p.parseStatement()
				if err != nil {
					return err
				}
			}
		}

		node.AddArgument(namedArg, statement)

		if !p.peekIs(lexer.CommaToken) {
			break
		}
		p.next()
	}

	_, err = p.expectedAndConsumeValue(lexer.RightParenthesesToken)
	if err != nil {
		return err
	}

	return nil
}

func (p *parser) parseValue() (ast.AST, error) {
	var statement ast.AST

	var err error
	if p.peekIs(lexer.LeftParenthesesToken) {
		_ = p.next() // consume the "("

		statement, err = p.parseStatement()
		if err != nil {
			return nil, err
		}

		_, err = p.expectedAndConsumeValue(lexer.RightParenthesesToken)
		if err != nil {
			return nil, err
		}

		statement = ast.NewBracketGroup(statement)

	} else if p.peekIs(lexer.StringToken) {
		statement = ast.NewTextBlock(p.next())

	} else if p.peekIs(lexer.NumberToken) {
		token := p.next()
		number, err := strconv.ParseFloat(token.Value, 64)
		if err != nil {
			return nil, p.errorAt(token, fmt.Sprintf("Unable to parse number: %s", err))
		}

		statement = ast.NewNumber(token, number)

	} else if p.peekIs(lexer.NullToken) {
		statement = ast.NewNullValue(p.next())

	} else if p.peekIs(lexer.NoneToken) {
		statement = ast.NewNoneValue(p.next())

	} else if p.peekIs(lexer.TrueToken) || p.peekIs(lexer.FalseToken) {
		statement = ast.NewBoolValue(p.next())

	} else if p.peekIs(lexer.MinusToken) {
		op := p.next()

		subStatement, err := p.parseValue() // We don't want the statement here, as we don't want greedy operators
		if err != nil {
			return nil, err
		}

		statement = ast.NewUniaryMathsOp(op, subStatement)

	} else if p.peekIs(lexer.LeftBracketToken) {
		statement, err = p.parseList()
		if err != nil {
			return nil, err
		}

	} else if p.peekIs(lexer.LeftBraceToken) {
		statement, err = p.parseMap()
		if err != nil {
			return nil, err
		}
	} else if p.peekIs(lexer.IdentToken) && p.peek().Value == "not" {
		notToken := p.next()

		sub, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		statement = ast.NewNotOperator(notToken, sub)

	} else {
		statement, err = p.parseVariable(nil)
		if err != nil {
			return nil, err
		}
	}

	return statement, nil
}

func (p *parser) parseStatement() (ast.AST, error) {
	statement, err := p.parseValue()

	// Check if the variable has a maths operation
	statement, err = p.parsePossibleMathsOps(statement)
	if err != nil {
		return nil, err
	}

	// Check for filters
	for p.peekIs(lexer.PipeToken) {
		pipeToken := p.next() // consume the "|"

		ident, err := p.expectedAndConsumeValue(lexer.IdentToken)
		if err != nil {
			return nil, err
		}

		fc := ast.NewFunctionCall(pipeToken, ident.Value)
		fc.AddArgument("", statement)

		if p.peekIs(lexer.LeftParenthesesToken) {
			if err := p.parseArgumentList(fc); err != nil {
				return nil, err
			}
		}

		statement = fc
	}

	for p.peekIs(lexer.TildeToken) {
		tildeToken := p.next() // consume the "~"

		rhs, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		statement = ast.NewStringConcat(tildeToken, statement, rhs)
	}

	// Do we have a suffix if op
	// i.e. `"foo" if bar else "baz"`
	if p.peekIs(lexer.IdentToken) && p.peek().Value == "if" {
		ifToken := p.next() // consume the "if'

		condition, err := p.parseCondition()
		if err != nil {
			return nil, err
		}

		ifs := ast.NewIfStatement(ifToken, condition)
		ifs.AppendBody(statement)

		if p.peekIs(lexer.IdentToken) && p.peek().Value == "else" {
			if err := p.expectedAndConsumeIdentifier("else"); err != nil {
				return nil, err
			}

			elseValue, err := p.parseStatement()
			if err != nil {
				return nil, err
			}

			ifs.AppendElse(elseValue)
		}

		statement = ifs
	}

	// Does the statement get turned into a condition?
	if p.peekIs(lexer.IsEqualsToken) || p.peekIs(lexer.NotEqualsToken) ||
		p.peekIs(lexer.LessThanToken) || p.peekIs(lexer.LessThanEqualsToken) ||
		p.peekIs(lexer.GreaterThanToken) || p.peekIs(lexer.GreaterThanEqualsToken) {
		opToken := p.next() // consume the operator token

		otherSide, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		statement = ast.NewLogicalOp(opToken, statement, otherSide)

		// Check if we and it or not
		if p.peekIs(lexer.IdentToken) && p.peek().Value == "and" {
			_ = p.next() // consume and

			otherSide, err := p.parseStatement()
			if err != nil {
				return nil, err
			}

			statement = ast.NewAndCondition(statement, otherSide)
		}

		if p.peekIs(lexer.IdentToken) && p.peek().Value == "or" {
			_ = p.next() // consume or

			otherSide, err := p.parseStatement()
			if err != nil {
				return nil, err
			}

			statement = ast.NewOrCondition(statement, otherSide)
		}

	}

	return statement, nil
}

func (p *parser) parsePossibleMathsOps(lhs ast.AST) (ast.AST, error) {
	switch p.peek().Type {
	case lexer.MultiplyToken, lexer.DivideToken, lexer.PlusToken, lexer.MinusToken, lexer.PowerToken:
		op := p.next()

		rhs, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		return ast.NewMathsOp(op, lhs, rhs).ApplyOperatorPrecedenceRules(), nil
	}

	return lhs, nil
}

func (p *parser) parseVariable(ident *lexer.Token) (*ast.Variable, error) {
	var err error

	// ident might have already been consumed
	if ident == nil {
		ident, err = p.expectedAndConsumeValue(lexer.IdentToken)
		if err != nil {
			return nil, err
		}
	}

	variable := ast.NewVariable(ident)

	for {
		switch p.peek().Type {
		case lexer.LeftParenthesesToken:
			// This variable is being treated like a function call (`a(b)`)
			variable = variable.AsCallable()

			if err := p.parseArgumentList(variable); err != nil {
				return nil, err
			}

		case lexer.LeftBracketToken:
			_ = p.next() // consume "["

			// This variable is being accesed like a map or array (`a["b"]`)
			key, err := p.parseStatement()
			if err != nil {
				return nil, err
			}

			if _, err := p.expectedAndConsumeValue(lexer.RightBracketToken); err != nil {
				return nil, err
			}

			variable = variable.AsIndexLookup(key)

		case lexer.PeriodToken:
			_ = p.next() // consume "."

			// This variable is being accesed like a object (`a.b`)
			ident, err := p.expectedAndConsumeValue(lexer.IdentToken)
			if err != nil {
				return nil, err
			}

			variable = variable.AsPropertyLookup(ident)

		default:
			// We've parsed the varaible completely now
			return variable, nil
		}
	}
}

func (p *parser) parseForLoop() (ast.AST, error) {
	keyIteratorName := ""
	valueIterator, err := p.expectedAndConsumeValue(lexer.IdentToken)
	if err != nil {
		return nil, err
	}

	if p.peekIs(lexer.CommaToken) {
		p.next()

		keyIteratorName = valueIterator.Value

		valueIterator, err = p.expectedAndConsumeValue(lexer.IdentToken)
		if err != nil {
			return nil, err
		}
	}

	if err := p.expectedAndConsumeIdentifier("in"); err != nil {
		return nil, err
	}

	list, err := p.parseVariable(nil)
	if err != nil {
		return nil, err
	}

	err = p.parseExpressionBlockClose()
	if err != nil {
		return nil, err
	}

	forLoop := ast.NewForLoop(valueIterator, keyIteratorName, list)

	if err := p.parseBodyUntilAtom("endfor", forLoop); err != nil {
		return nil, err
	}

	return forLoop, nil
}

func (p *parser) parseBodyUntilAtom(endAtom string, parentNode ast.BodyHoldingAST) error {
	for {
		node, err := p.parse()
		if err != nil {
			return err
		}

		if isAtomOrEOF(endAtom, node) {
			break
		} else {
			parentNode.AppendBody(node)
		}
	}

	return nil
}

func (p *parser) parseExpressionBlockOpen() error {
	if p.peekIs(lexer.ExpressionBlockOpenTrim) {
		if p.previousTextBlock != nil {
			p.previousTextBlock.TrimSuffixWhitespace()
		}

		_ = p.next()
	} else {
		_, err := p.expectedAndConsumeValue(lexer.ExpressionBlockOpen)
		if err != nil {
			return err
		}
	}

	// The previous block is no longer a text block
	p.previousTextBlock = nil

	return nil
}
func (p *parser) parseExpressionBlockClose() error {
	if p.peekIs(lexer.ExpressionBlockCloseTrim) {
		_ = p.next()
		p.trimFollowingWhitespace = true
	} else {
		_, err := p.expectedAndConsumeValue(lexer.ExpressionBlockClose)
		if err != nil {
			return err
		}

		p.trimFollowingWhitespace = false
	}

	return nil
}

func (p *parser) parseIfStatement(asElseIf bool) (ast.AST, error) {
	conditionToken := p.peek()

	condition, err := p.parseCondition()
	if err != nil {
		return nil, err
	}

	is := ast.NewIfStatement(conditionToken, condition)

	if err := p.parseExpressionBlockClose(); err != nil {
		return nil, err
	}

	inElse := false

	for {
		node, err := p.parse()
		if err != nil {
			return nil, err
		}

		if isAtomOrEOF("endif", node) {
			break
		} else if isAtomOrEOF("else", node) {
			inElse = true
		} else if elif, ok := node.(*ast.IfStatement); ok && elif.IsElseIf() {
			is.AppendElse(elif)
			break
		} else {
			if inElse {
				is.AppendElse(node)
			} else {
				is.AppendBody(node)
			}
		}
	}

	if asElseIf {
		is.SetAsElseIf()
	}

	return is, nil
}

func (p *parser) parseCondition() (ast.AST, error) {
	var condition ast.AST
	var err error

	if p.peekIs(lexer.LeftParenthesesToken) {
		_ = p.next() // consume the (

		// recursive without the the brackets
		condition, err = p.parseCondition()
		if err != nil {
			return nil, err
		}

		_, err = p.expectedAndConsumeValue(lexer.RightParenthesesToken)
		if err != nil {
			return nil, err
		}

		condition = ast.NewBracketGroup(condition)
	} else if p.peekIs(lexer.IdentToken) && p.peek().Value == "not" {
		notToken := p.next()

		sub, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		condition = ast.NewNotOperator(notToken, sub)
	} else {
		condition, err = p.parseStatement()
		if err != nil {
			return nil, err
		}
	}

	if p.peekIs(lexer.IsEqualsToken) || p.peekIs(lexer.NotEqualsToken) ||
		p.peekIs(lexer.LessThanToken) || p.peekIs(lexer.LessThanEqualsToken) ||
		p.peekIs(lexer.GreaterThanToken) || p.peekIs(lexer.GreaterThanEqualsToken) {
		opToken := p.next() // consume the operator token

		otherSide, err := p.parseCondition()
		if err != nil {
			return nil, err
		}

		condition = ast.NewLogicalOp(opToken, condition, otherSide)
	}

	if p.peekIs(lexer.IdentToken) && p.peek().Value == "is" {
		isToken := p.next() // consume the "is"

		value := "none"
		if p.peekIs(lexer.NoneToken) {
			p.next()
		} else {
			ident, err := p.expectedAndConsumeValue(lexer.IdentToken)
			if err != nil {
				return nil, err
			}

			value = ident.Value
		}

		if value == "not" {
			if p.peekIs(lexer.NoneToken) {
				p.next()
				value = "not none"
			} else {
				ident, err := p.expectedAndConsumeValue(lexer.IdentToken)
				if err != nil {
					return nil, err
				}

				value = "not " + ident.Value
			}
		}

		switch value {
		case "none", "defined", "not none", "not defined":
			condition = ast.NewDefineCheck(isToken, condition, value)
		default:
			return nil, p.errorAt(isToken, fmt.Sprintf("Expected `none` or `defined` got `%s`", value))
		}

	}

	if p.peekIs(lexer.IdentToken) && p.peek().Value == "in" {
		inToken := p.next() // consume the "in"

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		condition = ast.NewInOperator(inToken, condition, value)
	}

	if p.peekIs(lexer.IdentToken) && p.peek().Value == "and" {
		_ = p.next() // consume and

		otherSide, err := p.parseCondition()
		if err != nil {
			return nil, err
		}

		condition = ast.NewAndCondition(condition, otherSide)
	}

	if p.peekIs(lexer.IdentToken) && p.peek().Value == "or" {
		_ = p.next() // consume or

		otherSide, err := p.parseCondition()
		if err != nil {
			return nil, err
		}

		condition = ast.NewOrCondition(condition, otherSide)
	}

	return condition, nil
}

func (p *parser) parseSetCall() (ast.AST, error) {
	ident, err := p.expectedAndConsumeValue(lexer.IdentToken)
	if err != nil {
		return nil, err
	}

	_, err = p.expectedAndConsumeValue(lexer.EqualsToken)
	if err != nil {
		return nil, err
	}

	condition, err := p.parseCondition()
	if err != nil {
		return nil, err
	}

	err = p.parseExpressionBlockClose()
	if err != nil {
		return nil, err
	}

	return ast.NewSetCall(ident, condition), nil
}

func (p *parser) parseList() (ast.AST, error) {
	token, err := p.expectedAndConsumeValue(lexer.LeftBracketToken)
	if err != nil {
		return nil, err
	}
	list := ast.NewList(token)

	for !p.peekIs(lexer.RightBracketToken) {
		item, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		list.Append(item)

		// If the next character is a comma token then consume it
		// but a list may not be comma seperated in lists
		if p.peekIs(lexer.CommaToken) {
			_ = p.next()
		}
	}

	_, err = p.expectedAndConsumeValue(lexer.RightBracketToken)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (p *parser) parseCallBlock() (ast.AST, error) {
	ident, err := p.expectedAndConsumeValue(lexer.IdentToken)
	if err != nil {
		return nil, err
	}

	ifs, err := p.parseFunctionCall(ident)
	if err != nil {
		return nil, err
	}

	if err := p.parseExpressionBlockClose(); err != nil {
		return nil, err
	}

	cb := ast.NewCallBlock(ident, ifs)

	if err := p.parseBodyUntilAtom("endcall", cb); err != nil {
		return nil, err
	}

	return cb, nil
}

func (p *parser) parseUnsupportedBlockType(token *lexer.Token) (ast.AST, error) {
	block := ast.NewUnsupportedExpressionBlock(token)
	end := "end" + token.Value

	token = p.next()
	for token.Type != lexer.EOFToken {
		token = p.next()

		if (token.Type == lexer.ExpressionBlockOpen || token.Type == lexer.ExpressionBlockOpenTrim) &&
			p.peekIs(lexer.IdentToken) && p.peek().Value == end {
			_ = p.next() // consume

			err := p.parseExpressionBlockClose()
			if err != nil {
				return nil, err
			}

			return block, nil
		}
	}

	return block, nil
}

func (p *parser) parseMap() (ast.AST, error) {
	openingToken, err := p.expectedAndConsumeValue(lexer.LeftBraceToken)
	if err != nil {
		return nil, err
	}
	m := ast.NewMap(openingToken)

	for !p.peekIs(lexer.RightBraceToken) {
		key, err := p.expectedAndConsumeValue(lexer.StringToken)
		if err != nil {
			return nil, err
		}

		_, err = p.expectedAndConsumeValue(lexer.ColonToken)
		if err != nil {
			return nil, err
		}

		value, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		//value := p.next()
		//if value.Type != lexer.StringToken && value.Type != lexer.NumberToken && value.Type != lexer.NullToken {
		//	return nil, p.errorAt(value, "Expected string, number or null")
		//}

		m.Put(key, value)

		if !p.peekIs(lexer.CommaToken) {
			break
		}
		_ = p.next()
	}

	if _, err := p.expectedAndConsumeValue(lexer.RightBraceToken); err != nil {
		return nil, err
	}

	return m, nil
}

func isAtomOrEOF(expectedAtom string, node ast.AST) bool {
	if atom, ok := node.(*ast.AtomExpressionBlock); ok {
		return atom.Token().Value == expectedAtom
	}

	_, ok := node.(*ast.EndOfFile)
	return ok
}
