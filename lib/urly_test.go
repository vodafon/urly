// lib/urly_test.go

package lib

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestExtractURL(t *testing.T) {
	// Open the file and read its contents
	inputReader, err := os.Open("tests/input1.txt")
	if err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile("tests/result1.txt")
	if err != nil {
		t.Fatal(err)
	}

	// Call our ExtractURL function with the input data
	ret, err := ExtractURL(inputReader)

	// Check that the extracted URL matches the expected one
	if !compareByteSlices(ret, result) {
		t.Errorf("not passed")
	}
}

func compareByteSlices(input, result []byte) bool {
	// Split the byte slices into lines
	inputLines := bytes.Split(input, []byte{'\n'})
	resultLines := bytes.Split(result, []byte{'\n'})

	// Determine the maximum number of lines to compare
	maxLines := len(inputLines)
	if len(resultLines) > maxLines {
		maxLines = len(resultLines)
	}

	ret := true
	// Compare lines
	for i := 0; i < maxLines; i++ {
		var inputLine, resultLine []byte
		if i < len(inputLines) {
			inputLine = inputLines[i]
		}
		if i < len(resultLines) {
			resultLine = resultLines[i]
		}

		// Compare the lines
		if !bytes.Equal(inputLine, resultLine) {
			fmt.Printf("Line %d:\nInput: %s\nResult: %s\n", i+1, inputLine, resultLine)
			ret = false
		}
	}

	return ret
}
