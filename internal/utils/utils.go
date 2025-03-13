package utils

import (
	"strconv"
)

// ParseIntOption parses a string value to an integer, returning 0 if the string is empty or invalid
func ParseIntOption(value string) int {
	if value == "" {
		return 0
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return num
}
