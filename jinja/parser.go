package jinja

import (
	"errors"
	"fmt"

	"ddbt/fs"
	"ddbt/jinja/ast"
	"ddbt/jinja/lexer"
)

type parser struct {
	tokens         []*lexer.Token
	nextTokenIndex int
}

func Parse(file *fs.File) error {
	// TODO: Change tokens into a channel and lex the file async
	tokens, err := lexer.LexFile(file.Path)
	if err != nil {
		return err
	}

	p := &parser{
		tokens:         tokens,
		nextTokenIndex: 0,
	}

	body := ast.NewBody(p.peek())
	for {
		node, err := p.parse()
		if err != nil {
			return err
		}

		body.Append(node)

		if _, ok := node.(*ast.EndOfFile); ok {
			break
		}
	}

	fmt.Println(body.String())

	return nil
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
		fmt.Sprintf("%s at %d:%d", error, atToken.Start.Row, atToken.Start.Column),
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
	t := p.next()

	switch t.Type {
	case lexer.EOFToken:
		return ast.NewEndOfFile(t), nil

	case lexer.TextToken:
		return ast.NewTextBlock(t), nil

	case lexer.ExpressionBlockOpen:
		return p.parseExpressionBlock()

	case lexer.TemplateBlockOpen:
		return p.parseTemplateBlock()

	default:
		return nil, p.errorAt(t, "unexpected token: "+t.DisplayString())
	}
}

func (p *parser) parseExpressionBlock() (ast.AST, error) {
	p.consumeIfPossible(lexer.MinusToken) // FIXME: this is meant to strip the whitespace from before this block!

	t, err := p.expectedAndConsumeValue(lexer.IdentToken)
	if err != nil {
		return nil, err
	}

	// Assuming that atom expression blocks are always end markers, return early
	// i.e. {% endmacro %}  or {% endif %} or {% endfor %} or {% else %}
	if p.peekIs(lexer.ExpressionBlockClose) {
		_ = p.next() // consume the closing block
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
		return p.parseIfStatement()

	default:
		return nil, p.errorAt(t, "Expected `macro` or `set` got "+t.Value)
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

			if defaultValue.Type != lexer.StringToken && defaultValue.Type != lexer.NumberToken {
				return nil, p.errorAt(
					defaultValue,
					fmt.Sprintf("Expected string or number, got: %s", defaultValue.Type),
				)
			}
		}

		// Add the parameter
		macro.AddParameter(paramName.Value, defaultValue)

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

	_, err = p.expectedAndConsumeValue(lexer.ExpressionBlockClose)
	if err != nil {
		return nil, err
	}

	if err := p.parseBodyUntilAtom("endmacro", macro); err != nil {
		return nil, err
	}

	return macro, nil
}

