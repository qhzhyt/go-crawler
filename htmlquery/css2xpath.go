package htmlquery

// from github.com/moovweb/
import (
	"fmt"
	// "rubex"
	"regexp"
	"strings"
)

// Lexeme Lexeme
type Lexeme int

// all types
const (
	SPACES = iota
	COMMA
	UNIVERSAL
	TYPE
	ELEMENT
	CLASS
	ID
	LBRACKET
	RBRACKET
	AttrName
	AttrValue
	EQUALS
	ContainsClass
	DashPrefixed
	StartsWith
	EndsWith
	CONTAINS
	MatchOp
	PseudoClass
	FirstChild
	FirstOfType
	NthChild
	NthOfType
	OnlyChild
	OnlyOfType
	LastChild
	LastOfType
	NOT
	LPAREN
	RPAREN
	COEFFICIENT
	SIGNED
	UNSIGNED
	ODD
	EVEN
	N
	OPERATOR
	PLUS
	MINUS
	BINOMIAL
	AdjacentTo
	PRECEDES
	ParentOf
	AncestorOf
	// and a counter ... I can't believe I didn't think of this sooner
	NumLexemes
)

var pattern [NumLexemes]string
var matcher [NumLexemes]*regexp.Regexp

func init() {
	// some overlap in here, but it'll make the parsing functions clearer
	pattern[SPACES] = `\s+`
	pattern[COMMA] = `\s*,`
	pattern[UNIVERSAL] = `\*`
	pattern[TYPE] = `[_a-zA-Z]\w*`
	pattern[ELEMENT] = `(\*|[_a-zA-Z]\w*)`
	pattern[CLASS] = `\.[-\w]+`
	pattern[ID] = `\#[-\w]+`
	pattern[LBRACKET] = `\[`
	pattern[RBRACKET] = `\]`
	pattern[AttrName] = `[-_:a-zA-Z][-\w:.]*`
	pattern[AttrValue] = `("(\\.|[^"\\])*"|'(\\.|[^'\\])*')`
	pattern[EQUALS] = `=`
	pattern[ContainsClass] = `~=`
	pattern[DashPrefixed] = `\|=`
	pattern[StartsWith] = `\^=`
	pattern[EndsWith] = `\$=`
	pattern[CONTAINS] = `\*=`
	pattern[MatchOp] = "(" + strings.Join([]string{pattern[EQUALS], pattern[ContainsClass], pattern[DashPrefixed], pattern[StartsWith], pattern[EndsWith], pattern[CONTAINS]}, "|") + ")"
	pattern[PseudoClass] = `:[-a-z]+`
	pattern[FirstChild] = `:first-child`
	pattern[FirstOfType] = `:first-of-type`
	pattern[NthChild] = `:nth-child`
	pattern[NthOfType] = `:nth-of-type`
	pattern[OnlyChild] = `:only-child`
	pattern[OnlyOfType] = `:only-of-type`
	pattern[LastChild] = `:last-child`
	pattern[LastOfType] = `:last-of-type`
	pattern[NOT] = `:not`
	pattern[LPAREN] = `\s*\(`
	pattern[RPAREN] = `\s*\)`
	pattern[COEFFICIENT] = `[-+]?(\d+)?`
	pattern[SIGNED] = `[-+]?\d+`
	pattern[UNSIGNED] = `\d+`
	pattern[ODD] = `odd`
	pattern[EVEN] = `even`
	pattern[N] = `[nN]`
	pattern[OPERATOR] = `[-+]`
	pattern[PLUS] = `\+`
	pattern[MINUS] = `-`
	pattern[BINOMIAL] = strings.Join([]string{pattern[COEFFICIENT], pattern[N], `\s*`, pattern[OPERATOR], `\s*`, pattern[UNSIGNED]}, "")
	pattern[AdjacentTo] = `\s*\+`
	pattern[PRECEDES] = `\s*~`
	pattern[ParentOf] = `\s*>`
	pattern[AncestorOf] = `\s+`
	for i, p := range pattern {
		matcher[i] = regexp.MustCompile(`\A` + p)
	}
}

// Scope Scope
type Scope int

// all Scopes
const (
	GLOBAL = iota
	LOCAL
)

// CSS2Xpath 将css转为xpath
func CSS2Xpath(css string, scope Scope) string {
	xpath, _ := selectors([]byte(css), scope)
	return xpath
}

