package pipeutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

// mockStdin simulates os.Stdin for testing purposes.
func mockStdin(t *testing.T, input string) (restore func()) {
	t.Helper()
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe for mock stdin: %v", err)
	}
	os.Stdin = r
	go func() {
		defer w.Close()
		_, copyErr := io.WriteString(w, input)
		if copyErr != nil {
			// This runs in a goroutine, so Fatalf might not stop the test correctly.
			// Print error and let test potentially fail on read issues.
			fmt.Fprintf(os.Stderr, "Error writing to mock stdin pipe: %v\n", copyErr)
		}
	}()
	return func() {
		os.Stdin = origStdin
		r.Close() // Close the read end of the pipe as well
	}
}

// captureStderr captures os.Stderr output for testing.
func captureStderr(t *testing.T) (restore func(), getOutput func() string) {
	t.Helper()
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe for capturing stderr: %v", err)
	}
	os.Stderr = w

	return func() { // restore function
			w.Close()
			os.Stderr = origStderr
		},
		func() string { // getOutput function
			// w.Close() // Ensure writer is closed before reading to get EOF
			// No, closing w should be done by restore. Reader will see EOF when writer is closed.
			var buf bytes.Buffer
			_, readErr := io.Copy(&buf, r)
			if readErr != nil {
				t.Logf("Error reading from captured stderr pipe: %v", readErr) // Log instead of Fatal in getter
			}
			r.Close() // Close reader after copy
			return buf.String()
		}
}

func TestScanner(t *testing.T) {
	input := "line one\nline two\nline three"
	restore := mockStdin(t, input)
	defer restore()

	scanner := Scanner()
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Errorf("Scanner returned an error: %v", err)
	}

	expectedLines := []string{"line one", "line two", "line three"}
	if len(lines) != len(expectedLines) {
		t.Fatalf("Scanner read %d lines, want %d. Got: %v", len(lines), len(expectedLines), lines)
	}
	for i, line := range lines {
		if line != expectedLines[i] {
			t.Errorf("Scanner line %d mismatch: got '%s', want '%s'", i, line, expectedLines[i])
		}
	}
}

func TestError(t *testing.T) {
	restoreStderr, getStderrOutput := captureStderr(t)
	defer restoreStderr()

	testError := fmt.Errorf("this is a test error")
	Error(testError)

	output := getStderrOutput()
	expectedOutput := "Error: this is a test error\n"
	if output != expectedOutput {
		t.Errorf("Error output mismatch:\nGot:  %q\nWant: %q", output, expectedOutput)
	}

	// Test with nil error
	restoreStderrNil, getStderrOutputNil := captureStderr(t) // Need new capture as previous reader is closed
	Error(nil)
	outputNil := getStderrOutputNil()
	restoreStderrNil()
	if outputNil != "" {
		t.Errorf("Error(nil) produced output: %q, want empty string", outputNil)
	}
}

func TestErrorf(t *testing.T) {
	restoreStderr, getStderrOutput := captureStderr(t)
	defer restoreStderr()

	Errorf("formatted error with value %d and string %s", 123, "test")

	output := getStderrOutput()
	expectedOutput := "Error: formatted error with value 123 and string test\n"
	if output != expectedOutput {
		t.Errorf("Errorf output mismatch:\nGot:  %q\nWant: %q", output, expectedOutput)
	}
}

// Tests for ReadAll and Print/Println could also be added here for completeness,
// although they are simpler wrappers. ReadAll would need mockStdin.
// Print/Println would need to capture stdout.
