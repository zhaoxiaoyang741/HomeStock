package model

import (
	"crypto/rand"
	"fmt"
)

const categoryAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func newID() (string, error) {
	var raw [16]byte

	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}

	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		raw[0:4],
		raw[4:6],
		raw[6:8],
		raw[8:10],
		raw[10:16],
	), nil
}

func newCategoryID() (string, error) {
	const prefix = "cat"
	const randomLen = 7

	var raw [randomLen]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate category id: %w", err)
	}

	buf := make([]byte, 0, len(prefix)+randomLen)
	buf = append(buf, prefix...)
	for _, b := range raw {
		buf = append(buf, categoryAlphabet[int(b)%len(categoryAlphabet)])
	}

	return string(buf), nil
}
