package util

import (
	"go/ast"
	"hash/crc32"
	"strconv"
	"strings"
	"unicode"
)

type IDKind int

const (
	IDExported IDKind = iota
	IDInternal
	IDLocal
)

func Mangle(name string, kind IDKind) *ast.Ident {
	var buf strings.Builder
	buf.Grow(len(name))

	switch kind {
	case IDExported:
		buf.WriteByte('X')
	case IDInternal:
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
	if suffix && kind != IDLocal {
		buf.WriteByte('_')
		table := crc32.MakeTable(crc32.Castagnoli)
		checksum := crc32.Checksum([]byte(name), table)
		buf.WriteString(strconv.FormatUint(uint64(checksum), 36))
	}

	return ast.NewIdent(buf.String())
}
