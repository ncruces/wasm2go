package util

import "go/ast"

// UnwrapConversion avoids redundant conversions.
func UnwrapConversion(expr ast.Expr, target string) (base ast.Expr, done bool) {
	targetBits := intBits[target]

	for {
		fn, current, arg := unwrapCall(expr)

		// If the type of expr equals our target, we are done.
		if current == target {
			return expr, true
		}

		// Are target and expr both integer conversions?
		exprBits := intBits[fn]
		if targetBits == 0 || exprBits == 0 {
			break
		}

		// Is the type of arg known to be an integer?
		_, inner, _ := unwrapCall(arg)
		if intBits[inner] == 0 {
			break
		}

		// If we reach here, we're dealing with integers and want to do:
		//   target(fn(arg))

		// We can remove fn if it has at least as many bits as target.
		if targetBits > exprBits {
			break
		}
		expr = arg
	}

	// We peeled as much as safely possible, the types still don't match.
	return expr, false
}

func unwrapCall(expr ast.Expr) (fn, typ string, arg ast.Expr) {
	ce, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", "", nil
	}
	id, ok := ce.Fun.(*ast.Ident)
	if !ok {
		return "", "", nil
	}
	if len(ce.Args) == 1 {
		arg = ce.Args[0]
	}
	return id.Name, funcTypes[id.Name], arg
}

var (
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

	funcTypes = map[string]string{
		// Literals.

		"i32": "int32",
		"i64": "int64",
		"f32": "float32",
		"f64": "float64",

		// Conversions.

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

		// Helpers.

		"load16": "uint16",
		"load32": "uint32",
		"load64": "uint64",

		"i32_div_s": "int32",
		"i32_neg_s": "int32",
		"i32_shl":   "int32",
		"i32_shr_s": "int32",
		"i32_shr_u": "int32",
		"i32_rotl":  "int32",
		"i32_rotr":  "int32",

		"i64_div_s": "int64",
		"i64_neg_s": "int64",
		"i64_shl":   "int64",
		"i64_shr_s": "int64",
		"i64_shr_u": "int64",
		"i64_rotl":  "int64",
		"i64_rotr":  "int64",

		// Float to int conversions.

		"i32_trunc_f64_s":     "int32",
		"i32_trunc_f32_s":     "int32",
		"i32_trunc_f64_u":     "int32",
		"i32_trunc_f32_u":     "int32",
		"i32_trunc_sat_f64_s": "int32",
		"i32_trunc_sat_f32_s": "int32",
		"i32_trunc_sat_f64_u": "int32",
		"i32_trunc_sat_f32_u": "int32",

		"i64_trunc_f64_s":     "int64",
		"i64_trunc_f32_s":     "int64",
		"i64_trunc_f64_u":     "int64",
		"i64_trunc_f32_u":     "int64",
		"i64_trunc_sat_f64_s": "int64",
		"i64_trunc_sat_f32_s": "int64",
		"i64_trunc_sat_f64_u": "int64",
		"i64_trunc_sat_f32_u": "int64",

		// Atomics.

		"atomic_load8":    "uint8",
		"atomic_add8":     "uint8",
		"atomic_sub8":     "uint8",
		"atomic_and8":     "uint8",
		"atomic_or8":      "uint8",
		"atomic_xor8":     "uint8",
		"atomic_xchg8":    "uint8",
		"atomic_cmpxchg8": "uint8",

		"atomic_load16":    "uint16",
		"atomic_add16":     "uint16",
		"atomic_sub16":     "uint16",
		"atomic_and16":     "uint16",
		"atomic_or16":      "uint16",
		"atomic_xor16":     "uint16",
		"atomic_xchg16":    "uint16",
		"atomic_cmpxchg16": "uint16",

		"atomic_load32":    "uint32",
		"atomic_add32":     "uint32",
		"atomic_sub32":     "uint32",
		"atomic_and32":     "uint32",
		"atomic_or32":      "uint32",
		"atomic_xor32":     "uint32",
		"atomic_xchg32":    "uint32",
		"atomic_cmpxchg32": "uint32",

		"atomic_load64":    "uint64",
		"atomic_add64":     "uint64",
		"atomic_sub64":     "uint64",
		"atomic_and64":     "uint64",
		"atomic_or64":      "uint64",
		"atomic_xor64":     "uint64",
		"atomic_xchg64":    "uint64",
		"atomic_cmpxchg64": "uint64",
	}
)
