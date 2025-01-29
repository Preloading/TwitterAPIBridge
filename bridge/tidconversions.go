package bridge

import (
	"fmt"
	"strings"
)

var tidChars = "234567abcdefghijklmnopqrstuvwxyz"

// numToTid converts a 64-bit number into a Bluesky TID.
func NumToTid(number uint64) (string, error) {
	// Ensure the first bit is 0
	if (number & 0x8000000000000000) != 0 {
		return "", fmt.Errorf("first bit must be 0")
	}

	// Convert to base32 manually
	// the other one kept losing precision
	var result strings.Builder
	for i := 0; i < 13; i++ {
		index := number & 0x1F // Take 5 bits
		result.WriteByte(tidChars[index])
		number >>= 5
	}

	// Reverse the string since we built it backwards
	runes := []rune(result.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes), nil
}

// tidToNum converts a Bluesky TID into a 64-bit number.
func TidToNum(tid string) (uint64, error) {
	if len(tid) != 13 {
		return 0, fmt.Errorf("TID must be 13 characters")
	}

	var num uint64
	for _, c := range tid {
		index := strings.IndexRune(tidChars, c)
		if index == -1 {
			return 0, fmt.Errorf("invalid character in TID: %c", c)
		}
		num = (num << 5) | uint64(index)
	}

	// Verify the first bit is 0
	if (num & 0x8000000000000000) != 0 {
		return 0, fmt.Errorf("first bit must be 0")
	}

	return num, nil
}
