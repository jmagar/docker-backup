package docker

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"docker-backup-tool/internal/logutil"
)

// Note: The logic to detect docker compose v1/v2 command path
// has been moved to main.go's dependency check.
// Functions now accept the command path as an argument.

// runComposeCommand executes a docker compose command in a specific directory.
func runComposeCommand(projectDir string, dockerCmd string, args ...string) (string, error) {
	// Determine base command (docker or docker-compose)
	var baseCmd string
	var composeArgs []string
	if dockerCmd == "docker-compose" {
		baseCmd = dockerCmd
		composeArgs = args
	} else {
		baseCmd = dockerCmd // Should be "docker"
		composeArgs = append([]string{"compose"}, args...)
	}

	cmd := exec.Command(baseCmd, composeArgs...)
	cmd.Dir = projectDir

	logutil.Debug("Running command in %s: %s", projectDir, strings.Join(cmd.Args, " ")) // Use logutil

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stdout.String(), fmt.Errorf("failed to run %s in %s: %w\nStderr: %s",
			strings.Join(cmd.Args, " "), projectDir, err, stderr.String())
	}
	return stdout.String(), nil
}

// Down stops the docker compose stack.
func Down(projectDir string, dockerCmd string) error {
	_, err := runComposeCommand(projectDir, dockerCmd, "down")
	return err
}

// PsQuiet checks if any services are running for the project.
// Returns true if services are running, false otherwise.
func PsQuiet(projectDir string, dockerCmd string) (bool, error) {
	output, err := runComposeCommand(projectDir, dockerCmd, "ps", "-q")
	if err != nil {
		// If `ps -q` fails, we can't be sure, assume services might still be running?
		// Return the error, let the caller decide how to interpret.
		return true, fmt.Errorf("failed to check running services (ps -q): %w", err)
	}
	// If output is empty, no services are running
	logutil.Debug("'ps -q' output for %s: [%s]", projectDir, strings.TrimSpace(output)) // Log output
	return strings.TrimSpace(output) != "", nil
}

// Pull pulls the latest images for the project.
func Pull(projectDir string, dockerCmd string) error {
	_, err := runComposeCommand(projectDir, dockerCmd, "pull")
	return err
}

// UpDetached starts the docker compose stack in detached mode.
func UpDetached(projectDir string, dockerCmd string) error {
	_, err := runComposeCommand(projectDir, dockerCmd, "up", "-d")
	return err
}
