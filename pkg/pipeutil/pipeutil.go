package pipeutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// ReadAll reads all data from os.Stdin.
func ReadAll() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}

// Scanner returns a new bufio.Scanner for os.Stdin.
func Scanner() *bufio.Scanner {
	return bufio.NewScanner(os.Stdin)
}

// Print writes data to os.Stdout.
// It does not add a newline character at the end.
func Print(data []byte) (int, error) {
	return os.Stdout.Write(data)
}

// Println prints a string to os.Stdout, followed by a newline character.
func Println(s string) (int, error) {
	return fmt.Fprintln(os.Stdout, s)
}

// Error prints an error message to os.Stderr.
func Error(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
}

// Errorf prints a formatted error message to os.Stderr.
func Errorf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", a...)
}

// Constants for common exit codes
const (
	ExitSuccess = 0
	ExitFailure = 1
)
