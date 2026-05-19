package calculator

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Result struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Action   Action `json:"action"`
}

type Action struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func Query(query string) (Result, bool) {
	query = strings.TrimSpace(query)
	if !strings.HasPrefix(query, "=") {
		return Result{}, false
	}
	value, err := Evaluate(strings.TrimSpace(strings.TrimPrefix(query, "=")))
	if err != nil {
		return Result{}, false
	}
	text := formatFloat(value)
	return Result{
		Title:    text,
		Subtitle: "Copy result",
		Action: Action{
			Type: "clipboard.write",
			Text: text,
		},
	}, true
}

func Evaluate(expr string) (float64, error) {
	parser := expressionParser{input: expr}
	value, err := parser.parseExpression()
	if err != nil {
		return 0, err
	}
	parser.skipSpace()
	if parser.pos != len(parser.input) {
		return 0, fmt.Errorf("unexpected token %q", parser.input[parser.pos:])
	}
	return value, nil
}

type expressionParser struct {
	input string
	pos   int
}

func (p *expressionParser) parseExpression() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpace()
		if !p.consume('+') && !p.consume('-') {
			return left, nil
		}
		op := p.input[p.pos-1]
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
}

func (p *expressionParser) parseTerm() (float64, error) {
	left, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpace()
		if !p.consume('*') && !p.consume('/') {
			return left, nil
		}
		op := p.input[p.pos-1]
		right, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		if op == '*' {
			left *= right
		} else {
			if right == 0 {
				return 0, errors.New("division by zero")
			}
			left /= right
		}
	}
}

func (p *expressionParser) parseFactor() (float64, error) {
	p.skipSpace()
	if p.consume('-') {
		value, err := p.parseFactor()
		return -value, err
	}
	if p.consume('(') {
		value, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		p.skipSpace()
		if !p.consume(')') {
			return 0, errors.New("missing closing parenthesis")
		}
		return value, nil
	}
	return p.parseNumber()
}

func (p *expressionParser) parseNumber() (float64, error) {
	p.skipSpace()
	start := p.pos
	for p.pos < len(p.input) {
		r := rune(p.input[p.pos])
		if !unicode.IsDigit(r) && r != '.' {
			break
		}
		p.pos++
	}
	if start == p.pos {
		return 0, errors.New("expected number")
	}
	return strconv.ParseFloat(p.input[start:p.pos], 64)
}

func (p *expressionParser) skipSpace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *expressionParser) consume(ch byte) bool {
	p.skipSpace()
	if p.pos >= len(p.input) || p.input[p.pos] != ch {
		return false
	}
	p.pos++
	return true
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
