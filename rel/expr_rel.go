package rel

import (
	"strings"

	"github.com/arr-ai/wbnf/parser"
	"github.com/go-errors/errors"
)

// NewRelationExpr returns a new relation for the given data.
func NewRelationExpr(scanner parser.Scanner, names []string, tuples ...[]Expr) (Expr, error) {
	elements := make([]Expr, len(tuples))
	stringCharTuples := 0
	arrayItemTuples := 0
	dictEntryTuples := 0
	for i, tuple := range tuples {
		if len(tuple) != len(names) {
			return nil, errors.Errorf(
				"heading-tuple mismatch: %v vs %v", names, tuple)
		}
		attrs := make([]AttrExpr, len(names))
		for i, name := range names {
			attrs[i] = AttrExpr{ExprScanner{scanner}, name, tuple[i]}
		}
		if len(attrs) == 2 {
			if attrs[1].name == "@" {
				attrs[0], attrs[1] = attrs[1], attrs[0]
			}
			if attrs[0].name == "@" && strings.HasPrefix(attrs[1].name, "@") {
				switch attrs[1].name {
				case StringCharAttr:
					elements[i] = NewStringCharTupleExpr(scanner, attrs[0].expr, attrs[1].expr)
					stringCharTuples++
				case ArrayItemAttr:
					elements[i] = NewArrayItemTupleExpr(scanner, attrs[0].expr, attrs[1].expr)
					arrayItemTuples++
				case DictValueAttr:
					elements[i] = NewDictEntryTupleExpr(scanner, attrs[0].expr, attrs[1].expr)
					dictEntryTuples++
				default:
					elements[i] = NewTupleExpr(scanner, attrs...)
				}
				continue
			}
		}
		elements[i] = NewTupleExpr(scanner, attrs...)
	}
	switch len(elements) {
	case stringCharTuples:
		charExprs := make([]Expr, 0, len(elements))
		for _, e := range elements {
			charExprs = append(charExprs, e.(StringCharTupleExpr))
		}
		// TODO: Implement NewStringCharSetExpr.
		return NewSetExpr(scanner, charExprs...), nil
	case arrayItemTuples:
		entryExprs := make([]Expr, 0, len(elements))
		for _, e := range elements {
			entryExprs = append(entryExprs, e.(ArrayItemTupleExpr))
		}
		// TODO: Implement NewArrayItemSetExpr.
		return NewSetExpr(scanner, entryExprs...), nil
	case dictEntryTuples:
		entryExprs := make([]DictEntryTupleExpr, 0, len(elements))
		for _, e := range elements {
			entryExprs = append(entryExprs, e.(DictEntryTupleExpr))
		}
		return NewDictExpr(scanner, true, false, entryExprs...), nil
	}
	return NewSetExpr(scanner, elements...), nil
}

func newSetBinExpr(scanner parser.Scanner, a, b Expr, op string, f func(x, y Set) (Set, error)) Expr {
	return newBinExpr(scanner, a, b, op, "(%s "+op+" %s)",
		func(a, b Value, _ Scope) (Value, error) {
			if x, ok := a.(Set); ok {
				if y, ok := b.(Set); ok {
					return f(x, y)
				}
				return nil, errors.Errorf(op+" rhs must be a Set, not %T", b)
			}
			return nil, errors.Errorf(op+" lhs must be a Set, not %T", a)
		})
}

func newSetBinExprNoError(scanner parser.Scanner, a, b Expr, op string, f func(x, y Set) Set) Expr {
	return newSetBinExpr(scanner, a, b, op, func(x, y Set) (Set, error) {
		return f(x, y), nil
	})
}

// NewJoinExpr evaluates a <&> b.
func NewJoinExpr(scanner parser.Scanner, a, b Expr) Expr {
	return newSetBinExprNoError(scanner, a, b, "<&>", Join)
}

// NewUnionExpr evaluates a | b.
func NewUnionExpr(scanner parser.Scanner, a, b Expr) Expr {
	return newSetBinExprNoError(scanner, a, b, "|", Union)
}

// NewDiffExpr evaluates a &~ b.
func NewDiffExpr(scanner parser.Scanner, a, b Expr) Expr {
	return newSetBinExprNoError(scanner, a, b, "&~", Difference)
}

// NewSymmDiffExpr evaluates a ~~ b.
func NewSymmDiffExpr(scanner parser.Scanner, a, b Expr) Expr {
	return newSetBinExprNoError(scanner, a, b, "~~", SymmetricDifference)
}

// NewConcatExpr evaluates a ++ b.
func NewConcatExpr(scanner parser.Scanner, a, b Expr) Expr {
	return newSetBinExpr(scanner, a, b, "++", Concatenate)
}
