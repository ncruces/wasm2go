package util

import (
	"hash/crc32"
	"strconv"
	"strings"
	"unicode"
)

func Mangle(buf *strings.Builder, name string) {
	var suffix bool
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			suffix = true
			r = '_'
		}
		buf.WriteRune(r)
	}
	if suffix {
		buf.WriteByte('_')
		table := crc32.MakeTable(crc32.Castagnoli)
		checksum := crc32.Checksum([]byte(name), table)
		buf.WriteString(strconv.FormatUint(uint64(checksum), 36))
	}
}