func selectors(input []byte, scope Scope) (string, []byte) {
	x, input := selector(input, scope)
	xs := []string{x}
	for peek(COMMA, input) {
		_, input = token(COMMA, input)
		x, input = selector(input, scope)
		xs = append(xs, x)
	}
	return strings.Join(xs, " | "), input
}

func selector(input []byte, scope Scope) (string, []byte) {
	var combinator Lexeme
	var xs []string
	if scope == LOCAL {
		xs = []string{"."}
	}
	if matched, remainder := token(ParentOf, input); matched != nil {
		combinator, input = ParentOf, remainder
	} else {
		combinator = AncestorOf
	}
	x, input := sequence(input, combinator)
	xs = append(xs, x)
	for {
		if matched, remainder := token(AdjacentTo, input); matched != nil {
			combinator, input = AdjacentTo, remainder
		} else if matched, remainder := token(PRECEDES, input); matched != nil {
			combinator, input = PRECEDES, remainder
		} else if matched, remainder := token(ParentOf, input); matched != nil {
			combinator, input = ParentOf, remainder
		} else if matched, remainder := token(AncestorOf, input); matched != nil {
			combinator, input = AncestorOf, remainder
		} else {
			break
		}
		x, input = sequence(input, combinator)
		xs = append(xs, x)
	}
	return strings.Join(xs, ""), input
}

func sequence(input []byte, combinator Lexeme) (string, []byte) {
	_, input = token(SPACES, input)
	x, ps := "", []string{}

	switch combinator {
	case AncestorOf:
		x = "/descendant-or-self::*/*"
	case ParentOf:
		x = "/child::*"
	case PRECEDES:
		x = "/following-sibling::*"
	case AdjacentTo:
		x = "/following-sibling::*"
		ps = append(ps, "position()=1")
	}

	if e, remainder := token(ELEMENT, input); e != nil {
		input = remainder
		if len(ps) > 0 {
			ps = append(ps, " and ")
		}
		ps = append(ps, "self::"+string(e))
		if !(peek(ID, input) || peek(CLASS, input) || peek(PseudoClass, input) || peek(LBRACKET, input)) {
			pstr := strings.Join(ps, "")
			if pstr != "" {
				pstr = fmt.Sprintf("[%s]", pstr)
			}
			return x + pstr, input
		}
	}
	q, input, connective := qualifier(input)
	if q == "" {
		panic("Invalid CSS selector")
	}
	if len(ps) > 0 {
		ps = append(ps, connective)
	}
	ps = append(ps, q)
	for q, r, c := qualifier(input); q != ""; q, r, c = qualifier(input) {
		ps, input = append(ps, c, q), r
	}
	pstr := strings.Join(ps, "")
	if combinator != NOT {
		pstr = fmt.Sprintf("[%s]", pstr)
	}
	return x + pstr, input
}

func qualifier(input []byte) (string, []byte, string) {
	p, connective := "", ""
	if t, remainder := token(CLASS, input); t != nil {
		p = fmt.Sprintf(`contains(concat(" ", @class, " "), " %s ")`, string(t[1:]))
		input = remainder
		connective = " and "
	} else if t, remainder := token(ID, input); t != nil {
		p, input, connective = fmt.Sprintf(`@id="%s"`, string(t[1:])), remainder, " and "
	} else if peek(PseudoClass, input) {
		p, input, connective = pseudoClass(input)
	} else if peek(LBRACKET, input) {
		p, input = attribute(input)
		connective = " and "
	}
	return p, input, connective
}

func pseudoClass(input []byte) (string, []byte, string) {
	class, input := token(PseudoClass, input)
	var p, connective string
	switch string(class) {
	case ":first-child":
		p, connective = "position()=1", " and "
	case ":first-of-type":
		p, connective = "position()=1", "]["
	case ":last-child":
		p, connective = "position()=last()", " and "
	case ":last-of-type":
		p, connective = "position()=last()", "]["
	case ":only-child":
		p, connective = "position() = 1 and position() = last()", " and "
	case ":only-of-type":
		p, connective = "position() = 1 and position() = last()", "]["
	case ":nth-child":
		p, input = nth(input)
		connective = " and "
	case ":nth-of-type":
		p, input = nth(input)
		connective = "]["
	case ":not":
		p, input = negate(input)
		connective = " and "
	default:
		panic(`Cannot convert CSS pseudo-class "` + string(class) + `" to XPath.`)
	}
	return p, input, connective
}