func (p *parser) parseTemplateBlock() (ast.AST, error) {
	var node ast.AST
	var err error

	if p.peekIs(lexer.StringToken) {
		node = ast.NewTextBlock(p.next())
	} else {
		ident, err := p.expectedAndConsumeValue(lexer.IdentToken)
		if err != nil {
			return nil, err
		}

		switch {
		//Variable
		case p.peekIs(lexer.TemplateBlockClose) ||
			p.peekIs(lexer.PeriodToken) ||
			p.peekIs(lexer.LeftBracketToken):

			node, err = p.parseVariable(ident)
			if err != nil {
				return nil, err
			}

		case p.peekIs(lexer.LeftParenthesesToken):
			node, err = p.parseFunctionCall(ident)
			if err != nil {
				return nil, err
			}

		default:
			return nil, p.errorAt(ident, "Unable to determine how to process template block")
		}
	}

	// If this an if statement in the form "x if true"
	if p.peekIs(lexer.IdentToken) && p.peek().Value == "if" {
		ifToken := p.next() // consume the if

		condition, err := p.parseCondition()
		if err != nil {
			return nil, err
		}

		is := ast.NewIfStatement(ifToken, condition)
		is.AppendBody(node)

		node = is
	}

	_, err = p.expectedAndConsumeValue(lexer.TemplateBlockClose)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (p *parser) parseFunctionCall(ident *lexer.Token) (ast.AST, error) {
	funcCall := ast.NewFunctionCall(ident)

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
		var variable ast.AST

		namedArg := ""

		if p.peekIs(lexer.StringToken) {
			variable = ast.NewTextBlock(p.next())
		} else if p.peekIs(lexer.NumberToken) {
			variable = ast.NewNumber(p.next())
		} else {
			ident, err := p.expectedAndConsumeValue(lexer.IdentToken)
			if err != nil {
				return err
			}

			// check if we're making a named argument call; i.e.
			//   foo(bar="hello")
			if p.peekIs(lexer.EqualsToken) {
				// Then this is a named parameter call
				_ = p.next() // consume the =

				namedArg = ident.Value

				if p.peekIs(lexer.StringToken) {
					variable = ast.NewTextBlock(p.next())
				} else if p.peekIs(lexer.NumberToken) {
					variable = ast.NewNumber(p.next())
				} else {
					variable, err = p.parseVariable(nil)
					if err != nil {
						return err
					}
				}
			} else {
				variable, err = p.parseVariable(ident)
				if err != nil {
					return err
				}
			}
		}

		// Check if the variable has a maths operation
		variable, err := p.parsePossibleMathsOps(variable)
		if err != nil {
			return err
		}

		node.AddArgument(namedArg, variable)

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

func (p *parser) parsePossibleMathsOps(a ast.AST) (ast.AST, error) {
	// FIXME: Implement
	switch p.peek().Type {
	case lexer.MultiplyToken:
	case lexer.DivideToken:
	case lexer.PlusToken:
	case lexer.MinusToken:
	}

	return a, nil
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

	// Is this a reference into the variable? (i.e. `a.b`)
	if p.peekIs(lexer.PeriodToken) {
		_ = p.next() // consume the `.`

		subvar, err := p.parseVariable(nil)
		if err != nil {
			return nil, err
		}

		variable.SetSub(subvar)

	} else if p.peekIs(lexer.LeftBracketToken) {
		// Otherwise is this a map lookup? (i.e. `a["b"]`)
		_ = p.next() // consume the [

		str, err := p.expectedAndConsumeValue(lexer.StringToken)
		if err != nil {
			return nil, err
		}

		subvar := ast.NewVariable(str)
		variable.SetMapLookup(subvar)

		_, err = p.expectedAndConsumeValue(lexer.RightBracketToken)
		if err != nil {
			return nil, err
		}
	} else if p.peekIs(lexer.LeftParenthesesToken) {
		// This variable is being treated like a function call!

		variable.IsCalledAsFunc()
		if err := p.parseArgumentList(variable); err != nil {
			return nil, err
		}
	}

	return variable, nil
}

func (p *parser) parseForLoop() (ast.AST, error) {
	iteratorName, err := p.expectedAndConsumeValue(lexer.IdentToken)
	if err != nil {
		return nil, err
	}

	if err := p.expectedAndConsumeIdentifier("in"); err != nil {
		return nil, err
	}

	list, err := p.parseVariable(nil)
	if err != nil {
		return nil, err
	}

	p.consumeIfPossible(lexer.MinusToken) // FIXME: this should strip whitespace after

	_, err = p.expectedAndConsumeValue(lexer.ExpressionBlockClose)
	if err != nil {
		return nil, err
	}

	forLoop := ast.NewForLoop(iteratorName, list)

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

func (p *parser) parseIfStatement() (ast.AST, error) {
	conditionToken := p.peek()

	condition, err := p.parseCondition()
	if err != nil {
		return nil, err
	}

	is := ast.NewIfStatement(conditionToken, condition)

	_, err = p.expectedAndConsumeValue(lexer.ExpressionBlockClose)
	if err != nil {
		return nil, err
	}

	if err := p.parseBodyUntilAtom("endif", is); err != nil {
		return nil, err
	}

	return is, nil
}

func (p *parser) parseCondition() (ast.AST, error) {
	var condition ast.AST
	var err error

	if p.peekIs(lexer.StringToken) {
		condition = ast.NewTextBlock(p.next())

	} else if p.peekIs(lexer.LeftParenthesesToken) {
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
	} else {
		ident, err := p.expectedAndConsumeValue(lexer.IdentToken)
		if err != nil {
			return nil, err
		}

		switch {
		case ident.Value == "not": // prefix operator
			sub, err := p.parseCondition()
			if err != nil {
				return nil, err
			}

			condition = ast.NewNotOperator(ident, sub)

		case p.peekIs(lexer.LeftParenthesesToken): // function call
			condition, err = p.parseFunctionCall(ident)
			if err != nil {
				return nil, err
			}

		default:
			condition, err = p.parseVariable(ident)
			if err != nil {
				return nil, err
			}
		}
	}

	if p.peekIs(lexer.IsEqualsToken) {
		_ = p.next() // consume ==

		otherSide, err := p.parseCondition()
		if err != nil {
			return nil, err
		}

		condition = ast.NewEqualsCondition(condition, otherSide)
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

		condition = ast.NewAndCondition(condition, otherSide)
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

	_, err = p.expectedAndConsumeValue(lexer.ExpressionBlockClose)
	if err != nil {
		return nil, err
	}

	return ast.NewSetCall(ident, condition), nil
}

func isAtomOrEOF(expectedAtom string, node ast.AST) bool {
	if atom, ok := node.(*ast.AtomExpressionBlock); ok {
		return atom.Token().Value == expectedAtom
	}

	_, ok := node.(*ast.EndOfFile)
	return ok
}
