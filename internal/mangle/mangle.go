package mangle

import (
	"go/ast"
	"hash/crc32"
	"strconv"
	"strings"
	"unicode"
)

type Kind int

const (
	Exported Kind = iota
	Internal
	Local
)

func Name(name string, kind Kind) string {
	var buf strings.Builder
	buf.Grow(len(name))

	switch kind {
	case Exported:
		buf.WriteByte('X')
	case Internal:
		buf.WriteByte('_')
	}

	var suffix bool
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			suffix = true
			r = '_'
		}
		buf.WriteRune(r)
	}
	if suffix && kind != Local {
		buf.WriteByte('_')
		const mod = 36 * 36 * 36 * 36 * 36 * 36
		table := crc32.MakeTable(crc32.Castagnoli)
		checksum := crc32.Checksum([]byte(name), table) % mod
		buf.WriteString(strconv.FormatUint(uint64(checksum), 36))
	}

	return buf.String()
}

func ID(name string, kind Kind) *ast.Ident {
	return ast.NewIdent(Name(name, kind))
}
