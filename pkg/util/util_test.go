package util

import "testing"

func TestRandString(t *testing.T) {
	length := 4
	actualLength := len(RandString(length))
	if actualLength != length {
		t.Fatalf("Expected string of length %v got length %v", length, actualLength)
	}
}
