package main

import (
	"bufio"
	"fmt"
)

type sectionID byte

const (
	sectionCustom sectionID = iota
	sectionType
	sectionImport
	sectionFunction
	sectionTable
	sectionMemory
	sectionGlobal
	sectionExport
	sectionStart
	sectionElement
	sectionCode
	sectionData
	sectionDataCount
)

func peekSectionID(r *bufio.Reader) (sectionID, error) {
	b, err := r.Peek(1)
	if err != nil {
		return 0, err
	}
	return sectionID(b[0]), nil
}

func skipSection(r *bufio.Reader, id sectionID) error {
	if b, err := r.ReadByte(); err != nil {
		return err
	} else if sectionID(b) != id {
		return fmt.Errorf("unexpected section: %d", id)
	}
	size, err := readLEB128(r)
	if err != nil {
		return err
	}
	_, err = r.Discard(int(size))
	return err
}