func nth(input []byte) (string, []byte) {
	lparen, input := token(LPAREN, input)
	if lparen == nil {
		panic(":nth-child and :nth-of-type require an parenthesized argument")
	}
	_, input = token(SPACES, input)
	var expr string
	if e, rem := token(EVEN, input); e != nil {
		expr, input = "position() mod 2 = 0", rem
	} else if e, rem := token(ODD, input); e != nil {
		expr, input = "position() mod 2 = 1", rem
	} else if e, _ := token(BINOMIAL, input); e != nil {
		var coefficient, operator, constant []byte
		coefficient, input = token(COEFFICIENT, input)
		switch string(coefficient) {
		case "", "+":
			coefficient = []byte("1")
		case "-":
			coefficient = []byte("-1")
		}
		_, input = token(N, input)
		_, input = token(SPACES, input)
		operator, input = token(OPERATOR, input)
		_, input = token(SPACES, input)
		constant, input = token(UNSIGNED, input)
		expr = fmt.Sprintf("(position() %s %s) mod %s = 0", invert(string(operator)), string(constant), string(coefficient))
	} else if e, rem := token(SIGNED, input); e != nil {
		expr, input = "position() = "+string(e), rem
	} else {
		panic("Invalid argument to :nth-child or :nth-of-type.")
	}
	fmt.Println(string(input))
	_, input = token(SPACES, input)
	rparen, input := token(RPAREN, input)
	if rparen == nil {
		panic("Unterminated argument to :nth-child or :nth-of-type.")
	}
	return expr, input
}

func invert(op string) string {
	op = strings.TrimSpace(op)
	if op == "+" {
		op = "-"
	} else {
		op = "+"
	}
	return op
}

func negate(input []byte) (string, []byte) {
	_, input = token(SPACES, input)
	lparen, input := token(LPAREN, input)
	if lparen == nil {
		panic(":not requires a parenthesized argument.")
	}
	_, input = token(SPACES, input)
	p, input := sequence(input, NOT)
	_, input = token(SPACES, input)
	rparen, input := token(RPAREN, input)
	if rparen == nil {
		panic("Unterminated argument to :not.")
	}
	return fmt.Sprintf("not(%s)", p), input
}

func attribute(input []byte) (string, []byte) {
	_, input = token(LBRACKET, input)
	_, input = token(SPACES, input)
	name, input := token(AttrName, input)
	if name == nil {
		panic("Attribute selector requires an attribute name.")
	}
	_, input = token(SPACES, input)
	if rbracket, remainder := token(RBRACKET, input); rbracket != nil {
		return "@" + string(name), remainder
	}
	op, input := token(MatchOp, input)
	if op == nil {
		panic("Missing operator in attribute selector.")
	}
	_, input = token(SPACES, input)
	val, input := token(AttrValue, input)
	if val == nil {
		panic("Missing value in attribute selector.")
	}
	_, input = token(SPACES, input)
	rbracket, input := token(RBRACKET, input)
	if rbracket == nil {
		panic("Unterminated attribute selector.")
	}
	var expr string
	n, v := string(name), string(val)
	switch string(op) {
	case "=":
		expr = fmt.Sprintf("@%s=%s", n, v)
	case "~=":
		expr = fmt.Sprintf(`contains(concat(" ", @%s, " "), concat(" ", %s, " "))`, n, v)
	case "|=":
		expr = fmt.Sprintf(`(@%s=%s or starts-with(@%s, concat(%s, "-")))`, n, v, n, v)
	case "^=":
		expr = fmt.Sprintf("starts-with(@%s, %s)", n, v)
	case "$=":
		// oy, libxml doesn't support ends-with
		// generate something like: div[substring(@class, string-length(@class) - string-length('foo') + 1) = 'foo']
		expr = fmt.Sprintf("substring(@%s, string-length(@%s) - string-length(%s) + 1) = %s", n, n, v, v)
	case "*=":
		expr = fmt.Sprintf("contains(@%s, %s)", n, v)
	}
	return expr, input
}

func token(lexeme Lexeme, input []byte) ([]byte, []byte) {
	matched := matcher[lexeme].Find(input)
	length := len(matched)
	if length == 0 {
		matched = nil
	}
	return matched, input[length:]
}

func peek(lexeme Lexeme, input []byte) bool {
	matched, _ := token(lexeme, input)
	return matched != nil
}
