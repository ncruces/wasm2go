package main

import (
	"strconv"
	"strings"
	"unicode"
)

func exported(name string) string {
	return identifier("X" + name)
}

func unexported(name string) string {
	return identifier("_" + name)
}

func identifier(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return '_'
	}, s)
}

func local[T interface{ int | uint64 }](i T) string {
	return "v" + strconv.Itoa(int(i))
}
