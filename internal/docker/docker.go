package docker

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

var composeCommand = []string{"docker", "compose"} // Default to v2

func init() {
	// Check if `docker compose` (v2) exists, if not, fallback to `docker-compose` (v1)
	_, err := exec.LookPath("docker")
	if err != nil {
		// Docker itself isn't found, we'll catch this later maybe, but assume v2 for now?
		// Or should we error here? Let dependency check handle docker itself.
		return
	}
	cmd := exec.Command("docker", "compose", "version") // Simple command to check if v2 works
	if cmd.Run() != nil {                               // If running `docker compose version` fails, try v1
		_, errV1 := exec.LookPath("docker-compose")
		if errV1 == nil {
			composeCommand = []string{"docker-compose"}
			log.Println("Debug: Detected docker-compose (v1)")
		} else {
			// Neither v2 nor v1 seems to work or be found, keep default v2 and let execution fail
			log.Println("Warning: Neither 'docker compose' nor 'docker-compose' found or functional. Using default 'docker compose'. Expect errors.")
			// Dependency check in main should ideally catch this.
		}
	} else {
		log.Println("Debug: Detected docker compose (v2)")
	}
}

// runComposeCommand executes a docker compose command in a specific directory.
func runComposeCommand(projectDir string, args ...string) (string, error) {
	cmdArgs := append(composeCommand[1:], args...) // Get args for compose (e.g., "compose", "down") or just ("down") for v1
	cmd := exec.Command(composeCommand[0], cmdArgs...)
	cmd.Dir = projectDir

	log.Printf("Debug: Running command in %s: %s", projectDir, strings.Join(cmd.Args, " ")) // Log command

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
func Down(projectDir string) error {
	_, err := runComposeCommand(projectDir, "down")
	return err
}

// PsQuiet checks if any services are running for the project.
// Returns true if services are running, false otherwise.
func PsQuiet(projectDir string) (bool, error) {
	output, err := runComposeCommand(projectDir, "ps", "-q")
	if err != nil {
		// If `ps -q` fails, we can't be sure, assume services might still be running?
		// Return the error, let the caller decide how to interpret.
		return true, fmt.Errorf("failed to run ps -q: %w", err)
	}
	// If output is empty, no services are running
	log.Printf("Debug: 'ps -q' output for %s: %s", projectDir, strings.TrimSpace(output)) // Log output
	return strings.TrimSpace(output) != "", nil
}

// Pull pulls the latest images for the project.
func Pull(projectDir string) error {
	_, err := runComposeCommand(projectDir, "pull")
	return err
}

// UpDetached starts the docker compose stack in detached mode.
func UpDetached(projectDir string) error {
	_, err := runComposeCommand(projectDir, "up", "-d")
	return err
}
