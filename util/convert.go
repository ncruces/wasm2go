package util

import "go/ast"

// UnwrapConversion avoids redundant conversions.
func UnwrapConversion(expr ast.Expr, target string) (base ast.Expr, done bool) {
	targetBits := intBits[target]

	for {
		name, current, arg := unwrapCall(expr)

		// If the type of expr equals our target, we are done.
		if current == target {
			return expr, true
		}

		// Are we converting expr (itself an integer conversion) to an integer?
		exprBits := intBits[name]
		if targetBits == 0 || exprBits == 0 {
			break
		}

		// Is the type of arg known to be an integer?
		_, inner, _ := unwrapCall(arg)
		if intBits[inner] == 0 {
			break
		}

		// We can't peel away narrowing conversions.
		if targetBits > exprBits {
			break
		}

		// Otherwise, the conversion is redundant.
		expr = arg
	}

	// We peeled as much as safely possible, the types still don't match.
	return expr, false
}

// unwrapCall checks if an expression is a single argument function call.
func unwrapCall(expr ast.Expr) (name, typ string, arg ast.Expr) {
	ce, ok := expr.(*ast.CallExpr)
	if !ok || len(ce.Args) != 1 {
		return "", "", nil
	}
	id, ok := ce.Fun.(*ast.Ident)
	if !ok {
		return "", "", nil
	}
	return id.Name, funcTypes[id.Name], ce.Args[0]
}

var (
	funcTypes = map[string]string{
		"i32": "int32",
		"i64": "int64",
		"f32": "float32",
		"f64": "float64",

		"int":   "int",
		"int8":  "int8",
		"int16": "int16",
		"int32": "int32",
		"int64": "int64",

		"uint":   "uint",
		"uint8":  "uint8",
		"uint16": "uint16",
		"uint32": "uint32",
		"uint64": "uint64",

		"float32": "float32",
		"float64": "float64",

		"load16": "uint16",
		"load32": "uint32",
		"load64": "uint64",
	}

	intBits = map[string]int8{
		"int8":   8,
		"uint8":  8,
		"int16":  16,
		"uint16": 16,
		"int32":  32,
		"uint32": 32,
		"int64":  64,
		"uint64": 64,
		// For the purpose of checking if a conversion is widening or narrowing,
		// machine word sized types act as if they have 32<N<64 bits.
		"int":  63,
		"uint": 63,
	}
)
