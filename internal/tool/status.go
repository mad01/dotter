package tool

import (
	"os/exec"
	"strings"
)

// CheckStatus runs the tool's check_command and returns true if the command exits successfully (status 0).
// It returns false otherwise, or if the command is empty.
func CheckStatus(checkCommand string) bool {
	if strings.TrimSpace(checkCommand) == "" {
		return false // Or handle as an error/unknown status
	}

	// Note: Using "sh -c" to allow for shell constructs in the check_command (e.g., pipes, command -v).
	// This might have security implications if the check_command comes from untrusted sources,
	// but in this context, it comes from the user's own config.toml.
	cmd := exec.Command("sh", "-c", checkCommand)

	err := cmd.Run()
	return err == nil // `Run` returns nil on exit code 0
}
