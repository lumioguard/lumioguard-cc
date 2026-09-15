package thirdparty

import (
	"fmt"
	"os/exec"
	"strings"
)

// RepositoryRoot returns the main module's directory. Only tools import this
// package, so os/exec never reaches the product binary.
func RepositoryRoot() (string, error) {
	output, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return "", fmt.Errorf("locate repository root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
