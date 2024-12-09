// lib/urly_test.go

package lib

import (
	"bytes"
	"os"
	"testing"
)

func TestExtractURL(t *testing.T) {
	// Open the file and read its contents
	input, err := os.ReadFile("tests/input1.txt")
	if err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile("tests/result1.txt")
	if err != nil {
		t.Fatal(err)
	}

	// Call our ExtractURL function with the input data
	ret, err := ExtractURL(input)

	// Check that the extracted URL matches the expected one
	if !bytes.Equal(ret, result) {
		t.Errorf("expected %s but got %s", result, ret)
	}
}
